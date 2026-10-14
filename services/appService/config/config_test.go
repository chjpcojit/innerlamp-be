package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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

func TestLoad_ResolvesWebAPIKeySecret(t *testing.T) {
	source := writeYAML(t, strings.Replace(validYAML, "server:\n", "server:\n  web_api_key_secret: \"innerlamp-web-key\"\n", 1))
	sg := &fakeSecretGetter{values: map[string]string{
		"innerlamp-db-password": "x",
		"innerlamp-jwt-secret":  "y",
		"innerlamp-web-key":     "web-secret",
	}}

	cfg, err := Load(context.Background(), source, sg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Server.WebAPIKey != "web-secret" {
		t.Errorf("Server.WebAPIKey = %q, want resolved secret value", cfg.Server.WebAPIKey)
	}
}

func TestTempleSuggestionConfig_Validate(t *testing.T) {
	ok := TempleSuggestionConfig{RateLimit: TempleSuggestionRateLimit{Max: 5, Window: Duration(time.Hour)}}
	if err := ok.Validate(); err != nil {
		t.Errorf("max 5 / 1h should be valid: %v", err)
	}
	for name, c := range map[string]TempleSuggestionConfig{
		"zero value":  {},
		"zero max":    {RateLimit: TempleSuggestionRateLimit{Max: 0, Window: Duration(time.Hour)}},
		"zero window": {RateLimit: TempleSuggestionRateLimit{Max: 5}},
	} {
		if err := c.Validate(); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
}

func TestLoad_DevYAMLTempleSuggestionConfig(t *testing.T) {
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
	want := TempleSuggestionRateLimit{Max: 5, Window: Duration(time.Hour)}
	if cfg.TempleSuggestion.RateLimit != want {
		t.Errorf("RateLimit = %+v, want %+v", cfg.TempleSuggestion.RateLimit, want)
	}
	if err := cfg.TempleSuggestion.Validate(); err != nil {
		t.Errorf("dev config must be valid: %v", err)
	}
}

func TestAdministrativeUnitsConfig_Validate(t *testing.T) {
	ok := AdministrativeUnitsConfig{
		RateLimit:   AdministrativeUnitsRateLimit{Max: 60, Window: Duration(time.Minute)},
		CacheMaxAge: Duration(24 * time.Hour),
	}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	for name, c := range map[string]AdministrativeUnitsConfig{
		"zero value":     {},
		"zero max":       {RateLimit: AdministrativeUnitsRateLimit{Window: Duration(time.Minute)}, CacheMaxAge: Duration(time.Hour)},
		"negative max":   {RateLimit: AdministrativeUnitsRateLimit{Max: -1, Window: Duration(time.Minute)}, CacheMaxAge: Duration(time.Hour)},
		"zero window":    {RateLimit: AdministrativeUnitsRateLimit{Max: 60}, CacheMaxAge: Duration(time.Hour)},
		"zero cache age": {RateLimit: AdministrativeUnitsRateLimit{Max: 60, Window: Duration(time.Minute)}},
		"sub-second age": {RateLimit: AdministrativeUnitsRateLimit{Max: 60, Window: Duration(time.Minute)}, CacheMaxAge: Duration(500 * time.Millisecond)},
		"negative cache": {RateLimit: AdministrativeUnitsRateLimit{Max: 60, Window: Duration(time.Minute)}, CacheMaxAge: Duration(-time.Hour)},
	} {
		if err := c.Validate(); err == nil {
			t.Errorf("%s: want error, got nil", name)
		}
	}
}

func TestLoad_DevYAMLAdministrativeUnitsConfig(t *testing.T) {
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
	want := AdministrativeUnitsConfig{
		RateLimit:   AdministrativeUnitsRateLimit{Max: 60, Window: Duration(time.Minute)},
		CacheMaxAge: Duration(24 * time.Hour),
	}
	if cfg.AdministrativeUnits != want {
		t.Errorf("AdministrativeUnits = %+v, want %+v", cfg.AdministrativeUnits, want)
	}
	if err := cfg.AdministrativeUnits.Validate(); err != nil {
		t.Errorf("dev config must be valid: %v", err)
	}
}

func TestTempleConfig_Validate(t *testing.T) {
	ok := TempleConfig{TopLimit: 10, PageSize: 20, OptionsPageSize: 100}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	for name, c := range map[string]TempleConfig{
		"zero top_limit":         {TopLimit: 0, PageSize: 20, OptionsPageSize: 100},
		"zero page_size":         {TopLimit: 10, PageSize: 0, OptionsPageSize: 100},
		"zero options_page_size": {TopLimit: 10, PageSize: 20, OptionsPageSize: 0},
		"negative options":       {TopLimit: 10, PageSize: 20, OptionsPageSize: -1},
	} {
		if err := c.Validate(); err == nil {
			t.Errorf("%s: want error, got nil", name)
		}
	}
}

func TestGenealogyConfig_Validate(t *testing.T) {
	valid := GenealogyConfig{
		MaxPerUser: 10, MaxMembers: 5000, MaxGeneration: 60, MaxSpouses: 5,
		MaxViewers: 20, InvitationTTL: Duration(30 * 24 * time.Hour),
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	cases := map[string]func(c *GenealogyConfig){
		"max_per_user":   func(c *GenealogyConfig) { c.MaxPerUser = 0 },
		"max_members":    func(c *GenealogyConfig) { c.MaxMembers = 0 },
		"max_generation": func(c *GenealogyConfig) { c.MaxGeneration = 0 },
		"max_spouses":    func(c *GenealogyConfig) { c.MaxSpouses = 0 },
		"max_viewers":    func(c *GenealogyConfig) { c.MaxViewers = -1 },
		"invitation_ttl": func(c *GenealogyConfig) { c.InvitationTTL = 0 },
	}
	for field, mutate := range cases {
		c := valid
		mutate(&c)
		err := c.Validate()
		if err == nil {
			t.Errorf("%s: expected an error", field)
			continue
		}
		if !strings.Contains(err.Error(), field) {
			t.Errorf("%s: error %q does not name the field", field, err)
		}
	}
}

func TestGenealogyConfig_ZeroValueIsInvalid(t *testing.T) {
	if err := (GenealogyConfig{}).Validate(); err == nil {
		t.Error("zero GenealogyConfig must be invalid: a missing yaml block would otherwise mean limit 0")
	}
}

func TestLoad_DevYAMLGenealogyConfig(t *testing.T) {
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
	want := GenealogyConfig{
		MaxPerUser: 10, MaxMembers: 5000, MaxGeneration: 60, MaxSpouses: 5,
		MaxViewers: 20, InvitationTTL: Duration(30 * 24 * time.Hour),
	}
	if cfg.Genealogy != want {
		t.Errorf("Genealogy = %+v, want %+v", cfg.Genealogy, want)
	}
	if err := cfg.Genealogy.Validate(); err != nil {
		t.Errorf("dev config must validate: %v", err)
	}
}

func TestLampConfig_Validate(t *testing.T) {
	valid := LampConfig{
		MaxActivePerUser: 20, PageSize: 12, DedupeWindow: Duration(5 * time.Second), DurationsDays: []int{1, 7, 21, 49},
		RecipientMaxChars: 100, PrayerMaxChars: 300,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	cases := map[string]func(c *LampConfig){
		"max_active_per_user": func(c *LampConfig) { c.MaxActivePerUser = 0 },
		"page_size":           func(c *LampConfig) { c.PageSize = 0 },
		"dedupe_window":       func(c *LampConfig) { c.DedupeWindow = 0 },
		"durations_days":      func(c *LampConfig) { c.DurationsDays = nil },
		"recipient_max_chars": func(c *LampConfig) { c.RecipientMaxChars = 0 },
		"prayer_max_chars":    func(c *LampConfig) { c.PrayerMaxChars = 0 },
	}
	for field, mutate := range cases {
		c := valid
		mutate(&c)
		err := c.Validate()
		if err == nil {
			t.Errorf("%s: expected an error", field)
			continue
		}
		if !strings.Contains(err.Error(), field) {
			t.Errorf("%s: error %q does not name the field", field, err)
		}
	}
	for name, days := range map[string][]int{"zero value": {0, 7}, "negative": {-1}, "repeated": {7, 7}} {
		c := valid
		c.DurationsDays = days
		if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "durations_days") {
			t.Errorf("durations_days %s: want an error naming the field, got %v", name, err)
		}
	}
}

// The length limits may be lowered but never raised past the VARCHAR columns:
// a longer value would pass validation and then fail the INSERT with a 500.
func TestLampConfig_LengthLimitsAreCappedByTheColumns(t *testing.T) {
	base := LampConfig{
		MaxActivePerUser: 20, PageSize: 12, DedupeWindow: Duration(5 * time.Second), DurationsDays: []int{1},
		RecipientMaxChars: LampRecipientColumnChars, PrayerMaxChars: LampPrayerColumnChars,
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("limits equal to the columns must be accepted: %v", err)
	}
	if LampRecipientColumnChars != 100 || LampPrayerColumnChars != 300 {
		t.Fatalf("column constants drifted from 0027_lamps.sql: %d, %d", LampRecipientColumnChars, LampPrayerColumnChars)
	}

	over := base
	over.RecipientMaxChars = LampRecipientColumnChars + 1
	if err := over.Validate(); err == nil || !strings.Contains(err.Error(), "recipient_max_chars") {
		t.Errorf("recipient_max_chars above the column: want an error naming it, got %v", err)
	}
	over = base
	over.PrayerMaxChars = LampPrayerColumnChars + 1
	if err := over.Validate(); err == nil || !strings.Contains(err.Error(), "prayer_max_chars") {
		t.Errorf("prayer_max_chars above the column: want an error naming it, got %v", err)
	}

	lowered := base
	lowered.RecipientMaxChars, lowered.PrayerMaxChars = 1, 1
	if err := lowered.Validate(); err != nil {
		t.Errorf("lowering both limits must be accepted: %v", err)
	}
}

func TestLampConfig_ZeroValueIsInvalid(t *testing.T) {
	if err := (LampConfig{}).Validate(); err == nil {
		t.Error("zero LampConfig must be invalid: a missing yaml block would otherwise mean limit 0")
	}
}

func TestLoad_DevYAMLLampConfig(t *testing.T) {
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
	want := LampConfig{
		MaxActivePerUser: 20, PageSize: 12, DedupeWindow: Duration(5 * time.Second), DurationsDays: []int{1, 7, 21, 49},
		RecipientMaxChars: 100, PrayerMaxChars: 300,
	}
	if !reflect.DeepEqual(cfg.Lamp, want) {
		t.Errorf("Lamp = %+v, want %+v", cfg.Lamp, want)
	}
	if err := cfg.Lamp.Validate(); err != nil {
		t.Errorf("dev config must validate: %v", err)
	}
}

// validProfileConfig is the dev block of config.dev.yaml, spelled out so each
// case below breaks exactly one field.
func validProfileConfig() ProfileConfig {
	return ProfileConfig{
		DharmaNameMaxChars: 255,
		FullNameMaxChars:   300,
		AvatarMaxBytes:     5 << 20,
		AvatarContentTypes: []string{"image/jpeg", "image/png", "image/webp"},
		UploadURLTTL:       Duration(5 * time.Minute),
		AvatarReadURLTTL:   Duration(15 * time.Minute),
		Bucket:             "innerlamp-avatars-dev",
		AvatarTmpPrefix:    "avatars/tmp/",
		AvatarPrefix:       "avatars/",
		RateLimit:          ProfileRateLimit{Max: 10, Window: Duration(time.Minute)},
	}
}

func TestProfileConfig_Validate(t *testing.T) {
	valid := validProfileConfig()
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	cases := map[string]func(c *ProfileConfig){
		"dharma_name_max_chars zero":      func(c *ProfileConfig) { c.DharmaNameMaxChars = 0 },
		"dharma_name_max_chars negative":  func(c *ProfileConfig) { c.DharmaNameMaxChars = -1 },
		"full_name_max_chars zero":        func(c *ProfileConfig) { c.FullNameMaxChars = 0 },
		"full_name_max_chars over column": func(c *ProfileConfig) { c.FullNameMaxChars = ProfileFullNameColumnChars + 1 },
		"avatar_max_bytes zero":           func(c *ProfileConfig) { c.AvatarMaxBytes = 0 },
		"avatar_content_types empty":      func(c *ProfileConfig) { c.AvatarContentTypes = nil },
		"avatar_content_types blank":      func(c *ProfileConfig) { c.AvatarContentTypes = []string{"image/png", " "} },
		"avatar_content_types duplicate":  func(c *ProfileConfig) { c.AvatarContentTypes = []string{"image/png", "image/png"} },
		"upload_url_ttl zero":             func(c *ProfileConfig) { c.UploadURLTTL = 0 },
		"upload_url_ttl negative":         func(c *ProfileConfig) { c.UploadURLTTL = Duration(-time.Second) },
		"avatar_read_url_ttl zero":        func(c *ProfileConfig) { c.AvatarReadURLTTL = 0 },
		"bucket empty":                    func(c *ProfileConfig) { c.Bucket = "" },
		"avatar_tmp_prefix empty":         func(c *ProfileConfig) { c.AvatarTmpPrefix = "" },
		"avatar_prefix empty":             func(c *ProfileConfig) { c.AvatarPrefix = "" },
		"prefixes equal":                  func(c *ProfileConfig) { c.AvatarPrefix = c.AvatarTmpPrefix },
		"rate_limit.max zero":             func(c *ProfileConfig) { c.RateLimit.Max = 0 },
		"rate_limit.window zero":          func(c *ProfileConfig) { c.RateLimit.Window = 0 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := validProfileConfig()
			mutate(&c)
			if err := c.Validate(); err == nil {
				t.Errorf("expected Validate to reject %s", name)
			}
		})
	}
}

// The limit can be lowered freely but never raised past users.dharma_name
// VARCHAR(255): a higher limit would pass validation and then fail the write
// with a 500.
func TestProfileConfig_DharmaNameLimitIsCappedByTheColumn(t *testing.T) {
	if ProfileDharmaNameColumnChars != 255 {
		t.Fatalf("column constant drifted from users.dharma_name: %d", ProfileDharmaNameColumnChars)
	}
	atCap := validProfileConfig()
	atCap.DharmaNameMaxChars = ProfileDharmaNameColumnChars
	if err := atCap.Validate(); err != nil {
		t.Errorf("the column size itself must be allowed: %v", err)
	}
	over := validProfileConfig()
	over.DharmaNameMaxChars = ProfileDharmaNameColumnChars + 1
	if err := over.Validate(); err == nil {
		t.Error("a limit above the column must be rejected")
	}
}

func TestProfileConfig_ZeroValueIsInvalid(t *testing.T) {
	if err := (ProfileConfig{}).Validate(); err == nil {
		t.Error("zero ProfileConfig must be invalid: a missing yaml block would otherwise mean limit 0")
	}
}

func TestLoad_DevYAMLProfileConfig(t *testing.T) {
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
	if cfg.Profile.DharmaNameMaxChars != 255 {
		t.Errorf("Profile.DharmaNameMaxChars = %d, want 255", cfg.Profile.DharmaNameMaxChars)
	}
	if err := cfg.Profile.Validate(); err != nil {
		t.Errorf("dev config must validate: %v", err)
	}
	p := cfg.Profile
	if p.AvatarMaxBytes != 5242880 || p.UploadURLTTL.Duration() != 5*time.Minute || p.AvatarReadURLTTL.Duration() != 15*time.Minute {
		t.Errorf("avatar limits not loaded from dev yaml: %+v", p)
	}
	if p.AvatarTmpPrefix != "avatars/tmp/" || p.AvatarPrefix != "avatars/" || p.RateLimit.Max != 10 || p.RateLimit.Window.Duration() != time.Minute {
		t.Errorf("prefixes / rate limit not loaded from dev yaml: %+v", p)
	}
}

func TestStorageConfig_Validate(t *testing.T) {
	valid := StorageConfig{Backend: "gcs", ProjectID: "innerlamp-dev", DefaultBucket: "b", Timeout: Duration(30 * time.Second)}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	withEmulator := valid
	withEmulator.EmulatorHost = "localhost:4443"
	if err := withEmulator.Validate(); err != nil {
		t.Fatalf("host:port emulator rejected: %v", err)
	}

	cases := map[string]func(c *StorageConfig){
		"backend empty":              func(c *StorageConfig) { c.Backend = "" },
		"backend unknown":            func(c *StorageConfig) { c.Backend = "azure" },
		"project_id empty":           func(c *StorageConfig) { c.ProjectID = "" },
		"timeout zero":               func(c *StorageConfig) { c.Timeout = 0 },
		"timeout negative":           func(c *StorageConfig) { c.Timeout = Duration(-time.Second) },
		"emulator_host with scheme":  func(c *StorageConfig) { c.EmulatorHost = "http://localhost:4443" },
		"emulator_host with path":    func(c *StorageConfig) { c.EmulatorHost = "localhost:4443/x" },
		"emulator_host without port": func(c *StorageConfig) { c.EmulatorHost = "localhost" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := valid
			mutate(&c)
			if err := c.Validate(); err == nil {
				t.Errorf("expected Validate to reject %s", name)
			}
		})
	}
}

func TestLoad_DevYAMLStorageConfig(t *testing.T) {
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
	if err := cfg.Storage.Validate(); err != nil {
		t.Errorf("dev config must validate: %v", err)
	}
	if cfg.Storage.Timeout.Duration() != 30*time.Second {
		t.Errorf("Storage.Timeout = %s, want 30s", cfg.Storage.Timeout.Duration())
	}
	if cfg.Storage.EmulatorHost != "localhost:4443" {
		t.Errorf("Storage.EmulatorHost = %q, want localhost:4443 (dev runs against the compose emulator)", cfg.Storage.EmulatorHost)
	}
}

func TestCalendarConfig_Validate(t *testing.T) {
	ok := CalendarConfig{Timezone: "Asia/Ho_Chi_Minh", DefaultLanguage: "vi", EventsCacheMaxAge: Duration(24 * time.Hour)}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	for name, c := range map[string]CalendarConfig{
		"zero value":     {},
		"no language":    {EventsCacheMaxAge: Duration(time.Hour)},
		"long language":  {DefaultLanguage: "vietnamese", EventsCacheMaxAge: Duration(time.Hour)},
		"no cache age":   {DefaultLanguage: "vi"},
		"sub-second age": {DefaultLanguage: "vi", EventsCacheMaxAge: Duration(500 * time.Millisecond)},
		"negative cache": {DefaultLanguage: "vi", EventsCacheMaxAge: Duration(-time.Hour)},
	} {
		if err := c.Validate(); err == nil {
			t.Errorf("%s: want error, got nil", name)
		}
	}
}

func TestLoad_DevYAMLCalendarConfig(t *testing.T) {
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
	if cfg.Calendar.Timezone != "Asia/Ho_Chi_Minh" || cfg.Calendar.DefaultLanguage != "vi" ||
		cfg.Calendar.EventsCacheMaxAge.Duration() != 24*time.Hour {
		t.Errorf("Calendar = %+v", cfg.Calendar)
	}
	if err := cfg.Calendar.Validate(); err != nil {
		t.Errorf("dev config must be valid: %v", err)
	}
}

func TestTempleEventConfig_Validate(t *testing.T) {
	ok := TempleEventConfig{Timezone: "Asia/Ho_Chi_Minh", DefaultWindowMonths: 3, MaxWindowMonths: 3}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	for name, c := range map[string]TempleEventConfig{
		"zero value":       {},
		"no zone":          {DefaultWindowMonths: 3, MaxWindowMonths: 3},
		"unknown zone":     {Timezone: "Mars/Olympus", DefaultWindowMonths: 3, MaxWindowMonths: 3},
		"default zero":     {Timezone: "Asia/Ho_Chi_Minh", MaxWindowMonths: 3},
		"default over max": {Timezone: "Asia/Ho_Chi_Minh", DefaultWindowMonths: 4, MaxWindowMonths: 3},
	} {
		if err := c.Validate(); err == nil {
			t.Errorf("%s: want error, got nil", name)
		}
	}
}

func TestLoad_DevYAMLTempleEventConfig(t *testing.T) {
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
	want := TempleEventConfig{Timezone: "Asia/Ho_Chi_Minh", DefaultWindowMonths: 3, MaxWindowMonths: 3}
	if cfg.TempleEvent != want {
		t.Errorf("TempleEvent = %+v, want %+v", cfg.TempleEvent, want)
	}
}

func validTempleAnnouncementConfig() TempleAnnouncementConfig {
	return TempleAnnouncementConfig{
		PageSize:    20,
		MaxPageSize: 50,
		RateLimit:   TempleAnnouncementRateLimit{Max: 60, Window: Duration(time.Minute)},
	}
}

func TestTempleAnnouncementConfig_Validate(t *testing.T) {
	valid := validTempleAnnouncementConfig()
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	cases := map[string]func(c *TempleAnnouncementConfig){
		"page_size zero":         func(c *TempleAnnouncementConfig) { c.PageSize = 0 },
		"max_page_size < size":   func(c *TempleAnnouncementConfig) { c.MaxPageSize = c.PageSize - 1 },
		"rate_limit.max zero":    func(c *TempleAnnouncementConfig) { c.RateLimit.Max = 0 },
		"rate_limit.window zero": func(c *TempleAnnouncementConfig) { c.RateLimit.Window = 0 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := validTempleAnnouncementConfig()
			mutate(&c)
			if err := c.Validate(); err == nil {
				t.Errorf("expected Validate to reject %s", name)
			}
		})
	}
}

func TestLoad_DevYAMLTempleAnnouncementConfig(t *testing.T) {
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
	if !reflect.DeepEqual(cfg.TempleAnnouncement, validTempleAnnouncementConfig()) {
		t.Errorf("TempleAnnouncement = %+v", cfg.TempleAnnouncement)
	}
	if err := cfg.TempleAnnouncement.Validate(); err != nil {
		t.Errorf("dev config must validate: %v", err)
	}
}

func TestRateLimitStoreConfig_Validate(t *testing.T) {
	redisOK := RedisRateLimitConfig{
		Addr: "localhost:6379", KeyPrefix: "innerlamp:app:rl:",
		DialTimeout: Duration(time.Second), OpTimeout: Duration(100 * time.Millisecond),
	}
	tests := []struct {
		name    string
		cfg     RateLimitStoreConfig
		wantErr bool
	}{
		{name: "redis", cfg: RateLimitStoreConfig{Backend: RateLimitStoreRedis, Redis: redisOK}},
		{name: "memory needs no redis block", cfg: RateLimitStoreConfig{Backend: RateLimitStoreMemory}},
		{name: "zero value", cfg: RateLimitStoreConfig{}, wantErr: true},
		{name: "unknown backend", cfg: RateLimitStoreConfig{Backend: "memcached"}, wantErr: true},
		{name: "redis without addr", cfg: RateLimitStoreConfig{Backend: RateLimitStoreRedis, Redis: func() RedisRateLimitConfig { r := redisOK; r.Addr = ""; return r }()}, wantErr: true},
		{name: "redis without prefix", cfg: RateLimitStoreConfig{Backend: RateLimitStoreRedis, Redis: func() RedisRateLimitConfig { r := redisOK; r.KeyPrefix = " "; return r }()}, wantErr: true},
		{name: "redis negative db", cfg: RateLimitStoreConfig{Backend: RateLimitStoreRedis, Redis: func() RedisRateLimitConfig { r := redisOK; r.DB = -1; return r }()}, wantErr: true},
		{name: "redis zero dial timeout", cfg: RateLimitStoreConfig{Backend: RateLimitStoreRedis, Redis: func() RedisRateLimitConfig { r := redisOK; r.DialTimeout = 0; return r }()}, wantErr: true},
		{name: "redis zero op timeout", cfg: RateLimitStoreConfig{Backend: RateLimitStoreRedis, Redis: func() RedisRateLimitConfig { r := redisOK; r.OpTimeout = 0; return r }()}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestLoad_DevYAMLRateLimitStoreConfig(t *testing.T) {
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
	got := cfg.RateLimitStore
	if got.Backend != RateLimitStoreRedis || got.Redis.Addr != "localhost:6379" || got.Redis.KeyPrefix != "innerlamp:app:rl:" {
		t.Errorf("RateLimitStore = %+v", got)
	}
	if got.Redis.Password == "" {
		t.Error("redis password_secret must resolve from dev_secrets")
	}
	if err := got.Validate(); err != nil {
		t.Errorf("dev config must be valid: %v", err)
	}
}
