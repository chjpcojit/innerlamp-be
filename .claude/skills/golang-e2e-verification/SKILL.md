---
name: golang-e2e-verification
description: This skill should be used after a feature in a Go service with this repo's Clean Architecture layout (domain/usecase/repository/delivery, wired from a cmd composition root — currently services/appService) has finished TDD implementation, to verify it end-to-end before it is considered done. It audits whether every acceptance criterion (AC) in the feature's requirements.md (under docs/specs/features/) has a test that actually proves it, and builds black-box HTTP tests that run the real compiled binary against a real ephemeral Postgres (via testcontainers-go) to catch composition-root wiring bugs — a route registered without its middleware, a handler built with the wrong dependency — that layer-by-layer unit/mock tests structurally cannot see. Use it when the user asks to verify a feature end-to-end, audit AC/test coverage, "viết E2E test", check whether auth middleware or a route is actually wired, or before closing out a feature's tasks.md. Not for writing unit tests during implementation, and not for cloud storage (GCS/AWS) — both are out of scope for this skill.
---

# Go E2E Verification

## Purpose

`golang-clean-arch-tdd` writes a feature's unit tests *alongside* its code, one layer at a time, each layer tested against a mock of the layer below. That proves each layer is internally correct. It does not prove the layers are wired together correctly in `cmd/main.go` — the one place every mock is replaced with the real thing — because nothing in that skill's workflow ever runs the composition root as a black box.

This skill runs *after* implementation and does two things neither TDD unit tests nor a code read can:

1. **AC coverage audit** — for every acceptance criterion in the feature's `requirements.md`, confirm a specific test proves it, not "probably covered."
2. **Black-box HTTP E2E tests** — build the real binary, run it against a real ephemeral Postgres (testcontainers-go, no manual local DB needed), and hit it over HTTP exactly as a client would. This is what catches a route registered without its auth middleware, a handler wired to the wrong usecase, or a config field that silently zero-values — bugs that live in `cmd/main.go` itself, which per-layer mocked tests never execute.

## When NOT to use this skill

- **Repository-layer SQL correctness tests** stay with `golang-clean-arch-tdd` — `services/<service>/repository/*_test.go` already tests real Postgres via `TEST_DB_*` env vars against a manually-run local instance. Do not touch or duplicate that pattern; this skill's testcontainers setup is for the E2E harness only.
- **Writing new unit tests while implementing a feature** — that's `golang-clean-arch-tdd`.
- **Real cloud storage (GCS/AWS)** — explicitly out of scope for now. Only Postgres is exercised, via testcontainers. Do not add a real-GCS E2E test under this skill without the user asking for that scope to expand.
- **Refactoring `cmd/main.go` to make it more testable** — this skill tests the binary as-is, black-box, precisely so the thing under test is what actually ships. Do not extract a `NewRouter`/`BuildServer` function to make wiring easier to unit-test; that would test a re-implementation of the wiring, not the wiring itself.

## Step 1 — AC coverage audit

Read `docs/specs/features/<feature>/requirements.md`, list every AC id. For each, search the feature's existing tests (`usecase`, `repository`, `delivery`, and any prior `e2e` tests) for one that specifically exercises it. A gap is real when no test names or exercises that AC — not when it merely seems adjacent to one that does. Full process, search strategy, and the gap-table format: `references/ac_coverage_audit.md`.

## Step 2 — decide what kind of gap it is

Not every gap belongs to this skill. Before writing an E2E test, classify the gap:

- **Belongs in a unit test at the right layer** (a usecase branch, a repository query variant) — flag it back to `golang-clean-arch-tdd`'s territory rather than papering over it with an E2E test that's slower and less precise about which layer is wrong.
- **Belongs in an E2E test** — the gap is about wiring or cross-call behavior: whether a route requires auth, whether the right handler is mounted at the right path, whether config actually reaches the running server, or a flow spanning multiple HTTP calls in sequence (e.g. request-otp → verify-otp) where real DB state carried between calls matters.

## Step 3 — set up or reuse the E2E harness

Check whether `services/<service>/e2e/harness.go` already exists.

- **Exists** — reuse it as-is. Do not create a second harness or duplicate its container/binary-build logic.
- **Doesn't exist** — scaffold it from `assets/e2e_harness.go.template`, adjusting the module/service path for the target service. Full harness design (binary build, testcontainers Postgres, temp config generation, readiness polling, cleanup, skip rules) is in `references/e2e_binary_harness.md` — read it before scaffolding, the template alone omits the reasoning behind each part.

Add the dependency once, the first time a service gets an E2E harness:

```bash
go get github.com/testcontainers/testcontainers-go github.com/testcontainers/testcontainers-go/modules/postgres
go mod tidy
```

## Step 4 — write the E2E test

Table-driven, one scenario per `t.Run`, using `newE2EServer(t)` (same `e2e` package as the harness) to get a `baseURL` + `*http.Client` and driving it with plain HTTP calls — never an internal call into the handler, that would stop being black-box. For a wiring gap specifically, write the negative case explicitly: call the protected endpoint with no `Authorization` header and assert the status the spec requires (401), not just the happy path with a valid token. A route that "works when you send a token" and a route that "rejects when you don't" are two different assertions; only the second one catches a forgotten `Authenticate(...)` wrap.

Skip convention: same as the repository-layer tests — skip under `testing.Short()`, and skip (not fail) when Docker isn't reachable, so `go test -short ./...` stays green with nothing running and a plain `go test ./...` degrades gracefully on a machine without Docker instead of hard-failing the whole suite.

## Step 5 — run and report

```bash
go test ./services/<service>/e2e/...
```

Report back: the AC gap table from Step 1 (what's covered, what's newly covered, what's still a gap and belongs to a different layer), and which wiring bugs, if any, the new E2E tests caught. If a real wiring bug turns up in `cmd/main.go` (missing middleware, wrong handler wired), say so explicitly and ask before fixing it if the fix isn't obviously a one-line correction — this repo's culture (see `golang-clean-arch-tdd`) is to flag an ambiguous production-code decision rather than quietly picking a side.

**Persist the report.** Prepend a new dated entry to `docs/specs/features/<feature>/test-coverage-report.md` (create it from `assets/test_coverage_report.md.template` if it doesn't exist yet) — newest run first, same convention as `spec-writer`'s `changelog.md`. This file lives beside `requirements.md` but sits outside the requirements→design→tasks cascade `spec-writer` owns: it's an audit log, not a spec, so it needs no version bump and no changelog entry of its own. Do not skip this step even when reporting only in chat feels sufficient — the value of the audit is comparing this run's gap table against the last one, which is impossible if nothing was saved.

## What this skill does NOT do

- Does not test real GCS/AWS — Postgres only, via testcontainers, for now.
- Does not re-test repository-layer SQL correctness — `golang-clean-arch-tdd` already owns that, against a manually-run local Postgres.
- Does not restructure `cmd/main.go` for testability — it black-box tests the binary exactly as it ships.
- Does not generate mocks or write per-layer unit tests — that's `golang-clean-arch-tdd`.
