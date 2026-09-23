# Requirements — api-response-envelope

Version: 1.2
Last updated: 2026-09-22
Status: implemented (2026-09-21) — retroactive: code sửa và test xong trước khi viết spec này. Không còn marker `⚠️ Not confirmed`.
Module: hạ tầng dùng chung (`pkg/utils`, `pkg/jwt/middleware`, `pkg/ratelimit/middleware`, `pkg/middleware`) + composition root `services/appService/cmd`
Code: `pkg/utils/`, `pkg/jwt/middleware/`, `pkg/ratelimit/middleware/`, `pkg/middleware/`, `services/appService/delivery/http/fallback.go`, `services/appService/cmd/main.go`

> See `changelog.md` in this folder for the version history of this spec.

Spec này **tách ra từ** việc sửa tài liệu envelope của `onboarding` (v4.3) và `login` (v1.7): sau khi docs khớp code, quét toàn repo vẫn còn các response **không** đi qua `utils.ApiResponse`. Giống tiền lệ [`../json-body-hardening/`](../json-body-hardening/requirements.md): lỗi nằm ở hạ tầng dùng chung nên sửa ở gốc.

## Bằng chứng (trước khi sửa, 2026-09-21)

| Nguồn | Response cũ | Client của appService gặp được? |
|---|---|---|
| `net/http.ServeMux` — path không có route | `404` `text/plain` `404 page not found` | Có |
| `net/http.ServeMux` — sai method | `405` `text/plain` `Method Not Allowed` | Có |
| Panic trong handler | `net/http` đóng kết nối, không có response | Có |
| `pkg/jwt/middleware` `handleError` mặc định | `401` `{"error":"<message>"}` | Không (appService ghi đè bằng `WriteAuthError`) |
| `pkg/jwt/middleware` `RequireRole` | `401`/`403` `text/plain` qua `http.Error` | Không (chưa dùng) |
| `pkg/ratelimit/middleware` | `429` `text/plain` `Too Many Requests` | Không (chưa dùng) |
| `pkg/middleware.Recovery` | `500` `{"status":"error","error":{"code":"ERR_INTERNAL",...}}` | Không (chưa dùng) |
| `pkg/utils` `Write*` khi encode lỗi | `http.Error` ghi thêm text `500` **sau** status đã gửi → JSON hỏng | Hiếm |

## User Story

> Là một **client gọi API InnerLamp** (app mobile),
> Tôi muốn **mọi response có body đều cùng một định dạng JSON**, kể cả lỗi do router, middleware hay panic sinh ra,
> Để **chỉ viết một bộ parse và rẽ nhánh theo `error.code`, không phải đoán body là text, JSON phẳng hay envelope**.

## Acceptance Criteria (EARS notation)

1. THE SYSTEM SHALL write every successful response that has a body as `{"success": true, "data": <payload>}` with `Content-Type: application/json`.
2. THE SYSTEM SHALL write every error response as `{"success": false, "error": {"code", "message", "type"}, "details"?}` with `Content-Type: application/json`, where `details` (khi có) nằm cạnh `error`, `code` là `UPPER_SNAKE_CASE`, và `type` ∈ `validation` / `business` / `system`.
3. WHEN a request path matches no route in `services/appService`, THE SYSTEM SHALL respond `404` with code `NOT_FOUND`.
4. WHEN a request path matches a route but not its method, THE SYSTEM SHALL respond `405` with code `METHOD_NOT_ALLOWED` and keep the `Allow` header `ServeMux` computes.
5. WHEN a handler in `services/appService` panics, THE SYSTEM SHALL respond `500` with code `INTERNAL_ERROR`, the same body as any other 500, and SHALL NOT expose the panic value to the client.
6. WHEN `pkg/jwt/middleware` rejects a token and no `ErrorHandler` is configured, THE SYSTEM SHALL respond `401` with code `UNAUTHORIZED`; WHEN `RequireRole` finds no claims it SHALL respond `401` `UNAUTHORIZED`, and WHEN the role does not match it SHALL respond `403` `FORBIDDEN`.
7. WHEN `pkg/ratelimit/middleware` blocks a request, THE SYSTEM SHALL respond `429` with code `RATE_LIMITED`, `details.retry_after_seconds`, and the existing `Retry-After` / `X-RateLimit-*` headers.
8. WHEN `pkg/middleware.Recovery` recovers a panic, THE SYSTEM SHALL respond `500` with code `INTERNAL_ERROR` and type `system`.
9. WHEN encoding the envelope fails after the status line has been sent, THE SYSTEM SHALL NOT append any further bytes to the body.
10. WHEN a route matches, THE SYSTEM SHALL leave its response untouched — mọi status/code/body hiện có của US-001 và US-002 giữ nguyên, trừ AC11.
11. WHEN `POST /v1/auth/logout` succeeds (session revoked or already gone), THE SYSTEM SHALL respond `200` with `{"success": true}` (no `data`) and `Content-Type: application/json`, instead of `204 No Content` (login AC13/AC14).

## Ngoại lệ (không có body nên không có envelope)

- Redirect `307` của `ServeMux` khi path cần làm sạch (`/a/../b`) — client HTTP tự theo redirect.

> Preflight `OPTIONS` của `pkg/middleware.CORS` từng là ngoại lệ ở đây (`200` rỗng, chưa dùng ở appService). Từ v1.2 nó **không còn** là ngoại lệ: khi `CORS` được wire ở appService (`docs/specs/features/http-middleware-wiring/` AC4/AC10), preflight trả `200` `application/json` `{"success": true}` qua `utils.WriteSuccessResponseWithStatus` — cùng envelope, cùng bộ parse.

## Out of scope

- gRPC (`pkg/*/middleware/grpc.go`, `pkg/middleware` interceptors) — giao thức khác, lỗi đi qua `status.Status`, không có JSON body.
- Rò rỉ stack trace qua `RecoveryInterceptor` gRPC — đã ghi ở `pkg/middleware/README.md`, không đổi ở đây.
- Đổi message tiếng Anh sang tiếng Việt, i18n message.

## Pending

- Không có.

## Open questions

- Không có.

## Open questions (resolved)

- ~~Logout có đổi từ `204 No Content` sang `200 {"success": true}` không?~~ → **Có** — để client không cần nhánh riêng cho body rỗng; mọi response thành công đều parse cùng một envelope. Thành AC11. Resolved 2026-09-21.

## Ghi chú kỹ thuật liên quan

- Định nghĩa envelope là `pkg/utils.ApiResponse` + `ErrorDetail`; đặc tả dạng văn bản ở `pkg/utils/README.md` §Envelope response chuẩn. `docs/specs/api/openapi.yaml` dùng `ApiErrorResponse` và các schema `*Response` bọc `{success, data}`.
- `pkg/jwt/middleware`, `pkg/ratelimit/middleware`, `pkg/middleware` giờ import `pkg/utils` — `pkg/utils` chỉ import stdlib nên không tạo vòng.
