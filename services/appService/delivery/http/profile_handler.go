package http

import (
	"context"
	"net/http"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// profileResponse mirrors domain.User's profile-facing fields without
// omitempty — US-004 AC1 requires unset fields to render as an explicit
// empty value, never a missing key or "null".
type profileResponse struct {
	ID              int64  `json:"id"`
	Email           string `json:"email"`
	PhoneNumber     string `json:"phone"`
	DharmaName      string `json:"dharma_name"`
	OrdinationPlace string `json:"ordination_place"`
	AvatarURL       string `json:"avatar_url"`
}

func newProfileResponse(u *domain.User) profileResponse {
	return profileResponse{
		ID:              u.ID,
		Email:           u.Email,
		PhoneNumber:     u.PhoneNumber,
		DharmaName:      u.DharmaName,
		OrdinationPlace: u.OrdinationPlace,
		AvatarURL:       u.AvatarURL,
	}
}

type profileUpdateRequest struct {
	DharmaName      string `json:"dharma_name"`
	OrdinationPlace string `json:"ordination_place"`
}

// profileUsecase is the subset of *usecase.ProfileUsecase this handler
// depends on, defined here (the consumer) so it can be unit tested with a
// mock instead of the real use-case.
type profileUsecase interface {
	GetProfile(ctx context.Context, userID int64) (*domain.User, error)
	UpdateProfile(ctx context.Context, userID int64, input domain.ProfileUpdateInput) (*domain.User, error)
}

// ProfileHandler implements GET/PATCH /v1/users/me (US-004).
type ProfileHandler struct {
	usecase profileUsecase
}

// NewProfileHandler wires the HTTP handler to its use-case.
func NewProfileHandler(uc profileUsecase) *ProfileHandler {
	return &ProfileHandler{usecase: uc}
}

func (h *ProfileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(r)
	if !ok {
		writeUnauthorized(w)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, userID)
	case http.MethodPatch:
		h.handlePatch(w, r, userID)
	default:
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, utils.ErrorTypeValidation, "METHOD_NOT_ALLOWED", "method not allowed", nil)
	}
}

func (h *ProfileHandler) handleGet(w http.ResponseWriter, r *http.Request, userID int64) {
	user, err := h.usecase.GetProfile(r.Context(), userID)
	if err != nil {
		utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
		return
	}
	utils.WriteSuccessResponse(w, newProfileResponse(user))
}

func (h *ProfileHandler) handlePatch(w http.ResponseWriter, r *http.Request, userID int64) {
	var req profileUpdateRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		field, message := utils.GetDecodeErrorDetails(err)
		var details map[string]any
		if field != "" {
			details = map[string]any{"field": field}
		}
		utils.WriteValidationError(w, "INVALID_REQUEST_BODY", message, details)
		return
	}

	input := domain.ProfileUpdateInput{
		DharmaName:      req.DharmaName,
		OrdinationPlace: req.OrdinationPlace,
	}

	user, err := h.usecase.UpdateProfile(r.Context(), userID, input)
	if err != nil {
		utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
		return
	}
	utils.WriteSuccessResponse(w, newProfileResponse(user))
}

// writeUnauthorized writes the US-004 AC6 response for a missing or
// invalid authenticated user.
func writeUnauthorized(w http.ResponseWriter) {
	utils.WriteBusinessError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", nil)
}
