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

var testArticleCfg = config.ArticleConfig{TopLimit: 5, PageSize: 10}

func newArticleUsecase(repo *domain.MockArticleRepository) *ArticleUsecase {
	return NewArticleUsecase(repo, testArticleCfg)
}

func TestArticleUsecase_Top(t *testing.T) {
	t.Run("asks the repository for the configured limit", func(t *testing.T) {
		repo := new(domain.MockArticleRepository)
		want := []domain.Article{{ID: "a"}, {ID: "b"}}
		repo.On("ListTop", mock.Anything, 5).Return(want, nil)

		got, err := newArticleUsecase(repo).Top(context.Background())

		require.NoError(t, err)
		assert.Equal(t, want, got)
		repo.AssertExpectations(t)
	})

	t.Run("wraps repository errors", func(t *testing.T) {
		repo := new(domain.MockArticleRepository)
		boom := errors.New("db down")
		repo.On("ListTop", mock.Anything, 5).Return(nil, boom)
		_, err := newArticleUsecase(repo).Top(context.Background())
		assert.ErrorIs(t, err, boom)
	})
}

func TestArticleUsecase_List(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		total      int
		wantOffset int
		wantPages  int
	}{
		{"first page", 1, 30, 0, 3},
		{"second page", 2, 30, 10, 3},
		{"partial last page rounds up", 1, 11, 0, 2},
		{"page past the end is empty, not an error", 99, 30, 980, 3},
		{"empty catalogue has zero pages", 1, 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockArticleRepository)
			repo.On("List", mock.Anything, tt.wantOffset, 10).Return(nil, tt.total, nil)

			got, err := newArticleUsecase(repo).List(context.Background(), tt.page)

			require.NoError(t, err)
			assert.NotNil(t, got.Items, "items must marshal as [] not null")
			assert.Equal(t, tt.page, got.Page)
			assert.Equal(t, 10, got.PageSize)
			assert.Equal(t, tt.total, got.Total)
			assert.Equal(t, tt.wantPages, got.TotalPages)
			repo.AssertExpectations(t)
		})
	}

	for _, page := range []int{0, -1, math.MaxInt32 + 1} {
		t.Run("invalid page", func(t *testing.T) {
			repo := new(domain.MockArticleRepository)
			_, err := newArticleUsecase(repo).List(context.Background(), page)
			assert.ErrorIs(t, err, domain.ErrInvalidPage)
			repo.AssertNotCalled(t, "List")
		})
	}

	t.Run("wraps repository errors", func(t *testing.T) {
		repo := new(domain.MockArticleRepository)
		boom := errors.New("db down")
		repo.On("List", mock.Anything, 0, 10).Return(nil, 0, boom)
		_, err := newArticleUsecase(repo).List(context.Background(), 1)
		assert.ErrorIs(t, err, boom)
	})
}

func TestArticleUsecase_Get(t *testing.T) {
	t.Run("returns the article", func(t *testing.T) {
		repo := new(domain.MockArticleRepository)
		want := &domain.Article{ID: "vipassana-nghe-thuat-song"}
		repo.On("FindByID", mock.Anything, "vipassana-nghe-thuat-song").Return(want, nil)
		got, err := newArticleUsecase(repo).Get(context.Background(), "vipassana-nghe-thuat-song")
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("not found stays matchable through the wrap", func(t *testing.T) {
		repo := new(domain.MockArticleRepository)
		repo.On("FindByID", mock.Anything, "x").Return(nil, domain.ErrArticleNotFound)
		_, err := newArticleUsecase(repo).Get(context.Background(), "x")
		assert.ErrorIs(t, err, domain.ErrArticleNotFound)
	})
}

func TestArticleUsecase_SeedIfEmpty(t *testing.T) {
	seed := []domain.Article{{ID: "a"}, {ID: "b"}}
	load := func() ([]domain.Article, error) { return seed, nil }

	t.Run("empty table: loads and inserts the seed", func(t *testing.T) {
		repo := new(domain.MockArticleRepository)
		repo.On("Count", mock.Anything).Return(0, nil)
		repo.On("InsertMissing", mock.Anything, seed).Return(2, nil)

		n, err := newArticleUsecase(repo).SeedIfEmpty(context.Background(), load)

		require.NoError(t, err)
		assert.Equal(t, 2, n)
		repo.AssertExpectations(t)
	})

	t.Run("non-empty table: does not even load the seed", func(t *testing.T) {
		repo := new(domain.MockArticleRepository)
		repo.On("Count", mock.Anything).Return(3, nil)
		loaded := false

		n, err := newArticleUsecase(repo).SeedIfEmpty(context.Background(), func() ([]domain.Article, error) {
			loaded = true
			return seed, nil
		})

		require.NoError(t, err)
		assert.Zero(t, n)
		assert.False(t, loaded)
		repo.AssertNotCalled(t, "InsertMissing")
	})

	t.Run("count error", func(t *testing.T) {
		repo := new(domain.MockArticleRepository)
		boom := errors.New("db down")
		repo.On("Count", mock.Anything).Return(0, boom)
		_, err := newArticleUsecase(repo).SeedIfEmpty(context.Background(), load)
		assert.ErrorIs(t, err, boom)
	})

	t.Run("load error", func(t *testing.T) {
		repo := new(domain.MockArticleRepository)
		boom := errors.New("bad json")
		repo.On("Count", mock.Anything).Return(0, nil)
		_, err := newArticleUsecase(repo).SeedIfEmpty(context.Background(), func() ([]domain.Article, error) { return nil, boom })
		assert.ErrorIs(t, err, boom)
		repo.AssertNotCalled(t, "InsertMissing")
	})

	t.Run("insert error", func(t *testing.T) {
		repo := new(domain.MockArticleRepository)
		boom := errors.New("tx failed")
		repo.On("Count", mock.Anything).Return(0, nil)
		repo.On("InsertMissing", mock.Anything, seed).Return(0, boom)
		_, err := newArticleUsecase(repo).SeedIfEmpty(context.Background(), load)
		assert.ErrorIs(t, err, boom)
	})
}
