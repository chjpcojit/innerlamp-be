package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

type mockResetPasswordUsecase struct {
	mock.Mock
}

func (m *mockResetPasswordUsecase) Reset(ctx context.Context, input domain.ResetPasswordInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

const validResetPasswordBody = `{"token": "raw-token", "new_password": "new-s3cret!"}`

// TestResetPasswordHandler_Success covers AC3.
func TestResetPasswordHandler_Success(t *testing.T) {
	uc := &mockResetPasswordUsecase{}
	uc.On("Reset", mock.Anything, domain.ResetPasswordInput{Token: "raw-token", NewPassword: "new-s3cret!"}).Return(nil)

	h := NewResetPasswordHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodPost, validResetPasswordBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	got := decodeResponse(t, rec)
	data, ok := got["data"].(map[string]any)
	if !ok || data["message"] != resetPasswordMessage {
		t.Errorf("data = %v, want message %q", got["data"], resetPasswordMessage)
	}
}

// TestResetPasswordHandler_MethodNotAllowed covers non-POST requests.
func TestResetPasswordHandler_MethodNotAllowed(t *testing.T) {
	uc := &mockResetPasswordUsecase{}
	h := NewResetPasswordHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodGet, ""))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	uc.AssertNotCalled(t, "Reset", mock.Anything, mock.Anything)
}

// TestResetPasswordHandler_MalformedBody covers request decode failures.
func TestResetPasswordHandler_MalformedBody(t *testing.T) {
	uc := &mockResetPasswordUsecase{}
	h := NewResetPasswordHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodPost, "{"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	uc.AssertNotCalled(t, "Reset", mock.Anything, mock.Anything)
}

// TestResetPasswordHandler_ErrorMapping covers AC3/AC4: token invalid vs
// weak password vs an unexpected infra failure must map to distinct
// statuses/codes.
func TestResetPasswordHandler_ErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		usecaseErr error
		wantStatus int
		wantType   string
		wantCode   string
	}{
		{"invalid token", domain.ErrResetTokenInvalid, http.StatusBadRequest, "business", "RESET_TOKEN_INVALID"},
		{"invalid password", domain.ErrInvalidPassword, http.StatusBadRequest, "validation", "INVALID_PASSWORD"},
		{"unexpected error", context.DeadlineExceeded, http.StatusInternalServerError, "system", "INTERNAL_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockResetPasswordUsecase{}
			uc.On("Reset", mock.Anything, mock.Anything).Return(tt.usecaseErr)

			h := NewResetPasswordHandler(uc)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, newRequest(t, http.MethodPost, validResetPasswordBody))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			got := decodeResponse(t, rec)
			errObj, ok := got["error"].(map[string]any)
			if !ok {
				t.Fatalf("error field missing or wrong type: %v", got["error"])
			}
			if errObj["type"] != tt.wantType || errObj["code"] != tt.wantCode {
				t.Errorf("error = %v, want type=%q code=%q", errObj, tt.wantType, tt.wantCode)
			}
		})
	}
}
