package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

type mockAvatarUsecase struct {
	mock.Mock
}

func (m *mockAvatarUsecase) UpdateAvatar(ctx context.Context, userID int64, reader io.Reader, contentType string) (*domain.User, error) {
	args := m.Called(ctx, userID, reader, contentType)
	if u := args.Get(0); u != nil {
		return u.(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

// pngBytes is a minimal buffer starting with the PNG magic signature — real
// enough for http.DetectContentType to sniff "image/png" without needing a
// fully valid PNG file structure.
const pngBytes = "\x89PNG\r\n\x1a\n" + "0123456789abcdef"

func avatarRequest(t *testing.T, method, body, contentType, userID string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, "/v1/users/me/avatar", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if userID != "" {
		req = req.WithContext(jwtcontext.WithUserID(req.Context(), userID))
	}
	return req
}

// TestAvatarHandler_Success covers AC3: a new avatar upload returns the
// updated profile with the new avatar_url. The content type passed to the
// use-case is the server-sniffed one, not the client-supplied header.
func TestAvatarHandler_Success(t *testing.T) {
	uc := &mockAvatarUsecase{}
	uc.On("UpdateAvatar", mock.Anything, int64(7), mock.Anything, "image/png").
		Return(&domain.User{ID: 7, AvatarURL: "https://cdn.example.com/avatars/7"}, nil)

	h := NewAvatarHandler(uc)
	rec := httptest.NewRecorder()
	// Client claims JPEG but sends PNG bytes — the sniffed type must win.
	h.ServeHTTP(rec, avatarRequest(t, http.MethodPost, pngBytes, "image/jpeg", "7"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	got := decodeResponse(t, rec)
	data := got["data"].(map[string]any)
	if data["avatar_url"] != "https://cdn.example.com/avatars/7" {
		t.Errorf("data.avatar_url = %v, want new URL", data["avatar_url"])
	}
	uc.AssertExpectations(t)
}

// TestAvatarHandler_RejectsNonImage covers the content-type spoofing case:
// a client claiming image/jpeg but sending non-image bytes must be rejected
// on the server-sniffed type, never on the client's header.
func TestAvatarHandler_RejectsNonImage(t *testing.T) {
	uc := &mockAvatarUsecase{}
	h := NewAvatarHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, avatarRequest(t, http.MethodPost, "<html><body>not an image</body></html>", "image/jpeg", "7"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	uc.AssertNotCalled(t, "UpdateAvatar", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestAvatarHandler_TooLarge covers the size cap: a body over maxAvatarBytes
// is rejected before the use-case is ever called.
func TestAvatarHandler_TooLarge(t *testing.T) {
	uc := &mockAvatarUsecase{}
	h := NewAvatarHandler(uc)

	oversized := pngBytes + strings.Repeat("x", maxAvatarBytes)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, avatarRequest(t, http.MethodPost, oversized, "image/png", "7"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	uc.AssertNotCalled(t, "UpdateAvatar", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestAvatarHandler_Unauthorized covers AC6.
func TestAvatarHandler_Unauthorized(t *testing.T) {
	uc := &mockAvatarUsecase{}
	h := NewAvatarHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, avatarRequest(t, http.MethodPost, pngBytes, "image/jpeg", ""))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	uc.AssertNotCalled(t, "UpdateAvatar", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestAvatarHandler_MethodNotAllowed covers non-POST requests.
func TestAvatarHandler_MethodNotAllowed(t *testing.T) {
	uc := &mockAvatarUsecase{}
	h := NewAvatarHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, avatarRequest(t, http.MethodGet, "", "", "7"))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
	}
	uc.AssertNotCalled(t, "UpdateAvatar", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestAvatarHandler_UsecaseError maps use-case failures to 500.
func TestAvatarHandler_UsecaseError(t *testing.T) {
	uc := &mockAvatarUsecase{}
	uc.On("UpdateAvatar", mock.Anything, int64(7), mock.Anything, "image/png").
		Return(nil, context.DeadlineExceeded)

	h := NewAvatarHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, avatarRequest(t, http.MethodPost, pngBytes, "image/png", "7"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}
