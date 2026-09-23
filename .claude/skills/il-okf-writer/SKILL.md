---
name: il-okf-writer
description: This skill should be used when the user wants to create or update a knowledge concept in the InnerLamp OKF (Open Knowledge Format) bundle under docs/knowledge/ — a post-implementation reference document describing a feature, an API endpoint, a domain entity, a UseCase, an architecture pattern, a pkg/ infrastructure package, or a DB migration, with YAML frontmatter (type, title, resource, tags, status, generated, sources, verified), a resource: path pointing at the real Go code location, and cross-links to related concepts. It enforces that concepts are generated only AFTER the code is implemented and verified (never as a work order — that is il-spec-writer's job under docs/specs/features/), that every resource: path is checked against the filesystem rather than guessed, that status stays draft until the flow is exercised end-to-end, that the verified: field is never written by the AI (only a human adds it during PR review), and that every create/update also refreshes index.md and log.md. It never writes Go code and never touches docs/specs/.
---

# InnerLamp OKF Writer

## Overview

Generates and maintains concepts in the InnerLamp OKF bundle at `docs/knowledge/` — the
post-implementation, durable knowledge layer for this Go backend repo.

A **concept** is one markdown file with YAML frontmatter, covering exactly one thing: one feature,
one API endpoint, one domain entity, one UseCase, one architecture pattern, one `pkg/` package, or
one DB migration. Concepts cross-link to each other; they are not a single monolithic document.

Read these first, in this order, before generating anything:

1. `docs/knowledge/index.md` — the bundle's table of contents (current sections + concept table).
   The bundle is **populated**: `features/register.md` (`draft`) and `features/login.md`
   (`stable`) are the worked examples to copy. Read whichever is closest to what you are writing
   before inventing structure, and keep `index.md` and `log.md` in sync (rule 5).
2. The `README.md` of whatever folder the concept describes — `pkg/<name>/README.md`,
   `services/appService/<layer>/README.md`. Check it exists first: folder-level READMEs in this
   repo are incomplete (`config/`, `domain/`, `usecase/` have one; `repository/`, `delivery/http/`
   and `cmd/` do not), and a missing one is not an error to work around — if the folder handles
   non-trivial logic and has no README, say so in the concept's Pending work. Where a README does
   exist it may be stale: read the code, not the README (rule 6).

**This skill is post-implementation only.** A concept documents code that exists and works. It is
not a spec and must never be used as a work order — writing specs before/during implementation is
`il-spec-writer`'s job under `docs/specs/features/<feature>/`. The one documented exception is a
complex business rule (e.g. a multi-step OTP flow) drafted as a concept *before* coding without a
`resource:` field; once implemented, that same concept gains `resource:` and is reconciled against
the real code.

**Note on the current repo state.** Both halves of the repo are real now. `pkg/` holds **16**
provider-agnostic packages — `config`, `database`, `httpclient`, `jwt`, `logging`, `mailer`,
`metrics`, `middleware`, `ratelimit`, `secrets`, `session`, `sms`, `storage`, `tracing`, `utils`,
`validation`. `services/appService/` has a full Clean Architecture vertical slice for US-001
(register) and US-002 (login): `cmd/`, `config/`, `domain/`, `usecase/`, `repository/`,
`delivery/http/` and `e2e/`, with `migrations/0001`–`0003` applied. Under `docs/specs/features/`,
`onboarding/` (register) is implemented but stays `draft` because of dev placeholders, and
`login/` is implemented and audited end-to-end (hence `stable`); `content/`, `journal/` and
`practice/` are still prospective `requirements.md`-only specs — write no concept for them until
code exists.

There is no central architecture doc (`docs/ARCHITECTURE.md` was removed as stale). Any prose doc
can drift: never derive a concept from a doc — `ls` the path and read the code. The current tree
implements US-001 (register) and US-002 (login) only; profile/password-reset and the content
features exist as specs, not code.

## Hard rules that override everything else below

1. **Never write a `resource:` path without verifying it exists on disk.** Run `ls`/`Glob` against
   the path before writing the frontmatter. A guessed or stale path silently breaks the concept's
   only link to reality. See `references/resource_verification.md`.
2. **Never write the `verified:` field.** It is a human attestation added during PR review by the
   person who confirmed the concept matches the code. An AI-authored `verified` destroys the
   trust-tier distinction the format exists to provide. Only omit, or preserve an existing value
   untouched — never create or change one.
3. **`status:` is `draft` until the described behavior is exercised end-to-end** against a real or
   staging backend. A feature whose usecase method returns `errors.New("not implemented")` is
   `draft` no matter how complete its handler and tests look — and the concept must say so plainly
   in a "Pending work" section, never folded into present-tense claims of working behavior.
4. **`generated.by` must name the agent actually authoring it.** Use `claude-code` when this skill
   writes the concept. Never copy a different producer string forward from an existing file into a
   file this skill authored.
5. **Every create and every update also updates `index.md` and `log.md` in the same pass.** A
   concept that is not listed in `index.md` is effectively invisible; a change with no `log.md`
   entry has no history. See `references/bundle_management.md`.
6. **Read the code, do not trust the concept or README.** When updating an existing concept,
   re-read the source files it points at. Documentation drift — a concept describing a method that
   was renamed, a layer that was deleted — is the specific failure this bundle exists to prevent.
   See `references/resource_verification.md`.
7. **Never invent a claim to fill a section.** If a section does not apply (a feature with no
   validation rules, an endpoint with no auth), omit the section. Do not write placeholder prose
   and do not guess an unconfirmed API contract — mark it pending instead.

## Reference map

Load only what the task needs; do not read all of these upfront.

| Reference | Covers |
|---|---|
| `references/frontmatter_schema.md` | Every frontmatter field, its type, allowed values, and the `verified`/`stale_after` trust-tier rules |
| `references/concept_types.md` | The seven `type:` values, which bundle section each lives in, and how to pick the right one |
| `references/bundle_management.md` | `index.md` concept-table row format, `log.md` entry format and reverse-chronological ordering, naming and cross-link rules |
| `references/resource_verification.md` | How to verify a `resource:` path, what to do when the target is a folder vs a file vs missing, and how to detect drift against an existing concept |
| `references/gathering_questions.md` | The question set to ask before generating, and when a question can be skipped |

## Workflow

### 1. Determine intent and gather required decisions

Establish, asking the user only for what cannot be determined from the code and the bundle:

1. **What is being documented** — a feature, endpoint, entity, UseCase, architecture pattern,
   `pkg/` package, or migration. This picks the `type:` and the section folder (see
   `references/concept_types.md`).
2. **Create or update?** Check whether the target file already exists under `docs/knowledge/`. An
   existing file means update — go to step 4.
3. **Is the thing actually implemented?** If it is not built yet, stop: this is spec work for
   `il-spec-writer`, not OKF work. The only exception is the complex-business-rule draft described
   in the Overview.

See `references/gathering_questions.md` for exact phrasing. Ask no more than necessary — most
answers are discoverable by reading the code and the existing bundle.

### 2. Read the source of truth

Before writing a single line, read the real code the concept will describe:

| Concept type | Read |
|---|---|
| Feature | every file of the vertical slice — `services/appService/delivery/http/<feature>_handler.go`, `services/appService/usecase/<feature>_usecase.go`, `services/appService/domain/*.go`, `services/appService/repository/postgres_<feature>_repository.go`, and the route wiring in `services/appService/cmd/main.go`. Note the request/response structs and the sentinel errors the handler maps. |
| API Endpoint | the handler file's request/response structs and its error→status mapping, the route registration in `cmd/main.go` (and whether the route is wrapped in `pkg/jwt/middleware` auth), plus the `docs/specs/features/<feature>/requirements.md` contract notes if present |
| Domain Entity | the entity file itself under `services/appService/domain/`, plus every usecase that constructs or consumes it |
| UseCase | the usecase file, its constructor signature, and the handler that calls it (the handler declares the narrow consumer interface) |
| Architecture Pattern | two or three real implementations that follow the pattern, the relevant `README.md`, and the real imports (`grep` for `"chjpcoj.com.innerlamp/pkg/..."`) to establish the dependency direction |
| Package | `pkg/<name>/<name>.go` (the interface + `Register`/`New`), `pkg/<name>/README.md`, the driver subpackages, and a `grep` of its imports for which packages it actually depends on |
| Migration | the `.sql` file itself, the sibling `.sql` files for the numbering convention (`NNNN_<slug>.sql`), and `docs/specs/db/` ERD docs if present |

Then verify the `resource:` path per `references/resource_verification.md` — actually check it,
do not assume it from the folder listing you used to find the code.

Note what is stubbed, unwired, or known-broken while reading. Those become the concept's "Pending
work" and "Known gaps vs. design" sections, and they are usually the most valuable content in the
file — they are what a future reader (or agent) cannot rediscover cheaply. Today's repo-wide
dev-only wiring is the mock secret resolver (`appConfig.NewMockSecretGetterFromYAML`) and the
log-only SMS provider (`pkg/sms/logsms`), plus the deferred cleanup job and the hardcoded OTP
(`"111111"`); check whether a newer gap supersedes them before restating. There is no central
gap list — rediscover the gaps by reading `cmd/main.go` and the layers around the feature.

### 3. Create the concept

Copy the matching template from `assets/templates/`, replace the `{{...}}` placeholders, and fill
in real content:

| Type | Template | Destination |
|---|---|---|
| Feature | `assets/templates/feature.md.template` | `docs/knowledge/features/<feature-lower>.md` |
| API Endpoint | `assets/templates/api-endpoint.md.template` | `docs/knowledge/api/<domain>-<action>.md` |
| Domain Entity | `assets/templates/domain-entity.md.template` | `docs/knowledge/entities/<entity-lower>.md` |
| UseCase | `assets/templates/usecase.md.template` | `docs/knowledge/usecases/<usecase-lower>.md` |
| Architecture Pattern | `assets/templates/architecture-pattern.md.template` | `docs/knowledge/architecture/<pattern-lower>.md` |
| Package | `assets/templates/package.md.template` | `docs/knowledge/packages/<package-lower>.md` |
| Migration | `assets/templates/migration.md.template` | `docs/knowledge/migrations/<number>-<slug>.md` |

Sections the templates require for every type: `# Summary` (present tense, what it is and where it
sits), `# Architecture` (a layer/type/file table naming real files), and `# Related concepts`
(relative markdown links to sibling concepts — link liberally, and a link to a concept that does
not exist yet is acceptable and useful as a marker).

Then create any section folder the concept needs that does not exist yet, and register the concept
per step 5.

### 4. Update an existing concept

When the target file already exists:

1. Read the current concept, then re-read every file listed in its Architecture table and its
   `resource:` target. Note every discrepancy — renamed symbol, moved file, changed validation
   rule, a stub that has since been implemented.
2. Rewrite the affected sections from the code, not from the old prose.
3. Re-evaluate `status:`: a stub that is now a real call may promote the concept from `draft` to
   `stable` **only** if the flow has been exercised end-to-end. If unsure, leave it `draft` and say
   what is still unverified — do not promote on the strength of the code merely existing.
4. Update `generated.at` to the current timestamp.
5. Preserve `verified:` untouched if present. If the concept changed materially and a `verified:`
   field exists, note in your response to the user that the human verification is now stale and
   should be re-confirmed in review — but do not remove the field yourself.
6. Update `index.md` if the concept's row description or status changed, and add a `log.md` entry
   describing what changed and why.

### 5. Refresh index.md and log.md

Every create/update ends here — this is not optional (rule 5):

- Add or update the concept's row in `docs/knowledge/index.md`'s concept table, and move it out of
  any "Planned, not yet populated" list once its section has its first concept.
- Add an entry to `docs/knowledge/log.md`, newest first, describing what was reviewed for
  accuracy, what the concept documents, what remains pending, and why `status` is what it is.

`docs/knowledge/index.md` and `docs/knowledge/log.md` are already populated — **update** their
existing structure, do not rewrite them from scratch. Formats for both: `references/bundle_management.md`.

### 6. Verify before reporting done

- The `resource:` path exists on disk and points at the entity the concept actually describes.
- Every file named in the Architecture table exists at the path written.
- `generated.by` names this agent, and `generated.at` is current.
- No `verified:` field was authored or altered.
- `status:` matches reality — `draft` if anything described is stubbed, unwired, or unverified
  end-to-end, with a "Pending work" section listing exactly what.
- Every `# Related concepts` link uses a correct relative path from the new file's location.
- `index.md` and `log.md` both reflect this concept, and every markdown link target you claim
  exists actually exists.
- Nothing under `docs/specs/` was modified, and no `.go` file was touched.
