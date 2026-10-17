# Changelog — lamp

Version history of the spec set `requirements.md` / `design.md` / `flow.md` / `tasks.md` (one shared version at any point in time).

## v2.0 (2026-10-17)

- `requirements.md`: rebased on `docs/product/lamp/product.md` v1.0 — new `Product spec` / `Contract` / `Implements` header, every criterion tagged with its `FR-LMP-*`/`BR-LMP-*`. AC 1–24 keep their numbers; AC 6 changes meaning (the recipient is optional); AC 25–30 are new (`is_mine`, `recipient: null`, anonymous name absent from the garden, no edit/delete route, account deletion removes lamps, vector files). New "Contract changes in v2.0" table. Rewritten in English per the Language policy.
- `design.md`: rewritten in English. v2.0 changes: `Recipient *string`, `GardenLamp.IsMine`, `GardenQuery.ViewerID`, `Garden(ctx, userID, f)`, dedupe with `IS NOT DISTINCT FROM` on the recipient, a migration dropping `NOT NULL` on `lamps.recipient`, the `is_mine` expression in the page query, the vector-file test, OpenAPI 3.0.0. `ON DELETE CASCADE` confirmed as intended. Step L-2/L-3/L-4 and G-1..G-4 updated; two new rationale paragraphs (`is_mine` in SQL, `IS NOT DISTINCT FROM`).
- `flow.md`: rewritten in English; both diagrams updated to mirror the changed steps (blank recipient → nil, `IS NOT DISTINCT FROM`, `viewerID`/`is_mine`).
- `tasks.md`: rewritten in English; Phase 1–5 unchanged (still `[x]` as implemented, `erd.svg` and the OKF note still `[ ]`); Phase 6–8 added, every item `[ ]`.
- Reason: product spec v1.0 (be#53) unified the four platforms and the product owner confirmed 14 decisions on 2026-10-16. Two change the contract: an optional recipient and `is_mine` on garden items. This version is spec only — no code or OpenAPI change.

### Follow-ups (not part of this spec change)

- `docs/specs/api/openapi.yaml` 2.1.1 → 3.0.0, `web/docs/src/content/docs/changelog.md` entry, `web/docs/public/openapi.yaml` re-sync (Phase 8).
- New migration dropping `NOT NULL` on `lamps.recipient` (Phase 6); update `migrations/README.md`.
- `docs/specs/db/erd.mmd` (`recipient` nullable) and `erd.svg` regeneration.
- `docs/knowledge/features/lamp.md`: write with `il-okf-writer` after v2.0 is implemented and verified.
- Delete the legacy `SRS.md` in the cleanup PR once the user approves the file list.

## v1.3 (2026-10-11)

- `requirements.md`: AC 6 and AC 7 state the length limits through the config keys `lamp.recipient_max_chars` (dev: 100) and `lamp.prayer_max_chars` (dev: 300) instead of fixed numbers; technical note "every limit lives in config" with the `VARCHAR` ceiling rule.
- `design.md`: Config row gains `RecipientMaxChars`, `PrayerMaxChars` and the two schema ceiling constants; Usecase row reads all three limits from config. No flow change.
- `flow.md`: no change needed, verified — only the source of two numbers changed.
- `tasks.md`: four `[ ]` items (config + yaml + usecase + openapi/web docs) for moving the two limits into config.
- Reason: the product owner asked to move every lamp limit into `config.yaml`. No behavior change with the current dev values.

## v1.2 (2026-10-10)

- `requirements.md`: Status moved to implemented Phase 1–4 and verified; API contract note says "already in `openapi.yaml`"; Pending lists the rest (`erd.svg`, OKF concept); retention marker gets the temporary `ON DELETE CASCADE` (marker still open); migration number is `0027`. No AC changed.
- `design.md`: Layers table matches the real type names (`LampInput`, `GardenFilter`, `LightLampParams`, `GardenLamp.DisplayName()`, `Garden` returns `Limit+1` rows and the usecase trims); migration `0026` → `0027`; temporary `ON DELETE CASCADE`; Step G-4: mapping `lighter_name` NULL → `Vô danh` is in the handler via `DisplayName()` (same behavior, correct layer); Known gaps updated.
- `flow.md`: the garden diagram splits out the response mapping in the handler to match Step G-4; the lighting diagram unchanged.
- `tasks.md`: ticked Phase 1–4 and two Phase 5 items, all actually run. Still open: regenerate `erd.svg` (no `mmdc`), OKF concept.
- Reason: implemented test-first. Two differences from the v1.1 spec: (1) the migration number is `0027` because another change took `0026`; (2) `ON DELETE CASCADE` was chosen temporarily so the migration was not blocked.

## v1.1 (2026-10-10)

- `requirements.md`: closed the "cursor paging" marker — the product owner chose **cursor** for the garden and accepted the departure from `page`/`total_pages`; moved to "Open questions (resolved)". Two markers remain.
- `design.md`: no structural change — the answer confirmed the existing design (keyset, `cursor`/`next_cursor`/`has_more`); only removed the marker in Overview and Known gaps.
- `flow.md`: no change needed, verified.
- `tasks.md`: no change needed, verified; only the Status line.
- Reason: the user answered the paging question (2026-10-10).

## v1.0 (2026-10-10)

- `requirements.md`: created. Two user stories `US-LAMP-01` (light a lamp) and `US-LAMP-02` (see the garden) with 24 EARS criteria based on `FR-001`…`FR-009` of `SRS.md` v0.3; a proposed API contract table; three `⚠️ Not confirmed` markers (performance/volume, retention + account deletion, cursor paging).
- `design.md`: created. Layers table, schema `lamps` + three indexes, two Step-by-step Flows (`L-1..L-4`, `G-1..G-4`), why keyset instead of `OFFSET`, why dedupe shares the transaction with the cap.
- `flow.md`: created. Two sequence diagrams mirroring the flows.
- `tasks.md`: created. Five phases, all `[ ]`.
- Reason: the user asked for lighting a lamp and the lamp garden; the BA settled 26 questions in `SRS.md` v0.3.
