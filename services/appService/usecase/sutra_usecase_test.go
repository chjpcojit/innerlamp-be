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

var testSutraCfg = config.SutraConfig{PageSize: 20, LinesPageSize: 200, LinesMaxPageSize: 500}

func newSutraUsecase(repo *domain.MockSutraRepository, deities *domain.MockDeityRepository) *SutraUsecase {
	return NewSutraUsecase(repo, deities, testSutraCfg)
}

func strp(s string) *string { return &s }
func intp(i int) *int       { return &i }

func TestSutraUsecase_List_Paging(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		total      int
		wantOffset int
		wantPages  int
	}{
		{"first page", 1, 45, 0, 3},
		{"second page", 2, 45, 20, 3},
		{"partial last page rounds up", 1, 21, 0, 2},
		{"page past the end is empty, not an error", 99, 45, 1960, 3},
		{"empty catalogue has zero pages", 1, 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockSutraRepository)
			repo.On("Count", mock.Anything, domain.SutraFilter{}).Return(tt.total, nil)
			repo.On("List", mock.Anything, domain.SutraFilter{}, tt.wantOffset, 20).Return(nil, nil)

			got, err := newSutraUsecase(repo, nil).List(context.Background(), domain.SutraFilterInput{}, tt.page)

			require.NoError(t, err)
			assert.NotNil(t, got.Items, "items must marshal as [] not null")
			assert.Equal(t, tt.page, got.Page)
			assert.Equal(t, 20, got.PageSize)
			assert.Equal(t, tt.total, got.Total)
			assert.Equal(t, tt.wantPages, got.TotalPages)
			repo.AssertExpectations(t)
		})
	}
}

func TestSutraUsecase_List_InvalidPage(t *testing.T) {
	for _, page := range []int{0, -1, math.MaxInt32 + 1} {
		repo := new(domain.MockSutraRepository)
		_, err := newSutraUsecase(repo, nil).List(context.Background(), domain.SutraFilterInput{}, page)
		assert.ErrorIs(t, err, domain.ErrInvalidPage, "page %d", page)
		repo.AssertNotCalled(t, "Count", mock.Anything, mock.Anything)
	}
}

func TestSutraUsecase_List_PassesParsedFilter(t *testing.T) {
	repo := new(domain.MockSutraRepository)
	trad, kind, purpose := domain.TraditionMahayana, domain.KindDharani, domain.PurposeTriChu
	daily, max := true, 15
	want := domain.SutraFilter{
		Tradition: &trad, Kind: &kind, Collection: strp("Chú"), Purpose: &purpose,
		Daily: &daily, MaxMinutes: &max, Lang: strp("vi"), Query: strp("bát nhã"),
	}
	repo.On("Count", mock.Anything, want).Return(1, nil)
	repo.On("List", mock.Anything, want, 0, 20).Return([]domain.Sutra{{ID: "chu-dai-bi"}}, nil)

	got, err := newSutraUsecase(repo, nil).List(context.Background(), domain.SutraFilterInput{
		Tradition: strp("MAHAYANA"), Kind: strp("DHARANI"), Collection: strp("Chú"), Purpose: strp("TRI_CHU"),
		Daily: strp("true"), MaxMinutes: strp("15"), Lang: strp("vi"), Q: strp("  bát nhã "),
	}, 1)

	require.NoError(t, err)
	require.Len(t, got.Items, 1)
	assert.Equal(t, "chu-dai-bi", got.Items[0].ID)
	repo.AssertExpectations(t)
}

func TestSutraUsecase_List_InvalidFilter(t *testing.T) {
	tests := []struct {
		name      string
		in        domain.SutraFilterInput
		wantField string
	}{
		{"unknown tradition", domain.SutraFilterInput{Tradition: strp("ZEN")}, "tradition"},
		{"lowercase tradition", domain.SutraFilterInput{Tradition: strp("mahayana")}, "tradition"},
		{"empty tradition", domain.SutraFilterInput{Tradition: strp("")}, "tradition"},
		{"unknown kind", domain.SutraFilterInput{Kind: strp("POEM")}, "kind"},
		{"empty collection", domain.SutraFilterInput{Collection: strp("")}, "collection"},
		{"unknown purpose", domain.SutraFilterInput{Purpose: strp("LUCK")}, "purpose"},
		{"daily is not a bool", domain.SutraFilterInput{Daily: strp("yes")}, "daily"},
		{"daily empty", domain.SutraFilterInput{Daily: strp("")}, "daily"},
		{"max_minutes not a number", domain.SutraFilterInput{MaxMinutes: strp("abc")}, "max_minutes"},
		{"max_minutes zero", domain.SutraFilterInput{MaxMinutes: strp("0")}, "max_minutes"},
		{"max_minutes negative", domain.SutraFilterInput{MaxMinutes: strp("-5")}, "max_minutes"},
		{"max_minutes overflows int32", domain.SutraFilterInput{MaxMinutes: strp("2147483648")}, "max_minutes"},
		{"empty lang", domain.SutraFilterInput{Lang: strp("")}, "lang"},
		{"empty q", domain.SutraFilterInput{Q: strp("")}, "q"},
		{"blank q", domain.SutraFilterInput{Q: strp("   ")}, "q"},
		{"q over 100 characters", domain.SutraFilterInput{Q: strp(strings.Repeat("ả", 101))}, "q"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockSutraRepository)
			_, err := newSutraUsecase(repo, nil).List(context.Background(), tt.in, 1)

			require.ErrorIs(t, err, domain.ErrInvalidSutraFilter)
			var fe *domain.InvalidSutraFilterError
			require.ErrorAs(t, err, &fe)
			assert.Equal(t, tt.wantField, fe.Field)
			repo.AssertNotCalled(t, "Count", mock.Anything, mock.Anything)
		})
	}
}

func TestSutraUsecase_List_WrapsRepositoryErrors(t *testing.T) {
	boom := errors.New("db down")
	t.Run("count", func(t *testing.T) {
		repo := new(domain.MockSutraRepository)
		repo.On("Count", mock.Anything, mock.Anything).Return(0, boom)
		_, err := newSutraUsecase(repo, nil).List(context.Background(), domain.SutraFilterInput{}, 1)
		assert.ErrorIs(t, err, boom)
	})
	t.Run("list", func(t *testing.T) {
		repo := new(domain.MockSutraRepository)
		repo.On("Count", mock.Anything, mock.Anything).Return(3, nil)
		repo.On("List", mock.Anything, mock.Anything, 0, 20).Return(nil, boom)
		_, err := newSutraUsecase(repo, nil).List(context.Background(), domain.SutraFilterInput{}, 1)
		assert.ErrorIs(t, err, boom)
	})
}

func TestSutraUsecase_Get(t *testing.T) {
	t.Run("returns the sutra", func(t *testing.T) {
		repo := new(domain.MockSutraRepository)
		want := &domain.Sutra{ID: "tam-kinh"}
		repo.On("FindByID", mock.Anything, "tam-kinh").Return(want, nil)
		got, err := newSutraUsecase(repo, nil).Get(context.Background(), "tam-kinh")
		require.NoError(t, err)
		assert.Same(t, want, got)
	})
	t.Run("not found stays matchable", func(t *testing.T) {
		repo := new(domain.MockSutraRepository)
		repo.On("FindByID", mock.Anything, "x").Return(nil, domain.ErrSutraNotFound)
		_, err := newSutraUsecase(repo, nil).Get(context.Background(), "x")
		assert.ErrorIs(t, err, domain.ErrSutraNotFound)
	})
}

func TestSutraUsecase_Lines_Paging(t *testing.T) {
	tr := &domain.SutraTranslation{ID: 7, Key: "vi-chant", LineCount: 450}
	tests := []struct {
		name         string
		page         int
		pageSize     *int
		wantFrom     int64
		wantTo       int64
		wantSize     int
		wantTotalPgs int
	}{
		{"defaults to page 1 of the configured size", 1, nil, 0, 200, 200, 3},
		{"second page", 2, nil, 200, 400, 200, 3},
		{"last partial page", 3, nil, 400, 600, 200, 3},
		{"explicit page size", 2, intp(100), 100, 200, 100, 5},
		{"page size above the cap is clamped, not rejected", 1, intp(5000), 0, 500, 500, 1},
		{"page past the end is empty, not an error", 99, nil, 19600, 19800, 200, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockSutraRepository)
			repo.On("FindTranslation", mock.Anything, "tam-kinh", "vi-chant").Return(tr, nil)
			repo.On("ListLines", mock.Anything, int64(7), tt.wantFrom, tt.wantTo).Return(nil, nil)

			got, err := newSutraUsecase(repo, nil).Lines(context.Background(), "tam-kinh", "vi-chant", tt.page, tt.pageSize)

			require.NoError(t, err)
			assert.NotNil(t, got.Items, "items must marshal as [] not null")
			assert.Equal(t, "vi-chant", got.Translation)
			assert.Equal(t, tt.page, got.Page)
			assert.Equal(t, tt.wantSize, got.PageSize)
			assert.Equal(t, 450, got.Total)
			assert.Equal(t, tt.wantTotalPgs, got.TotalPages)
			repo.AssertExpectations(t)
		})
	}
}

// page * page_size does not fit in int32 for the largest accepted page; the range
// handed to the repository must not wrap.
func TestSutraUsecase_Lines_RangeDoesNotOverflow(t *testing.T) {
	repo := new(domain.MockSutraRepository)
	repo.On("FindTranslation", mock.Anything, "s", "k").Return(&domain.SutraTranslation{ID: 1, Key: "k", LineCount: 10}, nil)
	wantFrom := int64(math.MaxInt32-1) * 500
	repo.On("ListLines", mock.Anything, int64(1), wantFrom, wantFrom+500).Return(nil, nil)

	_, err := newSutraUsecase(repo, nil).Lines(context.Background(), "s", "k", math.MaxInt32, intp(500))

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestSutraUsecase_Lines_Invalid(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		page     int
		pageSize *int
		want     error
	}{
		{"empty translation key", "", 1, nil, domain.ErrInvalidTranslation},
		{"page zero", "k", 0, nil, domain.ErrInvalidPage},
		{"page negative", "k", -1, nil, domain.ErrInvalidPage},
		{"page over int32", "k", math.MaxInt32 + 1, nil, domain.ErrInvalidPage},
		{"page size zero", "k", 1, intp(0), domain.ErrInvalidPageSize},
		{"page size negative", "k", 1, intp(-3), domain.ErrInvalidPageSize},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockSutraRepository)
			_, err := newSutraUsecase(repo, nil).Lines(context.Background(), "s", tt.key, tt.page, tt.pageSize)
			assert.ErrorIs(t, err, tt.want)
			repo.AssertNotCalled(t, "FindTranslation", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestSutraUsecase_Lines_NotFound(t *testing.T) {
	for _, notFound := range []error{domain.ErrSutraNotFound, domain.ErrSutraTranslationNotFound} {
		t.Run(notFound.Error(), func(t *testing.T) {
			repo := new(domain.MockSutraRepository)
			repo.On("FindTranslation", mock.Anything, "s", "k").Return(nil, notFound)
			_, err := newSutraUsecase(repo, nil).Lines(context.Background(), "s", "k", 1, nil)
			assert.ErrorIs(t, err, notFound)
			repo.AssertNotCalled(t, "ListLines", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestSutraUsecase_Lines_WrapsListError(t *testing.T) {
	boom := errors.New("db down")
	repo := new(domain.MockSutraRepository)
	repo.On("FindTranslation", mock.Anything, "s", "k").Return(&domain.SutraTranslation{ID: 1, Key: "k", LineCount: 5}, nil)
	repo.On("ListLines", mock.Anything, int64(1), int64(0), int64(200)).Return(nil, boom)
	_, err := newSutraUsecase(repo, nil).Lines(context.Background(), "s", "k", 1, nil)
	assert.ErrorIs(t, err, boom)
}

func TestSutraUsecase_ListByDeity(t *testing.T) {
	deity := &domain.Deity{ID: "bodhisattva_avalokiteshvara"}

	t.Run("pages the linked sutras and carries the role", func(t *testing.T) {
		repo, deities := new(domain.MockSutraRepository), new(domain.MockDeityRepository)
		deities.On("FindByID", mock.Anything, deity.ID).Return(deity, nil)
		repo.On("CountByDeity", mock.Anything, deity.ID, (*domain.DeityRole)(nil)).Return(45, nil)
		items := []domain.DeitySutra{{Sutra: domain.Sutra{ID: "pho-mon"}, Role: domain.RoleSubject}}
		repo.On("ListByDeity", mock.Anything, deity.ID, (*domain.DeityRole)(nil), 20, 20).Return(items, nil)

		got, err := newSutraUsecase(repo, deities).ListByDeity(context.Background(), deity.ID, nil, 2)

		require.NoError(t, err)
		assert.Equal(t, items, got.Items)
		assert.Equal(t, 2, got.Page)
		assert.Equal(t, 20, got.PageSize)
		assert.Equal(t, 45, got.Total)
		assert.Equal(t, 3, got.TotalPages)
		repo.AssertExpectations(t)
	})

	t.Run("role filter is passed through", func(t *testing.T) {
		repo, deities := new(domain.MockSutraRepository), new(domain.MockDeityRepository)
		deities.On("FindByID", mock.Anything, deity.ID).Return(deity, nil)
		role := domain.RoleSubject
		repo.On("CountByDeity", mock.Anything, deity.ID, &role).Return(0, nil)
		repo.On("ListByDeity", mock.Anything, deity.ID, &role, 0, 20).Return(nil, nil)

		got, err := newSutraUsecase(repo, deities).ListByDeity(context.Background(), deity.ID, strp("SUBJECT"), 1)

		require.NoError(t, err)
		assert.NotNil(t, got.Items, "items must marshal as [] not null")
		assert.Equal(t, 0, got.TotalPages)
		repo.AssertExpectations(t)
	})

	t.Run("invalid role names the field", func(t *testing.T) {
		for _, role := range []string{"", "speaker", "AUTHOR"} {
			repo, deities := new(domain.MockSutraRepository), new(domain.MockDeityRepository)
			_, err := newSutraUsecase(repo, deities).ListByDeity(context.Background(), deity.ID, strp(role), 1)
			require.ErrorIs(t, err, domain.ErrInvalidSutraFilter, "role %q", role)
			var fe *domain.InvalidSutraFilterError
			require.ErrorAs(t, err, &fe)
			assert.Equal(t, "role", fe.Field)
			deities.AssertNotCalled(t, "FindByID", mock.Anything, mock.Anything)
		}
	})

	t.Run("invalid page", func(t *testing.T) {
		for _, page := range []int{0, -1, math.MaxInt32 + 1} {
			repo, deities := new(domain.MockSutraRepository), new(domain.MockDeityRepository)
			_, err := newSutraUsecase(repo, deities).ListByDeity(context.Background(), deity.ID, nil, page)
			assert.ErrorIs(t, err, domain.ErrInvalidPage, "page %d", page)
			deities.AssertNotCalled(t, "FindByID", mock.Anything, mock.Anything)
		}
	})

	t.Run("unknown deity stops before any sutra query", func(t *testing.T) {
		repo, deities := new(domain.MockSutraRepository), new(domain.MockDeityRepository)
		deities.On("FindByID", mock.Anything, "nobody").Return(nil, domain.ErrDeityNotFound)
		_, err := newSutraUsecase(repo, deities).ListByDeity(context.Background(), "nobody", nil, 1)
		assert.ErrorIs(t, err, domain.ErrDeityNotFound)
		repo.AssertNotCalled(t, "CountByDeity", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("wraps repository errors", func(t *testing.T) {
		boom := errors.New("db down")
		repo, deities := new(domain.MockSutraRepository), new(domain.MockDeityRepository)
		deities.On("FindByID", mock.Anything, deity.ID).Return(deity, nil)
		repo.On("CountByDeity", mock.Anything, deity.ID, (*domain.DeityRole)(nil)).Return(0, boom)
		_, err := newSutraUsecase(repo, deities).ListByDeity(context.Background(), deity.ID, nil, 1)
		assert.ErrorIs(t, err, boom)
	})
}

func seedWithDeities(ids ...string) []domain.SutraSeed {
	var links []domain.SutraDeity
	for _, id := range ids {
		links = append(links, domain.SutraDeity{DeityID: id, Role: domain.RoleSubject})
	}
	return []domain.SutraSeed{{Sutra: domain.Sutra{ID: "s", Deities: links}}}
}

func TestSutraUsecase_SeedIfEmpty(t *testing.T) {
	t.Run("seeds an empty table", func(t *testing.T) {
		repo := new(domain.MockSutraRepository)
		seeds := seedWithDeities("b", "a", "b")
		want := domain.SutraSeedResult{Sutras: 1, Translations: 2, Lines: 30, DeityLinks: 3}
		repo.On("Count", mock.Anything, domain.SutraFilter{}).Return(0, nil)
		repo.On("MissingDeities", mock.Anything, []string{"a", "b"}).Return(nil, nil)
		repo.On("InsertAll", mock.Anything, seeds).Return(want, nil)

		got, err := newSutraUsecase(repo, nil).SeedIfEmpty(context.Background(), func() ([]domain.SutraSeed, error) { return seeds, nil })

		require.NoError(t, err)
		assert.Equal(t, want, got)
		repo.AssertExpectations(t)
	})

	t.Run("skips everything when the table already has sutras", func(t *testing.T) {
		repo := new(domain.MockSutraRepository)
		repo.On("Count", mock.Anything, domain.SutraFilter{}).Return(153, nil)
		called := false

		got, err := newSutraUsecase(repo, nil).SeedIfEmpty(context.Background(), func() ([]domain.SutraSeed, error) {
			called = true
			return nil, nil
		})

		require.NoError(t, err)
		assert.Equal(t, domain.SutraSeedResult{}, got)
		assert.False(t, called, "the seed must not even be parsed on a normal start")
		repo.AssertNotCalled(t, "InsertAll", mock.Anything, mock.Anything)
		repo.AssertNotCalled(t, "MissingDeities", mock.Anything, mock.Anything)
	})

	t.Run("aborts and names the deities that are not seeded", func(t *testing.T) {
		repo := new(domain.MockSutraRepository)
		repo.On("Count", mock.Anything, domain.SutraFilter{}).Return(0, nil)
		repo.On("MissingDeities", mock.Anything, []string{"a", "b"}).Return([]string{"b"}, nil)

		_, err := newSutraUsecase(repo, nil).SeedIfEmpty(context.Background(), func() ([]domain.SutraSeed, error) {
			return seedWithDeities("a", "b"), nil
		})

		require.ErrorIs(t, err, domain.ErrSutraSeedDeitiesMissing)
		assert.Contains(t, err.Error(), "b")
		assert.Contains(t, err.Error(), "cmd/seeddeities")
		repo.AssertNotCalled(t, "InsertAll", mock.Anything, mock.Anything)
	})

	t.Run("a seed without deity links never asks for deities", func(t *testing.T) {
		repo := new(domain.MockSutraRepository)
		seeds := seedWithDeities()
		repo.On("Count", mock.Anything, domain.SutraFilter{}).Return(0, nil)
		repo.On("InsertAll", mock.Anything, seeds).Return(domain.SutraSeedResult{Sutras: 1}, nil)

		_, err := newSutraUsecase(repo, nil).SeedIfEmpty(context.Background(), func() ([]domain.SutraSeed, error) { return seeds, nil })

		require.NoError(t, err)
		repo.AssertNotCalled(t, "MissingDeities", mock.Anything, mock.Anything)
	})

	t.Run("wraps every failure", func(t *testing.T) {
		boom := errors.New("boom")
		cases := []struct {
			name  string
			setup func(r *domain.MockSutraRepository)
			load  func() ([]domain.SutraSeed, error)
		}{
			{"count", func(r *domain.MockSutraRepository) {
				r.On("Count", mock.Anything, mock.Anything).Return(0, boom)
			}, nil},
			{"load", func(r *domain.MockSutraRepository) {
				r.On("Count", mock.Anything, mock.Anything).Return(0, nil)
			}, func() ([]domain.SutraSeed, error) { return nil, boom }},
			{"missing-deities lookup", func(r *domain.MockSutraRepository) {
				r.On("Count", mock.Anything, mock.Anything).Return(0, nil)
				r.On("MissingDeities", mock.Anything, mock.Anything).Return(nil, boom)
			}, func() ([]domain.SutraSeed, error) { return seedWithDeities("a"), nil }},
			{"insert", func(r *domain.MockSutraRepository) {
				r.On("Count", mock.Anything, mock.Anything).Return(0, nil)
				r.On("InsertAll", mock.Anything, mock.Anything).Return(nil, boom)
			}, func() ([]domain.SutraSeed, error) { return seedWithDeities(), nil }},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				repo := new(domain.MockSutraRepository)
				c.setup(repo)
				_, err := newSutraUsecase(repo, nil).SeedIfEmpty(context.Background(), c.load)
				assert.ErrorIs(t, err, boom)
			})
		}
	})
}
