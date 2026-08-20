package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

type mockLoginUsecase struct {
	mock.Mock
}

func (m *mockLoginUsecase) Login(ctx context.Context, input domain.LoginInput) (*domain.LoginResult, error) {
	args := m.Called(ctx, input)
	if r := args.Get(0); r != nil {
		return r.(*domain.LoginResult), args.Error(1)
	}
	return nil, args.Error(1)
}

const validLoginBody = `{
	"email": "new.user@example.com",
	"password": "s3cret!!",
	"device": {"platform": "iOS"}
}`

// TestLoginHandler_Success covers AC1: valid credentials return 200 with
// the token pair, in the same envelope shape as register.
func TestLoginHandler_Success(t *testing.T) {
	uc := &mockLoginUsecase{}
	uc.On("Login", mock.Anything, mock.MatchedBy(func(in domain.LoginInput) bool {
		return in.Email == "new.user@example.com" && in.Password == "s3cret!!" && in.Device.Platform == "iOS"
	})).Return(&domain.LoginResult{
		User:         &domain.User{ID: 7, Email: "new.user@example.com"},
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}, nil)

	h := NewLoginHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodPost, validLoginBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	got := decodeResponse(t, rec)
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("data field missing or wrong type: %v", got["data"])
	}
	if data["access_token"] != "access-token" || data["refresh_token"] != "refresh-token" {
		t.Errorf("tokens = %v/%v, want access-token/refresh-token", data["access_token"], data["refresh_token"])
	}
	uc.AssertExpectations(t)
}

// TestLoginHandler_MethodNotAllowed covers non-POST requests.
func TestLoginHandler_MethodNotAllowed(t *testing.T) {
	uc := &mockLoginUsecase{}
	h := NewLoginHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodGet, ""))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	uc.AssertNotCalled(t, "Login", mock.Anything, mock.Anything)
}

// TestLoginHandler_MalformedBody covers request decode failures.
func TestLoginHandler_MalformedBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"invalid json", "{"},
		{"unknown field", `{"email":"a@b.com","password":"x","bogus_field":true}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockLoginUsecase{}
			h := NewLoginHandler(uc)

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, newRequest(t, http.MethodPost, tt.body))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			uc.AssertNotCalled(t, "Login", mock.Anything, mock.Anything)
		})
	}
}

// TestLoginHandler_InvalidCredentials covers AC2/AC3: the use-case's single
// generic error maps to a single generic 401 response, regardless of
// whether the email or the password was wrong.
func TestLoginHandler_InvalidCredentials(t *testing.T) {
	uc := &mockLoginUsecase{}
	uc.On("Login", mock.Anything, mock.Anything).Return(nil, domain.ErrInvalidCredentials)

	h := NewLoginHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodPost, validLoginBody))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	got := decodeResponse(t, rec)
	errObj, ok := got["error"].(map[string]any)
	if !ok {
		t.Fatalf("error field missing or wrong type: %v", got["error"])
	}
	if errObj["code"] != "INVALID_CREDENTIALS" {
		t.Errorf("error.code = %v, want INVALID_CREDENTIALS", errObj["code"])
	}
}

// TestLoginHandler_UnexpectedError covers infra failures mapping to 500,
// distinct from bad credentials.
func TestLoginHandler_UnexpectedError(t *testing.T) {
	uc := &mockLoginUsecase{}
	uc.On("Login", mock.Anything, mock.Anything).Return(nil, context.DeadlineExceeded)

	h := NewLoginHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodPost, validLoginBody))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}
