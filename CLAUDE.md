# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

`innerlamp` (module `chjpcoj.com.innerlamp`, Go 1.26) is a shared Go backend framework: a collection of reusable infrastructure packages under `pkg/` (config, secrets, database, storage, JWT, session, HTTP client, rate limiting, logging, metrics, tracing, validation), built for containerized/serverless services (GCP Cloud Run, GKE, AWS Fargate).

`services/appService` is the one consuming service — a Clean Architecture app implementing the InnerLamp onboarding user stories (US-001..004: register, login, forgot/reset password, profile, avatar) and content user stories (US-005..010: sutras, mantras, bodhisattvas) end-to-end against Postgres + GCS. `cmd/main.go` wiring is real but uses dev-only config/secrets/mailer (search `⚠️ DEV ONLY`) — not production-ready as-is. See [`services/appService/README.md`](services/appService/README.md) for layout and what's implemented.

`docs/specs/onboarding/onboarding.md` holds product user-stories (Vietnamese) for a planned meditation/dharma-tracking app ("InnerLamp") — identity/auth flows (register, login, JWT access+refresh tokens). Not implementation, just spec.

## Known broken state — read before building

- **`go.sum` present.** Still run `go mod tidy` after touching `go.mod`/imports.
- **`base_project/pkg/...` import mismatch is fixed in code** — all `.go` files correctly import `chjpcoj.com.innerlamp/...`. Stale `base_project/pkg/...` references only remain in a few docs (`QUICK_START.md`, some subpackage `README.md`/`INDEX.md`), each already self-flagged as leftover template text — don't copy import lines from those docs.
- **`cmd/main.go` dev-only wiring is marked, not hidden.** Search `⚠️ DEV ONLY` / `⚠️ DEV CONFIG` (not `MOCK_DATA_MARKER`, which no longer appears in `main.go`) for the spots to swap before production: `config.MockSecretGetter` (plaintext `dev_secrets` from YAML instead of `pkg/secrets/gcp`/`aws`) and `mailer.NewLogMailer` (logs instead of sending).
- **JWT auth middleware exists but isn't wired to `/v1/users/me` or `/v1/users/me/avatar`** in `cmd/main.go` — `pkg/jwt/middleware.NewHTTPAuthMiddleware(...).Authenticate(...)` is never wrapped around those routes, so those handlers currently accept requests without enforcing auth despite being documented as requiring it. See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) §4 for the full known-gaps list.
- Before relying on any bullet above, re-verify (`ls go.sum`, `grep -rl base_project/pkg --include=*.go .`, `grep -n users/me services/appService/cmd/main.go`) — this section drifts as the repo evolves and is exactly the kind of fact that goes stale silently.

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
