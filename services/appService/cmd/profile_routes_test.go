package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	jwtclaims "chjpcoj.com.innerlamp/pkg/jwt/claims"
	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	deliveryhttp "chjpcoj.com.innerlamp/services/appService/delivery/http"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// fakeProfileUsecase counts calls, so a test can prove a rejected request never
// reached the use case.
type fakeProfileUsecase struct{ calls int }

func (f *fakeProfileUsecase) view() (*domain.ProfileView, error) {
	f.calls++
	return &domain.ProfileView{Profile: domain.Profile{Email: "an@example.com", PhoneNumber: "0901234567"}}, nil
}

func (f *fakeProfileUsecase) Get(context.Context, int64) (*domain.ProfileView, error) {
	return f.view()
}
func (f *fakeProfileUsecase) Update(context.Context, int64, domain.ProfilePatch) (*domain.ProfileView, error) {
	return f.view()
}
func (f *fakeProfileUsecase) ConfirmAvatar(context.Context, int64, string) (*domain.ProfileView, error) {
	return f.view()
}
func (f *fakeProfileUsecase) RemoveAvatar(context.Context, int64) (*domain.ProfileView, error) {
	return f.view()
}
func (f *fakeProfileUsecase) RequestAvatarUpload(context.Context, int64, domain.AvatarUploadRequest) (*domain.AvatarUploadTicket, error) {
	f.calls++
	return &domain.AvatarUploadTicket{UploadID: "u", URL: "https://x", Method: "PUT", ExpiresAt: time.Now()}, nil
}

// fakeAuth stands in for authMiddleware.AuthenticateFunc: no Authorization
// header is a 401, anything else carries claims for user 7 into the handler.
func fakeAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			deliveryhttp.WriteAuthError(w, r, nil)
			return
		}
		claims := &jwtclaims.CustomClaims{UserID: "7", TokenType: jwtclaims.TokenTypeAccess, RegisteredClaims: jwt.RegisteredClaims{}}
		next(w, r.WithContext(jwtcontext.WithClaims(r.Context(), claims)))
	}
}

var profileRoutes = []struct{ method, path, body string }{
	{http.MethodGet, "/v1/users/me", ""},
	{http.MethodPatch, "/v1/users/me", `{}`},
	{http.MethodPost, "/v1/users/me/avatar/upload-url", `{"content_type":"image/png","size_bytes":10}`},
	{http.MethodPost, "/v1/users/me/avatar/confirm", `{"upload_id":"0b9c4f3e-5a1d-4c7e-9b2a-6d8e1f0a3c11"}`},
	{http.MethodDelete, "/v1/users/me/avatar", ""},
}

func newProfileRouteMux(t *testing.T, max int64) (*http.ServeMux, *fakeProfileUsecase) {
	t.Helper()
	limit, err := deliveryhttp.NewUserRateLimit(max, time.Hour)
	if err != nil {
		t.Fatalf("NewUserRateLimit: %v", err)
	}
	uc := &fakeProfileUsecase{}
	mux := http.NewServeMux()
	registerProfileRoutes(mux, fakeAuth, limit, deliveryhttp.NoStore{}, deliveryhttp.NewProfileHandler(uc))
	return mux, uc
}

func do(mux *http.ServeMux, method, path, body string, authed bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if authed {
		req.Header.Set("Authorization", "Bearer x")
	}
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr
}

// Every route must sit behind the auth middleware (AC7, AC17, AC27): a route
// registered without it would leak a profile, and no unit test of the handler
// alone could see that.
func TestRegisterProfileRoutes_EveryRouteRequiresAuth(t *testing.T) {
	for _, rt := range profileRoutes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			mux, uc := newProfileRouteMux(t, 100)

			rr := do(mux, rt.method, rt.path, rt.body, false)

			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401; body %s", rr.Code, rr.Body.String())
			}
			if uc.calls != 0 {
				t.Errorf("use case ran %d times for an unauthenticated request", uc.calls)
			}
		})
	}
}

func TestRegisterProfileRoutes_AuthenticatedRoutesAreRegisteredAndNoStore(t *testing.T) {
	for _, rt := range profileRoutes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			mux, _ := newProfileRouteMux(t, 100)

			rr := do(mux, rt.method, rt.path, rt.body, true)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body %s", rr.Code, rr.Body.String())
			}
			if got := rr.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control = %q, want no-store: the body has PII or a signed URL", got)
			}
		})
	}
}

// One budget for the whole group, counted per account and after auth, so reads
// and edits spend the same allowance (AC28) and anonymous traffic spends none.
func TestRegisterProfileRoutes_RateLimitIsSharedAndAfterAuth(t *testing.T) {
	mux, uc := newProfileRouteMux(t, 2)

	for i := 0; i < 5; i++ {
		if rr := do(mux, http.MethodGet, "/v1/users/me", "", false); rr.Code != http.StatusUnauthorized {
			t.Fatalf("anonymous request %d: status = %d, want 401", i+1, rr.Code)
		}
	}
	if rr := do(mux, http.MethodGet, "/v1/users/me", "", true); rr.Code != http.StatusOK {
		t.Fatalf("1st authed read: %d", rr.Code)
	}
	if rr := do(mux, http.MethodPatch, "/v1/users/me", `{}`, true); rr.Code != http.StatusOK {
		t.Fatalf("2nd authed request (a different route): %d", rr.Code)
	}
	before := uc.calls
	rr := do(mux, http.MethodDelete, "/v1/users/me/avatar", "", true)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("3rd authed request: status = %d, want 429", rr.Code)
	}
	if uc.calls != before {
		t.Error("a limited request must not reach the use case")
	}
}
