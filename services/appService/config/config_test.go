package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pkgconfig "chjpcoj.com.innerlamp/pkg/config"
	"chjpcoj.com.innerlamp/pkg/secrets"
)

// fakeSecretGetter is a test double for SecretGetter — narrower than the
// full secrets.Client, so no need to stub 20+ unused methods.
type fakeSecretGetter struct {
	values map[string]string
	err    error
	calls  []string
}

func (f *fakeSecretGetter) Get(_ context.Context, name string, _ *secrets.GetOptions) (*secrets.SecretValue, error) {
	f.calls = append(f.calls, name)
	if f.err != nil {
		return nil, f.err
	}
	v, ok := f.values[name]
	if !ok {
		return nil, secrets.ErrSecretNotFound
	}
	return &secrets.SecretValue{Name: name, Value: []byte(v)}, nil
}

func writeYAML(t *testing.T, content string) pkgconfig.LoadSource {
	t.Helper()
	path := filepath.Join(t.TempDir(), "appservice.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return pkgconfig.LoadSource(path)
}

const validYAML = `
server:
  addr: ":8080"
database:
  host: "localhost"
  port: 5432
  username: "postgres"
  password_secret: "innerlamp-db-password"
  database: "innerlamp"
  ssl_mode: "disable"
jwt:
  algorithm: "HS256"
  secret_key_secret: "innerlamp-jwt-secret"
  access_token_expiration: "15m"
  refresh_token_expiration: "168h"
  issuer: "innerlamp"
  audience: "innerlamp_users"
storage:
  backend: "gcs"
  project_id: "innerlamp-dev"
  default_bucket: "innerlamp-avatars-dev"
`

func TestLoad_ParsesYAMLAndResolvesSecrets(t *testing.T) {
	source := writeYAML(t, validYAML)
	sg := &fakeSecretGetter{values: map[string]string{
		"innerlamp-db-password": "s3cret-db-pass",
		"innerlamp-jwt-secret":  "s3cret-jwt-key",
	}}

	cfg, err := Load(context.Background(), source, sg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.Addr != ":8080" {
		t.Errorf("Server.Addr = %q, want %q", cfg.Server.Addr, ":8080")
	}
	if cfg.Database.Host != "localhost" || cfg.Database.Port != 5432 {
		t.Errorf("Database host/port = %q/%d, want localhost/5432", cfg.Database.Host, cfg.Database.Port)
	}
	if cfg.Database.Password != "s3cret-db-pass" {
		t.Errorf("Database.Password = %q, want resolved secret value", cfg.Database.Password)
	}
	if cfg.JWT.SecretKey != "s3cret-jwt-key" {
		t.Errorf("JWT.SecretKey = %q, want resolved secret value", cfg.JWT.SecretKey)
	}
	if cfg.JWT.AccessTokenExpiration.Duration() != 15*time.Minute {
		t.Errorf("JWT.AccessTokenExpiration = %s, want 15m", cfg.JWT.AccessTokenExpiration.Duration())
	}
	if cfg.JWT.RefreshTokenExpiration.Duration() != 168*time.Hour {
		t.Errorf("JWT.RefreshTokenExpiration = %s, want 168h", cfg.JWT.RefreshTokenExpiration.Duration())
	}
	if cfg.Storage.DefaultBucket != "innerlamp-avatars-dev" {
		t.Errorf("Storage.DefaultBucket = %q, want innerlamp-avatars-dev", cfg.Storage.DefaultBucket)
	}

	// Secret names must never leak into the resolved struct's plaintext fields
	// under any other name, and the fake must have been asked for exactly
	// the two secrets referenced by the YAML.
	if len(sg.calls) != 2 {
		t.Errorf("secret lookups = %v, want exactly 2 calls", sg.calls)
	}
}

func TestLoad_MissingSecretReturnsError(t *testing.T) {
	source := writeYAML(t, validYAML)
	sg := &fakeSecretGetter{values: map[string]string{
		// db password secret intentionally absent
		"innerlamp-jwt-secret": "s3cret-jwt-key",
	}}

	_, err := Load(context.Background(), source, sg)
	if err == nil {
		t.Fatal("Load should fail when a referenced secret is missing")
	}
	if !errors.Is(err, secrets.ErrSecretNotFound) {
		t.Errorf("Load error = %v, want wrapped secrets.ErrSecretNotFound", err)
	}
}

func TestLoad_SecretProviderErrorIsWrapped(t *testing.T) {
	source := writeYAML(t, validYAML)
	wantErr := errors.New("secret manager unavailable")
	sg := &fakeSecretGetter{err: wantErr}

	_, err := Load(context.Background(), source, sg)
	if err == nil {
		t.Fatal("Load should fail when the secrets provider errors")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("Load error = %v, want wrapped %v", err, wantErr)
	}
}

func TestLoad_NoSecretRefsSkipsLookup(t *testing.T) {
	const yaml = `
database:
  host: "localhost"
jwt:
  algorithm: "HS256"
  access_token_expiration: "15m"
  refresh_token_expiration: "168h"
`
	source := writeYAML(t, yaml)
	sg := &fakeSecretGetter{values: map[string]string{}}

	cfg, err := Load(context.Background(), source, sg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Database.Password != "" || cfg.JWT.SecretKey != "" {
		t.Errorf("expected empty resolved secrets when no *_secret refs given, got db=%q jwt=%q", cfg.Database.Password, cfg.JWT.SecretKey)
	}
	if len(sg.calls) != 0 {
		t.Errorf("secrets provider should not be called when no secret refs present, got calls=%v", sg.calls)
	}
}

func TestLoad_InvalidYAMLSourceReturnsError(t *testing.T) {
	missing := pkgconfig.LoadSource(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	sg := &fakeSecretGetter{}

	_, err := Load(context.Background(), missing, sg)
	if err == nil {
		t.Fatal("Load should fail for a nonexistent YAML source")
	}
}

func TestLoad_InvalidDurationReturnsError(t *testing.T) {
	const yaml = `
jwt:
  access_token_expiration: "not-a-duration"
`
	source := writeYAML(t, yaml)
	sg := &fakeSecretGetter{}

	_, err := Load(context.Background(), source, sg)
	if err == nil {
		t.Fatal("Load should fail when a duration field is not parseable")
	}
}

// TestLoad_DevYAMLLoginPolicy loads the shipped config.dev.yaml and checks the
// login block arrives intact. A missing key would decode to a Go zero value —
// and a zero LockDuration means "locked for no time at all" — so the failure
// this guards against is silent: nothing errors, the brute-force lock just
// stops working.
func TestLoad_DevYAMLLoginPolicy(t *testing.T) {
	ctx := context.Background()
	source := pkgconfig.LoadSource("config.dev.yaml")

	raw, err := pkgconfig.LoadYAML(ctx, source)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}
	sg, err := NewMockSecretGetterFromYAML(raw)
	if err != nil {
		t.Fatalf("NewMockSecretGetterFromYAML: %v", err)
	}

	cfg, err := Load(ctx, source, sg)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Login.MaxFailedAttempts != 5 {
		t.Errorf("MaxFailedAttempts = %d, want 5", cfg.Login.MaxFailedAttempts)
	}
	if got := cfg.Login.LockDuration.Duration(); got != 15*time.Minute {
		t.Errorf("LockDuration = %v, want 15m", got)
	}
	if got := cfg.Login.FailureResetWindow.Duration(); got != 15*time.Minute {
		t.Errorf("FailureResetWindow = %v, want 15m", got)
	}
}

// TestLoad_DevYAMLAttestationConfig loads the shipped config.dev.yaml and checks
// the attestation block arrives intact. A missing key decodes to a Go zero
// value — and a zero challenge_ttl / freshness_window would mint nonces that
// expire the instant they are issued — so the failure this guards against is
// silent: nothing errors, attestation just never verifies.
func TestLoad_DevYAMLAttestationConfig(t *testing.T) {
	ctx := context.Background()
	source := pkgconfig.LoadSource("config.dev.yaml")

	raw, err := pkgconfig.LoadYAML(ctx, source)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}
	sg, err := NewMockSecretGetterFromYAML(raw)
	if err != nil {
		t.Fatalf("NewMockSecretGetterFromYAML: %v", err)
	}

	cfg, err := Load(ctx, source, sg)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Attestation.Android.Provider != "mock" {
		t.Errorf("Android.Provider = %q, want mock", cfg.Attestation.Android.Provider)
	}
	if cfg.Attestation.IOS.Provider != "mock" {
		t.Errorf("IOS.Provider = %q, want mock", cfg.Attestation.IOS.Provider)
	}
	if got := cfg.Attestation.ChallengeTTL.Duration(); got != 5*time.Minute {
		t.Errorf("ChallengeTTL = %v, want 5m", got)
	}
	if got := cfg.Attestation.FreshnessWindow.Duration(); got != 5*time.Minute {
		t.Errorf("FreshnessWindow = %v, want 5m", got)
	}
	if cfg.Attestation.Android.PackageName != "com.innerlamp.app" {
		t.Errorf("Android.PackageName = %q, want com.innerlamp.app", cfg.Attestation.Android.PackageName)
	}
	if cfg.Attestation.Android.RequiredAppRecognitionVerdict != "PLAY_RECOGNIZED" {
		t.Errorf("RequiredAppRecognitionVerdict = %q, want PLAY_RECOGNIZED", cfg.Attestation.Android.RequiredAppRecognitionVerdict)
	}
	if cfg.Attestation.IOS.AppID == "" {
		t.Errorf("IOS.AppID is empty; want the {team_id}.{bundle_id} placeholder")
	}
}

func TestLoad_DevYAMLOTPPolicy(t *testing.T) {
	ctx := context.Background()
	source := pkgconfig.LoadSource("config.dev.yaml")

	raw, err := pkgconfig.LoadYAML(ctx, source)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}
	sg, err := NewMockSecretGetterFromYAML(raw)
	if err != nil {
		t.Fatalf("NewMockSecretGetterFromYAML: %v", err)
	}

	cfg, err := Load(ctx, source, sg)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got := cfg.OTP.TTL.Duration(); got != 90*time.Second {
		t.Errorf("TTL = %v, want 90s", got)
	}
	if got := cfg.OTP.SendCooldown.Duration(); got != 60*time.Second {
		t.Errorf("SendCooldown = %v, want 60s", got)
	}
	if cfg.OTP.MaxSendAttempts != 3 {
		t.Errorf("MaxSendAttempts = %d, want 3", cfg.OTP.MaxSendAttempts)
	}
	if got := cfg.OTP.SendLockDuration.Duration(); got != 30*time.Minute {
		t.Errorf("SendLockDuration = %v, want 30m", got)
	}
	if cfg.OTP.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", cfg.OTP.MaxAttempts)
	}
	if got := cfg.OTP.LockDuration.Duration(); got != 30*time.Minute {
		t.Errorf("LockDuration = %v, want 30m", got)
	}
	if got := cfg.OTP.PendingRegistrationTTL.Duration(); got != 24*time.Hour {
		t.Errorf("PendingRegistrationTTL = %v, want 24h", got)
	}
}

func TestSutraConfig_Validate(t *testing.T) {
	ok := SutraConfig{PageSize: 20, LinesPageSize: 200, LinesMaxPageSize: 500}
	tests := []struct {
		name    string
		mutate  func(c *SutraConfig)
		wantErr string
	}{
		{"valid", func(c *SutraConfig) {}, ""},
		{"default equal to cap is fine", func(c *SutraConfig) { c.LinesPageSize = 500 }, ""},
		{"zero page size", func(c *SutraConfig) { c.PageSize = 0 }, "sutra.page_size"},
		{"negative page size", func(c *SutraConfig) { c.PageSize = -1 }, "sutra.page_size"},
		{"zero lines page size", func(c *SutraConfig) { c.LinesPageSize = 0 }, "sutra.lines_page_size"},
		{"zero lines max", func(c *SutraConfig) { c.LinesMaxPageSize = 0 }, "sutra.lines_max_page_size"},
		{"default above cap", func(c *SutraConfig) { c.LinesPageSize = 501 }, "must not exceed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ok
			tt.mutate(&c)
			err := c.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

// A missing sutra block decodes to the zero value, which Validate must reject —
// that is what makes cmd/main.go fail fast on a config.<env>.yaml without it.
func TestSutraConfig_ZeroValueIsInvalid(t *testing.T) {
	if err := (SutraConfig{}).Validate(); err == nil {
		t.Fatal("zero SutraConfig must be invalid")
	}
}

func TestLoad_DevYAMLSutraConfig(t *testing.T) {
	ctx := context.Background()
	source := pkgconfig.LoadSource("config.dev.yaml")
	raw, err := pkgconfig.LoadYAML(ctx, source)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}
	sg, err := NewMockSecretGetterFromYAML(raw)
	if err != nil {
		t.Fatalf("NewMockSecretGetterFromYAML: %v", err)
	}
	cfg, err := Load(ctx, source, sg)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := SutraConfig{PageSize: 20, LinesPageSize: 200, LinesMaxPageSize: 500}
	if cfg.Sutra != want {
		t.Errorf("Sutra = %+v, want %+v", cfg.Sutra, want)
	}
	if err := cfg.Sutra.Validate(); err != nil {
		t.Errorf("shipped config.dev.yaml must be valid: %v", err)
	}
}

func TestDharmaTalkConfig_Validate(t *testing.T) {
	if err := (DharmaTalkConfig{PageSize: 10}).Validate(); err != nil {
		t.Errorf("PageSize 10 should be valid: %v", err)
	}
	for _, n := range []int{0, -1} {
		if err := (DharmaTalkConfig{PageSize: n}).Validate(); err == nil {
			t.Errorf("PageSize %d should be rejected", n)
		}
	}
}

func TestLoad_DevYAMLDharmaTalkConfig(t *testing.T) {
	ctx := context.Background()
	source := pkgconfig.LoadSource("config.dev.yaml")
	raw, err := pkgconfig.LoadYAML(ctx, source)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}
	sg, err := NewMockSecretGetterFromYAML(raw)
	if err != nil {
		t.Fatalf("NewMockSecretGetterFromYAML: %v", err)
	}
	cfg, err := Load(ctx, source, sg)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if want := (DharmaTalkConfig{PageSize: 10}); cfg.DharmaTalk != want {
		t.Errorf("DharmaTalk = %+v, want %+v", cfg.DharmaTalk, want)
	}
}
