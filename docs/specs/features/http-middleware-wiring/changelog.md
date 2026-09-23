# Changelog — http-middleware-wiring Spec

Tracks version history for `requirements.md`, `design.md`, and `tasks.md` in this folder. All three
files share one changelog since they change together as one spec unit. Newest version first.

## v1.1 (implemented, 2026-09-22)

Implement xong AC1–AC11. Không đổi nội dung AC/design đã chốt ở v1.0, chỉ chốt trạng thái + sửa một chi tiết chữ ký.

- `pkg/middleware/middleware.go`: `CORS` nhánh `OPTIONS` → `utils.WriteSuccessResponseWithStatus(w, 200, nil)`.
- `services/appService/cmd/main.go`: tách `newHandler(mux *http.ServeMux) http.Handler` = `Recovery(CORS(Logger(WithAPIFallback(mux))))`; `main` gọi `newHandler(mux)`.
- Test: `pkg/middleware/middleware_test.go` thêm `TestCORS_PreflightIsEnvelope`, `TestCORS_NonOptionsPassesThrough`, `TestCORS_NonOptionsSetsHeaders`; `services/appService/cmd/main_test.go` mới phủ AC1/AC3/AC4/AC6/AC7/AC8/AC9/AC10.
- Docs: `pkg/middleware/README.md`; cross-spec `api-response-envelope` (v1.2 — bỏ ngoại lệ preflight); `docs/specs/api/openapi.yaml` v3.9 (mô tả CORS/preflight).
- Reason: yêu cầu "wire thêm 2 middleware logger và cors" — hoàn tất implement.
- Ghi chú: `newHandler` nhận `*http.ServeMux` (design v1.0 phác `http.Handler`) vì `WithAPIFallback` yêu cầu `*http.ServeMux`.

### Verify

- `go build ./...`, `go vet ./pkg/middleware/... ./services/appService/cmd/...`, `go test -short -count=1 ./...` đều xanh.
- `scripts/check-architecture.sh` = 0 lỗi; `scripts/check-api-standards.sh` còn 1 warning `HARDCODED_CONFIG` có trước ở `services/appService/e2e/harness.go`.
- Chạy binary thật + Postgres Docker (curl OPTIONS/404/400) chưa thực hiện trong phiên này — hành vi đã được phủ bằng unit test.

## v1.0 (initial draft, 2026-09-22)

Spec mới, **chưa implement**. Wire hai middleware sẵn có nhưng chưa dùng trong `pkg/middleware` — `Logger` và `CORS` — vào
composition root `services/appService/cmd/main.go`, kèm một sửa nhỏ nhánh preflight của `CORS`.

Hướng đã chốt (2026-09-22):

1. Dùng `pkg/middleware.Logger` thô (không dùng `pkg/logging/middleware.HTTPLogger`).
2. Giữ `pkg/middleware.CORS` hardcode `Access-Control-Allow-Origin: *`, không làm cấu hình được.
3. Thứ tự chain: `Recovery → CORS → Logger → WithAPIFallback → mux`.
4. Spec đặt tại `docs/specs/features/http-middleware-wiring/`.
5. Viết đủ 4 file `requirements.md` + `design.md` + `tasks.md` + `changelog.md`.
6. Preflight `OPTIONS` trả envelope `{"success": true}` kèm `200` (đổi so với `200` rỗng hiện tại).

- `requirements.md`: bằng chứng trước khi wire; User Story; AC1–AC11 (EARS); ngoại lệ (redirect `307`); out of scope (CORS cấu hình được, structured logging, gRPC, rate limit/metrics/tracing, server timeout); Pending về status code trong log.
- `design.md`: bảng Layers (`pkg/middleware/middleware.go`, `main.go`, tests, README, cross-spec api-response-envelope); request flow; lý do thứ tự chain; test plan theo AC.
- `tasks.md`: Phase 1 (`pkg/middleware`) – Phase 2 (composition root + test) – Phase 3 (verify + chạy binary thật) – Phase 4 (tài liệu); Deferred 3 mục.
- Reason: yêu cầu "wire thêm 2 middleware logger và cors" của anh; xác nhận qua 6 câu hỏi chốt hướng.

### Thay đổi hành vi quan sát được ở client appService (khi implement)

- Mọi response có thêm `Access-Control-Allow-Origin/Methods/Headers`.
- Preflight `OPTIONS`: `200` rỗng → `200` `application/json` `{"success": true}`.
- Mọi request không-phải-preflight có thêm 2 dòng log stdout (`Started …`, `Completed in …`). Preflight không được log.
- Status/body của mọi route hiện có (US-001 / US-002) không đổi.

### Cross-file follow-ups

- `docs/specs/features/api-response-envelope/` — cập nhật §Ngoại lệ (preflight không còn `200` rỗng). Chưa sửa ở v1.0, là task Phase 4.
- `pkg/middleware/README.md` — bỏ câu "`Logger`/`CORS` chưa có nơi nào dùng", mô tả preflight envelope.
- `docs/specs/api/openapi.yaml` — tuỳ chọn, mô tả preflight/CORS.
- `openwiki/` — generated, không sửa tay; sẽ refresh theo workflow.
