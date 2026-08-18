// Package secrets provides unit tests for the secrets package
package secrets

import (
	"errors"
	"testing"
	"time"
)

// TestConfigValidate tests the Config.Validate method
func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr error
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: ErrInvalidConfig,
		},
		{
			name: "empty provider",
			config: &Config{
				Provider: "",
				Timeout:  10 * time.Second,
			},
			wantErr: ErrInvalidConfig,
		},
		{
			name: "zero timeout",
			config: &Config{
				Provider: ProviderGCP,
				Timeout:  0,
				GCP: &GCPConfig{
					ProjectID: "test-project",
				},
			},
			wantErr: ErrInvalidConfig,
		},
		{
			name: "negative timeout",
			config: &Config{
				Provider: ProviderGCP,
				Timeout:  -1 * time.Second,
				GCP: &GCPConfig{
					ProjectID: "test-project",
				},
			},
			wantErr: ErrInvalidConfig,
		},
		{
			name: "unsupported provider",
			config: &Config{
				Provider: "unsupported",
				Timeout:  10 * time.Second,
			},
			wantErr: ErrUnsupportedProvider,
		},
		{
			name: "GCP provider without GCP config",
			config: &Config{
				Provider: ProviderGCP,
				Timeout:  10 * time.Second,
			},
			wantErr: ErrInvalidConfig,
		},
		{
			name: "AWS provider without AWS config",
			config: &Config{
				Provider: ProviderAWS,
				Timeout:  10 * time.Second,
			},
			wantErr: ErrInvalidConfig,
		},
		{
			name: "valid GCP config",
			config: &Config{
				Provider: ProviderGCP,
				Timeout:  30 * time.Second,
				GCP: &GCPConfig{
					ProjectID: "test-project",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid AWS config",
			config: &Config{
				Provider: ProviderAWS,
				Timeout:  30 * time.Second,
				AWS: &AWSConfig{
					Region: "us-east-1",
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Validate() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestGCPConfigValidate tests the GCPConfig.Validate method
func TestGCPConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *GCPConfig
		wantErr error
	}{
		{
			name: "empty project ID",
			config: &GCPConfig{
				ProjectID: "",
			},
			wantErr: ErrInvalidConfig,
		},
		{
			name: "valid with project ID",
			config: &GCPConfig{
				ProjectID: "test-project",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Validate() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestAWSConfigValidate tests the AWSConfig.Validate method
func TestAWSConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *AWSConfig
		wantErr error
	}{
		{
			name: "empty region",
			config: &AWSConfig{
				Region: "",
			},
			wantErr: ErrInvalidConfig,
		},
		{
			name: "valid with region",
			config: &AWSConfig{
				Region: "us-east-1",
			},
			wantErr: nil,
		},
		{
			name: "valid with different region",
			config: &AWSConfig{
				Region: "us-west-2",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Validate() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestValidateSecretName tests the ValidateSecretName function
func TestValidateSecretName(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantErr error
	}{
		{
			name:    "empty name",
			secret:  "",
			wantErr: ErrEmptySecretName,
		},
		{
			name:    "valid simple name",
			secret:  "my-secret",
			wantErr: nil,
		},
		{
			name:    "valid name with underscores",
			secret:  "my_secret_name",
			wantErr: nil,
		},
		{
			name:    "valid name with numbers",
			secret:  "secret123",
			wantErr: nil,
		},
		{
			name:    "valid name with path",
			secret:  "prod/database/password",
			wantErr: nil,
		},
		{
			name:    "valid name with special chars",
			secret:  "secret+name=test@domain",
			wantErr: nil,
		},
		{
			name:    "invalid name with space",
			secret:  "my secret",
			wantErr: ErrInvalidSecretName,
		},
		{
			name:    "invalid name with special char",
			secret:  "secret#name",
			wantErr: ErrInvalidSecretName,
		},
		{
			name:    "valid long name (255 chars)",
			secret:  string(make([]byte, 255)),
			wantErr: ErrInvalidSecretName, // Will fail because null bytes are invalid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSecretName(tt.secret)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ValidateSecretName() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateSecretName() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateSecretName() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestValidateSecretNameLength tests secret name length validation
func TestValidateSecretNameLength(t *testing.T) {
	// Create a valid name that exceeds 255 characters
	longName := ""
	for i := 0; i < 260; i++ {
		longName += "a"
	}

	err := ValidateSecretName(longName)
	if err == nil {
		t.Error("ValidateSecretName() should fail for names > 255 characters")
	}
	if !errors.Is(err, ErrInvalidSecretName) {
		t.Errorf("ValidateSecretName() error = %v, want %v", err, ErrInvalidSecretName)
	}
}

// TestNormalizeVersion tests the NormalizeVersion function
func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{
			name:    "empty version",
			version: "",
			want:    VersionLatest,
		},
		{
			name:    "latest version",
			version: "latest",
			want:    "latest",
		},
		{
			name:    "numeric version",
			version: "1",
			want:    "1",
		},
		{
			name:    "version string",
			version: "v1.0.0",
			want:    "v1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeVersion(tt.version)
			if got != tt.want {
				t.Errorf("NormalizeVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsLatestVersion tests the IsLatestVersion function
func TestIsLatestVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{
			name:    "empty version",
			version: "",
			want:    true,
		},
		{
			name:    "latest string",
			version: "latest",
			want:    true,
		},
		{
			name:    "AWSCURRENT",
			version: "AWSCURRENT",
			want:    true,
		},
		{
			name:    "numeric version",
			version: "1",
			want:    false,
		},
		{
			name:    "AWSPREVIOUS",
			version: "AWSPREVIOUS",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLatestVersion(tt.version)
			if got != tt.want {
				t.Errorf("IsLatestVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestProviderConstants tests provider constant values
func TestProviderConstants(t *testing.T) {
	tests := []struct {
		provider Provider
		want     string
	}{
		{ProviderGCP, "gcp"},
		{ProviderAWS, "aws"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			if string(tt.provider) != tt.want {
				t.Errorf("Provider = %v, want %v", tt.provider, tt.want)
			}
		})
	}
}

// TestVersionStateConstants tests version state constant values
func TestVersionStateConstants(t *testing.T) {
	tests := []struct {
		state VersionState
		want  string
	}{
		{VersionStateEnabled, "enabled"},
		{VersionStateDisabled, "disabled"},
		{VersionStateDestroyed, "destroyed"},
		{VersionStatePending, "pending"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if string(tt.state) != tt.want {
				t.Errorf("VersionState = %v, want %v", tt.state, tt.want)
			}
		})
	}
}

// TestSecretValueString tests the SecretValue.String method
func TestSecretValueString(t *testing.T) {
	tests := []struct {
		name  string
		value *SecretValue
		want  string
	}{
		{
			name: "simple string value",
			value: &SecretValue{
				Value: []byte("my-secret-value"),
			},
			want: "my-secret-value",
		},
		{
			name: "empty value",
			value: &SecretValue{
				Value: []byte{},
			},
			want: "",
		},
		{
			name: "json value",
			value: &SecretValue{
				Value: []byte(`{"key": "value"}`),
			},
			want: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.value.String()
			if got != tt.want {
				t.Errorf("SecretValue.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestErrors tests error types
func TestErrors(t *testing.T) {
	tests := []struct {
		err  error
		name string
	}{
		{ErrNotConnected, "ErrNotConnected"},
		{ErrInvalidConfig, "ErrInvalidConfig"},
		{ErrConnectionFailed, "ErrConnectionFailed"},
		{ErrSecretNotFound, "ErrSecretNotFound"},
		{ErrVersionNotFound, "ErrVersionNotFound"},
		{ErrSecretAlreadyExists, "ErrSecretAlreadyExists"},
		{ErrPermissionDenied, "ErrPermissionDenied"},
		{ErrInvalidSecretName, "ErrInvalidSecretName"},
		{ErrInvalidSecretValue, "ErrInvalidSecretValue"},
		{ErrCreateFailed, "ErrCreateFailed"},
		{ErrGetFailed, "ErrGetFailed"},
		{ErrUpdateFailed, "ErrUpdateFailed"},
		{ErrDeleteFailed, "ErrDeleteFailed"},
		{ErrListFailed, "ErrListFailed"},
		{ErrVersionDisabled, "ErrVersionDisabled"},
		{ErrVersionDestroyed, "ErrVersionDestroyed"},
		{ErrOperationTimeout, "ErrOperationTimeout"},
		{ErrAlreadyConnected, "ErrAlreadyConnected"},
		{ErrUnsupportedProvider, "ErrUnsupportedProvider"},
		{ErrEmptySecretName, "ErrEmptySecretName"},
		{ErrEmptySecretValue, "ErrEmptySecretValue"},
		{ErrInvalidVersion, "ErrInvalidVersion"},
		{ErrDestroyVersionFailed, "ErrDestroyVersionFailed"},
		{ErrDisableVersionFailed, "ErrDisableVersionFailed"},
		{ErrEnableVersionFailed, "ErrEnableVersionFailed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
			if tt.err.Error() == "" {
				t.Errorf("%s.Error() should not be empty", tt.name)
			}
		})
	}
}

// TestNewWithoutProvider tests New function without registered provider
func TestNewWithoutProvider(t *testing.T) {
	// Clear any registered factories for this test
	// Note: In real scenario, you would need to handle this differently
	// since the factories map is package-level

	config := &Config{
		Provider: "nonexistent",
		Timeout:  10 * time.Second,
	}

	_, err := New(config)
	if err == nil {
		t.Error("New() should fail for unsupported provider")
	}
}

// TestNewWithNilConfig tests New function with nil config
func TestNewWithNilConfig(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Error("New() should fail for nil config")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("New() error = %v, want %v", err, ErrInvalidConfig)
	}
}

// TestSecretStruct tests the Secret struct
func TestSecretStruct(t *testing.T) {
	now := time.Now()
	secret := &Secret{
		Name:           "test-secret",
		Description:    "Test description",
		Labels:         map[string]string{"env": "test"},
		CreatedAt:      now,
		UpdatedAt:      now,
		VersionCount:   3,
		CurrentVersion: "3",
		ARN:            "arn:aws:secretsmanager:us-east-1:123456789:secret:test-secret",
		ResourceName:   "projects/test-project/secrets/test-secret",
		Metadata:       map[string]string{"custom": "value"},
	}

	if secret.Name != "test-secret" {
		t.Errorf("Secret.Name = %v, want %v", secret.Name, "test-secret")
	}
	if secret.Description != "Test description" {
		t.Errorf("Secret.Description = %v, want %v", secret.Description, "Test description")
	}
	if secret.Labels["env"] != "test" {
		t.Errorf("Secret.Labels[env] = %v, want %v", secret.Labels["env"], "test")
	}
	if secret.VersionCount != 3 {
		t.Errorf("Secret.VersionCount = %v, want %v", secret.VersionCount, 3)
	}
	if secret.CurrentVersion != "3" {
		t.Errorf("Secret.CurrentVersion = %v, want %v", secret.CurrentVersion, "3")
	}
}

// TestSecretVersionStruct tests the SecretVersion struct
func TestSecretVersionStruct(t *testing.T) {
	now := time.Now()
	destroyedAt := now.Add(24 * time.Hour)

	version := &SecretVersion{
		Version:       "1",
		State:         VersionStateEnabled,
		CreatedAt:     now,
		DestroyedAt:   &destroyedAt,
		ResourceName:  "projects/test-project/secrets/test-secret/versions/1",
		VersionStages: []string{"AWSCURRENT"},
	}

	if version.Version != "1" {
		t.Errorf("SecretVersion.Version = %v, want %v", version.Version, "1")
	}
	if version.State != VersionStateEnabled {
		t.Errorf("SecretVersion.State = %v, want %v", version.State, VersionStateEnabled)
	}
	if version.DestroyedAt == nil {
		t.Error("SecretVersion.DestroyedAt should not be nil")
	}
	if len(version.VersionStages) != 1 || version.VersionStages[0] != "AWSCURRENT" {
		t.Errorf("SecretVersion.VersionStages = %v, want [AWSCURRENT]", version.VersionStages)
	}
}

// TestCreateOptions tests the CreateOptions struct
func TestCreateOptions(t *testing.T) {
	opts := &CreateOptions{
		Description: "My secret description",
		Labels: map[string]string{
			"environment": "production",
			"team":        "backend",
		},
		Replication: &ReplicationPolicy{
			Automatic: true,
		},
	}

	if opts.Description != "My secret description" {
		t.Errorf("CreateOptions.Description = %v, want %v", opts.Description, "My secret description")
	}
	if len(opts.Labels) != 2 {
		t.Errorf("CreateOptions.Labels length = %v, want %v", len(opts.Labels), 2)
	}
	if opts.Replication == nil || !opts.Replication.Automatic {
		t.Error("CreateOptions.Replication should be automatic")
	}
}

// TestGetOptions tests the GetOptions struct
func TestGetOptions(t *testing.T) {
	opts := &GetOptions{
		Version:      "1",
		VersionStage: "AWSCURRENT",
	}

	if opts.Version != "1" {
		t.Errorf("GetOptions.Version = %v, want %v", opts.Version, "1")
	}
	if opts.VersionStage != "AWSCURRENT" {
		t.Errorf("GetOptions.VersionStage = %v, want %v", opts.VersionStage, "AWSCURRENT")
	}
}

// TestDeleteOptions tests the DeleteOptions struct
func TestDeleteOptions(t *testing.T) {
	opts := &DeleteOptions{
		Force:              true,
		RecoveryWindowDays: 7,
	}

	if !opts.Force {
		t.Error("DeleteOptions.Force should be true")
	}
	if opts.RecoveryWindowDays != 7 {
		t.Errorf("DeleteOptions.RecoveryWindowDays = %v, want %v", opts.RecoveryWindowDays, 7)
	}
}

// TestListOptions tests the ListOptions struct
func TestListOptions(t *testing.T) {
	opts := &ListOptions{
		Prefix:     "prod/",
		MaxResults: 100,
		PageToken:  "next-page-token",
		Filter:     "name:database",
	}

	if opts.Prefix != "prod/" {
		t.Errorf("ListOptions.Prefix = %v, want %v", opts.Prefix, "prod/")
	}
	if opts.MaxResults != 100 {
		t.Errorf("ListOptions.MaxResults = %v, want %v", opts.MaxResults, 100)
	}
	if opts.PageToken != "next-page-token" {
		t.Errorf("ListOptions.PageToken = %v, want %v", opts.PageToken, "next-page-token")
	}
	if opts.Filter != "name:database" {
		t.Errorf("ListOptions.Filter = %v, want %v", opts.Filter, "name:database")
	}
}

// TestListVersionsOptions tests the ListVersionsOptions struct
func TestListVersionsOptions(t *testing.T) {
	opts := &ListVersionsOptions{
		MaxResults:       50,
		PageToken:        "version-page-token",
		IncludeDisabled:  true,
		IncludeDestroyed: false,
	}

	if opts.MaxResults != 50 {
		t.Errorf("ListVersionsOptions.MaxResults = %v, want %v", opts.MaxResults, 50)
	}
	if !opts.IncludeDisabled {
		t.Error("ListVersionsOptions.IncludeDisabled should be true")
	}
	if opts.IncludeDestroyed {
		t.Error("ListVersionsOptions.IncludeDestroyed should be false")
	}
}

// TestListResult tests the ListResult struct
func TestListResult(t *testing.T) {
	result := &ListResult{
		Secrets: []*Secret{
			{Name: "secret1"},
			{Name: "secret2"},
		},
		NextPageToken: "next-token",
	}

	if len(result.Secrets) != 2 {
		t.Errorf("ListResult.Secrets length = %v, want %v", len(result.Secrets), 2)
	}
	if result.NextPageToken != "next-token" {
		t.Errorf("ListResult.NextPageToken = %v, want %v", result.NextPageToken, "next-token")
	}
}

// TestListVersionsResult tests the ListVersionsResult struct
func TestListVersionsResult(t *testing.T) {
	result := &ListVersionsResult{
		Versions: []*SecretVersion{
			{Version: "1", State: VersionStateEnabled},
			{Version: "2", State: VersionStateDisabled},
		},
		NextPageToken: "version-next-token",
	}

	if len(result.Versions) != 2 {
		t.Errorf("ListVersionsResult.Versions length = %v, want %v", len(result.Versions), 2)
	}
	if result.NextPageToken != "version-next-token" {
		t.Errorf("ListVersionsResult.NextPageToken = %v, want %v", result.NextPageToken, "version-next-token")
	}
}

// TestUpdateMetadataOptions tests the UpdateMetadataOptions struct
func TestUpdateMetadataOptions(t *testing.T) {
	desc := "Updated description"
	opts := &UpdateMetadataOptions{
		Description: &desc,
		Labels: map[string]string{
			"updated": "true",
		},
	}

	if opts.Description == nil || *opts.Description != "Updated description" {
		t.Error("UpdateMetadataOptions.Description should be 'Updated description'")
	}
	if opts.Labels["updated"] != "true" {
		t.Errorf("UpdateMetadataOptions.Labels[updated] = %v, want %v", opts.Labels["updated"], "true")
	}
}

// TestReplicationPolicy tests the ReplicationPolicy struct
func TestReplicationPolicy(t *testing.T) {
	// Test automatic replication
	autoPolicy := &ReplicationPolicy{
		Automatic: true,
	}
	if !autoPolicy.Automatic {
		t.Error("ReplicationPolicy.Automatic should be true")
	}

	// Test user-managed replication
	userPolicy := &ReplicationPolicy{
		Automatic: false,
		Locations: []string{"us-east1", "us-west1", "europe-west1"},
	}
	if userPolicy.Automatic {
		t.Error("ReplicationPolicy.Automatic should be false")
	}
	if len(userPolicy.Locations) != 3 {
		t.Errorf("ReplicationPolicy.Locations length = %v, want %v", len(userPolicy.Locations), 3)
	}
}

// BenchmarkValidateSecretName benchmarks the ValidateSecretName function
func BenchmarkValidateSecretName(b *testing.B) {
	names := []string{
		"simple-secret",
		"prod/database/password",
		"my_secret_with_underscores",
		"secret123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, name := range names {
			ValidateSecretName(name)
		}
	}
}

// BenchmarkNormalizeVersion benchmarks the NormalizeVersion function
func BenchmarkNormalizeVersion(b *testing.B) {
	versions := []string{"", "latest", "1", "10", "100"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range versions {
			NormalizeVersion(v)
		}
	}
}

// BenchmarkIsLatestVersion benchmarks the IsLatestVersion function
func BenchmarkIsLatestVersion(b *testing.B) {
	versions := []string{"", "latest", "AWSCURRENT", "1", "AWSPREVIOUS"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range versions {
			IsLatestVersion(v)
		}
	}
}
