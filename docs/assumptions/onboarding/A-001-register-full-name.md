# A-001 — Họ tên (`full_name`) khi đăng ký

Ngày: 2026-10-11 · Phạm vi: US-001 (register), `migrations/0033_users_full_name.sql`

Yêu cầu gốc chỉ nói "đăng ký đang thiếu fullname". Các điểm sau do em tự quyết để không chặn việc, cần xác nhận khi review:

| # | Giả định | Lý do | Nếu sai |
|---|----------|-------|---------|
| 1 | `full_name` **bắt buộc** ở `request-otp` (thiếu → `VALIDATION_ERROR`, rỗng/toàn khoảng trắng hoặc quá dài → `400 INVALID_FULL_NAME`) | "đang thiếu" ngụ ý là trường chính của tài khoản, khác `dharma_name` tuỳ chọn | Bỏ `validate:"required"` + nhánh blank trong `normalizeFullName` |
| 2 | Cột `users.full_name`/`pending_registration.full_name` **nullable**, không backfill | Tài khoản đã có không có tên; không bịa dữ liệu. Luật "bắt buộc" nằm ở use case | Thêm migration `NOT NULL` sau khi có chiến lược cập nhật tên cho user cũ |
| 3 | Giới hạn **300 ký tự** (`profile.full_name_max_chars`), cột `VARCHAR(300)` | Anh chỉ định (2026-10-11) | Đổi giá trị trong `config.dev.yaml` (trần cột 300; muốn hơn phải đổi migration) |
| 4 | `full_name` trả về trong `user` của `verify-otp` và `login` (dùng chung `userResponse`), vắng mặt nếu `NULL` | Cùng quy ước `omitempty` với `dharma_name` | — |
| 5 | `GET/PATCH /v1/users/me` có `full_name`; PATCH sửa được nhưng **không xoá được** (`null`/rỗng/quá dài → `400 INVALID_FULL_NAME`) | Anh yêu cầu thêm /me; trường bắt buộc nên không cho xoá | Cho phép xoá nếu BA muốn tên tuỳ chọn |

Chưa làm: cascade spec `docs/specs/features/onboarding/` (requirements → design → tasks → flow → changelog) và OKF `docs/knowledge/features/register.md`.
