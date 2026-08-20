// Package examples provides usage examples for the storage package.
// These examples demonstrate common patterns for using GCS and S3 backends.
package examples

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"chjpcoj.com.innerlamp/pkg/storage"
	_ "chjpcoj.com.innerlamp/pkg/storage/gcs" // Register GCS backend
	_ "chjpcoj.com.innerlamp/pkg/storage/s3"  // Register S3 backend
)

// ExampleGCSBasic demonstrates basic GCS operations.
func ExampleGCSBasic() {
	ctx := context.Background()

	// Create GCS configuration - all fields must be explicitly set
	config := &storage.Config{
		Backend: storage.BackendGCS,
		Timeout: 30 * time.Second, // Required: must be positive
		GCS: &storage.GCSConfig{
			ProjectID:             "my-gcp-project-id", // Required
			UseDefaultCredentials: true,                // Uses Application Default Credentials
			Location:              "US",                // Used for bucket creation
		},
	}

	// Create and connect client
	client, err := storage.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Upload a file
	content := strings.NewReader("Hello, Google Cloud Storage!")
	info, err := client.Upload(ctx, "my-bucket", "hello.txt", content, &storage.UploadOptions{
		ContentType: "text/plain",
		Metadata: map[string]string{
			"author":  "example",
			"version": "1.0",
		},
	})
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}
	log.Printf("Uploaded: %s (%d bytes)", info.Name, info.Size)

	// Download the file
	var buf bytes.Buffer
	_, err = client.Download(ctx, "my-bucket", "hello.txt", &buf, nil)
	if err != nil {
		log.Fatalf("Download failed: %v", err)
	}
	log.Printf("Downloaded content: %s", buf.String())

	// Delete the file
	if err := client.Delete(ctx, "my-bucket", "hello.txt"); err != nil {
		log.Fatalf("Delete failed: %v", err)
	}
	log.Println("File deleted successfully")
}

// ExampleS3Basic demonstrates basic S3 operations.
func ExampleS3Basic() {
	ctx := context.Background()

	// Create S3 configuration - all fields must be explicitly set
	config := &storage.Config{
		Backend: storage.BackendS3,
		Timeout: 30 * time.Second, // Required: must be positive
		S3: &storage.S3Config{
			Region:          "us-east-1",              // Required
			AccessKeyID:     "YOUR_ACCESS_KEY_ID",     // Required if not using IAM roles
			SecretAccessKey: "YOUR_SECRET_ACCESS_KEY", // Required if not using IAM roles
		},
	}

	// Create and connect client
	client, err := storage.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Upload a file
	content := strings.NewReader("Hello, Amazon S3!")
	info, err := client.Upload(ctx, "my-bucket", "hello.txt", content, nil)
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}
	log.Printf("Uploaded: %s", info.Name)

	// Download the file
	var buf bytes.Buffer
	_, err = client.Download(ctx, "my-bucket", "hello.txt", &buf, nil)
	if err != nil {
		log.Fatalf("Download failed: %v", err)
	}
	log.Printf("Downloaded content: %s", buf.String())
}

// ExampleS3Compatible demonstrates usage with S3-compatible services (MinIO, DigitalOcean Spaces, etc.).
func ExampleS3Compatible() {
	ctx := context.Background()

	// Create configuration for S3-compatible service - all fields must be explicitly set
	config := &storage.Config{
		Backend: storage.BackendS3,
		Timeout: 30 * time.Second, // Required: must be positive
		S3: &storage.S3Config{
			Region:          "us-east-1",             // Required
			AccessKeyID:     "minioadmin",            // Required
			SecretAccessKey: "minioadmin",            // Required
			Endpoint:        "http://localhost:9000", // MinIO endpoint
			UsePathStyle:    true,                    // Required for most S3-compatible services
		},
	}

	client, err := storage.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Use like regular S3
	content := strings.NewReader("Hello, MinIO!")
	_, err = client.Upload(ctx, "test-bucket", "hello.txt", content, nil)
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}
}

// ExampleUploadWithOptions demonstrates advanced upload options.
func ExampleUploadWithOptions() {
	ctx := context.Background()

	// Assuming client is already created and connected
	var client storage.Client
	// client = ...

	// Open a file
	file, err := os.Open("document.pdf")
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Upload with full options
	opts := &storage.UploadOptions{
		ContentType:        "application/pdf",
		StorageClass:       storage.StorageClassStandard,
		CacheControl:       "max-age=86400",
		ContentDisposition: "attachment; filename=\"document.pdf\"",
		ContentLanguage:    "en-US",
		ACL:                storage.ACLPrivate,
		Metadata: map[string]string{
			"document-type": "invoice",
			"customer-id":   "12345",
			"year":          "2024",
		},
		Conditions: &storage.UploadConditions{
			DoesNotExist: true, // Only upload if object doesn't exist
		},
	}

	info, err := client.Upload(ctx, "documents-bucket", "invoices/2024/invoice-001.pdf", file, opts)
	if err != nil {
		if errors.Is(err, storage.ErrBucketAlreadyExists) {
			log.Println("Object already exists")
			return
		}
		log.Fatalf("Upload failed: %v", err)
	}

	log.Printf("Uploaded: %s (Size: %d, ETag: %s)", info.Name, info.Size, info.ETag)
}

// ExampleDownloadWithRange demonstrates partial file downloads.
func ExampleDownloadWithRange() {
	ctx := context.Background()

	var client storage.Client
	// client = ...

	// Download first 1MB of a large file
	opts := &storage.DownloadOptions{
		Range: &storage.ByteRange{
			Start: 0,
			End:   1024*1024 - 1, // First 1MB
		},
	}

	var buf bytes.Buffer
	_, err := client.Download(ctx, "my-bucket", "large-file.zip", &buf, opts)
	if err != nil {
		log.Fatalf("Download failed: %v", err)
	}
	log.Printf("Downloaded %d bytes", buf.Len())

	// Download to a file
	file, err := os.Create("downloaded-file.zip")
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	_, err = client.Download(ctx, "my-bucket", "file.zip", file, nil)
	if err != nil {
		log.Fatalf("Download failed: %v", err)
	}
}

// ExampleListObjects demonstrates listing objects with various options.
func ExampleListObjects() {
	ctx := context.Background()

	var client storage.Client
	// client = ...

	// List all objects with a prefix
	result, err := client.List(ctx, "my-bucket", &storage.ListOptions{
		Prefix:     "images/",
		MaxResults: 100,
	})
	if err != nil {
		log.Fatalf("List failed: %v", err)
	}

	for _, obj := range result.Objects {
		fmt.Printf("Object: %s (Size: %d, Modified: %s)\n",
			obj.Name, obj.Size, obj.Updated.Format(time.RFC3339))
	}

	// List with delimiter for directory-like structure
	result, err = client.List(ctx, "my-bucket", &storage.ListOptions{
		Prefix:    "data/",
		Delimiter: "/",
	})
	if err != nil {
		log.Fatalf("List failed: %v", err)
	}

	fmt.Println("Files:")
	for _, obj := range result.Objects {
		fmt.Printf("  %s\n", obj.Name)
	}

	fmt.Println("Directories:")
	for _, prefix := range result.Prefixes {
		fmt.Printf("  %s\n", prefix)
	}

	// Handle pagination
	var allObjects []*storage.ObjectInfo
	opts := &storage.ListOptions{Prefix: "logs/"}

	for {
		result, err := client.List(ctx, "my-bucket", opts)
		if err != nil {
			log.Fatalf("List failed: %v", err)
		}

		allObjects = append(allObjects, result.Objects...)

		if result.NextPageToken == "" {
			break
		}
		opts.PageToken = result.NextPageToken
	}

	fmt.Printf("Total objects: %d\n", len(allObjects))

	// Or use ListAll for automatic pagination
	allObjects, err = client.ListAll(ctx, "my-bucket", &storage.ListOptions{
		Prefix: "logs/",
	})
	if err != nil {
		log.Fatalf("ListAll failed: %v", err)
	}
	fmt.Printf("Total objects: %d\n", len(allObjects))
}

// ExampleSignedURLs demonstrates generating pre-signed URLs.
// SignedURLOptions is required - you must specify Method and Expires.
func ExampleSignedURLs() {
	ctx := context.Background()

	var client storage.Client
	// client = ...

	// Generate download URL (valid for 15 minutes)
	// SignedURLOptions is required - Method and Expires must be set
	downloadURL, err := client.SignedURL(ctx, "my-bucket", "private/report.pdf", &storage.SignedURLOptions{
		Method:  storage.SignedURLMethodGET, // Required
		Expires: 15 * time.Minute,           // Required: must be positive duration
	})
	if err != nil {
		log.Fatalf("Failed to generate signed URL: %v", err)
	}
	fmt.Printf("Download URL: %s\n", downloadURL)

	// Generate upload URL (valid for 1 hour)
	// SignedURLOptions is required - Method and Expires must be set
	uploadURL, err := client.SignedURL(ctx, "my-bucket", "uploads/new-file.jpg", &storage.SignedURLOptions{
		Method:      storage.SignedURLMethodPUT, // Required
		Expires:     1 * time.Hour,              // Required: must be positive duration
		ContentType: "image/jpeg",
	})
	if err != nil {
		log.Fatalf("Failed to generate signed URL: %v", err)
	}
	fmt.Printf("Upload URL: %s\n", uploadURL)

	// Generate delete URL
	// SignedURLOptions is required - Method and Expires must be set
	deleteURL, err := client.SignedURL(ctx, "my-bucket", "temp/old-file.txt", &storage.SignedURLOptions{
		Method:  storage.SignedURLMethodDELETE, // Required
		Expires: 5 * time.Minute,               // Required: must be positive duration
	})
	if err != nil {
		log.Fatalf("Failed to generate signed URL: %v", err)
	}
	fmt.Printf("Delete URL: %s\n", deleteURL)
}

// ExampleBucketOperations demonstrates bucket management.
func ExampleBucketOperations() {
	ctx := context.Background()

	var client storage.Client
	// client = ...

	// Create a new bucket
	err := client.CreateBucket(ctx, "new-bucket", &storage.BucketCreateOptions{
		Location:     "us-central1",
		StorageClass: storage.StorageClassStandard,
		Versioning:   true,
		Labels: map[string]string{
			"environment": "production",
			"team":        "backend",
		},
		PublicAccessPrevention: true,
	})
	if err != nil {
		if errors.Is(err, storage.ErrBucketAlreadyExists) {
			log.Println("Bucket already exists")
		} else {
			log.Fatalf("Failed to create bucket: %v", err)
		}
	}

	// Check if bucket exists
	exists, err := client.BucketExists(ctx, "my-bucket")
	if err != nil {
		log.Fatalf("Failed to check bucket: %v", err)
	}
	fmt.Printf("Bucket exists: %v\n", exists)

	// Get bucket information
	info, err := client.GetBucketInfo(ctx, "my-bucket")
	if err != nil {
		log.Fatalf("Failed to get bucket info: %v", err)
	}
	fmt.Printf("Bucket: %s\n", info.Name)
	fmt.Printf("Location: %s\n", info.Location)
	fmt.Printf("Storage Class: %s\n", info.StorageClass)
	fmt.Printf("Versioning: %v\n", info.Versioning)

	// List all buckets
	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		log.Fatalf("Failed to list buckets: %v", err)
	}
	for _, b := range buckets {
		fmt.Printf("- %s (Location: %s, Created: %s)\n",
			b.Name, b.Location, b.Created.Format(time.RFC3339))
	}

	// Delete bucket (must be empty)
	err = client.DeleteBucket(ctx, "old-bucket")
	if err != nil {
		if errors.Is(err, storage.ErrBucketNotFound) {
			log.Println("Bucket not found")
		} else {
			log.Fatalf("Failed to delete bucket: %v", err)
		}
	}
}

// ExampleCopyAndMove demonstrates copying and moving objects.
func ExampleCopyAndMove() {
	ctx := context.Background()

	var client storage.Client
	// client = ...

	// Copy within same bucket
	_, err := client.Copy(ctx,
		"my-bucket", "original.txt",
		"my-bucket", "copy.txt",
		nil,
	)
	if err != nil {
		log.Fatalf("Copy failed: %v", err)
	}

	// Copy to different bucket with options
	_, err = client.Copy(ctx,
		"source-bucket", "data/file.json",
		"archive-bucket", "archived/file.json",
		&storage.CopyOptions{
			StorageClass: storage.StorageClassNearline,
			Metadata: map[string]string{
				"archived-at": time.Now().Format(time.RFC3339),
				"source":      "source-bucket",
			},
		},
	)
	if err != nil {
		log.Fatalf("Copy failed: %v", err)
	}

	// Move (copy + delete original)
	_, err = client.Copy(ctx,
		"bucket", "old-path/file.txt",
		"bucket", "new-path/file.txt",
		nil,
	)
	if err != nil {
		log.Fatalf("Copy failed: %v", err)
	}

	err = client.Delete(ctx, "bucket", "old-path/file.txt")
	if err != nil {
		log.Fatalf("Delete failed: %v", err)
	}
}

// ExampleMetadata demonstrates metadata operations.
func ExampleMetadata() {
	ctx := context.Background()

	var client storage.Client
	// client = ...

	// Get object info/metadata
	info, err := client.GetObjectInfo(ctx, "my-bucket", "document.pdf")
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotFound) {
			log.Println("Object not found")
			return
		}
		log.Fatalf("Failed to get object info: %v", err)
	}

	fmt.Printf("Name: %s\n", info.Name)
	fmt.Printf("Size: %d bytes\n", info.Size)
	fmt.Printf("Content-Type: %s\n", info.ContentType)
	fmt.Printf("ETag: %s\n", info.ETag)
	fmt.Printf("Created: %s\n", info.Created.Format(time.RFC3339))
	fmt.Printf("Updated: %s\n", info.Updated.Format(time.RFC3339))
	fmt.Printf("Storage Class: %s\n", info.StorageClass)
	fmt.Println("Custom Metadata:")
	for k, v := range info.Metadata {
		fmt.Printf("  %s: %s\n", k, v)
	}

	// Update metadata
	newMetadata := map[string]string{
		"reviewed":    "true",
		"reviewed-by": "john.doe",
		"reviewed-at": time.Now().Format(time.RFC3339),
	}

	updatedInfo, err := client.UpdateMetadata(ctx, "my-bucket", "document.pdf", newMetadata)
	if err != nil {
		log.Fatalf("Failed to update metadata: %v", err)
	}
	fmt.Printf("Metadata updated at: %s\n", updatedInfo.Updated.Format(time.RFC3339))
}

// ExampleCloudRunIntegration demonstrates best practices for Cloud Run deployments.
// The client is created once at startup and reused across all requests.
// All configuration must be explicitly provided - no defaults are applied.
func ExampleCloudRunIntegration() {
	ctx := context.Background()

	// Create client ONCE at startup (not per request)
	// All configuration fields must be explicitly set
	config := &storage.Config{
		Backend: storage.BackendGCS,
		Timeout: 30 * time.Second, // Required: must be positive
		GCS: &storage.GCSConfig{
			ProjectID:             "my-project", // Required
			UseDefaultCredentials: true,
		},
	}

	client, err := storage.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close() // Close on shutdown

	// Create handler with injected client
	handler := &FileHandler{storage: client}

	// The same client instance handles all incoming requests
	// Cloud Run instances are reused, so this is efficient
	log.Println("Server starting - storage client will be reused across requests")
	_ = handler // In real code: http.ListenAndServe(":8080", handler)
}

// FileHandler demonstrates dependency injection of storage client
type FileHandler struct {
	storage storage.Client
}

// HandleUpload shows how the injected client is reused per request
func (h *FileHandler) HandleUpload(ctx context.Context, bucket, name string, data io.Reader) error {
	// Reuse the same client for every request - no new connections needed
	_, err := h.storage.Upload(ctx, bucket, name, data, nil)
	return err
}

// HandleDownload shows downloading with the reused client
func (h *FileHandler) HandleDownload(ctx context.Context, bucket, name string, w io.Writer) error {
	_, err := h.storage.Download(ctx, bucket, name, w, nil)
	return err
}

// ExampleErrorHandling demonstrates proper error handling patterns.
// Note: storage.ErrInvalidConfig is returned when required configuration is missing.
func ExampleErrorHandling() {
	ctx := context.Background()

	var client storage.Client
	// client = ...

	// Handle specific errors
	info, err := client.GetObjectInfo(ctx, "bucket", "file.txt")
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrObjectNotFound):
			log.Println("Object does not exist")
		case errors.Is(err, storage.ErrBucketNotFound):
			log.Println("Bucket does not exist")
		case errors.Is(err, storage.ErrAccessDenied):
			log.Println("Permission denied - check credentials")
		case errors.Is(err, storage.ErrNotConnected):
			log.Println("Client not connected - call Connect() first")
		default:
			log.Printf("Unexpected error: %v", err)
		}
		return
	}
	fmt.Printf("Object: %s (%d bytes)\n", info.Name, info.Size)

	// Handle upload errors
	content := strings.NewReader("data")
	_, err = client.Upload(ctx, "bucket", "file.txt", content, nil)
	if err != nil {
		if errors.Is(err, storage.ErrUploadFailed) {
			log.Printf("Upload failed: %v", err)
		}
		return
	}

	// Handle download errors
	var buf bytes.Buffer
	_, err = client.Download(ctx, "bucket", "file.txt", &buf, nil)
	if err != nil {
		if errors.Is(err, storage.ErrDownloadFailed) {
			log.Printf("Download failed: %v", err)
		}
		return
	}
}

// ExampleStreamUpload demonstrates uploading from a stream/reader.
func ExampleStreamUpload() {
	ctx := context.Background()

	var client storage.Client
	// client = ...

	// Upload from string
	content := strings.NewReader("Hello, World!")
	_, err := client.Upload(ctx, "bucket", "hello.txt", content, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Upload from bytes
	data := []byte(`{"key": "value"}`)
	_, err = client.Upload(ctx, "bucket", "data.json", bytes.NewReader(data), &storage.UploadOptions{
		ContentType: "application/json",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Upload from file
	file, err := os.Open("local-file.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = client.Upload(ctx, "bucket", "remote-file.txt", file, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Upload from pipe (for streaming data)
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		// Write data to pipe
		for i := 0; i < 10; i++ {
			fmt.Fprintf(pw, "Line %d\n", i)
		}
	}()

	_, err = client.Upload(ctx, "bucket", "streamed.txt", pr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

// ExampleGCSWithServiceAccount demonstrates GCS with explicit credentials.
// All configuration must be explicitly provided - no defaults are applied.
func ExampleGCSWithServiceAccount() {
	ctx := context.Background()

	// Using credentials file path - all fields must be set
	configWithFile := &storage.Config{
		Backend: storage.BackendGCS,
		Timeout: 30 * time.Second, // Required: must be positive
		GCS: &storage.GCSConfig{
			ProjectID:       "my-project",                    // Required
			CredentialsFile: "/path/to/service-account.json", // Required when not using default credentials
		},
	}

	client, err := storage.New(configWithFile)
	if err != nil {
		log.Fatal(err)
	}
	if err := client.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Or using credentials JSON directly - all fields must be set
	credentialsJSON := []byte(`{
		"type": "service_account",
		"project_id": "my-project",
		"private_key_id": "...",
		"private_key": "...",
		"client_email": "...",
		"client_id": "...",
		...
	}`)

	configWithJSON := &storage.Config{
		Backend: storage.BackendGCS,
		Timeout: 30 * time.Second, // Required: must be positive
		GCS: &storage.GCSConfig{
			ProjectID:       "my-project",    // Required
			CredentialsJSON: credentialsJSON, // Required when not using default credentials
		},
	}

	client2, err := storage.New(configWithJSON)
	if err != nil {
		log.Fatal(err)
	}
	if err := client2.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer client2.Close()
}

// ExampleBackendAgnostic demonstrates writing backend-agnostic code.
// All configuration must be explicitly provided - no defaults are applied.
func ExampleBackendAgnostic() {
	ctx := context.Background()

	// This function works with any backend
	uploadFile := func(client storage.Client, bucket, name string, data []byte) error {
		_, err := client.Upload(ctx, bucket, name, bytes.NewReader(data), &storage.UploadOptions{
			ContentType: storage.DetectContentType(name),
		})
		return err
	}

	downloadFile := func(client storage.Client, bucket, name string) ([]byte, error) {
		var buf bytes.Buffer
		_, err := client.Download(ctx, bucket, name, &buf, nil)
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	// Use with GCS - all configuration must be explicitly set
	gcsClient, _ := storage.New(&storage.Config{
		Backend: storage.BackendGCS,
		Timeout: 30 * time.Second, // Required
		GCS: &storage.GCSConfig{
			ProjectID:             "my-project", // Required
			UseDefaultCredentials: true,
		},
	})
	_ = gcsClient.Connect(ctx) // #nosec
	defer func() { _ = gcsClient.Close() }()

	_ = uploadFile(gcsClient, "gcs-bucket", "file.txt", []byte("data")) // #nosec
	data, _ := downloadFile(gcsClient, "gcs-bucket", "file.txt")
	fmt.Printf("GCS: %s\n", data)

	// Use with S3 - all configuration must be explicitly set
	s3Client, _ := storage.New(&storage.Config{
		Backend: storage.BackendS3,
		Timeout: 30 * time.Second, // Required
		S3: &storage.S3Config{
			Region: "us-east-1", // Required
		},
	})
	_ = s3Client.Connect(ctx) // #nosec
	defer func() { _ = s3Client.Close() }()

	_ = uploadFile(s3Client, "s3-bucket", "file.txt", []byte("data")) // #nosec
	data, _ = downloadFile(s3Client, "s3-bucket", "file.txt")
	fmt.Printf("S3: %s\n", data)
}
