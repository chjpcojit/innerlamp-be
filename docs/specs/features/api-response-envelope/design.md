# Design — api-response-envelope

Version: 1.2
Last updated: 2026-09-22
Status: implemented (2026-09-21) — bảng Layers khớp code thật; verify bằng unit test + chạy binary thật với Postgres (curl 404/405/401/400). v1.2 chỉ là cross-spec follow-up từ `http-middleware-wiring` (preflight `OPTIONS` có body envelope), code đã verify ở spec đó.
Module: hạ tầng dùng chung + `services/appService/cmd`
Feature folder: `pkg/` + `services/appService/` + `docs/specs/features/api-response-envelope/`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Một nguồn duy nhất ghi response: các hàm `utils.Write*` (`pkg/utils/http.go`). Mọi nơi khác từng tự ghi body (`http.Error`, `w.Write([]byte(...))`, `json.NewEncoder(w)`, fallback của `ServeMux`) được chuyển sang gọi các hàm này. Không thêm hàm exported mới vào `pkg/utils`; không đổi chữ ký public nào của `pkg/jwt/middleware`, `pkg/ratelimit/middleware`, `pkg/middleware`.

## Layers & components

| Layer | File | Thay đổi |
|---|---|---|
| Shared / envelope | `pkg/utils/http.go` | `WriteSuccessResponseWithStatus` / `WriteErrorResponse`: bỏ `http.Error` sau khi encode lỗi (status đã gửi, ghi thêm chỉ làm hỏng JSON) — AC9. Bỏ import `strconv`. |
| Shared / middleware | `pkg/jwt/middleware/http.go` | `handleError` mặc định → `utils.WriteBusinessError(401, "UNAUTHORIZED", message)`; `RequireRole` → `401 UNAUTHORIZED` / `403 FORBIDDEN` — AC6. `ErrorHandler` tuỳ biến vẫn được ưu tiên. |
| Shared / middleware | `pkg/ratelimit/middleware/http.go` | `429` → `utils.WriteBusinessError(429, "RATE_LIMITED", "too many requests", {"retry_after_seconds": N})`; header giữ nguyên — AC7. |
| Shared / middleware | `pkg/middleware/middleware.go` | `Recovery` → `utils.WriteSystemError("INTERNAL_ERROR", "internal server error")` — AC8. v1.2: nhánh preflight `OPTIONS` của `CORS` → `utils.WriteSuccessResponseWithStatus(w, 200, nil)` (`{"success": true}`) thay cho `200` rỗng; đây là thay đổi ở `http-middleware-wiring` (AC4/AC10), ghi chéo ở đây vì nó xoá ngoại lệ preflight của spec này. |
| Delivery | `services/appService/delivery/http/fallback.go` *(mới)* | `WithAPIFallback(mux)`: request không khớp route (`mux.Handler` trả pattern rỗng) chạy qua `fallbackWriter` nuốt status `404`/`405` + body text, rồi ghi envelope `NOT_FOUND` / `METHOD_NOT_ALLOWED` (type `business`). Header `Allow` do `ServeMux` set được giữ. Status khác (redirect `307`) đi thẳng — AC3, AC4, AC10. |
| Delivery | `services/appService/delivery/http/login_handler.go` | `Logout` thành công: `w.WriteHeader(204)` → `utils.WriteSuccessResponse(w, nil)` → `200` `{"success": true}` (`data` `omitempty`) — AC11. Status/body lỗi 401/500 không đổi. |
| Composition root | `services/appService/cmd/main.go` | `handler := httpmiddleware.Recovery(deliveryhttp.WithAPIFallback(mux))`; `ListenAndServe(addr, handler)` — AC5. `Recovery` bọc ngoài cùng để bắt cả panic trong fallback. |
| Examples / docs | `pkg/jwt/examples/http/main.go`, `pkg/metrics/examples/complete_example.go`, `pkg/ratelimit/examples/{login_limit,chained_limit}/main.go`, `pkg/secrets/examples/example_usage.go`, các `README.md` / `QUICK_START.md` / `CLOUD_RUN.md` liên quan | Code mẫu dùng `utils.Write*` để không dạy lại định dạng cũ. |

## Request flow

```
client → Recovery → WithAPIFallback → ServeMux ─┬─ route khớp → (jwt middleware) → handler → utils.Write*
                                                └─ không khớp → ServeMux 404/405 → fallbackWriter → utils.WriteBusinessError
panic ở bất kỳ tầng trong → Recovery → utils.WriteSystemError 500
```

Từ v1.2 chain thật ở `appService` rộng hơn: `Recovery → CORS → Logger → WithAPIFallback → ServeMux`. `CORS` đứng trước mọi thứ và tự trả preflight `OPTIONS` bằng `utils.WriteSuccessResponseWithStatus` — cùng một nguồn ghi envelope; xem `../http-middleware-wiring/design.md`.

Không có Step-by-step Flow: không có lời gọi ngoài hay side effect bền vững — mỗi nhánh ghi đúng một response.

## Quyết định

- **404/405 là `type: business`**, không phải `validation`: `validation` dành cho `400` input sai hình dạng; mọi 4xx khác trong repo (401, 409, 413, 429) đã là `business`.
- **Chặn bằng `mux.Handler(r)` rồi mới bọc writer**, thay vì đăng ký route bắt-hết `"/"`: route `"/"` khớp mọi method nên `GET /v1/auth/login` sẽ thành 404 thay vì 405, và làm mất header `Allow`.
- **`Recovery` của `pkg/middleware`** được dùng lại thay vì viết recovery riêng trong `delivery/http` — package đã tồn tại, chỉ cần sửa body.
- `pkg/jwt/middleware` giữ message phân biệt (`Token has expired`, …) ở nhánh mặc định như cũ; appService vẫn ghi đè bằng `WriteAuthError` để không lộ nguyên nhân (login design).

## Deliberately stubbed for now

- Không có.
