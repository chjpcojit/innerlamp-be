---
name: il-spec-writer
description: This skill should be used when the user wants a written spec (requirements, design, task breakdown, or flow diagram) for an InnerLamp backend feature — before, during, or retroactively after building it — covering the five-file convention under `docs/specs/features/<feature>/` (requirements.md, design.md, tasks.md, flow.md) plus a changelog.md. It enforces a strict one-way cascade (requirements → design → tasks: a requirement change always updates design, which always updates tasks, never the reverse), EARS-notation acceptance criteria in requirements.md, a mandatory "Step-by-step Flow" section in design.md for any flow with 2+ sequential external calls or persistent side effects (spelling out trigger/pre-condition/call/success-failure per step so hidden steps like a duplicate check, a transaction boundary, or a secondary write are never silently assumed), a matching Mermaid flow diagram in flow.md that mirrors that Step-by-step Flow 1:1 (sequence diagram by default, flowchart only for branching logic like a rate-limit/lock state machine), version headers + changelog entries on every file for every update, and a hard rule that any unanswered question from the user is recorded inline as "⚠️ Not confirmed" — never skipped, never guessed, never silently defaulted. It is a docs-only skill: it never writes Go code (that is `go-clean-architecture`'s job) and never writes to `docs/knowledge/` (the separate, post-implementation-only OKF convention, owned by `bdd-okf-docs`).
---

# InnerLamp Backend Spec Writer

## Overview

Generates and maintains the five-file spec convention for an InnerLamp backend feature under
`docs/specs/features/<feature>/`:

- `requirements.md` — **what** the system must do, from the API consumer's perspective (user story +
  EARS-notation acceptance criteria + confirmed API contract notes).
- `design.md` — **how** it is built and how it operates (Clean Architecture layers/components,
  request flow, and a step-by-step technical sequence for any multi-call flow).
- `flow.md` — **the visual of the flow**, a Mermaid sequence diagram (or flowchart for branching
  logic) that mirrors `design.md`'s Step-by-step Flow 1:1 — no new steps, none dropped. Omitted when
  the feature has no multi-call flow.
- `tasks.md` — **what to actually do, in order**, broken down directly from `design.md`'s layers,
  organized into numbered Phases, each item a checkbox (`[ ]` pending / `[x]` done).
- `changelog.md` — one shared version history for all four files above (they change together as
  one spec unit).

This skill is modeled on the existing `docs/specs/features/onboarding/` and
`docs/specs/features/article/` files — read them first as the canonical worked examples before
generating a new spec, especially to see the expected tone/format (Vietnamese prose, user-story
blockquote shape, "Ghi chú kỹ thuật liên quan" closing section).

**Language**: section headers, EARS keywords (`WHEN`/`THE SYSTEM SHALL`), table columns, and
version headers stay in English exactly as templated. The *prose* — user story text, rationale,
"Ghi chú kỹ thuật liên quan" bullets — is written in Vietnamese, matching every existing file under
`docs/specs/`. Never mix: an EARS criterion is English, the sentence explaining it can be Vietnamese.

**This skill never writes application code.** Turning a finished, fully-resolved spec into actual Go
files (domain entity, usecase, repository, handler, migration, tests, `cmd/main.go` wiring) is the
`go-clean-architecture` skill's job, not this one. This skill also never touches `docs/knowledge/` —
that is the separate OKF convention (see `docs/knowledge/index.md`), generated *after* code is
implemented and verified, as a trusted reference artifact, not a work-order spec. When a feature's
behavior changes, the OKF concept is refreshed by `bdd-okf-docs` afterwards; this skill only notes
which paths are stale.

## Hard rules that override everything else below

1. **One-way cascade, always in this order: `requirements.md` → `design.md` → `tasks.md`.** A
   requirement changes first; that change is then reflected in the relevant part of `design.md`;
   only then does `tasks.md` get new/updated checklist items. Never edit `tasks.md` based on a
   requirement change without first updating `design.md` to explain *how* that requirement will be
   met technically.
2. **Never mark a task `[x]` unless it is actually implemented in the codebase.** Newly-scoped or
   newly-changed work is always `[ ]`, even if the change itself (i.e. writing the spec) is done —
   writing the spec is not the same as implementing the feature.
3. **Any user question left unanswered is recorded, never skipped or guessed.** Use this exact
   inline format at the most specific relevant location (not just dumped at the top of the file):
   ```
   > ⚠️ Not confirmed: <the question, in one sentence> — <what it blocks / what depends on it>.
   ```
   When the user later answers it, move it to "Open questions (resolved)" in `requirements.md`
   with the resolution and date, and re-check whether that answer changes anything in `design.md`
   or `tasks.md` (it may simplify, expand, or leave them unchanged — record which).
4. **Every file update bumps `Version`/`Last updated` in that file's header and adds a
   corresponding entry to `changelog.md`.** No silent overwrites. See
   `references/versioning_and_changelog.md` for the exact bump rule and entry format.
5. **A "Step-by-step Flow" section in `design.md` is mandatory whenever the feature's main flow
   involves 2 or more sequential external calls or persistent side effects** — e.g. register →
   hash password → insert user → issue token pair → send verification mail, or list → count query
   → page query. See `references/step_by_step_flow.md` for why and how — this is the mechanism that
   prevents hidden steps (a duplicate-email check, a transaction boundary, a secondary write) from
   being silently assumed away in a one-sentence user story.
6. **Whenever rule 5 applies, a matching Mermaid flow diagram in `flow.md` is mandatory too.** It
   mirrors the Step-by-step Flow 1:1 — every arrow traces to a Step in `design.md`, every Step
   appears in the diagram; it never invents or drops a step. Sequence diagram is the default;
   flowchart only for pure branching logic (rate-limit/lock state machine, retry loops). A change
   to `design.md`'s flow requires updating `flow.md` in the same version bump. See
   `references/flow_diagram.md`.
7. **Questions asked to the user must be short, concrete, and easy to answer** — one topic per
   question, with an example or a suggested default where possible. See
   `references/gathering_questions.md` for the required question set and phrasing style.

## Reference map

Read these as needed — don't load all upfront if the task is narrow.

| Reference | Covers |
|---|---|
| `references/modes.md` | The two operating modes — **Prospective** (spec written before code exists) vs **Retroactive** (spec written from an already-implemented feature) — and how each changes the generation process |
| `references/gathering_questions.md` | The exact question set to ask before generating anything, phrasing rules, and the "never skip, mark ⚠️" rule for unanswered ones |
| `references/ears_notation.md` | How to phrase `requirements.md` acceptance criteria in EARS notation, with good/bad examples |
| `references/step_by_step_flow.md` | When and how to write the `design.md` Step-by-step Flow section; the per-step fields required (trigger/pre-condition/API/success/failure) |
| `references/flow_diagram.md` | When and how to write the `flow.md` Mermaid diagram (participant naming, arrow/transaction/branch notation, flowchart vs sequence, Vietnamese labels, the "mirror 1:1, never invent" rule) |
| `references/versioning_and_changelog.md` | Version bump rule (patch vs minor), required header format, and the `changelog.md` entry format with a worked multi-version example |

## Workflow

### 1. Determine mode and gather required decisions

Ask the user (see `references/gathering_questions.md` for exact phrasing) — do not proceed to
generation until each question has either an answer or an explicit "skip, mark unconfirmed":

1. **Feature name** (lowercase, e.g. `journal`, `practice`, `settings`) — the `<feature>` folder name
   under `docs/specs/features/`. Must match the service module/user-story module name already used
   in `docs/specs/features/<feature>/requirements.md`.
2. **Mode**: Prospective (no code yet) or Retroactive (feature already implemented)? See
   `references/modes.md` — this materially changes step 2 below.
3. **One-sentence user story**: who is the API consumer, what do they want, why.
4. **Auth model?** Does every route in this feature require `RequireAuth` middleware (user-scoped
   data), or is it public read (like `content`)? This decides whether auth is part of the feature's
   design or explicitly out of scope. If a route is *documented* as auth-required but
   `services/appService/cmd/main.go` doesn't wrap it yet, say so plainly — do not describe an
   intended middleware wiring as if it exists.
5. **Does the main flow involve more than one sequential external call or persistent side effect**
   (e.g. this feature's own insert plus a follow-on write owned by another module, or a
   storage/email call after a DB write)? If yes, the Step-by-step Flow section in `design.md`
   (rule 5) **and** the matching Mermaid diagram in `flow.md` (rule 6) are both mandatory.
6. **Confirmed contracts?** For each endpoint identified in step 5, is the exact contract (method,
   path, request/response shape, error cases, status codes) already confirmed and reflected in
   `docs/specs/api/openapi.yaml`, or still unknown? Unknown ones become `⚠️ Not confirmed` items,
   not assumed shapes.
7. **Schema impact?** Does this feature add/alter tables or columns? If yes, which `migrations/000N_*.sql`
   file is expected, and is the ERD under `docs/specs/db/` affected?

### 2. Populate the four files + changelog

Copy each file in `assets/templates/` to `docs/specs/features/<feature>/`, replacing `{{feature}}`
with the lowercase feature name, then fill in real content:

| Template | Destination | Key sections to fill |
|---|---|---|
| `assets/templates/requirements.md.template` | `docs/specs/features/<feature>/requirements.md` | User Story, EARS Acceptance Criteria (`references/ears_notation.md`), Out of scope, Pending, Open questions |
| `assets/templates/design.md.template` | `docs/specs/features/<feature>/design.md` | Overview, Layers & components table, Request flow, Step-by-step Flow (if rule 5 applies), Deliberately-stubbed-for-now section for any unconfirmed endpoint |
| `assets/templates/flow.md.template` | `docs/specs/features/<feature>/flow.md` | One Mermaid sequence diagram per Step-by-step Flow section, mirroring it 1:1; optional flowchart for branching logic (`references/flow_diagram.md`). Omit this file entirely if rule 5 does not apply |
| `assets/templates/tasks.md.template` | `docs/specs/features/<feature>/tasks.md` | Phases broken down 1:1 from `design.md`'s Layers & components table — never invent a task that doesn't trace back to something named in `design.md` |
| `assets/templates/changelog.md.template` | `docs/specs/features/<feature>/changelog.md` | Single `v1.0 (initial)` entry describing what this first version covers |

**Retroactive mode**: before filling `requirements.md`/`design.md`, read the actual feature code
(`services/appService/domain/`, `usecase/`, `repository/`, `delivery/http/`, `cmd/main.go` wiring,
and the relevant `migrations/NNNN_*.sql`) so `tasks.md` accurately marks what's `[x]` done vs `[ ]`
pending — never assume a layer exists or is complete without checking.

**Prospective mode**: every task in `tasks.md` starts `[ ]`; `design.md` describes the intended
architecture, clearly distinguishing "will call a real repository/service" vs "stubbed until a
contract exists" per the confirmed-contracts answer from step 1.6.

### 3. Handle an existing spec being updated (not created fresh)

If `docs/specs/features/<feature>/design.md` already exists, this is an **update**, not a fresh
generation:

1. Identify which file the change originates in. A new/changed user-facing behavior starts in
   `requirements.md`. A newly-discovered technical detail (e.g. confirming an endpoint shape) may
   resolve directly in `design.md`/`requirements.md` without necessarily changing behavior.
2. Cascade per rule 1 above: `requirements.md` change → check/update `design.md` → check/update
   `flow.md` (if it exists, or add it if rule 5/6 now applies) → check/update `tasks.md`. Explicitly
   state in your response (and in the changelog entry) if a downstream file did *not* need to
   change, and why — don't silently skip it without noting that you checked. A change to
   `design.md`'s Step-by-step Flow always changes `flow.md` in the same bump (rule 6).
3. Bump version + add a changelog entry per `references/versioning_and_changelog.md`, describing
   what changed and why in plain language (not just "updated file").
4. If the change resolves a `⚠️ Not confirmed` item, move it to "Open questions (resolved)" in
   `requirements.md` with the resolution, and explicitly check (and state) whether it simplifies,
   expands, or leaves unchanged the affected part of `design.md`/`flow.md`/`tasks.md`.
5. If the change touches an HTTP contract, flag `docs/specs/api/openapi.yaml` for the same change
   (it is generated from the handlers and must not drift). If it touches schema, flag
   `migrations/` and the matching `docs/specs/db/` ERD. If it changes shipped behavior, flag the
   matching `docs/knowledge/features/*.md` concept for `bdd-okf-docs` to refresh — record these as
   explicit follow-ups, do not silently edit them here.

### 4. Verify before reporting done

- Every acceptance criterion in `requirements.md` is phrased in EARS notation (WHEN/THE SYSTEM
  SHALL), not vague prose.
- Every task in `tasks.md` traces back to a named layer/component in `design.md`'s table — no
  orphan tasks.
- No task is checked `[x]` unless it is genuinely implemented in the codebase (verify by reading
  the actual files if unsure, don't assume from the spec's own prior wording).
- Every `⚠️ Not confirmed` marker is either still open (and blocking the tasks that depend on it)
  or has been moved to "Open questions (resolved)" with a date and resolution.
- `Version`/`Last updated` headers match across all changed files, and `changelog.md` has one new
  entry per version bump, in reverse-chronological order (newest first).
- If the flow involves 2+ sequential external calls or side effects and no Step-by-step Flow
  section exists yet, add one before reporting done — this is not optional (rule 5).
- If a Step-by-step Flow section exists, `flow.md` exists too, with one sequence diagram per flow
  section that mirrors it 1:1 — no diagram step without a matching `design.md` Step, and no
  `design.md` Step missing from the diagram (rule 6). Mermaid keywords are English, labels are
  Vietnamese; transaction boundaries are marked with `rect` + `Note`, and any `⚠️ Not confirmed`
  step appears as a `Note`, not drawn as settled.
- Cross-file follow-ups (`docs/specs/api/openapi.yaml`, `migrations/`, `docs/specs/db/`,
  `docs/knowledge/`) are listed in the response and in the changelog entry.
