// Package metrics provides a factory for creating GCP collectors
package metrics

import (
	"context"
	"fmt"
)

// CollectorFactory creates collectors based on configuration
type CollectorFactory struct{}

// NewCollectorFactory creates a new collector factory
func NewCollectorFactory() *CollectorFactory {
	return &CollectorFactory{}
}

// CreateCollector creates a GCP collector based on the configuration
func (f *CollectorFactory) CreateCollector(config *Config) (Collector, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if !config.Enabled {
		return NewNoOpCollector(), nil
	}

	// Only GCP backend is supported
	if config.Backend != BackendGCP {
		return nil, fmt.Errorf("unsupported backend: %s (only 'gcp' is supported)", config.Backend)
	}

	// GCP collector will be registered by the gcp package
	if gcpCreator == nil {
		return nil, fmt.Errorf("GCP backend not initialized (import _ \"pkg/metrics/gcp\")")
	}

	return gcpCreator(config)
}

var defaultFactory *CollectorFactory
var gcpCreator CollectorCreator

func init() {
	defaultFactory = NewCollectorFactory()
}

// CollectorCreator is a function that creates a collector
type CollectorCreator func(config *Config) (Collector, error)

// RegisterGCPBackend registers the GCP backend creator function
// This is called by the gcp package's init() function
func RegisterGCPBackend(creator CollectorCreator) {
	gcpCreator = creator
}

// NewCollector creates a new GCP collector with the given configuration
func NewCollector(config *Config) (Collector, error) {
	return defaultFactory.CreateCollector(config)
}

// NoOpCollector is a collector that does nothing
type NoOpCollector struct{}

// NewNoOpCollector creates a new no-op collector
func NewNoOpCollector() *NoOpCollector {
	return &NoOpCollector{}
}

// RegisterMetric does nothing
func (n *NoOpCollector) RegisterMetric(def MetricDefinition) error {
	return nil
}

// IncrementCounter does nothing
func (n *NoOpCollector) IncrementCounter(name string, labels Labels) error {
	return nil
}

// IncrementCounterBy does nothing
func (n *NoOpCollector) IncrementCounterBy(name string, value float64, labels Labels) error {
	return nil
}

// SetGauge does nothing
func (n *NoOpCollector) SetGauge(name string, value float64, labels Labels) error {
	return nil
}

// IncrementGauge does nothing
func (n *NoOpCollector) IncrementGauge(name string, labels Labels) error {
	return nil
}

// DecrementGauge does nothing
func (n *NoOpCollector) DecrementGauge(name string, labels Labels) error {
	return nil
}

// IncrementGaugeBy does nothing
func (n *NoOpCollector) IncrementGaugeBy(name string, value float64, labels Labels) error {
	return nil
}

// DecrementGaugeBy does nothing
func (n *NoOpCollector) DecrementGaugeBy(name string, value float64, labels Labels) error {
	return nil
}

// ObserveHistogram does nothing
func (n *NoOpCollector) ObserveHistogram(name string, value float64, labels Labels) error {
	return nil
}

// ObserveSummary does nothing
func (n *NoOpCollector) ObserveSummary(name string, value float64, labels Labels) error {
	return nil
}

// Start does nothing
func (n *NoOpCollector) Start(ctx context.Context) error {
	return nil
}

// Stop does nothing
func (n *NoOpCollector) Stop(ctx context.Context) error {
	return nil
}

// Flush does nothing
func (n *NoOpCollector) Flush(ctx context.Context) error {
	return nil
}
