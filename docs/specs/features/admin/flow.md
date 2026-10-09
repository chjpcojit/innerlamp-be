# Flow — admin

Version: 1.0
Last updated: 2026-09-29
Status: implemented (2026-09-29) — vẽ lại đúng sáu "Step-by-step Flow" của `design.md` (Bootstrap, Login, Đổi mật khẩu, Bảo vệ route, Tạo tài khoản, Reset mật khẩu) bằng Mermaid; không thêm, không bớt bước.
Module: `backoffice`

> See `changelog.md` in this folder for the version history of this spec.

Sáu sequence diagram dưới đây là gương 1:1 của sáu section "Step-by-step Flow" trong `design.md` (cùng tên, cùng thứ tự). Mỗi message trace về đúng một Step; ngược lại mọi Step của `design.md` đều xuất hiện ở đây. Các flow "Đổi mật khẩu", "Tạo tài khoản", "Reset mật khẩu" dùng lại flow "Bảo vệ route" ở Step 1 (vẽ gộp thành một message `MW`).

## Sequence — Bootstrap admin khi khởi động

```mermaid
sequenceDiagram
    autonumber
    participant M as cmd/main.go
    participant U as AdminAuthUsecase
    participant R as AdminRepository
    participant DB as Postgres

    M->>M: Step 1 — cfg.BootstrapAdmin.Enabled?
    alt enabled = false
        Note over M: bỏ qua, không tạo gì
    else enabled = true
        M->>U: EnsureBootstrapAdmin(email, password, name)
        U->>U: Step 2 — NormalizeEmail + kiểm rỗng + bcrypt băm
        alt email/password rỗng hoặc lỗi bcrypt
            U--xM: error
            M->>M: log.Fatalf — service không lên
        end
        rect rgb(235, 240, 255)
            Note over R,DB: Step 3 — MỘT statement (kiểm rỗng + insert)
            U->>R: CreateBootstrapIfEmpty(email, hash, name)
            R->>DB: INSERT … SELECT … WHERE NOT EXISTS (SELECT 1 FROM admins) ON CONFLICT DO NOTHING
            alt RowsAffected = 1
                R-->>U: created = true
                U->>U: log WARN "bootstrap admin created…"
            else RowsAffected = 0 (đã có admin)
                R-->>U: created = false (no-op)
            else lỗi DB
                R--xU: error
                U--xM: error → log.Fatalf
            end
        end
    end
```

## Sequence — Login: `POST /v1/admin/auth/login`

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as AdminAuthHandler<br/>(delivery/http)
    participant U as AdminAuthUsecase
    participant R as AdminRepository
    participant DB as Postgres

    C->>H: POST /v1/admin/auth/login {email, password}
    H->>H: Step 1 — DecodeJSONBodyStrict
    alt JSON hỏng / field lạ / quá lớn
        H--xC: 400 VALIDATION_ERROR / 413 PAYLOAD_TOO_LARGE
    end
    H->>U: Login(ctx, input)
    U->>U: Step 2 — NormalizeEmail, email/password rỗng?
    alt rỗng
        U--xH: ErrInvalidCredentials
        H--xC: 401 INVALID_CREDENTIALS (không chạm DB)
    end
    U->>R: Step 3 — FindByEmail(email)
    R->>DB: SELECT … FROM admins WHERE email = $1
    alt lỗi DB
        R--xU: error
        H--xC: 500 INTERNAL_ERROR
    else không có row
        R-->>U: ErrAdminNotFound → admin = nil (đi tiếp)
    else có row
        R-->>U: Admin
    end
    U->>U: Step 4 — bcrypt.Compare(hash thật hoặc dummy, password)
    alt admin nil / sai mật khẩu / is_active = false
        U--xH: ErrInvalidCredentials
        H--xC: 401 INVALID_CREDENTIALS (cùng thông điệp, cùng thời gian)
    end
    U->>U: Step 5 — MustChangePassword && hạn tạm đã qua? (chỉ sau khi mật khẩu khớp)
    alt hết hạn
        U--xH: ErrPasswordExpired
        H--xC: 401 PASSWORD_EXPIRED
    end
    U->>U: Step 6 — GenerateAccessToken (extra must_change_password nếu còn cờ)
    alt lỗi ký token
        U--xH: error
        H--xC: 500 INTERNAL_ERROR
    end
    U->>R: Step 7 — TouchLastLogin(id, now)
    R->>DB: UPDATE admins SET last_login_at
    alt lỗi
        Note over U: log WARN, VẪN thành công
    end
    U-->>H: AdminLoginResult
    H-->>C: 200 {access_token, admin, must_change_password}
```

## Sequence — Đổi mật khẩu: `POST /v1/admin/auth/change-password`

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant MW as Middleware<br/>(JWT + session + guard)
    participant H as AdminAuthHandler<br/>(delivery/http)
    participant U as AdminAuthUsecase
    participant R as AdminRepository
    participant DB as Postgres

    C->>MW: POST /v1/admin/auth/change-password (Bearer)
    MW->>MW: Step 1 — JWT → RequireActiveSession → RequireStaffAllowPasswordChange
    alt token / phiên / role không hợp lệ
        MW--xC: 401 UNAUTHORIZED / 403 FORBIDDEN
    end
    MW->>H: ChangePassword
    H->>H: Step 2 — adminID = claims.UserID, DecodeJSONBodyStrict {current_password, new_password}
    alt UserID không phải số
        H--xC: 401 UNAUTHORIZED
    else body lỗi / field lạ
        H--xC: 400 VALIDATION_ERROR
    end
    H->>U: ChangePassword(ctx, input)
    U->>R: Step 3 — FindByID(adminID)
    R->>DB: SELECT … FROM admins WHERE id = $1
    alt không có row / is_active = false
        U--xH: ErrInvalidCredentials
        H--xC: 401 INVALID_CREDENTIALS
    else lỗi DB
        H--xC: 500 INTERNAL_ERROR
    end
    U->>U: Step 4 — bcrypt so current_password, len(new) ≥ 12 và new ≠ current
    alt sai mật khẩu hiện tại
        U--xH: ErrWrongCurrentPassword
        H--xC: 400 WRONG_CURRENT_PASSWORD
    else quá ngắn / trùng cũ
        U--xH: ErrWeakPassword
        H--xC: 400 WEAK_PASSWORD
    end
    rect rgb(235, 240, 255)
        Note over R,DB: Step 5 — MỘT statement UPDATE
        U->>R: UpdatePassword(id, hash mới)
        R->>DB: UPDATE admins SET password_hash, must_change_password=FALSE, tokens_valid_after=now(), temp_password_expires_at=NULL
        alt lỗi DB
            R--xU: error
            H--xC: 500 INTERNAL_ERROR
        end
    end
    U->>U: Step 6 — issueToken (không có extra)
    U-->>H: AdminLoginResult (token mới)
    H-->>C: 200 {access_token, admin, must_change_password:false}
```

## Sequence — Bảo vệ route: mọi request `/v1/admin/*` trừ login

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant J as JWT middleware<br/>(pkg/jwt/middleware)
    participant S as RequireActiveSession<br/>(delivery/http/session.go)
    participant U as AdminAuthUsecase
    participant R as AdminRepository
    participant DB as Postgres
    participant G as guard<br/>(RequireRole / RequireStaff)
    participant H as Handler của route

    C->>J: request + Authorization: Bearer
    J->>J: Step 1 — verify chữ ký + audience + exp, token_type = access
    alt không token / refresh token / khác secret-audience (vd appService)
        J--xC: 401 UNAUTHORIZED
    end
    J->>S: claims trong context
    S->>U: Step 2 — ValidateSession(adminID, iat)
    U->>R: FindByID(adminID)
    R->>DB: SELECT … FROM admins WHERE id = $1
    alt không có admin / is_active = false / iat < tokens_valid_after (cắt về giây)
        U--xS: ErrSessionRevoked
        S--xC: 401 UNAUTHORIZED
    else lỗi DB
        S--xC: 500 INTERNAL_ERROR (không cho đi tiếp)
    end
    S->>G: còn hiệu lực
    G->>G: Step 3 — role hợp lệ? cờ must-change?
    alt role không hợp lệ / không phải admin
        G--xC: 403 FORBIDDEN
    else token còn cờ must-change trên route không cho phép
        G--xC: 403 PASSWORD_CHANGE_REQUIRED
    end
    G->>H: cho qua
```

## Sequence — Tạo tài khoản: `POST /v1/admin/users`

```mermaid
sequenceDiagram
    autonumber
    participant C as Client (admin)
    participant MW as Middleware<br/>(JWT + session + RequireRole admin)
    participant H as AdminUserHandler<br/>(delivery/http)
    participant U as AdminUserUsecase
    participant R as AdminRepository
    participant DB as Postgres
    participant ML as pkg/mailer

    C->>MW: POST /v1/admin/users (Bearer)
    MW->>MW: Step 1 — flow "Bảo vệ route", role = admin
    alt 401 / 403 FORBIDDEN / 403 PASSWORD_CHANGE_REQUIRED
        MW--xC: lỗi (use case KHÔNG được gọi)
    end
    MW->>H: Create
    H->>H: Step 2 — DecodeJSONBodyStrict {email, name, role}
    alt JSON hỏng / field lạ / quá lớn
        H--xC: 400 VALIDATION_ERROR / 413
    end
    H->>U: CreateAccount(ctx, input)
    U->>U: Step 3 — NormalizeEmail + mail.ParseAddress, name ≠ "", ValidRole
    alt đầu vào sai
        U--xH: ErrInvalidAccountInput
        H--xC: 400 VALIDATION_ERROR (không chạm DB, không gửi email)
    end
    U->>U: Step 4 — mật khẩu tạm (fixed ⚠️ DEV ONLY hoặc random 16 ký tự), bcrypt, expiry = now + TTL
    rect rgb(235, 240, 255)
        Note over R,DB: Step 5 — MỘT statement INSERT
        U->>R: CreateAccount(email, hash, name, role, expiry)
        R->>DB: INSERT … ON CONFLICT (email) DO NOTHING RETURNING id
        alt không có row trả về
            R--xU: ErrEmailTaken
            H--xC: 409 EMAIL_TAKEN (không gửi email)
        else lỗi DB
            H--xC: 500 INTERNAL_ERROR
        end
    end
    U->>ML: Step 6 — SendAccountInviteEmail(email, name, tempPassword)
    Note over U,ML: KHÔNG có transaction bao Step 5 + 6 — bù trừ bằng Step 7
    alt gửi lỗi
        U->>R: Step 7 — DeleteByID(id)
        R->>DB: DELETE FROM admins WHERE id = $1
        Note over R,DB: nếu xoá cũng lỗi → log ERROR, còn tài khoản mồ côi
        U--xH: ErrInviteEmailFailed
        H--xC: 502 EMAIL_SEND_FAILED
    else gửi ok
        U-->>H: CreateAccountResult (không kèm mật khẩu)
        H-->>C: 201 {admin, must_change_password:true}
    end
```

## Sequence — Reset mật khẩu: `POST /v1/admin/users/{id}/reset-password`

```mermaid
sequenceDiagram
    autonumber
    participant C as Client (admin)
    participant MW as Middleware<br/>(JWT + session + RequireRole admin)
    participant H as AdminUserHandler<br/>(delivery/http)
    participant U as AdminUserUsecase
    participant R as AdminRepository
    participant DB as Postgres
    participant ML as pkg/mailer

    C->>MW: POST /v1/admin/users/{id}/reset-password (Bearer)
    MW->>MW: Step 1 — flow "Bảo vệ route", role = admin
    alt 401 / 403
        MW--xC: lỗi (use case KHÔNG được gọi)
    end
    MW->>H: ResetPassword
    H->>H: Step 2 — actorID = claims.UserID, targetID = PathValue("id")
    alt targetID không phải số dương
        H--xC: 400 VALIDATION_ERROR (details {field:id})
    end
    H->>U: ResetPassword(ctx, actorID, targetID)
    U->>U: Step 3 — actorID == targetID?
    alt tự reset
        U--xH: ErrCannotResetSelf
        H--xC: 409 CANNOT_RESET_SELF (không chạm DB)
    end
    U->>U: Step 4 — mật khẩu tạm mới + bcrypt + expiry = now + TTL
    rect rgb(235, 240, 255)
        Note over R,DB: Step 5 — MỘT statement UPDATE (mật khẩu, cờ, hạn, thu hồi phiên)
        U->>R: ResetTempPassword(targetID, hash, expiry)
        R->>DB: UPDATE admins SET password_hash, must_change_password=TRUE, temp_password_expires_at, tokens_valid_after=now() WHERE id RETURNING email,name,role,is_active
        alt không có row
            R--xU: ErrAdminNotFound
            H--xC: 404 ADMIN_NOT_FOUND (không gửi email)
        else lỗi DB
            H--xC: 500 INTERNAL_ERROR
        end
    end
    U->>ML: Step 6 — SendAccountInviteEmail(email, name, tempPassword)
    alt gửi lỗi
        Note over U,ML: KHÔNG hoàn tác được — admin phải reset lại
        U--xH: ErrInviteEmailFailed
        H--xC: 502 EMAIL_SEND_FAILED
    else gửi ok
        U-->>H: CreateAccountResult (không kèm mật khẩu)
        H-->>C: 200 {admin, must_change_password:true}
    end
```
