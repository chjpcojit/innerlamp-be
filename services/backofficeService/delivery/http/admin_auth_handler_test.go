package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/jwt"
	jwtmiddleware "chjpcoj.com.innerlamp/pkg/jwt/middleware"
	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

type mockAdminAuthUsecase struct{ mock.Mock }

func (m *mockAdminAuthUsecase) Login(ctx context.Context, in domain.AdminLoginInput) (*domain.AdminLoginResult, error) {
	args := m.Called(ctx, in)
	if r := args.Get(0); r != nil {
		return r.(*domain.AdminLoginResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAdminAuthUsecase) ChangePassword(ctx context.Context, in domain.ChangePasswordInput) (*domain.AdminLoginResult, error) {
	args := m.Called(ctx, in)
	if r := args.Get(0); r != nil {
		return r.(*domain.AdminLoginResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func errCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), rec.Body.String())
	return body.Error.Code
}

func TestAdminAuthHandler_Login(t *testing.T) {
	post := func(uc *mockAdminAuthUsecase, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/admin/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		NewAdminAuthHandler(uc).Login(rec, req)
		return rec
	}

	t.Run("200 with token and admin", func(t *testing.T) {
		uc := new(mockAdminAuthUsecase)
		uc.On("Login", mock.Anything, domain.AdminLoginInput{Email: "a@x.com", Password: "pw"}).
			Return(&domain.AdminLoginResult{AccessToken: "tok", Admin: &domain.Admin{ID: 1, Email: "a@x.com", Name: "A", Role: "admin"}}, nil)

		rec := post(uc, `{"email":"a@x.com","password":"pw"}`)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"access_token":"tok"`)
		assert.NotContains(t, rec.Body.String(), "password_hash")
	})

	t.Run("401 INVALID_CREDENTIALS", func(t *testing.T) {
		uc := new(mockAdminAuthUsecase)
		uc.On("Login", mock.Anything, mock.Anything).Return(nil, domain.ErrInvalidCredentials)

		rec := post(uc, `{"email":"a@x.com","password":"bad"}`)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "INVALID_CREDENTIALS", errCode(t, rec))
	})

	t.Run("401 PASSWORD_EXPIRED", func(t *testing.T) {
		uc := new(mockAdminAuthUsecase)
		uc.On("Login", mock.Anything, mock.Anything).Return(nil, domain.ErrPasswordExpired)

		rec := post(uc, `{"email":"a@x.com","password":"111111"}`)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "PASSWORD_EXPIRED", errCode(t, rec))
	})

	t.Run("500 hides the cause", func(t *testing.T) {
		uc := new(mockAdminAuthUsecase)
		uc.On("Login", mock.Anything, mock.Anything).Return(nil, errors.New("pq: secret detail"))

		rec := post(uc, `{"email":"a@x.com","password":"pw"}`)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.NotContains(t, rec.Body.String(), "secret detail")
	})

	t.Run("400 on unknown field", func(t *testing.T) {
		rec := post(new(mockAdminAuthUsecase), `{"email":"a@x.com","password":"pw","x":1}`)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "VALIDATION_ERROR", errCode(t, rec))
	})
}

// TestAdminRoutes_AuthAndRole wires the middleware the way cmd/main.go does and
// proves a user-audience token, a refresh token and a wrong-role token are all
// kept out of /v1/admin/me.
func TestAdminRoutes_AuthAndRole(t *testing.T) {
	cfg := func(aud, key string) *jwt.Config {
		return &jwt.Config{Algorithm: jwt.AlgorithmHS256, SecretKey: key, AccessTokenExpiration: time.Minute,
			RefreshTokenExpiration: time.Hour, Issuer: "test", Audience: aud}
	}
	adminJWT, err := jwt.New(cfg("backoffice", "backoffice-secret-key-32-characters!!"))
	require.NoError(t, err)
	userJWT, err := jwt.New(cfg("users", "user-app-secret-key-32-characters!!!!"))
	require.NoError(t, err)

	l := &logging.MockLogger{}
	l.On("Warn", mock.Anything, mock.Anything).Return()
	l.On("Debug", mock.Anything, mock.Anything).Return()
	l.On("Info", mock.Anything, mock.Anything).Return()
	l.On("WithContext", mock.Anything).Return(l)
	l.On("WithFields", mock.Anything).Return(l)
	mw := jwtmiddleware.NewHTTPAuthMiddlewareWithOptions(adminJWT, l, jwtmiddleware.HTTPAuthOptions{
		RequireTokenType: "access", ErrorHandler: WriteAuthError,
	})
	h := NewAdminAuthHandler(new(mockAdminAuthUsecase))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/admin/me", mw.AuthenticateFunc(RequireRole(domain.RoleAdmin, h.Me)))
	mux.HandleFunc("POST /v1/admin/auth/change-password", mw.AuthenticateFunc(RequireStaffAllowPasswordChange(h.Me)))

	call := func(method, path, token string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		mux.ServeHTTP(rec, req)
		return rec
	}
	get := func(token string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/admin/me", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		mux.ServeHTTP(rec, req)
		return rec
	}

	adminTok, _ := adminJWT.GenerateAccessToken("1", "a@x.com", domain.RoleAdmin, nil)
	viewerTok, _ := adminJWT.GenerateAccessToken("2", "v@x.com", "viewer", nil)
	refreshTok, _ := adminJWT.GenerateRefreshToken("1", nil)
	userTok, _ := userJWT.GenerateAccessToken("3", "u@x.com", "admin", nil)

	assert.Equal(t, http.StatusOK, get(adminTok).Code)
	assert.Equal(t, http.StatusUnauthorized, get("").Code, "no token")
	assert.Equal(t, http.StatusUnauthorized, get(refreshTok).Code, "refresh token")
	assert.Equal(t, http.StatusUnauthorized, get(userTok).Code, "token signed by appService")
	rec := get(viewerTok)
	assert.Equal(t, http.StatusForbidden, rec.Code, "wrong role")
	assert.Equal(t, "FORBIDDEN", errCode(t, rec))

	// A must-change-password token reaches only the change-password route.
	pendingTok, _ := adminJWT.GenerateAccessToken("1", "a@x.com", domain.RoleAdmin, map[string]any{domain.ExtraMustChangePassword: true})
	rec = call(http.MethodGet, "/v1/admin/me", pendingTok)
	assert.Equal(t, http.StatusForbidden, rec.Code, "must-change token on a normal route")
	assert.Equal(t, "PASSWORD_CHANGE_REQUIRED", errCode(t, rec))
	assert.Equal(t, http.StatusOK, call(http.MethodPost, "/v1/admin/auth/change-password", pendingTok).Code)
}

func TestAdminAuthHandler_ChangePassword(t *testing.T) {
	jwtSvc, err := jwt.New(&jwt.Config{Algorithm: jwt.AlgorithmHS256, SecretKey: "backoffice-secret-key-32-characters!!",
		AccessTokenExpiration: time.Minute, RefreshTokenExpiration: time.Hour, Issuer: "test", Audience: "backoffice"})
	require.NoError(t, err)
	l := &logging.MockLogger{}
	l.On("Warn", mock.Anything, mock.Anything).Return()
	l.On("Debug", mock.Anything, mock.Anything).Return()
	l.On("Info", mock.Anything, mock.Anything).Return()
	l.On("WithContext", mock.Anything).Return(l)
	l.On("WithFields", mock.Anything).Return(l)
	mw := jwtmiddleware.NewHTTPAuthMiddlewareWithOptions(jwtSvc, l, jwtmiddleware.HTTPAuthOptions{
		RequireTokenType: "access", ErrorHandler: WriteAuthError,
	})
	tok, _ := jwtSvc.GenerateAccessToken("7", "a@x.com", domain.RoleAdmin, map[string]any{domain.ExtraMustChangePassword: true})

	post := func(uc *mockAdminAuthUsecase, body string) *httptest.ResponseRecorder {
		h := NewAdminAuthHandler(uc)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+tok)
		mw.AuthenticateFunc(RequireStaffAllowPasswordChange(h.ChangePassword))(rec, req)
		return rec
	}

	t.Run("200 uses the token subject and returns the new token", func(t *testing.T) {
		uc := new(mockAdminAuthUsecase)
		uc.On("ChangePassword", mock.Anything, domain.ChangePasswordInput{AdminID: 7, CurrentPassword: "admin", NewPassword: "a-much-better-password"}).
			Return(&domain.AdminLoginResult{AccessToken: "new", Admin: &domain.Admin{ID: 7, Email: "a@x.com", Role: domain.RoleAdmin}}, nil)

		rec := post(uc, `{"current_password":"admin","new_password":"a-much-better-password"}`)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"access_token":"new"`)
		assert.Contains(t, rec.Body.String(), `"must_change_password":false`)
	})

	for name, tc := range map[string]struct {
		err    error
		status int
		code   string
	}{
		"wrong current password": {domain.ErrWrongCurrentPassword, http.StatusBadRequest, "WRONG_CURRENT_PASSWORD"},
		"weak new password":      {domain.ErrWeakPassword, http.StatusBadRequest, "WEAK_PASSWORD"},
		"invalid credentials":    {domain.ErrInvalidCredentials, http.StatusUnauthorized, "INVALID_CREDENTIALS"},
	} {
		t.Run(name, func(t *testing.T) {
			uc := new(mockAdminAuthUsecase)
			uc.On("ChangePassword", mock.Anything, mock.Anything).Return(nil, tc.err)

			rec := post(uc, `{"current_password":"x","new_password":"y"}`)

			assert.Equal(t, tc.status, rec.Code)
			assert.Equal(t, tc.code, errCode(t, rec))
		})
	}

	t.Run("400 on unknown field", func(t *testing.T) {
		assert.Equal(t, http.StatusBadRequest, post(new(mockAdminAuthUsecase), `{"admin_id":1}`).Code)
	})
}

type mockSessionValidator struct{ mock.Mock }

func (m *mockSessionValidator) ValidateSession(ctx context.Context, id int64, iat time.Time) error {
	return m.Called(ctx, id, iat).Error(0)
}

func TestRequireActiveSession(t *testing.T) {
	jwtSvc, err := jwt.New(&jwt.Config{Algorithm: jwt.AlgorithmHS256, SecretKey: "backoffice-secret-key-32-characters!!",
		AccessTokenExpiration: time.Minute, RefreshTokenExpiration: time.Hour, Issuer: "test", Audience: "backoffice"})
	require.NoError(t, err)
	l := &logging.MockLogger{}
	l.On("Warn", mock.Anything, mock.Anything).Return()
	l.On("Debug", mock.Anything, mock.Anything).Return()
	l.On("Info", mock.Anything, mock.Anything).Return()
	l.On("WithContext", mock.Anything).Return(l)
	l.On("WithFields", mock.Anything).Return(l)
	mw := jwtmiddleware.NewHTTPAuthMiddlewareWithOptions(jwtSvc, l, jwtmiddleware.HTTPAuthOptions{
		RequireTokenType: "access", ErrorHandler: WriteAuthError,
	})
	tok, _ := jwtSvc.GenerateAccessToken("7", "a@x.com", domain.RoleAdmin, nil)

	call := func(v *mockSessionValidator) (*httptest.ResponseRecorder, bool) {
		reached := false
		next := func(http.ResponseWriter, *http.Request) { reached = true }
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		mw.AuthenticateFunc(RequireActiveSession(v, next))(rec, req)
		return rec, reached
	}

	t.Run("passes the token subject and iat to the validator", func(t *testing.T) {
		v := new(mockSessionValidator)
		v.On("ValidateSession", mock.Anything, int64(7), mock.MatchedBy(func(iat time.Time) bool {
			return time.Since(iat) < time.Minute
		})).Return(nil)
		_, reached := call(v)
		assert.True(t, reached)
		v.AssertExpectations(t)
	})

	t.Run("401 when revoked", func(t *testing.T) {
		v := new(mockSessionValidator)
		v.On("ValidateSession", mock.Anything, mock.Anything, mock.Anything).Return(domain.ErrSessionRevoked)
		rec, reached := call(v)
		assert.False(t, reached)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "UNAUTHORIZED", errCode(t, rec))
	})

	t.Run("500 on a validator failure, not a silent pass", func(t *testing.T) {
		v := new(mockSessionValidator)
		v.On("ValidateSession", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db down"))
		rec, reached := call(v)
		assert.False(t, reached)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
