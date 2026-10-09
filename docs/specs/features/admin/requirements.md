# Requirements — admin

Version: 1.0
Last updated: 2026-09-29
Status: implemented (2026-09-29) — đăng nhập, tài khoản mặc định, đổi mật khẩu, thu hồi phiên, tạo tài khoản, reset mật khẩu chạy thật qua binary + Postgres; có test usecase và handler (mock). **Chưa có** test repository trên DB thật, chưa có E2E tự động, chưa cập nhật `openapi.yaml`/ERD. Còn 4 marker `⚠️ Not confirmed`.
Module: `backoffice`
Code: `services/backofficeService/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này được viết **retroactively** từ phần backoffice admin đã được implement (Retroactive mode). Mọi AC dưới đây mô tả đúng hành vi code hiện có; thứ chưa làm nằm ở "Pending", không gộp vào AC.

backofficeService là binary riêng (`:8081`), dùng chung Postgres `innerlamp` với appService nhưng **khác trust domain**: bảng riêng `admins` (không phải `users`), JWT ký bằng secret + audience riêng (`innerlamp_backoffice`) nên token appService bị từ chối ở đây và ngược lại. Không có tự đăng ký; tài khoản chỉ sinh ra từ bootstrap, `cmd/seedadmin`, hoặc admin tạo.

---

## US-BO-01: Đăng nhập backoffice

## User Story

> Là một **nhân sự vận hành (admin/content_editor/viewer)**,
> Tôi muốn **đăng nhập backoffice bằng email và mật khẩu**,
> Để **vận hành hệ thống InnerLamp qua các API quản trị**.

## Acceptance Criteria (EARS notation)

1. WHEN a `POST /v1/admin/auth/login` request arrives with an `email` and `password` that match an active row in `admins`, THE SYSTEM SHALL return `200 OK` with an `access_token`, the `admin` object (`id`, `email`, `name`, `role`) and `must_change_password`, and SHALL record `last_login_at`. (Email được `trim` + lower-case trước khi tra cứu. Chỉ có access token; chưa có refresh/session.)
2. WHEN a login request arrives with an unknown email, a wrong password, a deactivated account (`is_active = false`), or an empty email/password, THE SYSTEM SHALL return the **same** `401 INVALID_CREDENTIALS` in all cases, and SHALL pay exactly one bcrypt comparison on every path so response time does not reveal which emails exist.
3. WHEN a login succeeds, THE SYSTEM SHALL issue an access token valid for `jwt.access_token_expiration` (dev: 30 phút) whose `role` claim is the admin's role.
4. WHEN a login request body has an unknown field, malformed JSON or is oversized, THE SYSTEM SHALL reject it with `400 VALIDATION_ERROR` (hoặc `413 PAYLOAD_TOO_LARGE`).
5. WHEN a `GET /v1/admin/me` request arrives with a valid access token of any known role, THE SYSTEM SHALL return `user_id`, `email` and `role` from the token claims.

## US-BO-02: Tài khoản admin mặc định

## User Story

> Là một **người triển khai hệ thống**,
> Tôi muốn **hệ thống khởi động là có sẵn một tài khoản admin mặc định phải đổi mật khẩu ở lần đăng nhập đầu**,
> Để **có tài khoản đầu tiên tạo ra các tài khoản khác mà không cần thao tác ngoài hệ thống**.

## Acceptance Criteria (EARS notation)

6. WHEN backofficeService starts with `bootstrap_admin.enabled: true` and the `admins` table is empty, THE SYSTEM SHALL insert one admin (`bootstrap_admin.email` / `password` / `name`, role `admin`) with `must_change_password = TRUE`, and SHALL log a warning that the default password must be changed.
7. WHEN backofficeService starts and at least one row exists in `admins`, THE SYSTEM SHALL NOT create the bootstrap admin — kể cả khi admin mặc định đã bị xoá/đổi tên. The emptiness check and the insert SHALL be one statement so several instances starting together create at most one.
8. WHEN `bootstrap_admin.enabled` is false, THE SYSTEM SHALL NOT create any admin at startup. (Dùng `cmd/seedadmin` với `ADMIN_PASSWORD` ≥ 12 ký tự.)
9. WHEN an admin with `must_change_password = TRUE` logs in, THE SYSTEM SHALL return `must_change_password: true` and a token carrying the extra claim `must_change_password`; WHEN that token calls any route other than `POST /v1/admin/auth/change-password`, THE SYSTEM SHALL reject it with `403 PASSWORD_CHANGE_REQUIRED`.

## US-BO-03: Đổi mật khẩu

## User Story

> Là một **tài khoản backoffice đang đăng nhập (kể cả lần đầu)**,
> Tôi muốn **đổi mật khẩu của chính mình**,
> Để **thay mật khẩu mặc định/tạm thời bằng mật khẩu riêng**.

## Acceptance Criteria (EARS notation)

10. WHEN a `POST /v1/admin/auth/change-password` request arrives with a valid access token of any known role (kể cả token còn cờ `must_change_password`) and a `current_password` that matches, and a `new_password` of at least 12 characters that differs from the current one, THE SYSTEM SHALL store the new bcrypt hash, clear `must_change_password` and the temp-password expiry, set `tokens_valid_after` to now, and return `200` with a **new** access token without the must-change restriction.
11. WHEN the `current_password` does not match, THE SYSTEM SHALL return `400 WRONG_CURRENT_PASSWORD` (details `{field: current_password}`) and SHALL NOT change anything.
12. WHEN the `new_password` is shorter than 12 characters or equals the current password, THE SYSTEM SHALL return `400 WEAK_PASSWORD` (details `{field: new_password}`) and SHALL NOT change anything.
13. THE SYSTEM SHALL take the target admin from the token subject, never from the request body (body có field lạ bị `400 VALIDATION_ERROR`). WHEN the token's admin no longer exists or is inactive, THE SYSTEM SHALL return `401 INVALID_CREDENTIALS`.

## US-BO-04: Thu hồi phiên

## User Story

> Là một **quản trị hệ thống**,
> Tôi muốn **token cũ của một tài khoản mất hiệu lực ngay khi mật khẩu đổi/tài khoản bị khoá/mật khẩu bị reset**,
> Để **token bị lộ không dùng được đến hết hạn tự nhiên**.

## Acceptance Criteria (EARS notation)

14. WHEN a request to any `/v1/admin/*` route except login carries a token whose `iat` is earlier than the admin's `tokens_valid_after` (so sánh ở độ chính xác 1 giây), THE SYSTEM SHALL return `401 UNAUTHORIZED`. `tokens_valid_after` được set khi đổi mật khẩu và khi reset mật khẩu.
15. WHEN a request carries a validly signed token of an admin that has been deactivated (`is_active = false`) or no longer exists, THE SYSTEM SHALL return `401 UNAUTHORIZED` immediately, without waiting for the token to expire.
16. WHEN the session check itself fails (lỗi DB), THE SYSTEM SHALL return `500 INTERNAL_ERROR` and SHALL NOT let the request through.
17. WHEN a request carries no token, a refresh token, or a token signed for another audience/secret (ví dụ token appService), THE SYSTEM SHALL return `401 UNAUTHORIZED`.

## US-BO-05: Tạo tài khoản

## User Story

> Là một **admin**,
> Tôi muốn **tạo tài khoản backoffice với role cụ thể và để hệ thống gửi mật khẩu tạm qua email**,
> Để **giao quyền vận hành cho người khác mà không phải biết mật khẩu của họ**.

## Acceptance Criteria (EARS notation)

18. WHEN a `POST /v1/admin/users` request arrives with `email`, `name` and `role` from a caller whose token role is `admin`, THE SYSTEM SHALL create the account with `must_change_password = TRUE`, email it the temporary password, and return `201 Created` with the `admin` object and `must_change_password: true`. THE SYSTEM SHALL NOT return the temporary password in the response.
19. WHEN the caller's token role is any role other than `admin` (`content_editor`, `viewer`, hoặc role lạ), THE SYSTEM SHALL return `403 FORBIDDEN` before the use case runs; WHEN the caller is an `admin` whose token still carries `must_change_password`, THE SYSTEM SHALL return `403 PASSWORD_CHANGE_REQUIRED`.
20. WHEN the `email` is not a valid address, the `name` is blank, or the `role` is not one of `admin`, `content_editor`, `viewer`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` naming the problem, and SHALL NOT create the account or send an email. (Email được `trim` + lower-case.)
21. WHEN the `email` is already used by another admin, THE SYSTEM SHALL return `409 EMAIL_TAKEN` (details `{field: email}`) and SHALL NOT send an email.
22. WHEN the invite email cannot be sent, THE SYSTEM SHALL delete the account it just created and return `502 EMAIL_SEND_FAILED`, so no account exists that nobody can sign in to.
23. WHEN `account.fixed_temp_password` is empty, THE SYSTEM SHALL generate a random temporary password (12 byte ngẫu nhiên, 16 ký tự) per account; WHEN it is set, THE SYSTEM SHALL use that value for every new account. (⚠️ DEV ONLY — dev đặt `111111`.)
24. WHEN `account.temp_password_ttl` is greater than zero, THE SYSTEM SHALL stamp the new account's `temp_password_expires_at` = now + TTL (dev: 24h); WHEN it is zero, THE SYSTEM SHALL leave it empty (không hết hạn).
25. WHEN an admin whose `must_change_password` is TRUE and whose `temp_password_expires_at` has passed logs in with the **correct** password, THE SYSTEM SHALL return `401 PASSWORD_EXPIRED`; WHEN the password is wrong, THE SYSTEM SHALL still return `401 INVALID_CREDENTIALS` (không lộ việc đã hết hạn). Expiry SHALL be evaluated at login; no background job. It SHALL NOT apply to an admin that has already chosen their own password, nor to one with no expiry (bootstrap admin).

## US-BO-06: Reset mật khẩu tài khoản khác

## User Story

> Là một **admin**,
> Tôi muốn **cấp lại mật khẩu tạm cho một tài khoản khác (mật khẩu tạm hết hạn hoặc bị mất)**,
> Để **người đó đăng nhập lại được mà tôi không phải xoá và tạo lại tài khoản**.

## Acceptance Criteria (EARS notation)

26. WHEN a `POST /v1/admin/users/{id}/reset-password` request arrives from an `admin` for another existing account, THE SYSTEM SHALL replace its password with a new temporary one, set `must_change_password = TRUE`, a fresh `temp_password_expires_at` (theo TTL), and `tokens_valid_after` = now (thu hồi mọi phiên đang có), email the new password, and return `200` with the `admin` object. THE SYSTEM SHALL NOT return the password.
27. WHEN the `{id}` is the caller's own id, THE SYSTEM SHALL return `409 CANNOT_RESET_SELF` and SHALL NOT change anything. (Caller lấy từ token subject.)
28. WHEN no admin has that `{id}`, THE SYSTEM SHALL return `404 ADMIN_NOT_FOUND` and send no email; WHEN `{id}` is not a positive integer, THE SYSTEM SHALL return `400 VALIDATION_ERROR`.
29. WHEN the caller is not an `admin`, THE SYSTEM SHALL return `403 FORBIDDEN` (hoặc `401` nếu token bị thu hồi) before the use case runs.
30. WHEN the reset email cannot be sent, THE SYSTEM SHALL return `502 EMAIL_SEND_FAILED`; the account keeps the new, unsent password (không thể hoàn tác) and the admin SHALL retry the reset.

## API contract (confirmed surface for this feature)

Chưa có trong `docs/specs/api/openapi.yaml` — bảng này lấy từ handler thực tế (xem "Pending").

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/healthz` | public | 200 `{status}` | — |
| POST | `/v1/admin/auth/login` | public | 200 `{access_token, admin, must_change_password}` | 400, 413, 401 `INVALID_CREDENTIALS` / `PASSWORD_EXPIRED` |
| POST | `/v1/admin/auth/change-password` | mọi role; chấp nhận token còn cờ must-change | 200 (cùng shape login) | 400 `WRONG_CURRENT_PASSWORD` / `WEAK_PASSWORD` / `VALIDATION_ERROR`, 401, 403 |
| GET | `/v1/admin/me` | mọi role, token không còn cờ must-change | 200 `{user_id, email, role}` | 401, 403 `PASSWORD_CHANGE_REQUIRED` |
| POST | `/v1/admin/users` | role `admin` | 201 `{admin, must_change_password}` | 400, 401, 403, 409 `EMAIL_TAKEN`, 502 `EMAIL_SEND_FAILED` |
| POST | `/v1/admin/users/{id}/reset-password` | role `admin` | 200 `{admin, must_change_password}` | 400, 401, 403, 404 `ADMIN_NOT_FOUND`, 409 `CANNOT_RESET_SELF`, 502 |

## Out of scope (this feature boundary, not tracked here)

- Đăng nhập/đăng ký của **người dùng app** — thuộc `identity` (`docs/specs/features/login/`, `onboarding/`).
- Quản lý temple/deity qua backoffice — có spec riêng (`temple/`, `deity/`); spec này chỉ phủ tài khoản vận hành và xác thực.

## Pending (not yet implemented)

- Rate limit / khoá đăng nhập theo brute-force cho `/v1/admin/auth/login` và cho việc thử `current_password` (appService có, backoffice chưa) — tasks.md Phase 6.
- Refresh token / session / logout riêng từng thiết bị; hiện chỉ thu hồi được **toàn bộ** token của một admin qua `tokens_valid_after` — Phase 6.
- Route liệt kê / sửa role / khoá-mở khoá tài khoản; nhờ `tokens_valid_after` việc khoá đã có cơ chế thu hồi sẵn — Phase 6.
- RBAC (quyền chi tiết theo role). Hiện `RequireRole` chỉ so khớp chuỗi role và 3 role là nhãn — Phase 6.
- Provider gửi email thật (hiện chỉ `LogMailer`) — Phase 6.
- Audit log (ai tạo/reset tài khoản nào). Hiện chỉ có dòng log `admin account created` / `admin password reset` (reset có `by`).
- Cập nhật `docs/specs/api/openapi.yaml`, ERD `docs/specs/db/erd.mmd` (bảng `admins`), README backoffice — Phase 4/5.

> ⚠️ Not confirmed: khi deploy production, tài khoản đầu tiên tạo bằng `cmd/seedadmin` hay bootstrap với mật khẩu lấy từ secret manager — chặn việc chốt cấu hình `bootstrap_admin` ngoài dev (hiện chỉ có `enabled: false` + `seedadmin`).
>
> ⚠️ Not confirmed: mật khẩu tạm gửi qua email thật có cần ngắn hạn hơn 24h hoặc dùng link một lần thay vì mật khẩu — chặn thiết kế email provider thật (Phase 6).
>
> ⚠️ Not confirmed: định danh đăng nhập của admin mặc định là chuỗi `"admin"` lưu trong cột `email` (login không validate định dạng email) — khi nào gán email thật cho tài khoản này, và bằng route nào (chưa có route sửa profile).
>
> ⚠️ Not confirmed: role nằm trong token nên đổi role sau này phải set `tokens_valid_after` để role cũ hết hiệu lực ngay — chưa có route đổi role, chặn thiết kế RBAC.

## Ghi chú kỹ thuật liên quan

- **Bảng `admins`** (`migrations/0006_admins.sql`, `0008_admin_must_change_password.sql`, `0009_admin_tokens_valid_after.sql`, `0010_admin_temp_password_expiry.sql`): `email` UNIQUE (lưu lower-case bởi app), `password_hash` bcrypt, `role` mặc định `admin`, `is_active`, `must_change_password`, `tokens_valid_after` (NULL = không giới hạn), `temp_password_expires_at` (NULL = không hết hạn), `last_login_at`.
- **Token:** JWT HS256, audience `innerlamp_backoffice`, secret riêng. Claim `extra.must_change_password = true` khi còn bị buộc đổi mật khẩu. Server **không lưu token**; thu hồi bằng so `iat` với `tokens_valid_after` mỗi request (một lần đọc DB/request).
- **Thứ tự middleware mỗi route bảo vệ:** xác thực JWT (chỉ `access`) → `RequireActiveSession` → `RequireRole`/`RequireStaff` (kèm kiểm cờ must-change). Trừ `change-password` dùng biến thể cho phép cờ must-change.
- **Role** (`domain/admin.go`): `admin`, `content_editor`, `viewer`. Chỉ `admin` được quản lý tài khoản; các role còn lại hiện là nhãn.
- **Mật khẩu:** đặt mới ≥ 12 ký tự (`MinPasswordLen`), khác mật khẩu hiện tại. Mật khẩu mặc định `admin` ngắn hơn ngưỡng đó là cố ý — ngưỡng chỉ áp dụng cho mật khẩu do người dùng tự đặt.
- **⚠️ DEV ONLY:** `config.dev.yaml` chứa `bootstrap_admin` (`admin`/`admin`), `account.fixed_temp_password: "111111"`, `dev_secrets` plaintext; `cmd/main.go` dùng `mailer.NewLogMailer` (chỉ log, kể cả mật khẩu tạm). Không được deploy nguyên trạng.
- Email được gửi qua `pkg/mailer.Mailer.SendAccountInviteEmail` (method mới thêm cho feature này).

## Open questions (resolved during implementation)

- ~~Bảng admin dùng chung `users` hay tách riêng?~~ → tách riêng `admins` (không OTP/SMS, không đăng nhập được qua appService). Resolved 2026-09-29.
- ~~Role: enum/bảng hay một cột?~~ → một cột `role` mặc định `admin`; RBAC làm sau. Resolved 2026-09-29.
- ~~Thu hồi token bằng blacklist hay mốc thời gian?~~ → cột `tokens_valid_after` theo từng admin (server không theo dõi token đã phát nên blacklist theo `jti` không thu hồi được "tất cả"). Resolved 2026-09-29.
- ~~Mật khẩu tạm do admin đặt hay server sinh, và có trả về API không?~~ → server sinh, gửi email, không trả về trong response. Resolved 2026-09-29.
- ~~Gửi email lỗi khi tạo tài khoản thì sao?~~ → xoá tài khoản vừa tạo, trả 502. (Reset không hoàn tác được — xem AC30.) Resolved 2026-09-29.
- ~~Mật khẩu tạm có hạn không?~~ → có, `account.temp_password_ttl` (dev: 24h, `0` = tắt). Resolved 2026-09-29.
