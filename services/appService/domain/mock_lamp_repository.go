package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ LampRepository = (*MockLampRepository)(nil)

// MockLampRepository is a hand-written mock of LampRepository (see
// mock_memorial_repository.go for the pattern).
type MockLampRepository struct {
	mock.Mock
}

// Light mocks the Light method. Return(lamp, created, err).
func (m *MockLampRepository) Light(ctx context.Context, p LightLampParams) (*Lamp, bool, error) {
	args := m.Called(ctx, p)
	if r := args.Get(0); r != nil {
		return r.(*Lamp), args.Bool(1), args.Error(2)
	}
	return nil, args.Bool(1), args.Error(2)
}

// Garden mocks the Garden method.
func (m *MockLampRepository) Garden(ctx context.Context, q GardenQuery) (*GardenPage, error) {
	args := m.Called(ctx, q)
	if r := args.Get(0); r != nil {
		return r.(*GardenPage), args.Error(1)
	}
	return nil, args.Error(1)
}
