package logging

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
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

// gRPC metadata keys for correlation ID (lowercase as per gRPC metadata spec)
var correlationIDMetadataKeys = []string{
	"x-correlation-id",
	"x-request-id",
	"x-trace-id",
	"correlation-id",
	"request-id",
}

// CorrelationIDKey is the context key used to store correlation ID
type correlationIDKeyType string

const CorrelationIDKey correlationIDKeyType = "correlation_id"

// GetOrCreateCorrelationIDFromHTTP extracts correlation ID from HTTP request headers.
// If no correlation ID is found, generates a new UUID.
// Returns the correlation ID string.
func GetOrCreateCorrelationIDFromHTTP(r *http.Request) string {
	// Try to extract from headers
	for _, header := range correlationIDHeaders {
		if value := strings.TrimSpace(r.Header.Get(header)); value != "" {
			return value
		}
	}

	// No correlation ID found, generate new one
	return uuid.New().String()
}

// GetOrCreateCorrelationIDFromGRPC extracts correlation ID from gRPC metadata.
// If no correlation ID is found, generates a new UUID.
// Returns the correlation ID string.
func GetOrCreateCorrelationIDFromGRPC(ctx context.Context) string {
	// Try to extract from incoming metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for _, key := range correlationIDMetadataKeys {
			if values := md.Get(key); len(values) > 0 && strings.TrimSpace(values[0]) != "" {
				return strings.TrimSpace(values[0])
			}
		}
	}

	// No correlation ID found, generate new one
	return uuid.New().String()
}

// WithCorrelationID adds correlation ID to context using the standard key.
// This allows the logger to extract it automatically.
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// GetCorrelationIDFromContext retrieves correlation ID from context.
// Returns empty string if not found.
func GetCorrelationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if val := ctx.Value(CorrelationIDKey); val != nil {
		if cid, ok := val.(string); ok {
			return cid
		}
	}

	return ""
}

// AttachCorrelationIDToHTTPContext extracts or creates correlation ID from HTTP request
// and attaches it to the request context. Returns the new context and the correlation ID.
func AttachCorrelationIDToHTTPContext(r *http.Request) (context.Context, string) {
	correlationID := GetOrCreateCorrelationIDFromHTTP(r)
	ctx := WithCorrelationID(r.Context(), correlationID)
	return ctx, correlationID
}

// AttachCorrelationIDToGRPCContext extracts or creates correlation ID from gRPC context
// and attaches it to the context. Returns the new context and the correlation ID.
func AttachCorrelationIDToGRPCContext(ctx context.Context) (context.Context, string) {
	correlationID := GetOrCreateCorrelationIDFromGRPC(ctx)
	ctx = WithCorrelationID(ctx, correlationID)
	return ctx, correlationID
}
