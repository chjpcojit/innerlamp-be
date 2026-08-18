package middleware

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"base_project/pkg/ratelimit"
)

const (
	// ContextKeyGRPCMethod is the key for the gRPC method in the context
	ContextKeyGRPCMethod contextKey = "ratelimit_grpc_method"
)

// GRPCMiddleware provides gRPC interceptors for rate limiting
type GRPCMiddleware struct {
	limiter *Limiter
}

// NewGRPCMiddleware creates a new gRPC middleware
func NewGRPCMiddleware(config ratelimit.Config) (*GRPCMiddleware, error) {
	limiter, err := NewLimiter(config)
	if err != nil {
		return nil, err
	}
	return &GRPCMiddleware{
		limiter: limiter,
	}, nil
}

// UnaryInterceptor returns a UnaryServerInterceptor
func (m *GRPCMiddleware) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Inject method name into context
		ctx = context.WithValue(ctx, ContextKeyGRPCMethod, info.FullMethod)

		res, err := m.limiter.Allow(ctx)
		if err != nil {
			// Fail open on error
			return handler(ctx, req)
		}

		if !res.Allowed {
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded, retry after %s", res.RetryAfter)
		}

		return handler(ctx, req)
	}
}

// StreamInterceptor returns a StreamServerInterceptor
func (m *GRPCMiddleware) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		// Inject method name into context
		ctx = context.WithValue(ctx, ContextKeyGRPCMethod, info.FullMethod)

		// Wrap ServerStream to pass the new context
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		res, err := m.limiter.Allow(ctx)
		if err != nil {
			// Fail open on error
			return handler(srv, wrappedStream)
		}

		if !res.Allowed {
			return status.Errorf(codes.ResourceExhausted, "rate limit exceeded, retry after %s", res.RetryAfter)
		}

		return handler(srv, wrappedStream)
	}
}

// wrappedServerStream wraps grpc.ServerStream to modify the context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
