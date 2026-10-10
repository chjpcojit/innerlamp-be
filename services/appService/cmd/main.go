// ⚠️ DEV CONFIG - NOT FOR PRODUCTION
// Non-sensitive settings load from config/config.dev.yaml. Sensitive fields
// (DB password, JWT secret key) resolve through a SecretGetter; here that's
// appConfig.MockSecretGetter, which reads plaintext values from the same
// YAML's dev_secrets section instead of a real secret manager.
//
// TODO before deploying: point CONFIG_PATH at a real config source
// (s3:// or gs://) and swap MockSecretGetter for pkg/secrets/gcp (or aws).
package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"
	_ "time/tzdata" // memorial reminders name IANA zones; runtime images may ship no zoneinfo

	"chjpcoj.com.innerlamp/pkg/attestation"
	"chjpcoj.com.innerlamp/pkg/attestation/appattest"
	"chjpcoj.com.innerlamp/pkg/attestation/mock"
	"chjpcoj.com.innerlamp/pkg/attestation/playintegrity"
	pkgconfig "chjpcoj.com.innerlamp/pkg/config"
	"chjpcoj.com.innerlamp/pkg/database"
	_ "chjpcoj.com.innerlamp/pkg/database/postgresql"
	"chjpcoj.com.innerlamp/pkg/jwt"
	jwtmiddleware "chjpcoj.com.innerlamp/pkg/jwt/middleware"
	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/pkg/logging/formatter"
	httpmiddleware "chjpcoj.com.innerlamp/pkg/middleware"
	"chjpcoj.com.innerlamp/pkg/sms"
	_ "chjpcoj.com.innerlamp/pkg/sms/logsms"
	"chjpcoj.com.innerlamp/pkg/validation"

	"chjpcoj.com.innerlamp/assets"
	appConfig "chjpcoj.com.innerlamp/services/appService/config"
	deliveryhttp "chjpcoj.com.innerlamp/services/appService/delivery/http"
	"chjpcoj.com.innerlamp/services/appService/domain"
	"chjpcoj.com.innerlamp/services/appService/repository"
	"chjpcoj.com.innerlamp/services/appService/usecase"
)

const defaultConfigPath = "services/appService/config/config.dev.yaml"

// sutraSeedTimeout bounds the first-run seed of the sutra library (~64,000 lines in
// one transaction). It has its own budget because connectCtx's 10s is shared with
// config loading, the DB connection and the article seed, and the sutra seed is
// by far the heaviest of them (about a second on a local Postgres).
// TD-008: hardcoded, and repeated in cmd/seedsutras; not yet in config.
const sutraSeedTimeout = 2 * time.Minute

func main() {
	logger := logging.New(logging.Config{
		Level:     logging.InfoLevel,
		Formatter: formatter.NewJSON(),
	})

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = defaultConfigPath
	}
	configSource := pkgconfig.LoadSource(configPath)

	connectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rawConfig, err := pkgconfig.LoadYAML(connectCtx, configSource)
	if err != nil {
		log.Fatalf("failed to load config %s: %v", configPath, err)
	}

	// ⚠️ DEV ONLY: resolves *_secret names against config.dev.yaml's
	// dev_secrets map. Swap for pkg/secrets/gcp (or aws) in real environments.
	secretGetter, err := appConfig.NewMockSecretGetterFromYAML(rawConfig)
	if err != nil {
		log.Fatalf("failed to load dev secrets from %s: %v", configPath, err)
	}

	cfg, err := appConfig.Load(connectCtx, configSource, secretGetter)
	if err != nil {
		log.Fatalf("failed to load app config: %v", err)
	}
	// A zero here would make every sutra page empty and divide by zero in total_pages,
	// so a config.<env>.yaml without a valid sutra block must stop the service now.
	if err := cfg.Sutra.Validate(); err != nil {
		log.Fatalf("invalid sutra config: %v", err)
	}
	// Same for dharma_talk.page_size: zero would divide by zero in total_pages.
	if err := cfg.DharmaTalk.Validate(); err != nil {
		log.Fatalf("invalid dharma_talk config: %v", err)
	}
	// Same for the temple page sizes: zero would divide by zero in total_pages.
	if err := cfg.Temple.Validate(); err != nil {
		log.Fatalf("invalid temple config: %v", err)
	}
	// The temple-suggestion route is anonymous and writes to the DB; without a valid
	// per-IP limit it would be an open write endpoint, so stop now rather than serve it.
	if err := cfg.TempleSuggestion.Validate(); err != nil {
		log.Fatalf("invalid temple_suggestion config: %v", err)
	}

	// Zero genealogy limits would mean nobody can create a tree or an invitation is
	// born expired, so a missing genealogy block must stop the service now.
	if err := cfg.Genealogy.Validate(); err != nil {
		log.Fatalf("invalid genealogy config: %v", err)
	}

	// Zero lamp limits would mean nobody can light a lamp or a garden page is empty, so a
	// missing lamp block must stop the service now.
	if err := cfg.Lamp.Validate(); err != nil {
		log.Fatalf("invalid lamp config: %v", err)
	}

	dbConfig := &database.Config{
		Type:     database.PostgreSQL,
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		Username: cfg.Database.Username,
		Password: cfg.Database.Password,
		Database: cfg.Database.Database,
		SSLMode:  cfg.Database.SSLMode,
	}

	db, err := database.New(dbConfig)
	if err != nil {
		log.Fatalf("failed to create database client: %v", err)
	}

	if err := db.Connect(connectCtx); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer func() { _ = db.Close() }()

	jwtConfig := &jwt.Config{
		Algorithm:              jwt.Algorithm(cfg.JWT.Algorithm),
		SecretKey:              cfg.JWT.SecretKey,
		AccessTokenExpiration:  cfg.JWT.AccessTokenExpiration.Duration(),
		RefreshTokenExpiration: cfg.JWT.RefreshTokenExpiration.Duration(),
		Issuer:                 cfg.JWT.Issuer,
		Audience:               cfg.JWT.Audience,
	}
	jwtService, err := jwt.New(jwtConfig)
	if err != nil {
		log.Fatalf("failed to create jwt service: %v", err)
	}

	// validate.Register/SetFieldNameTag are not thread-safe and must both run
	// before the first Struct() call — see pkg/validation's doc comments —
	// so this happens once here, before any handler is built.
	validator := validation.New()
	if err := validator.Register(validation.PhoneTag, validation.PhoneRule); err != nil {
		log.Fatalf("failed to register %s validation: %v", validation.PhoneTag, err)
	}
	validator.SetFieldNameTag("json")

	// ⚠️ DEV ONLY: pkg/sms/logsms logs the OTP instead of sending it via a
	// real provider (Twilio, AWS SNS, a GCP equivalent). Swap
	// cfg.SMS.Provider (config.dev.yaml) and its blank import above once a
	// real provider subpackage exists — pkg/sms.Sender does not change.
	smsSender, err := sms.New(&sms.Config{
		Provider: sms.ProviderType(cfg.SMS.Provider),
		Logger:   logger,
	})
	if err != nil {
		log.Fatalf("failed to create sms sender: %v", err)
	}

	registrationRepo := repository.NewRegistrationRepository(db)
	otpRepo := repository.NewOTPRepository(db)
	otpSender := repository.NewSMSOTPSender(smsSender)
	otpService := usecase.NewOTPService(otpRepo, otpSender, logger, cfg.OTP)
	registerUsecase := usecase.NewRegisterUsecase(
		registrationRepo,
		otpService,
		jwtService,
		logger,
		cfg.JWT.RefreshTokenExpiration.Duration(),
		cfg.OTP.PendingRegistrationTTL.Duration(),
	)
	// accessTTL is what jwtService mints access tokens with; handlers report it as expires_in.
	accessTTL := deliveryhttp.WithAccessTokenTTL(cfg.JWT.AccessTokenExpiration.Duration())
	registerHandler := deliveryhttp.NewRegisterHandler(registerUsecase, validator, accessTTL)

	loginRepo := repository.NewLoginRepository(db)
	loginUsecase := usecase.NewLoginUsecase(
		loginRepo,
		jwtService,
		logger,
		cfg.JWT.RefreshTokenExpiration.Duration(),
		cfg.Login,
	)
	loginHandler := deliveryhttp.NewLoginHandler(loginUsecase, accessTTL)

	// Attestation: each platform selects its own pkg/attestation driver, because
	// the two backends are asymmetric — android is playintegrity (server-to-
	// server decode), ios is appattest (local crypto). In dev both are the mock
	// driver (⚠️ DEV ONLY — it fakes results and echoes the configured package
	// name / verdict back so the verify flow round-trips). Point at the real
	// drivers via cfg.Attestation.*.Provider once service-account credentials
	// and the published app id exist.
	attestationAndroid, err := attestation.New(&attestation.Config{
		Provider: attestation.ProviderType(cfg.Attestation.Android.Provider),
		Options: map[string]string{
			mock.OptionPackageName:              cfg.Attestation.Android.PackageName,
			mock.OptionAppRecognitionVerdict:    cfg.Attestation.Android.RequiredAppRecognitionVerdict,
			playintegrity.OptionCredentialsJSON: cfg.Attestation.Android.Credentials,
		},
	})
	if err != nil {
		log.Fatalf("failed to create android attestation verifier: %v", err)
	}

	attestationIOS, err := attestation.New(&attestation.Config{
		Provider: attestation.ProviderType(cfg.Attestation.IOS.Provider),
		Options: map[string]string{
			appattest.OptionAppID: cfg.Attestation.IOS.AppID,
		},
	})
	if err != nil {
		log.Fatalf("failed to create ios attestation verifier: %v", err)
	}

	attestationRepo := repository.NewAttestationRepository(db)
	attestationUsecase := usecase.NewAttestationUsecase(
		attestationRepo,
		repository.NewAttestationVerifier(attestationAndroid, attestationIOS),
		logger,
		cfg.Attestation,
	)
	attestationHandler := deliveryhttp.NewAttestationHandler(attestationUsecase, validator)

	// Temple catalogue: read-only browse endpoints (top / paged list / detail).
	templeRepo := repository.NewTempleRepository(db)
	templeUsecase := usecase.NewTempleUsecase(templeRepo, cfg.Temple)
	// Catalogue image paths are stored root-relative and resolved per response, so a new
	// network, host or CDN is a config change, not a data change.
	imageBase := deliveryhttp.WithImageBase(cfg.Server.PublicBaseURL)
	templeHandler := deliveryhttp.NewTempleHandler(templeUsecase, imageBase)

	// Deity catalogue (Phật / Bồ tát): read-only top / bodhisattva list / detail.
	deityRepo := repository.NewDeityRepository(db)
	deityUsecase := usecase.NewDeityUsecase(deityRepo, cfg.Deity)
	deityHandler := deliveryhttp.NewDeityHandler(deityUsecase, imageBase)

	// Gallery (ảnh đẹp Phật giáo): read-only top / paged list (optional type filter) / detail.
	// Data is loaded by cmd/seedgallery, never at startup.
	galleryRepo := repository.NewGalleryRepository(db)
	galleryUsecase := usecase.NewGalleryUsecase(galleryRepo, cfg.Gallery)
	galleryHandler := deliveryhttp.NewGalleryHandler(galleryUsecase, imageBase)

	// Calendar image (ảnh Phật theo ngày cho màn Lịch âm): no database — the list comes from
	// the embedded assets/images/home-calendar/manifest.json, validated here so a bad manifest
	// or time zone stops the service now instead of failing per request.
	calendarImages, err := repository.ParseCalendarManifest(assets.CalendarManifestJSON)
	if err != nil {
		log.Fatalf("invalid calendar manifest: %v", err)
	}
	calendarUsecase, err := usecase.NewCalendarUsecase(calendarImages, cfg.Calendar, time.Now)
	if err != nil {
		log.Fatalf("invalid calendar config: %v", err)
	}
	calendarHandler := deliveryhttp.NewCalendarHandler(calendarUsecase, cfg.Calendar.ImageBaseURL)

	// Article catalogue (bài viết Phật pháp): read-only top / paged list / detail.
	// First run only: seed from the embedded assets/data/buddhist_articles.json
	// when the table is empty (needs migrations/0012 applied). A restart never
	// re-seeds, so it cannot overwrite or resurrect edited rows.
	articleRepo := repository.NewArticleRepository(db)
	articleUsecase := usecase.NewArticleUsecase(articleRepo, cfg.Article)
	seeded, err := articleUsecase.SeedIfEmpty(connectCtx, func() ([]domain.Article, error) {
		return repository.ParseArticleSeed(assets.BuddhistArticlesJSON)
	})
	if err != nil {
		log.Fatalf("failed to seed articles: %v", err)
	}
	if seeded > 0 {
		logger.Info("seeded articles on first run", "count", seeded)
	}
	articleHandler := deliveryhttp.NewArticleHandler(articleUsecase, imageBase)

	// Dharma talks (pháp thoại): read-only paged list / detail. First run only: seed
	// from the embedded assets/data/dharma_talks.json when the table is empty
	// (needs migrations/0014 applied). Talks flagged needs_review are stored but
	// never served; the emptiness check counts them so a restart cannot reseed.
	dharmaTalkRepo := repository.NewDharmaTalkRepository(db)
	dharmaTalkUsecase := usecase.NewDharmaTalkUsecase(dharmaTalkRepo, cfg.DharmaTalk)
	seededTalks, err := dharmaTalkUsecase.SeedIfEmpty(connectCtx, func() ([]domain.DharmaTalk, error) {
		return repository.ParseDharmaTalkSeed(assets.DharmaTalksJSON)
	})
	if err != nil {
		log.Fatalf("failed to seed dharma talks: %v", err)
	}
	if seededTalks > 0 {
		logger.Info("seeded dharma talks on first run", "count", seededTalks)
	}
	dharmaTalkHandler := deliveryhttp.NewDharmaTalkHandler(dharmaTalkUsecase)

	// Sutra library (kinh / kệ / chú): read-only list / detail / lines, plus the sutras
	// linked to a deity. First run only: seed from the embedded assets/data/sutras when
	// the table is empty (needs migrations/0013 applied and the deities seeded with
	// cmd/seeddeities, because every sutra links to deities; a missing deity stops the
	// service rather than dropping links the seed could never add later). A restart
	// never re-seeds, so it cannot overwrite or resurrect edited rows.
	sutraRepo := repository.NewSutraRepository(db)
	sutraUsecase := usecase.NewSutraUsecase(sutraRepo, deityRepo, cfg.Sutra)
	seedCtx, cancelSeed := context.WithTimeout(context.Background(), sutraSeedTimeout)
	sutraSeed, err := sutraUsecase.SeedIfEmpty(seedCtx, func() ([]domain.SutraSeed, error) {
		dir, err := fs.Sub(assets.SutrasFS, "data/sutras")
		if err != nil {
			return nil, err
		}
		return repository.ParseSutraSeed(dir)
	})
	cancelSeed()
	if err != nil {
		log.Fatalf("failed to seed sutras: %v", err)
	}
	if sutraSeed.Sutras > 0 {
		logger.Info("seeded sutras on first run", "sutras", sutraSeed.Sutras, "translations", sutraSeed.Translations,
			"lines", sutraSeed.Lines, "deity_links", sutraSeed.DeityLinks)
	}
	sutraHandler := deliveryhttp.NewSutraHandler(sutraUsecase, imageBase)

	// memorials (sổ giỗ chạp): per-user data, so every route below is wrapped by authMiddleware.
	memorialRepo := repository.NewMemorialRepository(db)
	memorialUsecase := usecase.NewMemorialUsecase(memorialRepo, cfg.Memorial)
	memorialHandler := deliveryhttp.NewMemorialHandler(memorialUsecase)

	// lamps (thắp đèn / vườn đèn): both routes need a signed-in user. The garden shows every
	// user's lit lamps, but the lighter is always the access token's user_id claim.
	lampRepo := repository.NewLampRepository(db)
	lampUsecase := usecase.NewLampUsecase(lampRepo, cfg.Lamp)
	lampHandler := deliveryhttp.NewLampHandler(lampUsecase)

	// temple suggestions: anonymous "suggest a temple" write. Review (list / create the
	// temple / reject) is backofficeService's. docs/specs/features/temple-suggestion/.
	templeSuggestionRepo := repository.NewTempleSuggestionRepository(db)
	templeSuggestionUsecase := usecase.NewTempleSuggestionUsecase(templeSuggestionRepo)
	templeSuggestionHandler := deliveryhttp.NewTempleSuggestionHandler(templeSuggestionUsecase)
	templeSuggestionLimit, err := deliveryhttp.NewClientIPRateLimit(
		int64(cfg.TempleSuggestion.RateLimit.Max), cfg.TempleSuggestion.RateLimit.Window.Duration(),
		cfg.TempleSuggestion.RateLimit.TrustForwardedFor)
	if err != nil {
		log.Fatalf("failed to build temple suggestion rate limit: %v", err)
	}

	// The RequireAttestation middleware is provided but deliberately NOT wired
	// to any route yet: choosing which routes to gate is a business decision
	// that belongs to a future feature (design.md "Out of scope"). It is built
	// and tested in delivery/http/attestation_middleware_test.go.

	// logout is the one route in this service that requires an access token. It
	// must be accepted only as type "access" (so a refresh token cannot be used
	// as a bearer), and a rejection must use the same ApiResponse envelope as
	// every other route rather than the middleware's default flat body.
	authMiddleware := jwtmiddleware.NewHTTPAuthMiddlewareWithOptions(jwtService, logger, jwtmiddleware.HTTPAuthOptions{
		RequireTokenType: "access",
		ErrorHandler:     deliveryhttp.WriteAuthError,
	})

	mux := http.NewServeMux()
	// US-001: all three routes are public — the caller has no account yet,
	// so there is nothing to authenticate. See design.md's Overview.
	// Every /v1/auth/* route goes through webAPIKey: a wrong X-API-Key is 401, a
	// missing one means mobile (see RequireWebAPIKey).
	webAPIKey := deliveryhttp.NewRequireWebAPIKey(cfg.Server.WebAPIKey)
	mux.HandleFunc("POST /v1/auth/register/request-otp", webAPIKey.WrapFunc(registerHandler.RequestOTP))
	mux.HandleFunc("POST /v1/auth/resend-otp", webAPIKey.WrapFunc(registerHandler.ResendOTP))
	// verify-otp, login and refresh return tokens: noStore keeps them out of any cache (AC20).
	noStore := deliveryhttp.NoStore{}
	mux.HandleFunc("POST /v1/auth/register/verify-otp", webAPIKey.WrapFunc(noStore.WrapFunc(registerHandler.VerifyOTP)))

	// US-002: login and refresh are public — the caller proves itself with a
	// password / refresh token in the body. logout is wrapped: it identifies the
	// session from the access token's "sid" claim.
	mux.HandleFunc("POST /v1/auth/login", webAPIKey.WrapFunc(noStore.WrapFunc(loginHandler.Login)))
	mux.HandleFunc("POST /v1/auth/refresh", webAPIKey.WrapFunc(noStore.WrapFunc(loginHandler.Refresh)))
	mux.HandleFunc("POST /v1/auth/logout", webAPIKey.WrapFunc(authMiddleware.AuthenticateFunc(loginHandler.Logout)))

	// memorials: private to the logged-in user. The owner is the access token's user_id claim;
	// there is no route that takes a user id. The photo routes are not here yet (TD-001, memorial).
	mux.HandleFunc("POST /v1/memorials", authMiddleware.AuthenticateFunc(memorialHandler.Create))
	mux.HandleFunc("GET /v1/memorials", authMiddleware.AuthenticateFunc(memorialHandler.List))
	mux.HandleFunc("GET /v1/memorials/{id}", authMiddleware.AuthenticateFunc(memorialHandler.Get))
	mux.HandleFunc("PUT /v1/memorials/{id}", authMiddleware.AuthenticateFunc(memorialHandler.Update))
	mux.HandleFunc("DELETE /v1/memorials/{id}", authMiddleware.AuthenticateFunc(memorialHandler.Delete))

	// lamps: no public read — a visitor sees neither the garden nor its counts.
	mux.HandleFunc("POST /v1/lamps", authMiddleware.AuthenticateFunc(lampHandler.Light))
	mux.HandleFunc("GET /v1/lamps", authMiddleware.AuthenticateFunc(lampHandler.Garden))

	// temple suggestions: public by design (the caller may have no account). webAPIKey
	// runs first so a wrong key is refused without spending the caller's rate budget;
	// the per-IP limit then guards the DB write.
	mux.HandleFunc("POST /v1/temple-suggestions", webAPIKey.WrapFunc(templeSuggestionLimit.WrapFunc(templeSuggestionHandler.Create)))

	// attestation: all three routes are public — the caller has no account yet.
	// They mint challenges and return a verdict; the per-request gate is the
	// (unwired) RequireAttestation middleware, not these routes.
	mux.HandleFunc("POST /v1/attestation/challenge", attestationHandler.Challenge)
	mux.HandleFunc("POST /v1/assertion/challenge", attestationHandler.AssertionChallenge)
	mux.HandleFunc("POST /v1/attestation/verify", attestationHandler.Verify)

	// temples: all four routes are public — browsing needs no account. "/top"
	// and "/options" are literal segments, so ServeMux prefers them over "/{id}".
	mux.HandleFunc("GET /v1/temples/top", templeHandler.Top)
	mux.HandleFunc("GET /v1/temples/options", templeHandler.Options)
	mux.HandleFunc("GET /v1/temples", templeHandler.List)
	mux.HandleFunc("GET /v1/temples/{id}", templeHandler.Get)

	// deities: public, like temples. "/top" is a literal segment, so ServeMux
	// prefers it over "/{id}".
	mux.HandleFunc("GET /v1/deities/top", deityHandler.Top)
	mux.HandleFunc("GET /v1/deities/{id}", deityHandler.Get)
	mux.HandleFunc("GET /v1/deities", deityHandler.ListDeities)

	// gallery: public, like temples. "/top" is a literal segment, so ServeMux
	// prefers it over "/{id}".
	mux.HandleFunc("GET /v1/gallery/top", galleryHandler.Top)
	mux.HandleFunc("GET /v1/gallery", galleryHandler.List)
	mux.HandleFunc("GET /v1/gallery/{id}", galleryHandler.Get)

	// calendar image: public, like gallery.
	mux.HandleFunc("GET /v1/calendar/image", calendarHandler.Image)

	// articles: public, like temples. "/top" is a literal segment, so ServeMux
	// prefers it over "/{id}".
	mux.HandleFunc("GET /v1/articles/top", articleHandler.Top)
	mux.HandleFunc("GET /v1/articles", articleHandler.List)
	mux.HandleFunc("GET /v1/articles/{id}", articleHandler.Get)

	// dharma talks: public, like articles.
	mux.HandleFunc("GET /v1/dharma-talks", dharmaTalkHandler.List)
	mux.HandleFunc("GET /v1/dharma-talks/{id}", dharmaTalkHandler.Get)

	// sutras: public, like articles. /lines is a longer pattern than /{id}, so the two
	// do not collide; /v1/deities/{id}/sutras is a longer pattern than
	// /v1/deities/{id}, so the deity routes above keep answering exactly as before.
	mux.HandleFunc("GET /v1/sutras", sutraHandler.List)
	mux.HandleFunc("GET /v1/sutras/{id}", sutraHandler.Get)
	mux.HandleFunc("GET /v1/sutras/{id}/lines", sutraHandler.Lines)
	mux.HandleFunc("GET /v1/deities/{id}/sutras", sutraHandler.ListByDeity)

	// ⚠️ DEV ONLY: images are served from disk until there is a CDN. Enabled by
	// server.static_images_dir; a configured-but-missing folder stops the service now
	// instead of leaving every image_url answering 404.
	if cfg.Server.StaticImagesDir != "" {
		images, err := deliveryhttp.NewStaticImages(cfg.Server.StaticImagesDir)
		if err != nil {
			log.Fatalf("invalid server.static_images_dir: %v", err)
		}
		mux.Handle("GET /static/images/", http.StripPrefix("/static/images", images))
		logger.Info("serving static images", "dir", cfg.Server.StaticImagesDir, "route", "/static/images/")
	}

	addr := cfg.Server.Addr
	logger.Info("starting appService", "addr", addr)
	if err := http.ListenAndServe(addr, newHandler(mux)); err != nil { // #nosec G114 -- no timeouts configured yet, MVP
		log.Fatalf("server failed: %v", err)
	}
}

// newHandler builds the top-level handler every request passes through:
//
//	Recovery(CORS(Logger(WithAPIFallback(mux))))
//
// The order is load-bearing:
//   - Recovery is outermost so it catches a panic from any layer inside it
//     (CORS, Logger, the fallback, or a handler) and answers 500
//     INTERNAL_ERROR through the same envelope as everything else.
//   - CORS sits outside Logger so a preflight OPTIONS is answered and stops
//     there, before Logger runs — preflight never reaches the log lines.
//   - Logger is outside the fallback and mux, so a matched route and the
//     ServeMux 404/405 both get their Started/Completed lines.
//   - WithAPIFallback is innermost and only rewrites ServeMux's own 404/405
//     bodies into the utils.ApiResponse envelope; it never touches a real
//     handler's response.
//
// It is a separate function so cmd/main_test.go asserts the wiring main
// actually uses, not a chain reconstructed in a test.
func newHandler(mux *http.ServeMux) http.Handler {
	return httpmiddleware.Recovery(
		httpmiddleware.CORS(
			httpmiddleware.Logger(
				deliveryhttp.WithAPIFallback(mux),
			),
		),
	)
}
