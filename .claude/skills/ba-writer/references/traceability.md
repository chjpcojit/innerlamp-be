# Traceability, Versioning and Change Propagation

The product spec is only useful if every platform can say which version it implements and which IDs
it covers. This file defines the matrix, the header platform specs carry, and how a change travels.

## The §4.2 matrix

One row per `BR-*`, `FR-*` and `NFR-*`; one column per platform; a note column.

| Status | Meaning |
|---|---|
| `planned` | In scope for the platform, not started |
| `in-progress` | Platform spec or code exists, not complete |
| `done` | Implemented and verified (tests/vectors pass) on that platform |
| `drift` | Implemented differently from this spec — the note says how and links the platform file |
| `n/a` | Does not apply to that platform (e.g. a rule only the server enforces) |
| `later` | Deferred for that platform |

This skill fills the matrix when it writes or updates the spec. In retroactive mode it fills real
statuses from what it read; otherwise `planned` / `n/a` / `later`. Platform teams update their own
column when they ship — a status change alone is a MINOR bump with a one-line changelog entry.

## Header every platform spec carries

Platform specs (written by each repo's spec writer, not by this skill) start with:

```markdown
Product spec: innerlamp-be `docs/product/<feature>/product.md` @ v<MAJOR.MINOR>
Contract: innerlamp-be `docs/specs/api/openapi.yaml` @ <commit or version>
Implements: BR-<CODE>-01, BR-<CODE>-03, FR-<CODE>-001..004
```

and every platform-level acceptance criterion names the `FR-*` / `BR-*` it implements. This skill
only documents the convention and reports gaps; it never edits platform specs.

## Versioning

`product.md` header `Version` is `MAJOR.MINOR`, starting at `0.1` while questions are open and `1.0`
when the user confirms the spec.

| Change | Bump |
|---|---|
| A BR/FR/NFR changes observable behavior, or is removed | MAJOR |
| New BR/FR/NFR, clarification, resolved question, new vector cases, matrix status | MINOR |
| Typos, formatting | none (no changelog entry needed) |

A platform spec citing an older **MAJOR** version is stale and must be re-checked; an older MINOR is
fine.

## `changelog.md`

Newest entry first:

```markdown
## v1.1 — 2026-10-20

- Changed: BR-MEM-05 now covers leap months (was: leap month ignored).
- Added: FR-MEM-007 (export the memorial book).
- Resolved: "Is the OS reminder iOS-only?" → yes, recorded in §3.6.
- Platforms to follow up: iOS (BR-MEM-05, vectors), Android (BR-MEM-05, vectors), Backend (none).
- Reason: <user decision + date>.
```

Every entry lists the affected IDs and the platforms that must follow up, so the change can be
handed to each repo's spec writer without re-reading the whole spec.
