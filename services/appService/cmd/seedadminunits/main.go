// Command seedadminunits loads assets/data/administrative-units.json — the
// Vietnamese administrative units under the old (63-province) and new
// (34-province) schemes plus the ward-level mapping between them — into the
// addr_* tables.
//
// It is idempotent and never deletes: rows are upserted in one transaction, so
// re-running after editing the JSON updates in place, and a unit absent from a
// newer file is left as it was. The file is parsed and verified
// (domain.Dataset.Verify) before the database is touched; a file that fails is
// refused whole with a row-by-row report. Database settings come from the same
// config file and dev secret resolver appService uses (CONFIG_PATH, default
// config.dev.yaml) — ⚠️ DEV ONLY, same as cmd/main.go.
//
//	go run ./services/appService/cmd/seedadminunits [-file assets/data/administrative-units.json] [-dry-run]
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

// maxIssuesPrinted keeps a badly broken file from printing tens of thousands
// of lines; the total is always reported.
const maxIssuesPrinted = 50

func main() {
	file := flag.String("file", "assets/data/administrative-units.json", "administrative units data file (JSON)")
	dryRun := flag.Bool("dry-run", false, "parse and verify the file, print the counts, and stop before touching the database")
	flag.Parse()

	// Step 1 — read the file.
	src, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("read %s: %v", *file, err)
	}
	// Step 2 — decode it.
	ds, rep, err := parseDataset(src)
	if err != nil {
		log.Fatalf("parse %s: %v", *file, err)
	}
	// Step 3 — verify it. A file that fails is refused whole; nothing is written.
	if issues := ds.Verify(); len(issues) > 0 {
		fmt.Fprintf(os.Stderr, "%s failed integrity verification: %d issue(s)\n", *file, len(issues))
		for i, is := range issues {
			if i == maxIssuesPrinted {
				fmt.Fprintf(os.Stderr, "  … and %d more\n", len(issues)-maxIssuesPrinted)
				break
			}
			fmt.Fprintf(os.Stderr, "  %s\n", is)
		}
		os.Exit(1)
	}
	report(*file, ds, rep)
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

	// Steps 4–6 — one transaction, upserts only.
	start := time.Now()
	if err := repository.NewAdministrativeUnitRepository(db).ReplaceDataset(ctx, ds); err != nil {
		log.Fatalf("seed administrative units: %v", err)
	}
	// Step 7 — done.
	log.Printf("seeded administrative units, dataset %s, in %s", ds.Version, time.Since(start).Round(time.Millisecond))
}

// report prints what is about to be (or, for -dry-run, would be) loaded, so an
// operator can compare it with the counts the file declares and with what the
// previous dataset had.
func report(file string, ds domain.Dataset, rep parseReport) {
	c := ds.Counts()
	log.Printf("%s: dataset %s (%s)", file, ds.Version, ds.SourceNote)
	log.Printf("  current scheme: %d provinces, %d wards", c.CurrentProvinces, c.CurrentWards)
	log.Printf("  legacy scheme:  %d provinces, %d districts, %d wards", c.LegacyProvinces, c.LegacyDistricts, c.LegacyWards)
	log.Printf("  ward mappings:  %d (district mappings read, not stored: %d)", c.WardMappings, c.DistrictMappings)
	log.Printf("  already corrected in the file: %d; left out as unresolved: %d; strings NFC-normalised: %d",
		rep.Corrections, rep.Unresolved, rep.Normalized)
}
