package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/pkg/validation"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// registerUsecase is the subset of *usecase.RegisterUsecase this handler
// depends on, declared here (the consumer) so RegisterHandler can be unit
// tested with a mock instead of the real use case and its collaborators
// (repository, JWT service, SMS sender).
type registerUsecase interface {
	RequestOTP(ctx context.Context, input domain.RequestRegisterOTPInput) (*domain.RequestOTPResult, error)
	ResendOTP(ctx context.Context, input domain.ResendOTPInput) (*domain.ResendOTPResult, error)
	VerifyOTP(ctx context.Context, input domain.VerifyRegisterOTPInput) (*domain.RegisterResult, error)
}

// RegisterHandler serves US-001's three routes. All three are public — the
// caller has no account yet, so there is nothing to authenticate — and none
// are wrapped with auth middleware in cmd/main.go.
type RegisterHandler struct {
	usecase   registerUsecase
	validator *validation.Validator
	tokens    tokenSettings
}

// NewRegisterHandler wires the handler. validator must already have the
// `phone` tag registered (validation.PhoneTag/PhoneRule) and its field-name
// tag set to "json" — both are composition-root concerns done once, before
// any handler is built, per design.md's cmd/main.go row.
func NewRegisterHandler(usecase registerUsecase, validator *validation.Validator, opts ...HandlerOption) *RegisterHandler {
	return &RegisterHandler{usecase: usecase, validator: validator, tokens: newTokenSettings(opts)}
}

// requestOTPRequest is step 1's body. Password has no `min` tag here on
// purpose: the use case owns the length floor (AC7) as a business rule on
// the *normalized* password, and design.md's Step 1 validates format first,
// leaving the length check to RequestOTP itself — duplicating it here would
// just risk the two disagreeing.
type requestOTPRequest struct {
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required"`
	PhoneNumber     string `json:"phone_number" validate:"required,phone"`
	DharmaName      string `json:"dharma_name,omitempty"`
	OrdinationPlace string `json:"ordination_place,omitempty"`
}

type requestOTPResponse struct {
	Message            string `json:"message"`
	ExpiresInSeconds   int    `json:"expires_in_seconds"`
	ResendAfterSeconds int    `json:"resend_after_seconds"`
}

// resendOTPRequest carries only phone_number and purpose — the account
// payload already lives in pending_registration from step 1, per design.md
// flow 3 Step 1. oneof lists all four OTPPurpose values even though US-001
// only implements register; the other three are rejected by the use case
// (ErrUnsupportedOTPPurpose), not by this tag, so US-002/US-003 can start
// using them without a delivery change.
type resendOTPRequest struct {
	PhoneNumber string `json:"phone_number" validate:"required,phone"`
	Purpose     string `json:"purpose" validate:"required,oneof=register login reset_password verify_email"`
}

type resendOTPResponse struct {
	Message            string `json:"message"`
	ExpiresInSeconds   int    `json:"expires_in_seconds"`
	ResendAfterSeconds int    `json:"resend_after_seconds"`
	ResendRemaining    int    `json:"resend_remaining"`
}

type verifyOTPRequest struct {
	PhoneNumber string `json:"phone_number" validate:"required,phone"`
	OTPCode     string `json:"otp_code" validate:"required,len=6,numeric"`
}

type verifyOTPResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"`
	User         userResponse `json:"user"`
}

// userResponse is this handler's own wire shape for a registered account —
// never domain.User directly (see layer_conventions.md's "Entity vs response
// model" rule). A different handler (login, profile) is free to name or omit
// fields differently; each answers to its own contract.
type userResponse struct {
	ID              int64     `json:"id"`
	Email           string    `json:"email"`
	PhoneNumber     string    `json:"phone_number"`
	DharmaName      *string   `json:"dharma_name,omitempty"`
	OrdinationPlace *string   `json:"ordination_place,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

func newUserResponse(u *domain.User) userResponse {
	return userResponse{
		ID:              u.ID,
		Email:           u.Email,
		PhoneNumber:     u.PhoneNumber,
		DharmaName:      u.DharmaName,
		OrdinationPlace: u.OrdinationPlace,
		CreatedAt:       u.CreatedAt,
	}
}

// RequestOTP serves POST /v1/auth/register/request-otp — design.md's flow 1.
func (h *RegisterHandler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req requestOTPRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if err := h.validateStruct(w, req); err != nil {
		return
	}

	result, err := h.usecase.RequestOTP(r.Context(), domain.RequestRegisterOTPInput{
		Email:           req.Email,
		Password:        req.Password,
		PhoneNumber:     req.PhoneNumber,
		DharmaName:      req.DharmaName,
		OrdinationPlace: req.OrdinationPlace,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	utils.WriteSuccessResponse(w, requestOTPResponse{
		Message:            "otp sent",
		ExpiresInSeconds:   result.ExpiresInSeconds,
		ResendAfterSeconds: result.ResendAfterSeconds,
	})
}

// ResendOTP serves POST /v1/auth/resend-otp — design.md's flow 3. Shared
// across every OTP purpose; US-001 only implements register.
func (h *RegisterHandler) ResendOTP(w http.ResponseWriter, r *http.Request) {
	var req resendOTPRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if err := h.validateStruct(w, req); err != nil {
		return
	}

	result, err := h.usecase.ResendOTP(r.Context(), domain.ResendOTPInput{
		PhoneNumber: req.PhoneNumber,
		Purpose:     domain.OTPPurpose(req.Purpose),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	utils.WriteSuccessResponse(w, resendOTPResponse{
		Message:            "otp sent",
		ExpiresInSeconds:   result.ExpiresInSeconds,
		ResendAfterSeconds: result.ResendAfterSeconds,
		ResendRemaining:    result.ResendRemaining,
	})
}

// VerifyOTP serves POST /v1/auth/register/verify-otp — design.md's flow 2.
// Success creates the account already logged in (AC1/AC7), so the response
// carries the session's tokens.
func (h *RegisterHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req verifyOTPRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if err := h.validateStruct(w, req); err != nil {
		return
	}

	result, err := h.usecase.VerifyOTP(r.Context(), domain.VerifyRegisterOTPInput{
		PhoneNumber: req.PhoneNumber,
		OTPCode:     req.OTPCode,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	utils.WriteSuccessResponseWithStatus(w, http.StatusCreated, verifyOTPResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    h.tokens.expiresIn(),
		User:         newUserResponse(result.User),
	})
}

// validateStruct runs req through the shared validator and, on failure,
// writes the response itself and returns a non-nil error so the caller can
// bail with `if err := h.validateStruct(w, req); err != nil { return }`
// instead of repeating the type-switch in every handler method.
func (h *RegisterHandler) validateStruct(w http.ResponseWriter, req any) error {
	err := h.validator.Struct(req)
	if err == nil {
		return nil
	}

	if verrs, ok := errors.AsType[validator.ValidationErrors](err); ok {
		writeValidationError(w, verrs)
		return err
	}
	utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
	return err
}
