# Normalization & Inconsistency Detection

The skill's value is not formatting — it is turning scattered, partly contradictory material from up
to four repos into one product truth, and surfacing everything that does not add up.

## Normalizing input

### From a raw description
1. Extract every actor, action and object the description implies.
2. Turn each behavior into a candidate `FR-*` (EARS: `WHEN <trigger> THE SYSTEM SHALL <behavior>`)
   and each decision or calculation into a candidate `BR-*`.
3. Qualities (fast, private, reliable) become `NFR-*`.
4. For each FR ask "what happens when it fails / is empty / is offline / on a second device?" — the
   answer is usually missing and becomes a question.

### From an existing document (BRD, PRD, legacy SRS)
1. Read it whole once before mapping anything.
2. Build the requirement inventory with source references (section / paragraph).
3. Map each item into the skeleton, keeping its source for the traceability matrix. Drop nothing: an
   item that doesn't fit is out of scope (§1.2), a duplicate (merge, keep both sources), or
   implementation/contract detail (move to the handoff list, not the spec).
4. Legacy SRS files written for one platform (iOS `Docs/Specs/<Feature>/SRS.md`, this repo's
   `docs/specs/features/<feature>/SRS.md`) are inputs, not templates: lift their product content,
   leave their platform content behind.

### Retroactive, from several platforms
1. Read each platform's spec **and** assumptions, plus the code where the spec is thin.
2. For every rule, write down what each platform actually does, side by side:

   | Rule | Backend | Website | iOS | Android |
   |---|---|---|---|---|
   | Leap month | accepted and stored | — | dropped on load | not offered |

3. Where all agree → candidate `BR-*` (still confirm with the user if it was never an explicit
   decision). Where they differ → a cross-platform disagreement question. Where only one platform
   decided → a platform-only decision question.
4. The matrix in §4.2 records the outcome: `done`, `drift` (with a note), `planned`, `n/a`.

## Detecting problems

### Contradictions
Two sources state different rules for the same situation. Quote both verbatim and ask which is
authoritative.

### Cross-platform disagreement
The same rule implemented or specified differently on two platforms. Quote both sides (repo + file);
ask which is the product rule; the losing platform gets `drift` in §4.2 until fixed.

### Platform-only decisions
A choice in one platform's assumptions or spec that changes what the user sees (sort order, rounding,
limits, what "today" means). Ask whether it is a product rule (→ `BR-*`) or platform-local (→ §3.6 or
dropped as implementation detail).

### Terminology drift
The same concept with different names (memorial / Sổ giỗ / gio; OTP / verification code). Pick one
canonical English term, put it in the glossary with the Vietnamese term, ask the user to confirm.

### Ambiguity
Vague qualifiers ("sometimes", "etc.", "as needed", "reasonable"), undefined domain terms, pronouns
without antecedents, unstated time zone or calendar on a date. Each becomes a question or a glossary
entry.

### Silent gaps
- No error / empty / offline state.
- No roles or permissions for a multi-actor feature.
- No NFRs at all.
- A platform in scope with no statement of how it behaves.
- A dependency implied but never named.

### Edge cases
For every FR and BR, ask the boundary questions in `references/gathering_questions.md` §2.

### Technical risks
Flag, don't design: background limits, notifications, offline sync, media upload, store review. Frame
as scope questions.

## Presenting findings

Before asking questions, show the user:
1. A 3–5 line plain-language restatement, including which platforms are in scope.
2. The findings ordered by blocking impact: contradictions and cross-platform disagreements, then
   platform-only decisions and parity gaps, then gaps and edge cases, then ambiguity and terminology,
   then risks.

Each finding quotes the source (repo + file + the offending text) so the user can react without
opening it.
