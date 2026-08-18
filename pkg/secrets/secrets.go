// Package secrets provides a unified interface for secret management
// supporting multiple providers (GCP Secret Manager, AWS Secrets Manager)
// with a factory pattern for runtime provider selection.
//
// This package is designed for serverless environments (like Cloud Run) where:
// - A single client is created at startup and reused across requests
// - All configuration is passed via parameters (no environment variable reading)
// - The client handles connection management internally
package secrets

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Common errors for secret management operations
var (
	ErrNotConnected         = errors.New("secrets client not connected")
	ErrInvalidConfig        = errors.New("invalid secrets configuration")
	ErrConnectionFailed     = errors.New("failed to connect to secrets provider")
	ErrSecretNotFound       = errors.New("secret not found")
	ErrVersionNotFound      = errors.New("secret version not found")
	ErrSecretAlreadyExists  = errors.New("secret already exists")
	ErrPermissionDenied     = errors.New("permission denied")
	ErrInvalidSecretName    = errors.New("invalid secret name")
	ErrInvalidSecretValue   = errors.New("invalid secret value")
	ErrCreateFailed         = errors.New("failed to create secret")
	ErrGetFailed            = errors.New("failed to get secret")
	ErrUpdateFailed         = errors.New("failed to update secret")
	ErrDeleteFailed         = errors.New("failed to delete secret")
	ErrListFailed           = errors.New("list operation failed")
	ErrVersionDisabled      = errors.New("secret version is disabled")
	ErrVersionDestroyed     = errors.New("secret version is destroyed")
	ErrOperationTimeout     = errors.New("operation timed out")
	ErrAlreadyConnected     = errors.New("client is already connected")
	ErrUnsupportedProvider  = errors.New("unsupported secrets provider")
	ErrEmptySecretName      = errors.New("secret name cannot be empty")
	ErrEmptySecretValue     = errors.New("secret value cannot be empty")
	ErrInvalidVersion       = errors.New("invalid version identifier")
	ErrDestroyVersionFailed = errors.New("failed to destroy secret version")
	ErrDisableVersionFailed = errors.New("failed to disable secret version")
	ErrEnableVersionFailed  = errors.New("failed to enable secret version")
)

// Provider represents the secret management provider type
type Provider string

const (
	// ProviderGCP uses Google Cloud Secret Manager
	ProviderGCP Provider = "gcp"
	// ProviderAWS uses AWS Secrets Manager
	ProviderAWS Provider = "aws"
)

// VersionState represents the state of a secret version
type VersionState string

const (
	// VersionStateEnabled indicates the version is enabled and accessible
	VersionStateEnabled VersionState = "enabled"
	// VersionStateDisabled indicates the version is disabled
	VersionStateDisabled VersionState = "disabled"
	// VersionStateDestroyed indicates the version is destroyed
	VersionStateDestroyed VersionState = "destroyed"
	// VersionStatePending indicates the version is pending creation
	VersionStatePending VersionState = "pending"
)

// VersionAlias represents common version aliases
const (
	// VersionLatest refers to the most recent version
	VersionLatest = "latest"
	// VersionPrevious refers to the previous version (AWS only)
	VersionPrevious = "previous"
)

// Config holds secrets provider configuration
// All fields must be provided explicitly - no default values are assumed
type Config struct {
	// Provider specifies which secret manager to use (required)
	Provider Provider

	// Timeout is the default timeout for operations (required)
	Timeout time.Duration

	// GCP holds GCP Secret Manager specific configuration
	GCP *GCPConfig

	// AWS holds AWS Secrets Manager specific configuration
	AWS *AWSConfig
}

// GCPConfig holds GCP Secret Manager specific configuration
// For serverless environments (Cloud Run, Cloud Functions), the service
// automatically uses the attached service account via Application Default Credentials (ADC)
type GCPConfig struct {
	// ProjectID is the GCP project ID (required)
	ProjectID string

	// CredentialsFile is the path to the service account JSON file
	CredentialsFile string

	// CredentialsJSON contains the service account JSON directly
	CredentialsJSON []byte

	// UseDefaultCredentials uses Application Default Credentials (ADC)
	// When true, CredentialsFile and CredentialsJSON are ignored
	// For serverless (Cloud Run, Cloud Functions), set this to true
	UseDefaultCredentials bool
}

// AWSConfig holds AWS Secrets Manager specific configuration
// For serverless environments (Lambda, ECS), the service
// automatically uses the attached IAM role via the default credential chain
type AWSConfig struct {
	// Region is the AWS region (required)
	Region string

	// AccessKeyID is the AWS access key ID
	// If empty, SDK will use default credential chain
	AccessKeyID string

	// SecretAccessKey is the AWS secret access key
	SecretAccessKey string

	// SessionToken is the AWS session token (for temporary credentials)
	SessionToken string

	// Endpoint is a custom endpoint URL (for testing or custom deployments)
	Endpoint string

	// UseDefaultCredentials uses the default AWS credential chain
	// When true, AccessKeyID and SecretAccessKey are ignored
	// For serverless (Lambda, ECS), set this to true
	UseDefaultCredentials bool
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("%w: config is nil", ErrInvalidConfig)
	}

	if c.Provider == "" {
		return fmt.Errorf("%w: Provider is required", ErrInvalidConfig)
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("%w: Timeout must be greater than 0", ErrInvalidConfig)
	}

	switch c.Provider {
	case ProviderGCP:
		if c.GCP == nil {
			return fmt.Errorf("%w: GCP configuration is required for GCP provider", ErrInvalidConfig)
		}
		if err := c.GCP.Validate(); err != nil {
			return err
		}
	case ProviderAWS:
		if c.AWS == nil {
			return fmt.Errorf("%w: AWS configuration is required for AWS provider", ErrInvalidConfig)
		}
		if err := c.AWS.Validate(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedProvider, c.Provider)
	}

	return nil
}

// Validate validates GCP configuration
func (c *GCPConfig) Validate() error {
	if c.ProjectID == "" {
		return fmt.Errorf("%w: GCP ProjectID is required", ErrInvalidConfig)
	}

	return nil
}

// Validate validates AWS configuration
func (c *AWSConfig) Validate() error {
	if c.Region == "" {
		return fmt.Errorf("%w: AWS Region is required", ErrInvalidConfig)
	}

	return nil
}

// Secret represents secret metadata
type Secret struct {
	// Name is the secret name/identifier
	Name string

	// Description is an optional description of the secret
	Description string

	// Labels are key-value pairs associated with the secret
	Labels map[string]string

	// CreatedAt is when the secret was created
	CreatedAt time.Time

	// UpdatedAt is when the secret was last updated
	UpdatedAt time.Time

	// VersionCount is the number of versions (GCP specific)
	VersionCount int

	// CurrentVersion is the current/latest version identifier
	CurrentVersion string

	// ARN is the Amazon Resource Name (AWS specific)
	ARN string

	// ResourceName is the full resource name (GCP specific)
	// Format: projects/*/secrets/*
	ResourceName string

	// Metadata contains provider-specific metadata
	Metadata map[string]string
}

// SecretValue represents a secret's value
type SecretValue struct {
	// Name is the secret name
	Name string

	// Value is the secret data
	Value []byte

	// Version is the version identifier
	Version string

	// CreatedAt is when this version was created
	CreatedAt time.Time

	// State is the version state (enabled, disabled, destroyed)
	State VersionState
}

// String returns the secret value as a string
func (s *SecretValue) String() string {
	return string(s.Value)
}

// SecretVersion represents metadata about a secret version
type SecretVersion struct {
	// Version is the version identifier
	Version string

	// State is the version state
	State VersionState

	// CreatedAt is when this version was created
	CreatedAt time.Time

	// DestroyedAt is when this version was destroyed (if applicable)
	DestroyedAt *time.Time

	// ResourceName is the full resource name (GCP specific)
	ResourceName string

	// VersionStages are the staging labels (AWS specific)
	// e.g., AWSCURRENT, AWSPREVIOUS
	VersionStages []string
}

// CreateOptions holds options for creating a secret
type CreateOptions struct {
	// Description is an optional description
	Description string

	// Labels are key-value pairs to associate with the secret
	Labels map[string]string

	// Replication specifies replication policy (GCP specific)
	// If empty, automatic replication is used
	Replication *ReplicationPolicy
}

// ReplicationPolicy specifies how secret data is replicated (GCP specific)
type ReplicationPolicy struct {
	// Automatic uses automatic replication across all regions
	Automatic bool

	// Locations specifies specific locations for user-managed replication
	Locations []string
}

// GetOptions holds options for getting a secret value
type GetOptions struct {
	// Version specifies which version to retrieve
	// Use "latest" for the most recent version
	// If empty, defaults to "latest"
	Version string

	// VersionStage specifies the staging label (AWS specific)
	// e.g., "AWSCURRENT", "AWSPREVIOUS"
	VersionStage string
}

// UpdateOptions holds options for updating a secret value
type UpdateOptions struct {
	// Description updates the secret description
	Description string
}

// UpdateMetadataOptions holds options for updating secret metadata
type UpdateMetadataOptions struct {
	// Description updates the secret description
	Description *string

	// Labels updates the secret labels
	// Set a label to empty string to remove it
	Labels map[string]string
}

// DeleteOptions holds options for deleting a secret
type DeleteOptions struct {
	// Force deletes the secret even if it has versions
	Force bool

	// RecoveryWindowDays specifies the recovery window (AWS specific)
	// Default is 30 days, minimum is 7, maximum is 30
	// Set to 0 for immediate deletion (not recommended)
	RecoveryWindowDays int
}

// ListOptions holds options for listing secrets
type ListOptions struct {
	// Prefix filters secrets by name prefix
	Prefix string

	// MaxResults limits the number of results
	// If 0, returns all results
	MaxResults int

	// PageToken is the pagination token for fetching next page
	PageToken string

	// Filter is a filter expression (provider-specific)
	Filter string
}

// ListResult holds the result of a list operation
type ListResult struct {
	// Secrets is the list of secrets
	Secrets []*Secret

	// NextPageToken is the token for fetching the next page
	// Empty if no more results
	NextPageToken string
}

// ListVersionsOptions holds options for listing secret versions
type ListVersionsOptions struct {
	// MaxResults limits the number of results
	MaxResults int

	// PageToken is the pagination token
	PageToken string

	// IncludeDisabled includes disabled versions
	IncludeDisabled bool

	// IncludeDestroyed includes destroyed versions
	IncludeDestroyed bool
}

// ListVersionsResult holds the result of listing versions
type ListVersionsResult struct {
	// Versions is the list of versions
	Versions []*SecretVersion

	// NextPageToken is the token for fetching the next page
	NextPageToken string
}

// Client is the main interface for secret management operations
type Client interface {
	// Connection management
	Connect(ctx context.Context) error
	Close() error
	IsConnected() bool

	// Secret CRUD operations
	// Create creates a new secret with an initial value
	Create(ctx context.Context, name string, value []byte, opts *CreateOptions) (*Secret, error)

	// Get retrieves a secret value
	// Use opts.Version to specify a version, defaults to "latest"
	Get(ctx context.Context, name string, opts *GetOptions) (*SecretValue, error)

	// GetMetadata retrieves secret metadata without the value
	GetMetadata(ctx context.Context, name string) (*Secret, error)

	// Update adds a new version with the provided value
	Update(ctx context.Context, name string, value []byte) (*SecretVersion, error)

	// UpdateMetadata updates secret metadata (description, labels)
	UpdateMetadata(ctx context.Context, name string, opts *UpdateMetadataOptions) (*Secret, error)

	// Delete removes a secret and all its versions
	Delete(ctx context.Context, name string, opts *DeleteOptions) error

	// List retrieves all secrets
	List(ctx context.Context, opts *ListOptions) (*ListResult, error)

	// Version operations
	// GetVersion retrieves a specific version of a secret
	GetVersion(ctx context.Context, name string, version string) (*SecretValue, error)

	// ListVersions lists all versions of a secret
	ListVersions(ctx context.Context, name string, opts *ListVersionsOptions) (*ListVersionsResult, error)

	// DestroyVersion permanently destroys a secret version
	DestroyVersion(ctx context.Context, name string, version string) error

	// DisableVersion disables a secret version (can be re-enabled)
	DisableVersion(ctx context.Context, name string, version string) error

	// EnableVersion enables a previously disabled secret version
	EnableVersion(ctx context.Context, name string, version string) error

	// Utility methods
	// GetProvider returns the provider type
	GetProvider() Provider

	// GetConfig returns the configuration
	GetConfig() *Config

	// Exists checks if a secret exists
	Exists(ctx context.Context, name string) (bool, error)
}

// FactoryFunc is the function type for creating client instances
type FactoryFunc func(*Config) (Client, error)

var (
	factories = make(map[Provider]FactoryFunc)
)

// Register registers a secrets provider factory function
// This is typically called from an init() function in the provider package
func Register(provider Provider, factory FactoryFunc) {
	factories[provider] = factory
}

// New creates a new secrets client based on configuration.
// All configuration must be explicitly provided - no defaults are applied.
func New(config *Config) (Client, error) {
	if config == nil {
		return nil, fmt.Errorf("%w: config cannot be nil", ErrInvalidConfig)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Get factory for the provider type
	factory, ok := factories[config.Provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s (ensure you import the provider package)", ErrUnsupportedProvider, config.Provider)
	}

	return factory(config)
}

// Helper functions

// ValidateSecretName validates a secret name
func ValidateSecretName(name string) error {
	if name == "" {
		return ErrEmptySecretName
	}

	// Check for invalid characters
	// Both GCP and AWS have restrictions on secret names
	// GCP: must match [a-zA-Z0-9_-]+
	// AWS: must match [a-zA-Z0-9/_+=.@-]+

	for _, r := range name {
		if !isValidSecretNameChar(r) {
			return fmt.Errorf("%w: contains invalid character '%c'", ErrInvalidSecretName, r)
		}
	}

	// Check length (GCP max is 255, AWS max is 512)
	if len(name) > 255 {
		return fmt.Errorf("%w: name exceeds maximum length of 255 characters", ErrInvalidSecretName)
	}

	return nil
}

// isValidSecretNameChar checks if a character is valid in a secret name
func isValidSecretNameChar(r rune) bool {
	// Allow alphanumeric, underscore, hyphen, forward slash, plus, equals, period, at sign
	if r >= 'a' && r <= 'z' {
		return true
	}
	if r >= 'A' && r <= 'Z' {
		return true
	}
	if r >= '0' && r <= '9' {
		return true
	}
	switch r {
	case '_', '-', '/', '+', '=', '.', '@':
		return true
	}
	return false
}

// NormalizeVersion normalizes a version identifier
func NormalizeVersion(version string) string {
	if version == "" {
		return VersionLatest
	}
	return version
}

// IsLatestVersion checks if the version refers to the latest version
func IsLatestVersion(version string) bool {
	return version == "" || version == VersionLatest || version == "AWSCURRENT"
}
