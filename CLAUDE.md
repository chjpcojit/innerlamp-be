# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

`innerlamp` (module `chjpcoj.com.innerlamp`, Go 1.26) is a shared Go backend framework: a collection of reusable infrastructure packages under `pkg/` (config, secrets, database, storage, JWT, session, mailer, HTTP client, rate limiting, logging, metrics, tracing, validation), built for containerized/serverless services (GCP Cloud Run, GKE, AWS Fargate).

`services/appService` is the one consuming service — a Clean Architecture app that currently implements US-001 (register: email + 6-digit code sent by email, ADR-0003) and US-002 (login / refresh / logout) end-to-end against Postgres. There is no GCS usage yet. `cmd/main.go` wiring is real but uses dev-only config/secrets/mailer (search `⚠️ DEV ONLY` / `⚠️ DEV CONFIG`) — not production-ready as-is (dev uses `otp.dev_fixed_code: 111111` and `mailer.provider: log`; `config.Validate()` refuses both outside `app.environment: dev`). Profile/avatar, password reset and the content user stories are still specs only, not code.

`docs/specs/features/onboarding/requirements.md` and `docs/specs/features/login/requirements.md` hold the identity specs (Vietnamese) — they describe email + email code (ADR-0003); registration v8.1 is built (2026-10-16), login is still phone-based until login v2 Phases 1–6; `docs/specs/features/journal/requirements.md` is a prospective product spec for the planned meditation/dharma-tracking app. `docs/knowledge/` is the post-implementation OKF reference bundle (e.g. `features/temple.md`; the register/login concepts were deleted 2026-10-15 and are rewritten after the email flow ships), distinct from `docs/specs/`.

`docs/product/<feature>/` is the cross-platform product spec layer (one spec per feature, shared by this backend, the website, iOS and Android; written by the `ba-writer` skill). It sits above `docs/specs/api/openapi.yaml` (contract) and `docs/specs/features/<feature>/` (backend platform spec) — see `docs/product/README.md`. Existing `SRS.md` files under `docs/specs/features/` are legacy until migrated.

`docs/adr/` holds Architecture Decision Records (written by the `adr-writer` skill). It is an optional step **before** `ba-writer`, only for decisions that are hard to reverse — infrastructure, a new or reshaped data model, offline sync, auth/security, cross-cutting conventions, technology choices. Product and platform specs cite ADRs by ID (`ADR-NNNN`) and never copy them; `ba-writer` stops and asks when a feature needs such a decision and no Accepted ADR covers it — see `docs/adr/README.md`.

## Known broken state — read before building

- **`go.sum` present.** Still run `go mod tidy` after touching `go.mod`/imports.
- **`base_project/pkg/...` import mismatch is fixed in code** — all `.go` files correctly import `chjpcoj.com.innerlamp/...`. Stale `base_project/pkg/...` references only remain in a few docs (`QUICK_START.md`, some subpackage `README.md`/`INDEX.md`), each already self-flagged as leftover template text — don't copy import lines from those docs.
- **`cmd/main.go` dev-only wiring is marked, not hidden.** Search `⚠️ DEV ONLY` / `⚠️ DEV CONFIG` (not `MOCK_DATA_MARKER`, which no longer appears in `main.go`) for the spots to swap before production: `appConfig.NewMockSecretGetterFromYAML` (plaintext `dev_secrets` from YAML instead of `pkg/secrets/gcp`/`aws`) and `pkg/mailer.LogMailer` (`mailer.provider: log`, logs the email incl. the code instead of sending it). The OTP is `crypto/rand` unless `otp.dev_fixed_code` is set (dev yaml sets `111111`).
- **JWT auth middleware (`pkg/jwt/middleware`, `RequireTokenType: "access"`) is wired to six routes** in `cmd/main.go`: `POST /v1/auth/logout` and the five `/v1/memorials` routes (sổ giỗ chạp; user-scoped, owner = the token's `user_id`). Register/login/refresh and the catalogue reads are public by design. There are no `/v1/users/me` routes in the tree. Memorial photo upload is not built (GCS) — `docs/tech-debt/memorial/TD-001`.
- Before relying on any bullet above, re-verify (`ls go.sum`, `grep -rl base_project/pkg --include=*.go .`, `grep -n users/me services/appService/cmd/main.go`) — this section drifts as the repo evolves and is exactly the kind of fact that goes stale silently.

## Commands

```bash
go mod tidy                       # fetch deps (required first — no go.sum checked in)
go build ./...                    # build everything
go test -short ./...              # run tests, skip cloud-integration tests needing real AWS/GCP creds
go test ./services/appService/e2e/ # black-box E2E: real binary + testcontainers Postgres (needs Docker; skipped under -short)
go test -v ./pkg/<package>/...    # run one package's tests, e.g. ./pkg/jwt/...
go vet ./...
```

There is a `Makefile` with local-stack helpers (`make help`, `make up`, `make run`, `make psql`; see `docs/development/local-stack.md` and `docs/development/database-cli.md`). There is no linter config or CI workflow in this repo.

## Architecture

There is no central architecture doc. Each `pkg/` package's `README.md` is the source of truth for that package, and `services/appService/<layer>/README.md` for the layers that have one (`config/`, `domain/`, `usecase/` do; `repository/`, `delivery/http/` and `cmd/` do not). Verify package/layer relationships from the real imports (`grep` for `"chjpcoj.com.innerlamp/pkg/..."`) rather than trusting any prose, including this file.

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

**Package relationships** (verified against actual imports). The only `pkg/` cross-imports are: `jwt`→`jwt/claims`; `session`→`jwt`; `mailer`→`logging`; `tracing`→`tracing/gcp`, `tracing/types`; `middleware`→`utils`; `jwt/middleware`→`jwt`, `jwt/claims`, `jwt/context`, `logging`, `utils`; `ratelimit/middleware`→`ratelimit`, `ratelimit/store`, `utils`. Every other `pkg/` root is independent. The old claims that used to live in this paragraph (`config`→`secrets`, `session`→`database`+`tracing`, `httpclient`→`ratelimit`+`logging`+`metrics`) were checked against actual imports and found false; don't restate them, and re-verify with `grep` before relying on this list.

**Cloud bias**: abstractions cover both AWS and GCP, but GCP is the primary target — several packages auto-detect GCP serverless environments via env vars (`K_SERVICE`, `KUBERNETES_SERVICE_HOST`) and GCS/GCP Secret Manager/Cloud Tracing default to Application Default Credentials rather than passed-in keys.

When adding a new driver/provider to any of these packages: implement the interface in a new subpackage, register it in `init()`, and don't block on network I/O in the constructor.

## Where to look

**Convention: every top-level folder that isn't a single-purpose leaf has its own `README.md`** describing what it is, what it contains, and pointers to the relevant user story/spec — `pkg/<name>/README.md` (some also have `QUICK_START.md`), `migrations/README.md`, `scripts/README.md`. Check the folder's own README before re-deriving usage patterns or assuming what's implemented; if a folder handling non-trivial logic lacks one, add it rather than working from inference. (There is no central index of the READMEs.)

Exception: `docs/specs/` and `docs/knowledge/` don't follow the per-folder pattern (spec/OKF docs, not code folders) — linked explicitly where relevant instead.

<!-- OPENWIKI:START -->

## OpenWiki

This repository has a generated `openwiki/` evidence index. It is optional just-in-time context, not required startup reading.

- Treat source code and tests as authoritative. A brief's unknowns and review items are verification gaps, not automatic requirements.
- Prefer the narrowest quiet validation that proves the changed behavior. Preserve complete failure output.

The scheduled OpenWiki GitHub Actions workflow refreshes the repository wiki. Do not hand-edit generated OpenWiki pages unless explicitly asked; prefer updating source code/docs and letting OpenWiki regenerate.

<!-- OPENWIKI:END -->
