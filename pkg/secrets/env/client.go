// Package env provides a secrets.Client implementation that reads values
// from environment variables. Intended for local development only.
//
// Secret name → env var transformation:
//
//	hyphens and forward slashes → underscores, then uppercase
//	e.g.  "elcora-mongodb-uri"  →  ELCORA_MONGODB_URI
//	e.g.  "elcora/users/db"     →  ELCORA_USERS_DB
//
// Usage:
//
//	import secretsenv "base_project/pkg/secrets/env"
//	client := secretsenv.New()
//	// Set env vars: ELCORA_MONGODB_URI=mongodb://...
package env

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"base_project/pkg/secrets"
)

type client struct{}

// New returns a secrets.Client backed by environment variables.
// No connection setup is required — IsConnected always returns true.
func New() secrets.Client {
	return &client{}
}

// Connect is a no-op for the env provider.
func (c *client) Connect(_ context.Context) error { return nil }

// Close is a no-op for the env provider.
func (c *client) Close() error { return nil }

// IsConnected always returns true for the env provider.
func (c *client) IsConnected() bool { return true }

// Get retrieves a secret value from an environment variable.
// The env var name is derived by replacing hyphens/slashes with underscores
// and converting to uppercase (e.g. "elcora-mongodb-uri" → ELCORA_MONGODB_URI).
// Returns ErrSecretNotFound if the env var is not set or is empty.
func (c *client) Get(_ context.Context, name string, _ *secrets.GetOptions) (*secrets.SecretValue, error) {
	if name == "" {
		return nil, secrets.ErrEmptySecretName
	}

	envKey := nameToEnvVar(name)
	value := os.Getenv(envKey)
	if value == "" {
		return nil, fmt.Errorf("%w: %q (env var: %s)", secrets.ErrSecretNotFound, name, envKey)
	}

	return &secrets.SecretValue{
		Name:      name,
		Value:     []byte(value),
		Version:   "env",
		CreatedAt: time.Time{},
		State:     secrets.VersionStateEnabled,
	}, nil
}

// Exists reports whether the corresponding env var is set and non-empty.
func (c *client) Exists(_ context.Context, name string) (bool, error) {
	if name == "" {
		return false, secrets.ErrEmptySecretName
	}
	return os.Getenv(nameToEnvVar(name)) != "", nil
}

// GetProvider returns the provider identifier.
func (c *client) GetProvider() secrets.Provider { return secrets.Provider("env") }

// GetConfig returns a minimal Config representing this provider.
func (c *client) GetConfig() *secrets.Config {
	return &secrets.Config{Provider: secrets.Provider("env")}
}

// nameToEnvVar converts a secret name to an env var name.
// e.g. "elcora-mongodb-uri" → "ELCORA_MONGODB_URI"
func nameToEnvVar(name string) string {
	r := strings.NewReplacer("-", "_", "/", "_")
	return strings.ToUpper(r.Replace(name))
}

// — Read-only provider: write operations are not supported ———————————————

func (c *client) Create(_ context.Context, _ string, _ []byte, _ *secrets.CreateOptions) (*secrets.Secret, error) {
	return nil, secrets.ErrUnsupportedProvider
}

func (c *client) GetMetadata(_ context.Context, _ string) (*secrets.Secret, error) {
	return nil, secrets.ErrUnsupportedProvider
}

func (c *client) Update(_ context.Context, _ string, _ []byte) (*secrets.SecretVersion, error) {
	return nil, secrets.ErrUnsupportedProvider
}

func (c *client) UpdateMetadata(_ context.Context, _ string, _ *secrets.UpdateMetadataOptions) (*secrets.Secret, error) {
	return nil, secrets.ErrUnsupportedProvider
}

func (c *client) Delete(_ context.Context, _ string, _ *secrets.DeleteOptions) error {
	return secrets.ErrUnsupportedProvider
}

func (c *client) List(_ context.Context, _ *secrets.ListOptions) (*secrets.ListResult, error) {
	return nil, secrets.ErrUnsupportedProvider
}

func (c *client) GetVersion(_ context.Context, _ string, _ string) (*secrets.SecretValue, error) {
	return nil, secrets.ErrUnsupportedProvider
}

func (c *client) ListVersions(_ context.Context, _ string, _ *secrets.ListVersionsOptions) (*secrets.ListVersionsResult, error) {
	return nil, secrets.ErrUnsupportedProvider
}

func (c *client) DestroyVersion(_ context.Context, _ string, _ string) error {
	return secrets.ErrUnsupportedProvider
}

func (c *client) DisableVersion(_ context.Context, _ string, _ string) error {
	return secrets.ErrUnsupportedProvider
}

func (c *client) EnableVersion(_ context.Context, _ string, _ string) error {
	return secrets.ErrUnsupportedProvider
}
