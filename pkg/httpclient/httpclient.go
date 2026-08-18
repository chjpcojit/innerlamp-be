package httpclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ErrCircuitOpen is returned when a request is blocked because the circuit breaker is open.
var ErrCircuitOpen = errors.New("httpclient: circuit breaker open")

// cbState represents the three states of a circuit breaker.
type cbState int8

const (
	cbClosed   cbState = iota // normal — all requests allowed
	cbOpen                    // tripped — all requests rejected immediately
	cbHalfOpen                // probe — one request allowed to test recovery
)

// circuitBreaker tracks consecutive failures and blocks requests when open.
// It is safe for concurrent use.
type circuitBreaker struct {
	mu           sync.Mutex
	state        cbState
	failures     int
	threshold    int
	openedAt     time.Time
	resetTimeout time.Duration
}

func newCircuitBreaker(threshold int, resetTimeout time.Duration) *circuitBreaker {
	return &circuitBreaker{threshold: threshold, resetTimeout: resetTimeout}
}

// allow returns true if the caller may proceed with the request.
func (cb *circuitBreaker) allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == cbOpen {
		if time.Since(cb.openedAt) >= cb.resetTimeout {
			cb.state = cbHalfOpen
			return true
		}
		return false
	}
	return true
}

// record updates the circuit breaker state based on the outcome of an attempt.
func (cb *circuitBreaker) record(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if success {
		cb.failures = 0
		cb.state = cbClosed
		return
	}
	cb.failures++
	if cb.state == cbHalfOpen || cb.failures >= cb.threshold {
		cb.state = cbOpen
		cb.openedAt = time.Now()
	}
}

// Config configures the Client behaviour.
type Config struct {
	// Timeout is the per-request HTTP timeout. Default: 30s.
	Timeout time.Duration

	// MaxRetries is the number of additional attempts after the initial request.
	// 0 means a single attempt with no retry. Default: 0.
	MaxRetries int

	// InitialBackoff is the base duration for exponential backoff between retries.
	// Each subsequent retry waits twice the previous duration. Default: 1s.
	InitialBackoff time.Duration

	// FailureThreshold is the number of consecutive failures (network errors, 4xx,
	// or 5xx) needed to open the circuit breaker. 0 disables circuit breaking.
	// Default: 0 (disabled).
	FailureThreshold int

	// ResetTimeout is how long the circuit remains open before a single probe
	// request is allowed (half-open state). Default: 60s.
	ResetTimeout time.Duration
}

func (c *Config) applyDefaults() {
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	if c.InitialBackoff == 0 {
		c.InitialBackoff = time.Second
	}
	if c.ResetTimeout == 0 {
		c.ResetTimeout = 60 * time.Second
	}
}

// Client is a reusable HTTP client with configurable retry and circuit breaker.
// It is safe for concurrent use.
type Client struct {
	http *http.Client
	cfg  Config
	cb   *circuitBreaker // nil when FailureThreshold == 0
}

// New creates a Client using the provided Config.
// An internal *http.Client is created with cfg.Timeout applied.
func New(cfg Config) *Client {
	cfg.applyDefaults()
	c := &Client{
		http: &http.Client{Timeout: cfg.Timeout},
		cfg:  cfg,
	}
	if cfg.FailureThreshold > 0 {
		c.cb = newCircuitBreaker(cfg.FailureThreshold, cfg.ResetTimeout)
	}
	return c
}

// NewWithHTTPClient creates a Client backed by the provided *http.Client.
// Intended for testing: pass an httptest.Server's client to target the test server.
func NewWithHTTPClient(httpClient *http.Client, cfg Config) *Client {
	cfg.applyDefaults()
	c := &Client{
		http: httpClient,
		cfg:  cfg,
	}
	if cfg.FailureThreshold > 0 {
		c.cb = newCircuitBreaker(cfg.FailureThreshold, cfg.ResetTimeout)
	}
	return c
}

// Do executes the HTTP request, applying the configured retry and circuit breaker policies.
//
// Retry behaviour:
//   - Retries on network/transport errors and 5xx responses.
//   - 4xx responses are returned immediately without retry.
//   - Backoff between retries starts at InitialBackoff and doubles on each attempt.
//   - The request body must be replayable across retries (req.GetBody must be non-nil or
//     req.Body must be nil). Requests built with bytes.NewReader satisfy this automatically
//     because the standard library sets GetBody for *bytes.Reader bodies.
//
// Circuit breaker behaviour (when FailureThreshold > 0):
//   - Opens after FailureThreshold consecutive failures (network errors, 4xx, 5xx).
//   - When open, returns ErrCircuitOpen without making a network call.
//   - Transitions to half-open after ResetTimeout; the next request acts as a probe.
//   - A successful probe closes the circuit; a failed probe re-opens it.
//
// The caller is responsible for closing the response body on non-error returns.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if c.cb != nil && !c.cb.allow() {
		return nil, ErrCircuitOpen
	}

	var (
		lastErr error
		backoff = c.cfg.InitialBackoff
	)

	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			// Replay the request body before the backoff wait so that any
			// GetBody error surfaces before consuming the wait duration.
			if req.Body != nil {
				if req.GetBody == nil {
					return nil, fmt.Errorf("httpclient: body not replayable after attempt %d: %w", attempt, lastErr)
				}
				body, err := req.GetBody()
				if err != nil {
					return nil, fmt.Errorf("httpclient: recreate request body: %w", err)
				}
				req.Body = body
			}

			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("httpclient: retry cancelled: %w", ctx.Err())
			case <-time.After(backoff):
				backoff *= 2
			}
		}

		resp, err := c.http.Do(req) // #nosec
		if err != nil {
			// Network/transport error — schedule a retry.
			lastErr = err
			c.recordFailure()
			continue
		}

		if resp.StatusCode >= 500 {
			// Transient server error — close body and schedule a retry.
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("httpclient: server returned %d", resp.StatusCode)
			c.recordFailure()
			continue
		}

		if resp.StatusCode >= 400 {
			// Client error — return immediately without retry, but still count
			// towards the circuit breaker failure threshold.
			c.recordFailure()
			return resp, nil
		}

		// 2xx / 3xx — success.
		c.recordSuccess()
		return resp, nil
	}

	return nil, lastErr
}

func (c *Client) recordSuccess() {
	if c.cb != nil {
		c.cb.record(true)
	}
}

func (c *Client) recordFailure() {
	if c.cb != nil {
		c.cb.record(false)
	}
}
