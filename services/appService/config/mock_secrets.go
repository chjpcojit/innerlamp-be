package config

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v3"

	"chjpcoj.com.innerlamp/pkg/secrets"
)

// devSecretsFile is the shape of the dev_secrets section of the YAML config.
// It's read directly rather than via AppConfig so unrelated top-level keys
// (server, database, jwt, storage) don't need to know about it.
type devSecretsFile struct {
	DevSecrets map[string]string `yaml:"dev_secrets"`
}

// MockSecretGetter is a SecretGetter backed by a plain name->value map.
//
// ⚠️ DEV ONLY: it resolves secret names against plaintext values instead of
// a real secret manager. Use a real secrets.Client (GCP/AWS) for any
// non-local environment.
type MockSecretGetter struct {
	values map[string]string
}

// NewMockSecretGetter returns a MockSecretGetter backed by values.
func NewMockSecretGetter(values map[string]string) *MockSecretGetter {
	return &MockSecretGetter{values: values}
}

// NewMockSecretGetterFromYAML builds a MockSecretGetter from the dev_secrets
// section of a YAML document (see AppConfig / config.dev.yaml).
func NewMockSecretGetterFromYAML(data []byte) (*MockSecretGetter, error) {
	var f devSecretsFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse dev_secrets: %w", err)
	}
	return NewMockSecretGetter(f.DevSecrets), nil
}

// Get implements SecretGetter by looking name up in the backing map.
func (m *MockSecretGetter) Get(_ context.Context, name string, _ *secrets.GetOptions) (*secrets.SecretValue, error) {
	v, ok := m.values[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", secrets.ErrSecretNotFound, name)
	}
	return &secrets.SecretValue{
		Name:  name,
		Value: []byte(v),
		State: secrets.VersionStateEnabled,
	}, nil
}
