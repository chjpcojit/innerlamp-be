package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ AdministrativeUnitRepository = (*MockAdministrativeUnitRepository)(nil)

// MockAdministrativeUnitRepository is a hand-written mock of
// AdministrativeUnitRepository (see mock_login_repository.go for the pattern).
type MockAdministrativeUnitRepository struct {
	mock.Mock
}

// ListCurrentProvinces mocks the ListCurrentProvinces method.
func (m *MockAdministrativeUnitRepository) ListCurrentProvinces(ctx context.Context) ([]CurrentProvince, error) {
	args := m.Called(ctx)
	if v := args.Get(0); v != nil {
		return v.([]CurrentProvince), args.Error(1)
	}
	return nil, args.Error(1)
}

// ListCurrentWards mocks the ListCurrentWards method.
func (m *MockAdministrativeUnitRepository) ListCurrentWards(ctx context.Context, provinceCode string) ([]CurrentWard, error) {
	args := m.Called(ctx, provinceCode)
	if v := args.Get(0); v != nil {
		return v.([]CurrentWard), args.Error(1)
	}
	return nil, args.Error(1)
}

// ListLegacyProvinces mocks the ListLegacyProvinces method.
func (m *MockAdministrativeUnitRepository) ListLegacyProvinces(ctx context.Context) ([]LegacyProvince, error) {
	args := m.Called(ctx)
	if v := args.Get(0); v != nil {
		return v.([]LegacyProvince), args.Error(1)
	}
	return nil, args.Error(1)
}

// ListLegacyDistricts mocks the ListLegacyDistricts method.
func (m *MockAdministrativeUnitRepository) ListLegacyDistricts(ctx context.Context, provinceCode string) ([]LegacyDistrict, error) {
	args := m.Called(ctx, provinceCode)
	if v := args.Get(0); v != nil {
		return v.([]LegacyDistrict), args.Error(1)
	}
	return nil, args.Error(1)
}

// ListLegacyWards mocks the ListLegacyWards method.
func (m *MockAdministrativeUnitRepository) ListLegacyWards(ctx context.Context, districtCode string) ([]LegacyWard, error) {
	args := m.Called(ctx, districtCode)
	if v := args.Get(0); v != nil {
		return v.([]LegacyWard), args.Error(1)
	}
	return nil, args.Error(1)
}

// ConvertLegacyWard mocks the ConvertLegacyWard method.
func (m *MockAdministrativeUnitRepository) ConvertLegacyWard(ctx context.Context, wardCode string) ([]CurrentWardCandidate, error) {
	args := m.Called(ctx, wardCode)
	if v := args.Get(0); v != nil {
		return v.([]CurrentWardCandidate), args.Error(1)
	}
	return nil, args.Error(1)
}

// ConvertCurrentWard mocks the ConvertCurrentWard method.
func (m *MockAdministrativeUnitRepository) ConvertCurrentWard(ctx context.Context, wardCode string) ([]LegacyWardCandidate, error) {
	args := m.Called(ctx, wardCode)
	if v := args.Get(0); v != nil {
		return v.([]LegacyWardCandidate), args.Error(1)
	}
	return nil, args.Error(1)
}

// ConvertLegacyProvince mocks the ConvertLegacyProvince method.
func (m *MockAdministrativeUnitRepository) ConvertLegacyProvince(ctx context.Context, provinceCode string) ([]CurrentProvince, error) {
	args := m.Called(ctx, provinceCode)
	if v := args.Get(0); v != nil {
		return v.([]CurrentProvince), args.Error(1)
	}
	return nil, args.Error(1)
}

// ConvertCurrentProvince mocks the ConvertCurrentProvince method.
func (m *MockAdministrativeUnitRepository) ConvertCurrentProvince(ctx context.Context, provinceCode string) ([]LegacyProvince, error) {
	args := m.Called(ctx, provinceCode)
	if v := args.Get(0); v != nil {
		return v.([]LegacyProvince), args.Error(1)
	}
	return nil, args.Error(1)
}

// ReplaceDataset mocks the ReplaceDataset method.
func (m *MockAdministrativeUnitRepository) ReplaceDataset(ctx context.Context, ds Dataset) error {
	return m.Called(ctx, ds).Error(0)
}
