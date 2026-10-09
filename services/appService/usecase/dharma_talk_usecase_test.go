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

var testDharmaTalkCfg = config.DharmaTalkConfig{PageSize: 10}

func newDharmaTalkUsecase(repo *domain.MockDharmaTalkRepository) *DharmaTalkUsecase {
	return NewDharmaTalkUsecase(repo, testDharmaTalkCfg)
}

func TestDharmaTalkUsecase_List(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		total      int
		wantOffset int
		wantPages  int
	}{
		{"first page", 1, 36, 0, 4},
		{"second page", 2, 36, 10, 4},
		{"partial last page rounds up", 1, 11, 0, 2},
		{"page past the end is empty, not an error", 99, 36, 980, 4},
		{"empty catalogue has zero pages", 1, 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockDharmaTalkRepository)
			repo.On("List", mock.Anything, tt.wantOffset, 10).Return(nil, tt.total, nil)

			got, err := newDharmaTalkUsecase(repo).List(context.Background(), tt.page)

			require.NoError(t, err)
			assert.Equal(t, tt.page, got.Page)
			assert.Equal(t, 10, got.PageSize)
			assert.Equal(t, tt.total, got.Total)
			assert.Equal(t, tt.wantPages, got.TotalPages)
			assert.NotNil(t, got.Items, "items must be [] not nil")
			assert.Empty(t, got.Items)
			repo.AssertExpectations(t)
		})
	}

	t.Run("returns the repository items", func(t *testing.T) {
		repo := new(domain.MockDharmaTalkRepository)
		want := []domain.DharmaTalk{{ID: "a"}, {ID: "b"}}
		repo.On("List", mock.Anything, 0, 10).Return(want, 2, nil)
		got, err := newDharmaTalkUsecase(repo).List(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, want, got.Items)
	})

	for _, page := range []int{0, -1, math.MinInt, math.MaxInt32 + 1, math.MaxInt} {
		t.Run("rejects page out of range", func(t *testing.T) {
			repo := new(domain.MockDharmaTalkRepository)
			_, err := newDharmaTalkUsecase(repo).List(context.Background(), page)
			assert.ErrorIs(t, err, domain.ErrInvalidPage)
			repo.AssertNotCalled(t, "List", mock.Anything, mock.Anything, mock.Anything)
		})
	}

	t.Run("wraps repository errors", func(t *testing.T) {
		repo := new(domain.MockDharmaTalkRepository)
		boom := errors.New("db down")
		repo.On("List", mock.Anything, 0, 10).Return(nil, 0, boom)
		_, err := newDharmaTalkUsecase(repo).List(context.Background(), 1)
		assert.ErrorIs(t, err, boom)
	})
}

func TestDharmaTalkUsecase_Get(t *testing.T) {
	t.Run("returns the talk", func(t *testing.T) {
		repo := new(domain.MockDharmaTalkRepository)
		want := &domain.DharmaTalk{ID: "x", Title: "T"}
		repo.On("FindByID", mock.Anything, "x").Return(want, nil)
		got, err := newDharmaTalkUsecase(repo).Get(context.Background(), "x")
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("passes through ErrDharmaTalkNotFound", func(t *testing.T) {
		repo := new(domain.MockDharmaTalkRepository)
		repo.On("FindByID", mock.Anything, "nope").Return(nil, domain.ErrDharmaTalkNotFound)
		_, err := newDharmaTalkUsecase(repo).Get(context.Background(), "nope")
		assert.ErrorIs(t, err, domain.ErrDharmaTalkNotFound)
	})
}

func TestDharmaTalkUsecase_SeedIfEmpty(t *testing.T) {
	seed := []domain.DharmaTalk{{ID: "a"}, {ID: "b"}}
	loadOK := func() ([]domain.DharmaTalk, error) { return seed, nil }

	t.Run("seeds when the table has no row", func(t *testing.T) {
		repo := new(domain.MockDharmaTalkRepository)
		repo.On("Count", mock.Anything).Return(0, nil)
		repo.On("InsertMissing", mock.Anything, seed).Return(2, nil)
		n, err := newDharmaTalkUsecase(repo).SeedIfEmpty(context.Background(), loadOK)
		require.NoError(t, err)
		assert.Equal(t, 2, n)
		repo.AssertExpectations(t)
	})

	t.Run("does nothing, and does not load, when rows exist (hidden rows count)", func(t *testing.T) {
		repo := new(domain.MockDharmaTalkRepository)
		repo.On("Count", mock.Anything).Return(3, nil)
		loaded := false
		n, err := newDharmaTalkUsecase(repo).SeedIfEmpty(context.Background(), func() ([]domain.DharmaTalk, error) {
			loaded = true
			return seed, nil
		})
		require.NoError(t, err)
		assert.Zero(t, n)
		assert.False(t, loaded, "load must not run on a normal start")
		repo.AssertNotCalled(t, "InsertMissing", mock.Anything, mock.Anything)
	})

	t.Run("count error is wrapped and nothing is written", func(t *testing.T) {
		repo := new(domain.MockDharmaTalkRepository)
		boom := errors.New("relation does not exist")
		repo.On("Count", mock.Anything).Return(0, boom)
		_, err := newDharmaTalkUsecase(repo).SeedIfEmpty(context.Background(), loadOK)
		assert.ErrorIs(t, err, boom)
		repo.AssertNotCalled(t, "InsertMissing", mock.Anything, mock.Anything)
	})

	t.Run("load error is wrapped and nothing is written", func(t *testing.T) {
		repo := new(domain.MockDharmaTalkRepository)
		repo.On("Count", mock.Anything).Return(0, nil)
		boom := errors.New("bad seed")
		_, err := newDharmaTalkUsecase(repo).SeedIfEmpty(context.Background(), func() ([]domain.DharmaTalk, error) { return nil, boom })
		assert.ErrorIs(t, err, boom)
		repo.AssertNotCalled(t, "InsertMissing", mock.Anything, mock.Anything)
	})

	t.Run("insert error is wrapped", func(t *testing.T) {
		repo := new(domain.MockDharmaTalkRepository)
		repo.On("Count", mock.Anything).Return(0, nil)
		boom := errors.New("check violated")
		repo.On("InsertMissing", mock.Anything, seed).Return(0, boom)
		_, err := newDharmaTalkUsecase(repo).SeedIfEmpty(context.Background(), loadOK)
		assert.ErrorIs(t, err, boom)
	})
}
