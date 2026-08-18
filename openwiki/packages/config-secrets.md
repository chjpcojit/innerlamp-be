---
type: Reference
title: Configuration & Secrets Management
description: Detailed usage, symbols, and architectural integration for loading structured configuration and managing secrets via GCP/AWS.
tags: [configuration, secrets, gcp-secret-manager, aws-secrets-manager]
openwiki:
  roles: [domain, integration]
  change_kinds: [lifecycle, public-api]
  source_paths: [pkg/config/loader.go, pkg/secrets/secrets.go]
  symbols: [LoadYAML, LoadSource, Client, Provider, Config]
  test_paths: [pkg/secrets/secrets_test.go]
  invariants: [All fields in secrets.Config must be provided explicitly - no default values assumed.]
  validation_commands: [go test -v ./pkg/secrets/...]
---

# Configuration & Secrets Management

The `innerlamp` framework separates structured file-based application configuration (`pkg/config`) from secure, runtime-resolved credentials and environment secrets (`pkg/secrets`). 

---

## ⚙️ Configuration Loading (`pkg/config`)

The `pkg/config` package provides a unified mechanism to load YAML configuration files. It supports loading from three distinct targets:
1. **Local Filesystem**: Traditional file paths (e.g. `configs/app.yaml`).
2. **AWS S3**: Bucket URIs prefixed with `s3://` (e.g. `s3://my-config-bucket/app.yaml`).
3. **Google Cloud Storage (GCS)**: Bucket URIs prefixed with `gs://` (e.g. `gs://my-config-bucket/app.yaml`).

### Core Types & Methods
* `LoadSource` (type `string`): Represents the URI or path where the configuration is stored.
* `LoadYAML(ctx context.Context, source LoadSource) ([]byte, error)`: Fetches raw YAML content from the targeted source.
* `UnmarshalYAML(data []byte, dest interface{}) error`: Unmarshals raw YAML bytes into a destination struct using `gopkg.in/yaml.v3`.
* `LoadAndUnmarshal(ctx context.Context, source LoadSource, dest interface{}) error`: A helper function that handles both loading and unmarshaling in a single step.

### Local vs. Cloud Auto-Authentication
* **Local**: Relies on standard file permissions.
* **S3**: Requires AWS credentials configured via standard environment variables, the shared AWS config file (`~/.aws/config`), or IAM roles.
* **GCS**: Utilizes Google Application Default Credentials (ADC). When running on GKE or Cloud Run, the credentials are automatically resolved from the attached Service Account.

---

## 🔒 Secrets Management (`pkg/secrets`)

The `pkg/secrets` package provides a cloud-agnostic interface (`secrets.Client`) for CRUD and metadata operations on enterprise secret managers. It utilizes a factory registration pattern to select between **Google Cloud Secret Manager** and **AWS Secrets Manager** at runtime.

### The `secrets.Client` Interface

The primary surface of `pkg/secrets` exposes the following API contract:

```go
type Client interface {
    // Connection Management
    Connect(ctx context.Context) error
    Close() error
    IsConnected() bool

    // Secret CRUD Operations
    Create(ctx context.Context, name string, value []byte, opts *CreateOptions) (*Secret, error)
    Get(ctx context.Context, name string, opts *GetOptions) (*SecretValue, error)
    GetMetadata(ctx context.Context, name string) (*Secret, error)
    Update(ctx context.Context, name string, value []byte, opts *UpdateOptions) (*SecretValue, error)
    UpdateMetadata(ctx context.Context, name string, opts *UpdateMetadataOptions) (*Secret, error)
    Delete(ctx context.Context, name string, opts *DeleteOptions) error

    // Version Management
    AddVersion(ctx context.Context, name string, value []byte) (*SecretValue, error)
    ListVersions(ctx context.Context, name string, opts *ListVersionsOptions) (*ListVersionsResult, error)
    DisableVersion(ctx context.Context, name string, version string) error
    EnableVersion(ctx context.Context, name string, version string) error
    DestroyVersion(ctx context.Context, name string, version string) error
}
```

### Config & Validation Requirements

* **Zero Defaults Rule**: All configuration fields in the `secrets.Config` struct must be explicitly declared. No default values are assumed for timeout or cloud providers.
* **Config Structure**:
  ```go
  type Config struct {
      Provider Provider       // "gcp" or "aws"
      Timeout  time.Duration  // Maximum operation timeout (required)
      GCP      *GCPConfig     // Required if Provider is ProviderGCP
      AWS      *AWSConfig     // Required if Provider is ProviderAWS
  }
  ```

---

## 🔗 Semantic Relationships (OKF Mapped)

* **Secret Resolution**: Configuration loading [is configured through](config-secrets.md) secrets resolution. Applications usually read a bootstrap `LoadSource` URI, unpack the environment, and then instantiate a `secrets.Client` to securely fetch DB passwords or signing keys.
* **Database & Storage Integration**: The database credentials resolved via `pkg/secrets` are passed directly into [Data Persistence & Blob Storage](data-storage.md) constructors.

---

## 🛠️ Code Extension Recipe: Adding a Secret Provider

To add a new secret manager provider (e.g., HashiCorp Vault):

1. **Register the Provider Constant**:
   In `/pkg/secrets/secrets.go`, add your new provider:
   ```go
   const ProviderVault Provider = "vault"
   ```
2. **Implement the Client**:
   Create a new package under `/pkg/secrets/vault/vault.go` and implement the `secrets.Client` interface.
3. **Register via `init()`**:
   Inside your new provider package, register the factory function:
   ```go
   func init() {
       secrets.Register(secrets.ProviderVault, NewVaultClient)
   }
   ```
4. **Blank Import**:
   Ensure your application imports the package to run the `init()` registration:
   ```go
   import _ "base_project/pkg/secrets/vault"
   ```

---

## 🧪 Focused Validation

The secrets package tests reside under `/pkg/secrets/secrets_test.go`. 

To validate any changes to the config or secrets sub-system:
```bash
# Run unit tests
go test -v ./pkg/secrets/...
```
