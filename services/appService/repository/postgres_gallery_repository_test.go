package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func mkGallery(id, typ string, order int) domain.GalleryImage {
	return domain.GalleryImage{
		ID: id, Type: typ, SortOrder: order, Title: "Ảnh " + id, Location: "Việt Nam",
		ThumbnailURL: "https://x/" + id + "_t.jpg", ImageURL: "https://x/" + id + ".jpg",
		Author: "A", License: "CC0", SourceURL: "https://x/src/" + id,
	}
}

func newGalleryRepo(t *testing.T) domain.GalleryRepository {
	t.Helper()
	return NewGalleryRepository(connectTestDB(t))
}

func galleryIDs(gs []domain.GalleryImage) []string {
	out := make([]string, 0, len(gs))
	for _, g := range gs {
		out = append(out, g.ID)
	}
	return out
}

func TestGalleryRepository_UpsertAndFind(t *testing.T) {
	repo := newGalleryRepo(t)
	ctx := context.Background()

	g := mkGallery("g1", domain.GalleryChua, 0)
	require.NoError(t, repo.Upsert(ctx, []domain.GalleryImage{g}))

	got, err := repo.FindByID(ctx, "g1")
	require.NoError(t, err)
	assert.Equal(t, g, *got)

	// Upsert again with a change: updates in place, no duplicate row.
	g.Title = "Renamed"
	require.NoError(t, repo.Upsert(ctx, []domain.GalleryImage{g}))
	got, err = repo.FindByID(ctx, "g1")
	require.NoError(t, err)
	assert.Equal(t, "Renamed", got.Title)
	_, total, err := repo.List(ctx, "", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestGalleryRepository_FindByID_NotFound(t *testing.T) {
	repo := newGalleryRepo(t)
	_, err := repo.FindByID(context.Background(), "missing")
	assert.ErrorIs(t, err, domain.ErrGalleryImageNotFound)
}

func TestGalleryRepository_Upsert_IsAtomic(t *testing.T) {
	repo := newGalleryRepo(t)
	ctx := context.Background()

	bad := mkGallery("bad", "NOT_A_TYPE", 1) // violates chk_gallery_images_type
	err := repo.Upsert(ctx, []domain.GalleryImage{mkGallery("ok", domain.GalleryChua, 0), bad})
	require.Error(t, err)

	_, findErr := repo.FindByID(ctx, "ok")
	assert.ErrorIs(t, findErr, domain.ErrGalleryImageNotFound, "rolled back: earlier good row must not persist")
}

func TestGalleryRepository_Ordering(t *testing.T) {
	repo := newGalleryRepo(t)
	ctx := context.Background()

	// Inserted out of order; sort_order decides, id breaks the tie.
	require.NoError(t, repo.Upsert(ctx, []domain.GalleryImage{
		mkGallery("c", domain.GalleryThien, 2),
		mkGallery("b2", domain.GalleryLeHoi, 1),
		mkGallery("a", domain.GalleryChua, 0),
		mkGallery("b1", domain.GalleryTuongPhat, 1),
	}))

	top, err := repo.ListTop(ctx, 3)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b1", "b2"}, galleryIDs(top), "top mixes types")
}

func TestGalleryRepository_List_PagingAndTypeFilter(t *testing.T) {
	repo := newGalleryRepo(t)
	ctx := context.Background()

	require.NoError(t, repo.Upsert(ctx, []domain.GalleryImage{
		mkGallery("c1", domain.GalleryChua, 0),
		mkGallery("t1", domain.GalleryTuongPhat, 1),
		mkGallery("c2", domain.GalleryChua, 2),
		mkGallery("c3", domain.GalleryChua, 3),
		mkGallery("t2", domain.GalleryTuongPhat, 4),
	}))

	p1, total, err := repo.List(ctx, "", 0, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, total, "unfiltered total counts every type")
	assert.Equal(t, []string{"c1", "t1"}, galleryIDs(p1))

	chua, total, err := repo.List(ctx, domain.GalleryChua, 0, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, total, "filtered total counts only that type")
	assert.Equal(t, []string{"c1", "c2"}, galleryIDs(chua))

	chuaP2, _, err := repo.List(ctx, domain.GalleryChua, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"c3"}, galleryIDs(chuaP2))

	none, total, err := repo.List(ctx, domain.GalleryThien, 0, 2)
	require.NoError(t, err)
	assert.Empty(t, none)
	assert.NotNil(t, none)
	assert.Equal(t, 0, total)

	past, total, err := repo.List(ctx, "", 100, 2)
	require.NoError(t, err)
	assert.Empty(t, past)
	assert.NotNil(t, past)
	assert.Equal(t, 5, total)
}
