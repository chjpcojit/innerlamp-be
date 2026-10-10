package domain

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

var (
	_ GenealogyRepository       = (*MockGenealogyRepository)(nil)
	_ GenealogyMemberRepository = (*MockGenealogyMemberRepository)(nil)
	_ ViewerRepository          = (*MockViewerRepository)(nil)
	_ InvitationRepository      = (*MockInvitationRepository)(nil)
	_ UserPhoneLookup           = (*MockUserPhoneLookup)(nil)
)

// MockGenealogyRepository is a hand-written mock of GenealogyRepository (see
// mock_temple_repository.go for the pattern).
type MockGenealogyRepository struct {
	mock.Mock
}

// Create mocks the Create method.
func (m *MockGenealogyRepository) Create(ctx context.Context, ownerUserID int64, in CreateGenealogyParams, maxPerUser int) (int64, error) {
	args := m.Called(ctx, ownerUserID, in, maxPerUser)
	return args.Get(0).(int64), args.Error(1)
}

// ListByUser mocks the ListByUser method.
func (m *MockGenealogyRepository) ListByUser(ctx context.Context, userID int64) ([]GenealogySummary, error) {
	args := m.Called(ctx, userID)
	if r := args.Get(0); r != nil {
		return r.([]GenealogySummary), args.Error(1)
	}
	return nil, args.Error(1)
}

// ResolveRole mocks the ResolveRole method.
func (m *MockGenealogyRepository) ResolveRole(ctx context.Context, genealogyID, userID int64) (Role, error) {
	args := m.Called(ctx, genealogyID, userID)
	return args.Get(0).(Role), args.Error(1)
}

// FindDetail mocks the FindDetail method.
func (m *MockGenealogyRepository) FindDetail(ctx context.Context, genealogyID int64) (*GenealogyDetail, error) {
	args := m.Called(ctx, genealogyID)
	if r := args.Get(0); r != nil {
		return r.(*GenealogyDetail), args.Error(1)
	}
	return nil, args.Error(1)
}

// UpdateInfo mocks the UpdateInfo method.
func (m *MockGenealogyRepository) UpdateInfo(ctx context.Context, genealogyID int64, name, ancestralHome string) error {
	return m.Called(ctx, genealogyID, name, ancestralHome).Error(0)
}

// Delete mocks the Delete method.
func (m *MockGenealogyRepository) Delete(ctx context.Context, genealogyID int64) error {
	return m.Called(ctx, genealogyID).Error(0)
}

// UpsertChronicle mocks the UpsertChronicle method.
func (m *MockGenealogyRepository) UpsertChronicle(ctx context.Context, c *Chronicle) error {
	return m.Called(ctx, c).Error(0)
}

// MockGenealogyMemberRepository is a hand-written mock of
// GenealogyMemberRepository.
type MockGenealogyMemberRepository struct {
	mock.Mock
}

// List mocks the List method.
func (m *MockGenealogyMemberRepository) List(ctx context.Context, genealogyID int64) ([]GenealogyMember, error) {
	args := m.Called(ctx, genealogyID)
	if r := args.Get(0); r != nil {
		return r.([]GenealogyMember), args.Error(1)
	}
	return nil, args.Error(1)
}

// FindByID mocks the FindByID method.
func (m *MockGenealogyMemberRepository) FindByID(ctx context.Context, genealogyID, memberID int64) (*GenealogyMember, error) {
	args := m.Called(ctx, genealogyID, memberID)
	if r := args.Get(0); r != nil {
		return r.(*GenealogyMember), args.Error(1)
	}
	return nil, args.Error(1)
}

// Add mocks the Add method.
func (m *MockGenealogyMemberRepository) Add(ctx context.Context, genealogyID int64, relation MemberRelation, relatedID int64, mem *GenealogyMember, limits MemberLimits) (*GenealogyMember, error) {
	args := m.Called(ctx, genealogyID, relation, relatedID, mem, limits)
	if r := args.Get(0); r != nil {
		return r.(*GenealogyMember), args.Error(1)
	}
	return nil, args.Error(1)
}

// Update mocks the Update method.
func (m *MockGenealogyMemberRepository) Update(ctx context.Context, genealogyID int64, mem *GenealogyMember) (*GenealogyMember, error) {
	args := m.Called(ctx, genealogyID, mem)
	if r := args.Get(0); r != nil {
		return r.(*GenealogyMember), args.Error(1)
	}
	return nil, args.Error(1)
}

// Delete mocks the Delete method.
func (m *MockGenealogyMemberRepository) Delete(ctx context.Context, genealogyID, memberID int64) error {
	return m.Called(ctx, genealogyID, memberID).Error(0)
}

// Move mocks the Move method.
func (m *MockGenealogyMemberRepository) Move(ctx context.Context, genealogyID, memberID int64, in MoveMemberInput, limits MemberLimits) (*GenealogyMember, error) {
	args := m.Called(ctx, genealogyID, memberID, in, limits)
	if r := args.Get(0); r != nil {
		return r.(*GenealogyMember), args.Error(1)
	}
	return nil, args.Error(1)
}

// FindMemberIDByPhone mocks the FindMemberIDByPhone method.
func (m *MockGenealogyMemberRepository) FindMemberIDByPhone(ctx context.Context, genealogyID int64, phone string) (*int64, error) {
	args := m.Called(ctx, genealogyID, phone)
	if r := args.Get(0); r != nil {
		return r.(*int64), args.Error(1)
	}
	return nil, args.Error(1)
}

// MockViewerRepository is a hand-written mock of ViewerRepository.
type MockViewerRepository struct {
	mock.Mock
}

// List mocks the List method.
func (m *MockViewerRepository) List(ctx context.Context, genealogyID int64) ([]Viewer, error) {
	args := m.Called(ctx, genealogyID)
	if r := args.Get(0); r != nil {
		return r.([]Viewer), args.Error(1)
	}
	return nil, args.Error(1)
}

// Remove mocks the Remove method.
func (m *MockViewerRepository) Remove(ctx context.Context, genealogyID, userID int64) error {
	return m.Called(ctx, genealogyID, userID).Error(0)
}

// MockInvitationRepository is a hand-written mock of InvitationRepository.
type MockInvitationRepository struct {
	mock.Mock
}

// Create mocks the Create method.
func (m *MockInvitationRepository) Create(ctx context.Context, genealogyID, invitedBy int64, in InviteInput, maxViewers int, ttl time.Duration) (*Invitation, error) {
	args := m.Called(ctx, genealogyID, invitedBy, in, maxViewers, ttl)
	if r := args.Get(0); r != nil {
		return r.(*Invitation), args.Error(1)
	}
	return nil, args.Error(1)
}

// ListByGenealogy mocks the ListByGenealogy method.
func (m *MockInvitationRepository) ListByGenealogy(ctx context.Context, genealogyID int64) ([]Invitation, error) {
	args := m.Called(ctx, genealogyID)
	if r := args.Get(0); r != nil {
		return r.([]Invitation), args.Error(1)
	}
	return nil, args.Error(1)
}

// ListMine mocks the ListMine method.
func (m *MockInvitationRepository) ListMine(ctx context.Context, phone string) ([]Invitation, error) {
	args := m.Called(ctx, phone)
	if r := args.Get(0); r != nil {
		return r.([]Invitation), args.Error(1)
	}
	return nil, args.Error(1)
}

// Cancel mocks the Cancel method.
func (m *MockInvitationRepository) Cancel(ctx context.Context, genealogyID, invitationID int64) error {
	return m.Called(ctx, genealogyID, invitationID).Error(0)
}

// Accept mocks the Accept method.
func (m *MockInvitationRepository) Accept(ctx context.Context, invitationID, userID int64, phone string, maxViewers int) (*AcceptResult, error) {
	args := m.Called(ctx, invitationID, userID, phone, maxViewers)
	if r := args.Get(0); r != nil {
		return r.(*AcceptResult), args.Error(1)
	}
	return nil, args.Error(1)
}

// Decline mocks the Decline method.
func (m *MockInvitationRepository) Decline(ctx context.Context, invitationID int64, phone string) error {
	return m.Called(ctx, invitationID, phone).Error(0)
}

// MockUserPhoneLookup is a hand-written mock of UserPhoneLookup.
type MockUserPhoneLookup struct {
	mock.Mock
}

// FindPhoneByID mocks the FindPhoneByID method.
func (m *MockUserPhoneLookup) FindPhoneByID(ctx context.Context, userID int64) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}
