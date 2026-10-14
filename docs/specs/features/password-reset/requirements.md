# Requirements — password-reset

> Superseded by [`../login/requirements.md`](../login/requirements.md) US-003 (v2.0, mã qua email) — kept for history until the old-docs cleanup; do not update.

Version: 1.1
Last updated: 2026-09-25
Status: prospective — chưa có code; đây là work order để build (US-003, chuyển từ onboarding)
Module: `identity`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này **chuyển từ** `docs/specs/features/onboarding/requirements.md` (US-003 cũ, mức user story) sang folder riêng theo đúng precedent `login/`, và được **viết lại** theo một quyết định quan trọng: kênh khôi phục là **SMS OTP** (`otp_codes.purpose = 'reset_password'`), **không phải email**.

Lý do bỏ email ở bản này: luật *"Gate email chưa verify"* của US-001 (AC13/AC14 + bảng "Ảnh hưởng lên các US") nói *"Gửi link reset qua email **chỉ** khi email đã verify"*, mà `users.email_verified_at` hiện **luôn NULL** (luồng xác thực email chưa tồn tại). Kênh email thuần túy vì thế là dead-on-arrival — không ai dùng được cho tới khi build luồng verify email trước. SĐT đã là định danh chính đã verify OTP ở US-001, nên SMS OTP là đường khôi phục hoạt động ngay.

---

## US-003: Quên mật khẩu (đặt lại bằng SMS OTP)

**Module:** `identity`
**Priority:** P1 (không chặn MVP đầu tiên, nhưng cần sớm — rủi ro UX nếu thiếu)

## User Story

> Là một **người dùng đã có tài khoản nhưng quên mật khẩu**,
> Tôi muốn **đặt lại mật khẩu bằng mã OTP gửi qua SMS tới số điện thoại đã đăng ký**,
> Để **lấy lại quyền truy cập tài khoản mà không cần tạo tài khoản mới**.

## Acceptance Criteria (EARS notation)

Cả hai endpoint đều **public** (người gọi đang mất quyền truy cập, không có token hợp lệ).

**Request OTP — `POST /v1/auth/password-reset/request-otp`**

1. WHEN a request arrives with a `phone_number` that has a `users` row, THE SYSTEM SHALL issue a fresh reset OTP (`purpose = 'reset_password'`) through the shared OTP mechanism, send it via SMS, and return `200` with a generic message — the same message a non-existent number gets (AC2), so the response never reveals which numbers are registered.
2. WHEN a request arrives with a `phone_number` that has **no** `users` row (including one that exists only in `pending_registration`), THE SYSTEM SHALL return the **same** `200` generic message, SHALL **NOT** send an SMS, but SHALL **still** apply the per-phone send policy (cooldown / send cap / send lock) so a `429` also cannot reveal whether the number is registered.
3. WHEN a request arrives with a `phone_number` that is not a valid phone number even after normalisation, THE SYSTEM SHALL reject it with `400 INVALID_PHONE` and SHALL NOT touch the database or the send-limit counter. (A malformed request is not a send attempt.)
4. WHEN a request arrives for a number whose reset OTP is wrong-guess-locked (`otp_codes.locked_until` in force), THE SYSTEM SHALL return `429 OTP_LOCKED` with `locked_until`, without issuing a fresh code. (Only a registered number that already requested a reset can be in this state — see Ghi chú kỹ thuật.)
5. WHEN a request exceeds the send policy — a send within the cooldown, or a send beyond the per-phone cap — THE SYSTEM SHALL return `429 OTP_RESEND_TOO_SOON` / `429 OTP_SEND_LIMITED`, uniformly for registered and unregistered numbers alike (AC2).
6. WHEN the SMS provider fails after the OTP row was persisted, THE SYSTEM SHALL return `500 SMS_SEND_FAILED` — same contract as US-001 (a code that never arrives is not a success).

**Confirm — `POST /v1/auth/password-reset/confirm`**

7. WHEN a request arrives with a correct, active, unexpired, unused reset OTP and a valid `new_password` (≥ 8 bytes), THE SYSTEM SHALL update the account's `password_hash`, mark the OTP used, and revoke **every** session of that user (`revoked_reason = 'password_reset'`) — all inside one transaction — and return `200` `{"success": true}`. No tokens are issued: the user must log in again everywhere.
8. WHEN a request arrives with a wrong OTP, THE SYSTEM SHALL return `400 OTP_INVALID` and record the attempt; the 3rd consecutive miss locks the code, after which any further attempt returns `429 OTP_LOCKED` (same policy as US-001 AC12).
9. WHEN a request arrives with an OTP that is expired, already used, or was never issued for this number, THE SYSTEM SHALL return `400 OTP_EXPIRED` — one code for all three states, so the caller cannot probe which one occurred.
10. WHEN a request arrives with a `new_password` shorter than 8 bytes, THE SYSTEM SHALL return `400 INVALID_PASSWORD`; with a malformed `phone_number`, `400 INVALID_PHONE` — both before any database access.

## API contract (dự kiến — chưa có trong openapi.yaml)

> Hợp đồng dưới đây sẽ được đưa vào `docs/specs/api/openapi.yaml` **sau khi** handler chạy thật qua `curl` (đúng convention repo). Cột mã lỗi là cam kết thiết kế.

| Method | Path | Auth | Request | Success | Failure codes |
|---|---|---|---|---|---|
| POST | `/v1/auth/password-reset/request-otp` | public | `{ phone_number }` | `200` `{ message }` | `400 INVALID_PHONE`, `429 OTP_LOCKED`/`OTP_RESEND_TOO_SOON`/`OTP_SEND_LIMITED`, `500 SMS_SEND_FAILED`/`INTERNAL_ERROR` |
| POST | `/v1/auth/password-reset/confirm` | public | `{ phone_number, otp_code, new_password }` | `200` `{"success": true}` | `400 INVALID_PHONE`/`INVALID_PASSWORD`/`OTP_INVALID`/`OTP_EXPIRED`, `429 OTP_LOCKED`, `500 INTERNAL_ERROR` |

- Response `request-otp` **cố định, không có** `expires_in_seconds` / `resend_after_seconds`: vì nhánh "số chưa đăng ký" không cấp code thật nên báo số giây sẽ nói dối; client học cooldown qua `retry_after_seconds` của `429 OTP_RESEND_TOO_SOON`. Message gợi ý: "Nếu số điện thoại đã đăng ký, mã xác thực đã được gửi."
- `confirm` trả `200` `{"success": true}` — cùng shape với `logout`, không cấp token (AC7).

## Out of scope (this feature boundary, not tracked here)

- **Đặt lại qua email** — bị chặn bởi gate email chưa verify; làm khi có luồng xác thực email (đặt `email_verified_at`). `pkg/mailer` (đã có sẵn, interface + `LogMailer`) **không được wire** ở feature này, giữ lại cho feature email sau.
- **Đặt lại qua cả hai kênh** (email khi đã verify, SMS khi chưa) — giai đoạn sau.
- **2FA khi đặt lại** — giai đoạn sau.
- **Chống bot bằng `RequireAttestation`** — chốt MVP chỉ dùng `otp_send_limits`; attestation gate là quyết định nghiệp vụ của một feature sau.
- **`resend-otp` dùng chung với `purpose=reset_password`** — không cần: request-otp của reset là **re-callable** (không có pending payload để giữ/ghi đè), mỗi lần gọi lại chính là một lần resend, chịu send policy. Endpoint `POST /v1/auth/resend-otp` giữ nguyên chỉ `register`.
- **TTL riêng cho OTP reset** (dài hơn 90s) — dùng chung `OTPConfig` ở MVP; mở rộng per-purpose sau nếu cần.
- **Cấm dùng lại mật khẩu cũ** — chỉ kiểm `≥ 8` ở MVP.

## Ghi chú kỹ thuật liên quan

- **Kênh khôi phục = SMS OTP, `purpose = 'reset_password'`** (chốt qua phỏng vấn). Tái dùng toàn bộ cơ chế OTP đã refactor ở `otp-mechanism` (`OTPService.Generate`/`Issue`/`Verify`, bảng `otp_codes` + `otp_send_limits` scope theo `(phone_number, purpose)`). **Không cần migration**: `chk_otp_codes_purpose` và `chk_otp_send_limits_purpose` đã liệt kê `reset_password` từ `0001`/`0003`; `users.password_hash` và `user_sessions.revoked_reason` đều đã có. **Không cần bảng `password_reset_tokens`** như spec cũ đề xuất — hàng `otp_codes` chính là trạng thái giữa hai bước (không có payload phụ để lưu như `pending_registration` của register).
- **Enumeration (AC2):** request-otp trả một response duy nhất cho cả "có tài khoản" lẫn "không có". Hai thứ phải đồng bộ, không chỉ body: (a) **không gửi SMS** cho số chưa đăng ký (tiết kiệm + chống SMS bombing), nhưng (b) **vẫn áp send-limit** cho số chưa đăng ký — nếu không, số lạ không bao giờ nhận `429 OTP_SEND_LIMITED` trong khi số thật nhận sau 4 lượt, và chính cái `429` tiết lộ trạng thái. Vì vậy cần thêm một primitive **`ApplySendLimit`** trên `OTPRepository` (ghi send-limit mà không cấp code) — xem "Phụ thuộc: mở rộng `otp-mechanism`".
- **Không lộ thời gian (nhẹ):** nhánh "có tài khoản" làm `Issue` (bcrypt hash + insert + send), nhánh "không có" làm `ApplySendLimit` (một UPSERT). Số round-trip DB gần bằng nhau; chi phí chi phối là DB, không phải bcrypt (khác login cần `dummyPasswordHash`). Chấp nhận ở MVP; ghi ở Known gaps.
- **Wrong-guess lock (AC4) là registered-only, chấp nhận có ý thức:** khoá nhập sai sống trên hàng `otp_codes`, mà hàng đó chỉ được cấp cho số có tài khoản (AC2 không cấp code cho số lạ). Nên `429 OTP_LOCKED` chỉ xuất hiện với số đã đăng ký — nhưng để chạm tới nó, kẻ tấn công đã phải khiến hệ thống cấp OTP cho số đó trước, tức là số đó đã đăng ký và vừa request reset. Rò rỉ tối thiểu, đúng cùng đánh đổi đã chấp nhận ở register (`OTP_LOCKED` tiết lộ số đang mid-registration).
- **Revoke tất cả session (AC7):** cần một method mới nằm trong transaction `ResetPassword` — `UPDATE user_sessions SET is_revoked = true, revoked_reason = 'password_reset' WHERE user_id = $1 AND is_revoked = false`. Không tách method riêng `RevokeAllSessionsForUser` (YAGNI — chỉ reset dùng). Refresh token của các session đó chết ngay (vì `FindSessionByRefreshHash` kiểm `is_revoked`); **access token vẫn sống tới hết hạn ≤ 15 phút** (stateless) — cùng Known gap đã chốt ở `login/`.
- **Thêm hằng `RevokeReasonPasswordReset = "password_reset"`** vào nhóm các hằng revoke reason (hiện ở `domain/login.go`), để `revoked_reason` đọc được ý nghĩa, không phải chuỗi tự do.
- **Không cần sentinel mới, không đổi `delivery/http/errors.go`:** mọi lỗi usecase trả ra đều là sentinel đã có mapping — `ErrInvalidPhone`, `ErrInvalidPassword`, `ErrOTPInvalid`, `ErrOTPExpired`, `*OTPLockedError`, `*OTPResendTooSoonError`, `*OTPSendLimitedError`, `ErrSMSSendFailed`. `ErrUserNotFound` là nội bộ (request-otp nuốt nó; confirm map nó sang `ErrOTPInvalid` khi user bị xoá giữa request và confirm).
- **Hash mật khẩu mới ở usecase** (`bcrypt.GenerateFromPassword`, `bcrypt.DefaultCost`) trước khi gọi repository — repository chỉ nhận `passwordHash`, không bao giờ thấy plaintext. `minPasswordLength = 8` là hằng package-level trong `usecase` (đã có từ `register_usecase.go`), tái dùng trực tiếp.
- **TTL/policy dùng chung `OTPConfig`** (90s / cooldown 60s / cap 4 lượt / khoá gửi 30 phút / khoá nhập sai 3 lần 30 phút) — chốt chấp nhận 90s hơi ngắn cho reset ở MVP.
- **Dependency rule:** `delivery → usecase → domain`; `repository → domain`; `usecase` import `OTPService` (cùng package) + `pkg/validation` + `domain`; không import repository/delivery. `PasswordResetUsecase` là type mới, dựng ở composition root.

## Open questions (resolved during interview — 2026-09-25)

| Câu hỏi | Trả lời | Ảnh hưởng |
|---|---|---|
| Kênh khôi phục (email vs SMS vs cả hai) | **SMS OTP** (`purpose=reset_password`) | Không cần verify email; tái dùng OTP mechanism |
| Số chưa đăng ký: gửi SMS hay không | **Không gửi, vẫn áp send-limit** | Thêm primitive `ApplySendLimit`; chống enumeration qua 429 |
| Chính sách mật khẩu mới | **Chỉ ≥ 8 ký tự** | Không so với mật khẩu cũ ở MVP |
| Phạm vi revoke sau reset | **Revoke TẤT CẢ session** | `ResetPassword` revoke mọi session của user trong 1 tx |
| Chống bot (attestation?) | **Không, chỉ send-limit** | Không wire `RequireAttestation` ở MVP |
| TTL OTP cho reset | **Dùng chung `OTPConfig`** | 90s; không đổi schema/config |
| Tên folder spec | **`password-reset`** | Khớp `purpose` + tên method |
