package http

import (
	"bytes"
	"encoding/json"
	"errors"
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

func newTestHandler(t *testing.T) (*RegisterHandler, *mockRegisterUsecase) {
	t.Helper()
	uc := &mockRegisterUsecase{}
	return NewRegisterHandler(uc, newTestValidator(t)), uc
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

func TestRegisterHandler_RequestOTP_Success(t *testing.T) {
	h, uc := newTestHandler(t)
	uc.On("RequestOTP", mock.Anything, domain.RequestRegisterOTPInput{
		Email:       "a@example.com",
		Password:    "password123",
		FullName:    "Nguyen Van A",
		PhoneNumber: "0901234567",
	}).Return(&domain.RequestOTPResult{ExpiresInSeconds: 90, ResendAfterSeconds: 60}, nil)

	rec := doRequest(h.RequestOTP, `{"email":"a@example.com","password":"password123","full_name":"Nguyen Van A","phone_number":"0901234567"}`)

	require.Equal(t, http.StatusOK, rec.Code)
	var body requestOTPResponse
	decodeSuccessData(t, rec, &body)
	require.Equal(t, 90, body.ExpiresInSeconds)
	require.Equal(t, 60, body.ResendAfterSeconds)
}

// TestRegisterHandler_RequestOTP_TempleID covers AC4/AC18 at the wire: the
// id travels from ordination_temple_id into the use case input.
func TestRegisterHandler_RequestOTP_TempleID(t *testing.T) {
	h, uc := newTestHandler(t)
	uc.On("RequestOTP", mock.Anything, domain.RequestRegisterOTPInput{
		Email:              "a@example.com",
		Password:           "password123",
		FullName:           "Nguyen Van A",
		PhoneNumber:        "0901234567",
		OrdinationTempleID: "temple_north_011",
	}).Return(&domain.RequestOTPResult{ExpiresInSeconds: 90, ResendAfterSeconds: 60}, nil)

	rec := doRequest(h.RequestOTP, `{"email":"a@example.com","password":"password123","full_name":"Nguyen Van A","phone_number":"0901234567","ordination_temple_id":"temple_north_011"}`)

	require.Equal(t, http.StatusOK, rec.Code)
	uc.AssertExpectations(t)
}

// TestRegisterHandler_RequestOTP_LegacyOrdinationPlaceRejected: the old text
// field is gone (v7.0) and strict decoding rejects it like any unknown field.
func TestRegisterHandler_RequestOTP_LegacyOrdinationPlaceRejected(t *testing.T) {
	h, uc := newTestHandler(t)
	rec := doRequest(h.RequestOTP, `{"email":"a@example.com","password":"password123","full_name":"Nguyen Van A","phone_number":"0901234567","ordination_place":"Chùa Hoằng Pháp"}`)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "VALIDATION_ERROR", decodeErrorBody(t, rec).Code)
	uc.AssertNotCalled(t, "RequestOTP", mock.Anything, mock.Anything)
}

func TestRegisterHandler_RequestOTP_InvalidTempleDetails(t *testing.T) {
	h, uc := newTestHandler(t)
	uc.On("RequestOTP", mock.Anything, mock.Anything).Return(nil, domain.ErrInvalidTemple)

	rec := doRequest(h.RequestOTP, `{"email":"a@example.com","password":"password123","full_name":"Nguyen Van A","phone_number":"0901234567","ordination_temple_id":"nope"}`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	body := decodeErrorBody(t, rec)
	require.Equal(t, "INVALID_TEMPLE", body.Code)
	require.Equal(t, "ordination_temple_id", body.Details["field"])
}

func TestRegisterHandler_RequestOTP_InvalidDharmaNameDetails(t *testing.T) {
	h, uc := newTestHandler(t)
	uc.On("RequestOTP", mock.Anything, mock.Anything).Return(nil, domain.ErrInvalidDharmaName)

	rec := doRequest(h.RequestOTP, `{"email":"a@example.com","password":"password123","full_name":"Nguyen Van A","phone_number":"0901234567","dharma_name":"too long"}`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	body := decodeErrorBody(t, rec)
	require.Equal(t, "INVALID_DHARMA_NAME", body.Code)
	require.Equal(t, "dharma_name", body.Details["field"])
}

func TestRegisterHandler_RequestOTP_MalformedJSON(t *testing.T) {
	h, uc := newTestHandler(t)
	rec := doRequest(h.RequestOTP, `{not json`)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "VALIDATION_ERROR", decodeErrorBody(t, rec).Code)
	uc.AssertNotCalled(t, "RequestOTP", mock.Anything, mock.Anything)
}

func TestRegisterHandler_RequestOTP_UnknownField(t *testing.T) {
	h, uc := newTestHandler(t)
	rec := doRequest(h.RequestOTP, `{"email":"a@example.com","password":"password123","full_name":"Nguyen Van A","phone_number":"0901234567","surprise":true}`)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "VALIDATION_ERROR", decodeErrorBody(t, rec).Code)
	uc.AssertNotCalled(t, "RequestOTP", mock.Anything, mock.Anything)
}

func TestRegisterHandler_RequestOTP_MissingRequiredField(t *testing.T) {
	h, uc := newTestHandler(t)
	rec := doRequest(h.RequestOTP, `{"password":"password123","full_name":"Nguyen Van A","phone_number":"0901234567"}`)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	body := decodeErrorBody(t, rec)
	require.Equal(t, "VALIDATION_ERROR", body.Code)
	require.Contains(t, body.Details, "email")
	uc.AssertNotCalled(t, "RequestOTP", mock.Anything, mock.Anything)
}

func TestRegisterHandler_RequestOTP_MalformedPhoneCaughtByValidator(t *testing.T) {
	h, uc := newTestHandler(t)
	rec := doRequest(h.RequestOTP, `{"email":"a@example.com","password":"password123","full_name":"Nguyen Van A","phone_number":"not-a-phone"}`)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	body := decodeErrorBody(t, rec)
	require.Equal(t, "VALIDATION_ERROR", body.Code)
	require.Contains(t, body.Details, "phone_number")
	uc.AssertNotCalled(t, "RequestOTP", mock.Anything, mock.Anything)
}

func TestRegisterHandler_RequestOTP_DomainErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"phone taken", domain.ErrPhoneTaken, http.StatusConflict, "PHONE_TAKEN"},
		{"email taken", domain.ErrEmailTaken, http.StatusConflict, "EMAIL_TAKEN"},
		{"invalid password (business rule, not caught by validator)", domain.ErrInvalidPassword, http.StatusBadRequest, "INVALID_PASSWORD"},
		{"sms send failed", domain.ErrSMSSendFailed, http.StatusInternalServerError, "SMS_SEND_FAILED"},
		{"otp locked", &domain.OTPLockedError{LockedUntil: time.Now().Add(30 * time.Minute)}, http.StatusTooManyRequests, "OTP_LOCKED"},
		{"otp send limited", &domain.OTPSendLimitedError{LockedUntil: time.Now().Add(30 * time.Minute)}, http.StatusTooManyRequests, "OTP_SEND_LIMITED"},
		{"unmapped infra error", errUnmapped, http.StatusInternalServerError, "INTERNAL_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			uc.On("RequestOTP", mock.Anything, mock.Anything).Return(nil, tt.err)

			rec := doRequest(h.RequestOTP, `{"email":"a@example.com","password":"password123","full_name":"Nguyen Van A","phone_number":"0901234567"}`)

			require.Equal(t, tt.wantStatus, rec.Code)
			require.Equal(t, tt.wantCode, decodeErrorBody(t, rec).Code)
		})
	}
}

func TestRegisterHandler_RequestOTP_OTPLockedIncludesDeadline(t *testing.T) {
	h, uc := newTestHandler(t)
	deadline := time.Now().Add(15 * time.Minute).UTC().Truncate(time.Second)
	uc.On("RequestOTP", mock.Anything, mock.Anything).Return(nil, &domain.OTPLockedError{LockedUntil: deadline})

	rec := doRequest(h.RequestOTP, `{"email":"a@example.com","password":"password123","full_name":"Nguyen Van A","phone_number":"0901234567"}`)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	body := decodeErrorBody(t, rec)
	require.Equal(t, "OTP_LOCKED", body.Code)
	require.Contains(t, body.Details, "locked_until")
}

// --- ResendOTP ----------------------------------------------------------

func TestRegisterHandler_ResendOTP_Success(t *testing.T) {
	h, uc := newTestHandler(t)
	uc.On("ResendOTP", mock.Anything, domain.ResendOTPInput{
		PhoneNumber: "0901234567",
		Purpose:     domain.OTPPurposeRegister,
	}).Return(&domain.ResendOTPResult{ExpiresInSeconds: 90, ResendAfterSeconds: 60, ResendRemaining: 2}, nil)

	rec := doRequest(h.ResendOTP, `{"phone_number":"0901234567","purpose":"register"}`)

	require.Equal(t, http.StatusOK, rec.Code)
	var body resendOTPResponse
	decodeSuccessData(t, rec, &body)
	require.Equal(t, 2, body.ResendRemaining)
}

func TestRegisterHandler_ResendOTP_PurposeOutsideOneof(t *testing.T) {
	h, uc := newTestHandler(t)
	rec := doRequest(h.ResendOTP, `{"phone_number":"0901234567","purpose":"not-a-real-purpose"}`)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	body := decodeErrorBody(t, rec)
	require.Equal(t, "VALIDATION_ERROR", body.Code)
	require.Contains(t, body.Details, "purpose")
	uc.AssertNotCalled(t, "ResendOTP", mock.Anything, mock.Anything)
}

func TestRegisterHandler_ResendOTP_UnsupportedPurposeFromUsecase(t *testing.T) {
	h, uc := newTestHandler(t)
	uc.On("ResendOTP", mock.Anything, mock.Anything).Return(nil, domain.ErrUnsupportedOTPPurpose)

	rec := doRequest(h.ResendOTP, `{"phone_number":"0901234567","purpose":"login"}`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "UNSUPPORTED_OTP_PURPOSE", decodeErrorBody(t, rec).Code)
}

func TestRegisterHandler_ResendOTP_TooSoonIncludesRetryAfter(t *testing.T) {
	h, uc := newTestHandler(t)
	uc.On("ResendOTP", mock.Anything, mock.Anything).Return(nil, &domain.OTPResendTooSoonError{RetryAfterSeconds: 42})

	rec := doRequest(h.ResendOTP, `{"phone_number":"0901234567","purpose":"register"}`)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	body := decodeErrorBody(t, rec)
	require.Equal(t, "OTP_RESEND_TOO_SOON", body.Code)
	require.EqualValues(t, 42, body.Details["retry_after_seconds"])
}

// TestRegisterHandler_ResendOTP_SendLimited pins the AC5 cap response: unlike
// the old permanent OTP_RESEND_EXHAUSTED, the send cap is a temporary lock, so
// the 429 carries locked_until for the client to count down.
func TestRegisterHandler_ResendOTP_SendLimited(t *testing.T) {
	h, uc := newTestHandler(t)
	deadline := time.Now().Add(30 * time.Minute)
	uc.On("ResendOTP", mock.Anything, mock.Anything).Return(nil, &domain.OTPSendLimitedError{LockedUntil: deadline})

	rec := doRequest(h.ResendOTP, `{"phone_number":"0901234567","purpose":"register"}`)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	body := decodeErrorBody(t, rec)
	require.Equal(t, "OTP_SEND_LIMITED", body.Code)

	raw, ok := body.Details["locked_until"].(string)
	require.True(t, ok, "OTP_SEND_LIMITED must carry locked_until: %v", body.Details)
	got, err := time.Parse(time.RFC3339Nano, raw)
	require.NoError(t, err)
	require.WithinDuration(t, deadline, got, time.Second)
}

// --- VerifyOTP ------------------------------------------------------------

func TestRegisterHandler_VerifyOTP_Success(t *testing.T) {
	h, uc := newTestHandler(t)
	createdAt := time.Now()
	dharma := "Thich Test"
	uc.On("VerifyOTP", mock.Anything, domain.VerifyRegisterOTPInput{
		PhoneNumber: "0901234567",
		OTPCode:     "111111",
	}).Return(&domain.RegisterResult{
		User: &domain.User{
			ID:          1,
			Email:       "a@example.com",
			PhoneNumber: "+84901234567",
			DharmaName:  &dharma,
			CreatedAt:   createdAt,
		},
		AccessToken:  "access",
		RefreshToken: "refresh",
	}, nil)

	rec := doRequest(h.VerifyOTP, `{"phone_number":"0901234567","otp_code":"111111"}`)

	require.Equal(t, http.StatusCreated, rec.Code)
	var body verifyOTPResponse
	decodeSuccessData(t, rec, &body)
	require.Equal(t, "access", body.AccessToken)
	require.Equal(t, "refresh", body.RefreshToken)
	require.Equal(t, int64(1), body.User.ID)
	require.Equal(t, "a@example.com", body.User.Email)
	require.Equal(t, "Thich Test", *body.User.DharmaName)
}

func TestRegisterHandler_VerifyOTP_OTPCodeWrongShape(t *testing.T) {
	h, uc := newTestHandler(t)
	rec := doRequest(h.VerifyOTP, `{"phone_number":"0901234567","otp_code":"12"}`)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	body := decodeErrorBody(t, rec)
	require.Equal(t, "VALIDATION_ERROR", body.Code)
	require.Contains(t, body.Details, "otp_code")
	uc.AssertNotCalled(t, "VerifyOTP", mock.Anything, mock.Anything)
}

func TestRegisterHandler_VerifyOTP_PendingRegistrationNotFoundMapsToOTPInvalidNot404(t *testing.T) {
	h, uc := newTestHandler(t)
	uc.On("VerifyOTP", mock.Anything, mock.Anything).Return(nil, domain.ErrPendingRegistrationNotFound)

	rec := doRequest(h.VerifyOTP, `{"phone_number":"0901234567","otp_code":"111111"}`)

	require.Equal(t, http.StatusBadRequest, rec.Code, "must not be 404 — design.md flow 2 Step 2: a 404 would confirm which phone numbers are mid-registration")
	require.Equal(t, "OTP_INVALID", decodeErrorBody(t, rec).Code)
}

func TestRegisterHandler_VerifyOTP_Expired(t *testing.T) {
	h, uc := newTestHandler(t)
	uc.On("VerifyOTP", mock.Anything, mock.Anything).Return(nil, domain.ErrOTPExpired)

	rec := doRequest(h.VerifyOTP, `{"phone_number":"0901234567","otp_code":"111111"}`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "OTP_EXPIRED", decodeErrorBody(t, rec).Code)
}

func TestRegisterHandler_VerifyOTP_Locked(t *testing.T) {
	h, uc := newTestHandler(t)
	uc.On("VerifyOTP", mock.Anything, mock.Anything).Return(nil, &domain.OTPLockedError{LockedUntil: time.Now().Add(15 * time.Minute)})

	rec := doRequest(h.VerifyOTP, `{"phone_number":"0901234567","otp_code":"111111"}`)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, "OTP_LOCKED", decodeErrorBody(t, rec).Code)
}

// TestRegisterHandler_RequestOTP_BodyTooLarge is AC7: an oversized body is
// the one decode failure that is not the client's syntax but its size, so it
// answers 413 PAYLOAD_TOO_LARGE instead of 400 VALIDATION_ERROR. Every other
// decode failure keeps the 400 mapping the tests above pin.
func TestRegisterHandler_RequestOTP_BodyTooLarge(t *testing.T) {
	h, uc := newTestHandler(t)

	// Over DecodeJSONBodyStrict's 1 MB cap; valid JSON otherwise, so only the
	// size gate can reject it.
	oversized := `{"email":"a@example.com","password":"` + strings.Repeat("x", 1<<20) + `","phone_number":"0901234567"}`
	rec := doRequest(h.RequestOTP, oversized)

	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	body := decodeErrorBody(t, rec)
	require.Equal(t, "PAYLOAD_TOO_LARGE", body.Code)
	require.Equal(t, utils.ErrBodyTooLarge.Error(), body.Message)
	uc.AssertNotCalled(t, "RequestOTP", mock.Anything, mock.Anything)
}

// full_name is required: a body without it is rejected by the validator before
// the use case runs.
func TestRegisterHandler_RequestOTP_MissingFullName(t *testing.T) {
	h, uc := newTestHandler(t)

	rec := doRequest(h.RequestOTP, `{"email":"a@example.com","password":"password123","phone_number":"0901234567"}`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	uc.AssertNotCalled(t, "RequestOTP", mock.Anything, mock.Anything)
}
