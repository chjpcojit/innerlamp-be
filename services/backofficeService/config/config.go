// Package config loads backofficeService's configuration from a YAML file and
// resolves sensitive fields (DB password, JWT signing key) through a secret
// manager rather than storing them in the YAML itself. Same shape as
// appService/config, deliberately trimmed to what the backoffice uses.
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
type SecretGetter interface {
	Get(ctx context.Context, name string, opts *secrets.GetOptions) (*secrets.SecretValue, error)
}

// Duration wraps time.Duration so it can be written as a string in YAML.
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
func (d Duration) Duration() time.Duration { return time.Duration(d) }

// AppConfig is backofficeService's full runtime configuration.
type AppConfig struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	// BootstrapAdmin is the default admin created when the admins table is empty.
	BootstrapAdmin BootstrapAdminConfig `yaml:"bootstrap_admin"`
	Account        AccountConfig        `yaml:"account"`
}

// AccountConfig holds account-creation settings.
type AccountConfig struct {
	// FixedTempPassword, when non-empty, is the temporary password given to
	// EVERY new account instead of a random one. ⚠️ DEV ONLY: anyone who knows a
	// new account's email can sign in with it before its owner does. Leave empty
	// in production so each account gets a random password.
	FixedTempPassword string `yaml:"fixed_temp_password"`
	// TempPasswordTTL is how long a temporary password (new account or reset)
	// works. Zero disables expiry.
	TempPasswordTTL Duration `yaml:"temp_password_ttl"`
}

// BootstrapAdminConfig describes the first-run admin. The account is created
// with must_change_password, so the password here only ever lives until the
// first login. Set enabled: false (and use cmd/seedadmin) in production.
type BootstrapAdminConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Addr string `yaml:"addr"`
}

// DatabaseConfig holds Postgres connection settings. Password is resolved from
// PasswordSecret at load time and never read from YAML.
type DatabaseConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	Username       string `yaml:"username"`
	PasswordSecret string `yaml:"password_secret"`
	Password       string `yaml:"-"`
	Database       string `yaml:"database"`
	SSLMode        string `yaml:"ssl_mode"`
}

// JWTConfig holds admin-token signing settings. The secret key and audience
// MUST differ from appService's so a user token can never authenticate against
// the backoffice (and vice versa). Only an access token is issued today;
// RefreshTokenExpiration is set only because pkg/jwt.New rejects a config
// without a positive value.
type JWTConfig struct {
	Algorithm              string   `yaml:"algorithm"`
	SecretKeySecret        string   `yaml:"secret_key_secret"`
	SecretKey              string   `yaml:"-"`
	AccessTokenExpiration  Duration `yaml:"access_token_expiration"`
	RefreshTokenExpiration Duration `yaml:"refresh_token_expiration"`
	Issuer                 string   `yaml:"issuer"`
	Audience               string   `yaml:"audience"`
}

// Load reads YAML from source and resolves every *_secret reference against sg.
func Load(ctx context.Context, source pkgconfig.LoadSource, sg SecretGetter) (*AppConfig, error) {
	var cfg AppConfig
	if err := pkgconfig.LoadAndUnmarshal(ctx, source, &cfg); err != nil {
		return nil, fmt.Errorf("load backoffice config: %w", err)
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
