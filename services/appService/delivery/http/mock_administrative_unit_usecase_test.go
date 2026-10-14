package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockAdministrativeUnitUsecase implements administrativeUnitUsecase for
// handler tests.
type mockAdministrativeUnitUsecase struct {
	mock.Mock
}

func (m *mockAdministrativeUnitUsecase) ListCurrentProvinces(ctx context.Context) ([]domain.CurrentProvince, error) {
	args := m.Called(ctx)
	if r := args.Get(0); r != nil {
		return r.([]domain.CurrentProvince), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAdministrativeUnitUsecase) ListCurrentWards(ctx context.Context, provinceCode string) ([]domain.CurrentWard, error) {
	args := m.Called(ctx, provinceCode)
	if r := args.Get(0); r != nil {
		return r.([]domain.CurrentWard), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAdministrativeUnitUsecase) ListLegacyProvinces(ctx context.Context) ([]domain.LegacyProvince, error) {
	args := m.Called(ctx)
	if r := args.Get(0); r != nil {
		return r.([]domain.LegacyProvince), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAdministrativeUnitUsecase) ListLegacyDistricts(ctx context.Context, provinceCode string) ([]domain.LegacyDistrict, error) {
	args := m.Called(ctx, provinceCode)
	if r := args.Get(0); r != nil {
		return r.([]domain.LegacyDistrict), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAdministrativeUnitUsecase) ListLegacyWards(ctx context.Context, districtCode string) ([]domain.LegacyWard, error) {
	args := m.Called(ctx, districtCode)
	if r := args.Get(0); r != nil {
		return r.([]domain.LegacyWard), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAdministrativeUnitUsecase) ConvertLegacyWard(ctx context.Context, wardCode string) (domain.Conversion[domain.CurrentWardCandidate], error) {
	args := m.Called(ctx, wardCode)
	return args.Get(0).(domain.Conversion[domain.CurrentWardCandidate]), args.Error(1)
}

func (m *mockAdministrativeUnitUsecase) ConvertCurrentWard(ctx context.Context, wardCode string) (domain.Conversion[domain.LegacyWardCandidate], error) {
	args := m.Called(ctx, wardCode)
	return args.Get(0).(domain.Conversion[domain.LegacyWardCandidate]), args.Error(1)
}

func (m *mockAdministrativeUnitUsecase) ConvertLegacyProvince(ctx context.Context, provinceCode string) (domain.Conversion[domain.CurrentProvince], error) {
	args := m.Called(ctx, provinceCode)
	return args.Get(0).(domain.Conversion[domain.CurrentProvince]), args.Error(1)
}

func (m *mockAdministrativeUnitUsecase) ConvertCurrentProvince(ctx context.Context, provinceCode string) (domain.Conversion[domain.LegacyProvince], error) {
	args := m.Called(ctx, provinceCode)
	return args.Get(0).(domain.Conversion[domain.LegacyProvince]), args.Error(1)
}

func (m *mockAdministrativeUnitUsecase) SearchCurrentWards(ctx context.Context, provinceCode, query string) ([]domain.CurrentWard, error) {
	args := m.Called(ctx, provinceCode, query)
	if r := args.Get(0); r != nil {
		return r.([]domain.CurrentWard), args.Error(1)
	}
	return nil, args.Error(1)
}
