// ⚠️ DEV CONFIG - NOT FOR PRODUCTION
// Config loads from config/config.dev.yaml and secrets resolve through the
// plaintext MockSecretGetter, exactly like appService. Swap for pkg/secrets/gcp
// (or aws) and a real CONFIG_PATH before deploying.
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
	jwtmiddleware "chjpcoj.com.innerlamp/pkg/jwt/middleware"
	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/pkg/logging/formatter"
	"chjpcoj.com.innerlamp/pkg/mailer"
	httpmiddleware "chjpcoj.com.innerlamp/pkg/middleware"
	"chjpcoj.com.innerlamp/pkg/utils"

	appConfig "chjpcoj.com.innerlamp/services/backofficeService/config"
	deliveryhttp "chjpcoj.com.innerlamp/services/backofficeService/delivery/http"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
	"chjpcoj.com.innerlamp/services/backofficeService/repository"
	"chjpcoj.com.innerlamp/services/backofficeService/usecase"
)

const defaultConfigPath = "services/backofficeService/config/config.dev.yaml"

func main() {
	logger := logging.New(logging.Config{Level: logging.InfoLevel, Formatter: formatter.NewJSON()})

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
	// ⚠️ DEV ONLY: see the file header.
	secretGetter, err := appConfig.NewMockSecretGetterFromYAML(rawConfig)
	if err != nil {
		log.Fatalf("failed to load dev secrets from %s: %v", configPath, err)
	}
	cfg, err := appConfig.Load(connectCtx, configSource, secretGetter)
	if err != nil {
		log.Fatalf("failed to load backoffice config: %v", err)
	}

	if err := cfg.TempleSuggestion.Validate(); err != nil {
		log.Fatalf("invalid backoffice config: %v", err)
	}

	db, err := database.New(&database.Config{
		Type: database.PostgreSQL, Host: cfg.Database.Host, Port: cfg.Database.Port,
		Username: cfg.Database.Username, Password: cfg.Database.Password,
		Database: cfg.Database.Database, SSLMode: cfg.Database.SSLMode,
	})
	if err != nil {
		log.Fatalf("failed to create database client: %v", err)
	}
	if err := db.Connect(connectCtx); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer func() { _ = db.Close() }()

	jwtService, err := jwt.New(&jwt.Config{
		Algorithm:              jwt.Algorithm(cfg.JWT.Algorithm),
		SecretKey:              cfg.JWT.SecretKey,
		AccessTokenExpiration:  cfg.JWT.AccessTokenExpiration.Duration(),
		RefreshTokenExpiration: cfg.JWT.RefreshTokenExpiration.Duration(),
		Issuer:                 cfg.JWT.Issuer,
		Audience:               cfg.JWT.Audience,
	})
	if err != nil {
		log.Fatalf("failed to create jwt service: %v", err)
	}

	adminUsecase := usecase.NewAdminAuthUsecase(repository.NewAdminRepository(db), jwtService, logger)
	if b := cfg.BootstrapAdmin; b.Enabled {
		if err := adminUsecase.EnsureBootstrapAdmin(connectCtx, domain.BootstrapAdminInput{
			Email: b.Email, Password: b.Password, Name: b.Name,
		}); err != nil {
			log.Fatalf("failed to ensure bootstrap admin: %v", err)
		}
	}
	userUsecase := usecase.NewAdminUserUsecase(repository.NewAdminRepository(db),
		mailer.NewLogMailer(logger), // ⚠️ DEV ONLY: logs the invite instead of sending it; swap for a real provider
		cfg.Account.FixedTempPassword, cfg.Account.TempPasswordTTL.Duration(), logger)
	userHandler := deliveryhttp.NewAdminUserHandler(userUsecase)
	adminHandler := deliveryhttp.NewAdminAuthHandler(adminUsecase)

	// The same repository value is both the suggestion store and the ward
	// resolver: both read the one Postgres, and the resolver is just its
	// addr_new_* lookup.
	suggestionRepo := repository.NewTempleSuggestionRepository(db)
	suggestionUsecase := usecase.NewTempleSuggestionUsecase(suggestionRepo, suggestionRepo.(domain.WardResolver), cfg.TempleSuggestion.PageSize)
	suggestionHandler := deliveryhttp.NewTempleSuggestionHandler(suggestionUsecase, cfg.Server.PublicBaseURL)

	// Every /v1/admin/* route except login goes through this: access tokens
	// only, rejections in the shared envelope, then a role check per route.
	authMiddleware := jwtmiddleware.NewHTTPAuthMiddlewareWithOptions(jwtService, logger, jwtmiddleware.HTTPAuthOptions{
		RequireTokenType: "access",
		ErrorHandler:     deliveryhttp.WriteAuthError,
	})
	adminOnly := func(h http.HandlerFunc) http.HandlerFunc {
		return authMiddleware.AuthenticateFunc(deliveryhttp.RequireActiveSession(adminUsecase, deliveryhttp.RequireRole(domain.RoleAdmin, h)))
	}

	// change-password is the one route a must-change-password token may use.
	adminPasswordChange := func(h http.HandlerFunc) http.HandlerFunc {
		return authMiddleware.AuthenticateFunc(deliveryhttp.RequireActiveSession(adminUsecase, deliveryhttp.RequireStaffAllowPasswordChange(h)))
	}
	// Reviewers: admin and content_editor; viewer is refused (AC15).
	reviewer := func(h http.HandlerFunc) http.HandlerFunc {
		return authMiddleware.AuthenticateFunc(deliveryhttp.RequireActiveSession(adminUsecase,
			deliveryhttp.RequireAnyRole(h, domain.RoleAdmin, domain.RoleContentEditor)))
	}
	// Any backoffice account, whatever its role.
	staff := func(h http.HandlerFunc) http.HandlerFunc {
		return authMiddleware.AuthenticateFunc(deliveryhttp.RequireActiveSession(adminUsecase, deliveryhttp.RequireStaff(h)))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		utils.WriteSuccessResponse(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /v1/admin/auth/login", adminHandler.Login)
	mux.HandleFunc("POST /v1/admin/auth/change-password", adminPasswordChange(adminHandler.ChangePassword))
	mux.HandleFunc("GET /v1/admin/me", staff(adminHandler.Me))
	mux.HandleFunc("POST /v1/admin/users", adminOnly(userHandler.Create))
	mux.HandleFunc("POST /v1/admin/users/{id}/reset-password", adminOnly(userHandler.ResetPassword))
	mux.HandleFunc("GET /v1/admin/temple-suggestions", reviewer(suggestionHandler.List))
	mux.HandleFunc("GET /v1/admin/temple-suggestions/{id}", reviewer(suggestionHandler.Get))
	mux.HandleFunc("POST /v1/admin/temple-suggestions/{id}/create-temple", reviewer(suggestionHandler.CreateTemple))

	addr := cfg.Server.Addr
	logger.Info("starting backofficeService", "addr", addr)
	if err := http.ListenAndServe(addr, newHandler(mux)); err != nil { // #nosec G114 -- no timeouts configured yet, MVP
		log.Fatalf("server failed: %v", err)
	}
}

// newHandler builds the top-level chain Recovery(CORS(Logger(fallback(mux)))),
// in the same order and for the same reasons as appService's newHandler.
func newHandler(mux *http.ServeMux) http.Handler {
	return httpmiddleware.Recovery(
		httpmiddleware.CORS(
			httpmiddleware.Logger(
				deliveryhttp.WithAPIFallback(mux),
			),
		),
	)
}
