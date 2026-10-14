# Upstream: product spec and API contract

Every InnerLamp platform spec sits under two documents it does not own. This file is identical in
the spec skills of all four repos (backend `il-spec-be`, website `il-spec-web`, iOS `il-spec-ios`,
Android `il-spec-android`) so the four agree on how they cite them.

```
Layer 0  Product spec   innerlamp-be  docs/product/<feature>/product.md   (+ rules/*.json)
         owner: ba-writer (backend repo only) — what the product does, BR-*/FR-*/NFR-* IDs
Layer 1  API contract   innerlamp-be  docs/specs/api/openapi.yaml
         owner: backend (il-spec-be + code) — paths, fields, error codes
Layer 2  Platform spec  this repo — how THIS platform builds it (this skill)
```

## Where to read them

- From the backend repo, the paths above are local.
- From a client repo, read them from a checkout of `innerlamp-be`, normally a sibling folder
  (`../innerlamp-be/docs/product/...`). If no checkout is available, say so and ask the user for the
  file or a link — never write the platform spec from memory of the product spec.
- **Never copy** `product.md`, a rule, or `openapi.yaml` into this repo. Cite by path, version and
  ID. A copy is the root cause of the drift these layers exist to prevent.

## Before writing a new platform spec

1. Find the product spec by the feature slug (`docs/product/<feature>/`). Platform folder names may
   differ (`Memorial`, `gio`); the product spec's §1.5 lists the mapping.
2. **No product spec for a new feature → stop.** Tell the user the feature needs a product spec
   first, written with `ba-writer` in `innerlamp-be`. Do not invent product behavior here.
3. **Legacy feature** (platform spec already exists, no product spec yet): updates are allowed. Put
   `Product spec: none yet (legacy)` in the header and keep going; recommend migrating the feature
   with `ba-writer` in the report.
4. Note the product spec's `Version`, the IDs in scope for this platform (its §4.2 matrix column),
   any §3.6 Platform variance rows for this platform, and every open `⚠️ Not confirmed` that names
   this platform.

## Header every platform `requirements.md` carries

Right under the version block:

```markdown
Product spec: innerlamp-be `docs/product/<feature>/product.md` @ v<MAJOR.MINOR>
Contract: innerlamp-be `docs/specs/api/openapi.yaml` @ <commit sha or info.version>
Implements: BR-<CODE>-01, BR-<CODE>-03, FR-<CODE>-001..004
```

(The backend's own spec uses `Contract: docs/specs/api/openapi.yaml (this repo)`.)

## Tagging acceptance criteria

Every EARS criterion ends with the product ID it implements, in parentheses:

```
- WHEN the user saves an entry with an empty name, THE SYSTEM SHALL ... (FR-MEM-002)
```

A criterion that exists only because of this platform (layout, navigation, a platform API, caching)
and has no product counterpart is tagged `(platform)`. Every in-scope `FR-*` for this platform must
appear on at least one criterion; a missing one is a gap to report, not to paper over.

## What the platform spec may and may not say

| Platform spec may | Platform spec must not |
|---|---|
| Cite a `BR-*` and say how this platform implements it | Restate or reword a business rule's outcome |
| Describe screens, components, layers, storage, caching, error → UI mapping | Decide a product question (scope, a rule's edge case, what the user sees in a business sense) |
| Cite contract operations by method + path and link `openapi.yaml` | Copy request/response schemas or the error-code list from `openapi.yaml` |
| Add `(platform)` criteria for platform-only behavior | Add behavior that contradicts a product `FR-*` or §3.6 |

## When the product spec is wrong, missing or ambiguous

Do not fix it here, and never edit `product.md`. Record it inline where it bites:

```
> ⚠️ Not confirmed (product): <the question> — blocks <criterion / task>; hand back to ba-writer in innerlamp-be.
```

and list it in the report as a hand-back item for `ba-writer`. If the platform has already shipped
behavior that differs from the product spec, say so in the report: the product matrix should show
`drift` for this platform until it is resolved.

## Computational business rules

When an in-scope `BR-*` has a test-vector file (`docs/product/<feature>/rules/*.json`), the platform
`tasks.md` must contain a task to run every case of that file in this platform's unit tests, loading
it from the backend repo (or a pinned copy refreshed by a script — never hand-edited). The platform
spec cites the file; it never restates the cases.

## Keeping in sync

- When the product spec's **MAJOR** version moves past the one in the header, the platform spec is
  stale: re-check the changed IDs (the product `changelog.md` lists them) and bump the header in
  the same update.
- A **MINOR** change only needs a header bump when this platform's IDs or §3.6 rows changed.
- Contract changes come from the backend; client specs update the `Contract:` line when they adopt
  a newer contract.
