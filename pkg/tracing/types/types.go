package types

// ExporterType defines the type of trace exporter to use
type ExporterType string

const (
	// ExporterTypeGCP uses Google Cloud Trace
	ExporterTypeGCP ExporterType = "gcp"
	// ExporterTypeStdout uses stdout (useful for local dev)
	ExporterTypeStdout ExporterType = "stdout"
	// ExporterTypeNoop disables tracing
	ExporterTypeNoop ExporterType = "noop"
)

// Config holds the configuration for the tracing package
type Config struct {
	// ServiceName is the name of the service (e.g., "users-service")
	ServiceName string

	// ProjectID is the GCP project ID (required for GCP exporter)
	ProjectID string

	// SamplingRate is the fraction of traces to sample (0.0 to 1.0)
	SamplingRate float64

	// ExporterType specifies which exporter to use
	ExporterType ExporterType

	// Environment is the deployment environment (e.g., "production", "staging")
	Environment string
}

// Option is a functional option for configuring the tracer
type Option func(*Config)

// WithServiceName sets the service name
func WithServiceName(name string) Option {
	return func(c *Config) {
		c.ServiceName = name
	}
}

// WithProjectID sets the project ID
func WithProjectID(id string) Option {
	return func(c *Config) {
		c.ProjectID = id
	}
}

// WithSamplingRate sets the sampling rate
func WithSamplingRate(rate float64) Option {
	return func(c *Config) {
		c.SamplingRate = rate
	}
}

// WithExporterType sets the exporter type
func WithExporterType(t ExporterType) Option {
	return func(c *Config) {
		c.ExporterType = t
	}
}

// WithEnvironment sets the environment
func WithEnvironment(env string) Option {
	return func(c *Config) {
		c.Environment = env
	}
}
