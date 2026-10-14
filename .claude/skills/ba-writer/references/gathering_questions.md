# Gathering Questions

The skill is two-way: it asks a short, prioritized set of questions *before* writing the product spec.
This file defines the categories, the phrasing style, and the "never skip" rule.

## Phrasing style

- **Ask in Vietnamese** (anh/em), technical terms kept in English. Only the spec file is English.
- **One topic per question**, answerable in a word or a short choice. Offer a suggested default or an
  example and mark it as the recommendation.
- **Short** — one or two lines; one sentence of context if needed.
- **Say why it matters and which platforms it blocks** ("quyết định này đổi cách iOS và Android tính
  ngày giỗ").

## Categories (ask the ones that apply)

### 1. Cross-platform parity — ask first when more than one platform is in scope
- Which platforms get the feature now, which later, which never.
- Each cross-platform disagreement found while reading the repos: quote both sides, ask which is the
  product rule.
- Decisions recorded only in one platform's assumptions (iOS `AS-*`, Android `*_assumptions.md`,
  website `.assumptions.md`): is this a product rule for everyone, or a platform-local choice?
- OS integrations that exist on one platform only (OS reminders, widgets, share sheet, biometrics):
  equivalent elsewhere, drop, or platform-only (→ §3.6)?
- Is the user's data the same across devices and platforms once signed in?

### 2. Edge cases & boundary behavior
- Empty / missing / invalid input; boundary values and whether limits are inclusive.
- Duplicate actions (repeated submit, re-sending a code while one is pending).
- Offline / server unreachable / slow — per platform if they may differ.
- Concurrency: two devices, two sessions, editing the same item.
- Empty and error states the user sees.
- Dates: time zone, calendar (solar/lunar), leap months, day 29/30/31, today vs tomorrow at midnight.

### 3. Business rules
- Validation rules with exact parameters (lengths, ranges, formats).
- Permissions and roles (who sees / does what).
- Limits and quotas (per user, per day, per item).
- Rules with exceptions — force the "unless" to be spelled out.
- For every computational rule: the authority for expected values (reference table, confirmed
  implementation, the user's own examples).

### 4. Non-functional requirements
- User-perceived performance targets.
- Behavior when the backend is down.
- Privacy: personal data involved, who may see it, deletion with the account.
- Policy constraints (app store review, religious content, legal).

### 5. Data
- What the user creates or sees, whether it belongs to the account (follows the user across devices)
  or to one device, retention and deletion.
- Migration of data that already exists on some platform (keep, upload once, discard).

### 6. Dependencies & integration
- Other InnerLamp features this depends on or changes (sign-in, profile, temples, calendar…).
- External services by role (SMS, maps, storage for photos) and whether they exist yet.
- Ordering constraints across platforms (backend contract before client work, etc.).

### 7. Technical feasibility & risk (flag, don't design)
- Anything costly or risky on some platform (background execution, notifications, offline sync,
  large media) — framed as a scope question.
- Anything that needs a hard-to-reverse architecture decision (infrastructure, new data model,
  offline sync, auth) with no Accepted ADR: do not ask the user to pick the design here — ask the
  three-way question of SKILL.md hard rule 11 (write the ADR with `adr-writer` first / continue with
  a `⚠️ Not confirmed` / not architectural), and ask it before every other question.

## The "never skip" rule

Every applicable question is either answered or, if skipped, recorded inline as:

```
> ⚠️ Not confirmed: <the question, in one sentence> — <what it blocks / which platforms>.
```

Never invent the answer and never drop the question because it is awkward. When answered later, move
it to "Open questions (resolved)" with date and resolution, and note the downstream effect in the
changelog.

## Prioritization

Most blocking first: cross-platform disagreements and scope, then questions that change data
ownership or business rules, then edge cases, then wording. Batch by category if there are many.
