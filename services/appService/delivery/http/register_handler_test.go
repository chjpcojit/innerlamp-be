package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/pkg/validation"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// errUnmapped stands in for an infrastructure failure the sentinel switch in
// writeDomainError does not recognize — it must fall through to 500
// INTERNAL_ERROR rather than leak whatever this error says.
var errUnmapped = errors.New("some unmapped infra failure")

// newTestValidator mirrors what cmd/main.go's composition root must do
// before any handler is built: register the `phone` tag and pick json as
// the field-name tag, so validation errors report json keys
// ("phone_number") rather than Go field names ("PhoneNumber").
func newTestValidator(t *testing.T) *validation.Validator {
	t.Helper()
	v := validation.New()
	require.NoError(t, v.Register(validation.PhoneTag, validation.PhoneRule))
	v.SetFieldNameTag("json")
	return v
}

func newTestHandler(t *testing.T, opts ...HandlerOption) (*RegisterHandler, *mockRegisterUsecase) {
	t.Helper()
	uc := &mockRegisterUsecase{}
	return NewRegisterHandler(uc, newTestValidator(t), opts...), uc
}

func doRequest(h http.HandlerFunc, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

// decodedError is the test's view of an error reply — pulled out of the
// shared utils.ApiResponse envelope (Success/Error/Details) every handler
// writes via utils.WriteValidationError/WriteBusinessError/WriteSystemError.
type decodedError struct {
	Code    string
	Message string
	Details map[string]any
}

func decodeErrorBody(t *testing.T, rec *httptest.ResponseRecorder) decodedError {
	t.Helper()
	var resp utils.ApiResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.NotNil(t, resp.Error)
	return decodedError{Code: resp.Error.Code, Message: resp.Error.Message, Details: resp.Details}
}

// decodeSuccessData unwraps a utils.WriteSuccessResponse(WithStatus) reply's
// "data" field into dst.
func decodeSuccessData(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	var resp struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.NoError(t, json.Unmarshal(resp.Data, dst))
}

// --- RequestOTP -------------------------------------------------------------

const requestBody = `{"email":"A@Example.com","password":"password123","phone_country":"VN","phone_number":"090 123 4567","full_name":"Nguyen Van A","dharma_name":"Thich A","ordination_temple_id":"temple_north_011"}`

func TestRegisterHandler_RequestOTP_Success(t *testing.T) {
	h, uc := newTestHandler(t)
	uc.On("RequestOTP", mock.Anything, domain.RequestRegisterOTPInput{
		Email:              "A@Example.com",
		Password:           "password123",
		PhoneCountry:       "VN",
		PhoneNumber:        "090 123 4567",
		FullName:           "Nguyen Van A",
		DharmaName:         "Thich A",
		OrdinationTempleID: "temple_north_011",
	}).Return(&domain.RequestOTPResult{ExpiresInSeconds: 600, ResendAfterSeconds: 60}, nil)

	rec := doRequest(h.RequestOTP, requestBody)

	require.Equal(t, http.StatusOK, rec.Code)
	var data map[string]any
	decodeSuccessData(t, rec, &data)
	require.Equal(t, map[string]any{"message": "otp sent", "expires_in_seconds": float64(600), "resend_after_seconds": float64(60)}, data,
		"exactly the API table's body — nothing that could differ between AC1 and AC2")
	uc.AssertExpectations(t)
}

// Empty fields reach the use case so each gets its own code (AC3–AC6), not a
// generic VALIDATION_ERROR.
func TestRegisterHandler_RequestOTP_EmptyFieldsReachUsecase(t *testing.T) {
	h, uc := newTestHandler(t)
	uc.On("RequestOTP", mock.Anything, domain.RequestRegisterOTPInput{}).Return(nil, domain.ErrInvalidEmail)

	rec := doRequest(h.RequestOTP, `{}`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	body := decodeErrorBody(t, rec)
	require.Equal(t, "INVALID_EMAIL", body.Code)
	require.Equal(t, "email", body.Details["field"])
}

func TestRegisterHandler_RequestOTP_DecodeErrors(t *testing.T) {
	cases := map[string]string{
		"malformed JSON":               `{not json`,
		"unknown field":                `{"email":"a@example.com","surprise":true}`,
		"legacy ordination_place":      `{"email":"a@example.com","ordination_place":"Chùa"}`,
		"phone_number wrong JSON type": `{"email":"a@example.com","phone_number":901234567}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			rec := doRequest(h.RequestOTP, body)
			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Equal(t, "VALIDATION_ERROR", decodeErrorBody(t, rec).Code)
			uc.AssertNotCalled(t, "RequestOTP", mock.Anything, mock.Anything)
		})
	}
}

func TestRegisterHandler_RequestOTP_DomainErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantField  string
	}{
		{"AC3 invalid email", domain.ErrInvalidEmail, http.StatusBadRequest, "INVALID_EMAIL", "email"},
		{"AC4 invalid phone", domain.ErrInvalidPhone, http.StatusBadRequest, "INVALID_PHONE", "phone_number"},
		{"AC5 invalid password", domain.ErrInvalidPassword, http.StatusBadRequest, "INVALID_PASSWORD", "password"},
		{"AC6 invalid full name", domain.ErrInvalidFullName, http.StatusBadRequest, "INVALID_FULL_NAME", "full_name"},
		{"AC6 invalid dharma name", domain.ErrInvalidDharmaName, http.StatusBadRequest, "INVALID_DHARMA_NAME", "dharma_name"},
		{"AC7 phone taken", domain.ErrPhoneTaken, http.StatusConflict, "PHONE_TAKEN", ""},
		{"AC8 invalid temple", domain.ErrInvalidTemple, http.StatusBadRequest, "INVALID_TEMPLE", "ordination_temple_id"},
		{"AC10 email send failed", fmt.Errorf("%w: smtp 451 secret-host", domain.ErrEmailSendFailed), http.StatusInternalServerError, "EMAIL_SEND_FAILED", ""},
		{"unmapped infra failure", errUnmapped, http.StatusInternalServerError, "INTERNAL_ERROR", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			uc.On("RequestOTP", mock.Anything, mock.Anything).Return(nil, tt.err)

			rec := doRequest(h.RequestOTP, requestBody)

			require.Equal(t, tt.wantStatus, rec.Code)
			body := decodeErrorBody(t, rec)
			require.Equal(t, tt.wantCode, body.Code)
			if tt.wantField != "" {
				require.Equal(t, tt.wantField, body.Details["field"])
			}
			require.NotContains(t, rec.Body.String(), "secret-host", "the mailer's own error never reaches the client")
			require.NotContains(t, rec.Body.String(), errUnmapped.Error())
		})
	}
}

// AC9: both send-policy refusals carry their number.
func TestRegisterHandler_RequestOTP_SendRefused(t *testing.T) {
	t.Run("too soon", func(t *testing.T) {
		h, uc := newTestHandler(t)
		uc.On("RequestOTP", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("save: %w", &domain.OTPResendTooSoonError{RetryAfterSeconds: 42}))

		rec := doRequest(h.RequestOTP, requestBody)

		require.Equal(t, http.StatusTooManyRequests, rec.Code)
		body := decodeErrorBody(t, rec)
		require.Equal(t, "OTP_RESEND_TOO_SOON", body.Code)
		require.Equal(t, float64(42), body.Details["retry_after_seconds"])
	})
	t.Run("send limited", func(t *testing.T) {
		h, uc := newTestHandler(t)
		deadline := time.Now().Add(30 * time.Minute).UTC().Truncate(time.Second)
		uc.On("RequestOTP", mock.Anything, mock.Anything).Return(nil, &domain.OTPSendLimitedError{LockedUntil: deadline})

		rec := doRequest(h.RequestOTP, requestBody)

		require.Equal(t, http.StatusTooManyRequests, rec.Code)
		body := decodeErrorBody(t, rec)
		require.Equal(t, "OTP_SEND_LIMITED", body.Code)
		got, err := time.Parse(time.RFC3339, body.Details["locked_until"].(string))
		require.NoError(t, err)
		require.WithinDuration(t, deadline, got, time.Second)
	})
}

// TestRegisterHandler_RequestOTP_BodyTooLarge: an oversized body answers 413
// PAYLOAD_TOO_LARGE instead of 400 VALIDATION_ERROR.
func TestRegisterHandler_RequestOTP_BodyTooLarge(t *testing.T) {
	h, uc := newTestHandler(t)
	oversized := `{"email":"a@example.com","password":"` + strings.Repeat("x", 1<<20) + `"}`

	rec := doRequest(h.RequestOTP, oversized)

	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	body := decodeErrorBody(t, rec)
	require.Equal(t, "PAYLOAD_TOO_LARGE", body.Code)
	require.Equal(t, utils.ErrBodyTooLarge.Error(), body.Message)
	uc.AssertNotCalled(t, "RequestOTP", mock.Anything, mock.Anything)
}

// --- ResendOTP --------------------------------------------------------------

func TestRegisterHandler_ResendOTP_Success(t *testing.T) {
	h, uc := newTestHandler(t)
	uc.On("ResendOTP", mock.Anything, domain.ResendOTPInput{Email: "a@example.com"}).
		Return(&domain.ResendOTPResult{ExpiresInSeconds: 600, ResendAfterSeconds: 60, ResendRemaining: 2}, nil)

	rec := doRequest(h.ResendOTP, `{"email":"a@example.com"}`)

	require.Equal(t, http.StatusOK, rec.Code)
	var data map[string]any
	decodeSuccessData(t, rec, &data)
	require.Equal(t, map[string]any{"message": "otp sent", "expires_in_seconds": float64(600), "resend_after_seconds": float64(60), "resend_remaining": float64(2)}, data)
}

func TestRegisterHandler_ResendOTP_Errors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"AC3 invalid email", domain.ErrInvalidEmail, http.StatusBadRequest, "INVALID_EMAIL"},
		{"AC12 pending expired", domain.ErrPendingRegistrationExpired, http.StatusBadRequest, "PENDING_REGISTRATION_EXPIRED"},
		{"AC12 pending not found reads as expired", domain.ErrPendingRegistrationNotFound, http.StatusBadRequest, "PENDING_REGISTRATION_EXPIRED"},
		{"AC9 too soon", &domain.OTPResendTooSoonError{RetryAfterSeconds: 5}, http.StatusTooManyRequests, "OTP_RESEND_TOO_SOON"},
		{"AC9 send limited", &domain.OTPSendLimitedError{LockedUntil: time.Now().Add(time.Minute)}, http.StatusTooManyRequests, "OTP_SEND_LIMITED"},
		{"AC10 email send failed", domain.ErrEmailSendFailed, http.StatusInternalServerError, "EMAIL_SEND_FAILED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			uc.On("ResendOTP", mock.Anything, mock.Anything).Return(nil, tt.err)

			rec := doRequest(h.ResendOTP, `{"email":"a@example.com"}`)

			require.Equal(t, tt.wantStatus, rec.Code)
			require.Equal(t, tt.wantCode, decodeErrorBody(t, rec).Code)
		})
	}
}

func TestRegisterHandler_ResendOTP_LegacyBodyRejected(t *testing.T) {
	h, uc := newTestHandler(t)
	rec := doRequest(h.ResendOTP, `{"phone_number":"0901234567","purpose":"register"}`)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "VALIDATION_ERROR", decodeErrorBody(t, rec).Code)
	uc.AssertNotCalled(t, "ResendOTP", mock.Anything, mock.Anything)
}

// --- VerifyOTP ------------------------------------------------------------

func TestRegisterHandler_VerifyOTP_Success(t *testing.T) {
	h, uc := newTestHandler(t, WithAccessTokenTTL(15*time.Minute))
	createdAt := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	dharma := "Thich Test"
	fullName := "Nguyen Van A"
	verified := createdAt
	uc.On("VerifyOTP", mock.Anything, domain.VerifyRegisterOTPInput{Email: "a@example.com", OTPCode: "042917"}).
		Return(&domain.RegisterResult{
			User: &domain.User{
				ID: 1, Email: "a@example.com", EmailVerifiedAt: &verified, PhoneNumber: "+84901234567",
				PasswordHash: "never-serialised", FullName: &fullName, DharmaName: &dharma, CreatedAt: createdAt,
			},
			AccessToken:  "access",
			RefreshToken: "refresh",
		}, nil)

	rec := doRequest(h.VerifyOTP, `{"email":"a@example.com","otp_code":"042917"}`)

	require.Equal(t, http.StatusCreated, rec.Code)
	var data map[string]any
	decodeSuccessData(t, rec, &data)
	require.ElementsMatch(t, []string{"access_token", "refresh_token", "expires_in", "user"}, keys(data))
	require.Equal(t, float64(900), data["expires_in"])
	user := data["user"].(map[string]any)
	require.ElementsMatch(t, []string{"id", "email", "phone_number", "full_name", "dharma_name", "created_at"}, keys(user),
		"user keeps the old shape; no password hash")
	require.Equal(t, "+84901234567", user["phone_number"], "E.164")
	require.NotContains(t, rec.Body.String(), "never-serialised")
}

func TestRegisterHandler_VerifyOTP_OTPCodeWrongShape(t *testing.T) {
	for _, code := range []string{`"12"`, `"abcdef"`, `""`} {
		h, uc := newTestHandler(t)
		rec := doRequest(h.VerifyOTP, `{"email":"a@example.com","otp_code":`+code+`}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		body := decodeErrorBody(t, rec)
		require.Equal(t, "VALIDATION_ERROR", body.Code)
		require.Contains(t, body.Details, "otp_code")
		uc.AssertNotCalled(t, "VerifyOTP", mock.Anything, mock.Anything)
	}
}

func TestRegisterHandler_VerifyOTP_Errors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"AC3 invalid email", domain.ErrInvalidEmail, http.StatusBadRequest, "INVALID_EMAIL"},
		{"AC15 wrong code", domain.ErrOTPInvalid, http.StatusBadRequest, "OTP_INVALID"},
		{"AC15 expired code", domain.ErrOTPExpired, http.StatusBadRequest, "OTP_EXPIRED"},
		{"AC17 pending expired", domain.ErrPendingRegistrationExpired, http.StatusBadRequest, "PENDING_REGISTRATION_EXPIRED"},
		{"AC18 phone taken", domain.ErrPhoneTaken, http.StatusConflict, "PHONE_TAKEN"},
		{"AC18 temple gone", domain.ErrInvalidTemple, http.StatusBadRequest, "INVALID_TEMPLE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			uc.On("VerifyOTP", mock.Anything, mock.Anything).Return(nil, tt.err)

			rec := doRequest(h.VerifyOTP, `{"email":"a@example.com","otp_code":"042917"}`)

			require.Equal(t, tt.wantStatus, rec.Code)
			require.Equal(t, tt.wantCode, decodeErrorBody(t, rec).Code)
		})
	}
}

func TestRegisterHandler_VerifyOTP_Locked(t *testing.T) {
	h, uc := newTestHandler(t)
	deadline := time.Now().Add(30 * time.Minute).UTC().Truncate(time.Second)
	uc.On("VerifyOTP", mock.Anything, mock.Anything).Return(nil, &domain.OTPLockedError{LockedUntil: deadline})

	rec := doRequest(h.VerifyOTP, `{"email":"a@example.com","otp_code":"042917"}`)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	body := decodeErrorBody(t, rec)
	require.Equal(t, "OTP_LOCKED", body.Code)
	got, err := time.Parse(time.RFC3339, body.Details["locked_until"].(string))
	require.NoError(t, err)
	require.WithinDuration(t, deadline, got, time.Second)
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
