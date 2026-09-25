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
	Server      ServerConfig      `yaml:"server"`
	Database    DatabaseConfig    `yaml:"database"`
	JWT         JWTConfig         `yaml:"jwt"`
	Storage     StorageConfig     `yaml:"storage"`
	OTP         OTPConfig         `yaml:"otp"`
	SMS         SMSConfig         `yaml:"sms"`
	Login       LoginConfig       `yaml:"login"`
	Attestation AttestationConfig `yaml:"attestation"`
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

// SMSConfig selects which pkg/sms provider the composition root wires up —
// "log" (pkg/sms/logsms, the only implementation today) in every
// environment until a real provider (Twilio, AWS SNS, a GCP equivalent)
// ships its own subpackage.
type SMSConfig struct {
	Provider string `yaml:"provider"`
}

// OTPConfig holds the OTP policy: how long a code lives, how often a code may
// be sent, and when a number gets locked.
//
// These numbers used to be a const block inside the registration use case,
// justified there as "business decisions, not configuration". That reasoning is
// wrong for all of them: each is a number a dev environment shortens to exercise
// expiry and production lengthens under load, which is the definition of
// configuration. The use case now takes them through its constructor, the way
// it already took sessionTTL.
//
// v5.0 split the old "resend" cap into a per-phone *send* policy: the first
// send of a cycle is free, every later request-otp AND resend-otp counts, and
// hitting the cap locks sending for SendLockDuration. See design.md
// §"OTP send-limit policy".
type OTPConfig struct {
	// TTL is how long an issued code can be verified.
	TTL Duration `yaml:"ttl"`

	// SendCooldown is the floor between two sends for one number (AC5), applied
	// to both request-otp and resend-otp.
	SendCooldown Duration `yaml:"send_cooldown"`

	// MaxSendAttempts caps the counted sends in one cycle (AC5). The first send
	// is free, so a cycle produces at most MaxSendAttempts+1 codes: the free
	// first plus the counted ones. It does NOT count wrong-code guesses — that
	// is MaxAttempts.
	MaxSendAttempts int `yaml:"max_send_attempts"`

	// SendLockDuration is how long sending is blocked once a cycle hits the cap
	// (AC5), and also how long a quiet cycle's counter survives before it resets.
	// The two use the same number so "the lock expired" and "the counter aged
	// out" are one boundary.
	SendLockDuration Duration `yaml:"send_lock_duration"`

	// MaxAttempts is how many wrong codes lock the number (AC12).
	MaxAttempts int `yaml:"max_attempts"`

	// LockDuration is how long that lock holds (AC12).
	LockDuration Duration `yaml:"lock_duration"`

	// PendingRegistrationTTL is how long the captured account payload survives
	// an abandoned registration. v5.1 decoupled it from TTL: the payload now
	// outlives the 30-minute send lock by a wide margin (24h) so a caller can
	// resume by resending after the lock lifts, and so a reinstall can restart
	// by re-requesting (request-otp overwrites the payload).
	//
	// The cost is a bcrypt password hash sitting on the server for that long.
	// The cleanup job (Deferred) deletes expired rows; this TTL is the backstop
	// for when that job is not running yet.
	PendingRegistrationTTL Duration `yaml:"pending_registration_ttl"`
}

// LoginConfig holds US-002's brute-force policy: how many consecutive wrong
// passwords lock a phone number, for how long, and how much idle time resets
// the counter.
//
// Same reasoning as OTPConfig: these are numbers a dev environment shortens to
// exercise the lock and production lengthens under real traffic, so they are
// configuration rather than constants in the use case. The refresh token's own
// lifetime is not here — it is JWTConfig.RefreshTokenExpiration, the same value
// the signed token and the stored session row both use.
type LoginConfig struct {
	// MaxFailedAttempts is how many consecutive failures lock the number (AC7).
	// The failure that reaches this count still answers 401; only the next
	// request sees 429.
	MaxFailedAttempts int `yaml:"max_failed_attempts"`

	// LockDuration is how long the lock holds (AC7), measured from the failure
	// that tripped it.
	LockDuration Duration `yaml:"lock_duration"`

	// FailureResetWindow is how long a failure stays "consecutive" (AC7). A
	// failure arriving more than this long after the previous one starts the
	// count over at 1. Shipped equal to LockDuration — an expired lock and a
	// stale counter are the same boundary — but settable separately because the
	// equality is a default, not a constraint.
	FailureResetWindow Duration `yaml:"failure_reset_window"`
}

// AttestationConfig holds the attestation policy and backend selection for the
// attestation feature. It mirrors the shape of the other policy blocks
// (OTPConfig, LoginConfig): values a dev environment shortens (the challenge
// TTL, the freshness window) and a real deployment fills in (the package name /
// app id) are configuration, never constants in the use case.
//
// Each platform selects its own pkg/attestation provider, because the two
// backends are asymmetric and implemented by different drivers: android is
// playintegrity (server-to-server), ios is appattest (local crypto). Both
// default to "mock" in dev.
type AttestationConfig struct {
	// ChallengeTTL is how long a minted nonce can be used (AC1/AC3).
	ChallengeTTL Duration `yaml:"challenge_ttl"`

	// FreshnessWindow is how recent a Play Integrity token's timestampMillis
	// must be to pass (AC5).
	FreshnessWindow Duration `yaml:"freshness_window"`

	Android AttestationAndroidConfig `yaml:"android"`
	IOS     AttestationIOSConfig     `yaml:"ios"`
}

// AttestationAndroidConfig holds Play Integrity settings.
type AttestationAndroidConfig struct {
	// Provider selects the pkg/attestation driver for Android — "mock" in dev,
	// "playintegrity" in a real deployment.
	Provider string `yaml:"provider"`

	// PackageName is the app's real Play package; the decoded token's
	// requestPackageName must match it (AC5). Empty until the app is published.
	PackageName string `yaml:"package_name"`

	// RequiredAppRecognitionVerdict is the threshold the decoded token's
	// appRecognitionVerdict must meet (AC5). Defaults to PLAY_RECOGNIZED; kept
	// in config so raising the bar (MEETS_DEVICE_INTEGRITY, ...) later is a
	// config edit, not a code change (design.md decision #11).
	RequiredAppRecognitionVerdict string `yaml:"required_app_recognition_verdict"`

	// CredentialsSecret names the service-account JSON secret the playintegrity
	// driver uses to authenticate the decode call; resolved into Credentials at
	// load time. Empty in dev (the mock ignores it).
	CredentialsSecret string `yaml:"credentials_secret"`
	Credentials       string `yaml:"-"`
}

// AttestationIOSConfig holds App Attest settings.
type AttestationIOSConfig struct {
	// Provider selects the pkg/attestation driver for iOS — "mock" in dev,
	// "appattest" in a real deployment.
	Provider string `yaml:"provider"`

	// AppID is the {team_id}.{bundle_id} used to compute the RP ID hash —
	// SHA256(app_id) is what the attestation's authenticatorData must carry.
	// Empty until the app is published.
	AppID string `yaml:"app_id"`
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

	if c.Attestation.Android.CredentialsSecret != "" {
		v, err := getSecret(ctx, sg, c.Attestation.Android.CredentialsSecret)
		if err != nil {
			return fmt.Errorf("attestation.android.credentials_secret %q: %w", c.Attestation.Android.CredentialsSecret, err)
		}
		c.Attestation.Android.Credentials = v
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
