package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type genealogyMemberRepository struct {
	db database.Database
}

// NewGenealogyMemberRepository returns the Postgres-backed
// domain.GenealogyMemberRepository.
func NewGenealogyMemberRepository(db database.Database) domain.GenealogyMemberRepository {
	return &genealogyMemberRepository{db: db}
}

// List returns every member of the tree, flat, ordered by generation then id.
func (r *genealogyMemberRepository) List(ctx context.Context, genealogyID int64) ([]domain.GenealogyMember, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+genealogyMemberColumns+` FROM genealogy_members WHERE genealogy_id = $1 ORDER BY generation, id`,
		genealogyID)
	if err != nil {
		return nil, fmt.Errorf("query genealogy members: %w", err)
	}
	return listGenealogyMembers(rows)
}

func listGenealogyMembers(rows *sql.Rows) ([]domain.GenealogyMember, error) {
	return scanRows(rows, "genealogy member", func(s rowScanner) (domain.GenealogyMember, error) {
		m, err := scanGenealogyMember(s)
		if err != nil {
			return domain.GenealogyMember{}, err
		}
		return *m, nil
	})
}

// FindByID returns the member of this tree. Filtering on genealogy_id is what
// keeps a member of another tree unreadable (AC22, AC29).
func (r *genealogyMemberRepository) FindByID(ctx context.Context, genealogyID, memberID int64) (*domain.GenealogyMember, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+genealogyMemberColumns+` FROM genealogy_members WHERE id = $1 AND genealogy_id = $2`,
		memberID, genealogyID)
	m, err := scanGenealogyMember(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrMemberNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find genealogy member: %w", err)
	}
	return m, nil
}

// FindMemberIDByPhone returns the member of the tree carrying this phone number
// (AC59), or nil.
func (r *genealogyMemberRepository) FindMemberIDByPhone(ctx context.Context, genealogyID int64, phone string) (*int64, error) {
	var id int64
	err := r.db.QueryRow(ctx,
		`SELECT id FROM genealogy_members WHERE genealogy_id = $1 AND phone_number = $2`,
		genealogyID, phone).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find member by phone: %w", err)
	}
	return &id, nil
}

// Add is Step M-2: attach m to relatedID in one transaction that holds the
// tree's row lock, so the member cap, the spouse cap, the default birth order
// and the clan-head swap are all decided against a stable tree.
func (r *genealogyMemberRepository) Add(ctx context.Context, genealogyID int64, relation domain.MemberRelation, relatedID int64, m *domain.GenealogyMember, limits domain.MemberLimits) (*domain.GenealogyMember, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockGenealogy(ctx, tx, genealogyID); err != nil {
		return nil, err
	}

	var (
		relKind, relGender string
		relGeneration      int
	)
	err = tx.QueryRowContext(ctx,
		`SELECT kind, gender, generation FROM genealogy_members WHERE id = $1 AND genealogy_id = $2`,
		relatedID, genealogyID).Scan(&relKind, &relGender, &relGeneration)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrMemberNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read related member: %w", err)
	}

	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM genealogy_members WHERE genealogy_id = $1`, genealogyID).Scan(&count); err != nil {
		return nil, fmt.Errorf("count members: %w", err)
	}
	if count >= limits.MaxMembers {
		return nil, domain.ErrMemberLimitReached
	}

	toInsert := *m
	toInsert.GenealogyID = genealogyID
	toInsert.ID = 0
	switch relation {
	case domain.RelationChild:
		if relKind != string(domain.MemberKindBlood) || relGender != string(domain.GenderMale) {
			return nil, genValidation("related_member_id", "a child is attached to a male member of the clan")
		}
		if relGeneration+1 > limits.MaxGeneration {
			return nil, genValidation("related_member_id", "the tree cannot go that deep")
		}
		toInsert.Kind = domain.MemberKindBlood
		toInsert.ParentID = &relatedID
		toInsert.SpouseOfID = nil
		toInsert.Generation = relGeneration + 1

		if toInsert.OtherParentID != nil {
			var ok bool
			if err := tx.QueryRowContext(ctx, `
				SELECT EXISTS (SELECT 1 FROM genealogy_members
				               WHERE id = $1 AND genealogy_id = $2 AND kind = 'in_law' AND spouse_of_id = $3)`,
				*toInsert.OtherParentID, genealogyID, relatedID).Scan(&ok); err != nil {
				return nil, fmt.Errorf("check other parent: %w", err)
			}
			if !ok {
				return nil, genValidation("other_parent_id", "other_parent_id must be a spouse of the parent")
			}
		}
		if toInsert.BirthOrder == nil {
			var next int
			if err := tx.QueryRowContext(ctx,
				`SELECT coalesce(max(birth_order), 0) + 1 FROM genealogy_members WHERE parent_id = $1`,
				relatedID).Scan(&next); err != nil {
				return nil, fmt.Errorf("next birth order: %w", err)
			}
			toInsert.BirthOrder = &next
		}
	case domain.RelationSpouse:
		if relKind != string(domain.MemberKindBlood) {
			return nil, genValidation("related_member_id", "a spouse is attached to a member of the clan")
		}
		var spouses int
		if err := tx.QueryRowContext(ctx,
			`SELECT count(*) FROM genealogy_members WHERE spouse_of_id = $1`, relatedID).Scan(&spouses); err != nil {
			return nil, fmt.Errorf("count spouses: %w", err)
		}
		if spouses >= limits.MaxSpouses {
			return nil, genValidation("related_member_id", "this member already has the maximum number of spouses")
		}
		toInsert.Kind = domain.MemberKindInLaw
		toInsert.SpouseOfID = &relatedID
		toInsert.ParentID, toInsert.OtherParentID, toInsert.BirthOrder = nil, nil, nil
		toInsert.IsClanHead = false
		toInsert.Generation = relGeneration
	default:
		return nil, genValidation("relation", "relation must be child or spouse")
	}

	if toInsert.IsClanHead {
		if err := clearClanHead(ctx, tx, genealogyID); err != nil {
			return nil, err
		}
	}
	created, err := insertGenealogyMember(ctx, tx, &toInsert)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return created, nil
}

// Update replaces the editable fields (never relationships). In the same
// transaction it swaps the clan-head flag and cancels the pending invitation of
// a member whose phone number changed or was cleared (AC30, AC82).
func (r *genealogyMemberRepository) Update(ctx context.Context, genealogyID int64, m *domain.GenealogyMember) (*domain.GenealogyMember, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockGenealogy(ctx, tx, genealogyID); err != nil {
		return nil, err
	}

	var curPhone sql.NullString
	err = tx.QueryRowContext(ctx,
		`SELECT phone_number FROM genealogy_members WHERE id = $1 AND genealogy_id = $2 FOR UPDATE`,
		m.ID, genealogyID).Scan(&curPhone)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrMemberNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read member: %w", err)
	}

	if m.IsClanHead {
		if err := clearClanHeadExcept(ctx, tx, genealogyID, m.ID); err != nil {
			return nil, err
		}
	}
	if phoneChanged(curPhone, m.PhoneNumber) {
		if _, err := tx.ExecContext(ctx, `
			UPDATE genealogy_invitations SET status = 'cancelled', responded_at = now()
			WHERE member_id = $1 AND status = 'pending'`, m.ID); err != nil {
			return nil, fmt.Errorf("cancel invitation: %w", err)
		}
	}

	row := tx.QueryRowContext(ctx, `
		UPDATE genealogy_members SET
			gender = $3, full_name = $4, birth_year = $5, is_alive = $6, death_year = $7, residence = $8,
			dharma_name = $9, tablet_location = $10, burial_place = $11, death_month = $12, death_day = $13,
			death_entered_as_lunar = $14, phone_number = $15, note = $16, is_clan_head = $17,
			birth_order = $18, updated_at = now()
		WHERE id = $1 AND genealogy_id = $2
		RETURNING `+genealogyMemberColumns,
		m.ID, genealogyID, string(m.Gender), m.FullName, nullableInt(m.BirthYear), m.IsAlive, nullableInt(m.DeathYear),
		nullableStringPtr(m.Residence), nullableStringPtr(m.DharmaName), nullableStringPtr(m.TabletLocation),
		nullableStringPtr(m.BurialPlace), nullableInt(m.DeathMonth), nullableInt(m.DeathDay),
		nullableBool(m.DeathEnteredAsLunar), nullableStringPtr(m.PhoneNumber), nullableStringPtr(m.Note),
		m.IsClanHead, nullableInt(m.BirthOrder))
	updated, err := scanGenealogyMember(row)
	if err != nil {
		if isUniqueViolation(err, constraintMemberPhone) {
			return nil, domain.ErrMemberPhoneInUse
		}
		return nil, fmt.Errorf("update genealogy member: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return updated, nil
}

func phoneChanged(cur sql.NullString, next *string) bool {
	switch {
	case !cur.Valid && next == nil:
		return false
	case cur.Valid && next != nil:
		return cur.String != *next
	default:
		return true
	}
}

// Delete is Step D-2: a blood member without children is deleted together with
// its in-laws; an in-law is deleted alone. The founder and a member with
// children are refused.
func (r *genealogyMemberRepository) Delete(ctx context.Context, genealogyID, memberID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockGenealogy(ctx, tx, genealogyID); err != nil {
		return err
	}

	var (
		kind     string
		parentID sql.NullInt64
	)
	err = tx.QueryRowContext(ctx,
		`SELECT kind, parent_id FROM genealogy_members WHERE id = $1 AND genealogy_id = $2 FOR UPDATE`,
		memberID, genealogyID).Scan(&kind, &parentID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrMemberNotFound
	}
	if err != nil {
		return fmt.Errorf("read member: %w", err)
	}

	if kind == string(domain.MemberKindBlood) {
		if !parentID.Valid {
			return domain.ErrFounderNotDeletable
		}
		var has bool
		if err := tx.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM genealogy_members WHERE parent_id = $1)`, memberID).Scan(&has); err != nil {
			return fmt.Errorf("check children: %w", err)
		}
		if has {
			return domain.ErrMemberHasDescendants
		}
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM genealogy_members WHERE id = $1 AND genealogy_id = $2`, memberID, genealogyID); err != nil {
		if isForeignKeyViolation(err, constraintMemberParentFK) {
			return domain.ErrMemberHasDescendants
		}
		return fmt.Errorf("delete genealogy member: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// Move is Step R-2: re-attach a member and shift the generation of its whole
// branch. The tree is loaded under the row lock and every rule is checked in
// memory, so nothing is half-applied and a loop is caught before any write.
func (r *genealogyMemberRepository) Move(ctx context.Context, genealogyID, memberID int64, in domain.MoveMemberInput, limits domain.MemberLimits) (*domain.GenealogyMember, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockGenealogy(ctx, tx, genealogyID); err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx,
		`SELECT `+genealogyMemberColumns+` FROM genealogy_members WHERE genealogy_id = $1`, genealogyID)
	if err != nil {
		return nil, fmt.Errorf("load tree: %w", err)
	}
	all, err := listGenealogyMembers(rows)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]*domain.GenealogyMember, len(all))
	for i := range all {
		byID[all[i].ID] = &all[i]
	}

	target, ok := byID[memberID]
	if !ok {
		return nil, domain.ErrMemberNotFound
	}
	related, ok := byID[in.RelatedMemberID]
	if !ok {
		return nil, domain.ErrMemberNotFound
	}
	if target.IsFounder() {
		return nil, domain.ErrFounderNotMovable
	}

	var (
		newParent, newOther *int64
		newOrder            *int
		newGeneration       int
	)
	switch domain.MemberRelation(in.Relation) {
	case domain.RelationChild:
		if target.Kind != domain.MemberKindBlood {
			return nil, genValidation("relation", "an in-law is re-attached with relation spouse")
		}
		if related.Kind != domain.MemberKindBlood || related.Gender != domain.GenderMale {
			return nil, genValidation("related_member_id", "a child is attached to a male member of the clan")
		}
		// No loop: the new parent must not be the member or one of its descendants.
		for cur, n := related, 0; cur != nil && n <= 1000; n++ {
			if cur.ID == target.ID {
				return nil, genValidation("related_member_id", "a member cannot be moved under itself or its descendants")
			}
			if cur.ParentID == nil {
				break
			}
			cur = byID[*cur.ParentID]
		}
		if in.OtherParentID != nil {
			o, ok := byID[*in.OtherParentID]
			if !ok || o.Kind != domain.MemberKindInLaw || o.SpouseOfID == nil || *o.SpouseOfID != related.ID {
				return nil, genValidation("other_parent_id", "other_parent_id must be a spouse of the parent")
			}
		}

		branch := branchOf(all, target.ID)
		delta := related.Generation + 1 - target.Generation
		deepest := 0
		for _, id := range branch {
			if g := byID[id].Generation; g > deepest {
				deepest = g
			}
		}
		if deepest+delta > limits.MaxGeneration {
			return nil, genValidation("related_member_id", "the tree cannot go that deep")
		}

		newParent, newOther = &related.ID, in.OtherParentID
		newGeneration = target.Generation + delta
		switch {
		case in.BirthOrder != nil:
			newOrder = in.BirthOrder
		case target.ParentID != nil && *target.ParentID == related.ID:
			newOrder = target.BirthOrder
		default:
			next := 1
			for i := range all {
				c := &all[i]
				if c.ParentID != nil && *c.ParentID == related.ID && c.ID != target.ID &&
					c.BirthOrder != nil && *c.BirthOrder >= next {
					next = *c.BirthOrder + 1
				}
			}
			newOrder = &next
		}

		if err := moveTarget(ctx, tx, genealogyID, target.ID, newParent, nil, newOther, newOrder, newGeneration); err != nil {
			return nil, err
		}
		if delta != 0 {
			others := make([]int64, 0, len(branch))
			for _, id := range branch {
				if id != target.ID {
					others = append(others, id)
				}
			}
			if len(others) > 0 {
				if _, err := tx.ExecContext(ctx, `
					UPDATE genealogy_members SET generation = generation + $1, updated_at = now()
					WHERE genealogy_id = $2 AND id = ANY($3)`, delta, genealogyID, pq.Array(others)); err != nil {
					return nil, fmt.Errorf("shift generations: %w", err)
				}
			}
		}

	case domain.RelationSpouse:
		if target.Kind != domain.MemberKindInLaw {
			return nil, genValidation("relation", "a member of the clan is re-attached with relation child")
		}
		if related.Kind != domain.MemberKindBlood {
			return nil, genValidation("related_member_id", "a spouse is attached to a member of the clan")
		}
		spouses := 0
		for i := range all {
			if all[i].SpouseOfID != nil && *all[i].SpouseOfID == related.ID && all[i].ID != target.ID {
				spouses++
			}
		}
		if spouses >= limits.MaxSpouses {
			return nil, genValidation("related_member_id", "this member already has the maximum number of spouses")
		}
		if err := moveTarget(ctx, tx, genealogyID, target.ID, nil, &related.ID, nil, nil, related.Generation); err != nil {
			return nil, err
		}
		// Children that named this in-law as the other parent lose that link (AC58, AC76).
		if _, err := tx.ExecContext(ctx, `
			UPDATE genealogy_members SET other_parent_id = NULL, updated_at = now()
			WHERE genealogy_id = $1 AND other_parent_id = $2`, genealogyID, target.ID); err != nil {
			return nil, fmt.Errorf("clear other parent: %w", err)
		}
	default:
		return nil, genValidation("relation", "relation must be child or spouse")
	}

	row := tx.QueryRowContext(ctx,
		`SELECT `+genealogyMemberColumns+` FROM genealogy_members WHERE id = $1 AND genealogy_id = $2`,
		target.ID, genealogyID)
	moved, err := scanGenealogyMember(row)
	if err != nil {
		return nil, fmt.Errorf("read moved member: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return moved, nil
}

func moveTarget(ctx context.Context, q genQuerier, genealogyID, id int64, parent, spouse, other *int64, order *int, generation int) error {
	_, err := q.ExecContext(ctx, `
		UPDATE genealogy_members SET parent_id = $3, spouse_of_id = $4, other_parent_id = $5,
			birth_order = $6, generation = $7, updated_at = now()
		WHERE id = $1 AND genealogy_id = $2`,
		id, genealogyID, nullableInt64(parent), nullableInt64(spouse), nullableInt64(other),
		nullableInt(order), generation)
	if err != nil {
		return fmt.Errorf("re-attach member: %w", err)
	}
	return nil
}

// branchOf lists the ids of a member, every blood descendant, and the in-laws
// of each of them: everything whose generation moves with the member.
func branchOf(all []domain.GenealogyMember, rootID int64) []int64 {
	children := map[int64][]int64{}
	inLaws := map[int64][]int64{}
	for i := range all {
		m := &all[i]
		if m.Kind == domain.MemberKindBlood && m.ParentID != nil {
			children[*m.ParentID] = append(children[*m.ParentID], m.ID)
		}
		if m.Kind == domain.MemberKindInLaw && m.SpouseOfID != nil {
			inLaws[*m.SpouseOfID] = append(inLaws[*m.SpouseOfID], m.ID)
		}
	}
	out := []int64{}
	stack := []int64{rootID}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		out = append(out, id)
		out = append(out, inLaws[id]...)
		stack = append(stack, children[id]...)
	}
	return out
}

func clearClanHead(ctx context.Context, q genQuerier, genealogyID int64) error {
	if _, err := q.ExecContext(ctx,
		`UPDATE genealogy_members SET is_clan_head = false, updated_at = now() WHERE genealogy_id = $1 AND is_clan_head`,
		genealogyID); err != nil {
		return fmt.Errorf("clear clan head: %w", err)
	}
	return nil
}

func clearClanHeadExcept(ctx context.Context, q genQuerier, genealogyID, keepID int64) error {
	if _, err := q.ExecContext(ctx, `
		UPDATE genealogy_members SET is_clan_head = false, updated_at = now()
		WHERE genealogy_id = $1 AND is_clan_head AND id <> $2`, genealogyID, keepID); err != nil {
		return fmt.Errorf("clear clan head: %w", err)
	}
	return nil
}
