# Step-by-step Flow in design.md

> Every "Step-by-step Flow" section in `design.md` has a **matching Mermaid drawing in `flow.md`**
> (sequence diagram first, flowchart for branching logic). Read `flow_diagram.md` for the drawing
> conventions and the "mirror 1:1, add/drop no step" rule. This file only governs the text in
> `design.md`.

## Why this section exists

A one-sentence user story ("the user registers with email + password") cannot accurately describe a
multi-step backend flow — it silently hides steps like "check the email does not already exist
before hashing the password", "open a transaction before writing 2 tables", or "send the email after
commit, not before". This is exactly the kind of detail that makes an AI (or a developer) implement a
step too few or in the wrong order, because nothing in the spec names them.

Step-by-step Flow fixes that by requiring every external call (DB, external API, storage, mailer,
cache) in the flow to be written as its own step, with its trigger, pre-condition, and both the
success and failure branches — so nothing is assumed or made up later.

## When it is mandatory

Per rule 5 of `SKILL.md`: **whenever the feature's main flow makes 2 or more sequential calls to
external systems** — DB query/transaction, external HTTP API, object storage, mailer/SMS, cache,
message queue — even when part of it belongs to another service/module (e.g. this feature only calls
another module's usecase, not over HTTP).

If the flow has exactly one side-effect step (one DB query, no transaction, no external call), it may
be replaced by a short "Data flow" paragraph.

Quick test: if you have to write the word "then" when describing the flow, or a transaction spans
several tables, or a side effect happens after the DB write — you need this section.

## Required shape of each step

```
Step N — <short name>
- Trigger: <which request/handler, or which state, makes this step run>
- Pre-condition: <what must be true beforehand — e.g. validation passed, token still valid,
  transaction already open>
- Call: <kind> <details> (<main fields>)
  <known constraints, or the confirmed/unconfirmed details of this step itself>
    - Success → <what happens, which step comes next>
    - Failure (<specific case>) → <which HTTP status / error code / domain error, is there a rollback>
    - Failure (<other case, if any>) → <...>
```

For a step owned by another module/service, say so explicitly (e.g. "Step 3 — Send verification
email (owned by `pkg/mailer`, outside this feature's scope)") and point to that part's spec instead
of repeating its details.

## Worked example (built in the US-001 Register format)

```
Step 1 — Validate request (this feature)
- Trigger: handler receives POST /v1/auth/register
- Pre-condition: body parses, JSON is valid
- Call: go-playground/validator on RegisterRequest (email, password >= 8, phone_number E.164,
  dharma_name optional, ordination_place optional)
    - Success → Step 2
    - Failure (validation) → 400 with field errors, stop; the DB is not touched

Step 2 — Check whether email/phone already exist (this feature)
- Trigger: Step 1 passed
- Pre-condition: input validated
- Call: repository.FindByEmail + repository.FindByPhone (2 queries, may be merged into 1 OR query)
  Confirmed: dedupe relies on the DB UNIQUE constraint, not only on this earlier check — this check
  only exists to return a friendly error; a race is still caught by the unique violation on INSERT.
    - Success (does not exist) → Step 3
    - Failure (email taken) → domain.ErrEmailTaken → 409, stop
    - Failure (phone taken) → domain.ErrPhoneTaken → 409, stop

Step 3 — Hash password (this feature)
- Trigger: Step 2 passed
- Pre-condition: plaintext password passed the length validation
- Call: bcrypt.GenerateFromPassword (cost from config)
    - Success → Step 4
    - Failure (hash error) → 500, log; nothing written to the DB

Step 4 — Create user (this feature)
- Trigger: Step 3 passed
- Pre-condition: password_hash available
- Call: INSERT INTO users (...) RETURNING id
    - Success → Step 5
    - Failure (unique violation from a race) → map to 409 as in Step 2, stop
    - Failure (other DB error) → 500, log; nothing is partially written (single INSERT, no
      transaction needed)

Step 5 — Issue tokens (this feature)
- Trigger: Step 4 passed
- Pre-condition: user.id and role available
- Call: pkg/jwt generate access token (15m) + refresh token (30 days), store the refresh token in
  the refresh_tokens table (or devices)
    - Success → response 201 {access_token, refresh_token, user}
    - Failure (token generation/write error) → 500. Note: the user WAS created in Step 4 and is not
      rolled back — acceptable because the user can log in again; state this decision explicitly,
      do not leave it implied.
```

The example shows the most important property of the section: **it forces the unclear spots to be
said out loud** — if Step 5 fails, is the created user rolled back? If the spec does not answer, the
implementer picks, and two implementers will pick differently.

## How an unsettled detail is shown

If a step's contract/behavior is not confirmed yet, mark it right at that step (not only in
`requirements.md`), because this is the most specific place for a technical detail:

```
- Call: UPDATE users SET avatar_url = ... after the GCS upload completes
  ⚠️ Not confirmed: if the DB UPDATE fails after the file was uploaded to GCS, is the GCS file
  deleted (compensating action) or left as an orphan object? — blocks tasks.md Phase 3 completion.
```

When the user answers, rewrite that step as a confirmed statement and record it in `changelog.md` —
stating clearly whether the answer changed the number of steps/the flow structure, or only removed
one uncertain point (both are valid; say which one it was).
