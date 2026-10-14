package store

import (
	"context"
	"testing"
	"time"

	"chjpcoj.com.innerlamp/pkg/ratelimit"
)

func TestTokenBucket(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	key := "test-tb"
	limit := int64(10)
	window := time.Second
	burst := int64(10)

	// Should allow 10 requests
	for i := 0; i < 10; i++ {
		res, err := s.Allow(ctx, key, limit, window, burst, ratelimit.TokenBucket)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.Allowed {
			t.Errorf("request %d should be allowed", i)
		}
	}

	// Should block 11th request
	res, err := s.Allow(ctx, key, limit, window, burst, ratelimit.TokenBucket)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Allowed {
		t.Error("request 11 should be blocked")
	}
}

func TestSlidingWindow(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	key := "test-sw"
	limit := int64(5)
	window := time.Second
	burst := int64(5)

	// Should allow 5 requests
	for i := 0; i < 5; i++ {
		res, err := s.Allow(ctx, key, limit, window, burst, ratelimit.SlidingWindow)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.Allowed {
			t.Errorf("request %d should be allowed", i)
		}
	}

	// Should block 6th request
	res, err := s.Allow(ctx, key, limit, window, burst, ratelimit.SlidingWindow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Allowed {
		t.Error("request 6 should be blocked")
	}
}

func TestFixedWindow(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	key := "test-fw"
	limit := int64(5)
	window := time.Second
	burst := int64(5)

	// Should allow 5 requests
	for i := 0; i < 5; i++ {
		res, err := s.Allow(ctx, key, limit, window, burst, ratelimit.FixedWindow)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.Allowed {
			t.Errorf("request %d should be allowed", i)
		}
	}

	// Should block 6th request
	res, err := s.Allow(ctx, key, limit, window, burst, ratelimit.FixedWindow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Allowed {
		t.Error("request 6 should be blocked")
	}
}

// A denied sliding-window request must say when to retry (was always 0).
// Request sliding-window bị từ chối phải cho biết khi nào thử lại (trước đây luôn 0).
func TestSlidingWindowRetryAfter(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	window := time.Minute

	if _, err := s.Allow(ctx, "k", 1, window, 1, ratelimit.SlidingWindow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	res, err := s.Allow(ctx, "k", 1, window, 1, ratelimit.SlidingWindow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Allowed {
		t.Fatal("second request should be blocked")
	}
	if res.RetryAfter <= 0 || res.RetryAfter > window {
		t.Errorf("RetryAfter = %v, want in (0, %v]", res.RetryAfter, window)
	}
}
