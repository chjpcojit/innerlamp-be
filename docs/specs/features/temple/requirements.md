# Requirements — temple

Version: 1.2
Last updated: 2026-10-03
Status: implemented (2026-09-30) — 3 route đọc công khai chạy thật qua binary + Postgres dev (179 chùa, theo `api-contract.md`); có test usecase, handler, seed parser và repository. **Chưa có** E2E tự động, chưa có OKF concept. Còn 6 marker `⚠️ Not confirmed`.
Module: `content`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này được viết **retroactively** từ danh mục chùa đã implement (Retroactive mode). Mọi AC dưới đây mô tả đúng hành vi code hiện có; thứ chưa làm nằm ở "Pending", không gộp vào AC. `api-contract.md` cùng thư mục là tài liệu **dành cho client iOS** (viết trước spec này, giữ nguyên); khi hai file lệch nhau thì code là nguồn sự thật và spec này được sửa theo code.

Đây là danh mục **chỉ đọc**, công khai: một danh sách "nổi bật", một danh sách phân trang, một trang chi tiết — cùng hình dạng với `deity` và `article`. Khác `article` ở chỗ dữ liệu **không** tự nạp khi service khởi động: người vận hành chạy lệnh `cmd/seedtemples` bằng tay.

> ⚠️ Not confirmed: mã user story của tính năng này chưa được product chốt — spec tạm đánh `US-TMP-01..05` theo tiền lệ `US-ART-01` của `article`; đổi khi product chốt số.

---

## US-TMP-01: Xem chùa nổi bật

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **thấy vài ngôi chùa tiêu biểu ngay ở màn hình chủ**,
> Để **biết nên đến đâu mà không phải tìm kiếm**.

## Acceptance Criteria (EARS notation)

1. WHEN a `GET /v1/temples/top` request arrives, THE SYSTEM SHALL return `200 OK` with `data.items` holding the first `temple.top_limit` temples (dev: 10) ordered by `heritage_priority` ASC (di sản càng quan trọng càng đứng trước), then `established_year` ASC (chùa lâu đời hơn trước), then `id` ASC. (Không cần `Authorization`.)
2. WHEN temples are returned by `top`, THE SYSTEM SHALL return each one in the **summary** shape only — `id`, `name`, `type`, `region`, `thumbnail_url`, `image_url`, `sect`, `heritage_rank`, `established_year`, `location` (`lat`, `lng`, `address`) — and SHALL NOT include `aliases`, `founder`, `current_abbot`, `operating_hours`, `amenities`, `introduction`, `architecture_heritage` or `visiting_notes`.
3. WHEN the table holds fewer temples than `temple.top_limit`, or none (chưa chạy seed), THE SYSTEM SHALL return all that exist, and SHALL return `items: []` (không phải `null`) for an empty table.

## US-TMP-02: Duyệt danh sách chùa theo trang

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **cuộn qua toàn bộ chùa theo từng trang**,
> Để **tìm thêm chùa ngoài phần nổi bật**.

## Acceptance Criteria (EARS notation)

4. WHEN a `GET /v1/temples` request arrives without `page` (or with an empty `page=`), THE SYSTEM SHALL serve page 1. Thứ tự giống hệt `top`, nên trang 1 bắt đầu bằng đúng các chùa của `top`.
5. WHEN a page is served, THE SYSTEM SHALL return `data` with `items` (summary shape), `page`, `page_size` (`temple.page_size`, dev: 20), `total` (số chùa trong toàn bảng) and `total_pages` = `ceil(total / page_size)`.
6. WHEN `page` is greater than `total_pages`, THE SYSTEM SHALL return `200 OK` with `items: []` and the same `page_size`/`total`/`total_pages` — không phải lỗi. Client dừng khi `page >= total_pages` hoặc `items` rỗng.
7. WHEN `page` is not a positive integer (`0`, negative, non-numeric, an integer that overflows, or larger than 2^31−1), THE SYSTEM SHALL return `400 INVALID_PAGE` with `details.field = "page"`.

## US-TMP-03: Xem chi tiết một ngôi chùa

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **mở một ngôi chùa để xem đầy đủ thông tin**,
> Để **biết lịch sử, giờ mở cửa, tiện ích và lưu ý trước khi đến thăm**.

## Acceptance Criteria (EARS notation)

8. WHEN a `GET /v1/temples/{id}` request arrives with an existing `id`, THE SYSTEM SHALL return `200 OK` with the summary fields plus `aliases`, `founder`, `current_abbot`, `operating_hours`, `amenities` (`parking_car`, `parking_motorcycle`, `vegetarian_meals`, `summer_retreat`), `introduction`, `architecture_heritage` and `visiting_notes`.
9. WHEN `{id}` matches no temple, THE SYSTEM SHALL return `404 TEMPLE_NOT_FOUND`.
10. WHEN the literal path `/v1/temples/top` is requested, THE SYSTEM SHALL route it to the top list, not to detail (`ServeMux` ưu tiên segment literal). `id` của chùa có dạng `temple_<region>_<n>` nên không bao giờ trùng `top`.

## US-TMP-04: Hình dạng dữ liệu nhất quán

## User Story

> Là một **lập trình viên client**,
> Tôi muốn **các field thiếu dữ liệu luôn có dạng xác định**,
> Để **decode một lần mà không phải đoán `null` hay chuỗi rỗng**.

## Acceptance Criteria (EARS notation)

11. WHEN a temple has an image or thumbnail, THE SYSTEM SHALL return `image_url` / `thumbnail_url` as a string; WHEN it has none (hiện đúng với cả 179 chùa), THE SYSTEM SHALL return JSON `null` — không trả chuỗi rỗng.
12. WHEN a temple has no aliases or no visiting notes, THE SYSTEM SHALL return `[]` for `aliases` / `visiting_notes`, never `null`; WHEN it has no `introduction`, `architecture_heritage`, `founder`, `current_abbot` or `operating_hours`, THE SYSTEM SHALL return `""`, never `null`.
13. WHEN any temple route fails, THE SYSTEM SHALL answer in the shared envelope `{"success": false, "error": {code, message, type}, "details"?}`; a non-`GET` request returns `405 METHOD_NOT_ALLOWED` (kèm header `Allow`), an unknown path returns `404 NOT_FOUND`, and an infrastructure failure returns `500 INTERNAL_ERROR` without leaking the cause.

## US-TMP-05: Nạp dữ liệu chùa bằng lệnh seed

## User Story

> Là một **người vận hành hệ thống**,
> Tôi muốn **nạp và cập nhật danh mục chùa từ một file dữ liệu bằng một lệnh**,
> Để **sửa nội dung trong file rồi chạy lại mà không tạo trùng và không làm hỏng dữ liệu nửa chừng**.

## Acceptance Criteria (EARS notation)

14. WHEN the operator runs `go run ./services/appService/cmd/seedtemples [-file <path>]`, THE SYSTEM SHALL read the file (mặc định `assets/data/temple.json`, tính từ thư mục đang đứng), strip its `/* … */` comments, validate it, and upsert every temple in **one transaction** keyed by `id`, then log `seeded N temples from <file>`.
15. WHEN the seed is validated, THE SYSTEM SHALL reject the whole file (write nothing) if it has no temple, a duplicate `id`, an `id` that is not `temple_<north|central|south>_<n>`, or a `heritage_rank` outside the eight known values; `region` is derived from the `id` and `heritage_priority` from `heritage_rank`.
16. WHEN the seed is run again after the file was edited, THE SYSTEM SHALL update the existing rows in place by `id` — overwriting **every** column from the file and setting `updated_at = now()` — and SHALL NOT create a second row for an existing `id`.
17. WHEN the file is invalid, the database is unreachable, or the database rejects a row (vd. `type` ngoài 5 giá trị cho phép, migration `0005`/`0011` chưa apply), THE SYSTEM SHALL exit non-zero with a fatal log and SHALL leave the `temples` table unchanged (transaction rolled back).
18. WHEN appService starts, THE SYSTEM SHALL NOT seed or otherwise write the `temples` table — một DB chưa chạy seed trả `items: []` (AC 3, 6), không phải lỗi.

## API contract (confirmed surface for this feature)

Đọc từ handler, `cmd/main.go`; đã có trong `docs/specs/api/openapi.yaml` (tag Temples). `api-contract.md` ghi 3 route này đã được xác nhận trên server dev với 179 chùa.

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/temples/top` | public | `200` `{items: [TempleSummary]}` | 404 `NOT_FOUND` (sai path), 405, 500 |
| GET | `/v1/temples?page=N` | public | `200` `{items, page, page_size, total, total_pages}` | 400 `INVALID_PAGE`, 405, 500 |
| GET | `/v1/temples/{id}` | public | `200` `TempleDetail` | 404 `TEMPLE_NOT_FOUND`, 405, 500 |

`TempleSummary`: `id` (`temple_<region>_<n>`), `name`, `type` (`CHUA` \| `THIEN_VIEN` \| `DAO_TRANG` \| `TINH_XA` \| `THIEN_XA`), `region` (`north` \| `central` \| `south`), `thumbnail_url` (string \| `null`), `image_url` (string \| `null`), `sect`, `heritage_rank` (8 giá trị, xem `design.md`), `established_year` (int), `location` `{lat, lng, address}`.

`TempleDetail` = `TempleSummary` + `aliases` (`[string]`), `founder`, `current_abbot`, `operating_hours`, `amenities` `{parking_car, parking_motorcycle, vegetarian_meals, summer_retreat}` (bool), `introduction`, `architecture_heritage`, `visiting_notes` (`[string]`).

Các chuỗi kiểu enum (`type`, `sect`, `heritage_rank`) là tập **mở** với client: `type` và `heritage_rank` bị `CHECK` ở DB, còn `sect` thì không, nên giá trị mới có thể xuất hiện sau một lần seed.

## Out of scope (this feature boundary, not tracked here)

- Tạo/sửa/xoá chùa qua API hoặc backoffice — hiện chỉ có `cmd/seedtemples` và SQL trực tiếp. `admin/requirements.md` ghi việc quản lý temple/deity qua backoffice sẽ có spec riêng.
- Tìm kiếm, lọc theo `region`/`sect`/`heritage_rank`, tìm gần vị trí, yêu thích, đánh giá.
- Lưu trữ/phân phối ảnh: chỉ có hai cột `image_url`/`thumbnail_url`, hiện rỗng cho mọi chùa.
- Đăng nhập: cả ba route công khai như `deity` và `article`.

## Pending (not yet implemented)

- Test E2E chạy binary thật + Postgres cho ba route — `tasks.md` Phase 5.
- OKF concept `docs/knowledge/features/temple.md` — chưa có; do `bdd-okf-docs` tạo sau khi có E2E, `tasks.md` Phase 5.

> ⚠️ Not confirmed: quy trình nạp dữ liệu ở môi trường thật — `cmd/seedtemples` đang dùng config và secret **dev** (`config.dev.yaml`, `NewMockSecretGetterFromYAML`, đánh dấu `⚠️ DEV ONLY`), và file JSON đọc từ đĩa chứ không nhúng vào binary (khác `article`); chưa chốt ai chạy seed ở staging/production, bằng công cụ nào, hay chuyển sang nạp tự động khi khởi động.
> ⚠️ Not confirmed: ảnh chùa — cả 179 chùa đều không có `image_url`/`thumbnail_url` (API trả `null`, client dùng placeholder); chưa chốt nguồn ảnh, nơi lưu (GCS?) và quyền sử dụng.
> ⚠️ Not confirmed: quyền dùng và ghi công của nội dung biên tập — `introduction`, `architecture_heritage`, `visiting_notes` được biên soạn từ nguồn công khai (Wikipedia tiếng Việt, trang du lịch, trang tin Phật giáo) và bảng `temples` **không** lưu nguồn; chưa xác nhận điều kiện hiển thị và có phải ghi nguồn hay không.
> ⚠️ Not confirmed: độ chính xác dữ liệu — `established_year` đến từ file seed và có thể khác với năm nêu trong `introduction`; chưa chốt ai rà soát và theo quy trình nào.
> ⚠️ Not confirmed: tiêu chí "nổi bật" — hiện là hạng di sản rồi đến năm thành lập (chùa cổ trước); chưa xác nhận với product rằng thứ tự này là đủ cho màn hình chủ (vd. có cần xét vùng miền hay vị trí người dùng không).
> ⚠️ Not confirmed: phạm vi spec backoffice — `admin/requirements.md` viết rằng quản lý temple/deity có spec riêng trong `temple/` và `deity/`, nhưng hiện thư mục này mới chỉ có spec danh mục đọc; cần chốt spec CRUD backoffice nằm ở đây hay ở một feature mới.

## Ghi chú kỹ thuật liên quan

- Dữ liệu seed (`assets/data/temple.json`, 179 chùa): 60 miền Bắc, 59 miền Trung, 60 miền Nam (theo `id`); `type`: 114 `CHUA`, 25 `THIEN_VIEN`, 18 `DAO_TRANG`, 14 `TINH_XA`, 8 `THIEN_XA`; `heritage_rank`: 3 `DI_SAN_THE_GIOI`, 24 `DI_TICH_QUOC_GIA_DAC_BIET`, 1 `KHU_DU_LICH_QUOC_GIA`, 52 `DI_TICH_QUOC_GIA`, 34 `DANH_LAM_THANG_CANH`, 30 `DI_TICH_CAP_TINH`, 15 `DI_TICH_CAP_THANH_PHO`, 20 `CHUA_XEP_HANG`; cả 179 chùa có `introduction`, `architecture_heritage`, `visiting_notes`; 0 chùa có ảnh. File dùng chú thích `/* … */` làm tiêu đề vùng nên không phải JSON thuần — `encoding/json` từ chối nếu không qua bước bỏ chú thích của seeder.
- `id` là chuỗi ổn định lấy từ dữ liệu nguồn (`temple_north_011`, …), không phải số tự tăng: seeder upsert theo `id` nên id client đã cache vẫn đúng sau mọi lần nạp lại.
- Thứ tự `heritage_priority ASC, established_year ASC, id ASC` có `id` là tie-break cuối để hai chùa cùng hạng và cùng năm không đổi chỗ giữa các trang; index `idx_temples_ranking` phục vụ đúng thứ tự này.
- `heritage_priority` là cột dẫn xuất từ `heritage_rank` bằng một bảng ánh xạ duy nhất trong `domain/temple.go`; thêm hạng mới là một migration cộng một dòng trong bảng đó, không phải chỉnh config.
- `total` và trang được đọc bằng hai câu lệnh riêng (đếm rồi lấy trang), không cùng một snapshot — xem `design.md`.
- Kích thước trang nằm ở block `temple:` của `config.<env>.yaml` (dev: `top_limit: 10`, `page_size: 20`); cả hai bắt buộc — giá trị 0 nghĩa là trang rỗng.
- Bảng `temples` phải tồn tại trước khi service chạy (`make db-up` áp `0005_temples.sql` và `0011_temple_content.sql`).

## Open questions (resolved during implementation)

Chưa có câu hỏi nào được người dùng trả lời trong phiên bản này; các câu còn mở nằm ở "Pending" phía trên.
