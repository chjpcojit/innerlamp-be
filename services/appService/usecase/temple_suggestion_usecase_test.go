package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func validSuggestionInput() domain.TempleSuggestionInput {
	return domain.TempleSuggestionInput{
		Name: "Chùa Một Cột", ProvinceCode: "01", WardCode: "00004", Address: "Phố Chùa Một Cột",
	}
}

var resolvedWard = &domain.ResolvedWard{ProvinceName: "Hà Nội", WardFullName: "Phường Ba Đình"}

func TestTempleSuggestionUsecase_Create_Success(t *testing.T) {
	repo := new(domain.MockTempleSuggestionRepository)
	wards := new(domain.MockAdministrativeUnitRepository)
	uc := NewTempleSuggestionUsecase(repo, wards)
	now := time.Now()

	wards.On("ResolveCurrentWard", mock.Anything, "01", "00004").Return(resolvedWard, nil)
	// name and address are trimmed and otherwise stored as typed; the province
	// and ward names are the dataset's, never the client's.
	repo.On("Create", mock.Anything, &domain.TempleSuggestion{
		Name: "Chùa  Một Cột", ProvinceCode: "01", Province: "Hà Nội",
		WardCode: "00004", Ward: "Phường Ba Đình", Address: "12 Nguyễn Huệ",
	}).Return(&domain.TempleSuggestion{
		ID: 7, Status: domain.TempleSuggestionPending, CreatedAt: now,
	}, nil)

	got, err := uc.Create(context.Background(), domain.TempleSuggestionInput{
		Name: "  Chùa  Một Cột ", ProvinceCode: " 01 ", WardCode: "\t00004\n", Address: " 12 Nguyễn Huệ  ",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(7), got.ID)
	assert.Equal(t, domain.TempleSuggestionPending, got.Status)
	repo.AssertExpectations(t)
	wards.AssertExpectations(t)
}

func TestTempleSuggestionUsecase_Create_Validation(t *testing.T) {
	long := func(n int) string { return strings.Repeat("ạ", n) } // multi-byte: counts runes, not bytes

	tests := []struct {
		name      string
		mutate    func(in *domain.TempleSuggestionInput)
		wantField string
	}{
		{"empty name", func(in *domain.TempleSuggestionInput) { in.Name = "" }, "name"},
		{"blank province code", func(in *domain.TempleSuggestionInput) { in.ProvinceCode = "   " }, "province_code"},
		{"blank ward code", func(in *domain.TempleSuggestionInput) { in.WardCode = "" }, "ward_code"},
		{"blank address", func(in *domain.TempleSuggestionInput) { in.Address = " \t " }, "address"},
		{"name too long", func(in *domain.TempleSuggestionInput) { in.Name = long(256) }, "name"},
		{"address too long", func(in *domain.TempleSuggestionInput) { in.Address = long(501) }, "address"},
		{"first offender wins", func(in *domain.TempleSuggestionInput) { in.WardCode = ""; in.Address = "" }, "ward_code"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockTempleSuggestionRepository)
			wards := new(domain.MockAdministrativeUnitRepository)
			in := validSuggestionInput()
			tt.mutate(&in)

			_, err := NewTempleSuggestionUsecase(repo, wards).Create(context.Background(), in)

			require.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrTempleSuggestionInvalid)
			bad, ok := errors.AsType[*domain.TempleSuggestionValidationError](err)
			require.True(t, ok)
			assert.Equal(t, tt.wantField, bad.Field)
			// Shape errors are caught before any lookup or write.
			wards.AssertNotCalled(t, "ResolveCurrentWard", mock.Anything, mock.Anything, mock.Anything)
			repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		})
	}
}

func TestTempleSuggestionUsecase_Create_UnknownCodesAreValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		resolveEr error
		wantField string
	}{
		{"province not in the new scheme", domain.ErrAdministrativeUnitNotFound, "province_code"},
		{"ward missing or in another province", domain.ErrWardNotInProvince, "ward_code"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockTempleSuggestionRepository)
			wards := new(domain.MockAdministrativeUnitRepository)
			wards.On("ResolveCurrentWard", mock.Anything, "01", "00004").Return(nil, tt.resolveEr)

			_, err := NewTempleSuggestionUsecase(repo, wards).Create(context.Background(), validSuggestionInput())

			require.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrTempleSuggestionInvalid)
			bad, ok := errors.AsType[*domain.TempleSuggestionValidationError](err)
			require.True(t, ok)
			assert.Equal(t, tt.wantField, bad.Field)
			repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		})
	}
}

func TestTempleSuggestionUsecase_Create_ResolverErrorIsNotAValidationError(t *testing.T) {
	repo := new(domain.MockTempleSuggestionRepository)
	wards := new(domain.MockAdministrativeUnitRepository)
	boom := errors.New("db down")
	wards.On("ResolveCurrentWard", mock.Anything, "01", "00004").Return(nil, boom)

	_, err := NewTempleSuggestionUsecase(repo, wards).Create(context.Background(), validSuggestionInput())

	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
	assert.NotErrorIs(t, err, domain.ErrTempleSuggestionInvalid)
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestTempleSuggestionUsecase_Create_AtLimitIsAccepted(t *testing.T) {
	repo := new(domain.MockTempleSuggestionRepository)
	wards := new(domain.MockAdministrativeUnitRepository)
	wards.On("ResolveCurrentWard", mock.Anything, "01", "00004").Return(resolvedWard, nil)
	repo.On("Create", mock.Anything, mock.Anything).Return(&domain.TempleSuggestion{ID: 1}, nil)
	in := validSuggestionInput()
	in.Name = strings.Repeat("a", domain.TempleSuggestionNameMax)
	in.Address = strings.Repeat("ạ", domain.TempleSuggestionAddressMax)

	_, err := NewTempleSuggestionUsecase(repo, wards).Create(context.Background(), in)

	require.NoError(t, err)
}

func TestTempleSuggestionUsecase_Create_RepositoryErrorIsWrapped(t *testing.T) {
	repo := new(domain.MockTempleSuggestionRepository)
	wards := new(domain.MockAdministrativeUnitRepository)
	wards.On("ResolveCurrentWard", mock.Anything, "01", "00004").Return(resolvedWard, nil)
	boom := errors.New("db down")
	repo.On("Create", mock.Anything, mock.Anything).Return(nil, boom)

	_, err := NewTempleSuggestionUsecase(repo, wards).Create(context.Background(), validSuggestionInput())

	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
	assert.NotErrorIs(t, err, domain.ErrTempleSuggestionInvalid)
}
