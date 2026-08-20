package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"chjpcoj.com.innerlamp/pkg/logging"
	logctx "chjpcoj.com.innerlamp/pkg/logging/context"
)

// gRPC metadata keys for correlation ID (lowercase as per gRPC metadata spec)
var correlationIDMetadataKeys = []string{
	"x-correlation-id",
	"x-request-id",
	"x-trace-id",
	"correlation-id",
	"request-id",
}

// GRPCUnaryInterceptor returns a unary server interceptor that logs requests and attaches correlation ID to context
func GRPCUnaryInterceptor(logger logging.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Extract correlation ID from metadata
		correlationID := extractCorrelationIDFromMetadata(ctx)

		// If no correlation ID found, generate a new one
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Add correlation ID to context
		ctx = context.WithValue(ctx, CorrelationIDKey, correlationID)
		ctx = logctx.WithCorrelationID(ctx, CorrelationIDKey, correlationID)

		// Create logger with context and fields
		rpcLogger := logger.WithContext(ctx).WithFields(logging.Fields{
			"method": info.FullMethod,
			"system": "grpc",
		})

		// Inject logger into context
		ctx = logctx.WithLogger(ctx, rpcLogger)

		// Add correlation ID to outgoing metadata for client visibility
		if err := grpc.SetHeader(ctx, metadata.Pairs("x-correlation-id", correlationID)); err == nil {
			// Header set successfully
		}

		// Call handler
		resp, err := handler(ctx, req)

		duration := time.Since(start)
		code := status.Code(err)

		fields := logging.Fields{
			"duration_ms": duration.Milliseconds(),
			"code":        code.String(),
		}

		if err != nil {
			fields["error"] = err.Error()
			rpcLogger.WithFields(fields).Error("gRPC Request Failed")
		} else {
			rpcLogger.WithFields(fields).Info("gRPC Request Processed")
		}

		return resp, err
	}
}

// GRPCStreamInterceptor returns a stream server interceptor that logs requests and attaches correlation ID to context
func GRPCStreamInterceptor(logger logging.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx := ss.Context()

		// Extract correlation ID from metadata
		correlationID := extractCorrelationIDFromMetadata(ctx)

		// If no correlation ID found, generate a new one
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Add correlation ID to context
		ctx = context.WithValue(ctx, CorrelationIDKey, correlationID)
		ctx = logctx.WithCorrelationID(ctx, CorrelationIDKey, correlationID)

		// Create logger with context and fields
		rpcLogger := logger.WithContext(ctx).WithFields(logging.Fields{
			"method": info.FullMethod,
			"system": "grpc",
			"kind":   "stream",
		})

		// Inject logger into context
		ctx = logctx.WithLogger(ctx, rpcLogger)

		// Add correlation ID to outgoing metadata for client visibility
		if err := ss.SetHeader(metadata.Pairs("x-correlation-id", correlationID)); err == nil {
			// Header set successfully
		}

		// Wrap the server stream to override context
		wrapper := &wrappedStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// Call handler
		err := handler(srv, wrapper)

		duration := time.Since(start)
		code := status.Code(err)

		fields := logging.Fields{
			"duration_ms": duration.Milliseconds(),
			"code":        code.String(),
		}

		if err != nil {
			fields["error"] = err.Error()
			rpcLogger.WithFields(fields).Error("gRPC Stream Failed")
		} else {
			rpcLogger.WithFields(fields).Info("gRPC Stream Finished")
		}

		return err
	}
}

// wrappedStream wraps grpc.ServerStream to override the Context method
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapper's context which includes the correlation ID
func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

// extractCorrelationIDFromMetadata checks multiple standard metadata keys for a correlation ID
func extractCorrelationIDFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	for _, key := range correlationIDMetadataKeys {
		if values := md.Get(key); len(values) > 0 {
			if correlationID := strings.TrimSpace(values[0]); correlationID != "" {
				return correlationID
			}
		}
	}

	return ""
}
