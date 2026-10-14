// Command seedtempleschedules loads assets/data/temple-schedules.json — the
// standing practice schedule of temples (sessions with weekdays, instructor,
// location, preparation and running-order steps) — into the temple_schedules
// and temple_schedule_steps tables.
//
// It is idempotent and never deletes a session: sessions are upserted by id in
// one transaction, the steps of every session in the file are replaced by the
// file's, and is_active is never written, so a session back-office turned off
// stays off. The file is parsed and verified (domain.TempleScheduleCatalogue
// .Verify) before the database is touched; a file that fails is refused whole.
// Every temple_id must already exist in temples (run the temple seed first).
// Database settings come from the same config file and dev secret resolver
// appService uses (CONFIG_PATH, default config.dev.yaml) — ⚠️ DEV ONLY, same
// as cmd/main.go.
//
//	go run ./services/appService/cmd/seedtempleschedules [-file assets/data/temple-schedules.json] [-dry-run]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	pkgconfig "chjpcoj.com.innerlamp/pkg/config"
	"chjpcoj.com.innerlamp/pkg/database"
	_ "chjpcoj.com.innerlamp/pkg/database/postgresql"

	appConfig "chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
	"chjpcoj.com.innerlamp/services/appService/repository"
)

const defaultConfigPath = "services/appService/config/config.dev.yaml"

func main() {
	file := flag.String("file", "assets/data/temple-schedules.json", "temple schedules data file (JSON)")
	dryRun := flag.Bool("dry-run", false, "parse and verify the file, print the counts, and stop before touching the database")
	flag.Parse()

	// Step 1 — read and decode the file strictly.
	src, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("read %s: %v", *file, err)
	}
	cat, rep, err := parseCatalogue(src)
	if err != nil {
		log.Fatalf("parse %s: %v", *file, err)
	}
	// Step 2 — verify it. A file that fails is refused whole.
	if err := cat.Verify(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed verification: %v\n", *file, err)
		os.Exit(1)
	}
	report(*file, cat, rep)
	if *dryRun {
		log.Printf("dry run: %s is consistent; nothing was written", *file)
		return
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = defaultConfigPath
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	configSource := pkgconfig.LoadSource(configPath)
	rawConfig, err := pkgconfig.LoadYAML(ctx, configSource)
	if err != nil {
		log.Fatalf("load config %s: %v", configPath, err)
	}
	// ⚠️ DEV ONLY: see cmd/main.go (docs/tech-debt/config/TD-001).
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

	// Step 3 — one transaction.
	start := time.Now()
	if err := repository.NewTempleScheduleRepository(db).ReplaceCatalogue(ctx, cat); err != nil {
		log.Fatalf("seed temple schedules: %v", err)
	}
	log.Printf("seeded %d schedule items (%d steps) in %s",
		len(cat.Items), countSteps(cat), time.Since(start).Round(time.Millisecond))
}

func countSteps(c domain.TempleScheduleCatalogue) int {
	n := 0
	for _, it := range c.Items {
		n += len(it.Steps)
	}
	return n
}

// report prints what is about to be (or, for -dry-run, would be) loaded.
func report(file string, c domain.TempleScheduleCatalogue, rep parseReport) {
	temples := map[string]bool{}
	for _, it := range c.Items {
		temples[it.TempleID] = true
	}
	log.Printf("%s: %d items for %d temples, %d steps (source %q)",
		file, len(c.Items), len(temples), countSteps(c), c.Source)
	log.Printf("  strings NFC-normalised: %d", rep.Normalized)
}
