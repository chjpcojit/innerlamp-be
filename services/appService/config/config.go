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
	"net"
	"strings"
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
	App      AppEnvConfig   `yaml:"app"`
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	Storage  StorageConfig  `yaml:"storage"`
	OTP      OTPConfig      `yaml:"otp"`
	Mailer   MailerConfig   `yaml:"mailer"`
	Login    LoginConfig    `yaml:"login"`

	// PasswordReset is the forgot-password flow's timing policy (login US-003).
	PasswordReset PasswordResetConfig `yaml:"password_reset"`

	// AuthRateLimit is the per-IP budget of every /v1/auth/* route (login AC31).
	AuthRateLimit AuthRateLimitConfig `yaml:"auth_rate_limit"`

	Attestation AttestationConfig `yaml:"attestation"`
	Temple      TempleConfig      `yaml:"temple"`
	Deity       DeityConfig       `yaml:"deity"`
	Gallery     GalleryConfig     `yaml:"gallery"`
	Calendar    CalendarConfig    `yaml:"calendar"`
	TempleEvent TempleEventConfig `yaml:"temple_event"`
	Article     ArticleConfig     `yaml:"article"`
	Memorial    MemorialConfig    `yaml:"memorial"`
	Lamp        LampConfig        `yaml:"lamp"`
	Profile     ProfileConfig     `yaml:"profile"`
	Genealogy   GenealogyConfig   `yaml:"genealogy"`
	DharmaTalk  DharmaTalkConfig  `yaml:"dharma_talk"`
	Sutra       SutraConfig       `yaml:"sutra"`

	TempleSuggestion TempleSuggestionConfig `yaml:"temple_suggestion"`

	// TempleAnnouncement is the announcement read side + receive switch (docs/specs/features/temple-announcement).
	TempleAnnouncement TempleAnnouncementConfig `yaml:"temple_announcement"`

	AdministrativeUnits AdministrativeUnitsConfig `yaml:"administrative_units"`

	// RateLimitStore is where every rate limiter keeps its counters (Redis, shared by all
	// instances, or memory, one instance only). docs/knowledge/packages/ratelimit.md.
	RateLimitStore RateLimitStoreConfig `yaml:"rate_limit_store"`
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
	// Backend selects the pkg/storage driver; only "gcs" is wired by cmd/main.go.
	Backend       string `yaml:"backend"`
	ProjectID     string `yaml:"project_id"`
	DefaultBucket string `yaml:"default_bucket"`
	// Timeout bounds every storage call (pkg/storage requires it > 0).
	Timeout Duration `yaml:"timeout"`
	// EmulatorHost, when set ("host:port", plain HTTP), points the service at a
	// storage emulator and makes avatar URLs unsigned instead of signed for a
	// real bucket. Leave it EMPTY outside dev and test: it switches signing off.
	EmulatorHost string `yaml:"emulator_host"`
}

// Validate rejects a storage block the client could not be built from, so a
// missing key stops the service at startup instead of at the first avatar.
func (c StorageConfig) Validate() error {
	if c.Backend != "gcs" {
		return fmt.Errorf("storage.backend must be %q (the only wired backend), got %q", "gcs", c.Backend)
	}
	if strings.TrimSpace(c.ProjectID) == "" {
		return fmt.Errorf("storage.project_id must not be empty")
	}
	if c.Timeout.Duration() <= 0 {
		return fmt.Errorf("storage.timeout must be > 0, got %s", c.Timeout.Duration())
	}
	if h := c.EmulatorHost; h != "" {
		if strings.Contains(h, "/") {
			return fmt.Errorf("storage.emulator_host must be host:port without scheme or path, got %q", h)
		}
		if _, _, err := net.SplitHostPort(h); err != nil {
			return fmt.Errorf("storage.emulator_host must be host:port, got %q: %w", h, err)
		}
	}
	return nil
}

// EnvironmentDev is the only app.environment in which dev shortcuts
// (otp.dev_fixed_code, mailer.provider "log") are allowed (onboarding AC21).
const EnvironmentDev = "dev"

// MailerProviderLog is the dev-only pkg/mailer.LogMailer: it writes the whole
// email, code included, to the service log.
const MailerProviderLog = "log"

// AppEnvConfig names the deployment environment ("dev", "staging", "prod").
// It gates the dev shortcuts in AppConfig.Validate.
type AppEnvConfig struct {
	Environment string `yaml:"environment"`
}

// MailerConfig selects the pkg/mailer driver the composition root wires up
// and the sender address every auth email goes out from. "log" is the only
// driver today; a real one (Resend/SES) ships when the sending domain exists
// (onboarding design.md "Mailer").
type MailerConfig struct {
	Provider string `yaml:"provider"`
	From     string `yaml:"from"`

	// DailyBudget is how many auth emails may be sent per UTC day across all
	// instances (login AC33). Past it every route that would send answers
	// 429 RATE_LIMITED until the next UTC day. Sized to the provider's free
	// tier (ADR-0003 rule 7: 100/day).
	DailyBudget int `yaml:"daily_budget"`
}

// Validate rejects a missing budget: zero would block every email, which is
// not "unlimited" and would only show up at the first registration.
func (c MailerConfig) Validate() error {
	if c.DailyBudget < 1 {
		return fmt.Errorf("mailer.daily_budget must be >= 1, got %d", c.DailyBudget)
	}
	return nil
}

// Validate enforces the cross-section startup guard of onboarding AC21 /
// otp-mechanism AC3: outside app.environment "dev", a fixed OTP code or the
// log mailer would hand codes to whoever reads the logs (or to nobody), so
// the service must refuse to start. Load calls it.
func (c *AppConfig) Validate() error {
	if c.App.Environment == EnvironmentDev {
		return nil
	}
	if c.OTP.DevFixedCode != "" {
		return fmt.Errorf("otp.dev_fixed_code must be empty when app.environment is %q (only allowed in %q)", c.App.Environment, EnvironmentDev)
	}
	if c.Mailer.Provider == MailerProviderLog {
		return fmt.Errorf("mailer.provider %q is not allowed when app.environment is %q (only allowed in %q)", MailerProviderLog, c.App.Environment, EnvironmentDev)
	}
	return nil
}

// OTPConfig holds the OTP policy: how long a code lives, how often a code may
// be sent, and when a code gets locked. Every limit is scoped by (email, purpose).
//
// These numbers used to be a const block inside the registration use case,
// justified there as "business decisions, not configuration". That reasoning is
// wrong for all of them: each is a number a dev environment shortens to exercise
// expiry and production lengthens under load, which is the definition of
// configuration. The use case now takes them through its constructor, the way
// it already took sessionTTL.
//
// v5.0 split the old "resend" cap into a per-(email, purpose) *send* policy: the first
// send of a cycle is free, every later request-otp AND resend-otp counts, and
// hitting the cap locks sending for SendLockDuration. See design.md
// §"OTP send-limit policy".
type OTPConfig struct {
	// TTL is how long an issued code can be verified.
	TTL Duration `yaml:"ttl"`

	// SendCooldown is the floor between two sends for one (email, purpose) (onboarding AC9), applied
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

	// MaxAttempts is how many wrong codes lock the code (onboarding AC15).
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

	// DevFixedCode, when non-empty, replaces the random code with this value
	// so a developer can register without reading mail. Empty = crypto/rand.
	// AppConfig.Validate refuses it outside app.environment "dev" (AC21).
	DevFixedCode string `yaml:"dev_fixed_code"`
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

// AuthRateLimitConfig holds the per-IP budgets of every /v1/auth/* route
// (login AC31). The counters live in the shared rate-limit store
// (rate_limit_store), so the limit holds across instances when that store is
// Redis. `logout` is deliberately absent: it is not limited.
type AuthRateLimitConfig struct {
	// Send covers register request-otp, register resend-otp and
	// password-reset request-otp: every route that can send an email.
	Send RouteRateLimit `yaml:"send"`

	// Verify covers register verify-otp and password-reset confirm.
	Verify RouteRateLimit `yaml:"verify"`

	// Login covers POST /v1/auth/login.
	Login RouteRateLimit `yaml:"login"`

	// Refresh covers POST /v1/auth/refresh.
	Refresh RouteRateLimit `yaml:"refresh"`

	// TrustForwardedFor reads the client IP from the last X-Forwarded-For entry
	// when no valid web API key vouches for X-InnerLamp-Client-IP (AC34 (b)).
	// Set it only behind a proxy that appends the real client address (Cloud
	// Run does).
	TrustForwardedFor bool `yaml:"trust_forwarded_for"`
}

// RouteRateLimit is one group's budget: Max requests per client IP per Window.
type RouteRateLimit struct {
	Max    int      `yaml:"max"`
	Window Duration `yaml:"window"`
}

// Validate rejects a group that is not a positive count over a positive
// window; a zero would silently block (or, in some stores, never limit).
func (c AuthRateLimitConfig) Validate() error {
	for name, g := range map[string]RouteRateLimit{"send": c.Send, "verify": c.Verify, "login": c.Login, "refresh": c.Refresh} {
		if g.Max < 1 {
			return fmt.Errorf("auth_rate_limit.%s.max must be >= 1, got %d", name, g.Max)
		}
		if g.Window.Duration() <= 0 {
			return fmt.Errorf("auth_rate_limit.%s.window must be > 0, got %s", name, g.Window.Duration())
		}
	}
	return nil
}

// PasswordResetConfig holds the forgot-password flow's tunables (login AC25).
type PasswordResetConfig struct {
	// MinResponseTime is the floor on request-otp's response time, so the mail
	// call made only for an email with an account does not reveal the branch.
	// Tests and dev set it low; production keeps it above the slowest mail call.
	MinResponseTime Duration `yaml:"min_response_time"`
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

	// DefaultLanguage is the language GET /v1/calendar/events returns event text
	// in (dev: "vi"). Required: an event only appears when it has text in this
	// language.
	DefaultLanguage string `yaml:"default_language"`

	// EventsCacheMaxAge is how long a client or shared cache may reuse the events of an
	// explicit year, sent as Cache-Control: public, max-age=<seconds> (dev: "24h"). The
	// catalogue only changes when an operator reloads it.
	EventsCacheMaxAge Duration `yaml:"events_cache_max_age"`

	// EventsRateLimit caps how many GET /v1/calendar/events a client IP may make per
	// window. The route is public (v3.0); a request with the valid web API key is not
	// counted. Required: a missing or zero value is rejected, never "unlimited".
	EventsRateLimit CalendarEventsRateLimit `yaml:"events_rate_limit"`
}

// CalendarEventsRateLimit is the per-IP limit of GET /v1/calendar/events.
type CalendarEventsRateLimit struct {
	// Max is the number of requests allowed per Window (dev: 60).
	Max int `yaml:"max"`

	// Window is the length of the counting window (dev: "1m").
	Window Duration `yaml:"window"`

	// TrustForwardedFor reads the client IP from the last X-Forwarded-For entry
	// instead of the TCP peer; see TempleSuggestionRateLimit for when it is safe.
	TrustForwardedFor bool `yaml:"trust_forwarded_for"`
}

// Validate rejects a missing events language, cache lifetime or rate limit. The timezone
// and image base URL are checked where they are used (NewCalendarUsecase).
func (c CalendarConfig) Validate() error {
	if c.DefaultLanguage == "" {
		return fmt.Errorf("calendar.default_language is required")
	}
	if len(c.DefaultLanguage) > 8 {
		return fmt.Errorf("calendar.default_language %q is longer than 8 characters", c.DefaultLanguage)
	}
	if c.EventsCacheMaxAge.Duration() < time.Second {
		return fmt.Errorf("calendar.events_cache_max_age must be >= 1s, got %s", c.EventsCacheMaxAge.Duration())
	}
	if c.EventsRateLimit.Max < 1 {
		return fmt.Errorf("calendar.events_rate_limit.max must be >= 1, got %d", c.EventsRateLimit.Max)
	}
	if c.EventsRateLimit.Window.Duration() <= 0 {
		return fmt.Errorf("calendar.events_rate_limit.window must be > 0, got %s", c.EventsRateLimit.Window.Duration())
	}
	return nil
}

// TempleAnnouncementConfig holds the temple-announcement read side settings
// (docs/specs/features/temple-announcement, GET /v1/users/me/temple-announcements).
type TempleAnnouncementConfig struct {
	// PageSize is the page size used when the request gives none.
	PageSize int `yaml:"page_size"`

	// MaxPageSize caps the page_size a request may ask for.
	MaxPageSize int `yaml:"max_page_size"`

	// RateLimit is the per-account limit of the three announcement routes,
	// separate from the profile limiter (TA32, TA33).
	RateLimit TempleAnnouncementRateLimit `yaml:"rate_limit"`
}

// TempleAnnouncementRateLimit is the per-account request limit.
type TempleAnnouncementRateLimit struct {
	Max    int64    `yaml:"max"`
	Window Duration `yaml:"window"`
}

// Validate rejects a missing block (zero values) and an inverted page range.
func (c TempleAnnouncementConfig) Validate() error {
	if c.PageSize < 1 {
		return fmt.Errorf("temple_announcement.page_size must be >= 1, got %d", c.PageSize)
	}
	if c.MaxPageSize < c.PageSize {
		return fmt.Errorf("temple_announcement.max_page_size (%d) must be >= page_size (%d)", c.MaxPageSize, c.PageSize)
	}
	if c.RateLimit.Max < 1 {
		return fmt.Errorf("temple_announcement.rate_limit.max must be >= 1, got %d", c.RateLimit.Max)
	}
	if c.RateLimit.Window.Duration() <= 0 {
		return fmt.Errorf("temple_announcement.rate_limit.window must be > 0, got %s", c.RateLimit.Window.Duration())
	}
	return nil
}

// TempleEventConfig holds the temple-event endpoint's settings
// (GET /v1/temples/{id}/events).
type TempleEventConfig struct {
	// Timezone is the IANA zone in which an event's clock time, "today" and
	// all-day events are read. Required.
	Timezone string `yaml:"timezone"`

	// DefaultWindowMonths is how far past "from" the window reaches when the
	// request gives no "to".
	DefaultWindowMonths int `yaml:"default_window_months"`

	// MaxWindowMonths is the longest window a request may ask for.
	MaxWindowMonths int `yaml:"max_window_months"`
}

// Validate rejects a missing or unknown zone and window lengths that are not
// 1 <= default <= max, so a missing temple_event block stops the service now.
func (c TempleEventConfig) Validate() error {
	if c.Timezone == "" { // LoadLocation("") silently means UTC
		return fmt.Errorf("temple_event.timezone is required")
	}
	if _, err := time.LoadLocation(c.Timezone); err != nil {
		return fmt.Errorf("temple_event.timezone %q: %w", c.Timezone, err)
	}
	if c.DefaultWindowMonths < 1 {
		return fmt.Errorf("temple_event.default_window_months must be >= 1, got %d", c.DefaultWindowMonths)
	}
	if c.MaxWindowMonths < c.DefaultWindowMonths {
		return fmt.Errorf("temple_event.max_window_months (%d) must be >= default_window_months (%d)",
			c.MaxWindowMonths, c.DefaultWindowMonths)
	}
	return nil
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

	// RecipientMaxChars is the longest "dành cho ai" accepted, in characters
	// (dev: 300). It can be lowered freely but never raised past
	// LampRecipientColumnChars.
	RecipientMaxChars int `yaml:"recipient_max_chars"`

	// PrayerMaxChars is the longest lời nguyện accepted, in characters
	// (dev: 300). It can be lowered freely but never raised past
	// LampPrayerColumnChars.
	PrayerMaxChars int `yaml:"prayer_max_chars"`
}

// The sizes of the VARCHAR columns lamps.recipient and lamps.prayer in
// migrations/0027_lamps.sql. They are the ceiling for the two length limits
// above, not the values used to check input: a limit above its column would
// pass validation and then fail the INSERT with a 500. Raising a ceiling takes
// a migration that widens the column first.
const (
	LampRecipientColumnChars = 100
	LampPrayerColumnChars    = 300
)

// Validate rejects a LampConfig that would make lighting or paging impossible:
// a limit below 1, a non-positive window, a duration list that is empty, holds a
// value below 1 or repeats one, or a length limit below 1 or above its column.
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
	if c.RecipientMaxChars < 1 || c.RecipientMaxChars > LampRecipientColumnChars {
		return fmt.Errorf("lamp.recipient_max_chars must be between 1 and %d, got %d", LampRecipientColumnChars, c.RecipientMaxChars)
	}
	if c.PrayerMaxChars < 1 || c.PrayerMaxChars > LampPrayerColumnChars {
		return fmt.Errorf("lamp.prayer_max_chars must be between 1 and %d, got %d", LampPrayerColumnChars, c.PrayerMaxChars)
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

// ProfileConfig holds the hồ sơ cá nhân settings (docs/specs/features/profile):
// the dharma name limit shared with registration, the avatar upload policy, where
// avatar objects live, and the per-account request limit.
//
// Every value is required; Validate rejects a zero, because a missing yaml block
// would otherwise decode to "no avatar may be larger than 0 bytes" or "a URL that
// expired the instant it was signed".
type ProfileConfig struct {
	// DharmaNameMaxChars is the longest dharma name accepted, in characters
	// after trimming and NFC normalisation (dev: 255). It can be lowered
	// freely but never raised past ProfileDharmaNameColumnChars.
	DharmaNameMaxChars int `yaml:"dharma_name_max_chars"`

	// FullNameMaxChars is the longest full name accepted at registration, in
	// characters after trimming and NFC normalisation (dev: 100). Never raised
	// past ProfileFullNameColumnChars.
	FullNameMaxChars int `yaml:"full_name_max_chars"`

	// AvatarMaxBytes is the largest avatar accepted, checked both against the
	// client's claim and against the size storage recorded (dev: 5 MiB).
	AvatarMaxBytes int64 `yaml:"avatar_max_bytes"`
	// AvatarContentTypes is the allowed avatar media types.
	AvatarContentTypes []string `yaml:"avatar_content_types"`

	// UploadURLTTL is how long a signed upload URL is valid (dev: 5m).
	UploadURLTTL Duration `yaml:"upload_url_ttl"`
	// AvatarReadURLTTL is how long a signed read URL is valid (dev: 15m).
	AvatarReadURLTTL Duration `yaml:"avatar_read_url_ttl"`

	// Bucket holds the avatar objects. It is not a secret but differs per
	// environment, so it is config rather than a constant.
	Bucket string `yaml:"bucket"`
	// AvatarTmpPrefix is the folder a client uploads into before confirming.
	AvatarTmpPrefix string `yaml:"avatar_tmp_prefix"`
	// AvatarPrefix is the folder confirmed avatars are copied to.
	AvatarPrefix string `yaml:"avatar_prefix"`

	// RateLimit caps how many profile requests one account may make per window.
	RateLimit ProfileRateLimit `yaml:"rate_limit"`
}

// ProfileRateLimit is the per-account request limit of the profile routes.
type ProfileRateLimit struct {
	Max    int64    `yaml:"max"`
	Window Duration `yaml:"window"`
}

// ProfileDharmaNameColumnChars is the size of users.dharma_name and
// pending_registration.dharma_name (VARCHAR(255), migrations/0001). It is the
// ceiling for DharmaNameMaxChars, not the value used to check input: a limit
// above the column would pass validation and then fail the INSERT with a 500.
const ProfileDharmaNameColumnChars = 255

// ProfileFullNameColumnChars is the size of users.full_name and
// pending_registration.full_name (VARCHAR(300), migrations/0033).
const ProfileFullNameColumnChars = 300

// Validate rejects a ProfileConfig that would reject every request or pass
// values the schema cannot hold.
func (c ProfileConfig) Validate() error {
	if c.DharmaNameMaxChars < 1 || c.DharmaNameMaxChars > ProfileDharmaNameColumnChars {
		return fmt.Errorf("profile.dharma_name_max_chars must be between 1 and %d, got %d", ProfileDharmaNameColumnChars, c.DharmaNameMaxChars)
	}
	if c.FullNameMaxChars < 1 || c.FullNameMaxChars > ProfileFullNameColumnChars {
		return fmt.Errorf("profile.full_name_max_chars must be between 1 and %d, got %d", ProfileFullNameColumnChars, c.FullNameMaxChars)
	}
	if c.AvatarMaxBytes < 1 {
		return fmt.Errorf("profile.avatar_max_bytes must be >= 1, got %d", c.AvatarMaxBytes)
	}
	if len(c.AvatarContentTypes) == 0 {
		return fmt.Errorf("profile.avatar_content_types must list at least one media type")
	}
	seen := make(map[string]bool, len(c.AvatarContentTypes))
	for _, ct := range c.AvatarContentTypes {
		if strings.TrimSpace(ct) == "" {
			return fmt.Errorf("profile.avatar_content_types contains a blank entry")
		}
		if seen[ct] {
			return fmt.Errorf("profile.avatar_content_types repeats %q", ct)
		}
		seen[ct] = true
	}
	if c.UploadURLTTL.Duration() <= 0 {
		return fmt.Errorf("profile.upload_url_ttl must be > 0, got %s", c.UploadURLTTL.Duration())
	}
	if c.AvatarReadURLTTL.Duration() <= 0 {
		return fmt.Errorf("profile.avatar_read_url_ttl must be > 0, got %s", c.AvatarReadURLTTL.Duration())
	}
	if strings.TrimSpace(c.Bucket) == "" {
		return fmt.Errorf("profile.bucket must not be empty")
	}
	if c.AvatarTmpPrefix == "" || c.AvatarPrefix == "" {
		return fmt.Errorf("profile.avatar_tmp_prefix and profile.avatar_prefix must not be empty")
	}
	if c.AvatarTmpPrefix == c.AvatarPrefix {
		return fmt.Errorf("profile.avatar_tmp_prefix and profile.avatar_prefix must differ, both are %q", c.AvatarPrefix)
	}
	if c.RateLimit.Max < 1 {
		return fmt.Errorf("profile.rate_limit.max must be >= 1, got %d", c.RateLimit.Max)
	}
	if c.RateLimit.Window.Duration() <= 0 {
		return fmt.Errorf("profile.rate_limit.window must be > 0, got %s", c.RateLimit.Window.Duration())
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

// AdministrativeUnitsConfig holds what the public administrative-units lookups
// (docs/specs/features/administrative-units) need beyond the data itself: an
// abuse guard and a cache lifetime. The routes are read-only and reachable
// without an account, and the web API key can be skipped by a caller that
// omits the header, so the per-IP limit is the only other brake; a missing or
// zero value is rejected rather than defaulted to "unlimited".
type AdministrativeUnitsConfig struct {
	// RateLimit caps how many lookups one client IP may make per window. All
	// administrative-units routes share one budget.
	RateLimit AdministrativeUnitsRateLimit `yaml:"rate_limit"`

	// CacheMaxAge is how long a client may reuse a successful answer, sent as
	// Cache-Control: public, max-age=<seconds> (dev: "24h"). The data only
	// changes when an operator loads a new dataset version.
	CacheMaxAge Duration `yaml:"cache_max_age"`
}

// AdministrativeUnitsRateLimit is the per-IP lookup limit.
type AdministrativeUnitsRateLimit struct {
	// Max is the number of lookups allowed per Window (dev: 60).
	Max int `yaml:"max"`

	// Window is the length of the counting window (dev: "1m").
	Window Duration `yaml:"window"`

	// TrustForwardedFor reads the client IP from the last X-Forwarded-For entry
	// instead of the TCP peer; see TempleSuggestionRateLimit for when it is safe.
	TrustForwardedFor bool `yaml:"trust_forwarded_for"`
}

// Validate rejects a missing limit or cache lifetime: the limit must be a
// positive count over a positive window, and the lifetime must be at least one
// second (Cache-Control counts whole seconds).
func (c AdministrativeUnitsConfig) Validate() error {
	if c.RateLimit.Max < 1 {
		return fmt.Errorf("administrative_units.rate_limit.max must be >= 1, got %d", c.RateLimit.Max)
	}
	if c.RateLimit.Window.Duration() <= 0 {
		return fmt.Errorf("administrative_units.rate_limit.window must be > 0, got %s", c.RateLimit.Window.Duration())
	}
	if c.CacheMaxAge.Duration() < time.Second {
		return fmt.Errorf("administrative_units.cache_max_age must be >= 1s, got %s", c.CacheMaxAge.Duration())
	}
	return nil
}

// RateLimitStore backends.
const (
	RateLimitStoreRedis  = "redis"
	RateLimitStoreMemory = "memory"
)

// RateLimitStoreConfig selects the store behind every rate limiter. Redis
// counts across all instances; memory counts per instance, so N instances
// allow N times the configured limit. There is no default: a missing block
// must stop the service rather than silently fall back to per-instance counts.
// RateLimitStoreConfig chọn store cho mọi rate limiter. Redis đếm chung cho mọi
// instance; memory đếm theo từng instance nên N instance cho qua gấp N lần limit.
// Không có giá trị mặc định: thiếu block thì service phải dừng, không âm thầm
// chuyển sang đếm theo instance.
type RateLimitStoreConfig struct {
	// Backend is "redis" or "memory" ("memory" only for tests or a single instance).
	// Backend là "redis" hoặc "memory" ("memory" chỉ dùng cho test hoặc một instance).
	Backend string `yaml:"backend"`

	Redis RedisRateLimitConfig `yaml:"redis"`
}

// RedisRateLimitConfig is the Redis connection the rate limiters share.
// PasswordSecret names the secret resolved into Password; Password itself is
// never read from YAML.
// RedisRateLimitConfig là kết nối Redis các rate limiter dùng chung.
// PasswordSecret là tên secret được resolve vào Password; Password không bao giờ
// đọc từ YAML.
type RedisRateLimitConfig struct {
	Addr           string `yaml:"addr"`
	PasswordSecret string `yaml:"password_secret"`
	Password       string `yaml:"-"`
	DB             int    `yaml:"db"`

	// KeyPrefix starts every key, so one Redis can serve several services or
	// environments without their counters colliding (dev: "innerlamp:app:rl:").
	// KeyPrefix đứng đầu mọi key để một Redis phục vụ nhiều service/môi trường mà
	// counter không đè nhau.
	KeyPrefix string `yaml:"key_prefix"`

	// DialTimeout bounds the startup ping and every reconnect (dev: "1s").
	// DialTimeout giới hạn lần ping lúc khởi động và mỗi lần kết nối lại.
	DialTimeout Duration `yaml:"dial_timeout"`

	// OpTimeout bounds each limit check. It sits on the request path, so it is
	// kept short: past it the check falls back to a per-instance count instead
	// of holding the request (dev: "100ms").
	// OpTimeout giới hạn mỗi lần kiểm tra limit. Nó nằm trên đường đi của request
	// nên phải ngắn: quá hạn thì chuyển sang đếm theo instance thay vì giữ request.
	OpTimeout Duration `yaml:"op_timeout"`
}

// Validate rejects an unknown backend and, for Redis, a missing address,
// prefix or timeout.
// Validate từ chối backend lạ và, với Redis, thiếu địa chỉ, prefix hoặc timeout.
func (c RateLimitStoreConfig) Validate() error {
	switch c.Backend {
	case RateLimitStoreMemory:
		return nil
	case RateLimitStoreRedis:
	default:
		return fmt.Errorf("rate_limit_store.backend must be %q or %q, got %q",
			RateLimitStoreRedis, RateLimitStoreMemory, c.Backend)
	}
	r := c.Redis
	if strings.TrimSpace(r.Addr) == "" {
		return fmt.Errorf("rate_limit_store.redis.addr is required")
	}
	if strings.TrimSpace(r.KeyPrefix) == "" {
		return fmt.Errorf("rate_limit_store.redis.key_prefix is required")
	}
	if r.DB < 0 {
		return fmt.Errorf("rate_limit_store.redis.db must be >= 0, got %d", r.DB)
	}
	if r.DialTimeout.Duration() <= 0 {
		return fmt.Errorf("rate_limit_store.redis.dial_timeout must be > 0, got %s", r.DialTimeout.Duration())
	}
	if r.OpTimeout.Duration() <= 0 {
		return fmt.Errorf("rate_limit_store.redis.op_timeout must be > 0, got %s", r.OpTimeout.Duration())
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

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate app config: %w", err)
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

	if c.RateLimitStore.Redis.PasswordSecret != "" {
		v, err := getSecret(ctx, sg, c.RateLimitStore.Redis.PasswordSecret)
		if err != nil {
			return fmt.Errorf("rate_limit_store.redis.password_secret %q: %w", c.RateLimitStore.Redis.PasswordSecret, err)
		}
		c.RateLimitStore.Redis.Password = v
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
