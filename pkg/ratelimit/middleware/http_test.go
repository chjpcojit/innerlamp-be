package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chjpcoj.com.innerlamp/pkg/ratelimit"
	"chjpcoj.com.innerlamp/pkg/ratelimit/store"
	"chjpcoj.com.innerlamp/pkg/utils"
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

func TestHTTPMiddleware_BlockedUsesEnvelope(t *testing.T) {
	mw, err := NewHTTPMiddleware(ratelimit.Config{
		Rate:      1,
		Burst:     1,
		Window:    time.Minute,
		Algorithm: ratelimit.TokenBucket,
		Store:     store.NewMemoryStore(),
		KeyFunc:   func(context.Context) (string, error) { return "k", nil },
	})
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
	var resp utils.ApiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body is not the ApiResponse envelope: %v (%q)", err, w.Body.String())
	}
	if resp.Success || resp.Error == nil || resp.Error.Code != "RATE_LIMITED" || resp.Error.Type != utils.ErrorTypeBusiness {
		t.Fatalf("unexpected envelope: %q", w.Body.String())
	}
	if _, ok := resp.Details["retry_after_seconds"]; !ok {
		t.Errorf("expected details.retry_after_seconds, got %v", resp.Details)
	}
	// A sub-second wait must round up, never tell the client to retry now.
	// Thời gian chờ dưới 1 giây phải làm tròn lên, không bảo client thử lại ngay.
	if got := w.Header().Get("Retry-After"); got == "" || got == "0" {
		t.Errorf("Retry-After = %q, want >= 1", got)
	}
}
