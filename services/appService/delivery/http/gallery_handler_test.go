package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// newGalleryMux registers the routes the way cmd/main.go does, so path-value
// extraction and the /top vs /{id} precedence are exercised for real.
func newGalleryMux(uc *mockGalleryUsecase) *http.ServeMux {
	h := NewGalleryHandler(uc)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/gallery/top", h.Top)
	mux.HandleFunc("GET /v1/gallery", h.List)
	mux.HandleFunc("GET /v1/gallery/{id}", h.Get)
	return mux
}

func sampleGalleryImage() domain.GalleryImage {
	return domain.GalleryImage{
		ID: "gallery_001", Type: domain.GalleryChua, Title: "Chùa Một Cột", Location: "Hà Nội, Việt Nam",
		ThumbnailURL: "https://x/thumb.jpg", ImageURL: "https://x/full.jpg",
		Author: "thalling55", License: "CC BY 2.0", SourceURL: "https://commons.wikimedia.org/wiki/File:X.jpg",
	}
}

func TestGalleryHandler_Top(t *testing.T) {
	uc := new(mockGalleryUsecase)
	uc.On("Top", mock.Anything).Return([]domain.GalleryImage{sampleGalleryImage()}, nil)

	rec := doGet(newGalleryMux(uc), "/v1/gallery/top")

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	body := decode(t, rec)
	assert.Equal(t, true, body["success"])
	items := body["data"].(map[string]any)["items"].([]any)
	require.Len(t, items, 1)
	first := items[0].(map[string]any)
	assert.Equal(t, "gallery_001", first["id"])
	assert.Equal(t, "CHUA", first["type"])
	assert.Equal(t, "Chùa Một Cột", first["title"])
	assert.Equal(t, "https://x/thumb.jpg", first["thumbnail_url"])
	assert.NotContains(t, first, "author", "summary must stay light")
	assert.NotContains(t, first, "license")
}

func TestGalleryHandler_Top_ServerError(t *testing.T) {
	uc := new(mockGalleryUsecase)
	uc.On("Top", mock.Anything).Return(nil, errors.New("db down"))
	rec := doGet(newGalleryMux(uc), "/v1/gallery/top")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "db down")
}

func TestGalleryHandler_List(t *testing.T) {
	t.Run("defaults to page 1 and every type", func(t *testing.T) {
		uc := new(mockGalleryUsecase)
		uc.On("List", mock.Anything, "", 1).Return(&domain.GalleryPage{
			Items: []domain.GalleryImage{sampleGalleryImage()}, Page: 1, PageSize: 20, Total: 86, TotalPages: 5,
		}, nil)

		rec := doGet(newGalleryMux(uc), "/v1/gallery")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.EqualValues(t, 1, data["page"])
		assert.EqualValues(t, 20, data["page_size"])
		assert.EqualValues(t, 86, data["total"])
		assert.EqualValues(t, 5, data["total_pages"])
		assert.Len(t, data["items"], 1)
	})

	t.Run("passes type and page through; empty page is []", func(t *testing.T) {
		uc := new(mockGalleryUsecase)
		uc.On("List", mock.Anything, "THIEN", 5).Return(&domain.GalleryPage{
			Items: []domain.GalleryImage{}, Page: 5, PageSize: 20, Total: 12, TotalPages: 1,
		}, nil)
		rec := doGet(newGalleryMux(uc), "/v1/gallery?type=THIEN&page=5")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, []any{}, decode(t, rec)["data"].(map[string]any)["items"])
	})

	t.Run("non-numeric page is 400 INVALID_PAGE", func(t *testing.T) {
		uc := new(mockGalleryUsecase)
		rec := doGet(newGalleryMux(uc), "/v1/gallery?page=abc")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "INVALID_PAGE")
		uc.AssertNotCalled(t, "List")
	})

	t.Run("usecase ErrInvalidPage is 400", func(t *testing.T) {
		uc := new(mockGalleryUsecase)
		uc.On("List", mock.Anything, "", 0).Return(nil, domain.ErrInvalidPage)
		rec := doGet(newGalleryMux(uc), "/v1/gallery?page=0")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "INVALID_PAGE")
	})

	t.Run("unknown type is 400 INVALID_GALLERY_TYPE with details.field", func(t *testing.T) {
		uc := new(mockGalleryUsecase)
		uc.On("List", mock.Anything, "X", 1).Return(nil, domain.ErrInvalidGalleryType)
		rec := doGet(newGalleryMux(uc), "/v1/gallery?type=X")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "INVALID_GALLERY_TYPE")
		assert.Contains(t, rec.Body.String(), `"field":"type"`)
	})
}

func TestGalleryHandler_Get(t *testing.T) {
	t.Run("returns detail with attribution", func(t *testing.T) {
		uc := new(mockGalleryUsecase)
		g := sampleGalleryImage()
		uc.On("Get", mock.Anything, g.ID).Return(&g, nil)

		rec := doGet(newGalleryMux(uc), "/v1/gallery/"+g.ID)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.Equal(t, g.ID, data["id"])
		assert.Equal(t, "https://x/full.jpg", data["image_url"])
		assert.Equal(t, "thalling55", data["author"])
		assert.Equal(t, "CC BY 2.0", data["license"])
		assert.Equal(t, g.SourceURL, data["source_url"])
	})

	t.Run("unknown id is 404 GALLERY_IMAGE_NOT_FOUND", func(t *testing.T) {
		uc := new(mockGalleryUsecase)
		uc.On("Get", mock.Anything, "nope").Return(nil, domain.ErrGalleryImageNotFound)
		rec := doGet(newGalleryMux(uc), "/v1/gallery/nope")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "GALLERY_IMAGE_NOT_FOUND")
	})

	t.Run("top is not captured by {id}", func(t *testing.T) {
		uc := new(mockGalleryUsecase)
		uc.On("Top", mock.Anything).Return([]domain.GalleryImage{}, nil)
		rec := doGet(newGalleryMux(uc), "/v1/gallery/top")
		assert.Equal(t, http.StatusOK, rec.Code)
		uc.AssertNotCalled(t, "Get")
	})
}

// Stored image paths are root-relative; the API answers with an absolute URL built from
// the configured base, else from the request's own Host, so a network change never
// invalidates stored data.
func TestGalleryHandler_Get_ResolvesRelativeImagePaths(t *testing.T) {
	img := sampleGalleryImage()
	img.ThumbnailURL, img.ImageURL = "/static/images/gallery/thumb/g1.jpg", "/static/images/gallery/full/g1.jpg"

	cases := []struct {
		name string
		opts []ImageOption
		host string
		want string
	}{
		{"request host", nil, "192.168.1.50:8080", "http://192.168.1.50:8080/static/images/gallery/full/g1.jpg"},
		{"configured base", []ImageOption{WithImageBase("https://cdn.x.vn")}, "ignored", "https://cdn.x.vn/static/images/gallery/full/g1.jpg"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			uc := new(mockGalleryUsecase)
			uc.On("Get", mock.Anything, "g1").Return(&img, nil)
			h := NewGalleryHandler(uc, c.opts...)
			mux := http.NewServeMux()
			mux.HandleFunc("GET /v1/gallery/{id}", h.Get)

			req := httptest.NewRequest(http.MethodGet, "/v1/gallery/g1", nil)
			req.Host = c.host
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			data := decode(t, rec)["data"].(map[string]any)
			assert.Equal(t, c.want, data["image_url"])
			assert.Contains(t, data["thumbnail_url"], "/static/images/gallery/thumb/g1.jpg")
		})
	}
}
