package main

import (
	"net/http"

	deliveryhttp "chjpcoj.com.innerlamp/services/appService/delivery/http"
)

// registerProfileRoutes mounts the five /v1/users/me routes. Each one is wrapped
// as auth → per-account rate limit → no-store, outermost first:
//
//   - auth first, because the limiter keys on the account the token names and an
//     anonymous request must be refused without spending anyone's budget;
//   - no-store innermost, because the body carries PII and a signed URL.
//
// It is a function (not inline in main) so cmd's tests assert the wiring main
// actually uses: a route registered without its middleware is a wiring bug no
// handler-level test can see.
func registerProfileRoutes(
	mux *http.ServeMux,
	auth func(http.HandlerFunc) http.HandlerFunc,
	limit *deliveryhttp.UserRateLimit,
	noStore deliveryhttp.NoStore,
	h *deliveryhttp.ProfileHandler,
) {
	guard := func(next http.HandlerFunc) http.HandlerFunc {
		return auth(limit.WrapFunc(noStore.WrapFunc(next)))
	}
	mux.HandleFunc("GET /v1/users/me", guard(h.Get))
	mux.HandleFunc("PATCH /v1/users/me", guard(h.Patch))
	mux.HandleFunc("POST /v1/users/me/avatar/upload-url", guard(h.AvatarUploadURL))
	mux.HandleFunc("POST /v1/users/me/avatar/confirm", guard(h.AvatarConfirm))
	mux.HandleFunc("DELETE /v1/users/me/avatar", guard(h.AvatarDelete))
}
