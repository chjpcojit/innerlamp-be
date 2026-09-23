# OKF frontmatter schema

Every concept in `docs/knowledge/` opens with YAML frontmatter delimited by `---`. This file
documents each field, its allowed values, and the rules that keep the trust tiers meaningful.

## Required fields

```yaml
---
type: Feature | API Endpoint | Domain Entity | UseCase | Architecture Pattern | Package | Migration
title: Human-readable name, title case (e.g. "Sign Up", "pkg/database", "0001 Register Onboarding")
description: One sentence — what it is and what it does. Used by index.md and by agents deciding whether to read further.
resource: <repo-relative path>
tags: [kebab-case, ...]
status: draft | stable | deprecated
generated: { by: <agent-name>, at: <ISO-8601 with offset> }
---
```

### `type`

Exactly one of the seven values listed above, spelled and capitalized as shown. It determines the
bundle section the file lives in — see `concept_types.md`.

### `title`

Title case, human-readable, no trailing punctuation. "Sign Up", not "signup" or "RegisterHandler".
For a package, use the import path tail (`pkg/jwt`, `pkg/storage`). For a migration, use the
migration's number and slug (`0001 Register Onboarding`), matching the SQL filename.

### `description`

One sentence. Present tense. Says what the thing is and the single most useful fact about it —
usually its user-visible purpose, its contract, or its role in the architecture. Match the density
of these examples:

```yaml
description: New-user registration flow — request-otp, resend-otp, and verify-otp, writing no users row until the OTP is confirmed.
description: Provider-agnostic database abstraction with self-registering drivers for Postgres, MySQL, and MongoDB.
description: Creates the users, otp_codes, pending_registration, and user_sessions tables that US-001 writes to.
```

### `resource`

The repo-relative path to the code this concept describes. **Verify it exists before writing it**
(see `resource_verification.md`). Trailing slash for a folder, the filename for a single file:

```yaml
resource: pkg/jwt/                                          # package — the folder
resource: pkg/jwt/jwt.go                                    # interface — the file declaring it
resource: services/appService/usecase/register_usecase.go   # usecase — a file
resource: migrations/0001_register_onboarding.sql           # migration — the .sql file
```

Omit `resource:` entirely only for the pre-implementation complex-business-rule draft described in
the Overview of `SKILL.md`. A concept without `resource:` cannot be drift-checked and must stay
`status: draft`.

### `tags`

Kebab-case, lowercase. Reuse tags that already exist in the bundle before inventing new ones —
grep the existing concepts for `tags:` first. Suggested starting set for this repo: `auth`,
`onboarding`, `content`, `infra`, `postgres`, `gcp`, `networking`, `validation`, `migration`,
`architecture`, `domain`, `usecase`, `package`, `api`.

### `status`

| Value | Meaning |
|---|---|
| `draft` | Something described is stubbed, unwired, or not yet exercised end-to-end. Must be paired with a "Pending work" section stating what. |
| `stable` | The behavior described was exercised end-to-end and no known gaps remain. |
| `deprecated` | The concept describes code that has been removed or superseded. Keep the file if it has inbound links worth redirecting; otherwise delete it and its index row. |

`draft` is the correct default whenever a usecase method returns a not-implemented error or a
handler is not wired in `cmd/main.go`. Never promote to `stable` on the grounds that the code
"looks complete" or that tests pass — the criterion is exercise against a real or staging backend,
not inspection.

Note that a `Package` concept is often the one type that can legitimately reach `stable` today:
`pkg/*` packages are compiled and tested independently of any service. `stable` still requires the
package's behavior to have been exercised — its own test suite passing is sufficient evidence for a
pure library, and the concept should say which command proved it.

### `generated`

```yaml
generated: { by: claude-code, at: 2026-09-17T14:32:00+07:00 }
```

`by` names the agent that wrote the current revision. Use `claude-code` when this skill authors the
file. If an existing file carries a different producer string from an earlier revision, leave it
alone unless this skill rewrites the file, in which case `by` becomes `claude-code`.

`at` is ISO-8601 with the local UTC offset. Update it on every revision.

## Optional fields

### `verified`

```yaml
verified: { by: human:ducnguyen, at: 2026-09-18T09:15:00+07:00 }
```

A human attestation that the concept was reviewed against the code in a PR. **This skill never
writes, creates, or alters this field** — an AI-authored `verified` is a lie that defeats the
field's entire purpose, since its value is precisely that a human looked.

If a concept you materially rewrote already carries `verified:`, tell the user in your response
that the attestation predates your change and should be re-confirmed in review. Do not remove it,
do not bump it.

A concept with `verified:` is a trusted reference. A concept with only `generated:` is
AI-authored and unreviewed. The presence or absence of this one field is the trust tier — there is
no separate field for it.

### `sources`

```yaml
sources:
  - path: docs/specs/features/onboarding/requirements.md
    kind: spec
  - path: docs/specs/features/onboarding/design.md
    kind: spec
  - path: pkg/jwt/README.md
    kind: implementation-notes
  - path: migrations/0001_register_onboarding.sql
    kind: migration
```

Where the concept's content came from. Include this whenever a spec or a README existed and was
actually used — it lets a future reader trace a claim back past the concept.

Allowed `kind` values:

| `kind` | Use for |
|---|---|
| `spec` | Anything under `docs/specs/` |
| `implementation-notes` | A folder or package `README.md` |
| `code` | A source file read directly, with no spec or README involved |
| `migration` | A `.sql` file under `migrations/` |
| `erd` | A schema diagram doc under `docs/specs/db/` (the folder exists but is currently empty — drop the entry while that holds) |
| `openapi` | An OpenAPI spec, typically `docs/specs/api/openapi.yaml` |

Omit `sources:` only when the concept was derived purely from reading code with no spec or README
involved.

### `stale_after`

```yaml
stale_after: 2027-03-17
```

A date after which the concept should be re-checked even if nothing appeared to change — useful for
concepts describing external contracts (a third-party API that may drift, an SDK version, a
provider's IAM behavior). Use sparingly; most concepts go stale by code change, not by clock, and a
date nobody honors is noise. Migration and domain-entity concepts almost never need one.

## Full example

```yaml
---
type: Package
title: pkg/jwt
description: JWT generation, verification, claims, and HTTP/gRPC auth middleware, provider-agnostic and backed by a swappable signing key source.
resource: pkg/jwt/jwt.go
tags: [infra, auth, package]
status: stable
generated: { by: claude-code, at: 2026-09-17T14:32:00+07:00 }
sources:
  - path: pkg/jwt/README.md
    kind: implementation-notes
---
```
