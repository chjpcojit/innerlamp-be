# Business Rules and Test Vectors

Business rules are where platforms drift: each one re-implements "the same" rule from prose and gets
an edge case slightly different. A `BR-*` states the decision once; a vector file makes every platform
prove it implements that decision identically.

## Business rule vs functional requirement

| | Business rule (`BR-*`) | Functional requirement (`FR-*`) |
|---|---|---|
| Says | a decision or calculation, true regardless of screen | what the user observes |
| Example | "A lunar day 30 falls on day 29 in a 29-day lunar month." | "The memorial list shows the days left until each next anniversary." |
| Cited by | FRs, platform specs, code, tests | platform specs, tests |

An FR that contains a rule ("…sorted by soonest anniversary, where day 30 becomes 29…") is split:
the rule goes to a `BR-*`, the FR links it.

## Writing a rule

```markdown
**BR-MEM-04 — Day 30 in a short lunar month**
- **Rule:** When a lunar anniversary falls on day 30 and that lunar month has only 29 days in the
  target year, the anniversary falls on day 29 of that month.
- **Examples:** day 30 of month 7, year with 29-day month 7 → day 29; same with 30-day month 7 → day 30.
- **Computational:** yes → `rules/next-anniversary.json`
- **Applies to:** iOS, Android, Website (Backend stores the date only)
- **Source:** user decision 2026-10-03; prototype `gioNext`
```

Rules of thumb:

- **One decision per rule.** "and" joining two decisions → two rules.
- **Name every boundary.** Inclusive or exclusive, which time zone, which calendar, what happens on the
  exact limit, what happens when input is missing.
- **Time zone and calendar are always explicit** for any date rule (InnerLamp's default product time
  zone is `Asia/Ho_Chi_Minh` unless the user decides otherwise).
- **Examples are mandatory**, including at least one edge case. If the user can't give the outcome of
  an edge case, it is a `⚠️ Not confirmed`, not a guess.

## When a rule is computational

A rule is computational when it maps inputs to a value that a test can compare: a date, a number, a
label chosen from a fixed set, an ordering, an eligibility yes/no. Validation limits (max length,
allowed range) count too when more than one platform enforces them.

Computational rules get a vector file. Pure policy ("only the owner may see an entry") does not —
it is verified by scenarios.

## Vector file format

Path: `docs/product/<feature>/rules/<rule-slug>.json`. One file per rule (or per tightly coupled group
of rules that share inputs). JSON, UTF-8, two-space indent.

```json
{
  "rule": ["BR-MEM-03", "BR-MEM-04"],
  "description": "Next anniversary date (solar) for a memorial entry, given today's date.",
  "product_spec_version": "1.0",
  "inputs": {
    "today": "solar date YYYY-MM-DD in Asia/Ho_Chi_Minh",
    "calendar": "lunar | solar",
    "day": "1-30 (lunar) or 1-31 (solar)",
    "month": "1-12",
    "leap_month": "boolean, lunar only"
  },
  "output": {
    "next_date": "solar date YYYY-MM-DD",
    "days_left": "integer >= 0"
  },
  "cases": [
    {
      "name": "solar day 31 in a 30-day month falls on the month's last day",
      "input": { "today": "2026-04-10", "calendar": "solar", "day": 31, "month": 4, "leap_month": false },
      "expected": { "next_date": "2026-04-30", "days_left": 20 }
    },
    {
      "name": "anniversary today counts as 0 days left",
      "input": { "today": "2026-11-15", "calendar": "solar", "day": 15, "month": 11, "leap_month": false },
      "expected": { "next_date": "2026-11-15", "days_left": 0 }
    }
  ]
}
```

- `inputs` / `output` describe each field in words so platforms can map them to their own types; the
  names are **test-local**, not the API contract.
- Each case has a unique `name` that reads as the behavior it proves.
- Cover: the happy path, every boundary named in the rule, every example in the `BR-*`, and every
  edge case found while gathering questions.
- **Never invent an expected value.** Compute it from an authority the user accepts (a reference
  table, an existing implementation the user confirmed as correct, or the user's own answer) and say
  which in the case's optional `"source"` field. If no authority exists, leave the case out and record
  `⚠️ Not confirmed` in the rule.
- When a rule changes, update the vectors in the same change and bump `product_spec_version`.

## How platforms use vectors

Each platform that implements the rule loads the JSON from this repo (by path in a checkout, or a
pinned copy refreshed by a script — never hand-edited) and runs every case in its unit tests. A
failing case on one platform is drift, recorded as `drift` in the §4.2 matrix until fixed.
