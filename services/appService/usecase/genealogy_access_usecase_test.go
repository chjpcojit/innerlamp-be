package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/pkg/validation"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type accessFixture struct {
	u           *GenealogyAccessUsecase
	genes       *domain.MockGenealogyRepository
	viewers     *domain.MockViewerRepository
	invitations *domain.MockInvitationRepository
	userID      int64
	gid         int64
}

func newAccessFixture(t *testing.T, role domain.Role) *accessFixture {
	t.Helper()
	f := &accessFixture{
		genes:       new(domain.MockGenealogyRepository),
		viewers:     new(domain.MockViewerRepository),
		invitations: new(domain.MockInvitationRepository),
		userID:      7,
		gid:         42,
	}
	f.u = NewGenealogyAccessUsecase(f.genes, f.viewers, f.invitations, genCfg())
	if role != "" {
		f.genes.On("ResolveRole", mock.Anything, f.gid, f.userID).Return(role, nil)
	}
	return f
}

func TestAccessUsecase_Invite(t *testing.T) {
	ctx := context.Background()
	ttl := 30 * 24 * time.Hour
	phone := validation.NormalizePhone("0912345678")

	t.Run("creates a pending invitation with the normalised number and configured limits", func(t *testing.T) {
		f := newAccessFixture(t, domain.RoleOwner)
		want := domain.InviteInput{MemberID: 11, PhoneNumber: phone}
		f.invitations.On("Create", ctx, f.gid, f.userID, want, 20, ttl).
			Return(&domain.Invitation{ID: 1, MemberID: 11, PhoneNumber: phone, Status: domain.InvitationPending}, nil)

		got, err := f.u.Invite(ctx, f.userID, f.gid, domain.InviteInput{MemberID: 11, PhoneNumber: " 0912345678 "})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Status != domain.InvitationPending {
			t.Errorf("Status = %q, want pending", got.Status)
		}
		f.invitations.AssertExpectations(t)
	})

	t.Run("the number may be omitted when the member already has one", func(t *testing.T) {
		f := newAccessFixture(t, domain.RoleOwner)
		f.invitations.On("Create", ctx, f.gid, f.userID, domain.InviteInput{MemberID: 11}, 20, ttl).
			Return(&domain.Invitation{ID: 1}, nil)
		if _, err := f.u.Invite(ctx, f.userID, f.gid, domain.InviteInput{MemberID: 11}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	invalid := []struct {
		name      string
		in        domain.InviteInput
		wantField string
	}{
		{"member_id missing", domain.InviteInput{PhoneNumber: "0912345678"}, "member_id"},
		{"member_id negative", domain.InviteInput{MemberID: -3, PhoneNumber: "0912345678"}, "member_id"},
		{"phone not a number", domain.InviteInput{MemberID: 11, PhoneNumber: "abc"}, "phone_number"},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			f := newAccessFixture(t, domain.RoleOwner)
			_, err := f.u.Invite(ctx, f.userID, f.gid, tt.in)
			if err == nil || fieldOf(t, err) != tt.wantField {
				t.Errorf("err = %v, want validation on %s", err, tt.wantField)
			}
			f.invitations.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}

	t.Run("repository outcomes are passed through", func(t *testing.T) {
		for _, e := range []error{
			domain.ErrMemberNotFound, domain.ErrViewerExists, domain.ErrInvitationAlreadyPending,
			domain.ErrViewerLimitReached, domain.ErrMemberPhoneInUse,
		} {
			f := newAccessFixture(t, domain.RoleOwner)
			f.invitations.On("Create", ctx, f.gid, f.userID, mock.Anything, 20, ttl).Return(nil, e)
			if _, err := f.u.Invite(ctx, f.userID, f.gid, domain.InviteInput{MemberID: 11, PhoneNumber: phone}); !errors.Is(err, e) {
				t.Errorf("err = %v, want %v", err, e)
			}
		}
	})

	t.Run("a viewer is refused", func(t *testing.T) {
		f := newAccessFixture(t, domain.RoleViewer)
		_, err := f.u.Invite(ctx, f.userID, f.gid, domain.InviteInput{MemberID: 11, PhoneNumber: phone})
		if !errors.Is(err, domain.ErrGenealogyForbidden) {
			t.Errorf("err = %v, want ErrGenealogyForbidden", err)
		}
	})
}

func TestAccessUsecase_ListsAndCancel(t *testing.T) {
	ctx := context.Background()

	t.Run("owner lists viewers", func(t *testing.T) {
		f := newAccessFixture(t, domain.RoleOwner)
		f.viewers.On("List", ctx, f.gid).Return([]domain.Viewer{{UserID: 9, MemberID: i64(11)}}, nil)
		got, err := f.u.ListViewers(ctx, f.userID, f.gid)
		if err != nil || len(got) != 1 || got[0].MemberID == nil {
			t.Fatalf("got %+v, %v", got, err)
		}
	})

	t.Run("a viewer cannot list viewers", func(t *testing.T) {
		f := newAccessFixture(t, domain.RoleViewer)
		if _, err := f.u.ListViewers(ctx, f.userID, f.gid); !errors.Is(err, domain.ErrGenealogyForbidden) {
			t.Errorf("err = %v, want ErrGenealogyForbidden", err)
		}
	})

	t.Run("owner lists invitations, empty is not nil", func(t *testing.T) {
		f := newAccessFixture(t, domain.RoleOwner)
		f.invitations.On("ListByGenealogy", ctx, f.gid).Return(nil, nil)
		got, err := f.u.ListInvitations(ctx, f.userID, f.gid)
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("got %#v, %v", got, err)
		}
	})

	t.Run("owner cancels", func(t *testing.T) {
		f := newAccessFixture(t, domain.RoleOwner)
		f.invitations.On("Cancel", ctx, f.gid, int64(5)).Return(nil)
		if err := f.u.CancelInvitation(ctx, f.userID, f.gid, 5); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("cancelling a non-positive id is not found", func(t *testing.T) {
		f := newAccessFixture(t, domain.RoleOwner)
		if err := f.u.CancelInvitation(ctx, f.userID, f.gid, 0); !errors.Is(err, domain.ErrInvitationNotFound) {
			t.Errorf("err = %v, want ErrInvitationNotFound", err)
		}
	})

	t.Run("a viewer cannot cancel", func(t *testing.T) {
		f := newAccessFixture(t, domain.RoleViewer)
		if err := f.u.CancelInvitation(ctx, f.userID, f.gid, 5); !errors.Is(err, domain.ErrGenealogyForbidden) {
			t.Errorf("err = %v, want ErrGenealogyForbidden", err)
		}
	})
}

func TestAccessUsecase_RemoveViewer(t *testing.T) {
	ctx := context.Background()

	t.Run("owner removes a viewer", func(t *testing.T) {
		f := newAccessFixture(t, domain.RoleOwner)
		f.viewers.On("Remove", ctx, f.gid, int64(9)).Return(nil)
		if err := f.u.RemoveViewer(ctx, f.userID, f.gid, 9); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("a viewer leaves by removing themself", func(t *testing.T) {
		f := newAccessFixture(t, domain.RoleViewer)
		f.viewers.On("Remove", ctx, f.gid, f.userID).Return(nil)
		if err := f.u.RemoveViewer(ctx, f.userID, f.gid, f.userID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("a viewer cannot remove someone else", func(t *testing.T) {
		f := newAccessFixture(t, domain.RoleViewer)
		err := f.u.RemoveViewer(ctx, f.userID, f.gid, 99)
		if !errors.Is(err, domain.ErrGenealogyForbidden) {
			t.Errorf("err = %v, want ErrGenealogyForbidden", err)
		}
		f.viewers.AssertNotCalled(t, "Remove", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("the owner cannot leave through this action", func(t *testing.T) {
		f := newAccessFixture(t, domain.RoleOwner)
		err := f.u.RemoveViewer(ctx, f.userID, f.gid, f.userID)
		if !errors.Is(err, domain.ErrViewerNotFound) {
			t.Errorf("err = %v, want ErrViewerNotFound", err)
		}
		f.viewers.AssertNotCalled(t, "Remove", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("a user who is not a viewer is not found", func(t *testing.T) {
		f := newAccessFixture(t, domain.RoleOwner)
		f.viewers.On("Remove", ctx, f.gid, int64(9)).Return(domain.ErrViewerNotFound)
		if err := f.u.RemoveViewer(ctx, f.userID, f.gid, 9); !errors.Is(err, domain.ErrViewerNotFound) {
			t.Errorf("err = %v, want ErrViewerNotFound", err)
		}
	})
}

// --- invitee side ------------------------------------------------------------

type inviteeFixture struct {
	u           *GenealogyInvitationUsecase
	invitations *domain.MockInvitationRepository
	phones      *domain.MockUserPhoneLookup
	userID      int64
}

func newInviteeFixture(t *testing.T) *inviteeFixture {
	t.Helper()
	f := &inviteeFixture{
		invitations: new(domain.MockInvitationRepository),
		phones:      new(domain.MockUserPhoneLookup),
		userID:      9,
	}
	f.u = NewGenealogyInvitationUsecase(f.invitations, f.phones, genCfg())
	return f
}

func TestInvitationUsecase(t *testing.T) {
	ctx := context.Background()

	t.Run("lists the invitations of the caller's own number", func(t *testing.T) {
		f := newInviteeFixture(t)
		f.phones.On("FindPhoneByID", ctx, f.userID).Return("+84900000009", nil)
		f.invitations.On("ListMine", ctx, "+84900000009").Return([]domain.Invitation{{ID: 1, GenealogyName: "Họ Nguyễn", MemberFullName: "B"}}, nil)
		got, err := f.u.ListMine(ctx, f.userID)
		if err != nil || len(got) != 1 {
			t.Fatalf("got %+v, %v", got, err)
		}
	})

	t.Run("an empty inbox is an empty list", func(t *testing.T) {
		f := newInviteeFixture(t)
		f.phones.On("FindPhoneByID", ctx, f.userID).Return("+84900000009", nil)
		f.invitations.On("ListMine", ctx, "+84900000009").Return(nil, nil)
		got, err := f.u.ListMine(ctx, f.userID)
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("got %#v, %v", got, err)
		}
	})

	t.Run("accept uses the caller's own number and the configured cap", func(t *testing.T) {
		f := newInviteeFixture(t)
		f.phones.On("FindPhoneByID", ctx, f.userID).Return("+84900000009", nil)
		f.invitations.On("Accept", ctx, int64(5), f.userID, "+84900000009", 20).
			Return(&domain.AcceptResult{GenealogyID: 42, MemberID: 11, Role: domain.RoleViewer}, nil)
		got, err := f.u.Accept(ctx, f.userID, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.GenealogyID != 42 || got.MemberID != 11 || got.Role != domain.RoleViewer {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("accept outcomes are passed through", func(t *testing.T) {
		for _, e := range []error{
			domain.ErrInvitationNotFound, domain.ErrInvitationNotPending, domain.ErrViewerExists, domain.ErrViewerLimitReached,
		} {
			f := newInviteeFixture(t)
			f.phones.On("FindPhoneByID", ctx, f.userID).Return("+84900000009", nil)
			f.invitations.On("Accept", ctx, int64(5), f.userID, "+84900000009", 20).Return(nil, e)
			if _, err := f.u.Accept(ctx, f.userID, 5); !errors.Is(err, e) {
				t.Errorf("err = %v, want %v", err, e)
			}
		}
	})

	t.Run("a non-positive id is not found without reading anything", func(t *testing.T) {
		f := newInviteeFixture(t)
		if _, err := f.u.Accept(ctx, f.userID, 0); !errors.Is(err, domain.ErrInvitationNotFound) {
			t.Errorf("Accept err = %v", err)
		}
		if err := f.u.Decline(ctx, f.userID, -1); !errors.Is(err, domain.ErrInvitationNotFound) {
			t.Errorf("Decline err = %v", err)
		}
		f.phones.AssertNotCalled(t, "FindPhoneByID", mock.Anything, mock.Anything)
	})

	t.Run("decline", func(t *testing.T) {
		f := newInviteeFixture(t)
		f.phones.On("FindPhoneByID", ctx, f.userID).Return("+84900000009", nil)
		f.invitations.On("Decline", ctx, int64(5), "+84900000009").Return(nil)
		if err := f.u.Decline(ctx, f.userID, 5); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("a phone lookup failure is an error, not a silent empty inbox", func(t *testing.T) {
		f := newInviteeFixture(t)
		boom := errors.New("db down")
		f.phones.On("FindPhoneByID", ctx, f.userID).Return("", boom)
		if _, err := f.u.ListMine(ctx, f.userID); !errors.Is(err, boom) {
			t.Errorf("err = %v, want %v", err, boom)
		}
	})
}

// --- profile -----------------------------------------------------------------

func TestProfileUsecase(t *testing.T) {
	ctx := context.Background()

	setup := func(t *testing.T, role domain.Role, phone string) (*GenealogyProfileUsecase, *domain.MockGenealogyMemberRepository) {
		t.Helper()
		genes := new(domain.MockGenealogyRepository)
		members := new(domain.MockGenealogyMemberRepository)
		phones := new(domain.MockUserPhoneLookup)
		genes.On("ResolveRole", mock.Anything, int64(42), int64(9)).Return(role, nil)
		phones.On("FindPhoneByID", mock.Anything, int64(9)).Return(phone, nil)
		u := NewGenealogyProfileUsecase(genes, members, phones)
		u.now = func() time.Time { return time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC) }
		return u, members
	}

	tree := func() []domain.GenealogyMember {
		all := fixture()
		for i := range all {
			if all[i].ID == 11 { // C: the viewer's own node
				all[i].PhoneNumber = strP("+84900000009")
				all[i].BirthYear = intP(1990)
			}
		}
		return all
	}

	t.Run("a viewer opens a member and sees how it relates to them", func(t *testing.T) {
		u, members := setup(t, domain.RoleViewer, "+84900000009")
		members.On("List", mock.Anything, int64(42)).Return(tree(), nil)

		p, err := u.GetProfile(ctx, 9, 42, 14) // Z, father's elder brother
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.MyMemberID == nil || *p.MyMemberID != 11 {
			t.Fatalf("MyMemberID = %v, want 11", p.MyMemberID)
		}
		if p.RelationToMe == nil || p.RelationToMe.Label != "Bác" {
			t.Errorf("RelationToMe = %+v, want Bác", p.RelationToMe)
		}
		if p.Member.ID != 14 {
			t.Errorf("Member.ID = %d", p.Member.ID)
		}
		if len(p.Lineage) != 3 { // Tổ, Lộc, Z
			t.Errorf("Lineage = %+v", p.Lineage)
		}
		if p.Chi == nil || p.Chi.Ancestor.FullName != "Z" {
			t.Errorf("Chi = %+v, want Z", p.Chi)
		}
		if len(p.Children) != 1 || p.Children[0].ID != 19 {
			t.Errorf("Children = %+v", p.Children)
		}
	})

	t.Run("my own profile carries my age", func(t *testing.T) {
		u, members := setup(t, domain.RoleViewer, "+84900000009")
		members.On("List", mock.Anything, int64(42)).Return(tree(), nil)
		p, err := u.GetProfile(ctx, 9, 42, 11)
		if err != nil {
			t.Fatal(err)
		}
		if p.Age == nil || *p.Age != 36 {
			t.Errorf("Age = %v, want 36", p.Age)
		}
		if p.TraditionalAge == nil || *p.TraditionalAge != 37 {
			t.Errorf("TraditionalAge = %v, want 37 (tuổi mụ = tuổi tây + 1)", p.TraditionalAge)
		}
		if p.RelationToMe == nil || p.RelationToMe.Code != domain.KinshipSelf {
			t.Errorf("RelationToMe = %+v, want SELF", p.RelationToMe)
		}
	})

	t.Run("no member carries the caller's number: no relation to me", func(t *testing.T) {
		u, members := setup(t, domain.RoleOwner, "+84911111111")
		members.On("List", mock.Anything, int64(42)).Return(fixture(), nil)
		p, err := u.GetProfile(ctx, 9, 42, 14)
		if err != nil {
			t.Fatal(err)
		}
		if p.MyMemberID != nil || p.RelationToMe != nil {
			t.Errorf("MyMemberID=%v RelationToMe=%+v, want both nil", p.MyMemberID, p.RelationToMe)
		}
	})

	t.Run("an unknown member is not found", func(t *testing.T) {
		u, members := setup(t, domain.RoleViewer, "+84900000009")
		members.On("List", mock.Anything, int64(42)).Return(tree(), nil)
		if _, err := u.GetProfile(ctx, 9, 42, 999); !errors.Is(err, domain.ErrMemberNotFound) {
			t.Errorf("err = %v, want ErrMemberNotFound", err)
		}
	})

	t.Run("a stranger is not found and the tree is not read", func(t *testing.T) {
		genes := new(domain.MockGenealogyRepository)
		members := new(domain.MockGenealogyMemberRepository)
		phones := new(domain.MockUserPhoneLookup)
		genes.On("ResolveRole", mock.Anything, int64(42), int64(9)).Return(domain.Role(""), domain.ErrGenealogyNotFound)
		u := NewGenealogyProfileUsecase(genes, members, phones)
		if _, err := u.GetProfile(ctx, 9, 42, 14); !errors.Is(err, domain.ErrGenealogyNotFound) {
			t.Errorf("err = %v, want ErrGenealogyNotFound", err)
		}
		members.AssertNotCalled(t, "List", mock.Anything, mock.Anything)
	})
}
