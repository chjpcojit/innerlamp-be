package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// GenealogyMemberUsecase manages the people of a tree: list, open, add, edit,
// delete and re-attach (US-GEN-03, 04, 07, 10). Reads are open to the owner and
// to viewers; every write is the owner's (AC81).
type GenealogyMemberUsecase struct {
	genealogies domain.GenealogyRepository
	members     domain.GenealogyMemberRepository
	cfg         config.GenealogyConfig
	now         func() time.Time
}

// NewGenealogyMemberUsecase wires the use case.
func NewGenealogyMemberUsecase(
	genealogies domain.GenealogyRepository,
	members domain.GenealogyMemberRepository,
	cfg config.GenealogyConfig,
) *GenealogyMemberUsecase {
	return &GenealogyMemberUsecase{genealogies: genealogies, members: members, cfg: cfg, now: time.Now}
}

func (u *GenealogyMemberUsecase) limits() domain.MemberLimits {
	return domain.MemberLimits{
		MaxMembers:    u.cfg.MaxMembers,
		MaxGeneration: u.cfg.MaxGeneration,
		MaxSpouses:    u.cfg.MaxSpouses,
	}
}

// List returns every member of the tree (AC28). The tree is flat: the client
// builds it from parent_id and spouse_of_id.
func (u *GenealogyMemberUsecase) List(ctx context.Context, userID, genealogyID int64) ([]domain.MemberView, error) {
	if _, err := resolveRole(ctx, u.genealogies, userID, genealogyID); err != nil {
		return nil, err
	}
	list, err := u.members.List(ctx, genealogyID)
	if err != nil {
		return nil, err
	}
	idx := BuildIndex(list)
	out := make([]domain.MemberView, 0, len(list))
	for i := range list {
		out = append(out, viewOf(idx, list[i]))
	}
	return out, nil
}

// Get returns one member of the tree (AC29).
func (u *GenealogyMemberUsecase) Get(ctx context.Context, userID, genealogyID, memberID int64) (*domain.MemberView, error) {
	if _, err := resolveRole(ctx, u.genealogies, userID, genealogyID); err != nil {
		return nil, err
	}
	if memberID <= 0 {
		return nil, domain.ErrMemberNotFound
	}
	m, err := u.members.FindByID(ctx, genealogyID, memberID)
	if err != nil {
		return nil, err
	}
	v := domain.MemberView{GenealogyMember: *m}
	attachDeathLunar(&v.GenealogyMember)
	if m.Kind == domain.MemberKindInLaw && m.SpouseOfID != nil {
		a, err := u.members.FindByID(ctx, genealogyID, *m.SpouseOfID)
		if err != nil && !errors.Is(err, domain.ErrMemberNotFound) {
			return nil, err
		}
		if err == nil {
			v.InLawLabel = labelFor(a.Gender)
		}
	}
	return &v, nil
}

// Add attaches a new member as a child or a spouse of an existing one
// (AC18–AC27, AC55–AC57, AC60, AC82). Owner only.
func (u *GenealogyMemberUsecase) Add(ctx context.Context, userID, genealogyID int64, in domain.AddMemberInput) (*domain.MemberResult, error) {
	if err := requireOwner(ctx, u.genealogies, userID, genealogyID); err != nil {
		return nil, err
	}
	relation := domain.MemberRelation(in.Relation)
	if !relation.Valid() {
		return nil, genInvalid("relation", "relation must be child or spouse")
	}
	if in.RelatedMemberID <= 0 {
		return nil, domain.ErrMemberNotFound
	}
	related, err := u.members.FindByID(ctx, genealogyID, in.RelatedMemberID)
	if err != nil {
		return nil, err
	}

	var (
		kind          domain.MemberKind
		defaultGender *domain.Gender
	)
	switch relation {
	case domain.RelationChild:
		if related.Kind != domain.MemberKindBlood {
			return nil, genInvalid("related_member_id", "a child is attached to a member of the clan, not to an in-law")
		}
		if related.Gender != domain.GenderMale {
			return nil, genInvalid("related_member_id", "children of a female member are not part of this tree")
		}
		if related.Generation+1 > u.cfg.MaxGeneration {
			return nil, genInvalid("related_member_id", fmt.Sprintf("the tree cannot go deeper than %d generations", u.cfg.MaxGeneration))
		}
		kind = domain.MemberKindBlood
	case domain.RelationSpouse:
		if related.Kind != domain.MemberKindBlood {
			return nil, genInvalid("related_member_id", "a spouse is attached to a member of the clan, not to an in-law")
		}
		if in.OtherParentID != nil {
			return nil, genInvalid("other_parent_id", "other_parent_id only applies to a child")
		}
		kind = domain.MemberKindInLaw
		opposite := oppositeGender(related.Gender)
		defaultGender = &opposite
	}

	m, err := buildMember(in.Member, memberCtx{kind: kind, currentYear: currentYearVN(u.now())}, defaultGender)
	if err != nil {
		return nil, err
	}

	if relation == domain.RelationChild && in.OtherParentID != nil {
		other, err := u.members.FindByID(ctx, genealogyID, *in.OtherParentID)
		if errors.Is(err, domain.ErrMemberNotFound) {
			return nil, genInvalid("other_parent_id", "other_parent_id is not a member of this tree")
		}
		if err != nil {
			return nil, err
		}
		if other.Kind != domain.MemberKindInLaw || other.SpouseOfID == nil || *other.SpouseOfID != related.ID {
			return nil, genInvalid("other_parent_id", "other_parent_id must be a spouse of the parent")
		}
		m.OtherParentID = in.OtherParentID
	}

	created, err := u.members.Add(ctx, genealogyID, relation, related.ID, m, u.limits())
	if err != nil {
		return nil, err
	}

	res := &domain.MemberResult{MemberView: domain.MemberView{GenealogyMember: *created}, Warnings: []domain.MemberWarning{}}
	attachDeathLunar(&res.GenealogyMember)
	if created.Kind == domain.MemberKindInLaw {
		res.InLawLabel = labelFor(related.Gender)
	}
	if relation == domain.RelationChild {
		if w, ok := birthBeforeParent(created, related); ok {
			res.Warnings = append(res.Warnings, w)
		}
	}
	return res, nil
}

// Update replaces the editable fields of a member (AC30–AC33, AC55, AC60,
// AC82). It never changes relationships (AC31). Owner only.
func (u *GenealogyMemberUsecase) Update(ctx context.Context, userID, genealogyID, memberID int64, info domain.MemberInfo) (*domain.MemberResult, error) {
	if err := requireOwner(ctx, u.genealogies, userID, genealogyID); err != nil {
		return nil, err
	}
	if memberID <= 0 {
		return nil, domain.ErrMemberNotFound
	}
	cur, err := u.members.FindByID(ctx, genealogyID, memberID)
	if err != nil {
		return nil, err
	}

	m, err := buildMember(info, memberCtx{
		kind: cur.Kind, isFounder: cur.IsFounder(), currentYear: currentYearVN(u.now()),
	}, nil)
	if err != nil {
		return nil, err
	}

	// The clan-head flag (AC82): absent keeps it, but it must still hold for the
	// member as edited — a clan head who is now female or deceased is refused
	// rather than silently unseated.
	switch {
	case info.IsClanHead == nil:
		if cur.IsClanHead {
			if m.Gender != domain.GenderMale {
				return nil, genInvalid("gender", "the clan head must be male: clear is_clan_head first")
			}
			if !m.IsAlive {
				return nil, genInvalid("is_alive", "the clan head must be alive: clear is_clan_head first")
			}
			m.IsClanHead = true
		}
	case *info.IsClanHead:
		m.IsClanHead = true // validated by buildMember
	default:
		m.IsClanHead = false
	}

	all, err := u.members.List(ctx, genealogyID)
	if err != nil {
		return nil, err
	}
	idx := BuildIndex(all)

	// Only a male member can father children, and the founder must stay male
	// (AC4, AC19, AC30).
	if cur.Gender == domain.GenderMale && m.Gender == domain.GenderFemale {
		if cur.IsFounder() {
			return nil, genInvalid("gender", "the founder must be male")
		}
		if len(idx.children[cur.ID]) > 0 {
			return nil, genInvalid("gender", "a member with children must stay male")
		}
	}

	m.ID, m.GenealogyID, m.Kind = cur.ID, cur.GenealogyID, cur.Kind
	m.ParentID, m.OtherParentID, m.SpouseOfID = cur.ParentID, cur.OtherParentID, cur.SpouseOfID
	m.Generation, m.CreatedAt = cur.Generation, cur.CreatedAt
	if m.BirthOrder == nil {
		m.BirthOrder = cur.BirthOrder
	}

	saved, err := u.members.Update(ctx, genealogyID, m)
	if err != nil {
		return nil, err
	}
	return resultOf(idx, saved), nil
}

// Delete removes a blood member with no children together with its in-laws, or
// a single in-law (AC34–AC37). The repository decides inside its transaction,
// so a child added a moment ago is not missed. Owner only.
func (u *GenealogyMemberUsecase) Delete(ctx context.Context, userID, genealogyID, memberID int64) error {
	if err := requireOwner(ctx, u.genealogies, userID, genealogyID); err != nil {
		return err
	}
	if memberID <= 0 {
		return domain.ErrMemberNotFound
	}
	return u.members.Delete(ctx, genealogyID, memberID)
}

// Move re-attaches a member under another parent or spouse-holder and shifts
// the generation of its whole branch (AC76–AC80). Owner only.
func (u *GenealogyMemberUsecase) Move(ctx context.Context, userID, genealogyID, memberID int64, in domain.MoveMemberInput) (*domain.MemberResult, error) {
	if err := requireOwner(ctx, u.genealogies, userID, genealogyID); err != nil {
		return nil, err
	}
	if memberID <= 0 || in.RelatedMemberID <= 0 {
		return nil, domain.ErrMemberNotFound
	}
	if !domain.MemberRelation(in.Relation).Valid() {
		return nil, genInvalid("relation", "relation must be child or spouse")
	}
	if in.BirthOrder != nil && (*in.BirthOrder < 1 || *in.BirthOrder > genBirthOrderMax) {
		return nil, genInvalid("birth_order", fmt.Sprintf("birth_order must be between 1 and %d", genBirthOrderMax))
	}
	if in.Relation == string(domain.RelationSpouse) && in.OtherParentID != nil {
		return nil, genInvalid("other_parent_id", "other_parent_id only applies to a child")
	}

	moved, err := u.members.Move(ctx, genealogyID, memberID, in, u.limits())
	if err != nil {
		return nil, err
	}
	all, err := u.members.List(ctx, genealogyID)
	if err != nil {
		return nil, err
	}
	return resultOf(BuildIndex(all), moved), nil
}

// --- helpers ----------------------------------------------------------------

func oppositeGender(g domain.Gender) domain.Gender {
	if g == domain.GenderMale {
		return domain.GenderFemale
	}
	return domain.GenderMale
}

// labelFor is DAU for an in-law of a male member, RE for one of a female (AC33).
func labelFor(attached domain.Gender) *domain.InLawLabel {
	l := domain.InLawLabelDau
	if attached == domain.GenderFemale {
		l = domain.InLawLabelRe
	}
	return &l
}

// attachDeathLunar derives the lunar anniversary from the stored solar one
// (AC55).
func attachDeathLunar(m *domain.GenealogyMember) {
	m.DeathLunar = nil
	if m.DeathYear == nil || m.DeathMonth == nil || m.DeathDay == nil {
		return
	}
	if l, ok := SolarToLunar(*m.DeathYear, *m.DeathMonth, *m.DeathDay); ok {
		m.DeathLunar = &l
	}
}

func viewOf(idx *TreeIndex, m domain.GenealogyMember) domain.MemberView {
	v := domain.MemberView{GenealogyMember: m}
	attachDeathLunar(&v.GenealogyMember)
	v.InLawLabel = InLawLabel(idx, &m)
	return v
}

// resultOf builds a write result from the stored member and a tree snapshot
// taken for the same write: label, lunar anniversary and warnings.
func resultOf(idx *TreeIndex, saved *domain.GenealogyMember) *domain.MemberResult {
	res := &domain.MemberResult{MemberView: viewOf(idx, *saved), Warnings: []domain.MemberWarning{}}
	if saved.Kind != domain.MemberKindBlood {
		return res
	}
	if saved.ParentID != nil {
		if parent, ok := idx.Get(*saved.ParentID); ok {
			if w, bad := birthBeforeParent(saved, parent); bad {
				res.Warnings = append(res.Warnings, w)
			}
		}
	}
	for _, c := range idx.children[saved.ID] {
		if w, bad := birthBeforeParent(c, saved); bad {
			res.Warnings = append(res.Warnings, w)
		}
	}
	return res
}

// birthBeforeParent reports the advisory warning of AC27: a child whose birth
// year is earlier than the parent's. Equal years and missing years never warn.
func birthBeforeParent(child, parent *domain.GenealogyMember) (domain.MemberWarning, bool) {
	if child.BirthYear == nil || parent.BirthYear == nil || *child.BirthYear >= *parent.BirthYear {
		return domain.MemberWarning{}, false
	}
	return domain.MemberWarning{
		Code:            domain.WarningChildBornBeforeParent,
		MemberID:        child.ID,
		RelatedMemberID: parent.ID,
	}, true
}
