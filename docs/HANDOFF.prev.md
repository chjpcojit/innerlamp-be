# Handoff — 2026-10-14 17:13
<!-- tự sinh khi kết thúc phiên 8dcaaa6c bằng claude-haiku-4-5-20251001; /handoff để ghi bản chất lượng cao hơn -->

## Đang làm dở
- E2E test temple-event verification (dừng ở TEST_DB_NAME=inn)

## Việc tiếp theo
1. Commit temple-event changes & verify full E2E test
2. Fix E2E fail administrative-units & dharma-talk (root cause TBD)
3. Update openapi.yaml Profile schema (full_name + phone_number E.164)

## Tồn đọng / blocker
- E2E administrative-units & dharma-talk FAIL (root cause TBD)
- E2E 4 parallel test fail (DB state/transaction)
- Shared file conflicts config.go/main.go/errors.go blocking temple-suggestion & genealogy
- 19 backofficeService files need OKF documentation (deity v2.2)

## Quyết định gần đây
- temple-announcement: content_editor role được đăng thông báo như admin
- temple-announcement: phân trang page + page_size + total, v1 giữ nguyên
- temple-announcement: rate limit tách riêng max:60/window:1m, 429 RATE_LIMITED
- temple-announcement: p95 < 500ms với 200 thông báo còn hạn (NFR-001)
- User auth: Bearer only; 401 UNAUTHORIZED unified (missing/invalid/expired)
- Rate limit /v1/users/me*: 10 req/phút/account, return 429 RATE_LIMITED
- Ordination temple: PATCH /v1/users/me {ordination_temple_id}, null=delete, 10 req/phút
- Practice schedule: GET /v1/temples/{id}/schedule, weekday 0=Sun…6=Sat, [] means daily

## Nợ tài liệu (tự động)
Code đã đổi nhưng chưa có concept OKF nào trỏ tới (dùng skill `il-okf-writer` sau khi code đã kiểm chứng):
- pkg/mailer/ (2 file: log_mailer.go, mailer.go)
- services/backofficeService/cmd/ (1 file: main.go)
- services/backofficeService/cmd/seedadmin/ (1 file: main.go)
- services/backofficeService/config/ (1 file: config.go)
- services/backofficeService/delivery/http/ (6 file: admin_auth_handler.go, admin_user_handler.go, errors.go, +3 file)
- services/backofficeService/domain/ (3 file: admin.go, heritage.go, temple_suggestion.go)
- services/backofficeService/e2e/ (1 file: harness.go)
- services/backofficeService/repository/ (2 file: postgres_admin_repository.go, postgres_temple_suggestion_repository.go)
- services/backofficeService/usecase/ (3 file: admin_auth_usecase.go, admin_user_usecase.go, temple_suggestion_usecase.go)
