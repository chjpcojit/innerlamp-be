# Design — memorial

Version: 3.0
Last updated: 2026-10-14
Status: v1 (5 route CRUD, Step C-1..C-4 và D-1/D-2/D-4) đã implement và kiểm chứng 2026-10-07; **phần ảnh/GCS (⏸ Phase 6) hoãn, nợ ở `docs/tech-debt/memorial/TD-001`**. Còn marker `⚠️ Not confirmed` ở bucket GCS (Step P-2, chỉ chặn Phase 6). v3.0: thêm test vector (Phase 7, chưa làm); các flow không đổi.
Module: `memorial`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Bảy route `/v1/memorials/*` (v1 làm **5 route CRUD**; hai route ảnh ⏸ hoãn tới Phase 6) đều **user-scoped**, bọc `authMiddleware.AuthenticateFunc` (`RequireTokenType: "access"`) như `POST /v1/auth/logout`; `user_id` lấy từ context do middleware gắn (`pkg/jwt/context`), không bao giờ từ request. Feature chạm đủ bốn layer của `services/appService` (`domain` → `usecase` → `repository` / `delivery/http`, nối dây ở `cmd/main.go`), sở hữu migration `0018_memorials.sql`, và khi làm phần ảnh sẽ là **feature đầu tiên dùng `pkg/storage` (GCS) thật** — hiện repo chưa có chỗ nào gọi nó. Cho tới Phase 6, v1 **không import `pkg/storage`**, không dựng storage client, và `photo_url` luôn `null`; cột `photo_object_key` vẫn có sẵn (NULL) để Phase 6 khỏi cần migration. Không có logic lịch âm hay lập lịch nhắc ở BE lúc chạy (product spec §2.7, xem `requirements.md` — Ràng buộc từ product spec). Phần duy nhất chạm lịch âm là test vector: `domain/lunarcal` là nguồn chuẩn của BR-MEM-10 nên phải chạy `docs/product/memorial/rules/death-date-conversion.json` để vector và bộ chuyển đổi không trôi khỏi nhau.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/memorial.go` | `Memorial` (id, userID, fullName, `Relationship`, `DeathDate *time.Time` (ngày dương, `nil` khi năm không rõ), `DeathCalendar`, `DeathDay/DeathMonth *int` + `DeathLeapMonth *bool` (chỉ khi `DeathDate == nil`), `Note *string`, `PhotoObjectKey *string`, `Reminder *Reminder`, addToOSReminder, timestamps), `Reminder{DaysBefore int, Time string, Timezone string}`. Hằng enum `Relationship` (10 giá trị) và `DeathCalendar`. Không JSON/db tag. |
| Domain / Errors | `domain/memorial.go` | `ErrMemorialNotFound`, `ErrMemorialLimitReached`, `ErrPhotoTooLarge`, `ErrPhotoUnsupportedType`. `ErrMemorialInvalid` + `*MemorialValidationError{Field, Message}` (có `Unwrap` về sentinel) — handler đọc `Field` để trả `400 VALIDATION_ERROR` kèm `details.field`; không dùng thẻ `validator` vì các luật phụ thuộc nhau (hai dạng ngày mất, reminder). |
| Domain / Repository interface | `domain/memorial.go` | `MemorialRepository`: `Create(ctx, m, max)` (đếm + chèn trong một tx), `ListByUser`, `FindByIDAndUser`, `Update`, `Delete` (chỉ trả `error` ở v1). ⏸ Phase 6: `Delete` đổi sang trả `photo_object_key` cũ và thêm `SetPhotoKey(ctx, id, userID, key)` (trả key cũ). Mọi method một mục nhận cả `id` **và** `userID`. |
| Domain / Storage port ⏸ Phase 6 | `domain/memorial.go` | `MemorialPhotoStore`: `Put(ctx, key, r, contentType)`, `Delete(ctx, key)`, `SignedURL(ctx, key, ttl)`. Định nghĩa ở domain để usecase không import `pkg/storage`. |
| Domain / Mock | `domain/mock_memorial_repository.go` (v1), `domain/mock_memorial_photo_store.go` (⏸ Phase 6) | Mock viết tay cho test usecase. |
| Usecase | `services/appService/usecase/memorial_usecase.go` | v1: `Create`, `List`, `Get`, `Update`, `Delete`; ⏸ Phase 6: `SetPhoto`, `RemovePhoto`. Validation AC3–AC10 gồm AC6a/6b (nhánh ngày đầy đủ vs ngày+tháng theo lịch đã nhập, `time.LoadLocation`, "hôm nay" theo `Asia/Ho_Chi_Minh`), điều phối thứ tự DB↔GCS, build `photo_url` qua `SignedURL`. |
| Repository | `services/appService/repository/postgres_memorial_repository.go` | SQL bảng `memorials`; `Create` chạy một tx (khoá `users`, đếm, chèn); mọi `WHERE` có `user_id`; `NULL` ↔ `nil` cho `note`/`photo_object_key`/cột nhắc; map `reminder_*` ↔ `*Reminder`. |
| Storage adapter ⏸ Phase 6 | `services/appService/repository/gcs_memorial_photo_store.go` | Bọc `pkg/storage` (backend `gcs`): key `memorials/<userID>/<memorialID>/<uuid>.<ext>`, bucket từ config, `SignedURL` GET 15 phút. Constructor không mở kết nối (quy ước `pkg/`). |
| Delivery / Handler | `services/appService/delivery/http/memorial_handler.go` | v1: `Create`, `List`, `Get`, `Update`, `Delete`; ⏸ Phase 6: `UploadPhoto`, `DeletePhoto`; lấy `userID` từ context; `DecodeJSONBodyStrict`; ảnh: `http.MaxBytesReader` 5 MB + multipart + sniff `http.DetectContentType`. |
| Delivery / DTO | `delivery/http/memorial_handler.go` | `memorialRequest`, `reminderDTO`, `memorialResponse`, `memorialListResponse` — JSON tag chỉ ở đây; `death_date` định dạng `2006-01-02`; `items` luôn `[]` không `null`. |
| Delivery / Error map | `delivery/http/errors.go` | `ErrMemorialNotFound` → `404 MEMORIAL_NOT_FOUND`; `ErrMemorialLimitReached` → `409 MEMORIAL_LIMIT_REACHED`; ⏸ Phase 6: `ErrPhotoTooLarge` → `413 PHOTO_TOO_LARGE`; `ErrPhotoUnsupportedType` → `415 PHOTO_UNSUPPORTED_TYPE`. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | v1: `MemorialConfig{MaxPerUser (100)}`; ⏸ Phase 6 thêm `PhotoMaxBytes (5MB)`, `PhotoURLTTL (15m)`, `PhotoBucket`, `PhotoPrefix` (bucket: ⚠️ xem Step P-2). `StorageConfig` đã có `backend: gcs`, `default_bucket`. |
| Wiring | `services/appService/cmd/main.go` | Dựng repo → usecase → handler; đăng ký 5 route CRUD (v1), **mỗi route bọc `authMiddleware.AuthenticateFunc`**. ⏸ Phase 6: dựng storage client (`Connect` fail-fast lúc khởi động) và đăng ký thêm 2 route ảnh, cũng bọc middleware. Phải kiểm tra thật sự đã bọc, không chỉ ghi là cần. |
| Migration | `migrations/0018_memorials.sql` | Bảng `memorials` (bên dưới) + index `idx_memorials_user (user_id, created_at DESC, id DESC)`. |
| API spec | `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) | 5 route (v1) + 2 route ảnh khi Phase 6, security Bearer, schema `Memorial`/`MemorialRequest`; cập nhật cùng thay đổi handler. |
| DB spec | `docs/specs/db/erd.mmd`, `erd.svg` | Thêm bảng `memorials` + quan hệ `users 1—N memorials`. |
| Product vectors (BR-MEM-03, 04, 10) | `services/appService/domain/lunarcal/memorial_vectors_test.go` | Đọc `docs/product/memorial/rules/death-date-conversion.json` từ repo (đường dẫn tương đối lên gốc repo, không chép file); mỗi case: `input.solar` → `SolarToLunar` so `expected.lunar`; `input.lunar` → `LunarToSolar` so `expected.solar`, hoặc phải trả lỗi khi `expected.valid = false`. Chỉ là test — usecase memorial không gọi `lunarcal`. Lệch vector = hoặc `lunarcal` sai, hoặc vector cần sinh lại qua `ba-writer`; không sửa tay file JSON. |

### Schema `memorials`

```
id                   BIGSERIAL PRIMARY KEY
user_id              BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE
full_name            VARCHAR(100) NOT NULL
relationship         VARCHAR(24)  NOT NULL CHECK (IN 10 giá trị enum)
death_date           DATE CHECK (death_date >= DATE '1900-01-01')   -- NULL khi năm không rõ
death_calendar       VARCHAR(5)   NOT NULL CHECK (IN ('solar','lunar'))
death_day            SMALLINT CHECK (BETWEEN 1 AND 31)              -- chỉ khi death_date IS NULL
death_month          SMALLINT CHECK (BETWEEN 1 AND 12)
death_leap_month     BOOLEAN
note                 VARCHAR(1000)            -- NULL = không có
photo_object_key     TEXT                     -- NULL = không có ảnh
reminder_days_before SMALLINT CHECK (BETWEEN 0 AND 30)
reminder_time        TIME
reminder_timezone    VARCHAR(64)
add_to_os_reminder   BOOLEAN NOT NULL DEFAULT false
created_at, updated_at TIMESTAMPTZ NOT NULL DEFAULT now()

CHECK (hai dạng ngày mất loại trừ nhau:
         (death_date NOT NULL AND death_day, death_month, death_leap_month đều NULL)
      OR (death_date NULL AND death_day, death_month NOT NULL AND death_leap_month NOT NULL))
CHECK (death_calendar = 'lunar' OR death_leap_month IS NOT TRUE)
CHECK (death_calendar = 'solar' OR death_day IS NULL OR death_day <= 30)
CHECK (reminder_*: cả ba cùng NULL hoặc cùng NOT NULL)
CHECK (add_to_os_reminder = false OR reminder_days_before IS NOT NULL)
```

Ràng buộc "không ở tương lai" **không** đặt ở DB (phụ thuộc "hôm nay"); chỉ usecase kiểm.

## Request flow

`Get`, `List`, `Update` chỉ một câu SQL (hoặc đọc + ghi một dòng) nên giữ dạng ASCII; `Create` (khoá + đếm + chèn), `UploadPhoto`/`DeletePhoto` (DB + GCS) và `Delete` (DB + GCS) có nhiều bước nên nằm ở Step-by-step Flow và `flow.md`.

```
GET /v1/memorials/{id}   (tương tự PUT /v1/memorials/{id}, GET /v1/memorials)
  └─ authMiddleware.AuthenticateFunc  (401 nếu thiếu/sai token; gắn userID vào ctx)
       └─ MemorialHandler.Get   (delivery/http)
            ├─ userID từ context; parse {id} (không phải số dương → 404)
            └─ MemorialUsecase.Get(ctx, userID, id)
                 ├─ MemorialRepository.FindByIDAndUser → SELECT … WHERE id=$1 AND user_id=$2
                 │     └─ sql.ErrNoRows → ErrMemorialNotFound (404, giống "của người khác")
                 └─ MemorialPhotoStore.SignedURL (chỉ khi có photo_object_key)
```

## Step-by-step Flow — Thêm người vào sổ giỗ (POST /v1/memorials)

```
Step C-1 — Xác thực (middleware)
- Trigger: request đến POST /v1/memorials
- Pre-condition: header Authorization: Bearer <access token>
- Call: authMiddleware.AuthenticateFunc
    - Success → userID vào context
    - Failure (thiếu/hết hạn/sai loại token) → 401, dừng (AC2)

Step C-2 — Parse + validate (this module)
- Trigger: sau C-1
- Pre-condition: userID có trong context
- Call: Handler DecodeJSONBodyStrict → MemorialUsecase.Create validate AC3–AC10
    - Success → entity Memorial (hoặc death_date dương đầy đủ, hoặc death_day/month/leap theo death_calendar — AC6/6a/6b; reminder đã chuẩn hoá)
    - Failure → 400 VALIDATION_ERROR kèm details.field (AC3–AC10, AC12); chưa chạm DB

Step C-3 — Kiểm trần + chèn (this module, MỘT transaction)
- Trigger: sau C-2
- Pre-condition: dữ liệu hợp lệ
- Call: MemorialRepository.Create(ctx, m, MaxPerUser)
    a. BEGIN
    b. SELECT id FROM users WHERE id=$1 FOR UPDATE   (tuần tự hoá các Create của cùng user — chặn vượt trần khi chạy song song)
    c. SELECT count(*) FROM memorials WHERE user_id=$1
    d. nếu count >= 100 → ROLLBACK, ErrMemorialLimitReached
    e. INSERT INTO memorials (...) RETURNING id, created_at, updated_at
    f. COMMIT
    - Success → Memorial đã có id
    - Failure (đủ 100) → 409 MEMORIAL_LIMIT_REACHED (AC11)
    - Failure (lỗi DB) → 500 INTERNAL_ERROR, rollback
    - Transaction boundary: b–f cùng một tx; lỗi ở bất kỳ bước nào rollback toàn bộ.

Step C-4 — Trả response (this module)
- Trigger: sau C-3 thành công
- Pre-condition: row đã commit
- Call: Handler map entity → memorialResponse (photo_url = null vì mới tạo)
    - Success → 201 Memorial (AC1)
```

## Step-by-step Flow — Tải/thay ảnh (PUT /v1/memorials/{id}/photo)

> ⏸ **HOÃN (Phase 6) — chặn bởi GCS.** Giữ nguyên để làm tiếp; v1 không implement flow này.

```
Step P-1 — Xác thực + kiểm quyền sở hữu TRƯỚC khi đọc body (this module)
- Trigger: request đến PUT /v1/memorials/{id}/photo
- Pre-condition: access token hợp lệ (401 nếu không)
- Call: MemorialRepository.FindByIDAndUser(id, userID)
    - Success → có Memorial (và photo_object_key cũ, nếu có)
    - Failure (không có / của người khác) → 404 MEMORIAL_NOT_FOUND; body upload chưa được đọc (AC31)

Step P-2 — Đọc + kiểm file (this module)
- Trigger: sau P-1
- Pre-condition: memorial thuộc user
- Call: http.MaxBytesReader(5 MB) → multipart part "photo" → sniff 512 byte đầu bằng http.DetectContentType
    - Success → bytes ảnh + contentType ∈ {jpeg, png, webp}
    - Failure (thiếu/rỗng part) → 400 (AC28); (vượt 5 MB) → 413 (AC26); (sai kiểu) → 415 (AC27) — chưa ghi storage
    - ⚠️ Not confirmed: bucket/prefix riêng cho ảnh sổ giỗ và quyền ký signed URL của service account Cloud Run — chặn `MemorialConfig.PhotoBucket` và adapter GCS; dev tạm dùng `innerlamp-avatars-dev`.

Step P-3 — Ghi object mới (this module → GCS)
- Trigger: sau P-2
- Pre-condition: file hợp lệ
- Call: MemorialPhotoStore.Put(ctx, "memorials/<userID>/<id>/<uuid>.<ext>", file, contentType)
    - Success → newKey
    - Failure → 500 INTERNAL_ERROR; DB chưa đổi, ảnh cũ (nếu có) vẫn nguyên (AC32)

Step P-4 — Ghi key mới vào DB (this module)
- Trigger: sau P-3
- Pre-condition: newKey đã tồn tại trên GCS
- Call: MemorialRepository.SetPhotoKey(id, userID, newKey) — UPDATE … WHERE id AND user_id, trả oldKey; refresh updated_at
    - Success → oldKey (có thể rỗng)
    - Failure (lỗi DB) → best-effort MemorialPhotoStore.Delete(newKey) rồi 500; DB giữ key cũ (AC32)
    - Failure (0 dòng — vừa bị xoá song song) → xoá newKey best-effort, 404

Step P-5 — Dọn object cũ (this module → GCS, best-effort)
- Trigger: sau P-4 thành công AND oldKey khác rỗng
- Pre-condition: DB đã trỏ sang newKey
- Call: MemorialPhotoStore.Delete(oldKey)
    - Success → xong
    - Failure → chỉ log (key, không log tên/ghi chú), vẫn tiếp tục; object mồ côi chờ dọn (AC29; cơ chế dọn ở Pending của requirements)

Step P-6 — Trả response (this module)
- Trigger: sau P-4/P-5
- Call: SignedURL(newKey, 15m) → memorialResponse
    - Success → 200 Memorial có photo_url (AC25)
    - Failure (ký URL) → 500 INTERNAL_ERROR (ảnh đã lưu; client gọi GET lại để lấy URL)
```

`DELETE /v1/memorials/{id}/photo` dùng đúng thứ tự "DB trước, GCS sau": `SetPhotoKey(id, userID, nil)` → xoá object cũ best-effort → `200 Memorial` (`photo_url: null`); không có ảnh vẫn `200` (AC30).

## Step-by-step Flow — Xoá một mục (DELETE /v1/memorials/{id})

```
Step D-1 — Xác thực (middleware)
- Trigger: request đến DELETE /v1/memorials/{id}
- Call: authMiddleware.AuthenticateFunc
    - Failure → 401, dừng

Step D-2 — Xoá dòng (this module)
- Trigger: sau D-1
- Pre-condition: {id} là số dương (không thì 404)
- Call: MemorialRepository.Delete(id, userID) — DELETE FROM memorials WHERE id=$1 AND user_id=$2 (v1: kiểm RowsAffected; ⏸ Phase 6 đổi sang RETURNING photo_object_key để biết object cần xoá)
    - Success → xoá được 1 dòng
    - Failure (0 dòng: không tồn tại/của người khác/đã xoá) → 404 MEMORIAL_NOT_FOUND (AC24)
    - Failure (lỗi DB) → 500, không xoá gì

Step D-3 — Xoá ảnh (this module → GCS, best-effort) — ⏸ HOÃN tới Phase 6; v1 bỏ qua bước này (`photo_object_key` luôn NULL)
- Trigger: sau D-2 AND key khác NULL
- Pre-condition: dòng đã mất
- Call: MemorialPhotoStore.Delete(key)
    - Success/Failure → vẫn tiếp tục; thất bại chỉ log (AC23)

Step D-4 — Trả response
- Call: Handler → 200 {"success": true} (AC22)
```

**Vì sao "DB trước, GCS sau" ở cả ba flow ảnh/xoá:** nếu xoá/đổi object trước rồi DB lỗi, dòng sẽ trỏ vào object không còn → user thấy ảnh hỏng. Làm ngược lại thì xấu nhất là một object mồ côi user không thấy — chấp nhận được và dọn được sau.

## Known gaps carried by this feature

- ⏸ Phase 6: `cmd/main.go` chưa tạo storage client nào; `config.dev.yaml` có block `storage:` nhưng chưa có code feature nào đọc nó — Phase 4 là lần đầu nối thật. Khi chưa có bucket thật, dev có thể chạy với fake store để Phase 2–4 không bị chặn.
- 5 route v1 đã được bọc `authMiddleware.AuthenticateFunc` trong `cmd/main.go` và E2E (`e2e/memorial_e2e_test.go`) chứng minh cả 5 trả 401 khi thiếu token, token rác hoặc dùng refresh token. Khi làm Phase 6, hai route ảnh phải được thêm vào đúng bài kiểm tra này.
- `docs/specs/db/erd.svg` đã được sinh lại từ `erd.mmd` (2026-10-07); lần sau đổi schema phải sinh lại bằng `mmdc`, repo chưa có script cho việc này.
- Chưa có cơ chế dọn object GCS mồ côi (xem Pending của `requirements.md`).
- Validation giữ khoảng nhắc rộng hơn product (BR-MEM-11) và không kiểm tháng nhuận / năm âm khi có `death_date` (BR-MEM-04) — cả hai có chủ đích, lý do ở Ghi chú của `requirements.md`; không phải gap cần sửa.
- `openapi.yaml` chưa mô tả rằng cờ nhuận khôi phục từ `death_date` và `reminder.timezone` là múi giờ máy (product Appendix B mục 1, 3) — follow-up Phase 7.
