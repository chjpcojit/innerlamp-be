# Flow — login

Version: 1.13
Last updated: 2026-10-10
Status: implemented (2026-09-21) — vẽ lại đúng ba "Step-by-step Flow" của `design.md` (Login,
Refresh, Logout) bằng Mermaid; không thêm, không bớt bước so với `design.md`.
Module: `identity`

> See `changelog.md` in this folder for the version history of this spec.

Ba sequence diagram dưới đây là gương 1:1 của ba section "Step-by-step Flow" trong `design.md`
(cùng tên, cùng thứ tự). Mỗi mũi tên/note trace về đúng một Step (hoặc sub-step `(a)`–`(f)`) của
`design.md`; ngược lại mọi Step của `design.md` đều xuất hiện ở đây. `design.md` §Request flow trỏ
sang file này thay vì giữ song song một bản ASCII.

## Sequence — Login: `POST /v1/auth/login`

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as LoginHandler<br/>(delivery/http)
    participant U as LoginUsecase
    participant R as LoginRepository
    participant DB as Postgres

    C->>H: POST /v1/auth/login
    H->>H: DecodeJSONBodyStrict + điền Device.IPAddress/UserAgent từ *http.Request
    H->>U: Login(ctx, input)

    U->>U: Step 1 — NormalizePhone + password != "" + device.platform/device_model != ""
    alt định dạng hợp lệ
        Note over U: → Step 2
    else định dạng sai
        U--xH: ErrInvalidPhone / ErrInvalidPassword / ErrInvalidDevice
        H--xC: 400 INVALID_PHONE / INVALID_PASSWORD / INVALID_DEVICE
        Note over U,DB: KHÔNG chạm DB, KHÔNG đếm login_attempts (AC5, AC6)
    end

    U->>R: Step 2 — FindLoginAttempt(phone)
    R->>DB: SELECT phone_number, failed_count, locked_until, last_failed_at FROM login_attempts WHERE phone_number = $1
    alt không có hàng / locked_until IS NULL / locked_until <= now()
        R-->>U: nil hoặc attempt hết hạn
        Note over U: → Step 3
    else locked_until > now()
        R-->>U: LoginAttempt{LockedUntil}
        U--xH: LoginLockedError{LockedUntil}
        H--xC: 429 LOGIN_LOCKED {locked_until}
        Note over U,R: DỪNG — không tra user, không so mật khẩu (AC7)
    end

    U->>R: Step 3 — FindUserByPhone(phone)
    R->>DB: SELECT <cột tường minh> FROM users WHERE phone_number = $1
    alt có hàng
        R-->>U: user + password_hash thật
    else không có hàng (chưa đăng ký / chỉ pending_registration)
        R-->>U: ErrUserNotFound (nội bộ)
        U->>U: dùng dummyPasswordHash thay password_hash thật
    else lỗi DB
        R--xU: wrapped "find user: %w"
        U--xH: lỗi
        H--xC: 500 INTERNAL_ERROR
    end

    U->>U: Step 4 — bcrypt.CompareHashAndPassword(hash, password) — luôn đúng 1 lần (AC2, timing)
    alt khớp (user tồn tại AND bcrypt khớp)
        Note over U: → Step 6
    else không khớp / user không tồn tại
        U->>R: Step 5 — RecordLoginFailure(phone, maxAttempts=5, lockDuration=15m, resetWindow=15m)
        R->>DB: INSERT INTO login_attempts (...) VALUES (...) ON CONFLICT (phone_number) DO UPDATE SET failed_count=.., locked_until=.. RETURNING failed_count, locked_until
        alt success
            R-->>U: failed_count, locked_until
            U--xH: ErrInvalidCredentials
            H--xC: 401 INVALID_CREDENTIALS
        else lỗi DB
            R--xU: wrapped "record login failure: %w"
            U--xH: lỗi
            H--xC: 500
        end
    end

    U->>R: Step 6 — CreateLoginSession(ctx, user, device, sessionTTL, issuer)
    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction
        R->>R: (a) fingerprint = SHA-256(platform|os_version|device_model|user_agent) — tính ở repository, không nhận từ client
        R->>DB: (b) INSERT user_devices (...) ON CONFLICT (user_id, device_id) DO UPDATE ... RETURNING id
        R->>DB: (c) UPDATE user_sessions SET is_revoked=true, revoked_reason='superseded_by_login' WHERE device_id=<id từ b> AND is_revoked=false
        R->>R: (d) sessionID = uuid.NewString(); issuer(userID, sessionID) → access{sid} + refresh — mint TRONG tx
        R->>DB: (e) INSERT user_sessions (user_id, device_id, session_id, refresh_token_hash, token_version=1, expires_at)
        R->>DB: (f) DELETE FROM login_attempts WHERE phone_number = $1 (reset đếm, AC7)
        alt COMMIT
            R-->>U: LoginResult{access_token, refresh_token, user}
        else issuer lỗi
            R--xU: rollback (b)-(f) → wrapped "issue tokens: %w"
        else unique violation (refresh_token_hash / uq_user_sessions_active_device)
            R--xU: rollback → 500 (bug, không phải lỗi người dùng)
        else lỗi DB khác
            R--xU: rollback → wrapped "create login session: %w"
        end
    end

    alt Step 6 COMMIT
        U-->>H: LoginResult
        H-->>C: Step 7 — 200 {access_token, refresh_token, user}
    else Step 6 lỗi
        U--xH: lỗi
        H--xC: 500
    end
```

## Sequence — Refresh: `POST /v1/auth/refresh`

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as LoginHandler<br/>(delivery/http)
    participant U as LoginUsecase
    participant R as LoginRepository
    participant DB as Postgres

    C->>H: POST /v1/auth/refresh
    H->>H: DecodeJSONBodyStrict
    H->>U: Refresh(ctx, input)

    U->>U: Step 1 — refresh_token != "" → jwtService.ValidateToken(refresh_token), yêu cầu TokenType = refresh
    alt hợp lệ
        Note over U: → Step 2
    else body hỏng / thiếu field
        U--xH: lỗi decode
        H--xC: 400
    else chữ ký sai / hết hạn / sai loại token
        U--xH: ErrRefreshTokenInvalid
        H--xC: 401 REFRESH_TOKEN_INVALID
        Note over U,DB: KHÔNG chạm DB (AC10)
    end

    U->>R: Step 2 — FindSessionByRefreshHash(domain.HashRefreshToken(token))
    R->>DB: SELECT ... FROM user_sessions WHERE refresh_token_hash = $1 OR previous_refresh_token_hash = $1
    alt khớp refresh_token_hash, active, chưa revoke, chưa hết hạn
        R-->>U: (Session, matchedPrevious=false)
        Note over U: → Step 3
    else không có hàng
        R-->>U: ErrRefreshTokenInvalid
        U--xH: ErrRefreshTokenInvalid
        H--xC: 401 REFRESH_TOKEN_INVALID
    else khớp hàng nhưng is_revoked / hết hạn
        R-->>U: ErrRefreshTokenInvalid
        U--xH: ErrRefreshTokenInvalid
        H--xC: 401 REFRESH_TOKEN_INVALID
    else khớp previous_refresh_token_hash (token đã rotate, bị dùng lại)
        R-->>U: (Session, matchedPrevious=true)
        Note over U: → Step 2b (AC11)
    else lỗi DB
        R--xU: wrapped "find session: %w"
        U--xH: lỗi
        H--xC: 500
    end

    U->>R: Step 2b — RevokeSession(userID, sessionID, reason='refresh_token_reuse')
    R->>DB: UPDATE user_sessions SET is_active=false, is_revoked=true, revoked_at=now(), revoked_reason=$3 WHERE user_id=$1 AND session_id=$2 AND is_revoked=false
    alt success
        R-->>U: ok — chỉ revoke session của device đó (AC11)
        U--xH: ErrRefreshTokenInvalid
        H--xC: 401 REFRESH_TOKEN_INVALID
        Note over U,DB: KHÔNG cấp token nào
    else lỗi DB
        R--xU: lỗi
        U--xH: lỗi
        H--xC: 500 — vẫn KHÔNG cấp token nào
    end

    U->>R: Step 3 — RotateSession(ctx, sessionRowID, oldHash, sessionTTL, issuer)
    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction
        R->>DB: (a) SELECT ... FROM user_sessions WHERE id = $1 FOR UPDATE
        alt refresh_token_hash == oldHash AND chưa revoke
            R->>DB: (b1) FindUserByID(userID) — lấy email/role để mint access token (method thứ 8 của LoginRepository, bước đọc thêm)
            R->>R: (b2) issuer(userID, session.SessionID) → access{sid} (jti riêng) + refresh mới
            R->>DB: (c) UPDATE previous_refresh_token_hash=hash cũ, refresh_token_hash=hash mới, token_version+=1, last_activity_at=now(), expires_at=now()+sessionTTL (sliding, AC9)
            R-->>U: COMMIT → RefreshResult{access_token, refresh_token}
        else hash lệch / đã revoke (request refresh khác đã thắng, hoặc logout xen vào)
            R--xU: rollback → ErrRefreshTokenInvalid
        else issuer lỗi
            R--xU: rollback → wrapped "issue tokens: %w"
        else lỗi DB khác
            R--xU: rollback → 500
        end
    end

    alt Step 3 COMMIT
        U-->>H: RefreshResult
        H-->>C: 200 {access_token, refresh_token}
    else Step 3 lỗi
        U--xH: lỗi
        H--xC: 401 REFRESH_TOKEN_INVALID / 500
    end
```

## Sequence — Logout: `POST /v1/auth/logout`

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as HTTPAuthMiddleware<br/>(pkg/jwt/middleware)
    participant H as LoginHandler<br/>(delivery/http)
    participant U as LoginUsecase
    participant R as LoginRepository
    participant DB as Postgres

    C->>M: POST /v1/auth/logout (Authorization: Bearer <access_token>)
    M->>M: Step 1 — extract Bearer → jwtService.ValidateToken → kiểm TokenType == access
    alt hợp lệ
        M->>H: WithClaims / WithUserID → Logout
    else thiếu header / token sai / hết hạn / là refresh token
        M--xC: 401 UNAUTHORIZED (ErrorHandler = WriteAuthError)
        Note over M,DB: KHÔNG chạm DB (AC15)
    end

    H->>H: Step 2 — jwtcontext.GetClaims(ctx) → userID = claims.UserID; sid = claims.Extra["sid"].(string)
    alt sid là string không rỗng
        H->>U: Logout(ctx, LogoutInput{UserID, SessionID: sid})
    else thiếu claims, hoặc sid thiếu/sai kiểu (vd token cấp trước khi sửa US-001)
        H--xC: 401 UNAUTHORIZED
        Note over H,DB: KHÔNG chạm DB (AC15)
    end

    U->>R: Step 3 — RevokeSession(ctx, userID, sessionID, reason='logout')
    R->>DB: UPDATE user_sessions SET is_active=false, is_revoked=true, revoked_at=now(), revoked_reason='logout' WHERE user_id=$1 AND session_id=$2 AND is_revoked=false
    alt 1 hàng đổi
        R-->>U: ok
        U-->>H: ok
        H-->>C: 200 {"success": true}
    else 0 hàng đổi (đã revoke / không tồn tại)
        R-->>U: ok — idempotent, không đổi state (AC14)
        U-->>H: ok
        H-->>C: 200 {"success": true}
    else lỗi DB
        R--xU: wrapped "revoke session: %w"
        U--xH: lỗi
        H--xC: 500
    end
```

## Web `X-API-Key` — mọi route `/v1/auth/*` (v1.12)

Vẽ lại 1:1 §Cross-cutting: web `X-API-Key` của `design.md`. Dùng flowchart vì đây là logic rẽ nhánh, không phải chuỗi gọi qua lại.

```mermaid
flowchart TD
    A[Request vào /v1/auth/*] --> B{Step 1: có header X-API-Key?}
    B -- không --> H[Step 4: chạy handler, chưa đánh dấu web]
    B -- có --> C[Step 2: SHA-256 giá trị gửi, ConstantTimeCompare với key cấu hình]
    C --> D{Step 3: khớp và key đã cấu hình?}
    D -- không --> E[401 INVALID_API_KEY, dừng - AC17]
    D -- có --> F[Đánh dấu web key verified vào context - AC16]
    F --> H
    H --> L{Chỉ Login: device.platform?}
    L -- rỗng --> M[Đặt platform = mobile - AC19]
    L -- web, chưa verified --> E
    L -- web, đã verified --> N[Tiếp tục AC1-AC8]
    L -- giá trị khác --> N
    M --> N
```

## Response cấp token — `verify-otp`, `login`, `refresh` (v1.13)

Vẽ lại 1:1 §Cross-cutting: response cấp token của `design.md`.

```mermaid
sequenceDiagram
    autonumber
    participant C as Client (web / mobile)
    participant N as NoStore.WrapFunc
    participant H as Handler (verify-otp / login / refresh)
    participant U as Usecase

    C->>N: POST /v1/auth/{register/verify-otp | login | refresh}
    N->>N: Step 1 — đặt Cache-Control: no-store và Pragma: no-cache (AC20)
    N->>H: gọi handler
    H->>U: Step 2 — chạy use case như AC1-AC15
    alt thành công
        U-->>H: tokens (+ user với verify-otp / login)
        H->>H: Step 3 — expires_in = int(accessTokenTTL.Seconds()) (AC21, AC22)
        H-->>C: 200/201 {success, data{access_token, refresh_token, expires_in, user?}} + header no-store
    else lỗi
        U--xH: lỗi
        H--xC: 4xx/5xx envelope + header no-store (đặt ở Step 1)
    end
```
