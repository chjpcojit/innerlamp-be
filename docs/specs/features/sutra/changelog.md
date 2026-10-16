# Changelog — sutra Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first.

## v4.1 — 2026-10-16

Cleanup of the documents that v4.0 flagged as superseded. No requirement or design change, so MINOR.

- `tasks.md` only: the follow-ups for `test-coverage-report.md` (deleted), `docs/knowledge/features/sutra.md`
  (deleted, to be rewritten after Phase 7) and the assumptions / tech-debt entries (A-002, A-008, TD-006
  deleted) now say what was done.

## v4.0 — 2026-10-16

Rebased on the product spec `docs/product/sutra/product.md` @ v1.0 (be#44) and rewritten in English per the
Language policy; the module name is now `sutra` (it was `content`). MAJOR because the criteria are renumbered
and the contract gains two statements (search term limit, a non-null duration).

- `requirements.md`: header `Product spec` / `Contract` / `Implements`; 44 criteria, each tagged with its
  BR-/FR-/NFR-STR ID or `(platform)`. Brought in line with the code (shipped after v3.0, previously listed
  out of scope): **search `q`** (AC 4–5: titles in every language, collection, citation code; diacritic- and
  case-insensitive; 1–100 characters, empty → `400 INVALID_FILTER`) and **`audio_url`** (AC 20: `""` = no
  audio). New (not implemented): AC 34 `duration_minutes` never `null`, AC 38 the seeder refuses a text without
  `durationMinutes`; AC 8 and AC 44 (vector files, search timing). Written down without code change: AC 14
  versions in source order, AC 18 `review_status`, AC 36 public access. Added `⚠️ Not confirmed (product)` on
  AC 34 (formula-filled durations of the 10 Vajrayāna texts).
- `design.md`: layers table updated (search, `audio_url`, split fields, migrations `0019`/`0020`), data model
  with `audio_url`, the five flows in English (the list flow now includes `q`), Known gaps trimmed to what is
  still true; the "Data file" section replaces the long classification table.
- `flow.md`: the five diagrams in English, mirroring `design.md` 1:1; the v4.0 refusal drawn as a `Note`.
- `tasks.md`: rewritten in English with the old evidence kept in short form; new Phase 7 (all `[ ]`): seeder
  duration rule, `openapi.yaml` follow-up, vector-file tests, search timing, coverage audit.
- Follow-ups flagged, not edited: `docs/specs/api/openapi.yaml` (+ web copy), `docs/knowledge/features/sutra.md`
  (stale), `docs/specs/db/erd.mmd` (`audio_url`), `test-coverage-report.md` (old numbering), and the assumptions
  and tech-debt entries superseded by the product spec (cleanup PR).
- Criterion crosswalk, old (v3.0) → new: 1→1, 2→2, 3→3, 4→5, 5→6, 6→7, 7→9, 8→10, 9→15, 10→11, 11→14,
  12–13→21–22, 14→23, 15→24, 16→25, 17→26, 18→27, 19→21, 20→28, 21→29, 22–23→30, 24→31, 25→32, 26→34,
  27→33, 28→35, 29→37, 30→39, 31→40, 32→41, 33→42, 34→43, 35→37–43, 36→12, 37→13, 38→16, 39→17, 40→37,
  41→19.

## Earlier versions (v1.0 – v3.0, 2026-10-03 to 2026-10-11, written in Vietnamese)

- v1.0 (2026-10-03): first spec of the read-only sutra catalogue; implemented the same day (4 routes, 4 tables,
  `cmd/seedsutras`, 35 criteria, E2E against the real binary).
- v2.x (2026-10-03 – 10-08): pages by `page`/`page_size`/`total`/`total_pages`, lines by `n` range, `key` per
  version, `daily_chant`/`purposes`/`duration_minutes`, `cmd/seedsutras` as a second entry; v2.12–v2.13 split
  `reference` and `license`, added `summary` (cards and detail), migrations `0019` (audio, search, duration
  backfill) and `0020`.
- v3.0 (2026-10-11): the service stopped seeding on start; the command is the only way to load.
- The full Vietnamese history is in git (`git log -p -- docs/specs/features/sutra/changelog.md`).
