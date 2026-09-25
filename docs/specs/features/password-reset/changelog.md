# Changelog — password-reset Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version first.

## v1.1 (2026-09-25)

Đưa spec vào đúng quy ước `il-spec-writer` hiện hành: thêm `flow.md` (bản vẽ Mermaid **bắt buộc**
theo rule 5/6 khi `design.md` có Step-by-step Flow — spec v1.0 viết trước khi skill cập nhật quy
ước này nên thiếu file). Không đổi behavior, không đổi AC, không đổi task, không đổi code.

- `flow.md` (mới): hai sequence diagram mirror 1:1 hai section "Step-by-step Flow" của `design.md`
  — `Request OTP` (4 bước) và `Confirm` (4 bước). Mỗi Step xuất hiện dưới dạng message/alt/rect;
  transaction của `ResetPassword` (Step 4 confirm) được bọc bằng `rect` + `Note`; bước gọi
  `OTPService` ghi `Note` "thuộc otp-mechanism, ngoài scope feature này" (không vẽ nội bộ SQL của
  nó). Không có `⚠️ Not confirmed` nào cần vẽ (mọi câu hỏi đã chốt ở v1.0).
- `design.md`: thay ASCII "Request flow" bằng con trỏ `> Xem flow.md ...` (theo `flow_diagram.md` —
  không giữ hai bản vẽ cùng một thứ để tránh drift). Các section "Step-by-step Flow" giữ nguyên nội
  dung, chỉ bump version.
- `requirements.md`: no change needed — chỉ bump version cho đồng bộ (rule "cùng một version cho cả
  4 file").
- `tasks.md`: no change needed — chỉ bump version. Không thêm Phase: `flow.md` là bản vẽ sinh từ
  `design.md`, không phải việc implement.
- `changelog.md`: sửa header "three files" → "four files" (thêm `flow.md`).
- Reason: skill `il-spec-writer` được cập nhật để bắt buộc `flow.md` khi có Step-by-step Flow; spec
  `password-reset` viết trước bản cập nhật nên thiếu file này.

## v1.0 (initial — 2026-09-25)

Spec đầu tiên của folder `password-reset/`, viết ở chế độ **prospective** (chưa có code). US-003 được
**chuyển từ** `docs/specs/features/onboarding/requirements.md` (mức user story) và viết lại đủ bộ ba
file; folder `onboarding/` bump lên v6.0 để bỏ US-003 (xem `../onboarding/changelog.md`).

- `requirements.md`: US-003 viết lại từ "đặt lại qua email" thành **SMS OTP** (`purpose =
  'reset_password'`) — vì `users.email_verified_at` luôn NULL (luồng verify email chưa tồn tại) nên
  email-based là dead-on-arrival. 10 AC theo EARS cho hai endpoint
  `POST /v1/auth/password-reset/request-otp` + `POST /v1/auth/password-reset/confirm`. Thêm ràng buộc
  enumeration: response request-otp là một message cố định; **không gửi SMS** cho số chưa đăng ký
  nhưng **vẫn áp send-limit** để 429 không tiết lộ trạng thái.
- `design.md`: 2 Step-by-step Flow (request-otp 4 bước, confirm 4 bước); bảng Layers; mục "Phụ thuộc:
  mở rộng `otp-mechanism`" (`ApplySendLimit`); không có migration (CHECK đã gồm `reset_password`).
- `tasks.md`: 5 Phase + Phase 0 (điều kiện tiên quyết), mọi item `[ ]`.
- Reason: yêu cầu đặc tả US-003 và phỏng vấn edge case; US-003 cũ lệch với hiện trạng (email gate).

### Quyết định từ phỏng vấn (2026-09-25)

| Câu hỏi | Trả lời | Ảnh hưởng |
|---|---|---|
| Kênh khôi phục | SMS OTP | Tái dùng OTP mechanism; không cần verify email |
| Số chưa đăng ký | Không gửi SMS, vẫn áp send-limit | Thêm primitive `ApplySendLimit` |
| Mật khẩu mới | Chỉ ≥ 8 ký tự | Không so mật khẩu cũ ở MVP |
| Revoke sau reset | Tất cả session | `ResetPassword` revoke mọi session, 1 tx |
| Chống bot | Không attestation, chỉ send-limit | Không wire `RequireAttestation` |
| TTL OTP reset | Dùng chung `OTPConfig` | 90s; không đổi schema/config |
| Tên folder | `password-reset` | Khớp `purpose` + method |

### Phát hiện khi đọc code (không phải câu hỏi cho user)

- `pkg/mailer` (interface + `LogMailer` + `MockMailer`) đã được tạo từ trước cho US-003 email-based,
  nhưng **chưa được import ở đâu** — feature này dùng SMS nên `pkg/mailer` vẫn là package mồ côi, giữ
  lại cho feature email sau (không xoá, không wire).
- `otp_codes` và `otp_send_limits` đều đã có `CHECK ... purpose IN (...,'reset_password',...)` từ
  `0001`/`0003` → **không cần migration**. Đây là hệ quả của quyết định "khai đủ bốn purpose ngay từ
  đầu" ở onboarding v3.0.

### Cross-file follow-ups

- `docs/specs/features/onboarding/` — bump v6.0: xoá section US-003, thay bằng con trỏ sang folder này.
- `docs/specs/features/otp-mechanism/` — thêm mục nhỏ ghi `ApplySendLimit` (bổ sung additive).
- `docs/specs/api/openapi.yaml` — thêm 2 path **sau khi** handler chạy thật.
- `docs/knowledge/features/` — concept password-reset cần `il-okf-writer` tạo **sau khi** code implement
  và verify; `status: draft`.
