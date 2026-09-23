package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"

	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// loginUsecase is the subset of *usecase.LoginUsecase this handler depends on,
// declared here (the consumer) so LoginHandler can be unit tested with a mock
// instead of the real use case and its repository and JWT service.
type loginUsecase interface {
	Login(ctx context.Context, input domain.LoginInput) (*domain.LoginResult, error)
	Refresh(ctx context.Context, input domain.RefreshInput) (*domain.RefreshResult, error)
	Logout(ctx context.Context, input domain.LogoutInput) error
}

// LoginHandler serves US-002's three routes. login and refresh are public;
// logout must be wrapped by the auth middleware in cmd/main.go, which is what
// puts the claims this handler reads into the request context.
type LoginHandler struct {
	usecase loginUsecase
}

// NewLoginHandler wires the handler.
func NewLoginHandler(usecase loginUsecase) *LoginHandler {
	return &LoginHandler{usecase: usecase}
}

// loginRequest deliberately carries no validate tags: the use case owns the
// format rules (phone normalization, empty password, required device fields)
// and reports them as domain sentinels, so there is one place they can disagree
// with themselves rather than two. A missing "device" object decodes to the zero
// value and is rejected by the use case as INVALID_DEVICE.
type loginRequest struct {
	PhoneNumber string        `json:"phone_number"`
	Password    string        `json:"password"`
	Device      deviceRequest `json:"device"`
}

// deviceRequest is only what the client can honestly say about itself. The IP
// address and user agent are taken from the *http.Request, never from the body.
type deviceRequest struct {
	Platform    string `json:"platform"`
	DeviceModel string `json:"device_model"`
	OSVersion   string `json:"os_version,omitempty"`
	AppVersion  string `json:"app_version,omitempty"`
	DeviceName  string `json:"device_name,omitempty"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Login serves POST /v1/auth/login.
func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	result, err := h.usecase.Login(r.Context(), domain.LoginInput{
		PhoneNumber: req.PhoneNumber,
		Password:    req.Password,
		Device: domain.DeviceInfo{
			Platform:    req.Device.Platform,
			DeviceModel: req.Device.DeviceModel,
			OSVersion:   req.Device.OSVersion,
			AppVersion:  req.Device.AppVersion,
			DeviceName:  req.Device.DeviceName,
			IPAddress:   clientIP(r),
			UserAgent:   r.UserAgent(),
		},
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	utils.WriteSuccessResponse(w, verifyOTPResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User:         newUserResponse(result.User),
	})
}

// Refresh serves POST /v1/auth/refresh.
func (h *LoginHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	result, err := h.usecase.Refresh(r.Context(), domain.RefreshInput{RefreshToken: req.RefreshToken})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	utils.WriteSuccessResponse(w, refreshResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}

// Logout serves POST /v1/auth/logout. There is no body: the session to revoke
// comes from the access token's own claims, which the middleware has already
// verified, so the client cannot name a session that is not its own (AC13/AC14).
//
// A missing claim is 401, not a silent no-op: a token without "sid" (issued
// before US-001 carried it) cannot say which session to end, and reporting
// success would tell the client it is logged out when the server-side session
// lives on.
//
// Success is 200 {"success": true} rather than 204 so logout goes through the
// same envelope parser as every other route (api-response-envelope).
func (h *LoginHandler) Logout(w http.ResponseWriter, r *http.Request) {
	claims, ok := jwtcontext.GetClaims(r.Context())
	if !ok {
		WriteAuthError(w, r, errors.New("no claims in context"))
		return
	}

	userID, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil {
		WriteAuthError(w, r, errors.New("user_id claim is not numeric"))
		return
	}

	sessionID, _ := claims.Extra["sid"].(string)
	if sessionID == "" {
		WriteAuthError(w, r, errors.New("sid claim missing"))
		return
	}

	if err := h.usecase.Logout(r.Context(), domain.LogoutInput{UserID: userID, SessionID: sessionID}); err != nil {
		writeDomainError(w, err)
		return
	}

	utils.WriteSuccessResponse(w, nil)
}

// clientIP is the address the request arrived from, without the port. It does
// not consult X-Forwarded-For: that header is client-controlled unless a trusted
// proxy overwrites it, and this value only feeds device metadata, never a
// security decision.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
