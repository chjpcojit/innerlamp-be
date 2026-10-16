# Handoff — 2026-10-16 15:54
<!-- tự sinh khi kết thúc phiên 29af7e1c bằng claude-haiku-4-5-20251001; /handoff để ghi bản chất lượng cao hơn -->

## Đang làm dở
- Tạo PR login v2 (chưa tạo vì không có gh CLI)
- Tạo PR OpenAPI + web/docs (chưa tạo vì không có gh CLI)

## Việc tiếp theo
1. Cài gh, tạo hai PR và gửi links
2. Điều tra npm run build fail trong worktree
3. Regenerate erd.svg, write OKF concept cho temple-announcement, fix E2E tests

## Tồn đọng / blocker
- Máy không có gh CLI để tạo PR
- npm run build fail trong docs/openapi-auth-v2 worktree (pass ngoài worktree, root cause chưa xác định)
- mmdc tool không có để regenerate erd.svg

## Quyết định gần đây
- OpenAPI paths: /v1/auth/resend-otp bỏ, /v1/auth/password-reset/* thêm vào, theo code thực tế
- Error codes mới: INVALID_EMAIL, REGISTRATION_IN_PROGRESS (kèm retry_after_seconds), EMAIL_SEND_FAILED, OTP_LOCKED, RATE_LIMITED
- PR 2 base vào feat/login-v2 (không phải main) vì PR 1 chưa merge

## Nợ tài liệu (tự động)
Code đã đổi nhưng chưa có concept OKF nào trỏ tới (dùng skill `il-okf-writer` sau khi code đã kiểm chứng):
- pkg/mailer/ (2 file: log_mailer.go, mailer.go)
- pkg/validation/ (3 file: email.go, phone_countries.go, validation.go)
- services/backofficeService/cmd/ (1 file: main.go)
- services/backofficeService/cmd/seedadmin/ (1 file: main.go)
- services/backofficeService/config/ (1 file: config.go)
- services/backofficeService/delivery/http/ (7 file: admin_auth_handler.go, admin_user_handler.go, errors.go, +4 file)
- services/backofficeService/domain/ (4 file: admin.go, heritage.go, temple_announcement.go, +1 file)
- services/backofficeService/e2e/ (1 file: harness.go)
- services/backofficeService/repository/ (3 file: postgres_admin_repository.go, postgres_temple_announcement_repository.go, postgres_temple_suggestion_repository.go)
- services/backofficeService/usecase/ (4 file: admin_auth_usecase.go, admin_user_usecase.go, temple_announcement_usecase.go, +1 file)
