# Handoff — 2026-10-09 11:20
<!-- tự sinh khi kết thúc phiên be15fe96 bằng claude-haiku-4-5-20251001; /handoff để ghi bản chất lượng cao hơn -->

## Đang làm dở
- Genealogy spec v3.0 — chưa validate Mermaid diagram, chưa code, migration, openapi.yaml, ERD

## Việc tiếp theo
1. Validate Mermaid flow.md trong genealogy/flow.md
2. Tạo migration genealogy_invitations và cập nhật openapi.yaml spec
3. Code genealogy service (handlers, usecases, repositories) theo spec v3.0

## Tồn đọng / blocker
- AC45: USER_NOT_FOUND lộ số — spec v3.0 giải quyết (invitation), chưa code
- AC69: xưng hô v1 — chờ duyệt
- AC markers mở: 10, 19, 24, 25, 27, 31, 35, 39, 40, 44, 47, 54, 55, 56, 63, 65, 66
- iOS logging: device foreground setup chưa debug

## Quyết định gần đây
- Genealogy v3.0: mọi invitation qua genealogy_invitations, owner không phân biệt số có/chưa user
- genealogy_invitations: partial unique (genealogy_id, phone_number), tối đa 1 lời mời mở/số
- AC70-75: thêm 5 route invitation, 2 step design (I-1 chấp nhận, I-2 từ chối), thay AC44-47
- A-002: ghi mặc định genealogy invitations (replace A-001 đánh dấu sai)

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
