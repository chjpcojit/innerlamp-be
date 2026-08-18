# Storage Package

A comprehensive cloud storage abstraction layer for Go that supports multiple backends (GCP Cloud Storage, AWS S3) with a unified interface.

## Features

- **Multi-Backend Support**: Google Cloud Storage (GCS) and Amazon S3
- **Unified Interface**: Common interface for all storage operations
- **Factory Pattern**: Runtime backend selection based on configuration
- **Complete Operations**: Upload, download, delete, copy, list, and metadata operations
- **Bucket Management**: Create, delete, and list buckets
- **Signed URLs**: Generate pre-signed URLs for temporary access
- **Serverless Optimized**: Designed for Cloud Run and similar environments
- **Error Handling**: Consistent error handling across all backends
- **Dependency Injection**: Configuration passed through parameters, no environment variable reading

## Serverless Deployment (Cloud Run)

This package is optimized for serverless environments like GCP Cloud Run:

- **Create client once at startup** - not per request
- **Client handles connection pooling internally** - no external pool needed
- **Reuse client across requests** - Cloud Run instances handle multiple requests
- **Close client on shutdown** - for graceful termination

```go
// In main.go - create once, reuse across requests
func main() {
    config := &storage.Config{
        Backend: storage.BackendGCS,
        Timeout: 30 * time.Second,
        GCS: &storage.GCSConfig{
            ProjectID:             "my-project-id",
            UseDefaultCredentials: true,
        },
    }
    
    client, _ := storage.New(config)
    client.Connect(ctx)
    defer client.Close()
    
    // Inject client into handlers
    handler := NewHandler(client)
    http.ListenAndServe(":8080", handler)
}
```

## Supported Backends

- **GCS** (Google Cloud Storage) - Using `cloud.google.com/go/storage`
- **S3** (Amazon S3) - Using `github.com/aws/aws-sdk-go-v2`

## Installation

```bash
go get base_project/pkg/storage
```

## Quick Start

### Google Cloud Storage

```go
package main

import (
    "context"
    "log"
    "strings"

    "base_project/pkg/storage"
    _ "base_project/pkg/storage/gcs" // Register GCS backend
)

func main() {
    // Create configuration - all fields must be explicitly set
    config := &storage.Config{
        Backend: storage.BackendGCS,
        Timeout: 30 * time.Second,
        GCS: &storage.GCSConfig{
            ProjectID:             "my-project-id",
            UseDefaultCredentials: true,
            DefaultBucket:         "my-bucket",
            Location:              "US",
        },
    }

    // Create storage client
    client, err := storage.New(config)
    if err != nil {
        log.Fatal(err)
    }

    // Connect
    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Upload a file
    content := strings.NewReader("Hello, World!")
    info, err := client.Upload(ctx, "my-bucket", "hello.txt", content, &storage.UploadOptions{
        ContentType: "text/plain",
        Metadata: map[string]string{
            "author": "me",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Uploaded: %s (%d bytes)", info.Name, info.Size)
}
```

### Amazon S3

```go
package main

import (
    "context"
    "log"
    "strings"

    "base_project/pkg/storage"
    _ "base_project/pkg/storage/s3" // Register S3 backend
)

func main() {
    // Create configuration - all fields must be explicitly set
    config := &storage.Config{
        Backend: storage.BackendS3,
        Timeout: 30 * time.Second,
        S3: &storage.S3Config{
            Region:          "us-east-1",
            AccessKeyID:     "YOUR_ACCESS_KEY",
            SecretAccessKey: "YOUR_SECRET_KEY",
            DefaultBucket:   "my-bucket",
        },
    }

    // Create storage client
    client, err := storage.New(config)
    if err != nil {
        log.Fatal(err)
    }

    // Connect
    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Upload a file
    content := strings.NewReader("Hello from S3!")
    info, err := client.Upload(ctx, "my-bucket", "hello.txt", content, nil)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Uploaded: %s", info.Name)
}
```

## Configuration

### Storage Config

All configuration fields are required - no defaults are applied.

```go
type Config struct {
    Backend Backend           // BackendGCS or BackendS3 (required)
    Timeout time.Duration     // Operation timeout (required, must be positive)
    GCS     *GCSConfig        // GCS-specific configuration (required for GCS backend)
    S3      *S3Config         // S3-specific configuration (required for S3 backend)
    Options map[string]string // Additional options
}
```

### GCS Configuration

```go
type GCSConfig struct {
    ProjectID             string // GCP project ID (required)
    CredentialsFile       string // Path to service account JSON (required if not using default credentials)
    CredentialsJSON       []byte // Service account JSON content (required if not using default credentials)
    UseDefaultCredentials bool   // Use Application Default Credentials
    DefaultBucket         string // Default bucket for operations
    Location              string // Location for new buckets (e.g., "US", "EU", "asia-east1")
}
```

### S3 Configuration

```go
type S3Config struct {
    Region          string // AWS region (required)
    AccessKeyID     string // AWS access key ID (optional if using IAM roles)
    SecretAccessKey string // AWS secret access key (optional if using IAM roles)
    SessionToken    string // AWS session token (for temporary credentials)
    Endpoint        string // Custom S3-compatible endpoint (optional)
    UsePathStyle    bool   // Use path-style addressing (required for some S3-compatible services)
    DefaultBucket   string // Default bucket for operations
}
```

## Interface

### Client Interface

```go
type Client interface {
    // Connection management
    Connect(ctx context.Context) error
    Close() error
    IsConnected() bool

    // Object operations
    Upload(ctx context.Context, bucket, objectName string, reader io.Reader, opts *UploadOptions) (*ObjectInfo, error)
    Download(ctx context.Context, bucket, objectName string, writer io.Writer, opts *DownloadOptions) (*ObjectInfo, error)
    Delete(ctx context.Context, bucket, objectName string) error
    GetObjectInfo(ctx context.Context, bucket, objectName string) (*ObjectInfo, error)
    UpdateMetadata(ctx context.Context, bucket, objectName string, metadata map[string]string) (*ObjectInfo, error)
    Copy(ctx context.Context, srcBucket, srcObject, dstBucket, dstObject string, opts *CopyOptions) (*ObjectInfo, error)
    Exists(ctx context.Context, bucket, objectName string) (bool, error)

    // Listing
    List(ctx context.Context, bucket string, opts *ListOptions) (*ListResult, error)
    ListAll(ctx context.Context, bucket string, opts *ListOptions) ([]*ObjectInfo, error)

    // Bucket operations
    CreateBucket(ctx context.Context, bucket string, opts *BucketCreateOptions) error
    DeleteBucket(ctx context.Context, bucket string) error
    GetBucketInfo(ctx context.Context, bucket string) (*BucketInfo, error)
    BucketExists(ctx context.Context, bucket string) (bool, error)
    ListBuckets(ctx context.Context) ([]*BucketInfo, error)

    // Signed URLs
    SignedURL(ctx context.Context, bucket, objectName string, opts *SignedURLOptions) (string, error)

    // Metadata
    GetBackend() Backend
    GetConfig() *Config
}
```

## Usage Examples

### Upload with Options

```go
opts := &storage.UploadOptions{
    ContentType:  "application/json",
    StorageClass: storage.StorageClassStandard,
    Metadata: map[string]string{
        "version":    "1.0",
        "created-by": "my-service",
    },
    CacheControl: "max-age=3600",
    ACL:          storage.ACLPrivate,
}

file, _ := os.Open("data.json")
defer file.Close()

info, err := client.Upload(ctx, "my-bucket", "data/config.json", file, opts)
```

### Download with Range

```go
var buf bytes.Buffer

opts := &storage.DownloadOptions{
    Range: &storage.ByteRange{
        Start: 0,
        End:   1023, // First 1KB
    },
}

info, err := client.Download(ctx, "my-bucket", "large-file.bin", &buf, opts)
```

### List with Prefix and Delimiter

```go
opts := &storage.ListOptions{
    Prefix:    "images/",
    Delimiter: "/",
    MaxResults: 100,
}

result, err := client.List(ctx, "my-bucket", opts)
if err != nil {
    log.Fatal(err)
}

// Objects in images/ directory
for _, obj := range result.Objects {
    fmt.Printf("Object: %s (%d bytes)\n", obj.Name, obj.Size)
}

// Subdirectories
for _, prefix := range result.Prefixes {
    fmt.Printf("Directory: %s\n", prefix)
}

// Pagination
if result.NextPageToken != "" {
    opts.PageToken = result.NextPageToken
    // ... continue listing
}
```

### Generate Signed URL

SignedURLOptions is required and must include Method and Expires:

```go
// SignedURLOptions is required - all fields must be explicitly set
opts := &storage.SignedURLOptions{
    Method:  storage.SignedURLMethodGET,
    Expires: 15 * time.Minute, // Required: must be positive duration
}

url, err := client.SignedURL(ctx, "my-bucket", "private-file.pdf", opts)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Download URL: %s\n", url)

// For uploads - opts is required
uploadOpts := &storage.SignedURLOptions{
    Method:      storage.SignedURLMethodPUT,
    Expires:     1 * time.Hour,
    ContentType: "image/png",
}

uploadURL, err := client.SignedURL(ctx, "my-bucket", "uploads/image.png", uploadOpts)
```

### Copy Objects

```go
opts := &storage.CopyOptions{
    StorageClass: storage.StorageClassNearline,
    Metadata: map[string]string{
        "archived": "true",
    },
}

info, err := client.Copy(ctx, 
    "source-bucket", "source/file.txt",
    "dest-bucket", "archive/file.txt",
    opts,
)
```

### Bucket Operations

```go
// Create bucket
opts := &storage.BucketCreateOptions{
    Location:     "us-central1",
    StorageClass: storage.StorageClassStandard,
    Versioning:   true,
    Labels: map[string]string{
        "env":     "production",
        "project": "my-app",
    },
}

if err := client.CreateBucket(ctx, "new-bucket", opts); err != nil {
    log.Fatal(err)
}

// List all buckets
buckets, err := client.ListBuckets(ctx)
for _, b := range buckets {
    fmt.Printf("Bucket: %s (Location: %s)\n", b.Name, b.Location)
}

// Check if bucket exists
exists, err := client.BucketExists(ctx, "my-bucket")

// Get bucket info
info, err := client.GetBucketInfo(ctx, "my-bucket")
fmt.Printf("Versioning enabled: %v\n", info.Versioning)

// Delete bucket (must be empty)
if err := client.DeleteBucket(ctx, "old-bucket"); err != nil {
    log.Fatal(err)
}
```

## Error Handling

The package defines common errors:

```go
var (
    ErrUnsupportedBackend   = errors.New("unsupported storage backend")
    ErrNotConnected         = errors.New("storage client not connected")
    ErrInvalidConfig        = errors.New("invalid storage configuration")
    ErrConnectionFailed     = errors.New("failed to connect to storage")
    ErrBucketNotFound       = errors.New("bucket not found")
    ErrObjectNotFound       = errors.New("object not found")
    ErrBucketAlreadyExists  = errors.New("bucket already exists")
    ErrAccessDenied         = errors.New("access denied")
    ErrInvalidObjectName    = errors.New("invalid object name")
    ErrUploadFailed         = errors.New("upload failed")
    ErrDownloadFailed       = errors.New("download failed")
    ErrDeleteFailed         = errors.New("delete failed")
    ErrListFailed           = errors.New("list operation failed")
    ErrSignedURLFailed      = errors.New("failed to generate signed URL")
    ErrCopyFailed           = errors.New("copy operation failed")
)
```

Example error handling:

```go
info, err := client.GetObjectInfo(ctx, "bucket", "file.txt")
if err != nil {
    switch {
    case errors.Is(err, storage.ErrObjectNotFound):
        log.Println("File does not exist")
    case errors.Is(err, storage.ErrAccessDenied):
        log.Println("Permission denied")
    case errors.Is(err, storage.ErrNotConnected):
        log.Println("Client not connected")
    default:
        log.Printf("Unexpected error: %v", err)
    }
}
```

## Storage Classes

### GCS Storage Classes

```go
StorageClassStandard  // Frequently accessed data
StorageClassNearline  // Access less than once a month
StorageClassColdline  // Access less than once a quarter
StorageClassArchive   // Access less than once a year
```

### S3 Storage Classes

```go
StorageClassS3Standard           // General purpose
StorageClassS3IntelligentTiering // Auto-tiering
StorageClassS3StandardIA         // Infrequent access
StorageClassS3OneZoneIA          // Single AZ infrequent access
StorageClassS3Glacier            // Archive storage
StorageClassS3GlacierDeepArchive // Long-term archive
```

## ACL Options

```go
ACLPrivate           // Owner only
ACLPublicRead        // Public read access
ACLPublicReadWrite   // Public read/write access
ACLAuthenticatedRead // Authenticated users can read
ACLBucketOwnerRead   // Bucket owner can read
ACLBucketOwnerFull   // Bucket owner has full control
```

## S3-Compatible Services

Use with MinIO, DigitalOcean Spaces, Backblaze B2, etc.:

```go
config := &storage.Config{
    Backend: storage.BackendS3,
    Timeout: 30 * time.Second,
    S3: &storage.S3Config{
        Region:          "us-east-1",
        AccessKeyID:     "your-access-key",
        SecretAccessKey: "your-secret-key",
        Endpoint:        "https://s3.example.com",
        UsePathStyle:    true, // Often required for S3-compatible services
    },
}
```

## Best Practices

1. **Create client once**: Create at startup, reuse across requests (Cloud Run reuses instances)
2. **Always close connections**: Use `defer client.Close()` in main
3. **Use contexts**: Pass context for cancellation and timeout support
4. **Handle errors**: Check all errors, especially for network operations
5. **Set timeouts**: Configure appropriate timeouts for your use case
6. **Validate inputs**: Use `ValidateBucketName()` and `ValidateObjectName()`
7. **Set content types**: Specify content types or let `DetectContentType()` help
8. **Inject dependencies**: Pass client to handlers, don't create globally

## Testing

```bash
go test ./pkg/storage/...
```

## Contributing

When adding a new backend:

1. Create a new package under `pkg/storage/`
2. Implement the `Client` interface
3. Register the factory function in `init()`
4. Add tests and documentation

Example:

```go
package mycloud

import "base_project/pkg/storage"

func init() {
    storage.Register(storage.Backend("mycloud"), NewMyCloudClient)
}

func NewMyCloudClient(config *storage.Config) (storage.Client, error) {
    // Implementation
}
```

## License

See the project's main LICENSE file.