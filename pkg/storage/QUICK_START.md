# Storage Package - Quick Start Guide

Get up and running with the storage package in 5 minutes.

## Important: All Configuration is Required

This package follows dependency injection principles - **all configuration must be explicitly provided**. No default values are applied. This ensures the package is fully reusable across different services.

## Serverless Best Practice (Cloud Run)

**Important**: Create the client ONCE at startup and reuse it across requests. Cloud Run instances handle multiple requests, so client reuse improves performance.

```go
// main.go - Create once, inject into handlers
func main() {
    config := &storage.Config{
        Backend: storage.BackendGCS,
        Timeout: 30 * time.Second,  // Required
        GCS: &storage.GCSConfig{
            ProjectID:             "my-project",
            UseDefaultCredentials: true,
        },
    }
    
    client, _ := storage.New(config)
    client.Connect(ctx)
    defer client.Close()  // Close on shutdown
    
    handler := NewHandler(client)  // Inject into handler
    http.ListenAndServe(":8080", handler)
}
```

## Installation

Import the storage package and the backend you need:

```go
import (
    "base_project/pkg/storage"
    _ "base_project/pkg/storage/gcs" // For Google Cloud Storage
    // OR
    _ "base_project/pkg/storage/s3"  // For Amazon S3
)
```

## GCS Quick Start

```go
package main

import (
    "context"
    "log"
    "strings"
    "time"

    "base_project/pkg/storage"
    _ "base_project/pkg/storage/gcs"
)

func main() {
    ctx := context.Background()

    // 1. Create config - all fields are required
    config := &storage.Config{
        Backend: storage.BackendGCS,
        Timeout: 30 * time.Second,  // Required
        GCS: &storage.GCSConfig{
            ProjectID:             "my-gcp-project",
            UseDefaultCredentials: true,
        },
    }

    // 2. Create and connect
    client, err := storage.New(config)
    if err != nil {
        log.Fatal(err)
    }
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 3. Upload
    content := strings.NewReader("Hello, GCS!")
    _, err = client.Upload(ctx, "my-bucket", "hello.txt", content, nil)
    if err != nil {
        log.Fatal(err)
    }
    log.Println("Uploaded successfully!")
}
```

## S3 Quick Start

```go
package main

import (
    "context"
    "log"
    "strings"
    "time"

    "base_project/pkg/storage"
    _ "base_project/pkg/storage/s3"
)

func main() {
    ctx := context.Background()

    // 1. Create config - all fields are required
    config := &storage.Config{
        Backend: storage.BackendS3,
        Timeout: 30 * time.Second,  // Required
        S3: &storage.S3Config{
            Region:          "us-east-1",
            AccessKeyID:     "AKIAXXXXXXXXXXXXXXXX",
            SecretAccessKey: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
        },
    }

    // 2. Create and connect
    client, err := storage.New(config)
    if err != nil {
        log.Fatal(err)
    }
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 3. Upload
    content := strings.NewReader("Hello, S3!")
    _, err = client.Upload(ctx, "my-bucket", "hello.txt", content, nil)
    if err != nil {
        log.Fatal(err)
    }
    log.Println("Uploaded successfully!")
}
```

## Common Operations

### Upload a File

```go
file, _ := os.Open("document.pdf")
defer file.Close()

info, err := client.Upload(ctx, "bucket", "docs/document.pdf", file, &storage.UploadOptions{
    ContentType: "application/pdf",
})
```

### Download a File

```go
file, _ := os.Create("downloaded.pdf")
defer file.Close()

_, err := client.Download(ctx, "bucket", "docs/document.pdf", file, nil)
```

### Check if File Exists

```go
exists, err := client.Exists(ctx, "bucket", "docs/document.pdf")
if exists {
    log.Println("File exists!")
}
```

### Delete a File

```go
err := client.Delete(ctx, "bucket", "docs/document.pdf")
```

### List Files

```go
result, err := client.List(ctx, "bucket", &storage.ListOptions{
    Prefix: "docs/",
})

for _, obj := range result.Objects {
    fmt.Printf("%s (%d bytes)\n", obj.Name, obj.Size)
}
```

### Generate Signed URL

```go
// SignedURLOptions is required with Method and Expires
url, err := client.SignedURL(ctx, "bucket", "private.pdf", &storage.SignedURLOptions{
    Method:  storage.SignedURLMethodGET,   // Required
    Expires: 1 * time.Hour,                 // Required: must be positive
})
fmt.Printf("Download: %s\n", url)
```

### Copy a File

```go
_, err := client.Copy(ctx, 
    "source-bucket", "original.txt",
    "dest-bucket", "copy.txt", 
    nil,
)
```

## Switching Backends

Just change the import and config:

```go
// From GCS...
import _ "base_project/pkg/storage/gcs"
config := &storage.Config{
    Backend: storage.BackendGCS,
    Timeout: 30 * time.Second,
    GCS: &storage.GCSConfig{...},
}

// To S3...
import _ "base_project/pkg/storage/s3"
config := &storage.Config{
    Backend: storage.BackendS3,
    Timeout: 30 * time.Second,
    S3: &storage.S3Config{...},
}

// The rest of your code stays the same!
client, _ := storage.New(config)
client.Connect(ctx)
client.Upload(ctx, "bucket", "file.txt", reader, nil)
```

## Error Handling

```go
_, err := client.Download(ctx, "bucket", "file.txt", writer, nil)
if errors.Is(err, storage.ErrObjectNotFound) {
    log.Println("File not found")
} else if errors.Is(err, storage.ErrAccessDenied) {
    log.Println("Access denied")
} else if err != nil {
    log.Printf("Error: %v", err)
}
```

## Configuration Tips

### GCS with Service Account File

```go
config := &storage.Config{
    Backend: storage.BackendGCS,
    Timeout: 30 * time.Second,
    GCS: &storage.GCSConfig{
        ProjectID:       "my-project",
        CredentialsFile: "/path/to/service-account.json",
    },
}
```

### S3 with IAM Role (no explicit credentials)

```go
config := &storage.Config{
    Backend: storage.BackendS3,
    Timeout: 30 * time.Second,
    S3: &storage.S3Config{
        Region: "us-east-1",
        // Leave AccessKeyID and SecretAccessKey empty
        // SDK will use IAM role attached to EC2/ECS/Lambda
    },
}
```

### S3-Compatible (MinIO, DigitalOcean Spaces, etc.)

```go
config := &storage.Config{
    Backend: storage.BackendS3,
    Timeout: 30 * time.Second,
    S3: &storage.S3Config{
        Region:          "us-east-1",
        Endpoint:        "https://minio.example.com",
        UsePathStyle:    true,
        AccessKeyID:     "minioadmin",
        SecretAccessKey: "minioadmin",
    },
}
```

## Cloud Run Integration Example

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "base_project/pkg/storage"
    _ "base_project/pkg/storage/gcs"
)

func main() {
    ctx := context.Background()

    // Create client once at startup - all config is required
    client, err := storage.New(&storage.Config{
        Backend: storage.BackendGCS,
        Timeout: 30 * time.Second,
        GCS: &storage.GCSConfig{
            ProjectID:             "my-project",
            UseDefaultCredentials: true,
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Inject into handlers
    handler := &MyHandler{storage: client}
    
    // Graceful shutdown
    go func() {
        sigCh := make(chan os.Signal, 1)
        signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
        <-sigCh
        client.Close()
        os.Exit(0)
    }()

    http.ListenAndServe(":8080", handler)
}

type MyHandler struct {
    storage storage.Client
}

func (h *MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Use h.storage for all requests - it's reused!
}
```

## Next Steps

- See [README.md](README.md) for full documentation
- Check [examples/](examples/) for more code samples
- Review the `Client` interface for all available methods