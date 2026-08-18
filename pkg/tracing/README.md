# Tracing Package

OpenTelemetry-based distributed tracing for microservices with GCP Cloud Trace integration.

## Overview

The tracing package provides a simple, unified interface for distributed tracing across HTTP and gRPC services using OpenTelemetry. It supports automatic span creation, context propagation, and multiple exporter backends.

## Features

- **OpenTelemetry Integration** - Industry-standard distributed tracing
- **Multiple Exporters** - GCP Cloud Trace, stdout, and no-op
- **Automatic Instrumentation** - HTTP and gRPC middleware for transparent tracing
- **Context Propagation** - W3C Trace Context standard support
- **Flexible Configuration** - Easy setup with sensible defaults
- **Production Ready** - Battle-tested with GCP Cloud Trace

## Quick Start

### Installation

```go
import (
    "base_project/pkg/tracing"
    tracingHTTP "base_project/pkg/tracing/middleware/http"
    tracingGRPC "base_project/pkg/tracing/middleware/grpc"
    "base_project/pkg/tracing/types"
)
```

### Initialize Tracer

```go
config := types.Config{
    ServiceName:  "my-service",
    Environment:  "production",
    ExporterType: types.ExporterTypeGCP,
    SamplingRate: 1.0, // 100% sampling
    GCP: types.GCPConfig{
        ProjectID: "my-gcp-project",
    },
}

tracer, err := tracing.InitTracer(config)
if err != nil {
    log.Fatal(err)
}
defer tracer.Shutdown(context.Background())
```

### HTTP Service

```go
// Create HTTP middleware
tracingMiddleware := tracingHTTP.NewHTTPMiddleware(tracer)

// Setup routes
mux := http.NewServeMux()
mux.HandleFunc("/api/users", handleUsers)

// Wrap with tracing
handler := tracingMiddleware(mux)
http.ListenAndServe(":8080", handler)
```

### gRPC Service

```go
// Create gRPC server with tracing
server := grpc.NewServer(
    grpc.StatsHandler(tracingGRPC.NewServerStatsHandler(tracer)),
)

// Register services
pb.RegisterUserServiceServer(server, &userService{})

// Start server
lis, _ := net.Listen("tcp", ":9090")
server.Serve(lis)
```

## Package Structure

```
pkg/tracing/
├── middleware/          # HTTP and gRPC middleware
│   ├── http/           # HTTP tracing middleware
│   ├── grpc/           # gRPC tracing interceptors
│   └── README.md       # Middleware documentation
├── gcp/                # GCP Cloud Trace exporter
├── types/              # Configuration types
├── tracing.go          # Core tracing functionality
└── README.md           # This file
```

## Configuration

### Types

```go
type Config struct {
    ServiceName  string       // Service name for traces
    Environment  string       // Environment (prod, dev, staging)
    ExporterType ExporterType // GCP, Stdout, or Noop
    SamplingRate float64      // 0.0 to 1.0 (0% to 100%)
    GCP          GCPConfig    // GCP-specific config (if using GCP)
}

type GCPConfig struct {
    ProjectID string // GCP project ID
}
```

### Exporter Types

- **`ExporterTypeGCP`** - Send traces to GCP Cloud Trace (production)
- **`ExporterTypeStdout`** - Print traces to stdout (development)
- **`ExporterTypeNoop`** - Disable tracing (testing)

### Example Configurations

**Production (GCP Cloud Trace):**
```go
config := types.Config{
    ServiceName:  "user-service",
    Environment:  "production",
    ExporterType: types.ExporterTypeGCP,
    SamplingRate: 0.1, // 10% sampling for high-traffic
    GCP: types.GCPConfig{
        ProjectID: os.Getenv("GCP_PROJECT_ID"),
    },
}
```

**Development (Stdout):**
```go
config := types.Config{
    ServiceName:  "user-service",
    Environment:  "development",
    ExporterType: types.ExporterTypeStdout,
    SamplingRate: 1.0, // 100% sampling in dev
}
```

**Testing (No-op):**
```go
config := types.Config{
    ServiceName:  "user-service",
    Environment:  "test",
    ExporterType: types.ExporterTypeNoop,
    SamplingRate: 0.0,
}
```

## Middleware

### HTTP Middleware

Automatically instruments HTTP requests with tracing:

```go
import tracingHTTP "base_project/pkg/tracing/middleware/http"

// Basic usage
tracingMiddleware := tracingHTTP.NewHTTPMiddleware(tracer)
handler := tracingMiddleware(mux)

// With route tags for better span names
userHandler := tracingHTTP.WithRouteTag(
    http.HandlerFunc(handleUser),
    "/api/users/:id",
)
```

**Captured Attributes:**
- HTTP method, URL, status code
- User agent, host, scheme
- Route pattern (if using WithRouteTag)

### gRPC Middleware

Automatically instruments gRPC calls with tracing:

```go
import tracingGRPC "base_project/pkg/tracing/middleware/grpc"

// Server-side
server := grpc.NewServer(
    grpc.StatsHandler(tracingGRPC.NewServerStatsHandler(tracer)),
)

// Client-side
conn, err := grpc.Dial(
    "localhost:9090",
    grpc.WithInsecure(),
    tracingGRPC.NewClientDialOption(tracer),
)
```

**Captured Attributes:**
- RPC method, service, status code
- Message sent/received events

See [middleware/README.md](middleware/README.md) for complete middleware documentation.

## Manual Instrumentation

### Create Custom Spans

```go
import "go.opentelemetry.io/otel"

func processUser(ctx context.Context, userID string) error {
    tracer := otel.Tracer("my-service")
    
    // Create span
    ctx, span := tracer.Start(ctx, "process-user")
    defer span.End()
    
    // Add attributes
    span.SetAttributes(
        attribute.String("user.id", userID),
        attribute.String("operation", "update"),
    )
    
    // Business logic
    if err := updateUser(ctx, userID); err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }
    
    return nil
}
```

### Add Events

```go
span.AddEvent("user.validation.started")
span.AddEvent("user.validation.completed", trace.WithAttributes(
    attribute.Int("validation.rules", 5),
))
```

### Record Errors

```go
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
}
```

## Context Propagation

Traces automatically propagate through:
- HTTP requests (via headers)
- gRPC calls (via metadata)
- Function calls (via context.Context)

**Always propagate context:**
```go
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context() // Contains trace context
    
    // Pass context to all functions
    user, err := fetchUser(ctx, userID)
    if err != nil {
        // Error handling
    }
}

func fetchUser(ctx context.Context, userID string) (*User, error) {
    // Use ctx in database calls, HTTP clients, etc.
    // Span context automatically propagates
}
```

## Integration

### With Logging

Correlate logs with traces:

```go
import (
    "base_project/pkg/logging"
    "go.opentelemetry.io/otel/trace"
)

func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    span := trace.SpanFromContext(ctx)
    
    logger.WithContext(ctx).WithFields(logging.Fields{
        "trace_id": span.SpanContext().TraceID().String(),
        "span_id":  span.SpanContext().SpanID().String(),
    }).Info("Processing request")
}
```

### With Metrics

Combine tracing with metrics:

```go
import (
    tracingHTTP "base_project/pkg/tracing/middleware/http"
    metricsHTTP "base_project/pkg/metrics/middleware/http"
    loggingHTTP "base_project/pkg/logging/middleware"
)

// Stack middleware
handler := tracingHTTP.NewHTTPMiddleware(tracer)(
    metricsHTTP.Middleware(
        loggingHTTP.HTTPLogger(logger)(mux),
    ),
)
```

**Recommended order:** tracing → metrics → logging → auth → business logic

### With JWT Authentication

```go
import jwtMiddleware "base_project/pkg/jwt/middleware"

handler := tracingMiddleware(
    jwtAuth.Authenticate(
        protectedHandler,
    ),
)
```

## Sampling

Control which traces are recorded:

```go
// Sample all traces (development)
config.SamplingRate = 1.0

// Sample 10% of traces (production)
config.SamplingRate = 0.1

// Sample 1% of traces (high-traffic)
config.SamplingRate = 0.01

// No sampling (testing)
config.SamplingRate = 0.0
```

**Note:** Sampling is applied at trace creation. Child spans inherit the parent's sampling decision.

## Viewing Traces

### GCP Cloud Trace

1. Navigate to Cloud Console → Trace
2. Select your project
3. Filter by service name, time range, or latency
4. Click on a trace to see detailed span information

### Local Development (Stdout)

Traces print to stdout with pretty formatting:
```json
{
  "Name": "GET /api/users",
  "SpanContext": {
    "TraceID": "4bf92f3577b34da6a3ce929d0e0e4736",
    "SpanID": "00f067aa0ba902b7"
  },
  "Parent": {...},
  "SpanKind": "Server",
  "StartTime": "2025-11-26T10:30:00Z",
  "EndTime": "2025-11-26T10:30:00.150Z",
  "Attributes": {...},
  "Events": [...]
}
```

## Performance

- **Overhead:** ~20-50μs per HTTP request, ~15-40μs per gRPC call
- **Memory:** Minimal - spans are batched and exported asynchronously
- **Network:** Efficient batching reduces API calls to Cloud Trace
- **CPU:** Negligible impact on application performance

## Best Practices

1. **Initialize Once** - Create tracer during service startup
2. **Use Context** - Always propagate context through function calls
3. **Name Spans Clearly** - Use descriptive operation names
4. **Sample Appropriately** - Adjust sampling based on traffic volume
5. **Limit Cardinality** - Keep span attribute values bounded
6. **Handle Errors** - Record errors in spans for debugging
7. **Clean Shutdown** - Call `tracer.Shutdown()` on service exit
8. **Avoid PII** - Don't include sensitive data in span attributes
9. **Use Route Tags** - Normalize HTTP paths with route patterns
10. **Test Locally** - Use stdout exporter during development

## Security

- **No PII in Spans** - Avoid logging sensitive user data
- **Secure Credentials** - Use GCP default credentials or service accounts
- **Network Security** - Traces sent over HTTPS to GCP
- **Access Control** - Use IAM to control trace viewer access

## Troubleshooting

### Traces Not Appearing

1. Verify tracer initialization: `tracer, err := tracing.InitTracer(config)`
2. Check GCP project ID is correct
3. Verify GCP credentials are configured
4. Check sampling rate is > 0
5. Ensure `tracer.Shutdown()` is called on exit

### Missing Spans

1. Ensure context is propagated through function calls
2. Verify middleware is applied to routes
3. Check sampling rate isn't filtering traces
4. Look for errors during span creation

### Performance Issues

1. Reduce sampling rate for high-traffic services
2. Limit span attribute cardinality
3. Avoid creating too many child spans
4. Use batch processors (default in GCP exporter)

### Context Not Propagating

1. Always pass `context.Context` to functions
2. Use `r.Context()` in HTTP handlers
3. Use `stream.Context()` in gRPC handlers
4. Don't create new contexts with `context.Background()`

## Examples

See [middleware/README.md](middleware/README.md) for complete examples:
- HTTP service with tracing
- gRPC service with tracing
- Client-side tracing
- Custom span creation
- Error handling

## Dependencies

- `go.opentelemetry.io/otel` - OpenTelemetry SDK
- `go.opentelemetry.io/contrib/instrumentation` - Instrumentation libraries
- `cloud.google.com/go/trace` - GCP Cloud Trace (for GCP exporter)

## Related Documentation

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [GCP Cloud Trace](https://cloud.google.com/trace/docs)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)
- [OpenTelemetry Go SDK](https://pkg.go.dev/go.opentelemetry.io/otel)

## Related Packages

- `pkg/logging` - Structured logging with correlation IDs
- `pkg/metrics` - Metrics collection and monitoring
- `pkg/jwt` - JWT authentication
- `pkg/middleware` - Common middleware utilities

## Changelog

See [MIGRATION_MIDDLEWARE_REORGANIZATION.md](MIGRATION_MIDDLEWARE_REORGANIZATION.md) for recent changes.

## License

Part of the base_project microservices framework.