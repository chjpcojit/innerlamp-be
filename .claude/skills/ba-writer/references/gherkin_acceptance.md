# Gherkin Acceptance Criteria

Every functional requirement (`FR-*`) in §3.3 carries its acceptance criteria as Gherkin scenarios.
The EARS sentence says *what the product shall do*; the scenarios say *how anyone — on any platform —
can tell it does*. Each platform spec turns these into its own platform-level criteria and tests, and
cites the `FR-*` ID.

## Format

Plain Gherkin in a fenced ```` ```gherkin ```` block under the FR's **Acceptance criteria** bullet.
English keywords and text.

```gherkin
Scenario: Add a relative to the memorial book
  Given the user is signed in
  And the memorial book is empty
  When the user saves a relative named "Nguyễn Văn A" (gloss: a full name) with a lunar date of death
  Then the memorial book lists "Nguyễn Văn A"
  And the entry shows the days left until the next anniversary

Scenario Outline: Reject a name that is empty after trimming
  Given the user is adding a relative
  When the user saves with the name "<name>"
  Then the user is told the name is required
  And nothing is added to the memorial book

  Examples:
    | name |
    |      |
    | "  " |
```

Skip the `Feature:` header. Use `Background:` only when three or more scenarios of one FR share the
same `Given`.

## Platform-neutral wording

- Actions are intent verbs: **opens, chooses, enters, saves, submits, cancels, goes back** — never
  tap, click, swipe, long-press.
- Outcomes are what the user sees or can later observe ("the entry appears", "the user is told…",
  "after signing in on another device the entry is still there") — never what code, storage or the
  network does.
- No endpoint paths, field names, error codes, HTTP statuses, screen class names or control ids.
- If the outcome legitimately differs per platform, write the common part here and record the
  difference in §3.6 Platform variance; do not branch inside a scenario.
- A scenario that only one platform can exercise (e.g. an OS-integration) is tagged with the platform:
  `@ios-only`, `@android-only`, `@web-only`, `@backend-only` — and must match a §3.6 row.

## Coverage

For each FR, in this order:

1. **One happy path.**
2. **One scenario per edge case or failure** found during gathering: empty input, boundaries,
   duplicates, expired, offline / server unreachable, not signed in, not permitted, repeated submit,
   interrupted flow, second device.
3. **One scenario per business rule** that changes the outcome — cite it in the scenario name, e.g.
   `Scenario: Day 30 in a short lunar month (BR-MEM-04)`. Detailed rule cases live in the vector file,
   not here; one representative scenario per rule is enough.

If an FR has no failure path, say so in a one-line comment rather than inventing one. Use
`Scenario Outline` + `Examples` when only data changes, not behavior.

## Unconfirmed behavior

Never invent an outcome. If the expected result depends on an open question:

1. Tag the scenario `@not-confirmed`.
2. Write what is known and stop the `Then` at what is certain.
3. Add `> ⚠️ Not confirmed: <question> — <what it blocks / which platforms>.` right after the block.
4. List it in Appendix A.

When answered: remove the tag, complete the scenario, move the question to "resolved".

## Traceability

The §4.2 matrix note column carries the scenario count per FR (e.g. `3 scenarios`). Platform specs
cite `FR-*` IDs on their own criteria so a reviewer can walk from product scenario to platform test.
