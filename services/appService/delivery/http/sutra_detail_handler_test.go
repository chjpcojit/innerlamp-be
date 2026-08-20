package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

type mockGetSutraUsecase struct {
	mock.Mock
}

func (m *mockGetSutraUsecase) GetSutraBySlug(ctx context.Context, slug string) (*domain.Sutra, error) {
	args := m.Called(ctx, slug)
	if s := args.Get(0); s != nil {
		return s.(*domain.Sutra), args.Error(1)
	}
	return nil, args.Error(1)
}

// sutraDetailRequest builds a request carrying the {slug} path value the
// way Go 1.22+ http.ServeMux sets it when the handler is registered as
// "GET /v1/content/sutras/{slug}" — SetPathValue mirrors that at test time
// without needing a real mux.
func sutraDetailRequest(t *testing.T, slug string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/content/sutras/"+slug, nil)
	req.SetPathValue("slug", slug)
	return req
}

// TestSutraDetailHandler_Success covers AC1: title, sect, and full body are
// all present in the response.
func TestSutraDetailHandler_Success(t *testing.T) {
	uc := &mockGetSutraUsecase{}
	now := time.Now()
	uc.On("GetSutraBySlug", mock.Anything, "kinh-phap-hoa").Return(&domain.Sutra{
		ID: "sutra-1", Slug: "kinh-phap-hoa", Title: "Kinh Pháp Hoa", Sect: "Bắc Tông",
		Body: "full body text", CreatedAt: now, UpdatedAt: now,
	}, nil)

	h := NewSutraDetailHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, sutraDetailRequest(t, "kinh-phap-hoa"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	got := decodeResponse(t, rec)
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("data field missing or wrong type: %v", got["data"])
	}
	if data["title"] != "Kinh Pháp Hoa" || data["sect"] != "Bắc Tông" || data["body"] != "full body text" {
		t.Errorf("data = %v, unexpected fields", data)
	}
	uc.AssertExpectations(t)
}

// TestSutraDetailHandler_AudioURLPresentWhenSet covers AC2: an audio_url is
// present in the response when the sutra has one.
func TestSutraDetailHandler_AudioURLPresentWhenSet(t *testing.T) {
	uc := &mockGetSutraUsecase{}
	uc.On("GetSutraBySlug", mock.Anything, "kinh-phap-hoa").Return(&domain.Sutra{
		Slug: "kinh-phap-hoa", AudioURL: "https://cdn.example.com/a.mp3",
	}, nil)

	h := NewSutraDetailHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, sutraDetailRequest(t, "kinh-phap-hoa"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	data := decodeResponse(t, rec)["data"].(map[string]any)
	if data["audio_url"] != "https://cdn.example.com/a.mp3" {
		t.Errorf("data.audio_url = %v, want the URL", data["audio_url"])
	}
}

// TestSutraDetailHandler_NoAudioURLNoError covers AC2's second half: no
// audio_url renders as an empty string, not an error or missing field.
func TestSutraDetailHandler_NoAudioURLNoError(t *testing.T) {
	uc := &mockGetSutraUsecase{}
	uc.On("GetSutraBySlug", mock.Anything, "kinh-tu-niem-xu").Return(&domain.Sutra{
		Slug: "kinh-tu-niem-xu", AudioURL: "",
	}, nil)

	h := NewSutraDetailHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, sutraDetailRequest(t, "kinh-tu-niem-xu"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	data := decodeResponse(t, rec)["data"].(map[string]any)
	if v, ok := data["audio_url"]; !ok || v != "" {
		t.Errorf("data.audio_url = %v (present=%v), want empty string present", v, ok)
	}
}

// TestSutraDetailHandler_NotFound covers an unknown slug mapping to 404.
func TestSutraDetailHandler_NotFound(t *testing.T) {
	uc := &mockGetSutraUsecase{}
	uc.On("GetSutraBySlug", mock.Anything, "ghost-sutra").Return(nil, domain.ErrSutraNotFound)

	h := NewSutraDetailHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, sutraDetailRequest(t, "ghost-sutra"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

// TestSutraDetailHandler_MissingSlug covers an empty path value (shouldn't
// normally happen via the registered mux pattern, but the handler must not
// panic or call the use-case with an empty string).
func TestSutraDetailHandler_MissingSlug(t *testing.T) {
	uc := &mockGetSutraUsecase{}
	h := NewSutraDetailHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, sutraDetailRequest(t, ""))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	uc.AssertNotCalled(t, "GetSutraBySlug", mock.Anything, mock.Anything)
}

// TestSutraDetailHandler_UsecaseErrorMapsToSystemError covers a non-
// not-found repository failure mapping to 500, not a bare 200/404.
func TestSutraDetailHandler_UsecaseErrorMapsToSystemError(t *testing.T) {
	uc := &mockGetSutraUsecase{}
	uc.On("GetSutraBySlug", mock.Anything, "kinh-phap-hoa").Return(nil, http.ErrHandlerTimeout)

	h := NewSutraDetailHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, sutraDetailRequest(t, "kinh-phap-hoa"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

// TestSutraDetailHandler_MethodNotAllowed covers unsupported verbs.
func TestSutraDetailHandler_MethodNotAllowed(t *testing.T) {
	uc := &mockGetSutraUsecase{}
	h := NewSutraDetailHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/v1/content/sutras/kinh-phap-hoa", nil)
	req.SetPathValue("slug", "kinh-phap-hoa")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
	}
	uc.AssertNotCalled(t, "GetSutraBySlug", mock.Anything, mock.Anything)
}
