// Package http holds backofficeService's HTTP delivery: DTOs, handlers and the
// error mapping onto the shared utils.ApiResponse envelope.
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

// adminAuthUsecase is the subset of *usecase.AdminAuthUsecase the handler needs,
// declared here (the consumer) so the handler can be tested with a mock.
type adminAuthUsecase interface {
	Login(ctx context.Context, input domain.AdminLoginInput) (*domain.AdminLoginResult, error)
	ChangePassword(ctx context.Context, input domain.ChangePasswordInput) (*domain.AdminLoginResult, error)
}

// AdminAuthHandler serves the admin login and /me routes.
type AdminAuthHandler struct {
	usecase adminAuthUsecase
}

// NewAdminAuthHandler wires the handler.
func NewAdminAuthHandler(usecase adminAuthUsecase) *AdminAuthHandler {
	return &AdminAuthHandler{usecase: usecase}
}

type adminLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type adminResponse struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

type adminLoginResponse struct {
	AccessToken string        `json:"access_token"`
	Admin       adminResponse `json:"admin"`
	// MustChangePassword tells the client to send the admin to the change-password
	// screen; the token is refused everywhere else until they do.
	MustChangePassword bool `json:"must_change_password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func toLoginResponse(result *domain.AdminLoginResult) adminLoginResponse {
	return adminLoginResponse{
		AccessToken: result.AccessToken,
		Admin: adminResponse{
			ID: result.Admin.ID, Email: result.Admin.Email, Name: result.Admin.Name, Role: result.Admin.Role,
		},
		MustChangePassword: result.Admin.MustChangePassword,
	}
}

// Login serves POST /v1/admin/auth/login (public).
func (h *AdminAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req adminLoginRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	result, err := h.usecase.Login(r.Context(), domain.AdminLoginInput{Email: req.Email, Password: req.Password})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	utils.WriteSuccessResponse(w, toLoginResponse(result))
}

// ChangePassword serves POST /v1/admin/auth/change-password. It must be wrapped
// by the auth middleware and RequireRoleAllowPasswordChange; the admin is the
// token's subject, never a body field.
func (h *AdminAuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := jwtcontext.GetClaims(r.Context())
	if !ok {
		WriteAuthError(w, r, errors.New("no claims in context"))
		return
	}
	adminID, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil {
		WriteAuthError(w, r, err)
		return
	}
	var req changePasswordRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	result, err := h.usecase.ChangePassword(r.Context(), domain.ChangePasswordInput{
		AdminID: adminID, CurrentPassword: req.CurrentPassword, NewPassword: req.NewPassword,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, toLoginResponse(result))
}

// Me serves GET /v1/admin/me. It must be wrapped by the auth middleware and
// RequireRole; it answers from the verified token claims, with no DB read.
func (h *AdminAuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := jwtcontext.GetClaims(r.Context())
	if !ok {
		WriteAuthError(w, r, errors.New("no claims in context"))
		return
	}
	utils.WriteSuccessResponse(w, map[string]any{
		"user_id": claims.UserID,
		"email":   claims.Email,
		"role":    claims.Role,
	})
}
