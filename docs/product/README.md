# docs/product — product specs (layer 0)

One product spec per feature, shared by every InnerLamp platform: this backend, the website
(`innerlamp-website`), the iOS app (`innerlamp-iOS`) and the Android app (`innerlamp-Android`).
It says **what the product does and why** in terms a user can observe on any platform, plus the
business rules behind it. It never describes how a platform builds it.

```
docs/product/<feature>/product.md     ← layer 0: product spec (this folder)
docs/specs/api/openapi.yaml            ← layer 1: API contract (backend-owned)
docs/specs/features/<feature>/         ← layer 2: backend platform spec
<client repo>/Docs/…/<feature>/        ← layer 2: website / iOS / Android platform specs
```

A decision lives in exactly one layer: product decisions here, request/response shape and error
codes in the contract, implementation in the platform spec. Platform specs reference this folder by
version and ID; they never copy it.

## Layout

```
docs/product/<feature>/
  product.md        IEEE 29148-structured spec: glossary, BR-*, FR-*, NFR-*, platform variance,
                    requirement × platform traceability matrix, open questions
  changelog.md      version history; each entry lists affected IDs and platforms to follow up
  rules/*.json      test vectors for computational business rules, run by every platform's tests
```

`<feature>` is the same lowercase kebab-case slug as `docs/specs/features/<feature>/`. Each feature
has a short feature code (e.g. `MEM`) used in its IDs: `BR-MEM-01`, `FR-MEM-001`, `NFR-MEM-01`.

## How specs are written

- `ba-writer` (`.claude/skills/ba-writer/`) writes and updates everything in this folder. It asks the
  open questions first and records anything unanswered as `⚠️ Not confirmed`.
- Each platform's `spec-writer` skill — in this repo and in each client repo — then writes the layer-2 spec, starting with this header:

  ```markdown
  Product spec: innerlamp-be `docs/product/<feature>/product.md` @ v<MAJOR.MINOR>
  Contract: innerlamp-be `docs/specs/api/openapi.yaml` @ <commit or version>
  Implements: BR-<CODE>-…, FR-<CODE>-…
  ```

- A MAJOR version bump means observable behavior changed: platform specs citing the old MAJOR must be
  re-checked.

## Status

New convention as of 2026-10-14; no feature has been migrated yet. Existing `SRS.md` files under
`docs/specs/features/<feature>/` (and in the iOS/website repos) are legacy. They stay as they are
until their feature is rewritten here, at which point the backend copy gets a "Superseded by" line.
