---
name: golang-clean-arch-tdd
description: Implement a feature in a Go service that follows Clean Architecture (domain / usecase / repository / delivery layers), driven by a markdown spec doc and backed by full unit + integration test coverage using gomock/mockery-style mocks. Use this whenever the user asks to implement, add, or extend a feature in a Go backend that follows Clean Architecture, references a spec file (e.g. something in /docs), or asks for a feature to be built "with tests" / "with full coverage" / "theo TDD" in a Go codebase. Also use when the user asks to review whether an existing Go feature respects layer boundaries and dependency direction, or asks to add missing tests to an existing usecase/repository.
---

# Go Clean Architecture — Spec-Driven TDD Implementation

## Purpose

This skill implements a single feature (or a well-scoped slice of one) in a Go service that follows Clean Architecture, starting from a written spec, and produces both the implementation and its test suite together. The goal is not just "code that works" — it's code where every layer's responsibility is testable in isolation, and where the dependency direction (domain → usecase → repository/delivery, never the reverse) is never violated, because that's what makes the architecture worth having.

Don't treat this as "write code, then bolt tests on at the end." Write the test for a unit of behavior right alongside the code that implements it, one layer at a time, so that by the time a layer is "done" its tests already prove it does what the spec says — not what you assume it does.

## Step 1 — Read the spec before touching code

The spec is almost always a markdown file in the repo (commonly under `/docs`). Read it fully before writing anything. Extract:

- The business rule(s) in plain terms — what should happen, and what should explicitly NOT happen (edge cases, validation rules, error conditions the spec mentions).
- Any acceptance criteria or examples given — these become your first test cases.
- Which existing domain entities / usecases / repositories this touches, versus what's genuinely new.

If the spec is ambiguous or missing an important detail (e.g., what error to return when a record isn't found, whether an operation should be idempotent), don't guess silently — surface the ambiguity to the user before implementing. A wrong assumption here propagates into every layer.

If the user pastes the requirement directly into chat instead of pointing at a file, treat that message as the spec and apply the same extraction.

## Step 2 — Map the change onto the four layers

This codebase's Clean Architecture uses four layers, each with a single direction of dependency:

```
domain  <--  usecase  <--  repository (implementation)
                <--  delivery
```

- **domain**: entities, value objects, domain errors, and the *interfaces* usecases depend on — repository interfaces are defined here, in `domain`, not in `usecase`. `repository` implements these interfaces; `usecase` only ever imports the interface from `domain`, never a concrete repository struct. No dependency on any other layer, no framework imports, no I/O.
- **usecase**: application business logic. Orchestrates domain entities and calls repository interfaces (never concrete implementations). This is where the feature's actual behavior lives, and where most of the unit test value is.
- **repository**: concrete implementation of the interfaces usecase depends on — DB, cache, external API clients. Talks to real infrastructure.
- **delivery**: the entrypoint — HTTP handler, gRPC handler, whatever calls into usecase. Thin: parse/validate input, call usecase, map output/errors to the transport format.

Before writing code, decide explicitly which layers this feature touches. A common mistake is putting business logic in delivery or repository "because it's convenient" — if you catch yourself doing that, stop and move it to usecase.

## Step 3 — Implement and test layer by layer

Work domain outward: domain → usecase → repository → delivery. For each layer:

1. Write the unit (function/method) and its test together — it's fine to iterate between the two rather than strictly writing the test first, but don't consider a unit finished until its test exists and passes.
2. Mock the layer below using the existing gomock/mockery-generated mocks in the repo (look for a `mocks/` directory or `//go:generate mockery` / `//go:generate mockgen` directives near the interface). **Do not generate new mocks yourself** — if a new interface needs a mock that doesn't exist yet, tell the user which interface needs one and let them run mockery/mockgen; use whatever mock stubbing you need in the meantime or ask before proceeding.
3. Use table-driven tests (`t.Run` subtests) as the default style for usecase and domain logic — it's the idiomatic way to cover multiple scenarios per unit without duplicating setup, and it matches most existing Go codebases' conventions. Check a couple of existing `_test.go` files in the repo first and match their exact style (naming, assertion style, table field names) rather than inventing a new one.
4. For every unit under test, cover at minimum: the happy path, each validation/error condition mentioned in the spec, and boundary conditions implied by the domain rule (empty input, nil, zero values, already-exists, not-found).

### Layer-specific testing notes

- **domain**: pure unit tests, no mocks needed — these are the cheapest and most valuable tests to get right.
- **usecase**: unit tests with the repository interface mocked. This is where most of the business-rule coverage should live, because usecase is where the business rule actually is.
- **repository**: this layer benefits from integration tests against a real dependency (e.g. testcontainers for the DB) rather than mocks, since its whole job is talking to real infrastructure correctly — a mocked-DB test here doesn't prove much. Check whether the repo already has a testcontainer/integration-test setup (often behind a build tag like `//go:build integration` or a separate `_test` package) and follow that pattern; don't introduce a new integration-test harness without flagging it to the user first.
- **delivery**: unit test the handler with the usecase mocked — assert request parsing/validation and status-code/error-mapping behavior, not business logic (that's already covered at the usecase layer).

## Step 4 — Verify before calling it done

Run the test suite for the packages you touched (`go test ./... -run <package>` or equivalent) and confirm everything passes. Then do a quick self-check against the spec: for each acceptance criterion or rule you extracted in Step 1, can you point to the specific test that proves it? If not, that's a gap — add the missing test rather than assuming the existing ones cover it implicitly.

Also sanity-check the dependency direction one more time: does domain import anything from usecase/repository/delivery? Does usecase import a concrete repository struct instead of the interface defined in `domain`? Either of those is a layering violation worth flagging and fixing even if it "works."

## What this skill does NOT do

- It does not run `mockery generate` / `mockgen` on its own — new mocks are the user's call.
- It does not decide the project's overall folder/package layout from scratch — it follows the existing convention in the repo it's working in.
- It is not a strict red-green-refactor enforcer — writing code and its test close together, in either order, is fine as long as coverage ends up complete.
