package tracing

import (
	"context"
	"fmt"

	"base_project/pkg/tracing/gcp"
	"base_project/pkg/tracing/types"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// Tracer wraps the OpenTelemetry tracer
type Tracer struct {
	tp     *sdktrace.TracerProvider
	tracer trace.Tracer
}

// InitTracer initializes the tracing system
func InitTracer(config types.Config) (*Tracer, error) {
	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporter based on config
	var spanProcessor sdktrace.SpanProcessor

	switch config.ExporterType {
	case types.ExporterTypeGCP:
		processor, err := gcp.NewBatchSpanProcessor(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create GCP span processor: %w", err)
		}
		spanProcessor = processor
	case types.ExporterTypeStdout:
		// Simple stdout exporter for development
		exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout exporter: %w", err)
		}
		spanProcessor = sdktrace.NewSimpleSpanProcessor(exporter)
	case types.ExporterTypeNoop:
		// No-op
		return &Tracer{tp: nil, tracer: otel.Tracer(config.ServiceName)}, nil
	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", config.ExporterType)
	}

	// Create TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SamplingRate)),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(spanProcessor),
	)

	// Set global TracerProvider
	otel.SetTracerProvider(tp)

	// Set global Propagator to W3C Trace Context
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return &Tracer{
		tp:     tp,
		tracer: tp.Tracer(config.ServiceName),
	}, nil
}

// Shutdown shuts down the tracer provider
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t.tp != nil {
		return t.tp.Shutdown(ctx)
	}
	return nil
}

// GetTracer returns the underlying OpenTelemetry tracer
func (t *Tracer) GetTracer() trace.Tracer {
	return t.tracer
}

// Start creates a new span
func (t *Tracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}
