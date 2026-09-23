# Verifying `resource:` and detecting drift

The `resource:` field is the concept's only mechanical link to reality. Everything else is prose,
and prose cannot be checked by a tool. This file covers verifying a path before writing it, what to
do when the target is not a plain file, and how to detect that an existing concept has drifted away
from the code.

## The absolute rule

**Never write a `resource:` path you have not confirmed exists.** Do not infer it from the folder
listing that led you to the code, do not copy it from a spec, and do not reconstruct it from the
naming convention — a file that looks like `services/appService/repository/postgres_user_repository.go`
may be named `user_repository.go` on disk.

Confirm with an actual filesystem check:

```bash
ls -d pkg/jwt/
ls services/appService/domain/user.go
ls migrations/0001_register_onboarding.sql
```

or a `Glob` against the pattern. Run it before writing the frontmatter, not after.

Note the repo convention for Go filenames — **snake_case**, not the Go-idiomatic short name:
`postgres_registration_repository.go`, `register_usecase.go`, `mock_registration_repository.go`.
Do not "correct" a filename to `register.go` because Go style prefers it; write what is on disk.

## Folder vs file targets

| Subject | `resource:` points at | Trailing slash |
|---|---|---|
| Feature | the vertical slice root — `services/appService/` | yes |
| API Endpoint | the handler file defining the route's request/response structs | no |
| Domain Entity | `services/appService/domain/<entity>.go` | no |
| UseCase | `services/appService/usecase/<feature>_usecase.go` | no |
| Architecture Pattern | the primary source file or README defining the pattern (e.g. `pkg/database/database.go` for the registry pattern) | no |
| Package | `pkg/<name>/<name>.go` — the file holding the interface + `Register`/`New` | no |
| Migration | the `.sql` file itself | no |

A Feature concept points at the `services/appService/` folder because in this repo the vertical
slice is spread across four sibling directories (`domain/`, `usecase/`, `repository/`,
`delivery/http/`) plus `cmd/` wiring — there is no single feature folder. The entry point is the
handler (`delivery/http/<feature>_handler.go`), and the per-file detail belongs in the Architecture
table inside the concept. This is what `features/register.md` and `features/login.md` do; follow
them.

If a feature's slice grows enough that a folder target would drift-check better, that is a signal the
slice is really two features — split the concept rather than loosening the `resource:`.

## When the path does not exist

This is a finding, not an obstacle. Handle it in this order:

1. **Search for the real location.** The thing may have moved:
   `find services pkg migrations -name '*registration*' -not -path './.git/*'`.
2. **If it moved**, write the real path, and if you are updating an existing concept, note the move
   in the `log.md` entry — a moved file is exactly the kind of drift a reader needs told.
3. **If it genuinely does not exist** (the feature was never built, or was deleted), stop. Do not
   write a concept about code that is not there. Report to the user what you expected to find and
   what is actually present, and ask how to proceed.

**This repo has a documented case of exactly that**, but it is now *historical*: a commit once
removed `services/appService/domain/`, `usecase/`, `repository/`, `delivery/`, `cmd/main.go` and
all seven original migration files, leaving only `config/`. Those layers have since been rebuilt
for US-001/US-002 (`cmd/`, `domain/`, `usecase/`, `repository/`, `delivery/http/`, `e2e/`) and
`migrations/` holds `0001`–`0003`. The live trap today is the opposite one: **docs/specs that still
describe the removed US-003…US-010 work** (profile/avatar, password reset,
sutras/mantras/bodhisattvas). **A stale doc naming a path is not evidence the path exists.** Before
writing any `resource:` under
`services/appService/`, run `ls` on it; a feature that only lives in a spec is spec work, not a
concept.

Never substitute a plausible path to make the frontmatter complete. A concept whose `resource:`
points at nothing is worse than no concept, because it is trusted and wrong.

## Verifying the Architecture table

Every file named in a concept's Architecture table gets the same treatment as `resource:` — it must
exist at the path written. This is the most common source of drift, because the table lists many
files and only one of them is covered by `resource:`.

Read each file, at least enough to confirm:

- The type name is what you are about to write (a renamed struct is a renamed struct).
- The layer classification is right. A file under `domain/` that imports `net/http` or a `pkg/*`
  backend is not Domain, whatever folder it sits in — that is a dependency-rule violation and should
  be reported, not documented as if it were correct.
- The one-line role you write is true of the current code, not of the code as it was when the
  concept was last updated.

For a `Package` concept, the Architecture table names the driver subpackages. Confirm each one
actually calls `Register(...)` in its `init()` — a subpackage folder that exists but never
registers is inert, and that is worth recording.

## Detecting drift in an existing concept

When updating a concept, do not trust its prose. Re-derive it:

1. Read the concept's Architecture table and `resource:` target.
2. Open each file listed. Diff your reading against the table.
3. Look specifically for the drift patterns that actually occur in this codebase:

| Drift pattern | Looks like |
|---|---|
| Renamed symbol | Table says `UserRepository`, code has `RegistrationRepository` — or the interface moved from `repository/` into `domain/` |
| Moved file | Table says `repository/user_repository.go`, code has `repository/postgres_user_repository.go` |
| Layer deleted | Concept describes `usecase/`, `repository/`, `delivery/` files that no longer exist — this repo has had exactly that happen |
| Stub became real, or real became stub | Prose says a usecase returns `errors.New("not implemented")`; code now calls a repository — or the reverse |
| Dependency rewired | Concept says `delivery` calls the repository directly; code now goes through a usecase — or a usecase started importing `repository/`, breaking the dependency rule |
| Validation rule changed | Concept states a minimum password length; the usecase or the request struct's `validate` tag has a different number |
| JSON tag leaked into `domain/` | A `domain` struct carries `json:"..."` or `db:"..."` — a delivery/repository concern that must not sit in the innermost layer |
| Route moved or unwrapped | Concept says the route is wrapped in `pkg/jwt/middleware` auth; `cmd/main.go` no longer wraps it (or vice versa) |

4. Rewrite the affected sections from what you read, and record the change in `log.md` with the
   reason. "Corrected the Architecture table after `X` was renamed to `Y`" is the shape.

There is no longer a central architecture doc (`docs/ARCHITECTURE.md` was removed as stale). When
prose and the source disagree, the source wins; fix the concept rather than copying the prose
forward.

## Handling `status` during a drift check

Drift in the *description* does not by itself change `status`. Status tracks whether the described
behavior is verified working end-to-end, which is a separate question from whether the description
is accurate.

Both directions are possible and both must be handled honestly:

- A stub that is now a real call does **not** automatically make the concept `stable`. The criterion
  is that the flow was exercised end-to-end against a real or staging backend. If you cannot confirm
  that, leave `draft` and say what is still unverified.
- Real behavior that was reverted to a stub **does** make a `stable` concept `draft`, immediately,
  with the reverted item listed in Pending work. Never leave a `stable` marking on a concept that
  now describes a stub.

## Reporting, not silently fixing

When your drift check turns up something a human needs to decide — a feature whose code was deleted
entirely, a concept whose subject turned out to be two things, a `verified:` attestation that your
rewrite invalidates — surface it in your response to the user rather than resolving it unilaterally.
The concept is a shared artifact; a silent rewrite of a verified one is the worst outcome.
