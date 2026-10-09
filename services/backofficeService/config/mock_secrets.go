package config

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v3"

	"chjpcoj.com.innerlamp/pkg/secrets"
)

type devSecretsFile struct {
	DevSecrets map[string]string `yaml:"dev_secrets"`
}

// MockSecretGetter is a SecretGetter backed by a plain name->value map.
//
// ⚠️ DEV ONLY: it resolves secret names against plaintext values instead of a
// real secret manager. Use pkg/secrets/gcp (or aws) in any non-local environment.
type MockSecretGetter struct {
	values map[string]string
}

// NewMockSecretGetterFromYAML builds a MockSecretGetter from the dev_secrets
// section of a YAML document.
func NewMockSecretGetterFromYAML(data []byte) (*MockSecretGetter, error) {
	var f devSecretsFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse dev_secrets: %w", err)
	}
	return &MockSecretGetter{values: f.DevSecrets}, nil
}

// Get implements SecretGetter by looking name up in the backing map.
func (m *MockSecretGetter) Get(_ context.Context, name string, _ *secrets.GetOptions) (*secrets.SecretValue, error) {
	v, ok := m.values[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", secrets.ErrSecretNotFound, name)
	}
	return &secrets.SecretValue{Name: name, Value: []byte(v), State: secrets.VersionStateEnabled}, nil
}
