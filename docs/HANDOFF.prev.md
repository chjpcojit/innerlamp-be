# Handoff — 2026-10-10 20:33
<!-- tự sinh khi kết thúc phiên da948378 bằng claude-haiku-4-5-20251001; /handoff để ghi bản chất lượng cao hơn -->

## Đang làm dở
- Temple-suggestion backoffice phase 3-5: domain/mock xong, pending handler/usecase/repository/route admin
- Genealogy spec v3.0: pending Mermaid flow.md validation, service layer code, migration, openapi
- E2E test timeout: 4 tests timeout khi chạy song song, pass riêng lẻ, root cause TBD

## Việc tiếp theo
1. Investigate/fix E2E test timeout - run full suite, check race conditions, verify build confidence
2. Temple-suggestion handler/usecase/repository (phase 3-5) + wire route GET queue, PATCH approve/reject
3. Genealogy v3.0: validate Mermaid, code service, migration, update openapi.yaml

## Tồn đọng / blocker
- OTP hardcode '111111' - blocker go-live, risk deploy production
- CORS '*' - security risk, cần siết chỉ FE domain server
- Validation thiếu: mật khẩu >72 byte, ordination_place >255 ký tự trả 500 thay 400
- AC69 xưng hô v1 chờ duyệt, AC markers (10,19,24,25,27,31,35,39,40,44,47,54,55,56,63,65,66)
- 16 file backofficeService chưa có concept OKF (config/handlers/domain/repository/usecase/mailer)
- Client repo chưa xử lý expires_in từ server (struct mẫu doc chưa cập nhật field)

## Quyết định gần đây
- Temple suggestion: public anonymous submit webAPIKey, rate limit 5/IP/hour, free-text province
- Temple ID format temple_<region>_<NNN>, admin từ hạng di sản + năm thành lập
- Heritage priority table copy backofficeService, không dùng pkg/ chung (AC21)
- Genealogy invitations: tất cả mời dù có/chưa user, owner không dò tình trạng
- genealogy_invitations: partial unique (genealogy_id, phone_number), max 1 mời/số
- AuthResult trả expires_in (dev 900s), client thật ở repo khác
- Ordination place: ordination_temple_id (FK temples), validation 400 INVALID_TEMPLE
- Register v7.0: ordination_temple_id bắt buộc, drop ordination_place migration 0023

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
