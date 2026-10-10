# Gherkin Acceptance Criteria

Every functional requirement (`FR-*`) in SRS section 3.2 carries its acceptance criteria as Gherkin
scenarios. The EARS sentence in **Description** says *what the system shall do*; the scenarios say
*how a reviewer or tester knows it does*. They share the same `FR-*` ID, so they cannot drift apart
without the traceability matrix showing it.

## Format

Use plain Gherkin inside a fenced ```` ```gherkin ```` block, indented under the FR's
**Acceptance criteria** bullet. English keywords and English text (project language rule).

```gherkin
Scenario: Sign up with a valid phone number
  Given the user is on the sign-up screen
  And the phone number field contains a valid number
  When the user taps "Continue"
  Then the user is asked to enter a verification code

Scenario Outline: Reject a malformed phone number
  Given the user is on the sign-up screen
  When the user enters "<phone>" and taps "Continue"
  Then an error says the phone number is not valid
  And the user stays on the sign-up screen

  Examples:
    | phone     |
    | 123       |
    | abcdefghi |
```

Skip the `Feature:` header: the SRS section already names the feature area. Use `Background:` only
when three or more scenarios of one FR share the same `Given`.

## Coverage

For each FR write, in this order:

1. **One happy path.**
2. **One scenario per edge case or failure** found in workflow step 3 (empty input, boundary values,
   duplicate, expired, offline, permission denied, repeated tap, interrupted flow).
3. **One scenario per business rule** that changes the outcome (limits, ordering, eligibility).

If an FR has no failure path, say so in a one-line comment rather than inventing one. Use
`Scenario Outline` + `Examples` when only the data changes, not when the behavior changes.

## Wording rules

- **Observable behavior only.** `Then` describes what the user sees or what state the app is in, never
  how it is built. No class names, endpoint paths, JSON field names, error codes, or UI control ids.
  Those belong to the API contract and `design.md`.
- **One `When` per scenario**, one behavior per scenario. A second `When` means a second scenario.
- **Concrete values** in `Examples` where the rule depends on them; do not write "valid input" if the
  valid range is the whole point. If the range is unknown, see the next section.
- **User-facing text** is quoted with its original language and an English gloss, as elsewhere in the SRS.

## Unconfirmed behavior

Never invent an outcome to complete a scenario. If the expected result depends on an open question:

1. Tag the scenario `@not-confirmed` on the line above it.
2. Write the part that *is* known and stop the `Then` at what is certain.
3. Add the inline marker directly after the block:
   `> ⚠️ Not confirmed: <the question, in one sentence> — <what it blocks>.`
4. List the same question under Appendix A with its "what it blocks" note.

When the user answers, remove the tag, complete the scenario, and move the question to "Open questions
(resolved)" as usual.

## Traceability

The **Test** column of the traceability matrix (section 4.2) lists the scenario count per FR (for
example `3 scenarios`). It is filled in by this skill, not left as `TBD`. `il-spec-writer` takes the SRS as input,
so the scenarios are available when it writes the EARS criteria in `requirements.md`; the unit and UI
tests that implement them are written later, in the build steps.
