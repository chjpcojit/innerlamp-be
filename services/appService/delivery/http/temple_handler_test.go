package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// newTempleMux registers the routes the way cmd/main.go does, so path-value
// extraction and the /top vs /{id} precedence are exercised for real.
func newTempleMux(uc *mockTempleUsecase) *http.ServeMux {
	h := NewTempleHandler(uc)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/temples/top", h.Top)
	mux.HandleFunc("GET /v1/temples/options", h.Options)
	mux.HandleFunc("GET /v1/temples", h.List)
	mux.HandleFunc("GET /v1/temples/{id}", h.Get)
	return mux
}

func doGet(mux *http.ServeMux, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &m), rec.Body.String())
	return m
}

func sampleTemple() domain.Temple {
	return domain.Temple{
		ID: "temple_north_001", Name: "Chùa Trấn Quốc", Type: "CHUA", Region: "north",
		Sect: "BẮC_TÔNG", Founder: "Vua Lý Nam Đế", Abbot: "Thích Thanh Nhã", Established: 541,
		Heritage: "DI_TICH_QUOC_GIA_DAC_BIET", Latitude: 21.0478, Longitude: 105.8368,
		Address: "Quận Hoàn Kiếm, Hà Nội", Province: "Hà Nội", OperatingHours: "07:30 - 18:00", Aliases: []string{"Khai Quốc Tự"},
		Amenities:    domain.TempleAmenities{ParkingMotorcycle: true},
		Introduction: "Giới thiệu", ArchitectureHeritage: "Kiến trúc", VisitingNotes: []string{"Ăn mặc kín đáo"},
	}
}

func TestTempleHandler_Top(t *testing.T) {
	t.Run("200 with envelope, items, null images", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		uc.On("Top", mock.Anything).Return([]domain.Temple{sampleTemple()}, nil)

		rec := doGet(newTempleMux(uc), "/v1/temples/top")

		require.Equal(t, http.StatusOK, rec.Code)
		body := decode(t, rec)
		assert.Equal(t, true, body["success"])
		items := body["data"].(map[string]any)["items"].([]any)
		require.Len(t, items, 1)
		item := items[0].(map[string]any)
		assert.Equal(t, "temple_north_001", item["id"])
		assert.Nil(t, item["image_url"])
		assert.Nil(t, item["thumbnail_url"])
		assert.Equal(t, "DI_TICH_QUOC_GIA_DAC_BIET", item["heritage_rank"])
		assert.Equal(t, 21.0478, item["location"].(map[string]any)["lat"])
		assert.Equal(t, "Hà Nội", item["location"].(map[string]any)["province"], "province is part of location")
		assert.Equal(t, "Quận Hoàn Kiếm, Hà Nội", item["location"].(map[string]any)["address"], "address stays the full string")
		assert.NotContains(t, item, "founder", "summary must not carry detail-only fields")
	})

	t.Run("empty result encodes items as []", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		uc.On("Top", mock.Anything).Return([]domain.Temple{}, nil)
		rec := doGet(newTempleMux(uc), "/v1/temples/top")
		assert.JSONEq(t, `{"success":true,"data":{"items":[]}}`, rec.Body.String())
	})

	t.Run("usecase failure is a 500 that leaks nothing", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		uc.On("Top", mock.Anything).Return(nil, errors.New("pq: secret detail"))
		rec := doGet(newTempleMux(uc), "/v1/temples/top")
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.NotContains(t, rec.Body.String(), "secret detail")
	})
}

// noQuery matches the nil *string the handler passes when ?q= is absent.
var noQuery = mock.MatchedBy(func(q *string) bool { return q == nil })

// queryIs matches a non-nil *string equal to want.
func queryIs(want string) any {
	return mock.MatchedBy(func(q *string) bool { return q != nil && *q == want })
}

func TestTempleHandler_Options_Query(t *testing.T) {
	t.Run("passes q through, URL-decoded, together with the page", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		uc.On("Options", mock.Anything, queryIs("bích động"), 2).Return(&domain.TempleOptionPage{
			Items: []domain.TempleOption{{ID: "x", Name: "Chùa Bích Động", Province: "Ninh Bình"}},
			Page:  2, PageSize: 100, Total: 101, TotalPages: 2,
		}, nil)

		rec := doGet(newTempleMux(uc), "/v1/temples/options?page=2&q=b%C3%ADch+%C4%91%E1%BB%99ng")

		require.Equal(t, http.StatusOK, rec.Code)
		data := decode(t, rec)["data"].(map[string]any)
		assert.EqualValues(t, 101, data["total"])
		uc.AssertExpectations(t)
	})

	t.Run("an empty q is passed through as present-but-empty and becomes 400 INVALID_FILTER on field q", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		uc.On("Options", mock.Anything, queryIs(""), 1).Return(nil, domain.ErrInvalidTempleQuery)

		rec := doGet(newTempleMux(uc), "/v1/temples/options?q=")

		require.Equal(t, http.StatusBadRequest, rec.Code)
		body := decode(t, rec)
		assert.Equal(t, "INVALID_FILTER", body["error"].(map[string]any)["code"])
		assert.Equal(t, "q", body["details"].(map[string]any)["field"])
	})

	t.Run("a wrapped ErrInvalidTempleQuery is still 400 INVALID_FILTER", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		uc.On("Options", mock.Anything, queryIs("x"), 1).Return(nil, fmt.Errorf("options: %w", domain.ErrInvalidTempleQuery))
		rec := doGet(newTempleMux(uc), "/v1/temples/options?q=x")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestTempleHandler_Options(t *testing.T) {
	t.Run("200 with exactly id, name, province per item and paging metadata", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		uc.On("Options", mock.Anything, noQuery, 1).Return(&domain.TempleOptionPage{
			Items: []domain.TempleOption{{ID: "temple_north_001", Name: "Chùa Trấn Quốc", Province: "Hà Nội"}},
			Page:  1, PageSize: 100, Total: 250, TotalPages: 3,
		}, nil)

		rec := doGet(newTempleMux(uc), "/v1/temples/options")

		require.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"success":true,"data":{"items":[{"id":"temple_north_001","name":"Chùa Trấn Quốc","province":"Hà Nội"}],"page":1,"page_size":100,"total":250,"total_pages":3}}`, rec.Body.String())
	})

	t.Run("passes the page query through", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		uc.On("Options", mock.Anything, noQuery, 3).Return(&domain.TempleOptionPage{Items: []domain.TempleOption{}, Page: 3, PageSize: 100}, nil)
		rec := doGet(newTempleMux(uc), "/v1/temples/options?page=3")
		assert.Equal(t, http.StatusOK, rec.Code)
		uc.AssertExpectations(t)
	})

	t.Run("empty result encodes items as []", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		uc.On("Options", mock.Anything, noQuery, 1).Return(&domain.TempleOptionPage{Items: []domain.TempleOption{}, Page: 1, PageSize: 100}, nil)
		rec := doGet(newTempleMux(uc), "/v1/temples/options")
		assert.JSONEq(t, `{"success":true,"data":{"items":[],"page":1,"page_size":100,"total":0,"total_pages":0}}`, rec.Body.String())
	})

	t.Run("non-numeric page is 400 INVALID_PAGE and never reaches the usecase", func(t *testing.T) {
		for _, q := range []string{"abc", "1.5", "99999999999999999999"} {
			uc := &mockTempleUsecase{}
			rec := doGet(newTempleMux(uc), "/v1/temples/options?page="+q)
			assert.Equal(t, http.StatusBadRequest, rec.Code, q)
			assert.Equal(t, "INVALID_PAGE", decode(t, rec)["error"].(map[string]any)["code"])
			uc.AssertNotCalled(t, "Options")
		}
	})

	t.Run("usecase ErrInvalidPage (page 0) is 400 INVALID_PAGE", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		uc.On("Options", mock.Anything, noQuery, 0).Return(nil, domain.ErrInvalidPage)
		rec := doGet(newTempleMux(uc), "/v1/temples/options?page=0")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "INVALID_PAGE", decode(t, rec)["error"].(map[string]any)["code"])
	})

	t.Run("/options is not captured by /{id}", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		uc.On("Options", mock.Anything, noQuery, 1).Return(&domain.TempleOptionPage{Items: []domain.TempleOption{}, Page: 1, PageSize: 100}, nil)
		rec := doGet(newTempleMux(uc), "/v1/temples/options")
		assert.Equal(t, http.StatusOK, rec.Code)
		uc.AssertNotCalled(t, "Get")
	})

	t.Run("usecase failure is a 500 that leaks nothing", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		uc.On("Options", mock.Anything, noQuery, 1).Return(nil, errors.New("pq: secret detail"))
		rec := doGet(newTempleMux(uc), "/v1/temples/options")
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.NotContains(t, rec.Body.String(), "secret detail")
	})
}

func TestTempleHandler_List(t *testing.T) {
	t.Run("defaults to page 1 and returns paging metadata", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		uc.On("List", mock.Anything, 1).Return(&domain.TemplePage{
			Items: []domain.Temple{sampleTemple()}, Page: 1, PageSize: 20, Total: 180, TotalPages: 9,
		}, nil)

		rec := doGet(newTempleMux(uc), "/v1/temples")

		require.Equal(t, http.StatusOK, rec.Code)
		data := decode(t, rec)["data"].(map[string]any)
		assert.EqualValues(t, 1, data["page"])
		assert.EqualValues(t, 20, data["page_size"])
		assert.EqualValues(t, 180, data["total"])
		assert.EqualValues(t, 9, data["total_pages"])
		assert.Len(t, data["items"], 1)
	})

	t.Run("passes the page query through", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		uc.On("List", mock.Anything, 3).Return(&domain.TemplePage{Items: []domain.Temple{}, Page: 3, PageSize: 20}, nil)
		rec := doGet(newTempleMux(uc), "/v1/temples?page=3")
		assert.Equal(t, http.StatusOK, rec.Code)
		uc.AssertExpectations(t)
	})

	t.Run("non-numeric page is 400 INVALID_PAGE and never reaches the usecase", func(t *testing.T) {
		for _, q := range []string{"abc", "1.5", "99999999999999999999"} {
			uc := &mockTempleUsecase{}
			rec := doGet(newTempleMux(uc), "/v1/temples?page="+q)
			assert.Equal(t, http.StatusBadRequest, rec.Code, q)
			assert.Equal(t, "INVALID_PAGE", decode(t, rec)["error"].(map[string]any)["code"])
			uc.AssertNotCalled(t, "List")
		}
	})

	t.Run("usecase ErrInvalidPage (page 0) is 400 INVALID_PAGE", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		uc.On("List", mock.Anything, 0).Return(nil, domain.ErrInvalidPage)
		rec := doGet(newTempleMux(uc), "/v1/temples?page=0")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "INVALID_PAGE", decode(t, rec)["error"].(map[string]any)["code"])
	})
}

func TestTempleHandler_Get(t *testing.T) {
	t.Run("200 with detail fields", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		tp := sampleTemple()
		uc.On("Get", mock.Anything, "temple_north_001").Return(&tp, nil)

		rec := doGet(newTempleMux(uc), "/v1/temples/temple_north_001")

		require.Equal(t, http.StatusOK, rec.Code)
		d := decode(t, rec)["data"].(map[string]any)
		assert.Equal(t, "Chùa Trấn Quốc", d["name"])
		assert.Equal(t, "Vua Lý Nam Đế", d["founder"])
		assert.Equal(t, "Thích Thanh Nhã", d["current_abbot"])
		assert.Equal(t, "07:30 - 18:00", d["operating_hours"])
		assert.Equal(t, []any{"Khai Quốc Tự"}, d["aliases"])
		assert.Equal(t, true, d["amenities"].(map[string]any)["parking_motorcycle"])
		assert.Equal(t, false, d["amenities"].(map[string]any)["parking_car"])
		assert.Equal(t, "Giới thiệu", d["introduction"])
		assert.Equal(t, "Kiến trúc", d["architecture_heritage"])
		assert.Equal(t, []any{"Ăn mặc kín đáo"}, d["visiting_notes"])
	})

	t.Run("unknown id is 404 TEMPLE_NOT_FOUND", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		// The usecase wraps with the id; the client message must stay fixed.
		uc.On("Get", mock.Anything, "nope").Return(nil, fmt.Errorf("get temple %q: %w", "nope", domain.ErrTempleNotFound))
		rec := doGet(newTempleMux(uc), "/v1/temples/nope")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "temple not found", decode(t, rec)["error"].(map[string]any)["message"])
		assert.Equal(t, "TEMPLE_NOT_FOUND", decode(t, rec)["error"].(map[string]any)["code"])
	})

	t.Run("/top is not captured by /{id}", func(t *testing.T) {
		uc := &mockTempleUsecase{}
		uc.On("Top", mock.Anything).Return([]domain.Temple{}, nil)
		rec := doGet(newTempleMux(uc), "/v1/temples/top")
		assert.Equal(t, http.StatusOK, rec.Code)
		uc.AssertNotCalled(t, "Get")
	})
}
