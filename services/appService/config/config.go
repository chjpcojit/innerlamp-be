// Package config loads appService's configuration from a YAML file and
// resolves sensitive fields (DB password, JWT signing key, ...) through a
// secret manager rather than storing them in the YAML itself.
//
// YAML holds only the *name* of a secret (e.g. database.password_secret);
// Load fetches the actual value at startup via the injected SecretGetter.
package config

import (
	"context"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"

	pkgconfig "chjpcoj.com.innerlamp/pkg/config"
	"chjpcoj.com.innerlamp/pkg/secrets"
)

// SecretGetter is the minimal slice of secrets.Client this package needs.
// secrets.Client already implements this method with an identical signature,
// so any real provider (GCP/AWS) satisfies it with no adapter code.
type SecretGetter interface {
	Get(ctx context.Context, name string, opts *secrets.GetOptions) (*secrets.SecretValue, error)
}

// Duration wraps time.Duration so it can be written as a plain string in
// YAML (e.g. "15m", "168h") — yaml.v3 has no built-in time.Duration support.
type Duration time.Duration

// UnmarshalYAML parses a duration string into d.
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	if s == "" {
		*d = 0
		return nil
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(parsed)
	return nil
}

// Duration returns the wrapped time.Duration.
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// AppConfig is appService's full runtime configuration.
type AppConfig struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	Storage  StorageConfig  `yaml:"storage"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Addr string `yaml:"addr"`
}

// DatabaseConfig holds Postgres connection settings. PasswordSecret names the
// secret to resolve into Password at load time; Password itself is never
// read from YAML.
type DatabaseConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	Username       string `yaml:"username"`
	PasswordSecret string `yaml:"password_secret"`
	Password       string `yaml:"-"`
	Database       string `yaml:"database"`
	SSLMode        string `yaml:"ssl_mode"`
}

// JWTConfig holds JWT signing settings. SecretKeySecret names the secret to
// resolve into SecretKey at load time; SecretKey itself is never read from YAML.
type JWTConfig struct {
	Algorithm              string   `yaml:"algorithm"`
	SecretKeySecret        string   `yaml:"secret_key_secret"`
	SecretKey              string   `yaml:"-"`
	AccessTokenExpiration  Duration `yaml:"access_token_expiration"`
	RefreshTokenExpiration Duration `yaml:"refresh_token_expiration"`
	Issuer                 string   `yaml:"issuer"`
	Audience               string   `yaml:"audience"`
}

// StorageConfig holds object storage settings.
type StorageConfig struct {
	Backend       string `yaml:"backend"`
	ProjectID     string `yaml:"project_id"`
	DefaultBucket string `yaml:"default_bucket"`
}

// Load reads YAML from source (local file, s3://, or gs:// — see
// pkg/config.LoadYAML) and resolves every *_secret reference against sg.
func Load(ctx context.Context, source pkgconfig.LoadSource, sg SecretGetter) (*AppConfig, error) {
	var cfg AppConfig
	if err := pkgconfig.LoadAndUnmarshal(ctx, source, &cfg); err != nil {
		return nil, fmt.Errorf("load app config: %w", err)
	}

	if err := cfg.resolveSecrets(ctx, sg); err != nil {
		return nil, fmt.Errorf("resolve secrets: %w", err)
	}

	return &cfg, nil
}

func (c *AppConfig) resolveSecrets(ctx context.Context, sg SecretGetter) error {
	if c.Database.PasswordSecret != "" {
		v, err := getSecret(ctx, sg, c.Database.PasswordSecret)
		if err != nil {
			return fmt.Errorf("database.password_secret %q: %w", c.Database.PasswordSecret, err)
		}
		c.Database.Password = v
	}

	if c.JWT.SecretKeySecret != "" {
		v, err := getSecret(ctx, sg, c.JWT.SecretKeySecret)
		if err != nil {
			return fmt.Errorf("jwt.secret_key_secret %q: %w", c.JWT.SecretKeySecret, err)
		}
		c.JWT.SecretKey = v
	}

	return nil
}

func getSecret(ctx context.Context, sg SecretGetter, name string) (string, error) {
	sv, err := sg.Get(ctx, name, nil)
	if err != nil {
		return "", err
	}
	return sv.String(), nil
}
