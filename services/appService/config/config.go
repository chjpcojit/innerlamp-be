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
	Temple      TempleConfig      `yaml:"temple"`
	Deity       DeityConfig       `yaml:"deity"`
	Gallery     GalleryConfig     `yaml:"gallery"`
	Calendar    CalendarConfig    `yaml:"calendar"`
	Article     ArticleConfig     `yaml:"article"`
	Memorial    MemorialConfig    `yaml:"memorial"`
	Lamp        LampConfig        `yaml:"lamp"`
	Genealogy   GenealogyConfig   `yaml:"genealogy"`
	DharmaTalk  DharmaTalkConfig  `yaml:"dharma_talk"`
	Sutra       SutraConfig       `yaml:"sutra"`

	TempleSuggestion TempleSuggestionConfig `yaml:"temple_suggestion"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Addr string `yaml:"addr"`
	// StaticImagesDir, when set, makes appService serve that folder at
	// /static/images/ so image_url can point here while there is no CDN yet.
	// Empty (the default) disables it. DEV-ONLY: production serves images from a CDN.
	StaticImagesDir string `yaml:"static_images_dir"`
	// PublicBaseURL is the origin root-relative image paths ("/static/images/...") are
	// resolved against in API responses, e.g. "https://cdn.example.com". Empty = the
	// request's own origin, so a dev machine keeps working after its IP changes.
	PublicBaseURL string `yaml:"public_base_url"`
	// WebAPIKeySecret names the secret resolved into WebAPIKey: the key the web
	// frontend's server sends in X-API-Key. Empty = no web key configured, so any
	// presented X-API-Key is rejected.
	WebAPIKeySecret string `yaml:"web_api_key_secret"`
	WebAPIKey       string `yaml:"-"`
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

// TempleConfig holds the temple catalogue's page sizes. Config rather than
// constants: the iOS list screen's page size and the home carousel length are
// tuning knobs a deployment may change without a code release. A zero value
// would return empty pages (and divide by zero in total_pages), so every
// config.<env>.yaml must set all three; Validate enforces it at startup.
type TempleConfig struct {
	// TopLimit is how many temples GET /v1/temples/top returns.
	TopLimit int `yaml:"top_limit"`

	// PageSize is how many temples each page of GET /v1/temples holds.
	PageSize int `yaml:"page_size"`

	// OptionsPageSize is how many options each page of GET /v1/temples/options
	// holds. Separate from PageSize: a pick-list wants bigger pages than the
	// browse list because it carries three small fields per item.
	OptionsPageSize int `yaml:"options_page_size"`
}

// Validate rejects a value that would return empty pages or divide by zero in
// total_pages.
func (c TempleConfig) Validate() error {
	if c.TopLimit < 1 {
		return fmt.Errorf("temple.top_limit must be >= 1, got %d", c.TopLimit)
	}
	if c.PageSize < 1 {
		return fmt.Errorf("temple.page_size must be >= 1, got %d", c.PageSize)
	}
	if c.OptionsPageSize < 1 {
		return fmt.Errorf("temple.options_page_size must be >= 1, got %d", c.OptionsPageSize)
	}
	return nil
}

// DeityConfig holds the Phật / Bồ tát catalogue's page sizes. Same reasoning as
// TempleConfig: tuning knobs, and zero would return empty pages, so every
// config.<env>.yaml must set both.
type DeityConfig struct {
	// TopLimit is how many deities GET /v1/deities/top returns.
	TopLimit int `yaml:"top_limit"`

	// PageSize is how many deities (Buddhas and bodhisattvas) each page of GET /v1/deities holds.
	PageSize int `yaml:"page_size"`
}

// GalleryConfig holds the Buddhist photo gallery's sizes. Same reasoning as
// TempleConfig: tuning knobs, and zero would return empty pages, so every
// config.<env>.yaml must set both.
type GalleryConfig struct {
	// TopLimit is how many images GET /v1/gallery/top returns.
	TopLimit int `yaml:"top_limit"`

	// PageSize is how many images each page of GET /v1/gallery holds.
	PageSize int `yaml:"page_size"`
}

// CalendarConfig holds the lunar-calendar image endpoint's settings.
type CalendarConfig struct {
	// Timezone is the IANA zone whose "today" GET /v1/calendar/image uses when the
	// request carries no date. Required: an empty or unknown zone stops the service at startup.
	Timezone string `yaml:"timezone"`

	// ImageBaseURL is the absolute URL the image file names are appended to, e.g.
	// "https://cdn.example.com/home-calendar". Empty = derive "<scheme>://<host>/static/images/home-calendar"
	// from the request. DEV-ONLY default: production points this at a CDN.
	ImageBaseURL string `yaml:"image_base_url"`
}

// ArticleConfig holds the Phật pháp article catalogue's sizes. Same reasoning as
// TempleConfig: tuning knobs, and zero would return empty pages, so every
// config.<env>.yaml must set both.
type ArticleConfig struct {
	// TopLimit is how many articles GET /v1/articles/top returns.
	TopLimit int `yaml:"top_limit"`

	// PageSize is how many articles each page of GET /v1/articles holds.
	PageSize int `yaml:"page_size"`
}

// MemorialConfig holds the sổ giỗ chạp limits. Zero would mean a user can create
// nothing, so every config.<env>.yaml must set it.
type MemorialConfig struct {
	// MaxPerUser is how many memorials one user may own.
	MaxPerUser int `yaml:"max_per_user"`
}

// LampConfig holds the thắp đèn / vườn đèn limits (docs/specs/features/lamp).
// They are operational numbers a dev environment shortens to exercise a limit
// and prod may stretch, so they live here. Zero would mean "nobody can light a
// lamp" or "an empty garden page", so Validate rejects it and every
// config.<env>.yaml must set the block.
type LampConfig struct {
	// MaxActivePerUser is how many lit lamps one user may have at once (dev: 20).
	MaxActivePerUser int `yaml:"max_active_per_user"`

	// PageSize is how many lamps one garden page holds, and the most a client
	// may ask for with limit (dev: 12).
	PageSize int `yaml:"page_size"`

	// DedupeWindow is how long an identical lamp from the same user counts as
	// the same submission, so a double tap lights one lamp (dev: 5s).
	DedupeWindow Duration `yaml:"dedupe_window"`

	// DurationsDays are the burn durations a user may choose, in days
	// (dev: 1, 7, 21, 49).
	DurationsDays []int `yaml:"durations_days"`
}

// Validate rejects a LampConfig that would make lighting or paging impossible:
// a limit below 1, a non-positive window, or a duration list that is empty,
// holds a value below 1 or repeats one.
func (c LampConfig) Validate() error {
	if c.MaxActivePerUser < 1 {
		return fmt.Errorf("lamp.max_active_per_user must be >= 1, got %d", c.MaxActivePerUser)
	}
	if c.PageSize < 1 {
		return fmt.Errorf("lamp.page_size must be >= 1, got %d", c.PageSize)
	}
	if c.DedupeWindow.Duration() <= 0 {
		return fmt.Errorf("lamp.dedupe_window must be > 0, got %s", c.DedupeWindow.Duration())
	}
	if len(c.DurationsDays) == 0 {
		return fmt.Errorf("lamp.durations_days must not be empty")
	}
	seen := make(map[int]bool, len(c.DurationsDays))
	for _, d := range c.DurationsDays {
		if d < 1 {
			return fmt.Errorf("lamp.durations_days must hold values >= 1, got %d", d)
		}
		if seen[d] {
			return fmt.Errorf("lamp.durations_days repeats %d", d)
		}
		seen[d] = true
	}
	return nil
}

// GenealogyConfig holds the gia phả size limits and the invitation lifetime
// (docs/specs/features/genealogy AC10, AC24, AC25, AC20, AC47, AC75). They are
// operational numbers a dev environment shortens to exercise a limit and prod
// may stretch, so they live here rather than as constants. Zero would mean
// "nobody can create anything" or "invitations are born expired", so Validate
// rejects any value below 1 and every config.<env>.yaml must set the block.
type GenealogyConfig struct {
	// MaxPerUser is how many genealogies one user may own (dev: 10).
	MaxPerUser int `yaml:"max_per_user"`

	// MaxMembers is how many members, in-laws included, one genealogy may hold
	// (dev: 5000).
	MaxMembers int `yaml:"max_members"`

	// MaxGeneration is the deepest generation a member may be placed in; the
	// founder is generation 1 (dev: 60).
	MaxGeneration int `yaml:"max_generation"`

	// MaxSpouses is how many spouses one blood member may have (dev: 5).
	MaxSpouses int `yaml:"max_spouses"`

	// MaxViewers caps viewers plus pending invitations per genealogy (dev: 20).
	MaxViewers int `yaml:"max_viewers"`

	// InvitationTTL is how long an invitation stays acceptable (dev: "720h",
	// 30 days).
	InvitationTTL Duration `yaml:"invitation_ttl"`
}

// Validate rejects any limit below 1 and a non-positive invitation lifetime.
func (c GenealogyConfig) Validate() error {
	checks := []struct {
		name string
		v    int
	}{
		{"max_per_user", c.MaxPerUser},
		{"max_members", c.MaxMembers},
		{"max_generation", c.MaxGeneration},
		{"max_spouses", c.MaxSpouses},
		{"max_viewers", c.MaxViewers},
	}
	for _, ch := range checks {
		if ch.v < 1 {
			return fmt.Errorf("genealogy.%s must be >= 1, got %d", ch.name, ch.v)
		}
	}
	if c.InvitationTTL.Duration() <= 0 {
		return fmt.Errorf("genealogy.invitation_ttl must be > 0, got %s", c.InvitationTTL.Duration())
	}
	return nil
}

// TempleSuggestionConfig holds the public "suggest a temple" route's abuse guard.
// The route is anonymous and writes to the database, so Validate rejects a
// missing limit: zero would either block everyone or (with a lax limiter) allow
// unbounded writes, and neither is something a config typo should decide.
type TempleSuggestionConfig struct {
	// RateLimit caps how many suggestions one client IP may submit per window.
	RateLimit TempleSuggestionRateLimit `yaml:"rate_limit"`
}

// TempleSuggestionRateLimit is the per-IP submission limit.
type TempleSuggestionRateLimit struct {
	// Max is the number of suggestions allowed per Window (dev: 5).
	Max int `yaml:"max"`

	// Window is the length of the counting window (dev: "1h").
	Window Duration `yaml:"window"`

	// TrustForwardedFor reads the client IP from the last X-Forwarded-For entry
	// instead of the TCP peer. Set it only behind a proxy that appends the real
	// client address (Cloud Run does); anywhere else the header is attacker-
	// controlled. Off, every caller behind a proxy would share one bucket.
	TrustForwardedFor bool `yaml:"trust_forwarded_for"`
}

// Validate rejects a limit that is not a positive count over a positive window.
func (c TempleSuggestionConfig) Validate() error {
	if c.RateLimit.Max < 1 {
		return fmt.Errorf("temple_suggestion.rate_limit.max must be >= 1, got %d", c.RateLimit.Max)
	}
	if c.RateLimit.Window.Duration() <= 0 {
		return fmt.Errorf("temple_suggestion.rate_limit.window must be > 0, got %s", c.RateLimit.Window.Duration())
	}
	return nil
}

// DharmaTalkConfig holds the pháp thoại catalogue's page size. Same reasoning as
// ArticleConfig: a tuning knob, and zero would return empty pages (and divide by
// zero in the page count), so every config.<env>.yaml must set it.
type DharmaTalkConfig struct {
	// PageSize is how many talks each page of GET /v1/dharma-talks holds.
	PageSize int `yaml:"page_size"`
}

// Validate rejects a page size that would divide by zero in total_pages.
func (c DharmaTalkConfig) Validate() error {
	if c.PageSize < 1 {
		return fmt.Errorf("dharma_talk.page_size must be >= 1, got %d", c.PageSize)
	}
	return nil
}

// SutraConfig holds the kinh / kệ / chú catalogue's page sizes. Same reasoning as
// TempleConfig (tuning knobs a deployment may change without a code release), but
// unlike the other catalogues a zero here is not a harmless empty page: the lines
// endpoint divides by the page size, so Validate rejects anything below 1 and
// cmd/main.go calls it right after Load.
type SutraConfig struct {
	// PageSize is how many sutras each page of GET /v1/sutras and
	// GET /v1/deities/{id}/sutras holds.
	PageSize int `yaml:"page_size"`

	// LinesPageSize is the default number of lines per page of
	// GET /v1/sutras/{id}/lines when the caller sends no page_size.
	LinesPageSize int `yaml:"lines_page_size"`

	// LinesMaxPageSize caps a caller-supplied page_size; a larger value is served
	// at this size instead of failing.
	LinesMaxPageSize int `yaml:"lines_max_page_size"`
}

// Validate rejects a SutraConfig that would break paging: every size must be at
// least 1 and the default lines page cannot exceed the cap.
func (c SutraConfig) Validate() error {
	switch {
	case c.PageSize < 1:
		return fmt.Errorf("sutra.page_size must be >= 1, got %d", c.PageSize)
	case c.LinesPageSize < 1:
		return fmt.Errorf("sutra.lines_page_size must be >= 1, got %d", c.LinesPageSize)
	case c.LinesMaxPageSize < 1:
		return fmt.Errorf("sutra.lines_max_page_size must be >= 1, got %d", c.LinesMaxPageSize)
	case c.LinesPageSize > c.LinesMaxPageSize:
		return fmt.Errorf("sutra.lines_page_size (%d) must not exceed sutra.lines_max_page_size (%d)", c.LinesPageSize, c.LinesMaxPageSize)
	}
	return nil
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

	if c.Server.WebAPIKeySecret != "" {
		v, err := getSecret(ctx, sg, c.Server.WebAPIKeySecret)
		if err != nil {
			return fmt.Errorf("server.web_api_key_secret %q: %w", c.Server.WebAPIKeySecret, err)
		}
		c.Server.WebAPIKey = v
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
