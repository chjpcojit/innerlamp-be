# Changelog — otp-mechanism Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version first.

## v1.2 (2026-09-25)

Đưa spec `otp-mechanism` vào đúng quy ước `il-spec-writer` hiện hành. **Không đổi behavior, không đổi
AC, không đổi code, không tick/untick task** — chỉ chuẩn hoá hình thức: thêm `flow.md` (bản vẽ Mermaid
bắt buộc theo rule 5/6 khi `design.md` có Step-by-step Flow — spec cũ viết trước khi skill cập nhật
quy ước nên thiếu file), đồng bộ version 4 file.

- `flow.md` (mới): hai sequence diagram mirror 1:1 hai section "Step-by-step Flow" của `design.md` —
  Issue OTP (3 bước: Generate → `IssueOTP` một transaction → Send) và Verify OTP + completion (4 bước:
  `FindActiveOTP` → kiểm khoá → so mã → `CreateUserAndSession`). Transaction (`IssueOTP`,
  `CreateUserAndSession`) được bọc bằng `rect` + `Note`; bước gửi SMS ghi `Note` "thuộc pkg/sms, ngoài
  scope cơ chế này"; `OTPRepository` và `RegistrationRepository` vẽ thành hai participant riêng kèm
  ghi chú là cùng một concrete `registrationRepository`. Không có `⚠️ Not confirmed` nào cần vẽ.
- `requirements.md`: no change needed — chỉ bump version (AC đã ở dạng EARS từ v1.0).
- `design.md`: thay ASCII "Request flow" bằng con trỏ `> Xem flow.md ...` (theo `flow_diagram.md` —
  không giữ hai bản vẽ cùng một thứ để tránh drift). Các section "Step-by-step Flow" giữ nguyên nội
  dung, chỉ bump version.
- `tasks.md`: no change needed — không thêm Phase (`flow.md` là bản vẽ sinh từ `design.md`, không phải
  việc implement); chỉ bump version.
- `changelog.md`: sửa header "three files" → "four files" (thêm `flow.md`).
- Reason: skill `il-spec-writer` bắt buộc `flow.md` khi `design.md` có Step-by-step Flow; spec
  `otp-mechanism` viết trước bản cập nhật nên thiếu `flow.md`.

### Cross-file follow-ups

- **Không có follow-up nào.** Lần này chỉ chuẩn hoá hình thức spec, **không đổi behavior/contract/schema**:
  `docs/specs/api/openapi.yaml` (không đổi contract), `migrations/` (không đổi schema),
  `docs/knowledge/features/register.md` (OKF concept — không cần `bdd-okf-docs` refresh vì hành vi
  không đổi) đều giữ nguyên. Đã kiểm, không bỏ sót.

## v1.1 (implemented) (2026-09-23)

- `requirements.md`: Status → complete; mục "Pending (not yet implemented)" đổi thành "Implemented"
  ghi nhận AC1–AC9 đã được test phủ. Nội dung AC không đổi — đây là spec refactor, không đổi
  wire-contract hay schema.
- `design.md`: Status → complete; bảng Layers & Step-by-step Flow giữ nguyên (nay mô tả trạng thái
  đã implement, không phải target).
- `tasks.md`: toàn bộ task `[x]` (Phase 1 → 6); Status → complete.
- Reason: implement refactor tách cơ chế OTP dùng chung theo đúng spec v1.0 — `domain.OTPRepository`
  + `usecase.OTPService` tách khỏi `RegisterUsecase`/`RegistrationRepository`; signup giữ nguyên
  hành vi ngoài (route/schema/response không đổi).

### Cross-file follow-ups

- `docs/knowledge/features/register.md` cần `il-okf-writer` refresh: mô tả `RegisterUsecase` (nay
  gọi `OTPService`, constructor nhận `*OTPService` + `pendingRegistrationTTL` thay vì `cfg.OTP`) và
  `RegistrationRepository` (nay còn 5 method; 5 method OTP đã chuyển sang `OTPRepository`) đã đổi.

## v1.0 (initial) (2026-09-23)

- `requirements.md`: initial — viết retroactively từ cơ chế OTP hiện có của US-001 (`register`) và
  prospectively cho phần refactor tách cơ chế dùng chung. Phạm vi: chỉ refactor cơ chế OTP + chỉnh
  nơi đang dùng (signup); không thêm feature mới, không đổi route/schema.
- `design.md`: initial — định nghĩa tách `domain.OTPRepository` + `usecase.OTPService` khỏi
  `RegisterUsecase`/`RegistrationRepository`; Step-by-step Flow cho Issue OTP và Verify OTP; quyết
  định giữ `SavePendingAndIssueOTP` atomic.
- `tasks.md`: initial — 6 Phase (Domain, Usecase, Repository, Wiring, Verification, Docs), toàn bộ
  `[ ]`.
- Reason: người dùng yêu cầu spec cho việc "tách riêng cơ chế OTP và chỉnh sửa ở các nơi đang sử
  dụng" sau khi chốt thiết kế ở các lượt thảo luận trước (gộp ở tầng cơ chế, không gộp endpoint,
  signup giữ nguyên).
