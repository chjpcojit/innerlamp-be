# Tasks — otp-mechanism

Version: 2.1
Last updated: 2026-10-16
Status: v2.1 — Phase 1–3 xong (cùng đợt onboarding v8.1). Task v1.x đã xong, xem `changelog.md`.
Module: `otp`

> See `changelog.md` in this folder for the version history of this spec.

Làm sau `onboarding/tasks.md` Phase 2 (migration `0037`).

## Phase 1 — Domain + repository

- [x] `domain/otp.go`, `otp_repository.go`, `otp_send_limit.go`: khoá `email`; `ApplySendLimit`; `ConsumeOTPAttempt` thay `IncrementOTPAttempts` + `LockOTP`; xoá `OTPSender` + `mock_otp_sender.go`; cập nhật `mock_otp_repository.go`.
- [x] Repository: SQL theo email; `ConsumeOTPAttempt` (design Step V-3); `ApplySendLimit`.
- [x] Test repository trên Postgres thật: 10 lần verify sai đồng thời chỉ giữ được 3 lượt; `ApplySendLimit` cho cùng `429` như `IssueOTP`.

## Phase 2 — Usecase + config

- [x] `OTPService.Generate`: `crypto/rand` 6 chữ số; mã dev khi có `DevFixedCode`.
- [x] Bỏ `Send`/`wrapSendError`; thêm `ApplySendLimit`; `Verify` theo Step V-1…V-4.
- [x] Config: `otp.ttl: 600s`, `otp.dev_fixed_code`, kiểm AC3 trong `Validate()` (test: dev được, `staging`/`prod` bị chặn).
- [x] Test usecase AC1–AC11 (mã có số 0 đầu, phân phối đủ 6 chữ số, không lộ số lượt còn lại).

## Phase 3 — Theo sau

- [x] `services/appService/domain/README.md`, `usecase/README.md`: bỏ nhắc SMS/`OTPSender`.

## Deferred

- [ ] Job dọn `otp_codes` hết hạn (chung với onboarding Deferred).
