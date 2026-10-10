# Handoff — 2026-10-10 21:22
<!-- tự sinh khi kết thúc phiên 84ab2ef4 bằng claude-haiku-4-5-20251001; /handoff để ghi bản chất lượng cao hơn -->

## Đang làm dở
- Temple-suggestion backoffice phase 3-5: domain/mock xong, pending handler/usecase/repository/route admin
- Genealogy spec v3.0: pending Mermaid flow.md validation, service layer code, migration, openapi
- E2E test timeout: 4 tests timeout khi chạy song song, pass riêng lẻ, root cause TBD

## Việc tiếp theo
1. Temple-suggestion handler/usecase/repository + wire route GET queue, PATCH approve/reject (phase 3-5)
2. Genealogy v3.0: validate Mermaid flow, code service layer, migration, openapi update
3. Investigate E2E test timeout - run full suite, check race conditions

## Tồn đọng / blocker
- OTP hardcode '111111' - blocker go-live, risk production deploy
- CORS '*' - security risk, cần siết chỉ FE domain server
- Validation thiếu: mật khẩu >72 byte, ordination_place >255 ký tự trả 500 thay 400
- AC69 xưng hô v1 chờ duyệt, AC markers (10,19,24,25,27,31,35,39,40,44,47,54,55,56,63,65,66)
- 16 file backofficeService chưa có concept OKF (config/handlers/domain/repository/usecase/mailer)
- Client repo chưa xử lý expires_in từ server (struct mẫu doc chưa cập nhật field)

## Quyết định gần đây
- Dùng 34 đơn vị hành chính mới (1/7/2025) - 6 TP + 28 tỉnh - thay 63 cũ; danh sách cứng trong domain/province.go
- Khớp tìm kiếm không dấu dùng Postgres unaccent extension (reuse từ 0019), hôm rồi bảo bệ
- Thêm cột province tường minh thay deduce từ address - tránh lỗi quy tắc dữ liệu kém rõ ràng
- Query tìm kiếm validation: q trống hoặc >100 ký tự → 400 INVALID_FILTER (cùng mã GET /v1/sutras)
- Bỏ hàm đoán tỉnh từ address - để dữ liệu rõ ràng, seed bắt buộc có province
- Migration 0025 không đụng address, chỉ UPDATE tên cũ → tên mới (áp lại idempotent)
- cmd/seedtemples từ chối file nếu province ngoài danh sách 34 đơn vị

## Nợ tài liệu (tự động)
Code đã đổi nhưng chưa có concept OKF nào trỏ tới (dùng skill `il-okf-writer` sau khi code đã kiểm chứng):
- pkg/mailer/ (2 file: log_mailer.go, mailer.go)
- services/backofficeService/cmd/ (1 file: main.go)
- services/backofficeService/cmd/seedadmin/ (1 file: main.go)
- services/backofficeService/config/ (1 file: config.go)
- services/backofficeService/delivery/http/ (5 file: admin_auth_handler.go, admin_user_handler.go, errors.go, +2 file)
- services/backofficeService/domain/ (3 file: admin.go, heritage.go, temple_suggestion.go)
- services/backofficeService/repository/ (1 file: postgres_admin_repository.go)
- services/backofficeService/usecase/ (2 file: admin_auth_usecase.go, admin_user_usecase.go)
