package domain

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

var _ AnnouncementRepository = (*MockAnnouncementRepository)(nil)

// MockAnnouncementRepository is a hand-written mock of AnnouncementRepository.
type MockAnnouncementRepository struct {
	mock.Mock
}

func (m *MockAnnouncementRepository) TempleExists(ctx context.Context, templeID string) (bool, error) {
	args := m.Called(ctx, templeID)
	return args.Bool(0), args.Error(1)
}

func (m *MockAnnouncementRepository) IssuingBodyBelongs(ctx context.Context, issuingBodyID, templeID string) (bool, error) {
	args := m.Called(ctx, issuingBodyID, templeID)
	return args.Bool(0), args.Error(1)
}

func (m *MockAnnouncementRepository) Insert(ctx context.Context, a Announcement) (*Announcement, error) {
	args := m.Called(ctx, a)
	if r := args.Get(0); r != nil {
		return r.(*Announcement), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAnnouncementRepository) FindByID(ctx context.Context, id string) (*Announcement, error) {
	args := m.Called(ctx, id)
	if r := args.Get(0); r != nil {
		return r.(*Announcement), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAnnouncementRepository) Update(ctx context.Context, a Announcement) (*Announcement, error) {
	args := m.Called(ctx, a)
	if r := args.Get(0); r != nil {
		return r.(*Announcement), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAnnouncementRepository) Revoke(ctx context.Context, id string, at time.Time) (*Announcement, error) {
	args := m.Called(ctx, id, at)
	if r := args.Get(0); r != nil {
		return r.(*Announcement), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAnnouncementRepository) ListByTemple(ctx context.Context, templeID string, offset, limit int) ([]Announcement, int, error) {
	args := m.Called(ctx, templeID, offset, limit)
	if r := args.Get(0); r != nil {
		return r.([]Announcement), args.Int(1), args.Error(2)
	}
	return nil, args.Int(1), args.Error(2)
}

var _ IssuingBodyRepository = (*MockIssuingBodyRepository)(nil)

// MockIssuingBodyRepository is a hand-written mock of IssuingBodyRepository.
type MockIssuingBodyRepository struct {
	mock.Mock
}

func (m *MockIssuingBodyRepository) Insert(ctx context.Context, templeID, name string) (*IssuingBody, error) {
	args := m.Called(ctx, templeID, name)
	if r := args.Get(0); r != nil {
		return r.(*IssuingBody), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockIssuingBodyRepository) ListByTemple(ctx context.Context, templeID string) ([]IssuingBody, error) {
	args := m.Called(ctx, templeID)
	if r := args.Get(0); r != nil {
		return r.([]IssuingBody), args.Error(1)
	}
	return nil, args.Error(1)
}
