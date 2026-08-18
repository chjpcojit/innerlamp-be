package context

import (
	"context"

	"base_project/pkg/logging"
)

type contextKey struct{}

var loggerKey = contextKey{}

// WithLogger returns a new context with the provided logger.
func WithLogger(ctx context.Context, logger logging.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves the logger from the context.
// If no logger is found, it returns a default no-op logger or the global logger if configured (not implemented here, so nil check required by caller or return default).
// For safety, we'll return a default logger if none is found to avoid panics.
func FromContext(ctx context.Context) logging.Logger {
	if logger, ok := ctx.Value(loggerKey).(logging.Logger); ok {
		return logger
	}
	// Return a default logger that writes to stdout with Info level if none found
	// In a real app, you might want a global default.
	return logging.New(logging.Config{
		Level: logging.InfoLevel,
	})
}

// WithCorrelationID sets the correlation ID in the context.
// This is useful if you want to manually set it before creating the logger.
// However, the logger itself extracts it from context using the key defined in Config.
// So this helper might just set a standard key.
func WithCorrelationID(ctx context.Context, key string, id string) context.Context {
	return context.WithValue(ctx, key, id)
}
