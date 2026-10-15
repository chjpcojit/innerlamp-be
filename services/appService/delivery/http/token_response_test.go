package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func TestNoStore_SetsHeadersOnSuccessAndError(t *testing.T) {
	// AC20: the headers are set before the handler runs, so a response that
	// fails (or panics later) is covered exactly like a success.
	for name, status := range map[string]int{"success": http.StatusOK, "error": http.StatusUnauthorized} {
		t.Run(name, func(t *testing.T) {
			wrapped := NoStore{}.WrapFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			})
			rec := httptest.NewRecorder()
			wrapped(rec, httptest.NewRequest(http.MethodPost, "/", nil))

			require.Equal(t, status, rec.Code)
			assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
			assert.Equal(t, "no-cache", rec.Header().Get("Pragma"))
		})
	}
}

func TestLoginHandler_ExpiresIn(t *testing.T) {
	t.Run("login returns the access token TTL in seconds (AC21)", func(t *testing.T) {
		uc := &mockLoginUsecase{}
		h := NewLoginHandler(uc, WithAccessTokenTTL(15*time.Minute))
		uc.On("Login", mock.Anything, mock.Anything).Return(&domain.LoginResult{
			User:         &domain.User{ID: 7, Email: "a@example.com", PhoneNumber: "+84901234567"},
			AccessToken:  "acc",
			RefreshToken: "ref",
		}, nil)

		rec := doRequest(h.Login, validLoginBody)

		require.Equal(t, http.StatusOK, rec.Code)
		var data verifyOTPResponse
		decodeSuccessData(t, rec, &data)
		assert.Equal(t, int64(900), data.ExpiresIn)
		assert.Equal(t, "acc", data.AccessToken)
	})

	t.Run("refresh returns it too, still without user (AC22)", func(t *testing.T) {
		uc := &mockLoginUsecase{}
		h := NewLoginHandler(uc, WithAccessTokenTTL(15*time.Minute))
		uc.On("Refresh", mock.Anything, domain.RefreshInput{RefreshToken: "old"}).
			Return(&domain.RefreshResult{AccessToken: "acc2", RefreshToken: "ref2"}, nil)

		rec := doRequest(h.Refresh, `{"refresh_token":"old"}`)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"expires_in":900`)
		assert.NotContains(t, rec.Body.String(), `"user"`)
	})
}

func TestRegisterHandler_VerifyOTP_ExpiresIn(t *testing.T) {
	// AC21 for the route that lives in onboarding's handler.
	uc := &mockRegisterUsecase{}
	h := NewRegisterHandler(uc, newTestValidator(t), WithAccessTokenTTL(15*time.Minute))
	uc.On("VerifyOTP", mock.Anything, mock.Anything).Return(&domain.RegisterResult{
		User:         &domain.User{ID: 1, Email: "a@example.com", PhoneNumber: "+84901234567"},
		AccessToken:  "access",
		RefreshToken: "refresh",
	}, nil)

	rec := doRequest(h.VerifyOTP, `{"email":"a@example.com","otp_code":"111111"}`)

	require.Equal(t, http.StatusCreated, rec.Code)
	var data verifyOTPResponse
	decodeSuccessData(t, rec, &data)
	assert.Equal(t, int64(900), data.ExpiresIn)
}
