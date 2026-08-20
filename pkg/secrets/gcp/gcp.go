// Package gcp provides GCP Secret Manager implementation for the secrets package
package gcp

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"chjpcoj.com.innerlamp/pkg/secrets"
)

func init() {
	// Register the GCP provider factory
	secrets.Register(secrets.ProviderGCP, NewClient)
}

// gcpClient implements the secrets.Client interface for GCP Secret Manager
type gcpClient struct {
	config    *secrets.Config
	client    *secretmanager.Client
	connected bool
	mu        sync.RWMutex
}

// NewClient creates a new GCP Secret Manager client
func NewClient(config *secrets.Config) (secrets.Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	if config.Provider != secrets.ProviderGCP {
		return nil, fmt.Errorf("%w: expected GCP provider", secrets.ErrInvalidConfig)
	}

	return &gcpClient{
		config: config,
	}, nil
}

// Connect initializes the GCP Secret Manager client
func (c *gcpClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return secrets.ErrAlreadyConnected
	}

	var opts []option.ClientOption

	// Determine credential option
	if c.config.GCP.UseDefaultCredentials {
		// Use Application Default Credentials - no explicit option needed
	} else if len(c.config.GCP.CredentialsJSON) > 0 {
		opts = append(opts, option.WithCredentialsJSON(c.config.GCP.CredentialsJSON))
	} else if c.config.GCP.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(c.config.GCP.CredentialsFile))
	}

	// Create the Secret Manager client
	client, err := secretmanager.NewClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("%w: %v", secrets.ErrConnectionFailed, err)
	}

	c.client = client
	c.connected = true

	return nil
}

// Close closes the GCP Secret Manager client
func (c *gcpClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		if err := c.client.Close(); err != nil {
			return err
		}
	}

	c.connected = false
	c.client = nil

	return nil
}

// IsConnected returns true if the client is connected
func (c *gcpClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Create creates a new secret with an initial value
func (c *gcpClient) Create(ctx context.Context, name string, value []byte, opts *secrets.CreateOptions) (*secrets.Secret, error) {
	if !c.IsConnected() {
		return nil, secrets.ErrNotConnected
	}

	if err := secrets.ValidateSecretName(name); err != nil {
		return nil, err
	}

	if len(value) == 0 {
		return nil, secrets.ErrEmptySecretValue
	}

	// Build the secret request
	createReq := &secretmanagerpb.CreateSecretRequest{
		Parent:   c.projectPath(),
		SecretId: name,
		Secret: &secretmanagerpb.Secret{
			Replication: c.buildReplication(opts),
		},
	}

	// Add labels if provided
	if opts != nil && len(opts.Labels) > 0 {
		createReq.Secret.Labels = opts.Labels
	}

	// Create the secret (metadata only)
	secret, err := c.client.CreateSecret(ctx, createReq)
	if err != nil {
		return nil, c.mapError(err, secrets.ErrCreateFailed)
	}

	// Add the initial version with the secret value
	addReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: secret.Name,
		Payload: &secretmanagerpb.SecretPayload{
			Data: value,
		},
	}

	version, err := c.client.AddSecretVersion(ctx, addReq)
	if err != nil {
		// Try to clean up the secret we just created
		_ = c.Delete(ctx, name, nil)
		return nil, c.mapError(err, secrets.ErrCreateFailed)
	}

	return c.secretToModel(secret, version), nil
}

// Get retrieves a secret value
func (c *gcpClient) Get(ctx context.Context, name string, opts *secrets.GetOptions) (*secrets.SecretValue, error) {
	if !c.IsConnected() {
		return nil, secrets.ErrNotConnected
	}

	version := secrets.VersionLatest
	if opts != nil && opts.Version != "" {
		version = opts.Version
	}

	return c.GetVersion(ctx, name, version)
}

// GetMetadata retrieves secret metadata without the value
func (c *gcpClient) GetMetadata(ctx context.Context, name string) (*secrets.Secret, error) {
	if !c.IsConnected() {
		return nil, secrets.ErrNotConnected
	}

	req := &secretmanagerpb.GetSecretRequest{
		Name: c.secretPath(name),
	}

	secret, err := c.client.GetSecret(ctx, req)
	if err != nil {
		return nil, c.mapError(err, secrets.ErrGetFailed)
	}

	return c.secretToModel(secret, nil), nil
}

// Update adds a new version with the provided value
func (c *gcpClient) Update(ctx context.Context, name string, value []byte) (*secrets.SecretVersion, error) {
	if !c.IsConnected() {
		return nil, secrets.ErrNotConnected
	}

	if len(value) == 0 {
		return nil, secrets.ErrEmptySecretValue
	}

	req := &secretmanagerpb.AddSecretVersionRequest{
		Parent: c.secretPath(name),
		Payload: &secretmanagerpb.SecretPayload{
			Data: value,
		},
	}

	version, err := c.client.AddSecretVersion(ctx, req)
	if err != nil {
		return nil, c.mapError(err, secrets.ErrUpdateFailed)
	}

	return c.versionToModel(version), nil
}

// UpdateMetadata updates secret metadata
func (c *gcpClient) UpdateMetadata(ctx context.Context, name string, opts *secrets.UpdateMetadataOptions) (*secrets.Secret, error) {
	if !c.IsConnected() {
		return nil, secrets.ErrNotConnected
	}

	if opts == nil {
		return c.GetMetadata(ctx, name)
	}

	// First, get the current secret
	getReq := &secretmanagerpb.GetSecretRequest{
		Name: c.secretPath(name),
	}

	secret, err := c.client.GetSecret(ctx, getReq)
	if err != nil {
		return nil, c.mapError(err, secrets.ErrGetFailed)
	}

	// Build update mask and update the secret
	var paths []string

	if opts.Labels != nil {
		secret.Labels = opts.Labels
		paths = append(paths, "labels")
	}

	if len(paths) == 0 {
		return c.secretToModel(secret, nil), nil
	}

	updateReq := &secretmanagerpb.UpdateSecretRequest{
		Secret: secret,
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: paths,
		},
	}

	updatedSecret, err := c.client.UpdateSecret(ctx, updateReq)
	if err != nil {
		return nil, c.mapError(err, secrets.ErrUpdateFailed)
	}

	return c.secretToModel(updatedSecret, nil), nil
}

// Delete removes a secret and all its versions
func (c *gcpClient) Delete(ctx context.Context, name string, opts *secrets.DeleteOptions) error {
	if !c.IsConnected() {
		return secrets.ErrNotConnected
	}

	req := &secretmanagerpb.DeleteSecretRequest{
		Name: c.secretPath(name),
	}

	if err := c.client.DeleteSecret(ctx, req); err != nil {
		return c.mapError(err, secrets.ErrDeleteFailed)
	}

	return nil
}

// List retrieves all secrets
func (c *gcpClient) List(ctx context.Context, opts *secrets.ListOptions) (*secrets.ListResult, error) {
	if !c.IsConnected() {
		return nil, secrets.ErrNotConnected
	}

	req := &secretmanagerpb.ListSecretsRequest{
		Parent: c.projectPath(),
	}

	if opts != nil {
		if opts.MaxResults > 0 {
			maxResults := opts.MaxResults
			if maxResults > math.MaxInt32 {
				maxResults = math.MaxInt32
			}
			req.PageSize = int32(maxResults) // #nosec
		}
		if opts.PageToken != "" {
			req.PageToken = opts.PageToken
		}
		if opts.Filter != "" {
			req.Filter = opts.Filter
		}
	}

	var secretsList []*secrets.Secret
	it := c.client.ListSecrets(ctx, req)

	for {
		secret, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, c.mapError(err, secrets.ErrListFailed)
		}

		// Apply prefix filter if specified
		if opts != nil && opts.Prefix != "" {
			secretName := extractSecretName(secret.Name)
			if !strings.HasPrefix(secretName, opts.Prefix) {
				continue
			}
		}

		secretsList = append(secretsList, c.secretToModel(secret, nil))
	}

	return &secrets.ListResult{
		Secrets:       secretsList,
		NextPageToken: it.PageInfo().Token,
	}, nil
}

// GetVersion retrieves a specific version of a secret
func (c *gcpClient) GetVersion(ctx context.Context, name string, version string) (*secrets.SecretValue, error) {
	if !c.IsConnected() {
		return nil, secrets.ErrNotConnected
	}

	version = secrets.NormalizeVersion(version)

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: c.versionPath(name, version),
	}

	result, err := c.client.AccessSecretVersion(ctx, req)
	if err != nil {
		return nil, c.mapError(err, secrets.ErrGetFailed)
	}

	// Get version metadata
	versionReq := &secretmanagerpb.GetSecretVersionRequest{
		Name: c.versionPath(name, version),
	}

	versionMeta, err := c.client.GetSecretVersion(ctx, versionReq)
	if err != nil {
		// We got the value, so return it even if we can't get metadata
		return &secrets.SecretValue{
			Name:    name,
			Value:   result.Payload.Data,
			Version: version,
			State:   secrets.VersionStateEnabled,
		}, nil
	}

	return &secrets.SecretValue{
		Name:      name,
		Value:     result.Payload.Data,
		Version:   extractVersionNumber(versionMeta.Name),
		CreatedAt: versionMeta.CreateTime.AsTime(),
		State:     c.mapVersionState(versionMeta.State),
	}, nil
}

// ListVersions lists all versions of a secret
func (c *gcpClient) ListVersions(ctx context.Context, name string, opts *secrets.ListVersionsOptions) (*secrets.ListVersionsResult, error) {
	if !c.IsConnected() {
		return nil, secrets.ErrNotConnected
	}

	req := &secretmanagerpb.ListSecretVersionsRequest{
		Parent: c.secretPath(name),
	}

	if opts != nil {
		if opts.MaxResults > 0 {
			maxResults := opts.MaxResults
			if maxResults > math.MaxInt32 {
				maxResults = math.MaxInt32
			}
			req.PageSize = int32(maxResults) // #nosec
		}
		if opts.PageToken != "" {
			req.PageToken = opts.PageToken
		}

		// Build filter for version states
		var filters []string
		if !opts.IncludeDisabled && !opts.IncludeDestroyed {
			filters = append(filters, "state:ENABLED")
		} else if opts.IncludeDisabled && !opts.IncludeDestroyed {
			filters = append(filters, "state:ENABLED OR state:DISABLED")
		}
		// If both are included, no filter needed

		if len(filters) > 0 {
			req.Filter = strings.Join(filters, " AND ")
		}
	}

	var versions []*secrets.SecretVersion
	it := c.client.ListSecretVersions(ctx, req)

	for {
		version, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, c.mapError(err, secrets.ErrListFailed)
		}

		versions = append(versions, c.versionToModel(version))
	}

	return &secrets.ListVersionsResult{
		Versions:      versions,
		NextPageToken: it.PageInfo().Token,
	}, nil
}

// DestroyVersion permanently destroys a secret version
func (c *gcpClient) DestroyVersion(ctx context.Context, name string, version string) error {
	if !c.IsConnected() {
		return secrets.ErrNotConnected
	}

	req := &secretmanagerpb.DestroySecretVersionRequest{
		Name: c.versionPath(name, version),
	}

	_, err := c.client.DestroySecretVersion(ctx, req)
	if err != nil {
		return c.mapError(err, secrets.ErrDestroyVersionFailed)
	}

	return nil
}

// DisableVersion disables a secret version
func (c *gcpClient) DisableVersion(ctx context.Context, name string, version string) error {
	if !c.IsConnected() {
		return secrets.ErrNotConnected
	}

	req := &secretmanagerpb.DisableSecretVersionRequest{
		Name: c.versionPath(name, version),
	}

	_, err := c.client.DisableSecretVersion(ctx, req)
	if err != nil {
		return c.mapError(err, secrets.ErrDisableVersionFailed)
	}

	return nil
}

// EnableVersion enables a previously disabled secret version
func (c *gcpClient) EnableVersion(ctx context.Context, name string, version string) error {
	if !c.IsConnected() {
		return secrets.ErrNotConnected
	}

	req := &secretmanagerpb.EnableSecretVersionRequest{
		Name: c.versionPath(name, version),
	}

	_, err := c.client.EnableSecretVersion(ctx, req)
	if err != nil {
		return c.mapError(err, secrets.ErrEnableVersionFailed)
	}

	return nil
}

// GetProvider returns the provider type
func (c *gcpClient) GetProvider() secrets.Provider {
	return secrets.ProviderGCP
}

// GetConfig returns the configuration
func (c *gcpClient) GetConfig() *secrets.Config {
	return c.config
}

// Exists checks if a secret exists
func (c *gcpClient) Exists(ctx context.Context, name string) (bool, error) {
	if !c.IsConnected() {
		return false, secrets.ErrNotConnected
	}

	req := &secretmanagerpb.GetSecretRequest{
		Name: c.secretPath(name),
	}

	_, err := c.client.GetSecret(ctx, req)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, c.mapError(err, secrets.ErrGetFailed)
	}

	return true, nil
}

// Helper methods

// projectPath returns the project path
func (c *gcpClient) projectPath() string {
	return fmt.Sprintf("projects/%s", c.config.GCP.ProjectID)
}

// secretPath returns the full secret resource path
func (c *gcpClient) secretPath(name string) string {
	return fmt.Sprintf("projects/%s/secrets/%s", c.config.GCP.ProjectID, name)
}

// versionPath returns the full version resource path
func (c *gcpClient) versionPath(name, version string) string {
	return fmt.Sprintf("projects/%s/secrets/%s/versions/%s", c.config.GCP.ProjectID, name, version)
}

// buildReplication builds the replication policy
func (c *gcpClient) buildReplication(opts *secrets.CreateOptions) *secretmanagerpb.Replication {
	if opts != nil && opts.Replication != nil && len(opts.Replication.Locations) > 0 {
		// User-managed replication
		var replicas []*secretmanagerpb.Replication_UserManaged_Replica
		for _, loc := range opts.Replication.Locations {
			replicas = append(replicas, &secretmanagerpb.Replication_UserManaged_Replica{
				Location: loc,
			})
		}
		return &secretmanagerpb.Replication{
			Replication: &secretmanagerpb.Replication_UserManaged_{
				UserManaged: &secretmanagerpb.Replication_UserManaged{
					Replicas: replicas,
				},
			},
		}
	}

	// Default to automatic replication
	return &secretmanagerpb.Replication{
		Replication: &secretmanagerpb.Replication_Automatic_{
			Automatic: &secretmanagerpb.Replication_Automatic{},
		},
	}
}

// secretToModel converts a GCP secret to the common model
func (c *gcpClient) secretToModel(secret *secretmanagerpb.Secret, version *secretmanagerpb.SecretVersion) *secrets.Secret {
	result := &secrets.Secret{
		Name:         extractSecretName(secret.Name),
		Labels:       secret.Labels,
		ResourceName: secret.Name,
		Metadata:     make(map[string]string),
	}

	if secret.CreateTime != nil {
		result.CreatedAt = secret.CreateTime.AsTime()
	}

	if version != nil {
		result.CurrentVersion = extractVersionNumber(version.Name)
		if version.CreateTime != nil {
			result.UpdatedAt = version.CreateTime.AsTime()
		}
	}

	return result
}

// versionToModel converts a GCP secret version to the common model
func (c *gcpClient) versionToModel(version *secretmanagerpb.SecretVersion) *secrets.SecretVersion {
	result := &secrets.SecretVersion{
		Version:      extractVersionNumber(version.Name),
		State:        c.mapVersionState(version.State),
		ResourceName: version.Name,
	}

	if version.CreateTime != nil {
		result.CreatedAt = version.CreateTime.AsTime()
	}

	if version.DestroyTime != nil {
		t := version.DestroyTime.AsTime()
		result.DestroyedAt = &t
	}

	return result
}

// mapVersionState maps GCP version state to common state
func (c *gcpClient) mapVersionState(state secretmanagerpb.SecretVersion_State) secrets.VersionState {
	switch state {
	case secretmanagerpb.SecretVersion_ENABLED:
		return secrets.VersionStateEnabled
	case secretmanagerpb.SecretVersion_DISABLED:
		return secrets.VersionStateDisabled
	case secretmanagerpb.SecretVersion_DESTROYED:
		return secrets.VersionStateDestroyed
	default:
		return secrets.VersionStatePending
	}
}

// mapError maps GCP errors to common errors
func (c *gcpClient) mapError(err error, defaultErr error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("%w: %v", defaultErr, err)
	}

	switch st.Code() {
	case codes.NotFound:
		if strings.Contains(err.Error(), "version") {
			return fmt.Errorf("%w: %v", secrets.ErrVersionNotFound, err)
		}
		return fmt.Errorf("%w: %v", secrets.ErrSecretNotFound, err)
	case codes.AlreadyExists:
		return fmt.Errorf("%w: %v", secrets.ErrSecretAlreadyExists, err)
	case codes.PermissionDenied:
		return fmt.Errorf("%w: %v", secrets.ErrPermissionDenied, err)
	case codes.InvalidArgument:
		return fmt.Errorf("%w: %v", secrets.ErrInvalidSecretName, err)
	case codes.FailedPrecondition:
		if strings.Contains(err.Error(), "DISABLED") {
			return fmt.Errorf("%w: %v", secrets.ErrVersionDisabled, err)
		}
		if strings.Contains(err.Error(), "DESTROYED") {
			return fmt.Errorf("%w: %v", secrets.ErrVersionDestroyed, err)
		}
		return fmt.Errorf("%w: %v", defaultErr, err)
	case codes.DeadlineExceeded:
		return fmt.Errorf("%w: %v", secrets.ErrOperationTimeout, err)
	default:
		return fmt.Errorf("%w: %v", defaultErr, err)
	}
}

// extractSecretName extracts the secret name from a full resource path
// e.g., "projects/my-project/secrets/my-secret" -> "my-secret"
func extractSecretName(resourceName string) string {
	parts := strings.Split(resourceName, "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	return resourceName
}

// extractVersionNumber extracts the version number from a full resource path
// e.g., "projects/my-project/secrets/my-secret/versions/1" -> "1"
func extractVersionNumber(resourceName string) string {
	parts := strings.Split(resourceName, "/")
	if len(parts) >= 6 {
		return parts[5]
	}
	return resourceName
}

// ParseVersion parses a version string to an integer
// Returns -1 if the version is not a valid integer (e.g., "latest")
func ParseVersion(version string) int {
	if version == secrets.VersionLatest || version == "" {
		return -1
	}
	v, err := strconv.Atoi(version)
	if err != nil {
		return -1
	}
	return v
}
