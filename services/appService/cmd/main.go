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

	pkgconfig "chjpcoj.com.innerlamp/pkg/config"
	"chjpcoj.com.innerlamp/pkg/database"
	_ "chjpcoj.com.innerlamp/pkg/database/postgresql"
	"chjpcoj.com.innerlamp/pkg/jwt"
	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/pkg/logging/formatter"
	"chjpcoj.com.innerlamp/pkg/mailer"
	"chjpcoj.com.innerlamp/pkg/storage"
	_ "chjpcoj.com.innerlamp/pkg/storage/gcs"

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

	userRepo := repository.NewUserRepository(db)
	registerUsecase := usecase.NewRegisterUsecase(userRepo, jwtService, logger)
	registerHandler := deliveryhttp.NewRegisterHandler(registerUsecase)
	loginUsecase := usecase.NewLoginUsecase(userRepo, jwtService, logger)
	loginHandler := deliveryhttp.NewLoginHandler(loginUsecase)

	// ⚠️ DEV ONLY: logs the reset email instead of sending it. Swap for a
	// real provider (SendGrid/SES/GCP equivalent) before production.
	devMailer := mailer.NewLogMailer(logger)
	passwordResetUsecase := usecase.NewPasswordResetUsecase(userRepo, devMailer, logger)
	forgotPasswordHandler := deliveryhttp.NewForgotPasswordHandler(passwordResetUsecase)
	resetPasswordHandler := deliveryhttp.NewResetPasswordHandler(passwordResetUsecase)

	profileUsecase := usecase.NewProfileUsecase(userRepo, logger)
	profileHandler := deliveryhttp.NewProfileHandler(profileUsecase)

	// ⚠️ DEV ONLY: Application Default Credentials — swap for a service
	// account / workload identity before deploying.
	storageConfig := &storage.Config{
		Backend: storage.Backend(cfg.Storage.Backend),
		GCS: &storage.GCSConfig{
			ProjectID:             cfg.Storage.ProjectID,
			UseDefaultCredentials: true,
			DefaultBucket:         cfg.Storage.DefaultBucket,
		},
	}
	storageClient, err := storage.New(storageConfig)
	if err != nil {
		log.Fatalf("failed to create storage client: %v", err)
	}
	if err := storageClient.Connect(connectCtx); err != nil {
		log.Fatalf("failed to connect to storage: %v", err)
	}
	defer func() { _ = storageClient.Close() }()

	avatarStorage := repository.NewGCSAvatarStorage(storageClient, cfg.Storage.DefaultBucket)
	avatarUsecase := usecase.NewAvatarUsecase(userRepo, avatarStorage, logger)
	avatarHandler := deliveryhttp.NewAvatarHandler(avatarUsecase)

	sutraRepo := repository.NewSutraRepository(db)
	listSutrasUsecase := usecase.NewListSutrasUsecase(sutraRepo, logger)
	sutrasHandler := deliveryhttp.NewSutrasHandler(listSutrasUsecase)
	getSutraUsecase := usecase.NewGetSutraUsecase(sutraRepo, logger)
	sutraDetailHandler := deliveryhttp.NewSutraDetailHandler(getSutraUsecase)

	bodhisattvaRepo := repository.NewBodhisattvaRepository(db)
	mantraRepo := repository.NewMantraRepository(db)

	listBodhisattvasUsecase := usecase.NewListBodhisattvasUsecase(bodhisattvaRepo, logger)
	bodhisattvasHandler := deliveryhttp.NewBodhisattvasHandler(listBodhisattvasUsecase)
	getBodhisattvaUsecase := usecase.NewGetBodhisattvaUsecase(bodhisattvaRepo, mantraRepo, logger)
	bodhisattvaDetailHandler := deliveryhttp.NewBodhisattvaDetailHandler(getBodhisattvaUsecase)

	listMantrasUsecase := usecase.NewListMantrasUsecase(mantraRepo, logger)
	mantrasHandler := deliveryhttp.NewMantrasHandler(listMantrasUsecase)
	getMantraUsecase := usecase.NewGetMantraUsecase(mantraRepo, bodhisattvaRepo, logger)
	mantraDetailHandler := deliveryhttp.NewMantraDetailHandler(getMantraUsecase)

	mux := http.NewServeMux()
	mux.Handle("/v1/auth/register", registerHandler)
	mux.Handle("/v1/auth/login", loginHandler)
	mux.Handle("/v1/auth/forgot-password", forgotPasswordHandler)
	mux.Handle("/v1/auth/reset-password", resetPasswordHandler)
	mux.Handle("/v1/users/me", profileHandler)
	mux.Handle("/v1/users/me/avatar", avatarHandler)
	mux.Handle("/v1/content/sutras", sutrasHandler)
	mux.Handle("GET /v1/content/sutras/{slug}", sutraDetailHandler)
	mux.Handle("/v1/content/mantras", mantrasHandler)
	mux.Handle("GET /v1/content/mantras/{slug}", mantraDetailHandler)
	mux.Handle("/v1/content/bodhisattvas", bodhisattvasHandler)
	mux.Handle("GET /v1/content/bodhisattvas/{slug}", bodhisattvaDetailHandler)

	addr := cfg.Server.Addr
	logger.Info("starting appService", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil { // #nosec G114 -- no timeouts configured yet, MVP
		log.Fatalf("server failed: %v", err)
	}
}
