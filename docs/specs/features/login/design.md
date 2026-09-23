# Design — login

Version: 1.10
Last updated: 2026-09-22
Status: implemented (2026-09-21) — domain, usecase, repository, delivery và wiring có code + test; login/refresh/logout đã chạy thật qua binary + Postgres 16 (curl, AC1–AC15). Audit AC bằng E2E tự động (testcontainers + binary thật) xong 2026-09-21 — `test-coverage-report.md`; concept OKF đã có. Phụ thuộc `jwt-token-uniqueness` và `onboarding` Phase 7 (claim `sid`) đều đã xong. Không còn marker `⚠️ Not confirmed`. (v1.10: sửa tài liệu cũ — helper `hashRefreshToken()` của repository nay là `domain.HashRefreshToken` (`domain/session_token.go`); mục rủi ro về dòng "30 ngày" ở `onboarding/requirements.md` đã hết vì onboarding v5.2 sửa dòng đó. Không đổi code/AC.)
Module: `identity`
Code: `services/appService/`
Feature folder: `services/appService/` + `docs/specs/features/login/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này phủ **US-002 (đăng nhập)** gồm ba route: `login`, `refresh`, `logout`. Các thứ tái dùng từ US-001 (đã có code, xem `docs/specs/features/onboarding/design.md`): `domain.User`, `domain.TokenIssuer` (dùng lại nguyên trạng, xem "Phụ thuộc: `jwt-token-uniqueness`"), `validation.NormalizePhone`, bảng `users` và `user_sessions`, `domain.HashRefreshToken` (`domain/session_token.go`), `repository.isUniqueViolation()`, `config.JWTConfig`, `usecase.userRole`, `delivery/http/errors.go`.

## Overview

Login là luồng **một bước**, khác với đăng ký hai bước: người gọi đã có tài khoản (đã verify OTP ở US-001), nên chỉ cần chứng minh bằng mật khẩu. Hai bất biến trung tâm:

1. **Sai thông tin đăng nhập không phân biệt được** (AC2): mật khẩu sai, SĐT không tồn tại và SĐT chỉ có `pending_registration` phải cho ra cùng một response, và (nhờ dummy bcrypt) cùng cỡ thời gian.
2. **Mỗi `(user, device)` có tối đa một session active** (AC4), đảm bảo bằng cả logic lẫn partial unique index ở DB.

`login`/`refresh` là **public**; `refresh` xác thực bằng chính refresh token (tra hash trong `user_sessions`). `logout` là **`RequireAuth`**: access token trong header + claim `sid` cho biết đúng session cần thu hồi (chốt 2026-09-21). Đây là route đầu tiên được bọc `pkg/jwt/middleware` trong `cmd/main.go`; các route user-scoped khác (US-004) vẫn còn là gap.

Ba flow đều ghi nhiều bảng hoặc có transaction (login: `login_attempts`, `user_devices`, `user_sessions`; refresh: `user_sessions` có khoá hàng), nên bắt buộc có Step-by-step Flow (rule 5 của skill). Login là feature thứ hai chạm `bcrypt` + `TokenIssuer`; điểm mới so với US-001 là **rate limit lưu ở DB** và **session gắn thiết bị**.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/login.go` *(mới)* | `DeviceInfo` (`Platform`, `DeviceModel`, `OSVersion`, `AppVersion`, `DeviceName` từ client; `IPAddress`, `UserAgent` do handler điền), `LoginAttempt` (`PhoneNumber`, `FailedCount`, `LockedUntil *time.Time`, `LastFailedAt`), `Session` (kết quả tra cứu refresh token: `ID`, `SessionID`, `UserID`, `DeviceID *int64`, `TokenVersion`, `ExpiresAt`, `IsRevoked`). Thuần dữ liệu, không JSON tag. |
| Domain / Input-Output | `services/appService/domain/login.go` *(mới)* | `LoginInput{PhoneNumber, Password, Device DeviceInfo}`, `LoginResult = RegisterResult` (cùng shape, dùng lại), `RefreshInput{RefreshToken}`, `RefreshResult{AccessToken, RefreshToken}`, `LogoutInput{UserID, SessionID}` (lấy từ claims của access token, không từ body). |
| Domain / Func type | `services/appService/domain/register.go` *(đã có — **sửa**)* | `TokenIssuer` đổi từ `func(userID string) (access, refresh string, err error)` thành `func(userID, sessionID string) (access, refresh string, err error)`. Closure do usecase tạo, bắt sẵn `email`/`role`, gắn `extra{"sid": sessionID}` vào **access** token. Dùng chung cho `CreateUserAndSession` (US-001), `CreateLoginSession`, `RotateSession`. Không thêm func type mới. Token duy nhất nhờ `jti` của `pkg/jwt` (xem "Phụ thuộc: `jwt-token-uniqueness`"). |
| Domain / Errors | `services/appService/domain/errors.go` *(thêm)* | Sentinels: `ErrInvalidCredentials`, `ErrLoginLocked`, `ErrInvalidDevice`, `ErrRefreshTokenInvalid`, `ErrUserNotFound` (nội bộ — usecase gộp về `ErrInvalidCredentials`, không bao giờ ra tới handler). Typed error `LoginLockedError{LockedUntil}` (`Unwrap()` về `ErrLoginLocked`), cùng kiểu `OTPLockedError`. |
| Domain / Repository interface | `services/appService/domain/login.go` *(mới)* | `LoginRepository` (khai ở `domain/`, `repository/` implement). Tám method: `FindUserByPhone`, `FindUserByID`, `FindLoginAttempt`, `RecordLoginFailure`, `CreateLoginSession`, `FindSessionByRefreshHash`, `RotateSession`, `RevokeSession`. `FindUserByID` là method thứ tám, thêm ở implement: `RotateSession` phải mint access token mang claim `email`, mà request refresh chỉ biết `user_id` của session (Flow 2 Step 3 đã ghi "cần thêm bước đọc"). Chữ ký chi tiết ở các Step bên dưới. |
| Domain / Mock | `services/appService/domain/mock_login_repository.go` *(mới)* | Mock tay cho usecase test — cùng kiểu `mock_registration_repository.go`. |
| Usecase | `services/appService/usecase/login_usecase.go` *(mới)* | Ba method: `Login` (validate → kiểm khoá → tra user → bcrypt → ghi thất bại **hoặc** tạo session), `Refresh` (validate → tra hash → rotate), `Logout` (tra hash → revoke, idempotent). Không sở hữu số liệu: `config.LoginConfig` (ngưỡng/khoá/cửa sổ) và `sessionTTL` đến qua constructor, cùng cách `RegisterUsecase` nhận `config.OTPConfig`. Tái dùng `userRole` và `jwt.Service`. |
| Usecase / Timing guard | `services/appService/usecase/login_usecase.go` *(mới)* | `dummyPasswordHash` — một hash bcrypt hợp lệ sinh một lần lúc package load (biến package-level khởi tạo bằng hàm, cùng thời điểm với `init()`; cùng `bcrypt.DefaultCost` với hash thật). SĐT không tồn tại vẫn chạy `CompareHashAndPassword` với nó, để AC2 không lộ qua thời gian. |
| Repository | `services/appService/repository/postgres_login_repository.go` *(mới)* | Toàn bộ SQL + transaction boundary. `CreateLoginSession` và `RotateSession` là **một method = một tx**. Tái dùng `isUniqueViolation()` từ `postgres_registration_repository.go` (cùng package) và `domain.HashRefreshToken` (helper ở `domain/session_token.go`, dùng chung cho cả usecase lẫn repository). Helper mới: `deviceFingerprint(DeviceInfo)`. |
| Delivery / Handler | `services/appService/delivery/http/login_handler.go` *(mới)* | Ba handler (`Login`, `Refresh`, `Logout`), DTO request/response, decode strict (`DecodeJSONBodyStrict`; `Logout` không có body). `Logout` đọc `claims` bằng `jwtcontext.GetClaims(ctx)` (`pkg/jwt/context`), lấy `claims.UserID` và `claims.Extra["sid"]` (string); thiếu/sai kiểu `sid` → 401 `UNAUTHORIZED`. Điền `Device.IPAddress`/`UserAgent` từ `*http.Request` (client không gửi được hai field này). Tái dùng `userResponse` của `register_handler.go`. |
| Delivery / Shared | `services/appService/delivery/http/errors.go` *(thêm)* | Thêm vào `writeDomainError`: `ErrInvalidCredentials` → 401 `INVALID_CREDENTIALS`; `LoginLockedError` → 429 `LOGIN_LOCKED` + `locked_until` (kiểm bằng `errors.AsType` **trước** `switch`, như `OTPLockedError`); `ErrInvalidDevice` → 400 `INVALID_DEVICE` `{field}`; `ErrRefreshTokenInvalid` → 401 `REFRESH_TOKEN_INVALID`; thêm hàm **`WriteAuthError`** (xuất khẩu — `cmd/main.go` ở package khác phải truyền nó vào middleware) làm `ErrorHandler` (401 `UNAUTHORIZED`). Envelope thật của mọi route là `utils.ApiResponse` = `{success, error{code,message,type}, details}`, không phải `{error:{code,message}}` như bản spec cũ ghi. |
| Config | `services/appService/config/config.go` + `config.dev.yaml` *(thêm)* | `LoginConfig{MaxFailedAttempts (5), LockDuration (15m), FailureResetWindow (15m)}` trong `AppConfig.Login`, cùng mẫu `OTPConfig`. Refresh TTL không thêm gì: dùng `JWTConfig.RefreshTokenExpiration` (168h). |
| Wiring | `services/appService/cmd/main.go` *(thêm)* | Constructor injection + 3 `mux.HandleFunc`: `POST /v1/auth/login`, `POST /v1/auth/refresh`, `POST /v1/auth/logout`. `login`/`refresh` **không** bọc middleware; `logout` bọc `middleware.NewHTTPAuthMiddlewareWithOptions(jwtService, logger, HTTPAuthOptions{RequireTokenType: "access", ErrorHandler: deliveryhttp.WriteAuthError}).AuthenticateFunc(...)`. Phải kiểm bằng đọc `main.go` rằng route logout thực sự được bọc (gap "documented but not wired" đã từng xảy ra với `/v1/users/me`). |
| Migration | `migrations/0002_login_devices_and_rate_limit.sql` *(mới)* | Tạo `user_devices`, `login_attempts`; `ALTER TABLE user_sessions` thêm `device_id`, `previous_refresh_token_hash`; partial unique index. Xem "Schema" bên dưới. Chưa apply lên Postgres thật. |
| API spec | `docs/specs/api/openapi.yaml` | Thêm 3 path — viết **sau** khi handler chạy thật qua `curl` (tiền lệ US-001), không viết trước. |

Dependency rule: `delivery → usecase → domain`; `repository → domain`; `domain` không import layer ngoài. Usecase import `pkg/jwt`, `pkg/validation`, `services/appService/config` như `RegisterUsecase` đã làm.

### Schema (`0002_login_devices_and_rate_limit.sql`)

```sql
-- user_devices: một hàng cho mỗi (user, thiết bị đã từng login).
CREATE TABLE IF NOT EXISTS user_devices (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id    VARCHAR(64) NOT NULL,          -- SHA-256 hex fingerprint, tính ở server
    platform     VARCHAR(32)  NOT NULL,
    device_model VARCHAR(128) NOT NULL,
    os_version   VARCHAR(64),
    app_version  VARCHAR(64),
    device_name  VARCHAR(128),
    ip_address   VARCHAR(45),
    user_agent   VARCHAR(512),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_devices_user_id_device_id_key UNIQUE (user_id, device_id)
);

ALTER TABLE user_sessions
    ADD COLUMN device_id BIGINT REFERENCES user_devices(id) ON DELETE SET NULL,  -- NULL = session của US-001
    ADD COLUMN previous_refresh_token_hash VARCHAR(64);                          -- phát hiện reuse (AC11)

-- Backstop cho AC4: tối đa một session chưa revoke trên mỗi device.
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_sessions_active_device
    ON user_sessions (device_id) WHERE is_revoked = false AND device_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_user_sessions_previous_refresh_hash
    ON user_sessions (previous_refresh_token_hash) WHERE previous_refresh_token_hash IS NOT NULL;

-- login_attempts: khoá theo SĐT, KHÔNG có FK sang users — phải đếm được cả SĐT chưa đăng ký (AC7).
CREATE TABLE IF NOT EXISTS login_attempts (
    phone_number   VARCHAR(20) PRIMARY KEY,
    failed_count   INT NOT NULL DEFAULT 0,
    locked_until   TIMESTAMPTZ,
    last_failed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

`user_sessions.device_id` nullable **có chủ ý**: session do `verify-otp` (US-001) tạo ra không có thiết bị, và AC12 yêu cầu chúng vẫn refresh/logout được. `user_devices.device_name`/`os_version`/`app_version` nullable vì là field tùy chọn.

### Phụ thuộc: `jwt-token-uniqueness`

`pkg/jwt` hiện **không sinh `jti`** và `iat` chỉ chính xác 1 giây → hai refresh token cùng user trong cùng giây giống hệt từng byte (tái hiện: `refresh identical: true`), vỡ `user_sessions.refresh_token_hash UNIQUE`. Xảy ra thật khi login hai thiết bị cùng lúc, hoặc `refresh` ngay sau `login` (rotate ra đúng token cũ, làm hỏng luôn phát hiện reuse). Lỗi được **tách ra và sửa ở gốc** tại [`docs/specs/features/jwt-token-uniqueness/`](../jwt-token-uniqueness/design.md): mọi token mới mang `jti` 128 bit từ `crypto/rand`.

Hệ quả cho login: (1) không cần `extra{sid, ver}` — phương án tạm ở v1.0 đã bị bỏ; (2) dùng lại `domain.TokenIssuer` nguyên trạng; (3) **thứ tự làm việc bắt buộc**: `jwt-token-uniqueness` Phase 2 → rồi mới `login/tasks.md` Phase 2–3. Test repository "hai `RotateSession` đồng thời" và "refresh ngay sau login" chỉ có nghĩa khi đã có `jti`.

### Phụ thuộc: sửa US-001 (`sid` trong access token)

Chọn `logout` bằng access token (2026-09-21) nghĩa là access token **phải** cho biết session. `CreateUserAndSession` (`repository/postgres_registration_repository.go`) hiện gọi `issueTokens(userID)` rồi mới `uuid.NewString()` cho `session_id`. Phải đổi thành: sinh `sessionID` trước, gọi `issueTokens(userID, sessionID)`, rồi insert với đúng `sessionID` đó; `usecase/register_usecase.go` (closure tại dòng ~369) gắn `extra{"sid": sessionID}` vào `GenerateAccessToken`; `domain.TokenIssuer` đổi chữ ký; các test đang mock `GenerateAccessToken(..., mock.Anything)` phải được kiểm lại. Hành vi ngoài của US-001 **không đổi** (response giữ nguyên). Việc này được ghi thành task ở `onboarding/tasks.md` Phase 7 và là điều kiện tiên quyết của `tasks.md` Phase 0. Session/access token phát hành **trước** khi sửa không có `sid` → logout trả 401 tới khi đăng nhập lại (chỉ ảnh hưởng dev, chưa có production client).

## Request flow

```
POST /v1/auth/login
  └─ public
       └─ LoginHandler.Login                                   (delivery/http)
            ├─ DecodeJSONBodyStrict → loginRequest {phone_number, password, device{...}}
            ├─ điền Device.IPAddress / UserAgent từ *http.Request
            └─ LoginUsecase.Login(ctx, input)                  (usecase)
                 ├─ normalize phone + validate + validate device      → 400, KHÔNG đếm thất bại
                 ├─ LoginRepository.FindLoginAttempt                   → đang khoá ⇒ 429 LOGIN_LOCKED
                 ├─ LoginRepository.FindUserByPhone
                 ├─ bcrypt.CompareHashAndPassword (dummy hash nếu không có user)
                 ├─ sai  ⇒ LoginRepository.RecordLoginFailure  ⇒ 401 INVALID_CREDENTIALS
                 └─ đúng ⇒ LoginRepository.CreateLoginSession(ctx, ..., TokenIssuer)
                            └─ 1 tx: upsert user_devices → revoke session cũ của device
                                     → insert user_sessions → reset login_attempts

POST /v1/auth/refresh
  └─ public
       └─ LoginHandler.Refresh → LoginUsecase.Refresh
            ├─ jwtService.ValidateToken(refresh_token)             → chữ ký/hạn/loại token
            ├─ LoginRepository.FindSessionByRefreshHash
            └─ LoginRepository.RotateSession (1 tx, SELECT ... FOR UPDATE)

POST /v1/auth/logout
  └─ RequireAuth  (HTTPAuthMiddleware: Bearer access token, RequireTokenType="access")
       └─ LoginHandler.Logout                                  (delivery/http)
            ├─ jwtcontext.GetClaims → claims.UserID + claims.Extra["sid"]   (thiếu → 401)
            └─ LoginUsecase.Logout(ctx, LogoutInput{UserID, SessionID})
                 └─ LoginRepository.RevokeSession(userID, sessionID, 'logout')   (idempotent)
```

## Step-by-step Flow — Login: `POST /v1/auth/login`

```
Step 1 — Validate và chuẩn hoá input (feature này)
- Trigger: LoginHandler nhận POST /v1/auth/login với body decode được
- Pre-condition: DecodeJSONBodyStrict pass (field lạ bị từ chối, cap 1 MB)
- Call: LoginUsecase.Login → validation.NormalizePhone → phoneE164Regex → password != ""
        → device.platform != "" && device.device_model != ""
  Không kiểm độ dài mật khẩu ≥ 8 ở đây: đó là luật lúc đăng ký; tài khoản cũ vẫn phải đăng nhập
  được nếu luật đổi. Chỉ từ chối mật khẩu rỗng.
    - Success → Step 2
    - Failure (phone sai định dạng, kể cả sau chuẩn hoá) → ErrInvalidPhone → 400 INVALID_PHONE
    - Failure (password rỗng) → ErrInvalidPassword → 400 INVALID_PASSWORD
    - Failure (thiếu device / platform / device_model) → ErrInvalidDevice → 400 INVALID_DEVICE
    - Không chạm DB và KHÔNG đếm vào login_attempts ở mọi nhánh failure (AC5, AC6): request lỗi
      định dạng không phải lần đoán mật khẩu, và đếm chúng cho phép ai đó khoá SĐT của người khác
      bằng request rác.

Step 2 — Kiểm tra SĐT đang bị khoá (feature này)
- Trigger: Step 1 pass
- Pre-condition: phone đã chuẩn hoá
- Call: LoginRepository.FindLoginAttempt(phone) → SELECT phone_number, failed_count, locked_until,
        last_failed_at FROM login_attempts WHERE phone_number = $1  (không có hàng → nil, nil)
    - Success (không có hàng, hoặc locked_until IS NULL, hoặc locked_until <= now()) → Step 3
    - Failure (locked_until > now()) → LoginLockedError{LockedUntil} → 429 LOGIN_LOCKED kèm
      locked_until; DỪNG, không tra user, không so mật khẩu (AC7: khoá rồi thì mật khẩu đúng cũng
      bị từ chối)
  Confirmed: check này chạy TRƯỚC khi tra user, và áp cho cả SĐT không tồn tại — nên response 429
  không tiết lộ số nào đã đăng ký. Không cập nhật hàng ở bước này (chỉ đọc).

Step 3 — Tra user theo SĐT (feature này)
- Trigger: Step 2 pass
- Pre-condition: SĐT không bị khoá
- Call: LoginRepository.FindUserByPhone(phone) → SELECT <cột tường minh> FROM users
        WHERE phone_number = $1
  Không đọc users.email hay email_verified_at để quyết định gì (gate email của US-001 được thoả
  mặc nhiên vì email không phải định danh).
    - Success (có hàng) → Step 4 với password_hash thật
    - Success (không có hàng — SĐT chưa đăng ký, hoặc chỉ có pending_registration) →
      ErrUserNotFound (nội bộ); usecase dùng dummyPasswordHash và vẫn đi Step 4
    - Failure (lỗi DB) → wrapped "find user: %w" → 500 INTERNAL_ERROR; không đếm thất bại

Step 4 — So khớp mật khẩu (feature này)
- Trigger: Step 3 pass
- Pre-condition: có password_hash (thật hoặc dummy)
- Call: bcrypt.CompareHashAndPassword(hash, []byte(password))
  Luôn chạy đúng một lần bcrypt, dù user có tồn tại hay không — thời gian phản hồi của hai nhánh
  cùng cỡ (AC2). Kết quả cuối = (user tồn tại) AND (bcrypt khớp).
    - Success (khớp) → Step 6
    - Failure (không khớp, hoặc user không tồn tại) → Step 5

Step 5 — Ghi nhận lần thất bại và quyết định khoá (feature này, MỘT câu lệnh atomic)
- Trigger: Step 4 thất bại
- Pre-condition: —
- Call: LoginRepository.RecordLoginFailure(phone, maxAttempts=5, lockDuration=15m, resetWindow)
        → INSERT INTO login_attempts (phone_number, failed_count, last_failed_at, updated_at)
          VALUES ($1, 1, now(), now())
          ON CONFLICT (phone_number) DO UPDATE SET
            failed_count   = CASE WHEN <hàng đã stale> THEN 1 ELSE login_attempts.failed_count + 1 END,
            locked_until   = CASE WHEN <count mới> >= $max THEN now() + $lock ELSE NULL END,
            last_failed_at = now(), updated_at = now()
          RETURNING failed_count, locked_until
  <hàng đã stale> = (locked_until IS NOT NULL AND locked_until <= now()) — khoá cũ đã hết, bắt đầu
  đếm lại từ 1 — HOẶC (last_failed_at < now() - resetWindow) — xem marker bên dưới.
  Confirmed: một câu lệnh UPSERT duy nhất nên hai request sai đồng thời trên cùng SĐT bị serialize
  bởi row lock của Postgres — 5 request song song không thể cùng đọc "count = 0". Không cần
  transaction nhiều câu lệnh.
  Confirmed (chốt 2026-09-21): cửa sổ đếm "liên tiếp" là **15 phút** (`FailureResetWindow` = `LockDuration`).
  Lưu ý về tác dụng: khoá hết hạn cũng reset bộ đếm nên tốc độ đoán tối đa của kẻ tấn công chỉ ở cỡ
  vài trăm lần/ngày/tài khoản dù cửa sổ dài hay ngắn — muốn chặn thật cần khoá tăng dần (Known gaps).
  Lần thất bại thứ 5 vẫn trả 401 INVALID_CREDENTIALS (nó chỉ đặt locked_until); lần thử thứ 6 mới
  nhận 429 ở Step 2 — cùng nhịp với OTP (AC12 US-001: 3 lần sai, lần thứ 4 mới bị chặn).
    - Success → 401 INVALID_CREDENTIALS, message "Số điện thoại hoặc mật khẩu không đúng"
    - Failure (lỗi DB) → wrapped "record login failure: %w" → 500. Chấp nhận: lỗi DB ở đây làm mất
      một lần đếm nhưng không bao giờ cho qua một lần đăng nhập.
  Hệ quả được chấp nhận có ý thức: một request đã qua Step 2 trước khi khoá được đặt vẫn có thể
  đăng nhập thành công nếu mật khẩu đúng (cửa sổ race rất hẹp, chỉ với người biết mật khẩu đúng —
  không phải đường brute-force).

Step 6 — Tạo session gắn thiết bị (feature này, MỘT transaction)
- Trigger: Step 4 khớp
- Pre-condition: user tồn tại, mật khẩu đúng; usecase truyền TokenIssuer + sessionTTL
- Call: LoginRepository.CreateLoginSession(ctx, user, device, sessionTTL, issuer) → BEGIN
        (a) fingerprint = SHA-256(platform|os_version|device_model|user_agent) — tính ở repository
            (helper deviceFingerprint), KHÔNG nhận từ client
        (b) INSERT INTO user_devices (...) VALUES (...)
            ON CONFLICT (user_id, device_id) DO UPDATE SET platform, device_model, os_version,
            app_version, device_name, ip_address, user_agent, last_seen_at = now()
            RETURNING id
        (c) UPDATE user_sessions SET is_active = false, is_revoked = true, revoked_at = now(),
            revoked_reason = 'superseded_by_login'
            WHERE device_id = <id từ (b)> AND is_revoked = false
        (d) sessionID = uuid.NewString() (cùng cách CreateUserAndSession của US-001); issuer(userID, sessionID)
            → access (mang extra{sid}) + refresh
        (e) INSERT INTO user_sessions (user_id, device_id, session_id, refresh_token_hash,
            token_version = 1, expires_at = now() + sessionTTL)
        (f) DELETE FROM login_attempts WHERE phone_number = $1   -- reset đếm (AC7)
  Confirmed (quyết định thiết kế):
    - (b) lấy row lock trên hàng user_devices, nên hai login đồng thời từ cùng một thiết bị bị
      serialize ở đây: lần thứ hai chờ lần đầu commit rồi (c) revoke đúng session vừa tạo. Partial
      unique index uq_user_sessions_active_device là backstop, không phải cơ chế chính.
    - (c) chỉ đụng session của ĐÚNG device này (AC3, AC4); session của device khác và session
      US-001 (device_id NULL) không bị chạm.
    - Token được mint TRONG tx qua closure `TokenIssuer` (cùng lý do như US-001: lỗi mint làm rollback
      cả session). `sessionID` phải được sinh **trước** khi mint vì access token mang claim `sid`
      để `logout` tìm đúng session. Chống trùng token do `jti` của `pkg/jwt` đảm nhiệm, không phải `sid`.
    - (f) nằm trong cùng tx: login đúng mà tx rollback thì bộ đếm không bị xoá nhầm.
    - Success → COMMIT → Step 7
    - Failure (issuer trả lỗi) → rollback cả (b)-(f) → wrapped "issue tokens: %w" → 500
    - Failure (unique violation trên refresh_token_hash / uq_user_sessions_active_device) →
      rollback → 500 (không thể xảy ra nếu `jti` đã có ở `pkg/jwt` và (b) serialize — nếu xảy ra là
      bug, không phải lỗi người dùng)
    - Failure (lỗi DB khác) → wrapped "create login session: %w" → 500; rollback toàn bộ

Step 7 — Trả response (feature này)
- Trigger: Step 6 COMMIT
- Call: handler map LoginResult → JSON
    - Success → 200 { access_token, refresh_token, user } (user = userResponse của US-001, KHÔNG
      lộ password_hash)
  Access token: GenerateAccessToken(userID, user.Email, userRole, extra{"sid": sessionID}), sống 15 phút (AC8); closure nằm ở `LoginUsecase.tokenIssuer`.
```

## Step-by-step Flow — Refresh: `POST /v1/auth/refresh`

```
Step 1 — Validate request (feature này)
- Trigger: LoginHandler nhận POST /v1/auth/refresh
- Pre-condition: DecodeJSONBodyStrict pass
- Call: refresh_token != "" → jwtService.ValidateToken(refresh_token), yêu cầu TokenType = refresh
    - Success → Step 2
    - Failure (body hỏng / thiếu field) → 400 (mã từ writeDecodeError)
    - Failure (chữ ký sai, hết hạn, sai loại token) → ErrRefreshTokenInvalid → 401
      REFRESH_TOKEN_INVALID; không chạm DB (AC10)

Step 2 — Tra session theo hash (feature này)
- Trigger: Step 1 pass
- Pre-condition: chữ ký + hạn của JWT hợp lệ
- Call: LoginRepository.FindSessionByRefreshHash(domain.HashRefreshToken(token)) →
        SELECT ... FROM user_sessions
        WHERE refresh_token_hash = $1 OR previous_refresh_token_hash = $1
        trả (*Session, matchedPrevious bool, error)
    - Success (khớp refresh_token_hash, session active, chưa revoke, expires_at > now()) → Step 3
    - Failure (không có hàng) → ErrRefreshTokenInvalid → 401 REFRESH_TOKEN_INVALID
    - Failure (khớp hàng nhưng is_revoked / hết hạn) → ErrRefreshTokenInvalid → 401 (cùng mã, AC10)
    - Failure (khớp previous_refresh_token_hash — token đã rotate bị dùng lại, AC11) → Step 2b
    - Failure (lỗi DB) → wrapped "find session: %w" → 500

Step 2b — Phát hiện reuse (feature này)
- Trigger: Step 2 khớp previous_refresh_token_hash
- Call: LoginRepository.RevokeSession(userID, sessionID, reason='refresh_token_reuse')
        → UPDATE user_sessions SET is_active = false, is_revoked = true, revoked_at = now(),
          revoked_reason = $3 WHERE user_id = $1 AND session_id = $2 AND is_revoked = false
    - Success → 401 REFRESH_TOKEN_INVALID (người gọi không phân biệt được với các case khác)
    - Failure (lỗi DB) → 500; vẫn KHÔNG cấp token nào
  Confirmed (đã chốt qua phỏng vấn 2026-09-20): revoke **chỉ session của device đó**, không revoke
  mọi session của user (AC11). Chỉ phủ một đời token (previous_refresh_token_hash chỉ giữ một
  giá trị) — xem Known gaps.

Step 3 — Rotate refresh token (feature này, MỘT transaction)
- Trigger: Step 2 khớp refresh_token_hash của session hợp lệ
- Pre-condition: session active, chưa revoke, chưa hết hạn
- Call: LoginRepository.RotateSession(ctx, sessionRowID, oldHash, sessionTTL, issuer) → BEGIN
        (a) SELECT ... FROM user_sessions WHERE id = $1 FOR UPDATE
            → nếu refresh_token_hash <> oldHash hoặc is_revoked: một request refresh khác đã thắng
            (hoặc logout xen vào) → rollback, ErrRefreshTokenInvalid
        (b) issuer(userID, session.SessionID) → access (cùng `sid`) + refresh mới (mỗi token có `jti` riêng nên khác token cũ dù cùng giây)
        (c) UPDATE user_sessions SET previous_refresh_token_hash = refresh_token_hash,
            refresh_token_hash = <hash mới>, token_version = token_version + 1,
            last_activity_at = now(), updated_at = now(), expires_at = now() + sessionTTL
  Confirmed: FOR UPDATE ở (a) làm hai request refresh đồng thời với CÙNG một token bị serialize —
  đúng một request thắng, request còn lại thấy hash đã đổi và nhận 401 (không sinh hai refresh
  token hợp lệ từ một token). Vì thế request thua sẽ rơi vào Step 2 lần sau như một "reuse".
  Access token mới cần email + role: usecase tra users theo user_id trong cùng tx hoặc trước
  Step 3 (cần thêm bước đọc — quyết định implement, không đổi hình dạng flow).
  Confirmed (chốt 2026-09-21): `expires_at = now() + sessionTTL` là **sliding** — mỗi lần refresh gia
  hạn thêm 7 ngày, người dùng hoạt động đều không bị đăng xuất. Xoay vòng token + phát hiện reuse (Step
  2b) vẫn chặn token bị đánh cắp. Trần tuyệt đối (`absolute_expires_at`) để lại sau.
    - Success → COMMIT → 200 { access_token, refresh_token }
    - Failure (thua race / token vừa bị rotate / revoke xen vào) → rollback → 401
      REFRESH_TOKEN_INVALID
    - Failure (issuer lỗi) → rollback → wrapped "issue tokens: %w" → 500
    - Failure (lỗi DB khác) → rollback → 500
```

## Step-by-step Flow — Logout: `POST /v1/auth/logout`

```
Step 1 — Xác thực access token (middleware, thuộc `pkg/jwt/middleware`)
- Trigger: request tới POST /v1/auth/logout
- Pre-condition: route được bọc AuthenticateFunc với RequireTokenType = "access"
- Call: HTTPAuthMiddleware.Authenticate → extract Bearer → jwtService.ValidateToken → kiểm
        TokenType == access → jwtcontext.WithClaims / WithUserID
    - Success → Step 2
    - Failure (thiếu header / token sai / hết hạn / là refresh token) → ErrorHandler (WriteAuthError) →
      401 UNAUTHORIZED; KHÔNG chạm DB (AC15)
  Không có body để decode. Access token hết hạn → 401; client gọi refresh rồi thử lại logout.

Step 2 — Lấy session từ claims (feature này)
- Trigger: Step 1 pass
- Pre-condition: claims đã có trong context
- Call: handler → jwtcontext.GetClaims(ctx) → userID = claims.UserID; sid = claims.Extra["sid"].(string)
    - Success (sid là string không rỗng) → Step 3
    - Failure (không có claims, hoặc thiếu/sai kiểu sid — vd token cấp trước khi sửa US-001) →
      401 UNAUTHORIZED (AC15); không chạm DB
  Confirmed: `sid` lấy từ token đã được middleware verify chữ ký, không lấy từ body/query — client
  không tự khai được session cần logout.

Step 3 — Thu hồi session (feature này)
- Trigger: Step 2 pass
- Pre-condition: userID + sessionID đã có
- Call: LoginRepository.RevokeSession(ctx, userID, sessionID, reason='logout') →
        UPDATE user_sessions SET is_active = false, is_revoked = true, revoked_at = now(),
        revoked_reason = 'logout' WHERE user_id = $1 AND session_id = $2 AND is_revoked = false
  Confirmed (quyết định thiết kế): điều kiện có CẢ user_id lẫn session_id — một user không thu hồi
  được session của người khác dù đoán được session_id (AC14). Không kiểm số hàng bị đổi: 0 hàng
  (đã revoke / không tồn tại) vẫn là thành công (idempotent).
    - Success (1 hàng đổi) → 200 {"success": true}
    - Success (0 hàng đổi) → 200 {"success": true}, không đổi state (AC14)
    - Failure (lỗi DB) → wrapped "revoke session: %w" → 500
  Sau logout, access token vừa dùng (và các access token khác của session đó) vẫn qua được middleware
  tới khi hết hạn (≤ 15 phút) — đã chốt chấp nhận, xem Known gaps.
```

## Deliberately stubbed for now

| Thứ | Trạng thái | Ghi chú |
|---|---|---|
| Code login | **Đã có** (2026-09-21) | `domain/login.go`, `usecase/login_usecase.go`, `repository/postgres_login_repository.go`, `delivery/http/login_handler.go`; `cmd/main.go` wire 3 route mới (6 route tổng). |
| `migrations/0002_*.sql` | **Đã có, đã apply thật** | `0001` không bị sửa. Áp lên Postgres 16 (Docker) bằng `psql -f`, chạy được E2E. |
| `docs/specs/api/openapi.yaml` | **Đã có 3 path** | Viết sau khi handler chạy thật (curl), khớp response thật. |
| ERD dưới `docs/specs/db/` | **Chưa có** | Thư mục rỗng; schema `0002` chỉ được mô tả trong file migration + `migrations/README.md`. |
| Dọn `login_attempts` | **Chưa có** | Hàng của SĐT không tồn tại tích tụ vô hạn; cần cron cùng với dọn `otp_codes` (chưa có scheduler trong repo). MVP chấp nhận. |
| Test E2E tự động | **Chưa có** | Đã chạy tay bằng curl (18 bước). Test black-box cần `testcontainers-go` (chưa có trong `go.mod`). |

## Known gaps carried by this feature

- **Access token sống tới hết hạn sau logout / revoke.** JWT là stateless; `pkg/jwt/middleware` chỉ verify chữ ký + hạn, không tra `user_sessions`. Sau `logout` hoặc reuse-revoke, access token đã cấp còn dùng được tối đa 15 phút. **Đã chốt 2026-09-21: chấp nhận** (không tra session mỗi request). Nếu sau này cần chặn ngay, middleware phải tra session (chi phí một truy vấn/request, cần claim `sid`) hoặc denylist theo `jti` — xem xét khi wire `RequireAuth` cho US-004.
- **Logout cần access token còn hạn.** Access token hết hạn → 401; client phải `refresh` rồi mới logout được. Nếu app muốn "đăng xuất" khi mất mạng/hết hạn, nó chỉ xoá token cục bộ — session phía server vẫn sống tới khi hết hạn. Chấp nhận ở MVP.
- **401 của middleware phải khớp envelope.** Lúc viết spec, `pkg/jwt/middleware` mặc định trả `{"error":"..."}` phẳng nên bắt buộc dùng `ErrorHandler` tuỳ biến (`WriteAuthError`). Từ `api-response-envelope` v1.0, mặc định đã là envelope `utils.ApiResponse` với code `UNAUTHORIZED`, nhưng message phân biệt lý do (`Token has expired`, …) — `WriteAuthError` vẫn giữ để message chỉ là `unauthorized`.
- **Reuse detection chỉ phủ một đời.** `previous_refresh_token_hash` giữ đúng một giá trị; token cũ hơn hai đời không khớp hàng nào nên chỉ nhận 401 thường, không kích hoạt revoke. Muốn phủ đủ phải có bảng lịch sử hash — ngoài phạm vi MVP.
- **Không giới hạn số session active mỗi user.** Mỗi thiết bị mới thêm một session; không có trần. Mặc định chấp nhận ở MVP (AC3: đăng nhập thiết bị mới không đăng xuất thiết bị cũ).
- **Khoá login là DoS-được với một SĐT cụ thể.** Ai cũng có thể gửi 5 mật khẩu sai để khoá SĐT của người khác 15 phút (đánh đổi thường gặp của khoá theo tài khoản). Giảm nhẹ bằng rate limit theo IP ở middleware — Deferred, xem tasks.md.
- **`fingerprint` không phải xác thực thiết bị.** Nó chỉ gom các lần login từ "cùng loại thiết bị" (`platform|os_version|device_model|user_agent`); hai máy cùng model + OS + UA có cùng fingerprint và sẽ đá session của nhau. Chấp nhận ở MVP; cần device id do client sinh để tách được.
- **`otp_codes.purpose = 'login'` chưa dùng.** Hằng `OTPPurposeLogin` và nhánh `resend-otp` cho `login` vẫn trả `ErrUnsupportedOTPPurpose` — login bằng OTP ngoài scope.
- **`refresh` không có rate limit riêng.** Chỉ được bảo vệ bởi độ khó đoán của JWT ký + tra hash. Rate limit theo IP — Deferred.
- **~~Bất đồng giữa nguồn dữ liệu về TTL refresh token.~~** `onboarding/requirements.md` §Ghi chú kỹ thuật từng ghi 30 ngày trong khi `config.dev.yaml` và `0001` dùng 168h (7 ngày). Spec này theo 7 ngày (AC8). Dòng cũ **đã được sửa** ở onboarding v5.2 — không còn bất đồng.
