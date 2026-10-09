// Package assets embeds the curated seed data under assets/data so a service
// binary carries it and first-run seeding does not depend on the working
// directory (containers on Cloud Run / GKE / Fargate have none of the repo).
package assets

import "embed"

// BuddhistArticlesJSON is assets/data/buddhist_articles.json:
// {"articles":[{id,title,category,author,...}]}.
//
//go:embed data/buddhist_articles.json
var BuddhistArticlesJSON []byte

// DharmaTalksJSON is assets/data/dharma_talks.json:
// {"talks":[{id,title,speaker,mediaType,mediaUrl,...}]}.
//
//go:embed data/dharma_talks.json
var DharmaTalksJSON []byte

// SutrasFS holds assets/data/sutras: index.json plus one <id>.json per sutra
// (kinh / kệ / chú). Root it at data/sutras with fs.Sub before parsing:
//
//	sub, _ := fs.Sub(assets.SutrasFS, "data/sutras")
//	seeds, err := repository.ParseSutraSeed(sub)
//
// Only the *.json files are embedded (not the README), ~11 MB.
//
//go:embed data/sutras/*.json
var SutrasFS embed.FS

// CalendarManifestJSON is assets/images/home-calendar/manifest.json: one
// {file,author,license,license_url,source_url,...} per lunar-calendar photo.
//
//go:embed images/home-calendar/manifest.json
var CalendarManifestJSON []byte
