// Command seedadmin creates a backoffice admin account.
//
// Admins have no self-registration; this is the only way to make the first one.
// The password is read from the ADMIN_PASSWORD env var (not a flag, so it stays
// out of shell history and `ps`). Database settings come from the same config
// file and dev secret resolver as cmd/main.go — ⚠️ DEV ONLY.
//
//	ADMIN_PASSWORD='...' go run ./services/backofficeService/cmd/seedadmin -email admin@example.com -name "Admin"
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	pkgconfig "chjpcoj.com.innerlamp/pkg/config"
	"chjpcoj.com.innerlamp/pkg/database"
	_ "chjpcoj.com.innerlamp/pkg/database/postgresql"

	appConfig "chjpcoj.com.innerlamp/services/backofficeService/config"
	"chjpcoj.com.innerlamp/services/backofficeService/repository"
	"chjpcoj.com.innerlamp/services/backofficeService/usecase"
)

const (
	defaultConfigPath = "services/backofficeService/config/config.dev.yaml"
	minPasswordLen    = 12
)

func main() {
	email := flag.String("email", "", "admin email (required)")
	name := flag.String("name", "", "display name (required)")
	flag.Parse()

	password := os.Getenv("ADMIN_PASSWORD")
	if *email == "" || *name == "" {
		log.Fatal("-email and -name are required")
	}
	if len(password) < minPasswordLen {
		log.Fatalf("ADMIN_PASSWORD must be set and at least %d characters", minPasswordLen)
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = defaultConfigPath
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	configSource := pkgconfig.LoadSource(configPath)
	rawConfig, err := pkgconfig.LoadYAML(ctx, configSource)
	if err != nil {
		log.Fatalf("load config %s: %v", configPath, err)
	}
	// ⚠️ DEV ONLY: see cmd/main.go.
	secretGetter, err := appConfig.NewMockSecretGetterFromYAML(rawConfig)
	if err != nil {
		log.Fatalf("load dev secrets: %v", err)
	}
	cfg, err := appConfig.Load(ctx, configSource, secretGetter)
	if err != nil {
		log.Fatalf("load backoffice config: %v", err)
	}

	db, err := database.New(&database.Config{
		Type: database.PostgreSQL, Host: cfg.Database.Host, Port: cfg.Database.Port,
		Username: cfg.Database.Username, Password: cfg.Database.Password,
		Database: cfg.Database.Database, SSLMode: cfg.Database.SSLMode,
	})
	if err != nil {
		log.Fatalf("create database client: %v", err)
	}
	if err := db.Connect(ctx); err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer func() { _ = db.Close() }()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	admin, err := repository.NewAdminRepository(db).Create(ctx, usecase.NormalizeEmail(*email), string(hash), *name)
	if err != nil {
		log.Fatalf("create admin: %v", err)
	}
	log.Printf("created admin id=%d email=%s", admin.ID, admin.Email)
}
