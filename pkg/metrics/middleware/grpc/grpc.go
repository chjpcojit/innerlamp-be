// Package grpc provides gRPC metrics collection and interceptors
package grpc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"chjpcoj.com.innerlamp/pkg/metrics"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Metrics provides gRPC metrics collection
type Metrics struct {
	collector metrics.Collector
	namespace string
	subsystem string
}

// NewMetrics creates a new gRPC metrics collector
func NewMetrics(collector metrics.Collector, namespace, subsystem string) (*Metrics, error) {
	if collector == nil {
		return nil, fmt.Errorf("collector is required")
	}

	m := &Metrics{
		collector: collector,
		namespace: namespace,
		subsystem: subsystem,
	}

	// Register standard gRPC metrics
	if err := metrics.RegisterStandardGRPCMetrics(collector); err != nil {
		return nil, fmt.Errorf("failed to register grpc metrics: %w", err)
	}

	return m, nil
}

// UnaryServerInterceptor returns a unary server interceptor for metrics collection
func (m *Metrics) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Parse method name
		service, method := splitMethodName(info.FullMethod)
		grpcType := "unary"

		// Increment in-flight requests
		metrics.IncrementGRPCInFlight(m.collector, service, method, grpcType)
		defer metrics.DecrementGRPCInFlight(m.collector, service, method, grpcType)

		// Call handler
		resp, err := handler(ctx, req)

		// Record metrics
		duration := time.Since(start)
		code := status.Code(err)
		codeStr := code.String()

		metrics.RecordGRPCRequest(m.collector, service, method, grpcType, codeStr, duration)

		return resp, err
	}
}

// StreamServerInterceptor returns a stream server interceptor for metrics collection
func (m *Metrics) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		// Parse method name
		service, method := splitMethodName(info.FullMethod)
		grpcType := getStreamType(info)

		// Increment in-flight requests
		metrics.IncrementGRPCInFlight(m.collector, service, method, grpcType)
		defer metrics.DecrementGRPCInFlight(m.collector, service, method, grpcType)

		// Wrap stream to track messages
		wrapped := newMonitoredServerStream(ss, m.collector, service, method)

		// Call handler
		err := handler(srv, wrapped)

		// Record metrics
		duration := time.Since(start)
		code := status.Code(err)
		codeStr := code.String()

		metrics.RecordGRPCRequest(m.collector, service, method, grpcType, codeStr, duration)

		return err
	}
}

// UnaryClientInterceptor returns a unary client interceptor for metrics collection
func (m *Metrics) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()

		// Parse method name
		service, methodName := splitMethodName(method)
		grpcType := "unary"

		// Increment in-flight requests
		metrics.IncrementGRPCInFlight(m.collector, service, methodName, grpcType)
		defer metrics.DecrementGRPCInFlight(m.collector, service, methodName, grpcType)

		// Call invoker
		err := invoker(ctx, method, req, reply, cc, opts...)

		// Record metrics
		duration := time.Since(start)
		code := status.Code(err)
		codeStr := code.String()

		metrics.RecordGRPCRequest(m.collector, service, methodName, grpcType, codeStr, duration)

		return err
	}
}

// StreamClientInterceptor returns a stream client interceptor for metrics collection
func (m *Metrics) StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		start := time.Now()

		// Parse method name
		service, methodName := splitMethodName(method)
		grpcType := getClientStreamType(desc)

		// Increment in-flight requests
		metrics.IncrementGRPCInFlight(m.collector, service, methodName, grpcType)

		// Call streamer
		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			metrics.DecrementGRPCInFlight(m.collector, service, methodName, grpcType)
			duration := time.Since(start)
			code := status.Code(err)
			codeStr := code.String()
			metrics.RecordGRPCRequest(m.collector, service, methodName, grpcType, codeStr, duration)
			return nil, err
		}

		// Wrap stream to track messages
		wrapped := newMonitoredClientStream(clientStream, m.collector, service, methodName, grpcType, start)

		return wrapped, nil
	}
}

// RecordRequest records metrics for a gRPC request
func (m *Metrics) RecordRequest(service, method, grpcType, code string, duration time.Duration) {
	metrics.RecordGRPCRequest(m.collector, service, method, grpcType, code, duration)
}

// RecordStreamMessage records a stream message metric
func (m *Metrics) RecordStreamMessage(service, method, direction string) {
	metrics.RecordGRPCStreamMessage(m.collector, service, method, direction)
}

// IncrementInFlight increments in-flight request gauge
func (m *Metrics) IncrementInFlight(service, method, grpcType string) {
	metrics.IncrementGRPCInFlight(m.collector, service, method, grpcType)
}

// DecrementInFlight decrements in-flight request gauge
func (m *Metrics) DecrementInFlight(service, method, grpcType string) {
	metrics.DecrementGRPCInFlight(m.collector, service, method, grpcType)
}

// monitoredServerStream wraps grpc.ServerStream to track messages
type monitoredServerStream struct {
	grpc.ServerStream
	collector metrics.Collector
	service   string
	method    string
}

func newMonitoredServerStream(ss grpc.ServerStream, collector metrics.Collector, service, method string) *monitoredServerStream {
	return &monitoredServerStream{
		ServerStream: ss,
		collector:    collector,
		service:      service,
		method:       method,
	}
}

func (s *monitoredServerStream) SendMsg(m interface{}) error {
	err := s.ServerStream.SendMsg(m)
	if err == nil {
		metrics.RecordGRPCStreamMessage(s.collector, s.service, s.method, "sent")
	}
	return err
}

func (s *monitoredServerStream) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if err == nil {
		metrics.RecordGRPCStreamMessage(s.collector, s.service, s.method, "received")
	}
	return err
}

// monitoredClientStream wraps grpc.ClientStream to track messages
type monitoredClientStream struct {
	grpc.ClientStream
	collector metrics.Collector
	service   string
	method    string
	grpcType  string
	start     time.Time
}

func newMonitoredClientStream(cs grpc.ClientStream, collector metrics.Collector, service, method, grpcType string, start time.Time) *monitoredClientStream {
	return &monitoredClientStream{
		ClientStream: cs,
		collector:    collector,
		service:      service,
		method:       method,
		grpcType:     grpcType,
		start:        start,
	}
}

func (s *monitoredClientStream) SendMsg(m interface{}) error {
	err := s.ClientStream.SendMsg(m)
	if err == nil {
		metrics.RecordGRPCStreamMessage(s.collector, s.service, s.method, "sent")
	}
	return err
}

func (s *monitoredClientStream) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m)
	if err == nil {
		metrics.RecordGRPCStreamMessage(s.collector, s.service, s.method, "received")
	}
	return err
}

func (s *monitoredClientStream) CloseSend() error {
	err := s.ClientStream.CloseSend()

	// Record metrics when stream closes
	metrics.DecrementGRPCInFlight(s.collector, s.service, s.method, s.grpcType)
	duration := time.Since(s.start)
	code := status.Code(err)
	codeStr := code.String()
	metrics.RecordGRPCRequest(s.collector, s.service, s.method, s.grpcType, codeStr, duration)

	return err
}

// splitMethodName splits a full method name into service and method
// Example: /package.Service/Method -> (package.Service, Method)
func splitMethodName(fullMethod string) (string, string) {
	fullMethod = strings.TrimPrefix(fullMethod, "/")
	parts := strings.Split(fullMethod, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "unknown", fullMethod
}

// getStreamType determines the stream type from StreamServerInfo
func getStreamType(info *grpc.StreamServerInfo) string {
	if info.IsClientStream && info.IsServerStream {
		return "bidi_stream"
	} else if info.IsClientStream {
		return "client_stream"
	} else if info.IsServerStream {
		return "server_stream"
	}
	return "unary"
}

// getClientStreamType determines the stream type from StreamDesc
func getClientStreamType(desc *grpc.StreamDesc) string {
	if desc.ClientStreams && desc.ServerStreams {
		return "bidi_stream"
	} else if desc.ClientStreams {
		return "client_stream"
	} else if desc.ServerStreams {
		return "server_stream"
	}
	return "unary"
}

// codeToString converts a gRPC status code to string
func codeToString(code codes.Code) string {
	return code.String()
}

// Timer provides convenient request timing for gRPC calls
type Timer struct {
	metrics  *Metrics
	service  string
	method   string
	grpcType string
	start    time.Time
}

// NewTimer creates a new gRPC request timer
func (m *Metrics) NewTimer(service, method, grpcType string) *Timer {
	m.IncrementInFlight(service, method, grpcType)
	return &Timer{
		metrics:  m,
		service:  service,
		method:   method,
		grpcType: grpcType,
		start:    time.Now(),
	}
}

// Stop stops the timer and records metrics
func (t *Timer) Stop(code codes.Code) {
	t.metrics.DecrementInFlight(t.service, t.method, t.grpcType)
	duration := time.Since(t.start)
	t.metrics.RecordRequest(t.service, t.method, t.grpcType, code.String(), duration)
}

// StopWithError stops the timer and records metrics, extracting code from error
func (t *Timer) StopWithError(err error) {
	code := status.Code(err)
	t.Stop(code)
}
