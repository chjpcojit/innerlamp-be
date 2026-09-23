# Concept types and section mapping

Seven `type:` values map onto the bundle's folder structure. Pick the type by asking what the
concept is *about* — the subject, not the file it happens to sit near.

## The types

| `type` | Section | File location | Subject |
|---|---|---|---|
| `Feature` | `features/` | `docs/knowledge/features/<name>.md` | One user-facing feature/flow end to end |
| `API Endpoint` | `api/` | `docs/knowledge/api/<domain>-<action>.md` | One route's contract |
| `Domain Entity` | `entities/` | `docs/knowledge/entities/<name>.md` | One domain model |
| `UseCase` | `usecases/` | `docs/knowledge/usecases/<name>.md` | One business action / business rule set |
| `Architecture Pattern` | `architecture/` | `docs/knowledge/architecture/<name>.md` | A cross-cutting pattern many features follow |
| `Package` | `packages/` | `docs/knowledge/packages/<name>.md` | One `pkg/` infrastructure package and its interface |
| `Migration` | `migrations/` | `docs/knowledge/migrations/<number>-<slug>.md` | One SQL migration / schema change |

## Choosing between the close calls

**Feature vs UseCase.** A `Feature` concept is the entry point and covers the whole vertical slice —
route, handler, usecase, repository, entity, error mapping, wiring in `cmd/main.go`, pending work.
A `UseCase` concept is written only when the business logic is worth documenting on its own: it is
consumed by more than one feature, or it encodes rules a reader would otherwise have to
reverse-engineer from code (an OTP attempt counter, a lockout policy, a retry/backoff rule). A
UseCase used by exactly one simple feature gets a row in that feature's Architecture table and no
file of its own.

**Feature vs API Endpoint.** The `Feature` concept describes what the caller experiences and links
to the endpoint concept. The `API Endpoint` concept describes the wire contract — method, path,
request struct + `validate` tags, response struct, error→status table — and links back to the
feature(s) that call it. When the contract is still unconfirmed (a "⚠️ Not confirmed" marker in
`docs/specs/`, or no spec at all), there is nothing to write: create no `api/` file at all and
record the gap in the feature's Pending work. Do not create an endpoint concept containing a
guessed shape.

**Feature vs Package.** A `Feature` is a vertical slice through `services/appService/`. A `Package`
is a horizontal, reusable thing under `pkg/` — `config`, `database`, `jwt`, `storage`, … — that any
service can consume. If the code lives under `pkg/`, it is a Package concept, even when exactly one
feature uses it today. Write which `pkg/` packages actually import it — establish that from the
real imports (`grep` for `"chjpcoj.com.innerlamp/pkg/..."`), not from the README, which has a
history of describing dependencies that do not exist in the code).

**Migration vs Domain Entity.** A `Domain Entity` documents a Go struct. A `Migration` documents the
SQL that creates or alters its tables — columns, constraints, indexes, seed rows. They are separate
concepts because they drift independently: a struct can gain a field the migration never added, or
a column can carry a CHECK constraint the entity does not enforce. Link them to each other and state
any mismatch — that mismatch is the single most valuable thing either concept can record.

**Architecture Pattern vs everything else.** Write one of these only for a pattern that is genuinely
cross-cutting and stable — the `Register(type, FactoryFunc)` + `init()` driver registry in `pkg/`,
the `Connect(ctx)`-not-in-constructor lifecycle rule, the dependency direction enforced by
`.claude/skills/golang-clean-arch-tdd/scripts/check_dependency_rule.sh`, the consumer-defined interface
at the `delivery` boundary. If it
describes one feature, it is a Feature. These files are written once and rarely updated; a change
here usually means a foundational refactor, and the concept should say which features are affected.

## Section folders

`docs/knowledge/features/` exists and is populated (`register.md`, `login.md`);
`docs/knowledge/glossary/` exists but is empty. No other section folder exists yet, and `index.md`
and `log.md` at the bundle root are populated. `features/` is therefore already in `index.md`'s
active list.

Create a section folder the first time a concept needs it, then update `index.md` to move that
section out of the planned list and into the active list (`bundle_management.md`). Do not create
empty section folders ahead of need — the planned list in `index.md` is what records the intent.

> `glossary/` is present but not owned by this skill — the seven types above do not include a
> glossary term. Leave it alone unless the user asks for it explicitly.

## One concept per thing

Do not merge two features into one concept, and do not split one feature across several. A concept
that covers "register + login" makes both harder to find and makes `resource:` ambiguous. Instead,
write the register concept and link to a `login` concept; if the login concept does not exist yet,
the link stands as a marker (see `bundle_management.md` on dangling links).

The same applies to a package and its drivers: `pkg/database` is one concept, `pkg/database/postgresql`
is a subpackage it documents in its Architecture table, not a second file — unless the driver has
enough independent surface to justify its own concept, in which case say so explicitly at the top of
`packages/database.md`.
