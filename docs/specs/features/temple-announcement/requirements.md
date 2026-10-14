# Requirements — temple-announcement

Version: 1.2
Last updated: 2026-10-14
Status: implemented (code + test pass; OKF concept chưa viết, xem `tasks.md`)
Module: `temple-announcement`
Code: `services/appService/` (đọc + công tắc) và `services/backofficeService/` (đăng, sửa, thu hồi)

> See `changelog.md` in this folder for the version history of this spec.
> Nguồn nghiệp vụ: `SRS.md` v0.2 (IEEE 29148) trong thư mục này — mọi `FR-*`/`NFR-*` ở đó được truy vết bằng cột "SRS" của bảng AC bên dưới.

## User Story

### US-TAN-001 — Thông báo của chùa tôi quy y

> Là một **người dùng đã đăng nhập (web và app)** đã có chùa quy y,
> Tôi muốn **xem các thông báo mà chùa đó đăng (ví dụ "ghi danh khoá tu mùa hè còn 10 ngày") và tắt/bật việc nhận chúng**,
> Để **không bỏ lỡ hạn chót hay thông tin của chùa, và không bị làm phiền khi tôi không muốn**.

### US-TAN-002 — Chùa đăng thông báo (backoffice)

> Là một **quản trị viên backoffice** (`admin` hoặc `content_editor`),
> Tôi muốn **đăng, sửa và thu hồi thông báo của một chùa, ghi rõ do ban nào phát hành**,
> Để **mọi người đang quy y tại chùa đó thấy thông tin chính thống và còn hiệu lực**.

Thông báo là **dữ liệu của chùa** (có hạn), hiển thị trong app/web. Nó **không** là sự kiện chùa (`temple-event`), **không** là lịch tu tập (`temple-schedule`) và **không** là push notification của hệ điều hành (quyết định 2026-10-14).

## Acceptance Criteria (EARS notation)

Đánh số `TA1…`, mỗi dòng truy vết về `FR-*`/`NFR-*` của `SRS.md`.

### Đường đọc (appService, cần JWT `access`)

| # | Acceptance criterion | SRS |
|---|---|---|
| TA1 | WHEN a request for the caller's announcements carries a valid access token, the caller has an ordination temple and `receive` is on, THE SYSTEM SHALL return `200 OK` with `receive = true`, the temple and the active announcements of that temple, newest `published_at` first. | FR-001 |
| TA2 | WHEN the request has no valid access token (không có `Authorization`, token hết hạn, hoặc không phải loại `access`), THE SYSTEM SHALL reject it through the existing JWT middleware and return no announcement. | FR-001 |
| TA3 | THE SYSTEM SHALL decide the temple from the caller's current `users.ordination_temple_id` at read time, and SHALL NOT accept any temple id from the request. | FR-001, FR-003 |
| TA4 | THE SYSTEM SHALL return only announcements whose `temple_id` equals the caller's temple, so announcements of another temple are never returned. | FR-001 |
| TA5 | WHEN the caller has no ordination temple, THE SYSTEM SHALL return `200 OK` with `temple = null`, empty `items` and no error. | FR-002 |
| TA6 | WHEN the caller changes or clears `ordination_temple_id`, THE SYSTEM SHALL return, from the next read, only the active announcements of the new temple (or none), and SHALL keep no per-user copy of announcements. | FR-003 |
| TA7 | WHEN `receive` is off, THE SYSTEM SHALL return `200 OK` with `receive = false` and empty `items` regardless of the temple. | FR-004 |
| TA8 | THE SYSTEM SHALL treat an announcement as active only when `revoked_at` is null, `published_at` is not in the future and `expires_at` is after the current moment; WHEN `expires_at` has passed or the announcement is revoked, THE SYSTEM SHALL NOT return it to the user. | FR-005, FR-007 |
| TA9 | WHEN an announcement is returned, THE SYSTEM SHALL include `id`, `title`, `body`, `published_at`, `expires_at` and `issuing_body` (`id`, `name`). | FR-001, FR-008 |
| TA10 | THE SYSTEM SHALL return announcements in pages of `temple_announcement.page_size` by default and at most `temple_announcement.max_page_size` (a larger `page_size` is capped, not rejected), with the total of active announcements, so two requests with the same state give the same order (`published_at` desc, then `id`). | FR-001, NFR-001 |
| TA11 | THE SYSTEM SHALL NOT keep any read/unread state and SHALL NOT return an unread count. | FR-001 |

### Công tắc nhận (appService, cần JWT `access`)

| # | Acceptance criterion | SRS |
|---|---|---|
| TA12 | THE SYSTEM SHALL keep one `receive` flag per user in `users`, `true` by default, shared by every device of that user. | FR-004 |
| TA13 | WHEN a request to set the flag carries a valid access token and a JSON boolean `receive`, THE SYSTEM SHALL store it and return `200 OK` with the stored value. | FR-004 |
| TA14 | WHEN `receive` is missing, is not a boolean, or the body has an unknown key, THE SYSTEM SHALL return `400 VALIDATION_ERROR` (với `details.field = "receive"` khi thiếu) and SHALL NOT change the flag. | FR-004 |
| TA15 | WHEN the caller changes or clears the ordination temple, THE SYSTEM SHALL leave the `receive` flag unchanged. | FR-004 |
| TA16 | WHEN a request to read the flag carries a valid access token, THE SYSTEM SHALL return `200 OK` with the stored `receive` value. | FR-004 |

### Đường ghi (backofficeService, token admin `access` + role)

| # | Acceptance criterion | SRS |
|---|---|---|
| TA17 | WHEN a backoffice account with role `admin` or `content_editor` publishes an announcement with an existing temple, an issuing body of that temple, a `title`, a `body` and a future `expires_at`, THE SYSTEM SHALL create it with `source = admin`, `published_at` = now and `created_by` = the account, and return `201`. | FR-006 |
| TA18 | WHEN the caller has role `viewer`, or has no valid backoffice session, THE SYSTEM SHALL return `403` / `401` through the existing backoffice middleware and write nothing. | FR-006 |
| TA19 | WHEN `title`, `body`, `issuing_body_id` or `expires_at` is missing or blank, THE SYSTEM SHALL return `400 INVALID_ANNOUNCEMENT` with `details.field` naming it and SHALL NOT write a row. | FR-006, FR-008 |
| TA20 | WHEN `title` is longer than `temple_announcement.title_max_chars` (dev 150) or `body` is longer than `body_max_chars` (dev 5000; giả định, chưa có số liệu thật) (đếm ký tự, không đếm byte), THE SYSTEM SHALL return `400 INVALID_ANNOUNCEMENT` with `details.field`. | FR-006 |
| TA21 | WHEN `expires_at` is not after the current moment, or is later than `temple_announcement.max_validity_days` from now, THE SYSTEM SHALL return `400 INVALID_ANNOUNCEMENT` with `details.field = "expires_at"`. | FR-006 |
| TA22 | WHEN the temple does not exist, THE SYSTEM SHALL return `404 TEMPLE_NOT_FOUND`; WHEN `issuing_body_id` does not belong to that temple, THE SYSTEM SHALL return `400 INVALID_ISSUING_BODY` (`details.field = "issuing_body_id"`; v1 không có trạng thái inactive). Neither case writes a row. | FR-006, FR-008 |
| TA23 | WHEN an admin edits an announcement that is not revoked, THE SYSTEM SHALL update only the given fields among `title`, `body`, `issuing_body_id`, `expires_at` under the same validation as TA19–TA22, keep `published_at` and `source`, and return the full announcement. | FR-007 |
| TA24 | WHEN an admin edits a revoked announcement, THE SYSTEM SHALL return `409 ANNOUNCEMENT_REVOKED` and change nothing. | FR-007 |
| TA25 | WHEN an admin revokes an announcement, THE SYSTEM SHALL set `revoked_at` once and return `200`; WHEN it is already revoked, THE SYSTEM SHALL return `200` with the original `revoked_at` (idempotent). | FR-007 |
| TA26 | WHEN no announcement has the given id, THE SYSTEM SHALL return `404 ANNOUNCEMENT_NOT_FOUND`. | FR-007 |
| TA27 | WHEN an admin lists the announcements of a temple, THE SYSTEM SHALL return all of them — active, expired and revoked — each with a derived `state` (`ACTIVE`, `EXPIRED`, `REVOKED`), newest first and paged. | FR-005, FR-007 |
| TA28 | WHEN an admin creates an issuing body with a non-blank `name` (tối đa `issuing_body_name_max_chars`) for an existing temple, THE SYSTEM SHALL create it and return `201`; WHEN the name already exists in that temple (không phân biệt hoa thường), THE SYSTEM SHALL return `409 ISSUING_BODY_EXISTS`. | FR-008 |
| TA29 | WHEN an admin or a user reads announcements, THE SYSTEM SHALL return the issuing body's `name` as stored, so renaming is out of v1 and an announcement never shows a different name from the one chosen. | FR-008 |
| TA30 | THE SYSTEM SHALL store `source` as `admin` or `system` (CHECK), and SHALL record `admin` for every announcement created through TA17. | FR-009 |
| TA31 | THE SYSTEM SHALL keep announcements, issuing bodies and the `receive` flag in database rows and the limits in config (không hằng trong code). | NFR-002, FR-009 |
| TA32 | THE SYSTEM SHALL apply to the three user routes of this feature a per-account request limit of its own (`temple_announcement.rate_limit.{max, window}`), independent of the `/v1/users/me*` profile limit, so reading announcements neither consumes nor is blocked by the profile budget. | NFR-003 |
| TA33 | WHEN an account exceeds that limit, THE SYSTEM SHALL return `429 RATE_LIMITED` and SHALL NOT read or change anything. | NFR-003 |
| TA34 | WHEN `page` or `page_size` is not an integer or is below 1 on the user list, or `page` is below 1 / `page_size` is negative / either is not an integer on the admin list, THE SYSTEM SHALL return `400 INVALID_PAGE` (`details.field = "page"`). Trên danh sách admin, `page_size` bỏ trống hoặc `0` là mặc định; lớn hơn `max_page_size` thì bị chặn. | FR-001, NFR-001 |
| TA35 | WHEN an issuing body `name` is blank or longer than `issuing_body_name_max_chars`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "name"` (không dùng `INVALID_ANNOUNCEMENT`) and SHALL NOT write a row. | FR-008 |
| TA36 | THE SYSTEM SHALL derive `state` at read time as: `REVOKED` when `revoked_at` is set (thắng mọi trạng thái khác), else `ACTIVE` strictly before `expires_at`, else `EXPIRED` (kể cả khi `expires_at` đúng bằng thời điểm hiện tại). | FR-005, FR-007 |

## API contract (surface dự kiến cho feature này)

| Method | Path | Service | Auth | Success | Failure codes |
|---|---|---|---|---|---|
| GET | `/v1/users/me/temple-announcements?page=&page_size=` | appService | JWT `access` | 200 `{receive, temple, items[], page}` | 400 `INVALID_PAGE`, 401, 429 `RATE_LIMITED`, 500 |
| GET | `/v1/users/me/temple-announcement-settings` | appService | JWT `access` | 200 `{receive}` | 401, 429 `RATE_LIMITED`, 500 |
| PUT | `/v1/users/me/temple-announcement-settings` | appService | JWT `access` | 200 `{receive}` | 400 `VALIDATION_ERROR`, 401, 429 `RATE_LIMITED`, 500 |
| POST | `/v1/admin/temples/{id}/announcements` | backofficeService | admin/content_editor | 201 announcement | 400 `INVALID_ANNOUNCEMENT`, `INVALID_ISSUING_BODY`, 401, 403, 404 `TEMPLE_NOT_FOUND`, 500 |
| GET | `/v1/admin/temples/{id}/announcements?page=&page_size=` | backofficeService | admin/content_editor | 200 `{items[], page}` | 400 `INVALID_PAGE`, 401, 403, 404 `TEMPLE_NOT_FOUND`, 500 |
| PATCH | `/v1/admin/temple-announcements/{id}` | backofficeService | admin/content_editor | 200 announcement | 400 `INVALID_ANNOUNCEMENT`/`INVALID_ISSUING_BODY`/`VALIDATION_ERROR`, 401, 403, 404 `ANNOUNCEMENT_NOT_FOUND`, 409 `ANNOUNCEMENT_REVOKED`, 500 |
| POST | `/v1/admin/temple-announcements/{id}/revoke` | backofficeService | admin/content_editor | 200 announcement | 401, 403, 404 `ANNOUNCEMENT_NOT_FOUND`, 500 |
| POST | `/v1/admin/temples/{id}/issuing-bodies` | backofficeService | admin/content_editor | 201 issuing body | 400 `VALIDATION_ERROR` (`details.field = name`), 401, 403, 404 `TEMPLE_NOT_FOUND`, 409 `ISSUING_BODY_EXISTS`, 500 |
| GET | `/v1/admin/temples/{id}/issuing-bodies` | backofficeService | admin/content_editor | 200 `{items[]}` | 401, 403, 404 `TEMPLE_NOT_FOUND`, 500 |

Chín route đã có trong `docs/specs/api/openapi.yaml` (tag `TempleAnnouncements`) và `api-contract.md`. Hình dạng response chi tiết ở `design.md`.

## Out of scope (this feature boundary, not tracked here)

- Push OS (FCM/APNs/web push), SMS, email: thông báo chỉ là dữ liệu hiển thị trong app/web.
- Đọc / chưa đọc, huy hiệu số chưa đọc (quyết định 2026-10-14).
- Công tắc theo loại hoặc theo chùa; chỉ có một công tắc chung theo user.
- Hẹn giờ đăng (`published_at` luôn là lúc tạo).
- Hệ thống tự quét và tự tạo thông báo: chỉ chừa `source = system`.
- Admin riêng của từng chùa và phân quyền theo chùa (v1: tài khoản backoffice toàn hệ thống).
- Đổi tên, tắt hoặc xoá đơn vị phát hành; thông báo có ảnh/file/liên kết.
- Xoá cứng thông báo: chỉ thu hồi.

## Pending (not yet implemented)

- OKF concept (`docs/knowledge/`): chưa viết; chỉ viết sau khi con người kiểm chứng (xem `tasks.md` Phase 8).
- Chưa kiểm bằng client thật (web/app); `erd.svg` chưa dựng lại (không có `mmdc` trong repo).

Không còn `⚠️ Not confirmed` nào mở (v1.2).

Giả định ghi lại ở v1.2: `body_max_chars = 5000` (dev) là con số chọn tạm, chưa đo.

## Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| "Chùa tu tập" là gì? | `users.ordination_temple_id`; "rời chùa" = xoá giá trị này. | 2026-10-14 |
| Thông báo là push OS hay dữ liệu của chùa? | Dữ liệu của chùa, có hạn; không push. | 2026-10-14 |
| Công tắc tắt nghĩa là gì? | Ẩn hoàn toàn danh sách (TA7). | 2026-10-14 |
| Hạn dùng bắt buộc? | Bắt buộc, tối đa `max_validity_days` = 365 (config) (TA21). | 2026-10-14 |
| Ai được đăng? | Tài khoản backoffice toàn hệ thống (TA17); xem ⚠️ về `content_editor`. | 2026-10-14 |
| Hẹn giờ đăng? | Không; đăng là hiện ngay. | 2026-10-14 |
| Đơn vị phát hành | Chọn từ danh sách có quản lý theo từng chùa (TA22, TA28). | 2026-10-14 |
| Đọc / chưa đọc | Không có (TA11). | 2026-10-14 |
| Quy mô theo dõi | Chưa biết; không fan-out vì thông báo được đọc theo yêu cầu, không đẩy. | 2026-10-14 |
| `content_editor` có được đăng như `admin`? | Có; `viewer` vẫn bị từ chối (TA17, TA18). | 2026-10-14 |
| Hình dạng phân trang | `page`/`page_size` + `total`, theo `temple-suggestion`. | 2026-10-14 |
| Ngưỡng thời gian phản hồi danh sách | p95 < 500 ms với 200 thông báo còn hạn (NFR-001). | 2026-10-14 |
| Giới hạn tốc độ | Tách riêng khỏi nhóm `/v1/users/me*` (TA32, TA33). | 2026-10-14 |

## Ghi chú kỹ thuật liên quan

- Hai service dùng chung Postgres (như `temple-suggestion`): `backofficeService` ghi, `appService` đọc. Migration `0036` sở hữu bảng mới; cột `users.receive_temple_announcements` thuộc cùng migration.
- Route đọc nằm dưới `/v1/users/me/…` nhưng **không** dùng giới hạn 10 req/phút của nhóm hồ sơ (quyết định 2026-10-14): có bộ giới hạn theo account riêng, cấu hình ở `temple_announcement.rate_limit`, tạo bằng `deliveryhttp.NewUserRateLimit` như `profile`. Cần kiểm khi wiring rằng route không bị bọc luôn bộ giới hạn của profile.
- Quyền xem suy ra từ `users.ordination_temple_id` đọc tại thời điểm gọi (TA3), không lấy từ token hay tham số, nên đổi chùa có hiệu lực ở lần đọc kế tiếp.
- `temples` không có cột ẩn (`is_active`); "chùa bị gỡ" quy về xoá dòng và `ON DELETE CASCADE` dọn thông báo, đơn vị phát hành; `ordination_temple_id` của user theo cơ chế sẵn có của `profile`.
