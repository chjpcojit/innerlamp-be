package http

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// newArticleMux registers the routes the way cmd/main.go does, so path-value
// extraction and the /top vs /{id} precedence are exercised for real.
func newArticleMux(uc *mockArticleUsecase) *http.ServeMux {
	h := NewArticleHandler(uc)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/articles/top", h.Top)
	mux.HandleFunc("GET /v1/articles", h.List)
	mux.HandleFunc("GET /v1/articles/{id}", h.Get)
	return mux
}

func sampleArticle() domain.Article {
	d := time.Date(2019, 4, 11, 0, 0, 0, 0, time.UTC)
	return domain.Article{
		ID: "tha-thu", Title: "Tha thứ", Category: "Tha thứ", Author: "A", AuthorInitials: "A",
		PublishedDate: &d, ReadMinutes: 7, Summary: "Tóm tắt", Tags: []string{"tha thứ"},
		BodyParagraphs: []string{"đoạn 1", "đoạn 2"}, Quote: "câu", QuoteSource: "A",
		Language: "vi", SourceName: "Báo Giác Ngộ", SourceURL: "https://example.org/x", License: "l",
	}
}

func TestArticleHandler_Top(t *testing.T) {
	uc := new(mockArticleUsecase)
	uc.On("Top", mock.Anything).Return([]domain.Article{sampleArticle()}, nil)

	rec := doGet(newArticleMux(uc), "/v1/articles/top")

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	body := decode(t, rec)
	assert.Equal(t, true, body["success"])
	items := body["data"].(map[string]any)["items"].([]any)
	require.Len(t, items, 1)
	first := items[0].(map[string]any)
	assert.Equal(t, "tha-thu", first["id"])
	assert.Equal(t, "2019-04-11", first["published_date"])
	assert.Nil(t, first["image_url"], "missing image is JSON null")
	assert.Contains(t, first, "image_url")
	assert.Nil(t, first["thumbnail_url"], "missing thumbnail is JSON null")
	assert.Contains(t, first, "thumbnail_url")
	assert.NotContains(t, first, "body_paragraphs", "summary must stay light")
}

func TestArticleHandler_Top_NullPublishedDate(t *testing.T) {
	a := sampleArticle()
	a.PublishedDate = nil
	a.ImageURL = "https://img/x.jpg"
	a.ThumbnailURL = "http://host/static/images/articles/thumb/x.webp"
	uc := new(mockArticleUsecase)
	uc.On("Top", mock.Anything).Return([]domain.Article{a}, nil)

	rec := doGet(newArticleMux(uc), "/v1/articles/top")

	first := decode(t, rec)["data"].(map[string]any)["items"].([]any)[0].(map[string]any)
	assert.Contains(t, first, "published_date")
	assert.Nil(t, first["published_date"])
	assert.Equal(t, "https://img/x.jpg", first["image_url"])
	assert.Equal(t, "http://host/static/images/articles/thumb/x.webp", first["thumbnail_url"])
}

func TestArticleHandler_Top_ServerError(t *testing.T) {
	uc := new(mockArticleUsecase)
	uc.On("Top", mock.Anything).Return(nil, errors.New("db down"))
	rec := doGet(newArticleMux(uc), "/v1/articles/top")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "db down")
}

func TestArticleHandler_List(t *testing.T) {
	t.Run("defaults to page 1", func(t *testing.T) {
		uc := new(mockArticleUsecase)
		uc.On("List", mock.Anything, 1).Return(&domain.ArticlePage{
			Items: []domain.Article{sampleArticle()}, Page: 1, PageSize: 10, Total: 30, TotalPages: 3,
		}, nil)

		rec := doGet(newArticleMux(uc), "/v1/articles")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.EqualValues(t, 1, data["page"])
		assert.EqualValues(t, 10, data["page_size"])
		assert.EqualValues(t, 30, data["total"])
		assert.EqualValues(t, 3, data["total_pages"])
		assert.Len(t, data["items"], 1)
	})

	t.Run("passes the page query", func(t *testing.T) {
		uc := new(mockArticleUsecase)
		uc.On("List", mock.Anything, 3).Return(&domain.ArticlePage{Items: []domain.Article{}, Page: 3, PageSize: 10, Total: 30, TotalPages: 3}, nil)
		rec := doGet(newArticleMux(uc), "/v1/articles?page=3")
		require.Equal(t, http.StatusOK, rec.Code)
		uc.AssertExpectations(t)
	})

	t.Run("non-numeric page is INVALID_PAGE", func(t *testing.T) {
		uc := new(mockArticleUsecase)
		rec := doGet(newArticleMux(uc), "/v1/articles?page=abc")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "INVALID_PAGE")
		uc.AssertNotCalled(t, "List")
	})

	t.Run("use case ErrInvalidPage maps to 400", func(t *testing.T) {
		uc := new(mockArticleUsecase)
		uc.On("List", mock.Anything, 0).Return(nil, domain.ErrInvalidPage)
		rec := doGet(newArticleMux(uc), "/v1/articles?page=0")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "INVALID_PAGE")
	})
}

func TestArticleHandler_Get(t *testing.T) {
	t.Run("returns full detail", func(t *testing.T) {
		uc := new(mockArticleUsecase)
		a := sampleArticle()
		uc.On("Get", mock.Anything, "tha-thu").Return(&a, nil)

		rec := doGet(newArticleMux(uc), "/v1/articles/tha-thu")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.Equal(t, "tha-thu", data["id"])
		assert.Equal(t, "Tha thứ", data["title"])
		assert.Equal(t, []any{"đoạn 1", "đoạn 2"}, data["body_paragraphs"])
		assert.Equal(t, "câu", data["quote"])
		assert.Equal(t, "Báo Giác Ngộ", data["source_name"])
		assert.Equal(t, "https://example.org/x", data["source_url"])
	})

	t.Run("not found is 404 ARTICLE_NOT_FOUND", func(t *testing.T) {
		uc := new(mockArticleUsecase)
		uc.On("Get", mock.Anything, "nope").Return(nil, domain.ErrArticleNotFound)
		rec := doGet(newArticleMux(uc), "/v1/articles/nope")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "ARTICLE_NOT_FOUND")
	})

	t.Run("top is not captured by {id}", func(t *testing.T) {
		uc := new(mockArticleUsecase)
		uc.On("Top", mock.Anything).Return([]domain.Article{}, nil)
		rec := doGet(newArticleMux(uc), "/v1/articles/top")
		assert.Equal(t, http.StatusOK, rec.Code)
		uc.AssertNotCalled(t, "Get")
	})
}
