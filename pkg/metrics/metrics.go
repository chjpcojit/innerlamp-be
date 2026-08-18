// Package metrics provides a unified interface for collecting and exporting metrics
// to Google Cloud Monitoring.
package metrics

import (
	"context"
	"errors"
	"time"
)

// MetricType defines the type of metric
type MetricType string

const (
	// Counter is a cumulative metric that only increases
	Counter MetricType = "counter"
	// Gauge is a metric that can increase or decrease
	Gauge MetricType = "gauge"
	// Histogram samples observations and counts them in configurable buckets
	Histogram MetricType = "histogram"
	// Summary similar to histogram but calculates quantiles
	Summary MetricType = "summary"
)

// Backend defines the metric backend type
type Backend string

const (
	// BackendGCP uses Google Cloud Monitoring
	BackendGCP Backend = "gcp"
)

// Labels represents metric labels as key-value pairs
type Labels map[string]string

// MetricDefinition defines a metric's structure
type MetricDefinition struct {
	Name        string
	Type        MetricType
	Description string
	Labels      []string
	Buckets     []float64           // For histograms
	Objectives  map[float64]float64 // For summaries
}

// Collector is the main interface for collecting metrics
type Collector interface {
	// Counter operations
	IncrementCounter(name string, labels Labels) error
	IncrementCounterBy(name string, value float64, labels Labels) error

	// Gauge operations
	SetGauge(name string, value float64, labels Labels) error
	IncrementGauge(name string, labels Labels) error
	DecrementGauge(name string, labels Labels) error
	IncrementGaugeBy(name string, value float64, labels Labels) error
	DecrementGaugeBy(name string, value float64, labels Labels) error

	// Histogram operations
	ObserveHistogram(name string, value float64, labels Labels) error

	// Summary operations
	ObserveSummary(name string, value float64, labels Labels) error

	// Metric registration
	RegisterMetric(def MetricDefinition) error

	// Lifecycle
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Flush(ctx context.Context) error
}

// Registry manages multiple collectors
type Registry interface {
	Register(name string, collector Collector) error
	Get(name string) (Collector, error)
	Unregister(name string) error
	List() []string
	Default() Collector
	SetDefault(collector Collector)
}

// Config holds configuration for metrics
type Config struct {
	// Backend specifies which backend to use
	Backend Backend

	// Namespace prefix for all metrics
	Namespace string

	// Subsystem prefix for metrics (optional)
	Subsystem string

	// Enabled controls whether metrics collection is active
	Enabled bool

	// ConstLabels are labels applied to all metrics
	ConstLabels Labels

	// GCP configuration
	GCP *GCPConfig

	// FlushInterval for batching metrics
	FlushInterval time.Duration

	// MaxBatchSize for batching metrics
	MaxBatchSize int
}

// GCPConfig configures Google Cloud Monitoring
type GCPConfig struct {
	// ProjectID is the GCP project ID
	ProjectID string

	// CredentialsFile path to service account JSON file
	CredentialsFile string

	// CredentialsJSON contains the service account JSON directly
	CredentialsJSON []byte

	// MetricPrefix for custom metrics (e.g., "custom.googleapis.com")
	MetricPrefix string

	// Timeout for API calls
	Timeout time.Duration

	// BatchSize for batching metric writes
	BatchSize int

	// UseDefaultCredentials uses Application Default Credentials
	UseDefaultCredentials bool

	// ResourceType (e.g., "gce_instance", "k8s_container", "generic_task")
	ResourceType string

	// ResourceLabels additional resource labels
	ResourceLabels map[string]string
}

// Timer provides convenient duration measurement
type Timer struct {
	start     time.Time
	collector Collector
	name      string
	labels    Labels
}

// NewTimer creates a new timer that records duration when stopped
func NewTimer(collector Collector, name string, labels Labels) *Timer {
	return &Timer{
		start:     time.Now(),
		collector: collector,
		name:      name,
		labels:    labels,
	}
}

// Stop records the duration since timer creation
func (t *Timer) Stop() time.Duration {
	duration := time.Since(t.start)
	if t.collector != nil {
		_ = t.collector.ObserveHistogram(t.name, duration.Seconds(), t.labels)
	}
	return duration
}

// ObserveDuration executes a function and records its duration
func ObserveDuration(collector Collector, name string, labels Labels, fn func()) time.Duration {
	timer := NewTimer(collector, name, labels)
	fn()
	return timer.Stop()
}

// Common metric names
const (
	MetricRequestsTotal       = "requests_total"
	MetricRequestDuration     = "request_duration_seconds"
	MetricRequestSize         = "request_size_bytes"
	MetricResponseSize        = "response_size_bytes"
	MetricRequestsInFlight    = "requests_in_flight"
	MetricErrorsTotal         = "errors_total"
	MetricStreamMessagesTotal = "stream_messages_total"
)

// Common label names
const (
	LabelMethod      = "method"
	LabelStatusCode  = "status_code"
	LabelPath        = "path"
	LabelEndpoint    = "endpoint"
	LabelService     = "service"
	LabelType        = "type"
	LabelDirection   = "direction"
	LabelErrorType   = "error_type"
	LabelProtocol    = "protocol"
	LabelGRPCMethod  = "grpc_method"
	LabelGRPCService = "grpc_service"
	LabelGRPCType    = "grpc_type"
	LabelGRPCCode    = "grpc_code"
)

// Standard histogram buckets
var (
	// DefaultDurationBuckets for request duration (in seconds)
	DefaultDurationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

	// DefaultSizeBuckets for request/response sizes (in bytes)
	DefaultSizeBuckets = []float64{100, 1000, 10000, 100000, 1000000, 10000000}

	// DefaultSummaryObjectives for quantiles
	DefaultSummaryObjectives = map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}
)

// DefaultConfig returns a default metrics configuration
func DefaultConfig() *Config {
	return &Config{
		Backend:       BackendGCP,
		Namespace:     "app",
		Subsystem:     "",
		Enabled:       true,
		ConstLabels:   make(Labels),
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  100,
		GCP: &GCPConfig{
			MetricPrefix:          "custom.googleapis.com",
			Timeout:               30 * time.Second,
			BatchSize:             200,
			UseDefaultCredentials: true,
			ResourceType:          "generic_task",
			ResourceLabels:        make(map[string]string),
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Namespace == "" {
		return errors.New("namespace is required")
	}

	// Only GCP backend is supported
	if c.Backend != BackendGCP {
		return errors.New("only gcp backend is supported, got: " + string(c.Backend))
	}

	if c.GCP == nil {
		return errors.New("gcp config is required")
	}
	if c.GCP.ProjectID == "" {
		return errors.New("gcp project_id is required")
	}
	if !c.GCP.UseDefaultCredentials && c.GCP.CredentialsFile == "" && len(c.GCP.CredentialsJSON) == 0 {
		return errors.New("gcp credentials are required when not using default credentials")
	}
	return nil
}

// MetricName constructs a full metric name with namespace and subsystem
func (c *Config) MetricName(name string) string {
	parts := []string{}
	if c.Namespace != "" {
		parts = append(parts, c.Namespace)
	}
	if c.Subsystem != "" {
		parts = append(parts, c.Subsystem)
	}
	if name != "" {
		parts = append(parts, name)
	}
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += "_"
		}
		result += part
	}
	return result
}

// MergeLabels merges multiple label maps, with later maps overwriting earlier ones
func MergeLabels(labelMaps ...Labels) Labels {
	result := make(Labels)
	for _, labels := range labelMaps {
		for k, v := range labels {
			result[k] = v
		}
	}
	return result
}

// CopyLabels creates a copy of labels
func CopyLabels(labels Labels) Labels {
	if labels == nil {
		return make(Labels)
	}
	result := make(Labels, len(labels))
	for k, v := range labels {
		result[k] = v
	}
	return result
}

// NewCounterMetric creates a counter metric definition
func NewCounterMetric(name, description string, labels []string) MetricDefinition {
	return MetricDefinition{
		Name:        name,
		Type:        Counter,
		Description: description,
		Labels:      labels,
	}
}

// NewGaugeMetric creates a gauge metric definition
func NewGaugeMetric(name, description string, labels []string) MetricDefinition {
	return MetricDefinition{
		Name:        name,
		Type:        Gauge,
		Description: description,
		Labels:      labels,
	}
}

// NewHistogramMetric creates a histogram metric definition with default buckets
func NewHistogramMetric(name, description string, labels []string) MetricDefinition {
	return MetricDefinition{
		Name:        name,
		Type:        Histogram,
		Description: description,
		Labels:      labels,
		Buckets:     DefaultDurationBuckets,
	}
}

// NewHistogramMetricWithBuckets creates a histogram metric definition with custom buckets
func NewHistogramMetricWithBuckets(name, description string, labels []string, buckets []float64) MetricDefinition {
	return MetricDefinition{
		Name:        name,
		Type:        Histogram,
		Description: description,
		Labels:      labels,
		Buckets:     buckets,
	}
}

// NewSummaryMetric creates a summary metric definition with default objectives
func NewSummaryMetric(name, description string, labels []string) MetricDefinition {
	return MetricDefinition{
		Name:        name,
		Type:        Summary,
		Description: description,
		Labels:      labels,
		Objectives:  DefaultSummaryObjectives,
	}
}

// NewSummaryMetricWithObjectives creates a summary metric definition with custom objectives
func NewSummaryMetricWithObjectives(name, description string, labels []string, objectives map[float64]float64) MetricDefinition {
	return MetricDefinition{
		Name:        name,
		Type:        Summary,
		Description: description,
		Labels:      labels,
		Objectives:  objectives,
	}
}

// Error definitions
var (
	ErrCollectorNotFound    = errors.New("collector not found")
	ErrCollectorExists      = errors.New("collector already exists")
	ErrMetricNotRegistered  = errors.New("metric not registered")
	ErrInvalidMetricType    = errors.New("invalid metric type")
	ErrInvalidConfiguration = errors.New("invalid configuration")
	ErrBackendNotSupported  = errors.New("backend not supported")
)
