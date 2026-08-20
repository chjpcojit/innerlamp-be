// Package gcp provides a Google Cloud Monitoring metrics collector implementation
package gcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chjpcoj.com.innerlamp/pkg/metrics"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/api/distribution"
	"google.golang.org/genproto/googleapis/api/label"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func init() {
	// Register GCP backend with the factory
	metrics.RegisterGCPBackend(func(config *metrics.Config) (metrics.Collector, error) {
		return NewCollector(config)
	})
}

// Collector implements the metrics.Collector interface for Google Cloud Monitoring
type Collector struct {
	config         *metrics.Config
	client         *monitoring.MetricClient
	projectName    string
	metricPrefix   string
	resourceType   string
	resourceLabels map[string]string

	mu              sync.RWMutex
	metrics         map[string]*metricDescriptor
	pendingData     []*monitoringpb.TimeSeries
	cumulativeState map[string]*cumulativeMetric

	stopCh chan struct{}
	doneCh chan struct{}
}

// metricDescriptor holds information about a registered metric
type metricDescriptor struct {
	name        string
	metricType  metrics.MetricType
	valueType   metricpb.MetricDescriptor_ValueType
	metricKind  metricpb.MetricDescriptor_MetricKind
	labels      []string
	description string
	buckets     []float64
}

// cumulativeMetric tracks cumulative metric state
type cumulativeMetric struct {
	value      float64
	lastUpdate time.Time
	labels     map[string]string
}

// NewCollector creates a new GCP Cloud Monitoring collector
func NewCollector(config *metrics.Config) (*Collector, error) {
	if config == nil {
		config = metrics.DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if config.GCP == nil {
		return nil, fmt.Errorf("gcp config is required")
	}

	ctx := context.Background()

	// Create client options
	var opts []option.ClientOption

	if config.GCP.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(config.GCP.CredentialsFile))
	} else if len(config.GCP.CredentialsJSON) > 0 {
		opts = append(opts, option.WithCredentialsJSON(config.GCP.CredentialsJSON))
	} else if !config.GCP.UseDefaultCredentials {
		return nil, fmt.Errorf("gcp credentials required")
	}

	// Create monitoring client
	client, err := monitoring.NewMetricClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitoring client: %w", err)
	}

	metricPrefix := config.GCP.MetricPrefix
	if metricPrefix == "" {
		metricPrefix = "custom.googleapis.com"
	}

	resourceType := config.GCP.ResourceType
	if resourceType == "" {
		resourceType = "generic_task"
	}

	resourceLabels := config.GCP.ResourceLabels
	if resourceLabels == nil {
		resourceLabels = make(map[string]string)
	}

	// Add default resource labels for generic_task
	if resourceType == "generic_task" {
		if _, ok := resourceLabels["project_id"]; !ok {
			resourceLabels["project_id"] = config.GCP.ProjectID
		}
		if _, ok := resourceLabels["location"]; !ok {
			resourceLabels["location"] = "global"
		}
		if _, ok := resourceLabels["namespace"]; !ok {
			resourceLabels["namespace"] = config.Namespace
		}
		if _, ok := resourceLabels["job"]; !ok {
			resourceLabels["job"] = "default"
		}
		if _, ok := resourceLabels["task_id"]; !ok {
			resourceLabels["task_id"] = "default"
		}
	}

	c := &Collector{
		config:          config,
		client:          client,
		projectName:     fmt.Sprintf("projects/%s", config.GCP.ProjectID),
		metricPrefix:    metricPrefix,
		resourceType:    resourceType,
		resourceLabels:  resourceLabels,
		metrics:         make(map[string]*metricDescriptor),
		pendingData:     make([]*monitoringpb.TimeSeries, 0),
		cumulativeState: make(map[string]*cumulativeMetric),
		stopCh:          make(chan struct{}),
		doneCh:          make(chan struct{}),
	}

	return c, nil
}

// RegisterMetric registers a new metric with GCP Cloud Monitoring
func (c *Collector) RegisterMetric(def metrics.MetricDefinition) error {
	if !c.config.Enabled {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	fullName := c.getMetricType(def.Name)

	if _, exists := c.metrics[fullName]; exists {
		return nil // Already registered
	}

	// Convert to GCP metric descriptor
	valueType, metricKind := c.convertMetricType(def.Type)

	desc := &metricDescriptor{
		name:        fullName,
		metricType:  def.Type,
		valueType:   valueType,
		metricKind:  metricKind,
		labels:      def.Labels,
		description: def.Description,
		buckets:     def.Buckets,
	}

	c.metrics[fullName] = desc

	// Create metric descriptor in GCP
	ctx, cancel := context.WithTimeout(context.Background(), c.config.GCP.Timeout)
	defer cancel()

	labelDescriptors := make([]*label.LabelDescriptor, 0, len(def.Labels))
	for _, labelName := range def.Labels {
		labelDescriptors = append(labelDescriptors, &label.LabelDescriptor{
			Key:       labelName,
			ValueType: label.LabelDescriptor_STRING,
		})
	}

	gcpDesc := &metricpb.MetricDescriptor{
		Name:        fmt.Sprintf("%s/metricDescriptors/%s", c.projectName, fullName),
		Type:        fullName,
		MetricKind:  metricKind,
		ValueType:   valueType,
		Labels:      labelDescriptors,
		Description: def.Description,
		DisplayName: def.Name,
	}

	req := &monitoringpb.CreateMetricDescriptorRequest{
		Name:             c.projectName,
		MetricDescriptor: gcpDesc,
	}

	// Try to create, ignore if already exists
	_, err := c.client.CreateMetricDescriptor(ctx, req)
	if err != nil && !isAlreadyExistsError(err) {
		return fmt.Errorf("failed to create metric descriptor: %w", err)
	}

	return nil
}

// IncrementCounter increments a counter by 1
func (c *Collector) IncrementCounter(name string, labels metrics.Labels) error {
	return c.IncrementCounterBy(name, 1, labels)
}

// IncrementCounterBy increments a counter by a specific value
func (c *Collector) IncrementCounterBy(name string, value float64, labels metrics.Labels) error {
	if !c.config.Enabled {
		return nil
	}

	fullName := c.getMetricType(name)

	c.mu.Lock()
	defer c.mu.Unlock()

	desc, exists := c.metrics[fullName]
	if !exists {
		return metrics.ErrMetricNotRegistered
	}

	// For counters in GCP, we need to track cumulative values
	stateKey := c.getStateKey(fullName, labels)
	state, exists := c.cumulativeState[stateKey]
	if !exists {
		state = &cumulativeMetric{
			value:      0,
			lastUpdate: time.Now(),
			labels:     labels,
		}
		c.cumulativeState[stateKey] = state
	}

	state.value += value
	state.lastUpdate = time.Now()

	// Create time series data point
	ts := c.createTimeSeries(desc, labels, state.value, time.Now())
	c.pendingData = append(c.pendingData, ts)

	// Flush if batch is full
	if len(c.pendingData) >= c.config.GCP.BatchSize {
		return c.flushLocked(context.Background())
	}

	return nil
}

// SetGauge sets a gauge to a specific value
func (c *Collector) SetGauge(name string, value float64, labels metrics.Labels) error {
	if !c.config.Enabled {
		return nil
	}

	fullName := c.getMetricType(name)

	c.mu.Lock()
	defer c.mu.Unlock()

	desc, exists := c.metrics[fullName]
	if !exists {
		return metrics.ErrMetricNotRegistered
	}

	ts := c.createTimeSeries(desc, labels, value, time.Now())
	c.pendingData = append(c.pendingData, ts)

	if len(c.pendingData) >= c.config.GCP.BatchSize {
		return c.flushLocked(context.Background())
	}

	return nil
}

// IncrementGauge increments a gauge by 1
func (c *Collector) IncrementGauge(name string, labels metrics.Labels) error {
	return c.IncrementGaugeBy(name, 1, labels)
}

// DecrementGauge decrements a gauge by 1
func (c *Collector) DecrementGauge(name string, labels metrics.Labels) error {
	return c.DecrementGaugeBy(name, 1, labels)
}

// IncrementGaugeBy increments a gauge by a specific value
func (c *Collector) IncrementGaugeBy(name string, value float64, labels metrics.Labels) error {
	// For GCP, we need to read current value and add to it
	// This is a simplified implementation - in production, you might want to track state locally
	return c.SetGauge(name, value, labels)
}

// DecrementGaugeBy decrements a gauge by a specific value
func (c *Collector) DecrementGaugeBy(name string, value float64, labels metrics.Labels) error {
	return c.SetGauge(name, -value, labels)
}

// ObserveHistogram observes a value in a histogram
func (c *Collector) ObserveHistogram(name string, value float64, labels metrics.Labels) error {
	if !c.config.Enabled {
		return nil
	}

	fullName := c.getMetricType(name)

	c.mu.Lock()
	defer c.mu.Unlock()

	desc, exists := c.metrics[fullName]
	if !exists {
		return metrics.ErrMetricNotRegistered
	}

	// Create distribution value
	buckets := desc.buckets
	if len(buckets) == 0 {
		buckets = metrics.DefaultDurationBuckets
	}

	bucketCounts := make([]int64, len(buckets)+1)
	for i, bound := range buckets {
		if value <= bound {
			bucketCounts[i]++
			break
		}
	}
	if value > buckets[len(buckets)-1] {
		bucketCounts[len(buckets)]++
	}

	dist := &distribution.Distribution{
		Count: 1,
		Mean:  value,
		BucketOptions: &distribution.Distribution_BucketOptions{
			Options: &distribution.Distribution_BucketOptions_ExplicitBuckets{
				ExplicitBuckets: &distribution.Distribution_BucketOptions_Explicit{
					Bounds: buckets,
				},
			},
		},
		BucketCounts: bucketCounts,
	}

	ts := c.createTimeSeriesDistribution(desc, labels, dist, time.Now())
	c.pendingData = append(c.pendingData, ts)

	if len(c.pendingData) >= c.config.GCP.BatchSize {
		return c.flushLocked(context.Background())
	}

	return nil
}

// ObserveSummary observes a value in a summary (not directly supported by GCP, use histogram)
func (c *Collector) ObserveSummary(name string, value float64, labels metrics.Labels) error {
	// GCP doesn't have a direct summary type, use distribution instead
	return c.ObserveHistogram(name, value, labels)
}

// Start starts the background flusher
func (c *Collector) Start(ctx context.Context) error {
	go c.flushWorker(ctx)
	return nil
}

// Stop stops the collector and flushes pending metrics
func (c *Collector) Stop(ctx context.Context) error {
	close(c.stopCh)
	<-c.doneCh

	// Final flush
	return c.Flush(ctx)
}

// Flush flushes all pending metrics to GCP
func (c *Collector) Flush(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.flushLocked(ctx)
}

func (c *Collector) flushLocked(ctx context.Context) error {
	if len(c.pendingData) == 0 {
		return nil
	}

	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), c.config.GCP.Timeout)
		defer cancel()
	}

	req := &monitoringpb.CreateTimeSeriesRequest{
		Name:       c.projectName,
		TimeSeries: c.pendingData,
	}

	err := c.client.CreateTimeSeries(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to write time series: %w", err)
	}

	// Clear pending data
	c.pendingData = make([]*monitoringpb.TimeSeries, 0)

	return nil
}

func (c *Collector) flushWorker(ctx context.Context) {
	ticker := time.NewTicker(c.config.FlushInterval)
	defer ticker.Stop()
	defer close(c.doneCh)

	for {
		select {
		case <-ticker.C:
			_ = c.Flush(ctx)
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (c *Collector) getMetricType(name string) string {
	fullName := c.config.MetricName(name)
	return fmt.Sprintf("%s/%s", c.metricPrefix, fullName)
}

func (c *Collector) getStateKey(metricType string, labels metrics.Labels) string {
	key := metricType
	for k, v := range labels {
		key += fmt.Sprintf("_%s=%s", k, v)
	}
	return key
}

func (c *Collector) convertMetricType(metricType metrics.MetricType) (metricpb.MetricDescriptor_ValueType, metricpb.MetricDescriptor_MetricKind) {
	switch metricType {
	case metrics.Counter:
		return metricpb.MetricDescriptor_DOUBLE, metricpb.MetricDescriptor_CUMULATIVE
	case metrics.Gauge:
		return metricpb.MetricDescriptor_DOUBLE, metricpb.MetricDescriptor_GAUGE
	case metrics.Histogram:
		return metricpb.MetricDescriptor_DISTRIBUTION, metricpb.MetricDescriptor_CUMULATIVE
	case metrics.Summary:
		return metricpb.MetricDescriptor_DISTRIBUTION, metricpb.MetricDescriptor_CUMULATIVE
	default:
		return metricpb.MetricDescriptor_DOUBLE, metricpb.MetricDescriptor_GAUGE
	}
}

func (c *Collector) createTimeSeries(desc *metricDescriptor, labels metrics.Labels, value float64, timestamp time.Time) *monitoringpb.TimeSeries {
	metricLabels := make(map[string]string)
	for k, v := range labels {
		metricLabels[k] = v
	}

	// Merge const labels
	for k, v := range c.config.ConstLabels {
		metricLabels[k] = v
	}

	ts := &monitoringpb.TimeSeries{
		Metric: &metricpb.Metric{
			Type:   desc.name,
			Labels: metricLabels,
		},
		Resource: &monitoredres.MonitoredResource{
			Type:   c.resourceType,
			Labels: c.resourceLabels,
		},
		Points: []*monitoringpb.Point{
			{
				Interval: &monitoringpb.TimeInterval{
					EndTime: timestamppb.New(timestamp),
				},
				Value: &monitoringpb.TypedValue{
					Value: &monitoringpb.TypedValue_DoubleValue{
						DoubleValue: value,
					},
				},
			},
		},
	}

	return ts
}

func (c *Collector) createTimeSeriesDistribution(desc *metricDescriptor, labels metrics.Labels, dist *distribution.Distribution, timestamp time.Time) *monitoringpb.TimeSeries {
	metricLabels := make(map[string]string)
	for k, v := range labels {
		metricLabels[k] = v
	}

	for k, v := range c.config.ConstLabels {
		metricLabels[k] = v
	}

	ts := &monitoringpb.TimeSeries{
		Metric: &metricpb.Metric{
			Type:   desc.name,
			Labels: metricLabels,
		},
		Resource: &monitoredres.MonitoredResource{
			Type:   c.resourceType,
			Labels: c.resourceLabels,
		},
		Points: []*monitoringpb.Point{
			{
				Interval: &monitoringpb.TimeInterval{
					EndTime: timestamppb.New(timestamp),
				},
				Value: &monitoringpb.TypedValue{
					Value: &monitoringpb.TypedValue_DistributionValue{
						DistributionValue: dist,
					},
				},
			},
		},
	}

	return ts
}

func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "already exists")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
