package store

import (
	"context"
	"sync"
	"time"

	"chjpcoj.com.innerlamp/pkg/ratelimit"
)

// MemoryStore implements an in-memory rate limit store
type MemoryStore struct {
	mu       sync.RWMutex
	limiters map[string]Limiter
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		limiters: make(map[string]Limiter),
	}
}

// Allow checks if the request is allowed
func (s *MemoryStore) Allow(ctx context.Context, key string, limit int64, window time.Duration, burst int64, algo ratelimit.Algorithm) (*ratelimit.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	l, exists := s.limiters[key]
	if !exists {
		l = s.createLimiter(algo)
		s.limiters[key] = l
	}

	return l.Allow(limit, window, burst), nil
}

func (s *MemoryStore) createLimiter(algo ratelimit.Algorithm) Limiter {
	switch algo {
	case ratelimit.TokenBucket:
		return newTokenBucket()
	case ratelimit.SlidingWindow:
		return newSlidingWindow()
	case ratelimit.FixedWindow:
		return newFixedWindow()
	case ratelimit.LeakyBucket:
		return newLeakyBucket()
	default:
		// Default to TokenBucket if unknown
		return newTokenBucket()
	}
}

// --- Algorithm Implementations ---

// tokenBucket implementation
type tokenBucket struct {
	tokens      float64
	lastUpdate  time.Time
	initialized bool
}

func newTokenBucket() *tokenBucket {
	return &tokenBucket{
		tokens:      0,
		lastUpdate:  time.Now(),
		initialized: false,
	}
}

func (tb *tokenBucket) Allow(limit int64, window time.Duration, burst int64) *ratelimit.Result {
	now := time.Now()

	if !tb.initialized {
		tb.tokens = float64(burst)
		tb.initialized = true
		tb.lastUpdate = now
	}

	// Calculate refill rate (tokens per second)
	rate := float64(limit) / window.Seconds()

	// Refill tokens
	elapsed := now.Sub(tb.lastUpdate).Seconds()
	tb.tokens += elapsed * rate
	if tb.tokens > float64(burst) {
		tb.tokens = float64(burst)
	}
	tb.lastUpdate = now

	allowed := false
	if tb.tokens >= 1 {
		tb.tokens -= 1
		allowed = true
	}

	remaining := int64(tb.tokens)
	resetTime := now.Add(time.Duration((1-tb.tokens)/rate) * time.Second)

	return &ratelimit.Result{
		Allowed:   allowed,
		Remaining: remaining,
		ResetTime: resetTime,
	}
}

// slidingWindow implementation
type slidingWindow struct {
	requests []time.Time
}

func newSlidingWindow() *slidingWindow {
	return &slidingWindow{
		requests: make([]time.Time, 0),
	}
}

func (sw *slidingWindow) Allow(limit int64, window time.Duration, burst int64) *ratelimit.Result {
	now := time.Now()
	windowStart := now.Add(-window)

	// Remove old requests
	validRequests := 0
	for _, t := range sw.requests {
		if t.After(windowStart) {
			sw.requests[validRequests] = t
			validRequests++
		}
	}
	sw.requests = sw.requests[:validRequests]

	allowed := int64(len(sw.requests)) < limit
	if allowed {
		sw.requests = append(sw.requests, now)
	}

	remaining := limit - int64(len(sw.requests))
	// Reset time is when the oldest request expires, or now + window if empty
	resetTime := now.Add(window)
	if len(sw.requests) > 0 {
		resetTime = sw.requests[0].Add(window)
	}

	return &ratelimit.Result{
		Allowed:   allowed,
		Remaining: remaining,
		ResetTime: resetTime,
	}
}

// fixedWindow implementation
type fixedWindow struct {
	count       int64
	windowStart time.Time
}

func newFixedWindow() *fixedWindow {
	return &fixedWindow{
		count:       0,
		windowStart: time.Now(),
	}
}

func (fw *fixedWindow) Allow(limit int64, window time.Duration, burst int64) *ratelimit.Result {
	now := time.Now()

	// Check if window has expired
	if now.After(fw.windowStart.Add(window)) {
		fw.count = 0
		fw.windowStart = now
	}

	allowed := fw.count < limit
	if allowed {
		fw.count++
	}

	remaining := limit - fw.count
	resetTime := fw.windowStart.Add(window)

	return &ratelimit.Result{
		Allowed:   allowed,
		Remaining: remaining,
		ResetTime: resetTime,
	}
}

// leakyBucket implementation (simplified as fixed rate output)
// For rate limiting, Leaky Bucket is often similar to Token Bucket but enforces a strict output rate.
// Here we implement it as a queue where requests are processed at a fixed rate.
// But since this is Allow(), we just check if the bucket has space and if the last request was long enough ago?
// Or we can implement it as a "GCRA" (Generic Cell Rate Algorithm).
// For simplicity, let's implement a basic version where we track the "next allowed time".
type leakyBucket struct {
	nextAllowed time.Time
}

func newLeakyBucket() *leakyBucket {
	return &leakyBucket{
		nextAllowed: time.Now(),
	}
}

func (lb *leakyBucket) Allow(limit int64, window time.Duration, burst int64) *ratelimit.Result {
	now := time.Now()

	// Interval between requests
	interval := window.Nanoseconds() / limit

	if now.Before(lb.nextAllowed) {
		// Too soon
		return &ratelimit.Result{
			Allowed:    false,
			Remaining:  0,
			ResetTime:  lb.nextAllowed,
			RetryAfter: lb.nextAllowed.Sub(now),
		}
	}

	// Allowed
	lb.nextAllowed = now.Add(time.Duration(interval))

	return &ratelimit.Result{
		Allowed:   true,
		Remaining: 1, // Not really applicable for leaky bucket in this simple view
		ResetTime: lb.nextAllowed,
	}
}
