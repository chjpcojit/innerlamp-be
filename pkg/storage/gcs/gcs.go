// Package gcs provides a Google Cloud Storage implementation of the storage.Client interface.
package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	storagepkg "base_project/pkg/storage"
)

func init() {
	// Register GCS factory with the storage package
	storagepkg.Register(storagepkg.BackendGCS, NewGCSClient)
}

// GCSClient represents a Google Cloud Storage client
type GCSClient struct {
	client    *storage.Client
	config    *storagepkg.Config
	connected bool
}

// NewGCSClient creates a new GCS client instance
func NewGCSClient(config *storagepkg.Config) (storagepkg.Client, error) {
	if config == nil {
		return nil, storagepkg.ErrInvalidConfig
	}

	if config.GCS == nil {
		return nil, fmt.Errorf("%w: GCS configuration is required", storagepkg.ErrInvalidConfig)
	}

	return &GCSClient{
		config:    config,
		connected: false,
	}, nil
}

// Connect establishes a connection to Google Cloud Storage
func (g *GCSClient) Connect(ctx context.Context) error {
	if g.connected {
		return nil
	}

	var opts []option.ClientOption

	// Set credentials
	if g.config.GCS.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(g.config.GCS.CredentialsFile))
	} else if len(g.config.GCS.CredentialsJSON) > 0 {
		opts = append(opts, option.WithCredentialsJSON(g.config.GCS.CredentialsJSON))
	}
	// If UseDefaultCredentials is true and no explicit credentials, let the SDK use ADC

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("%w: %v", storagepkg.ErrConnectionFailed, err)
	}

	g.client = client
	g.connected = true
	return nil
}

// Close closes the GCS client connection
func (g *GCSClient) Close() error {
	if !g.connected || g.client == nil {
		return nil
	}

	err := g.client.Close()
	g.connected = false
	g.client = nil
	return err
}

// IsConnected returns true if the client is connected
func (g *GCSClient) IsConnected() bool {
	return g.connected && g.client != nil
}

// Upload uploads an object to GCS
func (g *GCSClient) Upload(ctx context.Context, bucket, objectName string, reader io.Reader, opts *storagepkg.UploadOptions) (*storagepkg.ObjectInfo, error) {
	if !g.connected || g.client == nil {
		return nil, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return nil, err
	}
	if err := storagepkg.ValidateObjectName(objectName); err != nil {
		return nil, err
	}

	obj := g.client.Bucket(bucket).Object(objectName)
	writer := obj.NewWriter(ctx)

	// Apply upload options
	if opts != nil {
		if opts.ContentType != "" {
			writer.ContentType = opts.ContentType
		} else {
			writer.ContentType = storagepkg.DetectContentType(objectName)
		}

		if opts.Metadata != nil {
			writer.Metadata = opts.Metadata
		}

		if opts.StorageClass != "" {
			writer.StorageClass = string(opts.StorageClass)
		}

		if opts.CacheControl != "" {
			writer.CacheControl = opts.CacheControl
		}

		if opts.ContentEncoding != "" {
			writer.ContentEncoding = opts.ContentEncoding
		}

		if opts.ContentDisposition != "" {
			writer.ContentDisposition = opts.ContentDisposition
		}

		if opts.ContentLanguage != "" {
			writer.ContentLanguage = opts.ContentLanguage
		}

		if opts.ChunkSize > 0 {
			writer.ChunkSize = int(opts.ChunkSize)
		}

		// Handle conditions
		if opts.Conditions != nil {
			if opts.Conditions.DoesNotExist {
				obj = obj.If(storage.Conditions{DoesNotExist: true})
				writer = obj.NewWriter(ctx)
			} else if opts.Conditions.GenerationMatch > 0 {
				obj = obj.If(storage.Conditions{GenerationMatch: opts.Conditions.GenerationMatch})
				writer = obj.NewWriter(ctx)
			}
		}
	} else {
		writer.ContentType = storagepkg.DetectContentType(objectName)
	}

	// Copy data to GCS
	if _, err := io.Copy(writer, reader); err != nil {
		_ = writer.Close() // best-effort cleanup; original error takes precedence
		return nil, fmt.Errorf("%w: %v", storagepkg.ErrUploadFailed, err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("%w: %v", storagepkg.ErrUploadFailed, err)
	}

	// Get object attributes
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", storagepkg.ErrUploadFailed, err)
	}

	return g.attrsToObjectInfo(attrs), nil
}

// Download downloads an object from GCS
func (g *GCSClient) Download(ctx context.Context, bucket, objectName string, writer io.Writer, opts *storagepkg.DownloadOptions) (*storagepkg.ObjectInfo, error) {
	if !g.connected || g.client == nil {
		return nil, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return nil, err
	}
	if err := storagepkg.ValidateObjectName(objectName); err != nil {
		return nil, err
	}

	obj := g.client.Bucket(bucket).Object(objectName)

	// Apply conditions
	if opts != nil && opts.Conditions != nil {
		if opts.Conditions.GenerationMatch > 0 {
			obj = obj.If(storage.Conditions{GenerationMatch: opts.Conditions.GenerationMatch})
		}
	}

	// Get object attributes first
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, storagepkg.ErrObjectNotFound
		}
		return nil, fmt.Errorf("%w: %v", storagepkg.ErrDownloadFailed, err)
	}

	var reader *storage.Reader

	// Handle byte range
	if opts != nil && opts.Range != nil {
		length := opts.Range.End - opts.Range.Start + 1
		if opts.Range.End == -1 {
			length = -1
		}
		reader, err = obj.NewRangeReader(ctx, opts.Range.Start, length)
	} else {
		reader, err = obj.NewReader(ctx)
	}

	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, storagepkg.ErrObjectNotFound
		}
		return nil, fmt.Errorf("%w: %v", storagepkg.ErrDownloadFailed, err)
	}
	defer reader.Close()

	if _, err := io.Copy(writer, reader); err != nil {
		return nil, fmt.Errorf("%w: %v", storagepkg.ErrDownloadFailed, err)
	}

	return g.attrsToObjectInfo(attrs), nil
}

// Delete deletes an object from GCS
func (g *GCSClient) Delete(ctx context.Context, bucket, objectName string) error {
	if !g.connected || g.client == nil {
		return storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return err
	}
	if err := storagepkg.ValidateObjectName(objectName); err != nil {
		return err
	}

	obj := g.client.Bucket(bucket).Object(objectName)
	if err := obj.Delete(ctx); err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return storagepkg.ErrObjectNotFound
		}
		return fmt.Errorf("%w: %v", storagepkg.ErrDeleteFailed, err)
	}

	return nil
}

// GetObjectInfo retrieves metadata about an object
func (g *GCSClient) GetObjectInfo(ctx context.Context, bucket, objectName string) (*storagepkg.ObjectInfo, error) {
	if !g.connected || g.client == nil {
		return nil, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return nil, err
	}
	if err := storagepkg.ValidateObjectName(objectName); err != nil {
		return nil, err
	}

	obj := g.client.Bucket(bucket).Object(objectName)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, storagepkg.ErrObjectNotFound
		}
		return nil, err
	}

	return g.attrsToObjectInfo(attrs), nil
}

// UpdateMetadata updates an object's metadata
func (g *GCSClient) UpdateMetadata(ctx context.Context, bucket, objectName string, metadata map[string]string) (*storagepkg.ObjectInfo, error) {
	if !g.connected || g.client == nil {
		return nil, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return nil, err
	}
	if err := storagepkg.ValidateObjectName(objectName); err != nil {
		return nil, err
	}

	obj := g.client.Bucket(bucket).Object(objectName)
	attrs, err := obj.Update(ctx, storage.ObjectAttrsToUpdate{
		Metadata: metadata,
	})
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, storagepkg.ErrObjectNotFound
		}
		return nil, fmt.Errorf("%w: %v", storagepkg.ErrMetadataUpdateFailed, err)
	}

	return g.attrsToObjectInfo(attrs), nil
}

// Copy copies an object from source to destination
func (g *GCSClient) Copy(ctx context.Context, srcBucket, srcObject, dstBucket, dstObject string, opts *storagepkg.CopyOptions) (*storagepkg.ObjectInfo, error) {
	if !g.connected || g.client == nil {
		return nil, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(srcBucket); err != nil {
		return nil, err
	}
	if err := storagepkg.ValidateBucketName(dstBucket); err != nil {
		return nil, err
	}
	if err := storagepkg.ValidateObjectName(srcObject); err != nil {
		return nil, err
	}
	if err := storagepkg.ValidateObjectName(dstObject); err != nil {
		return nil, err
	}

	src := g.client.Bucket(srcBucket).Object(srcObject)
	dst := g.client.Bucket(dstBucket).Object(dstObject)

	copier := dst.CopierFrom(src)

	// Apply copy options
	if opts != nil {
		if opts.ContentType != "" {
			copier.ContentType = opts.ContentType
		}
		if opts.Metadata != nil {
			copier.Metadata = opts.Metadata
		}
		if opts.StorageClass != "" {
			copier.StorageClass = string(opts.StorageClass)
		}
	}

	attrs, err := copier.Run(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, storagepkg.ErrObjectNotFound
		}
		return nil, fmt.Errorf("%w: %v", storagepkg.ErrCopyFailed, err)
	}

	return g.attrsToObjectInfo(attrs), nil
}

// Exists checks if an object exists
func (g *GCSClient) Exists(ctx context.Context, bucket, objectName string) (bool, error) {
	if !g.connected || g.client == nil {
		return false, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return false, err
	}
	if err := storagepkg.ValidateObjectName(objectName); err != nil {
		return false, err
	}

	obj := g.client.Bucket(bucket).Object(objectName)
	_, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// List lists objects in a bucket
func (g *GCSClient) List(ctx context.Context, bucket string, opts *storagepkg.ListOptions) (*storagepkg.ListResult, error) {
	if !g.connected || g.client == nil {
		return nil, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return nil, err
	}

	query := &storage.Query{}

	if opts != nil {
		if opts.Prefix != "" {
			query.Prefix = opts.Prefix
		}
		if opts.Delimiter != "" {
			query.Delimiter = opts.Delimiter
		}
		if opts.Versions {
			query.Versions = true
		}
	}

	it := g.client.Bucket(bucket).Objects(ctx, query)

	result := &storagepkg.ListResult{
		Objects:  make([]*storagepkg.ObjectInfo, 0),
		Prefixes: make([]string, 0),
	}

	maxResults := 0
	if opts != nil && opts.MaxResults > 0 {
		maxResults = opts.MaxResults
	}

	count := 0
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", storagepkg.ErrListFailed, err)
		}

		// Handle prefixes (common prefixes when using delimiter)
		if attrs.Prefix != "" {
			result.Prefixes = append(result.Prefixes, attrs.Prefix)
			continue
		}

		result.Objects = append(result.Objects, g.attrsToObjectInfo(attrs))
		count++

		if maxResults > 0 && count >= maxResults {
			break
		}
	}

	return result, nil
}

// ListAll lists all objects in a bucket (handles pagination automatically)
func (g *GCSClient) ListAll(ctx context.Context, bucket string, opts *storagepkg.ListOptions) ([]*storagepkg.ObjectInfo, error) {
	if !g.connected || g.client == nil {
		return nil, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return nil, err
	}

	query := &storage.Query{}

	if opts != nil {
		if opts.Prefix != "" {
			query.Prefix = opts.Prefix
		}
		if opts.Delimiter != "" {
			query.Delimiter = opts.Delimiter
		}
		if opts.Versions {
			query.Versions = true
		}
	}

	it := g.client.Bucket(bucket).Objects(ctx, query)

	var objects []*storagepkg.ObjectInfo

	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", storagepkg.ErrListFailed, err)
		}

		// Skip prefixes
		if attrs.Prefix != "" {
			continue
		}

		objects = append(objects, g.attrsToObjectInfo(attrs))
	}

	return objects, nil
}

// CreateBucket creates a new bucket
func (g *GCSClient) CreateBucket(ctx context.Context, bucket string, opts *storagepkg.BucketCreateOptions) error {
	if !g.connected || g.client == nil {
		return storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return err
	}

	bkt := g.client.Bucket(bucket)
	attrs := &storage.BucketAttrs{}

	if opts != nil {
		if opts.Location != "" {
			attrs.Location = opts.Location
		} else if g.config.GCS.Location != "" {
			attrs.Location = g.config.GCS.Location
		}

		if opts.StorageClass != "" {
			attrs.StorageClass = string(opts.StorageClass)
		}

		if opts.Labels != nil {
			attrs.Labels = opts.Labels
		}

		if opts.Versioning {
			attrs.VersioningEnabled = true
		}

		if opts.PublicAccessPrevention {
			attrs.PublicAccessPrevention = storage.PublicAccessPreventionEnforced
		}
	} else if g.config.GCS.Location != "" {
		attrs.Location = g.config.GCS.Location
	}

	if err := bkt.Create(ctx, g.config.GCS.ProjectID, attrs); err != nil {
		if errors.Is(err, storage.ErrBucketNotExist) {
			return storagepkg.ErrBucketNotFound
		}
		// Check for "already exists" error
		if isAlreadyExistsError(err) {
			return storagepkg.ErrBucketAlreadyExists
		}
		return err
	}

	return nil
}

// DeleteBucket deletes a bucket
func (g *GCSClient) DeleteBucket(ctx context.Context, bucket string) error {
	if !g.connected || g.client == nil {
		return storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return err
	}

	bkt := g.client.Bucket(bucket)
	if err := bkt.Delete(ctx); err != nil {
		if errors.Is(err, storage.ErrBucketNotExist) {
			return storagepkg.ErrBucketNotFound
		}
		return fmt.Errorf("%w: %v", storagepkg.ErrDeleteFailed, err)
	}

	return nil
}

// GetBucketInfo retrieves metadata about a bucket
func (g *GCSClient) GetBucketInfo(ctx context.Context, bucket string) (*storagepkg.BucketInfo, error) {
	if !g.connected || g.client == nil {
		return nil, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return nil, err
	}

	bkt := g.client.Bucket(bucket)
	attrs, err := bkt.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrBucketNotExist) {
			return nil, storagepkg.ErrBucketNotFound
		}
		return nil, err
	}

	return &storagepkg.BucketInfo{
		Name:         attrs.Name,
		Location:     attrs.Location,
		StorageClass: storagepkg.StorageClass(attrs.StorageClass),
		Created:      attrs.Created,
		Labels:       attrs.Labels,
		Versioning:   attrs.VersioningEnabled,
	}, nil
}

// BucketExists checks if a bucket exists
func (g *GCSClient) BucketExists(ctx context.Context, bucket string) (bool, error) {
	if !g.connected || g.client == nil {
		return false, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return false, err
	}

	bkt := g.client.Bucket(bucket)
	_, err := bkt.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrBucketNotExist) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// ListBuckets lists all buckets in the project
func (g *GCSClient) ListBuckets(ctx context.Context) ([]*storagepkg.BucketInfo, error) {
	if !g.connected || g.client == nil {
		return nil, storagepkg.ErrNotConnected
	}

	it := g.client.Buckets(ctx, g.config.GCS.ProjectID)

	var buckets []*storagepkg.BucketInfo

	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", storagepkg.ErrListFailed, err)
		}

		buckets = append(buckets, &storagepkg.BucketInfo{
			Name:         attrs.Name,
			Location:     attrs.Location,
			StorageClass: storagepkg.StorageClass(attrs.StorageClass),
			Created:      attrs.Created,
			Labels:       attrs.Labels,
			Versioning:   attrs.VersioningEnabled,
		})
	}

	return buckets, nil
}

// SignedURL generates a signed URL for an object
func (g *GCSClient) SignedURL(ctx context.Context, bucket, objectName string, opts *storagepkg.SignedURLOptions) (string, error) {
	if !g.connected || g.client == nil {
		return "", storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return "", err
	}
	if err := storagepkg.ValidateObjectName(objectName); err != nil {
		return "", err
	}

	if opts == nil {
		return "", fmt.Errorf("%w: SignedURLOptions is required", storagepkg.ErrInvalidConfig)
	}

	if opts.Expires <= 0 {
		return "", fmt.Errorf("%w: Expires duration is required and must be positive", storagepkg.ErrInvalidConfig)
	}

	method := "GET"
	switch opts.Method {
	case storagepkg.SignedURLMethodPUT:
		method = "PUT"
	case storagepkg.SignedURLMethodDELETE:
		method = "DELETE"
	}

	signOpts := &storage.SignedURLOptions{
		Method:  method,
		Expires: time.Now().Add(opts.Expires),
	}

	if opts.ContentType != "" {
		signOpts.ContentType = opts.ContentType
	}

	if opts.Headers != nil {
		for k, v := range opts.Headers {
			signOpts.Headers = append(signOpts.Headers, fmt.Sprintf("%s:%s", k, v))
		}
	}

	if opts.QueryParams != nil {
		queryParams := make(url.Values)
		for k, v := range opts.QueryParams {
			queryParams.Set(k, v)
		}
		signOpts.QueryParameters = queryParams
	}

	url, err := g.client.Bucket(bucket).SignedURL(objectName, signOpts)
	if err != nil {
		return "", fmt.Errorf("%w: %v", storagepkg.ErrSignedURLFailed, err)
	}

	return url, nil
}

// GetBackend returns the storage backend type
func (g *GCSClient) GetBackend() storagepkg.Backend {
	return storagepkg.BackendGCS
}

// GetConfig returns the storage configuration
func (g *GCSClient) GetConfig() *storagepkg.Config {
	return g.config
}

// attrsToObjectInfo converts GCS ObjectAttrs to ObjectInfo
func (g *GCSClient) attrsToObjectInfo(attrs *storage.ObjectAttrs) *storagepkg.ObjectInfo {
	return &storagepkg.ObjectInfo{
		Name:               attrs.Name,
		Bucket:             attrs.Bucket,
		Size:               attrs.Size,
		ContentType:        attrs.ContentType,
		Created:            attrs.Created,
		Updated:            attrs.Updated,
		ETag:               attrs.Etag,
		StorageClass:       storagepkg.StorageClass(attrs.StorageClass),
		Metadata:           attrs.Metadata,
		CacheControl:       attrs.CacheControl,
		ContentEncoding:    attrs.ContentEncoding,
		ContentDisposition: attrs.ContentDisposition,
		ContentLanguage:    attrs.ContentLanguage,
	}
}

// isAlreadyExistsError checks if the error indicates the bucket already exists
func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	// GCS returns a 409 Conflict when bucket already exists
	return errors.Is(err, storage.ErrBucketNotExist) == false &&
		(contains(err.Error(), "already exists") ||
			contains(err.Error(), "409") ||
			contains(err.Error(), "You already own this bucket"))
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
