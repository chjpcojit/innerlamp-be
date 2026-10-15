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
// (repository, JWT service, mailer).
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

// NewRegisterHandler wires the handler. validator must have its field-name tag
// set to "json" — a composition-root concern done once, before any handler is
// built.
func NewRegisterHandler(usecase registerUsecase, validator *validation.Validator, opts ...HandlerOption) *RegisterHandler {
	return &RegisterHandler{usecase: usecase, validator: validator, tokens: newTokenSettings(opts)}
}

// requestOTPRequest is step 1's body (onboarding requirements.md API table).
// No validate tags on purpose: every field has its own business rule and error
// code in the use case (AC3–AC6: INVALID_EMAIL, INVALID_PHONE,
// INVALID_PASSWORD, INVALID_FULL_NAME, INVALID_DHARMA_NAME), so an empty field
// gets that field's code rather than a generic VALIDATION_ERROR. Structural
// problems (bad JSON, unknown field) are still VALIDATION_ERROR from the
// strict decoder.
type requestOTPRequest struct {
	Email              string `json:"email"`
	Password           string `json:"password"`
	PhoneCountry       string `json:"phone_country"`
	PhoneNumber        string `json:"phone_number"`
	FullName           string `json:"full_name"`
	DharmaName         string `json:"dharma_name,omitempty"`
	OrdinationTempleID string `json:"ordination_temple_id,omitempty"`
}

type requestOTPResponse struct {
	Message            string `json:"message"`
	ExpiresInSeconds   int    `json:"expires_in_seconds"`
	ResendAfterSeconds int    `json:"resend_after_seconds"`
}

// resendOTPRequest carries only the email — the payload already lives in
// pending_registration from step 1 (design Step S-2). Its format is checked
// by the use case (INVALID_EMAIL).
type resendOTPRequest struct {
	Email string `json:"email"`
}

type resendOTPResponse struct {
	Message            string `json:"message"`
	ExpiresInSeconds   int    `json:"expires_in_seconds"`
	ResendAfterSeconds int    `json:"resend_after_seconds"`
	ResendRemaining    int    `json:"resend_remaining"`
}

// verifyOTPRequest is step 2's body (design Step V-2): the email's format is
// the use case's INVALID_EMAIL; the code's shape (6 digits) is a plain
// VALIDATION_ERROR here.
type verifyOTPRequest struct {
	Email   string `json:"email"`
	OTPCode string `json:"otp_code" validate:"required,len=6,numeric"`
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
	ID                 int64     `json:"id"`
	Email              string    `json:"email"`
	PhoneNumber        string    `json:"phone_number"`
	FullName           *string   `json:"full_name,omitempty"`
	DharmaName         *string   `json:"dharma_name,omitempty"`
	OrdinationTempleID *string   `json:"ordination_temple_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

func newUserResponse(u *domain.User) userResponse {
	return userResponse{
		ID:                 u.ID,
		Email:              u.Email,
		PhoneNumber:        u.PhoneNumber,
		FullName:           u.FullName,
		DharmaName:         u.DharmaName,
		OrdinationTempleID: u.OrdinationTempleID,
		CreatedAt:          u.CreatedAt,
	}
}

// RequestOTP serves POST /v1/auth/register/request-otp — design Steps R-2…R-8.
// The body is identical whether or not the email already has an account (AC2).
func (h *RegisterHandler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req requestOTPRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	result, err := h.usecase.RequestOTP(r.Context(), domain.RequestRegisterOTPInput{
		Email:              req.Email,
		Password:           req.Password,
		PhoneCountry:       req.PhoneCountry,
		PhoneNumber:        req.PhoneNumber,
		FullName:           req.FullName,
		DharmaName:         req.DharmaName,
		OrdinationTempleID: req.OrdinationTempleID,
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

// ResendOTP serves POST /v1/auth/register/resend-otp — design Steps S-2…S-5.
func (h *RegisterHandler) ResendOTP(w http.ResponseWriter, r *http.Request) {
	var req resendOTPRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	result, err := h.usecase.ResendOTP(r.Context(), domain.ResendOTPInput{Email: req.Email})
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

// VerifyOTP serves POST /v1/auth/register/verify-otp — design Steps V-2…V-5.
// Success creates the account already logged in (AC14), so the response
// carries the session's tokens; cmd/main.go wraps the route with
// Cache-Control: no-store.
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
		Email:   req.Email,
		OTPCode: req.OTPCode,
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
