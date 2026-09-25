# Requirements — otp-mechanism

Version: 1.2
Last updated: 2026-09-25
Status: complete — refactor tách cơ chế OTP dùng chung đã implement; AC1–AC9 được test phủ
Module: `otp`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này viết **retroactively** từ cơ chế OTP hiện có của US-001 (`register`) — phần hiện trạng mô tả
đúng code thật — và **prospectively** cho phần refactor tách cơ chế dùng chung (chưa implement). Xem
`references/modes.md` ("Mixed case").

## Bằng chứng (hiện trạng trước refactor)

| Nguồn | Hiện trạng | Hệ quả quan sát được |
|---|---|---|
| `domain/register.go` | Interface `RegistrationRepository` trộn lẫn 5 method generic OTP (`ReissueOTP`, `FindActiveOTP`, `IncrementOTPAttempts`, `LockOTP`, `ReleaseSendLimit` — đều nhận `purpose`) với 5 method riêng register (`FindConflicts`, `SavePendingAndIssueOTP`, `FindPendingRegistration`, `CreateUserAndSession`, `DeletePendingRegistration`) | Feature OTP mới buộc phải import nguyên interface register; không inject được riêng tầng OTP |
| `usecase/register_usecase.go` | Cơ chế OTP nằm rải trong `RegisterUsecase`: `generateOTP()` (hằng `111111`), `sendOTP()` (hardcode `OTPPurposeRegister`), `recordFailedAttempt()`, `sendLimits()`, `isLocked()`, `wrapSendError()`, và toàn bộ logic verify | Feature mới muốn cấp/verify OTP buộc nhân bản các hàm này |
| `repository/postgres_registration_repository.go` | Helper SQL đã generic (`applySendLimits`, `invalidateActiveOTP`, `insertOTP`, `scanOTPRow`, `updateSendLimit` đều nhận `purpose`), nhưng `SavePendingAndIssueOTP` hardcode `otpPurposeRegister` | Tầng persistence đã gần generic; chỉ thiếu tách interface |
| `domain/otp.go` | `OTPPurpose` + `OTPCode` đã khai đủ 4 purpose; bảng `otp_codes` CHECK đã liệt kê `register/login/reset_password/verify_email` | Không cần migration nào cho refactor này |

## User Story

> Là một **kỹ sư backend InnerLamp**,
> Tôi muốn **cơ chế OTP (sinh mã, hash, gửi, verify, đếm nhập sai, khoá, giới hạn gửi) nằm trong một
> tầng dùng chung không biết gì về nghiệp vụ register**,
> Để **tái dùng cho các luồng OTP sau này (`reset_password`, login bằng OTP, `verify_email`) mà không
> nhân bản logic — trong khi luồng signup hiện tại giữ nguyên hành vi bên ngoài**.

## Acceptance Criteria (EARS notation)

1. WHEN the refactor is complete, THE SYSTEM SHALL provide a purpose-agnostic OTP mechanism
   (persistence + generation/hash/send/verify) whose API carries **no** feature-specific logic: it
   SHALL NOT check whether a user exists, SHALL NOT read or write `pending_registration`, SHALL NOT
   create a user or a session, and SHALL NOT update a password.

2. WHEN the mechanism issues, resends, or verifies an OTP, THE SYSTEM SHALL scope every operation —
   the `otp_codes` row, the `otp_send_limits` counter, and the wrong-guess lock — by the pair
   `(phone_number, purpose)`, so two active purposes on the same phone number never share a row, a
   send budget, or a lockout.

3. WHEN a `POST /v1/auth/register/request-otp` request arrives, THE SYSTEM SHALL behave exactly as
   before the refactor: same validation, same `409`/`429`/`500` error codes, same `200` body
   (`message`, `expires_in_seconds`, `resend_after_seconds`), and the same persisted effects (one
   `pending_registration` UPSERT + one fresh `otp_codes` row, inside a single transaction).

4. WHEN a `POST /v1/auth/resend-otp` request arrives with `purpose = register`, THE SYSTEM SHALL
   reissue a fresh code, invalidate the old one, and return `200` with `resend_remaining` exactly as
   before — while still rejecting any purpose other than `register` with `ErrUnsupportedOTPPurpose`
   mapped to `400` as today.

5. WHEN a `POST /v1/auth/register/verify-otp` request arrives with a correct code, THE SYSTEM SHALL
   still create the account and its first session and return `201` with tokens + user; the verify
   mechanism itself (find active code, compare hash, count attempts, lock on the 3rd miss) now runs
   inside the shared mechanism, but the account/session creation remains register's own completion
   step.

6. WHEN a `POST /v1/auth/register/verify-otp` request arrives with a wrong code, THE SYSTEM SHALL
   record the attempt and, on the 3rd consecutive miss, lock the code for the configured duration —
   returning `400 OTP_INVALID` / `429 OTP_LOCKED` exactly as today.

7. WHEN the register flow persists a pending payload together with its OTP, THE SYSTEM SHALL keep
   both writes inside one transaction — the refactor SHALL NOT split them into two round-trips that
   could leave a pending row with no OTP (or the reverse).

8. WHEN `generateOTP()` moves into the shared mechanism, THE SYSTEM SHALL keep returning the dev
   constant `111111` — switching to `crypto/rand` is a separate, out-of-scope change, not bundled
   into this refactor.

9. THE SYSTEM SHALL NOT add, remove, or rename any HTTP route, and SHALL NOT change
   `docs/specs/api/openapi.yaml` or any migration — this is a pure Go refactor with no schema change
   and no wire-contract change.

## API contract (confirmed surface for this feature)

Không thêm endpoint nào. Ba route hiện có giữ nguyên contract:

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| POST | `/v1/auth/register/request-otp` | public | 200 + `{message, expires_in_seconds, resend_after_seconds}` | 400 `INVALID_EMAIL`/`INVALID_PHONE`/`INVALID_PASSWORD`, 409 `PHONE_TAKEN`/`EMAIL_TAKEN`, 429 `OTP_LOCKED`/`OTP_RESEND_TOO_SOON`/`OTP_SEND_LIMITED`, 500 `SMS_SEND_FAILED` |
| POST | `/v1/auth/resend-otp` | public | 200 + `{message, expires_in_seconds, resend_after_seconds, resend_remaining}` | 400 `OTP_INVALID`, 429 `OTP_LOCKED`/`OTP_RESEND_TOO_SOON`/`OTP_SEND_LIMITED` |
| POST | `/v1/auth/register/verify-otp` | public | 201 + `{access_token, refresh_token, user}` | 400 `OTP_INVALID`/`OTP_EXPIRED`, 409 `EMAIL_TAKEN`/`PHONE_TAKEN`, 429 `OTP_LOCKED` |

## Out of scope (this feature boundary, not tracked here)

- Feature quên mật khẩu (`password-reset` / US-003) và login bằng OTP — đây là **consumer tương lai**
  của cơ chế này, có spec riêng. Refactor này không cấp endpoint hay usecase mới cho chúng, không
  dựng `purpose = reset_password`/`login` chạy thật.
- Đổi `generateOTP()` từ hằng `111111` sang `crypto/rand` — thay đổi hành vi riêng, làm sau.
- Biến nội dung SMS theo `purpose` (`smsOTPSender.SendOTP` hiện bỏ qua purpose trong message text) —
  làm khi có feature thứ hai thực sự dùng OTP.
- Gộp giới hạn gửi từ per-`purpose` sang trần toàn cục per-`phone` (chống spam dồn nhiều purpose vào
  một số) — đang là quyết định mở ở tầng vận hành, không thuộc refactor này.

## Implemented (hoàn tất refactor)

- Toàn bộ refactor đã implement — xem `tasks.md` (Phase 1 → 6, toàn bộ `[x]`). AC1–AC9 được phủ
  bởi `usecase/otp_service_test.go` (AC1, AC2, AC8), regression `usecase/register_usecase_test.go`
  (AC3–AC6), `repository/postgres_registration_repository_test.go` (AC7), và E2E
  `services/appService/e2e/register_e2e_test.go` (AC3, AC9).

## Ghi chú kỹ thuật liên quan

- **Điểm khó duy nhất của refactor là atomicity của `request-otp`.** Hiện `SavePendingAndIssueOTP`
  gộp "UPSERT `pending_registration` + invalidate OTP cũ + insert OTP mới" trong **một transaction**
  (tránh "pending có mà OTP không" nếu crash giữa hai lần gọi). Vì vậy tầng cơ chế dùng chung không
  được ép register đi qua một `IssueOTP` riêng lẻ; thay vào đó register giữ `SavePendingAndIssueOTP`
  riêng, và nó **tái dùng chung các SQL helper** (`applySendLimits`/`invalidateActiveOTP`/`insertOTP`)
  với `OTPRepository.IssueOTP`. Đây chính là "chia sẻ ở tầng primitive, không chia sẻ ở public
  method duy nhất".
- **`ResendOTP` thì hoàn toàn generic** — resend không đụng `pending_registration`, nên nó đi qua
  `OTPService.Issue(phone, register)` giống như luồng gửi đầu của mọi feature tương lai.
- **Verify trả `*domain.OTPCode` khi thành công** (không chỉ `error`), vì register cần `otp.ID` để
  `CreateUserAndSession` đánh `used_at`. Trả con trỏ row đã verify thay vì ép caller re-fetch.
- **Naming**: `ReissueOTP` đổi thành `IssueOTP` khi chuyển sang `OTPRepository` — "issue a fresh
  code" mô tả đúng thao tác chung hơn "reissue" (vốn ngụ ý đã có lần trước).
- **Cross-spec follow-up**: sau khi refactor xong, `docs/knowledge/features/register.md` cần được
  `il-okf-writer` refresh (mô tả `RegisterUsecase`/`RegistrationRepository` sẽ thay đổi) — ghi lại ở
  `tasks.md` Phase 6, không sửa trực tiếp ở đây.

## Open questions (resolved during implementation)

- ~~Tên feature / phạm vi spec~~ → `otp-mechanism`; chỉ refactor cơ chế OTP + chỉnh các nơi đang dùng
  (signup), không thêm feature mới. Resolved 2026-09-23.
- ~~Kênh reset trong scope~~ → không có; refactor này không đụng `password-reset`. Resolved
  2026-09-23.
- ~~Mode (Prospective/Retroactive)~~ → Mixed: hiện trạng OTP của signup mô tả retroactive, phần
  refactor là prospective. Resolved 2026-09-23.
