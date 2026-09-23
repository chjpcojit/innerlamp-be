# Gathering questions before generating

Most of what a concept needs is discoverable by reading the code and the bundle. Ask the user only
for what is genuinely not derivable — chiefly *intent*, and *whether the thing is verified working*,
since neither is visible in source.

Ask at most two or three questions in one message, and only ones whose answer changes what you
write. If you can determine the answer with a `Glob` or a read, do that instead of asking.

## The question set

### 1. Scope — which thing, and which type

Ask only if the request is ambiguous about what it covers.

> Document the register flow as a Feature concept, or just `RegisterUsecase` as a UseCase concept?
> A Feature concept covers the whole vertical slice — route, handler, usecase, repository, entity;
> the UseCase concept covers just the business rules of one file.

Try to resolve this yourself first: if the user named a route or a user-facing flow, that is a
Feature. If they named a Go type (`RegisterUsecase`, `User`), that is the type's concept. If they
named something under `pkg/`, it is a Package.

### 2. Is it actually implemented and working?

This is the one question that genuinely cannot be answered by reading code, because "the code
exists" and "the flow works against a real backend" are different facts, and `status:` depends on
the second.

> Is the register flow verified working end-to-end against a real or staging Postgres yet, or still
> stubbed? This sets `status: stable` vs `draft`, and whether the concept needs a Pending work
> section.

If the user does not know, default to `draft` and say so. Never default to `stable`.

In this repo the answer is now feature-dependent, so read the code rather than assuming:
`onboarding` (register) is exercised end-to-end with dev placeholders and stays `draft`;
`login` has a passing black-box E2E audit and is `stable`; `content`/`journal`/`practice` have no
code at all and get no concept yet. For `pkg/`, `stable` is reachable for a package whose own test
suite passes (e.g. `pkg/jwt`), though no Package concept has been written yet. When in doubt,
`draft`.

### 3. Update vs create, when the file already exists

If the target concept exists, confirm what is being changed rather than assuming:

> `features/register.md` already exists. Update it to reflect the newly-wired OTP resend endpoint,
> or leave it and add a separate concept for that flow?

### 4. Human verification

Only relevant when updating a concept that carries a `verified:` field. Do not ask whether to add
one — this skill never writes that field.

> `packages/database.md` is marked verified by a human. Your update changes the Architecture table,
> so that attestation is now stale — flag it in your PR description so a reviewer re-confirms?

## What not to ask

Do not ask the user for any of these; they are all discoverable:

| Not a question | How to find it |
|---|---|
| Which files implement the feature | `ls services/appService/<layer>/`, read them |
| The `resource:` path | verify it on disk |
| Which layers exist and what they are named | `ls services/appService/` — then read the layer `README.md` (may be stale; `ls` wins) |
| What is stubbed | read the usecase file — a body returning `errors.New("not implemented")` is self-evident |
| What the API contract is | read the handler's request/response structs and its error→status map |
| Which `pkg/` packages a package imports | `grep` its real imports, not the README |
| Which concepts already exist | read `index.md` |
| What `tags:` to use | grep existing concepts for `tags:` |

## Phrasing rules

- **One topic per question.** Do not stack two decisions into one sentence.
- **Offer the concrete options.** "Feature concept or UseCase concept?" beats "how should I
  document this?"
- **State the consequence.** Naming what the answer changes ("this sets `status: stable` vs
  `draft`") lets the user answer correctly without knowing the convention.
- **Suggest a default.** "If unsure, I'll mark it `draft`" gives a safe answer path.
- **Keep it short.** A question the user has to scroll to read is a question they will answer badly.

## When the user says "just do it"

Proceed, and record every unresolved decision explicitly in your response — not silently. For each
one, state what you chose and what would change it:

- Unverified end-to-end status → wrote `status: draft`, with the unverified parts listed under
  Pending work. Promote to `stable` once the flow is exercised against a real backend.
- Ambiguous scope → documented the Feature, not the individual entities; say so, and note that
  entity concepts can be added later if the models turn out to be shared.
- A `pkg/` package with no test file → noted in the concept's Pending work that the behavior is
  unverified; `status` cannot rise above `draft` on inspection alone.

Never let a default become an invisible assumption. The concept is a reference that others will
trust, so the parts you guessed must be visible to whoever reads it next.
