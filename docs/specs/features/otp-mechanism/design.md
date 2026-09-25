# Design — otp-mechanism

Version: 1.2
Last updated: 2026-09-25
Status: complete — bảng Layers mô tả target sau refactor, nay đã implement; hiện trạng trước refactor xem §"Hiện trạng". (v1.2: thêm `flow.md` bản vẽ Mermaid mirror 1:1 hai Step-by-step Flow, đổi ASCII "Request flow" thành con trỏ `flow.md` — không đổi code/AC/behavior.)
Module: `otp`
Feature folder: `services/appService/` + `docs/specs/features/otp-mechanism/`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Cơ chế OTP của US-001 hiện nằm trộn trong `RegisterUsecase` + `RegistrationRepository`. Refactor này
tách thành hai tầng dùng chung, không biết gì về nghiệp vụ register:

- `domain.OTPRepository` — persistence thuần OTP, mọi method nhận `purpose`.
- `usecase.OTPService` — cơ chế sinh/hash/gửi/verify/đếm/khoá, gói `OTPRepository` + `OTPSender` +
  `config.OTPConfig`.

`RegisterUsecase` thu hẹp lại chỉ còn nghiệp vụ register (validate, conflict-check, pending payload,
tạo user + session) và gọi `OTPService` cho phần OTP. Hợp đồng HTTP không đổi, schema không đổi.

## Hiện trạng (retroactive)

| File | Symbol | Vai trò hiện tại |
|---|---|---|
| `domain/register.go` | `RegistrationRepository` | 10 method, trong đó 5 generic OTP + 5 register-only |
| `domain/register.go` | `OTPSender` | `SendOTP(ctx, phone, code, purpose)` — đã generic, giữ nguyên |
| `domain/otp.go` | `OTPPurpose`, `OTPCode` | Đã purpose-aware, giữ nguyên |
| `usecase/register_usecase.go` | `RegisterUsecase` | Giữ cả nghiệp vụ register lẫn cơ chế OTP |
| `repository/postgres_registration_repository.go` | `registrationRepository` + helpers | Helper SQL đã generic; `SavePendingAndIssueOTP` hardcode register |

## Layers & components (target)

| Layer | File | Thay đổi |
|---|---|---|
| Domain / Entity | `services/appService/domain/otp.go` | Giữ nguyên `OTPPurpose` + `OTPCode`. Không đổi. |
| Domain / Repository interface | `services/appService/domain/otp_repository.go` (mới) | Interface generic `OTPRepository`: `IssueOTP`, `FindActiveOTP`, `IncrementOTPAttempts`, `LockOTP`, `ReleaseSendLimit`. Không có method nào đụng `pending_registration`/`users`. |
| Domain / Repository interface | `services/appService/domain/register.go` | Thu hẹp `RegistrationRepository` còn 5 method register-only: `FindConflicts`, `SavePendingAndIssueOTP`, `FindPendingRegistration`, `CreateUserAndSession`, `DeletePendingRegistration`. Bỏ 5 method generic đã chuyển sang `OTPRepository`. `OTPSender` giữ nguyên. |
| Domain / Mock | `services/appService/domain/mock_otp_repository.go` (mới) | Mock cho `OTPRepository` (phục vụ test `OTPService`). |
| Domain / Mock | `services/appService/domain/mock_registration_repository.go` | Bỏ các method generic đã chuyển đi; giữ 5 method register-only. |
| Usecase | `services/appService/usecase/otp_service.go` (mới) | `OTPService` với `Generate`, `Send`, `Issue`, `Verify`. Chuyển `generateOTP`, `sendOTP`, `recordFailedAttempt`, `sendLimits`, `isLocked`, `wrapSendError` vào đây. |
| Usecase | `services/appService/usecase/register_usecase.go` | Thu gọn: `RequestOTP` = validate + conflict + `Generate` + `SavePendingAndIssueOTP` + `Send`; `ResendOTP` = `Issue(register)`; `VerifyOTP` = `Verify(register, code)` + `CreateUserAndSession`. Bỏ field `otpPolicy` (chuyển sang `OTPService`). |
| Repository | `services/appService/repository/postgres_registration_repository.go` | Concrete `registrationRepository` implement **cả hai** interface; thêm constructor `NewOTPRepository(db)`. Đổi tên `ReissueOTP` → `IssueOTP`. `SavePendingAndIssueOTP` giữ nguyên atomic, tái dùng helper. |
| Delivery / Handler | `services/appService/delivery/http/register_handler.go` | Không đổi (chỉ gọi usecase; interface `registerUsecase` của handler không đổi). |
| Wiring | `services/appService/cmd/main.go` | Dựng `OTPService` (từ `NewOTPRepository(db)` + `otpSender` + `cfg.OTP`) rồi inject vào `NewRegisterUsecase`; bỏ tham số `cfg.OTP` của register usecase. |
| Migration | — | Không có (schema đã purpose-aware). |

Dependency rule: `delivery → usecase → domain`; `repository → domain`. `OTPService` (usecase) chỉ
import `domain` + `pkg/logging` + `config` (cho `OTPConfig`); không import repository/delivery.
`RegisterUsecase` import `OTPService` (cùng package `usecase`) — không vi phạm chiều phụ thuộc.

## Request flow

> Xem `flow.md` cho sơ đồ sequence Mermaid (mirror 1:1 của Step-by-step Flow bên dưới).

Ba route hiện có giữ nguyên contract (xem `requirements.md` §"API contract"):

- `POST /v1/auth/register/request-otp` — `RegisterUsecase.RequestOTP`: validate + `FindConflicts` →
  `OTPService.Generate` → `SavePendingAndIssueOTP` (atomic pending+OTP) → `OTPService.Send`. Luồng này
  là nghiệp vụ register (đụng `pending_registration`), không thuộc hai flow cơ chế dùng chung ở
  `flow.md`; chi tiết đầy đủ nằm ở spec `onboarding` (`../onboarding/design.md`).
- `POST /v1/auth/resend-otp` — `RegisterUsecase.ResendOTP` → `OTPService.Issue` (Generate → IssueOTP → Send).
- `POST /v1/auth/register/verify-otp` — `RegisterUsecase.VerifyOTP` → `OTPService.Verify` → `CreateUserAndSession`.

## Step-by-step Flow — Issue OTP (cơ chế dùng chung)

Áp dụng cho `resend-otp` (register) hôm nay, và cho luồng gửi đầu của mọi feature OTP tương lai.
Không đụng `pending_registration`.

```
Step 1 — Sinh mã + hash (cơ chế này)
- Trigger: usecase gọi OTPService.Issue(phone, purpose)
- Pre-condition: phone đã chuẩn hoá; purpose hợp lệ (do caller quyết định, không phải việc của Issue)
- Call: OTPService.Generate() → generateOTP() (hằng 111111) → bcrypt hash
    - Success → Step 2
    - Failure (lỗi hash) → wrapped error → 500, không ghi gì

Step 2 — Áp send-limit + ghi OTP (cơ chế này, MỘT transaction)
- Trigger: Step 1 pass
- Pre-condition: otpHash + expiresAt (now + OTPConfig.TTL) đã có
- Call: OTPRepository.IssueOTP(phone, purpose, otpHash, expiresAt, sendLimits) → BEGIN
        (a) applySendLimits(phone, purpose) — cooldown 60s / trần 4 lượt / khoá-gửi 30 phút
        (b) invalidateActiveOTP(phone, purpose)
        (c) insertOTP(phone, purpose, otpHash, expiresAt)
    - Success → COMMIT → Step 3
    - Failure (send-limit) → *OTPResendTooSoonError / *OTPSendLimitedError → 429, ROLLBACK, không ghi
    - Failure (DB khác) → wrapped → 500, ROLLBACK
  Transaction boundary: (a)-(c) chung MỘT transaction — không có "OTP cũ mất mà mới chưa ghi".

Step 3 — Gửi SMS (thuộc OTPService.Send, qua pkg/sms)
- Trigger: Step 2 COMMIT xong
- Pre-condition: otp_codes đã có hàng hiệu lực mới
- Call: OTPSender.SendOTP(ctx, phone, code, purpose) → pkg/sms.Sender
    - Success → return (code đã gửi)
    - Failure (provider lỗi) → ErrSMSSendFailed → 500; KHÔNG rollback Step 2 (đã commit)
```

## Step-by-step Flow — Verify OTP (cơ chế dùng chung) + completion của register

```
Step 1 — Tra OTP đang hiệu lực (cơ chế này)
- Trigger: usecase gọi OTPService.Verify(phone, purpose, code)
- Pre-condition: phone đã chuẩn hoá
- Call: OTPRepository.FindActiveOTP(phone, purpose)
    - Success (có hàng active/locked) → Step 2
    - Failure (không có) → ErrOTPExpired (nếu caller là verify) → 400, dừng

Step 2 — Kiểm tra khoá nhập sai (cơ chế này)
- Trigger: Step 1 pass
- Call: đọc otp.LockedUntil
    - Failure (locked_until > now) → *OTPLockedError → 429, không ghi gì
    - Success → Step 3

Step 3 — So mã (cơ chế này)
- Trigger: Step 2 pass
- Call: bcrypt.CompareHashAndPassword(otp.OTPHash, code)
    - Success → trả *OTPCode (caller tự completion)
    - Failure (sai) → IncrementOTPAttempts(otp.ID); nếu >= MaxAttempts → LockOTP(otp.ID, until);
      trả ErrOTPInvalid → 400/429

Step 4 — Completion (thuộc register, ngoài cơ chế)
- Trigger: Verify trả *OTPCode thành công
- Pre-condition: pending payload còn hạn
- Call: RegistrationRepository.CreateUserAndSession(pending, otp.ID, sessionTTL, issueTokens)
        — tạo users, xoá pending, đánh used_at cho OTP, mint token, ghi user_sessions (MỘT tx)
    - Success → 201 { access_token, refresh_token, user }
    - Failure (email/phone race) → 409; (DB khác) → 500
```

## Quyết định

- **Tách hai interface thay vì một interface nhúng.** `OTPRepository` (generic) và
  `RegistrationRepository` (register-only) là hai interface độc lập; concrete `registrationRepository`
  implement cả hai. Lý do: feature OTP tương lai chỉ cần inject `OTPRepository`, không kéo theo
  `FindConflicts`/`CreateUserAndSession`; đúng idiom Go "consumer-defined, small interfaces".
- **`SavePendingAndIssueOTP` ở lại `RegistrationRepository`** (không ép qua `IssueOTP` public) để giữ
  atomic "pending + OTP một transaction". Nó tái dùng chung helper SQL với `IssueOTP` — chia sẻ ở
  tầng primitive, không ở public method.
- **`OTPService` là type exported, dựng ở composition root.** Theo đúng pattern DI hiện tại
  (`NewRegisterUsecase` được dựng trong `main.go`); feature tương lai cũng inject chung một instance.
- **`Verify` trả `(*domain.OTPCode, error)`.** Thành công trả row đã verify (caller cần `otp.ID` để
  completion); thất bại trả sentinel (`ErrOTPInvalid`/`ErrOTPExpired`/`ErrOTPLocked`).
- **`generateOTP()` giữ hằng `111111`.** Refactor chỉ di chuyển hàm, không đổi hành vi; `crypto/rand`
  là việc riêng (AC8).
- **Đổi tên `ReissueOTP` → `IssueOTP`** khi vào `OTPRepository`: "issue" mô tả đúng thao tác chung;
  resend chỉ là một lần issue lại, không cần tên riêng.

## Test plan

| AC | Test | Kiểu |
|---|---|---|
| AC1, AC2 | `usecase/otp_service_test.go`: `Issue`/`Verify` chỉ gọi `OTPRepository` (mock) + `OTPSender` (mock), không đụng `pending_registration`; hai purpose cùng phone đi hai row/limit độc lập | unit (mock) |
| AC3–AC6 | Regression: `usecase/register_usecase_test.go`, `delivery/http/register_handler_test.go`, `repository/postgres_registration_repository_test.go` giữ xanh — assert không đổi response/error | regression |
| AC7 | `repository` test: `SavePendingAndIssueOTP` vẫn một tx (pending + OTP cùng commit/rollback) | unit (testcontainers) |
| AC8 | `otp_service_test.go`: `Generate` trả `111111` | unit |
| AC9 | grep: không route mới trong `cmd/main.go`; không file migration mới; `openapi.yaml` không đổi | review |
| — | `go test -short ./...` + `go vet ./...` xanh | CI |

## Deliberately stubbed for now

- Không có — refactor không tạo component stub mới; `OTPService` được implement đầy đủ trong chính
  thay đổi này (vì nó chỉ là "di chuyển code có sẵn").

## Known gaps carried by this feature

- `OTPService.Issue` hôm nay chỉ có **một** caller thật là `ResendOTP` (register). Luồng gửi đầu của
  feature OTP tương lai sẽ gọi cùng method đó; cho tới lúc đó `Issue` không phải dead code nhưng cũng
  chưa chứng minh được trên một purpose khác `register`. Ghi rõ để không đọc nhầm là "đã có feature
  thứ hai dùng OTP".
- `smsOTPSender.SendOTP` vẫn bỏ qua `purpose` trong nội dung tin nhắn (chỉ có register chạy thật) —
  xem Out of scope ở `requirements.md`.
