# Design — admin

Version: 1.0
Last updated: 2026-09-29
Status: implemented (2026-09-29) — 6 flow chạy thật qua binary + Postgres và có test usecase/handler (mock). Thiếu: test repository trên DB thật, E2E, `openapi.yaml`, ERD.
Module: `backoffice`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

`/v1/admin/*` (trừ `login`) đều đi qua ba lớp bảo vệ nối tiếp: xác thực JWT (chỉ `access`, audience backoffice) → `RequireActiveSession` (thu hồi phiên) → kiểm role (+ cờ must-change-password). `/healthz` và `login` là public. Feature chạm đủ bốn layer Clean Architecture của `services/backofficeService` (`domain` → `usecase` → `repository` / `delivery/http`, nối dây ở `cmd/main.go`) và sở hữu bốn migration `0006`, `0008`, `0009`, `0010`. Cùng hình dạng với `services/appService` (login là "public write", các route còn lại là "user-scoped write"). Tài khoản tách khỏi `users` để không bao giờ đăng nhập được qua appService.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/backofficeService/domain/admin.go` | `Admin`, `AdminLoginInput/Result`, `ChangePasswordInput`, `BootstrapAdminInput`, `CreateAccountInput/Result`; hằng role `RoleAdmin`/`RoleContentEditor`/`RoleViewer` + `ValidRole`; `MinPasswordLen`; `ExtraMustChangePassword`. Không JSON tag. |
| Domain / Errors | `domain/admin.go` | Sentinel: `ErrInvalidCredentials`, `ErrAdminNotFound`, `ErrPasswordChangeRequired`, `ErrSessionRevoked`, `ErrPasswordExpired`, `ErrCannotResetSelf`, `ErrInvalidAccountInput`, `ErrEmailTaken`, `ErrInviteEmailFailed`, `ErrWrongCurrentPassword`, `ErrWeakPassword`. |
| Domain / Repository interface | `domain/admin.go` | `AdminRepository`: `FindByEmail`, `FindByID`, `TouchLastLogin`, `UpdatePassword`, `CreateBootstrapIfEmpty`, `CreateAccount`, `ResetTempPassword`, `DeleteByID`, `Create` (dùng bởi `cmd/seedadmin`). |
| Domain / Mock | `domain/mock_admin_repository.go` | Mock viết tay cho test usecase. |
| Usecase / Auth | `usecase/admin_auth_usecase.go` | `Login`, `ChangePassword`, `EnsureBootstrapAdmin`, `ValidateSession`, `issueToken`; `NormalizeEmail`; bcrypt dummy hash cân bằng thời gian. |
| Usecase / Account | `usecase/admin_user_usecase.go` | `CreateAccount`, `ResetPassword`, sinh mật khẩu tạm (`tempPassword`, hoặc `fixedTempPassword` ⚠️ DEV ONLY), `tempExpiry` theo TTL; phụ thuộc `pkg/mailer.Mailer`. |
| Repository | `repository/postgres_admin_repository.go` | SQL thuần trên `admins`; `findOne`, `CreateBootstrapIfEmpty` (`INSERT … WHERE NOT EXISTS … ON CONFLICT DO NOTHING`), `CreateAccount` (`ON CONFLICT DO NOTHING RETURNING id` → không có row = `ErrEmailTaken`), `ResetTempPassword` (`UPDATE … RETURNING`), `UpdatePassword`, `DeleteByID`. |
| Delivery / Handler | `delivery/http/admin_auth_handler.go` | `Login`, `ChangePassword`, `Me`; DTO login/change-password (JSON tag ở đây). |
| Delivery / Handler | `delivery/http/admin_user_handler.go` | `Create`, `ResetPassword`; DTO. Không trả mật khẩu tạm. |
| Delivery / Middleware | `delivery/http/session.go` | `RequireActiveSession` — gọi `ValidateSession(adminID, iat)`. |
| Delivery / Middleware | `delivery/http/errors.go` | `RequireRole`, `RequireStaff`, `RequireStaffAllowPasswordChange` (chung `guard`); `WriteAuthError`; `writeDomainError`/`writeDecodeError` (map sentinel → envelope). |
| Mailer | `pkg/mailer/mailer.go`, `log_mailer.go`, `mock.go` | Thêm `SendAccountInviteEmail(toEmail, name, tempPassword)`; `LogMailer` (⚠️ DEV ONLY, log cả mật khẩu tạm). |
| Config | `config/config.go`, `config/config.dev.yaml` | `BootstrapAdminConfig`, `AccountConfig` (`fixed_temp_password`, `temp_password_ttl`); JWT audience/secret riêng. |
| Wiring | `services/backofficeService/cmd/main.go` | Bootstrap khi khởi động; đăng ký 5 route; `adminOnly` / `staff` / `adminPasswordChange` = JWT → session → role. `⚠️ DEV ONLY`: `MockSecretGetter`, `LogMailer`. |
| Seed CLI | `cmd/seedadmin/main.go` | Tạo admin đầu tiên ngoài hệ thống (`ADMIN_PASSWORD` ≥ 12 ký tự, không có cờ must-change). |
| Migration | `migrations/0006_admins.sql`, `0008_admin_must_change_password.sql`, `0009_admin_tokens_valid_after.sql`, `0010_admin_temp_password_expiry.sql` | Bảng `admins` + 3 cột thêm dần. |
| API spec | `docs/specs/api/openapi.yaml` | **Chưa cập nhật** — xem Known gaps. |

## Request flow

> Xem `flow.md` cho sơ đồ sequence Mermaid (mirror 1:1 của sáu Step-by-step Flow bên dưới).

## Step-by-step Flow — Bootstrap admin khi khởi động

```
Step 1 — Nạp config và bật bootstrap (cmd/main.go)
- Trigger: backofficeService khởi động, sau khi kết nối DB
- Pre-condition: DB kết nối được; `cfg.BootstrapAdmin.Enabled == true`
- Call: gọi AdminAuthUsecase.EnsureBootstrapAdmin(email, password, name)
    - Success → Step 2
    - Failure (enabled=false) → bỏ qua toàn bộ flow, không tạo gì

Step 2 — Chuẩn hoá và băm mật khẩu (usecase)
- Trigger: EnsureBootstrapAdmin
- Pre-condition: email và password không rỗng
- Call: NormalizeEmail; bcrypt.GenerateFromPassword(password)
    - Success → Step 3
    - Failure (email/password rỗng, hoặc lỗi bcrypt) → trả error, main `log.Fatalf` — service không lên

Step 3 — Chèn admin nếu bảng rỗng (repository, MỘT câu SQL)
- Trigger: Step 2 xong
- Pre-condition: —
- Call: CreateBootstrapIfEmpty → `INSERT INTO admins (…, must_change_password) SELECT $1,$2,$3,TRUE WHERE NOT EXISTS (SELECT 1 FROM admins) ON CONFLICT (email) DO NOTHING`
    - Success (RowsAffected = 1) → log WARN "bootstrap admin created…", tiếp tục khởi động
    - Success (RowsAffected = 0, bảng đã có admin) → no-op, không log WARN
    - Failure (DB) → `log.Fatalf`
    - Transaction boundary: kiểm tra rỗng + insert là một statement nên nhiều instance khởi động cùng lúc chỉ tạo tối đa một admin. `temp_password_expires_at` để NULL → admin mặc định không hết hạn.
```

## Step-by-step Flow — Login: `POST /v1/admin/auth/login`

```
Step 1 — Decode body (handler)
- Trigger: request đến route public
- Pre-condition: —
- Call: utils.DecodeJSONBodyStrict → adminLoginRequest{email, password}
    - Success → Step 2
    - Failure (JSON hỏng / field lạ) → 400 VALIDATION_ERROR; (quá lớn) → 413 PAYLOAD_TOO_LARGE

Step 2 — Chuẩn hoá và kiểm rỗng (usecase)
- Trigger: handler gọi Login
- Pre-condition: —
- Call: NormalizeEmail; email == "" || password == ""
    - Success → Step 3
    - Failure → ErrInvalidCredentials → 401 INVALID_CREDENTIALS (không chạm DB)

Step 3 — Tra admin (repository)
- Trigger: Step 2 ok
- Pre-condition: —
- Call: FindByEmail(email)
    - Success (có row hoặc ErrAdminNotFound) → Step 4 (không tìm thấy vẫn đi tiếp với admin = nil)
    - Failure (DB) → 500 INTERNAL_ERROR

Step 4 — So mật khẩu, một lần bcrypt trên mọi nhánh (usecase)
- Trigger: Step 3
- Pre-condition: —
- Call: bcrypt.CompareHashAndPassword(hash thật hoặc dummy hash, password)
    - Success (admin tồn tại, khớp, is_active) → Step 5
    - Failure (không có admin / sai mật khẩu / is_active=false) → ErrInvalidCredentials → 401 INVALID_CREDENTIALS (cùng thông điệp, cùng thời gian)

Step 5 — Kiểm hạn mật khẩu tạm (usecase, CHỈ sau khi mật khẩu đã khớp)
- Trigger: Step 4 ok
- Pre-condition: —
- Call: admin.MustChangePassword && TempPasswordExpiresAt != nil && now > TempPasswordExpiresAt
    - Success (không hết hạn / không có hạn / đã đổi mật khẩu riêng) → Step 6
    - Failure (hết hạn) → ErrPasswordExpired → 401 PASSWORD_EXPIRED

Step 6 — Cấp access token (usecase)
- Trigger: Step 5 ok
- Pre-condition: —
- Call: jwt.GenerateAccessToken(id, email, role, extra) — extra `{must_change_password:true}` khi admin còn bị buộc đổi
    - Success → Step 7
    - Failure → 500 INTERNAL_ERROR

Step 7 — Ghi last_login_at (repository, best-effort)
- Trigger: Step 6 ok
- Pre-condition: —
- Call: TouchLastLogin(id, now)
    - Success → 200 `{access_token, admin, must_change_password}`
    - Failure → log WARN, VẪN trả 200 (login đã thành công, bookkeeping không được làm hỏng nó)
```

## Step-by-step Flow — Đổi mật khẩu: `POST /v1/admin/auth/change-password`

```
Step 1 — Xác thực và kiểm phiên (middleware)
- Trigger: request đến route change-password
- Pre-condition: —
- Call: JWT auth (access token, audience backoffice) → RequireActiveSession (xem flow "Bảo vệ route") → RequireStaffAllowPasswordChange (role hợp lệ; KHÔNG chặn cờ must-change)
    - Success → Step 2
    - Failure → 401 UNAUTHORIZED / 403 FORBIDDEN

Step 2 — Lấy admin từ token, decode body (handler)
- Trigger: middleware cho qua
- Pre-condition: claims trong context
- Call: strconv.ParseInt(claims.UserID); DecodeJSONBodyStrict → {current_password, new_password}
    - Success → Step 3
    - Failure (UserID không phải số) → 401; (body lỗi / field lạ như `admin_id`) → 400 VALIDATION_ERROR

Step 3 — Tra admin (repository)
- Trigger: handler gọi ChangePassword
- Pre-condition: —
- Call: FindByID(adminID)
    - Success → Step 4
    - Failure (không có row / is_active=false) → ErrInvalidCredentials → 401; (DB) → 500

Step 4 — Kiểm mật khẩu hiện tại và mật khẩu mới (usecase)
- Trigger: Step 3 ok
- Pre-condition: —
- Call: bcrypt so `current_password`; `len(new) >= 12 && new != current`
    - Success → Step 5
    - Failure (sai mật khẩu hiện tại) → ErrWrongCurrentPassword → 400 WRONG_CURRENT_PASSWORD
    - Failure (quá ngắn / trùng cũ) → ErrWeakPassword → 400 WEAK_PASSWORD

Step 5 — Lưu mật khẩu mới và thu hồi token cũ (repository, MỘT câu UPDATE)
- Trigger: Step 4 ok
- Pre-condition: —
- Call: bcrypt băm mật khẩu mới; UpdatePassword → `UPDATE admins SET password_hash, must_change_password=FALSE, tokens_valid_after=now(), temp_password_expires_at=NULL, updated_at=now() WHERE id`
    - Success → Step 6
    - Failure (DB) → 500 INTERNAL_ERROR
    - Transaction boundary: một statement; hash, xoá cờ, xoá hạn và mốc thu hồi cùng thay đổi hoặc không gì cả.

Step 6 — Cấp token mới không còn cờ (usecase)
- Trigger: Step 5 ok
- Pre-condition: admin.MustChangePassword đã đặt false trong bộ nhớ
- Call: issueToken (không có extra)
    - Success → 200 `{access_token, admin, must_change_password:false}`
    - Failure → 500 INTERNAL_ERROR
    - Ghi chú: `iat` của token mới có độ chính xác giây và không nhỏ hơn mốc `tokens_valid_after` đã cắt về giây, nên token mới không bị chính Step 5 thu hồi.
```

## Step-by-step Flow — Bảo vệ route: mọi request `/v1/admin/*` trừ login

```
Step 1 — Xác thực JWT (pkg/jwt/middleware)
- Trigger: request có header Authorization
- Pre-condition: —
- Call: verify chữ ký + audience + exp; RequireTokenType = "access"
    - Success → Step 2
    - Failure (không token / refresh token / token khác secret-audience, ví dụ appService) → 401 UNAUTHORIZED

Step 2 — Kiểm phiên còn hiệu lực (RequireActiveSession → AdminAuthUsecase.ValidateSession)
- Trigger: Step 1 ok
- Pre-condition: claims có UserID số và `iat`
- Call: FindByID(adminID); so `iat` với `TokensValidAfter` (cắt về giây)
    - Success (admin còn, is_active, iat >= mốc hoặc mốc NULL) → Step 3
    - Failure (không có admin / is_active=false / iat < mốc) → ErrSessionRevoked → 401 UNAUTHORIZED
    - Failure (lỗi DB) → 500 INTERNAL_ERROR, KHÔNG cho request đi tiếp

Step 3 — Kiểm role và cờ must-change (guard)
- Trigger: Step 2 ok
- Pre-condition: —
- Call: `RequireRole(admin)` cho `/users*`; `RequireStaff` cho `/me`; `RequireStaffAllowPasswordChange` cho change-password
    - Success → handler của route
    - Failure (role không hợp lệ / không phải admin) → 403 FORBIDDEN
    - Failure (token còn cờ must-change trên route không cho phép) → 403 PASSWORD_CHANGE_REQUIRED
```

## Step-by-step Flow — Tạo tài khoản: `POST /v1/admin/users`

```
Step 1 — Bảo vệ route (middleware)
- Trigger: request đến route users
- Pre-condition: —
- Call: flow "Bảo vệ route" với `RequireRole(admin)`
    - Success → Step 2
    - Failure → 401 / 403 FORBIDDEN / 403 PASSWORD_CHANGE_REQUIRED; use case KHÔNG được gọi

Step 2 — Decode body (handler)
- Trigger: middleware cho qua
- Pre-condition: —
- Call: DecodeJSONBodyStrict → {email, name, role}
    - Success → Step 3
    - Failure → 400 VALIDATION_ERROR / 413

Step 3 — Validate đầu vào (usecase)
- Trigger: handler gọi CreateAccount
- Pre-condition: —
- Call: NormalizeEmail + `net/mail.ParseAddress`; name != ""; `ValidRole(role)`
    - Success → Step 4
    - Failure → ErrInvalidAccountInput (kèm lý do) → 400 VALIDATION_ERROR; không chạm DB, không gửi email

Step 4 — Sinh và băm mật khẩu tạm (usecase)
- Trigger: Step 3 ok
- Pre-condition: —
- Call: `fixedTempPassword` nếu cấu hình (⚠️ DEV ONLY), ngược lại 12 byte `crypto/rand` → base64url 16 ký tự; bcrypt băm; `tempExpiry()` = now + TTL (nil nếu TTL = 0)
    - Success → Step 5
    - Failure (rand/bcrypt) → 500 INTERNAL_ERROR

Step 5 — Chèn tài khoản (repository, MỘT câu INSERT)
- Trigger: Step 4 ok
- Pre-condition: —
- Call: CreateAccount → `INSERT … must_change_password=TRUE, temp_password_expires_at=$5 ON CONFLICT (email) DO NOTHING RETURNING id`
    - Success (có id) → Step 6
    - Failure (không có row trả về) → ErrEmailTaken → 409 EMAIL_TAKEN; không gửi email
    - Failure (DB) → 500 INTERNAL_ERROR

Step 6 — Gửi email mời (pkg/mailer, ngoài scope feature này)
- Trigger: Step 5 ok
- Pre-condition: tài khoản đã được ghi
- Call: Mailer.SendAccountInviteEmail(email, name, tempPassword) — dev: `LogMailer` chỉ log
    - Success → 201 `{admin, must_change_password:true}` (KHÔNG có mật khẩu)
    - Failure → Step 7
    - ⚠️ Không có transaction bao Step 5 và 6 (email không rollback được); bù trừ bằng Step 7.

Step 7 — Bù trừ khi gửi email lỗi (repository)
- Trigger: Step 6 lỗi
- Pre-condition: —
- Call: DeleteByID(id)
    - Success → ErrInviteEmailFailed → 502 EMAIL_SEND_FAILED
    - Failure (xoá lỗi) → log ERROR, VẪN trả 502 (tài khoản mồ côi; xem Known gaps)
```

## Step-by-step Flow — Reset mật khẩu: `POST /v1/admin/users/{id}/reset-password`

```
Step 1 — Bảo vệ route (middleware)
- Trigger: request đến route reset-password
- Pre-condition: —
- Call: flow "Bảo vệ route" với `RequireRole(admin)`
    - Success → Step 2
    - Failure → 401 / 403; use case KHÔNG được gọi

Step 2 — Xác định người gọi và đích (handler)
- Trigger: middleware cho qua
- Pre-condition: claims trong context
- Call: actorID = claims.UserID (token, không phải body); targetID = `r.PathValue("id")`
    - Success → Step 3
    - Failure (targetID không phải số dương) → 400 VALIDATION_ERROR (details `{field:id}`)

Step 3 — Chặn tự reset (usecase)
- Trigger: handler gọi ResetPassword
- Pre-condition: —
- Call: actorID == targetID
    - Success (khác nhau) → Step 4
    - Failure → ErrCannotResetSelf → 409 CANNOT_RESET_SELF; không chạm DB

Step 4 — Sinh và băm mật khẩu tạm mới (usecase)
- Trigger: Step 3 ok
- Pre-condition: —
- Call: như Step 4 của "Tạo tài khoản" (`newTempPassword`, `tempExpiry`)
    - Success → Step 5
    - Failure → 500 INTERNAL_ERROR

Step 5 — Thay mật khẩu và thu hồi phiên (repository, MỘT câu UPDATE)
- Trigger: Step 4 ok
- Pre-condition: —
- Call: ResetTempPassword → `UPDATE admins SET password_hash, must_change_password=TRUE, temp_password_expires_at=$3, tokens_valid_after=now() WHERE id RETURNING email,name,role,is_active`
    - Success → Step 6
    - Failure (không có row) → ErrAdminNotFound → 404 ADMIN_NOT_FOUND; không gửi email
    - Failure (DB) → 500 INTERNAL_ERROR
    - Transaction boundary: một statement; mật khẩu mới, cờ, hạn và mốc thu hồi cùng đổi. Mọi token cũ của tài khoản đích bị vô hiệu ngay.

Step 6 — Gửi email mật khẩu mới (pkg/mailer, ngoài scope feature này)
- Trigger: Step 5 ok
- Pre-condition: —
- Call: Mailer.SendAccountInviteEmail(email, name, tempPassword)
    - Success → 200 `{admin, must_change_password:true}` (KHÔNG có mật khẩu)
    - Failure → ErrInviteEmailFailed → 502 EMAIL_SEND_FAILED. KHÔNG hoàn tác được (mật khẩu cũ đã bị thay) — admin gọi lại reset.
```

## Deliberately stubbed for now

- **Provider email thật**: `cmd/main.go` dùng `mailer.NewLogMailer` (`⚠️ DEV ONLY`). Mọi Step "gửi email" hiện chỉ ghi log — kể cả mật khẩu tạm. Phase 6 của `tasks.md`.
- **Mật khẩu tạm cố định**: `account.fixed_temp_password: "111111"` trong `config.dev.yaml` để test không cần mở email. Để trống ở production thì mỗi tài khoản nhận mật khẩu ngẫu nhiên.
- **Secrets**: `config.NewMockSecretGetterFromYAML` (plaintext `dev_secrets`); thay bằng `pkg/secrets/gcp|aws` trước production.

## Known gaps carried by this feature

- **Không có rate limit / khoá brute-force** cho login và cho việc thử `current_password` (appService có, backoffice chưa).
- **Không có refresh/session/logout riêng lẻ**: thu hồi chỉ theo cả admin (`tokens_valid_after`), không thu hồi được từng token/thiết bị.
- **Cửa sổ 1 giây**: `iat` chính xác đến giây, nên token của admin phát trong đúng giây đổi mật khẩu không bị thu hồi (đổi lại token mới trả về dùng được ngay).
- **Mỗi request đọc DB một lần** (`ValidateSession`); chưa có cache. Chấp nhận với lưu lượng backoffice.
- **Token phát trước lúc hết hạn mật khẩu tạm** vẫn đổi mật khẩu được tối đa `access_token_expiration` sau đó.
- **Tạo tài khoản không atomic với gửi email**: nếu `DeleteByID` bù trừ cũng lỗi thì còn tài khoản mồ côi (chỉ có log ERROR).
- **Reset mật khẩu khi gửi email lỗi** để lại mật khẩu mới chưa ai nhận được (admin phải reset lại).
- **Tài khoản đã bị khoá (`is_active=false`) vẫn reset được** (không có route khoá/mở khoá để dùng chung).
- **`openapi.yaml` và ERD chưa có `admins` / các route backoffice**; README backoffice chưa phản ánh tạo/reset tài khoản.
- **Chưa có test repository trên DB thật** và **chưa có E2E** cho backoffice (chỉ kiểm bằng curl thủ công + Postgres thật).
