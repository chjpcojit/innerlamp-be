package middleware

import (
	"context"
	"fmt"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryInterceptor returns a unary server interceptor that recovers from panics
func RecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic and stack trace
				stack := debug.Stack()
				err = status.Errorf(codes.Internal, "panic recovered: %v\n%s", r, stack)
			}
		}()

		return handler(ctx, req)
	}
}

// StreamRecoveryInterceptor returns a stream server interceptor that recovers from panics
func StreamRecoveryInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic and stack trace
				stack := debug.Stack()
				err = status.Errorf(codes.Internal, "panic recovered: %v\n%s", r, stack)
			}
		}()

		return handler(srv, ss)
	}
}

// RecoveryWithLoggerInterceptor returns a unary server interceptor that recovers from panics and logs them
func RecoveryWithLoggerInterceptor(logFunc func(msg string, keysAndValues ...interface{})) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic and stack trace
				stack := debug.Stack()
				if logFunc != nil {
					logFunc("panic recovered in gRPC unary handler",
						"method", info.FullMethod,
						"panic", fmt.Sprintf("%v", r),
						"stack", string(stack),
					)
				}
				err = status.Errorf(codes.Internal, "panic recovered: %v", r)
			}
		}()

		return handler(ctx, req)
	}
}

// StreamRecoveryWithLoggerInterceptor returns a stream server interceptor that recovers from panics and logs them
func StreamRecoveryWithLoggerInterceptor(logFunc func(msg string, keysAndValues ...interface{})) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic and stack trace
				stack := debug.Stack()
				if logFunc != nil {
					logFunc("panic recovered in gRPC stream handler",
						"method", info.FullMethod,
						"panic", fmt.Sprintf("%v", r),
						"stack", string(stack),
					)
				}
				err = status.Errorf(codes.Internal, "panic recovered: %v", r)
			}
		}()

		return handler(srv, ss)
	}
}
