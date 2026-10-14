// Command seedcalendarevents loads assets/data/calendar-events.json — the
// Buddhist, Vietnamese and folk events of the calendar, each with its repeat
// rule, text and groups — into the calendar_* tables.
//
// It is idempotent and never deletes: rows are upserted in one transaction, so
// re-running after editing the JSON updates in place, an event absent from a
// newer file is left as it was, and an event back-office turned off stays off.
// The file is parsed and verified (domain.CalendarCatalogue.Verify) before the
// database is touched; a file that fails is refused whole with a report.
// Database settings come from the same config file and dev secret resolver
// appService uses (CONFIG_PATH, default config.dev.yaml) — ⚠️ DEV ONLY, same
// as cmd/main.go.
//
//	go run ./services/appService/cmd/seedcalendarevents [-file assets/data/calendar-events.json] [-dry-run]
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

// maxIssuesPrinted keeps a badly broken file from printing a screenful per
// event; the total is always reported.
const maxIssuesPrinted = 50

func main() {
	file := flag.String("file", "assets/data/calendar-events.json", "calendar events data file (JSON)")
	dryRun := flag.Bool("dry-run", false, "parse and verify the file, print the counts, and stop before touching the database")
	flag.Parse()

	// Step 1 — read the file.
	src, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("read %s: %v", *file, err)
	}
	// Step 2 — decode it strictly.
	cat, rep, err := parseCatalogue(src)
	if err != nil {
		log.Fatalf("parse %s: %v", *file, err)
	}
	// Step 2 (cont.) — verify it. A file that fails is refused whole.
	if issues := cat.Verify(); len(issues) > 0 {
		fmt.Fprintf(os.Stderr, "%s failed verification: %d issue(s)\n", *file, len(issues))
		for i, is := range issues {
			if i == maxIssuesPrinted {
				fmt.Fprintf(os.Stderr, "  … and %d more\n", len(issues)-maxIssuesPrinted)
				break
			}
			fmt.Fprintf(os.Stderr, "  %s\n", is)
		}
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

	// Step 3 — connect.
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

	// Step 4 — one transaction, upserts only.
	start := time.Now()
	if err := repository.NewCalendarEventRepository(db).ReplaceCatalogue(ctx, cat); err != nil {
		log.Fatalf("seed calendar events: %v", err)
	}
	// Step 5 — report.
	log.Printf("seeded %d events in %d groups (lang %s) in %s",
		len(cat.Events), len(cat.Groups), cat.Lang, time.Since(start).Round(time.Millisecond))
}

// report prints what is about to be (or, for -dry-run, would be) loaded.
func report(file string, c domain.CalendarCatalogue, rep parseReport) {
	byKind := map[domain.CalendarEventKind]int{}
	holidays := 0
	for _, e := range c.Events {
		byKind[e.Rule.Kind]++
		if e.IsPublicHoliday {
			holidays++
		}
	}
	log.Printf("%s: %d groups, %d events (lang %s)", file, len(c.Groups), len(c.Events), c.Lang)
	log.Printf("  by kind: solar %d, lunar %d, lunar_monthly %d, lunar_last_day %d, solar_term %d; public holidays %d",
		byKind[domain.KindSolar], byKind[domain.KindLunar], byKind[domain.KindLunarMonthly],
		byKind[domain.KindLunarLastDay], byKind[domain.KindSolarTerm], holidays)
	log.Printf("  strings NFC-normalised: %d", rep.Normalized)
}
