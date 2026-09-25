# Design — onboarding

Version: 6.1
Last updated: 2026-09-25
Status: done (US-001) — `domain/`, `usecase/`, `repository/`, `delivery/http/`, `cmd/main.go`,
`pkg/sms/`, `migrations/0001`+`0003` và `docs/specs/api/openapi.yaml` đều đã có code/tài liệu thật,
và đã chạy thật end-to-end (request-otp → resend-otp → verify-otp) qua Postgres. `tasks.md` không còn
`[ ]`: `go test -short ./...` toàn repo xanh; repository + E2E chạy Postgres thật xanh; job dọn hàng
hết hạn chốt chấp nhận ở MVP (Deferred). **Không còn marker `⚠️ Not confirmed`** — nhánh reset bộ đếm
gửi đã chốt ở v5.1. (v6.1: chuẩn hoá theo quy ước `il-spec-writer` hiện hành — thêm `flow.md` (bản vẽ Mermaid mirror 1:1 ba Step-by-step Flow), đổi Request flow ASCII thành con trỏ `flow.md`, AC của `requirements.md` viết lại sang EARS; không đổi code/AC/behavior. v5.2: sửa hai chỗ tài liệu cũ — helper `hashRefreshToken()` của repository nay là `domain.HashRefreshToken` (`domain/session_token.go`), và câu "refresh token 30 ngày" ở `requirements.md` đã thành **7 ngày**; không đổi code/AC. v5.1: **tách `PendingRegistrationTTL` khỏi OTP TTL (24h thay 90s)**;
`FindConflicts` không còn chặn bằng pending của chính SĐT — request-otp ghi đè payload để làm lại;
resend-otp cấp OTP mới kể cả khi OTP cũ đã hết hạn, miễn payload còn; đổi AC6 + thêm AC15; code +
test xong. v5.0: **đổi AC5/AC12** — bỏ trần resend carry-forward chặn vĩnh viễn, thay bằng
`otp_send_limits` + khoá-gửi 30 phút; `request-otp` cũng tiêu một lượt; thêm `OTP_SEND_LIMITED`; bỏ
`ErrOTPResendExhausted`; code + test xong ở Phase 9 (2026-09-22). v4.5: flow 1 Step 2 — `FindConflicts`
chỉ tính hàng pending **còn hạn**; v4.4: chỉ cập nhật Status; v4.3: sửa mô tả envelope response;
v4.2: Phase 7 xong — access token mang claim `sid`; v4.1: thêm thay đổi kế hoạch `sid`; v4.0: chỉ đổi
con trỏ US-002 → `login/`.)
Module: `identity`
Code: `services/appService/`
Feature folder: `services/appService/` + `docs/specs/features/onboarding/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này chỉ phủ **US-001 (đăng ký tài khoản)**. Không còn ở chế độ prospective: bảng Layers bên
dưới chia hai nhóm — dòng nào còn marker `*(chưa tồn tại)*` là chưa viết, dòng không có marker là
đã có file thật trên đĩa. Commit `0475281` đã xoá `domain/`, `usecase/`, `repository/`,
`delivery/`, `cmd/main.go` và cả 7 file migration; bốn nhóm trong số đó đã được viết lại
(`domain/`, `usecase/`, `repository/`, và migration `0001`).

`requirements.md` giữ nguyên bản prose — không câu nào của nó bị sửa trong lần này. Header của nó
đã được bump lên cùng version với hai file kia (xem `changelog.md` v3.1) để ba file mang một số
version chung. Các tiêu chí `AC1..AC14` được tham chiếu theo số, không chép lại.

**Lưu ý về endpoint thứ ba.** `requirements.md` §Ghi chú kỹ thuật liệt kê 2 endpoint và tự ghi rõ
*"Tên path là đề xuất, chưa chốt"* — danh sách đó không ràng buộc. Bản design này chốt **ba**
endpoint: thêm `POST /v1/auth/resend-otp` phục vụ AC5, thiết kế **dùng chung cho mọi `purpose`**
(không chỉ `register`) để US-002/US-003 tái sử dụng. Đây là quyết định ở tầng design; danh sách
endpoint trong `requirements.md` §Ghi chú kỹ thuật cần cập nhật khi người dùng cho phép sửa file đó
— ghi ở "Cross-file follow-ups" của `changelog.md`.

## Overview

US-001 là luồng **đăng ký 2 bước, verify OTP trước — tạo account sau**. Bất biến trung tâm: **không
tồn tại hàng `users` nào ở trạng thái "chưa verify"**. Hệ quả là mọi thứ thuộc về lượt đăng ký đang
dở — kể cả `password_hash` — phải nằm ở một bảng tạm (`pending_registration`) cho tới khi OTP đúng.

Cả **ba** route đều **public** (không `RequireAuth`) — người gọi chưa có tài khoản, nên không có gì
để xác thực. (Các route user-scoped như `GET/PATCH /v1/users/me` chưa được triển khai.)

US-001 là feature **write-heavy nhiều bảng**: một lượt verify OTP thành công ghi `users`,
`user_sessions`, xoá `pending_registration` và đánh dấu `otp_codes` — bốn thay đổi trạng thái, nên
bắt buộc có Step-by-step Flow (rule 5 của skill). Nó cũng là feature đầu tiên chạm tới
`pkg/sms`, một dependency ngoại vi đã có ở dạng log-only (`pkg/sms/logsms`, xem "Deliberately stubbed for now").

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/user.go` | `User` — `ID`, `Email`, `EmailVerifiedAt *time.Time`, `PhoneNumber`, `PasswordHash`, `DharmaName *string`, `OrdinationPlace *string`, `CreatedAt`, `UpdatedAt`. Plain data, không JSON tag. |
| Domain / Entity | `services/appService/domain/pending_registration.go` | `PendingRegistration` — payload tạm giữa 2 bước: `PhoneNumber`, `Email`, `PasswordHash`, `DharmaName`, `OrdinationPlace`, `ExpiresAt`. Thêm `ID` và `CreatedAt` so với bản thiết kế gốc — `ID` để `DeletePendingRegistration` có khoá, `CreatedAt` để chẩn đoán. |
| Domain / Entity | `services/appService/domain/otp.go` | `OTPCode` + `OTPPurpose`. Hằng số khai **đủ bốn** giá trị (`OTPPurposeRegister = "register"`, `OTPPurposeLogin = "login"`, `OTPPurposeResetPassword = "reset_password"`, `OTPPurposeVerifyEmail = "verify_email"`) để cột `otp_codes.purpose` và endpoint `resend-otp` dùng chung được — nhưng US-001 **chỉ hiện thực** `register`; ba giá trị còn lại thuộc US-002/US-003, khai ở đây để `chk_otp_codes_purpose` không phải `ALTER` lại. Không có entity `Device`/`Session` ở phạm vi US-001 — xem "Known gaps". |
| Domain / Input-Output | `services/appService/domain/register.go` | `RequestRegisterOTPInput`, `RequestOTPResult`, `ResendOTPInput` (`PhoneNumber`, `Purpose`), `ResendOTPResult`, `VerifyRegisterOTPInput`, `RegisterResult`, `RegistrationConflicts` (+ `HasPhoneConflict()` / `HasEmailConflict()`), `TokenIssuer` func type `func(userID, sessionID string)` (đã đổi ở Phase 7/v4.2 — xem Step 6 flow 2). Không HTTP DTO. |
| Domain / Errors | `services/appService/domain/errors.go` | Sentinels: `ErrPhoneTaken`, `ErrEmailTaken`, `ErrInvalidPhone`, `ErrInvalidEmail`, `ErrInvalidPassword`, `ErrOTPInvalid`, `ErrOTPExpired`, `ErrOTPLocked`, `ErrOTPResendTooSoon`, `ErrOTPSendLimited`, `ErrPendingRegistrationNotFound`, `ErrPendingRegistrationExpired`, `ErrUnsupportedOTPPurpose`, `ErrSMSSendFailed`. Handler map từng cái sang status. Ba typed error `OTPLockedError` / `OTPResendTooSoonError` / `OTPSendLimitedError` (mỗi cái `Unwrap()` về sentinel tương ứng) mang thêm `locked_until` / `retry_after`. **v5.0 bỏ `ErrOTPResendExhausted`** (không còn nhánh nào trả), **thêm `ErrOTPSendLimited`**. |
| Domain / Entity | `services/appService/domain/otp_send_limit.go` | `OTPSendLimit` — state của bộ đếm gửi theo SĐT: `PhoneNumber`, `Purpose`, `SendCount`, `LastSentAt *time.Time`, `LockedUntil *time.Time`. Kèm value type `SendLimits` (policy: `Cooldown`, `LockDuration`, `MaxAttempts`) mà usecase truyền xuống repository, để `domain` không phải import `config`. |
| Domain / Repository interface | `services/appService/domain/register.go` | `RegistrationRepository` — interface do `repository/` implement, khai báo ở `domain/`, không khai ở `repository/`. Mười method (v5.0: `SavePendingAndIssueOTP`/`ReissueOTP` nhận thêm `SendLimits`, thêm `ReleaseSendLimit`): `FindConflicts`, `SavePendingAndIssueOTP`, `ReissueOTP`, `FindPendingRegistration`, `FindActiveOTP`, `IncrementOTPAttempts`, `LockOTP`, `CreateUserAndSession`, `DeletePendingRegistration`, `ReleaseSendLimit`. |
| Domain / Port | `services/appService/domain/register.go` | `OTPSender` interface — **một method duy nhất** `SendOTP(ctx context.Context, phoneNumber, otpCode string, purpose OTPPurpose) error`. Không tách `SendRegistrationOTP` riêng: flow `resend-otp` dùng chung đường gửi với flow 1, và US-002/US-003 sẽ gọi đúng method này với `purpose` khác. usecase không import `pkg/sms` trực tiếp. |
| Domain / Mock | `services/appService/domain/mock_registration_repository.go` | Mock tay cho usecase test — không DB thật. Thêm `mock_otp_sender.go` cho port `OTPSender`, cùng package, cùng lý do. |
| Usecase | `services/appService/usecase/register_usecase.go` | Ba method: `RequestOTP` (validate → check trùng → áp send-limit → hash password → ghi tạm → gửi SMS), `ResendOTP` (validate → tra trạng thái pending → kiểm khoá-nhập-sai → áp send-limit → cấp OTP mới → gửi SMS), `VerifyOTP` (validate → tra OTP → so hash → tạo user + session trong 1 tx). **Không** sở hữu các con số của OTP policy — chúng đến từ `config.OTPConfig` (`services/appService/config/`) qua constructor, dưới dạng `config.OTPConfig` (usecase import thẳng `services/appService/config` — không có struct `OTPPolicy` trung gian, để chỉ tồn tại một định nghĩa duy nhất). Bảy giá trị (v5.1): `TTL` (90s), `SendCooldown` (60s), `MaxSendAttempts` (3), `SendLockDuration` (30m), `MaxAttempts` (3), `LockDuration` (30m), `PendingRegistrationTTL` (24h — tách khỏi `TTL` từ v5.1). Đã bỏ `MaxResend`/`ResendCooldown`. |
| Usecase / OTP source | `services/appService/usecase/register_usecase.go` | `generateOTP()` — **hằng số `111111`** ở giai đoạn này (requirements §Ghi chú kỹ thuật), tách riêng một hàm để sau chỉ đổi một điểm sang `crypto/rand`. |
| Repository | `services/appService/repository/postgres_registration_repository.go` | Toàn bộ SQL + transaction boundary của feature. Method export: `FindConflicts`, `SavePendingAndIssueOTP`, `ReissueOTP`, `FindPendingRegistration`, `DeletePendingRegistration`, `FindActiveOTP`, `IncrementOTPAttempts`, `LockOTP`, `CreateUserAndSession`, `ReleaseSendLimit`. Helper nội bộ: `nullableString()` (chuỗi rỗng → SQL NULL, AC9), `isUniqueViolation()` (dịch `pq` code `23505` + tên constraint sang sentinel domain). Việc hash refresh token (SHA-256 hex, flow 2 Step 6) nằm ở `domain.HashRefreshToken` (`domain/session_token.go`) vì cả usecase lẫn repository đều dùng — không còn helper riêng trong repository. `FindActiveOTP`'s thật sự dùng điều kiện `(locked_until > now() OR (used_at IS NULL AND expires_at > now()))` — tinh chỉnh so với câu SQL literal ở flow 2 Step 3 bên dưới, xem ghi chú ngay dưới bảng này. |
| Repository / Storage | `services/appService/repository/gcs_avatar_storage.go` | **Không thuộc US-001** — thuộc US-004. Không tạo trong feature này. |
| Repository / Port adapter | `services/appService/repository/sms_otp_sender.go` | `smsOTPSender` implement `domain.OTPSender` bằng cách bọc `pkg/sms.Sender`, format `otpCode` thành message text. **Không có trong bản thiết kế Layers gốc** — bổ sung ở Phase 5 vì `domain.OTPSender` cần `otpCode`+`purpose` còn `pkg/sms.Sender` cố ý chỉ biết "gửi text tới một số" (để `pkg/sms` không phụ thuộc domain type của service này). Sống ở `repository/` vì đó là nơi mọi domain port khác được implement bằng infra thật. |
| Delivery / Handler | `services/appService/delivery/http/register_handler.go` | Ba handler (`RequestOTP`, `ResendOTP`, `VerifyOTP`), DTO request/response (+ `userResponse` — không phải `domain.User` trực tiếp), decode strict, map domain error → status + error code. |
| Delivery / Shared | `services/appService/delivery/http/errors.go` | Helper map error → envelope `utils.ApiResponse` dùng chung; US-001 là consumer đầu tiên. Envelope thật: lỗi = `{success: false, error: {code, message, type}, details}` (`details` nằm **cạnh** `error`, không lồng trong; `type` ∈ `validation`/`business`/`system`), thành công = `{success: true, data: {...}}`. Các body `Success → 200 {...}` / `201 {...}` trong Step-by-step Flow bên dưới là nội dung của `data`. |
| Wiring | `services/appService/cmd/main.go` | Constructor injection theo thứ tự phụ thuộc + `mux.HandleFunc` cho 3 route (`POST /v1/auth/register/request-otp`, `POST /v1/auth/resend-otp`, `POST /v1/auth/register/verify-otp`) + đăng ký `validation.PhoneTag`/`SetFieldNameTag("json")`. Viết lại từ template cũ (commit `e622180`, trước luồng 2 bước), giữ style `⚠️ DEV ONLY`/`⚠️ DEV CONFIG`; bỏ phần thuộc US-002/003/004/content chưa có usecase. |
| External package | `pkg/sms/sms.go` | Interface `Sender` + `Register(type, FactoryFunc)` / `New(config)` theo pattern provider-agnostic của repo. `Config` có thêm field `Logger logging.Logger` (không có ở `pkg/database`'s Config) vì impl duy nhất hiện có (log-only) cần một sink để ghi. |
| External package | `pkg/sms/logsms/log_sms.go` | Implementation **chỉ ghi log** nội dung SMS; tự đăng ký qua `init()`. Đây là impl duy nhất ở giai đoạn này. |
| Migration | `migrations/0001_register_onboarding.sql` | Baseline `users` + `users.email_verified_at`, `users.phone_number`, `users.ordination_place`; tạo `otp_codes`, `pending_registration`, `user_sessions`. |
| Migration | `migrations/0003_otp_send_limits.sql` **(v5.0, đã có code)** | `CREATE TABLE otp_send_limits` (xem §"OTP send-limit policy") + `ALTER TABLE otp_codes DROP COLUMN resend_count` (bộ đếm cũ bỏ hẳn). `0002` thuộc `login/`. |
| API spec | `docs/specs/api/openapi.yaml` | OpenAPI 3.0.3, 3 path + schema request/response/error. Viết sau khi handler đã chạy thật qua `curl`, nên mọi field/status/code khớp code thật — không phải một bản thiết kế lý thuyết đi trước code. |

**`FindActiveOTP`'s real query, vs. the literal SQL in flow 2 Step 3 below.** Step 3's SQL snippet
(`WHERE ... AND used_at IS NULL ORDER BY created_at DESC LIMIT 1`) is what a fresh, unlocked OTP
matches, but taken literally it breaks two things repository/'s own tests caught: (a) an OTP whose
`expires_at` has passed but was never marked `used_at` (no cron clears it) would still read as
"active" and a stale code could still verify — AC4 broken; (b) `LockOTP` sets `used_at = now()`
*together with* `locked_until` (AC12), so the literal `used_at IS NULL` filter would make a locked
row invisible to the very check meant to report `OTP_LOCKED` on the caller's next attempt. The
implemented condition is `(locked_until > now() OR (used_at IS NULL AND expires_at > now()))`: a row
is active while genuinely unused-and-unexpired, or while its lock is still in force regardless of
the original 90s `expires_at` — and once `locked_until` itself passes, the row is dead for good (the
caller must call request-otp again, per flow 2 Step 4's "hết 30 phút... phải gọi lại request-otp").
This matches the doc comment already on `domain.OTPCode` ("FindActiveOTP matches on UsedAt OR
LockedUntil rather than UsedAt alone") written during Phase 1/2 — Phase 3 only made that comment's
SQL literal. Confirmed by
`repository/postgres_registration_repository_test.go`'s `TestRegistrationRepository_FindActiveOTP`.

Dependency rule: `delivery → usecase → domain`; `repository → domain`; `domain` không import gì từ
layer ngoài. DTO HTTP chỉ sống trong `delivery/http/register_handler.go`.

## OTP send-limit policy (v5.0)

Trần resend cũ (`otp_codes.resend_count` **carry-forward** mãi mãi) chặn SĐT **vĩnh viễn** — không
có đường gỡ. v5.0 thay bằng một bộ đếm **theo SĐT**, sống ở bảng riêng `otp_send_limits`, có khoá
tạm 30 phút. Bảng riêng (không nhồi vào `otp_codes`) vì hai lý do:

1. Bộ đếm phải sống lâu hơn hàng OTP 90 giây. Gắn vào hàng OTP là gốc của bug cũ.
2. **Khoá-gửi không được chặn `verify-otp`.** `LockOTP` trên `otp_codes` set `used_at` và bị
   `FindActiveOTP` (+ `VerifyOTP`) tôn trọng — đúng cho khoá-nhập-sai, sai cho khoá-gửi. Nếu dùng
   chung, OTP hợp lệ cuối cùng vừa gửi sẽ không nhập được trong 90 giây còn hạn của nó.

Hai cơ chế khoá **độc lập, cùng thời lượng 30 phút**:

| | Khoá-gửi | Khoá-nhập-sai |
|---|---|---|
| State | `otp_send_limits.locked_until` | `otp_codes.locked_until` (+ `used_at`) |
| Trigger | lượt gửi thứ 4 | 3 lần nhập sai OTP |
| Chặn | `request-otp`, `resend-otp` | `verify-otp` (và mọi lần gửi) |
| Không chặn | `verify-otp` | — (chặn cả gửi) |
| Mã lỗi | `429 OTP_SEND_LIMITED` | `429 OTP_LOCKED` |
| Config | `send_lock_duration` (30m) | `lock_duration` (30m) |

### Bộ đếm gửi — state machine theo `(phone_number, purpose='register')`

`send_count` = số **lượt tính** trong lượt hiện tại, **không** tính lần gửi đầu. `last_sent_at` =
lần gửi gần nhất. Một lượt (cycle) bắt đầu bằng lần gửi đầu (miễn phí, `send_count = 0`) và tự
đặt lại khi `now >= last_sent_at + send_lock_duration` (30 phút).

```
Lần gửi kế tiếp (request-otp hoặc resend-otp), sau khi qua validate/conflict/pending:
  (0) nếu chưa có hàng, HOẶC now >= last_sent_at + 30m     -- lần gửi đầu của cycle, MIỄN PHÍ
        → send_count = 0, last_sent_at = now, locked_until = NULL
        → cho gửi (COMMIT rồi mới gọi SMS)
      ngược lại (đang trong cycle):
  (a) nếu locked_until > now                  → 429 OTP_SEND_LIMITED + locked_until
  (b) nếu now < last_sent_at + send_cooldown  → 429 OTP_RESEND_TOO_SOON + retry_after_seconds
  (c) nếu send_count >= max_send_attempts (3) → locked_until = now + 30m
                                                 → 429 OTP_SEND_LIMITED + locked_until
  (d) ngược lại: send_count++; last_sent_at = now
                 nếu send_count >= 3          → locked_until = now + 30m
                 → cho gửi (COMMIT rồi mới gọi SMS)
```

Lượt gửi 1 (đầu): nhánh (0), `send_count = 0`, miễn phí. Lượt 2/3/4 (cách nhau ≥ 60s): nhánh (d),
`send_count` thành 1/2/3; lượt 4 **vẫn gửi** (đúng "tổng 4 lượt") và set `locked_until`. Lượt 5
trong 30 phút: nhánh (a)/(c) → `OTP_SEND_LIMITED`. Hết 30 phút, nhánh (0) đặt lại.

Chú thích:
- (0) và (d) tách nhau đúng một điểm: lần gửi **đầu** của cycle không tính lượt, các lần sau tính.
  `send_count` vì vậy là "số lượt đã tính", bằng `0` sau lần đầu và bằng `3` sau lượt thứ 4.
- (a) trước (b)/(c): đang trong khoá thì thời gian cooldown không quan trọng, báo khoá là thông tin
  hành động được (kèm `locked_until`).
- (b) trước (c): ngay sau lượt 4, trong 60s, caller nhận `OTP_RESEND_TOO_SOON`; đây là nhánh duy
  nhất có `retry_after_seconds`.
- **Không** dùng bảng này để chặn `verify-otp`. `VerifyOTP` chỉ đọc `locked_until` của `otp_codes`
  (khoá-nhập-sai) và `attempts` như cũ.
- Bộ đếm là per-`purpose` để `login`/`reset_password` sau này không dùng chung hạn mức với
  `register`. US-001 chỉ tạo hàng cho `register`.
- **Atomicity:** (0)–(d) và `INSERT otp_codes` (+ UPSERT `pending_registration` ở flow 1) nằm trong
  **cùng MỘT transaction** của repository — một lượt bị chặn ở (a)/(b)/(c) không ghi gì, và một lượt
  được phép không thể "tiêu lượt mà không gửi OTP". Flow 3 (resend) chỉ ghi OTP, không đụng payload
  (v5.1).

### Confirmed (v5.1, 2026-09-22)

- Nhánh (0) đặt lại khi **lần gửi cuối đã quá 30 phút** (`now >= last_sent_at + send_lock_duration`),
  dùng chung một con số cho "khoá hết" và "cycle cũ". Anh đã chốt: quá 30 phút thì request-otp /
  resend-otp mở lại, và lần gửi đó là lần đầu **miễn phí** của cycle mới.

## Pending payload và đường "làm lại" (v5.1)

`PendingRegistrationTTL` được **tách khỏi** OTP TTL: 24h thay vì 90s (trước v5.1 hai con số bằng
nhau). Lý do: sau khi khoá-gửi 30 phút hết, user phải còn payload để "gửi lại" hoặc "đăng ký lại" mà
không mất email/password. Với TTL 90s thì payload chết trước khoá rất lâu và resend-otp luôn trả lỗi.

Hai đường làm lại, cùng bị khoá-gửi gác (còn trong 30 phút → 429):

| Đường | Dùng khi | Payload | Điều kiện |
|---|---|---|---|
| `request-otp` | user cài lại app, nhập lại email/password | **ghi đè** (UPSERT theo SĐT) | không còn là conflict; chỉ chặn bởi send-limit |
| `resend-otp` | user còn app, bấm "gửi lại" | **giữ nguyên** | pending còn hạn (24h); OTP cũ hết hạn không sao |

- `FindConflicts` (AC6) **không** còn tính hàng pending của **chính SĐT** là trùng; chỉ email của một
  SĐT **khác** mới chặn. Nhờ vậy cả hai đường đi qua được khi hết khoá. Xem AC15.
- OTP cũ hết hạn (quá 90s) **không** làm `resend-otp` trả `OTP_EXPIRED` nữa — nó cấp OTP mới. Đây là
  điều kiện để "sau 30 phút bấm gửi lại" hoạt động, vì lúc đó OTP chắc chắn đã hết hạn.
- Đánh đổi có ý thức: `password_hash` nằm trên server tới 24h. Job dọn hàng hết hạn (Step 6, đã hoãn)
  là biện pháp bù; TTL 24h là chốt chặn cuối nếu job chưa chạy.

## Request flow

> Xem `flow.md` cho sơ đồ sequence Mermaid (mirror 1:1 của Step-by-step Flow bên dưới).

## Step-by-step Flow — Đăng ký bước 1: yêu cầu OTP

```
Step 1 — Validate và chuẩn hoá input (feature này)
- Trigger: RegisterHandler nhận POST /v1/auth/register/request-otp với body decode được
- Pre-condition: DecodeJSONBodyStrict pass (field lạ bị từ chối, cap 1 MB)
- Call: RegisterUsecase.RequestOTP → validation.NormalizeEmail → validator (required,email) →
        normalizePhone (xem bên dưới) → phoneE164Regex → len(password) >= 8
  Ghi chú: requirements AC7/AC8 mô tả validate ở phía client ("không gửi request lên server").
  Server vẫn phải validate lại — AC7/AC8 là ràng buộc UX, không phải miễn trừ kiểm tra phía server.
  Confirmed (đã chốt): số điện thoại **luôn được chuẩn hoá về E.164 với country code `+84`** trước
  khi validate và trước khi lưu. Input `"0901234567"` → `"+84901234567"`; input đã ở dạng
  `"+84..."` giữ nguyên; input có `+` với country code khác (`+1...`, `+81...`) **giữ nguyên**,
  không ép về +84. Separator người dùng gõ tay (space, `-`, `.`, `(`, `)`) bị bỏ trước khi kiểm.
  Giá trị lưu ở `pending_registration.phone_number` và `users.phone_number` là bản **đã chuẩn hoá**
  — nhờ vậy `UNIQUE` constraint bắt được `"0901234567"` và `"+84901234567"` là cùng một số.
  Không có nhánh nào từ chối số local.
  Confirmed (quyết định thiết kế): hàm chuẩn hoá sống ở **`pkg/validation`** với tên
  `NormalizePhone(raw string) string`, export cạnh `PhoneRule`/`IsValidPhone` — không đặt trong
  `usecase`. Lý do: US-002 (đăng nhập bằng SĐT + OTP) và US-003 (khôi phục mật khẩu qua SMS) sẽ
  cần đúng phép chuẩn hoá này; để trong usecase thì mỗi feature lại copy một bản, và hai bản lệch
  nhau sẽ làm `UNIQUE` constraint bắt hụt. Đây là hàm thuần (không I/O), không vi phạm dependency
  rule — `usecase` vốn đã import `pkg/validation`.
    - Success → Step 2
    - Failure (email sai định dạng) → ErrInvalidEmail → 400 INVALID_EMAIL, details {field: email}
    - Failure (phone sai định dạng, kể cả sau khi chuẩn hoá) → ErrInvalidPhone → 400 INVALID_PHONE,
      details {field: phone_number}
    - Failure (password < 8) → ErrInvalidPassword → 400 INVALID_PASSWORD, details {field: password}
    - Không chạm DB ở bất kỳ nhánh failure nào.

Step 2 — Kiểm tra trùng SĐT / email (feature này)
- Trigger: Step 1 pass
- Pre-condition: input đã chuẩn hoá
- Call: RegistrationRepository.FindConflicts(phone, email) — MỘT query quét 3 nguồn:
        users.phone_number | users.email | pending_registration.email
        (điều kiện pending có thêm `AND phone_number <> $1 AND expires_at > now()`)
  Thứ tự báo lỗi: SĐT trước, email sau (SĐT là định danh chính ở MVP) — khi cả hai cùng trùng,
  caller nhận PHONE_TAKEN.
    - Success (chưa trùng) → Step 3
    - Failure (phone đã có tài khoản) → domain.ErrPhoneTaken → 409 PHONE_TAKEN, dừng
    - Failure (email đã có tài khoản, hoặc đang là email của lượt pending ở SĐT **khác**) →
      domain.ErrEmailTaken → 409 EMAIL_TAKEN, dừng
  Confirmed (v5.1, 2026-09-22): hàng pending của **chính SĐT đang gọi** KHÔNG còn là trùng. Đó là
  lượt đăng ký dở của cùng người, và request-otp được phép ghi đè (Step 4 UPSERT theo phone_number)
  — nhờ vậy user cài lại app làm lại được ngay, không phải chờ hết `PendingRegistrationTTL`. Chỉ
  email mới còn bị hàng pending chặn, và chỉ khi hàng đó thuộc SĐT **khác**: email là unique key
  toàn hệ thống, đó là claim của người khác (AC6). Hàng hết hạn không tính.
  Confirmed: đây KHÔNG phải chốt chặn cuối. Unique constraint của DB mới là chốt chặn — check này
  chỉ để trả lỗi thân thiện sớm. Race giữa hai lượt request-otp cùng SĐT/email vẫn được xử lý ở
  Step 4 (ON CONFLICT) và ở Step 2 của flow verify (unique violation của users).

Step 3 — Hash mật khẩu (feature này)
- Trigger: Step 2 pass
- Pre-condition: password plaintext đã qua validate độ dài
- Call: bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    - Success → Step 4
    - Failure (lỗi hash) → wrapped "hash password: %w" → 500 INTERNAL_ERROR; không chạm DB
  Mật khẩu plaintext KHÔNG rời khỏi bộ nhớ usecase; không có nhánh nào lưu plaintext.

Step 4 — Áp send-limit + ghi tạm payload + cấp OTP mới (feature này, MỘT transaction)
- Trigger: Step 3 pass
- Pre-condition: password_hash đã có
- Call: RegistrationRepository.SavePendingAndIssueOTP(ctx, pending, otpHash, expiresAt, limits) → BEGIN
        (a) INSERT INTO otp_send_limits (phone_number, purpose='register') ON CONFLICT DO NOTHING
            rồi SELECT ... FOR UPDATE            -- khoá hàng bộ đếm, chống hai request song song
        (b) áp §"OTP send-limit policy": đặt lại cycle / khoá / cooldown / trần. Bị chặn →
            typed error, ROLLBACK, không ghi gì.
        (c) UPSERT INTO pending_registration (phone_number, email, password_hash, dharma_name,
            ordination_place, expires_at, created_at)
            ON CONFLICT (phone_number) DO UPDATE SET email, password_hash, dharma_name,
            ordination_place, expires_at, created_at
        (d) UPDATE otp_codes SET used_at = now()
            WHERE phone_number = $1 AND purpose = 'register' AND used_at IS NULL
        (e) INSERT INTO otp_codes (phone_number, purpose='register', otp_hash, expires_at = now() + 90s)
  Confirmed (quyết định thiết kế, ghi lại để không bị hiểu ngầm):
    - (d) + (e) là cơ chế "mỗi lần gửi tạo OTP mới và vô hiệu hoá OTP cũ" của AC5 — OTP cũ bị
      đánh `used_at`, không bị xoá, nên vẫn tra được lịch sử.
    - (a)+(b) là bước mới của v5.0: bộ đếm gửi nằm ở `otp_send_limits`, **không** carry-forward
      trên hàng OTP nữa. `request-otp` **cũng** tiêu một lượt, không còn là "restart miễn phí"
      (lần gửi đầu của một cycle vẫn miễn phí). Xem §"OTP send-limit policy" cho bảng quyết định.
    - Nếu SĐT đang trong khoá-nhập-sai 30 phút (AC12) → `*OTPLockedError` → 429 OTP_LOCKED kèm
      `locked_until`. Khoá-nhập-sai chặn cả request-otp (không có ngoại lệ), nhưng **không** có
      chuyện khoá-gửi chặn verify — hai khoá độc lập.
  Confirmed (v5.1, 2026-09-22): `pending_registration.expires_at = created_at + PendingRegistrationTTL`
  (**24h**), tách khỏi TTL của OTP (90s). Payload phải sống lâu hơn khoá-gửi 30 phút thì user mới
  "gửi lại / đăng ký lại" được sau khi khoá hết (§"Pending payload và đường làm lại"). Hệ quả chấp
  nhận có ý thức: một bản `password_hash` nằm trên server tới 24h, bù lại bằng job dọn hàng hết hạn
  (Step 6, đã hoãn). Ràng buộc "hai deadline bằng nhau" của v4.x không còn.
    - Success → COMMIT → Step 5
    - Failure (*OTPResendTooSoonError) → 429 OTP_RESEND_TOO_SOON + `retry_after_seconds`
    - Failure (*OTPSendLimitedError) → 429 OTP_SEND_LIMITED + `locked_until`
    - Failure (*OTPLockedError, khoá-nhập-sai) → 429 OTP_LOCKED + `locked_until`
    - Failure (lỗi DB khác) → wrapped "save pending registration: %w" → 500; rollback cả (a)-(e)
  Transaction boundary: (a)-(e) chia sẻ MỘT transaction. Không có trạng thái nửa vời kiểu
  "pending_registration đã ghi nhưng không có OTP nào hiệu lực", và không có chuyện "đã tiêu một
  lượt gửi nhưng OTP không được ghi".

Step 5 — Gửi OTP qua SMS (thuộc `pkg/sms`, ngoài scope feature này)
- Trigger: Step 4 COMMIT xong
- Pre-condition: otp_codes đã có hàng hiệu lực
- Call: OTPSender.SendOTP(ctx, phoneNumber, otpCode, OTPPurposeRegister) → pkg/sms.Sender.SendOTP →
        impl log-only
  Gửi SAU khi commit, không phải trước — nếu gửi trước mà tx rollback thì user nhận OTP vô hiệu.
  Confirmed (đã chốt): tầng SMS trả lỗi thì **API trả lỗi**, KHÔNG log-rồi-trả-200. Cụ thể:
  `500 SMS_SEND_FAILED`, kèm log lỗi gốc. Caller biết là hỏng thay vì tưởng đã gửi.
  Hệ quả được chấp nhận có ý thức: hàng OTP ở Step 4 **đã commit** nên vẫn còn hiệu lực; lần thử
  lại ngay sẽ bị chặn bởi cooldown 60 giây (ErrOTPResendTooSoon). Đây là đánh đổi có chủ ý — thà
  caller biết hỏng và chờ 60s, còn hơn im lặng nuốt lỗi. Ghi rõ để không bị "sửa" thành trả 200.
    - Success → 200 { message, expires_in_seconds: 90, resend_after_seconds: 60 }
    - Failure (SMS provider lỗi) → 500 SMS_SEND_FAILED; KHÔNG rollback Step 4 (đã commit)

Step 6 — Dọn dẹp định kỳ (feature này, ngoài request path)
- Trigger: tác vụ định kỳ bên ngoài (chưa có scheduler trong repo — xem tasks.md Phase 6)
- Pre-condition: —
- Call: DELETE FROM pending_registration WHERE expires_at <= now();
        DELETE FROM otp_codes WHERE expires_at <= now() AND used_at IS NOT NULL
        -- (v5.0, tuỳ chọn) DELETE FROM otp_send_limits WHERE last_sent_at < now() - send_lock_duration
    - Success → im lặng, ghi log số hàng đã xoá
  `otp_send_limits` không bắt buộc dọn: mỗi SĐT+purpose chỉ một hàng, tự đặt lại về logic khi quá
  `send_lock_duration`. requirements §Ghi chú kỹ thuật nói hàng hết hạn "dọn định kỳ cùng chỗ với
  otp_codes". Repo không
  có job scheduler nào (Cloud Run không tự chạy cron) — cần Cloud Scheduler + endpoint nội bộ, hoặc
  chấp nhận để rác tích tụ giai đoạn MVP. Ghi lại, không tự chọn.
```

## Step-by-step Flow — Đăng ký bước 2: xác thực OTP và tạo tài khoản

```
Step 1 — Validate request (feature này)
- Trigger: RegisterHandler nhận POST /v1/auth/register/verify-otp
- Pre-condition: DecodeJSONBodyStrict pass
- Call: validator trên verifyOTPRequest (phone_number required+phone, otp_code required,len=6,numeric)
    - Success → Step 2
    - Failure → 400 VALIDATION_ERROR + field errors, không chạm DB

Step 2 — Tra lượt đăng ký tạm (feature này)
- Trigger: Step 1 pass
- Pre-condition: phone_number đã chuẩn hoá giống hệt Step 1 của flow trước
- Call: RegistrationRepository.FindPendingRegistration(phone)
    - Success (còn hạn) → Step 3
    - Failure (không có hàng) → domain.ErrPendingRegistrationNotFound → 400 OTP_INVALID
      (không trả 404 — tránh cho caller dò xem SĐT nào đang có lượt đăng ký dở)
    - Failure (expires_at <= now()) → domain.ErrPendingRegistrationExpired → 400
      PENDING_REGISTRATION_EXPIRED, yêu cầu làm lại từ bước 1

Step 3 — Tra OTP đang hiệu lực (feature này)
- Trigger: Step 2 pass
- Pre-condition: pending_registration tồn tại và còn hạn
- Call: RegistrationRepository.FindActiveOTP(phone, 'register')
        WHERE phone_number = $1 AND purpose = 'register' AND used_at IS NULL
        ORDER BY created_at DESC LIMIT 1
  Không đặt unique trên (phone_number, purpose) — nhiều hàng cùng cặp vẫn hợp lệ, hàng hiệu lực
  xác định bằng used_at IS NULL (requirements §Ghi chú kỹ thuật). Lấy hàng mới nhất.
    - Success → Step 4
    - Failure (không có hàng hiệu lực) → domain.ErrOTPExpired → 400 OTP_EXPIRED
  Kiểm `locked_until > now()` ở đây: → domain.ErrOTPLocked → 429 OTP_LOCKED kèm `locked_until`.
  Trong lúc khoá, mọi request verify-otp của SĐT đó đều dừng ở đây, không so hash, không tăng attempts.

Step 4 — So khớp OTP (feature này)
- Trigger: Step 3 pass
- Pre-condition: hàng OTP hiệu lực, chưa khoá
- Call: bcrypt.CompareHashAndPassword(otp.OTPHash, []byte(otpCode))
  OTP lưu bằng **bcrypt**, không phải SHA-256. Lý do: OTP chỉ 6 chữ số (10^6 khả năng) — hash
  nhanh không salt coi như đảo ngược được trong vài giây. requirements chỉ nói "lưu dạng hash",
  bcrypt là cách thoả mãn nó cho không gian khoá nhỏ. Cùng cost với password hash.
    - Success → Step 5
    - Failure (hash lệch):
        RegistrationRepository.IncrementOTPAttempts(otpID)   -- attempts = attempts + 1
        nếu attempts + 1 >= OTPConfig.MaxAttempts (3) →
          RegistrationRepository.LockOTP(otpID, now()+OTPConfig.LockDuration)   -- 30m (v5.0)
            SET locked_until = ..., used_at = now()   -- AC12: khoá + vô hiệu hoá OTP hiện tại
        → 400 OTP_INVALID. Không trả về số lần còn lại — tránh tăng thông tin cho kẻ brute-force.
    - Failure (lỗi DB) → 500
  AC12: sau 3 lần sai, lần thử thứ 4 bị chặn ở Step 3 (locked_until). Hết 30 phút (v5.0, trước là
  15 phút), hàng OTP đã có used_at nên phải gọi lại request-otp để có OTP mới — đúng "buộc tôi chờ
  hết thời gian khoá rồi làm lại từ bước 1".

Step 5 — Tạo user + session (feature này, MỘT transaction)
- Trigger: Step 4 success
- Pre-condition: pending_registration đọc được, OTP đã khớp
- Call: RegistrationRepository.CreateUserAndSession(ctx, pending, issuer) → BEGIN
        (a) SELECT 1 FROM users WHERE email = $1              -- re-check trong tx
        (b) INSERT INTO users (email, email_verified_at, phone_number, password_hash,
            dharma_name, ordination_place)
            VALUES ($1, NULL, $2, $3, $4, $5) RETURNING id, created_at, updated_at
        (c) DELETE FROM pending_registration WHERE id = $1
        (d) UPDATE otp_codes SET used_at = now() WHERE id = $1
        (e) INSERT INTO user_sessions (...)                    -- xem Step 6
  Confirmed: (a) là re-check BẮT BUỘC, không phải trùng lặp thừa với Step 2 của flow trước. Hai
  người khác nhau có thể cùng nhập một email chưa verify: pending_registration.email KHÔNG unique
  (requirements §Ghi chú kỹ thuật, cố ý), nên cả hai đều qua được check lúc request-otp; người thứ
  hai phải bị chặn ở đây.
  Confirmed: `users.phone_number` và `users.email` giữ UNIQUE NOT NULL — đây là chốt chặn cuối cho
  race. Không dựa vào (a) để đảm bảo duy nhất.
  Confirmed: `email_verified_at` ghi NULL (AC2, AC3) — email copy sang users mà chưa có bằng chứng
  sở hữu. Không có bảng `pending_email` nào được tạo.
    - Success → Step 6
    - Failure (users_email_key, 23505) → domain.ErrEmailTaken → 409 EMAIL_TAKEN, rollback (a)-(e)
    - Failure (users_phone_number_key, 23505) → domain.ErrPhoneTaken → 409 PHONE_TAKEN, rollback
    - Failure (lỗi DB khác) → wrapped "create user: %w" → 500, rollback
  Transaction boundary: (a)-(e) chia sẻ MỘT transaction. Không có tài khoản nào được tạo mà thiếu
  session, và pending_registration không bị xoá nếu user chưa tạo xong — AC3 ("không hàng nào của
  dữ liệu đăng ký tạm được giữ lại sau khi tạo account") đạt được vì (c) nằm cùng tx với (b).

Step 6 — Sinh token và mở session (feature này, cùng tx với Step 5)
- Trigger: users row đã có id
- Pre-condition: user.ID, pending.Email đã có
- Call: closure TokenIssuer do RegisterUsecase truyền vào →
        jwtService.GenerateAccessToken(userID, email, "user", {"sid": sessionID})
        jwtService.GenerateRefreshToken(userID, nil)
      rồi INSERT INTO user_sessions (user_id, session_id, refresh_token_hash, token_version,
      is_active, is_revoked, expires_at)
  **Đã implement (v4.2 — tasks.md Phase 7):** access token mang claim `sid` = `session_id` để `login/`
  `logout` tìm đúng session. Repository sinh `sessionID := uuid.NewString()` **trước** → `issueTokens(userID,
  sessionID)` (closure của usecase gắn `extra{"sid": sessionID}` vào `GenerateAccessToken`; refresh token
  vẫn `GenerateRefreshToken(userID, nil)`) → INSERT `user_sessions` với đúng `sessionID` đó.
  `domain.TokenIssuer` = `func(userID, sessionID string) (access, refresh string, err error)`. Response
  của `verify-otp` **không đổi**.
  JWT được mint BÊN TRONG transaction, qua closure — `domain/` không import `pkg/jwt`. Cái giá: giữ
  tx mở trong lúc ký token; đổi lại không có user nào tồn tại mà không có session.
  Confirmed (quyết định thiết kế): cột lưu **`refresh_token_hash`** (SHA-256), không lưu token thô.
  Nhất quán với cách requirements mô tả OTP ("lưu dạng hash") và reset token ở US-003. Tra cứu
  refresh token về sau bằng cách hash token nhận được rồi so.
  Confirmed (quyết định thiết kế): US-001 KHÔNG có `POST /v1/auth/refresh` — ngoài scope (đã chốt).
  Refresh token vẫn được phát hành và lưu, nhưng chưa có route nào dùng nó.
  Confirmed (đã chốt): refresh token sống **7 ngày**. `user_sessions.expires_at = now() + 168h`,
  lấy từ `JWTConfig.RefreshTokenExpiration` trong `config.dev.yaml` (`refresh_token_expiration:
  "168h"`) — **không** hardcode trong usecase; usecase đọc cùng nguồn config mà `pkg/jwt` dùng để
  ký token, nên hai bên không thể lệch nhau. Giá trị "30 ngày" trong requirements §Ghi chú kỹ thuật
  đã được **sửa thành 7 ngày** ở requirements v5.2 (trước đó là follow-up trong `changelog.md`).
    - Success → COMMIT → 201 { access_token, refresh_token, user: {...} }
    - Failure (lỗi sinh token) → wrapped "issue tokens: %w" → 500; rollback cả (a)-(e)
    - Failure (lỗi insert session) → wrapped "insert session: %w" → 500; rollback
```

## Step-by-step Flow — Gửi lại OTP: `POST /v1/auth/resend-otp`

Endpoint thiết kế **dùng chung cho mọi `purpose`**, không riêng `register`. US-001 chỉ hiện thực
nhánh `purpose = "register"`; `login` / `reset_password` / `verify_email` được US-002/US-003 gọi
lại đúng endpoint này, chỉ khác giá trị `purpose` và điều kiện tiên quyết ở Step 2 (mỗi feature tự
định nghĩa "có được phép gửi lại không" cho purpose của mình).

```
Step 1 — Validate và chuẩn hoá input (feature này)
- Trigger: RegisterHandler nhận POST /v1/auth/resend-otp
- Pre-condition: DecodeJSONBodyStrict pass (field lạ bị từ chối, cap 1 MB)
- Call: validator trên resendOTPRequest (phone_number required+phone, purpose required,oneof=register
        login reset_password verify_email) → normalizePhone (CÙNG hàm `validation.NormalizePhone`
        với flow 1 — đây là lý do hàm đó phải nằm ở `pkg/validation`, không nằm ở usecase)
  Body chỉ gồm `phone_number` + `purpose`. Không có email/password: payload đăng ký đã nằm sẵn ở
  `pending_registration` từ flow 1, gửi lại chúng ở đây là thừa và tạo cơ hội ghi đè ngoài ý muốn.
    - Success → Step 2
    - Failure (thiếu field / sai định dạng) → 400 VALIDATION_ERROR + field errors, không chạm DB
    - Failure (phone sai định dạng kể cả sau chuẩn hoá) → ErrInvalidPhone → 400 INVALID_PHONE,
      details {field: phone_number}

Step 2 — Tra trạng thái lượt đăng ký tạm (feature này, chỉ nhánh purpose='register')
- Trigger: Step 1 pass
- Pre-condition: phone_number đã chuẩn hoá
- Call: RegistrationRepository.FindPendingRegistration(phone)
  Với `purpose = "register"`, gửi lại OTP **chỉ có nghĩa khi còn payload tạm để verify**. Không có
  hàng pending → không có gì để xác thực, gửi OTP mới là vô nghĩa.
    - Success (còn hạn) → Step 3
    - Failure (không có hàng pending) → domain.ErrPendingRegistrationNotFound → 400 OTP_INVALID
      (giống flow 2 Step 2 — không trả 404, tránh cho caller dò SĐT nào đang có lượt đăng ký dở)
    - Failure (`expires_at <= now()`) → domain.ErrPendingRegistrationExpired → 400
      PENDING_REGISTRATION_EXPIRED. Từ v5.1 `expires_at` là `PendingRegistrationTTL` (**24h**), không
      còn bằng OTP TTL, nên 400 này chỉ xảy ra khi lượt đăng ký đã bỏ dở quá 24h — người dùng bắt
      đầu lại bằng `request-otp` (không còn phải nhập lại chỉ sau 90 giây).
  Nhánh `purpose` khác `register` KHÔNG thuộc US-001 — US-002/US-003 tự cài điều kiện tiên quyết
  của mình ở đúng vị trí này (ví dụ: `reset_password` cần user tồn tại và email đã verify).

Step 3 — Kiểm tra khoá-nhập-sai, rồi áp send-limit (feature này)
- Trigger: Step 2 pass
- Pre-condition: pending_registration còn hạn
- Call: RegistrationRepository.FindActiveOTP(phone, purpose) → đọc `locked_until` (khoá-nhập-sai)
        của hàng OTP mới nhất
  Confirmed (v5.1, 2026-09-22): `ErrOTPInvalid` (không còn hàng OTP hiệu lực — OTP cũ đã hết hạn
  hoặc đã dùng) **không** phải lỗi ở đây. Không có hàng thì không có khoá-nhập-sai để kiểm, và việc
  cấp OTP mới chính là mục đích của resend. Trước v5.1 nhánh này trả `OTP_EXPIRED`, khiến "bấm gửi
  lại sau 30 phút" bất khả thi vì lúc đó OTP chắc chắn đã hết hạn.
    (a) `locked_until > now()` → `*OTPLockedError` → 429 OTP_LOCKED kèm `locked_until`.
        Không có ngoại lệ: đang khoá 30 phút vì sai 3 lần thì resend cũng bị chặn, nếu không thì
        AC12 vô hiệu — kẻ brute-force chỉ cần gọi resend-otp là xoá sạch bộ đếm.
  Rồi áp §"OTP send-limit policy" (cooldown 60s / trần 4 lượt / khoá-gửi 30 phút) — dùng **chung
  một hàm** với flow 1 Step 4(b), không viết lại logic hai lần:
    (b) `now() < last_sent_at + OTPConfig.SendCooldown` (60s) → `*OTPResendTooSoonError` → 429
        OTP_RESEND_TOO_SOON + `retry_after_seconds`.
    (c) `send_count >= OTPConfig.MaxSendAttempts` (3), hoặc `locked_until` của khoá-gửi > now() →
        `*OTPSendLimitedError` → 429 OTP_SEND_LIMITED + `locked_until`.
  Đọc/ghi bộ đếm thực sự nằm trong tx của Step 4 — bước này chỉ chốt thứ tự kiểm; một lượt bị chặn
  ở (b)/(c) không tiêu lượt và không ghi gì.

Step 4 — Áp send-limit + cấp OTP mới (feature này, MỘT transaction)
- Trigger: Step 3 pass
- Pre-condition: chưa khoá-nhập-sai
- Call: RegistrationRepository.ReissueOTP(ctx, phone, purpose, otpHash, expiresAt, limits) → BEGIN
        (a) INSERT INTO otp_send_limits (phone_number, purpose) ON CONFLICT DO NOTHING
            rồi SELECT ... FOR UPDATE
        (b) áp §"OTP send-limit policy" — bị chặn → typed error, ROLLBACK, không ghi gì
        (c) UPDATE otp_codes SET used_at = now()
            WHERE phone_number = $1 AND purpose = $2 AND used_at IS NULL
        (d) INSERT INTO otp_codes (phone_number, purpose, otp_hash, expires_at = now() + 90s)
  Confirmed (quyết định thiết kế):
    - (a)+(b) là bước mới của v5.0; bộ đếm gửi không còn `resend_count` carry-forward trên hàng OTP.
    - (c) giống hệt (d) của flow 1 Step 4 — dùng lại cùng một câu SQL, không viết bản thứ hai.
      OTP cũ bị đánh `used_at` chứ không xoá, nên AC5 ("OTP cũ không còn hiệu lực") vẫn thoả và
      lịch sử vẫn tra được.
    - v5.1: ReissueOTP **không** đụng `pending_registration`. Trước v5.1 nó gia hạn payload thêm
      90 giây (bước (e)); nay payload có TTL riêng 24h nên việc gia hạn là thừa, và "để yên payload"
      đúng với ý "bấm gửi lại thì không đổi thông tin đăng ký".
    - OTP mới sinh qua **cùng hàm `generateOTP()`** với flow 1 (hằng số `111111` giai đoạn này).
    - Success → COMMIT → Step 5
    - Failure (typed error send-limit) → 429 tương ứng + ROLLBACK, không ghi gì
    - Failure (lỗi DB khác) → wrapped "reissue otp: %w" → 500; rollback cả (a)-(d)
  Transaction boundary: (a)-(d) chia sẻ MỘT transaction. Không có trạng thái nửa vời kiểu
  "OTP cũ đã vô hiệu nhưng OTP mới chưa ghi" — nếu vậy user mất OTP mà không nhận được cái mới.

Step 5 — Gửi OTP qua SMS (thuộc `pkg/sms`, ngoài scope feature này)
- Trigger: Step 4 COMMIT xong
- Pre-condition: otp_codes đã có hàng hiệu lực mới
- Call: OTPSender.SendOTP(ctx, phoneNumber, otpCode, purpose) → pkg/sms.Sender → impl log-only
  Cùng contract lỗi với flow 1 Step 5 (đã chốt): SMS provider lỗi → **500 SMS_SEND_FAILED**, KHÔNG
  log-rồi-trả-200. Hệ quả chấp nhận có ý thức: `send_count` đã tăng và OTP cũ đã bị vô hiệu ở
  Step 4 (đã commit), nên lần thử lại sẽ tốn thêm một lượt trong trần gửi và phải chờ đủ 60 giây
  cooldown. Ghi rõ để không bị "sửa" thành trả 200.
    - Success → 200 { message, expires_in_seconds: 90, resend_after_seconds: 60,
      resend_remaining: <OTPConfig.MaxSendAttempts - send_count mới> }
    - Failure (SMS provider lỗi) → 500 SMS_SEND_FAILED; KHÔNG rollback Step 4 (đã commit)
```

## Deliberately stubbed for now

| Thứ | Trạng thái | Ghi chú |
|---|---|---|
| `pkg/sms` + `pkg/sms/logsms` | **Chưa tồn tại** | requirements §Ghi chú kỹ thuật yêu cầu tạo mới theo pattern interface + `Register`/`New` factory. Impl duy nhất là log-only. Provider thật (Twilio / AWS SNS / GCP tương đương) thêm sau bằng subpackage `init()` tự đăng ký, **không đổi interface**. Tasks.md Phase 5. |
| Sinh OTP | **Hằng số `111111`** | Cố định để dev/test. Tách riêng `generateOTP()` để đổi sang `crypto/rand` chỉ ở một điểm. Tasks.md Phase 2. |
| `cmd/main.go` wiring | **Chưa tồn tại** | File đã bị xoá ở commit `0475281`; phải viết lại từ đầu, gồm cả `validation.PhoneTag` registration ở composition root. Tasks.md Phase 4. |
| `migrations/` | **Ba file** | `0001_register_onboarding.sql` (`users`, `otp_codes`, `pending_registration`, `user_sessions`); `0002` thuộc `login/`; **`0003_otp_send_limits.sql` (v5.0, đã apply)** — `otp_send_limits` + drop `otp_codes.resend_count`. Các file cũ (`0002`–`0007` của bản trước commit `0475281`) không được tái tạo ở phạm vi US-001. |
| `docs/specs/api/openapi.yaml` | **Chưa tồn tại** | Thư mục `docs/specs/api/` rỗng. Contract 3 route trong spec này là nguồn duy nhất cho tới khi file được tạo. Tasks.md Phase 4. |

## Known gaps carried by this feature

- **`POST /v1/auth/refresh` ngoài scope** — refresh token được phát hành nhưng không có route nào
  tiêu thụ. Hệ quả trực tiếp: **AC11 vế thứ hai** ("refresh token tự động cấp access token mới")
  **không thoả được** ở phạm vi US-001. `pkg/jwt.Service.RefreshAccessToken` đã tồn tại sẵn nhưng
  không được wire. Ghi lại để không bị đọc nhầm là đã xong; luồng refresh cần user story riêng.
- **`user_devices` không được tạo trong US-001.** Schema cũ có bảng này (đăng ký cũ ghi một hàng
  device từ fingerprint `SHA-256(Platform|OSVersion|DeviceModel|UserAgent)`), nhưng contract 2 bước
  hiện tại **không mang trường device nào** — client chỉ gửi `phone_number`, `email`, `password`,
  `dharma_name?`, `ordination_place?`. Nên `user_sessions` ở `0001` **không có `device_id`**.
  US-002 (đăng nhập) — nay đặc tả ở `docs/specs/features/login/` — định nghĩa payload device và thêm `user_devices` + cột `user_sessions.device_id` (nullable) bằng migration `0002` riêng của nó.
  Ghi rõ ở đây vì đây là chỗ lệch có chủ ý so với schema trước.
- **Tên field trên wire là `phone_number`, không phải `phone`.** Bản changelog đã bị xoá có ghi
  một lần đổi tên `phone_number → phone` (v1.2). `requirements.md` hiện tại (v3.1) dùng
  `phone_number` xuyên suốt trong cả request lẫn response. Spec này theo
  `requirements.md` — **không** áp dụng lại lần đổi tên cũ. Cột DB vẫn là `users.phone_number`.
- **Gate `email_verified_at IS NULL` chưa chặn được gì trong US-001.** Luật gate (requirements
  §"Gate email chưa verify") áp lên US-002/US-003/US-004 và các tính năng gửi thư — tất cả đều
  ngoài scope lần này. Trong phạm vi US-001, gate chỉ hiện diện ở hai chỗ: `users.email_verified_at`
  ghi NULL (Step 5), và check trùng phải quét `pending_registration.email` (Step 2 flow 1). Không
  tự thêm logic gate nào khác vào feature này.
- **Không có tài liệu ERD.** `docs/specs/db/` rỗng — `register-erd.md` cũ đã bị xoá ở commit
  `0475281`. Schema trong spec này là nguồn duy nhất cho tới khi ERD được viết lại.
- **Auth model:** cả ba route là public theo thiết kế, nên không có gap middleware kiểu
  `/v1/users/me`. Các route user-scoped đó chưa được triển khai.
- **`ResendOTP` chỉ hiện thực nhánh `purpose = "register"`.** Endpoint và `OTPSender.SendOTP` được
  thiết kế dùng chung cho cả bốn `purpose`, nhưng Step 2 (điều kiện tiên quyết) và Step 4(e) (gia
  hạn bảng tạm) là phần riêng của từng feature. US-003 (US-002 ở `login/` không dùng `purpose = "login"` — login bằng OTP ngoài scope) sẽ bổ sung hai nhánh đó vào cùng file
  `usecase/register_usecase.go` — hoặc tách usecase riêng dùng chung repository, quyết định ở
  version đặc tả US-003. Ghi lại để không ai đọc nhầm là ba `purpose` kia đã chạy được.
- **v5.0 đã có code (Phase 9, 2026-09-22).** `otp_send_limits`, `OTP_SEND_LIMITED`,
  `OTPSendLimitedError`, `ReleaseSendLimit` và việc bỏ `ErrOTPResendExhausted` đã được implement và
  verify: migration `0003`, repository test chạy Postgres thật, E2E `register_e2e_test.go`
  (`TestRegisterFlow_SendLimitAndVerifyDuringLock_E2E`) pass. Hành vi cũ v4.5 không còn trong code.
- **Bộ đếm gửi là per-`purpose`.** v5.0 tạo hàng `otp_send_limits` cho `register`; `login` /
  `reset_password` sau này tự quyết định hạn mức cho purpose của mình (không dùng chung với
  `register`).
