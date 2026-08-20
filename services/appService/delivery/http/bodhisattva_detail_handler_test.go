package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

type mockGetBodhisattvaUsecase struct {
	mock.Mock
}

func (m *mockGetBodhisattvaUsecase) GetBodhisattvaBySlug(ctx context.Context, slug string) (*domain.Bodhisattva, []*domain.Mantra, error) {
	args := m.Called(ctx, slug)
	var b *domain.Bodhisattva
	var mantras []*domain.Mantra
	if v := args.Get(0); v != nil {
		b = v.(*domain.Bodhisattva)
	}
	if v := args.Get(1); v != nil {
		mantras = v.([]*domain.Mantra)
	}
	return b, mantras, args.Error(2)
}

func bodhisattvaDetailRequest(t *testing.T, slug string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/content/bodhisattvas/"+slug, nil)
	req.SetPathValue("slug", slug)
	return req
}

// TestBodhisattvaDetailHandler_Success covers AC1: name, description,
// image, and related mantras are all present in the response.
func TestBodhisattvaDetailHandler_Success(t *testing.T) {
	uc := &mockGetBodhisattvaUsecase{}
	uc.On("GetBodhisattvaBySlug", mock.Anything, "quan-the-am").Return(
		&domain.Bodhisattva{ID: "bodhisattva-1", Slug: "quan-the-am", Name: "Quán Thế Âm Bồ Tát", Description: "full description", ImageURL: "https://cdn.example.com/qta.png"},
		[]*domain.Mantra{{ID: "mantra-1", Slug: "chu-dai-bi", Title: "Chú Đại Bi", BodhisattvaID: "bodhisattva-1"}},
		nil,
	)

	h := NewBodhisattvaDetailHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, bodhisattvaDetailRequest(t, "quan-the-am"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	data := decodeResponse(t, rec)["data"].(map[string]any)
	if data["name"] != "Quán Thế Âm Bồ Tát" || data["description"] != "full description" {
		t.Errorf("data = %v, unexpected fields", data)
	}
	mantras, ok := data["mantras"].([]any)
	if !ok || len(mantras) != 1 {
		t.Fatalf("data.mantras missing, wrong type, or wrong length: %v", data["mantras"])
	}
	m := mantras[0].(map[string]any)
	if m["title"] != "Chú Đại Bi" || m["slug"] != "chu-dai-bi" {
		t.Errorf("data.mantras[0] = %v, unexpected fields", m)
	}
}

// TestBodhisattvaDetailHandler_NoMantrasReturnsEmptyArray covers AC2: no
// related mantras renders as an empty array, not null or an error, so the
// client can decide how to render "chưa có thần chú liên quan".
func TestBodhisattvaDetailHandler_NoMantrasReturnsEmptyArray(t *testing.T) {
	uc := &mockGetBodhisattvaUsecase{}
	uc.On("GetBodhisattvaBySlug", mock.Anything, "dia-tang").Return(
		&domain.Bodhisattva{ID: "bodhisattva-2", Slug: "dia-tang", Name: "Địa Tạng Vương Bồ Tát"},
		[]*domain.Mantra{},
		nil,
	)

	h := NewBodhisattvaDetailHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, bodhisattvaDetailRequest(t, "dia-tang"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	data := decodeResponse(t, rec)["data"].(map[string]any)
	mantras, ok := data["mantras"].([]any)
	if !ok || len(mantras) != 0 {
		t.Errorf("data.mantras = %v, want empty array", data["mantras"])
	}
}

// TestBodhisattvaDetailHandler_NotFound covers an unknown slug mapping to
// 404.
func TestBodhisattvaDetailHandler_NotFound(t *testing.T) {
	uc := &mockGetBodhisattvaUsecase{}
	uc.On("GetBodhisattvaBySlug", mock.Anything, "ghost").Return(nil, nil, domain.ErrBodhisattvaNotFound)

	h := NewBodhisattvaDetailHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, bodhisattvaDetailRequest(t, "ghost"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

// TestBodhisattvaDetailHandler_MissingSlug covers an empty path value.
func TestBodhisattvaDetailHandler_MissingSlug(t *testing.T) {
	uc := &mockGetBodhisattvaUsecase{}
	h := NewBodhisattvaDetailHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, bodhisattvaDetailRequest(t, ""))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	uc.AssertNotCalled(t, "GetBodhisattvaBySlug", mock.Anything, mock.Anything)
}

// TestBodhisattvaDetailHandler_UsecaseErrorMapsToSystemError covers a
// non-not-found failure mapping to 500, not a bare 200/404.
func TestBodhisattvaDetailHandler_UsecaseErrorMapsToSystemError(t *testing.T) {
	uc := &mockGetBodhisattvaUsecase{}
	uc.On("GetBodhisattvaBySlug", mock.Anything, "quan-the-am").Return(nil, nil, http.ErrHandlerTimeout)

	h := NewBodhisattvaDetailHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, bodhisattvaDetailRequest(t, "quan-the-am"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

// TestBodhisattvaDetailHandler_MethodNotAllowed covers unsupported verbs.
func TestBodhisattvaDetailHandler_MethodNotAllowed(t *testing.T) {
	uc := &mockGetBodhisattvaUsecase{}
	h := NewBodhisattvaDetailHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/v1/content/bodhisattvas/quan-the-am", nil)
	req.SetPathValue("slug", "quan-the-am")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
	}
	uc.AssertNotCalled(t, "GetBodhisattvaBySlug", mock.Anything, mock.Anything)
}
