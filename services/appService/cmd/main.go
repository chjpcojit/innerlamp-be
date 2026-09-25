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
	"log"
	"net/http"
	"os"
	"time"

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

	appConfig "chjpcoj.com.innerlamp/services/appService/config"
	deliveryhttp "chjpcoj.com.innerlamp/services/appService/delivery/http"
	"chjpcoj.com.innerlamp/services/appService/repository"
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
	registerHandler := deliveryhttp.NewRegisterHandler(registerUsecase, validator)

	loginRepo := repository.NewLoginRepository(db)
	loginUsecase := usecase.NewLoginUsecase(
		loginRepo,
		jwtService,
		logger,
		cfg.JWT.RefreshTokenExpiration.Duration(),
		cfg.Login,
	)
	loginHandler := deliveryhttp.NewLoginHandler(loginUsecase)

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
	mux.HandleFunc("POST /v1/auth/register/request-otp", registerHandler.RequestOTP)
	mux.HandleFunc("POST /v1/auth/resend-otp", registerHandler.ResendOTP)
	mux.HandleFunc("POST /v1/auth/register/verify-otp", registerHandler.VerifyOTP)

	// US-002: login and refresh are public — the caller proves itself with a
	// password / refresh token in the body. logout is wrapped: it identifies the
	// session from the access token's "sid" claim.
	mux.HandleFunc("POST /v1/auth/login", loginHandler.Login)
	mux.HandleFunc("POST /v1/auth/refresh", loginHandler.Refresh)
	mux.HandleFunc("POST /v1/auth/logout", authMiddleware.AuthenticateFunc(loginHandler.Logout))

	// attestation: all three routes are public — the caller has no account yet.
	// They mint challenges and return a verdict; the per-request gate is the
	// (unwired) RequireAttestation middleware, not these routes.
	mux.HandleFunc("POST /v1/attestation/challenge", attestationHandler.Challenge)
	mux.HandleFunc("POST /v1/assertion/challenge", attestationHandler.AssertionChallenge)
	mux.HandleFunc("POST /v1/attestation/verify", attestationHandler.Verify)

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
