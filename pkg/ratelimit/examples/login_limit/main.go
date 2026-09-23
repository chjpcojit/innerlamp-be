package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"chjpcoj.com.innerlamp/pkg/ratelimit"
	"chjpcoj.com.innerlamp/pkg/ratelimit/middleware"
	"chjpcoj.com.innerlamp/pkg/ratelimit/store"
	"chjpcoj.com.innerlamp/pkg/utils"
)

func main() {
	// 1. Configure Rate Limiter for Login
	// Strategy: Allow 5 login attempts per minute per user
	config := ratelimit.Config{
		Rate:      5,           // 5 requests
		Burst:     5,           // Max burst of 5
		Window:    time.Minute, // Per 1 minute
		Algorithm: ratelimit.TokenBucket,
		Store:     store.NewMemoryStore(), // Use Redis in production

		// KEY STRATEGY: Combine "login" prefix + User ID (or IP if not logged in)
		KeyFunc: func(ctx context.Context) (string, error) {
			// Get the path to ensure we only limit the login endpoint specially
			path, _ := ctx.Value(middleware.ContextKeyHTTPPath).(string)

			// If it's NOT the login endpoint, maybe use a different global limit?
			// For this example, we assume this middleware is ONLY applied to /login
			// or we return a different key.

			// Get User Identifier (e.g., email from form body, or IP)
			// NOTE: In a real login handler, you might parse the body first.
			// But rate limiting usually happens BEFORE body parsing to prevent DoS.
			// So usually we limit by IP for login.
			// BUT, if you want to limit by "User A", you imply you know who User A is.
			// Let's assume we limit by IP for this example as it's safer for login.
			// OR, if you have a multi-step login or API token, you use that.

			// Let's simulate extracting a "User-ID" header for demonstration
			// (e.g. a client sends X-User-ID)
			userID := "unknown"
			// In real code, you might need to access the request headers.
			// Since KeyFunc only gets context, you might need a custom middleware
			// before this to put headers into context if you need them.
			// OR, we can just use the IP which is common for login protection.

			return fmt.Sprintf("login_limit:%s:%s", path, userID), nil
		},
	}

	// 2. Create Middleware
	mw, err := middleware.NewHTTPMiddleware(config)
	if err != nil {
		panic(err)
	}

	// 3. Setup Router
	mux := http.NewServeMux()

	// Login Handler
	loginHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		utils.WriteSuccessResponse(w, map[string]any{"message": "Login successful!"})
	})

	// Wrap Login Handler with Rate Limit
	// We only apply this strict limit to /login
	mux.Handle("/login", mw.Handler(loginHandler))

	// Other endpoints might have no limit or a different limit
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		utils.WriteSuccessResponse(w, map[string]any{"message": "Home page (no limit)"})
	})

	fmt.Println("Server started at :8080")
	http.ListenAndServe(":8080", mux) // #nosec
}
