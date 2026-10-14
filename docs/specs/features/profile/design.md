# Design — profile

Version: 1.5
Last updated: 2026-10-11
Status: implemented (v1.4) — năm route đã đăng ký trong `cmd/main.go`; luồng avatar đã được kiểm chứng đầu-cuối trên emulator với URL không ký, chưa trên bucket thật (`TD-001`). Không còn marker `⚠️ Not confirmed`; hai nợ kỹ thuật `TD-001`, `TD-002` ở `docs/tech-debt/profile/`.
Module: `profile`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Năm route dưới `/v1/users/me`, **đều bọc `authMiddleware.AuthenticateFunc`** (`RequireTokenType: "access"` như `/v1/memorials`) rồi mới đến bộ giới hạn tần suất theo `user_id`. `userID` lấy từ `pkg/jwt/context`, không bao giờ từ body/path/query. Feature chạm đủ bốn layer của `appService` và thêm hai thứ mà trước đó repo chưa có: **một cột `users.avatar_object`** và **storage (GCS) được wiring thật** vào `cmd/main.go`.

Hai nhóm luồng: (a) đọc/sửa hồ sơ chỉ chạm Postgres; (b) avatar chạm cả Postgres lẫn storage, và có hai điểm "nhiều lần gọi, một hiệu ứng phụ" cần Step-by-step Flow: **xác nhận avatar** (đọc metadata → copy → ghi DB → xoá file cũ) và **gỡ avatar**.

Use case tách thành `ProfileUsecase` (không nhét vào `LoginUsecase`/`RegisterUsecase`): đây là CRUD hồ sơ, không phải nghiệp vụ xác thực (đúng ghi chú kỹ thuật của US-004 cũ).

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/profile.go` | `Profile` (email, `EmailVerified bool`, phoneNumber, **`FullName *string` (v1.5)**, `DharmaName *string`, `OrdinationTemple *TempleRef{ID, Name}`, `AvatarObject *string` — khoá object, **không phải URL**); `ProfilePatch` (ba trường kiểu `OptionalString{Set, Value}` — `FullName` (v1.5), `DharmaName`, `OrdinationTempleID`: vắng mặt / `null` / giá trị); `ProfileView` (Profile + `AvatarURL`, `AvatarURLExpiresAt`: kết quả của usecase, địa chỉ ảnh đã ký); `AvatarUploadRequest{ContentType, SizeBytes}`; `AvatarUploadTicket{UploadID, URL, Method, Headers, ExpiresAt}`. Plain data, không tag. Không đụng `domain.User` hiện có (thêm `AvatarObject *string` vào đó là đủ cho nơi cần, xem task Phase 1). |
| Domain / Errors | `domain/profile.go`, `domain/errors.go` | `ErrProfileInvalid` + `*ProfileValidationError{Field, Message}` (có `Unwrap`, cùng mẫu `LampValidationError`); `ErrAvatarUploadNotFound`, `ErrInvalidAvatar`. **Dùng lại** `ErrInvalidTemple`, `ErrInvalidDharmaName` (đã có từ v1.1, dùng chung với đăng ký) và `ErrUserNotFound`. |
| Domain / Repository interface | `domain/profile.go` | `ProfileRepository`: `Get(ctx, userID) (*Profile, error)` — `users` LEFT JOIN `temples`; `Update(ctx, userID, patch ProfileUpdate) (*Profile, error)` — một câu `UPDATE … RETURNING` rồi đọc lại (hoặc một CTE); `SwapAvatar(ctx, userID, newObject string) (oldObject *string, err error)` — đặt khoá mới, trả khoá cũ; `ClearAvatar(ctx, userID) (oldObject *string, err error)`; `TempleExists(ctx, id) (bool, error)` (dùng lại cách của `RegistrationRepository.TempleExists`, hoặc đưa vào interface chung nếu đã tách). |
| Domain / Storage interface | `domain/profile.go` | `AvatarStorage` (cổng hẹp, usecase không phụ thuộc `pkg/storage` trực tiếp): `SignUpload(ctx, key, contentType string, maxBytes int64, ttl time.Duration) (url string, headers map[string]string, err error)`, `SignRead(ctx, key string, ttl time.Duration) (string, error)`, `Stat(ctx, key string) (*ObjectMeta{ContentType, Size}, error)` (trả `ErrObjectNotFound`), `Copy(ctx, srcKey, dstKey string) error`, `Delete(ctx, key string) error`. |
| Domain / Mock | `domain/mock_profile_repository.go`, `domain/mock_avatar_storage.go` | Mock viết tay cho test usecase. |
| Usecase | `services/appService/usecase/profile_usecase.go` | `Get`, `Update`, `RequestAvatarUpload`, `ConfirmAvatar`, `RemoveAvatar`. Trim + NFC + đếm ký tự (AC9), kiểm enum content type và size (AC19–20), kiểm `TempleExists` (AC11), dựng khoá object (AC18, AC22), gọi storage, ký URL đọc (AC5). `now` inject được. Không biết SQL, không biết GCS. |
| Repository | `services/appService/repository/postgres_profile_repository.go` | SQL `users`/`temples`. `Update` chỉ SET các cột có trong patch (cột `NULL` khi xoá), luôn `updated_at = now()`. FK `users.ordination_temple_id → temples.id` vi phạm (23503) → `ErrInvalidTemple` (chốt chặn cuối khi chùa bị xoá giữa lúc kiểm và ghi). Explicit column list, `context` mọi query. |
| Infrastructure / Storage adapter | `services/appService/storage/avatar_storage.go` | Cài `AvatarStorage` trên `pkg/storage.Client`. `SignUpload` trả `SignedUpload{URL, Method, Headers}`: bucket thật = `PUT` ký V4 (kèm `x-goog-content-length-range`); chế độ emulator (`WithEmulator`, khi `storage.emulator_host` có giá trị) = `POST` **không ký** tới endpoint upload JSON của emulator, vì emulator không có PUT upload. `SignRead` tương tự (GET ký / URL download thường). `Stat`/`Copy`/`Delete` luôn qua client; `ErrObjectNotFound` của pkg thành lỗi domain; `Delete` khoá vắng mặt không phải lỗi. **Không** là `pkg/` mới: là adapter của service. |
| Delivery / Handler | `services/appService/delivery/http/profile_handler.go` | `Get`, `Patch`, `AvatarUploadURL`, `AvatarConfirm`, `AvatarDelete`. `userID` từ context → `DecodeJSONBodyStrict` → usecase → `200`. Không logic nghiệp vụ. |
| Delivery / DTO | `delivery/http/profile_handler.go` | `profileResponse` (`omitempty` cho `full_name`, `dharma_name`, `ordination_temple`, `avatar_url`, `avatar_url_expires_at`), `profilePatchRequest` (ba trường kiểu `optionalString` tự phân biệt vắng mặt/`null`/giá trị bằng `UnmarshalJSON`), `avatarUploadRequest`, `avatarConfirmRequest`, `avatarUploadTicketResponse`. JSON tag chỉ ở đây. |
| Delivery / Error map | `delivery/http/errors.go`, `profile_handler.go` | `ErrUserNotFound` → `401 UNAUTHORIZED` **chỉ ở handler profile** (`writeProfileError`, để không đổi mã lỗi của route khác); `ErrInvalidDharmaName` → `400 INVALID_DHARMA_NAME` (`details.field = "dharma_name"`, **đã có từ v1.1**, dùng chung với đăng ký); `*ProfileValidationError` → `400 VALIDATION_ERROR` kèm `details.field` (body/field lạ); `ErrInvalidTemple` → `400 INVALID_TEMPLE` (`details.field = "ordination_temple_id"`, như đăng ký); `ErrInvalidAvatar` → `400 INVALID_AVATAR`; `ErrAvatarUploadNotFound` → `404 AVATAR_UPLOAD_NOT_FOUND`. |
| Delivery / Rate limit | `delivery/http/user_rate_limit.go` | `UserRateLimit` nhân bản `ClientIPRateLimit`, khác ở `KeyFunc`: lấy `user_id` từ `pkg/jwt/context` (đã có sau `authMiddleware`). `WrapFunc(next)`; vượt trần → `429 RATE_LIMITED` (envelope + `Retry-After` do `pkg/ratelimit` ghi sẵn). Sliding window, `Burst = Max`. |
| Delivery / No-store | `delivery/http/token_response.go` (`NoStore` có sẵn) | Bọc cả năm route: response có PII và signed URL không được cache. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | **`DharmaNameMaxChars` + `ProfileDharmaNameColumnChars` + `Validate` + block `profile:` đã có từ v1.1**; các field còn lại thêm khi implement route. `ProfileConfig{DharmaNameMaxChars (255), AvatarMaxBytes (5242880), AvatarContentTypes ([image/jpeg, image/png, image/webp]), UploadURLTTL (5m), AvatarReadURLTTL (15m), Bucket, AvatarTmpPrefix ("avatars/tmp/"), AvatarPrefix ("avatars/"), RateLimit{Max (10), Window (1m)}}`. `Validate` từ chối `DharmaNameMaxChars < 1` hoặc `> 255` (trần cột), `AvatarMaxBytes < 1`, danh sách content type rỗng/trùng, TTL `<= 0`, `Bucket` rỗng, hai prefix rỗng hoặc trùng nhau, `RateLimit.Max < 1`, `Window <= 0`. `Bucket` **không** là bí mật nhưng đổi theo môi trường nên nằm ở config, không ở secret. Thêm `StorageConfig.Timeout` (`storage.timeout`, dev `30s`) + `StorageConfig.EmulatorHost` (`storage.emulator_host`, `host:port`; dev `localhost:4443`, **phải để trống ngoài dev/test** vì nó tắt việc ký URL) + `StorageConfig.Validate`. |
| Wiring | `services/appService/cmd/main.go` | Dựng `storage.Client` (GCS, ADC) + `Connect(ctx)` lúc khởi động (fail fast) → adapter → repo → usecase (truyền `cfg.Profile`) → handler; đăng ký 5 route, **mỗi route bọc `authMiddleware.AuthenticateFunc` rồi `userRateLimit.WrapFunc` rồi `noStore`**. Việc đăng ký nằm ở `registerProfileRoutes` (`cmd/profile_routes.go`) và được `cmd/profile_routes_test.go` kiểm chứng (401 cả 5 route khi thiếu auth, `no-store`, chung một ngân sách). `⚠️ DEV CONFIG` cho `Bucket`/emulator: khi `storage.emulator_host` có giá trị, `main` đặt `STORAGE_EMULATOR_HOST` (cơ chế của thư viện GCS: HTTP, không credential), bật `WithEmulator` cho adapter và tạo bucket nếu thiếu với ngân sách thời gian riêng (chỉ cảnh báo khi emulator chưa lên). Emulator chạy cùng `make up` / `make run` (service `gcs` trong `docker-compose.yml`). CORS (`pkg/middleware`) được thêm method `PATCH`. |
| Registration (sửa nhỏ) | `usecase/register_usecase.go` | **Đã làm (v1.1).** `normalizeDharmaName(raw, maxChars)` (trim, NFC, đếm ký tự, rỗng → `nil`) dùng chung với `ProfileUsecase` (cùng package `usecase`); quá độ dài trả `400 INVALID_DHARMA_NAME` thay vì `500`. `NewRegisterUsecase` nhận thêm `dharmaNameMaxChars`. |
| Migration | `migrations/0029_users_avatar.sql` | `ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_object VARCHAR(255);` (idempotent). Không backfill (mọi tài khoản hiện có chưa có avatar). Cập nhật `migrations/README.md`. |
| API spec | `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) | 5 route, security Bearer, tag `Profile`, 5 schema; cập nhật cùng thay đổi handler. |
| DB spec | `docs/specs/db/erd.mmd`, `erd.svg` | Thêm cột `avatar_object` vào `users`; sinh lại `erd.svg` bằng `mmdc`. |
| Web docs | `web/docs/src/content/docs/guides/catalogues/profile.md`, `astro.config.mjs`, `changelog.md` | Trang hướng dẫn tích hợp cho client + mục sidebar + dòng changelog. |
| Test E2E | `services/appService/e2e/profile_e2e_test.go` | Binary thật + Postgres: 401 cả 5 route khi thiếu/rác/refresh token; PATCH tri-state; `email`/`phone_number` bị từ chối cả request; chùa lạ không lưu gì; `429` ở request thứ 11 và không ảnh hưởng tài khoản khác; avatar (phụ thuộc quyết định storage dev — marker). |

### Schema thay đổi (`users`)

```
avatar_object  VARCHAR(255)   -- khoá object trong thư mục chính, vd "avatars/42/0b9c….jpg"; NULL = chưa có ảnh
```

Không thêm bảng. `full_name` (migration `0033`, `VARCHAR(300)`, nullable; v1.5), `dharma_name`, `ordination_temple_id`, `email`, `email_verified_at`, `phone_number` đã có. Khoá object **luôn mới** cho mỗi lần xác nhận (`avatars/{user_id}/{uuid}.{ext}`) để một ảnh thay thế không bị cache theo cùng tên cũ, và để "ghi DB trước, xoá file cũ sau" không bao giờ xoá nhầm file vừa đặt.

### Vị trí object trong bucket

| Giai đoạn | Khoá | Ai ghi |
|---|---|---|
| Tạm (client PUT lên bằng signed URL) | `{avatar_tmp_prefix}{user_id}/{upload_id}` (dev: `avatars/tmp/42/0b9c…`) | Client, qua signed URL do server ký |
| Chính (sau xác nhận) | `{avatar_prefix}{user_id}/{uuid}.{ext}` (dev: `avatars/42/7f3a….jpg`) | Server, bằng `Copy` |

`upload_id` là UUID do server sinh ở bước xin URL. Server **không lưu** `upload_id` ở đâu (không bảng, không cache): khoá tạm suy ra được từ `user_id` của token + `upload_id` client gửi lại, nên id của người khác chỉ trỏ vào thư mục tạm **của chính người gọi** và sẽ không tìm thấy (AC23).

## Request flow

Đọc và sửa hồ sơ chỉ có một lệnh DB (kèm kiểm chùa), không phải flow nhiều bước; mô tả ngắn ở bên dưới. Ba luồng avatar có từ hai lời gọi tuần tự hoặc hiệu ứng phụ bền vững, nên có Step-by-step Flow đầy đủ.

> Xem `flow.md` cho sơ đồ sequence Mermaid (mirror 1:1 của Step-by-step Flow bên dưới).

## Đọc và sửa hồ sơ (GET / PATCH /v1/users/me)

```
Step P-1 — Xác thực (middleware)
- Trigger: request đến một trong 5 route
- Pre-condition: header Authorization: Bearer <access token>
- Call: authMiddleware.AuthenticateFunc
    - Success → userID vào context
    - Failure (thiếu/hết hạn/sai loại token) → 401, dừng (AC7, AC17, AC27)

Step P-2 — Giới hạn tần suất (middleware)
- Trigger: sau P-1
- Pre-condition: userID có trong context
- Call: userRateLimit.WrapFunc (khoá theo userID, sliding window, Max/Window từ config)
    - Success → còn lượt → tiếp tục
    - Failure (vượt trần) → 429 RATE_LIMITED + Retry-After, dừng; usecase không chạy (AC28)

Step P-3 — Parse + validate (this module; chỉ PATCH)
- Trigger: sau P-2, request là PATCH
- Pre-condition: body JSON hợp lệ
- Call: Handler DecodeJSONBodyStrict (trường lạ như email/phone_number bị từ chối) → ProfileUsecase.Update: trim + NFC full_name, đếm ký tự ≤ FullNameMaxChars, null/blank → lỗi (không xoá được, AC30–AC31); trim + NFC dharma_name, đếm ký tự ≤ DharmaNameMaxChars; blank/null → xoá
    - Success → ProfilePatch (mỗi trường: vắng / xoá / giá trị)
    - Failure → 400 INVALID_FULL_NAME (AC30, AC31), 400 INVALID_DHARMA_NAME (AC9) hoặc 400 VALIDATION_ERROR kèm details.field (AC15); chưa chạm DB

Step P-4 — Kiểm chùa (this module; chỉ PATCH có ordination_temple_id khác rỗng)
- Trigger: sau P-3
- Pre-condition: patch mang ordination_temple_id là chuỗi không rỗng
- Call: ProfileRepository.TempleExists(ctx, id)
    - Success (có) → tiếp tục
    - Failure (không có) → 400 INVALID_TEMPLE, không ghi gì (AC11, AC14)
    - Failure (lỗi DB) → 500

Step P-5 — Ghi (this module; chỉ PATCH)
- Trigger: sau P-4 (hoặc sau P-3 nếu patch không có chùa)
- Pre-condition: patch hợp lệ
- Call: ProfileRepository.Update(ctx, userID, patch): UPDATE users SET <chỉ cột có trong patch>, updated_at = now() WHERE id = $1 — MỘT câu lệnh
    - Success → ghi xong; patch rỗng ({}) → không SET cột nào ngoài updated_at hoặc bỏ qua, vẫn thành công (AC13)
    - Failure (FK 23503 trên ordination_temple_id) → ErrInvalidTemple → 400 INVALID_TEMPLE (chùa bị xoá giữa P-4 và P-5)
    - Failure (lỗi DB) → 500
    - Transaction boundary: một câu UPDATE tự nó là nguyên tử, tất cả cột của request cùng đổi hoặc không cột nào (AC14). Hai request đồng thời: lần commit sau thắng, không kiểm tra phiên bản (AC16).

Step P-6 — Đọc hồ sơ (this module; GET, và sau P-5 của PATCH)
- Trigger: request GET, hoặc sau P-5
- Pre-condition: người dùng tồn tại
- Call: ProfileRepository.Get(ctx, userID): SELECT … FROM users LEFT JOIN temples ON temples.id = users.ordination_temple_id WHERE users.id = $1
    - Success → Profile
    - Failure (không có dòng — tài khoản đã bị xoá sau khi cấp token) → ErrUserNotFound → 401 UNAUTHORIZED (cùng cách `refresh` coi tài khoản mất là token không hợp lệ)
    - Failure (lỗi DB) → 500

Step P-7 — Ký URL ảnh và trả response (this module)
- Trigger: sau P-6
- Pre-condition: có Profile
- Call: nếu AvatarObject != nil → AvatarStorage.SignRead(ctx, key, AvatarReadURLTTL) ; Handler map entity → profileResponse (omitempty), kèm Cache-Control: no-store
    - Success → 200 Profile (AC1–AC6)
    - Failure (ký URL lỗi) → 500 INTERNAL_ERROR (không trả hồ sơ thiếu ảnh một cách im lặng)
```

## Step-by-step Flow — Xin quyền upload avatar (POST /v1/users/me/avatar/upload-url)

```
Step U-1 — Xác thực (middleware)
- Trigger: request đến POST /v1/users/me/avatar/upload-url
- Pre-condition: header Authorization: Bearer <access token>
- Call: authMiddleware.AuthenticateFunc
    - Success → userID vào context
    - Failure → 401, dừng (AC27)

Step U-2 — Giới hạn tần suất (middleware)
- Trigger: sau U-1
- Pre-condition: userID có trong context
- Call: userRateLimit.WrapFunc
    - Success → tiếp tục
    - Failure → 429 RATE_LIMITED, dừng (AC28)

Step U-3 — Validate (this module)
- Trigger: sau U-2
- Pre-condition: body JSON hợp lệ
- Call: Handler DecodeJSONBodyStrict → ProfileUsecase.RequestAvatarUpload: content_type ∈ AvatarContentTypes, 0 < size_bytes ≤ AvatarMaxBytes (bằng đúng trần vẫn hợp lệ)
    - Success → AvatarUploadRequest
    - Failure → 400 VALIDATION_ERROR, details.field = content_type | size_bytes (AC19, AC20); chưa chạm storage

Step U-4 — Ký URL upload (this module + storage)
- Trigger: sau U-3
- Pre-condition: yêu cầu hợp lệ
- Call: upload_id = uuid mới; key = AvatarTmpPrefix + userID + "/" + upload_id; AvatarStorage.SignUpload(ctx, key, content_type, AvatarMaxBytes, UploadURLTTL) — chữ ký ràng buộc method PUT, Content-Type, và giới hạn độ dài (header x-goog-content-length-range: 0,AvatarMaxBytes)
    - Success → URL + headers client phải gửi nguyên văn
    - Failure (ký lỗi) → 500 INTERNAL_ERROR
    - Không ghi gì vào DB; không giữ trạng thái phía server

Step U-5 — Trả vé (this module)
- Trigger: sau U-4
- Pre-condition: có URL
- Call: Handler map → avatarUploadTicketResponse{upload_id, upload_url, method: "PUT", headers, expires_at = now + UploadURLTTL}
    - Success → 200 AvatarUploadTicket (AC18)

Step U-6 — Client PUT lên storage (ngoài server)
- Trigger: client có vé
- Pre-condition: trước expires_at; đúng Content-Type; ≤ 5 MB
- Call: PUT upload_url (client → storage, backend không nhận byte)
    - Success → object nằm ở thư mục tạm
    - Failure (hết hạn, sai Content-Type, quá độ dài) → storage từ chối; avatar không đổi (AC21)
```

## Step-by-step Flow — Xác nhận avatar (POST /v1/users/me/avatar/confirm)

```
Step C-1 — Xác thực + giới hạn tần suất (middleware)
- Trigger: request đến POST /v1/users/me/avatar/confirm
- Pre-condition: header Authorization: Bearer <access token>
- Call: authMiddleware.AuthenticateFunc → userRateLimit.WrapFunc
    - Success → userID trong context, còn lượt
    - Failure → 401 hoặc 429, dừng (AC27, AC28)

Step C-2 — Validate upload_id (this module)
- Trigger: sau C-1
- Pre-condition: body JSON hợp lệ
- Call: Handler DecodeJSONBodyStrict → ProfileUsecase.ConfirmAvatar: upload_id phải là UUID
    - Success → tmpKey = AvatarTmpPrefix + userID + "/" + upload_id
    - Failure → 400 VALIDATION_ERROR, details.field = "upload_id" (AC23); chưa chạm storage

Step C-3 — Đọc metadata object tạm (this module + storage)
- Trigger: sau C-2
- Pre-condition: tmpKey đã dựng từ userID của token
- Call: AvatarStorage.Stat(ctx, tmpKey)
    - Success → ObjectMeta{ContentType, Size}
    - Failure (không có object) → 404 AVATAR_UPLOAD_NOT_FOUND (AC23)
    - Failure (lỗi storage) → 500
    - Chỉ đọc metadata, KHÔNG tải nội dung, KHÔNG kiểm tra nội dung (quyết định 2026-10-11).

Step C-4 — Kiểm loại và dung lượng đã ghi nhận (this module)
- Trigger: sau C-3
- Pre-condition: có ObjectMeta
- Call: ContentType ∈ AvatarContentTypes và Size ≤ AvatarMaxBytes (không phải đọc lại giá trị client khai)
    - Success → tiếp tục
    - Failure → 400 INVALID_AVATAR, avatar không đổi, object tạm để nguyên (AC24)

Step C-5 — Copy sang thư mục chính (this module + storage)
- Trigger: sau C-4
- Pre-condition: object tạm hợp lệ
- Call: newKey = AvatarPrefix + userID + "/" + uuid mới + đuôi theo content type; AvatarStorage.Copy(ctx, tmpKey, newKey)
    - Success → object mới ở thư mục chính; object tạm GIỮ NGUYÊN (dọn tạm hoãn)
    - Failure → 500 INTERNAL_ERROR, DB chưa đổi nên avatar cũ còn nguyên (AC25)
    - Copy xảy ra TRƯỚC khi ghi DB để DB không bao giờ trỏ tới object chưa tồn tại.

Step C-6 — Ghi khoá mới vào hồ sơ (this module)
- Trigger: sau C-5
- Pre-condition: newKey tồn tại trong thư mục chính
- Call: ProfileRepository.SwapAvatar(ctx, userID, newKey): đọc avatar_object cũ rồi UPDATE users SET avatar_object = $new, updated_at = now() WHERE id = $1, trong một transaction (khoá dòng bằng SELECT … FOR UPDATE để hai lần xác nhận đồng thời mỗi lần nhận đúng khoá cũ của mình)
    - Success → trả oldKey (có thể rỗng)
    - Failure (lỗi DB) → 500; newKey đã copy nhưng không ai trỏ tới (rác, nằm trong phần dọn hoãn); avatar cũ còn nguyên
    - Transaction boundary: SELECT … FOR UPDATE + UPDATE cùng một tx; lỗi thì rollback.

Step C-7 — Xoá ảnh cũ (this module + storage; best-effort)
- Trigger: sau C-6, oldKey != nil
- Pre-condition: DB đã trỏ sang newKey
- Call: AvatarStorage.Delete(ctx, oldKey)
    - Success → ảnh cũ không còn (AC25)
    - Failure → ghi log (chỉ user_id và key, không PII), vẫn trả 200; ảnh cũ thành rác
    - Quyết định 2026-10-11: chấp nhận để lại file cũ thành rác khi xoá lỗi — log, trả 200, không thử lại (nợ kỹ thuật TD-002).

Step C-8 — Dựng response (this module)
- Trigger: sau C-7
- Pre-condition: DB đã đổi
- Call: ProfileRepository.Get → AvatarStorage.SignRead(newKey, AvatarReadURLTTL) → profileResponse (như P-6, P-7)
    - Success → 200 Profile với avatar_url mới (AC22)
    - Failure → 500; dữ liệu đã đổi nhưng response lỗi — client đọc lại hồ sơ để thấy trạng thái thật
```

## Step-by-step Flow — Gỡ avatar (DELETE /v1/users/me/avatar)

```
Step D-1 — Xác thực + giới hạn tần suất (middleware)
- Trigger: request đến DELETE /v1/users/me/avatar
- Pre-condition: header Authorization: Bearer <access token>
- Call: authMiddleware.AuthenticateFunc → userRateLimit.WrapFunc
    - Success → tiếp tục
    - Failure → 401 hoặc 429, dừng (AC27, AC28)

Step D-2 — Gỡ khoá khỏi hồ sơ (this module)
- Trigger: sau D-1
- Pre-condition: userID trong context
- Call: ProfileRepository.ClearAvatar(ctx, userID): SELECT avatar_object … FOR UPDATE; UPDATE users SET avatar_object = NULL, updated_at = now() WHERE id = $1 AND avatar_object IS NOT NULL — một tx
    - Success → trả oldKey (rỗng nếu chưa có ảnh: không đổi gì)
    - Failure (lỗi DB) → 500
    - Transaction boundary: SELECT … FOR UPDATE + UPDATE cùng một tx.

Step D-3 — Xoá object (this module + storage; best-effort)
- Trigger: sau D-2, oldKey != nil
- Pre-condition: DB đã gỡ khoá
- Call: AvatarStorage.Delete(ctx, oldKey)
    - Success → ảnh không còn (AC26)
    - Failure → ghi log, vẫn trả 200 (cùng quyết định "log + 200" của C-7)
    - Như C-7: chấp nhận file cũ thành rác khi xoá lỗi (TD-002).

Step D-4 — Trả hồ sơ (this module)
- Trigger: sau D-3
- Pre-condition: không còn avatar
- Call: ProfileRepository.Get → profileResponse (không có avatar_url)
    - Success → 200 Profile; gọi khi chưa có ảnh cũng trả 200 với hồ sơ không đổi (AC26)
```

## Deliberately-stubbed-for-now

- **Storage dev dùng emulator, URL không ký** (quyết định 2026-10-11, chốt lại 2026-10-11: không ký ở dev/test, ký thật là nợ khi deploy): `storage.emulator_host` ở `config.dev.yaml` trỏ container `fake-gcs-server`. Emulator không kiểm chữ ký V4, header đã ký, giới hạn dung lượng hay hạn URL, nên các giới hạn đó **chỉ được usecase kiểm lại ở bước confirm** (loại và dung lượng storage ghi nhận) và **phải kiểm tra luồng ticket → upload → confirm với chữ ký thật trên bucket thật khi deploy** (nợ kỹ thuật TD-001). `method` của vé do store quyết định (bucket thật `PUT`, emulator `POST`).
- **Dọn thư mục tạm:** không có (nợ kỹ thuật TD-002). Upload không xác nhận, và object tạm sau xác nhận, nằm lại. Không viết job, không đặt lifecycle rule ở v1 (quyết định 2026-10-11). Khi làm, hướng đề xuất là lifecycle rule xoá object dưới `avatar_tmp_prefix` sau 1 ngày — không cần sửa code, chỉ cần cấu hình bucket.
- **Xoá ảnh khi xoá tài khoản:** chưa có route xoá tài khoản; chỉ ghi nhận phụ thuộc.
- **Kiểm tra nội dung / kiểm duyệt:** không làm (quyết định 2026-10-11). Hệ quả đã chấp nhận: file tên `.jpg` nhưng không phải ảnh vẫn thành avatar nếu khai đúng `content_type`/`size_bytes`.
