package http

import (
	"context"
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

type mockAdminUserUsecase struct{ mock.Mock }

func (m *mockAdminUserUsecase) CreateAccount(ctx context.Context, in domain.CreateAccountInput) (*domain.CreateAccountResult, error) {
	args := m.Called(ctx, in)
	if r := args.Get(0); r != nil {
		return r.(*domain.CreateAccountResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAdminUserUsecase) ResetPassword(ctx context.Context, actorID, targetID int64) (*domain.CreateAccountResult, error) {
	args := m.Called(ctx, actorID, targetID)
	if r := args.Get(0); r != nil {
		return r.(*domain.CreateAccountResult), args.Error(1)
	}
	return nil, args.Error(1)
}

// TestAdminUserRoute_OnlyAdminMayCreate wires POST /v1/admin/users the way
// cmd/main.go does and proves every non-admin caller is kept out before the
// use case runs.
func TestAdminUserRoute_OnlyAdminMayCreate(t *testing.T) {
	adminJWT, err := jwt.New(&jwt.Config{Algorithm: jwt.AlgorithmHS256, SecretKey: "backoffice-secret-key-32-characters!!",
		AccessTokenExpiration: time.Minute, RefreshTokenExpiration: time.Hour, Issuer: "test", Audience: "backoffice"})
	require.NoError(t, err)
	userJWT, err := jwt.New(&jwt.Config{Algorithm: jwt.AlgorithmHS256, SecretKey: "user-app-secret-key-32-characters!!!!",
		AccessTokenExpiration: time.Minute, RefreshTokenExpiration: time.Hour, Issuer: "test", Audience: "users"})
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

	const body = `{"email":"new@x.com","name":"New","role":"viewer"}`
	created := &domain.CreateAccountResult{
		Admin: &domain.Admin{ID: 9, Email: "new@x.com", Name: "New", Role: domain.RoleViewer},
	}
	do := func(uc *mockAdminUserUsecase, token string) *httptest.ResponseRecorder {
		h := NewAdminUserHandler(uc)
		mux := http.NewServeMux()
		mux.HandleFunc("POST /v1/admin/users", mw.AuthenticateFunc(RequireRole(domain.RoleAdmin, h.Create)))
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/admin/users", strings.NewReader(body))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		mux.ServeHTTP(rec, req)
		return rec
	}

	t.Run("admin gets 201 and the password is not in the response", func(t *testing.T) {
		uc := new(mockAdminUserUsecase)
		uc.On("CreateAccount", mock.Anything, domain.CreateAccountInput{Email: "new@x.com", Name: "New", Role: domain.RoleViewer}).Return(created, nil)
		tok, _ := adminJWT.GenerateAccessToken("1", "a@x.com", domain.RoleAdmin, nil)

		rec := do(uc, tok)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.NotContains(t, rec.Body.String(), "temporary_password")
		assert.Contains(t, rec.Body.String(), `"must_change_password":true`)
		assert.NotContains(t, rec.Body.String(), "password_hash")
	})

	denied := map[string]struct {
		token func() string
		code  int
		err   string
	}{
		"no token":              {func() string { return "" }, http.StatusUnauthorized, "UNAUTHORIZED"},
		"appService user token": {func() string { t, _ := userJWT.GenerateAccessToken("3", "u@x.com", "admin", nil); return t }, http.StatusUnauthorized, "UNAUTHORIZED"},
		"refresh token":         {func() string { t, _ := adminJWT.GenerateRefreshToken("1", nil); return t }, http.StatusUnauthorized, "UNAUTHORIZED"},
		"viewer":                {func() string { t, _ := adminJWT.GenerateAccessToken("2", "v@x.com", domain.RoleViewer, nil); return t }, http.StatusForbidden, "FORBIDDEN"},
		"content_editor": {func() string {
			t, _ := adminJWT.GenerateAccessToken("2", "c@x.com", domain.RoleContentEditor, nil)
			return t
		}, http.StatusForbidden, "FORBIDDEN"},
		"unknown role": {func() string { t, _ := adminJWT.GenerateAccessToken("2", "x@x.com", "root", nil); return t }, http.StatusForbidden, "FORBIDDEN"},
		"admin who must change pw": {func() string {
			t, _ := adminJWT.GenerateAccessToken("1", "a@x.com", domain.RoleAdmin, map[string]any{domain.ExtraMustChangePassword: true})
			return t
		}, http.StatusForbidden, "PASSWORD_CHANGE_REQUIRED"},
	}
	for name, tc := range denied {
		t.Run("denied: "+name, func(t *testing.T) {
			uc := new(mockAdminUserUsecase)

			rec := do(uc, tc.token())

			assert.Equal(t, tc.code, rec.Code)
			assert.Equal(t, tc.err, errCode(t, rec))
			uc.AssertNotCalled(t, "CreateAccount", mock.Anything, mock.Anything)
		})
	}

	t.Run("maps use-case errors", func(t *testing.T) {
		tok, _ := adminJWT.GenerateAccessToken("1", "a@x.com", domain.RoleAdmin, nil)
		for name, tc := range map[string]struct {
			err  error
			code int
			body string
		}{
			"duplicate email": {domain.ErrEmailTaken, http.StatusConflict, "EMAIL_TAKEN"},
			"invalid input":   {errors.Join(domain.ErrInvalidAccountInput), http.StatusBadRequest, "VALIDATION_ERROR"},
			"email send fail": {domain.ErrInviteEmailFailed, http.StatusBadGateway, "EMAIL_SEND_FAILED"},
			"unexpected":      {errors.New("pq: secret detail"), http.StatusInternalServerError, "INTERNAL_ERROR"},
		} {
			t.Run(name, func(t *testing.T) {
				uc := new(mockAdminUserUsecase)
				uc.On("CreateAccount", mock.Anything, mock.Anything).Return(nil, tc.err)

				rec := do(uc, tok)

				assert.Equal(t, tc.code, rec.Code)
				assert.Equal(t, tc.body, errCode(t, rec))
				assert.NotContains(t, rec.Body.String(), "secret detail")
			})
		}
	})
}

func TestAdminUserHandler_ResetPassword(t *testing.T) {
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
	adminTok, _ := jwtSvc.GenerateAccessToken("1", "a@x.com", domain.RoleAdmin, nil)
	viewerTok, _ := jwtSvc.GenerateAccessToken("2", "v@x.com", domain.RoleViewer, nil)

	do := func(uc *mockAdminUserUsecase, token, id string) *httptest.ResponseRecorder {
		h := NewAdminUserHandler(uc)
		mux := http.NewServeMux()
		mux.HandleFunc("POST /v1/admin/users/{id}/reset-password", mw.AuthenticateFunc(RequireRole(domain.RoleAdmin, h.ResetPassword)))
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/admin/users/"+id+"/reset-password", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		mux.ServeHTTP(rec, req)
		return rec
	}

	t.Run("200 uses the token subject as actor and the path id as target", func(t *testing.T) {
		uc := new(mockAdminUserUsecase)
		uc.On("ResetPassword", mock.Anything, int64(1), int64(9)).
			Return(&domain.CreateAccountResult{Admin: &domain.Admin{ID: 9, Email: "v@x.com", Role: domain.RoleViewer}}, nil)

		rec := do(uc, adminTok, "9")

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"must_change_password":true`)
		assert.NotContains(t, rec.Body.String(), "password_hash")
	})

	t.Run("403 for a non-admin, use case not called", func(t *testing.T) {
		uc := new(mockAdminUserUsecase)
		rec := do(uc, viewerTok, "9")
		assert.Equal(t, http.StatusForbidden, rec.Code)
		uc.AssertNotCalled(t, "ResetPassword", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("400 on a bad id", func(t *testing.T) {
		for _, id := range []string{"abc", "0", "-3"} {
			uc := new(mockAdminUserUsecase)
			assert.Equal(t, http.StatusBadRequest, do(uc, adminTok, id).Code, id)
			uc.AssertNotCalled(t, "ResetPassword", mock.Anything, mock.Anything, mock.Anything)
		}
	})

	for name, tc := range map[string]struct {
		err  error
		code int
		body string
	}{
		"self":      {domain.ErrCannotResetSelf, http.StatusConflict, "CANNOT_RESET_SELF"},
		"not found": {domain.ErrAdminNotFound, http.StatusNotFound, "ADMIN_NOT_FOUND"},
		"email":     {domain.ErrInviteEmailFailed, http.StatusBadGateway, "EMAIL_SEND_FAILED"},
	} {
		t.Run(name, func(t *testing.T) {
			uc := new(mockAdminUserUsecase)
			uc.On("ResetPassword", mock.Anything, mock.Anything, mock.Anything).Return(nil, tc.err)
			rec := do(uc, adminTok, "9")
			assert.Equal(t, tc.code, rec.Code)
			assert.Equal(t, tc.body, errCode(t, rec))
		})
	}
}
