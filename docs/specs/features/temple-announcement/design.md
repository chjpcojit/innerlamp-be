# Design — temple-announcement

Version: 1.2
Last updated: 2026-10-14
Status: implemented (code + test pass; OKF concept chưa viết, xem `tasks.md`)
Module: `temple-announcement` (appService + backofficeService)

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Ba route đọc/công tắc ở `appService` (cần đăng nhập) và sáu route quản trị ở `backofficeService` (đăng, xem danh sách, sửa, thu hồi, tạo và liệt kê đơn vị phát hành), dùng chung Postgres. Thêm hai bảng (`temple_announcements`, `temple_issuing_bodies`) và một cột `users.receive_temple_announcements` trong migration `0036`. Hình dạng route đọc giống `temple-event` (đọc DB rồi trả), hình dạng route ghi giống `temple-suggestion` (token access + `RequireActiveSession` + role). Không có push, không có fan-out: người dùng **đọc** thông báo theo chùa hiện tại của mình.

## Layers & components

### appService (đọc + công tắc)

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/temple_announcement.go` (new) | `TempleAnnouncement{ID, TempleID, IssuingBody, Title, Body, PublishedAt, ExpiresAt}`, `IssuingBodyRef{ID, Name}`, `AnnouncementPage{Items, Total, Page, PageSize}`. Không tag JSON. |
| Domain / Repository interface | `services/appService/domain/temple_announcement.go` | `TempleAnnouncementRepository{ListActiveByTemple(ctx, templeID, now, page)}` và `AnnouncementPreferenceRepository{GetSettings(ctx, userID) (AnnouncementSettings{Receive, Temple *TempleRef}, error), SetReceive(ctx, userID, bool)}`. `GetSettings` đọc công tắc và chùa quy y (kèm tên chùa, JOIN) trong một SELECT; `ErrUserNotFound` khi không có hàng. Định nghĩa ở domain. |
| Domain / Mock | `services/appService/domain/mock_temple_announcement_repository.go` (new) | Mock viết tay cho test usecase. |
| Config | `services/appService/config/config.go` + `config.dev.yaml` | `TempleAnnouncementConfig{PageSize, MaxPageSize, RateLimit{Max, Window}}`, `Validate()` (`page_size ≥ 1`, `max_page_size ≥ page_size`, `max ≥ 1`, `window > 0`, như `profile.rate_limit`); `main.go` gọi `Validate()` lúc khởi động. Dev: 20 / 50 / 60 mỗi `1m`. Giá trị không nhạy cảm → config. |
| Usecase | `services/appService/usecase/temple_announcement_usecase.go` (new) | `ListForUser(ctx, userID, page, pageSize)`, `GetReceive`, `SetReceive`. Phụ thuộc hai repo trên (`AnnouncementPreferenceRepository` đọc luôn chùa quy y, không qua `UserRepository`) + `now func() time.Time` tiêm vào. |
| Repository | `services/appService/repository/postgres_temple_announcement_repository.go` (new) | SQL đếm + lấy trang thông báo active; đọc/ghi cột `receive`. |
| Delivery / Handler | `services/appService/delivery/http/temple_announcement_handler.go` (new) | Parse `page`/`page_size`, body công tắc (strict decode) → usecase → DTO. DTO + tag JSON chỉ ở đây. |
| Delivery / Errors | `services/appService/delivery/http/errors.go` | `ErrInvalidPage` → `400 INVALID_PAGE` (`details.field = "page"`, đã có sẵn từ `temple`). |
| Wiring | `services/appService/cmd/main.go` + `services/appService/cmd/temple_announcement_routes.go` | `main.go` dựng repo/usecase/handler và một bộ giới hạn riêng `deliveryhttp.NewUserRateLimit(cfg.TempleAnnouncement.RateLimit…)` (không dùng chung bộ của `profile`). Helper `registerTempleAnnouncementRoutes(mux, auth, limit, h)` trong `temple_announcement_routes.go` gắn ba route bọc `auth → limit` (test của `cmd` kiểm đúng wiring này). |

### backofficeService (đăng, sửa, thu hồi)

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/backofficeService/domain/temple_announcement.go` (new) | `Announcement` (đủ `RevokedAt`, `Source`, `CreatedBy`), `IssuingBody`, `AnnouncementState` (`ACTIVE`/`EXPIRED`/`REVOKED`, **suy ra** từ thời gian), input `NewAnnouncement`/`AnnouncementPatch`, `AnnouncementInputError{Field, Message, Kind}` (`Kind` là `ErrInvalidAnnouncement` hoặc `ErrInvalidIssuingBodyName`). Sentinel `ErrAnnouncementNotFound`, `ErrAnnouncementRevoked`, `ErrIssuingBodyExists`, `ErrInvalidIssuingBody`, `ErrInvalidIssuingBodyName`, `ErrInvalidAnnouncement`, `ErrInvalidPage`. `Announcement.StateAt(now)`: `REVOKED` nếu `revoked_at != nil` (thắng), ngược lại `ACTIVE` khi `now < expires_at`, còn lại `EXPIRED` (`expires_at == now` là `EXPIRED`); usecase điền `State` cho mọi announcement trả về. Dùng lại `ErrTempleNotFound` nếu đã có; nếu chưa, khai báo bản riêng (không import `appService/domain`). |
| Domain / Repository interface | `services/backofficeService/domain/temple_announcement.go` | `AnnouncementRepository{TempleExists, IssuingBodyBelongs, Insert, FindByID, Update, Revoke, ListByTemple}` và `IssuingBodyRepository{Insert, ListByTemple}`. |
| Config | `services/backofficeService/config/config.go` + `config.dev.yaml` | `TempleAnnouncementConfig{TitleMaxChars, BodyMaxChars, MaxValidityDays, IssuingBodyNameMaxChars, PageSize, MaxPageSize}`, `Validate()` (mọi giá trị ≥ 1, `max_page_size ≥ page_size`). Dev: 150 / 5000 (giả định) / 365 / 100 / 20 / 100. |
| Usecase | `services/backofficeService/usecase/temple_announcement_usecase.go` (new) | `Publish`, `Edit`, `Revoke`, `List`, `CreateIssuingBody`, `ListIssuingBodies`. Validation theo config, `Clock` tiêm vào. |
| Repository | `services/backofficeService/repository/postgres_temple_announcement_repository.go` (new) | SQL cho hai bảng; **repository sinh id** (`tann_` / `tib_` + `uuid.NewString()`), không phải usecase; đếm ký tự ở usecase (không ở SQL). |
| Delivery / Handler | `services/backofficeService/delivery/http/temple_announcement_handler.go` (new) | Sáu route; strict-decode JSON; DTO + tag JSON. `page`/`page_size` không phải số → `ErrInvalidPage`. |
| Delivery / Errors | `services/backofficeService/delivery/http/errors.go` | Map các sentinel mới sang `400 INVALID_ANNOUNCEMENT` (`details.field`), `400 INVALID_ISSUING_BODY` (`details.field = issuing_body_id`), `400 VALIDATION_ERROR` cho tên ban rỗng/quá dài (`details.field = name`, **không** phải `INVALID_ANNOUNCEMENT`), `400 INVALID_PAGE`, `404 TEMPLE_NOT_FOUND`, `404 ANNOUNCEMENT_NOT_FOUND`, `409 ANNOUNCEMENT_REVOKED`, `409 ISSUING_BODY_EXISTS` (`details.field = name`). |
| Wiring | `services/backofficeService/cmd/main.go` | Sáu route bọc helper `reviewer` có sẵn (admin + content_editor; viewer bị từ chối). |

### Chung

| Layer | File | Responsibility |
|---|---|---|
| Migration | `migrations/0036_temple_announcements.sql` | Hai bảng, index, cột `users.receive_temple_announcements`. `0035` là số dùng gần nhất. |
| API spec | `docs/specs/api/openapi.yaml` | Chín route + schema + mã lỗi mới (đã thêm; tag `TempleAnnouncements`). |
| ERD | `docs/specs/db/erd.mmd` (+ `erd.svg`) | Hai bảng mới + cột `users`. |

Dependency rule: delivery → usecase → domain; repository → domain; domain không import tầng ngoài. Hai service **không** import lẫn nhau; mỗi bên có entity riêng cho cùng bảng (như `temple_suggestions`).

## Mô hình dữ liệu

### `temple_issuing_bodies`

| Cột | Kiểu | Ghi chú |
|---|---|---|
| `id` | `VARCHAR(64)` PK | Do server sinh, tiền tố `tib_`. |
| `temple_id` | `VARCHAR(64)` NOT NULL, FK `temples(id)` `ON DELETE CASCADE` | Danh sách ban là **của từng chùa**. |
| `name` | `VARCHAR(100)` NOT NULL | Vd "Ban Trị sự", "Ban Hoằng pháp". |
| `created_at` | `TIMESTAMPTZ` NOT NULL DEFAULT `now()` | |

Unique index `(temple_id, lower(name))` → `409 ISSUING_BODY_EXISTS`. v1 không có đổi tên/tắt/xoá nên không cần cột `is_active`; TA22 "inactive" tạm thời luôn sai (xem Known gap).

### `temple_announcements`

| Cột | Kiểu | Ghi chú |
|---|---|---|
| `id` | `VARCHAR(64)` PK | Do server sinh, tiền tố `tann_`. |
| `temple_id` | `VARCHAR(64)` NOT NULL, FK `temples(id)` `ON DELETE CASCADE` | Chùa sở hữu. |
| `issuing_body_id` | `VARCHAR(64)` NOT NULL, FK `temple_issuing_bodies(id)` | `ON DELETE RESTRICT` (không có xoá ở v1). |
| `title` | `VARCHAR(150)` NOT NULL | Giới hạn thật nằm ở config; cột là trần an toàn. |
| `body` | `TEXT` NOT NULL | Văn bản thuần. |
| `source` | `VARCHAR(16)` NOT NULL DEFAULT `'admin'`, CHECK `IN ('admin','system')` | |
| `published_at` | `TIMESTAMPTZ` NOT NULL | = lúc tạo ở v1. |
| `expires_at` | `TIMESTAMPTZ` NOT NULL, CHECK `expires_at > published_at` | |
| `revoked_at` | `TIMESTAMPTZ` NULL | Đặt một lần khi thu hồi. |
| `created_by` | `VARCHAR(64)` NULL | Id tài khoản backoffice; `NULL` khi `source = system`. Không FK (hai service, chung bảng `admins` nhưng không ràng buộc chéo). |
| `created_at`, `updated_at` | `TIMESTAMPTZ` NOT NULL DEFAULT `now()` | |

Index `idx_temple_announcements_active (temple_id, published_at DESC, id) WHERE revoked_at IS NULL` — phục vụ danh sách người dùng; điều kiện `expires_at > now()` lọc thêm khi truy vấn (không đưa `now()` vào index).

### `users`

Thêm cột `receive_temple_announcements BOOLEAN NOT NULL DEFAULT TRUE`. Một công tắc theo user, dùng chung mọi thiết bị.

## Request flow (đọc, rút gọn)

```
GET /v1/users/me/temple-announcements
  └─ JWT middleware (access)
       └─ TempleAnnouncementHandler.List
            └─ TempleAnnouncementUsecase.ListForUser
                 ├─ UserRepository / AnnouncementPreferenceRepository  (receive, ordination_temple_id)
                 └─ TempleAnnouncementRepository.ListActiveByTemple    (count + page)
```

## Step-by-step Flow — Đọc thông báo của tôi (appService)

Điều kiện: token `access` hợp lệ (middleware). Hai lần đọc DB nối tiếp, không ghi.

| Step | Trigger / Pre-condition | Call | Success | Failure |
|---|---|---|---|---|
| R-1 | Handler nhận `page`, `page_size` | Parse; thiếu thì dùng `page = 1`, `page_size = config.PageSize`; `page_size` bị chặn ở `MaxPageSize` | Phân trang hợp lệ | Không phải số, `page < 1` hoặc `page_size < 1` (kể cả `0` gửi tường minh) → `400 INVALID_PAGE`, `details.field = "page"`, dừng |
| R-2 | R-1 ok | `GetSettings(userID)` → `receive`, chùa quy y `{id, name}` (một câu SELECT trên `users` LEFT JOIN `temples`) | Có hàng | Không có hàng → lỗi `500` (token hợp lệ mà user mất là bất thường) |
| R-3 | R-2 ok | Nếu `receive = false` → trả `{receive:false, temple:null, items:[], page}` | Dừng, không đọc bảng thông báo | — |
| R-4 | `receive = true` | Nếu `ordination_temple_id` rỗng → trả `{receive:true, temple:null, items:[]}` | Dừng | — |
| R-5 | Có chùa | `ListActiveByTemple(templeID, now, page)`: `COUNT(*)` rồi `SELECT … ORDER BY published_at DESC, id LIMIT/OFFSET`, kèm tên đơn vị phát hành (JOIN) | Trang kết quả + tổng | Lỗi DB → `500` |
| R-6 | R-5 ok | Handler map sang DTO; `temple` = `{id, name}` | `200` | — |

Quyết định: tắt công tắc (R-3) trả `temple:null` để client không phải đoán; `receive` luôn có mặt để UI vẽ công tắc dù danh sách rỗng. Hai lần đọc ở R-2/R-5 không cần transaction: đổi chùa ngay giữa hai bước chỉ làm trả danh sách của chùa cũ cho một lần gọi, chấp nhận được.

## Step-by-step Flow — Đăng thông báo (backofficeService)

Điều kiện: token admin `access`, `RequireActiveSession`, role `admin`/`content_editor`.

| Step | Trigger / Pre-condition | Call | Success | Failure |
|---|---|---|---|---|
| P-1 | Request vào | Middleware xác thực + role | Qua | `401` / `403` (AC TA18), dừng |
| P-2 | Handler decode body (strict) | `title`, `body`, `issuing_body_id`, `expires_at` (RFC 3339) | Parse được | JSON lạ/khoá lạ → `400 VALIDATION_ERROR` |
| P-3 | P-2 ok | Usecase kiểm trường: không rỗng sau `TrimSpace` (giá trị lưu không bị cắt), độ dài theo ký tự (rune) so với config, `expires_at > now` và `<= now + max_validity_days`; `expires_at` sai RFC 3339 cũng là lỗi này | Hợp lệ | `400 INVALID_ANNOUNCEMENT` + `details.field` (TA19–TA21) |
| P-4 | P-3 ok | `TempleExists(templeID)` | Có chùa | `404 TEMPLE_NOT_FOUND` |
| P-5 | P-4 ok | `IssuingBodyBelongs(issuingBodyID, templeID)` | Thuộc chùa | `400 INVALID_ISSUING_BODY` |
| P-6 | P-5 ok | `Insert` một dòng `source = admin`, `published_at = now`, `created_by = admin id`; sinh `id` | `201` + announcement | Lỗi DB → `500` |

P-4, P-5 và P-6 là ba lệnh riêng: FK của bảng vẫn chặn trường hợp chùa bị xoá giữa chừng (lỗi FK → map về `404 TEMPLE_NOT_FOUND`), nên không cần transaction. Sửa (`PATCH`) dùng lại P-1…P-5 rồi `FindByID` (kiểm `404`/`409 ANNOUNCEMENT_REVOKED`) và `Update`; thu hồi là `UPDATE … SET revoked_at = COALESCE(revoked_at, now())` rồi trả lại dòng (idempotent). Hai thao tác này không có flow riêng vì chỉ có một lệnh ghi.

## Deliberately-stubbed-for-now

- `source = system`: cột và CHECK có, **không có đường tạo**. Khi có bộ quét tự động, nó gọi cùng usecase publish với `source = system` và `created_by = NULL`.
- Đơn vị phát hành: chỉ tạo và liệt kê. Đổi tên, tắt, xoá chưa làm (Out of scope).

## Known gaps

- **Không có `is_active` cho đơn vị phát hành** (TA22 nhắc "inactive"): chừng nào chưa có tắt/xoá, điều kiện này không bao giờ xảy ra. Nếu thêm tắt/xoá, thêm cột và kiểm ở P-5.
- **Giới hạn tốc độ**: đã tách riêng (TA32, TA33). Giá trị dev mặc định đề xuất `max: 60`, `window: 1m`; chỉnh theo số liệu thật, chưa đo.
- **`body_max_chars = 5000`** là giả định (chưa có số liệu); chỉnh ở `config.dev.yaml` khi có dữ liệu thật.
- **Tên ban dài/rỗng** trả `VALIDATION_ERROR` (không có mã riêng) theo quyết định triển khai, khác `INVALID_ANNOUNCEMENT` của thông báo.
- **`OFFSET` phân trang**: đủ cho vài trăm thông báo còn hạn mỗi chùa; nếu lượng lớn thì đổi sang con trỏ (đã chốt `page`/`page_size` cho v1, 2026-10-14).
