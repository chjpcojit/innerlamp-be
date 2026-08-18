package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"base_project/pkg/logging"
	logctx "base_project/pkg/logging/context"
)

// Common correlation ID header names to check (in order of priority)
var correlationIDHeaders = []string{
	"X-Correlation-ID",
	"X-Correlation-Id",
	"X-Request-ID",
	"X-Request-Id",
	"X-Trace-ID",
	"X-Trace-Id",
	"Correlation-ID",
	"Correlation-Id",
	"Request-ID",
	"Request-Id",
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// HTTPLogger creates a middleware that logs HTTP requests and attaches correlation ID to context
func HTTPLogger(logger logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Extract correlation ID from headers
			correlationID := extractCorrelationIDFromHeaders(r)

			// If no correlation ID found, generate a new one
			if correlationID == "" {
				correlationID = uuid.New().String()
			}

			// Add correlation ID to context
			ctx := r.Context()
			ctx = context.WithValue(ctx, CorrelationIDKey, correlationID)
			ctx = logctx.WithCorrelationID(ctx, CorrelationIDKey, correlationID)

			// Create a child logger with context and fields
			reqLogger := logger.WithContext(ctx).WithFields(logging.Fields{
				"method": r.Method,
				"path":   r.URL.Path,
				"ip":     r.RemoteAddr,
			})

			// Inject logger into context so handlers can use it
			ctx = logctx.WithLogger(ctx, reqLogger)

			// Wrap response writer to capture status
			rw := &responseWriter{ResponseWriter: w}

			// Add correlation ID to response header for client visibility
			rw.Header().Set("X-Correlation-ID", correlationID)

			// Continue with updated context
			next.ServeHTTP(rw, r.WithContext(ctx))

			duration := time.Since(start)

			// Log request completion
			reqLogger.Info("HTTP Request",
				"status", rw.status,
				"duration_ms", duration.Milliseconds(),
				"size", rw.size,
				"user_agent", r.UserAgent(),
			)
		})
	}
}

// extractCorrelationIDFromHeaders checks multiple standard headers for a correlation ID
func extractCorrelationIDFromHeaders(r *http.Request) string {
	for _, header := range correlationIDHeaders {
		if value := strings.TrimSpace(r.Header.Get(header)); value != "" {
			return value
		}
	}
	return ""
}
