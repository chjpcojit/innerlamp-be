# Requirements — sutra

Version: 2.13
Last updated: 2026-10-07
Status: implemented (2026-10-03) — 4 route, nạp lần đầu khi khởi động và `cmd/seedsutras`; cả 35 tiêu chí đã có test chứng minh, gồm E2E chạy binary thật + Postgres (xem `test-coverage-report.md`); không còn việc nào mở về `durationMinutes` (người dùng quyết định giữ nguyên giá trị đã sinh, 2026-10-03). Dữ liệu nguồn đã có ở `assets/data/sutras/` (153 văn bản) đã có đủ bốn trường `key`, `dailyChant`, `purposes`, `durationMinutes` (2026-10-03; `purposes` chưa được người dùng soát, `durationMinutes` được giữ nguyên theo quyết định của người dùng — xem `tasks.md` Phase 0). v2.12 (2026-10-07) thêm tách trường (`reference`, `license`) và `summary` — đã implement và kiểm chứng (xem `tasks.md` Phase 6); dữ liệu mới đã có trên DB dev (kiểm 2026-10-08).
Module: `content`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này được viết **prospectively** (trước khi có code) và đã được implement ở Phase 1–4 ngày 2026-10-03; các tiêu chí dưới đây mô tả hành vi đã chạy thật. Dữ liệu nguồn là bộ JSON ở `assets/data/sutras/` (xem `README.md` trong đó).

Đây là danh mục **chỉ đọc**, công khai, cho kinh / kệ / chú, cùng hình dạng với `temple`, `deity`, `article`: danh sách **phân trang theo `page`**, trang chi tiết, và dữ liệu **tự nạp một lần khi service khởi động lần đầu** từ file JSON nhúng trong binary. Dữ liệu được thu thập từ web nên JSON chỉ là **nguồn nạp một lần**: sau lần nạp đầu, mọi API đọc từ Postgres và JSON không được đọc lại. Khác ở ba điểm: (1) mỗi văn bản có **nhiều bản dịch** (ngôn ngữ, loại bản), (2) nội dung là **hàng chục nghìn dòng** nên được phân trang thêm **theo số dòng mỗi trang** (kinh dài nhất 6.967 dòng), (3) mỗi văn bản **liên kết tới Phật / Bồ tát** (`deities`) và ngược lại.

Mỗi văn bản mang các thuộc tính để client lọc và hiển thị: **tông phái** (Nam Tông / Bắc Tông / Kim Cang Thừa), **loại** (kinh / kệ / chú), **kinh nhật tụng hay không**, **mục đích** (cầu an, cầu siêu, trì chú, …) và **thời lượng ước tính** (vd. 15 phút, 30 phút — con số do mình ước tính và ghi, không phải số đo).

> Spec này **thay thế hoàn toàn** phần kinh điển và thần chú của spec `content` cũ. Spec `content` cũ (`docs/specs/features/content/requirements.md`) đã bị xoá ngày 2026-10-03 vì kinh và thần chú (spec này), Phật & Bồ Tát (spec `deity`) đều đã có feature riêng; xem lịch sử git nếu cần. `US-SUT-01..06` là mã chính thức.

---

## US-SUT-01: Duyệt và lọc danh mục kinh theo trang

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **xem danh mục kinh theo từng trang và lọc theo tông phái, loại, mục đích, kinh nhật tụng hay thời lượng**,
> Để **chọn đúng bài kinh phù hợp với lúc mình định đọc hoặc tụng**.

## Acceptance Criteria (EARS notation)

1. WHEN a `GET /v1/sutras` request arrives without `page` (or with an empty `page=`), THE SYSTEM SHALL serve page 1, and SHALL return `200 OK` with `data` holding `items` (summary shape), `page`, `page_size` (`sutra.page_size`, dev: 20), `total` (số văn bản khớp bộ lọc) and `total_pages` = `ceil(total / page_size)`, ordered by `sort_order` ASC then `id` ASC. (Không cần `Authorization`.)
2. WHEN sutras are returned by the list, THE SYSTEM SHALL return each one with `id`, `tradition`, `kind`, `collection`, `reference`, `titles`, `daily_chant`, `purposes`, `duration_minutes`, `deities` and `translations`, and SHALL NOT include any line text, `sections`, `source_url`, `source_name`, `note` or `review_status`.
3. WHEN the request carries any of `tradition`, `kind`, `collection`, `purpose`, `daily`, `max_minutes`, `lang`, THE SYSTEM SHALL return only the sutras matching **all** supplied filters (AND), keep the same ordering, and compute `total` / `total_pages` over the matching sutras only:
   - `tradition` ∈ `THERAVADA` | `MAHAYANA` | `VAJRAYANA`
   - `kind` ∈ `SUTRA` | `VERSE` | `DHARANI`
   - `collection` — khớp chính xác với `collection`
   - `purpose` — sutra có giá trị này trong `purposes`
   - `daily` ∈ `true` | `false` — khớp `daily_chant`
   - `max_minutes` — số nguyên ≥ 1; chỉ sutra có `duration_minutes` ≤ giá trị này (sutra `duration_minutes = null` bị loại)
   - `lang` — sutra có ít nhất một bản dịch ngôn ngữ đó (vd. `vi`)
4. WHEN a filter value is not allowed (an unknown `tradition`/`kind`/`purpose`, a `daily` that is not `true`/`false`, a `max_minutes` that is not an integer ≥ 1, or an empty value), THE SYSTEM SHALL return `400 INVALID_FILTER` with `details.field` set to the offending parameter name.
5. WHEN `page` is greater than `total_pages`, THE SYSTEM SHALL return `200 OK` with `items: []` and the same `page_size`/`total`/`total_pages` — không phải lỗi; WHEN the table is empty or no sutra matches, THE SYSTEM SHALL return `items: []` (không phải `null`), `total: 0` and `total_pages: 0`. Client dừng khi `page >= total_pages` hoặc `items` rỗng.
6. WHEN `page` is not a positive integer (`0`, negative, non-numeric, an integer that overflows, or larger than 2^31−1), THE SYSTEM SHALL return `400 INVALID_PAGE` with `details.field = "page"`.
7. WHEN a sutra is linked to deities, THE SYSTEM SHALL return them in `deities` as `{deity_id, role, name_vi, thumbnail_url}`, ordered by role (`SUBJECT`, `SPEAKER`, `MENTIONED`) then `deity_id`, so the client can show the related Phật / Bồ tát without a second request.

## US-SUT-02: Xem chi tiết một bài kinh

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **mở một bài kinh để xem đủ thông tin trước khi đọc**,
> Để **biết có những bản dịch nào, dịch giả và nguồn nào, bài kinh có những phẩm nào và nhảy thẳng tới phẩm mình cần**.

## Acceptance Criteria (EARS notation)

8. WHEN a `GET /v1/sutras/{id}` request arrives with an existing `id`, THE SYSTEM SHALL return `200 OK` with the summary fields plus `review_status`, and each element of `translations` extended with `source_name`, `source_url`, `note` and `sections`.
9. WHEN a translation has sections, THE SYSTEM SHALL return `sections` as `[{name, start_n, line_count}]` in file order, where `start_n` is the `n` of the section's first line, so the client can compute the page of that section as `floor((start_n − 1) / page_size) + 1`; WHEN a translation has none, THE SYSTEM SHALL return `[]`.
10. WHEN `{id}` matches no sutra, THE SYSTEM SHALL return `404 SUTRA_NOT_FOUND`.
11. WHEN a sutra has several translations with the same `lang`, THE SYSTEM SHALL distinguish them by `key` (unique within the sutra) and by `variant` (`translation` | `chant` | `meaning` | `poem`), and SHALL return them ordered as in the source file.

36. WHEN a `GET /v1/sutras/{id}` request arrives with an existing `id`, THE SYSTEM SHALL return, besides the fields of AC8, these flat fields (each its own JSON field, never packed into another): `original_title`, `author`, `han_translator`, `reference_note` (strings, `""` when the source has none), `han_translation_year` (integer year CE | `null`) and `summary` (string, Vietnamese, `""` when none was written). THE SYSTEM SHALL NOT return the five fields other than `summary` in the list (`GET /v1/sutras`) or the by-deity list, which keep the card shape plus `summary` (AC41).
37. THE SYSTEM SHALL keep in `reference` only the citation code of the text (e.g. `Taishō T0450`, `MN 86`, `Dhp 1-20`, `Toh 438, …`) or `""` when the text has none; the title in Hán / Sanskrit, the author, the Hán translator, the year and any remark SHALL be stored and returned in `original_title`, `author`, `han_translator`, `han_translation_year` and `reference_note`.
38. THE SYSTEM SHALL return, for every translation in the list, the by-deity list and the detail, `license` as one code from the closed set `CC0-1.0` | `CC-BY-NC-4.0` | `PUBLIC_DOMAIN` | `COPY_ALLOWED` | `UNKNOWN`; and, in the detail only, `license_url` (the URL the source gives, `""` when none) and `license_note` (the source's own statement, `""` when none), so the label and the explanation are separate fields.
39. WHEN a translation's `license` is `UNKNOWN`, THE SYSTEM SHALL still return the translation unchanged (AC8, AC11): the code is descriptive, it never hides a text (đã chốt 2026-10-03: giấy phép tạm không chặn phát hành).
40. WHEN the seed JSON is parsed (AC29), THE SYSTEM SHALL also treat the seed as invalid when a translation's `license` is outside the closed set of AC38, or when `hanTranslationYear` is present but is neither `null` nor an integer ≥ 1; the fields `originalTitle`, `author`, `hanTranslator`, `referenceNote`, `summary`, `licenseUrl` and `licenseNote` are optional and default to `""`.
41. THE SYSTEM SHALL return `summary` (string, Vietnamese, `""` when none was written) on every card of the list (`GET /v1/sutras`) and of the by-deity list (`GET /v1/deities/{id}/sutras`), the same value the detail returns, so a client can show the description without a second request per card.

## US-SUT-03: Đọc nội dung kinh theo trang dòng

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **đọc một bài kinh dài từng trang theo số dòng và cuộn sang trang kế**,
> Để **không phải chờ tải cả bộ kinh hàng nghìn dòng và có thể mở thẳng tới trang đang đọc**.

## Acceptance Criteria (EARS notation)

12. WHEN a `GET /v1/sutras/{id}/lines?translation=<key>` request arrives for an existing sutra and an existing translation `key`, THE SYSTEM SHALL return `200 OK` with `data` holding `translation` (the `key`), `items` (`[{n, text, section}]` ordered by `n` ASC), `page`, `page_size`, `total` (the translation's `line_count`) and `total_pages` = `ceil(total / page_size)`.
13. WHEN `page` is absent (or empty), THE SYSTEM SHALL serve page 1; WHEN `page_size` is absent, THE SYSTEM SHALL use `sutra.lines_page_size` (dev: 200); THE SYSTEM SHALL return as `items` exactly the lines whose `n` is in `((page − 1) × page_size, page × page_size]`.
14. WHEN `page` is greater than `total_pages`, THE SYSTEM SHALL return `200 OK` with `items: []` and the same `page_size`/`total`/`total_pages` — không phải lỗi. Client dừng khi `page >= total_pages` hoặc `items` rỗng.
15. WHEN `page_size` is greater than `sutra.lines_max_page_size` (dev: 500), THE SYSTEM SHALL serve `sutra.lines_max_page_size` lines per page instead of failing, and SHALL echo the value actually used in `page_size`; WHEN `page_size` is not an integer or is less than 1, THE SYSTEM SHALL return `400 INVALID_PAGE_SIZE` with `details.field = "page_size"`.
16. WHEN `page` is not a positive integer (`0`, negative, non-numeric, an integer that overflows, or larger than 2^31−1), THE SYSTEM SHALL return `400 INVALID_PAGE` with `details.field = "page"`.
17. WHEN the `translation` query parameter is absent or empty, THE SYSTEM SHALL return `400 INVALID_TRANSLATION` with `details.field = "translation"`.
18. WHEN `{id}` matches no sutra, THE SYSTEM SHALL return `404 SUTRA_NOT_FOUND`; WHEN the sutra exists but has no translation with that `key`, THE SYSTEM SHALL return `404 TRANSLATION_NOT_FOUND`. (Sutra được kiểm tra trước.)
19. WHEN a line is returned, THE SYSTEM SHALL return its `text` exactly as stored (không cắt, không sửa chính tả), and `section` as the section name string or JSON `null` when the line belongs to none; and `n` SHALL run 1..`total` without gaps within a translation, so line `n` is always on page `floor((n − 1) / page_size) + 1`.

## US-SUT-04: Từ Phật / Bồ tát xem các kinh liên quan

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **từ trang một vị Phật hay Bồ tát xem các bài kinh nói về vị đó hoặc do vị đó thuyết**,
> Để **đọc đúng những bài kinh gắn với vị mình đang tìm hiểu**.

## Acceptance Criteria (EARS notation)

20. WHEN a `GET /v1/deities/{id}/sutras` request arrives with an existing deity `id`, THE SYSTEM SHALL return `200 OK` with `data` holding `items` (the linked sutras in the **summary** shape, each with an extra `role` field = the deity's role in that sutra), `page`, `page_size` (`sutra.page_size`), `total` and `total_pages`, ordered by role (`SUBJECT`, `SPEAKER`, `MENTIONED`) then `sort_order` ASC then `id` ASC. `page` is handled as in US-SUT-01 (absent → 1; greater than `total_pages` → `items: []`; invalid → `400 INVALID_PAGE`).
21. WHEN the request carries `role`, THE SYSTEM SHALL return only the sutras where the deity has that role (and compute `total` over them); WHEN `role` is not `SPEAKER` | `SUBJECT` | `MENTIONED` (or is empty), THE SYSTEM SHALL return `400 INVALID_FILTER` with `details.field = "role"`.
22. WHEN `{id}` matches no deity, THE SYSTEM SHALL return `404 DEITY_NOT_FOUND` (cùng mã với `GET /v1/deities/{id}`).
23. WHEN the deity exists but is linked to no sutra (or none with the requested `role`), THE SYSTEM SHALL return `200 OK` with `items: []`, `total: 0` and `total_pages: 0`.
24. WHEN the path `/v1/deities/{id}/sutras` is requested, THE SYSTEM SHALL route it to this feature and SHALL NOT change the behavior of `/v1/deities/top`, `/v1/deities` or `/v1/deities/{id}`.

## US-SUT-05: Hình dạng dữ liệu nhất quán

## User Story

> Là một **lập trình viên client**,
> Tôi muốn **các field thiếu dữ liệu luôn có dạng xác định**,
> Để **decode một lần mà không phải đoán `null` hay chuỗi rỗng**.

## Acceptance Criteria (EARS notation)

25. WHEN a sutra has no purposes, no deities, no translations, or a translation has no sections, THE SYSTEM SHALL return `[]` for `purposes` / `deities` / `translations` / `sections`, never `null`.
26. WHEN a sutra has no authored duration estimate (vd. a text with no Vietnamese version), THE SYSTEM SHALL return `duration_minutes` as JSON `null`, not `0`; WHEN it has one, THE SYSTEM SHALL return it unchanged as an integer number of minutes (không làm tròn hay tính lại).
27. WHEN a sutra is returned, THE SYSTEM SHALL return `titles` as an object mapping language code to title that always contains `vi`, and SHALL return `reference` as `""` when the source has none.
28. WHEN any sutra route fails, THE SYSTEM SHALL answer in the shared envelope `{"success": false, "error": {code, message, type}, "details"?}`; a non-`GET` request returns `405 METHOD_NOT_ALLOWED` (kèm header `Allow: GET, HEAD`) and an infrastructure failure returns `500 INTERNAL_ERROR` without leaking the cause.

## US-SUT-06: Nạp dữ liệu lần đầu (khi khởi động hoặc bằng lệnh)

## User Story

> Là một **người triển khai hệ thống**,
> Tôi muốn **service tự có sẵn bộ kinh ngay lần chạy đầu, và có một lệnh nạp riêng khi cần chạy tay**,
> Để **môi trường mới (kể cả container không có mã nguồn) chạy được ngay và các lần khởi động sau không đụng vào dữ liệu đã có**.

## Acceptance Criteria (EARS notation)

29. WHEN appService starts AND the `sutras` table has no row, THE SYSTEM SHALL parse the JSON embedded in the binary (`assets/data/sutras/index.json` and every `<id>.json` it lists) and validate **all** of it before any write, and THE SYSTEM SHALL treat the seed as invalid, abort startup with a fatal log and write nothing, when any of these holds:
    - a file listed in `index.json` is missing, or an `id` is empty, duplicated, or different from the file name
    - `tradition`, `kind`, `variant`, a deity `role` or any `purposes` value is outside the allowed sets; `titles.vi` is empty
    - `dailyChant` is not a boolean; `durationMinutes` is neither `null` nor an integer ≥ 1
    - a translation has no `key`, or two translations of one sutra share a `key`, or its `n` does not run 1..N without gaps
    - one sutra lists the same `deity_id` twice
30. WHEN the seed is valid, THE SYSTEM SHALL check that every referenced `deity_id` exists in the `deities` table, and WHEN some do not, SHALL abort startup with a fatal log listing the missing ids and write nothing. (`cmd/seeddeities` phải chạy trước lần khởi động đầu tiên.)
31. WHEN the seed is valid and every deity exists, THE SYSTEM SHALL insert every sutra with its translations, lines and deity links in **one transaction**, with `sort_order` equal to the sutra's position in `index.json` (0, 1, 2, …), then log `seeded sutras on first run` with the counts of sutras, translations, lines and deity links.
32. WHEN appService starts AND the `sutras` table already has at least one row, THE SYSTEM SHALL NOT parse the seed and SHALL NOT write any row — so a restart never overwrites an edited sutra and never brings back a deleted one (miễn là bảng còn ít nhất một dòng).
33. WHEN a seed insert meets a sutra `id` that already exists, THE SYSTEM SHALL leave that sutra untouched and SHALL NOT insert its translations, lines or deity links (`ON CONFLICT (id) DO NOTHING`), so two instances starting at the same time cannot duplicate or overwrite each other.
34. WHEN the database rejects the seed (kể cả khi migration `0013` chưa được apply), THE SYSTEM SHALL abort startup with a fatal log and SHALL leave the tables unchanged (transaction rolled back).
35. WHEN `cmd/seedsutras` runs (flag `-dir`; mặc định dùng dữ liệu nhúng trong binary), THE SYSTEM SHALL apply exactly the same checks and the same only-when-empty rule as at startup (AC29–AC34): WHEN the `sutras` table already has at least one row, it SHALL write nothing, log that the sutras are already seeded, and exit with status 0; WHEN seeding succeeds, it SHALL log the counts and exit with status 0; WHEN seeding fails, it SHALL write nothing and exit with a non-zero status.

## API contract (confirmed surface for this feature)

Đã có trong `docs/specs/api/openapi.yaml` (tag Sutras) và `web/docs/public/openapi.yaml`, đối chiếu với handler đã chạy thật.

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/sutras?page=&tradition=&kind=&collection=&purpose=&daily=&max_minutes=&lang=` | public | `200` `{items: [SutraSummary], page, page_size, total, total_pages}` | 400 `INVALID_PAGE` / `INVALID_FILTER`, 405, 500 |
| GET | `/v1/sutras/{id}` | public | `200` `SutraDetail` | 404 `SUTRA_NOT_FOUND`, 405, 500 |
| GET | `/v1/sutras/{id}/lines?translation=&page=&page_size=` | public | `200` `{translation, items: [Line], page, page_size, total, total_pages}` | 400 `INVALID_TRANSLATION` / `INVALID_PAGE` / `INVALID_PAGE_SIZE`, 404 `SUTRA_NOT_FOUND` / `TRANSLATION_NOT_FOUND`, 405, 500 |
| GET | `/v1/deities/{id}/sutras?page=&role=` | public | `200` `{items: [SutraSummary + role], page, page_size, total, total_pages}` | 400 `INVALID_PAGE` / `INVALID_FILTER`, 404 `DEITY_NOT_FOUND`, 405, 500 |

`SutraSummary`: `id` (slug, string), `summary` (string, `""` nếu chưa có; v2.13), `tradition` (`THERAVADA` | `MAHAYANA` | `VAJRAYANA`), `kind` (`SUTRA` | `VERSE` | `DHARANI`), `collection`, `reference` (`""` nếu không có), `titles` (`{vi, pli?, en?, zh?, sa?, bo?}`), `daily_chant` (bool), `purposes` (`[string]`), `duration_minutes` (int | `null`), `deities` (`[{deity_id, role, name_vi, thumbnail_url}]`), `translations` (`[{key, lang, variant, translator, license, line_count}]`, `license` là mã trong tập đóng `CC0-1.0` | `CC-BY-NC-4.0` | `PUBLIC_DOMAIN` | `COPY_ALLOWED` | `UNKNOWN`).

`SutraDetail` = `SutraSummary` + `review_status` (`UNREVIEWED` | …), + `original_title`, `author`, `han_translator`, `reference_note`, `han_translation_year` (int | `null`) và mỗi phần tử `translations` thêm `source_name`, `source_url`, `note`, `license_url`, `license_note`, `sections` (`[{name, start_n, line_count}]`). `reference` chỉ còn mã trích dẫn (AC37).

`Line`: `n` (int, bắt đầu từ 1, liên tục **trong mỗi bản dịch**), `text`, `section` (string | `null`).

Giá trị `lang` trong dữ liệu hiện có: `vi`, `pli`, `en`, `zh`, `sa`, `bo`. Giá trị `variant`: `translation`, `chant` (âm Hán-Việt/phiên âm để tụng), `meaning` (dịch nghĩa), `poem`.

## Out of scope (this feature boundary, not tracked here)

- Tạo/sửa/xoá kinh qua API hoặc backoffice — hiện chỉ có SQL trực tiếp trên các bảng.
- Cờ `-prune` và việc ghi đè / reseed từ script: không có; `cmd/seedsutras` cũng chỉ nạp khi bảng `sutras` rỗng.
- Đồng bộ delta (`updated_since`), `audio_url`, `version` — từng có trong spec `content` cũ (đã xoá 2026-10-03, xem lịch sử git), chưa làm.
- Bookmark, kinh yêu thích, đánh dấu đã đọc, lịch sử tụng, đếm số biến tụng — thuộc module `journal`.
- Tìm kiếm toàn văn trong nội dung kinh.
- Gióng hàng từng dòng giữa các ngôn ngữ: `n` chỉ có nghĩa trong một bản dịch, các bản dịch không khớp 1-1.
- Nhãn hiển thị tiếng Việt của `purposes` (vd. `CAU_AN` → "Cầu an") — client tự ánh xạ.
- Đăng nhập: bốn route công khai như `temple`, `deity`, `article`.
- Cập nhật nội dung kinh sau lần seed đầu bằng lệnh reseed hoặc backoffice CRUD — đã chốt hoãn; hiện chỉ đổi được bằng SQL tay (xem "Open questions (resolved)").
- Xử lý giấy phép/bản quyền các văn bản — đã chốt tạm không quan tâm (xem "Open questions (resolved)"); mã `license` ở AC38 chỉ để mô tả, không chặn hay ẩn văn bản.
- Bản dịch `summary` sang ngôn ngữ khác (chỉ tiếng Việt) và đưa các trường tách khác ngoài `summary` vào danh sách (chỉ chi tiết).

## Pending (not yet implemented)

- Tách trường (`reference` → `reference` + `original_title` + `author` + `han_translator` + `han_translation_year` + `reference_note`; `license` → `license` + `license_url` + `license_note`) và thêm `summary` (AC36–AC40): đã implement 2026-10-07 (dữ liệu JSON, migration `0020`, domain, seed parser, repository, handler, OpenAPI, ERD, test, E2E); việc áp lên DB dev đã xong (2026-10-08).
> Đã đóng 2026-10-08: DB dev đã mang dữ liệu mới (153/153 văn bản khớp JSON, kiểm bằng truy vấn trực tiếp và `GET /v1/sutras/{id}`). Môi trường có dữ liệu thật vẫn cần script backfill riêng — chưa có môi trường như vậy.

- ~~Soát và chỉnh tay `durationMinutes`~~ — đã đóng 2026-10-03: người dùng quyết định giữ nguyên giá trị đã sinh (`tasks.md` Phase 0); độ lệch đã biết (Dược Sư 80 phút, các bộ 14–15 giờ) được chấp nhận, ghi ở TD-007.


## Ghi chú kỹ thuật liên quan

- Dữ liệu nguồn: **153** văn bản, **391** bản dịch, **64.009** dòng, **151** liên kết Phật/Bồ tát. Theo tông phái và loại: Nam Tông 53 kinh + 37 kệ; Bắc Tông 23 kinh + 18 kệ + 12 chú; Kim Cang Thừa 7 kệ + 3 chú. Ngôn ngữ: `vi` 170 bản, `en` 99, `pli` 90, `bo` 17, `zh` 10, `sa` 5. 10 văn bản (toàn Kim Cang Thừa) **không có tiếng Việt**.
- Bản dịch dài nhất: `phap-hoa/vi` 6.967 dòng (35 trang với 200 dòng/trang); 51 bản dịch có `sections`. Thời lượng đã ghi vào dữ liệu (2026-10-03; gợi ý `ceil(âm tiết / 80)` rồi làm tròn cho dễ đọc, chưa chỉnh tay) của 143 văn bản có tiếng Việt: ≤5 phút 63, ≤15 phút 27, ≤30 phút 13, ≤60 phút 21, >60 phút 19 — tham chiếu: Tâm Kinh ~3 phút, Chú Đại Bi ~6 phút, A Di Đà ~24 phút, Kim Cang ~57 phút.
- Liên kết Phật/Bồ tát theo vai: `SPEAKER` 84, `MENTIONED` 38, `SUBJECT` 29; `buddha_shakyamuni` đứng trong 81 văn bản (9 trang với `sutra.page_size` = 10, 5 trang với 20).
- `deity_id` là khoá ngoại tới `deities.id` (`phat_botat.json`, 18 vị); `deities` phải được seed trước (`cmd/seeddeities`) — nếu không, lần khởi động đầu abort (đã chốt — xem "Open questions (resolved)").
- `tam-kinh` có hai bản `vi` cùng `variant = meaning` (Bình Anson và Trần Trọng Kim) nên `lang` + `variant` không đủ để chọn bản — dữ liệu cần thêm `key` duy nhất theo từng văn bản (`vi-meaning`, `vi-meaning-2`, …).
- `id` là slug ổn định; một số id đã được đổi trong đợt gộp dữ liệu (vd. MN 10 là `dai-niem-xu-trung-bo`, DN 22 là `dai-niem-xu-truong-bo`) — client cache `id` an toàn sau khi hợp đồng này ra, không trước.
- Dữ liệu seed ~11 MB JSON được `go:embed` vào binary (`assets/embed.go`), nên binary lớn thêm cỡ đó; seed lần đầu ghi ~64.000 dòng trong một transaction nên khởi động lần đầu chậm hơn các lần sau — đo ngày 2026-10-03 trên Postgres cục bộ: từ lúc chạy process tới `200` đầu tiên mất median 1,82 s ở lần chạy đầu (1,73–2,93 s, 5 lần) và 0,13 s ở các lần sau; binary 70,91 MB khi nhúng thư viện so với 60,66 MB khi chỉ nhúng `index.json` (+10,25 MB, +17%).
- Bảng `sutras`, `sutra_translations`, `sutra_lines`, `sutra_deities` thuộc migration `0013_sutras.sql` và phải tồn tại trước khi service chạy (`make migrate`), nếu không bước seed lỗi và service không khởi động.
- Kích thước trang nằm ở block `sutra:` của `config.<env>.yaml` (dev: `page_size: 20`, `lines_page_size: 200`, `lines_max_page_size: 500`); cả ba bắt buộc và phải > 0.

## Open questions (resolved during implementation)

- ~~Tách các trường đang gộp chung và thêm tóm tắt~~ → **tách `reference` thành `reference` (mã trích dẫn) + `original_title` + `author` + `han_translator` + `han_translation_year` + `reference_note`; tách `license` thành mã trong tập đóng + `license_url` + `license_note`; thêm `summary` tiếng Việt**; sáu trường mới chỉ ở chi tiết (danh sách giữ dạng thẻ, tránh phình ~6 KB mỗi trang), `license` (mã) ở cả danh sách. `note` của bản dịch giữ nguyên (một ghi chú biên tập, không gộp nhiều thứ). Resolved 2026-10-07 (người dùng: "những phần gộp em tách ra dùm anh, em tự làm phần summary"; tên trường, tập mã `license` và phạm vi chi tiết-only do em chọn — xem `docs/assumptions/sutra/A-010`; `summary` do em soạn, chưa ai duyệt — xem A-011). Ảnh hưởng: AC36–AC40 mới; `design.md` đổi mô hình dữ liệu, parser và flow Chi tiết; `tasks.md` thêm Phase 6; `flow.md` đổi câu SELECT của Chi tiết (Step 1–2).

- ~~Cách nạp dữ liệu: lệnh seed thủ công hay tự nạp lúc khởi động~~ → **tự nạp một lần khi khởi động lần đầu**, nhúng JSON vào binary (theo cách của `article`); bỏ cờ `-prune` (lệnh `cmd/seedsutras` được thêm lại ở v2.3 làm cửa vào thứ hai, xem mục resolved bên dưới); nạp trong **một** transaction nên lỗi thì không để lại dữ liệu nửa chừng. Resolved 2026-10-03. Ảnh hưởng: `design.md` đổi flow nạp dữ liệu và bỏ `SutraSeeder.ReplaceSutra`/`PruneExcept`; `tasks.md` bỏ các việc của `cmd/seedsutras`, thêm nhúng dữ liệu và `SeedIfEmpty`. Làm rõ ở v2.2 và chốt ở v2.3: thêm script `cmd/seedsutras` làm cửa vào thứ hai (xem mục resolved bên dưới).
- ~~Danh sách kinh có phân trang không~~ → **có**, bằng `page` / `page_size` / `total` / `total_pages` như `article` và `deity`; route tra ngược theo Phật/Bồ tát cũng phân trang. Resolved 2026-10-03. Ảnh hưởng: `design.md` thêm bước đếm vào hai flow danh sách; `tasks.md` thêm `SutraConfig.PageSize`.
- ~~Kiểu phân trang nội dung dòng~~ → **theo trang với số dòng mỗi trang**: `page` + `page_size` (số dòng), thay cho con trỏ `after`/`limit` của v1.0. Resolved 2026-10-03 (cách hiểu của spec: `page_size` là tham số tuỳ chọn, mặc định `sutra.lines_page_size`, bị kẹp ở `sutra.lines_max_page_size`). Ảnh hưởng: `design.md` đổi flow đọc dòng sang truy vấn khoảng `n`; `tasks.md` đổi các mục usecase/handler tương ứng.
- ~~Quy trình cập nhật nội dung sau lần seed đầu (SQL tay, lệnh reseed hay backoffice CRUD)~~ → **dùng SQL tay như `article` hiện nay**; lệnh reseed và backoffice CRUD hoãn, không thuộc feature này. Resolved 2026-10-03 (người dùng "chốt" điểm này mà chưa nêu phương án cụ thể; spec ghi theo mặc định của `article` — sửa lại nếu muốn khác). Ảnh hưởng: không đổi `design.md` ngoài việc đổi ghi chú Known gaps từ "chưa chốt" thành "đã chốt"; `tasks.md` không thêm việc.
- ~~Hành vi khi lần khởi động đầu mà `deities` chưa được seed~~ → **abort startup** (fatal, nêu các id thiếu, không ghi gì), thứ tự triển khai là migration → `cmd/seeddeities` → service. Resolved 2026-10-03. Ảnh hưởng: `design.md` Step 3 và `flow.md` bỏ marker, ghi là đã chốt; AC30 giữ nguyên.
- ~~Bảng giá trị `purposes`~~ → **chấp nhận danh sách đề xuất** `CAU_AN`, `CAU_SIEU`, `TRI_CHU`, `SAM_HOI`, `HOI_HUONG`, `PHAT_NGUYEN`, `BAO_HIEU`, `THIEN_DINH`, `HOC_GIAO_LY`, `QUY_Y_THO_GIOI`, và giá trị khởi tạo của `daily_chant`/`purposes` theo bảng trong `design.md`. Resolved 2026-10-03. Ảnh hưởng: `tasks.md` Phase 0 không còn bị chặn bởi việc chốt danh sách; việc duyệt nhãn đã được chốt ở v2.3: không có người duyệt, em gán (xem mục resolved bên dưới).
- ~~Cách ước tính `duration_minutes`~~ → **ước tính biên tập, giá trị do mình ghi**: nghĩ bài kinh cần 30 phút thì ghi 30. Công thức `ceil(số âm tiết / 80)` (bản `vi` ưu tiên `chant` > `translation` > `meaning` > `poem`, tối thiểu 1) chỉ để **gợi ý giá trị ban đầu**, sau đó làm tròn hoặc chỉnh tay; không cam kết độ chính xác; `null` khi chưa có ước tính. Resolved 2026-10-03, làm rõ lại ở v2.2. Ảnh hưởng: `design.md` bảng "Dữ liệu cần bổ sung" đổi từ "tính bằng script" sang "do mình ghi, script chỉ gợi ý"; `tasks.md` Phase 0 đổi theo; AC26 sửa lời, cách validate của AC29 không đổi.
- ~~Quyền dùng nội dung (giấy phép, ẩn văn bản chưa rõ giấy phép / chưa duyệt, ghi nguồn)~~ → **tạm không quan tâm** vì ứng dụng miễn phí; API trả mọi văn bản, các trường `license`, `source_name`, `source_url` vẫn được lưu và trả trong chi tiết. Resolved 2026-10-03. Ảnh hưởng: không còn chặn phát hành; `design.md` Known gaps ghi lại là đã hoãn có chủ đích; không đổi `tasks.md`.
- ~~Mã user story và hợp đồng cũ của `content` (US-005/US-006: `/v1/content/sutras`, `updated_since`, `slug`, `audio_url`, `version`)~~ → **thay US-005 đến US-008 của `content` bằng spec này** (US-007/US-008 là thần chú, được `kind = DHARANI` đảm nhận); `US-SUT-01..06` là mã chính thức. Resolved 2026-10-03 (người dùng: bản cũ không đầy đủ, có bản đầy đủ thì bỏ). Ảnh hưởng: `docs/specs/features/content/requirements.md` đã được sửa cùng ngày rồi **xoá hoàn toàn ở v2.4** (người dùng: kinh, thần chú, Phật và Bồ tát đều đã có feature riêng, file cũ bỏ luôn); các tính năng chỉ có ở bản cũ — đồng bộ delta `updated_since`, `version`, offline-first, `audio_url` — nay chỉ còn ở "Out of scope" của spec này; spec `practice` từng tham chiếu `audio_url` đã bị xoá vì cũ.
- ~~Ai duyệt việc phân loại `daily_chant` và `purposes`~~ → **không có người duyệt**: nhãn do em (Claude) gán theo bảng trong `design.md`; `review_status` giữ `UNREVIEWED` trong dữ liệu để sau này có người am hiểu duyệt thì đổi; không chặn phát hành. Resolved 2026-10-03 (người dùng: không có ai am hiểu tôn giáo để gán nhãn, giao cho em). Ảnh hưởng: `tasks.md` Phase 0 bỏ mục xác định người duyệt; `design.md` ghi lại ở Known gaps.
- ~~Bản Chú Đại Bi chuẩn~~ → **giữ bản phatgiao.org.vn (84 câu đánh số)** làm bản cuối; do em quyết vì không có người kiểm tra từng âm. Những chỗ lệch với bản văn xuôi cùng trang và với bản khác (`kiết`/`yết`, `dà`/`già`, …) vẫn ghi trong `note` của file dữ liệu; `review_status` giữ `UNREVIEWED`. Resolved 2026-10-03. Ảnh hưởng: không đổi dữ liệu hay thiết kế.
- ~~Có thêm script nạp làm cửa vào thứ hai không~~ → **có**: thêm script `cmd/seedsutras` **và** nạp khi server khởi động lần đầu, dùng chung usecase `SeedIfEmpty`; bảng đã có dữ liệu thì bỏ qua ở cả hai cửa vào. Resolved 2026-10-03. Ảnh hưởng: AC35 mới; `design.md` thêm hàng `cmd/seedsutras` và bỏ marker ở Step 1; `flow.md` đổi tên participant và bỏ marker; `tasks.md` thêm việc tạo script.
