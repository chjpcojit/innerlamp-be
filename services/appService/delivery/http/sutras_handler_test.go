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

type mockListSutrasUsecase struct {
	mock.Mock
}

func (m *mockListSutrasUsecase) ListSutras(ctx context.Context, input domain.ListSutrasInput) ([]*domain.Sutra, error) {
	args := m.Called(ctx, input)
	if s := args.Get(0); s != nil {
		return s.([]*domain.Sutra), args.Error(1)
	}
	return nil, args.Error(1)
}

func sutrasRequest(t *testing.T, rawQuery string) *http.Request {
	t.Helper()
	url := "/v1/content/sutras"
	if rawQuery != "" {
		url += "?" + rawQuery
	}
	return httptest.NewRequest(http.MethodGet, url, nil)
}

// TestSutrasHandler_ListSuccess covers AC1: a plain GET with no query
// params returns the full list.
func TestSutrasHandler_ListSuccess(t *testing.T) {
	uc := &mockListSutrasUsecase{}
	now := time.Now()
	uc.On("ListSutras", mock.Anything, domain.ListSutrasInput{}).Return([]*domain.Sutra{
		{ID: "sutra-1", Slug: "kinh-phap-hoa", Title: "Kinh Pháp Hoa", Sect: "Bắc Tông", Language: "vi", CreatedAt: now, UpdatedAt: now},
	}, nil)

	h := NewSutrasHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, sutrasRequest(t, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	got := decodeResponse(t, rec)
	data, ok := got["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("data field missing, wrong type, or wrong length: %v", got["data"])
	}
	row := data[0].(map[string]any)
	if row["title"] != "Kinh Pháp Hoa" || row["sect"] != "Bắc Tông" {
		t.Errorf("data[0] = %v, unexpected fields", row)
	}
	// audio_url must be present as "" rather than omitted — same convention
	// as profileResponse — so the client cache schema stays uniform.
	if v, ok := row["audio_url"]; !ok || v != "" {
		t.Errorf("data[0].audio_url = %v (present=%v), want empty string present", v, ok)
	}
	uc.AssertExpectations(t)
}

// TestSutrasHandler_EmptyListReturnsEmptyArray ensures a nil/empty
// use-case result renders as "[]", not "null" — the client's JSON decoder
// shouldn't need to special-case an absent list.
func TestSutrasHandler_EmptyListReturnsEmptyArray(t *testing.T) {
	uc := &mockListSutrasUsecase{}
	uc.On("ListSutras", mock.Anything, domain.ListSutrasInput{}).Return([]*domain.Sutra{}, nil)

	h := NewSutrasHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, sutrasRequest(t, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := decodeResponse(t, rec); got["data"] == nil {
		t.Errorf("data = nil, want []")
	} else if arr, ok := got["data"].([]any); !ok || len(arr) != 0 {
		t.Errorf("data = %v, want empty array", got["data"])
	}
	uc.AssertExpectations(t)
}

// TestSutrasHandler_UpdatedSinceParsed covers AC3/AC6: a valid RFC3339
// updated_since query param is parsed and passed through to the use-case.
func TestSutrasHandler_UpdatedSinceParsed(t *testing.T) {
	uc := &mockListSutrasUsecase{}
	since := time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC)
	uc.On("ListSutras", mock.Anything, domain.ListSutrasInput{UpdatedSince: since}).Return([]*domain.Sutra{}, nil)

	h := NewSutrasHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, sutrasRequest(t, "updated_since=2026-08-01T12:30:00Z"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	uc.AssertExpectations(t)
}

// TestSutrasHandler_UpdatedSinceInvalidFormat covers rejecting a malformed
// updated_since instead of silently ignoring it or crashing.
func TestSutrasHandler_UpdatedSinceInvalidFormat(t *testing.T) {
	uc := &mockListSutrasUsecase{}
	h := NewSutrasHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, sutrasRequest(t, "updated_since=not-a-date"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	uc.AssertNotCalled(t, "ListSutras", mock.Anything, mock.Anything)
}

// TestSutrasHandler_SectFilter covers AC5: sect query param is passed
// through to the use-case.
func TestSutrasHandler_SectFilter(t *testing.T) {
	uc := &mockListSutrasUsecase{}
	uc.On("ListSutras", mock.Anything, domain.ListSutrasInput{Sect: "Nam Tông"}).Return([]*domain.Sutra{}, nil)

	h := NewSutrasHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, sutrasRequest(t, "sect=Nam+Tông"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	uc.AssertExpectations(t)
}

// TestSutrasHandler_UpdatedSinceAndSectCombined covers AC5+AC6 combined via
// the HTTP layer.
func TestSutrasHandler_UpdatedSinceAndSectCombined(t *testing.T) {
	uc := &mockListSutrasUsecase{}
	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	uc.On("ListSutras", mock.Anything, domain.ListSutrasInput{UpdatedSince: since, Sect: "Kim Cương Thừa"}).
		Return([]*domain.Sutra{}, nil)

	h := NewSutrasHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, sutrasRequest(t, "updated_since=2026-08-01T00:00:00Z&sect=Kim+Cương+Thừa"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	uc.AssertExpectations(t)
}

// TestSutrasHandler_UsecaseErrorMapsToSystemError ensures a repository/
// use-case failure never presents to the client as an empty-but-successful
// list.
func TestSutrasHandler_UsecaseErrorMapsToSystemError(t *testing.T) {
	uc := &mockListSutrasUsecase{}
	uc.On("ListSutras", mock.Anything, mock.Anything).Return(nil, http.ErrHandlerTimeout)

	h := NewSutrasHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, sutrasRequest(t, ""))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

// TestSutrasHandler_MethodNotAllowed covers unsupported verbs on this
// public read-only endpoint.
func TestSutrasHandler_MethodNotAllowed(t *testing.T) {
	uc := &mockListSutrasUsecase{}
	h := NewSutrasHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/content/sutras", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
	}
	uc.AssertNotCalled(t, "ListSutras", mock.Anything, mock.Anything)
}

// TestSutrasHandler_NoAuthRequired covers the "public read" note in
// content.md's Ghi chú kỹ thuật: a request with no auth context still
// succeeds, unlike ProfileHandler.
func TestSutrasHandler_NoAuthRequired(t *testing.T) {
	uc := &mockListSutrasUsecase{}
	uc.On("ListSutras", mock.Anything, domain.ListSutrasInput{}).Return([]*domain.Sutra{}, nil)

	h := NewSutrasHandler(uc)
	rec := httptest.NewRecorder()
	// sutrasRequest builds a plain request with no jwtcontext user id set.
	h.ServeHTTP(rec, sutrasRequest(t, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
}
