// Command seedtemples loads assets/data/temple.json into the temples table.
//
// It is idempotent: rows are upserted by id in one transaction, so re-running
// after editing the JSON updates in place and never duplicates. Database
// settings come from the same config file and dev secret resolver appService
// uses (CONFIG_PATH, default config.dev.yaml) — ⚠️ DEV ONLY, same as cmd/main.go.
//
//	go run ./services/appService/cmd/seedtemples [-file assets/data/temple.json]
//
// Run `go run ./services/appService/cmd/seedadminunits` first: every temple
// carries province_code/ward_code, which are checked against that dataset and
// whose names are taken from it (docs/specs/features/temple, AC 34–36).
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
	file := flag.String("file", "assets/data/temple.json", "temple data file (JSON, /* */ comments allowed)")
	flag.Parse()

	src, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("read %s: %v", *file, err)
	}
	temples, err := parseTemples(src)
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

	// Step 4: check every (province_code, ward_code) against the administrative-units
	// dataset and take the unit names from it. Read-only; a bad file stops here.
	if err := resolveAddresses(ctx, repository.NewAdministrativeUnitRepository(db), temples); err != nil {
		log.Fatalf("resolve addresses in %s: %v", *file, err)
	}

	if err := repository.NewTempleRepository(db).Upsert(ctx, temples); err != nil {
		log.Fatalf("seed temples: %v", err)
	}
	log.Printf("seeded %d temples from %s", len(temples), *file)
}
