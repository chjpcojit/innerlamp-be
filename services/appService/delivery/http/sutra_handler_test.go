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

// newSutraMux registers the routes the way cmd/main.go does, so path-value
// extraction and the /v1/deities/{id} vs /v1/deities/{id}/sutras precedence are
// exercised for real.
func newSutraMux(uc *mockSutraUsecase) *http.ServeMux {
	h := NewSutraHandler(uc)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/sutras", h.List)
	mux.HandleFunc("GET /v1/sutras/{id}", h.Get)
	mux.HandleFunc("GET /v1/sutras/{id}/lines", h.Lines)
	mux.HandleFunc("GET /v1/deities/{id}/sutras", h.ListByDeity)
	return mux
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func sampleSutra() domain.Sutra {
	return domain.Sutra{
		ID: "tam-kinh", SortOrder: 3, Tradition: domain.TraditionMahayana, Kind: domain.KindSutra,
		Collection: "Bát Nhã", Reference: "Taishō T0251", Titles: map[string]string{"vi": "Tâm Kinh", "en": "Heart Sutra"},
		DailyChant: true, Purposes: []domain.Purpose{domain.PurposeHocGiaoLy}, DurationMinutes: intPtr(4),
		ReviewStatus: "UNREVIEWED",
		OriginalTitle: "Bát Nhã Ba La Mật Đa Tâm Kinh", Author: "", HanTranslator: "Huyền Trang", HanTranslationYear: intPtr(649),
		ReferenceNote: "Bản tụng", Summary: "Kinh ngắn về tính Không.",
		Deities: []domain.SutraDeity{
			{DeityID: "bodhisattva_avalokiteshvara", Role: domain.RoleSpeaker, NameVI: "Quán Thế Âm", ThumbnailURL: "q.jpg"},
		},
		Translations: []domain.SutraTranslation{
			{
				Key: "vi-chant", Lang: "vi", Variant: domain.VariantChant, Translator: "T", License: "UNKNOWN", LicenseNote: "no statement on page",
				SourceName: "budsas", SourceURL: "https://x/y", Note: "ghi chú", LineCount: 13,
				Sections: []domain.SutraSection{{Name: "Phẩm 1", StartN: 2, LineCount: 5}},
			},
			{Key: "en-translation", Lang: "en", Variant: domain.VariantTranslation, License: "CC0-1.0", LicenseURL: "https://creativecommons.org/publicdomain/zero/1.0/", LineCount: 19},
		},
	}
}

func TestSutraHandler_List_Success(t *testing.T) {
	uc := new(mockSutraUsecase)
	uc.On("List", mock.Anything, domain.SutraFilterInput{}, 1).Return(&domain.SutraPage{
		Items: []domain.Sutra{sampleSutra()}, Page: 1, PageSize: 20, Total: 45, TotalPages: 3,
	}, nil)

	rec := doGet(newSutraMux(uc), "/v1/sutras")

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	body := decode(t, rec)
	assert.Equal(t, true, body["success"])
	data := body["data"].(map[string]any)
	assert.EqualValues(t, 1, data["page"])
	assert.EqualValues(t, 20, data["page_size"])
	assert.EqualValues(t, 45, data["total"])
	assert.EqualValues(t, 3, data["total_pages"])

	items := data["items"].([]any)
	require.Len(t, items, 1)
	s := items[0].(map[string]any)
	assert.Equal(t, "tam-kinh", s["id"])
	assert.Equal(t, "MAHAYANA", s["tradition"])
	assert.Equal(t, "SUTRA", s["kind"])
	assert.Equal(t, "Bát Nhã", s["collection"])
	assert.Equal(t, "Taishō T0251", s["reference"])
	assert.Equal(t, map[string]any{"vi": "Tâm Kinh", "en": "Heart Sutra"}, s["titles"])
	assert.Equal(t, true, s["daily_chant"])
	assert.Equal(t, []any{"HOC_GIAO_LY"}, s["purposes"])
	assert.EqualValues(t, 4, s["duration_minutes"])
	assert.Equal(t, []any{map[string]any{
		"deity_id": "bodhisattva_avalokiteshvara", "role": "SPEAKER", "name_vi": "Quán Thế Âm", "thumbnail_url": "q.jpg",
	}}, s["deities"])
	assert.Equal(t, []any{
		map[string]any{"key": "vi-chant", "lang": "vi", "variant": "chant", "translator": "T", "license": "UNKNOWN", "line_count": float64(13)},
		map[string]any{"key": "en-translation", "lang": "en", "variant": "translation", "translator": "", "license": "CC0-1.0", "line_count": float64(19)},
	}, s["translations"])

	// A card stays light: nothing that only the detail page needs.
	for _, k := range []string{"review_status", "sections", "source_url", "source_name", "note",
		"original_title", "author", "han_translator", "han_translation_year", "reference_note"} {
		assert.NotContains(t, s, k)
	}
	assert.Equal(t, "Kinh ngắn về tính Không.", s["summary"], "the card carries the summary")
	for _, tr := range s["translations"].([]any) {
		for _, k := range []string{"sections", "source_url", "source_name", "note", "license_url", "license_note"} {
			assert.NotContains(t, tr, k)
		}
	}
}

func TestSutraHandler_List_NullAndEmptyShapes(t *testing.T) {
	bare := domain.Sutra{ID: "x", Tradition: domain.TraditionVajrayana, Kind: domain.KindVerse, Titles: map[string]string{"vi": "X"}}
	uc := new(mockSutraUsecase)
	uc.On("List", mock.Anything, mock.Anything, 1).Return(&domain.SutraPage{Items: []domain.Sutra{bare}, Page: 1, PageSize: 20, Total: 1, TotalPages: 1}, nil)

	rec := doGet(newSutraMux(uc), "/v1/sutras")

	s := decode(t, rec)["data"].(map[string]any)["items"].([]any)[0].(map[string]any)
	assert.Contains(t, s, "duration_minutes")
	assert.Nil(t, s["duration_minutes"], "no estimate is JSON null, never 0")
	for _, k := range []string{"purposes", "deities", "translations"} {
		assert.Equal(t, []any{}, s[k], "%s must be [] not null", k)
	}
	assert.Equal(t, "", s["reference"])
	assert.Contains(t, s, "audio_url")
	assert.Equal(t, "", s["audio_url"], "no recording is \"\", never null and never omitted")
}

func TestSutraHandler_AudioURL_InListAndDetail(t *testing.T) {
	s := sampleSutra()
	s.AudioURL = "https://cdn.example/tam-kinh.mp3"
	uc := new(mockSutraUsecase)
	uc.On("List", mock.Anything, mock.Anything, 1).Return(&domain.SutraPage{Items: []domain.Sutra{s}, Page: 1, PageSize: 20, Total: 1, TotalPages: 1}, nil)
	uc.On("Get", mock.Anything, "tam-kinh").Return(&s, nil)
	mux := newSutraMux(uc)

	item := decode(t, doGet(mux, "/v1/sutras"))["data"].(map[string]any)["items"].([]any)[0].(map[string]any)
	assert.Equal(t, "https://cdn.example/tam-kinh.mp3", item["audio_url"])

	detail := decode(t, doGet(mux, "/v1/sutras/tam-kinh"))["data"].(map[string]any)
	assert.Equal(t, "https://cdn.example/tam-kinh.mp3", detail["audio_url"])
}

func TestSutraHandler_List_EmptyPageIsEmptyArray(t *testing.T) {
	uc := new(mockSutraUsecase)
	uc.On("List", mock.Anything, mock.Anything, 9).Return(&domain.SutraPage{Items: nil, Page: 9, PageSize: 20, Total: 5, TotalPages: 1}, nil)

	data := decode(t, doGet(newSutraMux(uc), "/v1/sutras?page=9"))["data"].(map[string]any)

	assert.Equal(t, []any{}, data["items"])
}

func TestSutraHandler_List_PassesPageAndFilters(t *testing.T) {
	t.Run("every filter is forwarded raw", func(t *testing.T) {
		uc := new(mockSutraUsecase)
		want := domain.SutraFilterInput{
			Tradition: strPtr("MAHAYANA"), Kind: strPtr("DHARANI"), Collection: strPtr("Thập Chú"), Purpose: strPtr("TRI_CHU"),
			Daily: strPtr("true"), MaxMinutes: strPtr("15"), Lang: strPtr("vi"), Q: strPtr("bát nhã"),
		}
		uc.On("List", mock.Anything, want, 3).Return(&domain.SutraPage{Items: []domain.Sutra{}, Page: 3, PageSize: 20}, nil)

		rec := doGet(newSutraMux(uc), "/v1/sutras?page=3&tradition=MAHAYANA&kind=DHARANI&collection=Th%E1%BA%ADp+Ch%C3%BA&purpose=TRI_CHU&daily=true&max_minutes=15&lang=vi&q=b%C3%A1t+nh%C3%A3")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		uc.AssertExpectations(t)
	})

	t.Run("a present-but-empty parameter reaches the use case as empty, an absent one as nil", func(t *testing.T) {
		uc := new(mockSutraUsecase)
		uc.On("List", mock.Anything, domain.SutraFilterInput{Tradition: strPtr("")}, 1).Return(nil, &domain.InvalidSutraFilterError{Field: "tradition"})

		rec := doGet(newSutraMux(uc), "/v1/sutras?tradition=")

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		uc.AssertExpectations(t)
	})

	t.Run("an empty page parameter means page 1", func(t *testing.T) {
		uc := new(mockSutraUsecase)
		uc.On("List", mock.Anything, domain.SutraFilterInput{}, 1).Return(&domain.SutraPage{Items: []domain.Sutra{}, Page: 1, PageSize: 20}, nil)
		rec := doGet(newSutraMux(uc), "/v1/sutras?page=")
		require.Equal(t, http.StatusOK, rec.Code)
		uc.AssertExpectations(t)
	})
}

func TestSutraHandler_List_Errors(t *testing.T) {
	t.Run("a non-numeric page never reaches the use case", func(t *testing.T) {
		uc := new(mockSutraUsecase)
		rec := doGet(newSutraMux(uc), "/v1/sutras?page=abc")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "INVALID_PAGE")
		assert.Equal(t, "page", errorDetails(t, rec)["field"])
		uc.AssertNotCalled(t, "List")
	})
	t.Run("an out-of-range page is INVALID_PAGE", func(t *testing.T) {
		uc := new(mockSutraUsecase)
		uc.On("List", mock.Anything, mock.Anything, 0).Return(nil, domain.ErrInvalidPage)
		rec := doGet(newSutraMux(uc), "/v1/sutras?page=0")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "INVALID_PAGE")
	})
	t.Run("a rejected filter is INVALID_FILTER and names the parameter", func(t *testing.T) {
		uc := new(mockSutraUsecase)
		uc.On("List", mock.Anything, mock.Anything, 1).Return(nil, &domain.InvalidSutraFilterError{Field: "max_minutes"})
		rec := doGet(newSutraMux(uc), "/v1/sutras?max_minutes=0")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "INVALID_FILTER")
		assert.Equal(t, "max_minutes", errorDetails(t, rec)["field"])
	})
	t.Run("an infrastructure failure is 500 and does not leak the cause", func(t *testing.T) {
		uc := new(mockSutraUsecase)
		uc.On("List", mock.Anything, mock.Anything, 1).Return(nil, errors.New("db down: password=hunter2"))
		rec := doGet(newSutraMux(uc), "/v1/sutras")
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "INTERNAL_ERROR")
		assert.NotContains(t, rec.Body.String(), "hunter2")
	})
}

// errorDetails returns the "details" object of an error envelope.
func errorDetails(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	body := decode(t, rec)
	require.Equal(t, false, body["success"])
	d, _ := body["details"].(map[string]any)
	require.NotNil(t, d, "no details in %s", rec.Body.String())
	return d
}

func TestSutraHandler_Get(t *testing.T) {
	t.Run("returns the detail shape", func(t *testing.T) {
		uc := new(mockSutraUsecase)
		s := sampleSutra()
		uc.On("Get", mock.Anything, "tam-kinh").Return(&s, nil)

		rec := doGet(newSutraMux(uc), "/v1/sutras/tam-kinh")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.Equal(t, "tam-kinh", data["id"])
		assert.Equal(t, "UNREVIEWED", data["review_status"])
		assert.EqualValues(t, 4, data["duration_minutes"])
		assert.Len(t, data["deities"], 1)

		// The reference is split: reference is only the code, the rest has its own field.
		assert.Equal(t, "Taishō T0251", data["reference"])
		assert.Equal(t, "Bát Nhã Ba La Mật Đa Tâm Kinh", data["original_title"])
		assert.Equal(t, "", data["author"])
		assert.Equal(t, "Huyền Trang", data["han_translator"])
		assert.EqualValues(t, 649, data["han_translation_year"])
		assert.Equal(t, "Bản tụng", data["reference_note"])
		assert.Equal(t, "Kinh ngắn về tính Không.", data["summary"])

		trs := data["translations"].([]any)
		require.Len(t, trs, 2)
		vi := trs[0].(map[string]any)
		assert.Equal(t, "vi-chant", vi["key"])
		assert.Equal(t, "budsas", vi["source_name"])
		assert.Equal(t, "https://x/y", vi["source_url"])
		assert.Equal(t, "ghi chú", vi["note"])
		assert.Equal(t, "UNKNOWN", vi["license"])
		assert.Equal(t, "", vi["license_url"], "no URL is \"\" not null")
		assert.Equal(t, "no statement on page", vi["license_note"])
		assert.Equal(t, "https://creativecommons.org/publicdomain/zero/1.0/", trs[1].(map[string]any)["license_url"])
		assert.Equal(t, "", trs[1].(map[string]any)["license_note"])
		assert.EqualValues(t, 13, vi["line_count"])
		assert.Equal(t, []any{map[string]any{"name": "Phẩm 1", "start_n": float64(2), "line_count": float64(5)}}, vi["sections"])
		assert.Equal(t, []any{}, trs[1].(map[string]any)["sections"], "no sections is [] not null")
	})

	t.Run("an unknown year is null and empty strings are not null", func(t *testing.T) {
		uc := new(mockSutraUsecase)
		s := domain.Sutra{ID: "x", Tradition: domain.TraditionTheravada, Kind: domain.KindSutra, Titles: map[string]string{"vi": "X"}}
		uc.On("Get", mock.Anything, "x").Return(&s, nil)

		rec := doGet(newSutraMux(uc), "/v1/sutras/x")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.Nil(t, data["han_translation_year"])
		assert.Contains(t, data, "han_translation_year", "null is present, not omitted")
		for _, k := range []string{"original_title", "author", "han_translator", "reference_note", "summary"} {
			assert.Equal(t, "", data[k], k)
		}
	})

	t.Run("not found is 404 SUTRA_NOT_FOUND", func(t *testing.T) {
		uc := new(mockSutraUsecase)
		uc.On("Get", mock.Anything, "nope").Return(nil, domain.ErrSutraNotFound)
		rec := doGet(newSutraMux(uc), "/v1/sutras/nope")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "SUTRA_NOT_FOUND")
	})

	t.Run("an infrastructure failure is 500", func(t *testing.T) {
		uc := new(mockSutraUsecase)
		uc.On("Get", mock.Anything, "x").Return(nil, errors.New("boom"))
		rec := doGet(newSutraMux(uc), "/v1/sutras/x")
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.NotContains(t, rec.Body.String(), "boom")
	})
}

func TestSutraHandler_Lines_Success(t *testing.T) {
	uc := new(mockSutraUsecase)
	uc.On("Lines", mock.Anything, "phap-hoa", "vi-translation", 2, intPtr(100)).Return(&domain.SutraLinesPage{
		Translation: "vi-translation", Page: 2, PageSize: 100, Total: 450, TotalPages: 5,
		Items: []domain.SutraLine{{N: 101, Text: "một", Section: "Phẩm 1"}, {N: 102, Text: "hai"}},
	}, nil)

	rec := doGet(newSutraMux(uc), "/v1/sutras/phap-hoa/lines?translation=vi-translation&page=2&page_size=100")

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	data := decode(t, rec)["data"].(map[string]any)
	assert.Equal(t, "vi-translation", data["translation"])
	assert.EqualValues(t, 2, data["page"])
	assert.EqualValues(t, 100, data["page_size"])
	assert.EqualValues(t, 450, data["total"])
	assert.EqualValues(t, 5, data["total_pages"])
	assert.Equal(t, []any{
		map[string]any{"n": float64(101), "text": "một", "section": "Phẩm 1"},
		map[string]any{"n": float64(102), "text": "hai", "section": nil},
	}, data["items"], "a line with no section carries JSON null")
}

func TestSutraHandler_Lines_Defaults(t *testing.T) {
	t.Run("page defaults to 1 and page_size to the use case's default", func(t *testing.T) {
		uc := new(mockSutraUsecase)
		uc.On("Lines", mock.Anything, "s", "k", 1, (*int)(nil)).Return(&domain.SutraLinesPage{Translation: "k", Items: nil, Page: 1, PageSize: 200, Total: 0}, nil)
		rec := doGet(newSutraMux(uc), "/v1/sutras/s/lines?translation=k")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, []any{}, decode(t, rec)["data"].(map[string]any)["items"], "items must be [] not null")
		uc.AssertExpectations(t)
	})
	t.Run("empty page and page_size mean the defaults", func(t *testing.T) {
		uc := new(mockSutraUsecase)
		uc.On("Lines", mock.Anything, "s", "k", 1, (*int)(nil)).Return(&domain.SutraLinesPage{Translation: "k", Page: 1, PageSize: 200}, nil)
		rec := doGet(newSutraMux(uc), "/v1/sutras/s/lines?translation=k&page=&page_size=")
		require.Equal(t, http.StatusOK, rec.Code)
		uc.AssertExpectations(t)
	})
}

func TestSutraHandler_Lines_Errors(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		wantCode  int
		wantErr   string
		wantField string
	}{
		{"translation missing", "/v1/sutras/s/lines", http.StatusBadRequest, "INVALID_TRANSLATION", "translation"},
		{"translation empty", "/v1/sutras/s/lines?translation=", http.StatusBadRequest, "INVALID_TRANSLATION", "translation"},
		{"page not an integer", "/v1/sutras/s/lines?translation=k&page=x", http.StatusBadRequest, "INVALID_PAGE", "page"},
		{"page_size not an integer", "/v1/sutras/s/lines?translation=k&page_size=x", http.StatusBadRequest, "INVALID_PAGE_SIZE", "page_size"},
		{"page_size overflowing int", "/v1/sutras/s/lines?translation=k&page_size=99999999999999999999", http.StatusBadRequest, "INVALID_PAGE_SIZE", "page_size"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := new(mockSutraUsecase)
			rec := doGet(newSutraMux(uc), tt.target)
			assert.Equal(t, tt.wantCode, rec.Code, rec.Body.String())
			assert.Contains(t, rec.Body.String(), tt.wantErr)
			assert.Equal(t, tt.wantField, errorDetails(t, rec)["field"])
			uc.AssertNotCalled(t, "Lines")
		})
	}

	useCaseErrors := []struct {
		name     string
		err      error
		wantCode int
		wantErr  string
	}{
		{"unknown sutra", domain.ErrSutraNotFound, http.StatusNotFound, "SUTRA_NOT_FOUND"},
		{"unknown translation", domain.ErrSutraTranslationNotFound, http.StatusNotFound, "TRANSLATION_NOT_FOUND"},
		{"out-of-range page", domain.ErrInvalidPage, http.StatusBadRequest, "INVALID_PAGE"},
		{"non-positive page_size", domain.ErrInvalidPageSize, http.StatusBadRequest, "INVALID_PAGE_SIZE"},
		{"infrastructure", errors.New("boom"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tt := range useCaseErrors {
		t.Run("use case: "+tt.name, func(t *testing.T) {
			uc := new(mockSutraUsecase)
			uc.On("Lines", mock.Anything, "s", "k", 1, (*int)(nil)).Return(nil, tt.err)
			rec := doGet(newSutraMux(uc), "/v1/sutras/s/lines?translation=k")
			assert.Equal(t, tt.wantCode, rec.Code, rec.Body.String())
			assert.Contains(t, rec.Body.String(), tt.wantErr)
		})
	}
}

func TestSutraHandler_ListByDeity(t *testing.T) {
	t.Run("returns summaries with the deity's role", func(t *testing.T) {
		uc := new(mockSutraUsecase)
		uc.On("ListByDeity", mock.Anything, "bodhisattva_avalokiteshvara", (*string)(nil), 1).Return(&domain.DeitySutraPage{
			Items: []domain.DeitySutra{{Sutra: sampleSutra(), Role: domain.RoleSubject}}, Page: 1, PageSize: 20, Total: 17, TotalPages: 1,
		}, nil)

		rec := doGet(newSutraMux(uc), "/v1/deities/bodhisattva_avalokiteshvara/sutras")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.EqualValues(t, 17, data["total"])
		assert.EqualValues(t, 1, data["total_pages"])
		item := data["items"].([]any)[0].(map[string]any)
		assert.Equal(t, "tam-kinh", item["id"])
		assert.Equal(t, "SUBJECT", item["role"])
		assert.Contains(t, item, "translations", "same summary shape as the list")
		assert.Contains(t, item, "deities")
		assert.NotContains(t, item, "review_status")
	})

	t.Run("role and page are forwarded; an empty page is an empty array", func(t *testing.T) {
		uc := new(mockSutraUsecase)
		uc.On("ListByDeity", mock.Anything, "d", strPtr("SPEAKER"), 2).Return(&domain.DeitySutraPage{Items: nil, Page: 2, PageSize: 20}, nil)
		rec := doGet(newSutraMux(uc), "/v1/deities/d/sutras?role=SPEAKER&page=2")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, []any{}, decode(t, rec)["data"].(map[string]any)["items"])
		uc.AssertExpectations(t)
	})

	t.Run("errors", func(t *testing.T) {
		tests := []struct {
			name     string
			err      error
			wantCode int
			wantErr  string
		}{
			{"unknown deity", domain.ErrDeityNotFound, http.StatusNotFound, "DEITY_NOT_FOUND"},
			{"invalid role", &domain.InvalidSutraFilterError{Field: "role"}, http.StatusBadRequest, "INVALID_FILTER"},
			{"invalid page", domain.ErrInvalidPage, http.StatusBadRequest, "INVALID_PAGE"},
			{"infrastructure", errors.New("boom"), http.StatusInternalServerError, "INTERNAL_ERROR"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				uc := new(mockSutraUsecase)
				uc.On("ListByDeity", mock.Anything, "d", (*string)(nil), 1).Return(nil, tt.err)
				rec := doGet(newSutraMux(uc), "/v1/deities/d/sutras")
				assert.Equal(t, tt.wantCode, rec.Code, rec.Body.String())
				assert.Contains(t, rec.Body.String(), tt.wantErr)
			})
		}
	})

	t.Run("a non-numeric page never reaches the use case", func(t *testing.T) {
		uc := new(mockSutraUsecase)
		rec := doGet(newSutraMux(uc), "/v1/deities/d/sutras?page=abc")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "INVALID_PAGE")
		uc.AssertNotCalled(t, "ListByDeity")
	})
}

// AC24: adding /v1/deities/{id}/sutras must not change what the existing deity
// routes answer. Both handlers are registered on one mux, as in cmd/main.go.
func TestSutraHandler_DoesNotShadowDeityRoutes(t *testing.T) {
	sutraUC, deityUC := new(mockSutraUsecase), new(mockDeityUsecase)
	deityUC.On("Get", mock.Anything, "buddha_amitabha").Return(&domain.Deity{ID: "buddha_amitabha"}, nil)
	sutraUC.On("ListByDeity", mock.Anything, "buddha_amitabha", (*string)(nil), 1).Return(&domain.DeitySutraPage{Items: []domain.DeitySutra{}, Page: 1, PageSize: 20}, nil)

	mux := newSutraMux(sutraUC)
	dh := NewDeityHandler(deityUC)
	mux.HandleFunc("GET /v1/deities/top", dh.Top)
	mux.HandleFunc("GET /v1/deities/{id}", dh.Get)

	rec := doGet(mux, "/v1/deities/buddha_amitabha")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "buddha_amitabha", decode(t, rec)["data"].(map[string]any)["id"])
	deityUC.AssertCalled(t, "Get", mock.Anything, "buddha_amitabha")
	sutraUC.AssertNotCalled(t, "ListByDeity")

	rec = doGet(mux, "/v1/deities/buddha_amitabha/sutras")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	sutraUC.AssertCalled(t, "ListByDeity", mock.Anything, "buddha_amitabha", (*string)(nil), 1)
}

// Every route is GET-only: through the same fallback cmd/main.go wraps the mux in,
// any other method is the shared 405 envelope with Allow: GET, HEAD.
func TestSutraHandler_NonGetIsMethodNotAllowed(t *testing.T) {
	h := WithAPIFallback(newSutraMux(new(mockSutraUsecase)))
	for _, path := range []string{"/v1/sutras", "/v1/sutras/tam-kinh", "/v1/sutras/tam-kinh/lines", "/v1/deities/d/sutras"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code, path)
		assert.Contains(t, rec.Body.String(), "METHOD_NOT_ALLOWED", path)
		assert.Equal(t, "GET, HEAD", rec.Header().Get("Allow"), path)
	}
}
