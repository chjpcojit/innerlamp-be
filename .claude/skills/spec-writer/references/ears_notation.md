# EARS notation for requirements.md

EARS (Easy Approach to Requirements Syntax) is the required phrasing style for every acceptance
criterion in `requirements.md`. It forces each requirement to name a trigger and a mandatory system
response, which is what makes requirements directly testable for both a human QA and an AI
implementing them.

In a backend spec the "system" is the HTTP API (`services/appService`), so the trigger is usually an
incoming request or a server-side condition, and the response is a status code + response body +
persisted state change — not a screen.

## The pattern

```
WHEN <trigger/event>, THE SYSTEM SHALL <mandatory behavior>.
```

Variants used in this project's specs:

- `WHEN <event>, THE SYSTEM SHALL <behavior>.` — the default form for a single trigger → single
  response.
- `WHEN <event> ..., THE SYSTEM SHALL <behavior>, then either: - on success: ... or - on failure
  (<condition>): ...` — for a trigger with a success/failure branch (a request that can be
  rejected).
- `WHEN <condition holds> (a state, not a discrete event), THE SYSTEM SHALL <behavior>.` — for
  conditions like "WHEN the request carries no valid access token" or "WHEN the same
  idempotency key was already processed".

## Good vs bad examples

**Bad** (vague, not testable, hides which layer is responsible):
> The register endpoint should validate the input.

**Good** (EARS — names the trigger and the exact response):
> WHEN a `POST /v1/auth/register` request arrives with a password shorter than 8 characters, THE
> SYSTEM SHALL reject it with `400 Bad Request` and error code `password_too_short`, and SHALL NOT
> create a user row.

**Bad** (bundles multiple behaviors into one criterion, hiding branches):
> Registering creates the account and handles duplicate email or phone.

**Good** (spells out both branches explicitly):
> WHEN a `POST /v1/auth/register` request arrives with all fields valid, THE SYSTEM SHALL hash the
> password, insert the user row, and return `201 Created` with an access token and a refresh
> token, then either:
> - on success: ... (as above), or
> - on failure (email or phone already exists): return `409 Conflict` with error code
>   `email_already_registered` / `phone_already_registered`, and SHALL NOT create a second row.

## Rules

1. **One criterion = one trigger.** If a single request has multiple distinct outcomes
   (success/failure, or several error codes), write them as branches of the same criterion (as in
   the example above), not as separate unrelated bullets that lose the shared trigger.
2. **Always name "THE SYSTEM SHALL"** — even though it reads slightly formal/repetitive, this is
   intentional: it keeps every criterion phrased as a testable system obligation, not a passive
   description.
3. **Name the observable contract.** A backend criterion is only testable if it fixes the status
   code, the error code from `domain/errors.go`, and the persisted effect (which row/column
   changes, or explicitly that nothing is written). "Returns an error" is not a criterion.
4. **Never leave a hidden step implicit.** If a trigger causes a validation check, a duplicate
   check, a transaction, or any other side effect before the visible response, that check gets its
   own criterion or its own explicit clause — see `step_by_step_flow.md` for how this maps into
   `design.md`. `requirements.md` states *that* the check happens and what the caller observes;
   `design.md` states *how* it's implemented (which usecase method, which repository call, in which
   order, inside which transaction).
5. **State which auth requirement applies.** Say plainly whether the endpoint is public read,
   requires a valid access token, or requires ownership of the resource. If token verification is
   designed but not yet wired into the route, that is a "Pending" fact, not a done criterion — it
   belongs in `requirements.md`'s Pending section and `tasks.md`, not silently folded into a
   passing criteria list. (This used to cite a real `/v1/users/me` gap; those routes no longer
   exist, so treat it as an example only.)
6. **Tag every criterion with the product ID it implements** — `(FR-MEM-002)`, `(BR-MEM-04)`, or
   `(platform)` for backend-only behavior with no product counterpart (an index, a header, a
   transaction rule). The tag is how a reviewer walks from `docs/product/<feature>/product.md` to
   this criterion and its test. See `upstream_product_spec.md`.
7. **Out of scope / Pending / Open questions are not EARS-phrased** — those are plain prose
   sections; EARS applies specifically to the "Acceptance Criteria" section.
