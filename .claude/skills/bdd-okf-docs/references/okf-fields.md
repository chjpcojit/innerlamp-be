# OKF v0.2 frontmatter field catalog

Full reference for the optional field families. `SKILL.md` covers the
common path; read this when a bundle uses a field that page does not
explain, or when you need exact semantics.

## Contents

- [Core fields](#core-fields)
- [Provenance: sources](#provenance-sources)
- [Trust: generated and verified](#trust-generated-and-verified)
- [Lifecycle: status and stale_after](#lifecycle-status-and-stale_after)
- [Actor convention](#actor-convention)
- [Path-valued fields](#path-valued-fields)
- [Reserved files](#reserved-files)
- [Conformance rules](#conformance-rules)
- [Legacy v0.1 fields](#legacy-v01-fields)
- [Deliberately out of scope](#deliberately-out-of-scope)

## Core fields

| Field | Required | Notes |
|---|---|---|
| `type` | **Yes** | The only always-required key. Non-empty string. Not centrally registered. A concept carrying only `type` is fully conformant. |
| `title` | Recommended | Display name. Consumers may derive one from the filename if absent. |
| `description` | Recommended | One sentence. Used by `index.md` generators, search snippets, previews. |
| `resource` | Optional | Canonical URI for the underlying asset. Omit for concepts describing abstract ideas — most BDD features have no `resource`. |
| `tags` | Optional | YAML list of short strings. Maps naturally onto Gherkin `@tags`. |

Producers may add any additional keys. Consumers must not reject documents
with unrecognized fields, and should preserve unknown keys when
round-tripping.

## Provenance: `sources`

```yaml
sources:
  - id: ac-431
    resource: https://tracker.acme/BILL-431
    title: "BILL-431 acceptance criteria"
    author: human:pchen
    usage_count: 120
    last_modified: 2026-08-14
usage_window: { from: 2026-07-01, to: 2026-07-31 }
```

| Key | Required | Meaning |
|---|---|---|
| `resource` | **Yes, within an entry** | An absolute URL, a bundle-relative path (`/…`), a relative path, or a *scope descriptor* a consumer cannot follow (e.g. `all support tickets tagged checkout`). |
| `id` | Optional | Stable key for per-claim attribution. Include it whenever the body cites the source. |
| `title` | Optional | Human-readable label. |
| `author` | Optional | Who produced the source, in the actor convention. An authority signal. |
| `usage_count` | Optional | How often the resource was exercised over `usage_window`. An adoption/liveness signal. |
| `last_modified` | Optional | `YYYY-MM-DD`, when the *source* last changed. Distinct from `generated.at`, which is when the *concept* was written. |
| `usage_window` | Optional | A `{ from, to }` sibling of `sources` framing every `usage_count`. A single entry may carry its own to override. |

**Credibility is inferred, never stored.** OKF records objective per-source
signals and deliberately does not store a score, because a score is
subjective, unportable between consumers, and goes stale. Do not invent a
`credibility:` or `confidence:` key.

`usage_count` is coarse. It is comparable at the alive-versus-dead and
order-of-magnitude level and against a source's own history, but is not a
precise cross-kind ranking — a scheduled job's executions and a person's
deliberate reads do not carry equal weight. Read it as liveness and trend.

**Lineage travels through links, not a field.** When a `resource` points at
another concept in the bundle, the derivation edge already exists in the
graph, and a consumer may recurse into that concept's own `sources`. There
is no `derived_from` key; deeper lineage is out of scope for v0.2.

### Per-claim attribution

Use a markdown footnote whose label matches a `sources[].id`:

```markdown
Payment is captured on dispatch.[^pay-policy]

[^pay-policy]: Payment capture policy
```

The label is the join key — consumers resolve attribution through the
matching `sources` entry, not by parsing the footnote prose. Keyed rather
than positional (`sources[0]`) because agents rewrite these documents
constantly and a positional index misattributes silently the moment the
list is reordered.

## Trust: `generated` and `verified`

```yaml
generated: { by: human:rgamble, at: 2026-08-20T10:00:00Z }
verified:
  - { by: human:pchen, at: 2026-08-21T15:30:00Z }
  - { by: process:ci-acceptance, at: 2026-08-22T02:14:00Z }
```

- `generated.by` — **required within `generated`**. An actor.
- `generated.at` — ISO 8601 datetime of the last meaningful content change.
- `verified` — a list of `{ by, at }` verification events. Multiple entries
  capture independent checks. "How recently" is the latest `at`.

`verified` is independent of `generated.at`: content can change without
re-confirmation, and facts can be re-confirmed without regeneration.

A single verifier may be written as a bare mapping without the list dash,
and consumers **must** treat it as a one-element list:

```yaml
verified: { by: human:pchen, at: 2026-08-21T15:30:00Z }
```

### Trust tiers

Derived from `verified`, lowest to highest:

| Tier | Condition |
|---|---|
| unverified | no `verified` key |
| machine-confirmed | `verified` by non-`human:` actors only |
| human-reviewed | at least one `human:<id>` actor |

Tiers are advisory signals, not access control. A concept with no trust
frontmatter is still consumable and must not be rejected.

## Lifecycle: `status` and `stale_after`

```yaml
status: stable        # draft | stable | deprecated
stale_after: 2026-11-20
```

- `draft` — not yet reviewed; possibly incomplete.
- `stable` — the default; ready for consumption. Absent `status` means `stable`.
- `deprecated` — kept for links and history; no longer current.

`stale_after` is an absolute `YYYY-MM-DD` date. A concept is stale when
`today >= stale_after`. Absolute rather than a relative TTL so the
staleness decision is a plain date comparison with no reference to when the
concept was read.

## Actor convention

Every identity field (`generated.by`, `verified[].by`, `sources[].author`)
uses one convention:

| Form | Use |
|---|---|
| `human:<id>` | A person. e.g. `human:pchen` |
| `process:<id>` | An automated process. e.g. `process:ci-acceptance` |
| `<producer>/<version>` | An agent or tool. e.g. `claude-opus-5/bdd-okf-docs` |

Consumers key trust classification off the `human:` prefix, so it must be
used for hand-authored or human-confirmed content — and must not be used
for agent output.

## Path-valued fields

`resource` and `sources[].resource` each accept:

- an absolute URL (`https://…`),
- a bundle-relative path beginning with `/` — **recommended**, since it
  survives a document moving within its subdirectory, or
- a relative path (`../glossary/basket.md`).

`sources[].resource` may additionally be a scope descriptor, in which case
it is not a path at all.

A `references/` subdirectory conventionally mirrors external material
inside the bundle as first-class concepts. It is a naming convention, not
a requirement.

### Links between concepts

Standard markdown links, absolute (`/features/checkout.md`) or relative
(`./other.md`). A link asserts an untyped relationship; the kind is
conveyed by surrounding prose. Consumers **must tolerate broken links** — a
link to a nonexistent target is not malformed, it may simply be
not-yet-written knowledge.

## Reserved files

| File | Frontmatter | Structure |
|---|---|---|
| `index.md` | **None**, except a bundle-root `index.md` may carry `okf_version: "0.2"` and nothing else | One or more `#` section headings, each a bullet list of `[Title](url) - description` |
| `log.md` | **None** | `##` date headings in ISO `YYYY-MM-DD`, newest first, bullet entries beneath |

These names are reserved at every level of the hierarchy and must never be
used for concept documents. All other `.md` files are concepts.

Entries in `index.md` should carry the `description` from the linked
concept's frontmatter. Producers may generate index files automatically;
consumers may synthesize one when none is present.

## Conformance rules

A bundle is conformant with v0.2 if:

1. Every non-reserved `.md` file contains a parseable YAML frontmatter block.
2. Every frontmatter block contains a non-empty `type`.
3. Every reserved filename present follows its structure.

Consumers **must not** reject a bundle for: missing optional fields,
unknown `type` values, unknown additional keys, broken cross-links, or
missing `index.md` files. Treat every other constraint as soft guidance.

## Legacy v0.1 fields

You may encounter these in older bundles. Both were superseded, and a v0.1
bundle remains consumable:

| v0.1 | v0.2 replacement |
|---|---|
| `timestamp` | `generated.at`. Consumers may fall back to `timestamp` when `generated` is absent. |
| body `# Citations` list | `sources` frontmatter. Consumers should read `sources` and may still parse a legacy `# Citations` list. |

When updating a v0.1 document, migrate both rather than leaving them
alongside the new fields.

## Deliberately out of scope

Do not invent frontmatter for these — v0.2 leaves them to a future
revision, and inventing keys creates exactly the incompatible conventions
the format exists to avoid:

- The full runtime protocol: receipt and verdict wire formats.
- The attester ABI, portability, and sandboxing.
- Attestation caching.
- Semantic-layer templates (Looker, dbt).
- Explicit external lineage (`derived_from`) and data lineage.
- A stored credibility or confidence score.

Bundles may declare their target version with `okf_version: "0.2"` in a
bundle-root `index.md` — the only place frontmatter is permitted in an
index file.
