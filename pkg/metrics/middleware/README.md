# Metrics Middleware

This directory contains HTTP and gRPC middleware implementations for automatic metrics collection and instrumentation.

## Overview

The middleware packages provide transparent metrics collection for HTTP and gRPC services without requiring manual instrumentation in your handlers or service methods.

## Structure

```
middleware/
├── http/          # HTTP middleware for REST APIs
│   └── http.go    # HTTP metrics collection and middleware
└── grpc/          # gRPC interceptors for RPC services
    └── grpc.go    # gRPC metrics collection and interceptors
```

## HTTP Middleware

The HTTP middleware automatically collects metrics for HTTP requests.

### Features

- Request duration tracking
- Status code distribution
- Request size tracking
- Response size tracking
- Active request tracking
- Path-based metrics with pattern matching
- Custom labels support

### Quick Start

```go
import (
    "base_project/pkg/metrics"
    httpMetrics "base_project/pkg/metrics/middleware/http"
    _ "base_project/pkg/metrics/gcp"
)

// Create collector
collector, _ := metrics.NewCollector(config)
collector.Start(ctx)

// Create HTTP metrics
httpMet, _ := httpMetrics.NewMetrics(collector, "my_service", "http")

// Setup routes
mux := http.NewServeMux()
mux.HandleFunc("/api/users", handleUsers)

// Wrap with metrics middleware
handler := httpMet.Middleware(mux)
http.ListenAndServe(":8080", handler)
```

### Metrics Collected

- `http_requests_total` - Total number of HTTP requests
- `http_request_duration_seconds` - HTTP request duration histogram
- `http_requests_in_flight` - Current number of HTTP requests being served
- `http_request_size_bytes` - HTTP request size histogram
- `http_response_size_bytes` - HTTP response size histogram

### Labels

Each metric includes:
- `method` - HTTP method (GET, POST, etc.)
- `path` - Request path pattern
- `status` - HTTP status code
- Custom labels (if provided)

## gRPC Middleware

The gRPC middleware provides interceptors for automatic metrics collection.

### Features

- RPC duration tracking
- Status code distribution
- Unary and stream RPC support
- Method-based metrics
- Custom labels support

### Quick Start

```go
import (
    "base_project/pkg/metrics"
    grpcMetrics "base_project/pkg/metrics/middleware/grpc"
    _ "base_project/pkg/metrics/gcp"
    "google.golang.org/grpc"
)

// Create collector
collector, _ := metrics.NewCollector(config)
collector.Start(ctx)

// Create gRPC metrics
grpcMet, _ := grpcMetrics.NewMetrics(collector, "my_service", "grpc")

// Create server with interceptors
server := grpc.NewServer(
    grpc.UnaryInterceptor(grpcMet.UnaryServerInterceptor()),
    grpc.StreamInterceptor(grpcMet.StreamServerInterceptor()),
)
```

### Metrics Collected

- `grpc_requests_total` - Total number of gRPC requests
- `grpc_request_duration_seconds` - gRPC request duration histogram
- `grpc_requests_in_flight` - Current number of gRPC requests being served

### Labels

Each metric includes:
- `method` - gRPC method (e.g., "/users.UserService/GetUser")
- `status` - gRPC status code (OK, CANCELLED, etc.)
- `type` - RPC type (unary or stream)
- Custom labels (if provided)

## Advanced Usage

### Custom Labels

Add custom labels to metrics:

```go
// HTTP
labels := metrics.Labels{
    "region": "us-west-1",
    "version": "v1.0.0",
}
httpMet.MiddlewareWithLabels(mux, labels)

// gRPC
grpcMet.UnaryServerInterceptorWithLabels(labels)
```

### Path Normalization (HTTP)

Configure path pattern matching for cleaner metrics:

```go
config := httpMetrics.HandlerConfig{
    PathPattern: "/api/users/:id",
    Labels: metrics.Labels{
        "api_version": "v1",
    },
}
httpMet.MiddlewareWithConfig(mux, config)
```

### Manual Metrics

Use the metrics instance directly for custom instrumentation:

```go
// HTTP
timer := httpMet.StartTimer()
defer timer.ObserveDuration()

// gRPC
timer := grpcMet.StartTimer()
defer timer.ObserveDuration()
```

## Integration

### With Logging

Combine metrics with logging middleware:

```go
// Create both middleware
logMiddleware := logging.HTTPMiddleware(logger)
metricsMiddleware := httpMet.Middleware

// Chain them
handler := logMiddleware(metricsMiddleware(mux))
```

### With Authentication

Stack middleware in the correct order:

```go
handler := authMiddleware(
    metricsMiddleware(
        logMiddleware(mux)
    )
)
```

Recommendation: Place metrics middleware early in the chain to capture all requests, including authentication failures.

## Performance

### HTTP Middleware

- Minimal overhead: ~50-100μs per request
- No heap allocations in hot path
- Efficient response writer wrapping
- Concurrent-safe

### gRPC Interceptors

- Minimal overhead: ~30-80μs per RPC
- Efficient status code extraction
- Supports both unary and streaming RPCs
- Concurrent-safe

## Best Practices

1. **Create metrics early**: Initialize metrics during service startup
2. **Reuse instances**: Create one metrics instance and reuse it
3. **Use meaningful labels**: Keep cardinality low (<100 unique values)
4. **Normalize paths**: Use path patterns instead of actual paths with IDs
5. **Monitor cardinality**: Watch for high-cardinality label combinations
6. **Test locally**: Use in-memory backend for development
7. **Production backend**: Use GCP Cloud Monitoring for production

## Configuration

Both HTTP and gRPC middleware share the same collector configuration:

```go
config := &metrics.Config{
    Backend:   metrics.BackendGCP,
    Namespace: "my_service",
    Enabled:   true,
    GCP: &metrics.GCPConfig{
        ProjectID:             os.Getenv("GCP_PROJECT_ID"),
        UseDefaultCredentials: true,
        Timeout:               30 * time.Second,
        MaxBatchSize:          200,
        FlushInterval:         60 * time.Second,
    },
}
```

## Examples

See the [examples directory](../examples/) for complete working examples:

- `complete_example.go` - Full HTTP service with metrics

## Troubleshooting

### Metrics not appearing

1. Verify collector is started: `collector.Start(ctx)`
2. Check GCP project ID is set
3. Verify credentials are configured
4. Check for errors in logs
5. Ensure metrics namespace is correct

### High cardinality

1. Use path patterns instead of full paths
2. Limit custom label values
3. Aggregate similar status codes (2xx, 4xx, 5xx)
4. Use sampling for high-traffic endpoints

### Performance issues

1. Increase flush interval for less frequent writes
2. Increase batch size for more efficient uploads
3. Consider sampling high-traffic endpoints
4. Review label cardinality

## Documentation

For more information, see:

- [Main Metrics README](../README.md) - Complete package documentation
- [Quick Reference](../QUICK_REFERENCE.md) - Quick reference guide
- [Usage Guide](../USAGE.md) - Detailed usage examples

## Related Packages

- `pkg/logging/middleware` - Logging middleware for HTTP and gRPC
- `pkg/jwt/middleware` - JWT authentication middleware
- `pkg/middleware` - Common middleware utilities