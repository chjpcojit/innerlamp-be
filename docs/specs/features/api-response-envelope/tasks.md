# Tasks — api-response-envelope

Version: 1.2
Last updated: 2026-09-22
Status: done — Phase 1–5 implement và verify xong (2026-09-21); Phase 6 (v1.2) là cross-spec follow-up, đã xong cùng `http-middleware-wiring`. Không còn marker `⚠️ Not confirmed`; AC11 (logout `200`) thêm ở v1.1, đã làm.
Module: hạ tầng dùng chung + `services/appService/cmd`

> See `changelog.md` in this folder for the version history of this spec.

## Phase 1 — `pkg/` envelope sources

- [x] `pkg/utils/http.go` — bỏ `http.Error` sau encode lỗi ở `WriteSuccessResponseWithStatus` và `WriteErrorResponse` (AC9).
- [x] `pkg/jwt/middleware/http.go` — `handleError` mặc định + `RequireRole` dùng `utils.WriteBusinessError` (AC6).
      <!-- Verify: TestAuthenticate_DefaultErrorUsesEnvelope, TestRequireRole_ErrorsUseEnvelope (http_envelope_test.go). -->
- [x] `pkg/ratelimit/middleware/http.go` — `429 RATE_LIMITED` + `details.retry_after_seconds` (AC7).
      <!-- Verify: TestHTTPMiddleware_BlockedUsesEnvelope. -->
- [x] `pkg/middleware/middleware.go` — `Recovery` dùng `utils.WriteSystemError` (AC8).
      <!-- Verify: TestRecovery_PanicUsesEnvelope (middleware_test.go, file test đầu tiên của package). -->

## Phase 2 — `services/appService`

- [x] `delivery/http/fallback.go` — `WithAPIFallback` + `fallbackWriter` (AC3, AC4, AC10).
      <!-- Verify: TestWithAPIFallback_UnknownPathIs404Envelope, _WrongMethodIs405EnvelopeWithAllow,
      _MatchedRouteUntouched, _PathCleanRedirectUntouched. Viết test trước, đỏ vì undefined, rồi xanh. -->
- [x] `cmd/main.go` — `Recovery(WithAPIFallback(mux))` làm handler của `ListenAndServe` (AC5).

## Phase 3 — Code mẫu trong `pkg/*/examples`

- [x] `pkg/jwt/examples/http/main.go` — `sendJSON`/`sendError` dùng `utils.Write*`; bỏ struct `ErrorResponse`; `http.Error` 405 → `sendError`.
- [x] `pkg/metrics/examples/complete_example.go`, `pkg/ratelimit/examples/login_limit/main.go`, `pkg/ratelimit/examples/chained_limit/main.go`, `pkg/secrets/examples/example_usage.go` — bỏ `w.Write`/`http.Error` thô.

## Phase 4 — Verify + tài liệu

- [x] `go build ./...`, `go vet ./pkg/... ./services/...`, `go test -short -count=1 ./...` xanh; `scripts/check-architecture.sh` và `scripts/check-api-standards.sh` = 0 lỗi.
- [x] Chạy binary thật (Postgres Docker `innerlamp-pg`): `GET /v1/nope` → 404 `NOT_FOUND`; `GET /v1/auth/login` → 405 `METHOD_NOT_ALLOWED` + `Allow: POST`; `POST /v1/auth/logout` không token → 401 `UNAUTHORIZED`; `POST /v1/auth/login {}` → 400 `INVALID_PHONE`. Cả 4 là envelope `application/json`.
- [x] `pkg/utils/README.md` — mục "Envelope response chuẩn"; ví dụ `code` đổi sang `UPPER_SNAKE_CASE`; mục Lưu ý về encode lỗi.
- [x] `pkg/jwt/README.md`, `pkg/ratelimit/README.md`, `pkg/middleware/README.md` — mô tả body lỗi mới + phụ thuộc `pkg/utils`.
- [x] `pkg/jwt/QUICK_START.md`, `pkg/jwt/examples/README.md`, `pkg/session/QUICK_START.md`, `pkg/session/CLOUD_RUN.md`, `pkg/tracing/middleware/README.md`, `pkg/metrics/USAGE.md` — snippet dùng `utils.Write*`, output mẫu bọc `{success, data}`.
- [x] `docs/specs/api/openapi.yaml` v3.6 — `NOT_FOUND`, `METHOD_NOT_ALLOWED` vào `ApiErrorResponse`; mô tả envelope ở `info.description`.
- [x] `docs/ARCHITECTURE.md` — đồ thị phụ thuộc `pkg/` + mục envelope + bỏ gap "`pkg/middleware` không được dùng".
- [x] `login` v1.8, `json-body-hardening` v1.4 — cập nhật các câu đã cũ.

## Phase 5 — Logout `200` envelope (v1.1, AC11)

- [x] `delivery/http/login_handler.go` `Logout` → `utils.WriteSuccessResponse(w, nil)` — verified 2026-09-21: `go test ./services/appService/delivery/http/` → `ok`; test `TestLoginHandler_Logout` assert `200`, `Content-Type: application/json`, body `{"success": true}`.
- [x] `docs/specs/api/openapi.yaml` v3.7 — logout `204` → `200` + schema `{success}`; bỏ câu "bodiless reply is logout's 204" ở `info.description`.
- [x] `login` v1.9 — AC13/AC14, bảng API contract, Logout flow Step 3.
- [x] `pkg/utils/README.md` — ngoại lệ không body bỏ `204`; ghi `WriteSuccessResponse(w, nil)` → `{"success": true}`.
- [x] Chạy binary thật: `POST /v1/auth/logout` bằng access token hợp lệ → `200` `{"success": true}`; gọi lại → vẫn `200` (idempotent) — verified 2026-09-21: binary thật + Postgres Docker `innerlamp-pg` + curl: register → verify-otp (`111111`) → login → logout (`200`, `application/json`, `{"success":true}`) → logout lại (`200`, cùng body) → refresh bằng refresh token của session đó (`401` `REFRESH_TOKEN_INVALID`) → logout không token (`401` `UNAUTHORIZED`). DB: session login `is_revoked = t`, `revoked_reason = 'logout'`; session của verify-otp không bị đụng.

## Phase 6 — Preflight `OPTIONS` có envelope (v1.2, cross-spec follow-up)

- [x] `pkg/middleware/middleware.go` — `CORS` nhánh `OPTIONS` dùng `utils.WriteSuccessResponseWithStatus(w, 200, nil)` thay `w.WriteHeader(http.StatusOK)`; preflight không còn là ngoại lệ không body.
      <!-- Verify: TestCORS_PreflightIsEnvelope, TestNewHandler_PreflightStopsAtCORS (pkg/middleware, services/appService/cmd). Code thuộc `http-middleware-wiring`. -->
- [x] `requirements.md`/`design.md` — bỏ preflight khỏi §Ngoại lệ, ghi chéo sang `http-middleware-wiring`.
- [x] `docs/specs/api/openapi.yaml` — follow-up tuỳ chọn mô tả preflight/CORS; không bắt buộc bởi AC nào (xem `http-middleware-wiring` tasks Phase 4).

## Deferred

- Không có.
