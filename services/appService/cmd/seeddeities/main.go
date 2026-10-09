// Command seeddeities loads assets/data/phat_botat.json into the deities table.
//
// It is idempotent: rows are upserted by id in one transaction, so re-running
// after editing the JSON updates in place and never duplicates. Database
// settings come from the same config file and dev secret resolver appService
// uses (CONFIG_PATH, default config.dev.yaml) — ⚠️ DEV ONLY, same as cmd/main.go.
//
//	go run ./services/appService/cmd/seeddeities [-file assets/data/phat_botat.json]
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	pkgconfig "chjpcoj.com.innerlamp/pkg/config"
	"chjpcoj.com.innerlamp/pkg/database"
	_ "chjpcoj.com.innerlamp/pkg/database/postgresql"

	appConfig "chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/repository"
)

const defaultConfigPath = "services/appService/config/config.dev.yaml"

func main() {
	file := flag.String("file", "assets/data/phat_botat.json", "deity data file (JSON)")
	flag.Parse()

	src, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("read %s: %v", *file, err)
	}
	deities, err := parseDeities(src)
	if err != nil {
		log.Fatalf("parse %s: %v", *file, err)
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
		log.Fatalf("load app config: %v", err)
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

	if err := repository.NewDeityRepository(db).Upsert(ctx, deities); err != nil {
		log.Fatalf("seed deities: %v", err)
	}
	log.Printf("seeded %d deities from %s", len(deities), *file)
}
