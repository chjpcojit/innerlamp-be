// Package metrics provides helper functions for common metric operations
package metrics

import (
	"fmt"
	"time"
)

// RegisterStandardHTTPMetrics registers all standard HTTP metrics
func RegisterStandardHTTPMetrics(collector Collector) error {
	metrics := []MetricDefinition{
		NewCounterMetric(
			"http_requests_total",
			"Total number of HTTP requests",
			[]string{LabelMethod, LabelPath, LabelStatusCode},
		),
		NewHistogramMetricWithBuckets(
			"http_request_duration_seconds",
			"HTTP request duration in seconds",
			[]string{LabelMethod, LabelPath, LabelStatusCode},
			DefaultDurationBuckets,
		),
		NewHistogramMetricWithBuckets(
			"http_request_size_bytes",
			"HTTP request size in bytes",
			[]string{LabelMethod, LabelPath},
			DefaultSizeBuckets,
		),
		NewHistogramMetricWithBuckets(
			"http_response_size_bytes",
			"HTTP response size in bytes",
			[]string{LabelMethod, LabelPath, LabelStatusCode},
			DefaultSizeBuckets,
		),
		NewGaugeMetric(
			"http_requests_in_flight",
			"Current number of HTTP requests being processed",
			[]string{LabelMethod, LabelPath},
		),
	}

	for _, metric := range metrics {
		if err := collector.RegisterMetric(metric); err != nil {
			return fmt.Errorf("failed to register metric %s: %w", metric.Name, err)
		}
	}

	return nil
}

// RegisterStandardGRPCMetrics registers all standard gRPC metrics
func RegisterStandardGRPCMetrics(collector Collector) error {
	metrics := []MetricDefinition{
		NewCounterMetric(
			"grpc_requests_total",
			"Total number of gRPC requests",
			[]string{LabelGRPCService, LabelGRPCMethod, LabelGRPCType, LabelGRPCCode},
		),
		NewHistogramMetricWithBuckets(
			"grpc_request_duration_seconds",
			"gRPC request duration in seconds",
			[]string{LabelGRPCService, LabelGRPCMethod, LabelGRPCType, LabelGRPCCode},
			DefaultDurationBuckets,
		),
		NewCounterMetric(
			"grpc_stream_messages_total",
			"Total number of gRPC stream messages",
			[]string{LabelGRPCService, LabelGRPCMethod, LabelDirection},
		),
		NewGaugeMetric(
			"grpc_requests_in_flight",
			"Current number of gRPC requests being processed",
			[]string{LabelGRPCService, LabelGRPCMethod, LabelGRPCType},
		),
	}

	for _, metric := range metrics {
		if err := collector.RegisterMetric(metric); err != nil {
			return fmt.Errorf("failed to register metric %s: %w", metric.Name, err)
		}
	}

	return nil
}

// RecordHTTPRequest records HTTP request metrics
func RecordHTTPRequest(collector Collector, method, path, statusCode string, duration time.Duration, requestSize, responseSize int64) {
	labels := Labels{
		LabelMethod:     method,
		LabelPath:       path,
		LabelStatusCode: statusCode,
	}

	_ = collector.IncrementCounter("http_requests_total", labels)
	_ = collector.ObserveHistogram("http_request_duration_seconds", duration.Seconds(), labels)

	sizeLabels := Labels{
		LabelMethod: method,
		LabelPath:   path,
	}
	_ = collector.ObserveHistogram("http_request_size_bytes", float64(requestSize), sizeLabels)

	responseSizeLabels := Labels{
		LabelMethod:     method,
		LabelPath:       path,
		LabelStatusCode: statusCode,
	}
	_ = collector.ObserveHistogram("http_response_size_bytes", float64(responseSize), responseSizeLabels)
}

// RecordGRPCRequest records gRPC request metrics
func RecordGRPCRequest(collector Collector, service, method, grpcType, code string, duration time.Duration) {
	labels := Labels{
		LabelGRPCService: service,
		LabelGRPCMethod:  method,
		LabelGRPCType:    grpcType,
		LabelGRPCCode:    code,
	}

	_ = collector.IncrementCounter("grpc_requests_total", labels)
	_ = collector.ObserveHistogram("grpc_request_duration_seconds", duration.Seconds(), labels)
}

// RecordGRPCStreamMessage records gRPC stream message metrics
func RecordGRPCStreamMessage(collector Collector, service, method, direction string) {
	labels := Labels{
		LabelGRPCService: service,
		LabelGRPCMethod:  method,
		LabelDirection:   direction,
	}

	_ = collector.IncrementCounter("grpc_stream_messages_total", labels)
}

// IncrementHTTPInFlight increments the in-flight HTTP requests gauge
func IncrementHTTPInFlight(collector Collector, method, path string) {
	labels := Labels{
		LabelMethod: method,
		LabelPath:   path,
	}
	_ = collector.IncrementGauge("http_requests_in_flight", labels)
}

// DecrementHTTPInFlight decrements the in-flight HTTP requests gauge
func DecrementHTTPInFlight(collector Collector, method, path string) {
	labels := Labels{
		LabelMethod: method,
		LabelPath:   path,
	}
	_ = collector.DecrementGauge("http_requests_in_flight", labels)
}

// IncrementGRPCInFlight increments the in-flight gRPC requests gauge
func IncrementGRPCInFlight(collector Collector, service, method, grpcType string) {
	labels := Labels{
		LabelGRPCService: service,
		LabelGRPCMethod:  method,
		LabelGRPCType:    grpcType,
	}
	_ = collector.IncrementGauge("grpc_requests_in_flight", labels)
}

// DecrementGRPCInFlight decrements the in-flight gRPC requests gauge
func DecrementGRPCInFlight(collector Collector, service, method, grpcType string) {
	labels := Labels{
		LabelGRPCService: service,
		LabelGRPCMethod:  method,
		LabelGRPCType:    grpcType,
	}
	_ = collector.DecrementGauge("grpc_requests_in_flight", labels)
}

// WithLabels creates a new label map with additional labels
func WithLabels(base Labels, additional Labels) Labels {
	result := CopyLabels(base)
	for k, v := range additional {
		result[k] = v
	}
	return result
}

// LabelValues converts a label map to a slice of values in the order of keys
func LabelValues(labels Labels, keys []string) []string {
	values := make([]string, len(keys))
	for i, key := range keys {
		values[i] = labels[key]
	}
	return values
}

// CreateLabels creates a label map from alternating key-value pairs
func CreateLabels(pairs ...string) Labels {
	if len(pairs)%2 != 0 {
		return make(Labels)
	}

	labels := make(Labels, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		labels[pairs[i]] = pairs[i+1]
	}
	return labels
}

// SanitizeLabelValue sanitizes a label value for use with metrics
// Removes or replaces characters that may cause issues
func SanitizeLabelValue(value string) string {
	// For now, just return the value
	// Could implement more sophisticated sanitization if needed
	return value
}

// SanitizeMetricName sanitizes a metric name
// Ensures it follows naming conventions for the backend
func SanitizeMetricName(name string) string {
	// Replace dots with underscores for compatibility
	result := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			result += string(r)
		} else if r == '.' || r == '-' || r == ' ' {
			result += "_"
		}
	}
	return result
}

// FormatDuration formats a duration for use as a metric value (in seconds)
func FormatDuration(d time.Duration) float64 {
	return d.Seconds()
}

// FormatBytes formats bytes for use as a metric value
func FormatBytes(bytes int64) float64 {
	return float64(bytes)
}

// MetricBuilder helps build metrics with a fluent API
type MetricBuilder struct {
	collector Collector
	name      string
	labels    Labels
}

// NewMetricBuilder creates a new metric builder
func NewMetricBuilder(collector Collector, name string) *MetricBuilder {
	return &MetricBuilder{
		collector: collector,
		name:      name,
		labels:    make(Labels),
	}
}

// WithLabel adds a label to the metric
func (b *MetricBuilder) WithLabel(key, value string) *MetricBuilder {
	b.labels[key] = value
	return b
}

// WithLabels adds multiple labels to the metric
func (b *MetricBuilder) WithLabels(labels Labels) *MetricBuilder {
	for k, v := range labels {
		b.labels[k] = v
	}
	return b
}

// Increment increments a counter
func (b *MetricBuilder) Increment() error {
	return b.collector.IncrementCounter(b.name, b.labels)
}

// IncrementBy increments a counter by a specific value
func (b *MetricBuilder) IncrementBy(value float64) error {
	return b.collector.IncrementCounterBy(b.name, value, b.labels)
}

// Set sets a gauge value
func (b *MetricBuilder) Set(value float64) error {
	return b.collector.SetGauge(b.name, value, b.labels)
}

// Observe observes a value for a histogram or summary
func (b *MetricBuilder) Observe(value float64) error {
	// Try histogram first, then summary
	err := b.collector.ObserveHistogram(b.name, value, b.labels)
	if err == ErrMetricNotRegistered {
		return b.collector.ObserveSummary(b.name, value, b.labels)
	}
	return err
}

// Time measures the duration of a function
func (b *MetricBuilder) Time(fn func()) time.Duration {
	return ObserveDuration(b.collector, b.name, b.labels, fn)
}
