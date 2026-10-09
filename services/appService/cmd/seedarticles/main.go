// Command seedarticles inserts the articles in assets/data/buddhist_articles.json
// that are not in the articles table yet.
//
// The service only seeds an EMPTY table on start (usecase.ArticleUsecase.SeedIfEmpty),
// so articles added to the JSON later never reach an existing database. This
// command fills that gap. It is idempotent and non-destructive: rows are
// inserted by id via InsertMissing, existing rows are never modified — except
// with -fill-images, which only sets image_url on rows where it is still NULL
// (FillMissingImages), so articles seeded before their image was known get it. Database
// settings come from the same config file and dev secret resolver appService
// uses (CONFIG_PATH, default config.dev.yaml) — ⚠️ DEV ONLY, same as cmd/main.go.
//
//	go run ./services/appService/cmd/seedarticles [-file assets/data/buddhist_articles.json] [-fill-images]
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
	file := flag.String("file", "assets/data/buddhist_articles.json", "article data file (JSON)")
	fillImages := flag.Bool("fill-images", false, "also set image_url on existing rows where it is NULL")
	flag.Parse()

	src, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("read %s: %v", *file, err)
	}
	articles, err := repository.ParseArticleSeed(src)
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

	repo := repository.NewArticleRepository(db)
	inserted, err := repo.InsertMissing(ctx, articles)
	if err != nil {
		log.Fatalf("seed articles: %v", err)
	}
	log.Printf("inserted %d new articles (%d in %s)", inserted, len(articles), *file)

	if *fillImages {
		filled, err := repo.FillMissingImages(ctx, articles)
		if err != nil {
			log.Fatalf("fill article images: %v", err)
		}
		log.Printf("filled image_url on %d existing articles", filled)
	}
}
