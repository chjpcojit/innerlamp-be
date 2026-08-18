# Tracing Middleware

This directory contains HTTP and gRPC middleware implementations for distributed tracing using OpenTelemetry.

## Overview

The middleware packages provide transparent distributed tracing for HTTP and gRPC services without requiring manual instrumentation in your handlers or service methods.

## Structure

```
middleware/
├── http/          # HTTP middleware for REST APIs
│   └── http.go    # HTTP tracing middleware
└── grpc/          # gRPC interceptors for RPC services
    └── grpc.go    # gRPC tracing stats handlers
```

## HTTP Middleware

The HTTP middleware automatically creates spans for HTTP requests and propagates trace context.

### Features

- Automatic span creation for each HTTP request
- Request/response header propagation (W3C Trace Context)
- Status code and error tracking
- Request method and path capture
- Route-based span naming
- Automatic context propagation

### Quick Start

```go
import (
    "base_project/pkg/tracing"
    tracingHTTP "base_project/pkg/tracing/middleware/http"
    "base_project/pkg/tracing/types"
)

// Initialize tracer
config := types.Config{
    ServiceName:   "my-service",
    Environment:   "production",
    ExporterType:  types.ExporterTypeGCP,
    SamplingRate:  1.0,
    GCP: types.GCPConfig{
        ProjectID: "my-gcp-project",
    },
}

tracer, err := tracing.InitTracer(config)
if err != nil {
    log.Fatal(err)
}
defer tracer.Shutdown(context.Background())

// Create HTTP middleware
tracingMiddleware := tracingHTTP.NewHTTPMiddleware(tracer)

// Setup routes
mux := http.NewServeMux()
mux.HandleFunc("/api/users", handleUsers)

// Wrap with tracing middleware
handler := tracingMiddleware(mux)
http.ListenAndServe(":8080", handler)
```

### Route Tags

For better trace visibility, use route tags to name spans with route patterns:

```go
import tracingHTTP "base_project/pkg/tracing/middleware/http"

// Wrap specific handler with route tag
userHandler := tracingHTTP.WithRouteTag(http.HandlerFunc(handleUser), "/api/users/:id")
mux.Handle("/api/users/", userHandler)
```

### Span Attributes

The HTTP middleware automatically captures:
- `http.method` - HTTP method (GET, POST, etc.)
- `http.url` - Request URL
- `http.status_code` - Response status code
- `http.user_agent` - User agent header
- `http.route` - Route pattern (if using WithRouteTag)
- `http.scheme` - HTTP scheme (http/https)
- `http.host` - Host header

## gRPC Middleware

The gRPC middleware uses OpenTelemetry stats handlers for automatic tracing of RPC calls.

### Features

- Automatic span creation for each RPC
- Unary and streaming RPC support
- Metadata propagation (W3C Trace Context)
- Status code and error tracking
- Method name capture
- Client and server instrumentation

### Quick Start - Server

```go
import (
    "base_project/pkg/tracing"
    tracingGRPC "base_project/pkg/tracing/middleware/grpc"
    "base_project/pkg/tracing/types"
    "google.golang.org/grpc"
)

// Initialize tracer
tracer, err := tracing.InitTracer(config)
if err != nil {
    log.Fatal(err)
}
defer tracer.Shutdown(context.Background())

// Create gRPC server with tracing stats handler
server := grpc.NewServer(
    grpc.StatsHandler(tracingGRPC.NewServerStatsHandler(tracer)),
)

// Register services
pb.RegisterUserServiceServer(server, &userService{})

// Start server
lis, _ := net.Listen("tcp", ":9090")
server.Serve(lis)
```

### Quick Start - Client

```go
import (
    tracingGRPC "base_project/pkg/tracing/middleware/grpc"
)

// Create gRPC client connection with tracing
conn, err := grpc.Dial(
    "localhost:9090",
    grpc.WithInsecure(),
    tracingGRPC.NewClientDialOption(tracer),
)
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

// Use client normally - tracing is automatic
client := pb.NewUserServiceClient(conn)
response, err := client.GetUser(ctx, &pb.GetUserRequest{Id: "123"})
```

### Span Attributes

The gRPC middleware automatically captures:
- `rpc.system` - RPC system (grpc)
- `rpc.service` - Service name
- `rpc.method` - Method name
- `rpc.grpc.status_code` - gRPC status code
- Message sent/received events

## Advanced Usage

### Manual Span Creation

Create custom spans within your handlers:

```go
func handleUser(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    tracer := otel.Tracer("my-service")
    
    // Create child span
    ctx, span := tracer.Start(ctx, "fetch-user-from-db")
    defer span.End()
    
    // Your business logic
    user, err := db.GetUser(ctx, userID)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return
    }
    
    span.SetAttributes(
        attribute.String("user.id", user.ID),
        attribute.String("user.email", user.Email),
    )
}
```

### Span Events

Add events to spans for important operations:

```go
span.AddEvent("user.validation.started")
if err := validateUser(user); err != nil {
    span.AddEvent("user.validation.failed", trace.WithAttributes(
        attribute.String("error", err.Error()),
    ))
}
span.AddEvent("user.validation.completed")
```

### Span Links

Link spans across different traces:

```go
ctx, span := tracer.Start(ctx, "process-batch",
    trace.WithLinks(trace.Link{
        SpanContext: parentSpanContext,
    }),
)
```

## Integration

### With Logging

Combine tracing with logging to correlate logs with traces:

```go
import (
    "base_project/pkg/logging"
    "go.opentelemetry.io/otel/trace"
)

func handleUser(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    span := trace.SpanFromContext(ctx)
    
    logger.WithContext(ctx).WithFields(logging.Fields{
        "trace_id": span.SpanContext().TraceID().String(),
        "span_id":  span.SpanContext().SpanID().String(),
    }).Info("Processing user request")
}
```

### With Metrics

Combine tracing with metrics for complete observability:

```go
// Stack all middleware
handler := tracingMiddleware(
    metricsMiddleware(
        loggingMiddleware(mux)
    )
)
```

Recommendation: Order middleware as: tracing → metrics → logging → auth → business logic

### With JWT Authentication

```go
import (
    jwtMiddleware "base_project/pkg/jwt/middleware"
)

// Create authenticated handler with tracing
handler := tracingMiddleware(
    jwtAuth.Authenticate(protectedHandler)
)
```

## Configuration

### Sampling

Control which traces are recorded:

```go
config := types.Config{
    // Sample 10% of traces
    SamplingRate: 0.1,
}
```

### Exporters

Choose where to send traces:

```go
// GCP Cloud Trace (production)
config.ExporterType = types.ExporterTypeGCP
config.GCP = types.GCPConfig{
    ProjectID: "my-project",
}

// Stdout (development)
config.ExporterType = types.ExporterTypeStdout

// No-op (testing)
config.ExporterType = types.ExporterTypeNoop
```

## Performance

### HTTP Middleware

- Minimal overhead: ~20-50μs per request
- Automatic context propagation
- Efficient span creation
- No heap allocations in hot path

### gRPC Middleware

- Minimal overhead: ~15-40μs per RPC
- Efficient metadata extraction
- Supports streaming RPCs
- Concurrent-safe

## Best Practices

1. **Initialize once**: Create tracer during service startup
2. **Use context**: Always propagate context through call chains
3. **Name spans clearly**: Use descriptive operation names
4. **Limit cardinality**: Keep span attribute values bounded
5. **Sample appropriately**: Use sampling in high-traffic services
6. **Handle errors**: Record errors in spans
7. **Clean shutdown**: Call `tracer.Shutdown()` on service termination
8. **Avoid sensitive data**: Don't include PII in span attributes

## Trace Context Propagation

### W3C Trace Context

The middleware automatically propagates trace context using W3C Trace Context headers:

- `traceparent` - Contains trace ID, span ID, and trace flags
- `tracestate` - Contains vendor-specific trace information

### Manual Propagation

For non-HTTP/gRPC communication (e.g., message queues):

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
)

// Inject context into carrier
propagator := otel.GetTextMapPropagator()
carrier := propagation.MapCarrier{}
propagator.Inject(ctx, carrier)

// Send carrier with message
message.Headers = carrier

// Extract context from carrier
propagator := otel.GetTextMapPropagator()
ctx := propagator.Extract(context.Background(), propagation.MapCarrier(message.Headers))
```

## Examples

### Complete HTTP Service

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "time"
    
    "base_project/pkg/tracing"
    tracingHTTP "base_project/pkg/tracing/middleware/http"
    "base_project/pkg/tracing/types"
)

func main() {
    // Initialize tracer
    config := types.Config{
        ServiceName:  "user-service",
        Environment:  "production",
        ExporterType: types.ExporterTypeGCP,
        SamplingRate: 1.0,
        GCP: types.GCPConfig{
            ProjectID: os.Getenv("GCP_PROJECT_ID"),
        },
    }
    
    tracer, err := tracing.InitTracer(config)
    if err != nil {
        log.Fatal(err)
    }
    defer tracer.Shutdown(context.Background())
    
    // Setup routes
    mux := http.NewServeMux()
    mux.HandleFunc("/health", handleHealth)
    mux.HandleFunc("/api/users", handleUsers)
    
    // Wrap with tracing
    handler := tracingHTTP.NewHTTPMiddleware(tracer)(mux)
    
    // Start server
    server := &http.Server{
        Addr:    ":8080",
        Handler: handler,
    }
    
    go func() {
        log.Println("Server starting on :8080")
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()
    
    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt)
    <-quit
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    server.Shutdown(ctx)
    tracer.Shutdown(ctx)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
    // Tracing is automatic - just use context
    ctx := r.Context()
    
    // Business logic here
    // Context will propagate trace information
    users := fetchUsers(ctx)
    
    // Return response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(users)
}
```

### Complete gRPC Service

```go
package main

import (
    "context"
    "log"
    "net"
    "os"
    "os/signal"
    
    "base_project/pkg/tracing"
    tracingGRPC "base_project/pkg/tracing/middleware/grpc"
    "base_project/pkg/tracing/types"
    pb "base_project/services/users/pb/users"
    "google.golang.org/grpc"
)

type userService struct {
    pb.UnimplementedUserServiceServer
}

func (s *userService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    // Tracing is automatic - context already has span
    // Business logic here
    return &pb.GetUserResponse{
        User: &pb.User{Id: req.Id, Name: "John Doe"},
    }, nil
}

func main() {
    // Initialize tracer
    config := types.Config{
        ServiceName:  "user-service",
        Environment:  "production",
        ExporterType: types.ExporterTypeGCP,
        SamplingRate: 1.0,
        GCP: types.GCPConfig{
            ProjectID: os.Getenv("GCP_PROJECT_ID"),
        },
    }
    
    tracer, err := tracing.InitTracer(config)
    if err != nil {
        log.Fatal(err)
    }
    defer tracer.Shutdown(context.Background())
    
    // Create gRPC server with tracing
    server := grpc.NewServer(
        grpc.StatsHandler(tracingGRPC.NewServerStatsHandler(tracer)),
    )
    
    // Register service
    pb.RegisterUserServiceServer(server, &userService{})
    
    // Start server
    lis, err := net.Listen("tcp", ":9090")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println("gRPC server starting on :9090")
    go func() {
        if err := server.Serve(lis); err != nil {
            log.Fatal(err)
        }
    }()
    
    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt)
    <-quit
    
    server.GracefulStop()
    tracer.Shutdown(context.Background())
}
```

## Troubleshooting

### Traces not appearing

1. Verify tracer is initialized: `tracer, err := tracing.InitTracer(config)`
2. Check GCP project ID is set
3. Verify credentials are configured
4. Check sampling rate: `config.SamplingRate = 1.0` for testing
5. Ensure tracer.Shutdown() is called on exit

### Missing spans

1. Ensure context is propagated through all function calls
2. Check middleware is applied to routes
3. Verify sampling rate isn't too low
4. Check for errors during span creation

### Performance issues

1. Reduce sampling rate for high-traffic services
2. Limit span attribute cardinality
3. Avoid creating too many child spans
4. Use batch span processors (default in GCP exporter)

## Documentation

For more information, see:

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [GCP Cloud Trace](https://cloud.google.com/trace/docs)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)

## Related Packages

- `pkg/logging/middleware` - Logging middleware for HTTP and gRPC
- `pkg/metrics/middleware` - Metrics middleware for HTTP and gRPC
- `pkg/jwt/middleware` - JWT authentication middleware