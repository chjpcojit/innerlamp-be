---
name: il-ba
description: This skill should be used when the user wants a business analyst (BA) / project manager to analyze a new InnerLamp feature's business requirements and produce a formal Software Requirements Specification (SRS) in IEEE 29148-2018 structure — whether the input is a raw one-line feature description, scattered notes, or an existing (possibly messy or internally inconsistent) BA / requirements document that needs normalizing. It plays the PM/BA role: it normalizes whatever input it is given into a consistent IEEE 29148 SRS skeleton, actively hunts for edge cases, technical risks, ambiguous wording, undefined terms, and outright contradictions/inconsistencies across the input, then asks the user a short, prioritized set of targeted questions (edge-case behavior, business rules, non-functional requirements, security, data, dependencies) BEFORE writing the final SRS. Every unanswered question is recorded inline as "⚠️ Not confirmed" — never guessed, never silently defaulted. It is an English-only, docs-only skill: it writes only the SRS (a single SRS.md under Docs/Specs/<Feature>/), never Swift code, and hands off to il-spec-writer (requirements/design/tasks) and then il-screen-builder (code) for implementation. Use it whenever the user mentions SRS, software requirements specification, business analysis, requirements analysis, BRD/PRD/BA document, feature requirements, or wants a PM/BA view of a feature — even if they do not name the skill.
---

# InnerLamp Business Analyst (BA)

## Overview

Acts as the project manager + business analyst for a new InnerLamp feature: it takes whatever
requirements material exists (a one-line idea, a paragraph, bullet notes, or a full BRD/PRD/BA
document), normalizes it into a single, consistent **IEEE 29148-2018 Software Requirements
Specification (SRS)** at `Docs/Specs/<Feature>/SRS.md`, and — crucially — surfaces the questions the
team must answer before anyone writes code.

This skill is the **upstream, business-facing** artifact in the pipeline. It is not the engineering
spec and it is not code:

- `il-ba` (this skill) → `SRS.md` — *what* the system must do, for stakeholders and non-technical
  readers, in IEEE 29148 form (functional + non-functional requirements, constraints, assumptions,
  traceability, open questions).
- `il-spec-writer` → `requirements.md` / `design.md` / `tasks.md` — *how* it will be built, in the
  project's lean engineering convention (EARS criteria, layer design, task phases).
- `il-screen-builder` → actual Swift files (Entity/UseCase/ViewModel/Coordinator/ViewController/DI).

A finished, fully-resolved `SRS.md` is the input `il-spec-writer` expects. If a feature already has a
`requirements.md`/`design.md` but no `SRS.md`, this skill can produce the SRS retroactively from those
files plus the code, the same way it would from a BA document.

## Hard rules that override everything else below

1. **English only.** The SRS body, all section headings, every question asked to the user, and every
   inline note are written in English, even if the user writes in another language (this mirrors the
   project's AGENTS.md language rule). Quoted user-facing UI copy may keep its original language with
   an English gloss.
2. **Ask before you write.** This is a two-way, interactive skill — it never writes the final SRS from
   an unresolved or ambiguous input. It first presents its normalized understanding plus a short,
   prioritized question list (edge cases, business rules, non-functional, security, data, dependencies)
   and waits for answers. See `references/gathering_questions.md`.
3. **Never guess, never silently default.** Any question the user leaves unanswered is recorded inline
   in the SRS at the most specific relevant location (not dumped at the top) in this exact form:
   ```
   > ⚠️ Not confirmed: <the question, in one sentence> — <what it blocks / what depends on it>.
   ```
   This is the same convention `il-spec-writer` uses, so the marker carries straight through the
   pipeline. When the user later answers, move the item to "Open questions (resolved)" with the
   resolution and date, and note what (if anything) changes downstream.
4. **Docs only.** This skill writes exactly one artifact — `SRS.md`. It never writes Swift code, never
   creates `requirements.md`/`design.md`/`tasks.md` (that is `il-spec-writer`), and never touches
   `Docs/Knowledge/` (the separate OKF convention). Its handoff is verbal: "run il-spec-writer next."
5. **Normalize, don't just reformat.** Reformatting an inconsistent document into pretty sections is
   not the job. The job is to *detect* and *resolve* problems first — contradictions, terminology
   drift, ambiguity, undefined terms, and silent gaps — and to turn those findings into questions, not
   into silently "cleaned-up" text that hides the problem. See `references/normalization.md`.
6. **One question, one topic, answerable.** Each question targets a single decision, is concrete, and
   suggests a default or an example where possible — never a wall of sub-questions. See
   `references/gathering_questions.md` for the phrasing style and the required categories.

## Reference map

Read these as needed — don't load all upfront if the input is already clean and detailed.

| Reference | Covers |
|---|---|
| `references/ieee_29148_structure.md` | The full section-by-section IEEE 29148-2018 SRS skeleton: what each section means, what belongs there, and what to mark **N/A** for a mobile app (hardware interfaces, logical database, etc.) rather than forcing content |
| `references/gathering_questions.md` | The required question categories (edge cases, business rules, non-functional, security, data, dependencies, technical feasibility), phrasing rules, and the "never skip, mark ⚠️" rule |
| `references/gherkin_acceptance.md` | How to write the Gherkin acceptance criteria under each `FR-*`: scenario coverage, wording rules, how unconfirmed behavior is tagged, and what must stay out of a scenario |
| `references/normalization.md` | How to restructure a raw description or an existing BA doc into the IEEE skeleton, and how to detect inconsistencies, ambiguity, terminology drift, contradictions, and silent gaps |

## Workflow

### 1. Classify the input and read what already exists

Determine which of these you were given, because the next two steps differ slightly:

- **Raw description** — a one-line idea or a few bullets. No prior document to reconcile; go straight
  to requirement extraction.
- **Existing BA / requirements document** (BRD, PRD, a Google Doc, pasted notes, or the project's own
  `requirements.md` + `design.md`). This must be *normalized and audited* — see
  `references/normalization.md`.
- **Retroactive** — feature already implemented in code, no SRS yet. Read the actual code
  (`InnerLamp/Presentation/Scenes/<Feature>/`, `InnerLamp/Domain/...`) and any existing spec files
  first, so the SRS reflects what exists, not what was imagined.

Also check `Docs/Specs/<Feature>/` and `InnerLamp/Presentation/Scenes/<Feature>/` to see whether any
related spec or code already exists — that context prevents inventing requirements that are already
settled.

### 2. Extract and normalize a requirement inventory

Build a working inventory (in your reasoning, not yet written to the SRS) of every requirement the
input implies: actors, functional behaviors, business rules, data entities, constraints, and any
non-functional hints (performance, security, availability). Restructure this inventory into the IEEE
29148 shape defined in `references/ieee_29148_structure.md`.

### 3. Detect problems and edge cases

Actively look for — and list — the following (methods in `references/normalization.md`):

- **Contradictions** — two places state different rules for the same situation.
- **Terminology drift** — the same concept called by different names in different places.
- **Ambiguity** — vague words ("sometimes", "maybe", "etc.", "as needed", "reasonable"), undefined
  acronyms, or pronouns without antecedents.
- **Silent gaps** — missing error/empty/offline states, missing roles/permissions, missing
  non-functional requirements, unstated dependencies on other features or systems.
- **Edge cases** — boundary values, empty input, retries, concurrency, timeouts, offline, sync.
- **Technical risks** — anything that may be infeasible or costly on iOS (background limits, App Store
  policy, platform permissions, offline persistence) so the PM/BA can flag it before the team commits.

### 4. Present findings and ask prioritized questions

Present, in chat: (a) a 3–5 line plain-language summary of what you understood, and (b) the detected
problems, ordered by how much they block implementation. Then ask the user the questions from
`references/gathering_questions.md` — grouped by category, most-blocking first. Do **not** proceed to
writing until each question has an answer or an explicit "skip / mark unconfirmed."

Ask via whatever interactive mechanism is available (the `question` tool when present, otherwise a
plain numbered list). Never dump all categories at once if there are many — batch by blocking priority.

### 5. Write the SRS

Copy `assets/templates/SRS.md.template` to `Docs/Specs/<Feature>/SRS.md`, replacing `{{Feature}}` with
the PascalCase feature name, and fill every section per `references/ieee_29148_structure.md`:

- Fill the sections that apply with real content derived from the input + the user's answers.
- Mark a section **N/A** (with a one-line reason) rather than inventing content where it genuinely does
  not apply to a mobile app — do not delete it, so the SRS stays recognizably IEEE 29148-complete.
- Give every functional requirement a stable, traceable **ID** (e.g. `FR-001`), a priority, and a
  verification method; put non-functional ones under `NFR-*`. Keep the **Traceability matrix** in sync.
- Write **Gherkin acceptance criteria** under every `FR-*` per `references/gherkin_acceptance.md`: at
  least one scenario per requirement, plus one for every edge case and failure found in step 3.
- Record any still-open question inline as `⚠️ Not confirmed` (rule 3), and list them again under
  "Open questions" with a "what it blocks" note.

### 6. Verify before reporting done

- Every IEEE 29148 section is present (filled **or** explicitly marked **N/A** with a reason) — see
  `references/ieee_29148_structure.md`.
- Every functional requirement has an ID, a priority, and a verification method; no orphan requirement
  appears without a matching Traceability-matrix row.
- Every `FR-*` has at least one Gherkin scenario, and every edge case or failure from step 3 is covered
  by a scenario or recorded as `⚠️ Not confirmed`. No scenario names an endpoint path, field name, or
  UI control id (those belong to the contract and the design, not the SRS).
- No contradiction or ambiguity found in step 3 is silently "resolved" in the prose — each one is
  either answered by the user (and the answer reflected) or left as `⚠️ Not confirmed`.
- No Swift code, no `requirements.md`/`design.md`/`tasks.md`, and nothing under `Docs/Knowledge/` was
  created.
- The document is entirely in English (except quoted UI copy, which carries an English gloss).
