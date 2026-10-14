package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

type mockGenealogyUsecase struct{ mock.Mock }

func (m *mockGenealogyUsecase) Create(ctx context.Context, userID int64, in domain.CreateGenealogyInput) (*domain.GenealogyDetail, error) {
	args := m.Called(ctx, userID, in)
	if r := args.Get(0); r != nil {
		return r.(*domain.GenealogyDetail), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGenealogyUsecase) List(ctx context.Context, userID int64) ([]domain.GenealogySummary, error) {
	args := m.Called(ctx, userID)
	if r := args.Get(0); r != nil {
		return r.([]domain.GenealogySummary), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGenealogyUsecase) Get(ctx context.Context, userID, id int64) (*domain.GenealogyDetail, error) {
	args := m.Called(ctx, userID, id)
	if r := args.Get(0); r != nil {
		return r.(*domain.GenealogyDetail), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGenealogyUsecase) UpdateInfo(ctx context.Context, userID, id int64, name, home string) (*domain.GenealogyDetail, error) {
	args := m.Called(ctx, userID, id, name, home)
	if r := args.Get(0); r != nil {
		return r.(*domain.GenealogyDetail), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGenealogyUsecase) Delete(ctx context.Context, userID, id int64) error {
	return m.Called(ctx, userID, id).Error(0)
}

func (m *mockGenealogyUsecase) UpdateChronicle(ctx context.Context, userID, id int64, in domain.Chronicle) (*domain.GenealogyDetail, error) {
	args := m.Called(ctx, userID, id, in)
	if r := args.Get(0); r != nil {
		return r.(*domain.GenealogyDetail), args.Error(1)
	}
	return nil, args.Error(1)
}

type mockGenealogyMemberUsecase struct{ mock.Mock }

func (m *mockGenealogyMemberUsecase) List(ctx context.Context, userID, gid int64) ([]domain.MemberView, error) {
	args := m.Called(ctx, userID, gid)
	if r := args.Get(0); r != nil {
		return r.([]domain.MemberView), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGenealogyMemberUsecase) Get(ctx context.Context, userID, gid, mid int64) (*domain.MemberView, error) {
	args := m.Called(ctx, userID, gid, mid)
	if r := args.Get(0); r != nil {
		return r.(*domain.MemberView), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGenealogyMemberUsecase) Add(ctx context.Context, userID, gid int64, in domain.AddMemberInput) (*domain.MemberResult, error) {
	args := m.Called(ctx, userID, gid, in)
	if r := args.Get(0); r != nil {
		return r.(*domain.MemberResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGenealogyMemberUsecase) Update(ctx context.Context, userID, gid, mid int64, info domain.MemberInfo) (*domain.MemberResult, error) {
	args := m.Called(ctx, userID, gid, mid, info)
	if r := args.Get(0); r != nil {
		return r.(*domain.MemberResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGenealogyMemberUsecase) Delete(ctx context.Context, userID, gid, mid int64) error {
	return m.Called(ctx, userID, gid, mid).Error(0)
}

func (m *mockGenealogyMemberUsecase) Move(ctx context.Context, userID, gid, mid int64, in domain.MoveMemberInput) (*domain.MemberResult, error) {
	args := m.Called(ctx, userID, gid, mid, in)
	if r := args.Get(0); r != nil {
		return r.(*domain.MemberResult), args.Error(1)
	}
	return nil, args.Error(1)
}

type mockGenealogyProfileUsecase struct{ mock.Mock }

func (m *mockGenealogyProfileUsecase) GetProfile(ctx context.Context, userID, gid, mid int64) (*domain.MemberProfile, error) {
	args := m.Called(ctx, userID, gid, mid)
	if r := args.Get(0); r != nil {
		return r.(*domain.MemberProfile), args.Error(1)
	}
	return nil, args.Error(1)
}

type mockGenealogyAccessUsecase struct{ mock.Mock }

func (m *mockGenealogyAccessUsecase) Invite(ctx context.Context, userID, gid int64, in domain.InviteInput) (*domain.Invitation, error) {
	args := m.Called(ctx, userID, gid, in)
	if r := args.Get(0); r != nil {
		return r.(*domain.Invitation), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGenealogyAccessUsecase) ListInvitations(ctx context.Context, userID, gid int64) ([]domain.Invitation, error) {
	args := m.Called(ctx, userID, gid)
	if r := args.Get(0); r != nil {
		return r.([]domain.Invitation), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGenealogyAccessUsecase) CancelInvitation(ctx context.Context, userID, gid, inv int64) error {
	return m.Called(ctx, userID, gid, inv).Error(0)
}

func (m *mockGenealogyAccessUsecase) ListViewers(ctx context.Context, userID, gid int64) ([]domain.Viewer, error) {
	args := m.Called(ctx, userID, gid)
	if r := args.Get(0); r != nil {
		return r.([]domain.Viewer), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGenealogyAccessUsecase) RemoveViewer(ctx context.Context, userID, gid, target int64) error {
	return m.Called(ctx, userID, gid, target).Error(0)
}

type mockGenealogyInvitationUsecase struct{ mock.Mock }

func (m *mockGenealogyInvitationUsecase) ListMine(ctx context.Context, userID int64) ([]domain.Invitation, error) {
	args := m.Called(ctx, userID)
	if r := args.Get(0); r != nil {
		return r.([]domain.Invitation), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGenealogyInvitationUsecase) Accept(ctx context.Context, userID, id int64) (*domain.AcceptResult, error) {
	args := m.Called(ctx, userID, id)
	if r := args.Get(0); r != nil {
		return r.(*domain.AcceptResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGenealogyInvitationUsecase) Decline(ctx context.Context, userID, id int64) error {
	return m.Called(ctx, userID, id).Error(0)
}
