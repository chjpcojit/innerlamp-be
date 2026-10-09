package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

// adminUserUsecase is the subset of *usecase.AdminUserUsecase the handler needs,
// declared here (the consumer) so the handler can be tested with a mock.
type adminUserUsecase interface {
	CreateAccount(ctx context.Context, input domain.CreateAccountInput) (*domain.CreateAccountResult, error)
	ResetPassword(ctx context.Context, actorID, targetID int64) (*domain.CreateAccountResult, error)
}

// AdminUserHandler serves backoffice account management.
type AdminUserHandler struct {
	usecase adminUserUsecase
}

// NewAdminUserHandler wires the handler.
func NewAdminUserHandler(usecase adminUserUsecase) *AdminUserHandler {
	return &AdminUserHandler{usecase: usecase}
}

type createAccountRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

type createAccountResponse struct {
	Admin adminResponse `json:"admin"`
	// The temporary password is emailed to the account, never returned here.
	MustChangePassword bool `json:"must_change_password"`
}

// Create serves POST /v1/admin/users. The route must be wrapped by the auth
// middleware and RequireRole(admin): that wrapper is the only authorization.
func (h *AdminUserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createAccountRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	result, err := h.usecase.CreateAccount(r.Context(), domain.CreateAccountInput{Email: req.Email, Name: req.Name, Role: req.Role})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	utils.WriteSuccessResponseWithStatus(w, http.StatusCreated, toCreateAccountResponse(result))
}

func toCreateAccountResponse(result *domain.CreateAccountResult) createAccountResponse {
	return createAccountResponse{
		Admin: adminResponse{
			ID: result.Admin.ID, Email: result.Admin.Email, Name: result.Admin.Name, Role: result.Admin.Role,
		},
		MustChangePassword: true,
	}
}

// ResetPassword serves POST /v1/admin/users/{id}/reset-password. Like Create it
// must be wrapped by the auth middleware and RequireRole(admin). The caller is
// the token's subject, so "not yourself" cannot be spoofed from the body.
func (h *AdminUserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := jwtcontext.GetClaims(r.Context())
	if !ok {
		WriteAuthError(w, r, errors.New("no claims in context"))
		return
	}
	actorID, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil {
		WriteAuthError(w, r, err)
		return
	}
	targetID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || targetID <= 0 {
		utils.WriteValidationError(w, "VALIDATION_ERROR", "id must be a positive integer", map[string]any{"field": "id"})
		return
	}

	result, err := h.usecase.ResetPassword(r.Context(), actorID, targetID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, toCreateAccountResponse(result))
}
