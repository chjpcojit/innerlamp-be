# Handoff — 2026-10-14 20:50
<!-- tự sinh khi kết thúc phiên 36af07db bằng claude-haiku-4-5-20251001; /handoff để ghi bản chất lượng cao hơn -->

## Đang làm dở
_(không có)_

## Việc tiếp theo
1. Regenerate erd.svg (cần tool mmdc hoặc giải pháp thay thế)
2. Write OKF concept cho temple-announcement packages (chờ user code review confirm)
3. Fix E2E administrative-units FAIL (root cause chưa xác định)

## Tồn đọng / blocker
- erd.svg: mmdc tool không có trên machine, không thể regenerate mermaid diagram
- OKF concept: chờ user confirm code review trước khi viết concept
- E2E administrative-units: FAIL, root cause chưa rõ
- E2E dharma-talk: FAIL, root cause chưa rõ
- E2E 4 parallel tests: FAIL do DB state/transaction issues
- Shared files (config.go, main.go, errors.go) block temple-suggestion & genealogy

## Quyết định gần đây
- body_max_chars=5000 (spec không định), page_size max 50 app/100 backoffice, dev 20
- Issuing body name rỗng/quá dài → 400 VALIDATION_ERROR, page_size vượt max hạ không error
- content_editor role post thông báo như admin, rate limit 60 req/1min, p95<500ms
- Temple-announcement không commit chưa, chờ code review confirm & OKF concept

## Nợ tài liệu (tự động)
Code đã đổi nhưng chưa có concept OKF nào trỏ tới (dùng skill `il-okf-writer` sau khi code đã kiểm chứng):
- pkg/mailer/ (2 file: log_mailer.go, mailer.go)
- services/backofficeService/cmd/ (1 file: main.go)
- services/backofficeService/cmd/seedadmin/ (1 file: main.go)
- services/backofficeService/config/ (1 file: config.go)
- services/backofficeService/delivery/http/ (7 file: admin_auth_handler.go, admin_user_handler.go, errors.go, +4 file)
- services/backofficeService/domain/ (4 file: admin.go, heritage.go, temple_announcement.go, +1 file)
- services/backofficeService/e2e/ (1 file: harness.go)
- services/backofficeService/repository/ (3 file: postgres_admin_repository.go, postgres_temple_announcement_repository.go, postgres_temple_suggestion_repository.go)
- services/backofficeService/usecase/ (4 file: admin_auth_usecase.go, admin_user_usecase.go, temple_announcement_usecase.go, +1 file)
