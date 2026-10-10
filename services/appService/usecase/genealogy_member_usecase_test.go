package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

type memFixture struct {
	u       *GenealogyMemberUsecase
	genes   *domain.MockGenealogyRepository
	members *domain.MockGenealogyMemberRepository
	userID  int64
	gid     int64
}

func newMemFixture(t *testing.T, role domain.Role) *memFixture {
	t.Helper()
	f := &memFixture{
		genes:   new(domain.MockGenealogyRepository),
		members: new(domain.MockGenealogyMemberRepository),
		userID:  7,
		gid:     42,
	}
	f.u = NewGenealogyMemberUsecase(f.genes, f.members, genCfg())
	f.u.now = func() time.Time { return time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC) }
	if role != "" {
		f.genes.On("ResolveRole", mock.Anything, f.gid, f.userID).Return(role, nil)
	}
	return f
}

func (f *memFixture) limits() domain.MemberLimits {
	c := genCfg()
	return domain.MemberLimits{MaxMembers: c.MaxMembers, MaxGeneration: c.MaxGeneration, MaxSpouses: c.MaxSpouses}
}

func father() *domain.GenealogyMember {
	m := blood(3, "Cha", domain.GenderMale, 2, i64(1), ip(1), nil)
	m.GenealogyID = 42
	return &m
}

func TestMemberUsecase_Add_Child(t *testing.T) {
	ctx := context.Background()
	const rel = domain.RelationChild

	t.Run("adds a blood child one generation below", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		p := father()
		f.members.On("FindByID", ctx, f.gid, p.ID).Return(p, nil)
		f.members.On("Add", ctx, f.gid, rel, p.ID, mock.MatchedBy(func(m *domain.GenealogyMember) bool {
			return m.Kind == domain.MemberKindBlood && m.FullName == "Con" && m.OtherParentID == nil
		}), f.limits()).Return(&domain.GenealogyMember{
			ID: 50, Kind: domain.MemberKindBlood, Gender: domain.GenderMale, FullName: "Con",
			ParentID: &p.ID, Generation: 3, IsAlive: true,
		}, nil)

		got, err := f.u.Add(ctx, f.userID, f.gid, domain.AddMemberInput{
			Relation: "child", RelatedMemberID: p.ID,
			Member: domain.MemberInfo{FullName: "Con", Gender: male(), IsAlive: trueP()},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 50 || got.InLawLabel != nil {
			t.Errorf("got %+v", got)
		}
		if len(got.Warnings) != 0 {
			t.Errorf("warnings = %v, want none", got.Warnings)
		}
	})

	t.Run("warns, but still saves, when the child is born before the parent", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		p := father()
		p.BirthYear = intP(1950)
		f.members.On("FindByID", ctx, f.gid, p.ID).Return(p, nil)
		f.members.On("Add", ctx, f.gid, rel, p.ID, mock.Anything, f.limits()).Return(&domain.GenealogyMember{
			ID: 50, Kind: domain.MemberKindBlood, Gender: domain.GenderMale, FullName: "Con",
			ParentID: &p.ID, BirthYear: intP(1945), IsAlive: true, Generation: 3,
		}, nil)

		got, err := f.u.Add(ctx, f.userID, f.gid, domain.AddMemberInput{
			Relation: "child", RelatedMemberID: p.ID,
			Member: domain.MemberInfo{FullName: "Con", Gender: male(), IsAlive: trueP(), BirthYear: intP(1945)},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.Warnings) != 1 {
			t.Fatalf("warnings = %v, want 1", got.Warnings)
		}
		w := got.Warnings[0]
		if w.Code != domain.WarningChildBornBeforeParent || w.MemberID != 50 || w.RelatedMemberID != p.ID {
			t.Errorf("warning = %+v", w)
		}
	})

	t.Run("equal birth years do not warn", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		p := father()
		p.BirthYear = intP(1950)
		f.members.On("FindByID", ctx, f.gid, p.ID).Return(p, nil)
		f.members.On("Add", ctx, f.gid, rel, p.ID, mock.Anything, f.limits()).Return(&domain.GenealogyMember{
			ID: 50, Kind: domain.MemberKindBlood, Gender: domain.GenderMale, ParentID: &p.ID,
			BirthYear: intP(1950), IsAlive: true, Generation: 3,
		}, nil)
		got, err := f.u.Add(ctx, f.userID, f.gid, domain.AddMemberInput{
			Relation: "child", RelatedMemberID: p.ID,
			Member: domain.MemberInfo{FullName: "Con", Gender: male(), IsAlive: trueP(), BirthYear: intP(1950)},
		})
		if err != nil || len(got.Warnings) != 0 {
			t.Errorf("err=%v warnings=%v", err, got.Warnings)
		}
	})

	t.Run("other parent must be an in-law spouse of the parent", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		p := father()
		wife := inlaw(6, "Vợ", domain.GenderFemale, 2, p.ID)
		f.members.On("FindByID", ctx, f.gid, p.ID).Return(p, nil)
		f.members.On("FindByID", ctx, f.gid, wife.ID).Return(&wife, nil)
		f.members.On("Add", ctx, f.gid, rel, p.ID, mock.MatchedBy(func(m *domain.GenealogyMember) bool {
			return m.OtherParentID != nil && *m.OtherParentID == wife.ID
		}), f.limits()).Return(&domain.GenealogyMember{ID: 51, Kind: domain.MemberKindBlood, OtherParentID: &wife.ID, Generation: 3}, nil)

		_, err := f.u.Add(ctx, f.userID, f.gid, domain.AddMemberInput{
			Relation: "child", RelatedMemberID: p.ID, OtherParentID: &wife.ID,
			Member: domain.MemberInfo{FullName: "Con", Gender: male(), IsAlive: trueP()},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	rejected := []struct {
		name      string
		setup     func(f *memFixture)
		in        domain.AddMemberInput
		wantField string
		wantErr   error
	}{
		{
			name:      "relation is neither child nor spouse",
			in:        domain.AddMemberInput{Relation: "friend", RelatedMemberID: 3, Member: domain.MemberInfo{FullName: "x", Gender: male(), IsAlive: trueP()}},
			wantField: "relation",
		},
		{
			name: "related member does not exist",
			setup: func(f *memFixture) {
				f.members.On("FindByID", mock.Anything, f.gid, int64(99)).Return(nil, domain.ErrMemberNotFound)
			},
			in:      domain.AddMemberInput{Relation: "child", RelatedMemberID: 99, Member: domain.MemberInfo{FullName: "x", Gender: male(), IsAlive: trueP()}},
			wantErr: domain.ErrMemberNotFound,
		},
		{
			name:    "a non-positive related id is not found",
			in:      domain.AddMemberInput{Relation: "child", RelatedMemberID: 0, Member: domain.MemberInfo{FullName: "x", Gender: male(), IsAlive: trueP()}},
			wantErr: domain.ErrMemberNotFound,
		},
		{
			name: "child of a daughter is not added",
			setup: func(f *memFixture) {
				d := blood(9, "Mai", domain.GenderFemale, 3, i64(3), ip(2), nil)
				f.members.On("FindByID", mock.Anything, f.gid, int64(9)).Return(&d, nil)
			},
			in:        domain.AddMemberInput{Relation: "child", RelatedMemberID: 9, Member: domain.MemberInfo{FullName: "x", Gender: male(), IsAlive: trueP()}},
			wantField: "related_member_id",
		},
		{
			name: "child of an in-law is not added",
			setup: func(f *memFixture) {
				w := inlaw(6, "Vợ", domain.GenderFemale, 2, 3)
				f.members.On("FindByID", mock.Anything, f.gid, int64(6)).Return(&w, nil)
			},
			in:        domain.AddMemberInput{Relation: "child", RelatedMemberID: 6, Member: domain.MemberInfo{FullName: "x", Gender: male(), IsAlive: trueP()}},
			wantField: "related_member_id",
		},
		{
			name: "child would be generation 61",
			setup: func(f *memFixture) {
				d := blood(8, "Sâu", domain.GenderMale, 60, i64(3), ip(1), nil)
				f.members.On("FindByID", mock.Anything, f.gid, int64(8)).Return(&d, nil)
			},
			in:        domain.AddMemberInput{Relation: "child", RelatedMemberID: 8, Member: domain.MemberInfo{FullName: "x", Gender: male(), IsAlive: trueP()}},
			wantField: "related_member_id",
		},
		{
			name: "other parent is not in this tree",
			setup: func(f *memFixture) {
				p := father()
				f.members.On("FindByID", mock.Anything, f.gid, p.ID).Return(p, nil)
				f.members.On("FindByID", mock.Anything, f.gid, int64(77)).Return(nil, domain.ErrMemberNotFound)
			},
			in:        domain.AddMemberInput{Relation: "child", RelatedMemberID: 3, OtherParentID: i64(77), Member: domain.MemberInfo{FullName: "x", Gender: male(), IsAlive: trueP()}},
			wantField: "other_parent_id",
		},
		{
			name: "other parent is a blood member",
			setup: func(f *memFixture) {
				p := father()
				b := blood(4, "Chú", domain.GenderMale, 2, i64(1), ip(2), nil)
				f.members.On("FindByID", mock.Anything, f.gid, p.ID).Return(p, nil)
				f.members.On("FindByID", mock.Anything, f.gid, int64(4)).Return(&b, nil)
			},
			in:        domain.AddMemberInput{Relation: "child", RelatedMemberID: 3, OtherParentID: i64(4), Member: domain.MemberInfo{FullName: "x", Gender: male(), IsAlive: trueP()}},
			wantField: "other_parent_id",
		},
		{
			name: "other parent is the spouse of someone else",
			setup: func(f *memFixture) {
				p := father()
				w := inlaw(6, "Vợ chú", domain.GenderFemale, 2, 4)
				f.members.On("FindByID", mock.Anything, f.gid, p.ID).Return(p, nil)
				f.members.On("FindByID", mock.Anything, f.gid, int64(6)).Return(&w, nil)
			},
			in:        domain.AddMemberInput{Relation: "child", RelatedMemberID: 3, OtherParentID: i64(6), Member: domain.MemberInfo{FullName: "x", Gender: male(), IsAlive: trueP()}},
			wantField: "other_parent_id",
		},
		{
			name: "member information is invalid",
			setup: func(f *memFixture) {
				p := father()
				f.members.On("FindByID", mock.Anything, f.gid, p.ID).Return(p, nil)
			},
			in:        domain.AddMemberInput{Relation: "child", RelatedMemberID: 3, Member: domain.MemberInfo{FullName: "", Gender: male(), IsAlive: trueP()}},
			wantField: "full_name",
		},
	}
	for _, tt := range rejected {
		t.Run(tt.name, func(t *testing.T) {
			f := newMemFixture(t, domain.RoleOwner)
			if tt.setup != nil {
				tt.setup(f)
			}
			_, err := f.u.Add(ctx, f.userID, f.gid, tt.in)
			if err == nil {
				t.Fatal("expected an error")
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("err = %v, want %v", err, tt.wantErr)
				}
			} else if got := fieldOf(t, err); got != tt.wantField {
				t.Errorf("field = %q, want %q", got, tt.wantField)
			}
			f.members.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestMemberUsecase_Add_Spouse(t *testing.T) {
	ctx := context.Background()

	t.Run("adds an in-law of the same generation, gender defaults to the opposite", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		p := father()
		f.members.On("FindByID", ctx, f.gid, p.ID).Return(p, nil)
		f.members.On("Add", ctx, f.gid, domain.RelationSpouse, p.ID, mock.MatchedBy(func(m *domain.GenealogyMember) bool {
			return m.Kind == domain.MemberKindInLaw && m.Gender == domain.GenderFemale
		}), f.limits()).Return(&domain.GenealogyMember{
			ID: 60, Kind: domain.MemberKindInLaw, Gender: domain.GenderFemale, SpouseOfID: &p.ID, Generation: 2, IsAlive: true,
		}, nil)

		got, err := f.u.Add(ctx, f.userID, f.gid, domain.AddMemberInput{
			Relation: "spouse", RelatedMemberID: p.ID,
			Member: domain.MemberInfo{FullName: "Vợ", IsAlive: trueP()},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.InLawLabel == nil || *got.InLawLabel != domain.InLawLabelDau {
			t.Errorf("InLawLabel = %v, want DAU (wife of a male member)", got.InLawLabel)
		}
	})

	t.Run("the husband of a daughter is a son-in-law", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		d := blood(9, "Mai", domain.GenderFemale, 3, i64(3), ip(2), nil)
		f.members.On("FindByID", ctx, f.gid, d.ID).Return(&d, nil)
		f.members.On("Add", ctx, f.gid, domain.RelationSpouse, d.ID, mock.MatchedBy(func(m *domain.GenealogyMember) bool {
			return m.Gender == domain.GenderMale
		}), f.limits()).Return(&domain.GenealogyMember{
			ID: 61, Kind: domain.MemberKindInLaw, Gender: domain.GenderMale, SpouseOfID: &d.ID, Generation: 3, IsAlive: true,
		}, nil)

		got, err := f.u.Add(ctx, f.userID, f.gid, domain.AddMemberInput{
			Relation: "spouse", RelatedMemberID: d.ID,
			Member: domain.MemberInfo{FullName: "Chồng", IsAlive: trueP()},
		})
		if err != nil {
			t.Fatal(err)
		}
		if got.InLawLabel == nil || *got.InLawLabel != domain.InLawLabelRe {
			t.Errorf("InLawLabel = %v, want RE", got.InLawLabel)
		}
	})

	t.Run("spouse of an in-law is rejected", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		w := inlaw(6, "Vợ", domain.GenderFemale, 2, 3)
		f.members.On("FindByID", ctx, f.gid, w.ID).Return(&w, nil)
		_, err := f.u.Add(ctx, f.userID, f.gid, domain.AddMemberInput{
			Relation: "spouse", RelatedMemberID: w.ID, Member: domain.MemberInfo{FullName: "x", IsAlive: trueP()},
		})
		if err == nil || fieldOf(t, err) != "related_member_id" {
			t.Errorf("err = %v, want validation on related_member_id", err)
		}
	})

	t.Run("other_parent_id is only for a child", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		p := father()
		f.members.On("FindByID", ctx, f.gid, p.ID).Return(p, nil)
		_, err := f.u.Add(ctx, f.userID, f.gid, domain.AddMemberInput{
			Relation: "spouse", RelatedMemberID: p.ID, OtherParentID: i64(6),
			Member: domain.MemberInfo{FullName: "x", IsAlive: trueP()},
		})
		if err == nil || fieldOf(t, err) != "other_parent_id" {
			t.Errorf("err = %v, want validation on other_parent_id", err)
		}
	})

	t.Run("a viewer is refused before anything is read", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleViewer)
		_, err := f.u.Add(ctx, f.userID, f.gid, domain.AddMemberInput{Relation: "child", RelatedMemberID: 3})
		if !errors.Is(err, domain.ErrGenealogyForbidden) {
			t.Errorf("err = %v, want ErrGenealogyForbidden", err)
		}
		f.members.AssertNotCalled(t, "FindByID", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestMemberUsecase_Update(t *testing.T) {
	ctx := context.Background()

	// tree: founder 1, Lộc 3 (clan head, has child 7), in-law 10 of 7.
	tree := func() []domain.GenealogyMember {
		root := blood(1, "Tổ", domain.GenderMale, 1, nil, nil, nil)
		loc := blood(3, "Lộc", domain.GenderMale, 2, i64(1), ip(1), nil)
		loc.IsClanHead = true
		kid := blood(7, "A", domain.GenderMale, 3, i64(3), ip(1), nil)
		return []domain.GenealogyMember{root, loc, kid}
	}
	save := func(f *memFixture, returned domain.GenealogyMember) {
		f.members.On("Update", ctx, f.gid, mock.Anything).Return(&returned, nil)
	}

	t.Run("replaces editable fields and keeps the relationships", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		all := tree()
		f.members.On("FindByID", ctx, f.gid, int64(7)).Return(&all[2], nil)
		f.members.On("List", ctx, f.gid).Return(all, nil)
		f.members.On("Update", ctx, f.gid, mock.MatchedBy(func(m *domain.GenealogyMember) bool {
			return m.ID == 7 && m.FullName == "A mới" && m.ParentID != nil && *m.ParentID == 3 &&
				m.Generation == 3 && m.Kind == domain.MemberKindBlood &&
				m.BirthOrder != nil && *m.BirthOrder == 1 // kept: not sent
		})).Return(&domain.GenealogyMember{ID: 7, FullName: "A mới", Kind: domain.MemberKindBlood, ParentID: i64(3), Generation: 3, Gender: domain.GenderMale, IsAlive: true}, nil)

		got, err := f.u.Update(ctx, f.userID, f.gid, 7, domain.MemberInfo{FullName: "A mới", Gender: male(), IsAlive: trueP()})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.FullName != "A mới" {
			t.Errorf("FullName = %q", got.FullName)
		}
	})

	t.Run("absent is_clan_head keeps the flag", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		all := tree()
		f.members.On("FindByID", ctx, f.gid, int64(3)).Return(&all[1], nil)
		f.members.On("List", ctx, f.gid).Return(all, nil)
		f.members.On("Update", ctx, f.gid, mock.MatchedBy(func(m *domain.GenealogyMember) bool {
			return m.IsClanHead
		})).Return(&domain.GenealogyMember{ID: 3, IsClanHead: true, Kind: domain.MemberKindBlood, Gender: domain.GenderMale, IsAlive: true, Generation: 2}, nil)
		if _, err := f.u.Update(ctx, f.userID, f.gid, 3, domain.MemberInfo{FullName: "Lộc", Gender: male(), IsAlive: trueP()}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("explicit false clears the flag", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		all := tree()
		f.members.On("FindByID", ctx, f.gid, int64(3)).Return(&all[1], nil)
		f.members.On("List", ctx, f.gid).Return(all, nil)
		f.members.On("Update", ctx, f.gid, mock.MatchedBy(func(m *domain.GenealogyMember) bool {
			return !m.IsClanHead
		})).Return(&domain.GenealogyMember{ID: 3, Kind: domain.MemberKindBlood, Gender: domain.GenderMale, IsAlive: true, Generation: 2}, nil)
		if _, err := f.u.Update(ctx, f.userID, f.gid, 3, domain.MemberInfo{FullName: "Lộc", Gender: male(), IsAlive: trueP(), IsClanHead: falseP()}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	rejected := []struct {
		name      string
		memberID  int64
		info      domain.MemberInfo
		wantField string
	}{
		{"the clan head dies without clearing the flag", 3, domain.MemberInfo{FullName: "Lộc", Gender: male(), IsAlive: falseP()}, "is_alive"},
		{"the clan head becomes female without clearing the flag", 3, domain.MemberInfo{FullName: "Lộc", Gender: female(), IsAlive: trueP()}, "gender"},
		{"a member with children cannot become female", 7, domain.MemberInfo{FullName: "A", Gender: female(), IsAlive: trueP()}, ""},
		{"founder cannot become female", 1, domain.MemberInfo{FullName: "Tổ", Gender: female(), IsAlive: trueP()}, "gender"},
		{"name is required", 7, domain.MemberInfo{FullName: "", Gender: male(), IsAlive: trueP()}, "full_name"},
		{"gender is required on an edit", 7, domain.MemberInfo{FullName: "A", IsAlive: trueP()}, "gender"},
		{"birth order does not apply to the founder", 1, domain.MemberInfo{FullName: "Tổ", Gender: male(), IsAlive: trueP(), BirthOrder: intP(2)}, "birth_order"},
	}
	for _, tt := range rejected {
		t.Run(tt.name, func(t *testing.T) {
			f := newMemFixture(t, domain.RoleOwner)
			all := tree()
			idx := map[int64]int{1: 0, 3: 1, 7: 2}
			// Give member 7 a child so "cannot become female" applies.
			all = append(all, blood(9, "B", domain.GenderMale, 4, i64(7), ip(1), nil))
			f.members.On("FindByID", ctx, f.gid, tt.memberID).Return(&all[idx[tt.memberID]], nil)
			f.members.On("List", ctx, f.gid).Return(all, nil)

			_, err := f.u.Update(ctx, f.userID, f.gid, tt.memberID, tt.info)
			if err == nil {
				t.Fatal("expected a validation error")
			}
			want := tt.wantField
			if want == "" {
				want = "gender"
			}
			if got := fieldOf(t, err); got != want {
				t.Errorf("field = %q, want %q", got, want)
			}
			f.members.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
		})
	}

	t.Run("a viewer is refused", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleViewer)
		_, err := f.u.Update(ctx, f.userID, f.gid, 7, domain.MemberInfo{FullName: "A", Gender: male(), IsAlive: trueP()})
		if !errors.Is(err, domain.ErrGenealogyForbidden) {
			t.Errorf("err = %v, want ErrGenealogyForbidden", err)
		}
	})

	t.Run("an unknown member is not found", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		f.members.On("FindByID", ctx, f.gid, int64(99)).Return(nil, domain.ErrMemberNotFound)
		_, err := f.u.Update(ctx, f.userID, f.gid, 99, domain.MemberInfo{FullName: "A", Gender: male(), IsAlive: trueP()})
		if !errors.Is(err, domain.ErrMemberNotFound) {
			t.Errorf("err = %v, want ErrMemberNotFound", err)
		}
	})

	t.Run("a phone number already used is surfaced", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		all := tree()
		f.members.On("FindByID", ctx, f.gid, int64(7)).Return(&all[2], nil)
		f.members.On("List", ctx, f.gid).Return(all, nil)
		f.members.On("Update", ctx, f.gid, mock.Anything).Return(nil, domain.ErrMemberPhoneInUse)
		_, err := f.u.Update(ctx, f.userID, f.gid, 7, domain.MemberInfo{FullName: "A", Gender: male(), IsAlive: trueP(), PhoneNumber: strP("0912345678")})
		if !errors.Is(err, domain.ErrMemberPhoneInUse) {
			t.Errorf("err = %v, want ErrMemberPhoneInUse", err)
		}
	})
	_ = save
}

func TestMemberUsecase_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("owner deletes", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		f.members.On("Delete", ctx, f.gid, int64(7)).Return(nil)
		if err := f.u.Delete(ctx, f.userID, f.gid, 7); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, e := range []error{domain.ErrMemberHasDescendants, domain.ErrFounderNotDeletable, domain.ErrMemberNotFound} {
		t.Run("passes through "+e.Error(), func(t *testing.T) {
			f := newMemFixture(t, domain.RoleOwner)
			f.members.On("Delete", ctx, f.gid, int64(7)).Return(e)
			if err := f.u.Delete(ctx, f.userID, f.gid, 7); !errors.Is(err, e) {
				t.Errorf("err = %v, want %v", err, e)
			}
		})
	}

	t.Run("a viewer is refused", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleViewer)
		if err := f.u.Delete(ctx, f.userID, f.gid, 7); !errors.Is(err, domain.ErrGenealogyForbidden) {
			t.Errorf("err = %v, want ErrGenealogyForbidden", err)
		}
		f.members.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("a non-positive member id is not found", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		if err := f.u.Delete(ctx, f.userID, f.gid, 0); !errors.Is(err, domain.ErrMemberNotFound) {
			t.Errorf("err = %v, want ErrMemberNotFound", err)
		}
	})
}

func TestMemberUsecase_Move(t *testing.T) {
	ctx := context.Background()

	t.Run("moves and warns about the new parent's birth year", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		in := domain.MoveMemberInput{Relation: "child", RelatedMemberID: 4}
		parent := blood(4, "Chú", domain.GenderMale, 2, i64(1), ip(2), nil)
		parent.BirthYear = intP(1960)
		kid := blood(7, "A", domain.GenderMale, 3, i64(4), ip(1), nil)
		kid.BirthYear = intP(1950)
		f.members.On("Move", ctx, f.gid, int64(7), in, f.limits()).Return(&kid, nil)
		f.members.On("List", ctx, f.gid).Return([]domain.GenealogyMember{blood(1, "Tổ", domain.GenderMale, 1, nil, nil, nil), parent, kid}, nil)

		got, err := f.u.Move(ctx, f.userID, f.gid, 7, in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.Warnings) != 1 || got.Warnings[0].MemberID != 7 || got.Warnings[0].RelatedMemberID != 4 {
			t.Errorf("warnings = %+v", got.Warnings)
		}
	})

	t.Run("rejects an unknown relation without reading anything", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		_, err := f.u.Move(ctx, f.userID, f.gid, 7, domain.MoveMemberInput{Relation: "sibling", RelatedMemberID: 4})
		if err == nil || fieldOf(t, err) != "relation" {
			t.Errorf("err = %v, want validation on relation", err)
		}
		f.members.AssertNotCalled(t, "Move", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("rejects an invalid birth order", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		_, err := f.u.Move(ctx, f.userID, f.gid, 7, domain.MoveMemberInput{Relation: "child", RelatedMemberID: 4, BirthOrder: intP(0)})
		if err == nil || fieldOf(t, err) != "birth_order" {
			t.Errorf("err = %v, want validation on birth_order", err)
		}
	})

	t.Run("a viewer is refused", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleViewer)
		_, err := f.u.Move(ctx, f.userID, f.gid, 7, domain.MoveMemberInput{Relation: "child", RelatedMemberID: 4})
		if !errors.Is(err, domain.ErrGenealogyForbidden) {
			t.Errorf("err = %v, want ErrGenealogyForbidden", err)
		}
	})

	t.Run("the founder cannot be moved", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		in := domain.MoveMemberInput{Relation: "child", RelatedMemberID: 4}
		f.members.On("Move", ctx, f.gid, int64(1), in, f.limits()).Return(nil, domain.ErrFounderNotMovable)
		_, err := f.u.Move(ctx, f.userID, f.gid, 1, in)
		if !errors.Is(err, domain.ErrFounderNotMovable) {
			t.Errorf("err = %v, want ErrFounderNotMovable", err)
		}
	})
}

func TestMemberUsecase_ListAndGet(t *testing.T) {
	ctx := context.Background()

	t.Run("a viewer lists every member with labels and lunar anniversaries", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleViewer)
		root := blood(1, "Tổ", domain.GenderMale, 1, nil, nil, nil)
		root.IsAlive = false
		root.DeathYear = intP(2024)
		root.DeathMonth, root.DeathDay = intP(2), intP(10)
		root.DeathEnteredAsLunar = falseP()
		wife := inlaw(2, "Bà", domain.GenderFemale, 1, 1)
		f.members.On("List", ctx, f.gid).Return([]domain.GenealogyMember{root, wife}, nil)

		got, err := f.u.List(ctx, f.userID, f.gid)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		if got[0].DeathLunar == nil || got[0].DeathLunar.Day != 1 || got[0].DeathLunar.Month != 1 {
			t.Errorf("DeathLunar = %+v, want 1/1 (10 Feb 2024 was Tết)", got[0].DeathLunar)
		}
		if got[1].InLawLabel == nil || *got[1].InLawLabel != domain.InLawLabelDau {
			t.Errorf("InLawLabel = %v, want DAU", got[1].InLawLabel)
		}
	})

	t.Run("an empty tree is an empty list, not nil", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleOwner)
		f.members.On("List", ctx, f.gid).Return(nil, nil)
		got, err := f.u.List(ctx, f.userID, f.gid)
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("got %#v, %v", got, err)
		}
	})

	t.Run("Get of a member of another tree is not found", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleViewer)
		f.members.On("FindByID", ctx, f.gid, int64(5)).Return(nil, domain.ErrMemberNotFound)
		if _, err := f.u.Get(ctx, f.userID, f.gid, 5); !errors.Is(err, domain.ErrMemberNotFound) {
			t.Errorf("err = %v, want ErrMemberNotFound", err)
		}
	})

	t.Run("Get of an in-law carries the label from the attached member", func(t *testing.T) {
		f := newMemFixture(t, domain.RoleViewer)
		husband := blood(9, "Mai", domain.GenderFemale, 3, i64(3), ip(2), nil)
		hub := inlaw(17, "Chồng", domain.GenderMale, 3, husband.ID)
		f.members.On("FindByID", ctx, f.gid, hub.ID).Return(&hub, nil)
		f.members.On("FindByID", ctx, f.gid, husband.ID).Return(&husband, nil)
		got, err := f.u.Get(ctx, f.userID, f.gid, hub.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.InLawLabel == nil || *got.InLawLabel != domain.InLawLabelRe {
			t.Errorf("InLawLabel = %v, want RE", got.InLawLabel)
		}
	})

	t.Run("a stranger is not found", func(t *testing.T) {
		f := newMemFixture(t, "")
		f.genes.On("ResolveRole", mock.Anything, f.gid, f.userID).Return(domain.Role(""), domain.ErrGenealogyNotFound)
		if _, err := f.u.List(ctx, f.userID, f.gid); !errors.Is(err, domain.ErrGenealogyNotFound) {
			t.Errorf("err = %v, want ErrGenealogyNotFound", err)
		}
	})
}
