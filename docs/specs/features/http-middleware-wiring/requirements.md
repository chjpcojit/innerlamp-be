# Requirements — http-middleware-wiring

Version: 1.1
Last updated: 2026-09-22
Status: implemented (2026-09-22) — AC1–AC11 đã implement và verify bằng unit test (`go test -short ./...`), `go build`/`go vet` xanh.
Module: hạ tầng dùng chung (`pkg/middleware`) + composition root `services/appService/cmd`
Code: `pkg/middleware/middleware.go`, `services/appService/cmd/main.go` (kèm test `pkg/middleware/middleware_test.go`, `services/appService/cmd/main_test.go`)

> See `changelog.md` in this folder for the version history of this spec.

`pkg/middleware` đã có sẵn `Logger` và `CORS` từ đầu, nhưng cả hai **chưa nơi nào dùng** — `cmd/main.go` mới chỉ bọc `Recovery(WithAPIFallback(mux))` (ghi rõ ở `pkg/middleware/README.md` §Cấu hình/dependency). Hệ quả quan sát được:

- Không có log request/response cho bất kỳ route nào của appService.
- Không có header CORS, nên client web (khác origin với API) không gọi được, kể cả preflight.

Spec này **chỉ wire** hai middleware đang có vào composition root, không viết middleware mới. Ngoại lệ duy nhất phải sửa code `pkg/`: nhánh preflight `OPTIONS` của `CORS` được đổi sang ghi envelope chuẩn (AC4, AC10) thay vì `200` rỗng — xem §"Ghi chú kỹ thuật liên quan" về ảnh hưởng lên `api-response-envelope`.

## Bằng chứng (trước khi wire)

| Nguồn | Hành vi hiện tại | Client gặp được? |
|---|---|---|
| `services/appService/cmd/main.go` | `Recovery(WithAPIFallback(mux))` — không có `Logger`, không có `CORS` | Có |
| `pkg/middleware.Logger` | Tồn tại, `log.Printf` method/path + thời gian; không nơi nào gọi | — |
| `pkg/middleware.CORS` | Tồn tại, set header cứng `*`; preflight `OPTIONS` trả `200` rỗng; không nơi nào gọi | — |
| Request tới `/v1/auth/*` | Không có dòng log nào, không có `Access-Control-Allow-*` | Có |

## User Story

> Là một **kỹ sư vận hành / lập trình viên client web InnerLamp**,
> Tôi muốn **mọi request không-phải-preflight được log (method, path, thời gian) và mọi response — kể cả preflight `OPTIONS` — mang header CORS cùng envelope chuẩn**,
> Để **debug được luồng request trên môi trường chạy thật, để client web khác origin gọi được API, và parse preflight bằng cùng một bộ parse envelope**.

## Acceptance Criteria (EARS notation)

1. THE SYSTEM SHALL build the top-level handler of appService as `Recovery(CORS(Logger(WithAPIFallback(mux))))` — `Recovery` ngoài cùng, rồi `CORS`, rồi `Logger`, `mux` trong cùng.

2. WHEN a request travels past the `Logger` layer, THE SYSTEM SHALL emit one start log line containing the request method and path **before** invoking the next handler, and one completion log line containing the elapsed duration **after** the next handler returns. Định dạng log giữ nguyên như `pkg/middleware.Logger` in ra hiện nay — spec này **không sửa** `Logger`.

3. WHEN any request receives a response from appService (route khớp, không khớp, preflight hay panic), THE SYSTEM SHALL set `Access-Control-Allow-Origin: *`, `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`, and `Access-Control-Allow-Headers: Content-Type, Authorization` on that response.

4. WHEN an `OPTIONS` request reaches `CORS` (preflight), THE SYSTEM SHALL respond `200` with `Content-Type: application/json` and body `{"success": true}` and SHALL NOT invoke the next handler.

5. WHEN a request's method is anything other than `OPTIONS`, THE SYSTEM SHALL set the CORS headers and continue to the next handler, and SHALL NOT alter that handler's status or body.

6. WHEN `Logger` is in the chain, THE SYSTEM SHALL NOT change any status code or response body that appService produced before this feature — toàn bộ hành vi của US-001 / US-002 giữ nguyên, chỉ thêm log line và header CORS.

7. WHEN a request hits an unknown path or a wrong method, THE SYSTEM SHALL still return the `404 NOT_FOUND` / `405 METHOD_NOT_ALLOWED` envelope (api-response-envelope AC3/AC4) **and** SHALL log that request like any other.

8. IF a panic occurs in `CORS`, `Logger`, the fallback, or a handler, THEN THE SYSTEM SHALL still respond `500` with code `INTERNAL_ERROR` (api-response-envelope AC5/AC8) — `Recovery` ở ngoài cùng bắt được panic của mọi tầng bên trong.

9. WHEN a preflight `OPTIONS` is answered by `CORS`, THE SYSTEM SHALL NOT emit the `Logger` start/completion lines for that request (CORS short-circuit trước `Logger`).

10. WHEN `CORS` writes its preflight response, THE SYSTEM SHALL write it through `utils.WriteSuccessResponseWithStatus` (nguồn ghi envelope duy nhất) instead of writing raw bytes or `http.Error`.

11. THE SYSTEM SHALL NOT introduce new configuration keys for CORS or request logging — `CORS` tiếp tục hardcode origin/methods/headers và `Logger` tiếp tục dùng `log.Printf` (xem §Out of scope).

## Ngoại lệ (không có body nên không có envelope)

- Redirect `307` của `ServeMux` khi path cần làm sạch (`/a/../b`) — client HTTP tự theo redirect; `CORS` vẫn set header lên response `307` này, nhưng không có body.
- Response `404`/`405` của `WithAPIFallback` vẫn là envelope như cũ — ngoại lệ cũ của api-response-envelope chỉ còn redirect, **preflight `OPTIONS` đã chuyển sang envelope ở AC4/AC10**.

## Out of scope

- Làm `CORS` cấu hình được (whitelist origin, methods, headers, `Access-Control-Allow-Credentials`) — AC11 giữ nguyên bản hardcode `*`. Xem `pkg/middleware/README.md` §Lưu ý: `*` + credentials không hợp lệ theo CORS spec.
- Chuyển sang structured logging / correlation ID / log status + duration + size (`pkg/logging/middleware.HTTPLogger`) — dùng `pkg/middleware.Logger` thô như hiện có.
- Sửa `pkg/middleware.Logger` để bắt status code hoặc xuất JSON.
- gRPC interceptor (`RecoveryInterceptor`, …) — chưa service nào dùng.
- Rate limit (`pkg/ratelimit/middleware`), metrics (`pkg/metrics/middleware`), tracing.
- Timeout/`ReadHeaderTimeout` cho `http.Server` (đang là TODO `#nosec G114` trong `main.go`).

## Pending

- AC2: hiện `Logger` không capture status code, nên completion line không kèm status. Nếu sau này cần status trong log thì phải là một spec riêng (out of scope ở đây).

## Open questions

- Không có.

## Ghi chú kỹ thuật liên quan

- `pkg/middleware` đã import `pkg/utils` (từ `Recovery`), nên `CORS` gọi `utils.WriteSuccessResponseWithStatus` **không tạo import mới hay vòng phụ thuộc**; `pkg/utils` vẫn là leaf chỉ dùng stdlib.
- **Hợp đồng phía client cho preflight**: browser tự phát `OPTIONS`; response giờ là `200` + `Content-Type: application/json` + `{"success": true}` + 3 header CORS, nên client dùng chung một bộ parse `utils.ApiResponse` cho cả response thường lẫn preflight. Đây là thay đổi theo hướng "`200` rỗng → có body"; code nào đang giả định preflight không body cần cập nhật.
- **Thứ tự thao tác trong nhánh preflight của `CORS`**: set 3 header `Access-Control-*` trước, rồi mới gọi `utils.WriteSuccessResponseWithStatus(w, http.StatusOK, nil)` (hàm này tự set `Content-Type` và ghi `200`). Header set trước `WriteHeader` nên không bị mất.
- `utils.WriteSuccessResponseWithStatus(w, http.StatusOK, nil)` in ra `{"success": true}` kèm `Content-Type: application/json` (`Data` là `omitempty`). Đúng định dạng envelope AC1 của api-response-envelope.
- **Cross-spec impact**: api-response-envelope §Ngoại lệ đang ghi preflight `OPTIONS` của `pkg/middleware.CORS` → `200` rỗng, và AC1 chỉ ràng buộc "every successful response **that has a body**". Sau AC4/AC10, preflight có body envelope — cần cập nhật `docs/specs/features/api-response-envelope/{requirements,design,tasks}.md` (xem `tasks.md` Phase 4).
- Thứ tự `Recovery → CORS → Logger` kéo theo AC9: preflight không được log vì `CORS` trả luôn. Nếu sau này cần log cả preflight thì đổi thứ tự `CORS` vào trong `Logger` — hành vi đổi, sẽ là spec riêng.
- `Logger` dùng `log.Printf` toàn cục, không cùng định dạng JSON với `logger` của service (`pkg/logging`) — chấp nhận trong phạm vi này; ghi ở `pkg/middleware/README.md` §Lưu ý.
