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

var testGalleryCfg = config.GalleryConfig{TopLimit: 10, PageSize: 20}

func newGalleryUsecase(repo *domain.MockGalleryRepository) *GalleryUsecase {
	return NewGalleryUsecase(repo, testGalleryCfg)
}

func TestGalleryUsecase_Top(t *testing.T) {
	t.Run("asks the repository for the configured limit", func(t *testing.T) {
		repo := new(domain.MockGalleryRepository)
		want := []domain.GalleryImage{{ID: "a"}, {ID: "b"}}
		repo.On("ListTop", mock.Anything, 10).Return(want, nil)

		got, err := newGalleryUsecase(repo).Top(context.Background())

		require.NoError(t, err)
		assert.Equal(t, want, got)
		repo.AssertExpectations(t)
	})

	t.Run("wraps repository errors", func(t *testing.T) {
		repo := new(domain.MockGalleryRepository)
		boom := errors.New("db down")
		repo.On("ListTop", mock.Anything, 10).Return(nil, boom)

		_, err := newGalleryUsecase(repo).Top(context.Background())

		assert.ErrorIs(t, err, boom)
	})
}

func TestGalleryUsecase_List(t *testing.T) {
	tests := []struct {
		name       string
		imageType  string
		page       int
		total      int
		wantOffset int
		wantPages  int
	}{
		{"first page, every type", "", 1, 86, 0, 5},
		{"second page", "", 2, 45, 20, 3},
		{"filtered by type", domain.GalleryThien, 1, 12, 0, 1},
		{"page past the end is empty, not an error", "", 99, 12, 1960, 1},
		{"empty gallery has zero pages", "", 1, 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockGalleryRepository)
			repo.On("List", mock.Anything, tt.imageType, tt.wantOffset, 20).Return(nil, tt.total, nil)

			got, err := newGalleryUsecase(repo).List(context.Background(), tt.imageType, tt.page)

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
			repo := new(domain.MockGalleryRepository)
			_, err := newGalleryUsecase(repo).List(context.Background(), "", page)
			assert.ErrorIs(t, err, domain.ErrInvalidPage)
			repo.AssertNotCalled(t, "List")
		})
	}

	for _, typ := range []string{"chua", "TEMPLE", " "} {
		t.Run("invalid type "+typ, func(t *testing.T) {
			repo := new(domain.MockGalleryRepository)
			_, err := newGalleryUsecase(repo).List(context.Background(), typ, 1)
			assert.ErrorIs(t, err, domain.ErrInvalidGalleryType)
			repo.AssertNotCalled(t, "List")
		})
	}

	t.Run("wraps repository errors", func(t *testing.T) {
		repo := new(domain.MockGalleryRepository)
		boom := errors.New("db down")
		repo.On("List", mock.Anything, "", 0, 20).Return(nil, 0, boom)
		_, err := newGalleryUsecase(repo).List(context.Background(), "", 1)
		assert.ErrorIs(t, err, boom)
	})
}

func TestGalleryUsecase_Get(t *testing.T) {
	t.Run("returns the image", func(t *testing.T) {
		repo := new(domain.MockGalleryRepository)
		want := &domain.GalleryImage{ID: "gallery_001"}
		repo.On("FindByID", mock.Anything, "gallery_001").Return(want, nil)
		got, err := newGalleryUsecase(repo).Get(context.Background(), "gallery_001")
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("not found stays matchable through the wrap", func(t *testing.T) {
		repo := new(domain.MockGalleryRepository)
		repo.On("FindByID", mock.Anything, "x").Return(nil, domain.ErrGalleryImageNotFound)
		_, err := newGalleryUsecase(repo).Get(context.Background(), "x")
		assert.ErrorIs(t, err, domain.ErrGalleryImageNotFound)
	})
}
