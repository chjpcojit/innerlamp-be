# Tasks — otp-mechanism

Version: 1.2
Last updated: 2026-09-25
Status: complete — toàn bộ task `[x]`; refactor đã implement và kiểm thử xanh
Module: `otp`

> See `changelog.md` in this folder for the version history of this spec.

Mỗi item dưới đây đều trace về một dòng trong bảng "Layers & components" của `design.md` hoặc một
bước trong Step-by-step Flow. Không có task nào tự phát ngoài `design.md`.

## Phase 1 — Domain

- [x] Tạo `services/appService/domain/otp_repository.go` — interface `OTPRepository` generic
      (`IssueOTP`, `FindActiveOTP`, `IncrementOTPAttempts`, `LockOTP`, `ReleaseSendLimit`), doc
      comment ghi rõ "không biết gì về pending_registration/users"
- [x] Thu hẹp `RegistrationRepository` trong `domain/register.go` còn 5 method register-only
      (`FindConflicts`, `SavePendingAndIssueOTP`, `FindPendingRegistration`, `CreateUserAndSession`,
      `DeletePendingRegistration`); xoá 5 method generic đã chuyển sang `OTPRepository`
- [x] Tạo `services/appService/domain/mock_otp_repository.go` (mock `OTPRepository`)
- [x] Sửa `domain/mock_registration_repository.go` — bỏ 5 method generic đã chuyển đi

## Phase 2 — Usecase

- [x] Tạo `services/appService/usecase/otp_service.go` — `OTPService` với `Generate`, `Send`,
      `Issue`, `Verify`; chuyển `generateOTP`, `sendOTP`, `recordFailedAttempt`, `sendLimits`,
      `isLocked`, `wrapSendError` từ `register_usecase.go` sang đây; constructor
      `NewOTPService(repo domain.OTPRepository, sender domain.OTPSender, logger logging.Logger,
      policy config.OTPConfig)`
- [x] Sửa `usecase/register_usecase.go`: `RequestOTP`/`ResendOTP`/`VerifyOTP` gọi `OTPService`;
      bỏ field `otpPolicy`; bỏ các hàm đã chuyển đi; giữ nguyên sentinel map và `tokenIssuer`
- [x] Unit test `usecase/otp_service_test.go` — table-driven, dùng `mock_otp_repository` +
      `MockOTPSender`, không DB thật (AC1, AC2, AC8)
- [x] Regression: `usecase/register_usecase_test.go` vẫn xanh với `RegisterUsecase` mới (AC3–AC6)

## Phase 3 — Repository

- [x] `repository/postgres_registration_repository.go`: concrete struct implement cả
      `domain.RegistrationRepository` lẫn `domain.OTPRepository`; đổi tên `ReissueOTP` → `IssueOTP`;
      thêm constructor `NewOTPRepository(db) domain.OTPRepository`
- [x] Giữ `SavePendingAndIssueOTP` atomic (pending + OTP một tx), tái dùng `applySendLimits`/
      `invalidateActiveOTP`/`insertOTP` (AC7)
- [x] Regression: `repository/postgres_registration_repository_test.go` xanh (testcontainers)

## Phase 4 — Wiring

- [x] `cmd/main.go`: dựng `otpService := usecase.NewOTPService(repository.NewOTPRepository(db),
      otpSender, logger, cfg.OTP)`; inject vào `NewRegisterUsecase`; bỏ tham số `cfg.OTP` của
      register usecase; không đổi route (AC9)

## Phase 5 — Verification

- [x] `go test -short ./...` và `go vet ./...` xanh
- [x] E2E `services/appService/e2e/register_e2e_test.go` vẫn pass (binary thật + testcontainers) —
      khẳng định wiring mới không vỡ luồng signup thật

## Phase 6 — Documentation & cross-spec

- [x] Ghi chú vào changelog: OKF concept `docs/knowledge/features/register.md` cần `il-okf-writer`
      refresh sau khi refactor xong (mô tả `RegisterUsecase`/`RegistrationRepository` đã đổi)
- [x] Xác nhận `docs/specs/api/openapi.yaml` và `migrations/` không đổi (AC9)

## Deferred (separate feature, not blocking otp-mechanism)

- `password-reset` (US-003) / login bằng OTP — consumer tương lai của `OTPService`, spec riêng
- Đổi `generateOTP` sang `crypto/rand`
- SMS copy theo `purpose`; trần gửi toàn cục per-phone
