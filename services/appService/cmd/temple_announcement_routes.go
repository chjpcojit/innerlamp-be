package main

import (
	"net/http"

	deliveryhttp "chjpcoj.com.innerlamp/services/appService/delivery/http"
)

// registerTempleAnnouncementRoutes mounts the three announcement routes under
// /v1/users/me. Each is wrapped auth → dedicated per-account rate limit, outermost
// first (TA2, TA32, TA33). The limiter is its own instance, NOT the profile one:
// reading announcements must not spend the profile budget, nor the reverse.
// It is a function so cmd's tests assert the wiring main actually uses.
func registerTempleAnnouncementRoutes(
	mux *http.ServeMux,
	auth func(http.HandlerFunc) http.HandlerFunc,
	limit *deliveryhttp.UserRateLimit,
	h *deliveryhttp.TempleAnnouncementHandler,
) {
	guard := func(next http.HandlerFunc) http.HandlerFunc {
		return auth(limit.WrapFunc(next))
	}
	mux.HandleFunc("GET /v1/users/me/temple-announcements", guard(h.List))
	mux.HandleFunc("GET /v1/users/me/temple-announcement-settings", guard(h.GetSettings))
	mux.HandleFunc("PUT /v1/users/me/temple-announcement-settings", guard(h.PutSettings))
}
