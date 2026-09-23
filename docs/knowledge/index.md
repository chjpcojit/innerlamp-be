# InnerLamp Knowledge Bundle (OKF)

This is the InnerLamp OKF (Open Knowledge Format) bundle — durable, post-implementation reference
documentation for `chjpcoj.com.innerlamp`. It is the opposite pole from `docs/specs/`: specs under
`docs/specs/features/<feature>/` are mutable work orders written before/during implementation,
while a concept here documents code that already exists and (once `status: stable`) has been
exercised end-to-end. There is no central architecture doc — the old `docs/ARCHITECTURE.md` was
removed as stale. Read the code, the per-package `pkg/<name>/README.md` files and the layer
`README.md`s, and verify any relationship from the real imports.

## Sections

- [`features/`](features/) — one concept per user-facing feature/flow end to end (`type: Feature`).
- [`packages/`](packages/) — one concept per `pkg/` infrastructure package (`type: Package`).

**Planned, not yet populated:**

- `api/` — one concept per route's wire contract (`type: API Endpoint`).
- `entities/` — one concept per domain model (`type: Domain Entity`).
- `usecases/` — one concept per business action worth documenting on its own (`type: UseCase`).
- `architecture/` — one concept per cross-cutting, stable architecture pattern (`type: Architecture Pattern`).
- `migrations/` — one concept per SQL migration (`type: Migration`).

(`docs/knowledge/glossary/` also exists as an empty folder but is not owned by this skill.)

## Concepts

| Concept | Type | Status |
|---|---|---|
| [features/register.md](features/register.md) | Feature | draft (OTP `111111` + log-only SMS are dev placeholders; no cleanup job for the 24h-lived pending payload; US-001 ACs not fully audited end-to-end) |
| [features/login.md](features/login.md) | Feature | stable |
| [packages/middleware.md](packages/middleware.md) | Package | draft (appService chain and CORS preflight covered by unit tests only — never exercised on a running binary; gRPC interceptors untested and unused) |
