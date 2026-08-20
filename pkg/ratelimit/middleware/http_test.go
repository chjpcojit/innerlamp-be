package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chjpcoj.com.innerlamp/pkg/ratelimit"
	"chjpcoj.com.innerlamp/pkg/ratelimit/store"
)

func TestHTTPMiddleware(t *testing.T) {
	// Setup
	cfg := ratelimit.Config{
		Rate:      5,
		Burst:     5,
		Window:    time.Second,
		Algorithm: ratelimit.TokenBucket,
		Store:     store.NewMemoryStore(),
		KeyFunc: func(ctx context.Context) (string, error) {
			path, ok := ctx.Value(ContextKeyHTTPPath).(string)
			if !ok {
				return "unknown", nil
			}
			return "path:" + path, nil
		},
	}

	mw, err := NewHTTPMiddleware(cfg)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test allowed requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("request %d failed: got status %d", i, w.Code)
		}

		if w.Header().Get("X-RateLimit-Remaining") == "" {
			t.Error("missing X-RateLimit-Remaining header")
		}
	}

	// Test blocked request
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("request 6 should be blocked: got status %d", w.Code)
	}
}
