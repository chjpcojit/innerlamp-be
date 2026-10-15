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
	"fmt"
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
	"chjpcoj.com.innerlamp/pkg/mailer"
	httpmiddleware "chjpcoj.com.innerlamp/pkg/middleware"
	pkgstorage "chjpcoj.com.innerlamp/pkg/storage"
	_ "chjpcoj.com.innerlamp/pkg/storage/gcs"
	"chjpcoj.com.innerlamp/pkg/validation"

	"chjpcoj.com.innerlamp/assets"
	appConfig "chjpcoj.com.innerlamp/services/appService/config"
	deliveryhttp "chjpcoj.com.innerlamp/services/appService/delivery/http"
	"chjpcoj.com.innerlamp/services/appService/repository"
	avatarstorage "chjpcoj.com.innerlamp/services/appService/storage"
	"chjpcoj.com.innerlamp/services/appService/usecase"
)

const defaultConfigPath = "services/appService/config/config.dev.yaml"

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
	// The administrative-units lookups are public and read-only; without a valid per-IP limit
	// and cache lifetime they would be an unmetered open endpoint, so stop now.
	if err := cfg.AdministrativeUnits.Validate(); err != nil {
		log.Fatalf("invalid administrative_units config: %v", err)
	}

	// The events route needs a language to read text in and a cache lifetime to send;
	// both are required, so a missing calendar block stops the service now.
	if err := cfg.Calendar.Validate(); err != nil {
		log.Fatalf("invalid calendar config: %v", err)
	}

	// A missing temple_announcement block would mean page size 0 and no rate limit: stop now.
	if err := cfg.TempleAnnouncement.Validate(); err != nil {
		log.Fatalf("invalid temple_announcement config: %v", err)
	}
	// A missing temple_event block would give an unknown zone and a zero window: stop now.
	if err := cfg.TempleEvent.Validate(); err != nil {
		log.Fatalf("invalid temple_event config: %v", err)
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
	if err := cfg.Profile.Validate(); err != nil {
		log.Fatalf("invalid profile config: %v", err)
	}
	// The avatar routes sign URLs and copy objects through the storage client below; a
	// missing block must stop the service now, not at the first avatar.
	if err := cfg.Storage.Validate(); err != nil {
		log.Fatalf("invalid storage config: %v", err)
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

	// Object storage for avatars. Connect is called now so a missing credential or a bad
	// project fails the start, not the first upload.
	// ⚠️ DEV CONFIG: with no Application Default Credentials the GCS client refuses to build
	// unless STORAGE_EMULATOR_HOST is set (make run and the e2e harness set it). The emulator
	// does not verify signed URLs, so TD-001 still requires a check on a real bucket before
	// the first deploy that serves avatar routes.
	// When storage.emulator_host is set the GCS client is pointed at the emulator through the
	// library's own STORAGE_EMULATOR_HOST (plain HTTP, no credentials) and avatar URLs are
	// unsigned. Config wins over an inherited variable so a run is decided by one file.
	if cfg.Storage.EmulatorHost != "" {
		if err := os.Setenv("STORAGE_EMULATOR_HOST", cfg.Storage.EmulatorHost); err != nil {
			log.Fatalf("failed to point storage at the emulator: %v", err)
		}
		logger.Warn("⚠️ DEV ONLY: avatar storage runs against an emulator with unsigned URLs",
			"emulator_host", cfg.Storage.EmulatorHost)
	}
	storageClient, err := pkgstorage.New(&pkgstorage.Config{
		Backend: pkgstorage.Backend(cfg.Storage.Backend),
		Timeout: cfg.Storage.Timeout.Duration(),
		GCS: &pkgstorage.GCSConfig{
			ProjectID:             cfg.Storage.ProjectID,
			UseDefaultCredentials: true,
			DefaultBucket:         cfg.Storage.DefaultBucket,
		},
	})
	if err != nil {
		log.Fatalf("failed to create storage client: %v", err)
	}
	if err := storageClient.Connect(connectCtx); err != nil {
		log.Fatalf("failed to connect to storage: %v", err)
	}
	defer func() { _ = storageClient.Close() }()
	if cfg.Storage.EmulatorHost != "" {
		// A fresh emulator has no bucket. Not fatal: an emulator that is not up yet only breaks
		// the avatar routes, and the rest of the service must still start.
		if err := ensureBucket(storageClient, cfg.Profile.Bucket); err != nil {
			logger.Warn("could not prepare the emulator bucket; avatar routes will fail until it is up",
				"bucket", cfg.Profile.Bucket, "error", err.Error())
		}
	}

	// One store behind every rate limiter below; each limiter gets its own key scope in it.
	// docs/knowledge/packages/ratelimit.md lists every limit and why.
	rateLimitStore, rlPrefix, closeRateLimitStore, err := newRateLimitStore(connectCtx, cfg.RateLimitStore, logger)
	if err != nil {
		log.Fatalf("failed to set up rate limit store: %v", err)
	}
	defer func() { _ = closeRateLimitStore() }()
	rlScope := func(name string) deliveryhttp.RateLimitOption {
		return deliveryhttp.WithSharedStore(rateLimitStore, rlPrefix+name)
	}

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

	// ⚠️ DEV ONLY: mailer.provider "log" (pkg/mailer.LogMailer) writes every
	// email — codes included — to the service log instead of sending it.
	// AppConfig.Validate already refuses "log" outside app.environment "dev"
	// (onboarding AC21); a real driver ships once the sending domain exists.
	appMailer, err := newMailer(cfg.Mailer, logger)
	if err != nil {
		log.Fatalf("failed to create mailer: %v", err)
	}
	authMailer := repository.NewEmailAuthMailer(appMailer, cfg.Mailer.From)
	passwordHasher := usecase.NewBcryptPasswordHasher()

	registrationRepo := repository.NewRegistrationRepository(db)
	otpRepo := repository.NewOTPRepository(db)
	otpService := usecase.NewOTPService(otpRepo, logger, cfg.OTP)
	registerUsecase := usecase.NewRegisterUsecase(
		registrationRepo,
		otpService,
		authMailer,
		passwordHasher,
		jwtService,
		logger,
		cfg.JWT.RefreshTokenExpiration.Duration(),
		cfg.OTP.PendingRegistrationTTL.Duration(),
		cfg.Profile.DharmaNameMaxChars,
		cfg.Profile.FullNameMaxChars,
	)
	// accessTTL is what jwtService mints access tokens with; handlers report it as expires_in.
	accessTTL := deliveryhttp.WithAccessTokenTTL(cfg.JWT.AccessTokenExpiration.Duration())
	registerHandler := deliveryhttp.NewRegisterHandler(registerUsecase, validator, accessTTL)

	loginRepo := repository.NewLoginRepository(db)
	loginUsecase := usecase.NewLoginUsecase(
		loginRepo,
		passwordHasher,
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

	// Calendar events (sự kiện lịch): the catalogue of repeat rules lives in Postgres and is loaded by
	// cmd/seedcalendarevents; dates of the asked year are computed per request. Needs a login.
	calendarEventRepo := repository.NewCalendarEventRepository(db)
	calendarEventUsecase, err := usecase.NewCalendarEventUsecase(calendarEventRepo, cfg.Calendar, time.Now, logger)
	if err != nil {
		log.Fatalf("invalid calendar config: %v", err)
	}
	calendarEventHandler := deliveryhttp.NewCalendarEventHandler(calendarEventUsecase, cfg.Calendar.EventsCacheMaxAge.Duration())

	// Temple schedule (thời khoá tu tập): the standing practice routine of one temple, rows loaded by
	// cmd/seedtempleschedules. Needs a login. docs/specs/features/temple-schedule/.
	templeScheduleRepo := repository.NewTempleScheduleRepository(db)
	templeScheduleUsecase := usecase.NewTempleScheduleUsecase(templeRepo, templeScheduleRepo)
	templeScheduleHandler := deliveryhttp.NewTempleScheduleHandler(templeScheduleUsecase)

	// Temple events (sự kiện của chùa): rows loaded by cmd/seedtempleevents; the occurrences in the asked
	// window are computed per request. Needs a login. docs/specs/features/temple-event/.
	templeEventRepo := repository.NewTempleEventRepository(db)
	templeEventUsecase, err := usecase.NewTempleEventUsecase(templeRepo, templeEventRepo, cfg.TempleEvent, time.Now, logger)
	if err != nil {
		log.Fatalf("invalid temple_event config: %v", err)
	}
	templeEventHandler := deliveryhttp.NewTempleEventHandler(templeEventUsecase)

	// Temple announcements (thông báo của chùa tôi quy y): read + receive switch of the caller, temple
	// read from users.ordination_temple_id at call time. Its own per-account limiter (not the profile
	// one). docs/specs/features/temple-announcement/.
	templeAnnouncementRepo := repository.NewTempleAnnouncementRepository(db)
	announcementPrefRepo := repository.NewAnnouncementPreferenceRepository(db)
	templeAnnouncementUsecase, err := usecase.NewTempleAnnouncementUsecase(announcementPrefRepo, templeAnnouncementRepo, cfg.TempleAnnouncement, time.Now)
	if err != nil {
		log.Fatalf("invalid temple_announcement config: %v", err)
	}
	templeAnnouncementHandler := deliveryhttp.NewTempleAnnouncementHandler(templeAnnouncementUsecase)
	templeAnnouncementLimit, err := deliveryhttp.NewUserRateLimit(cfg.TempleAnnouncement.RateLimit.Max, cfg.TempleAnnouncement.RateLimit.Window.Duration(),
		rlScope("temple_announcement:user"))
	if err != nil {
		log.Fatalf("failed to build temple announcement rate limit: %v", err)
	}

	// Article catalogue (bài viết Phật pháp): read-only top / paged list / detail.
	// The service never seeds: data is loaded by cmd/seedarticles (make ensure-articles).
	articleRepo := repository.NewArticleRepository(db)
	articleUsecase := usecase.NewArticleUsecase(articleRepo, cfg.Article)
	articleHandler := deliveryhttp.NewArticleHandler(articleUsecase, imageBase)

	// Dharma talks (pháp thoại): read-only paged list / detail. Talks flagged needs_review
	// are stored but never served. Data is loaded by cmd/seeddharmatalks (make ensure-dharma-talks).
	dharmaTalkRepo := repository.NewDharmaTalkRepository(db)
	dharmaTalkUsecase := usecase.NewDharmaTalkUsecase(dharmaTalkRepo, cfg.DharmaTalk)
	dharmaTalkHandler := deliveryhttp.NewDharmaTalkHandler(dharmaTalkUsecase)

	// Sutra library (kinh / kệ / chú): read-only list / detail / lines, plus the sutras
	// linked to a deity. Data is loaded by cmd/seedsutras (make seed-sutras), which needs the
	// deities seeded first because every sutra links to deities.
	sutraRepo := repository.NewSutraRepository(db)
	sutraUsecase := usecase.NewSutraUsecase(sutraRepo, deityRepo, cfg.Sutra)
	sutraHandler := deliveryhttp.NewSutraHandler(sutraUsecase, imageBase)

	// memorials (sổ giỗ chạp): per-user data, so every route below is wrapped by authMiddleware.
	memorialRepo := repository.NewMemorialRepository(db)
	memorialUsecase := usecase.NewMemorialUsecase(memorialRepo, cfg.Memorial)
	memorialHandler := deliveryhttp.NewMemorialHandler(memorialUsecase)

	// profile (hồ sơ cá nhân): the token's user only, no route takes a user id. Avatars go
	// straight to storage by signed URL; the backend never receives the bytes.
	// docs/specs/features/profile/. The limiter is per account and one instance, so the five
	// routes share one budget.
	profileRepo := repository.NewProfileRepository(db)
	var avatarOpts []avatarstorage.Option
	if cfg.Storage.EmulatorHost != "" {
		avatarOpts = append(avatarOpts, avatarstorage.WithEmulator(cfg.Storage.EmulatorHost))
	}
	profileStorage := avatarstorage.NewAvatarStorage(storageClient, cfg.Profile.Bucket, avatarOpts...)
	profileUsecase := usecase.NewProfileUsecase(profileRepo, profileStorage, cfg.Profile, logger)
	profileHandler := deliveryhttp.NewProfileHandler(profileUsecase)
	profileLimit, err := deliveryhttp.NewUserRateLimit(cfg.Profile.RateLimit.Max, cfg.Profile.RateLimit.Window.Duration(),
		rlScope("profile:user"))
	if err != nil {
		log.Fatalf("failed to build profile rate limit: %v", err)
	}

	// genealogies (gia phả): the owner edits, an invited viewer only reads; every route is
	// wrapped by authMiddleware and the acting user is always the token's user_id claim.
	// docs/specs/features/genealogy/. Limits and the invitation lifetime come from cfg.Genealogy.
	genealogyRepo := repository.NewGenealogyRepository(db)
	genealogyMemberRepo := repository.NewGenealogyMemberRepository(db)
	genealogyViewerRepo := repository.NewGenealogyViewerRepository(db)
	genealogyInvitationRepo := repository.NewGenealogyInvitationRepository(db)
	userPhoneLookup := repository.NewUserPhoneLookup(db)
	genealogyHandler := deliveryhttp.NewGenealogyHandler(
		usecase.NewGenealogyUsecase(genealogyRepo, genealogyMemberRepo, userPhoneLookup, cfg.Genealogy))
	genealogyMemberHandler := deliveryhttp.NewGenealogyMemberHandler(
		usecase.NewGenealogyMemberUsecase(genealogyRepo, genealogyMemberRepo, cfg.Genealogy),
		usecase.NewGenealogyProfileUsecase(genealogyRepo, genealogyMemberRepo, userPhoneLookup))
	genealogyAccessHandler := deliveryhttp.NewGenealogyAccessHandler(
		usecase.NewGenealogyAccessUsecase(genealogyRepo, genealogyViewerRepo, genealogyInvitationRepo, cfg.Genealogy))
	genealogyInvitationHandler := deliveryhttp.NewGenealogyInvitationHandler(
		usecase.NewGenealogyInvitationUsecase(genealogyInvitationRepo, userPhoneLookup, cfg.Genealogy))

	// lamps (thắp đèn / vườn đèn): both routes need a signed-in user. The garden shows every
	// user's lit lamps, but the lighter is always the access token's user_id claim.
	lampRepo := repository.NewLampRepository(db)
	lampUsecase := usecase.NewLampUsecase(lampRepo, cfg.Lamp)
	lampHandler := deliveryhttp.NewLampHandler(lampUsecase)

	// temple suggestions: anonymous "suggest a temple" write. Review (list / create the
	// temple / reject) is backofficeService's. docs/specs/features/temple-suggestion/.
	// The administrative-units repository doubles as the ward resolver: a suggestion's
	// province_code/ward_code are checked against the same dataset the lookups serve.
	administrativeUnitRepo := repository.NewAdministrativeUnitRepository(db)
	templeSuggestionRepo := repository.NewTempleSuggestionRepository(db)
	templeSuggestionUsecase := usecase.NewTempleSuggestionUsecase(templeSuggestionRepo, administrativeUnitRepo)
	templeSuggestionHandler := deliveryhttp.NewTempleSuggestionHandler(templeSuggestionUsecase)
	templeSuggestionLimit, err := deliveryhttp.NewClientIPRateLimit(
		int64(cfg.TempleSuggestion.RateLimit.Max), cfg.TempleSuggestion.RateLimit.Window.Duration(),
		cfg.TempleSuggestion.RateLimit.TrustForwardedFor, rlScope("temple_suggestion:ip"))
	if err != nil {
		log.Fatalf("failed to build temple suggestion rate limit: %v", err)
	}

	// administrative units (tỉnh/huyện/xã, hệ cũ và mới + chuyển đổi): public read-only lookups,
	// also used while registering. docs/specs/features/administrative-units/. One limiter
	// instance, so all nine routes share one per-IP budget.
	administrativeUnitUsecase := usecase.NewAdministrativeUnitUsecase(administrativeUnitRepo)
	administrativeUnitHandler := deliveryhttp.NewAdministrativeUnitHandler(
		administrativeUnitUsecase, cfg.AdministrativeUnits.CacheMaxAge.Duration())
	administrativeUnitLimit, err := deliveryhttp.NewClientIPRateLimit(
		int64(cfg.AdministrativeUnits.RateLimit.Max), cfg.AdministrativeUnits.RateLimit.Window.Duration(),
		cfg.AdministrativeUnits.RateLimit.TrustForwardedFor, rlScope("administrative_units:ip"))
	if err != nil {
		log.Fatalf("failed to build administrative units rate limit: %v", err)
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
	mux.HandleFunc("POST /v1/auth/register/resend-otp", webAPIKey.WrapFunc(registerHandler.ResendOTP))
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

	// profile: five routes under /v1/users/me, each auth → per-account limit → no-store.
	registerProfileRoutes(mux, authMiddleware.AuthenticateFunc, profileLimit, noStore, profileHandler)
	// temple announcements: three routes under /v1/users/me, auth → their own per-account limit.
	registerTempleAnnouncementRoutes(mux, authMiddleware.AuthenticateFunc, templeAnnouncementLimit, templeAnnouncementHandler)

	// genealogies: private to the owner and the people they invited. The caller's role is
	// resolved from the database on every request (never from the token), and a caller with no
	// role gets the same 404 as a tree that does not exist.
	mux.HandleFunc("POST /v1/genealogies", authMiddleware.AuthenticateFunc(genealogyHandler.Create))
	mux.HandleFunc("GET /v1/genealogies", authMiddleware.AuthenticateFunc(genealogyHandler.List))
	mux.HandleFunc("GET /v1/genealogies/{id}", authMiddleware.AuthenticateFunc(genealogyHandler.Get))
	mux.HandleFunc("PUT /v1/genealogies/{id}", authMiddleware.AuthenticateFunc(genealogyHandler.Update))
	mux.HandleFunc("DELETE /v1/genealogies/{id}", authMiddleware.AuthenticateFunc(genealogyHandler.Delete))
	mux.HandleFunc("PUT /v1/genealogies/{id}/chronicle", authMiddleware.AuthenticateFunc(genealogyHandler.UpdateChronicle))
	mux.HandleFunc("GET /v1/genealogies/{id}/members", authMiddleware.AuthenticateFunc(genealogyMemberHandler.List))
	mux.HandleFunc("POST /v1/genealogies/{id}/members", authMiddleware.AuthenticateFunc(genealogyMemberHandler.Add))
	mux.HandleFunc("GET /v1/genealogies/{id}/members/{memberId}", authMiddleware.AuthenticateFunc(genealogyMemberHandler.Get))
	mux.HandleFunc("PUT /v1/genealogies/{id}/members/{memberId}", authMiddleware.AuthenticateFunc(genealogyMemberHandler.Update))
	mux.HandleFunc("DELETE /v1/genealogies/{id}/members/{memberId}", authMiddleware.AuthenticateFunc(genealogyMemberHandler.Delete))
	mux.HandleFunc("PUT /v1/genealogies/{id}/members/{memberId}/relation", authMiddleware.AuthenticateFunc(genealogyMemberHandler.Move))
	mux.HandleFunc("GET /v1/genealogies/{id}/members/{memberId}/profile", authMiddleware.AuthenticateFunc(genealogyMemberHandler.Profile))
	mux.HandleFunc("POST /v1/genealogies/{id}/invitations", authMiddleware.AuthenticateFunc(genealogyAccessHandler.Invite))
	mux.HandleFunc("GET /v1/genealogies/{id}/invitations", authMiddleware.AuthenticateFunc(genealogyAccessHandler.ListInvitations))
	mux.HandleFunc("DELETE /v1/genealogies/{id}/invitations/{invitationId}", authMiddleware.AuthenticateFunc(genealogyAccessHandler.CancelInvitation))
	mux.HandleFunc("GET /v1/genealogies/{id}/viewers", authMiddleware.AuthenticateFunc(genealogyAccessHandler.ListViewers))
	mux.HandleFunc("DELETE /v1/genealogies/{id}/viewers/{userId}", authMiddleware.AuthenticateFunc(genealogyAccessHandler.RemoveViewer))
	mux.HandleFunc("GET /v1/invitations", authMiddleware.AuthenticateFunc(genealogyInvitationHandler.ListMine))
	mux.HandleFunc("POST /v1/invitations/{id}/accept", authMiddleware.AuthenticateFunc(genealogyInvitationHandler.Accept))
	mux.HandleFunc("POST /v1/invitations/{id}/decline", authMiddleware.AuthenticateFunc(genealogyInvitationHandler.Decline))

	// lamps: no public read — a visitor sees neither the garden nor its counts.
	mux.HandleFunc("POST /v1/lamps", authMiddleware.AuthenticateFunc(lampHandler.Light))
	mux.HandleFunc("GET /v1/lamps", authMiddleware.AuthenticateFunc(lampHandler.Garden))

	// temple suggestions: public by design (the caller may have no account). webAPIKey
	// runs first so a wrong key is refused without spending the caller's rate budget;
	// the per-IP limit then guards the DB write.
	mux.HandleFunc("POST /v1/temple-suggestions", webAPIKey.WrapFunc(templeSuggestionLimit.WrapFunc(templeSuggestionHandler.Create)))

	// administrative units: public by design (registration has no account yet). webAPIKey first so
	// a wrong key is refused without spending the caller's budget, then the shared per-IP limit.
	// No JWT: there is nothing user-scoped here. "new" = two-level scheme since 2025-07-01,
	// "old" = three-level scheme before it.
	adminUnits := func(h http.HandlerFunc) http.HandlerFunc {
		return webAPIKey.WrapFunc(administrativeUnitLimit.WrapFunc(h))
	}
	mux.HandleFunc("GET /v1/administrative-units/new/provinces", adminUnits(administrativeUnitHandler.CurrentProvinces))
	mux.HandleFunc("GET /v1/administrative-units/new/provinces/{code}/wards", adminUnits(administrativeUnitHandler.CurrentWards))
	mux.HandleFunc("GET /v1/administrative-units/new/provinces/{code}/conversion", adminUnits(administrativeUnitHandler.ConvertCurrentProvince))
	mux.HandleFunc("GET /v1/administrative-units/new/wards/{code}/conversion", adminUnits(administrativeUnitHandler.ConvertCurrentWard))
	mux.HandleFunc("GET /v1/administrative-units/old/provinces", adminUnits(administrativeUnitHandler.LegacyProvinces))
	mux.HandleFunc("GET /v1/administrative-units/old/provinces/{code}/districts", adminUnits(administrativeUnitHandler.LegacyDistricts))
	mux.HandleFunc("GET /v1/administrative-units/old/provinces/{code}/conversion", adminUnits(administrativeUnitHandler.ConvertLegacyProvince))
	mux.HandleFunc("GET /v1/administrative-units/old/districts/{code}/wards", adminUnits(administrativeUnitHandler.LegacyWards))
	mux.HandleFunc("GET /v1/administrative-units/old/wards/{code}/conversion", adminUnits(administrativeUnitHandler.ConvertLegacyWard))

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

	// temple schedule: NOT public, unlike the four routes above — it sits under /v1/temples/{id}
	// but needs a valid access token (US-TSC-001, FR-009).
	mux.HandleFunc("GET /v1/temples/{id}/schedule", authMiddleware.AuthenticateFunc(templeScheduleHandler.Schedule))

	// temple events: also needs a valid access token (US-TEV-001, FR-012), same reason as /schedule.
	mux.HandleFunc("GET /v1/temples/{id}/events", authMiddleware.AuthenticateFunc(templeEventHandler.Events))

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

	// calendar events: needs a valid access token, like lamps and memorials.
	mux.HandleFunc("GET /v1/calendar/events", authMiddleware.AuthenticateFunc(calendarEventHandler.Events))

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

// newMailer builds the pkg/mailer driver mailer.provider names. "log" is the
// only driver today; anything else is a startup error rather than a silent
// fallback, so a typo cannot ship codes into the log in production.
func newMailer(cfg appConfig.MailerConfig, logger logging.Logger) (mailer.Mailer, error) {
	switch cfg.Provider {
	case appConfig.MailerProviderLog:
		return mailer.NewLogMailer(logger), nil
	default:
		return nil, fmt.Errorf("unknown mailer.provider %q (supported: %q)", cfg.Provider, appConfig.MailerProviderLog)
	}
}
