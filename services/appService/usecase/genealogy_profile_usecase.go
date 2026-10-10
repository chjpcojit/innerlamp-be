package usecase

import (
	"context"
	"time"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// GenealogyProfileUsecase computes the profile of one selected member:
// age, relatives, line of descent, branch and relationship to "me"
// (US-GEN-09, AC64–AC69). The whole tree (at most a few thousand rows) is
// loaded once and everything is computed in memory by the pure functions in
// genealogy_kinship.go.
type GenealogyProfileUsecase struct {
	genealogies domain.GenealogyRepository
	members     domain.GenealogyMemberRepository
	phones      domain.UserPhoneLookup
	now         func() time.Time
}

// NewGenealogyProfileUsecase wires the use case.
func NewGenealogyProfileUsecase(
	genealogies domain.GenealogyRepository,
	members domain.GenealogyMemberRepository,
	phones domain.UserPhoneLookup,
) *GenealogyProfileUsecase {
	return &GenealogyProfileUsecase{genealogies: genealogies, members: members, phones: phones, now: time.Now}
}

// GetProfile returns the computed profile of memberID for the owner or a
// viewer. "Me" is the member whose phone number equals the caller's own
// (AC59); with no such member there is no relation to me (AC69).
func (u *GenealogyProfileUsecase) GetProfile(ctx context.Context, userID, genealogyID, memberID int64) (*domain.MemberProfile, error) {
	if _, err := resolveRole(ctx, u.genealogies, userID, genealogyID); err != nil {
		return nil, err
	}
	if memberID <= 0 {
		return nil, domain.ErrMemberNotFound
	}
	phone, err := u.phones.FindPhoneByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	all, err := u.members.List(ctx, genealogyID)
	if err != nil {
		return nil, err
	}
	idx := BuildIndex(all)
	m, ok := idx.Get(memberID)
	if !ok {
		return nil, domain.ErrMemberNotFound
	}

	p := &domain.MemberProfile{Member: viewOf(idx, *m)}
	p.Age = Age(m, currentYearVN(u.now()))

	rel, _ := RelativesOf(idx, memberID)
	p.Father, p.Mother = rel.Father, rel.Mother
	p.Spouses, p.Siblings, p.Children = rel.Spouses, rel.Siblings, rel.Children

	if lineage, ok := LineageOf(idx, memberID); ok {
		p.Lineage = lineage
	} else {
		p.Lineage = []domain.LineageItem{}
	}
	p.Chi, p.ChiPath, _ = ChiOf(idx, memberID)
	if p.ChiPath == nil {
		p.ChiPath = []domain.ChiItem{}
	}

	if phone != "" {
		for i := range all {
			if all[i].PhoneNumber != nil && *all[i].PhoneNumber == phone {
				id := all[i].ID
				p.MyMemberID = &id
				break
			}
		}
	}
	if p.MyMemberID != nil {
		k := KinshipOf(idx, *p.MyMemberID, memberID)
		p.RelationToMe = &k
	}
	return p, nil
}
