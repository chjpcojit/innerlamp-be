# Requirements — otp-mechanism

Version: 2.1
Last updated: 2026-10-16
Status: implemented (v2.1) — cơ chế OTP khoá theo email, gửi qua `AuthMailer`, mã `crypto/rand` (mã cố định chỉ ở dev), đếm lượt nguyên tử bằng `ConsumeOTPAttempt`.
Module: `otp`
Code: `services/appService/`
Product spec: none (cơ chế nội bộ, không có product spec riêng) — phục vụ `docs/product/onboarding/product.md` @ v1.2 và `docs/product/login/product.md` @ v1.2
Contract: không có route riêng; route dùng cơ chế này nằm ở `onboarding/` và `login/`
Implements: BR-ONB-09, BR-LGN-05 (phần mã), NFR-ONB-03
Architecture: ADR-0003 rule 6 (Accepted 2026-10-14)

> See `changelog.md` in this folder for the version history of this spec.

Spec của tầng dùng chung (`usecase.OTPService` + `domain.OTPRepository`). Hai consumer:
`register` (onboarding) và `reset_password` (login US-003). Purpose `login` và `verify_email` vẫn có
trong CHECK của bảng nhưng không ai dùng (product login Appendix B mục 5).

## User Story

> Là một **kỹ sư backend InnerLamp**,
> Tôi muốn **một cơ chế OTP dùng chung theo email, không biết gì về nghiệp vụ đăng ký hay đặt lại mật khẩu**,
> Để **cả hai luồng dùng cùng một chính sách mã mà không nhân bản logic và không còn phụ thuộc SMS**.

## Acceptance Criteria (EARS notation)

1. WHEN the mechanism issues, resends, verifies or limits a code, THE SYSTEM SHALL scope the `otp_codes` row, the `otp_send_limits` counter and the wrong-guess lock by the pair `(email, purpose)`, with the email already normalised by the caller. (BR-ONB-09)
2. WHEN a code is generated and `otp.dev_fixed_code` is empty, THE SYSTEM SHALL draw 6 digits uniformly from `crypto/rand` (leading zeros allowed) and store only the bcrypt hash. (NFR-ONB-03)
3. WHEN `otp.dev_fixed_code` is set and `app.environment = dev`, THE SYSTEM SHALL use that code; WHEN it is set in any other environment, THE SYSTEM SHALL refuse to start (config validation). (ADR-0003 rule 6)
4. WHEN a code is issued, THE SYSTEM SHALL, in one transaction, apply the send policy (cooldown `otp.send_cooldown` 60s; `otp.max_send_attempts` 3 counted resends after the free first send; on reaching the cap, send and then lock sending for `otp.send_lock_duration` 30m), invalidate the active code of that pair, and insert the new code expiring after `otp.ttl` (600s). (BR-ONB-09)
5. WHEN the send policy refuses, THE SYSTEM SHALL return `OTPResendTooSoonError{RetryAfter}` / `OTPSendLimitedError{LockedUntil}` and write nothing. (BR-ONB-09)
6. WHEN a caller must look like it sent a code without sending one (reset for an email with no account), THE SYSTEM SHALL offer `ApplySendLimit(email, purpose)` that applies exactly the send policy of AC4 and AC5 without creating a code. (NFR-LGN-01)
7. WHEN a code is verified, THE SYSTEM SHALL find the active code, reserve one attempt atomically (`UPDATE … SET attempts = attempts + 1 WHERE id = $1 AND used_at IS NULL AND attempts < max_attempts RETURNING attempts`), setting the lock in the same statement when the reservation reaches `otp.max_attempts` (3), and only then compare the hash; a failed reservation means locked. (BR-ONB-09; pentest M-3)
8. WHEN the code is wrong, THE SYSTEM SHALL return `ErrOTPInvalid` without saying how many attempts remain; WHEN no active code exists or it is expired or used, `ErrOTPExpired`; WHEN locked, `OTPLockedError{LockedUntil}`. (BR-ONB-09)
9. WHEN the code is right, THE SYSTEM SHALL return the code row and leave marking it used to the caller's own transaction (account creation, password change). (platform)
10. WHEN the send of an issued code fails, THE SYSTEM SHALL let the caller release the send slot (`ReleaseSendLimit`) so the failed send does not count. (platform)
11. THE SYSTEM SHALL NOT, inside this mechanism, check whether an account exists, read or write `pending_registration`, create users or sessions, or change passwords. (platform)

## API contract (confirmed surface for this feature)

Không có route riêng. Các route dùng cơ chế: `onboarding/requirements.md` (ba route đăng ký) và
`login/requirements.md` (hai route đặt lại mật khẩu).

## Out of scope (this feature boundary, not tracked here)

- Đăng nhập bằng mã một lần (`purpose = login`) — product không có.
- Trần gửi chung cho mọi purpose của một email — chưa cần; giới hạn theo IP và ngân sách thư mỗi ngày nằm ở `login/` AC31–AC35.
- Nội dung thư — `AuthMailer` (onboarding design).

## Pending (not yet implemented)

AC1–AC11 ở dạng v2.0 chưa có code — `tasks.md`.

## Ghi chú kỹ thuật liên quan

- **Gửi thư không nằm trong cơ chế này nữa.** v1.2 có `OTPService.Send` gọi `OTPSender` (SMS). v2.0 bỏ
  `Send`: caller tự gọi `AuthMailer` vì nội dung thư khác nhau theo nghiệp vụ (mã đăng ký, thư báo
  "email đã có tài khoản", mã đặt lại). `OTPService` chỉ trả mã thô cho caller.
- **Giữ lượt trước khi so (AC7)** thay cho "đọc → so → tăng": hai request đồng thời không thể cùng dùng
  lượt cuối.
- **Mã `crypto/rand`:** `rand.Int(rand.Reader, big.NewInt(1_000_000))`, định dạng `%06d`.
- **Thời hạn 10 phút** là cấu hình (`otp.ttl`), theo anh Đức 2026-10-14; response của các route trả số
  giây để client hiển thị.
- **Migration `0037`** (onboarding design) đổi cột `phone_number` → `email` của `otp_codes` và
  `otp_send_limits`, không backfill.

## Open questions (resolved during implementation)

- ~~Kênh gửi?~~ → Email, qua `AuthMailer` của caller (ADR-0003). Resolved 2026-10-14.
- ~~Mã cố định `111111`?~~ → Chỉ khi `otp.dev_fixed_code` ở môi trường dev, chặn khởi động nơi khác (ADR-0003 rule 6). Resolved 2026-10-14.
- Các quyết định v1.x (tách tầng dùng chung, register giữ transaction riêng cho pending + mã) giữ nguyên — xem `changelog.md`.
