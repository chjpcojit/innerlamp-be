# Secrets Management Package

A unified, production-ready Go package for secret management supporting multiple providers (GCP Secret Manager, AWS Secrets Manager) with a factory pattern for runtime provider selection.

## Features

- **Multi-Provider Support**: GCP Secret Manager and AWS Secrets Manager
- **Unified Interface**: Common interface for all secret operations
- **Factory Pattern**: Runtime provider selection based on configuration
- **Full CRUD Operations**: Create, Get, Update, Delete secrets
- **Version Management**: Get specific versions, list versions, enable/disable/destroy versions
- **Dependency Injection**: All configuration passed via parameters (no environment variable reading)
- **Cloud Run Ready**: Designed for serverless environments with proper connection reuse

## Installation

```bash
go get cloud.google.com/go/secretmanager
go get github.com/aws/aws-sdk-go-v2
```

## Quick Start

### GCP Secret Manager

```go
package main

import (
    "context"
    "log"
    "time"

    "base_project/pkg/secrets"
    _ "base_project/pkg/secrets/gcp" // Import to register GCP provider
)

func main() {
    config := &secrets.Config{
        Provider: secrets.ProviderGCP,
        Timeout:  30 * time.Second,
        GCP: &secrets.GCPConfig{
            ProjectID:             "your-project-id",
            UseDefaultCredentials: true, // Use ADC in Cloud Run
        },
    }

    client, err := secrets.New(config)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer client.Close()

    // Create a secret
    secret, err := client.Create(ctx, "my-secret", []byte("secret-value"), nil)
    if err != nil {
        log.Fatalf("Failed to create secret: %v", err)
    }
    log.Printf("Created secret: %s", secret.Name)

    // Get secret value
    value, err := client.Get(ctx, "my-secret", nil)
    if err != nil {
        log.Fatalf("Failed to get secret: %v", err)
    }
    log.Printf("Secret value: %s", value.String())
}
```

### AWS Secrets Manager

```go
package main

import (
    "context"
    "log"
    "time"

    "base_project/pkg/secrets"
    _ "base_project/pkg/secrets/aws" // Import to register AWS provider
)

func main() {
    config := &secrets.Config{
        Provider: secrets.ProviderAWS,
        Timeout:  30 * time.Second,
        AWS: &secrets.AWSConfig{
            Region:                "us-east-1",
            UseDefaultCredentials: true, // Use default credential chain
        },
    }

    client, err := secrets.New(config)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer client.Close()

    // Use client...
}
```

## Configuration

### Main Config

```go
type Config struct {
    // Provider specifies which secret manager to use (required)
    // Values: secrets.ProviderGCP, secrets.ProviderAWS
    Provider Provider

    // Timeout is the default timeout for operations (required)
    Timeout time.Duration

    // GCP holds GCP Secret Manager configuration
    GCP *GCPConfig

    // AWS holds AWS Secrets Manager configuration
    AWS *AWSConfig
}
```

### GCP Configuration

```go
type GCPConfig struct {
    // ProjectID is the GCP project ID (required)
    ProjectID string

    // CredentialsFile is the path to the service account JSON file
    CredentialsFile string

    // CredentialsJSON contains the service account JSON directly
    CredentialsJSON []byte

    // UseDefaultCredentials uses Application Default Credentials (ADC)
    // For serverless (Cloud Run, Cloud Functions), set this to true
    UseDefaultCredentials bool
}
```

### AWS Configuration

```go
type AWSConfig struct {
    // Region is the AWS region (required)
    Region string

    // AccessKeyID is the AWS access key ID
    AccessKeyID string

    // SecretAccessKey is the AWS secret access key
    SecretAccessKey string

    // SessionToken is the AWS session token (for temporary credentials)
    SessionToken string

    // Endpoint is a custom endpoint URL (for testing)
    Endpoint string

    // UseDefaultCredentials uses the default AWS credential chain
    // For serverless (Lambda, ECS), set this to true
    UseDefaultCredentials bool
}
```

## Secret Operations

### Create Secret

```go
// Create with default options
secret, err := client.Create(ctx, "my-secret", []byte("secret-value"), nil)

// Create with options
opts := &secrets.CreateOptions{
    Description: "My application secret",
    Labels: map[string]string{
        "environment": "production",
        "team":        "backend",
    },
}
secret, err := client.Create(ctx, "my-secret", []byte("secret-value"), opts)
```

### Get Secret Value

```go
// Get latest version
value, err := client.Get(ctx, "my-secret", nil)
fmt.Printf("Value: %s\n", value.String())

// Get specific version
opts := &secrets.GetOptions{
    Version: "1",
}
value, err := client.Get(ctx, "my-secret", opts)

// Get with AWS version stage
opts := &secrets.GetOptions{
    VersionStage: "AWSPREVIOUS",
}
value, err := client.Get(ctx, "my-secret", opts)
```

### Get Secret Metadata

```go
secret, err := client.GetMetadata(ctx, "my-secret")
fmt.Printf("Name: %s\n", secret.Name)
fmt.Printf("Created: %v\n", secret.CreatedAt)
fmt.Printf("Current Version: %s\n", secret.CurrentVersion)
```

### Update Secret (Add New Version)

```go
version, err := client.Update(ctx, "my-secret", []byte("new-secret-value"))
fmt.Printf("New version: %s\n", version.Version)
```

### Update Metadata

```go
description := "Updated description"
opts := &secrets.UpdateMetadataOptions{
    Description: &description,
    Labels: map[string]string{
        "updated": "true",
    },
}
secret, err := client.UpdateMetadata(ctx, "my-secret", opts)
```

### Delete Secret

```go
// Delete with default options (GCP: immediate, AWS: 30-day recovery)
err := client.Delete(ctx, "my-secret", nil)

// Force delete without recovery
opts := &secrets.DeleteOptions{
    Force: true,
}
err := client.Delete(ctx, "my-secret", opts)

// AWS: Specify recovery window
opts := &secrets.DeleteOptions{
    RecoveryWindowDays: 7,
}
err := client.Delete(ctx, "my-secret", opts)
```

### List Secrets

```go
// List all secrets
result, err := client.List(ctx, nil)
for _, secret := range result.Secrets {
    fmt.Printf("Secret: %s\n", secret.Name)
}

// List with options
opts := &secrets.ListOptions{
    Prefix:     "prod/",
    MaxResults: 100,
}
result, err := client.List(ctx, opts)

// Paginate through results
for {
    result, err := client.List(ctx, opts)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, secret := range result.Secrets {
        fmt.Printf("Secret: %s\n", secret.Name)
    }
    
    if result.NextPageToken == "" {
        break
    }
    opts.PageToken = result.NextPageToken
}
```

### Check If Secret Exists

```go
exists, err := client.Exists(ctx, "my-secret")
if exists {
    fmt.Println("Secret exists")
}
```

## Version Operations

### Get Specific Version

```go
value, err := client.GetVersion(ctx, "my-secret", "1")
fmt.Printf("Version %s: %s\n", value.Version, value.String())
```

### List Versions

```go
opts := &secrets.ListVersionsOptions{
    IncludeDisabled:  true,
    IncludeDestroyed: false,
}
result, err := client.ListVersions(ctx, "my-secret", opts)
for _, version := range result.Versions {
    fmt.Printf("Version %s: %s\n", version.Version, version.State)
}
```

### Disable Version (GCP only)

```go
err := client.DisableVersion(ctx, "my-secret", "1")
```

### Enable Version

```go
err := client.EnableVersion(ctx, "my-secret", "1")
```

### Destroy Version (GCP only)

```go
err := client.DestroyVersion(ctx, "my-secret", "1")
```

## Cloud Run Integration

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "base_project/pkg/secrets"
    _ "base_project/pkg/secrets/gcp"
)

var secretsClient secrets.Client

func main() {
    // Create client once at startup
    config := &secrets.Config{
        Provider: secrets.ProviderGCP,
        Timeout:  30 * time.Second,
        GCP: &secrets.GCPConfig{
            ProjectID:             "your-project-id",
            UseDefaultCredentials: true, // Cloud Run provides ADC automatically
        },
    }

    var err error
    secretsClient, err = secrets.New(config)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }

    ctx := context.Background()
    if err := secretsClient.Connect(ctx); err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }

    // Graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigChan
        log.Println("Shutting down...")
        secretsClient.Close()
        os.Exit(0)
    }()

    // Inject secretsClient into handlers
    http.HandleFunc("/api/config", makeConfigHandler(secretsClient))
    http.ListenAndServe(":8080", nil)
}

func makeConfigHandler(client secrets.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        value, err := client.Get(ctx, "app-config", nil)
        if err != nil {
            http.Error(w, "Failed to get config", http.StatusInternalServerError)
            return
        }
        w.Write(value.Value)
    }
}
```

## Error Handling

The package defines standard errors for consistent error handling:

```go
var (
    ErrNotConnected         // Client is not connected
    ErrInvalidConfig        // Configuration is invalid
    ErrConnectionFailed     // Failed to connect to provider
    ErrSecretNotFound       // Secret not found
    ErrVersionNotFound      // Secret version not found
    ErrSecretAlreadyExists  // Secret already exists
    ErrPermissionDenied     // Permission denied
    ErrInvalidSecretName    // Invalid secret name
    ErrInvalidSecretValue   // Invalid secret value
    ErrCreateFailed         // Create operation failed
    ErrGetFailed            // Get operation failed
    ErrUpdateFailed         // Update operation failed
    ErrDeleteFailed         // Delete operation failed
    ErrListFailed           // List operation failed
    ErrVersionDisabled      // Version is disabled
    ErrVersionDestroyed     // Version is destroyed
    ErrOperationTimeout     // Operation timed out
    ErrUnsupportedProvider  // Provider not supported
)
```

### Error Handling Example

```go
value, err := client.Get(ctx, "my-secret", nil)
if err != nil {
    switch {
    case errors.Is(err, secrets.ErrSecretNotFound):
        log.Println("Secret does not exist")
    case errors.Is(err, secrets.ErrPermissionDenied):
        log.Println("Permission denied - check IAM policies")
    case errors.Is(err, secrets.ErrVersionDisabled):
        log.Println("Secret version is disabled")
    default:
        log.Printf("Failed to get secret: %v", err)
    }
    return
}
```

## Provider Differences

| Feature | GCP Secret Manager | AWS Secrets Manager |
|---------|-------------------|---------------------|
| Version Format | Numeric (1, 2, 3) | UUID |
| Latest Version | "latest" | "AWSCURRENT" stage |
| Disable Version | Supported | Limited support |
| Destroy Version | Supported | Not supported |
| Secret Deletion | Immediate | Recovery window (7-30 days) |
| Replication | Automatic/User-managed | Automatic |

## Best Practices

1. **Create Once, Reuse**: Create the client once at application startup and reuse it
2. **Use Context**: Always pass appropriate contexts for timeout and cancellation
3. **Handle Errors**: Check all errors and handle them appropriately
4. **Close Client**: Always close the client on shutdown
5. **Use ADC/Default Credentials**: Use Application Default Credentials (GCP) or default credential chain (AWS) in cloud environments
6. **Version Secrets**: Update secrets instead of modifying - this creates an audit trail
7. **Label Secrets**: Use labels for organization and filtering

## Interface Reference

```go
type Client interface {
    // Connection management
    Connect(ctx context.Context) error
    Close() error
    IsConnected() bool

    // Secret CRUD operations
    Create(ctx context.Context, name string, value []byte, opts *CreateOptions) (*Secret, error)
    Get(ctx context.Context, name string, opts *GetOptions) (*SecretValue, error)
    GetMetadata(ctx context.Context, name string) (*Secret, error)
    Update(ctx context.Context, name string, value []byte) (*SecretVersion, error)
    UpdateMetadata(ctx context.Context, name string, opts *UpdateMetadataOptions) (*Secret, error)
    Delete(ctx context.Context, name string, opts *DeleteOptions) error
    List(ctx context.Context, opts *ListOptions) (*ListResult, error)

    // Version operations
    GetVersion(ctx context.Context, name string, version string) (*SecretValue, error)
    ListVersions(ctx context.Context, name string, opts *ListVersionsOptions) (*ListVersionsResult, error)
    DestroyVersion(ctx context.Context, name string, version string) error
    DisableVersion(ctx context.Context, name string, version string) error
    EnableVersion(ctx context.Context, name string, version string) error

    // Utility methods
    GetProvider() Provider
    GetConfig() *Config
    Exists(ctx context.Context, name string) (bool, error)
}
```

## Testing

```bash
# Run unit tests
go test ./pkg/secrets/...

# Run with verbose output
go test -v ./pkg/secrets/...

# Run benchmarks
go test -bench=. ./pkg/secrets/...
```

## License

See the project's main LICENSE file.