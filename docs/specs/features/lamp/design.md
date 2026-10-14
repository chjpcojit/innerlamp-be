# Design — lamp

Version: 1.3
Last updated: 2026-10-11
Status: v1 (Phase 1–4: domain, config, usecase, repository, migration `0027`, handler, wiring, openapi) đã implement và kiểm chứng 2026-10-10 — unit, repository (Postgres thật) và E2E (binary thật + Postgres) đều pass. Còn 2 marker `⚠️ Not confirmed` (hiệu năng/khối lượng; retention + xoá tài khoản — migration đang dùng `ON DELETE CASCADE` tạm thời). Chưa sinh lại `erd.svg` (thiếu `mmdc`).
Module: `lamp`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Hai route `/v1/lamps` đều **user-scoped về xác thực** (bọc `authMiddleware.AuthenticateFunc`, `RequireTokenType: "access"` như `/v1/memorials`), nhưng `GET` đọc dữ liệu của **mọi** người dùng — `userID` từ `pkg/jwt/context` chỉ dùng để thắp đèn, không dùng để lọc vườn đèn. Feature chạm đủ bốn layer của `services/appService` (`domain` → `usecase` → `repository` / `delivery/http`, nối dây ở `cmd/main.go`), sở hữu migration `0027_lamps.sql` và block config `lamp:`. Hình dạng gần nhất là `memorial` (tạo trong một transaction khoá dòng `users`, đếm trần, chèn) cộng một danh sách đọc chéo người dùng, phân trang bằng **cursor** (lệch quy ước `page`/`total_pages` của `temple`; đã chốt 2026-10-10, xem `requirements.md` → Open questions). Không gọi dịch vụ ngoài, không dùng `pkg/storage`.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/lamp.go` | `Lamp` (id, userID, `LampType`, `Intention`, recipient, `Prayer *string`, durationDays, isAnonymous, `LighterName *string`, litAt, endAt) + `DisplayName()`; `LampInput` (request thô); `GardenLamp` (id, `LampType`, recipient, `LighterName *string` + `DisplayName()` quy `nil` về `Vô danh`, `LitAt` chỉ để dựng cursor — không đi ra DTO, `EndAt`); `GardenFilter` (ba query thô, `nil` = vắng mặt); `GardenQuery`; `GardenPage` (items, total, counts `map[LampType]int`, hasMore, nextCursor *string); `Cursor{LitAt, ID}`; `LightLampParams{Lamp, MaxActive, DedupeWindow}`. Hằng enum `LampType` (3, kèm `AllLampTypes`), `Intention` (4), `AnonymousLighterName = "Vô danh"`. Không JSON/db tag. **Đã implement.** |
| Domain / Errors | `domain/lamp.go` | `ErrLampLimitReached`, `ErrInvalidLampFilter`, `ErrInvalidLimit`, `ErrInvalidCursor`. `ErrLampInvalid` + `*LampValidationError{Field, Message}` (có `Unwrap` về sentinel) — handler đọc `Field` trả `400 VALIDATION_ERROR` kèm `details.field`, cùng mẫu `MemorialValidationError`. |
| Domain / Repository interface | `domain/lamp.go` | `LampRepository`: `Light(ctx, p LightLampParams) (lamp *Lamp, created bool, err error)` — một transaction (khoá, chụp pháp danh, chống trùng, đếm trần, chèn); `Garden(ctx, q GardenQuery) (*GardenPage, error)` — một transaction đọc, cùng snapshot cho trang và đếm, trả tối đa `Limit+1` dòng. `LightLampParams` mang `Lamp.LitAt` (đồng hồ chung), `MaxActive`, `DedupeWindow` để repo không tự lấy giờ hay đọc config. **Đã implement.** |
| Domain / Mock | `domain/mock_lamp_repository.go` | Mock viết tay cho test usecase. |
| Usecase | `services/appService/usecase/lamp_usecase.go` | `Light(ctx, userID, in LampInput)`: trim + NFC, validate AC3–AC8 (enum, `durations_days`, `recipient_max_chars` và `prayer_max_chars` đều lấy từ config, độ dài đếm theo ký tự), tính `litAt = u.now()` (cắt về micro giây) và `endAt`, gọi `LampRepository.Light`. `Garden(ctx, f GardenFilter)`: validate `lamp_type`/`limit`/`cursor` (AC17, AC19, AC21), kẹp `limit` về `PageSize`, lấy `now`, gọi `LampRepository.Garden`, rồi cắt về `limit` dòng, dựng `has_more`/`next_cursor` (mã hoá/giải mã cursor nằm cùng file) và điền `0` cho loại vắng mặt trong `counts`. `now func() time.Time` inject được. **Đã implement.** |
| Repository | `services/appService/repository/postgres_lamp_repository.go` | SQL bảng `lamps`. `Light`: tx khoá `users`, đọc `dharma_name`, chống trùng, đếm trần, chèn. `Garden`: tx `READ ONLY` + `REPEATABLE READ`, `GROUP BY` đếm rồi lấy trang keyset. Mọi so sánh "đang sáng" là `end_at > $now` với `$now` do usecase truyền xuống. **Đã implement.** |
| Delivery / Handler | `services/appService/delivery/http/lamp_handler.go` | `Light`: `userID` từ context → `DecodeJSONBodyStrict` → usecase → `201`/`200`. `Garden`: đọc query `lamp_type`, `limit`, `cursor` (giữ nguyên kể cả rỗng để usecase từ chối) → usecase. Không logic nghiệp vụ. **Đã implement.** |
| Delivery / DTO | `delivery/http/lamp_handler.go` | `lampRequest`, `lampResponse`, `gardenLampResponse`, `gardenResponse` — JSON tag chỉ ở đây; `lit_at`/`end_at` RFC 3339 UTC; `items` luôn `[]` không `null`; `counts` luôn đủ 3 khoá; `next_cursor` là chuỗi base64url hoặc `null`. |
| Delivery / Error map | `delivery/http/errors.go` | `ErrLampLimitReached` → `409 LAMP_LIMIT_REACHED`; `ErrInvalidLampFilter` → `400 INVALID_FILTER` (`details.field = "lamp_type"`); `ErrInvalidLimit` → `400 INVALID_LIMIT`; `ErrInvalidCursor` → `400 INVALID_CURSOR`; `*LampValidationError` → `400 VALIDATION_ERROR` + `details.field`. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `LampConfig{MaxActivePerUser (20), PageSize (12), DedupeWindow (5s), DurationsDays ([1,7,21,49]), RecipientMaxChars (100), PrayerMaxChars (300)}`; `Validate` từ chối `MaxActivePerUser < 1`, `PageSize < 1`, `DedupeWindow <= 0`, `DurationsDays` rỗng / có giá trị `< 1` / trùng, `RecipientMaxChars`/`PrayerMaxChars` `< 1` hoặc **lớn hơn kích thước cột** (`LampRecipientColumnChars = 100`, `LampPrayerColumnChars = 300` — hai hằng trần này khớp `VARCHAR` của `0027_lamps.sql` và là hằng schema duy nhất còn lại; chúng chỉ là chốt chặn, không phải giá trị dùng để kiểm). Bắt buộc có block `lamp:` ở mọi `config.<env>.yaml`. |
| Wiring | `services/appService/cmd/main.go` | Dựng repo → usecase (truyền `cfg.Lamp`) → handler; đăng ký `POST /v1/lamps` và `GET /v1/lamps`, **cả hai bọc `authMiddleware.AuthenticateFunc`**. Phải kiểm tra thật sự đã bọc, không chỉ ghi là cần. |
| Migration | `migrations/0027_lamps.sql` | Bảng `lamps` (bên dưới) + ba index. Idempotent (`IF NOT EXISTS`). Cập nhật `migrations/README.md`. |
| API spec | `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) | 2 route, security Bearer, schema `Lamp`/`LampRequest`/`GardenLamp`/`GardenPage`; cập nhật cùng thay đổi handler. |
| DB spec | `docs/specs/db/erd.mmd`, `erd.svg` | Thêm bảng `lamps` + quan hệ `users 1—N lamps`; sinh lại `erd.svg` bằng `mmdc`. |
| Test E2E | `services/appService/e2e/lamp_e2e_test.go` | Binary thật + Postgres: 401 cả hai route khi thiếu/rác/refresh token, trần 20 kể cả song song, chống trùng, `counts` không đổi theo bộ lọc, cursor không lặp/sót khi đèn hết hạn. |

### Schema `lamps`

```
id             BIGSERIAL PRIMARY KEY
user_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE   -- tạm thời: retention chưa chốt (marker ở requirements.md)
lamp_type      VARCHAR(16)  NOT NULL CHECK (IN 'OIL_LAMP','LOTUS_CANDLE','FLOWER_LANTERN')
intention      VARCHAR(20)  NOT NULL CHECK (IN 'PEACE','SALVATION','MERIT_DEDICATION','GRATITUDE')
recipient      VARCHAR(100) NOT NULL
prayer         VARCHAR(300)                 -- NULL = không có
duration_days  SMALLINT     NOT NULL CHECK (duration_days > 0)
is_anonymous   BOOLEAN      NOT NULL DEFAULT false
lighter_name   VARCHAR(255)                 -- pháp danh chụp lúc thắp; NULL khi ẩn danh hoặc không có pháp danh
lit_at         TIMESTAMPTZ  NOT NULL        -- do usecase gán (đồng hồ server), không DEFAULT now()
end_at         TIMESTAMPTZ  NOT NULL

CHECK (end_at > lit_at)
CHECK (is_anonymous = false OR lighter_name IS NULL)   -- ẩn danh thì tuyệt đối không giữ tên

idx_lamps_lit       (lit_at DESC, id DESC)                 -- vườn đèn, mọi loại
idx_lamps_type_lit  (lamp_type, lit_at DESC, id DESC)      -- vườn đèn đã lọc loại
idx_lamps_user_end  (user_id, end_at)                      -- đếm trần và chống trùng
```

`duration_days` không có `CHECK IN (1,7,21,49)` ở DB vì tập giá trị nằm ở config (`lamp.durations_days`); DB chỉ chặn giá trị phi lý. Cột `recipient VARCHAR(100)` và `prayer VARCHAR(300)` là **trần cứng**: config có thể đặt giới hạn thấp hơn nhưng `Validate` không cho đặt cao hơn. `lit_at` không `DEFAULT now()` để `lit_at`/`end_at` luôn cùng một đồng hồ với phép so sánh `end_at > $now`.

## Request flow

Cả hai route có từ hai lệnh DB tuần tự trong một transaction (đếm + chèn; đếm + lấy trang), nên nằm ở Step-by-step Flow bên dưới.

> Xem `flow.md` cho sơ đồ sequence Mermaid (mirror 1:1 của Step-by-step Flow bên dưới).

## Step-by-step Flow — Thắp đèn (POST /v1/lamps)

```
Step L-1 — Xác thực (middleware)
- Trigger: request đến POST /v1/lamps
- Pre-condition: header Authorization: Bearer <access token>
- Call: authMiddleware.AuthenticateFunc
    - Success → userID vào context
    - Failure (thiếu/hết hạn/sai loại token) → 401, dừng (AC2)

Step L-2 — Parse + validate + tính thời gian (this module)
- Trigger: sau L-1
- Pre-condition: userID có trong context
- Call: Handler DecodeJSONBodyStrict → LampUsecase.Light: trim + NFC cho recipient/prayer, validate AC3–AC8, now := u.now(), litAt = now, endAt = now + durationDays×24h
    - Success → LightInput{userID, lamp (chưa có id/lighter_name), now, maxActive, dedupeWindow}
    - Failure → 400 VALIDATION_ERROR kèm details.field (AC3–AC8, AC11); chưa chạm DB

Step L-3 — Khoá, chụp pháp danh, chống trùng, kiểm trần, chèn (this module, MỘT transaction)
- Trigger: sau L-2
- Pre-condition: dữ liệu hợp lệ
- Call: LampRepository.Light(ctx, in)
    a. BEGIN
    b. SELECT dharma_name FROM users WHERE id=$1 FOR UPDATE   (tuần tự hoá các lần thắp của cùng user — chặn vượt trần và chặn hai lần bấm đúp cùng chèn; đồng thời chụp pháp danh trong cùng tx)
    c. SELECT … FROM lamps WHERE user_id=$1 AND lit_at > $now - $dedupeWindow AND lamp_type, intention, recipient, prayer (IS NOT DISTINCT FROM), duration_days, is_anonymous bằng nhau ORDER BY lit_at DESC LIMIT 1
    d. nếu có dòng → COMMIT, trả (dòng đó, created=false)   (AC10; kiểm trước trần)
    e. SELECT count(*) FROM lamps WHERE user_id=$1 AND end_at > $now
    f. nếu count >= maxActive → ROLLBACK, ErrLampLimitReached
    g. lighter_name := (is_anonymous ? NULL : dharma_name đọc ở b)   (AC12)
    h. INSERT INTO lamps (...) RETURNING id
    i. COMMIT → trả (lamp, created=true)
    - Success (mới) → created=true
    - Success (trùng trong cửa sổ) → created=false, không ghi gì
    - Failure (đủ trần) → 409 LAMP_LIMIT_REACHED (AC9)
    - Failure (lỗi DB) → 500 INTERNAL_ERROR, rollback
    - Transaction boundary: a–i cùng một tx; lỗi ở bất kỳ bước nào rollback toàn bộ.

Step L-4 — Trả response (this module)
- Trigger: sau L-3 thành công
- Pre-condition: tx đã commit
- Call: Handler map entity → lampResponse (lighter_name = "Vô danh" khi NULL)
    - Success (created=true) → 201 Lamp (AC1)
    - Success (created=false) → 200 Lamp là đèn đã có (AC10)
```

## Step-by-step Flow — Xem vườn đèn (GET /v1/lamps)

```
Step G-1 — Xác thực (middleware)
- Trigger: request đến GET /v1/lamps
- Pre-condition: header Authorization: Bearer <access token>
- Call: authMiddleware.AuthenticateFunc
    - Success → tiếp tục (userID không dùng để lọc)
    - Failure → 401, không trả đèn nào (AC14)

Step G-2 — Parse query + validate (this module)
- Trigger: sau G-1
- Pre-condition: request đã xác thực
- Call: Handler đọc lamp_type, limit, cursor → LampUsecase.Garden: validate AC17/AC19/AC21, kẹp limit về PageSize, giải mã cursor thành (litAt, id), now := u.now()
    - Success → GardenQuery{lampType?, limit, cursor?, now}
    - Failure (lamp_type lạ/rỗng) → 400 INVALID_FILTER; (limit không phải số nguyên dương) → 400 INVALID_LIMIT; (cursor hỏng) → 400 INVALID_CURSOR; chưa chạm DB

Step G-3 — Đếm và lấy trang (this module, MỘT transaction chỉ đọc)
- Trigger: sau G-2
- Pre-condition: query hợp lệ
- Call: LampRepository.Garden(ctx, q)
    a. BEGIN READ ONLY, ISOLATION REPEATABLE READ   (đếm và trang cùng một snapshot — AC18)
    b. SELECT lamp_type, count(*) FROM lamps WHERE end_at > $now GROUP BY lamp_type   → total = tổng, counts điền 0 cho loại vắng mặt (không phụ thuộc bộ lọc)
    c. SELECT id, lamp_type, recipient, lighter_name, lit_at, end_at FROM lamps WHERE end_at > $now [AND lamp_type=$t] [AND (lit_at, id) < ($cursorLitAt, $cursorId)] ORDER BY lit_at DESC, id DESC LIMIT $limit+1
    d. COMMIT
    - Success → dòng, total, counts
    - Failure (lỗi DB) → 500 INTERNAL_ERROR
    - Transaction boundary: a–d cùng một snapshot đọc; không có ghi.
    - ⚠️ Not confirmed: cách đếm (quét các dòng còn sáng bằng GROUP BY) có đạt mục tiêu thời gian phản hồi khi số đèn lớn không — chủ sản phẩm chưa có mục tiêu/khối lượng; chặn việc quyết định bảng đếm sẵn hay cache.

Step G-4 — Dựng response (this module)
- Trigger: sau G-3
- Pre-condition: có dòng, total, counts
- Call: Usecase cắt về `limit` dòng (hasMore = đã đọc được limit+1), nextCursor từ (lit_at, id) dòng cuối, điền 0 cho loại vắng mặt trong counts; Handler map sang gardenResponse (bỏ lit_at khỏi từng item, lighter_name NULL → "Vô danh" qua `GardenLamp.DisplayName()`)
    - Success → 200 {items, total, counts, has_more, next_cursor} (AC13–AC22)
```

**Vì sao keyset (cursor) thay cho `OFFSET`:** thứ tự `lit_at DESC, id DESC` ổn định; mốc là dòng cuối của trang trước nên đèn hết hạn hay đèn mới thắp giữa hai lần tải không làm dịch trang (AC20). `OFFSET` sẽ lặp hoặc sót đúng ở hai trường hợp đó. Cursor là base64url của `<lit_at unix micro>.<id>`, mờ với client; giải mã hỏng → `INVALID_CURSOR`.

**Vì sao chống trùng nằm trong cùng transaction với trần:** khoá dòng `users` ở bước b đã tuần tự hoá hai request của một user, nên "tìm đèn trùng rồi mới chèn" không bị race giữa hai lần bấm đúp; không cần cột/khoá idempotency riêng và không cần client gửi thêm gì.

## Deliberately stubbed for now (if applicable)

Không có. Retention/xoá tài khoản (marker ở `requirements.md`) chưa có route nên không có gì để stub; `ON DELETE CASCADE` là giá trị tạm.

## Known gaps carried by this feature

- `ON DELETE CASCADE` của `lamps.user_id` là **tạm thời** (marker retention ở `requirements.md`); phải chốt trước khi có tính năng xoá tài khoản — đổi bằng migration mới, không sửa `0027`.
- Hai route đã được bọc `authMiddleware.AuthenticateFunc` trong `cmd/main.go` và `e2e/lamp_e2e_test.go` chứng minh cả hai trả 401 khi thiếu token, token rác hoặc dùng refresh token.
- Cửa sổ chống trùng 5 giây được kiểm ở tầng repository (Postgres thật); E2E chỉ chứng minh `201` rồi `200`. Đoạn "quá 5 giây là đèn mới" cần chờ thật nên chưa có test E2E.
- `docs/specs/db/erd.mmd` đã có bảng `lamps`, nhưng `erd.svg` chưa sinh lại vì máy này không có `mmdc`.
- Gói `usecase` hiện không biên dịch được với `go test` thường vì `genealogy_member_usecase_test.go` (công việc genealogy dở dang) tham chiếu `GenealogyMemberUsecase` chưa tồn tại; test lamp đã chạy qua `-overlay` bỏ file đó. Không thuộc feature này.
- Cursor lệch quy ước phân trang `page`/`total_pages` của các feature danh sách khác (đã chốt có chủ đích); client phải hỗ trợ hai kiểu phân trang.
- `docs/specs/db/erd.svg` phải sinh lại bằng `mmdc` sau khi sửa `erd.mmd`; repo chưa có script cho việc này.
