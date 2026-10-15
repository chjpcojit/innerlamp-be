package main

import (
	"os"
	"strings"
	"testing"
)

// Login AC21: every Bearer route must run the active-session check. main.go
// wraps routes with `authed` (token check + session check); the only place the
// bare authMiddleware.AuthenticateFunc may appear is the definition of `authed`
// itself and POST /v1/auth/logout (idempotent, AC14). Any other use is a route
// that would accept a revoked session until its access token expires.
func TestEveryBearerRouteChecksSessionLiveness(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(string(src), "\n") {
		if !strings.Contains(line, "authMiddleware.AuthenticateFunc(") {
			continue
		}
		if strings.Contains(line, "activeSession.WrapFunc(") || strings.Contains(line, "/v1/auth/logout") {
			continue
		}
		t.Errorf("main.go:%d uses authMiddleware.AuthenticateFunc without the active-session check: %s", i+1, strings.TrimSpace(line))
	}
	for _, f := range []string{"profile_routes.go", "temple_announcement_routes.go"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "authMiddleware") {
			t.Errorf("%s must take the auth wrapper as a parameter, not build its own", f)
		}
	}
}
