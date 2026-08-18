package http

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewHTTPMiddleware creates a new HTTP middleware for tracing
func NewHTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Use otelhttp for automatic instrumentation
		// It captures headers, status codes, and latency automatically
		return otelhttp.NewHandler(
			next,
			"http_request", // Default operation name, can be overridden per route if needed
		)
	}
}

// WithRouteTag wraps a handler with a specific route name for better trace visibility
func WithRouteTag(handler http.Handler, route string) http.Handler {
	return otelhttp.NewHandler(handler, route)
}
