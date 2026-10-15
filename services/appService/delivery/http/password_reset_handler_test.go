package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func newResetTestHandler(t *testing.T) (*PasswordResetHandler, *mockPasswordResetUsecase) {
	uc := &mockPasswordResetUsecase{}
	return NewPasswordResetHandler(uc, newTestValidator(t)), uc
}

func TestPasswordResetHandler_RequestOTP(t *testing.T) {
	t.Run("success returns the neutral message and window numbers (AC23)", func(t *testing.T) {
		h, uc := newResetTestHandler(t)
		uc.On("RequestOTP", mock.Anything, domain.RequestPasswordResetInput{Email: "A@b.com"}).
			Return(&domain.RequestPasswordResetResult{ExpiresInSeconds: 600, ResendAfterSeconds: 60}, nil)

		rec := doRequest(h.RequestOTP, `{"email":"A@b.com"}`)

		require.Equal(t, http.StatusOK, rec.Code)
		var data passwordResetRequestResponse
		decodeSuccessData(t, rec, &data)
		assert.Equal(t, "Nếu email này có tài khoản, mã đã được gửi", data.Message)
		assert.Equal(t, 600, data.ExpiresInSeconds)
		assert.Equal(t, 60, data.ResendAfterSeconds)
	})

	t.Run("unknown field and malformed body are 400", func(t *testing.T) {
		h, _ := newResetTestHandler(t)
		assert.Equal(t, http.StatusBadRequest, doRequest(h.RequestOTP, `{"email":"a@b.com","x":1}`).Code)
		assert.Equal(t, http.StatusBadRequest, doRequest(h.RequestOTP, `{"email":`).Code)
	})

	errCases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"invalid email (AC27)", domain.ErrInvalidEmail, http.StatusBadRequest, "INVALID_EMAIL"},
		{"resend too soon (AC26)", &domain.OTPResendTooSoonError{RetryAfterSeconds: 30}, http.StatusTooManyRequests, "OTP_RESEND_TOO_SOON"},
		{"send limited (AC26)", &domain.OTPSendLimitedError{LockedUntil: time.Now()}, http.StatusTooManyRequests, "OTP_SEND_LIMITED"},
		{"locked (AC26)", &domain.OTPLockedError{LockedUntil: time.Now()}, http.StatusTooManyRequests, "OTP_LOCKED"},
	}
	for _, tc := range errCases {
		t.Run(tc.name, func(t *testing.T) {
			h, uc := newResetTestHandler(t)
			uc.On("RequestOTP", mock.Anything, mock.Anything).Return(nil, tc.err)

			rec := doRequest(h.RequestOTP, `{"email":"a@b.com"}`)

			require.Equal(t, tc.status, rec.Code)
			assert.Equal(t, tc.code, decodeErrorBody(t, rec).Code)
		})
	}
}

func TestPasswordResetHandler_Confirm(t *testing.T) {
	const body = `{"email":"a@b.com","otp_code":"042917","new_password":"a-new-password"}`

	t.Run("success is {success:true} with no tokens (AC28)", func(t *testing.T) {
		h, uc := newResetTestHandler(t)
		uc.On("Confirm", mock.Anything, domain.ConfirmPasswordResetInput{
			Email: "a@b.com", OTPCode: "042917", NewPassword: "a-new-password",
		}).Return(nil)

		rec := doRequest(h.Confirm, body)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.NotContains(t, rec.Body.String(), "token")
		assert.Contains(t, rec.Body.String(), `"success":true`)
	})

	t.Run("a code that is not six digits is VALIDATION_ERROR and never reaches the use case", func(t *testing.T) {
		h, uc := newResetTestHandler(t)

		for _, code := range []string{"", "12345", "1234567", "12345a"} {
			rec := doRequest(h.Confirm, `{"email":"a@b.com","otp_code":"`+code+`","new_password":"a-new-password"}`)
			assert.Equal(t, http.StatusBadRequest, rec.Code, code)
		}
		uc.AssertNotCalled(t, "Confirm", mock.Anything, mock.Anything)
	})

	errCases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"invalid email", domain.ErrInvalidEmail, http.StatusBadRequest, "INVALID_EMAIL"},
		{"invalid password (AC30)", domain.ErrInvalidPassword, http.StatusBadRequest, "INVALID_PASSWORD"},
		{"otp invalid (AC29)", domain.ErrOTPInvalid, http.StatusBadRequest, "OTP_INVALID"},
		{"otp expired (AC29)", domain.ErrOTPExpired, http.StatusBadRequest, "OTP_EXPIRED"},
		{"otp locked (AC29)", &domain.OTPLockedError{LockedUntil: time.Now()}, http.StatusTooManyRequests, "OTP_LOCKED"},
	}
	for _, tc := range errCases {
		t.Run(tc.name, func(t *testing.T) {
			h, uc := newResetTestHandler(t)
			uc.On("Confirm", mock.Anything, mock.Anything).Return(tc.err)

			rec := doRequest(h.Confirm, body)

			require.Equal(t, tc.status, rec.Code)
			assert.Equal(t, tc.code, decodeErrorBody(t, rec).Code)
		})
	}
}

// AC33: the daily mail budget surfaces as 429 RATE_LIMITED with the wait,
// rounded up and never 0, plus a Retry-After header.
func TestWriteDomainError_RateLimited(t *testing.T) {
	h, uc := newResetTestHandler(t)
	uc.On("RequestOTP", mock.Anything, mock.Anything).
		Return(nil, &domain.RateLimitedError{RetryAfter: 90*time.Minute + 200*time.Millisecond})

	rec := doRequest(h.RequestOTP, `{"email":"a@b.com"}`)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	got := decodeErrorBody(t, rec)
	assert.Equal(t, "RATE_LIMITED", got.Code)
	assert.EqualValues(t, 5401, got.Details["retry_after_seconds"])
	assert.Equal(t, "5401", rec.Header().Get("Retry-After"))
}
