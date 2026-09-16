---
name: golang-clean-arch-tdd
description: This skill should be used when implementing, extending, or reviewing a feature in a Go service that follows this repo's Clean Architecture layout (domain / usecase / repository / delivery, wired from a cmd composition root), with tests written alongside the code rather than bolted on afterward. Use it when the user asks to implement a use case or endpoint "theo TDD" / "với test đầy đủ", add a new feature to services/appService (or a future service with the same layout), review whether existing code respects the domain→usecase→repository/delivery dependency direction, or add missing tests to a usecase/repository/handler.
---

# Go Clean Architecture — TDD Implementation

## Purpose

Implement a feature end-to-end across all four layers — `domain`, `usecase`, `repository`, `delivery` — plus its wiring in the composition root, producing the implementation and its tests together, one layer at a time, inside-out. The Dependency Rule (inner layers never import outer layers) is not a style preference here — `domain` genuinely has zero non-stdlib imports in this repo today, and every layer above it depends on the one below only through an interface. Treat any violation of that as a bug, not a nitpick.

## Build order — always inside-out

Code the layers in this order. Each step only needs the interface the previous step already defined, never a not-yet-written concrete type:

1. **`domain`** — entity struct(s), the repository interface (port) the use case needs, use-case input/output structs, sentinel errors (`var ErrX = errors.New(...)`).
2. **Hand-write the mock** for any new/changed repository interface — see `references/mocking_convention.md`. This repo does not use gomock/mockery or `go:generate`; do not introduce either without being asked.
3. **`usecase`** — write the test first against the mock from step 2 (red), then the minimal usecase code to pass it (green), then refactor. The usecase must only import `domain` (plus reusable `pkg/*` infra abstractions like `pkg/jwt`, `pkg/logging` — never a concrete `repository` or `delivery` type).
4. **`repository`** — implement the `domain` interface for real (Postgres/GCS/...). Test this layer against real infrastructure (or the pattern already used by sibling `*_repository_test.go` files in the target service), not by re-mocking it — a mocked-DB test proves nothing about SQL correctness.
5. **`delivery/http`** — handler. Declare a small consumer-defined interface for the subset of the usecase it calls (see `references/layer_conventions.md`), so the handler can be unit-tested with a mock without importing the concrete usecase struct. Define a dedicated response struct — never serialize a `domain` entity directly.
6. **`cmd/main.go`** — composition root. Wire the real repository, real usecase, real handler together here, last. This is the only place allowed to know every concrete type.

Skip a step only when the feature genuinely doesn't touch that layer (e.g. a pure business-rule change confined to an existing usecase needs no new domain port).

## Per-layer rules

Full detail, including what each layer must NOT contain, is in `references/layer_conventions.md`. Load it before writing `domain` or `delivery` code — those two are where violations are easiest to introduce (a stray `json` tag on a `domain` struct, or a handler that calls a repository directly).

Quick reference:

| Layer | Depends on | Test style | Mocks what |
|---|---|---|---|
| `domain` | stdlib only (+ testify in `mock_*.go` files) | table-driven, no mocks | — |
| `usecase` | `domain` interfaces, reusable `pkg/*` | `t.Run` subtests + mocks | the `domain` repository interface(s) it consumes |
| `repository` | `domain` (implements it), `pkg/database`/`pkg/storage` | integration-style, real backend | nothing — this layer IS the real thing |
| `delivery/http` | `domain` types, a consumer-defined usecase interface | `t.Run` subtests + mock | the usecase interface it declares |
| `cmd` | everything (composition root) | none (wiring only) | — |

## Mocking convention

This repo hand-writes `testify.Mock`-based mocks **inside the same package as the interface** (e.g. `domain/mock_user_repository.go` implements `domain.UserRepository`), not in a separate `mocks/` package, and not via `gomock`/`mockery` codegen — there is no `go:generate` directive anywhere in `services/appService` or `pkg/`. When a repository interface gains a method, update its mock by hand in the same PR. Full pattern and a worked example: `references/mocking_convention.md`.

## Verification before calling a layer done

1. `go test ./services/<service>/... -run <Pkg>` for every package touched — must pass.
2. Run `scripts/check_dependency_rule.sh <path-to-service>` to catch an accidental inward-pointing import (`domain` importing `usecase`/`repository`/`delivery`, or `usecase` importing `repository`/`delivery`) before it's easy to forget about.
3. Re-read the acceptance criteria (from the user story / spec, if one exists) one by one and point to the specific test proving each — a gap here means a missing test, not an assumption that "it's probably covered."
4. Confirm no `domain` struct grew a `json`/`db` tag, and no handler response is a bare `domain` entity — see the "Entity vs response model" rule in `references/layer_conventions.md`.

## What this skill does NOT do

- Does not generate mocks via `mockery`/`mockgen` — this repo hand-writes them (see above).
- Does not invent a new folder layout — it follows `services/appService`'s existing one (`domain/usecase/repository/delivery/http/cmd`).
- Does not enforce strict single-test-before-any-code red/green/refactor at the letter — writing a unit and its test in close succession, in either order, is fine as long as no unit is considered finished until its test exists and passes.
- Does not decide production-readiness concerns (secrets, mailer, auth middleware wiring) — those are `cmd/main.go` composition-root gaps, tracked separately, not a layering issue this skill's checks catch.
