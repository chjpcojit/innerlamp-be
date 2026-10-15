---
name: adr-writer
description: This skill should be used when an InnerLamp feature or change involves an architecture decision that is hard to reverse — infrastructure (cloud services, deployment, queues, caches, storage), a new or reshaped data model, offline sync / conflict resolution between devices, authentication / authorization / session design, cross-platform data ownership, or any choice between technologies — and that decision must be recorded as an Architecture Decision Record (ADR) under `docs/adr/NNNN-<slug>.md` in the backend repo before the product spec (`ba-writer`) or platform specs (`spec-writer` in each of the four repos) are written. It wraps the generic `engineering:system-design` and `engineering:architecture` processes with InnerLamp's own context (CLAUDE.md, existing ADRs, product specs, the real stack), a project ADR template, an explicit "is an ADR needed at all?" gate, and the rules that it asks before writing, records unanswered questions as "⚠️ Not confirmed", always compares at least two real options plus the status quo, writes every new ADR as Proposed (only the user moves it to Accepted), and never edits an Accepted ADR (a changed decision is a new ADR that supersedes it). Docs-only: the ADR body is English, chat with the user is Vietnamese ("anh"/"em"); it never writes code, product specs or platform specs. Use it when the user says ADR, architecture decision, system design, "nên chọn X hay Y", "thiết kế kiến trúc", or when `ba-writer` stops because a feature needs an architecture decision that has no ADR yet.
---

# InnerLamp Architecture Decision Record writer (`adr-writer`)

## Overview

InnerLamp is one product on four platforms (this Go backend, the SvelteKit website, iOS, Android).
Most features need no architecture decision: they reuse the stack that already exists. A few do —
and those decisions are the expensive ones to get wrong, because every platform builds on top of
them. This skill records them, once, as an ADR in `docs/adr/` of this repo, **before** the product
spec is written.

```
Raw idea / request
   │
   ▼
(optional) adr-writer   docs/adr/NNNN-<slug>.md   ← this skill, only when a hard-to-reverse decision exists
   │ cited by ID (ADR-0003), never copied
   ▼
ba-writer               docs/product/<feature>/product.md          (layer 0)
   ▼
spec-writer (every repo)              platform specs                (layer 2)
   ▼
implementation skills (golang-*, il-ui, il-state, ui-builder, il-data-builder…) → CI
```

An ADR answers **how the system is shaped and why** — the one decision, the options considered and
the consequences. The product spec answers **what the user observes**; the platform specs answer
**how each platform builds it**. A product spec or platform spec that depends on a decision cites
`ADR-NNNN` and never restates its content.

This skill embeds the processes of the generic `engineering:system-design` (requirements → high-level
design → deep dive → scale/reliability → trade-offs) and `engineering:architecture` (ADR with options,
trade-off analysis, consequences) so it does not depend on those global skills being installed or
triggering. Where they and this file disagree, this file wins.

## Hard rules that override everything else below

1. **Read the project before anything else.** Always start with `CLAUDE.md`, `docs/adr/README.md`
   and every ADR whose title touches the same area (step 1). Never propose an option that an
   Accepted ADR already rejected without saying so and proposing to supersede it.
2. **Gate first: is an ADR needed?** Apply `references/when_to_write_adr.md`. If the answer is no,
   say so in one or two lines (Vietnamese) and stop — do not write an ADR for a decision that is
   cheap to reverse or already settled by the existing stack.
3. **One decision per ADR.** If the input bundles several decisions (e.g. "offline sync + new
   storage + auth change"), split them into separate ADRs and say which depends on which.
4. **Ask before you write.** Present the normalized problem, constraints and open questions in
   Vietnamese and wait for answers. Every unanswered question is recorded inline, in exactly this
   form, and listed again in the ADR's "Open questions" section:
   ```
   > ⚠️ Not confirmed: <the question, in one sentence> — <what it blocks>.
   ```
   Never guess a constraint (budget, scale, latency, data residency, deadline) and never silently
   default one.
5. **Real options only.** At least two genuine options **plus "keep the status quo / do nothing"**,
   each assessed on the same dimensions (`references/design_process.md` §4). A one-option ADR is a
   justification, not a decision record.
6. **Status discipline.** A new ADR is always written as `Proposed`. Only the user moves it to
   `Accepted` or `Rejected` — when they say so explicitly in the conversation; record their words and
   the date in the ADR's "Decision record" section. Never write `Accepted` on your own.
7. **Accepted ADRs are immutable.** A changed decision is a new ADR with `Supersedes: ADR-NNNN`; the
   old one gets only its status line changed to `Superseded by ADR-MMMM`. Typos and broken links may
   be fixed in place.
8. **Grounded in the real stack.** Every claim about what exists today (packages, DB, deployment,
   middleware, client architecture) is checked against the code or a repo README and cited as a
   path. If you could not verify something, say so in the ADR instead of asserting it.
9. **Language split.** ADR body is English (Vietnamese domain terms keep their original form with an
   English gloss, e.g. "sổ giỗ (memorial book)"). Everything said to the user in chat is Vietnamese,
   "anh"/"em", technical terms kept in English.
10. **Docs only.** Never write code, migrations, `product.md`, `requirements.md`/`design.md`/`tasks.md`,
    OKF concepts, or edit `docs/specs/api/openapi.yaml`. Contract, product and platform follow-ups
    go into the ADR's "Follow-ups" section and the chat report as handoff items.

## Reference map

| Reference | Covers |
|---|---|
| `references/when_to_write_adr.md` | The gate: which decisions need an ADR (and which do not), with InnerLamp examples |
| `references/design_process.md` | The system-design + architecture process adapted to InnerLamp: context to read, questions to ask, assessment dimensions, deep-dive checklists for infra / data model / offline sync / auth |
| `assets/templates/adr.md.template` | The ADR file layout |

## Workflow

### 1. Load project context

Read, in this order, and keep notes of what each says about the decision area:

| Where | What |
|---|---|
| `CLAUDE.md` | Stack, known broken state, conventions, package relationships |
| `docs/adr/README.md` + related `docs/adr/*.md` | Prior decisions — Accepted ones are constraints, Proposed ones may be the same question |
| `docs/product/<feature>/` | Product spec if one already exists (BR/FR that drive the decision) |
| `docs/specs/features/<feature>/`, `docs/assumptions/`, `docs/tech-debt/` | Current backend design, assumptions and known shortcuts in the area |
| Code | The parts the decision touches: `go.mod`, `services/appService/cmd/main.go`, `migrations/`, the relevant `pkg/<name>/README.md` |
| Client repos (when the decision crosses platforms) | `../innerlamp-website`, `../innerlamp-iOS`, `../innerlamp-Android` — their `CLAUDE.md`/`AGENTS.md` and the feature's spec folder. If a repo is not checked out, say so; never assume it has nothing |

### 2. Gate

Apply `references/when_to_write_adr.md`. Not needed → say so and stop. Needed → name the one
decision in a single sentence ("How do memorial records sync between a user's devices while
offline?") and, if the input bundles several, list the separate ADRs and their order.

### 3. Requirements and constraints (system-design §1)

Build the problem statement: functional drivers (cite BR/FR IDs when a product spec exists),
non-functional drivers (scale, latency, availability, cost, privacy/security, offline), and
constraints (solo developer, existing stack, budget, deadline, platforms in scope). Then in chat
(Vietnamese): a 3–5 line summary, the constraints you inferred **marked as inferred**, and a short
prioritized question list per `references/design_process.md` §2. Wait for answers.

### 4. Options and trade-offs (system-design §2–5, architecture)

For each option (≥2 real + status quo): a short description, how it fits the current stack, and the
assessment table from `references/design_process.md` §4. Do the deep-dive checklist that matches the
decision type (§5 of the same file) for the options that survive a first pass. Make the trade-offs
explicit; recommend one option and say why, but leave the decision to the user (hard rule 6).

### 5. Write the ADR

1. Next number = highest `NNNN` in `docs/adr/` + 1, zero-padded to 4 digits. Slug is lowercase
   kebab-case of the decision.
2. Copy `assets/templates/adr.md.template` to `docs/adr/NNNN-<slug>.md` and fill every section. A
   section that cannot apply is marked `N/A — <reason>`; an unanswered one keeps a `⚠️ Not confirmed`.
3. Status `Proposed`, date today, `Features` listing the feature slugs it affects.
4. Add the row to the index table in `docs/adr/README.md`.
5. If superseding: set the old ADR's status line to `Superseded by ADR-NNNN` and update its index row.

### 6. Verify

- Gate reason, one decision, ≥2 real options + status quo, same dimensions for every option.
- Every "exists today" claim has a path; nothing asserted that you did not check.
- Every open question appears inline and in "Open questions".
- Status is `Proposed` unless the user explicitly decided in this conversation.
- No code, no product/platform spec content beyond follow-up pointers.
- Index row added; numbering has no gap or duplicate.

### 7. Report and hand off (in Vietnamese)

Tell the user: the ADR file and number, the recommended option in one line, open `⚠️` items, and what
they need to do — accept / reject / answer questions. Then the next step: `ba-writer` for the product
spec (citing `ADR-NNNN`), `spec-writer` in this repo and in each client repo for platform specs, plus any
contract follow-ups.
