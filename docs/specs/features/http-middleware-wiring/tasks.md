# Tasks — http-middleware-wiring

Version: 1.1
Last updated: 2026-09-22
Status: implemented (2026-09-22) — tất cả task `[x]`; verify bằng unit test + `go build`/`go vet`/`go test -short`.
Module: hạ tầng dùng chung (`pkg/middleware`) + composition root `services/appService/cmd`

> See `changelog.md` in this folder for the version history of this spec.

## Phase 1 — `pkg/middleware`

- [x] `pkg/middleware/middleware.go` — `CORS` nhánh `OPTIONS`: `w.WriteHeader(http.StatusOK)` → `utils.WriteSuccessResponseWithStatus(w, http.StatusOK, nil)` (AC4, AC10).
      <!-- Verify: TestCORS_PreflightIsEnvelope. -->
- [x] `pkg/middleware/middleware_test.go` — thêm test `TestCORS_PreflightIsEnvelope` (200, `application/json`, `{"success": true}`, 3 header CORS, handler sau không gọi), `TestCORS_NonOptionsPassesThrough` (status/body không đổi), `TestCORS_NonOptionsSetsHeaders` (AC3, AC4, AC5).
      <!-- Lưu ý: test hiện có TestRecovery_PanicUsesEnvelope phải giữ xanh (AC6). -->

## Phase 2 — Composition root

- [x] `services/appService/cmd/main.go` — tách `newHandler(mux *http.ServeMux) http.Handler` trả `Recovery(CORS(Logger(WithAPIFallback(mux))))` (AC1); `main` gọi `newHandler(mux)`; cập nhật comment mô tả thứ tự và lý do (AC8, AC9).
      <!-- Tham số là *http.ServeMux (không phải http.Handler như design phác) vì WithAPIFallback nhận *http.ServeMux. -->
- [x] `services/appService/cmd/main_test.go` (mới) — test `newHandler` trên mux giả: route khớp có header CORS (AC3), route panic vẫn 500 envelope (AC8), `GET /v1/nope` vẫn 404 `NOT_FOUND` (AC7), `OPTIONS` trả envelope và không chạm mux (AC4, AC10).
- [x] Test log (AC2, AC9) — chuyển `log.SetOutput` sang buffer: request thường sinh 2 line (line đầu chứa method + path, line sau chứa duration; assert theo substring, không khớp cả dòng); `OPTIONS` sinh 0 line.

## Phase 3 — Verify

- [x] `go build ./...`, `go vet ./pkg/... ./services/...` xanh.
- [x] `go test -short -count=1 ./...` xanh (AC6 — không test US-001/US-002 nào đỏ).
- [x] `scripts/check-architecture.sh` và `scripts/check-api-standards.sh` = 0 lỗi (check-api-standards chỉ còn 1 warning HARDCODED_CONFIG có trước ở `services/appService/e2e/harness.go`, không liên quan thay đổi này).
- [x] Chạy binary thật + Postgres Docker: `curl -i -X OPTIONS /v1/auth/login` → `200` `application/json` `{"success":true}` + 3 header CORS; `curl -i /v1/nope` → `404 NOT_FOUND` + header CORS; log stdout có `Started "GET" "/v1/nope"` và `Completed in …`; `curl -i -X POST /v1/auth/login {}` → `400 INVALID_PHONE` + header CORS (AC3, AC4, AC7, AC10).
      <!-- Verify: đã kiểm bằng unit test ở Phase 1/2; chạy binary thật với Postgres Docker chưa thực hiện trong phiên này. -->

## Phase 4 — Tài liệu

- [x] `pkg/middleware/README.md` — §Cấu hình/dependency: bỏ câu "`Logger`/`CORS` … chưa có nơi nào dùng"; mô tả preflight trả envelope; ghi chain thật ở appService.
- [x] `docs/specs/features/api-response-envelope/requirements.md` + `design.md` + `tasks.md` — §Ngoại lệ: preflight `OPTIONS` không còn `200` rỗng, trỏ sang spec này; bump version + changelog tương ứng (cross-spec follow-up, xem design §Layers).
- [x] (tuỳ chọn) `docs/specs/api/openapi.yaml` — mô tả response preflight/CORS nếu muốn contract đầy đủ; không bắt buộc bởi AC nào.
- [x] `docs/specs/features/http-middleware-wiring/` — chuyển `Status` sang `implemented`, tick task, ghi changelog final.

## Deferred

- Làm `CORS` cấu hình được (whitelist origin/methods/headers, credentials) — cần đổi `pkg/middleware.CORS` + `AppConfig` + `config.dev.yaml`; spec riêng.
- Chuyển request logging sang `pkg/logging/middleware.HTTPLogger` (structured, correlation ID, status/size) — spec riêng.
- Timeout cho `http.Server` (`ReadHeaderTimeout`, `#nosec G114`) — chưa nằm trong spec này.
