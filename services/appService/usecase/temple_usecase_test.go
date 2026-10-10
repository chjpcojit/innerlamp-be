package usecase

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

var testTempleCfg = config.TempleConfig{TopLimit: 10, PageSize: 20, OptionsPageSize: 100}

func newTempleUsecase(repo *domain.MockTempleRepository) *TempleUsecase {
	return NewTempleUsecase(repo, testTempleCfg)
}

func TestTempleUsecase_Top(t *testing.T) {
	t.Run("asks the repository for the configured limit", func(t *testing.T) {
		repo := new(domain.MockTempleRepository)
		want := []domain.Temple{{ID: "a"}, {ID: "b"}}
		repo.On("ListTop", mock.Anything, 10).Return(want, nil)

		got, err := newTempleUsecase(repo).Top(context.Background())

		require.NoError(t, err)
		assert.Equal(t, want, got)
		repo.AssertExpectations(t)
	})

	t.Run("wraps repository errors", func(t *testing.T) {
		repo := new(domain.MockTempleRepository)
		boom := errors.New("db down")
		repo.On("ListTop", mock.Anything, 10).Return(nil, boom)

		_, err := newTempleUsecase(repo).Top(context.Background())

		assert.ErrorIs(t, err, boom)
	})
}

func TestTempleUsecase_Options(t *testing.T) {
	t.Run("first page asks the repository for offset 0 and the options page size", func(t *testing.T) {
		repo := new(domain.MockTempleRepository)
		want := []domain.TempleOption{{ID: "a", Name: "Chùa A", Province: "Hà Nội"}, {ID: "b", Name: "Chùa B"}}
		repo.On("ListOptions", mock.Anything, "", 0, 100).Return(want, 250, nil)

		got, err := newTempleUsecase(repo).Options(context.Background(), nil, 1)

		require.NoError(t, err)
		assert.Equal(t, want, got.Items)
		assert.Equal(t, 1, got.Page)
		assert.Equal(t, 100, got.PageSize)
		assert.Equal(t, 250, got.Total)
		assert.Equal(t, 3, got.TotalPages)
		repo.AssertExpectations(t)
	})

	t.Run("third page offsets by two pages", func(t *testing.T) {
		repo := new(domain.MockTempleRepository)
		repo.On("ListOptions", mock.Anything, "", 200, 100).Return([]domain.TempleOption{{ID: "z"}}, 250, nil)

		got, err := newTempleUsecase(repo).Options(context.Background(), nil, 3)

		require.NoError(t, err)
		assert.Equal(t, 3, got.Page)
		repo.AssertExpectations(t)
	})

	t.Run("page past the end is an empty page, not an error", func(t *testing.T) {
		repo := new(domain.MockTempleRepository)
		repo.On("ListOptions", mock.Anything, "", 9900, 100).Return(nil, 250, nil)

		got, err := newTempleUsecase(repo).Options(context.Background(), nil, 100)

		require.NoError(t, err)
		assert.NotNil(t, got.Items)
		assert.Empty(t, got.Items)
		assert.Equal(t, 250, got.Total)
		assert.Equal(t, 3, got.TotalPages)
	})

	t.Run("empty catalogue has zero pages", func(t *testing.T) {
		repo := new(domain.MockTempleRepository)
		repo.On("ListOptions", mock.Anything, "", 0, 100).Return(nil, 0, nil)

		got, err := newTempleUsecase(repo).Options(context.Background(), nil, 1)

		require.NoError(t, err)
		assert.Empty(t, got.Items)
		assert.Equal(t, 0, got.TotalPages)
	})

	t.Run("page < 1 or too large is ErrInvalidPage and never reaches the repository", func(t *testing.T) {
		for _, page := range []int{0, -1, math.MaxInt32 + 1} {
			repo := new(domain.MockTempleRepository)
			_, err := newTempleUsecase(repo).Options(context.Background(), nil, page)
			assert.ErrorIs(t, err, domain.ErrInvalidPage, "page %d", page)
			repo.AssertNotCalled(t, "ListOptions")
		}
	})

	t.Run("wraps repository errors", func(t *testing.T) {
		repo := new(domain.MockTempleRepository)
		boom := errors.New("db down")
		repo.On("ListOptions", mock.Anything, "", 0, 100).Return(nil, 0, boom)

		_, err := newTempleUsecase(repo).Options(context.Background(), nil, 1)

		assert.ErrorIs(t, err, boom)
	})
}

func TestTempleUsecase_Options_Query(t *testing.T) {
	t.Run("passes a trimmed term to the repository and pages over the matches", func(t *testing.T) {
		repo := new(domain.MockTempleRepository)
		want := []domain.TempleOption{{ID: "a", Name: "Chùa Bích Động", Province: "Ninh Bình"}}
		repo.On("ListOptions", mock.Anything, "bich dong", 0, 100).Return(want, 1, nil)

		got, err := newTempleUsecase(repo).Options(context.Background(), strPtr("  bich dong \t"), 1)

		require.NoError(t, err)
		assert.Equal(t, want, got.Items)
		assert.Equal(t, 1, got.Total, "total counts the matches")
		assert.Equal(t, 1, got.TotalPages)
		repo.AssertExpectations(t)
	})

	t.Run("no match is an empty page with zero pages", func(t *testing.T) {
		repo := new(domain.MockTempleRepository)
		repo.On("ListOptions", mock.Anything, "zzz", 0, 100).Return(nil, 0, nil)

		got, err := newTempleUsecase(repo).Options(context.Background(), strPtr("zzz"), 1)

		require.NoError(t, err)
		assert.NotNil(t, got.Items)
		assert.Empty(t, got.Items)
		assert.Equal(t, 0, got.TotalPages)
	})

	t.Run("exactly 100 characters is allowed", func(t *testing.T) {
		repo := new(domain.MockTempleRepository)
		term := strings.Repeat("ế", 100)
		repo.On("ListOptions", mock.Anything, term, 0, 100).Return(nil, 0, nil)

		_, err := newTempleUsecase(repo).Options(context.Background(), &term, 1)

		assert.NoError(t, err)
	})

	t.Run("present but empty, blank, or over 100 characters is ErrInvalidTempleQuery", func(t *testing.T) {
		for name, q := range map[string]string{
			"empty":    "",
			"blank":    "   \t ",
			"too long": strings.Repeat("ế", 101),
		} {
			repo := new(domain.MockTempleRepository)
			_, err := newTempleUsecase(repo).Options(context.Background(), strPtr(q), 1)
			assert.ErrorIs(t, err, domain.ErrInvalidTempleQuery, name)
			repo.AssertNotCalled(t, "ListOptions")
		}
	})

	t.Run("an invalid page wins over a valid term and never reaches the repository", func(t *testing.T) {
		repo := new(domain.MockTempleRepository)
		_, err := newTempleUsecase(repo).Options(context.Background(), strPtr("bich"), 0)
		assert.ErrorIs(t, err, domain.ErrInvalidPage)
		repo.AssertNotCalled(t, "ListOptions")
	})
}

func TestTempleUsecase_List(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		total      int
		wantOffset int
		wantPages  int
	}{
		{"first page", 1, 180, 0, 9},
		{"third page", 3, 180, 40, 9},
		{"last partial page", 2, 21, 20, 2},
		{"page past the end is empty, not an error", 99, 180, 1960, 9},
		{"empty catalogue has zero pages", 1, 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockTempleRepository)
			items := []domain.Temple{{ID: "x"}}
			repo.On("List", mock.Anything, tt.wantOffset, 20).Return(items, tt.total, nil)

			got, err := newTempleUsecase(repo).List(context.Background(), tt.page)

			require.NoError(t, err)
			assert.Equal(t, items, got.Items)
			assert.Equal(t, tt.page, got.Page)
			assert.Equal(t, 20, got.PageSize)
			assert.Equal(t, tt.total, got.Total)
			assert.Equal(t, tt.wantPages, got.TotalPages)
		})
	}

	t.Run("rejects page < 1 without touching the repository", func(t *testing.T) {
		for _, p := range []int{0, -1, math.MaxInt32 + 1} {
			repo := new(domain.MockTempleRepository)
			_, err := newTempleUsecase(repo).List(context.Background(), p)
			assert.ErrorIs(t, err, domain.ErrInvalidPage)
			repo.AssertNotCalled(t, "List")
		}
	})

	t.Run("items is an empty slice, never nil, so it encodes as []", func(t *testing.T) {
		repo := new(domain.MockTempleRepository)
		repo.On("List", mock.Anything, 20, 20).Return(nil, 5, nil)

		got, err := newTempleUsecase(repo).List(context.Background(), 2)

		require.NoError(t, err)
		assert.NotNil(t, got.Items)
		assert.Empty(t, got.Items)
	})
}

func TestTempleUsecase_Get(t *testing.T) {
	t.Run("returns the temple", func(t *testing.T) {
		repo := new(domain.MockTempleRepository)
		want := &domain.Temple{ID: "temple_north_001"}
		repo.On("FindByID", mock.Anything, "temple_north_001").Return(want, nil)

		got, err := newTempleUsecase(repo).Get(context.Background(), "temple_north_001")

		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("passes ErrTempleNotFound through", func(t *testing.T) {
		repo := new(domain.MockTempleRepository)
		repo.On("FindByID", mock.Anything, "nope").Return(nil, domain.ErrTempleNotFound)

		_, err := newTempleUsecase(repo).Get(context.Background(), "nope")

		assert.ErrorIs(t, err, domain.ErrTempleNotFound)
	})
}
