# Normalization & Inconsistency Detection

The BA skill's core value is *not* pretty formatting — it is turning a messy input into a consistent,
trustworthy SRS and surfacing everything that does not add up. This file covers both: how to restructure
input into the IEEE skeleton, and how to find the problems that must become questions.

## Normalizing input

### From a raw description (idea / bullets)
1. Extract every actor, action, and object the description implies.
2. Expand each implied behavior into a candidate functional requirement (`FR-*`) with a clear
   subject and verb, in EARS form: `WHEN <trigger> THE SYSTEM SHALL <behavior>`.
3. Sort the candidates into the §3.2 feature areas. Anything that reads as a quality (fast, secure,
   reliable) becomes an `NFR-*`, not an `FR-*`.
4. For each `FR-*`, ask "what is the failure/empty/offline behavior?" — the answer is usually
   missing and becomes a question.

### From an existing BA / requirements document
1. Read the whole document once to build a mental model before editing anything.
2. Build a "requirement inventory" — every statement that imposes a behavior, rule, or constraint —
   with a note of where it came from (section / page / paragraph).
3. Map each inventory item into the IEEE skeleton, preserving its source reference (this feeds the
   traceability matrix). Drop nothing: if an item does not fit cleanly, it is either out of scope
   (list it under §1.2) or a duplicate (merge, keep one source reference).
4. Do not silently reword to "fix" problems — a detected problem becomes a question or a `⚠️`
   marker, never a quiet cleanup that hides the discrepancy.

## Detecting problems

### Contradictions
Two places state different rules for the same situation (e.g. §2 says "password ≥ 8 chars", the
acceptance criteria says "≥ 6"). Detect by scanning the inventory for overlapping triggers with
different outcomes. Surface the two conflicting statements verbatim and ask which is authoritative.

### Terminology drift
The same concept named differently in different places (OTP vs "verification code" vs "SMS code").
Detect by looking for synonyms used interchangeably. Resolve by picking one canonical term and
recording it in the glossary (§1.4), then asking the user to confirm the canonical name.

### Ambiguity
- Vague qualifiers: "sometimes", "maybe", "etc.", "as needed", "appropriate", "reasonable", "some".
- Undefined acronyms or domain terms (dharma name, ordination place) used without a definition.
- Pronouns without antecedents ("it", "the user" when there are several user classes).
Each ambiguity becomes a targeted question or a glossary entry.

### Silent gaps
- No error / empty / offline state described.
- No permissions or roles stated for a multi-actor feature.
- No non-functional requirements at all (a feature with only happy-path `FR-*` items is a red flag).
- A dependency implied but never named (e.g. "send an OTP" without saying which feature owns OTP).

### Edge cases
For every `FR-*`, ask the boundary questions from `references/gathering_questions.md` §1. Empty
input, duplicate taps, retries, timeouts, backgrounding, and two-device scenarios are the ones most
often missing from a first draft.

### Technical risks (PM/BA-level, not design)
Flag — do not design — anything that may be infeasible or costly on iOS: background processing
limits, App Store review risk (e.g. collecting data without disclosure), required permissions
(camera, location, contacts), offline persistence complexity. These become questions framed at the
PM/BA level ("this implies offline sync — is that in scope?").

## Presenting findings

Before asking questions, show the user:
1. A 3–5 line plain-language restatement of what you understood (so they can catch a
   misunderstanding early).
2. The detected problems, ordered by blocking impact: **contradictions first** (they invalidate the
   spec), then **gaps and edge cases**, then **ambiguity and terminology**, then **risks**.

Each finding should quote the offending text (or paraphrase the gap) so the user can react without
re-reading their own document.
