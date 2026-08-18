// Package storage provides a unified interface for cloud storage operations
// supporting multiple backends (GCP Cloud Storage, AWS S3) with a factory pattern
// for runtime backend selection.
//
// This package is designed for serverless environments (like Cloud Run) where:
// - A single client is created at startup and reused across requests
// - The client handles connection pooling internally
// - No external connection pool management is needed
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

// Common errors
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
	ErrOperationTimeout     = errors.New("operation timed out")
	ErrInvalidContentType   = errors.New("invalid content type")
	ErrFileTooLarge         = errors.New("file exceeds maximum allowed size")
	ErrEmptyObjectName      = errors.New("object name cannot be empty")
	ErrEmptyBucketName      = errors.New("bucket name cannot be empty")
	ErrSignedURLFailed      = errors.New("failed to generate signed URL")
	ErrCopyFailed           = errors.New("copy operation failed")
	ErrMetadataUpdateFailed = errors.New("metadata update failed")
)

// Backend represents the storage backend type
type Backend string

const (
	// BackendGCS uses Google Cloud Storage
	BackendGCS Backend = "gcs"
	// BackendS3 uses Amazon S3
	BackendS3 Backend = "s3"
)

// StorageClass represents the storage class for objects
type StorageClass string

const (
	// GCS Storage Classes
	StorageClassStandard StorageClass = "STANDARD"
	StorageClassNearline StorageClass = "NEARLINE"
	StorageClassColdline StorageClass = "COLDLINE"
	StorageClassArchive  StorageClass = "ARCHIVE"

	// S3 Storage Classes
	StorageClassS3Standard           StorageClass = "STANDARD"
	StorageClassS3IntelligentTiering StorageClass = "INTELLIGENT_TIERING"
	StorageClassS3StandardIA         StorageClass = "STANDARD_IA"
	StorageClassS3OneZoneIA          StorageClass = "ONEZONE_IA"
	StorageClassS3Glacier            StorageClass = "GLACIER"
	StorageClassS3GlacierDeepArchive StorageClass = "DEEP_ARCHIVE"
)

// ACL represents access control level
type ACL string

const (
	ACLPrivate           ACL = "private"
	ACLPublicRead        ACL = "public-read"
	ACLPublicReadWrite   ACL = "public-read-write"
	ACLAuthenticatedRead ACL = "authenticated-read"
	ACLBucketOwnerRead   ACL = "bucket-owner-read"
	ACLBucketOwnerFull   ACL = "bucket-owner-full-control"
)

// SignedURLMethod represents the HTTP method for signed URLs
type SignedURLMethod string

const (
	SignedURLMethodGET    SignedURLMethod = "GET"
	SignedURLMethodPUT    SignedURLMethod = "PUT"
	SignedURLMethodDELETE SignedURLMethod = "DELETE"
)

// Config holds storage configuration
type Config struct {
	Backend Backend
	Timeout time.Duration

	// GCS configuration
	GCS *GCSConfig

	// S3 configuration
	S3 *S3Config

	// Common options
	Options map[string]string
}

// GCSConfig holds Google Cloud Storage specific configuration
type GCSConfig struct {
	// ProjectID is the GCP project ID
	ProjectID string

	// CredentialsFile path to service account JSON file
	CredentialsFile string

	// CredentialsJSON contains the service account JSON directly
	CredentialsJSON []byte

	// UseDefaultCredentials uses Application Default Credentials
	UseDefaultCredentials bool

	// DefaultBucket is the default bucket to use for operations
	DefaultBucket string

	// Location for bucket creation (e.g., "US", "EU", "asia-east1")
	Location string
}

// S3Config holds Amazon S3 specific configuration
type S3Config struct {
	// Region is the AWS region
	Region string

	// AccessKeyID is the AWS access key ID
	AccessKeyID string

	// SecretAccessKey is the AWS secret access key
	SecretAccessKey string

	// SessionToken is the AWS session token (optional, for temporary credentials)
	SessionToken string

	// Endpoint is a custom S3-compatible endpoint (optional)
	Endpoint string

	// UsePathStyle forces path-style addressing (required for some S3-compatible services)
	UsePathStyle bool

	// DefaultBucket is the default bucket to use for operations
	DefaultBucket string
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Backend == "" {
		return fmt.Errorf("%w: backend is required", ErrInvalidConfig)
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("%w: timeout is required and must be positive", ErrInvalidConfig)
	}

	switch c.Backend {
	case BackendGCS:
		if c.GCS == nil {
			return fmt.Errorf("%w: GCS configuration is required", ErrInvalidConfig)
		}
		if c.GCS.ProjectID == "" {
			return fmt.Errorf("%w: GCS project ID is required", ErrInvalidConfig)
		}
		if !c.GCS.UseDefaultCredentials && c.GCS.CredentialsFile == "" && len(c.GCS.CredentialsJSON) == 0 {
			return fmt.Errorf("%w: GCS credentials are required when not using default credentials", ErrInvalidConfig)
		}
	case BackendS3:
		if c.S3 == nil {
			return fmt.Errorf("%w: S3 configuration is required", ErrInvalidConfig)
		}
		if c.S3.Region == "" {
			return fmt.Errorf("%w: S3 region is required", ErrInvalidConfig)
		}
		// AccessKeyID and SecretAccessKey can be empty if using IAM roles
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedBackend, c.Backend)
	}

	return nil
}

// ObjectInfo contains metadata about a stored object
type ObjectInfo struct {
	// Name is the object name/key
	Name string

	// Bucket is the bucket containing the object
	Bucket string

	// Size is the object size in bytes
	Size int64

	// ContentType is the MIME type of the object
	ContentType string

	// Created is when the object was created
	Created time.Time

	// Updated is when the object was last updated
	Updated time.Time

	// ETag is the entity tag of the object
	ETag string

	// StorageClass is the storage class of the object
	StorageClass StorageClass

	// Metadata contains custom metadata
	Metadata map[string]string

	// CacheControl specifies caching directives
	CacheControl string

	// ContentEncoding specifies encoding of the object
	ContentEncoding string

	// ContentDisposition specifies presentation of the object
	ContentDisposition string

	// ContentLanguage specifies language of the object content
	ContentLanguage string
}

// BucketInfo contains metadata about a bucket
type BucketInfo struct {
	// Name is the bucket name
	Name string

	// Location is the geographic location of the bucket
	Location string

	// StorageClass is the default storage class
	StorageClass StorageClass

	// Created is when the bucket was created
	Created time.Time

	// Labels contains bucket labels
	Labels map[string]string

	// Versioning indicates if versioning is enabled
	Versioning bool
}

// UploadOptions specifies options for upload operations
type UploadOptions struct {
	// ContentType is the MIME type (auto-detected if empty)
	ContentType string

	// Metadata is custom metadata to attach to the object
	Metadata map[string]string

	// StorageClass specifies the storage class
	StorageClass StorageClass

	// CacheControl specifies caching directives
	CacheControl string

	// ContentEncoding specifies encoding
	ContentEncoding string

	// ContentDisposition specifies presentation
	ContentDisposition string

	// ContentLanguage specifies language
	ContentLanguage string

	// ACL specifies access control
	ACL ACL

	// ChunkSize for resumable uploads (0 = default)
	ChunkSize int64

	// DisableChunking disables chunked transfer
	DisableChunking bool

	// Conditions for conditional uploads
	Conditions *UploadConditions
}

// UploadConditions specifies conditions for uploads
type UploadConditions struct {
	// DoesNotExist ensures the object doesn't already exist
	DoesNotExist bool

	// GenerationMatch ensures generation matches
	GenerationMatch int64

	// MetagenerationMatch ensures metageneration matches
	MetagenerationMatch int64
}

// DownloadOptions specifies options for download operations
type DownloadOptions struct {
	// Range specifies byte range to download
	Range *ByteRange

	// Conditions for conditional downloads
	Conditions *DownloadConditions
}

// ByteRange specifies a byte range for partial downloads
type ByteRange struct {
	Start int64
	End   int64 // -1 means to the end
}

// DownloadConditions specifies conditions for downloads
type DownloadConditions struct {
	// GenerationMatch ensures generation matches
	GenerationMatch int64

	// IfModifiedSince only downloads if modified since
	IfModifiedSince time.Time

	// IfUnmodifiedSince only downloads if unmodified since
	IfUnmodifiedSince time.Time
}

// ListOptions specifies options for list operations
type ListOptions struct {
	// Prefix filters objects by prefix
	Prefix string

	// Delimiter for hierarchical listing (use "/" for directory-like listing)
	Delimiter string

	// MaxResults limits the number of results (0 = no limit)
	MaxResults int

	// PageToken for pagination
	PageToken string

	// IncludeMetadata includes full metadata in results
	IncludeMetadata bool

	// Versions includes all object versions (if versioning enabled)
	Versions bool
}

// ListResult contains the result of a list operation
type ListResult struct {
	// Objects is the list of objects
	Objects []*ObjectInfo

	// Prefixes is the list of common prefixes (when using delimiter)
	Prefixes []string

	// NextPageToken is the token for the next page (empty if no more pages)
	NextPageToken string
}

// CopyOptions specifies options for copy operations
type CopyOptions struct {
	// ContentType overrides the content type
	ContentType string

	// Metadata overrides metadata (nil = keep original)
	Metadata map[string]string

	// StorageClass specifies the storage class for the copy
	StorageClass StorageClass

	// DestinationACL specifies ACL for the copy
	DestinationACL ACL
}

// SignedURLOptions specifies options for generating signed URLs
type SignedURLOptions struct {
	// Method is the HTTP method (GET, PUT, DELETE)
	Method SignedURLMethod

	// Expires is when the URL expires
	Expires time.Duration

	// ContentType restricts uploads to this content type (for PUT)
	ContentType string

	// Headers to include in the signature
	Headers map[string]string

	// QueryParams to include in the URL
	QueryParams map[string]string
}

// BucketCreateOptions specifies options for bucket creation
type BucketCreateOptions struct {
	// Location specifies the geographic location
	Location string

	// StorageClass specifies the default storage class
	StorageClass StorageClass

	// Labels specifies bucket labels
	Labels map[string]string

	// Versioning enables object versioning
	Versioning bool

	// PublicAccessPrevention prevents public access
	PublicAccessPrevention bool
}

// Client is the main interface for cloud storage operations
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

// FactoryFunc is the function type for creating storage client instances
type FactoryFunc func(*Config) (Client, error)

var (
	factories = make(map[Backend]FactoryFunc)
)

// Register registers a storage backend factory function
func Register(backend Backend, factory FactoryFunc) {
	factories[backend] = factory
}

// New creates a new storage client based on configuration.
// All configuration must be explicitly provided - no defaults are applied.
func New(config *Config) (Client, error) {
	if config == nil {
		return nil, fmt.Errorf("%w: config cannot be nil", ErrInvalidConfig)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Get factory for the backend type
	factory, ok := factories[config.Backend]
	if !ok {
		return nil, fmt.Errorf("%w: %s (ensure you import the backend package)", ErrUnsupportedBackend, config.Backend)
	}

	return factory(config)
}

// Helper functions

// DetectContentType attempts to detect content type from file extension
func DetectContentType(objectName string) string {
	ext := ""
	for i := len(objectName) - 1; i >= 0 && objectName[i] != '/'; i-- {
		if objectName[i] == '.' {
			ext = objectName[i:]
			break
		}
	}

	contentTypes := map[string]string{
		".html":  "text/html",
		".htm":   "text/html",
		".css":   "text/css",
		".js":    "application/javascript",
		".json":  "application/json",
		".xml":   "application/xml",
		".txt":   "text/plain",
		".csv":   "text/csv",
		".pdf":   "application/pdf",
		".zip":   "application/zip",
		".gz":    "application/gzip",
		".tar":   "application/x-tar",
		".png":   "image/png",
		".jpg":   "image/jpeg",
		".jpeg":  "image/jpeg",
		".gif":   "image/gif",
		".svg":   "image/svg+xml",
		".webp":  "image/webp",
		".ico":   "image/x-icon",
		".mp3":   "audio/mpeg",
		".wav":   "audio/wav",
		".mp4":   "video/mp4",
		".webm":  "video/webm",
		".woff":  "font/woff",
		".woff2": "font/woff2",
		".ttf":   "font/ttf",
		".otf":   "font/otf",
		".eot":   "application/vnd.ms-fontobject",
		".yaml":  "application/x-yaml",
		".yml":   "application/x-yaml",
		".md":    "text/markdown",
		".go":    "text/x-go",
		".py":    "text/x-python",
		".rb":    "text/x-ruby",
		".java":  "text/x-java",
		".c":     "text/x-c",
		".cpp":   "text/x-c++",
		".h":     "text/x-c",
		".rs":    "text/x-rust",
		".ts":    "application/typescript",
		".tsx":   "application/typescript",
		".jsx":   "application/javascript",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}

	return "application/octet-stream"
}

// ValidateObjectName validates an object name
func ValidateObjectName(name string) error {
	if name == "" {
		return ErrEmptyObjectName
	}

	// Check for invalid characters or patterns
	if len(name) > 1024 {
		return fmt.Errorf("%w: object name exceeds 1024 characters", ErrInvalidObjectName)
	}

	// Check for consecutive slashes
	for i := 0; i < len(name)-1; i++ {
		if name[i] == '/' && name[i+1] == '/' {
			return fmt.Errorf("%w: object name contains consecutive slashes", ErrInvalidObjectName)
		}
	}

	return nil
}

// ValidateBucketName validates a bucket name
func ValidateBucketName(name string) error {
	if name == "" {
		return ErrEmptyBucketName
	}

	if len(name) < 3 || len(name) > 63 {
		return fmt.Errorf("%w: bucket name must be 3-63 characters", ErrInvalidConfig)
	}

	return nil
}

// CopyMetadata creates a copy of metadata
func CopyMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return make(map[string]string)
	}
	result := make(map[string]string, len(metadata))
	for k, v := range metadata {
		result[k] = v
	}
	return result
}

// MergeMetadata merges multiple metadata maps
func MergeMetadata(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
