package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

type mockForgotPasswordUsecase struct {
	mock.Mock
}

func (m *mockForgotPasswordUsecase) ForgotPassword(ctx context.Context, input domain.ForgotPasswordInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

const validForgotPasswordBody = `{"email": "new.user@example.com"}`

// TestForgotPasswordHandler_Success and
// TestForgotPasswordHandler_UnknownEmailStillSucceeds cover AC1/AC2: both
// a registered and unregistered email get the exact same 200 response —
// the use-case never reports which one happened.
func TestForgotPasswordHandler_Success(t *testing.T) {
	uc := &mockForgotPasswordUsecase{}
	uc.On("ForgotPassword", mock.Anything, domain.ForgotPasswordInput{Email: "new.user@example.com"}).Return(nil)

	h := NewForgotPasswordHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodPost, validForgotPasswordBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	got := decodeResponse(t, rec)
	data, ok := got["data"].(map[string]any)
	if !ok || data["message"] != forgotPasswordMessage {
		t.Errorf("data = %v, want message %q", got["data"], forgotPasswordMessage)
	}
}

func TestForgotPasswordHandler_UnknownEmailStillSucceeds(t *testing.T) {
	uc := &mockForgotPasswordUsecase{}
	// The real use-case swallows "not found" internally and returns nil —
	// this mock stands in for that behavior to prove delivery doesn't
	// special-case it either.
	uc.On("ForgotPassword", mock.Anything, mock.Anything).Return(nil)

	h := NewForgotPasswordHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodPost, `{"email":"ghost@example.com"}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestForgotPasswordHandler_MethodNotAllowed covers non-POST requests.
func TestForgotPasswordHandler_MethodNotAllowed(t *testing.T) {
	uc := &mockForgotPasswordUsecase{}
	h := NewForgotPasswordHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodGet, ""))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	uc.AssertNotCalled(t, "ForgotPassword", mock.Anything, mock.Anything)
}

// TestForgotPasswordHandler_MalformedBody covers request decode failures.
func TestForgotPasswordHandler_MalformedBody(t *testing.T) {
	uc := &mockForgotPasswordUsecase{}
	h := NewForgotPasswordHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodPost, "{"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	uc.AssertNotCalled(t, "ForgotPassword", mock.Anything, mock.Anything)
}

// TestForgotPasswordHandler_UnexpectedError covers a genuine infra
// failure surfacing as 500.
func TestForgotPasswordHandler_UnexpectedError(t *testing.T) {
	uc := &mockForgotPasswordUsecase{}
	uc.On("ForgotPassword", mock.Anything, mock.Anything).Return(errors.New("db down"))

	h := NewForgotPasswordHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodPost, validForgotPasswordBody))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}
