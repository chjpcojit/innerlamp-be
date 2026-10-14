package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// Constraint and index names from migrations/0028_genealogy.sql, matched by name
// (never by message text) when a write is translated into a domain error.
const (
	constraintMemberPhone      = "uq_genealogy_members_phone"
	constraintMemberFounder    = "uq_genealogy_members_founder"
	constraintMemberClanHead   = "uq_genealogy_members_clan_head"
	constraintMemberParentFK   = "fk_genealogy_members_parent"
	constraintInvitationActive = "uq_genealogy_invitations_pending"
)

// genealogyMemberColumns is shared by every member read so list, detail and
// RETURNING scan identically.
const genealogyMemberColumns = `id, genealogy_id, kind, gender, full_name, birth_year, is_alive, death_year,
		residence, dharma_name, tablet_location, burial_place, death_month, death_day, death_entered_as_lunar,
		phone_number, note, is_clan_head, parent_id, other_parent_id, spouse_of_id, birth_order, generation,
		created_at, updated_at`

// genQuerier is what both *sql.Tx and *sql.DB offer, so helpers work in or out
// of a transaction.
type genQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type genealogyRepository struct {
	db database.Database
}

// NewGenealogyRepository returns the Postgres-backed domain.GenealogyRepository.
func NewGenealogyRepository(db database.Database) domain.GenealogyRepository {
	return &genealogyRepository{db: db}
}

// Create writes the tree, its empty chronicle, the founder and the spouses in
// one transaction (Step G-2). The users row is locked first so two concurrent
// creates of one user cannot both pass the count check.
func (r *genealogyRepository) Create(ctx context.Context, ownerUserID int64, in domain.CreateGenealogyParams, maxPerUser int) (int64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var locked int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, ownerUserID).Scan(&locked); err != nil {
		return 0, fmt.Errorf("lock user %d: %w", ownerUserID, err)
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM genealogies WHERE owner_user_id = $1`, ownerUserID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count genealogies: %w", err)
	}
	if count >= maxPerUser {
		return 0, domain.ErrGenealogyLimitReached
	}

	var gid int64
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO genealogies (owner_user_id, name, ancestral_home) VALUES ($1, $2, $3) RETURNING id`,
		ownerUserID, in.Name, in.AncestralHome).Scan(&gid); err != nil {
		return 0, fmt.Errorf("insert genealogy: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO genealogy_chronicles (genealogy_id) VALUES ($1)`, gid); err != nil {
		return 0, fmt.Errorf("insert chronicle: %w", err)
	}

	founder := in.Founder
	founder.GenealogyID = gid
	created, err := insertGenealogyMember(ctx, tx, &founder)
	if err != nil {
		return 0, err
	}
	for i := range in.Spouses {
		s := in.Spouses[i]
		s.GenealogyID = gid
		s.SpouseOfID = &created.ID
		if _, err := insertGenealogyMember(ctx, tx, &s); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return gid, nil
}

// ListByUser returns the trees the user owns or views, newest first (AC13).
func (r *genealogyRepository) ListByUser(ctx context.Context, userID int64) ([]domain.GenealogySummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT g.id, g.name, g.ancestral_home,
		       CASE WHEN g.owner_user_id = $1 THEN 'owner' ELSE 'viewer' END,
		       g.created_at
		FROM genealogies g
		LEFT JOIN genealogy_viewers v ON v.genealogy_id = g.id AND v.user_id = $1
		WHERE g.owner_user_id = $1 OR v.user_id IS NOT NULL
		ORDER BY g.created_at DESC, g.id DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("query genealogies: %w", err)
	}
	return scanRows(rows, "genealogy summary", func(s rowScanner) (domain.GenealogySummary, error) {
		var g domain.GenealogySummary
		var role string
		if err := s.Scan(&g.ID, &g.Name, &g.AncestralHome, &role, &g.CreatedAt); err != nil {
			return g, err
		}
		g.MyRole = domain.Role(role)
		return g, nil
	})
}

// ResolveRole is Step A: one query that says whether the caller owns the tree,
// views it, or has no role (reported as not found, AC15).
func (r *genealogyRepository) ResolveRole(ctx context.Context, genealogyID, userID int64) (domain.Role, error) {
	var role sql.NullString
	err := r.db.QueryRow(ctx, `
		SELECT CASE WHEN g.owner_user_id = $2 THEN 'owner'
		            WHEN v.user_id IS NOT NULL THEN 'viewer' END
		FROM genealogies g
		LEFT JOIN genealogy_viewers v ON v.genealogy_id = g.id AND v.user_id = $2
		WHERE g.id = $1`, genealogyID, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !role.Valid) {
		return "", domain.ErrGenealogyNotFound
	}
	if err != nil {
		return "", fmt.Errorf("resolve role: %w", err)
	}
	return domain.Role(role.String), nil
}

// FindDetail returns the overview: info, chronicle and the counts computed from
// the tree now (AC14, AC41).
func (r *genealogyRepository) FindDetail(ctx context.Context, genealogyID int64) (*domain.GenealogyDetail, error) {
	var d domain.GenealogyDetail
	var (
		founder  sql.NullString
		clanHead sql.NullInt64
	)
	err := r.db.QueryRow(ctx, `
		SELECT g.id, g.owner_user_id, g.name, g.ancestral_home, g.created_at, g.updated_at,
		       (SELECT count(*) FROM genealogy_members WHERE genealogy_id = g.id),
		       (SELECT coalesce(max(generation), 0) FROM genealogy_members WHERE genealogy_id = g.id),
		       (SELECT full_name FROM genealogy_members WHERE genealogy_id = g.id AND kind = 'blood' AND parent_id IS NULL),
		       (SELECT id FROM genealogy_members WHERE genealogy_id = g.id AND is_clan_head)
		FROM genealogies g
		WHERE g.id = $1`, genealogyID).Scan(
		&d.ID, &d.OwnerUserID, &d.Name, &d.AncestralHome, &d.CreatedAt, &d.UpdatedAt,
		&d.MembersCount, &d.GenerationsCount, &founder, &clanHead)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrGenealogyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find genealogy: %w", err)
	}
	if founder.Valid {
		d.FounderName = founder.String
	}
	if clanHead.Valid {
		v := clanHead.Int64
		d.ClanHeadMemberID = &v
	}

	c, err := r.findChronicle(ctx, genealogyID)
	if err != nil {
		return nil, err
	}
	d.Chronicle = *c
	return &d, nil
}

func (r *genealogyRepository) findChronicle(ctx context.Context, genealogyID int64) (*domain.Chronicle, error) {
	var (
		c                                                               domain.Chronicle
		surname, location, origin, history, hallName, hallAddr          sql.NullString
		hoanhPhi, hoanhReading, hoanhMeaning                            sql.NullString
		upper, upperReading, upperMeaning, lower, lowerReading, lowerMn sql.NullString
		teaching                                                        sql.NullString
		since, chapYear, chapMonth, chapDay                             sql.NullInt64
	)
	err := r.db.QueryRow(ctx, `
		SELECT genealogy_id, surname, current_location, since_year, origin, history, hall_name, hall_address,
		       chap_year, chap_month, chap_day, chap_entered_as_lunar,
		       hoanh_phi, hoanh_phi_reading, hoanh_phi_meaning,
		       couplet_upper, couplet_upper_reading, couplet_upper_meaning,
		       couplet_lower, couplet_lower_reading, couplet_lower_meaning,
		       teaching, updated_at
		FROM genealogy_chronicles WHERE genealogy_id = $1`, genealogyID).Scan(
		&c.GenealogyID, &surname, &location, &since, &origin, &history, &hallName, &hallAddr,
		&chapYear, &chapMonth, &chapDay, &c.ChapEnteredAsLunar,
		&hoanhPhi, &hoanhReading, &hoanhMeaning,
		&upper, &upperReading, &upperMeaning, &lower, &lowerReading, &lowerMn,
		&teaching, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrGenealogyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find chronicle: %w", err)
	}
	c.Surname, c.CurrentLocation, c.Origin, c.History = strPtr(surname), strPtr(location), strPtr(origin), strPtr(history)
	c.HallName, c.HallAddress = strPtr(hallName), strPtr(hallAddr)
	c.SinceYear, c.ChapYear, c.ChapMonth, c.ChapDay = intPtr(since), intPtr(chapYear), intPtr(chapMonth), intPtr(chapDay)
	c.HoanhPhi, c.HoanhPhiReading, c.HoanhPhiMeaning = strPtr(hoanhPhi), strPtr(hoanhReading), strPtr(hoanhMeaning)
	c.CoupletUpper, c.CoupletUpperReading, c.CoupletUpperMeaning = strPtr(upper), strPtr(upperReading), strPtr(upperMeaning)
	c.CoupletLower, c.CoupletLowerReading, c.CoupletLowerMeaning = strPtr(lower), strPtr(lowerReading), strPtr(lowerMn)
	c.Teaching = strPtr(teaching)
	return &c, nil
}

// UpdateInfo changes name and hometown (AC16).
func (r *genealogyRepository) UpdateInfo(ctx context.Context, genealogyID int64, name, ancestralHome string) error {
	res, err := r.db.Exec(ctx,
		`UPDATE genealogies SET name = $2, ancestral_home = $3, updated_at = now() WHERE id = $1`,
		genealogyID, name, ancestralHome)
	if err != nil {
		return fmt.Errorf("update genealogy: %w", err)
	}
	return expectOneRow(res, domain.ErrGenealogyNotFound)
}

// Delete removes the tree; members, chronicle, viewers and invitations go with
// it by cascade (AC17).
func (r *genealogyRepository) Delete(ctx context.Context, genealogyID int64) error {
	res, err := r.db.Exec(ctx, `DELETE FROM genealogies WHERE id = $1`, genealogyID)
	if err != nil {
		return fmt.Errorf("delete genealogy: %w", err)
	}
	return expectOneRow(res, domain.ErrGenealogyNotFound)
}

// UpsertChronicle replaces every chronicle field (AC38). The row is created with
// the tree, but an upsert keeps the method safe for a tree that predates it.
func (r *genealogyRepository) UpsertChronicle(ctx context.Context, c *domain.Chronicle) error {
	res, err := r.db.Exec(ctx, `
		INSERT INTO genealogy_chronicles
			(genealogy_id, surname, current_location, since_year, origin, history, hall_name, hall_address,
			 chap_year, chap_month, chap_day, chap_entered_as_lunar,
			 hoanh_phi, hoanh_phi_reading, hoanh_phi_meaning,
			 couplet_upper, couplet_upper_reading, couplet_upper_meaning,
			 couplet_lower, couplet_lower_reading, couplet_lower_meaning, teaching)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22
		WHERE EXISTS (SELECT 1 FROM genealogies WHERE id = $1)
		ON CONFLICT (genealogy_id) DO UPDATE SET
			surname = EXCLUDED.surname, current_location = EXCLUDED.current_location,
			since_year = EXCLUDED.since_year, origin = EXCLUDED.origin, history = EXCLUDED.history,
			hall_name = EXCLUDED.hall_name, hall_address = EXCLUDED.hall_address,
			chap_year = EXCLUDED.chap_year, chap_month = EXCLUDED.chap_month, chap_day = EXCLUDED.chap_day,
			chap_entered_as_lunar = EXCLUDED.chap_entered_as_lunar,
			hoanh_phi = EXCLUDED.hoanh_phi, hoanh_phi_reading = EXCLUDED.hoanh_phi_reading,
			hoanh_phi_meaning = EXCLUDED.hoanh_phi_meaning,
			couplet_upper = EXCLUDED.couplet_upper, couplet_upper_reading = EXCLUDED.couplet_upper_reading,
			couplet_upper_meaning = EXCLUDED.couplet_upper_meaning,
			couplet_lower = EXCLUDED.couplet_lower, couplet_lower_reading = EXCLUDED.couplet_lower_reading,
			couplet_lower_meaning = EXCLUDED.couplet_lower_meaning,
			teaching = EXCLUDED.teaching, updated_at = now()`,
		c.GenealogyID, nullableStringPtr(c.Surname), nullableStringPtr(c.CurrentLocation), nullableInt(c.SinceYear),
		nullableStringPtr(c.Origin), nullableStringPtr(c.History), nullableStringPtr(c.HallName), nullableStringPtr(c.HallAddress),
		nullableInt(c.ChapYear), nullableInt(c.ChapMonth), nullableInt(c.ChapDay), c.ChapEnteredAsLunar,
		nullableStringPtr(c.HoanhPhi), nullableStringPtr(c.HoanhPhiReading), nullableStringPtr(c.HoanhPhiMeaning),
		nullableStringPtr(c.CoupletUpper), nullableStringPtr(c.CoupletUpperReading), nullableStringPtr(c.CoupletUpperMeaning),
		nullableStringPtr(c.CoupletLower), nullableStringPtr(c.CoupletLowerReading), nullableStringPtr(c.CoupletLowerMeaning),
		nullableStringPtr(c.Teaching))
	if err != nil {
		return fmt.Errorf("upsert chronicle: %w", err)
	}
	return expectOneRow(res, domain.ErrGenealogyNotFound)
}

// --- shared helpers for the genealogy repositories --------------------------

// lockGenealogy serialises every write to one tree: Add, Update, Delete, Move
// and Create-invitation all take this row lock first (Step C-2, M-2, D-2, R-2),
// so the limits and the one-clan-head / one-founder invariants hold under
// concurrent writes, and no two transactions lock in opposite orders.
func lockGenealogy(ctx context.Context, q genQuerier, genealogyID int64) error {
	var id int64
	err := q.QueryRowContext(ctx, `SELECT id FROM genealogies WHERE id = $1 FOR UPDATE`, genealogyID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrGenealogyNotFound
	}
	if err != nil {
		return fmt.Errorf("lock genealogy %d: %w", genealogyID, err)
	}
	return nil
}

// insertGenealogyMember inserts m and returns the stored row. m.Generation,
// ParentID, SpouseOfID, OtherParentID and BirthOrder must already be resolved.
func insertGenealogyMember(ctx context.Context, q genQuerier, m *domain.GenealogyMember) (*domain.GenealogyMember, error) {
	row := q.QueryRowContext(ctx, `
		INSERT INTO genealogy_members
			(genealogy_id, kind, gender, full_name, birth_year, is_alive, death_year, residence, dharma_name,
			 tablet_location, burial_place, death_month, death_day, death_entered_as_lunar, phone_number, note,
			 is_clan_head, parent_id, other_parent_id, spouse_of_id, birth_order, generation)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		RETURNING `+genealogyMemberColumns,
		m.GenealogyID, string(m.Kind), string(m.Gender), m.FullName, nullableInt(m.BirthYear), m.IsAlive,
		nullableInt(m.DeathYear), nullableStringPtr(m.Residence), nullableStringPtr(m.DharmaName),
		nullableStringPtr(m.TabletLocation), nullableStringPtr(m.BurialPlace), nullableInt(m.DeathMonth),
		nullableInt(m.DeathDay), nullableBool(m.DeathEnteredAsLunar), nullableStringPtr(m.PhoneNumber),
		nullableStringPtr(m.Note), m.IsClanHead, nullableInt64(m.ParentID), nullableInt64(m.OtherParentID),
		nullableInt64(m.SpouseOfID), nullableInt(m.BirthOrder), m.Generation)
	created, err := scanGenealogyMember(row)
	if err != nil {
		if isUniqueViolation(err, constraintMemberPhone) {
			return nil, domain.ErrMemberPhoneInUse
		}
		return nil, fmt.Errorf("insert genealogy member: %w", err)
	}
	return created, nil
}

func scanGenealogyMember(s rowScanner) (*domain.GenealogyMember, error) {
	var (
		m                                              domain.GenealogyMember
		kind, gender                                   string
		birth, death, dMonth, dDay, birthOrder         sql.NullInt64
		residence, dharma, tablet, burial, phone, note sql.NullString
		lunarFlag                                      sql.NullBool
		parent, other, spouse                          sql.NullInt64
		generation                                     int
	)
	if err := s.Scan(&m.ID, &m.GenealogyID, &kind, &gender, &m.FullName, &birth, &m.IsAlive, &death,
		&residence, &dharma, &tablet, &burial, &dMonth, &dDay, &lunarFlag,
		&phone, &note, &m.IsClanHead, &parent, &other, &spouse, &birthOrder, &generation,
		&m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, err
	}
	m.Kind, m.Gender, m.Generation = domain.MemberKind(kind), domain.Gender(gender), generation
	m.BirthYear, m.DeathYear = intPtr(birth), intPtr(death)
	m.Residence, m.DharmaName, m.TabletLocation, m.BurialPlace = strPtr(residence), strPtr(dharma), strPtr(tablet), strPtr(burial)
	m.DeathMonth, m.DeathDay = intPtr(dMonth), intPtr(dDay)
	if lunarFlag.Valid {
		v := lunarFlag.Bool
		m.DeathEnteredAsLunar = &v
	}
	m.PhoneNumber, m.Note = strPtr(phone), strPtr(note)
	m.ParentID, m.OtherParentID, m.SpouseOfID = int64Ptr(parent), int64Ptr(other), int64Ptr(spouse)
	m.BirthOrder = intPtr(birthOrder)
	return &m, nil
}

func strPtr(n sql.NullString) *string {
	if !n.Valid {
		return nil
	}
	v := n.String
	return &v
}

func intPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

func int64Ptr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

func nullableInt64(i *int64) any {
	if i == nil {
		return nil
	}
	return *i
}

func expectOneRow(res sql.Result, notFound error) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return notFound
	}
	return nil
}

// genValidation is the error a repository returns when a rule that needs the
// rows under lock fails (a cap, a loop, a wrong parent). The use case checked
// the same rules against a read taken earlier; this is the authoritative one.
func genValidation(field, message string) error {
	return &domain.GenealogyValidationError{Field: field, Message: message}
}
