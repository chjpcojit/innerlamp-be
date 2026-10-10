# Tasks — temple-suggestion

Version: 2.4
Last updated: 2026-10-10
Status: Phase 1 và Phase 2 (gửi đề xuất) đã implement và kiểm chứng 2026-10-10 — unit, repository (Postgres thật), E2E (binary thật + Postgres) đều pass. Phase 3–5 chưa làm; hành động từ chối ở Phase 6 (làm sau). Không còn câu hỏi mở.
Module: `content` + `backoffice`

> See `changelog.md` in this folder for the version history of this spec.

Mỗi mục truy về một dòng trong bảng "Layers & components" hoặc một Step trong `design.md`. Làm theo `golang-clean-arch-tdd` (test viết cùng code). Thứ tự khuyến nghị: Phase 1–2 (gửi đề xuất) trước, rồi 3–4 (duyệt).

## Phase 1 — Migration & domain

- [x] `migrations/0022_temple_suggestions.sql` — bảng (`status` gồm `rejected`, cột `reject_reason`), CHECK `chk_temple_suggestions_review`, unique index một chùa một đề xuất, index hàng chờ; cập nhật `migrations/README.md`. Test repository thêm `0006_admins.sql` + `0022` vào `applyMigration` (FK tới `admins`)
- [x] Cập nhật `docs/specs/db/erd.mmd` và sinh lại `erd.svg` bằng `mmdc` (render thành công = cú pháp hợp lệ)
- [x] appService: `domain/temple_suggestion.go` (entity, hằng trạng thái (`pending`/`created`/`rejected`), `TempleSuggestionValidationError`, interface `TempleSuggestionRepository.Create`) + `domain/mock_temple_suggestion_repository.go`
- [x] backofficeService: `domain/temple_suggestion.go` (entity đủ field, `NewTemple`, sentinel errors gồm `ErrReopenForbidden`, interface `List`/`FindByID`/`CreateTemple(ctx, suggestionID, adminID, role, NewTemple, priority)`) + mock — `priority` do usecase suy ra rồi truyền xuống, repository không tự tính

## Phase 2 — Gửi đề xuất (appService)

- [x] `usecase/temple_suggestion_usecase.go`: trim, rỗng, độ dài 255/100/500 theo ký tự Unicode (AC2–AC3, AC5); unit test table-driven (`testify`) với mock
- [x] `repository/postgres_temple_suggestion_repository.go`: `Create` (`INSERT … RETURNING id`); test repository trên Postgres thật (gồm CHECK từ chối hàng duyệt dở)
- [x] `delivery/http/temple_suggestion_handler.go`: DTO, `DecodeJSONBodyStrict`, `201 {id, status}`; handler test (thành công, từng nhánh 400, body hỏng)
- [x] `cmd/main.go`: đăng ký `POST /v1/temple-suggestions` qua `webAPIKey` rồi rate-limit; `cfg.TempleSuggestion.Validate()` ngay sau `Load`. Wiring chứng minh bằng E2E (`main_test.go` chỉ kiểm chuỗi middleware tổng, không liệt kê route)
- [x] Rate-limit (AC7): config `temple_suggestion.rate_limit` (`max`, `window`, `trust_forwarded_for`; dev 5/1h/false), `delivery/http/client_ip_rate_limit.go` bọc `pkg/ratelimit/middleware` (sliding window, bộ đếm trong bộ nhớ nên mỗi instance một bộ đếm); IP lấy từ TCP peer, hoặc entry cuối của `X-Forwarded-For` khi `trust_forwarded_for`; test 429, bucket riêng theo IP, không tin XFF khi tắt, không lưu IP. Hết cửa sổ do `pkg/ratelimit/store` có test. **Production phải đặt `trust_forwarded_for: true` sau Cloud Run** (hiện chỉ có `config.dev.yaml`)
- [x] Đảm bảo không có route client nào đọc đề xuất (AC8) — E2E kiểm `GET`/`DELETE /v1/temple-suggestions` và `GET /v1/temple-suggestions/1` đều ≥ 400 trong envelope

## Phase 3 — Duyệt: xem hàng chờ (backofficeService)

- [ ] `delivery/http/errors.go`: `RequireAnyRole(roles ...string)` + test (admin, content_editor được; viewer 403)
- [ ] `usecase/temple_suggestion_usecase.go`: `List` (status mặc định `pending`; nhận `pending|created|rejected`; `INVALID_STATUS`, `INVALID_PAGE`, paging), `Get`; unit test
- [ ] `repository/postgres_temple_suggestion_repository.go`: `List` (count + page, `ORDER BY created_at, id`), `FindByID`; test repository
- [ ] `config/config.go` + `config.dev.yaml`: `temple_suggestion.page_size`
- [ ] Handler `List`/`Get` + test (200, 400, 401, 403, 404)

## Phase 4 — Duyệt: tạo chùa (backofficeService)

- [x] `backofficeService/domain`: `HeritagePriority` (bản chép, N7) + `heritage_priority_sync_test.go` đối chiếu với `appService/domain.HeritagePriority`; N6 đã chốt `temple_<region>_<NNN>`
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
