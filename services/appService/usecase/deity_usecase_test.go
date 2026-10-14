package usecase

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

var testDeityCfg = config.DeityConfig{TopLimit: 10, PageSize: 20}

func newDeityUsecase(repo *domain.MockDeityRepository) *DeityUsecase {
	return NewDeityUsecase(repo, testDeityCfg)
}

func TestDeityUsecase_Top(t *testing.T) {
	t.Run("asks the repository for the configured limit", func(t *testing.T) {
		repo := new(domain.MockDeityRepository)
		want := []domain.Deity{{ID: "a"}, {ID: "b"}}
		repo.On("ListTop", mock.Anything, 10).Return(want, nil)

		got, err := newDeityUsecase(repo).Top(context.Background())

		require.NoError(t, err)
		assert.Equal(t, want, got)
		repo.AssertExpectations(t)
	})

	t.Run("wraps repository errors", func(t *testing.T) {
		repo := new(domain.MockDeityRepository)
		boom := errors.New("db down")
		repo.On("ListTop", mock.Anything, 10).Return(nil, boom)

		_, err := newDeityUsecase(repo).Top(context.Background())

		assert.ErrorIs(t, err, boom)
	})
}

func TestDeityUsecase_ListDeities(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		total      int
		wantOffset int
		wantPages  int
	}{
		{"first page", 1, 12, 0, 1},
		{"second page", 2, 45, 20, 3},
		{"page past the end is empty, not an error", 99, 12, 1960, 1},
		{"empty catalogue has zero pages", 1, 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockDeityRepository)
			repo.On("List", mock.Anything, "", tt.wantOffset, 20).
				Return(nil, tt.total, nil)

			got, err := newDeityUsecase(repo).ListDeities(context.Background(), "", tt.page)

			require.NoError(t, err)
			assert.NotNil(t, got.Items, "items must marshal as [] not null")
			assert.Empty(t, got.Items)
			assert.Equal(t, tt.page, got.Page)
			assert.Equal(t, 20, got.PageSize)
			assert.Equal(t, tt.total, got.Total)
			assert.Equal(t, tt.wantPages, got.TotalPages)
			repo.AssertExpectations(t)
		})
	}

	for _, page := range []int{0, -1, math.MaxInt32 + 1} {
		t.Run("invalid page", func(t *testing.T) {
			repo := new(domain.MockDeityRepository)
			_, err := newDeityUsecase(repo).ListDeities(context.Background(), "", page)
			assert.ErrorIs(t, err, domain.ErrInvalidPage)
			repo.AssertNotCalled(t, "List")
		})
	}

	for _, typ := range []string{domain.DeityBuddha, domain.DeityBodhisattva, domain.DeityDeva} {
		t.Run("filters by type "+typ, func(t *testing.T) {
			repo := new(domain.MockDeityRepository)
			repo.On("List", mock.Anything, typ, 20, 20).Return([]domain.Deity{{ID: "x", Type: typ}}, 25, nil)

			got, err := newDeityUsecase(repo).ListDeities(context.Background(), typ, 2)

			require.NoError(t, err)
			assert.Equal(t, 25, got.Total, "total is the filtered count")
			assert.Equal(t, 2, got.TotalPages)
			repo.AssertExpectations(t)
		})
	}

	for _, typ := range []string{"buddha", "X", " BUDDHA", "BUDDHA,DEVA"} {
		t.Run("invalid type "+typ, func(t *testing.T) {
			repo := new(domain.MockDeityRepository)
			_, err := newDeityUsecase(repo).ListDeities(context.Background(), typ, 1)
			assert.ErrorIs(t, err, domain.ErrInvalidDeityType)
			repo.AssertNotCalled(t, "List")
		})
	}

	t.Run("invalid page is reported before invalid type", func(t *testing.T) {
		repo := new(domain.MockDeityRepository)
		_, err := newDeityUsecase(repo).ListDeities(context.Background(), "X", 0)
		assert.ErrorIs(t, err, domain.ErrInvalidPage)
	})

	t.Run("wraps repository errors", func(t *testing.T) {
		repo := new(domain.MockDeityRepository)
		boom := errors.New("db down")
		repo.On("List", mock.Anything, "", 0, 20).Return(nil, 0, boom)
		_, err := newDeityUsecase(repo).ListDeities(context.Background(), "", 1)
		assert.ErrorIs(t, err, boom)
	})
}

func TestDeityUsecase_Get(t *testing.T) {
	t.Run("returns the deity", func(t *testing.T) {
		repo := new(domain.MockDeityRepository)
		want := &domain.Deity{ID: "bodhisattva_cundi"}
		repo.On("FindByID", mock.Anything, "bodhisattva_cundi").Return(want, nil)
		got, err := newDeityUsecase(repo).Get(context.Background(), "bodhisattva_cundi")
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("not found stays matchable through the wrap", func(t *testing.T) {
		repo := new(domain.MockDeityRepository)
		repo.On("FindByID", mock.Anything, "x").Return(nil, domain.ErrDeityNotFound)
		_, err := newDeityUsecase(repo).Get(context.Background(), "x")
		assert.ErrorIs(t, err, domain.ErrDeityNotFound)
	})
}
