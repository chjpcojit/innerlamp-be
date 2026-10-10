# Requirements — temple-suggestion

Version: 2.4
Last updated: 2026-10-10
Status: US-TSG-01 (AC1–AC8, appService) đã implement và kiểm chứng 2026-10-10; US-TSG-02..04 (backoffice) chưa làm. Đồng bộ với `SRS.md` v1.0 (đã duyệt). Không còn marker `⚠️ Not confirmed`. Hành động từ chối (AC28–AC29) làm ở phase sau.
Module: `content` (đề xuất từ client) + `backoffice` (duyệt)
Code: `services/appService/` (gửi đề xuất) + `services/backofficeService/` (duyệt, tạo chùa)

> See `changelog.md` in this folder for the version history of this spec. Nguồn nghiệp vụ: `SRS.md` (IEEE 29148) cùng thư mục.

Spec này được viết **trước khi có code** (Prospective mode); phần gửi đề xuất (US-TSG-01) đã có code, phần duyệt chưa. Nó mở rộng danh mục chùa của `docs/specs/features/temple/` (chỉ đọc, seed bằng tay) bằng một đường vào mới: người dùng app đề xuất một ngôi chùa chưa có, ban quản trị bổ sung thông tin rồi biến đề xuất thành chùa chính thức.

> Mã user story `US-TSG-01..04` do anh Đức tự chốt 2026-10-10 (làm một mình, không cần chờ product).

---

## US-TSG-01: Gửi đề xuất thêm chùa

## User Story

> Là một **người dùng app (không cần đăng nhập)**,
> Tôi muốn **gửi tên, tỉnh/thành phố và địa chỉ của một ngôi chùa chưa có trong app**,
> Để **ban quản trị xem xét và thêm chùa đó vào danh mục**.

## Acceptance Criteria (EARS notation)

1. WHEN a `POST /v1/temple-suggestions` request arrives with a valid JSON body `{name, province, address}`, THE SYSTEM SHALL store one suggestion with `status = "pending"` and return `201 Created` with `data` = `{id, status: "pending"}`. (Không cần `Authorization`; đi qua `webAPIKey` giống các route `/v1/auth/*`.)
2. WHEN `name`, `province` or `address` is missing, or empty after trimming leading/trailing whitespace, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field` naming the first offending field.
3. WHEN `name` is longer than 255 characters, `province` longer than 100, or `address` longer than 500 (đếm theo ký tự Unicode, sau khi trim), THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field`.
4. WHEN the body is not valid JSON, has an unknown field, a trailing value or exceeds the body limit, THE SYSTEM SHALL return `400` through the shared strict body decoder (`json-body-hardening`) without storing anything.
5. WHEN a suggestion is stored, THE SYSTEM SHALL keep the trimmed values exactly as sent (không tự sửa chính tả, không tự chuẩn hoá tên tỉnh) so ban quản trị thấy đúng điều người dùng gửi.
6. WHEN a suggestion repeats a temple that already exists or another pending suggestion, THE SYSTEM SHALL still accept it and return `201` — không dò trùng lúc gửi, để client không biết được kho chùa/đề xuất đang có gì.
   > Đã chốt 2026-10-10 (SRS FR-004): không dò trùng lúc gửi, ban quản trị tự lọc.
7. WHEN the same caller sends suggestions faster than the configured limit, THE SYSTEM SHALL return `429 RATE_LIMITED` and store nothing. Khoá là IP client, ngưỡng **5 đề xuất/IP/giờ** (config `temple_suggestion.rate_limit`); IP chỉ dùng tạm cho bộ đếm, không lưu cùng đề xuất.
   > Rủi ro đã chấp nhận cho v1: nhiều người dùng chung một IP (NAT nhà mạng, văn phòng) có thể bị giới hạn chung (SRS FR-003). Đã wire ở `appService` (`delivery/http/client_ip_rate_limit.go`); bộ đếm nằm trong bộ nhớ từng instance — xem Known gaps trong `design.md`.
8. THE SYSTEM SHALL NOT expose any client route that lists or reads suggestions; the client only gets `{id, status}` back once, at creation.

## US-TSG-02: Xem hàng chờ đề xuất

## User Story

> Là một **thành viên ban quản trị**,
> Tôi muốn **xem danh sách các đề xuất chùa đang chờ duyệt và mở từng đề xuất**,
> Để **biết người dùng đang đề xuất chùa nào và xử lý lần lượt**.

## Acceptance Criteria (EARS notation)

9. WHEN a `GET /v1/admin/temple-suggestions` request arrives with a valid backoffice access token of an allowed role, THE SYSTEM SHALL return `200 OK` with `data` = `{items, page, page_size, total, total_pages}`; each item has `id`, `name`, `province`, `address`, `status`, `created_at`.
10. WHEN `status` query is absent, THE SYSTEM SHALL list `pending` suggestions only; WHEN it is `pending`, `created` or `rejected`, THE SYSTEM SHALL list only that status; WHEN it is any other value, THE SYSTEM SHALL return `400 INVALID_STATUS` with `details.field = "status"`.
11. WHEN suggestions are listed, THE SYSTEM SHALL order them by `created_at` ASC then `id` ASC (cũ nhất trước — duyệt theo thứ tự gửi), so paging never repeats or skips a row.
12. WHEN `page` is absent or empty, THE SYSTEM SHALL serve page 1; WHEN it is not a positive integer, THE SYSTEM SHALL return `400 INVALID_PAGE`; WHEN it is beyond `total_pages`, THE SYSTEM SHALL return `200` with `items: []` — cùng quy tắc với `GET /v1/temples` (AC4–AC7 của `temple`).
13. WHEN a `GET /v1/admin/temple-suggestions/{id}` request arrives for an existing suggestion, THE SYSTEM SHALL return `200 OK` with the list-item fields plus `temple_id` (`null` trừ khi `created`), `reviewed_by` (id admin, `null` khi còn chờ), `reviewed_at` (`null` khi còn chờ) and `reject_reason` (`null` trừ khi `rejected` và có nhập lý do).
14. WHEN `{id}` is not a positive integer or matches no suggestion, THE SYSTEM SHALL return `404 TEMPLE_SUGGESTION_NOT_FOUND`.
15. WHEN the token is missing, malformed, a refresh token, or revoked, THE SYSTEM SHALL return `401`; WHEN it belongs to a role not allowed to review, THE SYSTEM SHALL return `403` — cùng envelope và middleware như `/v1/admin/*` hiện có. Role được phép: `admin` và `content_editor`; `viewer` bị `403` (chốt 2026-10-10, SRS FR-012). `RequireRole` hiện chỉ nhận đúng một role nên cần helper "một trong các role" (`RequireAnyRole`).

## US-TSG-03: Tạo chùa chính thức từ đề xuất

## User Story

> Là một **thành viên ban quản trị**,
> Tôi muốn **bổ sung thông tin còn thiếu (ảnh, loại hình, toạ độ, …) cho một đề xuất rồi xác nhận**,
> Để **đề xuất trở thành chùa chính thức trong danh mục và được đánh dấu "đã tạo chùa"**.

## Acceptance Criteria (EARS notation)

16. WHEN a `POST /v1/admin/temple-suggestions/{id}/create-temple` request arrives for a `pending` suggestion with a valid body, THE SYSTEM SHALL — in one transaction — insert one row into `temples` and mark the suggestion `created` with `temple_id`, `reviewed_by` (admin id từ token) and `reviewed_at` (now), then return `201 Created` with `data` = `{suggestion: {id, status, temple_id}, temple: <chi tiết chùa, cùng hình dạng GET /v1/temples/{id}>}`.
17. WHEN the request is processed, THE SYSTEM SHALL take every temple field from the request body only — nó không tự sao `name`/`address` từ đề xuất; form duyệt phía backoffice tự điền sẵn từ `GET .../{id}` rồi admin sửa.
18. WHEN `name`, `type`, `region`, `sect`, `established_year`, `heritage_rank`, `latitude`, `longitude` or `address` is missing/empty, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field`; các field còn lại (`image_url`, `thumbnail_url`, `aliases`, `founder`, `current_abbot`, `operating_hours`, `amenities`, `introduction`, `architecture_heritage`, `visiting_notes`) là tuỳ chọn và nhận mặc định như bảng `temples` (`''`, `{}`, `false`).
19. WHEN `type`, `region` or `heritage_rank` is not one of the values the `temples` table allows (`chk_temples_type`, `chk_temples_region`, `chk_temples_heritage_rank`), THE SYSTEM SHALL return `400 VALIDATION_ERROR`; WHEN `latitude` is outside −90..90, `longitude` outside −180..180 or `established_year` ≤ 0 or in the future, THE SYSTEM SHALL return `400 VALIDATION_ERROR` — trước khi chạm DB.
20. WHEN `image_url` or `thumbnail_url` is non-empty, THE SYSTEM SHALL accept it only if it starts with `/static/images/` or is an `http(s)` URL, otherwise `400 VALIDATION_ERROR`; WHEN empty or absent, THE SYSTEM SHALL store `''` (API đọc trả `null`, đúng AC11 của `temple`). Backend không nhận file ảnh ở v1.
21. WHEN a temple is created, THE SYSTEM SHALL set `heritage_priority` from `heritage_rank` with the same mapping as `domain.HeritagePriority` (không nhận từ client) and generate the temple `id` on the server.
   > Đã chốt 2026-10-10: id chùa mới dạng `temple_<region>_<NNN>` (NNN = số lớn nhất của region đó + 1, đệm 3 chữ số) để khớp 179 chùa seed.
   > Đã chốt 2026-10-10 (N7): backoffice giữ bản chép riêng của bảng ánh xạ `heritage_rank → heritage_priority`; một test đối chiếu với `appService/domain.HeritagePriority` chặn lệch.
22. WHEN the suggestion is already `created`, THE SYSTEM SHALL return `409 SUGGESTION_ALREADY_CREATED` and insert nothing; WHEN two requests race for the same pending suggestion, THE SYSTEM SHALL let exactly one succeed and answer the other `409`.
23. WHEN `{id}` is not a positive integer or matches no suggestion, THE SYSTEM SHALL return `404 TEMPLE_SUGGESTION_NOT_FOUND`.
24. WHEN any step after the validation fails (id collision, constraint violation, DB error), THE SYSTEM SHALL roll back so the suggestion stays `pending` and no `temples` row exists, and SHALL return `500 INTERNAL_ERROR` without leaking the cause.
25. WHEN a temple has been created, THE SYSTEM SHALL make it visible at once through `GET /v1/temples`, `/top` and `/{id}` (cùng bảng `temples`, không có cache) at the position its `heritage_priority`, `established_year`, `id` give it.
26. WHEN a suggestion is `created`, THE SYSTEM SHALL keep its original `name`/`province`/`address` unchanged and SHALL NOT allow a second `create-temple` or any edit of it.
27. WHEN the token/role rules of AC15 are violated, THE SYSTEM SHALL return `401`/`403` before reading the body.

## US-TSG-04: Từ chối đề xuất và xem lại đề xuất bị từ chối

## User Story

> Là một **thành viên ban quản trị**,
> Tôi muốn **từ chối đề xuất rác/trùng kèm lý do, và (nếu là admin) xem lại rồi vẫn tạo chùa từ đề xuất đã từ chối**,
> Để **hàng chờ gọn mà không mất quyền sửa quyết định**.

## Acceptance Criteria (EARS notation)

28. WHEN staff of an allowed role send a reject action for a `pending` suggestion with an optional `reason` of at most 500 characters, THE SYSTEM SHALL mark it `rejected` and record `reviewed_by`, `reviewed_at` and `reject_reason` (`null` khi không nhập lý do), so it leaves the pending queue. **Làm ở phase sau**; hợp đồng API (route, body) thiết kế khi đó.
29. WHEN the `reason` is longer than 500 characters, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "reason"` and keep the suggestion `pending`; WHEN the suggestion is already `created` or `rejected`, THE SYSTEM SHALL return `409 SUGGESTION_ALREADY_HANDLED` and change nothing.
30. WHEN an admin submits a valid temple form for a `rejected` suggestion through the create-temple route, THE SYSTEM SHALL create the temple and mark the suggestion `created` as in AC16, clearing `reject_reason` (lý do cũ không còn được lưu); WHEN the caller is `content_editor`, THE SYSTEM SHALL return `403` and leave the suggestion `rejected`.
31. WHEN `status=rejected` is listed or a rejected suggestion is opened, THE SYSTEM SHALL show it with its reason and reviewer like any other state (AC10, AC13) — đó là cách admin "xem lại" ở trang admin.

## API contract (confirmed surface for this feature)

Route `POST /v1/temple-suggestions` đã có trong `docs/specs/api/openapi.yaml` (và bản `web/docs/public/`). Ba route `/v1/admin/temple-suggestions…` còn là đề xuất của spec, chốt khi implement (`tasks.md` Phase 4).

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| POST | `/v1/temple-suggestions` | public (+ `webAPIKey`) | 201 `{id, status}` | 400 `VALIDATION_ERROR`, 429 `RATE_LIMITED`, 500 |
| GET | `/v1/admin/temple-suggestions` | backoffice token (admin, content_editor) | 200 `{items, page, page_size, total, total_pages}` | 400 `INVALID_STATUS`/`INVALID_PAGE`, 401, 403 |
| GET | `/v1/admin/temple-suggestions/{id}` | backoffice token (admin, content_editor) | 200 chi tiết | 401, 403, 404 `TEMPLE_SUGGESTION_NOT_FOUND` |
| POST | `/v1/admin/temple-suggestions/{id}/create-temple` | backoffice token (admin, content_editor; rejected → admin only) | 201 `{suggestion, temple}` | 400 `VALIDATION_ERROR`, 401, 403, 404, 409 `SUGGESTION_ALREADY_CREATED` |
| POST | `/v1/admin/temple-suggestions/{id}/reject` *(phase sau, route tạm)* | backoffice token (admin, content_editor) | 200 `{suggestion}` | 400 `VALIDATION_ERROR`, 401, 403, 404, 409 `SUGGESTION_ALREADY_HANDLED` |

## Out of scope (this feature boundary, not tracked here)

- Giao diện trang duyệt của backoffice (web) — repo này chỉ có API; form tự điền sẵn từ `GET .../{id}`.
- Upload file ảnh (GCS/`pkg/storage`) — ảnh nhập bằng đường dẫn; upload thật theo dõi như `docs/tech-debt/memorial/TD-001`.
- Sửa/xoá chùa đã có và seed hàng loạt — thuộc `temple` (`cmd/seedtemples`).
- Thông báo cho người gửi khi chùa được tạo (người gửi không có tài khoản, không có kênh liên lạc).

## Pending (not yet implemented)

- Phần duyệt ở backoffice (US-TSG-02, US-TSG-03, AC9–AC27) — `tasks.md` Phase 3–5. US-TSG-01 (AC1–AC8) đã xong.
- Hành động từ chối (US-TSG-04, AC28–AC29) — cần nhưng làm ở phase sau theo quyết định 2026-10-10; schema và AC30–AC31 (xem lại + tạo chùa từ đề xuất bị từ chối) đi cùng bản đầu vì chỉ là thêm giá trị `rejected` vào trạng thái. `tasks.md` Phase 6.

## Ghi chú kỹ thuật liên quan

- Schema mới: `temple_suggestions` (`migrations/0022_temple_suggestions.sql`, đã có cùng `erd.mmd`/`erd.svg`/`migrations/README.md`) — xem `design.md`. Đề xuất giữ vĩnh viễn (không có job dọn) và không lưu IP người gửi.
- Hai service dùng chung một Postgres (bảng `admins` của backoffice nằm trong `migrations/` chung), nên backoffice ghi thẳng vào `temples` mà không cần gọi qua appService.
- Cột `temples.image_url`/`thumbnail_url` lưu đường dẫn tương đối (`0021_relative_image_urls.sql`); API đọc tự ghép `server.public_base_url`. File ảnh phải được đặt vào `static/images/...` bằng tay/CI trước khi admin nhập đường dẫn — backend không kiểm tra file có tồn tại.
- Mọi lỗi trả trong envelope chung `{"success": false, "error": {code, message, type}, "details"?}`.
- Backoffice: `RequireRole` chỉ so khớp đúng một role (`guard(func(r) bool, ...)`); cần helper `RequireAnyRole` (một trong các role).

## Open questions (resolved during implementation)

- ~~Ai được gửi đề xuất?~~ → công khai, không cần đăng nhập. Resolved 2026-10-10.
- ~~Admin bổ sung ảnh bằng cách nào?~~ → nhập đường dẫn ảnh (`/static/images/...`), chưa upload file. Resolved 2026-10-10.
- ~~Chùa chính thức tạo thế nào khi `temples` có nhiều cột NOT NULL?~~ → admin điền đủ form; submit = INSERT `temples` + đánh dấu đề xuất `created` trong một transaction. Resolved 2026-10-10.
- ~~Làm spec hay code luôn?~~ → spec trước, anh duyệt rồi mới code TDD. Resolved 2026-10-10.

- ~~N1 — Khoá và ngưỡng rate-limit?~~ → theo IP, 5 đề xuất/IP/giờ; rủi ro IP dùng chung được chấp nhận ở v1. Resolved 2026-10-10.
- ~~N2 — `province` tự do hay danh sách cố định?~~ → chuỗi tự do ≤100 ký tự. Resolved 2026-10-10.
- ~~N3 — Dò trùng lúc gửi?~~ → không; ban quản trị tự lọc. Resolved 2026-10-10.
- ~~N4 — Có trạng thái `rejected`?~~ → có, lý do tuỳ chọn ≤500 ký tự; admin xem lại và vẫn tạo chùa được; route từ chối làm phase sau. Resolved 2026-10-10.
- ~~N5 — Role được duyệt?~~ → `admin` + `content_editor`; chuyển đề xuất đã từ chối thành chùa chỉ `admin`. Resolved 2026-10-10.
- ~~Thời hạn lưu / lưu IP?~~ → lưu vĩnh viễn; không lưu IP. Resolved 2026-10-10.

- ~~N6 — Dạng id chùa mới?~~ → `temple_<region>_<NNN>`. Resolved 2026-10-10.
- ~~Mã user story?~~ → `US-TSG-01..04`, anh Đức tự chốt. Resolved 2026-10-10.

- ~~N7 — Bảng ánh xạ priority nằm đâu?~~ → chép sang backoffice kèm test đối chiếu (không đưa vào `pkg/`). Resolved 2026-10-10.

Không còn câu hỏi mở.
