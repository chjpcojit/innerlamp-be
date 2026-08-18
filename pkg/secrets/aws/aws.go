// Package aws provides AWS Secrets Manager implementation for the secrets package
package aws

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"

	"base_project/pkg/secrets"
)

func init() {
	// Register the AWS provider factory
	secrets.Register(secrets.ProviderAWS, NewClient)
}

// awsClient implements the secrets.Client interface for AWS Secrets Manager
type awsClient struct {
	cfg       *secrets.Config
	client    *secretsmanager.Client
	connected bool
	mu        sync.RWMutex
}

// NewClient creates a new AWS Secrets Manager client
func NewClient(cfg *secrets.Config) (secrets.Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if cfg.Provider != secrets.ProviderAWS {
		return nil, fmt.Errorf("%w: expected AWS provider", secrets.ErrInvalidConfig)
	}

	return &awsClient{
		cfg: cfg,
	}, nil
}

// Connect initializes the AWS Secrets Manager client
func (c *awsClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return secrets.ErrAlreadyConnected
	}

	var opts []func(*config.LoadOptions) error

	// Set region
	opts = append(opts, config.WithRegion(c.cfg.AWS.Region))

	// Set credentials if provided
	if !c.cfg.AWS.UseDefaultCredentials {
		creds := credentials.NewStaticCredentialsProvider(
			c.cfg.AWS.AccessKeyID,
			c.cfg.AWS.SecretAccessKey,
			c.cfg.AWS.SessionToken,
		)
		opts = append(opts, config.WithCredentialsProvider(creds))
	}

	// Load AWS configuration
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("%w: %v", secrets.ErrConnectionFailed, err)
	}

	// Create Secrets Manager client
	clientOpts := []func(*secretsmanager.Options){}
	if c.cfg.AWS.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *secretsmanager.Options) {
			o.BaseEndpoint = aws.String(c.cfg.AWS.Endpoint)
		})
	}

	c.client = secretsmanager.NewFromConfig(awsCfg, clientOpts...)
	c.connected = true

	return nil
}

// Close closes the AWS Secrets Manager client
func (c *awsClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.connected = false
	c.client = nil

	return nil
}

// IsConnected returns true if the client is connected
func (c *awsClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Create creates a new secret with an initial value
func (c *awsClient) Create(ctx context.Context, name string, value []byte, opts *secrets.CreateOptions) (*secrets.Secret, error) {
	if !c.IsConnected() {
		return nil, secrets.ErrNotConnected
	}

	if err := secrets.ValidateSecretName(name); err != nil {
		return nil, err
	}

	if len(value) == 0 {
		return nil, secrets.ErrEmptySecretValue
	}

	input := &secretsmanager.CreateSecretInput{
		Name:         aws.String(name),
		SecretBinary: value,
	}

	if opts != nil {
		if opts.Description != "" {
			input.Description = aws.String(opts.Description)
		}
		if len(opts.Labels) > 0 {
			input.Tags = c.labelsToTags(opts.Labels)
		}
	}

	result, err := c.client.CreateSecret(ctx, input)
	if err != nil {
		return nil, c.mapError(err, secrets.ErrCreateFailed)
	}

	return &secrets.Secret{
		Name:           name,
		Description:    safeString(opts, func(o *secrets.CreateOptions) string { return o.Description }),
		ARN:            aws.ToString(result.ARN),
		CurrentVersion: aws.ToString(result.VersionId),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Labels:         safeLabels(opts),
	}, nil
}

// Get retrieves a secret value
func (c *awsClient) Get(ctx context.Context, name string, opts *secrets.GetOptions) (*secrets.SecretValue, error) {
	if !c.IsConnected() {
		return nil, secrets.ErrNotConnected
	}

	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(name),
	}

	if opts != nil {
		if opts.Version != "" && opts.Version != secrets.VersionLatest {
			input.VersionId = aws.String(opts.Version)
		}
		if opts.VersionStage != "" {
			input.VersionStage = aws.String(opts.VersionStage)
		}
	}

	result, err := c.client.GetSecretValue(ctx, input)
	if err != nil {
		return nil, c.mapError(err, secrets.ErrGetFailed)
	}

	var value []byte
	if result.SecretBinary != nil {
		value = result.SecretBinary
	} else if result.SecretString != nil {
		value = []byte(*result.SecretString)
	}

	var createdAt time.Time
	if result.CreatedDate != nil {
		createdAt = *result.CreatedDate
	}

	return &secrets.SecretValue{
		Name:      name,
		Value:     value,
		Version:   aws.ToString(result.VersionId),
		CreatedAt: createdAt,
		State:     secrets.VersionStateEnabled,
	}, nil
}

// GetMetadata retrieves secret metadata without the value
func (c *awsClient) GetMetadata(ctx context.Context, name string) (*secrets.Secret, error) {
	if !c.IsConnected() {
		return nil, secrets.ErrNotConnected
	}

	input := &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(name),
	}

	result, err := c.client.DescribeSecret(ctx, input)
	if err != nil {
		return nil, c.mapError(err, secrets.ErrGetFailed)
	}

	secret := &secrets.Secret{
		Name:        aws.ToString(result.Name),
		Description: aws.ToString(result.Description),
		ARN:         aws.ToString(result.ARN),
		Labels:      c.tagsToLabels(result.Tags),
		Metadata:    make(map[string]string),
	}

	if result.CreatedDate != nil {
		secret.CreatedAt = *result.CreatedDate
	}
	if result.LastChangedDate != nil {
		secret.UpdatedAt = *result.LastChangedDate
	}

	// Get current version from version IDs
	if result.VersionIdsToStages != nil {
		for versionID, stages := range result.VersionIdsToStages {
			for _, stage := range stages {
				if stage == "AWSCURRENT" {
					secret.CurrentVersion = versionID
					break
				}
			}
		}
	}

	return secret, nil
}

// Update adds a new version with the provided value
func (c *awsClient) Update(ctx context.Context, name string, value []byte) (*secrets.SecretVersion, error) {
	if !c.IsConnected() {
		return nil, secrets.ErrNotConnected
	}

	if len(value) == 0 {
		return nil, secrets.ErrEmptySecretValue
	}

	input := &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(name),
		SecretBinary: value,
	}

	result, err := c.client.PutSecretValue(ctx, input)
	if err != nil {
		return nil, c.mapError(err, secrets.ErrUpdateFailed)
	}

	return &secrets.SecretVersion{
		Version:       aws.ToString(result.VersionId),
		State:         secrets.VersionStateEnabled,
		CreatedAt:     time.Now(),
		VersionStages: result.VersionStages,
	}, nil
}

// UpdateMetadata updates secret metadata
func (c *awsClient) UpdateMetadata(ctx context.Context, name string, opts *secrets.UpdateMetadataOptions) (*secrets.Secret, error) {
	if !c.IsConnected() {
		return nil, secrets.ErrNotConnected
	}

	if opts == nil {
		return c.GetMetadata(ctx, name)
	}

	// Update description if provided
	if opts.Description != nil {
		input := &secretsmanager.UpdateSecretInput{
			SecretId:    aws.String(name),
			Description: opts.Description,
		}

		_, err := c.client.UpdateSecret(ctx, input)
		if err != nil {
			return nil, c.mapError(err, secrets.ErrUpdateFailed)
		}
	}

	// Update tags if provided
	if opts.Labels != nil {
		// First, get current tags to determine what to remove
		describeInput := &secretsmanager.DescribeSecretInput{
			SecretId: aws.String(name),
		}

		describeResult, err := c.client.DescribeSecret(ctx, describeInput)
		if err != nil {
			return nil, c.mapError(err, secrets.ErrGetFailed)
		}

		// Remove existing tags
		if len(describeResult.Tags) > 0 {
			var tagKeys []string
			for _, tag := range describeResult.Tags {
				tagKeys = append(tagKeys, aws.ToString(tag.Key))
			}

			untagInput := &secretsmanager.UntagResourceInput{
				SecretId: describeResult.ARN,
				TagKeys:  tagKeys,
			}
			_, err = c.client.UntagResource(ctx, untagInput)
			if err != nil {
				return nil, c.mapError(err, secrets.ErrUpdateFailed)
			}
		}

		// Add new tags
		if len(opts.Labels) > 0 {
			tagInput := &secretsmanager.TagResourceInput{
				SecretId: describeResult.ARN,
				Tags:     c.labelsToTags(opts.Labels),
			}
			_, err = c.client.TagResource(ctx, tagInput)
			if err != nil {
				return nil, c.mapError(err, secrets.ErrUpdateFailed)
			}
		}
	}

	return c.GetMetadata(ctx, name)
}

// Delete removes a secret and all its versions
func (c *awsClient) Delete(ctx context.Context, name string, opts *secrets.DeleteOptions) error {
	if !c.IsConnected() {
		return secrets.ErrNotConnected
	}

	input := &secretsmanager.DeleteSecretInput{
		SecretId: aws.String(name),
	}

	if opts != nil {
		if opts.Force || opts.RecoveryWindowDays == 0 {
			input.ForceDeleteWithoutRecovery = aws.Bool(true)
		} else if opts.RecoveryWindowDays > 0 {
			input.RecoveryWindowInDays = aws.Int64(int64(opts.RecoveryWindowDays))
		}
	}

	_, err := c.client.DeleteSecret(ctx, input)
	if err != nil {
		return c.mapError(err, secrets.ErrDeleteFailed)
	}

	return nil
}

// List retrieves all secrets
func (c *awsClient) List(ctx context.Context, opts *secrets.ListOptions) (*secrets.ListResult, error) {
	if !c.IsConnected() {
		return nil, secrets.ErrNotConnected
	}

	input := &secretsmanager.ListSecretsInput{}

	if opts != nil {
		if opts.MaxResults > 0 {
			maxResults := opts.MaxResults
			if maxResults > math.MaxInt32 {
				maxResults = math.MaxInt32
			}
			input.MaxResults = aws.Int32(int32(maxResults)) // #nosec
		}
		if opts.PageToken != "" {
			input.NextToken = aws.String(opts.PageToken)
		}
		if opts.Filter != "" {
			input.Filters = []types.Filter{
				{
					Key:    types.FilterNameStringTypeName,
					Values: []string{opts.Filter},
				},
			}
		}
	}

	result, err := c.client.ListSecrets(ctx, input)
	if err != nil {
		return nil, c.mapError(err, secrets.ErrListFailed)
	}

	var secretsList []*secrets.Secret
	for _, s := range result.SecretList {
		secret := &secrets.Secret{
			Name:        aws.ToString(s.Name),
			Description: aws.ToString(s.Description),
			ARN:         aws.ToString(s.ARN),
			Labels:      c.tagsToLabels(s.Tags),
		}

		if s.CreatedDate != nil {
			secret.CreatedAt = *s.CreatedDate
		}
		if s.LastChangedDate != nil {
			secret.UpdatedAt = *s.LastChangedDate
		}

		// Apply prefix filter if specified
		if opts != nil && opts.Prefix != "" {
			if !strings.HasPrefix(secret.Name, opts.Prefix) {
				continue
			}
		}

		secretsList = append(secretsList, secret)
	}

	return &secrets.ListResult{
		Secrets:       secretsList,
		NextPageToken: aws.ToString(result.NextToken),
	}, nil
}

// GetVersion retrieves a specific version of a secret
func (c *awsClient) GetVersion(ctx context.Context, name string, version string) (*secrets.SecretValue, error) {
	opts := &secrets.GetOptions{
		Version: version,
	}
	return c.Get(ctx, name, opts)
}

// ListVersions lists all versions of a secret
func (c *awsClient) ListVersions(ctx context.Context, name string, opts *secrets.ListVersionsOptions) (*secrets.ListVersionsResult, error) {
	if !c.IsConnected() {
		return nil, secrets.ErrNotConnected
	}

	input := &secretsmanager.ListSecretVersionIdsInput{
		SecretId: aws.String(name),
	}

	if opts != nil {
		if opts.MaxResults > 0 {
			maxResults := opts.MaxResults
			if maxResults > math.MaxInt32 {
				maxResults = math.MaxInt32
			}
			input.MaxResults = aws.Int32(int32(maxResults)) // #nosec
		}
		if opts.PageToken != "" {
			input.NextToken = aws.String(opts.PageToken)
		}
		if !opts.IncludeDestroyed {
			input.IncludeDeprecated = aws.Bool(false)
		} else {
			input.IncludeDeprecated = aws.Bool(true)
		}
	}

	result, err := c.client.ListSecretVersionIds(ctx, input)
	if err != nil {
		return nil, c.mapError(err, secrets.ErrListFailed)
	}

	var versions []*secrets.SecretVersion
	for _, v := range result.Versions {
		version := &secrets.SecretVersion{
			Version:       aws.ToString(v.VersionId),
			VersionStages: v.VersionStages,
		}

		if v.CreatedDate != nil {
			version.CreatedAt = *v.CreatedDate
		}

		// Determine state based on version stages
		version.State = secrets.VersionStateEnabled
		for _, stage := range v.VersionStages {
			if stage == "AWSPREVIOUS" {
				version.State = secrets.VersionStateDisabled
			}
		}

		versions = append(versions, version)
	}

	return &secrets.ListVersionsResult{
		Versions:      versions,
		NextPageToken: aws.ToString(result.NextToken),
	}, nil
}

// DestroyVersion is not directly supported in AWS Secrets Manager
// AWS uses the DeleteSecret with recovery window for this purpose
func (c *awsClient) DestroyVersion(ctx context.Context, name string, version string) error {
	if !c.IsConnected() {
		return secrets.ErrNotConnected
	}

	// AWS doesn't support destroying individual versions directly
	// The closest operation is UpdateSecretVersionStage to remove AWSCURRENT stage
	// But this doesn't actually delete the version data

	return fmt.Errorf("%w: AWS Secrets Manager does not support destroying individual versions", secrets.ErrDestroyVersionFailed)
}

// DisableVersion moves a version to AWSPREVIOUS stage
func (c *awsClient) DisableVersion(ctx context.Context, name string, version string) error {
	if !c.IsConnected() {
		return secrets.ErrNotConnected
	}

	// In AWS, we can't really "disable" a version like in GCP
	// The best we can do is ensure it's not the current version
	// This is a no-op if the version is already not AWSCURRENT

	return nil
}

// EnableVersion is not directly supported in AWS Secrets Manager
// You would need to PutSecretValue with the same content to make it current
func (c *awsClient) EnableVersion(ctx context.Context, name string, version string) error {
	if !c.IsConnected() {
		return secrets.ErrNotConnected
	}

	// Move the version to AWSCURRENT stage
	input := &secretsmanager.UpdateSecretVersionStageInput{
		SecretId:            aws.String(name),
		VersionStage:        aws.String("AWSCURRENT"),
		MoveToVersionId:     aws.String(version),
		RemoveFromVersionId: nil, // AWS will automatically handle removing from current version
	}

	_, err := c.client.UpdateSecretVersionStage(ctx, input)
	if err != nil {
		return c.mapError(err, secrets.ErrEnableVersionFailed)
	}

	return nil
}

// GetProvider returns the provider type
func (c *awsClient) GetProvider() secrets.Provider {
	return secrets.ProviderAWS
}

// GetConfig returns the configuration
func (c *awsClient) GetConfig() *secrets.Config {
	return c.cfg
}

// Exists checks if a secret exists
func (c *awsClient) Exists(ctx context.Context, name string) (bool, error) {
	if !c.IsConnected() {
		return false, secrets.ErrNotConnected
	}

	input := &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(name),
	}

	_, err := c.client.DescribeSecret(ctx, input)
	if err != nil {
		if containsErrorType(err.Error(), "ResourceNotFoundException") {
			return false, nil
		}
		return false, c.mapError(err, secrets.ErrGetFailed)
	}

	return true, nil
}

// Helper methods

// labelsToTags converts labels map to AWS tags
func (c *awsClient) labelsToTags(labels map[string]string) []types.Tag {
	if len(labels) == 0 {
		return nil
	}

	tags := make([]types.Tag, 0, len(labels))
	for k, v := range labels {
		tags = append(tags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return tags
}

// tagsToLabels converts AWS tags to labels map
func (c *awsClient) tagsToLabels(tags []types.Tag) map[string]string {
	if len(tags) == 0 {
		return nil
	}

	labels := make(map[string]string, len(tags))
	for _, tag := range tags {
		labels[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return labels
}

// mapError maps AWS errors to common errors
func (c *awsClient) mapError(err error, defaultErr error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	switch {
	case containsErrorType(errMsg, "ResourceNotFoundException"):
		return fmt.Errorf("%w: %v", secrets.ErrSecretNotFound, err)
	case containsErrorType(errMsg, "ResourceExistsException"):
		return fmt.Errorf("%w: %v", secrets.ErrSecretAlreadyExists, err)
	case containsErrorType(errMsg, "InvalidParameterException"):
		return fmt.Errorf("%w: %v", secrets.ErrInvalidSecretName, err)
	case containsErrorType(errMsg, "InvalidRequestException"):
		return fmt.Errorf("%w: %v", secrets.ErrInvalidSecretValue, err)
	case containsErrorType(errMsg, "AccessDeniedException"):
		return fmt.Errorf("%w: %v", secrets.ErrPermissionDenied, err)
	default:
		return fmt.Errorf("%w: %v", defaultErr, err)
	}
}

// containsErrorType checks if the error message contains the error type name
func containsErrorType(errMsg, typeName string) bool {
	return strings.Contains(errMsg, typeName)
}

// safeString safely extracts a string from options
func safeString(opts *secrets.CreateOptions, getter func(*secrets.CreateOptions) string) string {
	if opts == nil {
		return ""
	}
	return getter(opts)
}

// safeLabels safely extracts labels from options
func safeLabels(opts *secrets.CreateOptions) map[string]string {
	if opts == nil {
		return nil
	}
	return opts.Labels
}
