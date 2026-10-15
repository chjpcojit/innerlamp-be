# Changelog — deity Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first. (`api-contract.md` is a client document with its own version, not part of this set.)

## v3.1 (2026-10-15)

Translated to English per the repo Language policy; no requirement, design or task change. Editorial only.

- `requirements.md`: prose, user stories (`> As a **…**,` / `> I want **…**,` / `> So that **…**.`), AC explanations, ⚠️ markers and notes translated; "Ghi chú kỹ thuật liên quan" → "Technical notes", "Kinh liên quan" → "Related sutras". Quoted user-facing copy ("Hôm nay / Ngày mai / Còn n ngày", "Vía dd/MM âm lịch", "GIÊNG/CHẠP", "Báo thông tin sai", "Chư Thiên") kept in Vietnamese with an English gloss; domain terms (Phật (Buddha), Bồ tát (Bodhisattva), Chư Thiên (devas), ngày vía (a deity's commemoration day, "vía day"), đại nguyện (great vows), Thất Phật Quá Khứ (the Seven Past Buddhas), Đại Cát Tường Thiên Nữ (Śrī Mahādevī)) glossed on first use. AC numbering, IDs, endpoints, fields and SQL unchanged.
- `design.md`: table and Step-by-step Flow prose translated; no step added, removed or reordered.
- `flow.md`: Mermaid messages, notes, `alt`/`opt` labels and participant captions translated; no arrow or Step changed, still a 1:1 mirror of `design.md`.
- `tasks.md`: item text translated; no item added, removed or re-checked.
- `changelog.md`: earlier entries translated; their versions, dates and content unchanged.
- `api-contract.md`: v1.5, title glossed ("Phật / Bồ tát (Buddha / Bodhisattva)"); no contract change.
- Reason: the repo Language policy requires everything committed to be in English.

## v3.0 (2026-10-15)

The spec cites product spec `docs/product/deity/product.md` @ v1.0 (confirmed by Đức on 2026-10-15) and designs the contract changes of product Appendix B. It also fixes the parts of v2.3 that drifted from the code. **Spec only; the v3.0 part has no code yet, `openapi.yaml` unchanged.**

- `requirements.md`: header `Product spec` / `Contract` / `Implements`; `Module` changed `content` → `deity` (the `content` spec was deleted); every AC tagged `BR-DTY-*` / `FR-DTY-*` / `NFR-DTY-*` / `(platform)`. Corrected to match the code: 20 deities (7 Buddhas including the Seven Past Buddhas, 12 Bodhisattvas, 1 deva, Śrī Mahādevī), AC 13 rewritten for root-relative images + `public_base_url` (migration `0021`). v3.0 changes: AC 5 page size 12 (BR-DTY-03); AC 2, 8, 12, 22 add `next_via`, `via_events`, `date`; new US-DTY-06 with AC 23–32 (vía days only from the festival calendar, the server computes the next vía occurrence in the `Y..Y+1` window, `date` = device date, `INVALID_DATE`, inactive events dropped, `via_events` order per BR-CAL-12, `lunar_memorial_days` deprecated and derived, fixed query count, `deity-order` / `deity-next-via` vectors); AC 15, 16, 18 seed vía links and check event codes; AC 33 catalogue file in git; AC 34 related sutras through the existing `sutra` route. Old markers: US codes and unresolvable images → resolved; "featured", content review, production seeding → turned into `⚠️ Not confirmed (product)` pointing to product Appendix A. New markers: vía day ↔ calendar event mapping (product), related-sutras order (product), when to remove `lunar_memorial_days`, production image host.
- `design.md`: **(v3.0)** rows for Domain / Entity, Domain / Vía (new `domain/deity_via.go`, pure functions `PickNextVia` / `ResolveDeityVias` / `NextVia`), Repository interface (`ViaEventCodes`, `MissingCalendarEvents`), Usecase (depends on `CalendarEventRepository`, a clock, `calendar.timezone` / `default_language`), Repository (vía code array in the read query, JSONB dropped, `Upsert` replaces links), Seed CLI, Data, Handler / DTO, Config, Wiring, new Migration (`deity_via_events`, drop `lunar_memorial_days`), Makefile, E2E harness, API spec, ERD. The paged flow adds Steps 5–6 (read the calendar catalogue, compute vía); **adds** Step-by-step Flows for `top` and `detail` (previously one SQL query, now several steps) and drops the ASCII Request flow block; the seed flow adds Step 4 (event-code check) and writes the links in the same transaction. Known gaps updated (missing catalogue file makes `TestParseDeities_RealDataFile` fail, calendar catalogue not yet in git, two-year window, no caching).
- `flow.md`: paged and seed diagrams updated; `top` and `detail` diagrams added; 1:1 mirror of the four Step-by-step Flows.
- `tasks.md`: Phase 1–5 corrected to match the code (`DEVA` constant, `0015`, images `0021` + `imageHost` → `[x]`; catalogue file → `[ ]`; parse test fails); added Phase 8 (data, schema, seed), Phase 9 (reads, vectors, contract, E2E), Phase 10 (cleanup, docs) — all `[ ]`.
- `api-contract.md`: v1.4, only adds the banner "stale, see `openapi.yaml` + this spec"; delete it once the client specs are approved.
- This spec's technical decisions (pending review via PR): the server computes the next vía occurrence (BR-DTY-06 leaves it to the contract) so clients do not convert the lunar calendar; "today" = the `date` parameter from the device, falling back to `calendar.timezone`; the deity ↔ calendar event links live in the deity catalogue file (product §3.5); `lunar_memorial_days` kept for now, derived from `via_events`.
- Follow-ups outside the spec's scope (not changed here): `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml` (Phase 9); `migrations/` + `docs/specs/db/erd.mmd` (Phase 8); `.gitignore` + committing `assets/data/phat_botat.json` (Phase 8); `docs/specs/features/calendar/` E26 (calendar catalogue in git) is a prerequisite; OKF `docs/knowledge/features/deity.md` for `il-okf-writer`; the Backend column of product §4.2 for `ba-writer`.
- Reason: product spec v1.0 replaces the business source; Appendix B assigns the backend page size 12, vía from the festival calendar, the next vía occurrence on card and page, related sutras, the catalogue file in git, and bringing the v2.3 spec back in line with the code.

## v2.3 (2026-10-12)

Implemented the `type` filter (AC 20–22) specified in v2.2. At the user's "implement" request.

- Code: `domain/deity.go` (`IsValidDeityType`, `ErrInvalidDeityType`, `List(ctx, deityType, offset, limit)`), `usecase/deity_usecase.go`, `repository/postgres_deity_repository.go` (`WHERE type = $1` for both the count and the page fetch), `delivery/http/deity_handler.go`, `delivery/http/errors.go` (`INVALID_DEITY_TYPE`), `domain/mock_deity_repository.go`. `cmd/main.go` unchanged.
- Tests: usecase/handler unit tests, repository (`TestDeityRepository_List_FiltersByType`), new E2E `e2e/deity_e2e_test.go` (the first deity E2E). The repository test now also applies `0015_deities_deva_type.sql` in `applyMigration` because `deities.type` needs `DEVA`.
- `docs/specs/api/openapi.yaml` and `web/docs/public/openapi.yaml`: added the `type` parameter, the `400 INVALID_DEITY_TYPE` response, fixed the "no filter" description.
- `requirements.md`/`design.md`/`flow.md`: only Status changed and the "no code yet" notes removed; no requirement change. `tasks.md`: Phase 7 checked `[x]` after running build/vet/test. `api-contract.md`: v1.3.
- Remaining: E2E for `top`/`detail` (Phase 5), OKF `docs/knowledge/features/deity.md`, web docs `guides/catalogues/deities.md`.

## v2.2 (2026-10-12)

Added a `type` filter for the Buddha / Bodhisattva list (`GET /v1/deities?type=`). At the user's request after confirming there was no API to filter by type yet. **Spec only; no code, tests or `openapi.yaml` change yet.**

- `requirements.md`: added AC 20–22 (valid filter, `INVALID_DEITY_TYPE`, combined with `page`) to US-DTY-02, numbered at the end so AC 1–19 references do not shift; fixed AC 4, 5 (filter and `total`); AC 16 and `DeitySummary.type` add `DEVA` to match the code/migration `0015`; route table, Out of scope, Pending; decisions recorded under "Open questions (resolved)".
- `design.md`: added (v2.2) rows to Domain / Errors, Repository interface, Usecase, Repository, Handler, Error map; Steps 1–4 of the paged flow add `type`; Known gaps updated; `cmd/main.go` unchanged (same route).
- `flow.md`: the paged diagram adds the unknown-`type` branch and `WHERE type = $1`, mirroring `design.md` 1:1; no Step added/removed.
- `tasks.md`: added Phase 7 (7 items, all `[ ]`). No item checked `[x]` because there is no code yet.
- `api-contract.md`: v1.2, added a "Filter by type" section marked not implemented.
- Decisions taken by the spec, pending user/product review (details under "Open questions (resolved)"): a parameter on the existing route; exact, case-sensitive match; empty = no filter; one value per request; error code `INVALID_DEITY_TYPE` following the `INVALID_GALLERY_TYPE` precedent.
- Not breaking: clients that do not send `type` get exactly the old result.
- Existing drift, not fixed: the spec still says "18 entries / two types" in several places while the code and data already have `DEVA` (migration `0015`) and 20 entries (per `openapi.yaml`). Only the places touching the filter were fixed.
- Follow-ups outside the spec's scope: `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) — the `/v1/deities` description says "no filter"; OKF `docs/knowledge/features/deity.md` (does not exist yet); web docs `web/docs/src/content/docs/guides/catalogues/deities.md`.
- There is no `SRS.md` for this feature (only `administrative-units`, `genealogy`, `lamp`… have one), so there is no SRS to update.

## v2.1 (2026-10-03)

Removed references to the old `content` spec (deleted); only one note line changed, no requirement changed.

- `requirements.md`: the `⚠️ Not confirmed` marker on user story codes no longer mentions `content/requirements.md`; the question itself (product has not settled the US numbers) is unchanged.
- `design.md`, `flow.md`, `tasks.md`: version bump only, no change (checked).

## v2.0 (2026-09-30)

The paged list combines Buddhas and Bodhisattvas (same `deities` table); the `GET /v1/bodhisattvas` route is replaced by `GET /v1/deities?page=N`. At the user's request after reviewing the Buddha/Bodhisattva API.

- `requirements.md`: US-DTY-02 renamed to "Browse Buddhas and Bodhisattvas page by page"; AC 4–5 (route, no type filter, `total` is the number of entries in the whole table); route table; resolved the "how to view Buddhas" question (moved to "Open questions (resolved)"); 4 `⚠️ Not confirmed` markers remain.
- `design.md`: dropped `ListByType`, replaced with `List` (no type filter); Steps 3–4 change the SQL (`count(*)` over the whole table, no `WHERE type`); route note.
- `flow.md`: the paged sequence changes route, method name and SQL to match `design.md` 1:1; no step added/removed.
- `tasks.md`: renamed the usecase/handler/repository/test items; E2E expectations updated. No item newly checked `[x]` beyond what already exists in the code and has run under `-short`.
- Reason: same table, only `type` differs; previously Buddhas could only be viewed through `top`/`detail`.
- **Breaking change**: `/v1/bodhisattvas` is removed, with no alias; the iOS client must switch to `/v1/deities` and filter by `type` itself if it needs to show only Bodhisattvas. The repository test needs Postgres and the server has not been re-run since the change.
- Follow-ups outside the spec's scope: `api-contract.md` (updated in the same pass, v1.1), `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml`, OKF `docs/knowledge/features/deity.md` (via `/il-okf-writer`), web docs `web/docs/src/content/docs/guides/catalogues/deities.md` and `changelog.md`.

## v1.1 (2026-09-30)

Added the `deities` table to the ERD; spec updated to match.

- `requirements.md`: removed the Pending item about the ERD; Status drops "no table in the ERD yet".
- `design.md`: the ERD row moved to "exists"; Known gaps now only has the OKF. Flow unchanged.
- `flow.md`: version bump only, no change.
- `tasks.md`: checked `[x]` the ERD item.
- Reason: `docs/specs/db/erd.mmd` and `erd.svg` already have the `deities` table (columns taken from the migration, `CHECK` constraints noted in comments).
- Follow-up outside the spec's scope: the OKF concept `docs/knowledge/features/deity.md` still does not exist.

## v1.0 (2026-09-30)

Initial spec, written retroactively from the already-implemented Phật (Buddha) / Bồ tát (Bodhisattva) catalogue in `services/appService/` (Retroactive mode), at the request to "document the temple, Buddha/Bodhisattva and Dharma article parts". Covers five user stories: featured (Buddhas and Bodhisattvas mixed), paged Bodhisattva list, detail, consistent data shape, loading data with a seed command.

- `requirements.md`: 5 US, 19 AC (EARS), an API contract table of 3 routes read from the actual handler, 5 `⚠️ Not confirmed` markers (the "featured" criterion, how to view Buddhas without a list route, image source, the data-loading process in real environments, source and accuracy of the religious content) plus 1 marker on US codes.
- `design.md`: layers table tied to real files/symbols; 2 Step-by-step Flows (paged Bodhisattva list, loading data with `cmd/seeddeities`) with the transaction boundary; a Known gaps section (no E2E, the repository test needs Postgres and wipes dev data, images do not resolve and do not follow the `null` convention, no Buddha list route, the seed uses dev config, upsert overwrites and never deletes, seeder validation only partial, broken JSONB breaks the whole page, count and page fetch do not share a snapshot, ERD/OKF missing).
- `flow.md`: 2 sequence diagrams mirroring the two Step-by-step Flows 1:1.
- `tasks.md`: 5 Phases, `[x]` for items verified to exist (build, vet, `go test -short` pass); `[ ]` remains for usable images, seeding outside dev, ERD, E2E, OKF, reviewing `api-contract.md`.
- Reason: the feature had shipped and had an `api-contract.md` for iOS but no spec following the convention; a spec is needed so `admin/requirements.md` (which points to `deity/`) has something to reference.
- Note versus `api-contract.md`: the HTTP contract matches the code; the spec adds the seed, the consequence of Buddhas being viewable only through `top`/`detail`, and states the image URL convention that differs from `temple`/`article` (string, never `null`). The count of 18 entries (6/12) and the placeholder image URL values were cross-checked against `assets/data/phat_botat.json`.
- Follow-ups outside the spec's scope (not changed here): `docs/specs/db/erd.mmd`/`erd.svg` (add the `deities` table); the OKF concept `docs/knowledge/features/deity.md` does not exist yet, to be created by `bdd-okf-docs` once there is E2E; `docs/specs/api/openapi.yaml` already has the routes, so no change needed.
