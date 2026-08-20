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

type mockListBodhisattvasUsecase struct {
	mock.Mock
}

func (m *mockListBodhisattvasUsecase) ListBodhisattvas(ctx context.Context, input domain.ListBodhisattvasInput) ([]*domain.Bodhisattva, error) {
	args := m.Called(ctx, input)
	if s := args.Get(0); s != nil {
		return s.([]*domain.Bodhisattva), args.Error(1)
	}
	return nil, args.Error(1)
}

func bodhisattvasRequest(t *testing.T, rawQuery string) *http.Request {
	t.Helper()
	url := "/v1/content/bodhisattvas"
	if rawQuery != "" {
		url += "?" + rawQuery
	}
	return httptest.NewRequest(http.MethodGet, url, nil)
}

// TestBodhisattvasHandler_ListSuccess covers AC1: name and image_url are
// present in the response.
func TestBodhisattvasHandler_ListSuccess(t *testing.T) {
	uc := &mockListBodhisattvasUsecase{}
	now := time.Now()
	uc.On("ListBodhisattvas", mock.Anything, domain.ListBodhisattvasInput{}).Return([]*domain.Bodhisattva{
		{ID: "bodhisattva-1", Slug: "quan-the-am", Name: "Quán Thế Âm Bồ Tát", ImageURL: "https://cdn.example.com/qta.png", CreatedAt: now, UpdatedAt: now},
	}, nil)

	h := NewBodhisattvasHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, bodhisattvasRequest(t, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	data := decodeResponse(t, rec)["data"].([]any)
	row := data[0].(map[string]any)
	if row["name"] != "Quán Thế Âm Bồ Tát" || row["image_url"] != "https://cdn.example.com/qta.png" {
		t.Errorf("data[0] = %v, unexpected fields", row)
	}
}

// TestBodhisattvasHandler_NoImageURLNoError covers AC2: a bodhisattva with
// no image_url renders as "" present, not an error.
func TestBodhisattvasHandler_NoImageURLNoError(t *testing.T) {
	uc := &mockListBodhisattvasUsecase{}
	uc.On("ListBodhisattvas", mock.Anything, domain.ListBodhisattvasInput{}).Return([]*domain.Bodhisattva{
		{ID: "bodhisattva-2", Slug: "dia-tang", ImageURL: ""},
	}, nil)

	h := NewBodhisattvasHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, bodhisattvasRequest(t, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	row := decodeResponse(t, rec)["data"].([]any)[0].(map[string]any)
	if v, ok := row["image_url"]; !ok || v != "" {
		t.Errorf("data[0].image_url = %v (present=%v), want empty string present", v, ok)
	}
}

// TestBodhisattvasHandler_UpdatedSinceParsed covers AC4: a valid RFC3339
// updated_since query param is parsed and passed through to the use-case.
func TestBodhisattvasHandler_UpdatedSinceParsed(t *testing.T) {
	uc := &mockListBodhisattvasUsecase{}
	since := time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC)
	uc.On("ListBodhisattvas", mock.Anything, domain.ListBodhisattvasInput{UpdatedSince: since}).Return([]*domain.Bodhisattva{}, nil)

	h := NewBodhisattvasHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, bodhisattvasRequest(t, "updated_since=2026-08-01T12:30:00Z"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	uc.AssertExpectations(t)
}

// TestBodhisattvasHandler_UpdatedSinceInvalidFormat covers rejecting a
// malformed updated_since instead of silently ignoring it or crashing.
func TestBodhisattvasHandler_UpdatedSinceInvalidFormat(t *testing.T) {
	uc := &mockListBodhisattvasUsecase{}
	h := NewBodhisattvasHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, bodhisattvasRequest(t, "updated_since=not-a-date"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	uc.AssertNotCalled(t, "ListBodhisattvas", mock.Anything, mock.Anything)
}

// TestBodhisattvasHandler_UsecaseErrorMapsToSystemError ensures a
// repository/use-case failure never presents to the client as an
// empty-but-successful list.
func TestBodhisattvasHandler_UsecaseErrorMapsToSystemError(t *testing.T) {
	uc := &mockListBodhisattvasUsecase{}
	uc.On("ListBodhisattvas", mock.Anything, mock.Anything).Return(nil, http.ErrHandlerTimeout)

	h := NewBodhisattvasHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, bodhisattvasRequest(t, ""))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

// TestBodhisattvasHandler_MethodNotAllowed covers unsupported verbs on this
// public read-only endpoint.
func TestBodhisattvasHandler_MethodNotAllowed(t *testing.T) {
	uc := &mockListBodhisattvasUsecase{}
	h := NewBodhisattvasHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/content/bodhisattvas", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
	}
	uc.AssertNotCalled(t, "ListBodhisattvas", mock.Anything, mock.Anything)
}

// TestBodhisattvasHandler_NoAuthRequired covers the "public read" note in
// content.md's Ghi chú kỹ thuật.
func TestBodhisattvasHandler_NoAuthRequired(t *testing.T) {
	uc := &mockListBodhisattvasUsecase{}
	uc.On("ListBodhisattvas", mock.Anything, domain.ListBodhisattvasInput{}).Return([]*domain.Bodhisattva{}, nil)

	h := NewBodhisattvasHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, bodhisattvasRequest(t, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
}
