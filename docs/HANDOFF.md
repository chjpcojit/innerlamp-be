# Handoff — 2026-10-14 20:32
<!-- tự sinh khi kết thúc phiên 36af07db bằng claude-haiku-4-5-20251001; /handoff để ghi bản chất lượng cao hơn -->

## Đang làm dở
- erd.svg regenerate (blocked: mmdc tool not available on machine)
- OKF concept for temple-announcement (awaiting user/code review sign-off)

## Việc tiếp theo
1. Confirm temple-announcement code review & write OKF concept
2. Fix E2E administrative-units fail (root cause TBD)
3. Fix E2E dharma-talk + 4 parallel test DB state/transaction issues

## Tồn đọng / blocker
- erd.svg: mmdc tool not available, cannot regenerate mermaid diagram
- OKF concept pending user code review confirm for temple-announcement
- E2E administrative-units FAIL (root cause TBD)
- E2E dharma-talk FAIL (root cause TBD)
- E2E 4 parallel test FAIL (DB state/transaction issues)
- Shared file conflicts: config.go/main.go/errors.go block temple-suggestion & genealogy

## Quyết định gần đây
- temple-announcement: body_max_chars=5000 (spec không định), page_size max 50(app)/100(backoffice)
- temple-announcement: page_size vượt max hạ, không error; ban phát hành null/dài → 400 VALIDATION_ERROR
- temple-announcement: content_editor role đăng thông báo như admin; phân trang page+page_size+total
- temple-announcement: rate limit 60 req/window:1m, 429 RATE_LIMITED; p95<500ms với 200 thông báo
- User auth: Bearer only, 401 UNAUTHORIZED; Rate limit /v1/users/me*: 10 req/phút/account
- Ordination temple: PATCH /v1/users/me {ordination_temple_id}, null=delete, 10 req/phút
- Practice schedule: GET /v1/temples/{id}/schedule, weekday 0=Sun…6=Sat, [] means daily

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
