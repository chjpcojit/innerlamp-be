package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

type mockRegisterUsecase struct {
	mock.Mock
}

func (m *mockRegisterUsecase) Register(ctx context.Context, input domain.RegisterInput) (*domain.RegisterResult, error) {
	args := m.Called(ctx, input)
	if r := args.Get(0); r != nil {
		return r.(*domain.RegisterResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func newRequest(t *testing.T, method, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, "/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response body is not valid JSON: %v (body=%s)", err, rec.Body.String())
	}
	return got
}

const validBody = `{
	"email": "new.user@example.com",
	"password": "s3cret!!",
	"phone_number": "+84912345678",
	"device": {"platform": "iOS"}
}`

// TestRegisterHandler_Success covers AC1: a successful register returns 201
// with the token pair and user info, without touching business logic
// (already covered at the usecase layer).
func TestRegisterHandler_Success(t *testing.T) {
	uc := &mockRegisterUsecase{}
	uc.On("Register", mock.Anything, mock.MatchedBy(func(in domain.RegisterInput) bool {
		return in.Email == "new.user@example.com" && in.PhoneNumber == "+84912345678" && in.Device.Platform == "iOS"
	})).Return(&domain.RegisterResult{
		User:         &domain.User{ID: 1, Email: "new.user@example.com", PhoneNumber: "+84912345678"},
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}, nil)

	h := NewRegisterHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodPost, validBody))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusCreated, rec.Body.String())
	}

	got := decodeResponse(t, rec)
	if got["success"] != true {
		t.Errorf("success = %v, want true", got["success"])
	}
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("data field missing or wrong type: %v", got["data"])
	}
	if data["access_token"] != "access-token" || data["refresh_token"] != "refresh-token" {
		t.Errorf("tokens = %v/%v, want access-token/refresh-token", data["access_token"], data["refresh_token"])
	}

	uc.AssertExpectations(t)
}

// TestRegisterHandler_MethodNotAllowed covers non-POST requests.
func TestRegisterHandler_MethodNotAllowed(t *testing.T) {
	uc := &mockRegisterUsecase{}
	h := NewRegisterHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodGet, ""))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	uc.AssertNotCalled(t, "Register", mock.Anything, mock.Anything)
}

// TestRegisterHandler_MalformedBody covers request decode failures (bad
// JSON, unknown fields) which must 400 before the use-case is invoked.
func TestRegisterHandler_MalformedBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"invalid json", "{"},
		{"unknown field", `{"email":"a@b.com","password":"password1","phone_number":"+84912345678","bogus_field":true}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockRegisterUsecase{}
			h := NewRegisterHandler(uc)

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, newRequest(t, http.MethodPost, tt.body))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			got := decodeResponse(t, rec)
			if got["success"] != false {
				t.Errorf("success = %v, want false", got["success"])
			}
			uc.AssertNotCalled(t, "Register", mock.Anything, mock.Anything)
		})
	}
}

// TestRegisterHandler_ErrorMapping covers AC2/AC3/AC4: each domain error
// returned by the use-case must map to the documented status/type.
func TestRegisterHandler_ErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		domainErr  error
		wantStatus int
		wantType   string
		wantCode   string
	}{
		{"email taken", domain.ErrEmailTaken, http.StatusConflict, "business", "EMAIL_TAKEN"},
		{"phone taken", domain.ErrPhoneTaken, http.StatusConflict, "business", "PHONE_TAKEN"},
		{"invalid email", domain.ErrInvalidEmail, http.StatusBadRequest, "validation", "INVALID_EMAIL"},
		{"invalid phone", domain.ErrInvalidPhone, http.StatusBadRequest, "validation", "INVALID_PHONE"},
		{"invalid password", domain.ErrInvalidPassword, http.StatusBadRequest, "validation", "INVALID_PASSWORD"},
		{"unexpected error", bytes.ErrTooLarge, http.StatusInternalServerError, "system", "INTERNAL_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockRegisterUsecase{}
			uc.On("Register", mock.Anything, mock.Anything).Return(nil, tt.domainErr)

			h := NewRegisterHandler(uc)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, newRequest(t, http.MethodPost, validBody))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, tt.wantStatus, rec.Body.String())
			}

			got := decodeResponse(t, rec)
			errObj, ok := got["error"].(map[string]any)
			if !ok {
				t.Fatalf("error field missing or wrong type: %v", got["error"])
			}
			if errObj["type"] != tt.wantType {
				t.Errorf("error.type = %v, want %v", errObj["type"], tt.wantType)
			}
			if errObj["code"] != tt.wantCode {
				t.Errorf("error.code = %v, want %v", errObj["code"], tt.wantCode)
			}
		})
	}
}
