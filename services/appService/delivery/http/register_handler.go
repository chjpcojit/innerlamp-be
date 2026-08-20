package http

import (
	"context"
	"errors"
	"net/http"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type registerRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	PhoneNumber     string `json:"phone_number"`
	DharmaName      string `json:"dharma_name,omitempty"`
	OrdinationPlace string `json:"ordination_place,omitempty"`
	Device          struct {
		Platform    string `json:"platform"`
		OSVersion   string `json:"os_version"`
		AppVersion  string `json:"app_version"`
		DeviceModel string `json:"device_model"`
		DeviceName  string `json:"device_name"`
	} `json:"device"`
}

type userResponse struct {
	ID              int64  `json:"id"`
	Email           string `json:"email"`
	PhoneNumber     string `json:"phone_number"`
	DharmaName      string `json:"dharma_name,omitempty"`
	OrdinationPlace string `json:"ordination_place,omitempty"`
}

type registerResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         userResponse `json:"user"`
}

// registerUsecase is the subset of *usecase.RegisterUsecase this handler
// depends on, defined here (the consumer) so the handler can be unit
// tested with a mock instead of the real use-case.
type registerUsecase interface {
	Register(ctx context.Context, input domain.RegisterInput) (*domain.RegisterResult, error)
}

// RegisterHandler implements POST /v1/auth/register (US-001).
type RegisterHandler struct {
	usecase registerUsecase
}

// NewRegisterHandler wires the HTTP handler to its use-case.
func NewRegisterHandler(uc registerUsecase) *RegisterHandler {
	return &RegisterHandler{usecase: uc}
}

func (h *RegisterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, utils.ErrorTypeValidation, "METHOD_NOT_ALLOWED", "method not allowed", nil)
		return
	}

	var req registerRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		field, message := utils.GetDecodeErrorDetails(err)
		var details map[string]any
		if field != "" {
			details = map[string]any{"field": field}
		}
		utils.WriteValidationError(w, "INVALID_REQUEST_BODY", message, details)
		return
	}

	input := domain.RegisterInput{
		Email:           req.Email,
		Password:        req.Password,
		PhoneNumber:     req.PhoneNumber,
		DharmaName:      req.DharmaName,
		OrdinationPlace: req.OrdinationPlace,
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

	result, err := h.usecase.Register(r.Context(), input)
	if err != nil {
		writeRegisterError(w, err)
		return
	}

	utils.WriteSuccessResponseWithStatus(w, http.StatusCreated, registerResponse{
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

// writeRegisterError maps domain errors to the HTTP status/message shape
// expected by AC2/AC3/AC4.
func writeRegisterError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrEmailTaken):
		utils.WriteBusinessError(w, http.StatusConflict, "EMAIL_TAKEN", "Email đã được sử dụng", nil)
	case errors.Is(err, domain.ErrPhoneTaken):
		utils.WriteBusinessError(w, http.StatusConflict, "PHONE_TAKEN", "Số điện thoại đã được sử dụng", nil)
	case errors.Is(err, domain.ErrInvalidEmail):
		utils.WriteValidationError(w, "INVALID_EMAIL", err.Error(), map[string]any{"field": "email"})
	case errors.Is(err, domain.ErrInvalidPhone):
		utils.WriteValidationError(w, "INVALID_PHONE", err.Error(), map[string]any{"field": "phone_number"})
	case errors.Is(err, domain.ErrInvalidPassword):
		utils.WriteValidationError(w, "INVALID_PASSWORD", err.Error(), map[string]any{"field": "password"})
	default:
		utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
	}
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	return r.RemoteAddr
}
