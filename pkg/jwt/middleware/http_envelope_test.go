package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"chjpcoj.com.innerlamp/pkg/utils"
)

// assertEnvelope checks rr carries the utils.ApiResponse error envelope with
// the given status and code — the one error shape every framework response uses.
func assertEnvelope(t *testing.T, rr *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rr.Code != status {
		t.Fatalf("expected status %d, got %d", status, rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
	var resp utils.ApiResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body is not the ApiResponse envelope: %v (%q)", err, rr.Body.String())
	}
	if resp.Success || resp.Error == nil {
		t.Fatalf("expected success=false with error, got %q", rr.Body.String())
	}
	if resp.Error.Code != code || resp.Error.Type != utils.ErrorTypeBusiness || resp.Error.Message == "" {
		t.Errorf("unexpected error detail: %+v", *resp.Error)
	}
}

func TestAuthenticate_DefaultErrorUsesEnvelope(t *testing.T) {
	mw := NewHTTPAuthMiddleware(createTestJWTService(), createTestLogger())
	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for name, header := range map[string]string{
		"missing token": "",
		"invalid token": "Bearer invalid.token.here",
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if header != "" {
				req.Header.Set("Authorization", header)
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assertEnvelope(t, rr, http.StatusUnauthorized, "UNAUTHORIZED")
		})
	}
}

func TestRequireRole_ErrorsUseEnvelope(t *testing.T) {
	service := createTestJWTService()
	mw := NewHTTPAuthMiddleware(service, createTestLogger())
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	t.Run("no claims in context", func(t *testing.T) {
		rr := httptest.NewRecorder()
		mw.RequireRole("admin")(ok).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin", nil))
		assertEnvelope(t, rr, http.StatusUnauthorized, "UNAUTHORIZED")
	})

	t.Run("wrong role", func(t *testing.T) {
		token, err := service.GenerateAccessToken("user-1", "u@example.com", "user", nil)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		mw.Authenticate(mw.RequireRole("admin")(ok)).ServeHTTP(rr, req)
		assertEnvelope(t, rr, http.StatusForbidden, "FORBIDDEN")
	})
}
