# Bundle management — index.md, log.md, naming, cross-links

A concept is only useful if it can be found and if its history is traceable. Both files at the
bundle root carry that load, and both are updated in the same pass as the concept itself.

## index.md

`docs/knowledge/index.md` is the bundle's table of contents. It has three parts that this skill
maintains.

**It is already populated.** `index.md` has all three parts; update them in place rather than
rewriting the file. Do not reorder the concept table or change its column shape.

### 1. The introduction

A short prose block explaining what the bundle is and stating the distinction from `docs/specs/`
(mutable work-order specs vs durable post-implementation reference). There is no central
architecture doc to point at (`docs/ARCHITECTURE.md` was removed); do not reintroduce a link to
one unless a replacement is created. Update only if a structural fact changes.

### 2. The sections list

Two lists: active sections with a one-line description, and a "Planned, not yet populated" list.
When a concept creates a section's first file, move that section up into the active list and
describe it in one line:

```markdown
- [`packages/`](packages/) — one concept per `pkg/` infrastructure package (`type: Package`).
```

The seven sections are `features/`, `api/`, `entities/`, `usecases/`, `architecture/`,
`packages/`, `migrations/` (see `concept_types.md`). `docs/knowledge/features/` exists and holds
two concepts (`register.md`, `login.md`), so it is already in the active list; the other six
sections are still "Planned, not yet populated". `docs/knowledge/glossary/` exists as an empty
folder but is **not** owned by this skill — do not list it as one of its sections.

### 3. The concept table

Newest additions appended; keep the existing column shape exactly.

```markdown
| Concept | Type | Status |
|---|---|---|
| [packages/jwt.md](packages/jwt.md) | Package | stable |
| [features/register.md](features/register.md) | Feature | draft (endpoint not exercised end-to-end) |
```

The `Status` column is `status:` from the frontmatter **plus a short parenthetical reason whenever
the status is `draft`** — "draft (endpoint not exercised end-to-end)", not bare "draft". A reader
scanning the index should learn *why* something is provisional without opening the file. For
`deprecated`, state what replaced it.

Update a row when the concept's `title`, `type`, path, or `status` changes. Do not reorder rows
just to sort them.

## log.md

`docs/knowledge/log.md` is the bundle's change history. **Newest entry at the top**, reverse
chronological. It is already populated (entries for 2026-09-17 through 2026-09-22) — prepend new
entries, keep the `# Knowledge Bundle Update Log` heading, and follow the same day-suffix rule.

```markdown
# Knowledge Bundle Update Log

## 2026-09-17

- Documented `pkg/jwt` and added `packages/jwt.md` (`status: stable`). Marked `stable`
  because the package has its own test suite (`pkg/jwt/jwt_test.go`) and its behaviour is
  exercised by services that consume it; the package itself has no network dependency to stub.
- Linked `packages/jwt.md` in `index.md`'s concept table.

## 2026-09-15 (2)

- Documented the register flow and added `features/register.md` (`status: draft`).
```

Rules:

- Heading is `## <YYYY-MM-DD>`. If a second entry lands on the same day, suffix it `## <YYYY-MM-DD> (2)`.
- Each bullet is a complete sentence or two, written in past tense, naming the concept file in
  backticks and its `status`.
- State **why** the status is what it is — "marked `draft` because the register endpoint has never
  been exercised against a running Postgres" — not just what was written. The reason is the part a
  future reader cannot reconstruct.
- Note the follow-up trigger: "promote to `stable` once the flow is exercised end-to-end against a
  real/staging Postgres."
- If the entry also touched `index.md`, say so.
- For an update rather than a create, describe what changed and why (a renamed symbol, a stub that
  became real, a constraint that changed), not just "updated `packages/jwt.md`".

## File and folder naming

- Feature concepts: lowercase, kebab-case, matching the user story — `US-001 register` →
  `features/register.md`; `US-002 login` → `features/login.md`.
- Endpoint concepts: `<domain>-<action>.md` matching the route — `POST /v1/auth/register/request-otp`
  → `api/auth-register-request-otp.md`; `POST /v1/auth/logout` → `api/auth-logout.md`.
- Entity concepts: lowercase Go type name, no prefix — `User` → `entities/user.md`,
  `PendingRegistration` → `entities/pending-registration.md`. Keep the concept's `title:` as the
  real Go symbol name.
- UseCase concepts: lowercase Go type name without the `Usecase` suffix — `RegisterUsecase` →
  `usecases/register.md`. Keep `title:` as the real symbol.
- Package concepts: lowercase package name, matching `pkg/<name>/` — `pkg/jwt` → `packages/jwt.md`.
  A driver subpackage is documented inside its parent's Architecture table, not as its own file
  (`pkg/database/postgresql` belongs to `packages/database.md`).
- Migration concepts: the migration's numeric prefix + a slug — `0001_register_onboarding.sql` →
  `migrations/0001-register-onboarding.md`. Keep the prefix so the folder sorts in apply order.

One concept, one file, filename predictable from the thing it documents. A reader should be able to
guess the path.

## Cross-links

Concepts link to each other with **relative markdown links**, using the repo path relative to the
link's own file — not wiki-style `[[...]]` syntax (that is an Obsidian convention, not this
bundle's):

From `features/register.md`, linking to the login concept in the same folder:

```markdown
- Login feature (`services/appService/usecase/login_usecase.go`,
  [features/login.md](login.md)) — shares the `users` table and the OTP sender port.
```

From `features/register.md` linking to a concept in a sibling section, and to a section root:

```markdown
- See [packages/jwt.md](/packages/jwt.md) — where the access/refresh pair is minted.
```

Both forms are acceptable. Prefer the path relative to the linking file; the `/packages/...`
bundle-root form is also in use. Be consistent within a file.

### Dangling links are allowed and intentional

A link to a concept that does not exist yet is a marker, not a bug. When writing a link whose
target does not exist, say so explicitly next to it:

```markdown
- Register feature (`services/appService/usecase/register_usecase.go`) — the flow that
  calls this package's `SendOTP`. Not yet documented as an OKF concept in this bundle.
```

That trailing sentence is what distinguishes an intentional marker from a typo. Include it, and
resolve it by removing the sentence when the target concept is eventually written.

## Related-concepts section

Every concept ends with `# Related concepts`. Each bullet should carry three things where they
apply: the concept name, the real code path in backticks, and the nature of the relationship
(calls, is called by, shares a table, is implemented by, supersedes). A bullet that only names a
sibling adds nothing the index does not already show.
