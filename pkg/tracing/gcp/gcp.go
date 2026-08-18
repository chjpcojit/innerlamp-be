package gcp

import (
	"context"
	"fmt"

	"base_project/pkg/tracing/types"

	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/otel/sdk/trace"
)

// NewExporter creates a new GCP Cloud Trace exporter
func NewExporter(config types.Config) (trace.SpanExporter, error) {
	if config.ProjectID == "" {
		return nil, fmt.Errorf("project ID is required for GCP exporter")
	}

	// Create the exporter
	exporter, err := texporter.New(texporter.WithProjectID(config.ProjectID))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP trace exporter: %w", err)
	}

	return exporter, nil
}

// NewBatchSpanProcessor creates a new batch span processor with the GCP exporter
func NewBatchSpanProcessor(config types.Config) (trace.SpanProcessor, error) {
	exporter, err := NewExporter(config)
	if err != nil {
		return nil, err
	}

	// Use BatchSpanProcessor for better performance
	return trace.NewBatchSpanProcessor(exporter), nil
}

// Shutdown shuts down the exporter
func Shutdown(ctx context.Context, tp *trace.TracerProvider) error {
	if tp != nil {
		return tp.Shutdown(ctx)
	}
	return nil
}
