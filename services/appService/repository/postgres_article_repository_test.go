package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func mkArticle(id string, order int) domain.Article {
	return domain.Article{
		ID: id, SortOrder: order, Title: "Tiêu đề " + id, Category: "Thiền", Author: "A", AuthorInitials: "A",
		ReadMinutes: 5, Summary: "s", BodyParagraphs: []string{"p1", "p2"}, Quote: "q", QuoteSource: "qs",
		Tags: []string{"t1"}, Language: "vi", SourceName: "src", SourceURL: "https://x", License: "l",
	}
}

func newArticleRepo(t *testing.T) domain.ArticleRepository {
	t.Helper()
	return NewArticleRepository(connectTestDB(t))
}

func articleIDs(as []domain.Article) []string {
	out := make([]string, 0, len(as))
	for _, a := range as {
		out = append(out, a.ID)
	}
	return out
}

func TestArticleRepository_InsertAndFind(t *testing.T) {
	repo := newArticleRepo(t)
	ctx := context.Background()

	date := time.Date(2019, 4, 11, 0, 0, 0, 0, time.UTC)
	withDate := mkArticle("dated", 0)
	withDate.PublishedDate = &date
	withDate.ImageURL = "https://img/x.jpg"
	withDate.ThumbnailURL = "http://h/static/images/articles/thumb/x.webp"
	plain := mkArticle("plain", 1)

	n, err := repo.InsertMissing(ctx, []domain.Article{withDate, plain})
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	got, err := repo.FindByID(ctx, "dated")
	require.NoError(t, err)
	assert.Equal(t, withDate.Title, got.Title)
	assert.Equal(t, withDate.BodyParagraphs, got.BodyParagraphs)
	assert.Equal(t, withDate.Tags, got.Tags)
	require.NotNil(t, got.PublishedDate)
	assert.Equal(t, "2019-04-11", got.PublishedDate.Format(time.DateOnly))
	assert.Equal(t, "https://img/x.jpg", got.ImageURL)
	assert.Equal(t, "http://h/static/images/articles/thumb/x.webp", got.ThumbnailURL)

	got, err = repo.FindByID(ctx, "plain")
	require.NoError(t, err)
	assert.Nil(t, got.PublishedDate, "NULL date round-trips as nil")
	assert.Empty(t, got.ImageURL)
}

func TestArticleRepository_FindByID_NotFound(t *testing.T) {
	repo := newArticleRepo(t)
	_, err := repo.FindByID(context.Background(), "missing")
	assert.ErrorIs(t, err, domain.ErrArticleNotFound)
}

func TestArticleRepository_InsertMissing_NeverOverwrites(t *testing.T) {
	repo := newArticleRepo(t)
	ctx := context.Background()

	require.NoError(t, func() error { _, err := repo.InsertMissing(ctx, []domain.Article{mkArticle("a", 0)}); return err }())

	edited := mkArticle("a", 0)
	edited.Title = "Should not win"
	n, err := repo.InsertMissing(ctx, []domain.Article{edited, mkArticle("b", 1)})
	require.NoError(t, err)
	assert.Equal(t, 1, n, "only the missing id is inserted")

	got, err := repo.FindByID(ctx, "a")
	require.NoError(t, err)
	assert.Equal(t, "Tiêu đề a", got.Title)
	c, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, c)
}

func TestArticleRepository_Count_Empty(t *testing.T) {
	repo := newArticleRepo(t)
	c, err := repo.Count(context.Background())
	require.NoError(t, err)
	assert.Zero(t, c)
}

func TestArticleRepository_InsertMissing_IsAtomic(t *testing.T) {
	repo := newArticleRepo(t)
	ctx := context.Background()

	bad := mkArticle("bad", 1)
	bad.Language = "way-too-long-language-code" // exceeds VARCHAR(8)
	_, err := repo.InsertMissing(ctx, []domain.Article{mkArticle("ok", 0), bad})
	require.Error(t, err)

	_, findErr := repo.FindByID(ctx, "ok")
	assert.ErrorIs(t, findErr, domain.ErrArticleNotFound, "rolled back: earlier good row must not persist")
}

func TestArticleRepository_OrderingAndPaging(t *testing.T) {
	repo := newArticleRepo(t)
	ctx := context.Background()

	// Inserted out of order; sort_order decides, id breaks the tie.
	var all []domain.Article
	for i := 4; i >= 0; i-- {
		all = append(all, mkArticle(fmt.Sprintf("a%d", i), i))
	}
	all = append(all, mkArticle("a1b", 1))
	_, err := repo.InsertMissing(ctx, all)
	require.NoError(t, err)

	top, err := repo.ListTop(ctx, 3)
	require.NoError(t, err)
	assert.Equal(t, []string{"a0", "a1", "a1b"}, articleIDs(top))

	p1, total, err := repo.List(ctx, 0, 4)
	require.NoError(t, err)
	assert.Equal(t, 6, total)
	assert.Equal(t, []string{"a0", "a1", "a1b", "a2"}, articleIDs(p1))

	p2, _, err := repo.List(ctx, 4, 4)
	require.NoError(t, err)
	assert.Equal(t, []string{"a3", "a4"}, articleIDs(p2))

	past, total, err := repo.List(ctx, 100, 4)
	require.NoError(t, err)
	assert.Empty(t, past)
	assert.NotNil(t, past)
	assert.Equal(t, 6, total)
}

func TestArticleRepository_FillMissingImages(t *testing.T) {
	repo := newArticleRepo(t)
	ctx := context.Background()

	has := mkArticle("has", 0)
	has.ImageURL = "https://img/keep.jpg"
	_, err := repo.InsertMissing(ctx, []domain.Article{has, mkArticle("none", 1), mkArticle("skip", 2)})
	require.NoError(t, err)

	fill := func(id, url string) domain.Article { a := mkArticle(id, 0); a.ImageURL = url; return a }
	n, err := repo.FillMissingImages(ctx, []domain.Article{
		fill("has", "https://img/new.jpg"), fill("none", "https://img/new.jpg"), fill("skip", ""), fill("absent", "https://img/x.jpg"),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, n, "only the row with a NULL image and a non-empty source image changes")

	got, _ := repo.FindByID(ctx, "has")
	assert.Equal(t, "https://img/keep.jpg", got.ImageURL, "existing image is never overwritten")
	got, _ = repo.FindByID(ctx, "none")
	assert.Equal(t, "https://img/new.jpg", got.ImageURL)
	got, _ = repo.FindByID(ctx, "skip")
	assert.Equal(t, "", got.ImageURL)

	n, err = repo.FillMissingImages(ctx, []domain.Article{fill("none", "https://img/other.jpg")})
	require.NoError(t, err)
	assert.Equal(t, 0, n, "second run is a no-op")
}

func TestArticleRepository_FillMissingImages_Thumbnail(t *testing.T) {
	repo := newArticleRepo(t)
	ctx := context.Background()

	// "kept" already has both; "img" has an image but no thumbnail; "bare" has neither.
	kept := mkArticle("kept", 0)
	kept.ImageURL, kept.ThumbnailURL = "https://img/k.jpg", "http://h/kept-old.webp"
	img := mkArticle("img", 1)
	img.ImageURL = "https://img/i.jpg"
	_, err := repo.InsertMissing(ctx, []domain.Article{kept, img, mkArticle("bare", 2)})
	require.NoError(t, err)

	src := func(id, image, thumb string) domain.Article {
		a := mkArticle(id, 0)
		a.ImageURL, a.ThumbnailURL = image, thumb
		return a
	}
	n, err := repo.FillMissingImages(ctx, []domain.Article{
		src("kept", "https://img/k2.jpg", "http://h/kept-new.webp"),
		src("img", "https://img/i2.jpg", "http://h/img.webp"),
		src("bare", "", "http://h/bare.webp"),
	})
	require.NoError(t, err)
	assert.Equal(t, 2, n, "kept is untouched; img gains a thumbnail; bare gains one")

	got, _ := repo.FindByID(ctx, "kept")
	assert.Equal(t, "http://h/kept-old.webp", got.ThumbnailURL, "an existing thumbnail is never overwritten")
	got, _ = repo.FindByID(ctx, "img")
	assert.Equal(t, "https://img/i.jpg", got.ImageURL, "image stays: it was already set")
	assert.Equal(t, "http://h/img.webp", got.ThumbnailURL)
	got, _ = repo.FindByID(ctx, "bare")
	assert.Equal(t, "", got.ImageURL, "no source image means none is invented")
	assert.Equal(t, "http://h/bare.webp", got.ThumbnailURL)
}
