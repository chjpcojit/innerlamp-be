package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	jwtclaims "chjpcoj.com.innerlamp/pkg/jwt/claims"
	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

func newLoginTestHandler() (*LoginHandler, *mockLoginUsecase) {
	uc := &mockLoginUsecase{}
	return NewLoginHandler(uc), uc
}

const validLoginBody = `{"phone_number":"0901234567","password":"secret123","device":{"platform":"ios","device_model":"iPhone 15","os_version":"17.4","app_version":"1.0.0","device_name":"My phone"}}`

func TestLoginHandler_Login(t *testing.T) {
	t.Run("success maps body and request metadata into the input and returns 200 (AC1)", func(t *testing.T) {
		h, uc := newLoginTestHandler()
		created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		uc.On("Login", mock.Anything, domain.LoginInput{
			PhoneNumber: "0901234567",
			Password:    "secret123",
			Device: domain.DeviceInfo{
				Platform: "ios", DeviceModel: "iPhone 15", OSVersion: "17.4", AppVersion: "1.0.0", DeviceName: "My phone",
				IPAddress: "203.0.113.9", UserAgent: "innerlamp-ios/1.0",
			},
		}).Return(&domain.LoginResult{
			User:         &domain.User{ID: 7, Email: "a@example.com", PhoneNumber: "+84901234567", PasswordHash: "SECRET-HASH", CreatedAt: created},
			AccessToken:  "acc",
			RefreshToken: "ref",
		}, nil)

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(validLoginBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "innerlamp-ios/1.0")
		req.RemoteAddr = "203.0.113.9:51234"
		rec := httptest.NewRecorder()
		h.Login(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.NotContains(t, rec.Body.String(), "SECRET-HASH", "the password hash must never reach the wire")
		assert.NotContains(t, rec.Body.String(), "password_hash")

		var data verifyOTPResponse
		decodeSuccessData(t, rec, &data)
		assert.Equal(t, "acc", data.AccessToken)
		assert.Equal(t, "ref", data.RefreshToken)
		assert.Equal(t, int64(7), data.User.ID)
		assert.Equal(t, "+84901234567", data.User.PhoneNumber)
		uc.AssertExpectations(t)
	})

	t.Run("client cannot inject ip_address or user_agent through the body", func(t *testing.T) {
		h, uc := newLoginTestHandler()
		rec := doRequest(h.Login, `{"phone_number":"0901234567","password":"x","device":{"platform":"ios","device_model":"m","ip_address":"1.2.3.4"}}`)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "VALIDATION_ERROR", decodeErrorBody(t, rec).Code)
		uc.AssertNotCalled(t, "Login", mock.Anything, mock.Anything)
	})

	t.Run("body errors are 400 and never reach the use case", func(t *testing.T) {
		for name, body := range map[string]string{
			"empty":         ``,
			"malformed":     `{"phone_number":`,
			"unknown field": `{"phone_number":"0901234567","password":"x","nope":1}`,
		} {
			t.Run(name, func(t *testing.T) {
				h, uc := newLoginTestHandler()
				rec := doRequest(h.Login, body)
				require.Equal(t, http.StatusBadRequest, rec.Code)
				assert.Equal(t, "VALIDATION_ERROR", decodeErrorBody(t, rec).Code)
				uc.AssertNotCalled(t, "Login", mock.Anything, mock.Anything)
			})
		}
	})

	t.Run("oversized body is 413 PAYLOAD_TOO_LARGE, not 400 (json-body-hardening AC7)", func(t *testing.T) {
		h, uc := newLoginTestHandler()
		// Over DecodeJSONBodyStrict's 1 MB cap; valid JSON otherwise, so only
		// the size gate can reject it.
		oversized := `{"phone_number":"0901234567","password":"` + strings.Repeat("x", 1<<20) + `"}`
		rec := doRequest(h.Login, oversized)

		require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
		got := decodeErrorBody(t, rec)
		assert.Equal(t, "PAYLOAD_TOO_LARGE", got.Code)
		assert.Equal(t, utils.ErrBodyTooLarge.Error(), got.Message)
		uc.AssertNotCalled(t, "Login", mock.Anything, mock.Anything)
	})

	lockedUntil := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	errorCases := []struct {
		name    string
		err     error
		status  int
		code    string
		details map[string]any
	}{
		{"invalid credentials (AC2)", domain.ErrInvalidCredentials, http.StatusUnauthorized, "INVALID_CREDENTIALS", nil},
		{"wrapped invalid credentials", wrapErr(domain.ErrInvalidCredentials), http.StatusUnauthorized, "INVALID_CREDENTIALS", nil},
		{"locked (AC7)", &domain.LoginLockedError{LockedUntil: lockedUntil}, http.StatusTooManyRequests, "LOGIN_LOCKED", map[string]any{"locked_until": lockedUntil.Format(time.RFC3339)}},
		{"invalid device (AC5)", domain.ErrInvalidDevice, http.StatusBadRequest, "INVALID_DEVICE", map[string]any{"field": "device"}},
		{"invalid phone (AC6)", domain.ErrInvalidPhone, http.StatusBadRequest, "INVALID_PHONE", map[string]any{"field": "phone_number"}},
		{"invalid password (AC6)", domain.ErrInvalidPassword, http.StatusBadRequest, "INVALID_PASSWORD", map[string]any{"field": "password"}},
		{"unmapped infra error", errUnmapped, http.StatusInternalServerError, "INTERNAL_ERROR", nil},
	}
	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			h, uc := newLoginTestHandler()
			uc.On("Login", mock.Anything, mock.Anything).Return(nil, tc.err)

			rec := doRequest(h.Login, validLoginBody)

			require.Equal(t, tc.status, rec.Code)
			got := decodeErrorBody(t, rec)
			assert.Equal(t, tc.code, got.Code)
			assert.Equal(t, tc.details, got.Details)
			assert.NotContains(t, rec.Body.String(), errUnmapped.Error(), "an internal cause must not leak")
		})
	}
}

func TestLoginHandler_Refresh(t *testing.T) {
	t.Run("success returns the rotated pair (AC9)", func(t *testing.T) {
		h, uc := newLoginTestHandler()
		uc.On("Refresh", mock.Anything, domain.RefreshInput{RefreshToken: "old"}).
			Return(&domain.RefreshResult{AccessToken: "acc2", RefreshToken: "ref2"}, nil)

		rec := doRequest(h.Refresh, `{"refresh_token":"old"}`)

		require.Equal(t, http.StatusOK, rec.Code)
		var data refreshResponse
		decodeSuccessData(t, rec, &data)
		assert.Equal(t, refreshResponse{AccessToken: "acc2", RefreshToken: "ref2"}, data)
	})

	t.Run("invalid token is 401 REFRESH_TOKEN_INVALID (AC10/AC11)", func(t *testing.T) {
		h, uc := newLoginTestHandler()
		uc.On("Refresh", mock.Anything, mock.Anything).Return(nil, domain.ErrRefreshTokenInvalid)

		rec := doRequest(h.Refresh, `{"refresh_token":"old"}`)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "REFRESH_TOKEN_INVALID", decodeErrorBody(t, rec).Code)
	})

	t.Run("infra error is 500 without leaking", func(t *testing.T) {
		h, uc := newLoginTestHandler()
		uc.On("Refresh", mock.Anything, mock.Anything).Return(nil, errUnmapped)

		rec := doRequest(h.Refresh, `{"refresh_token":"old"}`)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.NotContains(t, rec.Body.String(), errUnmapped.Error())
	})

	t.Run("bad bodies are 400", func(t *testing.T) {
		for name, body := range map[string]string{
			"empty":         ``,
			"unknown field": `{"refresh_token":"x","extra":1}`,
			"malformed":     `{`,
		} {
			t.Run(name, func(t *testing.T) {
				h, uc := newLoginTestHandler()
				rec := doRequest(h.Refresh, body)
				require.Equal(t, http.StatusBadRequest, rec.Code)
				uc.AssertNotCalled(t, "Refresh", mock.Anything, mock.Anything)
			})
		}
	})

	t.Run("oversized body is 413 PAYLOAD_TOO_LARGE (json-body-hardening AC7)", func(t *testing.T) {
		h, uc := newLoginTestHandler()
		oversized := `{"refresh_token":"` + strings.Repeat("x", 1<<20) + `"}`
		rec := doRequest(h.Refresh, oversized)

		require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
		assert.Equal(t, "PAYLOAD_TOO_LARGE", decodeErrorBody(t, rec).Code)
		uc.AssertNotCalled(t, "Refresh", mock.Anything, mock.Anything)
	})
}

// logoutRequest builds a POST /logout with the claims the auth middleware would
// have put in the context. extra may be nil.
func logoutRequest(userID string, extra map[string]any) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	claims := &jwtclaims.CustomClaims{
		UserID:           userID,
		TokenType:        jwtclaims.TokenTypeAccess,
		Extra:            extra,
		RegisteredClaims: jwt.RegisteredClaims{},
	}
	return req.WithContext(jwtcontext.WithClaims(req.Context(), claims))
}

func TestLoginHandler_Logout(t *testing.T) {
	t.Run("200 envelope and revokes the session named by the token (AC13)", func(t *testing.T) {
		h, uc := newLoginTestHandler()
		uc.On("Logout", mock.Anything, domain.LogoutInput{UserID: 42, SessionID: "sess-1"}).Return(nil)

		rec := httptest.NewRecorder()
		h.Logout(rec, logoutRequest("42", map[string]any{"sid": "sess-1"}))

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"success": true}`, rec.Body.String())
		uc.AssertExpectations(t)
	})

	t.Run("no claims in context is 401 UNAUTHORIZED (AC15)", func(t *testing.T) {
		h, uc := newLoginTestHandler()
		rec := httptest.NewRecorder()
		h.Logout(rec, httptest.NewRequest(http.MethodPost, "/", nil))

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "UNAUTHORIZED", decodeErrorBody(t, rec).Code)
		uc.AssertNotCalled(t, "Logout", mock.Anything, mock.Anything)
	})

	badClaims := map[string]*http.Request{
		"missing sid":      logoutRequest("42", nil),
		"empty sid":        logoutRequest("42", map[string]any{"sid": ""}),
		"non-string sid":   logoutRequest("42", map[string]any{"sid": 123}),
		"non-numeric user": logoutRequest("abc", map[string]any{"sid": "s"}),
		"empty user id":    logoutRequest("", map[string]any{"sid": "s"}),
		"other extra keys": logoutRequest("42", map[string]any{"ver": 1}),
	}
	for name, req := range badClaims {
		t.Run(name+" is 401 and touches nothing (AC15)", func(t *testing.T) {
			h, uc := newLoginTestHandler()
			rec := httptest.NewRecorder()
			h.Logout(rec, req)

			require.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.Equal(t, "UNAUTHORIZED", decodeErrorBody(t, rec).Code)
			uc.AssertNotCalled(t, "Logout", mock.Anything, mock.Anything)
		})
	}

	t.Run("infra error is 500", func(t *testing.T) {
		h, uc := newLoginTestHandler()
		uc.On("Logout", mock.Anything, mock.Anything).Return(errUnmapped)

		rec := httptest.NewRecorder()
		h.Logout(rec, logoutRequest("42", map[string]any{"sid": "s"}))

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.NotContains(t, rec.Body.String(), errUnmapped.Error())
	})
}

func TestWriteAuthError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteAuthError(rec, httptest.NewRequest(http.MethodPost, "/", nil), assert.AnError)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	got := decodeErrorBody(t, rec)
	assert.Equal(t, "UNAUTHORIZED", got.Code)
	assert.NotContains(t, rec.Body.String(), assert.AnError.Error(), "the cause must not be echoed")

	var raw map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &raw))
	assert.Equal(t, false, raw["success"], "same ApiResponse envelope as every other route")
}

// wrapErr wraps err the way the use case does on its way out of a failing call.
func wrapErr(err error) error { return &wrapped{err} }

type wrapped struct{ err error }

func (w *wrapped) Error() string { return "login: " + w.err.Error() }
func (w *wrapped) Unwrap() error { return w.err }
