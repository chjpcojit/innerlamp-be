# Requirements — temple

Version: 2.0
Last updated: 2026-10-15
Status: v2.0 viết lại để trỏ lên product spec `temple` v1.1 (2026-10-15). Phần đã có (US-TMP-01..08, AC 1–38) đã implement và kiểm chứng trước đó (E2E `services/appService/e2e/temple_e2e_test.go`, migration tới `0032`). **Mới ở v2.0, chưa implement:** trang 12 chùa (AC 5), tìm `q` theo tên + tên khác và lọc `region` ở `GET /v1/temples` (US-TMP-09, AC 39–44), bốn nhóm trường mới ở chi tiết (US-TMP-10, AC 45–49), file dữ liệu vào git (AC 50) — `tasks.md` Phase 11–13, mọi mục `[ ]`. Marker mở: ghi nguồn nội dung và quyền dùng ảnh (product A1).
Module: `content`
Code: `services/appService/`
Product spec: `docs/product/temple/product.md` @ v1.1
Contract: `docs/specs/api/openapi.yaml` (this repo, tag Temples)
Architecture: không có ADR (product §2.7)
Implements: BR-TPL-01, BR-TPL-02, BR-TPL-03, BR-TPL-04, BR-TPL-05, BR-TPL-06, FR-TPL-001, FR-TPL-002, FR-TPL-003, FR-TPL-004, FR-TPL-005, FR-TPL-006, FR-TPL-007, FR-TPL-010, NFR-TPL-01, NFR-TPL-02 — BR-TPL-07, 08, 09 và FR-TPL-008, 009 là việc của client (Backend `n/a` trong product §4.2)

> See `changelog.md` in this folder for the version history of this spec.

Spec platform (layer 2) của **danh bạ chùa** cho backend. Hành vi sản phẩm (thứ tự chùa, chùa nổi bật, cỡ trang, ba miền, cách khớp tìm kiếm, trường thiếu dữ liệu) nằm ở product spec và **không** nhắc lại ở đây; file này nói backend hiện thực các `BR-TPL-*`/`FR-TPL-*` đó thế nào. v1.x viết retroactively từ code; v2.0 trỏ lên product spec, gắn ID cho từng AC và thêm phần mới (prospective). Code là nguồn sự thật cho phần đã có; `api-contract.md` cùng thư mục là bản chép cho iOS, sẽ xoá ở bước dọn tài liệu.

Đây là danh mục **chỉ đọc**, công khai: một danh sách "nổi bật", một danh sách phân trang (tìm + lọc miền từ v2.0), một danh sách rút gọn để chọn chùa, một trang chi tiết. Dữ liệu **không** tự nạp khi service khởi động: người vận hành chạy `cmd/seedtemples` với file JSON ở mọi môi trường (FR-TPL-010).

## Ràng buộc từ product spec

- **Thứ tự** (BR-TPL-01) và **chùa nổi bật** (BR-TPL-02) giữ đúng như code: hạng di sản → năm lập → `id`; `temple.top_limit` là N.
- **Trang 12 chùa** (BR-TPL-03): `temple.page_size` đổi 20 → 12. `options_page_size` (100) không đổi — product §3.6 cho danh sách rút gọn giữ hành vi cũ.
- **Tìm và lọc trên server** (BR-TPL-05, BR-TPL-04, FR-TPL-003, FR-TPL-004): `GET /v1/temples` nhận `q` (khớp `name` **hoặc** một phần tử `aliases`) và `region`; `options` vẫn chỉ khớp `name` (product §3.6).
- **Trường mới ở chi tiết** (BR-TPL-06, FR-TPL-007; product Appendix B mục 4): giờ Rằm & Mùng Một, liên hệ (hotline, fanpage, website, email), giáo hội, tháng cập nhật. Server **không** tính trạng thái mở/đóng hay khoảng cách (BR-TPL-07, BR-TPL-09 là của client) nhưng phải trả giờ ở dạng client tính được.
- **Nội dung từ file JSON ở mọi môi trường** (FR-TPL-010, Appendix B mục 5): file vào git.
- Mã user story `US-TMP-01..08` và số AC giữ nguyên để không gãy tham chiếu trong code/test; AC mới đánh số tiếp từ 39.

---

## US-TMP-01: Xem chùa nổi bật

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **thấy vài ngôi chùa tiêu biểu ngay ở màn hình chủ**,
> Để **biết nên đến đâu mà không phải tìm kiếm**.

## Acceptance Criteria (EARS notation)

1. WHEN a `GET /v1/temples/top` request arrives, THE SYSTEM SHALL return `200 OK` with `data.items` holding the first `temple.top_limit` temples (dev: 10) ordered by `heritage_priority` ASC (di sản càng quan trọng càng đứng trước), then `established_year` ASC (chùa lâu đời hơn trước), then `id` ASC. (Không cần `Authorization`.) [FR-TPL-001, BR-TPL-01, BR-TPL-02]
2. WHEN temples are returned by `top`, THE SYSTEM SHALL return each one in the **summary** shape only — `id`, `name`, `type`, `region`, `thumbnail_url`, `image_url`, `sect`, `heritage_rank`, `established_year`, `location` (`lat`, `lng`, `address`, `province`; v1.7 thêm `province_code`, `ward_code`, `ward`, `full_address` — AC 33) — and SHALL NOT include `aliases`, `founder`, `current_abbot`, `operating_hours`, `amenities`, `introduction`, `architecture_heritage` or `visiting_notes`. [FR-TPL-001]
3. WHEN the table holds fewer temples than `temple.top_limit`, or none (chưa chạy seed), THE SYSTEM SHALL return all that exist, and SHALL return `items: []` (không phải `null`) for an empty table. [BR-TPL-02]

## US-TMP-02: Duyệt danh sách chùa theo trang

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **cuộn qua toàn bộ chùa theo từng trang**,
> Để **tìm thêm chùa ngoài phần nổi bật**.

## Acceptance Criteria (EARS notation)

4. WHEN a `GET /v1/temples` request arrives without `page` (or with an empty `page=`), THE SYSTEM SHALL serve page 1. Thứ tự giống hệt `top`, nên trang 1 bắt đầu bằng đúng các chùa của `top`. [FR-TPL-002, BR-TPL-01]
5. WHEN a page is served, THE SYSTEM SHALL return `data` with `items` (summary shape), `page`, `page_size` (`temple.page_size`, **12** từ v2.0 — dev trước đó: 20), `total` (số chùa trong toàn bảng) and `total_pages` = `ceil(total / page_size)`. [FR-TPL-002, BR-TPL-03]
6. WHEN `page` is greater than `total_pages`, THE SYSTEM SHALL return `200 OK` with `items: []` and the same `page_size`/`total`/`total_pages` — không phải lỗi. Client dừng khi `page >= total_pages` hoặc `items` rỗng. [BR-TPL-03]
7. WHEN `page` is not a positive integer (`0`, negative, non-numeric, an integer that overflows, or larger than 2^31−1), THE SYSTEM SHALL return `400 INVALID_PAGE` with `details.field = "page"`. [(platform)]

## US-TMP-03: Xem chi tiết một ngôi chùa

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **mở một ngôi chùa để xem đầy đủ thông tin**,
> Để **biết lịch sử, giờ mở cửa, tiện ích và lưu ý trước khi đến thăm**.

## Acceptance Criteria (EARS notation)

8. WHEN a `GET /v1/temples/{id}` request arrives with an existing `id`, THE SYSTEM SHALL return `200 OK` with the summary fields plus `aliases`, `founder`, `current_abbot`, `operating_hours`, `amenities` (`parking_car`, `parking_motorcycle`, `vegetarian_meals`, `summer_retreat`), `introduction`, `architecture_heritage` and `visiting_notes`. [FR-TPL-006, FR-TPL-007]
9. WHEN `{id}` matches no temple, THE SYSTEM SHALL return `404 TEMPLE_NOT_FOUND`. [FR-TPL-006]
10. WHEN the literal path `/v1/temples/top` is requested, THE SYSTEM SHALL route it to the top list, not to detail (`ServeMux` ưu tiên segment literal). `id` của chùa có dạng `temple_<region>_<n>` nên không bao giờ trùng `top`. [(platform)]

## US-TMP-04: Hình dạng dữ liệu nhất quán

## User Story

> Là một **lập trình viên client**,
> Tôi muốn **các field thiếu dữ liệu luôn có dạng xác định**,
> Để **decode một lần mà không phải đoán `null` hay chuỗi rỗng**.

## Acceptance Criteria (EARS notation)

11. WHEN a temple has an image or thumbnail, THE SYSTEM SHALL return `image_url` / `thumbnail_url` as a string; WHEN it has none (file seed hiện có đường dẫn ảnh cho cả 93 chùa nên nhánh này chưa xảy ra với dữ liệu seed hiện nay), THE SYSTEM SHALL return JSON `null` — không trả chuỗi rỗng. [BR-TPL-06]
12. WHEN a temple has no aliases or no visiting notes, THE SYSTEM SHALL return `[]` for `aliases` / `visiting_notes`, never `null`; WHEN it has no `introduction`, `architecture_heritage`, `founder`, `current_abbot` or `operating_hours`, THE SYSTEM SHALL return `""`, never `null`. [BR-TPL-06]
13. WHEN any temple route fails, THE SYSTEM SHALL answer in the shared envelope `{"success": false, "error": {code, message, type}, "details"?}`; a non-`GET` request returns `405 METHOD_NOT_ALLOWED` (kèm header `Allow`), an unknown path returns `404 NOT_FOUND`, and an infrastructure failure returns `500 INTERNAL_ERROR` without leaking the cause. [(platform)]

## US-TMP-05: Nạp dữ liệu chùa bằng lệnh seed

## User Story

> Là một **người vận hành hệ thống**,
> Tôi muốn **nạp và cập nhật danh mục chùa từ một file dữ liệu bằng một lệnh**,
> Để **sửa nội dung trong file rồi chạy lại mà không tạo trùng và không làm hỏng dữ liệu nửa chừng**.

## Acceptance Criteria (EARS notation)

14. WHEN the operator runs `go run ./services/appService/cmd/seedtemples [-file <path>]`, THE SYSTEM SHALL read the file (mặc định `assets/data/temple.json`, tính từ thư mục đang đứng), strip its `/* … */` comments, validate it, and upsert every temple in **one transaction** keyed by `id`, then log `seeded N temples from <file>`. [FR-TPL-010]
15. WHEN the seed is validated, THE SYSTEM SHALL reject the whole file (write nothing) if it has no temple, a duplicate `id`, an `id` that is not `temple_<north|central|south>_<n>`, a `heritage_rank` outside the eight known values, or a missing, blank or unlisted `location.province` (AC 28, 30); `region` is derived from the `id` and `heritage_priority` from `heritage_rank`. [FR-TPL-010, BR-TPL-04]
16. WHEN the seed is run again after the file was edited, THE SYSTEM SHALL update the existing rows in place by `id` — overwriting **every** column from the file and setting `updated_at = now()` — and SHALL NOT create a second row for an existing `id`. [FR-TPL-010]
17. WHEN the file is invalid, the database is unreachable, or the database rejects a row (vd. `type` ngoài 5 giá trị cho phép, migration `0005`/`0011` chưa apply), THE SYSTEM SHALL exit non-zero with a fatal log and SHALL leave the `temples` table unchanged (transaction rolled back). [FR-TPL-010]
18. WHEN appService starts, THE SYSTEM SHALL NOT seed or otherwise write the `temples` table — một DB chưa chạy seed trả `items: []` (AC 3, 6), không phải lỗi. [FR-TPL-010]

## US-TMP-06: Danh sách rút gọn, phân trang, để chọn chùa khi đăng ký

## User Story

> Là một **người dùng app đang đăng ký (chưa có tài khoản)**,
> Tôi muốn **tìm chùa theo tên (gõ có dấu hay không dấu đều được) và chọn nơi quy y/xuất gia từ một danh sách chỉ có tên chùa và tỉnh/thành, tải từng trang**,
> Để **không phải tải ảnh, toạ độ, địa chỉ dài và các trường chi tiết mà màn đăng ký không dùng, và danh sách vẫn nhẹ khi số chùa tăng lên**.

## Acceptance Criteria (EARS notation)

19. WHEN a `GET /v1/temples/options` request arrives without `page` (or with an empty `page=`), THE SYSTEM SHALL serve page 1; THE SYSTEM SHALL return `200 OK` with `data` holding `items`, `page`, `page_size` (`temple.options_page_size`, dev: 100), `total` (số chùa khớp, hoặc trong toàn bảng khi không có `q`) and `total_pages` = `ceil(total / page_size)`, in the same order as `top` and the paged list (`heritage_priority` ASC, `established_year` ASC, `id` ASC). (Không cần `Authorization` và không cần `X-API-Key`.) [FR-TPL-005, BR-TPL-01]
20. WHEN temples are returned by `options`, THE SYSTEM SHALL return each one with exactly four fields — `id`, `name`, `province` and (v1.7) `province_code` (string, `null` ở chùa chưa nạp lại — AC 32) — and SHALL NOT include any other field of the summary or detail shape (`address`, `type`, `region`, `thumbnail_url`, `image_url`, `sect`, `heritage_rank`, `established_year`, `location`, …). `province` là giá trị của cột `temples.province` (US-TMP-07) — tên tỉnh/thành **không tiền tố** (vd. `Hà Nội`, `Hồ Chí Minh`, `Cần Thơ`, `Thừa Thiên Huế`); THE SYSTEM SHALL NOT suy ra nó từ `address` lúc phục vụ request. [FR-TPL-005]
21. WHEN `page` is greater than `total_pages`, THE SYSTEM SHALL return `200 OK` with `items: []` and the same `page_size`/`total`/`total_pages` — không phải lỗi; WHEN `page` is not a positive integer (`0`, negative, non-numeric, an integer that overflows, or larger than 2^31−1), THE SYSTEM SHALL return `400 INVALID_PAGE` with `details.field = "page"`. [FR-TPL-005]
22. WHEN the table holds no temple (chưa chạy seed), THE SYSTEM SHALL return `200 OK` with `items: []` (không phải `null`), `total: 0` and `total_pages: 0`, and WHEN the literal path `/v1/temples/options` is requested, THE SYSTEM SHALL route it to this list, not to detail (`ServeMux` ưu tiên segment literal; `id` của chùa dạng `temple_<region>_<n>` nên không bao giờ trùng `options`). [FR-TPL-005]

23. WHEN `q` is present and, after trimming, is non-empty and at most 100 Unicode characters, THE SYSTEM SHALL return only the temples whose `name` contains the term as a substring, ignoring case and Vietnamese diacritics (`bich dong` và `BÍCH ĐỘNG` đều khớp `Chùa Bích Động`; `d` và `đ` khớp nhau); `total` and `total_pages` count only the matches, `page` applies to the matches, and the order is unchanged (ranking, không theo độ liên quan). [FR-TPL-005, BR-TPL-05]
24. WHEN `q` is present but empty or whitespace-only, or longer than 100 characters, THE SYSTEM SHALL return `400 INVALID_FILTER` with `details.field = "q"`; the `%`, `_` and `\` characters in `q` SHALL be matched literally, not as wildcards. (Client muốn "không tìm kiếm" thì bỏ hẳn tham số `q`.) [BR-TPL-05]
25. WHEN no temple matches `q`, THE SYSTEM SHALL return `200 OK` with `items: []`, `total: 0` and `total_pages: 0`. [FR-TPL-005]

> Tìm kiếm chỉ khớp trên `name`; không khớp `aliases` hay `province` (xem Out of scope).

## US-TMP-07: Tỉnh/thành là dữ liệu tường minh của chùa

## User Story

> Là một **lập trình viên client / người vận hành**,
> Tôi muốn **mỗi chùa có tỉnh/thành riêng, được lưu và nạp tường minh**,
> Để **hiển thị, lọc và tìm theo tỉnh mà không phải đoán từ chuỗi địa chỉ tự do**.

## Acceptance Criteria (EARS notation)

26. THE SYSTEM SHALL store each temple's province/city in its own column `temples.province` (`VARCHAR(100) NOT NULL`, tên không tiền tố như `Hà Nội`, `Hồ Chí Minh`), separate from `address` (địa chỉ đầy đủ, dạng chuỗi hiển thị), and SHALL NOT derive `province` from `address` while serving a request. [FR-TPL-007]
27. WHEN `GET /v1/temples/top`, `GET /v1/temples` or `GET /v1/temples/{id}` returns a temple, THE SYSTEM SHALL include `location.province` next to `lat`, `lng` and `address`; WHEN `GET /v1/temples/options` returns a temple, THE SYSTEM SHALL return it as the flat `province` field (AC 20). [FR-TPL-007]
28. WHEN the seed file is validated (AC 15), THE SYSTEM SHALL also reject the whole file if any temple has a missing, blank or longer-than-100-character `location.province`; the value is taken from the file verbatim (trimmed), not parsed from `address`. [FR-TPL-010]
29. WHEN migration `0024_temple_province.sql` is applied to a database that already holds temples, THE SYSTEM SHALL fill `province` from `address` (last comma-separated segment, minus the `Tỉnh`/`Thành phố`/`TP.` prefix, `HCM` → `Hồ Chí Minh`) for every row whose `province` is `''`, SHALL leave rows that already have a `province` untouched, and SHALL be safe to apply again (idempotent). [(platform)]

30. THE SYSTEM SHALL treat the valid values of `temples.province` as exactly the **34 province-level units in force since 1 July 2025** (Nghị quyết 202/2025/QH15: 6 thành phố — `Hà Nội`, `Huế`, `Hải Phòng`, `Đà Nẵng`, `Hồ Chí Minh`, `Cần Thơ` — và 28 tỉnh), written without the `Tỉnh`/`Thành phố`/`TP.` prefix, spelled and cased exactly as in `domain/province.go`; WHEN the seed file has a `location.province` outside that list (kể cả tên tỉnh cũ như `Hà Nam`, `Thừa Thiên Huế`, hoặc dạng có tiền tố/viết tắt như `Tỉnh Ninh Bình`, `TP.HCM`), THE SYSTEM SHALL reject the whole file and write nothing (AC 17). [FR-TPL-010]
31. WHEN migration `0025_temple_province_34_units.sql` is applied, THE SYSTEM SHALL rewrite `temples.province` from each merged-away old name to the new unit it became part of (`Hà Nam`, `Nam Định` → `Ninh Bình`; `Bình Dương`, `Bà Rịa - Vũng Tàu` → `Hồ Chí Minh`; `Thừa Thiên Huế` → `Huế`; …), SHALL leave every name already in the new form untouched, SHALL NOT modify `address`, and SHALL be safe to apply again (idempotent). [(platform)]

> `address` vẫn là chuỗi hiển thị theo đơn vị hành chính tại thời điểm biên soạn (còn nêu tỉnh và huyện cũ, vd. `Xã Ninh Hải, Huyện Hoa Lư, Tỉnh Ninh Bình`); spec này **không** cập nhật nó, nên `province` và `address` có thể không khớp nhau về tên đơn vị.
>
> **v1.7:** đoạn trên chỉ còn đúng cho chùa **chưa nạp lại** theo US-TMP-08. AC 28 và AC 30 được US-TMP-08 thay thế ở bước nạp (AC 35): danh sách hợp lệ không còn là 34 tên trong code mà là dataset `addr_new_provinces`.

## US-TMP-08: Địa chỉ chùa theo metadata hành chính (v1.7)

## User Story

> Là một **lập trình viên client / người vận hành / ban quản trị**,
> Tôi muốn **địa chỉ của chùa gồm mã tỉnh, mã xã lấy từ metadata hành chính chính thức và phần số nhà, đường người ta nhập**,
> Để **mọi chỗ có địa chỉ (chùa, đề xuất chùa, sau này là nhà thờ họ) cùng một cấu trúc và cùng nguồn đơn vị hành chính, thay vì chuỗi tự do theo huyện/xã cũ**.

## Acceptance Criteria (EARS notation)

32. THE SYSTEM SHALL store a temple's structured address in three further columns of `temples`: `province_code` and `ward_code` (mã **hệ mới** của feature `administrative-units`, `VARCHAR(16)`) and `ward` (tên đầy đủ của xã tại thời điểm nạp, vd `Phường Tân Định`), next to `province` (tên ngắn của tỉnh, AC 26) and `address`; THE SYSTEM SHALL require the three new columns to be either all `NULL` or all set, SHALL NOT place a foreign key on them, and SHALL leave them `NULL` for a temple that has not been re-seeded from a v1.7 file. [FR-TPL-007]
33. WHEN `GET /v1/temples/top`, `GET /v1/temples` or `GET /v1/temples/{id}` returns a temple, THE SYSTEM SHALL return `location` with `lat`, `lng`, `address`, `province`, `province_code` (string \| `null`), `ward_code` (string \| `null`), `ward` (string \| `null`) and `full_address` (string, không bao giờ `null`); WHEN the temple has a `ward`, `full_address` SHALL be the non-empty parts `address`, `ward`, `province` joined by `", "` in that order, WHEN it has none, `full_address` SHALL equal the stored `address` unchanged — và `address` của chùa đã nạp lại chỉ chứa phần dưới cấp xã (số nhà, đường, ngõ, thôn), không còn chứa xã/huyện/tỉnh. [FR-TPL-007]
34. WHEN the seed file is validated (AC 15), THE SYSTEM SHALL also reject the whole file if any temple has a missing or blank `location.province_code` or `location.ward_code`; the two codes are the only source of the unit names (AC 35). `location.address` SHALL be present as a string but MAY be empty — a temple known only by its ward has nothing below ward level — and is stored trimmed and as given (`docs/assumptions/temple/A-004`). *(Khác `create-temple`, nơi `address` bắt buộc — `temple-suggestion` AC18: đề xuất luôn mang `address`.)* [FR-TPL-010]
35. WHEN `cmd/seedtemples` resolves a temple's codes, THE SYSTEM SHALL look them up in `addr_new_provinces` / `addr_new_wards` — the same check `POST /v1/temple-suggestions` makes (AC 3a của `temple-suggestion`) — and SHALL reject the whole file and write nothing if a `province_code` is unknown or a `ward_code` is unknown or belongs to another province; SHALL take `province` (tên ngắn) and `ward` (tên đầy đủ) from the dataset, never from the file; AND WHEN the file also has `location.province` that differs from the dataset's short name for that code, SHALL reject the whole file. [FR-TPL-010]
36. WHEN the administrative-units tables are empty (dataset chưa nạp bằng `cmd/seedadminunits`), THE SYSTEM SHALL make `cmd/seedtemples` exit non-zero with a message naming that command and SHALL write nothing — một lần nạp chùa không được chạy với dữ liệu hành chính rỗng rồi ghi `NULL`. [FR-TPL-010]
37. WHEN migration `0032_temple_address_codes.sql` is applied, THE SYSTEM SHALL add the three nullable columns and the all-or-none CHECK, SHALL NOT change `address`, `province` or any other column of an existing row, SHALL NOT backfill codes (không thể suy ra xã từ chuỗi tự do một cách đáng tin; việc gán mã là việc dữ liệu có người duyệt, `tasks.md` Phase 10), and SHALL be safe to apply again (idempotent). [(platform)]
38. WHEN a temple is created from a suggestion through `create-temple` (spec `temple-suggestion` v3.2, AC 16/AC 18), THE SYSTEM SHALL store it with `province_code`, `ward_code`, `province` and `ward` all set under the same rules as AC 35, so that every temple created after v1.7 satisfies AC 32's "all set" branch; client của feature này chỉ đọc (Out of scope: tạo/sửa chùa qua API). [FR-TPL-010]

## US-TMP-09: Tìm và lọc theo miền ở danh sách chùa (v2.0)

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **tìm chùa theo tên hoặc tên khác, gõ không dấu cũng được, và lọc theo miền Bắc / Trung / Nam**,
> Để **tìm ra chùa mình cần mà app không phải tải cả danh bạ về máy**.

## Acceptance Criteria (EARS notation)

39. WHEN `GET /v1/temples` has a `q` that, after trimming, is non-empty and at most 100 Unicode characters, THE SYSTEM SHALL return only the temples whose `name` **or any element of** `aliases` contains the term as a substring, ignoring case and Vietnamese diacritics (`đ` = `d`), with `%`, `_` and `\` matched literally; `total` and `total_pages` count only the matches, `page` applies to the matches, and the order is the one of AC 4. Vectors: `docs/product/temple/rules/temple-search-match.json`. [FR-TPL-003, BR-TPL-05]
40. WHEN `GET /v1/temples` has `q` present but empty or whitespace-only, or longer than 100 characters, THE SYSTEM SHALL return `400 INVALID_FILTER` with `details.field = "q"` (cùng quy tắc với `options`, AC 24). [BR-TPL-05]
41. WHEN `GET /v1/temples` has `region` equal to `north`, `central` or `south`, THE SYSTEM SHALL return only the temples of that region; WHEN both `q` and `region` are present, THE SYSTEM SHALL apply both. [FR-TPL-004, BR-TPL-04]
42. WHEN `region` is present with any other value, including empty `region=` or a different case such as `North`, THE SYSTEM SHALL return `400 INVALID_FILTER` with `details.field = "region"`; WHEN both `q` and `region` are invalid, `region` is reported first. *(Client muốn "Tất cả" thì bỏ hẳn tham số.)* [(platform)]
43. WHEN no temple matches the given `q` and/or `region`, THE SYSTEM SHALL return `200 OK` with `items: []`, `total: 0` and `total_pages: 0`. [FR-TPL-003]
44. WHEN `GET /v1/temples/top` or `GET /v1/temples/options` receives `region`, THE SYSTEM SHALL ignore it (không lọc, không lỗi); `options` keeps matching `q` on `name` only (AC 23). [BR-TPL-05]

> NFR-TPL-01 (trang đầu ≤ 2 s trên 4G) và NFR-TPL-02 (không tải cả danh bạ) được đáp ứng bằng tìm/lọc phía server theo trang; với vài trăm chùa quét tuần tự là đủ, xem `design.md` Known gaps. [NFR-TPL-01, NFR-TPL-02]

## US-TMP-10: Giờ lễ, liên hệ, giáo hội, tháng cập nhật ở chi tiết chùa (v2.0)

## User Story

> Là một **người dùng app**,
> Tôi muốn **xem giờ mở cửa ngày Rằm & Mùng Một, số điện thoại, fanpage, website, email, giáo hội quản lý và lần cập nhật gần nhất của chùa**,
> Để **biết chùa có mở không và liên hệ trước khi đến**.

## Acceptance Criteria (EARS notation)

45. WHEN `GET /v1/temples/{id}` returns a temple, THE SYSTEM SHALL also return `holiday_operating_hours` (string), `sangha_body` (string), `contacts` `{hotline, fanpage, website, email}` (strings) and `content_updated_month` (string `YYYY-MM` \| `null`); a missing text value is `""`, never `null` (cùng quy ước AC 12), a missing month is `null`. These fields SHALL NOT be added to the summary shape (`top`, list) or to `options`. [FR-TPL-007, BR-TPL-06]
46. THE SYSTEM SHALL return `operating_hours` and `holiday_operating_hours` either as `""` or in the exact form `HH:MM - HH:MM` (24 giờ, giờ Việt Nam, giờ mở < giờ đóng), so that clients can compute BR-TPL-09 without parsing free text; `holiday_operating_hours = ""` means the normal hours also apply on Rằm & Mùng Một. [BR-TPL-06]
47. WHEN the seed file is validated (AC 15), THE SYSTEM SHALL also reject the whole file if `operating_hours` or `holiday_operating_hours` is non-empty and not in the form of AC 46 (kể cả giờ mở ≥ giờ đóng), if `content_updated_month` is present and not a real month `YYYY-MM`, if `contacts.email` is non-empty and has no single `@` with text on both sides, or if `contacts.website`/`contacts.fanpage` is non-empty and does not start with `https://`; all new keys are optional in the file. [FR-TPL-010]
48. WHEN migration `0037_temple_contact_hours.sql` is applied, THE SYSTEM SHALL add `holiday_operating_hours`, `sangha_body`, `hotline`, `fanpage`, `website`, `email` as `NOT NULL DEFAULT ''` and `content_updated_month` as nullable `DATE` (ngày 1 của tháng), SHALL NOT change any existing column, and SHALL be safe to apply again. [(platform)]
49. WHEN the seed runs again (AC 16), THE SYSTEM SHALL overwrite the new columns from the file like every other column — a key missing from the file writes `""` / `NULL`. [FR-TPL-010]

## US-TMP-11: File dữ liệu chùa nằm trong git (v2.0)

## User Story

> Là một **người vận hành**,
> Tôi muốn **file dữ liệu chùa được quản lý phiên bản cùng backend**,
> Để **nạp cùng một nội dung ở mọi môi trường và xem được ai đổi gì**.

## Acceptance Criteria (EARS notation)

50. THE SYSTEM SHALL keep `assets/data/temple.json` under version control (an exception for that path in the root `.gitignore`, which ignores `*.json`), and `cmd/seedtemples` SHALL keep reading it as its default `-file`. [FR-TPL-010]

---

## API contract (confirmed surface for this feature)

Ba route đầu đọc từ handler, `cmd/main.go`; đã có trong `docs/specs/api/openapi.yaml` (tag Temples). **v2.0 chưa có trong openapi:** `q`/`region` của danh sách và bốn nhóm trường của chi tiết — cập nhật cùng lúc implement (`tasks.md` Phase 13). Route `options` (v1.4) phân trang; `openapi.yaml` (và bản đồng bộ `web/docs/public/openapi.yaml`) có route này. `api-contract.md` ghi 3 route này đã được xác nhận trên server dev (số chùa trong tài liệu đó là của lần chạy lúc ấy; file seed hiện có 93 chùa).

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/temples/top` | public | `200` `{items: [TempleSummary]}` | 404 `NOT_FOUND` (sai path), 405, 500 |
| GET | `/v1/temples?page=N&q=&region=` | public | `200` `{items, page, page_size, total, total_pages}` | 400 `INVALID_PAGE`, 400 `INVALID_FILTER` (`q`, `region`) *(v2.0)*, 405, 500 |
| GET | `/v1/temples/{id}` | public | `200` `TempleDetail` | 404 `TEMPLE_NOT_FOUND`, 405, 500 |
| GET | `/v1/temples/options?page=N&q=` | public | `200` `{items: [TempleOption], page, page_size, total, total_pages}` | 400 `INVALID_PAGE`, 400 `INVALID_FILTER` (`q`), 404 `NOT_FOUND` (sai path), 405, 500 |

`TempleSummary`: `id` (`temple_<region>_<n>`), `name`, `type` (`CHUA` \| `THIEN_VIEN` \| `DAO_TRANG` \| `TINH_XA` \| `THIEN_XA`), `region` (`north` \| `central` \| `south`), `thumbnail_url` (string \| `null`), `image_url` (string \| `null`), `sect`, `heritage_rank` (8 giá trị, xem `design.md`), `established_year` (int), `location` `{lat, lng, address, province, province_code, ward_code, ward, full_address}` *(v1.7: `province_code`, `ward_code`, `ward` là string \| `null`; `full_address` là string; `address` chỉ là phần dưới cấp xã với chùa đã nạp lại — AC 33)*.

`TempleOption`: `id` (string, dùng làm `ordination_temple_id` ở đăng ký), `name` (string), `province` (string, giá trị cột `temples.province`, tên tỉnh/thành không tiền tố), `province_code` (string \| `null`, mã tỉnh hệ mới — v1.7, AC 20).

`TempleDetail` = `TempleSummary` + `aliases` (`[string]`), `founder`, `current_abbot`, `operating_hours`, `amenities` `{parking_car, parking_motorcycle, vegetarian_meals, summer_retreat}` (bool), `introduction`, `architecture_heritage`, `visiting_notes` (`[string]`); v2.0 thêm `holiday_operating_hours`, `sangha_body`, `contacts` `{hotline, fanpage, website, email}`, `content_updated_month` (AC 45).

Các chuỗi kiểu enum (`type`, `sect`, `heritage_rank`) là tập **mở** với client: `type` và `heritage_rank` bị `CHECK` ở DB, còn `sect` thì không, nên giá trị mới có thể xuất hiện sau một lần seed.

## Out of scope (this feature boundary, not tracked here)

- Tạo/sửa/xoá chùa qua API hoặc backoffice — ngoài phạm vi (product §1.2); chùa chỉ vào bảng bằng `cmd/seedtemples` hoặc `create-temple` của `temple-suggestion`.
- Theo dõi chùa (feature riêng, sau), báo thông tin sai (bỏ), cúng dường (`offering`) — product §1.2.
- Tìm gần vị trí, sắp theo khoảng cách, lọc theo `sect`/`heritage_rank`/`province`, xếp theo độ liên quan.
- Tính trạng thái mở/đóng và khoảng cách — client (BR-TPL-07, BR-TPL-09).
- Kiến trúc chi tiết (kiểu, vật liệu, điểm nổi bật), tiện ích "Kinh sách", bộ ảnh — product để sau (BR-TPL-06).
- Lưu trữ/phân phối ảnh: chỉ có hai cột `image_url`/`thumbnail_url`.
- Đăng nhập: cả bốn route công khai.

## Pending (not yet implemented)

- v2.0: trang 12 (AC 5), `q` + `region` ở danh sách (AC 39–44), trường mới ở chi tiết (AC 45–49), file dữ liệu vào git (AC 50) — `tasks.md` Phase 11–13.
- Phần còn lại của Phase 10: `erd.svg`, xác minh 6 chùa không phải `high` (A-005).

> ⚠️ Not confirmed (product): nội dung biên tập có phải ghi nguồn không, và InnerLamp có quyền dùng ảnh trong file dữ liệu không (product Appendix A, mục 1) — chặn phát hành công khai nội dung, không chặn Phase 11–13.
> ⚠️ Not confirmed: độ chính xác dữ liệu — `established_year` có thể khác năm nêu trong `introduction`; chưa chốt ai rà soát — không chặn code, ảnh hưởng chất lượng nội dung.

## Ghi chú kỹ thuật liên quan

- Dữ liệu seed (`assets/data/temple.json`, 93 chùa, đếm lại 2026-10-11): 32 miền Bắc, 32 miền Trung, 29 miền Nam (theo `id`); `type`: 73 `CHUA`, 8 `THIEN_VIEN`, 7 `DAO_TRANG`, 2 `TINH_XA`, 3 `THIEN_XA`; `heritage_rank`: 3 `DI_SAN_THE_GIOI`, 19 `DI_TICH_QUOC_GIA_DAC_BIET`, 33 `DI_TICH_QUOC_GIA`, 16 `DANH_LAM_THANG_CANH`, 13 `DI_TICH_CAP_TINH`, 6 `DI_TICH_CAP_THANH_PHO`, 3 `CHUA_XEP_HANG` (không có `KHU_DU_LICH_QUOC_GIA`); cả 93 chùa có `introduction`, `architecture_heritage`, `visiting_notes`, `image_url` và `thumbnail_url`; 24 giá trị `location.province` khác nhau. File dùng chú thích `/* … */` làm tiêu đề vùng nên không phải JSON thuần — `encoding/json` từ chối nếu không qua bước bỏ chú thích của seeder.
- `id` là chuỗi ổn định lấy từ dữ liệu nguồn (`temple_north_011`, …), không phải số tự tăng: seeder upsert theo `id` nên id client đã cache vẫn đúng sau mọi lần nạp lại.
- Thứ tự `heritage_priority ASC, established_year ASC, id ASC` có `id` là tie-break cuối để hai chùa cùng hạng và cùng năm không đổi chỗ giữa các trang; index `idx_temples_ranking` phục vụ đúng thứ tự này.
- `heritage_priority` là cột dẫn xuất từ `heritage_rank` bằng một bảng ánh xạ duy nhất trong `domain/temple.go`; thêm hạng mới là một migration cộng một dòng trong bảng đó, không phải chỉnh config.
- `total` và trang được đọc bằng hai câu lệnh riêng (đếm rồi lấy trang), không cùng một snapshot — xem `design.md`.
- `options` phân trang theo `temple.options_page_size` (dev: 100, tách khỏi `temple.page_size` vì một pick-list cần trang lớn hơn danh sách duyệt); khoá này **bắt buộc** (`TempleConfig.Validate` từ chối `< 1` lúc khởi động). Như danh sách phân trang, `total` và trang được đọc bằng hai câu lệnh riêng, không cùng snapshot.
- `province` là cột riêng từ migration `0024` (v1.5), nhận đúng 34 giá trị của `domain/province.go` (v1.6, migration `0025` đổi dữ liệu cũ); thay cách suy ra từ `address` ở v1.4. Giá trị đến từ file seed (`location.province`), còn migration chỉ backfill một lần cho dòng đã có.
- Tìm kiếm dùng extension `unaccent` đã được bật ở migration `0019` (cùng cách `GET /v1/sutras?q=`): `unaccent(lower(name)) LIKE unaccent(lower($1))`, không index — đủ cho vài trăm chùa; nếu bảng lớn lên hàng chục nghìn thì thêm index trigram.
- Kích thước trang nằm ở block `temple:` của `config.<env>.yaml` (dev: `top_limit: 10`, `page_size: 20` — **v2.0: 12**, AC 5); cả hai bắt buộc — giá trị 0 nghĩa là trang rỗng.
- Bảng `temples` phải tồn tại trước khi service chạy (`make db-up` áp `0005_temples.sql` và `0011_temple_content.sql`).

## Open questions (resolved during implementation)

Đã chốt ở v2.0 (2026-10-15, qua product spec `temple` v1.1 — anh Đức):

- **Mã user story** — giữ `US-TMP-*`; ID sản phẩm là `BR-TPL-*`/`FR-TPL-*`, gắn ở cuối từng AC. Đóng marker cũ.
- **Quy trình nạp ở môi trường thật** — dùng file JSON nạp bằng `cmd/seedtemples` ở mọi môi trường (FR-TPL-010); file vào git (AC 50). Phần "seed đang dùng config dev" còn lại là việc kỹ thuật (`tasks.md` Phase 3, mục staging/production). Đóng marker cũ.
- **Tiêu chí "nổi bật"** — giữ nguyên (BR-TPL-01, BR-TPL-02). Đóng marker cũ.
- **Phạm vi spec backoffice** — quản lý chùa ở BO ngoài phạm vi (product §1.2). Đóng marker cũ.
- **Ảnh và ghi nguồn** — chuyển thành một marker product ở Pending.
- **Cỡ trang, tìm, lọc, trường mới** — AC 5, 39–50.

Ảnh hưởng: `design.md` (handler đọc `region`, usecase kiểm `region`, repository dựng `WHERE` chung cho đếm + trang có `aliases`, cột mới, seed kiểm định dạng, `.gitignore`), `flow.md` (sequence danh sách thêm bước kiểm `q`/`region`), `tasks.md` (Phase 11–13).

Chưa có câu hỏi nào được người dùng trả lời trong v1.0–v1.2; các câu còn mở nằm ở "Pending" và ở US-TMP-06.

Đã chốt ở v1.7 (2026-10-11), cho địa chỉ chùa (US-TMP-08):

- **`full_address` có cần tiền tố `Tỉnh`/`Thành phố` cho tỉnh không?** — Không: giữ tên ngắn như đã lưu, vd `Hồ Chí Minh` (anh Đức, 2026-10-11). Ảnh hưởng: AC 33 giữ nguyên (không thêm cột `province_full_name`); marker `⚠️ Not confirmed` ở US-TMP-08 của v1.7 được đóng; `docs/assumptions/temple/A-002` ghi phần này đã xác nhận.
- **Chuẩn hoá địa chỉ chùa theo metadata hành chính?** — Có: dữ liệu `options` đang seed ít khi chạy, cần bổ sung `province_code`/`ward_code` chính thức; mọi chỗ có địa chỉ gồm `province_code` + `ward_code` + `address` (phần số nhà, đường, … do người dùng nhập) (anh Đức, 2026-10-11). Ảnh hưởng: AC 32–38; migration `0032`; `assets/data/temple.json` đổi dạng `location`; seed kiểm mã qua `addr_new_*`; AC 28/30 bị thay ở bước nạp. Các lựa chọn spec tự thêm (không FK + tên chụp, `address` đổi nghĩa + `full_address`, `options` chỉ thêm `province_code`) ghi ở `docs/assumptions/temple/A-001..A-003`.

Đã chốt ở v1.4 (2026-10-10), cho `GET /v1/temples/options`:

- **`options` có phân trang không?** — Có, "cho tương lai số lượng lớn hơn" (người dùng, 2026-10-10). Thay quyết định mặc định "không phân trang" của v1.3. Ảnh hưởng: AC 19, 21, 22; `design.md` (repository đếm + lấy trang, config `options_page_size`); `tasks.md` Phase 7.
- **`address` trả nguyên chuỗi hay cắt gọn?** — Cắt gọn thành tỉnh/thành, "Hà Nội, Hồ Chí Minh…" (người dùng, 2026-10-10). Ảnh hưởng: AC 20; field đổi tên `address` → `province` vì giá trị không còn là địa chỉ; spec chọn suy ra từ `address` thay vì thêm cột (xem `design.md`).
- **Tìm theo tên có cần khớp không dấu không, và đặt ở đâu?** — Có khớp không dấu, thêm `?q=` vào `GET /v1/temples/options` (người dùng, 2026-10-10). Ảnh hưởng: AC 23–25; `design.md` (usecase `Options(query, page)`, repository `unaccent`); marker `⚠️ Not confirmed` về `?q=` của v1.4 được đóng.
- **Tách địa chỉ thế nào để không phải đoán?** — Thêm cột `temples.province`; migration tự lấy `address` tách ra rồi nạp vào DB (người dùng, 2026-10-10). Ảnh hưởng: US-TMP-07 (AC 26–29); migration `0024`; seed JSON có `location.province`; hàm suy ra `province` ở domain bị bỏ. Quyết định danh sách tên chuẩn **chưa** trả lời → marker `⚠️ Not confirmed` ở US-TMP-07.
- **Danh sách tên tỉnh/thành chuẩn: 34 đơn vị mới hay 63 tỉnh cũ?** — Dùng **34 đơn vị mới** (người dùng, 2026-10-10). Ảnh hưởng: AC 28, 30, 31; `domain/province.go`; seed từ chối tên ngoài danh sách; migration `0025`; dữ liệu `assets/data/temple.json` đổi 34 dòng. Marker `⚠️ Not confirmed` ở US-TMP-07 được đóng. Không giải quyết: cập nhật `address` theo đơn vị mới (ghi ở Known gaps của `design.md`).
