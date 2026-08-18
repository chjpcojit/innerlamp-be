package grpc

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

// NewServerStatsHandler creates a new gRPC server stats handler for tracing
// Note: Modern otelgrpc uses StatsHandler instead of Interceptors
func NewServerStatsHandler() stats.Handler {
	return otelgrpc.NewServerHandler()
}

// NewClientStatsHandler creates a new gRPC client stats handler for tracing
func NewClientStatsHandler() stats.Handler {
	return otelgrpc.NewClientHandler()
}

// NewClientDialOption creates a new gRPC client dial option for tracing
func NewClientDialOption() grpc.DialOption {
	return grpc.WithStatsHandler(otelgrpc.NewClientHandler())
}
