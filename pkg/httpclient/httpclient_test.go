package httpclient

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(server *httptest.Server, cfg Config) *Client {
	cfg.InitialBackoff = time.Millisecond
	return NewWithHTTPClient(server.Client(), cfg)
}

func jsonBody(t *testing.T, url string, body []byte) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestClient_2xxSuccess_ReturnsResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	client := newTestClient(server, Config{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/ping", nil)
	resp, err := client.Do(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestClient_4xx_ReturnsImmediately_NoRetry(t *testing.T) {
	hits := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()
	client := newTestClient(server, Config{MaxRetries: 3})
	resp, err := client.Do(context.Background(), jsonBody(t, server.URL+"/bad", []byte(`{}`)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits), "4xx must not trigger retry")
	resp.Body.Close()
}

func TestClient_422_ReturnsImmediately(t *testing.T) {
	hits := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer server.Close()
	client := newTestClient(server, Config{MaxRetries: 3})
	resp, err := client.Do(context.Background(), jsonBody(t, server.URL+"/unprocessable", []byte(`{}`)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits))
	resp.Body.Close()
}

func TestClient_5xx_RetriesUpToMax_ReturnsError(t *testing.T) {
	hits := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := newTestClient(server, Config{MaxRetries: 3})
	resp, err := client.Do(context.Background(), jsonBody(t, server.URL+"/flaky", []byte(`{}`)))
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, int32(4), atomic.LoadInt32(&hits))
	assert.Contains(t, err.Error(), "503")
}

func TestClient_5xx_SucceedsOnSecondAttempt(t *testing.T) {
	hits := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client := newTestClient(server, Config{MaxRetries: 3})
	resp, err := client.Do(context.Background(), jsonBody(t, server.URL+"/recover", []byte(`{}`)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), atomic.LoadInt32(&hits))
	resp.Body.Close()
}

func TestClient_NoRetry_SingleAttempt_5xx(t *testing.T) {
	hits := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	client := newTestClient(server, Config{MaxRetries: 0})
	_, err := client.Do(context.Background(), jsonBody(t, server.URL+"/err", []byte(`{}`)))
	require.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits))
}

func TestClient_NetworkError_Retries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()
	client := newTestClient(server, Config{MaxRetries: 2})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, serverURL+"/gone", nil)
	_, err := client.Do(context.Background(), req)
	require.Error(t, err)
}

func TestClient_ContextCancelledDuringBackoff_ReturnsError(t *testing.T) {
	hits := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := NewWithHTTPClient(server.Client(), Config{
		MaxRetries:     3,
		InitialBackoff: 200 * time.Millisecond,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := client.Do(ctx, jsonBody(t, server.URL+"/slow", []byte(`{}`)))
	require.Error(t, err)
	assert.True(t,
		errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled),
		"expected context error, got: %v", err,
	)
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits))
}

func TestClient_BodyReplayedOnRetry(t *testing.T) {
	var (
		mu     sync.Mutex
		bodies [][]byte
		hits   int32
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		mu.Lock()
		bodies = append(bodies, buf.Bytes())
		mu.Unlock()
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client := newTestClient(server, Config{MaxRetries: 3})
	payload := []byte(`{"hello":"world"}`)
	resp, err := client.Do(context.Background(), jsonBody(t, server.URL+"/body", payload))
	require.NoError(t, err)
	resp.Body.Close()
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, bodies, 3)
	for i, b := range bodies {
		assert.Equal(t, payload, b, "attempt %d body mismatch", i+1)
	}
}

func TestCircuitBreaker_OpensAfterThresholdFailures_4xx(t *testing.T) {
	serverHits := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&serverHits, 1)
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer server.Close()
	client := newTestClient(server, Config{FailureThreshold: 5})
	for i := 0; i < 5; i++ {
		resp, err := client.Do(context.Background(), jsonBody(t, server.URL+"/fail", []byte(`{}`)))
		require.NoError(t, err)
		resp.Body.Close()
	}
	_, err := client.Do(context.Background(), jsonBody(t, server.URL+"/fail", []byte(`{}`)))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCircuitOpen)
	assert.Equal(t, int32(5), atomic.LoadInt32(&serverHits))
}

func TestCircuitBreaker_OpensAfterThresholdFailures_5xx(t *testing.T) {
	serverHits := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&serverHits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	client := newTestClient(server, Config{MaxRetries: 0, FailureThreshold: 3})
	for i := 0; i < 3; i++ {
		_, err := client.Do(context.Background(), jsonBody(t, server.URL+"/fail", []byte(`{}`)))
		require.Error(t, err)
	}
	_, err := client.Do(context.Background(), jsonBody(t, server.URL+"/fail", []byte(`{}`)))
	assert.ErrorIs(t, err, ErrCircuitOpen)
	assert.Equal(t, int32(3), atomic.LoadInt32(&serverHits))
}

func TestCircuitBreaker_ClosesAfterSuccessfulProbe(t *testing.T) {
	hits := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n <= 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client := newTestClient(server, Config{
		MaxRetries:       0,
		FailureThreshold: 3,
		ResetTimeout:     5 * time.Millisecond,
	})
	for i := 0; i < 3; i++ {
		client.Do(context.Background(), jsonBody(t, server.URL+"/probe", []byte(`{}`)))
	}
	_, err := client.Do(context.Background(), jsonBody(t, server.URL+"/probe", []byte(`{}`)))
	assert.ErrorIs(t, err, ErrCircuitOpen)
	time.Sleep(10 * time.Millisecond)
	resp, err := client.Do(context.Background(), jsonBody(t, server.URL+"/probe", []byte(`{}`)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
	resp2, err := client.Do(context.Background(), jsonBody(t, server.URL+"/probe", []byte(`{}`)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	resp2.Body.Close()
}

func TestCircuitBreaker_ReOpensIfProbeFailsInHalfOpen(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	client := newTestClient(server, Config{
		MaxRetries:       0,
		FailureThreshold: 3,
		ResetTimeout:     5 * time.Millisecond,
	})
	for i := 0; i < 3; i++ {
		client.Do(context.Background(), jsonBody(t, server.URL+"/fail", []byte(`{}`)))
	}
	time.Sleep(10 * time.Millisecond)
	_, err := client.Do(context.Background(), jsonBody(t, server.URL+"/fail", []byte(`{}`)))
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrCircuitOpen, "probe itself must not be rejected")
	_, err = client.Do(context.Background(), jsonBody(t, server.URL+"/fail", []byte(`{}`)))
	assert.ErrorIs(t, err, ErrCircuitOpen)
}

func TestCircuitBreaker_Disabled_NeverBlocks(t *testing.T) {
	hits := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	client := newTestClient(server, Config{MaxRetries: 0, FailureThreshold: 0})
	for i := 0; i < 10; i++ {
		client.Do(context.Background(), jsonBody(t, server.URL+"/fail", []byte(`{}`)))
	}
	assert.Equal(t, int32(10), atomic.LoadInt32(&hits))
}

func TestConfig_ApplyDefaults(t *testing.T) {
	cfg := Config{}
	cfg.applyDefaults()
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.Equal(t, time.Second, cfg.InitialBackoff)
	assert.Equal(t, 60*time.Second, cfg.ResetTimeout)
}

func TestConfig_CustomValues_NotOverridden(t *testing.T) {
	cfg := Config{
		Timeout:        10 * time.Second,
		InitialBackoff: 500 * time.Millisecond,
		ResetTimeout:   5 * time.Second,
	}
	cfg.applyDefaults()
	assert.Equal(t, 10*time.Second, cfg.Timeout)
	assert.Equal(t, 500*time.Millisecond, cfg.InitialBackoff)
	assert.Equal(t, 5*time.Second, cfg.ResetTimeout)
}
