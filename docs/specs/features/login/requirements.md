# Requirements — login

Version: 1.13
Last updated: 2026-10-10
Status: implemented (2026-09-21; v1.12 thêm AC16–AC19 `X-API-Key`, đã có code + test; v1.13 thêm AC20–AC22 hợp đồng response cấp token, đã có code + test + E2E, `tasks.md` Phase 7) — login, refresh, logout chạy thật qua binary + Postgres 16; AC1–AC15 đã có test (usecase, repository trên DB thật, handler) và đã kiểm bằng curl. Audit AC bằng E2E tự động xong (`test-coverage-report.md`). Không còn marker `⚠️ Not confirmed`. (v1.11: không đổi nội dung file này — bump theo `design.md`/`flow.md` vì `flow.md` được thêm vào cùng version. Không đổi AC.)
Module: `identity`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này **chuyển từ** `docs/specs/features/onboarding/requirements.md` (US-002 cũ, mức user story) sang folder riêng và được viết lại cho khớp US-001 v4.0: định danh đăng nhập là **số điện thoại đã verify OTP**, không phải email. Prospective mode — chưa có code login nào để đối chiếu; code login cũ ở commit `e622180` là bản email-based, đã bị xoá ở commit `0475281`, chỉ dùng để tham khảo, không phải nguồn sự thật.

Spec này phủ ba endpoint: `login`, `refresh`, `logout`. `refresh` và `logout` trước đây nằm ngoài scope (US-001 ghi ở "Known gaps": *"`POST /v1/auth/refresh` ngoài scope"*) — nay được đưa vào đây để khép AC11 vế thứ hai của US-001.

---

## US-002: Đăng nhập

**Module:** `identity`
**Priority:** P0 (bắt buộc, chặn mọi tính năng cần user context)

## User Story

> Là một **người dùng đã có tài khoản**,
> Tôi muốn **đăng nhập bằng số điện thoại và mật khẩu**,
> Để **tiếp tục hành trình tu tập đã lưu trước đó trên thiết bị này hoặc thiết bị khác**.

## Acceptance Criteria (EARS notation)

`login` và `refresh` là **public** (người gọi chưa có access token hợp lệ); `logout` **bắt buộc `RequireAuth`** (`Authorization: Bearer <access_token>`) — xem "Auth model" ở Ghi chú kỹ thuật.

**Login — `POST /v1/auth/login`**

1. WHEN a `POST /v1/auth/login` request arrives with a `phone_number` and `password` that match a row in `users`, and a valid `device` object, THE SYSTEM SHALL return `200 OK` with `access_token`, `refresh_token` and the `user` object (cùng shape với response của `verify-otp` ở US-001), and SHALL create a new session bound to that device. (Số điện thoại được chuẩn hoá về E.164 trước khi tra cứu — cùng `validation.NormalizePhone` với US-001, nên `0901234567` và `+84901234567` là cùng một tài khoản.)
2. WHEN a login request arrives with a wrong password, a phone number that has no `users` row, or a phone number that only exists in `pending_registration` (đăng ký dở, chưa verify OTP), THE SYSTEM SHALL return the **same** `401 Unauthorized` with error code `INVALID_CREDENTIALS` and message "Số điện thoại hoặc mật khẩu không đúng" in all three cases, and SHALL NOT reveal which one occurred — neither in the body nor by a measurable response-time difference. (Tránh lộ số điện thoại nào đã đăng ký.)
3. WHEN a login request arrives with a `device` whose fingerprint the user has never used, THE SYSTEM SHALL create a separate session and SHALL NOT revoke or modify any session of the user's other devices. (Mỗi thiết bị có refresh token riêng; đăng nhập thiết bị mới không đăng xuất thiết bị cũ ở MVP.)
4. WHEN a login request arrives with a `device` whose fingerprint the user has already used, THE SYSTEM SHALL update the stored device metadata, revoke the previous active session of **that device only**, and issue a new session — so each `(user, device)` has at most one active session at any time.
5. WHEN a login request arrives with a `device` object that is missing or lacks `platform` or `device_model`, THE SYSTEM SHALL reject it with `400 Bad Request` and error code `INVALID_DEVICE` (details `{field}`), and SHALL NOT count it as a failed login attempt. (`os_version`, `app_version`, `device_name` là tùy chọn.)
6. WHEN a login request arrives with a `phone_number` that is not a valid phone number even after normalisation, or with an empty `password`, THE SYSTEM SHALL reject it with `400 Bad Request` (`INVALID_PHONE` / `INVALID_PASSWORD`, details `{field}`) and SHALL NOT count it as a failed login attempt.
7. WHEN the fifth consecutive wrong-password (or unknown-phone) attempt for one phone number is recorded, THE SYSTEM SHALL lock login for that phone number for 15 minutes; WHEN any login request for a locked phone number arrives — even with the correct password — THE SYSTEM SHALL reject it with `429 Too Many Requests`, error code `LOGIN_LOCKED` and `locked_until` in the details, without checking the password. The counter SHALL apply equally to phone numbers that have no account (so the lock cannot be used to tell which numbers are registered), and SHALL reset to zero on a successful login, once the lock has expired, or when the previous failed attempt is more than 15 minutes old (so "consecutive" means failures each within 15 minutes of the one before).
8. WHEN a login succeeds, THE SYSTEM SHALL issue an access token valid for 15 minutes and a refresh token valid for 7 days (`JWTConfig.RefreshTokenExpiration` = 168h), and SHALL store only the SHA-256 hash of the refresh token. The access token SHALL carry the claim `sid` (in `extra`) equal to the `session_id` of the session it belongs to.

**Refresh — `POST /v1/auth/refresh`**

9. WHEN a `POST /v1/auth/refresh` request arrives with a `refresh_token` that belongs to an active, unexpired, unrevoked session, THE SYSTEM SHALL return `200 OK` with a new `access_token` (carrying the same `sid` as the session) and a **new** `refresh_token` (rotation), and SHALL make the presented refresh token unusable from then on.
10. WHEN a refresh request arrives with a refresh token that is malformed, has a bad signature, is expired, belongs to a revoked/inactive session, or matches no session, THE SYSTEM SHALL return `401 Unauthorized` with error code `REFRESH_TOKEN_INVALID` — one code for all of these cases.
11. WHEN a refresh request arrives with a refresh token that was already rotated (the token immediately preceding the session's current one), THE SYSTEM SHALL return `401 REFRESH_TOKEN_INVALID` and SHALL revoke the active session of that device only (`revoked_reason = 'refresh_token_reuse'`), leaving the user's other devices' sessions untouched. (Dấu hiệu refresh token bị đánh cắp.)
12. WHEN a refresh or logout request concerns a session created by US-001's `verify-otp` (a session with no device), THE SYSTEM SHALL handle it exactly as any other session — device binding is optional at the session level. This requires US-001's access token to carry `sid` too (see "Phụ thuộc" in `design.md`).

**Logout — `POST /v1/auth/logout`**

13. WHEN a `POST /v1/auth/logout` request arrives with a valid, unexpired **access** token (`Authorization: Bearer`) whose `sid` matches an active session of the token's user, THE SYSTEM SHALL revoke that session (`is_revoked = true`, `revoked_reason = 'logout'`) and return `200` `{"success": true}`; any later refresh with that session's refresh token SHALL fail per AC10.
14. WHEN a logout request arrives with a valid access token whose `sid` matches no active session of that user (already revoked, expired, or unknown), THE SYSTEM SHALL still return `200` `{"success": true}` and SHALL NOT change any state. (Idempotent; và không thu hồi session của người khác dù `sid` bị giả — tra theo cả `user_id` lẫn `session_id`.)
15. WHEN a logout request arrives with no `Authorization` header, an invalid/expired token, a token of type `refresh`, or an access token that carries no `sid` claim, THE SYSTEM SHALL reject it with `401 Unauthorized` (error code `UNAUTHORIZED`) and SHALL NOT change any state. (Access token hết hạn → client gọi `refresh` rồi thử lại logout.)

**Web client — header `X-API-Key` (v1.12, áp cho mọi route `/v1/auth/*`, kể cả của `onboarding/`)**

16. WHEN a request to any `/v1/auth/*` route carries an `X-API-Key` header equal to the configured web key (`server.web_api_key_secret`), THE SYSTEM SHALL process it normally and treat the caller as a verified web client.
17. WHEN a request to any `/v1/auth/*` route carries an `X-API-Key` header that is wrong or empty — or the key is not configured at all — THE SYSTEM SHALL reject it with `401 Unauthorized`, error code `INVALID_API_KEY`, and SHALL NOT run the handler or count it as a failed login attempt. The comparison SHALL be constant-time.
18. WHEN a request carries no `X-API-Key` header, THE SYSTEM SHALL process it as a mobile caller, exactly as before v1.12.
19. WHEN a login request has `device.platform` empty, THE SYSTEM SHALL store `mobile`; WHEN `device.platform` is `web` (case-insensitive) and the request did not present a valid `X-API-Key`, THE SYSTEM SHALL reject it with `401` `INVALID_API_KEY` before calling the use case. Any other `platform` value is stored as sent. (`device_model` vẫn bắt buộc — AC5.)

**Response cấp token — `verify-otp` (onboarding), `login`, `refresh` (v1.13)**

20. WHEN any of `POST /v1/auth/register/verify-otp`, `POST /v1/auth/login` or `POST /v1/auth/refresh` responds — success or error — THE SYSTEM SHALL include the headers `Cache-Control: no-store` and `Pragma: no-cache`, so that no browser, proxy or CDN stores a response that carries (or could carry) a token.
21. WHEN `verify-otp` or `login` succeeds, THE SYSTEM SHALL add `expires_in` (integer, seconds) to the `data` object, equal to the access token's lifetime from `JWTConfig.AccessTokenExpiration` (900 at dev config), alongside `access_token`, `refresh_token` and `user`.
22. WHEN `refresh` succeeds, THE SYSTEM SHALL add the same `expires_in` to `data`, alongside the new `access_token` and `refresh_token`; `data` still carries no `user`.

`expires_in` chỉ nói về **access token** (đúng nghĩa `expires_in` của OAuth 2.0). Hạn của refresh token (7 ngày) không có trong response: client không cần đọc nó, vì refresh token hết hạn đã có `401 REFRESH_TOKEN_INVALID` (AC10). Giá trị này là TTL cấu hình, không phải `exp` đọc lại từ token, nên client có thể lên lịch refresh mà không phải giải mã JWT.

`X-API-Key` là **secret dùng chung của ứng dụng web**, giữ ở server SvelteKit; nó xác định *website nào đang gọi*, khác với access/refresh token xác định *user nào*. Nó **không** chặn caller bỏ header (AC18) — khoảng trống này là nợ `docs/tech-debt/auth/TD-001`.

## API contract (confirmed surface for this feature)

Path và shape dưới đây đã được duyệt 2026-09-21; logout dùng access token trong header (chốt 2026-09-21). Cả ba path đã có trong `docs/specs/api/openapi.yaml` (v3.7) — viết **sau khi** handler chạy thật, theo tiền lệ US-001.

| Method | Path | Auth | Request | Success | Failure codes |
|---|---|---|---|---|---|
| POST | `/v1/auth/login` | public | `{ phone_number, password, device: { platform, device_model, os_version?, app_version?, device_name? } }` | `200` `{ access_token, refresh_token, expires_in, user }` | `400` `INVALID_PHONE` / `INVALID_PASSWORD` / `INVALID_DEVICE`, `401` `INVALID_CREDENTIALS` / `INVALID_API_KEY`, `429` `LOGIN_LOCKED` (+`locked_until`), `500` |
| POST | `/v1/auth/refresh` | public | `{ refresh_token }` | `200` `{ access_token, refresh_token, expires_in }` | `400` malformed body, `401` `REFRESH_TOKEN_INVALID` / `INVALID_API_KEY`, `500` |
| POST | `/v1/auth/logout` | **RequireAuth** — `Authorization: Bearer <access_token>` | không có body | `200` `{"success": true}` | `401` `UNAUTHORIZED` / `INVALID_API_KEY`, `500` |

## Out of scope (this feature boundary, not tracked here)

- Đăng nhập qua social login (Google/Apple) — giai đoạn sau.
- **Đăng nhập bằng email** — email chưa verify (`email_verified_at IS NULL`) bị gate theo US-001 AC13, mà luồng xác thực email chưa tồn tại; nên US-002 chỉ có định danh SĐT. Khi có US xác thực email sẽ cân nhắc thêm.
- **Đăng nhập bằng SĐT + OTP** — `otp_codes.purpose = 'login'` và `resend-otp` đã có chỗ sẵn nhưng không dùng ở US này.
- Đăng xuất tất cả thiết bị khác ("logout everywhere") — giai đoạn sau.
- Xác thực 2 lớp (2FA) — giai đoạn sau.
- `refresh_expires_in` / hạn tuyệt đối của refresh token trong response — client không cần (xem AC21–AC22); thêm khi có nhu cầu thật.
- Đưa token vào cookie `HttpOnly` từ BE — BE vẫn chỉ trả token trong JSON body; nơi FE lưu token là việc của FE (server SvelteKit đặt cookie). Khuyến nghị cho FE: cookie `HttpOnly; Secure; SameSite` đặt bởi server SvelteKit, không dùng `localStorage`.
- Rate limit theo IP cho `login`/`refresh` (khác với khoá theo SĐT ở AC7) — cân nhắc ở middleware, xem tasks.md Deferred.
- Quên/đặt lại mật khẩu (US-003), hồ sơ (US-004) — vẫn ở `docs/specs/features/onboarding/requirements.md`.

## Pending

Feature đã implement và verify xong (2026-09-21) — xem `tasks.md` Phase 1–5. Audit AC + E2E binary thật: `test-coverage-report.md` (AC1–AC15 đều có test). Concept OKF: `docs/knowledge/features/login.md` (`stable`). `go test -short ./...` toàn repo xanh.

Hai phụ thuộc chặn trước đây đã xong: `jwt-token-uniqueness` (2026-09-21) và sửa `TokenIssuer`/`CreateUserAndSession` của US-001 để access token mang `sid` (`onboarding/tasks.md` Phase 7).

## Ghi chú kỹ thuật liên quan

- **Auth model (chốt 2026-09-21):** `login`/`refresh` public vì người gọi chưa có access token hợp lệ. `logout` **bắt buộc `RequireAuth`** — đây là route đầu tiên trong repo được bọc `pkg/jwt/middleware` ở `cmd/main.go` (các route user-scoped khác như `/v1/users/me` chưa được triển khai). Middleware cấu hình `RequireTokenType = "access"` (để refresh token không dùng làm bearer) và `ErrorHandler` riêng để 401 có cùng envelope `utils.ApiResponse` (`{success, error{code,message,type}, details}`) với các route khác. (Từ `api-response-envelope` v1.0, mặc định của middleware cũng đã là envelope này; `ErrorHandler` riêng vẫn giữ để message luôn là `unauthorized`, không lộ lý do token bị từ chối.)
- **Claim `sid` (chốt 2026-09-21).** Access token mang `extra{"sid": <session_id>}` để logout biết session nào của thiết bị nào mà không cần refresh token. `session_id` là `uuid.NewString()` (cùng cách `CreateUserAndSession` của US-001 đang làm) và phải được sinh **trước** khi mint token trong cùng transaction — nên `domain.TokenIssuer` đổi thành `func(userID, sessionID string)`. Đổi này chạm code đã ship của US-001; xem `design.md` "Phụ thuộc: sửa US-001".
- **Định danh & gate email:** login chỉ tra `users.phone_number`. Không có nhánh nào đọc `users.email` hay `email_verified_at`, nên US-001 AC13 (email chưa verify không login được) được thoả mặc nhiên — không có đường nào để email làm định danh.
- **Sai thông tin đăng nhập phải không phân biệt được** (AC2): cần (a) một sentinel duy nhất `ErrInvalidCredentials`, (b) khi SĐT không tồn tại vẫn chạy `bcrypt.CompareHashAndPassword` với một hash giả cố định để thời gian phản hồi không lộ tài khoản tồn tại.
- **Khoá login (AC7) lưu ở DB**, không ở `pkg/ratelimit` in-memory: sống qua restart và đúng khi Cloud Run chạy nhiều instance. Bảng `login_attempts` khoá theo `phone_number`, không phụ thuộc `users` nên áp được cho cả SĐT không tồn tại. Ngưỡng (5 lần / 15 phút) là cấu hình (`config.LoginConfig`), cùng cách `config.OTPConfig` đã làm cho OTP.
- **Device (AC3–AC5):** client gửi `device{platform, device_model, os_version?, app_version?, device_name?}`; server thêm `ip_address` và `user_agent` từ request. `device_id` = fingerprint `SHA-256(platform|os_version|device_model|user_agent)` tính **phía server**, không tin client gửi fingerprint. Ràng buộc `UNIQUE(user_id, device_id)` làm chốt upsert. Phiên tạo bởi US-001 không có device (`user_sessions.device_id` nullable) — AC12.
- **Refresh token phải duy nhất cho từng session/lần rotate — do `pkg/jwt` đảm bảo, không phải login.** `pkg/jwt` hiện không sinh `jti` nên hai token cùng user trong cùng một giây trùng byte (đã tái hiện). Lỗi này được tách ra và sửa ở gốc tại [`../jwt-token-uniqueness/`](../jwt-token-uniqueness/requirements.md); login **dùng lại** `domain.TokenIssuer` của US-001 và **không** nhét nonce vào `extra`. **Phụ thuộc cứng:** `tasks.md` Phase 2–3 không được bắt đầu trước khi `jwt-token-uniqueness` Phase 2 xong.
- **Reuse detection chỉ phủ một đời** (AC11): `user_sessions.previous_refresh_token_hash` giữ hash của token ngay trước đó. Token cũ hơn hai đời trở lên chỉ nhận 401 thường (không match hàng nào), không kích hoạt revoke.
- Access token 15 phút, refresh token 7 ngày — theo `config.dev.yaml`. Dòng "refresh token 30 ngày" ở `onboarding/requirements.md` §Ghi chú kỹ thuật (số cũ) **đã được sửa thành 7 ngày** ở onboarding v5.2.
- **Access token sau logout / revoke (chốt 2026-09-21):** giữ stateless — access token đã cấp vẫn dùng được tới hết hạn (≤ 15 phút), middleware **không** tra `user_sessions`. Ghi ở `design.md` Known gaps; xem xét lại khi wire `RequireAuth` cho US-004.
- **Cửa sổ đếm sai (chốt 2026-09-21):** `FailureResetWindow` = 15 phút (bằng `LockDuration`), lưu bằng cột `login_attempts.last_failed_at`. Khoá tăng dần (15 phút → 1 giờ → 24 giờ) để lại sau.
- **Session sliding (chốt 2026-09-21):** mỗi lần `refresh` thành công đặt `expires_at = now() + sessionTTL` (7 ngày). Trần tuyệt đối (ví dụ 90 ngày, cột `absolute_expires_at`) để lại sau.
- Thuộc `LoginUsecase` trong `usecase/` (tách khỏi `RegisterUsecase`), dùng chung `TokenIssuer` và `validation.NormalizePhone` của US-001.
- Dependency rule như US-001: `delivery → usecase → domain`, `repository → domain`.

## Open questions (resolved during implementation)

Đã chốt qua phỏng vấn 2026-09-20 (trước khi viết spec):

- ~~Định danh đăng nhập là email hay số điện thoại?~~ → **SĐT + mật khẩu**; email login không hỗ trợ. Resolved 2026-09-20.
- ~~`POST /v1/auth/refresh` có trong US-002 không?~~ → **Có**, kèm rotate refresh token. Resolved 2026-09-20.
- ~~Client có gửi thông tin thiết bị lúc login không?~~ → **Có**: `platform` + `device_model` bắt buộc, còn lại tùy chọn. Resolved 2026-09-20.
- ~~Login lại trên cùng thiết bị xử lý session cũ thế nào?~~ → **Upsert device, revoke session cũ của đúng device đó**. Resolved 2026-09-20.
- ~~Khoá brute-force cài ở đâu, ngưỡng bao nhiêu?~~ → **Bảng DB đếm theo SĐT; 5 lần sai → khoá 15 phút**. Resolved 2026-09-20.
- ~~Refresh token đã rotate bị dùng lại thì sao?~~ → **401 + revoke session của device đó**. Resolved 2026-09-20.
- ~~Thời hạn refresh token: 7 hay 30 ngày?~~ → **7 ngày** (khớp config + migration). Resolved 2026-09-20.
- ~~Logout thường có trong US-002 không?~~ → **Có** `POST /v1/auth/logout`. Cách xác định session còn mở — xem Pending. Resolved 2026-09-20.
- ~~Bộ đếm "5 lần sai liên tiếp" có cửa sổ thời gian không?~~ → **Có, 15 phút** (đếm lại từ 1 nếu lần sai gần nhất cách > 15 phút). Resolved 2026-09-21.
- ~~Sau logout, access token còn sống ≤ 15 phút có chặn không?~~ → **Không chặn** (stateless). Resolved 2026-09-21.
- ~~`refresh` gia hạn session kiểu sliding hay absolute?~~ → **Sliding**. Resolved 2026-09-21.
- ~~Path/shape 3 endpoint có được chốt không?~~ → **Duyệt như đề xuất**; `openapi.yaml` viết sau khi handler chạy thật. Resolved 2026-09-21.
- ~~`logout` xác định session bằng gì: refresh token (body/header) hay access token + `sid`?~~ → **Access token trong header `Authorization: Bearer` + claim `sid` (phương án B), `RequireAuth`.** Kéo theo sửa `TokenIssuer` và `CreateUserAndSession` của US-001. Resolved 2026-09-21.
- ~~US-002 đặc tả ở folder nào?~~ → **Folder `login/` riêng**; `onboarding/` không còn US-002. Resolved 2026-09-20.
