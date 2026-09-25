# Tasks — password-reset

Version: 1.1
Last updated: 2026-09-25
Status: prospective — toàn bộ item `[ ]`; chưa implement
Module: `identity`

> See `changelog.md` in this folder for the version history of this spec.

<!--
Mọi item dưới đây trace về một dòng trong bảng "Layers & components" của design.md hoặc một step
trong "Step-by-step Flow" của nó. Không có task mồ côi.

Item `[x]` = đã verify (code có thật + test tương ứng pass). Viết spec KHÔNG phải là implement,
nên toàn bộ item ở version này là `[ ]`.

Thứ tự build đi từ trong ra ngoài theo `.claude/skills/golang-clean-arch-tdd`:
domain → usecase → repository → delivery, test viết cùng lúc với code.
-->

## Phase 0 — Điều kiện tiên quyết

- [ ] Mở rộng `otp-mechanism` (additive): thêm `domain.OTPRepository.ApplySendLimit(ctx, phoneNumber,
  purpose, limits SendLimits) (*SendState, error)` — ghi send-limit mà **không** cấp code, tái dùng SQL
  helper `applySendLimits`; thêm `OTPService.ApplySendLimit(ctx, phoneNumber, purpose) error` bọc repo +
  `wrapSendError`. Thêm unit test cho cả hai (mock repo). Ghi một mục nhỏ ở
  `docs/specs/features/otp-mechanism/changelog.md` (bổ sung, không bump major). **Không bắt đầu Phase 2
  của spec này trước mục này.**
- [ ] `go test ./services/appService/usecase/...` vẫn xanh sau khi thêm method (không regression register).

## Phase 1 — Domain

- [ ] `domain/password_reset.go` — input `RequestPasswordResetOTPInput{PhoneNumber}`,
  `ConfirmPasswordResetInput{PhoneNumber, OTPCode, NewPassword}` (thuần dữ liệu, không JSON tag).
- [ ] `domain/password_reset.go` — interface `PasswordResetRepository`: `FindUserByPhone(ctx, phone)
  (*User, error)` (không hàng → `ErrUserNotFound`) + `ResetPassword(ctx, phone, passwordHash, otpID)
  error` (một tx). Doc comment ghi rõ `ResetPassword` phải revoke **mọi** session của user.
- [ ] `domain/login.go` — thêm hằng `RevokeReasonPasswordReset = "password_reset"` vào nhóm revoke
  reason (cạnh `RevokeReasonLogout`/`RevokeReasonRefreshReuse`/`RevokeReasonSupersededByLogin`).
- [ ] `domain/mock_password_reset_repository.go` — mock tay cho `PasswordResetRepository`
  (`var _ PasswordResetRepository = (*MockPasswordResetRepository)(nil)`).

## Phase 2 — Usecase

- [ ] `usecase/password_reset_usecase.go` — `PasswordResetUsecase` + constructor nhận
  `PasswordResetRepository`, `*OTPService`, `logging.Logger`.
- [ ] `RequestPasswordResetOTP` Step 1–2 — `validation.NormalizePhone`; rỗng → `ErrInvalidPhone`;
  `FindUserByPhone`; `ErrUserNotFound` → nhánh "không có" (không trả lỗi — AC2).
- [ ] `RequestPasswordResetOTP` Step 3 (có tài khoản) — `OTPService.EnsureNotLocked(phone,
  reset_password)` → `OTPService.Issue(phone, reset_password)`; pass-through typed error (AC4, AC5, AC6).
- [ ] `RequestPasswordResetOTP` Step 4 (không có) — `OTPService.ApplySendLimit(phone, reset_password)`;
  pass-through typed error (AC2, AC5). Cả hai nhánh trả về `nil` result (handler viết message cố định).
- [ ] `ConfirmPasswordReset` Step 1 — `NormalizePhone` + `len(new_password) >= minPasswordLength`;
  sai → `ErrInvalidPhone` / `ErrInvalidPassword` (AC10).
- [ ] `ConfirmPasswordReset` Step 2 — `OTPService.Verify(phone, reset_password, otp_code)`; pass-through
  `ErrOTPInvalid`/`ErrOTPExpired`/`*OTPLockedError` (AC8, AC9).
- [ ] `ConfirmPasswordReset` Step 3–4 — `bcrypt.GenerateFromPassword(new_password, DefaultCost)` →
  `ResetPassword(phone, passwordHash, otp.ID)`; map `ErrUserNotFound` từ `ResetPassword` sang
  `ErrOTPInvalid` (user bị xoá giữa request và confirm, không lộ).
- [ ] Unit tests `usecase/password_reset_usecase_test.go` — table-driven, mock tay; phủ AC1–AC10: có tài
  khoản gọi `Issue` (không `ApplySendLimit`), không có gọi `ApplySendLimit` (không `Issue`/`Send`),
  SĐT/password sai không chạm repo, mọi typed error pass-through, `ResetPassword` nhận đúng
  `passwordHash` (không phải plaintext) + `otpID`.

## Phase 3 — Repository

- [ ] `repository/postgres_password_reset_repository.go` — constructor `NewPasswordResetRepository(db)`;
  `FindUserByPhone` (cột tường minh, tái dùng `userColumns`; không hàng → `ErrUserNotFound`).
- [ ] `ResetPassword` — MỘT tx: `UPDATE users ... RETURNING id` (0 hàng → `ErrUserNotFound`) → mark
  `otp_codes.used_at` → `UPDATE user_sessions ... revoked_reason = 'password_reset' WHERE user_id AND
  is_revoked = false`. Explicit column list, `ctx` propagation, `%w` wrapping.
- [ ] `ApplySendLimit` (ở concrete của `OTPRepository`, thuộc `postgres_registration_repository.go` hoặc
  file OTP riêng) — tái dùng `applySendLimits`; ghi `otp_send_limits`, **không** ghi `otp_codes`.
- [ ] Repository tests `repository/postgres_password_reset_repository_test.go` — trên Postgres thật
  (testcontainers): `ResetPassword` đổi hash + mark OTP used + revoke mọi session (gồm device-less của
  US-001) trong một tx; rollback khi lỗi; `ApplySendLimit` không tạo hàng `otp_codes`; `FindUserByPhone`
  trả `ErrUserNotFound` cho số chỉ có `pending_registration`.

## Phase 4 — Delivery

- [ ] `delivery/http/password_reset_handler.go` — DTO `requestPasswordResetOTPRequest{phone_number}`
  (`validate:"required,phone"`), `confirmPasswordResetRequest{phone_number, otp_code, new_password}`
  (`otp_code` `len=6,numeric`; `new_password` `required`); interface usecase do handler định nghĩa.
- [ ] Handler `RequestPasswordResetOTP` — decode strict → validate → usecase → `200`
  `{message: "..."}` cố định (AC1, AC2); map lỗi qua `writeDomainError`.
- [ ] Handler `ConfirmPasswordReset` — decode strict → validate → usecase → `200` `{"success": true}`
  (AC7); map lỗi qua `writeDomainError`.
- [ ] Handler tests `delivery/http/password_reset_handler_test.go` + mock usecase — success, từng nhánh
  lỗi, body hỏng/field lạ; assert `request-otp` body **không** có `expires_in_seconds`/`resend_after_seconds`.
- [ ] `cmd/main.go` — `NewPasswordResetRepository(db)` → `NewPasswordResetUsecase(repo, otpService,
  logger)` → `NewPasswordResetHandler(...)`; `mux.HandleFunc` hai route (cả hai **public**, không bọc
  middleware). Kiểm bằng đọc `main.go` rằng hai route thực sự được đăng ký.

## Phase 5 — Integration & documentation

- [ ] Chạy end-to-end qua Postgres thật (Docker throwaway) + `curl`: register (US-001) → request-otp
  reset với SĐT đúng (200 generic) → confirm OTP đúng + password mới (200) → login bằng password mới
  (200) → login bằng password cũ (401) → refresh bằng token cũ (401, session bị revoke) → request-otp
  với SĐT chưa đăng ký (200 generic, không SMS) → lặp request-otp trong cooldown (429
  OTP_RESEND_TOO_SOON) cho cả số đã/đã không đăng ký (đồng bộ).
- [ ] `docs/specs/api/openapi.yaml` — thêm 2 path **sau khi** handler chạy thật, mọi status/code/field
  khớp response thật.
- [ ] `go build ./...`, `go vet ./...`, `go test -short ./...` sạch; `scripts/check-architecture.sh` +
  `.claude/skills/golang-clean-arch-tdd/scripts/check_dependency_rule.sh services/appService` sạch.
- [ ] Skill `golang-e2e-verification`: audit mỗi AC có test chứng minh + black-box test binary thật
  (harness `services/appService/e2e/` + `password_reset_e2e_test.go`), đặc biệt wiring hai route.
- [ ] Ghi vào `changelog.md` concept OKF cần `il-okf-writer` tạo dưới `docs/knowledge/features/`
  (password-reset) **sau khi** flow đã chạy thật; `status` giữ `draft` tới khi đó.

## Deferred (separate feature, not blocking password-reset)

- Đặt lại mật khẩu qua email — cần US xác thực email (đặt `email_verified_at`) trước; `pkg/mailer` đã có sẵn.
- Chống bot bằng `RequireAttestation` cho các endpoint nhạy cảm (chờ quyết định gate nghiệp vụ).
- Rate limit theo IP cho `request-otp`/`confirm` (`pkg/ratelimit/middleware`).
- TTL per-purpose cho OTP reset (dài hơn 90s).
- Cấm dùng lại mật khẩu cũ.
- Dọn định kỳ `otp_codes` / `otp_send_limits` (chung với register, chờ scheduler).
