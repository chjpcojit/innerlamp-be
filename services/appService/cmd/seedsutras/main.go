// Command seedsutras loads the kinh / kệ / chú library into the sutras tables.
//
// appService does not seed on startup; this is the only way the library gets its
// data. It loads only when the sutras table is empty, in one transaction, and does
// nothing (exit 0) when the table already holds data, so it is safe to run twice
// and never overwrites or resurrects rows. By default it reads the data embedded in the
// binary (assets/data/sutras); -dir reads a directory on disk instead, which is
// how to try an edited copy of the data before it is committed.
//
// Every deity the data links to must already exist: run cmd/seeddeities first.
// Database settings come from the same config file and dev secret resolver
// appService uses (CONFIG_PATH, default config.dev.yaml) — ⚠️ DEV ONLY, same as
// cmd/main.go.
//
//	go run ./services/appService/cmd/seedsutras [-dir assets/data/sutras]
package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
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

	// seedTimeout bounds the whole run, including the one-transaction insert of
	// ~64,000 lines.
	seedTimeout = 2 * time.Minute
)

func main() {
	dir := flag.String("dir", "", "read the data from this directory instead of the data embedded in the binary")
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

	uc := usecase.NewSutraUsecase(repository.NewSutraRepository(db), repository.NewDeityRepository(db), cfg.Sutra)
	res, err := run(ctx, uc, *dir)
	if err != nil {
		log.Fatalf("seed sutras: %v", err)
	}
	log.Print(report(res, *dir))
}

// dataFS returns the data directory to parse: the embedded copy when dir is empty,
// otherwise dir on disk.
func dataFS(dir string) (fs.FS, error) {
	if dir == "" {
		return fs.Sub(assets.SutrasFS, "data/sutras")
	}
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}
	return os.DirFS(dir), nil
}

// run seeds through the use case, so the rules are exactly the startup ones: nothing
// is parsed or written when the table already has sutras.
func run(ctx context.Context, uc *usecase.SutraUsecase, dir string) (domain.SutraSeedResult, error) {
	return uc.SeedIfEmpty(ctx, func() ([]domain.SutraSeed, error) {
		fsys, err := dataFS(dir)
		if err != nil {
			return nil, err
		}
		return repository.ParseSutraSeed(fsys)
	})
}

// report words the outcome for the log.
func report(res domain.SutraSeedResult, dir string) string {
	if res.Sutras == 0 {
		return "sutras already seeded, skipping (nothing written)"
	}
	src := "the embedded data"
	if dir != "" {
		src = dir
	}
	return fmt.Sprintf("seeded %d sutras, %d translations, %d lines, %d deity links from %s",
		res.Sutras, res.Translations, res.Lines, res.DeityLinks, src)
}
