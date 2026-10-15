package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/pkg/validation"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// passwordResetUsecase is the subset of *usecase.PasswordResetUsecase this
// handler depends on, declared here (the consumer) so it is tested with a mock.
type passwordResetUsecase interface {
	RequestOTP(ctx context.Context, input domain.RequestPasswordResetInput) (*domain.RequestPasswordResetResult, error)
	Confirm(ctx context.Context, input domain.ConfirmPasswordResetInput) error
}

// PasswordResetHandler serves US-003's two public routes. cmd/main.go wraps
// both in NoStore (AC18).
type PasswordResetHandler struct {
	usecase   passwordResetUsecase
	validator *validation.Validator
}

func NewPasswordResetHandler(usecase passwordResetUsecase, validator *validation.Validator) *PasswordResetHandler {
	return &PasswordResetHandler{usecase: usecase, validator: validator}
}

type passwordResetRequest struct {
	Email string `json:"email"`
}

// passwordResetRequestResponse carries the same message on both branches
// (AC23/AC24); the copy must not say whether an email was sent.
type passwordResetRequestResponse struct {
	Message            string `json:"message"`
	ExpiresInSeconds   int    `json:"expires_in_seconds"`
	ResendAfterSeconds int    `json:"resend_after_seconds"`
}

// passwordResetConfirmRequest: email and password rules are the use case's
// (INVALID_EMAIL / INVALID_PASSWORD); the code's shape is a plain
// VALIDATION_ERROR here, like register verify-otp.
type passwordResetConfirmRequest struct {
	Email       string `json:"email"`
	OTPCode     string `json:"otp_code" validate:"required,len=6,numeric"`
	NewPassword string `json:"new_password"`
}

// RequestOTP serves POST /v1/auth/password-reset/request-otp.
func (h *PasswordResetHandler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req passwordResetRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	result, err := h.usecase.RequestOTP(r.Context(), domain.RequestPasswordResetInput{Email: req.Email})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	utils.WriteSuccessResponse(w, passwordResetRequestResponse{
		Message:            "Nếu email này có tài khoản, mã đã được gửi",
		ExpiresInSeconds:   result.ExpiresInSeconds,
		ResendAfterSeconds: result.ResendAfterSeconds,
	})
}

// Confirm serves POST /v1/auth/password-reset/confirm. Success is
// {"success": true} with no tokens: the user signs in again (AC28).
func (h *PasswordResetHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	var req passwordResetConfirmRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if err := h.validator.Struct(req); err != nil {
		if verrs, ok := errors.AsType[validator.ValidationErrors](err); ok {
			writeValidationError(w, verrs)
			return
		}
		utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
		return
	}

	err := h.usecase.Confirm(r.Context(), domain.ConfirmPasswordResetInput{
		Email:       req.Email,
		OTPCode:     req.OTPCode,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	utils.WriteSuccessResponse(w, nil)
}
