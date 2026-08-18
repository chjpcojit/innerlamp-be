# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

`innerlamp` (module `chjpcoj.com.innerlamp`, Go 1.26) is a shared Go backend framework: a collection of reusable infrastructure packages under `pkg/` (config, secrets, database, storage, JWT, session, HTTP client, rate limiting, logging, metrics, tracing, validation), built for containerized/serverless services (GCP Cloud Run, GKE, AWS Fargate).

`services/appService/cmd/main.go` is the one consuming service, currently a stub (`func main() {}`) — not yet wired up.

`docs/specs/onboarding/onboarding.md` holds product user-stories (Vietnamese) for a planned meditation/dharma-tracking app ("InnerLamp") — identity/auth flows (register, login, JWT access+refresh tokens). Not implementation, just spec.

## Known broken state — read before building

- **No `go.sum`, dependencies never fetched.** `go build ./...` fails everywhere on missing modules (AWS SDK, GCP storage, mongo-driver, jwt/v5, validator/v10, etc). Run `go mod tidy` (needs network) before expecting anything to compile.
- **Import path mismatch across ~44 files.** `go.mod` declares module `chjpcoj.com.innerlamp`, but internal imports throughout `pkg/` reference `base_project/pkg/...` (e.g. `pkg/jwt/jwt.go`, `pkg/database/mysql/mysql.go`, `pkg/session/session.go`). This is a known leftover from cloning a `base_project` template — do not "fix" it opportunistically in an unrelated change; if asked to correct it, it requires a coordinated rename across every affected file, not a spot fix.
- Given both issues, assume nothing currently compiles as-is until dependencies are fetched and (if in scope) import paths are addressed.

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

**Package relationships**: `pkg/config` loads from local file / S3 / GCS and is itself configured via `pkg/secrets`. `pkg/session` depends on `pkg/database` and `pkg/jwt`, and propagates trace context via `pkg/tracing`. `pkg/httpclient` applies `pkg/ratelimit`, logs via `pkg/logging`, and registers metrics via `pkg/metrics`.

**Cloud bias**: abstractions cover both AWS and GCP, but GCP is the primary target — several packages auto-detect GCP serverless environments via env vars (`K_SERVICE`, `KUBERNETES_SERVICE_HOST`) and GCS/GCP Secret Manager/Cloud Tracing default to Application Default Credentials rather than passed-in keys.

When adding a new driver/provider to any of these packages: implement the interface in a new subpackage, register it in `init()`, and don't block on network I/O in the constructor.

## Where to look

Each `pkg/<name>` has its own `README.md` (and some have `QUICK_START.md`) with usage examples — check there before re-deriving usage patterns.

<!-- OPENWIKI:START -->

## OpenWiki

This repository has a generated `openwiki/` evidence index. It is optional just-in-time context, not required startup reading.

- Treat source code and tests as authoritative. A brief's unknowns and review items are verification gaps, not automatic requirements.
- Prefer the narrowest quiet validation that proves the changed behavior. Preserve complete failure output.

The scheduled OpenWiki GitHub Actions workflow refreshes the repository wiki. Do not hand-edit generated OpenWiki pages unless explicitly asked; prefer updating source code/docs and letting OpenWiki regenerate.

<!-- OPENWIKI:END -->
