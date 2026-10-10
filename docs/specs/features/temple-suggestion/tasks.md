# Tasks — temple-suggestion

Version: 2.2
Last updated: 2026-10-10
Status: not started — spec viết trước khi có code; mọi mục `[ ]`. Không còn câu hỏi mở; hành động từ chối ở Phase 6 (làm sau).
Module: `content` + `backoffice`

> See `changelog.md` in this folder for the version history of this spec.

Mỗi mục truy về một dòng trong bảng "Layers & components" hoặc một Step trong `design.md`. Làm theo `golang-clean-arch-tdd` (test viết cùng code). Thứ tự khuyến nghị: Phase 1–2 (gửi đề xuất) trước, rồi 3–4 (duyệt).

## Phase 1 — Migration & domain

- [ ] `migrations/0022_temple_suggestions.sql` — bảng (`status` gồm `rejected`, cột `reject_reason`), CHECK `chk_temple_suggestions_review`, unique index một chùa một đề xuất, index hàng chờ; cập nhật `migrations/README.md`
- [ ] Cập nhật `docs/specs/db/erd.mmd` và sinh lại `erd.svg` bằng `mmdc`
- [ ] appService: `domain/temple_suggestion.go` (entity, hằng trạng thái (`pending`/`created`/`rejected`), `TempleSuggestionValidationError`, interface `TempleSuggestionRepository.Create`) + `domain/mock_temple_suggestion_repository.go`
- [ ] backofficeService: `domain/temple_suggestion.go` (entity đủ field, `NewTemple`, sentinel errors gồm `ErrReopenForbidden`, interface `List`/`FindByID`/`CreateTemple`) + mock

## Phase 2 — Gửi đề xuất (appService)

- [ ] `usecase/temple_suggestion_usecase.go`: trim, rỗng, độ dài 255/100/500 theo ký tự Unicode (AC2–AC3, AC5); unit test table-driven (`testify`) với mock
- [ ] `repository/postgres_temple_suggestion_repository.go`: `Create` (`INSERT … RETURNING id`); test repository trên Postgres thật
- [ ] `delivery/http/temple_suggestion_handler.go`: DTO, `DecodeJSONBodyStrict`, `201 {id, status}`; handler test (thành công, từng nhánh 400, body hỏng)
- [ ] `cmd/main.go`: đăng ký `POST /v1/temple-suggestions` qua `webAPIKey`; kiểm `main_test.go` wiring
- [ ] Rate-limit (AC7): config `temple_suggestion.rate_limit` (dev 5/giờ/IP), wire `pkg/ratelimit/middleware` theo IP client (lấy IP thật qua header proxy của Cloud Run), test 429 và cửa sổ hết hạn; không lưu IP
- [ ] Đảm bảo không có route client nào đọc đề xuất (AC8) — E2E kiểm `GET /v1/temple-suggestions` → 404/405

## Phase 3 — Duyệt: xem hàng chờ (backofficeService)

- [ ] `delivery/http/errors.go`: `RequireAnyRole(roles ...string)` + test (admin, content_editor được; viewer 403)
- [ ] `usecase/temple_suggestion_usecase.go`: `List` (status mặc định `pending`; nhận `pending|created|rejected`; `INVALID_STATUS`, `INVALID_PAGE`, paging), `Get`; unit test
- [ ] `repository/postgres_temple_suggestion_repository.go`: `List` (count + page, `ORDER BY created_at, id`), `FindByID`; test repository
- [ ] `config/config.go` + `config.dev.yaml`: `temple_suggestion.page_size`
- [ ] Handler `List`/`Get` + test (200, 400, 401, 403, 404)

## Phase 4 — Duyệt: tạo chùa (backofficeService)

- [ ] `backofficeService/domain`: `HeritagePriority` (bản chép, N7) + `heritage_priority_sync_test.go` đối chiếu với `appService/domain.HeritagePriority`; N6 đã chốt `temple_<region>_<NNN>`
- [ ] `usecase`: `CreateTemple` — validate `NewTemple` AC18–AC20, suy `heritage_priority`; nhánh `rejected` chỉ cho `admin` (AC30); unit test từng AC
- [ ] `repository`: `CreateTemple` — transaction C-3..C-6 (`FOR UPDATE`, advisory lock, `INSERT temples`, `UPDATE temple_suggestions` xoá `reject_reason`); test trên Postgres thật gồm: rollback giữ `pending`/`rejected`, hai request song song chỉ một thắng (AC22), mở lại từ `rejected`, CHECK của DB
- [ ] Handler `CreateTemple` + test (201, 400, 403 mở lại bởi content_editor, 404, 409, 500 không lộ nguyên nhân)
- [ ] `cmd/main.go`: 3 route `/v1/admin/temple-suggestions…` qua middleware + role; kiểm wiring
- [ ] Cập nhật `docs/specs/api/openapi.yaml` (+ bản `web/docs/public/`) cho 4 route của bản đầu

## Phase 5 — Kiểm chứng (bản đầu)

- [ ] E2E (`golang-e2e-verification`): binary thật + testcontainers Postgres — gửi đề xuất → admin list thấy → create-temple → `GET /v1/temples/{id}` thấy chùa mới → create lần 2 trả 409; route duyệt 401/403 đúng
- [ ] Audit AC1–AC27, AC30–AC31 ↔ test
- [ ] `go test -short ./...` và `go vet ./...` sạch
- [ ] Ghi vào changelog concept OKF cần tạo/làm mới (`docs/knowledge/features/`) — để `il-okf-writer` làm sau khi code đã kiểm chứng

## Phase 6 — Từ chối đề xuất (làm sau, hợp đồng API thiết kế khi đó)

- [ ] Chốt route và body của hành động từ chối (route tạm trong `requirements.md`)
- [ ] `usecase`: `Reject` — `reason` tuỳ chọn ≤500 ký tự (AC28–AC29); unit test
- [ ] `repository`: `Reject` — `UPDATE … WHERE id=$1 AND status='pending' RETURNING`; 0 dòng thì phân biệt 404/409; test trên Postgres thật
- [ ] Handler + route `POST …/reject` qua middleware + `RequireAnyRole`; test 200/400/401/403/404/409
- [ ] Sentinel `ErrSuggestionAlreadyHandled`; cập nhật `openapi.yaml`; thêm E2E: gửi → reject → list `rejected` → admin tạo chùa từ đề xuất đã từ chối

## Deferred (separate feature, not blocking temple-suggestion)

- Upload ảnh thật qua GCS (theo dõi như `docs/tech-debt/memorial/TD-001`)
- Bảng lịch sử duyệt (giữ lý do từ chối cũ khi mở lại)
