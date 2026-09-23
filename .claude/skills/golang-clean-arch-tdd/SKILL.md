---
name: golang-clean-arch-tdd
description: This skill should be used when implementing, extending, or reviewing a feature in a Go service that follows this repo's Clean Architecture layout (domain / usecase / repository / delivery, wired from a cmd composition root), with tests written alongside the code rather than bolted on afterward. Use it when the user asks to implement a use case or endpoint "theo TDD" / "với test đầy đủ", add a new feature to services/appService (or a future service with the same layout), review whether existing code respects the domain→usecase→repository/delivery dependency direction, or add missing tests to a usecase/repository/handler.
---

# Go Clean Architecture — TDD Implementation

## Purpose

Implement a feature end-to-end across all four layers — `domain`, `usecase`, `repository`, `delivery` — plus its wiring in the composition root, producing the implementation and its tests together, one layer at a time, inside-out. The Dependency Rule (inner layers never import outer layers) is not a style preference here — `domain` genuinely has zero non-stdlib imports in this repo today, and every layer above it depends on the one below only through an interface. Treat any violation of that as a bug, not a nitpick.

## Step 0 — read the spec folder before writing any code

A feature in this repo is specified before it is coded, and the spec is the source of truth for **what** to build. Never start from the user's prose alone. Find the feature's folder under `docs/specs/features/<feature>/` (`onboarding`, `content`, `journal`, `practice`...) and read, in this order:

1. **`tasks.md`** — the authoritative work list. Every item names one concrete file/behaviour and is already traced to a row in `design.md`'s layer table or a step in its flow. Work the checkboxes in order; they follow the same inside-out sequence as the build order below. Implement against this list rather than inventing scope.
2. **`design.md`** — the component map, the step-by-step flows, and the "Known gaps" / confirmed-decisions sections. When a task item's inline comment cites `design.md flow N Step M`, that step governs the behaviour; read it rather than guessing from the item text. Column names, sentinel names, ordering constraints and the "why"s are usually already decided here — re-deriving them from scratch is how a feature drifts from its spec.
3. **`requirements.md`** — the acceptance criteria (AC1, AC2, ...) the tests must cover. `tasks.md` references ACs by number; this is where they are written out.
4. **`changelog.md`** — only when something seems inconsistent, to see which decisions changed and when.

If the feature has no folder under `docs/specs/features/`, say so rather than inventing a task list — then either ask for the spec first or treat the user's description as the spec and say explicitly that no `tasks.md` backs the work.

### Tick `tasks.md` as items complete, not at the end

`tasks.md` is a living checklist, not a historical record. After an item is genuinely done — code written **and** its test passing — flip its `- [ ]` to `- [x]` in the same change. Rules:

- Tick only on verified completion. "The file exists" is not done; "the file exists, `go build` passes, and the test named in the item passes" is done.
- A partially-done item stays unticked. Don't tick a file-creation item because some of its listed methods exist.
- When an item's text no longer matches what was built (a method renamed, a decision reversed in `design.md`), fix the item text and note the change — don't silently tick a stale item.
- Do not tick items in the `Deferred` section; those are out of scope by definition.
- If you finish work with no corresponding item, add one rather than leaving the checklist incomplete.
- Update the header's `Last updated:` date and the `Status:` line when a phase completes.

Never tick an item you have not verified. An unchecked box is a truthful "not done yet"; a wrongly ticked box makes the whole checklist untrustworthy, which is worse than having no checklist.

**When `tasks.md` and this skill disagree, stop and flag it** rather than silently picking one, and do not tick the item until the disagreement is settled. A worked example: `tasks.md` Phase 2 once pinned the OTP policy as a `const` block inside `usecase/`, which contradicted the configuration rule below. That was flagged rather than quietly hardcoded or quietly moved; the user settled it in favour of `config.OTPConfig`, and only then was the item's text rewritten and ticked. The flag-don't-pick rule is what made the resolution visible instead of silent.

## Build order — always inside-out

Code the layers in this order. Each step only needs the interface the previous step already defined, never a not-yet-written concrete type:

1. **`domain`** — entity struct(s), the repository interface (port) the use case needs, use-case input/output structs, sentinel errors (`var ErrX = errors.New(...)`).
2. **Hand-write the mock** for any new/changed repository interface — see `references/mocking_convention.md`. This repo does not use gomock/mockery or `go:generate`; do not introduce either without being asked.
3. **`usecase`** — write the test first against the mock from step 2 (red), then the minimal usecase code to pass it (green), then refactor. The usecase must only import `domain` (plus reusable `pkg/*` infra abstractions like `pkg/jwt`, `pkg/logging` — never a concrete `repository` or `delivery` type).
4. **`repository`** — implement the `domain` interface for real (Postgres/GCS/...). Test this layer against real infrastructure (or the pattern already used by sibling `*_repository_test.go` files in the target service), not by re-mocking it — a mocked-DB test proves nothing about SQL correctness.
5. **`delivery/http`** — handler. Declare a small consumer-defined interface for the subset of the usecase it calls (see `references/layer_conventions.md`), so the handler can be unit-tested with a mock without importing the concrete usecase struct. Define a dedicated response struct — never serialize a `domain` entity directly. Parse the request body and write every response through `pkg/utils` (`DecodeJSONBody`/`DecodeJSONBodyStrict` in, `WriteSuccessResponse*`/`WriteErrorResponse`/`WriteValidationError`/`WriteBusinessError`/`WriteSystemError` out) — this is the one shared wire contract (`utils.ApiResponse`) every handler in the service must return, so no handler hand-rolls its own `json.NewEncoder`/error-shape.
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

## Configuration — nothing deployment-varying is hardcoded

Every value that can differ between a laptop, staging and production lives in the service's own config package — **`services/<service>/config/`** — never inline in `domain/`, `usecase/`, `repository/` or `delivery/`. In this repo that is `services/appService/config/`: it owns `AppConfig` (`config.go`) and is loaded from `config.<env>.yaml` via `config.Load(ctx, source, secretGetter)`. The composition root reads it once and injects what each layer needs through its constructor. Full per-layer detail: `references/layer_conventions.md`.

**Belongs in `config/`:**

- Credentials and connection settings — always as a `*_secret` **name** resolved at load time through a `SecretGetter` (`pkg/secrets/gcp`/`aws`), never a literal in YAML. `resolveSecrets` fills the real field in; the YAML holds only the secret's name.
- Provider/backend selection — `storage.backend`, `database.*`, the SMS provider type. Anything a different environment could point at a different implementation.
- Operational limits and windows — TTLs, cooldowns, caps, lockout durations, timeouts, page sizes.
- Endpoints, bucket names, project IDs, issuer/audience strings.

**Stays a constant in the layer that owns it:**

- What the *schema or protocol* fixes — enum members (`domain.OTPPurpose*`), sentinel error messages, a column's CHECK list. Changing one of these is a migration or an API version, not a config edit.
- Nothing else without an explicit reason recorded next to it. "It felt like a business rule" is not a reason — a business rule that a different environment may tune is configuration.

**Shape of a config addition:**

1. Add the field to the relevant `AppConfig` sub-struct in `config.go` with a `yaml:` tag. Use `config.Duration` for any `time.Duration` — yaml.v3 has no built-in support, and `Duration.UnmarshalYAML` is what parses `"90s"` / `"15m"` / `"168h"` strings.
2. Add the value to **every** `config.<env>.yaml`. A missing key is a Go zero value, which for a TTL means "already expired", not "use the default".
3. Pass it into the constructor from `cmd/main.go`. When a whole sub-struct is tunable together, pass the sub-struct rather than exploding it into six parameters: `usecase.NewRegisterUsecase(..., cfg.OTP)`. A layer that reaches for a package-level config singleton has broken dependency injection; the value still arrives as a parameter.
4. A secret gets two fields: the `*_secret` name (yaml-tagged) plus the resolved value (`yaml:"-"`, set by `resolveSecrets`) — see `DatabaseConfig.PasswordSecret`/`Password`.

**Take the type from `config/`, don't redeclare it in the layer.** The inner layer must not *read* config — no package-level singleton, no `config.Get()`, no `os.Getenv`. But it may *name* the config type: declare the field and the constructor parameter as `config.OTPConfig`, not as a hand-written look-alike struct in `usecase/`.

A local mirror struct is the tempting middle ground and it is worse than either extreme. It keeps `usecase` free of the `config` import, but the composition root then has to map field-by-field, and nothing checks that mapping: a field missed there decodes to a Go zero value. For a TTL, zero means "expired the instant it was issued" — a silent failure that no compiler and no test catches, because the test fixture supplies the policy directly. Naming the config type means there is exactly one definition and no mapping to get wrong.

This is not a layering violation. `repository` and `delivery` are the packages the Dependency Rule fences off; `config/` is a leaf that imports nothing from the service, so `usecase → config` creates no cycle and no outward dependency. Check before assuming otherwise: `scripts/check-architecture.sh` rejects `usecase` importing infrastructure, a DB driver or `net/http` — it does not reject `config`.

**Copy this existing example:** `sessionTTL` is already threaded correctly — `JWTConfig.RefreshTokenExpiration` → `usecase.NewRegisterUsecase(..., sessionTTL)` → `CreateUserAndSession(ctx, ..., sessionTTL, ...)`. The stored session expiry and the signed refresh token's expiry agree by construction because both read the one config value, instead of each restating `168h`.

**When you hit a value that looks like configuration and `config/` has no field for it, ask — don't decide alone.** Do not hardcode it as a `const` with a comment justifying why it is "really a business rule", and do not silently promote it into `config/` either. Ask which it is, and say what makes it ambiguous.

The bar for asking is low, because the two failure modes are both quiet:

- Hardcoding something that should be tunable means testing expiry in dev needs a code change and a redeploy, and prod cannot be stretched under load without one either.
- Promoting something the schema or protocol actually fixes means the next person edits `config.dev.yaml` expecting an effect that cannot happen — an enum member moved into config still has to match the column's CHECK.

Ask, and name the specific value. "Should `otpMaxAttempts` be config or a constant?" gets an answer; "are there any configs missing?" does not.

The worked example that settled this: the OTP policy (`otpTTL`, `otpResendCooldown`, `otpMaxResend`, `otpMaxAttempts`, `otpLockDuration`, `pendingRegistrationTTL`) was a hardcoded `const` block in `usecase/register_usecase.go`, justified in-comment as "business decisions, not configuration". That reasoning was rejected — they are exactly the numbers a dev environment shortens and prod lengthens — and they now live in `services/appService/config/` as `OTPConfig`, passed to the constructor the way `sessionTTL` is. This was settled by asking, not by picking a side unilaterally.

## Verification before calling a layer done

1. `go test ./services/<service>/... -run <Pkg>` for every package touched — must pass.
2. Run `scripts/check_dependency_rule.sh <path-to-service>` to catch an accidental inward-pointing import (`domain` importing `usecase`/`repository`/`delivery`, or `usecase` importing `repository`/`delivery`) before it's easy to forget about.
3. Re-read the acceptance criteria (from the user story / spec, if one exists) one by one and point to the specific test proving each — a gap here means a missing test, not an assumption that "it's probably covered."
4. Confirm no `domain` struct grew a `json`/`db` tag, and no handler response is a bare `domain` entity — see the "Entity vs response model" rule in `references/layer_conventions.md`.
5. Grep the handler for `json.NewDecoder`/`json.NewEncoder`/a hand-rolled `writeJSON`/`writeError` — every `delivery/http` handler parses via `utils.DecodeJSONBody`/`DecodeJSONBodyStrict` and writes via `utils.WriteSuccessResponse*`/`WriteErrorResponse`/`WriteValidationError`/`WriteBusinessError`/`WriteSystemError`. Found a handler with its own envelope instead? Fix it before marking the layer done — see `references/layer_conventions.md`'s `delivery/http` section.
5. **Grep every literal you introduced for a non-sensitive, deployment-varying value** (a TTL, cooldown, cap, timeout, page size, endpoint, bucket name, provider name — anything not fixed by the schema/protocol) sitting outside `services/<service>/config/`. Each one must instead be a field on `AppConfig` (or a sub-struct), threaded in from `cmd/main.go` via the constructor — never a `const`, a literal default in a struct field, or a value read straight from `os.Getenv`/a package-level singleton inside `usecase`/`repository`/`delivery`. Secrets follow the separate `*_secret` name + `SecretGetter` flow (see "Configuration" above) — this check is about the non-sensitive kind, which has no excuse to live anywhere but `config/`. Found one that isn't wired that way? Fix it before marking the layer done, or ask per the "when a value looks like configuration" rule if it's genuinely ambiguous — don't wave it through.

## After finishing work — update the docs, don't leave them drifted

Finishing a layer is not done when the code and its tests pass. The spec files under
`docs/specs/features/<feature>/` describe this feature, and code that has moved past them turns them
into a document that lies about the repo. Close the loop in the same piece of work, without being
asked:

1. **Tick `tasks.md`** — the rules are in "Tick `tasks.md` as items complete" above. This is the
   floor, not the whole job.
2. **Fix any line the work made false.** A `design.md` row that still says `*(chưa tồn tại)*` for a
   file you just wrote, a `Status:` header listing a finished layer as open, a flow step naming a
   constant you moved into `config/` — each is a claim the code now contradicts. Grep for the marker
   you are invalidating rather than fixing only the row you remember editing.
3. **Bump the version in all three spec files and add a `changelog.md` entry** — see
   `il-spec-writer`'s `references/versioning_and_changelog.md` for the header format, the minor-vs-
   major bump rule, and the entry shape. All three files carry the same version number even when
   only one changed; that is what records "reviewed together at this version".
4. **Note what to write to the knowledge bundle, but do not write it yet.** `docs/knowledge/` is the
   post-implementation OKF layer and `il-okf-writer` owns it. Its rule is that a concept is created
   only once the flow is verified working **end to end** — a layer whose repository is still a mock
   and whose route is still unrouted does not qualify, because the concept would describe a flow
   nobody has run. Record the path in `changelog.md`'s "Cross-file follow-ups" instead, and call
   `il-okf-writer` when the feature is actually exercisable.

**Which skill owns what:** `il-spec-writer` owns `docs/specs/features/<feature>/` — the work-order
spec you tick and correct as you build. `il-okf-writer` owns `docs/knowledge/` — the durable
reference written after the fact. This skill writes Go code and keeps the spec files honest; it does
not generate either document from scratch. When a spec update is more than ticking and correcting —
a new phase, a restructured flow, a genuinely new decision — invoke `il-spec-writer` rather than
improvising its format.

## What this skill does NOT do

- Does not generate mocks via `mockery`/`mockgen` — this repo hand-writes them (see above).
- Does not invent a new folder layout — it follows `services/appService`'s existing one (`domain/usecase/repository/delivery/http/cmd`).
- Does not enforce strict single-test-before-any-code red/green/refactor at the letter — writing a unit and its test in close succession, in either order, is fine as long as no unit is considered finished until its test exists and passes.
- Does not decide production-readiness concerns (secrets, mailer, auth middleware wiring) — those are `cmd/main.go` composition-root gaps, tracked separately, not a layering issue this skill's checks catch.
