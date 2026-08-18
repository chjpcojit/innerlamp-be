package middleware

import (
	"context"
)

// CorrelationIDKey is the context key for storing correlation ID
const CorrelationIDKey = "correlation_id"

// GetCorrelationID retrieves the correlation ID from the context.
// Returns an empty string if no correlation ID is found.
func GetCorrelationID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if val := ctx.Value(CorrelationIDKey); val != nil {
		if correlationID, ok := val.(string); ok {
			return correlationID
		}
	}

	return ""
}
