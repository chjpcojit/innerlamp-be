package http

import (
	"context"
	"net/http"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// forgotPasswordMessage is the generic success message — AC1/AC2: an
// existing and a non-existing email must be indistinguishable to the
// caller.
const forgotPasswordMessage = "Nếu email tồn tại trong hệ thống, chúng tôi đã gửi hướng dẫn đặt lại mật khẩu"

// forgotPasswordUsecase is the subset of *usecase.PasswordResetUsecase
// this handler depends on, defined here (the consumer) so the handler can
// be unit tested with a mock instead of the real use-case.
type forgotPasswordUsecase interface {
	ForgotPassword(ctx context.Context, input domain.ForgotPasswordInput) error
}

// ForgotPasswordHandler implements POST /v1/auth/forgot-password (US-003).
type ForgotPasswordHandler struct {
	usecase forgotPasswordUsecase
}

// NewForgotPasswordHandler wires the HTTP handler to its use-case.
func NewForgotPasswordHandler(uc forgotPasswordUsecase) *ForgotPasswordHandler {
	return &ForgotPasswordHandler{usecase: uc}
}

func (h *ForgotPasswordHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, utils.ErrorTypeValidation, "METHOD_NOT_ALLOWED", "method not allowed", nil)
		return
	}

	var req forgotPasswordRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		field, message := utils.GetDecodeErrorDetails(err)
		var details map[string]any
		if field != "" {
			details = map[string]any{"field": field}
		}
		utils.WriteValidationError(w, "INVALID_REQUEST_BODY", message, details)
		return
	}

	// The use-case itself never distinguishes "email exists" from "email
	// doesn't" (AC2) — any error reaching here is a genuine infra failure.
	if err := h.usecase.ForgotPassword(r.Context(), domain.ForgotPasswordInput{Email: req.Email}); err != nil {
		utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
		return
	}

	utils.WriteSuccessResponse(w, map[string]string{"message": forgotPasswordMessage})
}
