// Command seeddharmatalks loads the dharma talks (pháp thoại) into the
// dharma_talks table.
//
// appService no longer seeds anything when it starts; this is the only way the
// table gets its data. It loads only when the table holds no row at all — talks
// flagged needs_review count, so a table whose every talk is under review is not
// mistaken for empty — and it does nothing (exit 0) otherwise, so it is safe to
// run twice and never overwrites an edited talk or brings back a deleted one. By
// default it reads the data embedded in the binary (assets/data/dharma_talks.json);
// -file reads a file on disk instead, which is how to try an edited copy before it
// is committed. Database settings come from the same config file and dev secret
// resolver appService uses (CONFIG_PATH, default config.dev.yaml) — ⚠️ DEV ONLY,
// same as cmd/main.go.
//
//	go run ./services/appService/cmd/seeddharmatalks [-file assets/data/dharma_talks.json]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"chjpcoj.com.innerlamp/assets"
	pkgconfig "chjpcoj.com.innerlamp/pkg/config"
	"chjpcoj.com.innerlamp/pkg/database"
	_ "chjpcoj.com.innerlamp/pkg/database/postgresql"

	appConfig "chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
	"chjpcoj.com.innerlamp/services/appService/repository"
	"chjpcoj.com.innerlamp/services/appService/usecase"
)

const (
	defaultConfigPath = "services/appService/config/config.dev.yaml"
	seedTimeout       = 2 * time.Minute
)

func main() {
	file := flag.String("file", "", "read the talks from this JSON file instead of the data embedded in the binary")
	flag.Parse()

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = defaultConfigPath
	}
	ctx, cancel := context.WithTimeout(context.Background(), seedTimeout)
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

	uc := usecase.NewDharmaTalkUsecase(repository.NewDharmaTalkRepository(db), cfg.DharmaTalk)
	n, err := uc.SeedIfEmpty(ctx, func() ([]domain.DharmaTalk, error) {
		src, err := source(*file)
		if err != nil {
			return nil, err
		}
		return repository.ParseDharmaTalkSeed(src)
	})
	if err != nil {
		log.Fatalf("seed dharma talks: %v", err)
	}
	log.Print(report(n, *file))
}

// source returns the JSON to parse: the embedded copy when file is empty,
// otherwise file on disk.
func source(file string) ([]byte, error) {
	if file == "" {
		return assets.DharmaTalksJSON, nil
	}
	return os.ReadFile(file)
}

// report words the outcome for the log.
func report(inserted int, file string) string {
	if inserted == 0 {
		return "dharma talks already seeded, skipping (nothing written)"
	}
	src := "the embedded data"
	if file != "" {
		src = file
	}
	return fmt.Sprintf("seeded %d dharma talks from %s", inserted, src)
}
