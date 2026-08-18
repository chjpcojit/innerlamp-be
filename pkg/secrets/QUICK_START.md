# Secrets Management - Quick Start Guide

Get up and running with the secrets management package in 5 minutes.

## 1. Install Dependencies

```bash
# For GCP Secret Manager
go get cloud.google.com/go/secretmanager

# For AWS Secrets Manager
go get github.com/aws/aws-sdk-go-v2
```

## 2. Import the Package

```go
import (
    "base_project/pkg/secrets"
    _ "base_project/pkg/secrets/gcp" // For GCP
    // OR
    _ "base_project/pkg/secrets/aws" // For AWS
)
```

## 3. Create and Connect Client

### GCP

```go
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
```

### AWS

```go
config := &secrets.Config{
    Provider: secrets.ProviderAWS,
    Timeout:  30 * time.Second,
    AWS: &secrets.AWSConfig{
        Region:                "us-east-1",
        UseDefaultCredentials: true, // Use default credential chain
    },
}

client, err := secrets.New(config)
// ... same as above
```

## 4. Basic Operations

### Create Secret

```go
secret, err := client.Create(ctx, "my-secret", []byte("secret-value"), nil)
```

### Get Secret

```go
value, err := client.Get(ctx, "my-secret", nil)
fmt.Printf("Value: %s\n", value.String())
```

### Update Secret (Add New Version)

```go
version, err := client.Update(ctx, "my-secret", []byte("new-value"))
fmt.Printf("New version: %s\n", version.Version)
```

### Delete Secret

```go
err := client.Delete(ctx, "my-secret", nil)
```

### List Secrets

```go
result, err := client.List(ctx, nil)
for _, secret := range result.Secrets {
    fmt.Printf("Secret: %s\n", secret.Name)
}
```

## 5. Version Operations

### Get Specific Version

```go
value, err := client.GetVersion(ctx, "my-secret", "1")
```

### List Versions

```go
result, err := client.ListVersions(ctx, "my-secret", nil)
for _, v := range result.Versions {
    fmt.Printf("Version %s: %s\n", v.Version, v.State)
}
```

## Credential Options

### GCP - Application Default Credentials (Cloud Run)

```go
GCP: &secrets.GCPConfig{
    ProjectID:             "your-project-id",
    UseDefaultCredentials: true,
}
```

### GCP - Service Account File

```go
GCP: &secrets.GCPConfig{
    ProjectID:       "your-project-id",
    CredentialsFile: "/path/to/service-account.json",
}
```

### AWS - Default Credential Chain

```go
AWS: &secrets.AWSConfig{
    Region:                "us-east-1",
    UseDefaultCredentials: true,
}
```

### AWS - Access Keys

```go
AWS: &secrets.AWSConfig{
    Region:          "us-east-1",
    AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
    SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
}
```

## Cloud Run Pattern

```go
var secretsClient secrets.Client

func main() {
    config := &secrets.Config{
        Provider: secrets.ProviderGCP,
        Timeout:  30 * time.Second,
        GCP: &secrets.GCPConfig{
            ProjectID:             "your-project-id",
            UseDefaultCredentials: true,
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
    defer secretsClient.Close()

    // Inject secretsClient into handlers
    http.HandleFunc("/api/config", makeHandler(secretsClient))
    http.ListenAndServe(":8080", nil)
}
```

## Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `ErrNotConnected` | Client not connected | Call `Connect()` before operations |
| `ErrInvalidConfig` | Missing required config | Check Provider, Timeout, and credentials |
| `ErrSecretNotFound` | Secret doesn't exist | Verify secret name |
| `ErrPermissionDenied` | IAM permission denied | Check IAM roles/policies |
| `ErrSecretAlreadyExists` | Secret name in use | Use different name or update existing |

## Provider-Specific Notes

### GCP
- Versions are numeric: `1`, `2`, `3`, ...
- Use `"latest"` for most recent version
- Supports disable/enable/destroy versions

### AWS
- Versions are UUIDs
- Use `"AWSCURRENT"` stage for current version
- Uses recovery window for deletion (default 30 days)
- Limited version disable support

## Next Steps

- See `README.md` for complete documentation
- See `examples/example_usage.go` for more examples
- Configure IAM permissions for production