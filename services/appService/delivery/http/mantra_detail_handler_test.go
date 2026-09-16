package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

type mockGetMantraUsecase struct {
	mock.Mock
}

func (m *mockGetMantraUsecase) GetMantraBySlug(ctx context.Context, slug string) (*domain.Mantra, *domain.Bodhisattva, error) {
	args := m.Called(ctx, slug)
	var mantra *domain.Mantra
	var b *domain.Bodhisattva
	if s := args.Get(0); s != nil {
		mantra = s.(*domain.Mantra)
	}
	if s := args.Get(1); s != nil {
		b = s.(*domain.Bodhisattva)
	}
	return mantra, b, args.Error(2)
}

func mantraDetailRequest(t *testing.T, slug string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/content/mantras/"+slug, nil)
	req.SetPathValue("slug", slug)
	return req
}

// TestMantraDetailHandler_Success covers AC1: title, body, sect,
// bodhisattva_id and linked bodhisattva object are all present in the response.
func TestMantraDetailHandler_Success(t *testing.T) {
	uc := &mockGetMantraUsecase{}
	mantra := &domain.Mantra{
		ID: "mantra-1", Slug: "chu-dai-bi", Title: "Chú Đại Bi", Sect: "bac_tong", Body: "full body text", BodhisattvaID: "bodhisattva-1",
	}
	bodhisattva := &domain.Bodhisattva{
		ID: "bodhisattva-1", Slug: "quan-the-am", Name: "Quán Thế Âm Bồ Tát",
	}
	uc.On("GetMantraBySlug", mock.Anything, "chu-dai-bi").Return(mantra, bodhisattva, nil)

	h := NewMantraDetailHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, mantraDetailRequest(t, "chu-dai-bi"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	data := decodeResponse(t, rec)["data"].(map[string]any)
	if data["title"] != "Chú Đại Bi" || data["body"] != "full body text" || data["bodhisattva_id"] != "bodhisattva-1" || data["sect"] != "bac_tong" {
		t.Errorf("data = %v, unexpected fields", data)
	}
	bMap, ok := data["bodhisattva"].(map[string]any)
	if !ok || bMap["slug"] != "quan-the-am" || bMap["name"] != "Quán Thế Âm Bồ Tát" {
		t.Errorf("data.bodhisattva = %v, want embedded slug and name", data["bodhisattva"])
	}
}

// TestMantraDetailHandler_NoBodhisattva covers mantra without related bodhisattva.
func TestMantraDetailHandler_NoBodhisattva(t *testing.T) {
	uc := &mockGetMantraUsecase{}
	mantra := &domain.Mantra{
		ID: "mantra-2", Slug: "om-mani-padme-hum", Title: "Om Mani Padme Hum", Body: "body", BodhisattvaID: "",
	}
	uc.On("GetMantraBySlug", mock.Anything, "om-mani-padme-hum").Return(mantra, nil, nil)

	h := NewMantraDetailHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, mantraDetailRequest(t, "om-mani-padme-hum"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	data := decodeResponse(t, rec)["data"].(map[string]any)
	if _, ok := data["bodhisattva"]; ok {
		t.Errorf("data.bodhisattva = %v, want omitted/nil", data["bodhisattva"])
	}
}

// TestMantraDetailHandler_NoAudioURLNoError covers AC2's second half: no
// audio_url renders as an empty string, not an error or missing field.
func TestMantraDetailHandler_NoAudioURLNoError(t *testing.T) {
	uc := &mockGetMantraUsecase{}
	uc.On("GetMantraBySlug", mock.Anything, "om-mani-padme-hum").Return(&domain.Mantra{
		Slug: "om-mani-padme-hum", AudioURL: "",
	}, nil, nil)

	h := NewMantraDetailHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, mantraDetailRequest(t, "om-mani-padme-hum"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	data := decodeResponse(t, rec)["data"].(map[string]any)
	if v, ok := data["audio_url"]; !ok || v != "" {
		t.Errorf("data.audio_url = %v (present=%v), want empty string present", v, ok)
	}
}

// TestMantraDetailHandler_NotFound covers an unknown slug mapping to 404.
func TestMantraDetailHandler_NotFound(t *testing.T) {
	uc := &mockGetMantraUsecase{}
	uc.On("GetMantraBySlug", mock.Anything, "ghost-mantra").Return(nil, nil, domain.ErrMantraNotFound)

	h := NewMantraDetailHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, mantraDetailRequest(t, "ghost-mantra"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

// TestMantraDetailHandler_MissingSlug covers an empty path value.
func TestMantraDetailHandler_MissingSlug(t *testing.T) {
	uc := &mockGetMantraUsecase{}
	h := NewMantraDetailHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, mantraDetailRequest(t, ""))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	uc.AssertNotCalled(t, "GetMantraBySlug", mock.Anything, mock.Anything)
}

// TestMantraDetailHandler_UsecaseErrorMapsToSystemError covers a non-
// not-found repository failure mapping to 500, not a bare 200/404.
func TestMantraDetailHandler_UsecaseErrorMapsToSystemError(t *testing.T) {
	uc := &mockGetMantraUsecase{}
	uc.On("GetMantraBySlug", mock.Anything, "chu-dai-bi").Return(nil, nil, http.ErrHandlerTimeout)

	h := NewMantraDetailHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, mantraDetailRequest(t, "chu-dai-bi"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

// TestMantraDetailHandler_MethodNotAllowed covers unsupported verbs.
func TestMantraDetailHandler_MethodNotAllowed(t *testing.T) {
	uc := &mockGetMantraUsecase{}
	h := NewMantraDetailHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/v1/content/mantras/chu-dai-bi", nil)
	req.SetPathValue("slug", "chu-dai-bi")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
	}
	uc.AssertNotCalled(t, "GetMantraBySlug", mock.Anything, mock.Anything)
}
