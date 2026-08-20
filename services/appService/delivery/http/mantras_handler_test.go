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

type mockListMantrasUsecase struct {
	mock.Mock
}

func (m *mockListMantrasUsecase) ListMantras(ctx context.Context, input domain.ListMantrasInput) ([]*domain.Mantra, error) {
	args := m.Called(ctx, input)
	if s := args.Get(0); s != nil {
		return s.([]*domain.Mantra), args.Error(1)
	}
	return nil, args.Error(1)
}

func mantrasRequest(t *testing.T, rawQuery string) *http.Request {
	t.Helper()
	url := "/v1/content/mantras"
	if rawQuery != "" {
		url += "?" + rawQuery
	}
	return httptest.NewRequest(http.MethodGet, url, nil)
}

// TestMantrasHandler_ListSuccess covers AC1: a plain GET with no query
// params returns the full list, including bodhisattva_id when set.
func TestMantrasHandler_ListSuccess(t *testing.T) {
	uc := &mockListMantrasUsecase{}
	now := time.Now()
	uc.On("ListMantras", mock.Anything, domain.ListMantrasInput{}).Return([]*domain.Mantra{
		{ID: "mantra-1", Slug: "chu-dai-bi", Title: "Chú Đại Bi", BodhisattvaID: "bodhisattva-1", CreatedAt: now, UpdatedAt: now},
	}, nil)

	h := NewMantrasHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, mantrasRequest(t, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	got := decodeResponse(t, rec)
	data, ok := got["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("data field missing, wrong type, or wrong length: %v", got["data"])
	}
	row := data[0].(map[string]any)
	if row["title"] != "Chú Đại Bi" || row["bodhisattva_id"] != "bodhisattva-1" {
		t.Errorf("data[0] = %v, unexpected fields", row)
	}
}

// TestMantrasHandler_NoBodhisattvaNoError covers AC2: a mantra with no
// bodhisattva_id renders as "" present, not an error.
func TestMantrasHandler_NoBodhisattvaNoError(t *testing.T) {
	uc := &mockListMantrasUsecase{}
	uc.On("ListMantras", mock.Anything, domain.ListMantrasInput{}).Return([]*domain.Mantra{
		{ID: "mantra-2", Slug: "om-mani-padme-hum", BodhisattvaID: ""},
	}, nil)

	h := NewMantrasHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, mantrasRequest(t, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	row := decodeResponse(t, rec)["data"].([]any)[0].(map[string]any)
	if v, ok := row["bodhisattva_id"]; !ok || v != "" {
		t.Errorf("data[0].bodhisattva_id = %v (present=%v), want empty string present", v, ok)
	}
}

// TestMantrasHandler_BodhisattvaFilter covers AC3: bodhisattva_id query
// param is passed through to the use-case.
func TestMantrasHandler_BodhisattvaFilter(t *testing.T) {
	uc := &mockListMantrasUsecase{}
	uc.On("ListMantras", mock.Anything, domain.ListMantrasInput{BodhisattvaID: "bodhisattva-1"}).Return([]*domain.Mantra{}, nil)

	h := NewMantrasHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, mantrasRequest(t, "bodhisattva_id=bodhisattva-1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	uc.AssertExpectations(t)
}

// TestMantrasHandler_UpdatedSinceParsed covers AC5: a valid RFC3339
// updated_since query param is parsed and passed through to the use-case.
func TestMantrasHandler_UpdatedSinceParsed(t *testing.T) {
	uc := &mockListMantrasUsecase{}
	since := time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC)
	uc.On("ListMantras", mock.Anything, domain.ListMantrasInput{UpdatedSince: since}).Return([]*domain.Mantra{}, nil)

	h := NewMantrasHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, mantrasRequest(t, "updated_since=2026-08-01T12:30:00Z"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	uc.AssertExpectations(t)
}

// TestMantrasHandler_UpdatedSinceInvalidFormat covers rejecting a malformed
// updated_since instead of silently ignoring it or crashing.
func TestMantrasHandler_UpdatedSinceInvalidFormat(t *testing.T) {
	uc := &mockListMantrasUsecase{}
	h := NewMantrasHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, mantrasRequest(t, "updated_since=not-a-date"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	uc.AssertNotCalled(t, "ListMantras", mock.Anything, mock.Anything)
}

// TestMantrasHandler_UsecaseErrorMapsToSystemError ensures a repository/
// use-case failure never presents to the client as an empty-but-successful
// list.
func TestMantrasHandler_UsecaseErrorMapsToSystemError(t *testing.T) {
	uc := &mockListMantrasUsecase{}
	uc.On("ListMantras", mock.Anything, mock.Anything).Return(nil, http.ErrHandlerTimeout)

	h := NewMantrasHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, mantrasRequest(t, ""))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

// TestMantrasHandler_MethodNotAllowed covers unsupported verbs on this
// public read-only endpoint.
func TestMantrasHandler_MethodNotAllowed(t *testing.T) {
	uc := &mockListMantrasUsecase{}
	h := NewMantrasHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/content/mantras", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
	}
	uc.AssertNotCalled(t, "ListMantras", mock.Anything, mock.Anything)
}

// TestMantrasHandler_NoAuthRequired covers the "public read" note in
// content.md's Ghi chú kỹ thuật.
func TestMantrasHandler_NoAuthRequired(t *testing.T) {
	uc := &mockListMantrasUsecase{}
	uc.On("ListMantras", mock.Anything, domain.ListMantrasInput{}).Return([]*domain.Mantra{}, nil)

	h := NewMantrasHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, mantrasRequest(t, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
}
