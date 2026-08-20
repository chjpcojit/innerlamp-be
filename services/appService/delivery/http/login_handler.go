package http

import (
	"context"
	"errors"
	"net/http"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Device   struct {
		Platform    string `json:"platform"`
		OSVersion   string `json:"os_version"`
		AppVersion  string `json:"app_version"`
		DeviceModel string `json:"device_model"`
		DeviceName  string `json:"device_name"`
	} `json:"device"`
}

// loginUsecase is the subset of *usecase.LoginUsecase this handler depends
// on, defined here (the consumer) so the handler can be unit tested with a
// mock instead of the real use-case.
type loginUsecase interface {
	Login(ctx context.Context, input domain.LoginInput) (*domain.LoginResult, error)
}

// LoginHandler implements POST /v1/auth/login (US-002).
type LoginHandler struct {
	usecase loginUsecase
}

// NewLoginHandler wires the HTTP handler to its use-case.
func NewLoginHandler(uc loginUsecase) *LoginHandler {
	return &LoginHandler{usecase: uc}
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, utils.ErrorTypeValidation, "METHOD_NOT_ALLOWED", "method not allowed", nil)
		return
	}

	var req loginRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		field, message := utils.GetDecodeErrorDetails(err)
		var details map[string]any
		if field != "" {
			details = map[string]any{"field": field}
		}
		utils.WriteValidationError(w, "INVALID_REQUEST_BODY", message, details)
		return
	}

	input := domain.LoginInput{
		Email:    req.Email,
		Password: req.Password,
		Device: domain.DeviceInfo{
			Platform:    req.Device.Platform,
			OSVersion:   req.Device.OSVersion,
			AppVersion:  req.Device.AppVersion,
			DeviceModel: req.Device.DeviceModel,
			DeviceName:  req.Device.DeviceName,
			IPAddress:   clientIP(r),
			UserAgent:   r.UserAgent(),
		},
	}

	result, err := h.usecase.Login(r.Context(), input)
	if err != nil {
		writeLoginError(w, err)
		return
	}

	// Response shape mirrors register's, per the US-002 spec note.
	utils.WriteSuccessResponse(w, registerResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User: userResponse{
			ID:              result.User.ID,
			Email:           result.User.Email,
			PhoneNumber:     result.User.PhoneNumber,
			DharmaName:      result.User.DharmaName,
			OrdinationPlace: result.User.OrdinationPlace,
		},
	})
}

// writeLoginError maps domain errors to the HTTP status/message shape
// expected by AC2/AC3: wrong password and unknown email are indistinguishable.
func writeLoginError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		utils.WriteBusinessError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", domain.ErrInvalidCredentials.Error(), nil)
	default:
		utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
	}
}
