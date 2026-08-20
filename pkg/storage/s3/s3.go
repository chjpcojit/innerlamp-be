// Package s3 provides an Amazon S3 implementation of the storage.Client interface.
package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	storagepkg "chjpcoj.com.innerlamp/pkg/storage"
)

func init() {
	// Register S3 factory with the storage package
	storagepkg.Register(storagepkg.BackendS3, NewS3Client)
}

// S3Client represents an Amazon S3 client
type S3Client struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	config        *storagepkg.Config
	connected     bool
}

// NewS3Client creates a new S3 client instance
func NewS3Client(cfg *storagepkg.Config) (storagepkg.Client, error) {
	if cfg == nil {
		return nil, storagepkg.ErrInvalidConfig
	}

	if cfg.S3 == nil {
		return nil, fmt.Errorf("%w: S3 configuration is required", storagepkg.ErrInvalidConfig)
	}

	return &S3Client{
		config:    cfg,
		connected: false,
	}, nil
}

// Connect establishes a connection to Amazon S3
func (s *S3Client) Connect(ctx context.Context) error {
	if s.connected {
		return nil
	}

	var opts []func(*config.LoadOptions) error

	// Set region
	if s.config.S3.Region != "" {
		opts = append(opts, config.WithRegion(s.config.S3.Region))
	}

	// Set credentials if provided
	if s.config.S3.AccessKeyID != "" && s.config.S3.SecretAccessKey != "" {
		creds := credentials.NewStaticCredentialsProvider(
			s.config.S3.AccessKeyID,
			s.config.S3.SecretAccessKey,
			s.config.S3.SessionToken,
		)
		opts = append(opts, config.WithCredentialsProvider(creds))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("%w: %v", storagepkg.ErrConnectionFailed, err)
	}

	// Create S3 client options
	s3Opts := []func(*s3.Options){}

	// Set custom endpoint if provided
	if s.config.S3.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(s.config.S3.Endpoint)
		})
	}

	// Set path-style addressing if required
	if s.config.S3.UsePathStyle {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	s.client = s3.NewFromConfig(cfg, s3Opts...)
	s.presignClient = s3.NewPresignClient(s.client)
	s.connected = true

	return nil
}

// Close closes the S3 client connection
func (s *S3Client) Close() error {
	if !s.connected || s.client == nil {
		return nil
	}

	s.connected = false
	s.client = nil
	s.presignClient = nil
	return nil
}

// IsConnected returns true if the client is connected
func (s *S3Client) IsConnected() bool {
	return s.connected && s.client != nil
}

// Upload uploads an object to S3
func (s *S3Client) Upload(ctx context.Context, bucket, objectName string, reader io.Reader, opts *storagepkg.UploadOptions) (*storagepkg.ObjectInfo, error) {
	if !s.connected || s.client == nil {
		return nil, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return nil, err
	}
	if err := storagepkg.ValidateObjectName(objectName); err != nil {
		return nil, err
	}

	// Read content into buffer to get size
	var buf bytes.Buffer
	size, err := io.Copy(&buf, reader)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read content: %v", storagepkg.ErrUploadFailed, err)
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectName),
		Body:   bytes.NewReader(buf.Bytes()),
	}

	// Apply upload options
	if opts != nil {
		if opts.ContentType != "" {
			input.ContentType = aws.String(opts.ContentType)
		} else {
			contentType := storagepkg.DetectContentType(objectName)
			input.ContentType = aws.String(contentType)
		}

		if opts.Metadata != nil {
			input.Metadata = opts.Metadata
		}

		if opts.StorageClass != "" {
			input.StorageClass = types.StorageClass(opts.StorageClass)
		}

		if opts.CacheControl != "" {
			input.CacheControl = aws.String(opts.CacheControl)
		}

		if opts.ContentEncoding != "" {
			input.ContentEncoding = aws.String(opts.ContentEncoding)
		}

		if opts.ContentDisposition != "" {
			input.ContentDisposition = aws.String(opts.ContentDisposition)
		}

		if opts.ContentLanguage != "" {
			input.ContentLanguage = aws.String(opts.ContentLanguage)
		}

		if opts.ACL != "" {
			input.ACL = s.mapACL(opts.ACL)
		}
	} else {
		contentType := storagepkg.DetectContentType(objectName)
		input.ContentType = aws.String(contentType)
	}

	_, err = s.client.PutObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", storagepkg.ErrUploadFailed, s.mapError(err))
	}

	// Get object info after upload
	return &storagepkg.ObjectInfo{
		Name:        objectName,
		Bucket:      bucket,
		Size:        size,
		ContentType: aws.ToString(input.ContentType),
		Metadata:    opts.Metadata,
		Updated:     time.Now(),
	}, nil
}

// Download downloads an object from S3
func (s *S3Client) Download(ctx context.Context, bucket, objectName string, writer io.Writer, opts *storagepkg.DownloadOptions) (*storagepkg.ObjectInfo, error) {
	if !s.connected || s.client == nil {
		return nil, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return nil, err
	}
	if err := storagepkg.ValidateObjectName(objectName); err != nil {
		return nil, err
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectName),
	}

	// Handle byte range
	if opts != nil && opts.Range != nil {
		rangeStr := fmt.Sprintf("bytes=%d-", opts.Range.Start)
		if opts.Range.End >= 0 {
			rangeStr = fmt.Sprintf("bytes=%d-%d", opts.Range.Start, opts.Range.End)
		}
		input.Range = aws.String(rangeStr)
	}

	// Apply conditions
	if opts != nil && opts.Conditions != nil {
		if !opts.Conditions.IfModifiedSince.IsZero() {
			input.IfModifiedSince = aws.Time(opts.Conditions.IfModifiedSince)
		}
		if !opts.Conditions.IfUnmodifiedSince.IsZero() {
			input.IfUnmodifiedSince = aws.Time(opts.Conditions.IfUnmodifiedSince)
		}
	}

	output, err := s.client.GetObject(ctx, input)
	if err != nil {
		mappedErr := s.mapError(err)
		if errors.Is(mappedErr, storagepkg.ErrObjectNotFound) {
			return nil, storagepkg.ErrObjectNotFound
		}
		return nil, fmt.Errorf("%w: %v", storagepkg.ErrDownloadFailed, mappedErr)
	}
	defer output.Body.Close()

	if _, err := io.Copy(writer, output.Body); err != nil {
		return nil, fmt.Errorf("%w: failed to write content: %v", storagepkg.ErrDownloadFailed, err)
	}

	return s.outputToObjectInfo(bucket, objectName, output), nil
}

// Delete deletes an object from S3
func (s *S3Client) Delete(ctx context.Context, bucket, objectName string) error {
	if !s.connected || s.client == nil {
		return storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return err
	}
	if err := storagepkg.ValidateObjectName(objectName); err != nil {
		return err
	}

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectName),
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("%w: %v", storagepkg.ErrDeleteFailed, s.mapError(err))
	}

	return nil
}

// GetObjectInfo retrieves metadata about an object
func (s *S3Client) GetObjectInfo(ctx context.Context, bucket, objectName string) (*storagepkg.ObjectInfo, error) {
	if !s.connected || s.client == nil {
		return nil, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return nil, err
	}
	if err := storagepkg.ValidateObjectName(objectName); err != nil {
		return nil, err
	}

	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectName),
	}

	output, err := s.client.HeadObject(ctx, input)
	if err != nil {
		mappedErr := s.mapError(err)
		if errors.Is(mappedErr, storagepkg.ErrObjectNotFound) {
			return nil, storagepkg.ErrObjectNotFound
		}
		return nil, mappedErr
	}

	return s.headOutputToObjectInfo(bucket, objectName, output), nil
}

// UpdateMetadata updates an object's metadata
func (s *S3Client) UpdateMetadata(ctx context.Context, bucket, objectName string, metadata map[string]string) (*storagepkg.ObjectInfo, error) {
	if !s.connected || s.client == nil {
		return nil, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return nil, err
	}
	if err := storagepkg.ValidateObjectName(objectName); err != nil {
		return nil, err
	}

	// S3 requires copying the object to itself to update metadata
	copySource := fmt.Sprintf("%s/%s", bucket, objectName)

	input := &s3.CopyObjectInput{
		Bucket:            aws.String(bucket),
		Key:               aws.String(objectName),
		CopySource:        aws.String(copySource),
		Metadata:          metadata,
		MetadataDirective: types.MetadataDirectiveReplace,
	}

	_, err := s.client.CopyObject(ctx, input)
	if err != nil {
		mappedErr := s.mapError(err)
		if errors.Is(mappedErr, storagepkg.ErrObjectNotFound) {
			return nil, storagepkg.ErrObjectNotFound
		}
		return nil, fmt.Errorf("%w: %v", storagepkg.ErrMetadataUpdateFailed, mappedErr)
	}

	return s.GetObjectInfo(ctx, bucket, objectName)
}

// Copy copies an object from source to destination
func (s *S3Client) Copy(ctx context.Context, srcBucket, srcObject, dstBucket, dstObject string, opts *storagepkg.CopyOptions) (*storagepkg.ObjectInfo, error) {
	if !s.connected || s.client == nil {
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

	copySource := fmt.Sprintf("%s/%s", srcBucket, srcObject)

	input := &s3.CopyObjectInput{
		Bucket:     aws.String(dstBucket),
		Key:        aws.String(dstObject),
		CopySource: aws.String(copySource),
	}

	// Apply copy options
	if opts != nil {
		if opts.ContentType != "" {
			input.ContentType = aws.String(opts.ContentType)
			input.MetadataDirective = types.MetadataDirectiveReplace
		}

		if opts.Metadata != nil {
			input.Metadata = opts.Metadata
			input.MetadataDirective = types.MetadataDirectiveReplace
		}

		if opts.StorageClass != "" {
			input.StorageClass = types.StorageClass(opts.StorageClass)
		}

		if opts.DestinationACL != "" {
			input.ACL = s.mapACL(opts.DestinationACL)
		}
	}

	_, err := s.client.CopyObject(ctx, input)
	if err != nil {
		mappedErr := s.mapError(err)
		if errors.Is(mappedErr, storagepkg.ErrObjectNotFound) {
			return nil, storagepkg.ErrObjectNotFound
		}
		return nil, fmt.Errorf("%w: %v", storagepkg.ErrCopyFailed, mappedErr)
	}

	return s.GetObjectInfo(ctx, dstBucket, dstObject)
}

// Exists checks if an object exists
func (s *S3Client) Exists(ctx context.Context, bucket, objectName string) (bool, error) {
	if !s.connected || s.client == nil {
		return false, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return false, err
	}
	if err := storagepkg.ValidateObjectName(objectName); err != nil {
		return false, err
	}

	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectName),
	}

	_, err := s.client.HeadObject(ctx, input)
	if err != nil {
		mappedErr := s.mapError(err)
		if errors.Is(mappedErr, storagepkg.ErrObjectNotFound) {
			return false, nil
		}
		return false, mappedErr
	}

	return true, nil
}

// List lists objects in a bucket
func (s *S3Client) List(ctx context.Context, bucket string, opts *storagepkg.ListOptions) (*storagepkg.ListResult, error) {
	if !s.connected || s.client == nil {
		return nil, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return nil, err
	}

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}

	if opts != nil {
		if opts.Prefix != "" {
			input.Prefix = aws.String(opts.Prefix)
		}
		if opts.Delimiter != "" {
			input.Delimiter = aws.String(opts.Delimiter)
		}
		if opts.MaxResults > 0 {
			maxKeys := opts.MaxResults
			if maxKeys > math.MaxInt32 {
				maxKeys = math.MaxInt32
			}
			input.MaxKeys = aws.Int32(int32(maxKeys)) // #nosec
		}
		if opts.PageToken != "" {
			input.ContinuationToken = aws.String(opts.PageToken)
		}
	}

	output, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", storagepkg.ErrListFailed, s.mapError(err))
	}

	result := &storagepkg.ListResult{
		Objects:  make([]*storagepkg.ObjectInfo, 0, len(output.Contents)),
		Prefixes: make([]string, 0, len(output.CommonPrefixes)),
	}

	for _, obj := range output.Contents {
		result.Objects = append(result.Objects, &storagepkg.ObjectInfo{
			Name:         aws.ToString(obj.Key),
			Bucket:       bucket,
			Size:         aws.ToInt64(obj.Size),
			ETag:         strings.Trim(aws.ToString(obj.ETag), "\""),
			Updated:      aws.ToTime(obj.LastModified),
			StorageClass: storagepkg.StorageClass(obj.StorageClass),
		})
	}

	for _, prefix := range output.CommonPrefixes {
		result.Prefixes = append(result.Prefixes, aws.ToString(prefix.Prefix))
	}

	if output.NextContinuationToken != nil {
		result.NextPageToken = aws.ToString(output.NextContinuationToken)
	}

	return result, nil
}

// ListAll lists all objects in a bucket (handles pagination automatically)
func (s *S3Client) ListAll(ctx context.Context, bucket string, opts *storagepkg.ListOptions) ([]*storagepkg.ObjectInfo, error) {
	if !s.connected || s.client == nil {
		return nil, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return nil, err
	}

	var allObjects []*storagepkg.ObjectInfo
	var pageToken string

	for {
		listOpts := &storagepkg.ListOptions{}
		if opts != nil {
			listOpts.Prefix = opts.Prefix
			listOpts.Delimiter = opts.Delimiter
		}
		listOpts.PageToken = pageToken

		result, err := s.List(ctx, bucket, listOpts)
		if err != nil {
			return nil, err
		}

		allObjects = append(allObjects, result.Objects...)

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}

	return allObjects, nil
}

// CreateBucket creates a new bucket
func (s *S3Client) CreateBucket(ctx context.Context, bucket string, opts *storagepkg.BucketCreateOptions) error {
	if !s.connected || s.client == nil {
		return storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return err
	}

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	}

	// Set location constraint (required for non-us-east-1 regions)
	if s.config.S3.Region != "" && s.config.S3.Region != "us-east-1" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(s.config.S3.Region),
		}
	}

	if opts != nil && opts.Location != "" && opts.Location != "us-east-1" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(opts.Location),
		}
	}

	_, err := s.client.CreateBucket(ctx, input)
	if err != nil {
		mappedErr := s.mapError(err)
		if errors.Is(mappedErr, storagepkg.ErrBucketAlreadyExists) {
			return storagepkg.ErrBucketAlreadyExists
		}
		return mappedErr
	}

	// Enable versioning if requested
	if opts != nil && opts.Versioning {
		versioningInput := &s3.PutBucketVersioningInput{
			Bucket: aws.String(bucket),
			VersioningConfiguration: &types.VersioningConfiguration{
				Status: types.BucketVersioningStatusEnabled,
			},
		}
		if _, err := s.client.PutBucketVersioning(ctx, versioningInput); err != nil {
			return fmt.Errorf("failed to enable versioning: %w", s.mapError(err))
		}
	}

	// Apply tags/labels if provided
	if opts != nil && len(opts.Labels) > 0 {
		var tags []types.Tag
		for k, v := range opts.Labels {
			tags = append(tags, types.Tag{
				Key:   aws.String(k),
				Value: aws.String(v),
			})
		}

		taggingInput := &s3.PutBucketTaggingInput{
			Bucket: aws.String(bucket),
			Tagging: &types.Tagging{
				TagSet: tags,
			},
		}
		if _, err := s.client.PutBucketTagging(ctx, taggingInput); err != nil {
			return fmt.Errorf("failed to set bucket tags: %w", s.mapError(err))
		}
	}

	return nil
}

// DeleteBucket deletes a bucket
func (s *S3Client) DeleteBucket(ctx context.Context, bucket string) error {
	if !s.connected || s.client == nil {
		return storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return err
	}

	input := &s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	}

	_, err := s.client.DeleteBucket(ctx, input)
	if err != nil {
		mappedErr := s.mapError(err)
		if errors.Is(mappedErr, storagepkg.ErrBucketNotFound) {
			return storagepkg.ErrBucketNotFound
		}
		return fmt.Errorf("%w: %v", storagepkg.ErrDeleteFailed, mappedErr)
	}

	return nil
}

// GetBucketInfo retrieves metadata about a bucket
func (s *S3Client) GetBucketInfo(ctx context.Context, bucket string) (*storagepkg.BucketInfo, error) {
	if !s.connected || s.client == nil {
		return nil, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return nil, err
	}

	// Check if bucket exists by calling HeadBucket
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		mappedErr := s.mapError(err)
		if errors.Is(mappedErr, storagepkg.ErrBucketNotFound) {
			return nil, storagepkg.ErrBucketNotFound
		}
		return nil, mappedErr
	}

	info := &storagepkg.BucketInfo{
		Name:   bucket,
		Labels: make(map[string]string),
	}

	// Get bucket location
	locOutput, err := s.client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucket),
	})
	if err == nil && locOutput.LocationConstraint != "" {
		info.Location = string(locOutput.LocationConstraint)
	} else {
		info.Location = "us-east-1" // Default region
	}

	// Get versioning status
	versioningOutput, err := s.client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(bucket),
	})
	if err == nil {
		info.Versioning = versioningOutput.Status == types.BucketVersioningStatusEnabled
	}

	// Get bucket tags
	tagsOutput, err := s.client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucket),
	})
	if err == nil {
		for _, tag := range tagsOutput.TagSet {
			info.Labels[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}

	return info, nil
}

// BucketExists checks if a bucket exists
func (s *S3Client) BucketExists(ctx context.Context, bucket string) (bool, error) {
	if !s.connected || s.client == nil {
		return false, storagepkg.ErrNotConnected
	}

	if err := storagepkg.ValidateBucketName(bucket); err != nil {
		return false, err
	}

	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		mappedErr := s.mapError(err)
		if errors.Is(mappedErr, storagepkg.ErrBucketNotFound) {
			return false, nil
		}
		return false, mappedErr
	}

	return true, nil
}

// ListBuckets lists all buckets
func (s *S3Client) ListBuckets(ctx context.Context) ([]*storagepkg.BucketInfo, error) {
	if !s.connected || s.client == nil {
		return nil, storagepkg.ErrNotConnected
	}

	output, err := s.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", storagepkg.ErrListFailed, s.mapError(err))
	}

	buckets := make([]*storagepkg.BucketInfo, 0, len(output.Buckets))
	for _, b := range output.Buckets {
		buckets = append(buckets, &storagepkg.BucketInfo{
			Name:    aws.ToString(b.Name),
			Created: aws.ToTime(b.CreationDate),
		})
	}

	return buckets, nil
}

// SignedURL generates a signed URL for an object
func (s *S3Client) SignedURL(ctx context.Context, bucket, objectName string, opts *storagepkg.SignedURLOptions) (string, error) {
	if !s.connected || s.client == nil {
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

	var presignedURL string

	switch opts.Method {
	case storagepkg.SignedURLMethodGET:
		input := &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(objectName),
		}
		presignOpts := []func(*s3.PresignOptions){
			s3.WithPresignExpires(opts.Expires),
		}
		result, err := s.presignClient.PresignGetObject(ctx, input, presignOpts...)
		if err != nil {
			return "", fmt.Errorf("%w: %v", storagepkg.ErrSignedURLFailed, err)
		}
		presignedURL = result.URL

	case storagepkg.SignedURLMethodPUT:
		input := &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(objectName),
		}
		if opts.ContentType != "" {
			input.ContentType = aws.String(opts.ContentType)
		}
		presignOpts := []func(*s3.PresignOptions){
			s3.WithPresignExpires(opts.Expires),
		}
		result, err := s.presignClient.PresignPutObject(ctx, input, presignOpts...)
		if err != nil {
			return "", fmt.Errorf("%w: %v", storagepkg.ErrSignedURLFailed, err)
		}
		presignedURL = result.URL

	case storagepkg.SignedURLMethodDELETE:
		input := &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(objectName),
		}
		presignOpts := []func(*s3.PresignOptions){
			s3.WithPresignExpires(opts.Expires),
		}
		result, err := s.presignClient.PresignDeleteObject(ctx, input, presignOpts...)
		if err != nil {
			return "", fmt.Errorf("%w: %v", storagepkg.ErrSignedURLFailed, err)
		}
		presignedURL = result.URL

	default:
		return "", fmt.Errorf("%w: unsupported method %s", storagepkg.ErrSignedURLFailed, opts.Method)
	}

	return presignedURL, nil
}

// GetBackend returns the storage backend type
func (s *S3Client) GetBackend() storagepkg.Backend {
	return storagepkg.BackendS3
}

// GetConfig returns the storage configuration
func (s *S3Client) GetConfig() *storagepkg.Config {
	return s.config
}

// outputToObjectInfo converts S3 GetObjectOutput to ObjectInfo
func (s *S3Client) outputToObjectInfo(bucket, key string, output *s3.GetObjectOutput) *storagepkg.ObjectInfo {
	return &storagepkg.ObjectInfo{
		Name:               key,
		Bucket:             bucket,
		Size:               aws.ToInt64(output.ContentLength),
		ContentType:        aws.ToString(output.ContentType),
		Updated:            aws.ToTime(output.LastModified),
		ETag:               strings.Trim(aws.ToString(output.ETag), "\""),
		StorageClass:       storagepkg.StorageClass(output.StorageClass),
		Metadata:           output.Metadata,
		CacheControl:       aws.ToString(output.CacheControl),
		ContentEncoding:    aws.ToString(output.ContentEncoding),
		ContentDisposition: aws.ToString(output.ContentDisposition),
		ContentLanguage:    aws.ToString(output.ContentLanguage),
	}
}

// headOutputToObjectInfo converts S3 HeadObjectOutput to ObjectInfo
func (s *S3Client) headOutputToObjectInfo(bucket, key string, output *s3.HeadObjectOutput) *storagepkg.ObjectInfo {
	return &storagepkg.ObjectInfo{
		Name:               key,
		Bucket:             bucket,
		Size:               aws.ToInt64(output.ContentLength),
		ContentType:        aws.ToString(output.ContentType),
		Updated:            aws.ToTime(output.LastModified),
		ETag:               strings.Trim(aws.ToString(output.ETag), "\""),
		StorageClass:       storagepkg.StorageClass(output.StorageClass),
		Metadata:           output.Metadata,
		CacheControl:       aws.ToString(output.CacheControl),
		ContentEncoding:    aws.ToString(output.ContentEncoding),
		ContentDisposition: aws.ToString(output.ContentDisposition),
		ContentLanguage:    aws.ToString(output.ContentLanguage),
	}
}

// mapACL maps storage ACL to S3 ACL
func (s *S3Client) mapACL(acl storagepkg.ACL) types.ObjectCannedACL {
	switch acl {
	case storagepkg.ACLPrivate:
		return types.ObjectCannedACLPrivate
	case storagepkg.ACLPublicRead:
		return types.ObjectCannedACLPublicRead
	case storagepkg.ACLPublicReadWrite:
		return types.ObjectCannedACLPublicReadWrite
	case storagepkg.ACLAuthenticatedRead:
		return types.ObjectCannedACLAuthenticatedRead
	case storagepkg.ACLBucketOwnerRead:
		return types.ObjectCannedACLBucketOwnerRead
	case storagepkg.ACLBucketOwnerFull:
		return types.ObjectCannedACLBucketOwnerFullControl
	default:
		return types.ObjectCannedACLPrivate
	}
}

// mapError maps S3 errors to storage errors
func (s *S3Client) mapError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoSuchKey", "NotFound":
			return storagepkg.ErrObjectNotFound
		case "NoSuchBucket":
			return storagepkg.ErrBucketNotFound
		case "BucketAlreadyExists", "BucketAlreadyOwnedByYou":
			return storagepkg.ErrBucketAlreadyExists
		case "AccessDenied", "Forbidden":
			return storagepkg.ErrAccessDenied
		}
	}

	// Check for specific error messages
	errStr := err.Error()
	if strings.Contains(errStr, "NoSuchKey") || strings.Contains(errStr, "404") {
		return storagepkg.ErrObjectNotFound
	}
	if strings.Contains(errStr, "NoSuchBucket") {
		return storagepkg.ErrBucketNotFound
	}
	if strings.Contains(errStr, "BucketAlreadyExists") || strings.Contains(errStr, "BucketAlreadyOwnedByYou") {
		return storagepkg.ErrBucketAlreadyExists
	}
	if strings.Contains(errStr, "AccessDenied") || strings.Contains(errStr, "Forbidden") {
		return storagepkg.ErrAccessDenied
	}

	return err
}
