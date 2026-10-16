# Requirements — deity

Version: 3.1
Last updated: 2026-10-15
Status: in-progress — AC 1–22 (except the parts marked **v3.0**) are implemented (top / paged list with `type` filter / detail, seed `cmd/seeddeities`); E2E exists only for the `type` filter. **v3.0 (2026-10-15) not implemented**: page size 12, ngày vía (vía days) taken from the festival calendar (`calendar`) with the next vía occurrence computed by the server, dropping the separately stored `lunar_memorial_days`, putting the catalogue file in git (AC 5, 8, 12, 15, 16, 23–34; `tasks.md` Phase 8–10).
Module: `deity`
Code: `services/appService/`
Product spec: `docs/product/deity/product.md` @ v1.0
Contract: `docs/specs/api/openapi.yaml` (this repo)
Implements: BR-DTY-01, BR-DTY-02, BR-DTY-03, BR-DTY-05, BR-DTY-06, FR-DTY-001…007, FR-DTY-011, NFR-DTY-01, NFR-DTY-02

> See `changelog.md` in this folder for the version history of this spec.

This spec was originally written **retroactively** from the already-implemented Phật (Buddha) / Bồ tát (Bodhisattva) catalogue; from v3.0 the business source is the product spec above (Mixed mode: the shipped part is Retroactive, the **v3.0** part is Prospective). This spec does not restate the product spec's rules — it only cites IDs and says how the backend does it. Where this spec diverges from the code in the shipped part, the code is the source of truth and the spec is corrected to match the code. `api-contract.md` in the same folder is an old client document, stale since v3.0 (see the banner in that file); the contract is `openapi.yaml`.

The catalogue is **read-only**, public, and shaped like `temple` and `article`. One `deities` table holds all three types (`BUDDHA`, `BODHISATTVA`, `DEVA`); `top` and the paged list mix the types in the same order (BR-DTY-02). Data is loaded by the manually run `cmd/seeddeities` command, not automatically when the service starts.

Product IDs whose Backend column is `n/a` (not part of this spec): BR-DTY-04, BR-DTY-07, BR-DTY-08, FR-DTY-008, FR-DTY-009, FR-DTY-010. The server side of BR-DTY-04 is only a deterministic data shape (US-DTY-04); hiding empty fields is the client's job. BR-CAL-13 (countdown) belongs to the client: the server returns the date of the vía occurrence, the client computes "Hôm nay / Ngày mai / Còn n ngày" (Today / Tomorrow / n days left).

---

## US-DTY-01: View featured Buddhas and Bodhisattvas

## User Story

> As an **app or website user (no login needed)**,
> I want **to see a few representative Buddhas and Bodhisattvas right on the home screen and in the library**,
> So that **I can get acquainted and tap to learn more**.

## Acceptance Criteria (EARS notation)

1. WHEN a `GET /v1/deities/top` request arrives, THE SYSTEM SHALL return `200 OK` with `data.items` holding the first `deity.top_limit` deities of **any type** (`top_limit` = 10) ordered by `sort_order` ASC then `id` ASC. (No `Authorization` needed.) (FR-DTY-001, BR-DTY-02)
2. WHEN deities are returned by `top` or by the paged list, THE SYSTEM SHALL return each one in the **summary** shape only — `id`, `type`, `name_vi`, `name_sanskrit`, `name_chinese`, `title_honorific`, `thumbnail_url`, `image_url`, `core_virtue`, `pure_land` and, **v3.0**, `next_via` (AC 24, 29) — and SHALL NOT include `canonical_source`, `lunar_memorial_days`, `via_events`, `iconography`, `biography_summary`, `great_vows` or `mantras`. (FR-DTY-001, FR-DTY-002, BR-DTY-06)
3. WHEN the table holds fewer deities than `deity.top_limit`, or none (seed not yet run), THE SYSTEM SHALL return all that exist, and SHALL return `items: []` (not `null`) for an empty table. (BR-DTY-02)

## US-DTY-02: Browse Buddhas and Bodhisattvas page by page

## User Story

> As an **app or website user (no login needed)**,
> I want **to browse all Buddhas and Bodhisattvas page by page, filtered by type**,
> So that **I can learn about the ones beyond the featured set**.

## Acceptance Criteria (EARS notation)

4. WHEN a `GET /v1/deities` request arrives without `page` (or with an empty `page=`) and without `type` (or with an empty `type=`), THE SYSTEM SHALL serve page 1 of the deities of **any type** (`BUDDHA`, `BODHISATTVA` and `DEVA` together), ordered by `sort_order` ASC then `id` ASC. Each item carries the `type` field so the client can tell them apart. (FR-DTY-002, BR-DTY-02)
5. WHEN a page is served, THE SYSTEM SHALL return `data` with `items` (summary shape), `page`, `page_size` (`deity.page_size`; **v3.0: 12**, the code currently uses 20), `total` (the number of entries matching the `type` filter if any, otherwise the number of entries in the whole table) and `total_pages` = `ceil(total / page_size)`. (FR-DTY-002, BR-DTY-03)
6. WHEN `page` is greater than `total_pages`, THE SYSTEM SHALL return `200 OK` with `items: []` and the same `page_size`/`total`/`total_pages` — not an error. (BR-DTY-03)
7. WHEN `page` is not a positive integer (`0`, negative, non-numeric, an integer that overflows, or larger than 2^31−1), THE SYSTEM SHALL return `400 INVALID_PAGE` with `details.field = "page"`. (platform)
20. WHEN a `GET /v1/deities?type=T` request arrives with `T` exactly one of `BUDDHA`, `BODHISATTVA`, `DEVA`, THE SYSTEM SHALL return only deities of that type, in the same order (`sort_order` ASC then `id` ASC), and `total` / `total_pages` SHALL count only that type. (The product has no "Chư Thiên" (Devas) chip; `type=DEVA` is still valid at the API.) (FR-DTY-003, BR-DTY-01, BR-DTY-02)
21. WHEN `type` is present and non-empty but not exactly one of the three values (including a different case such as `buddha`), THE SYSTEM SHALL return `400 INVALID_DEITY_TYPE` with `details.field = "type"`. (platform)
22. WHEN `type` is combined with `page`, THE SYSTEM SHALL page within the filtered list (offset = `(page-1) * page_size` on the filtered list); WHEN several parameters are invalid, THE SYSTEM SHALL report `INVALID_PAGE` first, then `INVALID_DEITY_TYPE`, then (**v3.0**) `INVALID_DATE` (AC 26). (FR-DTY-003)

> Numbering note: AC 20–22 were added in v2.2, AC 23–34 in v3.0; all are appended at the end so older AC references do not shift.

## US-DTY-03: View the detail of one Buddha or Bodhisattva

## User Story

> As an **app or website user (no login needed)**,
> I want **to open a Buddha or Bodhisattva and read everything about them**,
> So that **I know their vows, vía days, iconography and recitation phrases**.

## Acceptance Criteria (EARS notation)

8. WHEN a `GET /v1/deities/{id}` request arrives with an existing `id` of any type, THE SYSTEM SHALL return `200 OK` with the summary fields (incl. **v3.0** `next_via`) plus `canonical_source`, `iconography` (`posture`, `mount_or_throne`, `sacred_artifacts`, `symbolic_meaning`), `biography_summary`, `great_vows` (stored order), `mantras` (`short_mantra_sanskrit`, `short_mantra_sino_vietnamese`, `long_dharani_name`, `dharani_phonetic_snippet`), **v3.0** `via_events` (AC 28) and `lunar_memorial_days` (**v3.0: deprecated**, derived per AC 30 instead of read from a separate column). (FR-DTY-004, FR-DTY-005, FR-DTY-006)
9. WHEN `{id}` matches no deity, THE SYSTEM SHALL return `404 DEITY_NOT_FOUND`. (FR-DTY-004)
10. WHEN the literal path `/v1/deities/top` is requested, THE SYSTEM SHALL route it to the top list, not to detail (`ServeMux` prefers the literal segment). `id` has the form `buddha_*` / `bodhisattva_*` / `deva_*`, so it never collides with `top`. (platform)

## US-DTY-04: Consistent data shape

## User Story

> As a **client developer**,
> I want **fields with missing data to always have a deterministic shape**,
> So that **I can decode once without guessing between `null` and an empty string**.

## Acceptance Criteria (EARS notation)

11. WHEN a deity has no value for a text field (`name_sanskrit`, `name_chinese`, `title_honorific`, `core_virtue`, `pure_land`, `canonical_source`, `biography_summary`, the iconography and mantra strings), THE SYSTEM SHALL return `""`, never `null`. (platform — the client hides it per BR-DTY-04)
12. WHEN a deity has no sacred artifacts, great vows or (**v3.0**) linked vía events, THE SYSTEM SHALL return `[]` for `iconography.sacred_artifacts` / `great_vows` / `via_events` / `lunar_memorial_days`, never `null`; and WHEN a deity has no upcoming vía occurrence (AC 24), THE SYSTEM SHALL return `next_via: null`. (platform, BR-DTY-06)
13. WHEN `thumbnail_url` / `image_url` are returned, THE SYSTEM SHALL turn a stored root-relative path (`/static/images/buddhas/{thumbnails,full}/<id>.webp`, migration `0021`) into an absolute URL by prefixing `server.public_base_url`, or the request's own scheme and host when that is empty (dev); SHALL return an already absolute `http(s)://` value unchanged; and SHALL return an empty value as `""`, never `null` (unlike `temple` and `article`). (platform)
14. WHEN any deity route fails, THE SYSTEM SHALL answer in the shared envelope `{"success": false, "error": {code, message, type}, "details"?}`; a non-`GET` request returns `405 METHOD_NOT_ALLOWED` (with an `Allow` header), an unknown path returns `404 NOT_FOUND`, and an infrastructure failure returns `500 INTERNAL_ERROR` without leaking the cause. (platform)

## US-DTY-05: Load Buddha / Bodhisattva data with a seed command

## User Story

> As a **system operator**,
> I want **to load and update the catalogue from a data file with one command**,
> So that **I can change the order or content in the file and re-run it without creating duplicates or leaving half-written data**.

## Acceptance Criteria (EARS notation)

15. WHEN the operator runs `go run ./services/appService/cmd/seeddeities [-file <path>]`, THE SYSTEM SHALL read the file (default `assets/data/phat_botat.json`, relative to the current directory), validate it, and upsert every deity in **one transaction** keyed by `id`, with `sort_order` equal to the deity's position in the file (0, 1, 2, …) and, **v3.0**, replace each deity's vía links with the file's `via_event_codes` in the same transaction, then log `seeded N deities from <file>`. (FR-DTY-011, BR-DTY-02, BR-DTY-05)
16. WHEN the seed is validated, THE SYSTEM SHALL reject the whole file (write nothing) if it has no deity, an empty or duplicate `id`, a `type` other than `BUDDHA` / `BODHISATTVA` / `DEVA`, an empty `name_vi`, or, **v3.0**, the same vía event code twice on one deity, or a vía event code that does not exist in `calendar_events` (the command logs the list of missing codes). (FR-DTY-011, BR-DTY-01, BR-DTY-05)
17. WHEN the seed is run again after the file was edited or reordered, THE SYSTEM SHALL update the existing rows in place by `id` — overwriting every column from the file, including `sort_order`, and setting `updated_at = now()` — and SHALL NOT create a second row for an existing `id`. (FR-DTY-011)
18. WHEN the file is invalid, the database is unreachable, or the database rejects a row (e.g. a migration not yet applied), THE SYSTEM SHALL exit non-zero with a fatal log and SHALL leave the `deities` table and (**v3.0**) the vía links unchanged (transaction rolled back). (FR-DTY-011)
19. WHEN appService starts, THE SYSTEM SHALL NOT seed or otherwise write the `deities` table — a DB that has not been seeded returns `items: []` (AC 3, 6), not an error. (platform)
33. **v3.0** — THE SYSTEM SHALL keep the catalogue file `assets/data/phat_botat.json` in version control (a `.gitignore` exception like `administrative-units.json`), so `make ensure-deities` on a fresh clone, the E2E harness and `TestParseDeities_RealDataFile` all read the same committed file. (FR-DTY-011)

## US-DTY-06: Vía days taken from the festival calendar (v3.0)

## User Story

> As an **app or website user**,
> I want **to see each deity's next vía occurrence on its card and every vía day on the detail page, exactly as in the festival calendar**,
> So that **I never have to convert lunar dates myself and never see two places in the app disagree**.

## Acceptance Criteria (EARS notation)

23. THE SYSTEM SHALL take a deity's vía days only from the festival-calendar events linked to that deity (a deity ↔ `calendar_events.code` link table, loaded from the catalogue file), with their solar and lunar dates computed by the calendar's own rules (`domain.ResolveOccurrences`, BR-CAL-01…08), and SHALL NOT store or compute vía dates of its own. (BR-DTY-05)
24. WHEN a deity is returned by any of the three routes, THE SYSTEM SHALL set `next_via` to the occurrence, among the occurrences of all its linked active events in solar years `Y` and `Y+1` (`Y` = the year of "today", AC 25; `Y+1` skipped when `Y = 2100`), with the earliest `solar_date` on or after today (today included), ties broken by the event's calendar `sort_order` then `code` (BR-CAL-12); and to `null` when there is none. (BR-DTY-06)
25. WHEN a deity route request carries `date` (`YYYY-MM-DD`, year 1900..2100), THE SYSTEM SHALL use it as "today" for AC 24 and AC 28; WHEN `date` is absent or empty, THE SYSTEM SHALL use today's date in `calendar.timezone` (dev: `Asia/Ho_Chi_Minh`). (The client sends the device date to match BR-DTY-06 "today = device date".) (BR-DTY-06)
26. WHEN `date` is present and non-empty but not a real date in `YYYY-MM-DD` form or outside 1900..2100, THE SYSTEM SHALL return `400 INVALID_DATE` with `details.field = "date"` on `top`, the list and the detail alike (same rule as `GET /v1/calendar/image`). (platform)
27. WHEN a linked calendar event is inactive (`is_active = false`), THE SYSTEM SHALL ignore it for `next_via`, `via_events` and `lunar_memorial_days`, as if it were not linked. (BR-DTY-05, BR-CAL-16)
28. WHEN a deity detail is returned, THE SYSTEM SHALL list in `via_events` every linked active event exactly once, each with its next occurrence on or after today in years `Y`..`Y+1` (`next_occurrence`, or `null` when it has none in that window), ordered by `next_occurrence.solar_date`, then calendar `sort_order`, then `code` (BR-CAL-12), events without an occurrence last; the first item with an occurrence is the same as `next_via`. (FR-DTY-006, BR-DTY-05, BR-DTY-06)
29. WHEN `next_via` or a `via_events` item is returned, THE SYSTEM SHALL give the event's `code` and `name` (in the `calendar.default_language` language) and the occurrence as `solar_date` (`YYYY-MM-DD`) plus `lunar_date` (`day`, `month`, `year`, `is_leap_month`), in the same field shape as `GET /v1/calendar/events` occurrences, so no client converts lunar dates. (FR-DTY-002, FR-DTY-006, BR-CAL-08)
30. WHEN a deity detail is returned, THE SYSTEM SHALL fill the deprecated `lunar_memorial_days` (`[{lunar_day, lunar_month, event_name}]`) from `via_events`, one entry per item that has a `next_occurrence` (the lunar day/month of that occurrence, the event name), in the same order, so it never disagrees with the calendar. (BR-DTY-05)
31. WHEN `top`, a list page or a detail is served, THE SYSTEM SHALL read the deities and their vía links in the page query itself and the calendar catalogue in one further query, so the number of queries per request does not grow with the catalogue or the page size. (NFR-DTY-01, NFR-DTY-02)
32. THE SYSTEM SHALL pass every case of `docs/product/deity/rules/deity-order.json` (BR-DTY-02, repository test on Postgres) and the choice of `event` / `solar_date` / `days` in every case of `docs/product/deity/rules/deity-next-via.json` (BR-DTY-06, pure-function unit test), loading both files from `docs/product/`, not from a copy. (BR-DTY-02, BR-DTY-06)

## Related sutras

34. WHEN a client shows the related sutras of a deity, THE SYSTEM SHALL serve them only through the existing `GET /v1/deities/{id}/sutras` (owned by `docs/specs/features/sutra/`, AC 20–24 there), and the deity routes SHALL NOT embed sutras. (FR-DTY-007)

> ⚠️ Not confirmed (product): FR-DTY-007 says "in the sutra library's order", while `GET /v1/deities/{id}/sutras` sorts by role (`SUBJECT`, `SPEAKER`, `MENTIONED`) then `sort_order` — is that the order product wants, or only `sort_order`? — blocks the client from showing the right order (the backend changes nothing if the current order is right); hand back to ba-writer in innerlamp-be.

## API contract (confirmed surface for this feature)

Read from the handler and `cmd/main.go`; the three routes are already in `docs/specs/api/openapi.yaml` (tag Deities). The **v3.0** parts below are this spec's design and are **not yet** in `openapi.yaml` — update it in the same pass as the implementation (`tasks.md` Phase 9).

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/deities/top?date=` | public | `200` `{items: [DeitySummary]}` (≤ 10, all types) | 400 `INVALID_DATE` (v3.0), 404 `NOT_FOUND` (wrong path), 405, 500 |
| GET | `/v1/deities?type=&page=N&date=` | public | `200` `{items, page, page_size (12, v3.0), total, total_pages}` | 400 `INVALID_PAGE`, 400 `INVALID_DEITY_TYPE`, 400 `INVALID_DATE` (v3.0), 405, 500 |
| GET | `/v1/deities/{id}?date=` | public | `200` `DeityDetail` | 400 `INVALID_DATE` (v3.0), 404 `DEITY_NOT_FOUND`, 405, 500 |
| GET | `/v1/deities/{id}/sutras` | public | owned by the `sutra` spec (AC 34) | owned by the `sutra` spec |

`DeitySummary`: `id`, `type` (`BUDDHA` \| `BODHISATTVA` \| `DEVA`), `name_vi`, `name_sanskrit`, `name_chinese`, `title_honorific`, `thumbnail_url`, `image_url` (string, absolute or `""`), `core_virtue`, `pure_land`, **v3.0** `next_via` (`DeityVia` \| `null`).

`DeityVia` (v3.0): `{event_code, name, solar_date, lunar_date: {day, month, year, is_leap_month}}`.

`DeityDetail` = `DeitySummary` + `canonical_source`, `iconography` `{posture, mount_or_throne, sacred_artifacts: [string], symbolic_meaning}`, `biography_summary`, `great_vows` (`[string]`), `mantras` `{…4 strings}`, **v3.0** `via_events` (`[{event_code, name, next_occurrence: {solar_date, lunar_date} | null}]`), `lunar_memorial_days` (`[{lunar_day, lunar_month, event_name}]`, **deprecated since v3.0**).

Changes versus the running contract: `page_size` 20 → 12 (any client that assumes 20 must read `page_size`); new `next_via`, `via_events`, `date` parameter, `INVALID_DATE` error — all additive, not breaking; `lunar_memorial_days` is still present but its source changes (it may have fewer entries than today if an old vía day has no matching calendar event).

> ⚠️ Not confirmed: when to remove `lunar_memorial_days` from `DeityDetail` entirely (after the website, iOS and Android have moved to `via_events`, or keep it until a minimum app version milestone) — blocks the field-removal item of `tasks.md` Phase 10; does not block the rest.

## Out of scope (this feature boundary, not tracked here)

- Creating/editing/deleting entries via an API or backoffice — only `cmd/seeddeities` and direct SQL (product §1.2).
- Search, favourites, following, vía-day reminders, "Báo thông tin sai" (Report incorrect information) (product §1.2, BR-DTY-08).
- Filtering by several `type` values at once; any filter other than `type`.
- Content, recurrence rules and date placement of calendar events: owned by `docs/specs/features/calendar/` — this spec only reads the event catalogue and calls its date calculator.
- The countdown and the "Hôm nay / Ngày mai / Còn n ngày" (Today / Tomorrow / n days left) labels (BR-CAL-13), the "Vía dd/MM âm lịch" (Vía dd/MM lunar) format, the "GIÊNG/CHẠP" (first / twelfth lunar month) month names, hiding empty fields (BR-DTY-04), collapsing the đại nguyện (great vows) (BR-DTY-07): client.
- The related-sutras list: route and order owned by the `sutra` spec (AC 34).
- Image storage on a CDN / object store: images are currently served dev-only via `/static/images/` (`server.static_images_dir`).

## Pending (not yet implemented)

- Every AC/part marked **v3.0**: page size 12 (AC 5), `next_via` / `via_events` / `date` / `INVALID_DATE` (AC 2, 8, 12, 22–31), seeding vía links (AC 15, 16, 18), catalogue file in git (AC 33), vectors (AC 32) — `tasks.md` Phase 8–10.
- `TestParseDeities_RealDataFile` **fails** in this checkout because `assets/data/phat_botat.json` is caught by `.gitignore` (`*.json`) and is not on disk; the E2E harness also reads this file — AC 33.
- E2E for `top` and `detail` — `tasks.md` Phase 5.
- OKF concept `docs/knowledge/features/deity.md` — does not exist yet; `il-okf-writer` creates it after v3.0 ships.

> ⚠️ Not confirmed (product): for each of the 20 deities, which festival-calendar events are that deity's vía days, and when a vía day present in the catalogue file has no matching event in the festival calendar, does the calendar add a new event or is that day dropped (BR-DTY-05 only says the calendar is the single source) — blocks the content of `via_event_codes` in the catalogue file (`tasks.md` Phase 8, data item); both seed files (`phat_botat.json`, `calendar-events.json`) are missing from this checkout, so the spec could not cross-check them; hand back to ba-writer in innerlamp-be.
> ⚠️ Not confirmed (product): the "featured" criterion (product Appendix A item 2) — the backend keeps BR-DTY-02 (file order), only the file content may change; blocks no task.
> ⚠️ Not confirmed (product): who reviews the religious content and image rights before release (product Appendix A item 1) — blocks content release, blocks no task here.
> ⚠️ Not confirmed (product): who runs the seed on staging/production and from which version of the file (product Appendix A item 4); `cmd/seeddeities` still uses dev config + secrets (`⚠️ DEV ONLY`) — blocks the "seed outside dev" item of `tasks.md` Phase 3.
> ⚠️ Not confirmed: where production images live (which CDN / object store) — today images are files in `assets/images/buddhas/` served dev-only; only `server.public_base_url` needs setting once there is a host (AC 13), no re-seed.

## Technical notes

- Data: 20 deities — 7 `BUDDHA` (including `buddha_seven_past`, Thất Phật Quá Khứ (the Seven Past Buddhas)), 12 `BODHISATTVA`, 1 `DEVA` (`deva_sri_mahadevi`, Đại Cát Tường Thiên Nữ (Śrī Mahādevī); migration `0015`). This count is cross-checked from `parse_test.go` (`require.Len(t, got, 20)`) and the 20 images in `assets/images/buddhas/full/`; the catalogue file is missing, so the order of each entry could not be cross-checked. With page size 12: 2 pages (12 + 8).
- Images are stored as root-relative paths `/static/images/buddhas/{thumbnails,full}/<id>.webp` (migration `0021`); the handler prefixes `server.public_base_url` (empty = the request's origin, dev) via `imageHost`, shared with `temple`, `gallery`, `article`, `sutra`.
- `id` is a stable string and the seeder upserts by `id`; `sort_order` = position in the file and is the only ranking source; `id` is the tie-break (BR-DTY-02).
- **v3.0 — vía links**: a join table `deity_id` ↔ `calendar_events.code` (new migration), kept in the deity catalogue file (product §3.5: "kept with the deity"). The deity seed therefore needs the calendar catalogue loaded first (load order: `calendar` → `deity` → `sutra`). The `deities.lunar_memorial_days` column is dropped.
- **v3.0 — who computes the next vía occurrence**: **the server computes it** (BR-DTY-06 leaves it to the contract). Reason: the client need not convert lunar ↔ solar, need not download a whole year's calendar for one card, and every platform gets the same result; the client only counts down (BR-CAL-13). "Today" is the device date sent via `date`; without `date` the server uses `calendar.timezone` — same as `GET /v1/calendar/image`. Because the result depends on the date, the response adds no public cache header.
- The `Y`..`Y+1` window is enough for every yearly recurring event; an event that only exists in years with a leap month (BR-CAL-04) may have no occurrence in the window → `next_occurrence: null`.
- Page sizes live in the `deity:` block of `config.<env>.yaml` (dev currently `top_limit: 10`, `page_size: 20` → v3.0 `12`).

## Open questions (resolved during implementation)

- 2026-10-15 (v3.0): **User story codes** — product spec v1.0 assigns the `BR-DTY-*` / `FR-DTY-*` IDs; every AC is tagged with those IDs. `US-DTY-01..06` are kept as this spec's internal group headings, not product codes. Impact: no change to `design.md`/`tasks.md`.
- 2026-10-15 (v3.0): **Images do not resolve** (v1.0 marker) — resolved in code by migration `0021_relative_image_urls.sql` + `imageHost` + 20 image files in `assets/images/buddhas/`; AC 13 rewritten to match the code. The remaining questions (production host, image rights) keep their own marker under Pending.
- 2026-10-15 (v3.0): **Page size, vía-day source, next vía occurrence, related sutras, catalogue file** — settled by product spec v1.0 (BR-DTY-03, BR-DTY-05, BR-DTY-06, FR-DTY-007, Appendix B). This spec's technical decisions (pending review via PR): the server computes `next_via`; a `date` parameter for "today"; the deity ↔ calendar-event join table lives in the deity catalogue file; `lunar_memorial_days` kept for now, derived from `via_events`. Impact: `design.md` adds a vía-attaching Step to all three read flows, the seed adds an event-code check step; `tasks.md` Phase 8–10.
- 2026-10-12 (v2.2): **Is there an API to filter Buddhas / Bodhisattvas by `type`?** — added a `type` parameter on `GET /v1/deities`, exact match on the three values, empty = no filter, unknown value = `400 INVALID_DEITY_TYPE`, one value per request (following the `gallery` precedent). Product v1.0 FR-DTY-003 confirms the filter.
- 2026-09-30 (v2.0): **How to view Buddhas** — `GET /v1/bodhisattvas` replaced by `GET /v1/deities?page=N`, combining every type.
