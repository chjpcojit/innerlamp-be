package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func TestAdministrativeUnitUsecase_ListsWithoutParent(t *testing.T) {
	t.Run("current provinces are returned as the repository gave them", func(t *testing.T) {
		repo := new(domain.MockAdministrativeUnitRepository)
		want := []domain.CurrentProvince{{Code: "79", Name: "Hồ Chí Minh"}, {Code: "01", Name: "Hà Nội"}}
		repo.On("ListCurrentProvinces", mock.Anything).Return(want, nil)

		got, err := NewAdministrativeUnitUsecase(repo).ListCurrentProvinces(context.Background())

		require.NoError(t, err)
		assert.Equal(t, want, got, "order is the repository's, not re-sorted here")
		repo.AssertExpectations(t)
	})

	t.Run("legacy provinces are returned as the repository gave them", func(t *testing.T) {
		repo := new(domain.MockAdministrativeUnitRepository)
		want := []domain.LegacyProvince{{Code: "74", Name: "Bình Dương"}}
		repo.On("ListLegacyProvinces", mock.Anything).Return(want, nil)

		got, err := NewAdministrativeUnitUsecase(repo).ListLegacyProvinces(context.Background())

		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("a nil result becomes an empty list, not null", func(t *testing.T) {
		repo := new(domain.MockAdministrativeUnitRepository)
		repo.On("ListCurrentProvinces", mock.Anything).Return(nil, nil)
		repo.On("ListLegacyProvinces", mock.Anything).Return(nil, nil)
		u := NewAdministrativeUnitUsecase(repo)

		cur, err := u.ListCurrentProvinces(context.Background())
		require.NoError(t, err)
		assert.NotNil(t, cur)
		assert.Empty(t, cur)

		leg, err := u.ListLegacyProvinces(context.Background())
		require.NoError(t, err)
		assert.NotNil(t, leg)
		assert.Empty(t, leg)
	})

	t.Run("repository errors are wrapped", func(t *testing.T) {
		repo := new(domain.MockAdministrativeUnitRepository)
		boom := errors.New("db down")
		repo.On("ListCurrentProvinces", mock.Anything).Return(nil, boom)
		repo.On("ListLegacyProvinces", mock.Anything).Return(nil, boom)
		u := NewAdministrativeUnitUsecase(repo)

		_, err := u.ListCurrentProvinces(context.Background())
		assert.ErrorIs(t, err, boom)
		_, err = u.ListLegacyProvinces(context.Background())
		assert.ErrorIs(t, err, boom)
	})
}

// codeCase describes one use-case method that takes a unit code, so the same
// four guarantees are proved for every one of them.
type codeCase struct {
	name       string
	repoMethod string
	// stub is a non-empty value the repository returns on success.
	stub any
	// call runs the use case and reports how many items came back and whether
	// the returned slice was non-nil.
	call func(u *AdministrativeUnitUsecase, code string) (n int, nonNil bool, err error)
}

func codeCases() []codeCase {
	ctx := context.Background()
	return []codeCase{
		{
			name: "ListCurrentWards", repoMethod: "ListCurrentWards",
			stub: []domain.CurrentWard{{Code: "26737", ProvinceCode: "79"}},
			call: func(u *AdministrativeUnitUsecase, code string) (int, bool, error) {
				r, err := u.ListCurrentWards(ctx, code)
				return len(r), r != nil, err
			},
		},
		{
			name: "ListLegacyDistricts", repoMethod: "ListLegacyDistricts",
			stub: []domain.LegacyDistrict{{Code: "760", ProvinceCode: "79"}},
			call: func(u *AdministrativeUnitUsecase, code string) (int, bool, error) {
				r, err := u.ListLegacyDistricts(ctx, code)
				return len(r), r != nil, err
			},
		},
		{
			name: "ListLegacyWards", repoMethod: "ListLegacyWards",
			stub: []domain.LegacyWard{{Code: "26734", DistrictCode: "760"}},
			call: func(u *AdministrativeUnitUsecase, code string) (int, bool, error) {
				r, err := u.ListLegacyWards(ctx, code)
				return len(r), r != nil, err
			},
		},
		{
			name: "ConvertLegacyWard", repoMethod: "ConvertLegacyWard",
			stub: []domain.CurrentWardCandidate{{Ward: domain.CurrentWard{Code: "26737"}, MappingType: "merged"}},
			call: func(u *AdministrativeUnitUsecase, code string) (int, bool, error) {
				r, err := u.ConvertLegacyWard(ctx, code)
				return len(r.Candidates), r.Candidates != nil || err != nil, err
			},
		},
		{
			name: "ConvertCurrentWard", repoMethod: "ConvertCurrentWard",
			stub: []domain.LegacyWardCandidate{{Ward: domain.LegacyWard{Code: "26755"}, MappingType: "merged"}},
			call: func(u *AdministrativeUnitUsecase, code string) (int, bool, error) {
				r, err := u.ConvertCurrentWard(ctx, code)
				return len(r.Candidates), r.Candidates != nil || err != nil, err
			},
		},
		{
			name: "ConvertLegacyProvince", repoMethod: "ConvertLegacyProvince",
			stub: []domain.CurrentProvince{{Code: "79"}},
			call: func(u *AdministrativeUnitUsecase, code string) (int, bool, error) {
				r, err := u.ConvertLegacyProvince(ctx, code)
				return len(r.Candidates), r.Candidates != nil || err != nil, err
			},
		},
		{
			name: "ConvertCurrentProvince", repoMethod: "ConvertCurrentProvince",
			stub: []domain.LegacyProvince{{Code: "74"}, {Code: "77"}, {Code: "79"}},
			call: func(u *AdministrativeUnitUsecase, code string) (int, bool, error) {
				r, err := u.ConvertCurrentProvince(ctx, code)
				return len(r.Candidates), r.Candidates != nil || err != nil, err
			},
		},
	}
}

func TestAdministrativeUnitUsecase_CodeMethods(t *testing.T) {
	boom := errors.New("db down")

	for _, tc := range codeCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("passes the trimmed code to the repository and returns its items", func(t *testing.T) {
				repo := new(domain.MockAdministrativeUnitRepository)
				repo.On(tc.repoMethod, mock.Anything, "79").Return(tc.stub, nil)

				n, _, err := tc.call(NewAdministrativeUnitUsecase(repo), "  79 ")

				require.NoError(t, err)
				assert.Positive(t, n)
				repo.AssertExpectations(t)
			})

			t.Run("a blank code is not found and never reaches the repository", func(t *testing.T) {
				repo := new(domain.MockAdministrativeUnitRepository)

				for _, blank := range []string{"", "   ", "\t\n"} {
					_, _, err := tc.call(NewAdministrativeUnitUsecase(repo), blank)
					assert.ErrorIs(t, err, domain.ErrAdministrativeUnitNotFound, "code %q", blank)
				}
				repo.AssertNotCalled(t, tc.repoMethod, mock.Anything, mock.Anything)
			})

			t.Run("not found from the repository stays errors.Is-able", func(t *testing.T) {
				repo := new(domain.MockAdministrativeUnitRepository)
				repo.On(tc.repoMethod, mock.Anything, "999").Return(nil, domain.ErrAdministrativeUnitNotFound)

				_, _, err := tc.call(NewAdministrativeUnitUsecase(repo), "999")

				assert.ErrorIs(t, err, domain.ErrAdministrativeUnitNotFound)
			})

			t.Run("other repository errors are wrapped", func(t *testing.T) {
				repo := new(domain.MockAdministrativeUnitRepository)
				repo.On(tc.repoMethod, mock.Anything, "79").Return(nil, boom)

				_, _, err := tc.call(NewAdministrativeUnitUsecase(repo), "79")

				assert.ErrorIs(t, err, boom)
				assert.NotErrorIs(t, err, domain.ErrAdministrativeUnitNotFound)
			})

			t.Run("a nil result is an empty list, not null", func(t *testing.T) {
				repo := new(domain.MockAdministrativeUnitRepository)
				repo.On(tc.repoMethod, mock.Anything, "79").Return(nil, nil)

				n, nonNil, err := tc.call(NewAdministrativeUnitUsecase(repo), "79")

				require.NoError(t, err)
				assert.Zero(t, n)
				assert.True(t, nonNil, "empty result must be a non-nil slice so it serialises as []")
			})
		})
	}
}

func TestAdministrativeUnitUsecase_ConversionUniqueness(t *testing.T) {
	tests := []struct {
		name       string
		candidates []domain.CurrentWardCandidate
		want       bool
	}{
		{"whole merge gives one candidate", []domain.CurrentWardCandidate{{MappingType: "merged"}}, true},
		{"a split old ward gives several and is not unique",
			[]domain.CurrentWardCandidate{{MappingType: "ambiguous"}, {MappingType: "ambiguous"}}, false},
		{"no candidates is not unique", nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(domain.MockAdministrativeUnitRepository)
			repo.On("ConvertLegacyWard", mock.Anything, "26737").Return(tc.candidates, nil)

			got, err := NewAdministrativeUnitUsecase(repo).ConvertLegacyWard(context.Background(), "26737")

			require.NoError(t, err)
			assert.Equal(t, tc.want, got.Unique())
			assert.Len(t, got.Candidates, len(tc.candidates), "every candidate is returned, none is dropped or chosen")
		})
	}
}

func TestAdministrativeUnitUsecase_ConversionKeepsCandidateDetail(t *testing.T) {
	repo := new(domain.MockAdministrativeUnitRepository)
	want := []domain.LegacyWardCandidate{
		{
			Ward:        domain.LegacyWard{Code: "26755", Name: "Cô Giang"},
			District:    domain.LegacyDistrict{Code: "760", Name: "Quận 1"},
			Province:    domain.LegacyProvince{Code: "79", Name: "Hồ Chí Minh"},
			MappingType: "merged",
			Note:        "Nhập toàn bộ",
		},
		{
			Ward:        domain.LegacyWard{Code: "26758", Name: "Nguyễn Cư Trinh"},
			District:    domain.LegacyDistrict{Code: "760", Name: "Quận 1"},
			Province:    domain.LegacyProvince{Code: "79", Name: "Hồ Chí Minh"},
			MappingType: "merged",
			Note:        "Nhập toàn bộ",
		},
	}
	repo.On("ConvertCurrentWard", mock.Anything, "26758").Return(want, nil)

	got, err := NewAdministrativeUnitUsecase(repo).ConvertCurrentWard(context.Background(), "26758")

	require.NoError(t, err)
	assert.Equal(t, want, got.Candidates, "order and every field pass through unchanged")
	assert.False(t, got.Unique())
}
