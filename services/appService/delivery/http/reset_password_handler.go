package http

import (
	"context"
	"errors"
	"net/http"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

const resetPasswordMessage = "Mật khẩu đã được cập nhật, vui lòng đăng nhập lại"

// resetPasswordUsecase is the subset of *usecase.PasswordResetUsecase this
// handler depends on, defined here (the consumer) so the handler can be
// unit tested with a mock instead of the real use-case.
type resetPasswordUsecase interface {
	Reset(ctx context.Context, input domain.ResetPasswordInput) error
}

// ResetPasswordHandler implements POST /v1/auth/reset-password (US-003).
type ResetPasswordHandler struct {
	usecase resetPasswordUsecase
}

// NewResetPasswordHandler wires the HTTP handler to its use-case.
func NewResetPasswordHandler(uc resetPasswordUsecase) *ResetPasswordHandler {
	return &ResetPasswordHandler{usecase: uc}
}

func (h *ResetPasswordHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, utils.ErrorTypeValidation, "METHOD_NOT_ALLOWED", "method not allowed", nil)
		return
	}

	var req resetPasswordRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		field, message := utils.GetDecodeErrorDetails(err)
		var details map[string]any
		if field != "" {
			details = map[string]any{"field": field}
		}
		utils.WriteValidationError(w, "INVALID_REQUEST_BODY", message, details)
		return
	}

	input := domain.ResetPasswordInput{Token: req.Token, NewPassword: req.NewPassword}
	if err := h.usecase.Reset(r.Context(), input); err != nil {
		writeResetPasswordError(w, err)
		return
	}

	utils.WriteSuccessResponse(w, map[string]string{"message": resetPasswordMessage})
}

// writeResetPasswordError maps domain errors to the HTTP status/message
// shape expected by AC3/AC4.
func writeResetPasswordError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrResetTokenInvalid):
		utils.WriteBusinessError(w, http.StatusBadRequest, "RESET_TOKEN_INVALID", domain.ErrResetTokenInvalid.Error(), nil)
	case errors.Is(err, domain.ErrInvalidPassword):
		utils.WriteValidationError(w, "INVALID_PASSWORD", err.Error(), map[string]any{"field": "new_password"})
	default:
		utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
	}
}
