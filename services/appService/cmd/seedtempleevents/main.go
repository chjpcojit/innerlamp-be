// Command seedtempleevents loads assets/data/temple-events.json — the events
// each temple holds (khoá tu, pháp hội, lễ…, one-off or recurring) — into the
// temple_events table.
//
// It is idempotent and never deletes: events are upserted by id in one
// transaction, so re-running after editing the JSON updates in place, an event
// absent from a newer file is left as it was, and an event back-office turned
// off stays off (is_active is never written). The file is parsed and verified
// (domain.TempleEventCatalogue.Verify) before the database is touched; a file
// that fails is refused whole. Every temple_id must already exist in temples
// (run the temple seed first). Settings come from the same config file and dev
// secret resolver appService uses (CONFIG_PATH, default config.dev.yaml;
// temple_event.timezone decides how all-day events are read) — ⚠️ DEV ONLY, same
// as cmd/main.go.
//
//	go run ./services/appService/cmd/seedtempleevents [-file assets/data/temple-events.json] [-dry-run]
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
	file := flag.String("file", "assets/data/temple-events.json", "temple events data file (JSON)")
	dryRun := flag.Bool("dry-run", false, "parse and verify the file, print the counts, and stop before touching the database")
	flag.Parse()

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = defaultConfigPath
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Step 1 — settings (also needed by -dry-run: the zone is part of verifying).
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
	if err := cfg.TempleEvent.Validate(); err != nil {
		log.Fatalf("invalid temple_event config: %v", err)
	}
	loc, err := time.LoadLocation(cfg.TempleEvent.Timezone)
	if err != nil {
		log.Fatalf("temple_event timezone: %v", err)
	}

	// Step 2 — read and decode the file strictly.
	src, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("read %s: %v", *file, err)
	}
	cat, rep, err := parseCatalogue(src)
	if err != nil {
		log.Fatalf("parse %s: %v", *file, err)
	}
	// Step 3 — verify it. A file that fails is refused whole.
	if err := cat.Verify(loc); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed verification: %v\n", *file, err)
		os.Exit(1)
	}
	report(*file, cat, rep)
	if *dryRun {
		log.Printf("dry run: %s is consistent; nothing was written", *file)
		return
	}

	// Step 4 — one transaction.
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

	start := time.Now()
	if err := repository.NewTempleEventRepository(db).ReplaceCatalogue(ctx, cat); err != nil {
		log.Fatalf("seed temple events: %v", err)
	}
	log.Printf("seeded %d temple events in %s", len(cat.Events), time.Since(start).Round(time.Millisecond))
}

// report prints what is about to be (or, for -dry-run, would be) loaded.
func report(file string, c domain.TempleEventCatalogue, rep parseReport) {
	byRecurrence := map[domain.TempleRecurrence]int{}
	temples := map[string]bool{}
	for _, e := range c.Events {
		byRecurrence[e.Recurrence]++
		temples[e.TempleID] = true
	}
	log.Printf("%s: %d events for %d temples (source %q)", file, len(c.Events), len(temples), c.Source)
	log.Printf("  by recurrence: none %d, weekly %d, lunar_monthly %d",
		byRecurrence[domain.RecurrenceNone], byRecurrence[domain.RecurrenceWeekly], byRecurrence[domain.RecurrenceLunarMonthly])
	log.Printf("  strings NFC-normalised: %d", rep.Normalized)
}
