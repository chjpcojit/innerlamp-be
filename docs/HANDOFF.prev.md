# Handoff — 2026-10-08 20:04
<!-- tự sinh khi kết thúc phiên 1d5adedc bằng claude-haiku-4-5-20251001; /handoff để ghi bản chất lượng cao hơn -->

## Đang làm dở
- Chạy `go test` toàn repo để xác nhận tất cả test pass — chỉ chạy -short và nhóm Sutra
- Commit/push code + spec sutra sau khi test pass

## Việc tiếp theo
1. AC45: Sửa lỗi USER_NOT_FOUND lộ số điện thoại đã đăng ký — Phase 4 cấp quyền
2. AC69: Bảng xưng hô v1 chờ duyệt — quyết định link 2 gia phả hay Deferred v1
3. Chạy test iOS foreground logging setup

## Tồn đọng / blocker
- AC45: USER_NOT_FOUND lộ số đã đăng ký — chưa fix
- AC69: Bảng xưng hô v1 chưa duyệt — chặn Kinship function
- AC markers mở: 10, 19, 24, 25, 27, 31, 35, 39, 40, 44, 47, 54, 55, 56, 63, 65, 66
- Logging iOS: device foreground, cần debug request setup

## Quyết định gần đây
- Summary được trả trong danh sách sutra (GET /v1/sutras), không chỉ chi tiết — thay đổi schema AC36–AC38
- Sutra v2.13: tách reference + author, han_translator, han_translation_year, summary, license — khớp DB hiện tại
- License mã: CC0-1.0, CC-BY-NC-4.0, PUBLIC_DOMAIN, COPY_ALLOWED, UNKNOWN
- URL ảnh: lưu path tương đối DB, ghép host lúc response qua server.public_base_url

## Nợ tài liệu (tự động)
Code đã đổi nhưng chưa có concept OKF nào trỏ tới (dùng skill `il-okf-writer` sau khi code đã kiểm chứng):
- pkg/mailer/ (2 file: log_mailer.go, mailer.go)
- services/backofficeService/cmd/ (1 file: main.go)
- services/backofficeService/cmd/seedadmin/ (1 file: main.go)
- services/backofficeService/config/ (1 file: config.go)
- services/backofficeService/delivery/http/ (5 file: admin_auth_handler.go, admin_user_handler.go, errors.go, +2 file)
- services/backofficeService/domain/ (1 file: admin.go)
- services/backofficeService/repository/ (1 file: postgres_admin_repository.go)
- services/backofficeService/usecase/ (2 file: admin_auth_usecase.go, admin_user_usecase.go)
