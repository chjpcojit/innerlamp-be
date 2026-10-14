# Changelog — temple-announcement Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first. `SRS.md` (business-facing) is versioned in its own header.

## v1.2 (2026-10-14) — đồng bộ với code đã triển khai

Code và test đã xong (build, vet, `go test -short ./services/...`, E2E hai service với `TEST_DB_NAME=innerlamp_test` pass). Spec sửa theo code, không đổi hành vi.

- `requirements.md`: thêm `TA34` (`INVALID_PAGE` ở cả hai service; `page_size` quá lớn bị chặn, không lỗi), `TA35` (tên ban rỗng/quá dài là `400 VALIDATION_ERROR` với `details.field = name`), `TA36` (quy tắc `state`: `REVOKED` thắng, `expires_at == now` là `EXPIRED`); sửa `TA10`, `TA14`, `TA20` (ghi giá trị dev, `body_max_chars = 5000` là giả định), `TA22` (bỏ "inactive"); bảng route ghi đúng mã 400; Pending cập nhật.
- `design.md`: "năm route" -> sáu; repo công tắc là `GetSettings` (công tắc + chùa quy y có tên, một SELECT) thay cho `GetReceive`; config backoffice có `MaxPageSize`; ràng buộc `Validate()`; id `tann_`/`tib_` + uuid do repository sinh; helper `cmd/temple_announcement_routes.go`; `StateAt`; ánh xạ lỗi đầy đủ; R-1 chặt hơn (`page_size=0` tường minh là `INVALID_PAGE`).
- `flow.md`: đổi nhãn R-2 (`GetSettings`) và điều kiện lỗi R-1; không thêm/bớt Step.
- `tasks.md`: tích Phase 1 và Phase 8 (trừ OKF); ghi `erd.svg` chưa dựng lại.
- `api-contract.md`: file mới (v1.0) cho client web/app.
- Không đổi schema; `migrations/0036` giữ nguyên.

Follow-up chéo file:
- `docs/knowledge/`: OKF concept **chưa viết**; chỉ viết (`il-okf-writer`) sau khi con người kiểm chứng.
- `docs/specs/db/erd.svg`: chưa dựng lại từ `erd.mmd` (không có `mmdc` trong repo).
- `docs/specs/api/openapi.yaml`, `web/docs` (guide, sidebar, changelog, error-handling): đã cập nhật cùng đợt.

## v1.1 (2026-10-14) — chốt ba điểm mở + tách giới hạn tốc độ

Người dùng chốt: `content_editor` được đăng như `admin`; phân trang `page`/`page_size` + `total`; ngưỡng p95 < 500 ms; tách giới hạn tốc độ riêng.

- `requirements.md`: thêm AC `TA32` (giới hạn theo account riêng, độc lập nhóm hồ sơ) và `TA33` (`429 RATE_LIMITED`); thêm `429` vào ba route người dùng; xoá ba `⚠️ Not confirmed`, chuyển sang "Open questions (resolved)".
- `design.md`: `TempleAnnouncementConfig` có `RateLimit{Max, Window}`; wiring dùng `NewUserRateLimit` riêng; Known gap về giới hạn tốc độ được đóng lại.
- `tasks.md`: Phase 6/7 thêm config + wiring + E2E cho giới hạn; bỏ mục xác nhận `⚠️`.
- `flow.md`: không đổi (giới hạn nằm ở middleware, không thêm Step vào flow đọc/đăng).
- Không đổi schema; `migrations/0036` giữ nguyên.

## v1.0 (2026-10-14) — initial spec

Spec đầu tiên, viết từ `SRS.md` v0.2 sau khi người dùng chốt toàn bộ câu hỏi mở theo mặc định. Chưa có code.

- `requirements.md`: US-TAN-001/002; AC `TA1…TA31` (đọc, công tắc, ghi); bảng route; Out of scope; ba `⚠️ Not confirmed` còn mở (role `content_editor`, hình dạng phân trang, ngưỡng thời gian phản hồi).
- `design.md`: hai service dùng chung Postgres; hai bảng + một cột `users`; hai Step-by-step Flow (đọc `R-1…R-6`, đăng `P-1…P-6`).
- `flow.md`: hai sequence diagram mirror 1:1.
- `tasks.md`: 8 phase, mọi mục `[ ]`.
- Quyết định được ghi lại: công tắc tắt = ẩn hoàn toàn; hạn bắt buộc ≤ 365 ngày (config); không đọc/chưa đọc; không hẹn giờ; đơn vị phát hành là danh sách theo chùa (chỉ tạo + liệt kê ở v1); quyền xem suy ra từ `ordination_temple_id` tại thời điểm đọc.

Follow-up chéo file (chưa làm, đều nằm trong `tasks.md`):
- `migrations/0036_temple_announcements.sql`, `migrations/README.md`, `docs/specs/db/erd.mmd`/`erd.svg`.
- `docs/specs/api/openapi.yaml`: chín route mới.
- `docs/knowledge/`: chưa có concept; viết sau khi code được kiểm chứng.
