# Handoff — 2026-10-16 20:39
<!-- tự sinh khi kết thúc phiên 29af7e1c bằng claude-haiku-4-5-20251001; /handoff để ghi bản chất lượng cao hơn -->

## Đang làm dở
- Tạo PR login v2 → main (chưa tạo, cần gh CLI hoặc GitHub web)
- Tạo PR OpenAPI+web/docs → feat/login-v2 base (đã push docs/openapi-auth-v2, chưa mở PR)

## Việc tiếp theo
1. Cài gh CLI hoặc tạo PR qua GitHub web, gửi links
2. Điều tra npm run build fail trong docs/openapi-auth-v2 worktree (pass ngoài worktree)
3. Regenerate erd.svg, write OKF concepts, fix E2E tests

## Tồn đọng / blocker
- gh CLI chưa cài, không có GitHub token để tạo PR programmatically
- npm run build fail trong worktree nhưng pass ngoài, root cause chưa xác định
- mmdc tool chưa cài để regenerate erd.svg

## Quyết định gần đây
- OpenAPI: /v1/auth/resend-otp bỏ, /v1/auth/password-reset/* thêm, theo code thực tế
- Error codes: INVALID_EMAIL, REGISTRATION_IN_PROGRESS (kèm retry_after_seconds), EMAIL_SEND_FAILED, OTP_LOCKED, RATE_LIMITED
- PR 2 (OpenAPI) base vào feat/login-v2, không phải main, vì PR 1 chưa merge

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
