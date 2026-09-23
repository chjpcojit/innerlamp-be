# Design — http-middleware-wiring

Version: 1.1
Last updated: 2026-09-22
Status: implemented (2026-09-22) — bảng Layers khớp code thật.
Module: hạ tầng dùng chung (`pkg/middleware`) + composition root `services/appService/cmd`
Feature folder: `pkg/middleware/` + `services/appService/` + `docs/specs/features/http-middleware-wiring/`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Hai middleware cần dùng đã tồn tại sẵn trong `pkg/middleware/middleware.go` — `Logger` và `CORS` — nên thay đổi chính nằm ở **composition root**: đổi một dòng lắp handler trong `services/appService/cmd/main.go` để chèn hai middleware vào chain. Không thêm middleware, không thêm package, không thêm config.

Ngoại lệ duy nhất phải động vào code logic: nhánh preflight `OPTIONS` của `CORS` hiện trả `200` rỗng; AC4/AC10 đổi nó sang `utils.WriteSuccessResponseWithStatus(w, 200, nil)`. `pkg/middleware` vốn đã import `pkg/utils` (cho `Recovery`), nên không phát sinh import mới.

Phạm vi code rất nhỏ nhưng tác động quan sát được ở mọi request: mọi response (kể cả preflight) có header CORS, mọi request không-phải-preflight có 2 dòng log.

## Layers & components

| Layer | File | Thay đổi |
|---|---|---|
| Shared / middleware | `pkg/middleware/middleware.go` | Nhánh `r.Method == "OPTIONS"` trong `CORS`: thay `w.WriteHeader(http.StatusOK)` bằng `utils.WriteSuccessResponseWithStatus(w, http.StatusOK, nil)` — AC4, AC10. `Logger` và `Recovery` **không đổi**. |
| Shared / tests | `pkg/middleware/middleware_test.go` | Thêm test cho `CORS`: preflight trả `200` + `Content-Type: application/json` + `{"success": true}` + header `Access-Control-*`; request không phải `OPTIONS` đi tiếp và giữ nguyên status/body của handler sau — AC3, AC4, AC5. |
| Composition root | `services/appService/cmd/main.go` | Tách `newHandler(mux *http.ServeMux) http.Handler` trả `httpmiddleware.Recovery(httpmiddleware.CORS(httpmiddleware.Logger(deliveryhttp.WithAPIFallback(mux))))`; `main` gọi `newHandler(mux)` trong `ListenAndServe(addr, newHandler(mux))` — AC1. Cập nhật comment giải thích thứ tự. Việc tách hàm để AC1 test được **chính wiring thật**. (`*http.ServeMux` chứ không phải `http.Handler` vì `WithAPIFallback` nhận `*http.ServeMux`.) |
| Composition root / tests | `services/appService/cmd/main_test.go` (mới) | Test `newHandler` trên mux giả có route panic, route khớp, và route không khớp: panic vẫn 500 envelope (AC8), 404/405 vẫn envelope (AC7), response route khớp có header CORS (AC3, AC6), `OPTIONS` trả envelope và không vào mux (AC4, AC10). |
| Docs | `pkg/middleware/README.md` | §Cấu hình/dependency: bỏ câu "`Logger`/`CORS` … chưa có nơi nào dùng"; mô tả `CORS` preflight trả envelope; ghi thứ tự chain đang dùng ở appService. |
| Docs / cross-spec | `docs/specs/features/api-response-envelope/{requirements,design,tasks}.md` | Cập nhật §Ngoại lệ: preflight `OPTIONS` không còn là `200` rỗng — trỏ sang spec này. Không đổi AC1 (vẫn "response that has a body"). |
| Docs | `docs/specs/api/openapi.yaml` | Chỉ cần nếu muốn mô tả response preflight `OPTIONS`/CORS — hiện openapi chưa liệt kê preflight; ghi nhận là follow-up tuỳ chọn, không bắt buộc cho AC nào. |

Dependency rule: `pkg/middleware` vẫn là package dùng chung, không thêm import nào mới (`pkg/utils` đã có sẵn). `delivery → usecase → domain` không đổi. `services/appService/cmd` tiếp tục là nơi duy nhất lắp chain.

## Request flow

```
client
  └─ Recovery                       (ngoài cùng — bắt panic của mọi tầng dưới)
       └─ CORS                      (set Access-Control-*; OPTIONS → 200 envelope, DỪNG)
            └─ Logger                (Started "<METHOD>" "<PATH>" ... Completed in <dur>)
                 └─ WithAPIFallback
                      └─ ServeMux ─┬─ route khớp      → (jwt middleware) → handler
                                   ├─ không khớp      → 404/405 → fallbackWriter → envelope
                                   └─ path cần clean  → 307 redirect (giữ nguyên)
panic ở bất kỳ tầng trong CORS/Logger/fallback/handler → Recovery → 500 INTERNAL_ERROR
```

Hệ quả của thứ tự (AC9): preflight `OPTIONS` do `CORS` trả ngay nên **không** chạy qua `Logger` ⇒ không có dòng log cho preflight.

## Quyết định

- **`Recovery` ngoài cùng**: giống api-response-envelope, để bắt panic xảy ra ngay cả trong `CORS`/`Logger` (AC8). Đặt `Logger` ngoài `Recovery` sẽ bỏ lọt panic của chính `Recovery` và làm log dễ hiểu sai.
- **`CORS` ngoài `Logger`**: preflight phải được set header và trả ngay; đặt `Logger` ngoài `CORS` sẽ log cả preflight (`CORS` chạy sau) — đổi hành vi so với lựa chọn đã chốt. Trade-off: preflight không xuất hiện trong log (AC9).
- **Preflight trả envelope** thay vì `200` rỗng: mọi response có body trong appService dùng một envelope; AC10 buộc đi qua `utils.WriteSuccessResponseWithStatus` để giữ "một nguồn ghi response" (api-response-envelope §Overview). Điều này **thay đổi một ngoại lệ đã ghi** ở api-response-envelope → cross-spec follow-up ở trên.
- **Giữ `CORS` hardcode `*`** (AC11): whitelist origin/credentials là việc của spec cấu hình riêng, kèm sửa `AppConfig` + `config.dev.yaml`; gộp vào đây sẽ làm spec wiring đội scope.
- **Dùng `pkg/middleware.Logger` thô** thay vì `pkg/logging/middleware.HTTPLogger(logger)`: quyết định có chủ đích — giữ phạm vi "wire hai middleware đang có". Cái mất là structured log, correlation ID, status/size; `pkg/middleware/README.md` đã cảnh báo hai logger sẽ không cùng định dạng. Nếu cần, đó là spec riêng.
- **Không thêm config**: `Logger`/`CORS` hiện không nhận tham số; thêm bây giờ sẽ kéo theo đổi chữ ký public trong `pkg/middleware`.
- **Tách `newHandler(mux)` trong `main.go`**: để test AC1 khẳng định được **wiring thật** (hàm mà `main` dùng) thay vì một chain mô phỏng trong test — đúng loại lỗi composition root (route / middleware lắp sai) mà test tầng dưới không thấy. Hàm thuần tuý, không cần DB/config nên `cmd/main_test.go` chạy được trong `go test -short`.

## Test plan

| AC | Test | Kiểu |
|---|---|---|
| AC1 | `cmd/main_test.go`: gọi `newHandler(mux)` — chính là handler `main` truyền cho `ListenAndServe` — và chứng minh thứ tự qua hiệu ứng: panic → 500; preflight → không vào mux; route thường → mux được gọi | unit (`httptest`) |
| AC2 | `CORS → Logger → handler` ghi ra buffer log; assert có line chứa method + path trước, line chứa duration sau | unit (chuyển hướng `log.SetOutput`) |
| AC3 | Mọi nhánh (`route khớp`, `404`, `405`, `OPTIONS`) đều có 3 header `Access-Control-*` | unit |
| AC4 | `OPTIONS /bất-kỳ` → `200`, `Content-Type: application/json`, body `{"success": true}`, handler sau **không** được gọi | unit |
| AC5 | `POST` đi tiếp; status/body handler sau không đổi; header CORS vẫn có | unit |
| AC6 | Regression của US-001/US-002: `go test ./services/...` vẫn xanh; không test cũ nào đổi | regression |
| AC7 | `GET /v1/nope` qua chain → `404 NOT_FOUND` + vẫn có log | unit |
| AC8 | Handler panic qua chain `Recovery(CORS(Logger(...)))` → `500 INTERNAL_ERROR` envelope, kèm `Access-Control-Allow-Origin: *` | unit |
| AC9 | Chain `CORS → Logger`: `OPTIONS` không sinh log line nào | unit |
| AC10 | Preflight body parse ra `utils.ApiResponse{Success:true}`, không phải rỗng | unit |
| AC11 | Không có key CORS/logging mới trong `AppConfig`/`config.dev.yaml` — grep/không đổi file config | review |

## Deliberately stubbed for now

- Không có.
