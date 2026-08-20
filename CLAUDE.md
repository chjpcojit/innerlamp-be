# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

`innerlamp` (module `chjpcoj.com.innerlamp`, Go 1.26) is a shared Go backend framework: a collection of reusable infrastructure packages under `pkg/` (config, secrets, database, storage, JWT, session, HTTP client, rate limiting, logging, metrics, tracing, validation), built for containerized/serverless services (GCP Cloud Run, GKE, AWS Fargate).

`services/appService` is the one consuming service — a Clean Architecture app implementing the InnerLamp onboarding user stories (register, login, forgot/reset password, profile, avatar) end-to-end against Postgres + GCS. `cmd/main.go` wiring is real but uses hardcoded dev config (search for `MOCK_DATA_MARKER`) — not production-ready as-is. See `services/appService/README.md` for layout and what's implemented.

`docs/specs/onboarding/onboarding.md` holds product user-stories (Vietnamese) for a planned meditation/dharma-tracking app ("InnerLamp") — identity/auth flows (register, login, JWT access+refresh tokens). Not implementation, just spec.

## Known broken state — read before building

- **`go.sum` now present** (previously missing — if you recall it not existing, that's stale, re-check rather than trust memory). Still run `go mod tidy` after touching `go.mod`/imports.
- **Import path mismatch across ~44 files.** `go.mod` declares module `chjpcoj.com.innerlamp`, but internal imports throughout `pkg/` reference `base_project/pkg/...` (e.g. `pkg/jwt/jwt.go`, `pkg/database/mysql/mysql.go`, `pkg/session/session.go`). This is a known leftover from cloning a `base_project` template — do not "fix" it opportunistically in an unrelated change; if asked to correct it, it requires a coordinated rename across every affected file, not a spot fix.
- Before relying on either bullet above, verify with `ls go.sum` / `grep -rl base_project/pkg pkg/` — this section drifts as the repo evolves and is exactly the kind of fact that goes stale silently.

## Commands

```bash
go mod tidy                       # fetch deps (required first — no go.sum checked in)
go build ./...                    # build everything
go test -short ./...              # run tests, skip cloud-integration tests needing real AWS/GCP creds
go test -v ./pkg/<package>/...    # run one package's tests, e.g. ./pkg/jwt/...
go vet ./...
```

There is no Makefile, linter config, or CI workflow in this repo.

## Architecture

**Full architecture map, verified package dependency graph, and links to every `README.md`: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).** Read that first when you need to know what a package does or how layers relate — it's kept in sync with actual imports, not restated here.

Every major package follows the same **provider-agnostic interface + factory/registry** pattern:

1. Package root defines the interface and a `Register(type, FactoryFunc)` / `New(config)` pair backed by a package-level map (e.g. `pkg/database/database.go` → `Database` interface, `DBType`, `Register`, `New`).
2. Concrete drivers live in subpackages (`pkg/database/mysql`, `pkg/database/postgresql`, `pkg/database/mongodb`) and self-register via `init()`:
   ```go
   func init() {
       database.Register(database.MySQL, NewMySQL)
   }
   ```
3. Consumers activate a driver with a blank import: `import _ "chjpcoj.com.innerlamp/pkg/database/mysql"`.

Same shape applies to `pkg/secrets` (env/aws/gcp), `pkg/storage` (s3/gcs), `pkg/metrics` (gcp), `pkg/tracing` (gcp).

**Connection lifecycle convention**: constructors (`NewClient`, `NewMySQL`, ...) are synchronous and only validate config — they do not open remote connections. Actual connections happen via an explicit `Connect(ctx)`/`Start(ctx)` call, so services fail fast at startup rather than mid-request. Clients are built for concurrent reuse across goroutines (`database/sql` pooling, `sync.Mutex`/`sync.RWMutex`/`sync.Pool`).

**Package relationships**: superseded by the verified import graph in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) §2.2 — the claims that used to live in this paragraph (`config`→`secrets`, `session`→`database`+`tracing`, `httpclient`→`ratelimit`+`logging`+`metrics`) were checked against actual imports and found false; don't restate them.

**Cloud bias**: abstractions cover both AWS and GCP, but GCP is the primary target — several packages auto-detect GCP serverless environments via env vars (`K_SERVICE`, `KUBERNETES_SERVICE_HOST`) and GCS/GCP Secret Manager/Cloud Tracing default to Application Default Credentials rather than passed-in keys.

When adding a new driver/provider to any of these packages: implement the interface in a new subpackage, register it in `init()`, and don't block on network I/O in the constructor.

## Where to look

**Convention: every top-level folder that isn't a single-purpose leaf has its own `README.md`** describing what it is, what it contains, and pointers to the relevant user story/spec — `pkg/<name>/README.md` (some also have `QUICK_START.md`), `services/appService/README.md`, `migrations/README.md`, `scripts/README.md`. This file intentionally does not link to them individually — [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) does that, grouped by function with the real dependency graph; new folders automatically fit the pattern without needing an edit here. Check the folder's own README before re-deriving usage patterns or assuming what's implemented; if a folder handling non-trivial logic lacks one, add it rather than working from inference.

Exception: `docs/specs/` and `docs/db/` don't follow the per-folder pattern (spec/ERD docs, not code folders) — linked explicitly where relevant instead.

<!-- OPENWIKI:START -->

## OpenWiki

This repository has a generated `openwiki/` evidence index. It is optional just-in-time context, not required startup reading.

- Treat source code and tests as authoritative. A brief's unknowns and review items are verification gaps, not automatic requirements.
- Prefer the narrowest quiet validation that proves the changed behavior. Preserve complete failure output.

The scheduled OpenWiki GitHub Actions workflow refreshes the repository wiki. Do not hand-edit generated OpenWiki pages unless explicitly asked; prefer updating source code/docs and letting OpenWiki regenerate.

<!-- OPENWIKI:END -->
