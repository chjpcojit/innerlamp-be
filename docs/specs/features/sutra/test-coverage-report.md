# Test Coverage Report — sutra

AC coverage audits and E2E wiring findings for this feature, run by `golang-e2e-verification`.
Newest run first. This file is an audit log, not part of the requirements→design→tasks cascade —
it needs no version number and no `changelog.md` entry of its own.

Paths are relative to `services/appService/`. Abbreviations: **U** `usecase/sutra_usecase_test.go`,
**H** `delivery/http/sutra_handler_test.go`, **R** `repository/postgres_sutra_repository_test.go`,
**P** `repository/sutra_seed_test.go`, **C** `cmd/seedsutras/main_test.go`, **E** `e2e/sutra_e2e_test.go`.

## 2026-10-03

Audited against `requirements.md` v2.7 (35 AC). Ran on Go 1.26.5, Postgres 16 (testcontainers for E2E, a
throwaway container for the repository tests).

### AC coverage

| AC | Requirement (short) | Covered by | Status |
|----|---------------------|-----------|--------|
| AC1 | `GET /v1/sutras` is public, paged, ordered `sort_order, id`, `page_size` 20 | U:26 `List_Paging`; H:52 `List_Success`; R:367 `List_PagingAndShape`; E:204 (all 153 ids over 8 pages, no auth header, order equals `index.json`) | ✅ covered |
| AC2 | list items are summaries: no lines, sections, sources, `review_status` | H:52 `List_Success` (`NotContains`); E:217 (exact key set on every item, translations carry no sources) | ✅ covered |
| AC3 | filters `tradition`, `kind`, `collection`, `purpose`, `daily`, `max_minutes`, `lang`, combined with AND, `total` over matches | U:68 `List_PassesParsedFilter`; R:312 `ListAndCount_Filters` (15 cases incl. NULL duration dropped); E:267 (each filter and the 6-way combination vs the data, all pages) | ✅ covered |
| AC4 | bad filter value → `400 INVALID_FILTER` with `details.field` | U:90 `List_InvalidFilter` (13 cases); H:159 `List_Errors`, H:125 (present-but-empty reaches the use case); E:324 (13 requests) | ✅ covered |
| AC5 | page past the end → `items: []`; empty table / no match → `total 0`, `total_pages 0` | U:26; H:116 `List_EmptyPageIsEmptyArray`; E:204 (page 9 empty), E:267 ("match nothing") | ✅ covered |
| AC6 | bad `page` → `400 INVALID_PAGE` | U:59 `List_InvalidPage`; H:159; E:324 | ✅ covered |
| AC7 | deities in the summary with `name_vi`/`thumbnail_url`, ordered by role | R:82 `InsertAllAndFindByID`, R:367; H:52; E:217 | ✅ covered |
| AC8 | detail = summary + `review_status`, translations with `source_*`, `note`, `sections` | H:203 `Get`; R:82; E:354 | ✅ covered |
| AC9 | `sections` `[{name, start_n, line_count}]`, `[]` when none; `start_n` → page | P:222 `SectionsOf` (6 cases); R:82; H:203; E:423 (a real phẩm of Pháp Hoa is found on the page computed from `start_n`) | ✅ covered |
| AC10 | unknown sutra → `404 SUTRA_NOT_FOUND` | U:141 `Get`; H:203; R:82; E:414 | ✅ covered |
| AC11 | several texts of one language told apart by `key`, file order | R:82 (order); P:128 (duplicate key rejected); E:354 (`tam-kinh`: `vi-meaning` and `vi-meaning-2`) | ✅ covered |
| AC12 | `GET …/lines?translation=` shape with `page`, `page_size`, `total`, `total_pages` | H:247 `Lines_Success`; U:158 `Lines_Paging`; E:423 | ✅ covered |
| AC13 | default page 1 and `lines_page_size`; items are `(p−1)s < n ≤ p·s` | U:158; U:198 (no int32 overflow); R:238 `ListLines`; H:269 `Lines_Defaults`; E:423 (first/last n of pages 1 and 35) | ✅ covered |
| AC14 | page past the end → `items: []`, same totals | U:158; R:238; E:423 (page 36) | ✅ covered |
| AC15 | `page_size` above the cap is clamped; below 1 → `INVALID_PAGE_SIZE` | U:158 (clamp), U:210; H:287; E:423 (`page_size=100000` → 500), E:567 | ✅ covered |
| AC16 | bad `page` on lines → `INVALID_PAGE` | U:210; H:287; E:567 | ✅ covered |
| AC17 | missing/empty `translation` → `INVALID_TRANSLATION` | U:210; H:287; E:567 | ✅ covered |
| AC18 | unknown sutra / unknown key → two different 404s, sutra checked first | U:235; R:215 `FindTranslation`; H:287; E:567 | ✅ covered |
| AC19 | `text` as stored, `section` string or null, `n` contiguous 1..total | H:247 (null section); R:238, R:521 (`line_count` equals rows and max `n` on the real library); P:128 (gap/repeat rejected); E:519 (walks a whole text, no gap, no repeat, no re-trim) | ✅ covered |
| AC20 | `GET /v1/deities/{id}/sutras` paged, role in each item, ordered by role | U:256 `ListByDeity`; H:335; R:423 `ByDeity`; E:597 (3 deities × 4 role filters, ids and roles equal the oracle, all pages) | ✅ covered |
| AC21 | `role` filter; bad `role` → `INVALID_FILTER` | U:256; R:423; H:335; E:597, E:636 | ✅ covered |
| AC22 | unknown deity → `404 DEITY_NOT_FOUND` | U:256; H:335; E:636 | ✅ covered |
| AC23 | deity with no sutras → `items: []`, `total 0` | R:423; U:256; E:597 (`buddha_vairocana`, `bodhisattva_tara_white`) | ✅ covered |
| AC24 | the new route does not change the other deity routes | H:399 `DoesNotShadowDeityRoutes` (a test mux); **E:653** on the real `cmd/main.go` wiring | ✅ covered (wiring proven by E2E only) |
| AC25 | arrays are `[]`, never `null` | H:100 `List_NullAndEmptyShapes`; E:217 | ✅ covered |
| AC26 | `duration_minutes` is `null`, never `0`, when unknown; stored value returned unchanged | H:100; R:133; E:217 (null), E:354 (value equals `index.json`) | ✅ covered |
| AC27 | `titles` always holds `vi`; `reference` is `""` when absent | H:100; P:128 (`titles.vi` enforced at seed); E:217 | ✅ covered |
| AC28 | shared error envelope; non-GET → 405 `Allow: GET, HEAD`; 500 does not leak | H:422 (405), H:159 / H:203 / H:287 (500, `NotContains`); **E:669** (4 routes × POST/PUT/DELETE through the real binary), E:414 (404 envelope) | ✅ covered. Gap: the **500 without leaking** is proven at handler level only; no E2E can inject an infrastructure failure on a read route without changing the binary |
| AC29 | seed validated in full before any write; invalid → startup aborts | P:72–294 (27 invalid-sutra cases and 7 invalid-index cases covering every bullet of the AC, plus the shipped data); C:76 (`-dir` with bad data writes nothing) | ✅ covered. Gap: the abort **of the main binary** on invalid data is not E2E — the embedded data is valid and cannot be corrupted without rebuilding; the command and the binary share one use case and one parser |
| AC30 | missing deities → abort, list them, write nothing | U:339 `SeedIfEmpty`; **E:756** (binary exits non-zero, names `buddha_shakyamuni` and `cmd/seeddeities`, `sutras` and `sutra_lines` empty); E:822 (command, exit 1) | ✅ covered |
| AC31 | one transaction, `sort_order` = index position, log with counts | R:521 `ShippedData` (counts, 153/391/64009/151, ~1.2 s); **E:698** (every table count equals `index.json`) | ✅ covered. Gap: the `seeded sutras on first run` **log line** is not asserted by an automated test (seen by hand in the smoke run) |
| AC32 | table not empty → nothing parsed, nothing written | U:339 (`load` never called); **E:715** (an edited title survives, a deleted sutra stays deleted, no row written on restart) | ✅ covered |
| AC33 | `ON CONFLICT (id) DO NOTHING`, children skipped | R:150 `LeavesExistingSutrasAlone` | ✅ covered. Gap: two instances **starting at the same moment** are not tested (only "the row already exists") |
| AC34 | database rejects the seed → startup aborts, rolled back | R:172 `IsAllOrNothing`; **E:777** (`sutra_deities` dropped so the insert fails after the sutra, its translations and lines were written: binary exits non-zero, all three tables empty) | ✅ covered. The literal case "migration 0013 not applied" is stood in for by the dropped table |
| AC35 | `cmd/seedsutras`: same rules, exit 0 when seeded, non-zero on failure | C:23–76; **E:822** (no deities → exit 1; empty → seeds and exits 0 with the counts; seeded → exit 0, no write, edit kept; the service then starts and serves) | ✅ covered |

Beyond the AC: `config.SutraConfig.Validate` (`config/config_test.go`, 7 + 2 tests) and its wiring — **E:797** proves the
binary refuses to start (exit non-zero, before touching the database) for `page_size: 0`, `lines_page_size: 0`,
`lines_max_page_size: 0` and a default above the cap.

### E2E findings

**Wiring bugs in `cmd/main.go` caught by the new tests: none** — the four routes, the seed call and the `Validate` call were
already correct when the tests were written.

What the E2E work did establish:

- **Every E2E test is able to fail.** Six deliberate mutations of the real code were each caught, then reverted (`diff` against the saved
  copies was empty, `go build ./...` clean): (1) not mounting `GET /v1/deities/{id}/sutras` → `AC20/AC21/AC23` and `AC21/AC22` red;
  (2) the seed ignoring "table already has rows" → `SecondStartLeavesTheDatabaseAlone` and the command's "already seeded" case red;
  (3) dropping the `Validate()` call → all four `InvalidValuesRefuseToStart` cases red; (4) skipping the "deities must exist" check →
  `RefusesToStartWithoutDeities` and the command's "without the deities" case red; (5) `n >= from` instead of `n > from` in the lines range →
  `AC19 n is contiguous` red; (6) dropping the role rank from the by-deity `ORDER BY` → `AC20/AC21/AC23` red.
- **A regression the feature caused in the existing E2E suite, found and fixed earlier in this work (Phase 4)**: the three register/login E2E tests
  failed with "did not become ready" because the service now refuses its first start when `deities` is not seeded and the harness never seeded it.
  Fixed in `e2e/harness.go` (`seedDeities`, which mirrors `cmd/seeddeities` in a real deployment). The same change makes a binary that exits at
  startup report its stderr instead of a bare timeout. After it the whole suite passes (the existing register/login tests plus the 7 new sutra test functions).

### Follow-ups

- **No gap routes to another layer.** The four gaps above are limits of what a black-box test can reach, not missing unit tests:
  500-without-leak (covered at handler level), invalid embedded data aborting the main binary (covered through the parser and `cmd/seedsutras -dir`),
  the first-run log line, and two instances seeding simultaneously (the `ON CONFLICT` behaviour itself is tested).
- Run time: the 7 new E2E test functions take about 30 s together (each first start seeds the library, ~1.5–2 s).
- `go test ./services/appService/e2e/` needs Docker; under `-short` or without Docker the tests skip, as the existing ones do.
