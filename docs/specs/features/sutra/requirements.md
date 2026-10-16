# Requirements — sutra

Version: 4.0
Last updated: 2026-10-16
Status: implemented, with one open v4.0 part — 4 routes and the `cmd/seedsutras` command are shipped and covered by repository, handler and E2E tests (153 texts, 391 versions, 64,009 lines, 151 deity links). **v4.0 not implemented**: the seeder and the contract refuse a text without a duration estimate (AC 31, `tasks.md` Phase 7), the vector files are not run yet (AC 4, 8, `tasks.md` Phase 7), and the search route has no latency measurement (AC 30). Search by name (`q`) and the `audio_url` value are shipped (migration `0019`) although v3.0 listed them as out of scope.
Module: `sutra`
Code: `services/appService/`
Product spec: `docs/product/sutra/product.md` @ v1.0
Contract: `docs/specs/api/openapi.yaml` (this repo)
Implements: BR-STR-01, BR-STR-02, BR-STR-03, BR-STR-04, BR-STR-05, BR-STR-06, BR-STR-07, BR-STR-08, BR-STR-09, BR-STR-12, BR-STR-13, FR-STR-001, FR-STR-002, FR-STR-003, FR-STR-004, FR-STR-005, FR-STR-006, FR-STR-008, FR-STR-009, FR-STR-012, NFR-STR-01, NFR-STR-02, NFR-STR-03

> See `changelog.md` in this folder for the version history of this spec.

This spec was first written prospectively (v1.0, 2026-10-03) and implemented in the same month; v4.0 (2026-10-16) rebases it on the product spec above, rewrites it in English per the Language policy and brings it in line with the code (Retroactive for the shipped part, Prospective for the v4.0 part). It does not restate the product's rules — it cites IDs and says how the backend does them. Where the spec and the shipped code disagree, the code is the source of truth for the shipped part.

The sutra library is a **read-only, public** catalogue shaped like `temple`, `deity`, `article` and `gallery`: a paged list, a detail, and two things the others lack — each text has **several versions** (language × variant) whose lines are served **page by page** (the longest version has 6,967 lines), and each text is **linked to Buddhas and bodhisattvas** (`deities`) in both directions. Four tables (migration `0013`) hold it. The data comes from JSON files that are a **one-time source**: the manually run `cmd/seedsutras` loads them into an empty table, and every route reads Postgres afterwards (the service itself never seeds).

Wire values map to the product's labels (BR-STR-01) as: kind `SUTRA` → Kinh, `VERSE` → Kệ, `DHARANI` → Chú; tradition `THERAVADA` → Nam Tông, `MAHAYANA` → Bắc Tông, `VAJRAYANA` → Kim Cang Thừa; purposes `CAU_AN`, `CAU_SIEU`, `TRI_CHU`, `SAM_HOI`, `HOI_HUONG`, `PHAT_NGUYEN`, `BAO_HIEU`, `THIEN_DINH`, `HOC_GIAO_LY`, `QUY_Y_THO_GIOI` → the ten purposes of BR-STR-03. The labels themselves are the client's.

Product IDs whose Backend column is `n/a` or "Website, iOS, Android" only (not part of this spec): BR-STR-10 (quote of the day is chosen by the clients), BR-STR-11 (reading preferences), FR-STR-007, FR-STR-010, FR-STR-011. BR-STR-05 is decided by the clients; the backend only keeps the versions in source order (AC 14). BR-STR-06's line text is served unchanged (AC 21).

---

## US-SUT-01: Browse, filter and search the library page by page

## User Story

> As an **app or website user (no login needed)**,
> I want **to browse the library page by page, narrow it by topic and duration, and search it by name**,
> So that **I find the text I want to read or chant**.

## Acceptance Criteria (EARS notation)

1. WHEN a `GET /v1/sutras` request arrives without `page` (or with an empty `page=`), THE SYSTEM SHALL serve page 1 and return `200 OK` with `data` holding `items` (card shape), `page`, `page_size` (`sutra.page_size`, dev: 20), `total` (the number of texts matching the filters) and `total_pages` = `ceil(total / page_size)`, ordered by `sort_order` ASC then `id` ASC. (No `Authorization` needed.) (FR-STR-001, BR-STR-02, BR-STR-13)
2. WHEN texts are returned by the list, THE SYSTEM SHALL return each one in the **card shape**: `id`, `tradition`, `kind`, `collection`, `reference`, `summary`, `titles`, `daily_chant`, `purposes`, `duration_minutes`, `audio_url`, `deities` and `translations`, and SHALL NOT include any line text, `sections`, `source_url`, `source_name`, `note`, `review_status` or the other detail-only fields of AC 12–13. (FR-STR-001, BR-STR-01)
3. WHEN the request carries any of `tradition`, `kind`, `collection`, `purpose`, `daily`, `max_minutes`, `lang`, THE SYSTEM SHALL return only the texts matching **all** supplied filters (AND), keep the same ordering, and compute `total` / `total_pages` over the matching texts only (FR-STR-002, BR-STR-03):
   - `tradition` ∈ `THERAVADA` | `MAHAYANA` | `VAJRAYANA`
   - `kind` ∈ `SUTRA` | `VERSE` | `DHARANI`
   - `collection` — exact match with `collection`
   - `purpose` — the text has this value among its `purposes`
   - `daily` ∈ `true` | `false` — matches `daily_chant`
   - `max_minutes` — integer ≥ 1; only texts whose `duration_minutes` is **at most** this value (the limit itself is included); a text with `duration_minutes = null` is excluded
   - `lang` — the text has at least one version in this language (e.g. `vi`)
4. WHEN the request carries `q`, THE SYSTEM SHALL trim it and return only the texts where the term appears anywhere inside **any title (every language in `titles`), the collection name or the citation code (`reference`)**, ignoring letter case and Vietnamese diacritics (`đ` equals `d`), treating `%`, `_` and `\` literally; THE SYSTEM SHALL NOT search the summary, the translators or the lines; and SHALL combine `q` with the other filters (AND), keep the library order (AC 1) and not reorder by relevance. (FR-STR-003, BR-STR-04, BR-STR-02) — covered by the vector files `sutra-browse.json` and `sutra-search-match.json` (`tasks.md` Phase 7).
5. WHEN a filter value is not allowed — an unknown `tradition`, `kind` or `purpose`; a `daily` that is not `true`/`false`; a `max_minutes` that is not an integer ≥ 1 (or exceeds 2^31−1); a `q` that is empty or whitespace only after trimming, or longer than 100 characters (runes) after trimming; or any other supplied filter that is empty — THE SYSTEM SHALL return `400 INVALID_FILTER` with `details.field` set to the offending parameter name. (A client with an empty search box omits `q`.) (FR-STR-002, FR-STR-003, BR-STR-04)
6. WHEN `page` is greater than `total_pages`, THE SYSTEM SHALL return `200 OK` with `items: []` and the same `page_size`/`total`/`total_pages` — not an error; WHEN the table is empty or no text matches, THE SYSTEM SHALL return `items: []` (not `null`), `total: 0` and `total_pages: 0`. The client stops when `page >= total_pages` or `items` is empty. (FR-STR-001)
7. WHEN `page` is not a positive integer (`0`, negative, non-numeric, an integer that overflows, or larger than 2^31−1), THE SYSTEM SHALL return `400 INVALID_PAGE` with `details.field = "page"`. (FR-STR-001)
8. THE SYSTEM SHALL order and filter the list so that every case of `docs/product/sutra/rules/sutra-browse.json` and `docs/product/sutra/rules/sutra-search-match.json` gives the expected identifiers in the expected order. (BR-STR-02, BR-STR-03, BR-STR-04)
9. WHEN a text is linked to deities, THE SYSTEM SHALL return them in `deities` as `{deity_id, role, name_vi, thumbnail_url}`, ordered by role (`SUBJECT`, `SPEAKER`, `MENTIONED`) then `deity_id`, so the client shows the related Buddhas and bodhisattvas without a second request. (FR-STR-008, BR-STR-09)

## US-SUT-02: Open a sutra page

## User Story

> As an **app or website user (no login needed)**,
> I want **to open one text and see everything about it before reading**,
> So that **I know which versions exist, who translated them, where they come from, what the text contains and which part I want to jump to**.

## Acceptance Criteria (EARS notation)

10. WHEN a `GET /v1/sutras/{id}` request arrives with an existing `id`, THE SYSTEM SHALL return `200 OK` with the card fields plus `review_status`, and each element of `translations` extended with `source_name`, `source_url`, `note`, `license_url`, `license_note` and `sections`. (FR-STR-004, BR-STR-07)
11. WHEN `{id}` matches no text, THE SYSTEM SHALL return `404 SUTRA_NOT_FOUND`. (FR-STR-004)
12. WHEN the detail is returned, THE SYSTEM SHALL return these flat fields, each its own JSON field and never packed into another: `original_title`, `author`, `han_translator`, `reference_note` (strings, `""` when the source has none) and `han_translation_year` (integer year CE | `null`). THE SYSTEM SHALL NOT return these five fields in the list or the by-deity list. (FR-STR-004, BR-STR-07)
13. THE SYSTEM SHALL keep in `reference` only the citation code of the text (e.g. `Taishō T0450`, `MN 86`, `Dhp 1-20`) or `""` when it has none; the original title, the author, the Chinese translator, the year and any remark live in the fields of AC 12. (BR-STR-07)
14. WHEN a text has several versions, THE SYSTEM SHALL distinguish them by `key` (unique within the text) and by `variant` (`translation` | `chant` | `meaning` | `poem`), and return them in source order (the `ordinal` of the data file) — the clients choose the version that opens (BR-STR-05) from `lang`, `variant` and this order. (FR-STR-006, BR-STR-05)
15. WHEN a version has sections, THE SYSTEM SHALL return `sections` as `[{name, start_n, line_count}]` in file order, where `start_n` is the `n` of the section's first line, so the client builds the table of contents and computes the page of a section as `floor((start_n − 1) / page_size) + 1`; WHEN a version has none, THE SYSTEM SHALL return `[]`. (FR-STR-005, BR-STR-06)
16. THE SYSTEM SHALL return, for every version in the list, the by-deity list and the detail, `license` as one code from the closed set `CC0-1.0` | `CC-BY-NC-4.0` | `PUBLIC_DOMAIN` | `COPY_ALLOWED` | `UNKNOWN`, together with `translator`; and, in the detail only, `license_url` (the URL the source gives, `""` when none) and `license_note` (the source's own statement, `""` when none), so the label and the explanation are separate fields. (BR-STR-07)
17. WHEN a version's `license` is `UNKNOWN`, THE SYSTEM SHALL still return the version unchanged: the code is descriptive and never hides a text; whether the licence line is shown is the client's (BR-STR-07). (BR-STR-07)
18. THE SYSTEM SHALL return `review_status` in the detail (`UNREVIEWED` for every text today) so the client can show the notice while a text is unreviewed. (BR-STR-07)
19. THE SYSTEM SHALL return `summary` (string, Vietnamese, `""` when none was written) on every card of the list and the by-deity list and in the detail, the same value everywhere. (FR-STR-001, FR-STR-004)
20. THE SYSTEM SHALL return `audio_url` on every card and in the detail as a string, and SHALL return `""` for a text that has no recording (never `null`, never omitted); today no text has one, so every response says "no audio" and the clients decide what to show. (BR-STR-12, FR-STR-004)

## US-SUT-03: Read a version page by page

## User Story

> As an **app or website user (no login needed)**,
> I want **to read a long text a page of lines at a time and open it straight at the page I am reading**,
> So that **I never wait for thousands of lines to load**.

## Acceptance Criteria (EARS notation)

21. WHEN a `GET /v1/sutras/{id}/lines?translation=<key>` request arrives for an existing text and an existing version `key`, THE SYSTEM SHALL return `200 OK` with `data` holding `translation` (the `key`), `items` (`[{n, text, section}]` ordered by `n` ASC), `page`, `page_size`, `total` (the version's `line_count`) and `total_pages` = `ceil(total / page_size)`; each `text` SHALL be exactly as stored (no trimming, no spelling correction), `section` the section name or JSON `null` when the line belongs to none, and `n` SHALL run 1..`total` without gaps, so line `n` is always on page `floor((n − 1) / page_size) + 1`. (FR-STR-005, BR-STR-06)
22. WHEN `page` is absent (or empty), THE SYSTEM SHALL serve page 1; WHEN `page_size` is absent, THE SYSTEM SHALL use `sutra.lines_page_size` (dev: 200); THE SYSTEM SHALL return as `items` exactly the lines whose `n` is in `((page − 1) × page_size, page × page_size]`. (FR-STR-005, NFR-STR-01)
23. WHEN `page` is greater than `total_pages`, THE SYSTEM SHALL return `200 OK` with `items: []` and the same `page_size`/`total`/`total_pages` — not an error. (FR-STR-005)
24. WHEN `page_size` is greater than `sutra.lines_max_page_size` (dev: 500), THE SYSTEM SHALL serve `sutra.lines_max_page_size` lines per page instead of failing and echo the value actually used in `page_size`; WHEN `page_size` is not an integer or is less than 1, THE SYSTEM SHALL return `400 INVALID_PAGE_SIZE` with `details.field = "page_size"`. (FR-STR-005, NFR-STR-01)
25. WHEN `page` is not a positive integer (`0`, negative, non-numeric, an integer that overflows, or larger than 2^31−1), THE SYSTEM SHALL return `400 INVALID_PAGE` with `details.field = "page"`. (FR-STR-005)
26. WHEN the `translation` query parameter is absent or empty, THE SYSTEM SHALL return `400 INVALID_TRANSLATION` with `details.field = "translation"`. (FR-STR-005)
27. WHEN `{id}` matches no text, THE SYSTEM SHALL return `404 SUTRA_NOT_FOUND`; WHEN the text exists but has no version with that `key`, THE SYSTEM SHALL return `404 TRANSLATION_NOT_FOUND`. (The text is checked first.) (FR-STR-005)

## US-SUT-04: From a Buddha or bodhisattva to the related sutras

## User Story

> As an **app or website user (no login needed)**,
> I want **to see, on the page of a Buddha or bodhisattva, the texts about them or spoken by them**,
> So that **I read the texts tied to the one I am learning about**.

## Acceptance Criteria (EARS notation)

28. WHEN a `GET /v1/deities/{id}/sutras` request arrives with an existing deity `id`, THE SYSTEM SHALL return `200 OK` with `data` holding `items` (the linked texts in the card shape, each with an extra `role` = the deity's role in that text), `page`, `page_size` (`sutra.page_size`), `total` and `total_pages`, ordered by role (`SUBJECT`, `SPEAKER`, `MENTIONED`), then `sort_order` ASC, then `id` ASC. `page` is handled as in AC 1, 6 and 7. THE SYSTEM SHALL NOT cap the list; WHEN the deity is linked to more texts than one page holds, the client pages. (FR-STR-009, BR-STR-09)
29. WHEN the request carries `role`, THE SYSTEM SHALL return only the texts where the deity has that role (and compute `total` over them); WHEN `role` is not `SPEAKER` | `SUBJECT` | `MENTIONED` (or is empty), THE SYSTEM SHALL return `400 INVALID_FILTER` with `details.field = "role"`. (FR-STR-009)
30. WHEN `{id}` matches no deity, THE SYSTEM SHALL return `404 DEITY_NOT_FOUND` (the same code as `GET /v1/deities/{id}`); WHEN the deity exists but is linked to no text (or none with the requested `role`), THE SYSTEM SHALL return `200 OK` with `items: []`, `total: 0` and `total_pages: 0`. (FR-STR-009, BR-STR-09)
31. WHEN the path `/v1/deities/{id}/sutras` is requested, THE SYSTEM SHALL route it to this feature and SHALL NOT change the behaviour of `/v1/deities/top`, `/v1/deities` or `/v1/deities/{id}`. (platform)

## US-SUT-05: One predictable shape

## User Story

> As a **client developer**,
> I want **missing data to always have one fixed form**,
> So that **I decode once without guessing between `null` and an empty string**.

## Acceptance Criteria (EARS notation)

32. WHEN a text has no purposes, no deities or no versions, or a version has no sections, THE SYSTEM SHALL return `[]` for `purposes` / `deities` / `translations` / `sections`, never `null`. (platform)
33. WHEN a text is returned, THE SYSTEM SHALL return `titles` as an object mapping a language code to a title that always contains `vi` (the clients show the Vietnamese title whatever the app language, BR-STR-05), and `reference` as `""` when the text has no citation code. (BR-STR-05)
34. **v4.0** THE SYSTEM SHALL return `duration_minutes` as an integer number of minutes ≥ 1 for **every** text, unchanged from the stored editorial estimate (not rounded or recomputed), and SHALL NOT return `null`; the contract (`openapi.yaml`) SHALL declare it non-nullable. (BR-STR-08)
   > ⚠️ Not confirmed (product): BR-STR-08 says every text carries an editorial estimate, but the 10 Vajrayāna texts have none written by an editor — migration `0019` filled them as lines of the first version ÷ 4, rounded up (the migration says the seed JSON carries the same value; unchecked here because the JSON is not in git) — whether an editor must replace that formula value is open for `ba-writer`; this spec only makes the seeder and the contract require a value (AC 38) and keeps the current ones.
35. WHEN any sutra route fails, THE SYSTEM SHALL answer in the shared envelope `{"success": false, "error": {code, message, type}, "details"?}`; a non-`GET` request returns `405 METHOD_NOT_ALLOWED` (with header `Allow: GET, HEAD`) and an infrastructure failure returns `500 INTERNAL_ERROR` without leaking the cause. (platform)
36. THE SYSTEM SHALL serve the four routes without an `Authorization` header and without wrapping them in `RequireAuth`: a guest and a signed-in visitor get the same responses. (BR-STR-13)

## US-SUT-06: Load the library with the seed command

## User Story

> As a **deployer**,
> I want **one command that loads the library into an empty database, and that `make run` runs it when the table is empty**,
> So that **a new environment has the content, the service never writes to the database on start, and running it again never touches existing data**.

## Acceptance Criteria (EARS notation)

37. WHEN `cmd/seedsutras` runs AND the `sutras` table has no row, THE SYSTEM SHALL parse the JSON embedded in the binary (`assets/data/sutras/index.json` and every `<id>.json` it lists; flag `-dir` reads a directory on disk instead) and validate **all** of it before any write, and SHALL treat the seed as invalid, exit non-zero with a fatal log and write nothing, when any of these holds: (FR-STR-012)
    - a file listed in `index.json` is missing, or an `id` is empty, duplicated, or different from the file name
    - `tradition`, `kind`, `variant`, a deity `role` or any `purposes` value is outside the allowed sets; `titles.vi` is empty
    - `dailyChant` is not a boolean; `durationMinutes` is not an integer ≥ 1 (AC 38)
    - a version has no `key`, or two versions of one text share a `key`, or its `n` does not run 1..N without gaps
    - one text lists the same `deity_id` twice
    - a version's `license` is outside the closed set of AC 16, or `hanTranslationYear` is present but is neither `null` nor an integer ≥ 1 (the fields `originalTitle`, `author`, `hanTranslator`, `referenceNote`, `summary`, `licenseUrl` and `licenseNote` are optional and default to `""`)
38. **v4.0** WHEN a text in the seed has `durationMinutes` equal to `null` or missing, THE SYSTEM SHALL treat the seed as invalid as in AC 37 (until v4.0 the seeder accepted `null`). (BR-STR-08, FR-STR-012)
39. WHEN the seed is valid, THE SYSTEM SHALL check that every referenced `deity_id` exists in the `deities` table, and WHEN some do not, exit non-zero with a fatal log listing the missing ids and write nothing. (`cmd/seeddeities` runs first; `make ensure-sutras` keeps the order.) (FR-STR-012)
40. WHEN the seed is valid and every deity exists, THE SYSTEM SHALL insert every text with its versions, lines and deity links in **one transaction**, with `sort_order` equal to the text's position in `index.json` (0, 1, 2, …), which becomes the library order of AC 1, then log `seeded N sutras, …` with the counts of texts, versions, lines and deity links. (FR-STR-012, BR-STR-02)
41. WHEN `cmd/seedsutras` runs AND the `sutras` table already has at least one row, THE SYSTEM SHALL NOT parse the seed and SHALL NOT write any row, log that the sutras are already seeded and exit with status 0 — so running it again never overwrites an edited text and never brings back a deleted one (as long as the table keeps at least one row). (FR-STR-012)
42. WHEN a seed insert meets a text `id` that already exists, THE SYSTEM SHALL leave that text untouched and not insert its versions, lines or deity links (`ON CONFLICT (id) DO NOTHING`), so two concurrent runs cannot duplicate or overwrite each other. (FR-STR-012)
43. WHEN the database rejects the seed (including when migration `0013` has not been applied), THE SYSTEM SHALL make `cmd/seedsutras` exit non-zero with a fatal log and leave the tables unchanged (transaction rolled back). (FR-STR-012)

## Non-functional

44. WHEN the library holds 153 texts and 64,000 lines, THE SYSTEM SHALL answer a card list page, a detail and a page of 200 lines each in a few milliseconds on loopback (measured 2026-10-03: list 2.5 ms, detail ≤ 2.1 ms, 200 lines 1.8 ms — `tasks.md` Phase 5), far inside the 2 s budget of NFR-STR-01 and NFR-STR-02, and SHALL serve a version of any length only through pages of lines (AC 22, 24), never as a whole. A search request (AC 4) SHALL be measured the same way (`tasks.md` Phase 7). (NFR-STR-01, NFR-STR-02, NFR-STR-03)
    > ⚠️ Not confirmed: the search route (`unaccent` over every title) has not been timed on the current library, and there is no measurement under concurrent load or over a real network — whether the unindexed `unaccent` scan stays inside the budget at 153 texts is expected but unproven; blocks nothing now, and a library several times larger needs a review (NFR-STR-03).

## API contract (confirmed surface for this feature)

Already in `docs/specs/api/openapi.yaml` (tag Sutras), checked against the running handlers.

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/sutras?page=&tradition=&kind=&collection=&purpose=&daily=&max_minutes=&lang=&q=` | public | `200` `{items: [SutraSummary], page, page_size, total, total_pages}` | 400 `INVALID_PAGE` / `INVALID_FILTER`, 405, 500 |
| GET | `/v1/sutras/{id}` | public | `200` `SutraDetail` | 404 `SUTRA_NOT_FOUND`, 405, 500 |
| GET | `/v1/sutras/{id}/lines?translation=&page=&page_size=` | public | `200` `{translation, items: [Line], page, page_size, total, total_pages}` | 400 `INVALID_TRANSLATION` / `INVALID_PAGE` / `INVALID_PAGE_SIZE`, 404 `SUTRA_NOT_FOUND` / `TRANSLATION_NOT_FOUND`, 405, 500 |
| GET | `/v1/deities/{id}/sutras?page=&role=` | public | `200` `{items: [SutraSummary + role], page, page_size, total, total_pages}` | 400 `INVALID_PAGE` / `INVALID_FILTER`, 404 `DEITY_NOT_FOUND`, 405, 500 |

`SutraSummary` (the card): `id` (stable slug), `summary` (`""` when none), `tradition`, `kind`, `collection`, `reference` (`""` when none), `titles` (`{vi, pli?, en?, zh?, sa?, bo?}`, always `vi`), `daily_chant` (bool), `purposes` (`[string]`), `duration_minutes` (int; `null` until AC 34 ships), `audio_url` (string, `""` = no audio), `deities` (`[{deity_id, role, name_vi, thumbnail_url}]`), `translations` (`[{key, lang, variant, translator, license, line_count}]`).

`SutraDetail` = `SutraSummary` + `review_status`, `original_title`, `author`, `han_translator`, `reference_note`, `han_translation_year` (int | `null`), and every element of `translations` gains `source_name`, `source_url`, `note`, `license_url`, `license_note`, `sections` (`[{name, start_n, line_count}]`).

`Line`: `n` (int, from 1, continuous **within one version**), `text`, `section` (string | `null`).

`lang` values in the data today: `vi`, `pli`, `en`, `zh`, `sa`, `bo`. `variant` values: `translation`, `chant` (transliteration to chant), `meaning` (explained meaning), `poem`.

## Out of scope (this feature boundary, not tracked here)

- Creating, editing or deleting texts through an API or the back office — today only by hand-written SQL on the tables; no reseed command (product Appendix A item 15).
- Delta sync (`updated_since`), `version`, offline-first and downloads — separate later specs (product §1.2); each needs an ADR.
- Audio: the field says "no audio" (AC 20); storing and serving recordings is a later spec.
- The quote of the day (BR-STR-10, FR-STR-010): the clients pick it from the 300-verse file; the backend serves nothing for it. If the product moves the file to the backend it needs a contract first (product Appendix A item 3, Appendix B item 4).
- Bookmarks, favourites, reading progress, chant history and counters — module `journal`.
- Full-text search inside the lines (BR-STR-04: names, collection and citation code only).
- Line-by-line alignment between versions: `n` only means something inside one version.
- Vietnamese labels of kinds, traditions and purposes (e.g. `CAU_AN` → "Cầu an") — the clients map them.
- Translating `summary` or titles into other languages — Vietnamese only until content exists (BR-STR-05).
- Licence and provenance clearing of the texts — the code is descriptive and never hides a text; clearing is a product decision before any paid or commercial use (product Appendix A item 2).
- Tracking of the "request a text" action (product Appendix A item 9).

## Pending (not yet implemented)

- **v4.0 duration rule** (AC 34, AC 38): the seeder refuses a text without `durationMinutes`; the contract drops `null` from `duration_minutes`; the data file is checked for all 153 texts (the seed JSON is not in the git checkout, so the check runs where the file lives). `tasks.md` Phase 7.
- **Vector files** (AC 8): run `sutra-browse.json` and `sutra-search-match.json` against the repository. `tasks.md` Phase 7.
- **Search timing** (AC 44). `tasks.md` Phase 7.

## Technical notes

- Source data: **153** texts, **391** versions, **64,009** lines, **151** deity links. By tradition and kind: Nam Tông 53 sutras + 37 verses; Bắc Tông 23 sutras + 18 verses + 12 mantras; Kim Cang Thừa 7 verses + 3 mantras. Versions by language: `vi` 170, `en` 99, `pli` 90, `bo` 17, `zh` 10, `sa` 5. 10 texts (all Kim Cang Thừa) have **no Vietnamese version** (product Appendix A item 5).
- The longest version, `phap-hoa/vi`, has 6,967 lines (35 pages of 200); 51 versions have `sections`. Estimated durations of the 143 texts with a Vietnamese version: ≤5 min 63, ≤15 min 27, ≤30 min 13, ≤60 min 21, >60 min 19 (reference: Tâm Kinh ~3 min, Chú Đại Bi ~6, A Di Đà ~24, Kim Cang ~57). The estimate is an editorial value, not a measurement; known outliers are accepted (TD-007, A-005).
- Deity links by role: `SPEAKER` 84, `MENTIONED` 38, `SUBJECT` 29; `buddha_shakyamuni` appears in 81 texts (5 pages of 20).
- `deity_id` is a foreign key to `deities.id`; `deities` must be seeded first (`cmd/seeddeities`), otherwise `cmd/seedsutras` aborts (AC 39).
- `tam-kinh` has two `vi` versions of variant `meaning`, so `lang` + `variant` cannot select a version; `key` (`vi-meaning`, `vi-meaning-2`) does.
- `id` is a stable slug; some were renamed when the data was merged (e.g. MN 10 is `dai-niem-xu-trung-bo`, DN 22 is `dai-niem-xu-truong-bo`) — clients can cache an `id` safely from this contract on, not before.
- The seed is ~11 MB of JSON embedded into the binary (`assets/embed.go`, +10.25 MB, +17%); it is kept out of git by the ignore rule for JSON files (TD-001), so a fresh checkout cannot run `cmd/seedsutras` or the seed tests that read it. Loading ~64,000 lines in one transaction took about 1.2–1.8 s on a local Postgres.
- Page sizes live in the `sutra:` block of `config.<env>.yaml` (dev: `page_size: 20`, `lines_page_size: 200`, `lines_max_page_size: 500`); all three are required and must be > 0. Whether the list page stays 20 or becomes 12 is open in the product spec (Appendix A item 12); the backend follows the config value.
- Search uses the Postgres `unaccent` extension (migration `0019`); `đ`/`Đ` map to `d`/`D`. The term is a bound parameter and LIKE wildcards are escaped, so nothing from the request is concatenated into SQL.
- Tables `sutras`, `sutra_translations`, `sutra_lines`, `sutra_deities` belong to migration `0013_sutras.sql`; `0019` adds `audio_url`, the `unaccent` extension and the duration backfill; `0020` splits `reference` / `license` and adds `summary`. They must be applied before the seed (`make migrate`).

## Open questions (resolved)

- ~~Is an ADR needed?~~ → **No.** Resolved 2026-10-15 in the product spec. Effect: none on this spec.
- ~~Does the backend give the sections of a version for the table of contents?~~ → **Yes**, `sections` in the detail (AC 15). Resolved 2026-10-15 (anh Đức asked, product spec). Effect: no change — it already existed.
- ~~What does the backend say about audio?~~ → **"None"** (`audio_url: ""`), the clients decide later. Resolved 2026-10-15 (anh Đức: "BE cứ trả không có"). Effect: AC 20 written down; no code change.
- ~~Search~~ → server search by name on all clients (`q`, AC 4). Resolved 2026-10-15. Effect: AC 4–5 written from the shipped code; the 100-character term limit is enforced by the usecase (`maxSutraQueryRunes`).
- ~~Fields that were packed together, and a summary~~ → `reference` split into code + `original_title` + `author` + `han_translator` + `han_translation_year` + `reference_note`; `license` split into a code + `license_url` + `license_note`; Vietnamese `summary` added. Resolved 2026-10-07 (v2.12–v2.13; assumptions A-010, A-011). Effect: AC 10–13, 16, 19.
- ~~How the data is loaded~~ → manual `cmd/seedsutras` into an empty table, one transaction, validate-all first; the service does not seed on start. Resolved 2026-10-03, changed 2026-10-11. Effect: AC 37–43.
- ~~Is the list paged~~ → yes, `page` / `page_size` / `total` / `total_pages`; the lines are paged by line count. Resolved 2026-10-03.
- ~~Missing deities at seed time~~ → abort, write nothing. Resolved 2026-10-03 (AC 39).
- ~~Values of `purposes`, who reviews the labels, the Great Compassion Mantra version, content updates by hand SQL, licences~~ → ten purposes accepted; labels assigned by the author (no reviewer, `review_status` stays `UNREVIEWED`); phatgiao.org.vn 84-verse version kept; updates by SQL only; licences not blocking release. Resolved 2026-10-03 (A-004, A-006, A-003, TD-002, TD-003). The first and last two are still product open items (Appendix A items 1, 2, 15).
- ~~Old `content` spec codes (US-005/006)~~ → replaced by this spec on 2026-10-03; `content` was deleted.
