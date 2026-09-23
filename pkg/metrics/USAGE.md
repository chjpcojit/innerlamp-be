# Metrics Package Usage Guide

This guide provides detailed usage instructions for the metrics package with GCP Cloud Monitoring backend.

## Table of Contents

- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Metric Types](#metric-types)
- [HTTP Metrics](#http-metrics)
- [gRPC Metrics](#grpc-metrics)
- [Custom Metrics](#custom-metrics)
- [Advanced Usage](#advanced-usage)
- [Best Practices](#best-practices)

---

## Quick Start

### 1. Import the Package

```go
import (
    "base_project/pkg/metrics"
    _ "base_project/pkg/metrics/gcp"  // Import to register GCP backend
)
```

### 2. Create a Collector

```go
config := &metrics.Config{
    Backend:   metrics.BackendGCP,
    Namespace: "my_service",
    Enabled:   true,
    GCP: &metrics.GCPConfig{
        ProjectID:             "my-gcp-project",
        UseDefaultCredentials: true,
    },
}

collector, err := metrics.NewCollector(config)
if err != nil {
    log.Fatal(err)
}

// Start the collector
ctx := context.Background()
if err := collector.Start(ctx); err != nil {
    log.Fatal(err)
}
defer collector.Stop(ctx)
```

### 3. Register and Use Metrics

```go
// Register a metric
collector.RegisterMetric(metrics.MetricDefinition{
    Name:        "requests_total",
    Type:        metrics.Counter,
    Description: "Total number of requests",
    Labels:      []string{"method", "status"},
})

// Record a metric
labels := metrics.Labels{
    "method": "GET",
    "status": "200",
}
collector.IncrementCounter("requests_total", labels)

// Flush to GCP
collector.Flush(ctx)
```

---

## Configuration

### Basic Configuration

```go
config := &metrics.Config{
    Backend:   metrics.BackendGCP,
    Namespace: "my_service",
    Subsystem: "api",
    Enabled:   true,
}
```

### GCP Configuration

```go
config.GCP = &metrics.GCPConfig{
    // Required
    ProjectID: "my-gcp-project",
    
    // Authentication (choose one)
    UseDefaultCredentials: true,  // Recommended for Cloud Run/GKE
    // OR
    CredentialsFile: "/path/to/service-account.json",
    // OR
    CredentialsJSON: []byte(`{...service account json...}`),
    
    // Optional
    MetricPrefix:   "custom.googleapis.com",
    Timeout:        30 * time.Second,
    BatchSize:      200,
    ResourceType:   "generic_task",
    ResourceLabels: map[string]string{
        "project_id": "my-project",
        "location":   "us-central1",
        "namespace":  "default",
        "job":        "my-service",
        "task_id":    "task-1",
    },
}
```

### Constant Labels

Add labels to all metrics:

```go
config.ConstLabels = metrics.Labels{
    "service":     "users",
    "environment": "production",
    "version":     "1.0.0",
}
```

### Batching Configuration

```go
config.FlushInterval = 60 * time.Second  // How often to flush to GCP
config.MaxBatchSize = 200                // Max metrics per batch
```

### Environment-Based Configuration

```go
func getMetricsConfig() *metrics.Config {
    return &metrics.Config{
        Backend:   metrics.BackendGCP,
        Namespace: os.Getenv("SERVICE_NAME"),
        Enabled:   os.Getenv("METRICS_ENABLED") != "false",
        ConstLabels: metrics.Labels{
            "environment": os.Getenv("ENVIRONMENT"),
            "version":     os.Getenv("VERSION"),
        },
        GCP: &metrics.GCPConfig{
            ProjectID:             os.Getenv("GCP_PROJECT_ID"),
            UseDefaultCredentials: true,
        },
        FlushInterval: 60 * time.Second,
    }
}
```

---

## Metric Types

### Counter

A counter is a cumulative metric that only increases.

```go
// Define counter
collector.RegisterMetric(metrics.NewCounterMetric(
    "requests_total",
    "Total number of requests",
    []string{"method", "status"},
))

// Increment by 1
collector.IncrementCounter("requests_total", metrics.Labels{
    "method": "GET",
    "status": "200",
})

// Increment by N
collector.IncrementCounterBy("requests_total", 5, labels)
```

**Use cases**: Request counts, error counts, task completions

### Gauge

A gauge is a metric that can increase or decrease.

```go
// Define gauge
collector.RegisterMetric(metrics.NewGaugeMetric(
    "active_connections",
    "Number of active connections",
    []string{"protocol"},
))

// Set to specific value
collector.SetGauge("active_connections", 42, metrics.Labels{
    "protocol": "http",
})

// Increment
collector.IncrementGauge("active_connections", labels)
collector.IncrementGaugeBy("active_connections", 5, labels)

// Decrement
collector.DecrementGauge("active_connections", labels)
collector.DecrementGaugeBy("active_connections", 3, labels)
```

**Use cases**: Active connections, queue size, memory usage, temperature

### Histogram

A histogram samples observations and counts them in buckets.

```go
// Define histogram with default buckets
collector.RegisterMetric(metrics.NewHistogramMetric(
    "request_duration_seconds",
    "Request duration in seconds",
    []string{"method"},
))

// Define histogram with custom buckets
collector.RegisterMetric(metrics.NewHistogramMetricWithBuckets(
    "request_size_bytes",
    "Request size in bytes",
    []string{"method"},
    []float64{100, 1000, 10000, 100000, 1000000},
))

// Observe value
duration := 0.5  // 500ms
collector.ObserveHistogram("request_duration_seconds", duration, metrics.Labels{
    "method": "GET",
})
```

**Use cases**: Request durations, response sizes, latencies

### Summary

A summary is similar to histogram but calculates quantiles.

```go
// Define summary with default objectives
collector.RegisterMetric(metrics.NewSummaryMetric(
    "response_time_seconds",
    "Response time in seconds",
    []string{"endpoint"},
))

// Define summary with custom objectives
collector.RegisterMetric(metrics.NewSummaryMetricWithObjectives(
    "latency_seconds",
    "Latency in seconds",
    []string{"endpoint"},
    map[float64]float64{
        0.5:  0.05,   // 50th percentile with 5% error
        0.9:  0.01,   // 90th percentile with 1% error
        0.99: 0.001,  // 99th percentile with 0.1% error
    },
))

// Observe value
collector.ObserveSummary("response_time_seconds", 0.3, labels)
```

**Use cases**: When you need quantiles (p50, p90, p99)

---

## HTTP Metrics

### Automatic Instrumentation

```go
import (
    "net/http"
    httpMetrics "base_project/pkg/metrics/middleware/http"
)

// Create HTTP metrics
httpMet, err := httpMetrics.NewMetrics(collector, "my_service", "http")
if err != nil {
    log.Fatal(err)
}

// Setup routes
mux := http.NewServeMux()
mux.HandleFunc("/api/users", handleUsers)
mux.HandleFunc("/health", handleHealth)

// Wrap with metrics middleware
handler := httpMet.Middleware(mux)

// Start server
http.ListenAndServe(":8080", handler)
```

### Collected Metrics

The HTTP middleware automatically collects:

| Metric | Type | Labels |
|--------|------|--------|
| `http_requests_total` | Counter | method, status, endpoint |
| `http_request_duration_seconds` | Histogram | method, status, endpoint |
| `http_request_size_bytes` | Histogram | method, endpoint |
| `http_response_size_bytes` | Histogram | method, status, endpoint |
| `http_requests_in_flight` | Gauge | - |

### Custom Endpoint Labels

```go
func handleUsers(w http.ResponseWriter, r *http.Request) {
    // The middleware automatically uses the request path
    // To customize, you can set a context value
    utils.WriteSuccessResponse(w, map[string]string{"message": "Users response"}) // envelope chuẩn — pkg/utils/README.md
}
```

---

## gRPC Metrics

### Server Instrumentation

```go
import (
    grpcMetrics "base_project/pkg/metrics/middleware/grpc"
    "google.golang.org/grpc"
)

// Create gRPC metrics
grpcMet, err := grpcMetrics.NewMetrics(collector, "my_service", "grpc")
if err != nil {
    log.Fatal(err)
}

// Create server with interceptors
server := grpc.NewServer(
    grpc.UnaryInterceptor(grpcMet.UnaryServerInterceptor()),
    grpc.StreamInterceptor(grpcMet.StreamServerInterceptor()),
)

// Register services
pb.RegisterUserServiceServer(server, &userService{})

// Start server
lis, _ := net.Listen("tcp", ":9090")
server.Serve(lis)
```

### Client Instrumentation

```go
// Create client with interceptors
conn, err := grpc.Dial(
    "localhost:9090",
    grpc.WithUnaryInterceptor(grpcMet.UnaryClientInterceptor()),
    grpc.WithStreamInterceptor(grpcMet.StreamClientInterceptor()),
)
defer conn.Close()

client := pb.NewUserServiceClient(conn)
```

### Collected Metrics

| Metric | Type | Labels |
|--------|------|--------|
| `grpc_requests_total` | Counter | service, method, code |
| `grpc_request_duration_seconds` | Histogram | service, method, code |
| `grpc_requests_in_flight` | Gauge | - |
| `grpc_stream_messages_total` | Counter | service, method, direction |

---

## Custom Metrics

### Defining Custom Metrics

```go
// Method 1: Using helper functions
orderMetric := metrics.NewCounterMetric(
    "orders_total",
    "Total number of orders",
    []string{"status", "country"},
)

// Method 2: Manual definition
orderMetric := metrics.MetricDefinition{
    Name:        "orders_total",
    Type:        metrics.Counter,
    Description: "Total number of orders",
    Labels:      []string{"status", "country"},
}

// Register the metric
if err := collector.RegisterMetric(orderMetric); err != nil {
    log.Fatal(err)
}
```

### Recording Custom Metrics

```go
// Counter
collector.IncrementCounter("orders_total", metrics.Labels{
    "status":  "completed",
    "country": "US",
})

// Gauge
collector.SetGauge("inventory_level", 150, metrics.Labels{
    "product": "widget",
    "warehouse": "west",
})

// Histogram
collector.ObserveHistogram("processing_time_seconds", 1.5, metrics.Labels{
    "operation": "payment",
})
```

---

## Advanced Usage

### Timing Operations

#### Using Timer

```go
// Start timer
timer := metrics.NewTimer(
    collector,
    "operation_duration_seconds",
    metrics.Labels{"operation": "database_query"},
)

// Do work...
result := database.Query()

// Stop timer and record duration
duration := timer.Stop()
log.Printf("Query took %v", duration)
```

#### Using ObserveDuration Helper

```go
duration := metrics.ObserveDuration(
    collector,
    "operation_duration_seconds",
    metrics.Labels{"operation": "cache_write"},
    func() {
        cache.Write(key, value)
    },
)
```

### Label Helpers

```go
// Merge multiple label maps
labels := metrics.MergeLabels(
    baseLabels,
    metrics.Labels{"status": "200"},
    metrics.Labels{"method": "GET"},
)

// Copy labels
labelsCopy := metrics.CopyLabels(originalLabels)
```

### Registry Usage

```go
// Create registry
registry := metrics.NewRegistry()

// Register collectors
registry.Register("primary", collector1)
registry.Register("secondary", collector2)

// Get collector
collector, err := registry.Get("primary")

// Set default
registry.SetDefault(collector1)
defaultCollector := registry.Default()

// List all collectors
names := registry.List()
```

### Graceful Shutdown

```go
func main() {
    collector, _ := metrics.NewCollector(config)
    collector.Start(context.Background())
    
    // Setup signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    
    // Wait for signal
    <-sigChan
    
    // Graceful shutdown
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    // Flush pending metrics
    if err := collector.Flush(shutdownCtx); err != nil {
        log.Printf("Error flushing metrics: %v", err)
    }
    
    // Stop collector
    if err := collector.Stop(shutdownCtx); err != nil {
        log.Printf("Error stopping collector: %v", err)
    }
}
```

---

## Best Practices

### 1. Metric Naming

Follow these conventions:

```go
// ✅ Good names
"http_requests_total"           // Counter with _total suffix
"http_request_duration_seconds" // Histogram with unit suffix
"active_connections"            // Gauge (current state)
"database_errors_total"         // Descriptive counter

// ❌ Bad names
"requests"                      // Too generic
"time"                          // No unit
"counter1"                      // Not descriptive
```

### 2. Label Cardinality

Keep label cardinality low:

```go
// ✅ Good - Low cardinality (bounded)
labels := metrics.Labels{
    "method":      "GET",           // Limited: GET, POST, PUT, DELETE
    "status":      "200",           // Limited: HTTP status codes
    "endpoint":    "/api/users",    // Limited: API endpoints
}

// ❌ Bad - High cardinality (unbounded)
labels := metrics.Labels{
    "user_id":    "user-12345",    // Unique per user
    "request_id": uuid.New(),      // Unique per request
    "email":      user.Email,      // Unique per user
}
```

### 3. Register Metrics Early

Register all metrics at startup:

```go
func initializeMetrics(collector metrics.Collector) error {
    metricDefs := []metrics.MetricDefinition{
        metrics.NewCounterMetric("requests_total", "Total requests", []string{"method", "status"}),
        metrics.NewHistogramMetric("request_duration", "Request duration", []string{"method"}),
        metrics.NewGaugeMetric("active_connections", "Active connections", nil),
    }
    
    for _, def := range metricDefs {
        if err := collector.RegisterMetric(def); err != nil {
            return fmt.Errorf("failed to register %s: %w", def.Name, err)
        }
    }
    
    return nil
}
```

### 4. Use Appropriate Flush Intervals

```go
// Development - Fast feedback
FlushInterval: 10 * time.Second

// Production - Cost optimized
FlushInterval: 60 * time.Second

// High-throughput - More batching
FlushInterval: 120 * time.Second
```

### 5. Error Handling

```go
// Critical metrics - fail fast
if err := collector.RegisterMetric(criticalMetric); err != nil {
    log.Fatalf("Failed to register critical metric: %v", err)
}

// Non-critical metrics - log and continue
if err := collector.IncrementCounter("optional_metric", labels); err != nil {
    log.Printf("Failed to record metric: %v", err)
}
```

### 6. Resource Labels

Set resource labels appropriately for your environment:

```go
// Cloud Run
ResourceLabels: map[string]string{
    "project_id":         os.Getenv("GOOGLE_CLOUD_PROJECT"),
    "service_name":       os.Getenv("K_SERVICE"),
    "revision_name":      os.Getenv("K_REVISION"),
    "location":           os.Getenv("CLOUD_RUN_REGION"),
    "configuration_name": os.Getenv("K_CONFIGURATION"),
}

// GKE
ResourceLabels: map[string]string{
    "project_id":     os.Getenv("GCP_PROJECT_ID"),
    "location":       os.Getenv("GKE_CLUSTER_LOCATION"),
    "cluster_name":   os.Getenv("GKE_CLUSTER_NAME"),
    "namespace_name": os.Getenv("KUBERNETES_NAMESPACE"),
    "pod_name":       os.Getenv("HOSTNAME"),
    "container_name": "my-container",
}
```

---

## Troubleshooting

### Metrics Not Appearing

1. Verify GCP backend is imported:
   ```go
   import _ "base_project/pkg/metrics/gcp"
   ```

2. Check collector is started:
   ```go
   collector.Start(ctx)
   ```

3. Verify metric is registered:
   ```go
   err := collector.RegisterMetric(metricDef)
   if err != nil {
       log.Printf("Failed to register: %v", err)
   }
   ```

4. Force flush:
   ```go
   collector.Flush(ctx)
   ```

### Authentication Errors

1. Check credentials:
   ```bash
   gcloud auth application-default print-access-token
   ```

2. Verify project ID:
   ```bash
   echo $GCP_PROJECT_ID
   ```

3. Check IAM permissions:
   - Service account needs `roles/monitoring.metricWriter`

### High Costs

1. Increase flush interval:
   ```go
   FlushInterval: 120 * time.Second
   ```

2. Reduce label cardinality
3. Increase batch size:
   ```go
   BatchSize: 200
   ```

4. Monitor quotas in Cloud Console

---

## Examples

See the `examples/` directory for complete working examples.

---

## Next Steps

- Read the [README.md](README.md) for overview
- Check [QUICK_REFERENCE.md](QUICK_REFERENCE.md) for quick API reference
- Review [examples/](examples/) for complete code samples