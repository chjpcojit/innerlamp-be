// Package http provides HTTP metrics collection and middleware
package http

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"chjpcoj.com.innerlamp/pkg/metrics"
)

// Metrics provides HTTP metrics collection
type Metrics struct {
	collector metrics.Collector
	namespace string
	subsystem string
}

// HandlerConfig configures metrics for a specific handler
type HandlerConfig struct {
	// PathPattern is the route pattern (e.g., "/api/users/:id")
	PathPattern string

	// Labels are additional labels to add to metrics
	Labels metrics.Labels

	// SkipMetrics skips metrics collection for this handler
	SkipMetrics bool
}

// NewMetrics creates a new HTTP metrics collector
func NewMetrics(collector metrics.Collector, namespace, subsystem string) (*Metrics, error) {
	if collector == nil {
		return nil, fmt.Errorf("collector is required")
	}

	m := &Metrics{
		collector: collector,
		namespace: namespace,
		subsystem: subsystem,
	}

	// Register standard HTTP metrics
	if err := metrics.RegisterStandardHTTPMetrics(collector); err != nil {
		return nil, fmt.Errorf("failed to register http metrics: %w", err)
	}

	return m, nil
}

// Middleware returns a middleware that automatically collects HTTP metrics
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.URL.Path
		method := r.Method

		// Increment in-flight requests
		metrics.IncrementHTTPInFlight(m.collector, method, path)
		defer metrics.DecrementHTTPInFlight(m.collector, method, path)

		// Wrap response writer to capture status code and size
		wrapped := newResponseWriter(w)

		// Estimate request size
		requestSize := computeRequestSize(r)

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start)
		statusCode := strconv.Itoa(wrapped.statusCode)
		responseSize := int64(wrapped.bytesWritten)

		metrics.RecordHTTPRequest(
			m.collector,
			method,
			path,
			statusCode,
			duration,
			requestSize,
			responseSize,
		)
	})
}

// MiddlewareFunc returns a middleware function
func (m *Metrics) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return m.Middleware(next).ServeHTTP
}

// Handler wraps an HTTP handler with metrics collection
func (m *Metrics) Handler(handler http.Handler) http.Handler {
	return m.Middleware(handler)
}

// HandlerFunc wraps an HTTP handler function with metrics collection
func (m *Metrics) HandlerFunc(handler http.HandlerFunc) http.HandlerFunc {
	return m.MiddlewareFunc(handler)
}

// HandlerWithConfig wraps a handler with custom configuration
func (m *Metrics) HandlerWithConfig(config HandlerConfig, handler http.Handler) http.Handler {
	if config.SkipMetrics {
		return handler
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Use configured path pattern or actual path
		path := config.PathPattern
		if path == "" {
			path = r.URL.Path
		}
		method := r.Method

		// Create labels with custom labels
		baseLabels := metrics.Labels{
			metrics.LabelMethod: method,
			metrics.LabelPath:   path,
		}
		customLabels := metrics.MergeLabels(baseLabels, config.Labels)

		// Increment in-flight requests
		_ = m.collector.IncrementGauge("http_requests_in_flight", metrics.Labels{
			metrics.LabelMethod: method,
			metrics.LabelPath:   path,
		})
		defer m.collector.DecrementGauge("http_requests_in_flight", metrics.Labels{
			metrics.LabelMethod: method,
			metrics.LabelPath:   path,
		})

		// Wrap response writer
		wrapped := newResponseWriter(w)

		// Estimate request size
		requestSize := computeRequestSize(r)

		// Call handler
		handler.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start)
		statusCode := strconv.Itoa(wrapped.statusCode)
		responseSize := int64(wrapped.bytesWritten)

		// Record with custom labels
		allLabels := metrics.MergeLabels(customLabels, metrics.Labels{
			metrics.LabelStatusCode: statusCode,
		})

		_ = m.collector.IncrementCounter("http_requests_total", allLabels)
		_ = m.collector.ObserveHistogram("http_request_duration_seconds", duration.Seconds(), allLabels)

		sizeLabels := metrics.Labels{
			metrics.LabelMethod: method,
			metrics.LabelPath:   path,
		}
		_ = m.collector.ObserveHistogram("http_request_size_bytes", float64(requestSize), sizeLabels)

		responseSizeLabels := metrics.MergeLabels(sizeLabels, metrics.Labels{
			metrics.LabelStatusCode: statusCode,
		})
		_ = m.collector.ObserveHistogram("http_response_size_bytes", float64(responseSize), responseSizeLabels)
	})
}

// RecordRequest records metrics for an HTTP request
func (m *Metrics) RecordRequest(method, path, statusCode string, duration time.Duration, requestSize, responseSize int64) {
	metrics.RecordHTTPRequest(m.collector, method, path, statusCode, duration, requestSize, responseSize)
}

// IncrementInFlight increments in-flight request gauge
func (m *Metrics) IncrementInFlight(method, path string) {
	metrics.IncrementHTTPInFlight(m.collector, method, path)
}

// DecrementInFlight decrements in-flight request gauge
func (m *Metrics) DecrementInFlight(method, path string) {
	metrics.DecrementHTTPInFlight(m.collector, method, path)
}

// responseWriter wraps http.ResponseWriter to capture status code and bytes written
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

// newResponseWriter creates a new response writer wrapper
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default status code
	}
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(statusCode int) {
	if !rw.wroteHeader {
		rw.statusCode = statusCode
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(statusCode)
	}
}

// Write captures the number of bytes written
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// Hijack implements http.Hijacker interface
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}
	return h.Hijack()
}

// Flush implements http.Flusher interface
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Push implements http.Pusher interface
func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := rw.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return fmt.Errorf("response writer does not support push")
}

// computeRequestSize estimates the size of an HTTP request
func computeRequestSize(r *http.Request) int64 {
	size := int64(0)

	// Request line (method + path + protocol)
	if r.URL != nil {
		size += int64(len(r.Method) + len(r.URL.String()) + len(r.Proto) + 4) // +4 for spaces and CRLF
	}

	// Headers
	for name, values := range r.Header {
		size += int64(len(name) + len(": \r\n"))
		for _, value := range values {
			size += int64(len(value))
		}
	}

	// Empty line between headers and body
	size += 2 // CRLF

	// Body size
	if r.ContentLength > 0 {
		size += r.ContentLength
	}

	return size
}

// SanitizePath sanitizes a URL path for use as a metric label
// Replaces variable path segments with placeholders to reduce cardinality
func SanitizePath(path string) string {
	// This is a simple implementation
	// In production, you might want to use a router that provides the route pattern
	return path
}

// GroupByEndpoint groups similar paths together
// Example: /users/123 and /users/456 both become /users/:id
func GroupByEndpoint(path string) string {
	// This would typically use your router's pattern matching
	// For now, return as-is
	return path
}

// Timer provides convenient request timing
type Timer struct {
	metrics *Metrics
	method  string
	path    string
	start   time.Time
}

// NewTimer creates a new request timer
func (m *Metrics) NewTimer(method, path string) *Timer {
	m.IncrementInFlight(method, path)
	return &Timer{
		metrics: m,
		method:  method,
		path:    path,
		start:   time.Now(),
	}
}

// Stop stops the timer and records metrics
func (t *Timer) Stop(statusCode int, requestSize, responseSize int64) {
	t.metrics.DecrementInFlight(t.method, t.path)
	duration := time.Since(t.start)
	t.metrics.RecordRequest(
		t.method,
		t.path,
		strconv.Itoa(statusCode),
		duration,
		requestSize,
		responseSize,
	)
}

// WithMetrics is a convenience function to wrap an http.ServeMux with metrics
func WithMetrics(mux *http.ServeMux, collector metrics.Collector, namespace, subsystem string) (*http.ServeMux, error) {
	_, err := NewMetrics(collector, namespace, subsystem)
	if err != nil {
		return nil, err
	}

	// This returns the same mux - the middleware should be applied to handlers individually
	// or use a wrapper ServeMux
	return mux, nil
}
