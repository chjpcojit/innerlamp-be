# Tasks — temple-suggestion

Version: 3.4
Last updated: 2026-10-11
Status: **Phase 1–5, 7 và 8 đã implement và kiểm chứng 2026-10-11** — unit, repository (Postgres thật, gồm đua hai request và rollback), E2E của cả appService và backofficeService (binary thật + Postgres) đều pass. Còn lại: sinh lại `erd.svg` (máy chưa có `mmdc`/Chrome), ghi concept OKF (`il-okf-writer`), và Phase 6 (hành động từ chối, làm sau).
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

- [x] `delivery/http/errors.go`: `RequireAnyRole(roles ...string)` + test (admin, content_editor được; viewer 403)
- [x] `usecase/temple_suggestion_usecase.go`: `List` (status mặc định `pending`; nhận `pending|created|rejected`; `INVALID_STATUS`, `INVALID_PAGE`, paging), `Get`; unit test
- [x] `repository/postgres_temple_suggestion_repository.go`: `List` (count + page, `ORDER BY created_at, id`), `FindByID`; test repository
- [x] `config/config.go` + `config.dev.yaml`: `temple_suggestion.page_size`
- [x] Handler `List`/`Get` + test (200, 400, 401, 403, 404)

## Phase 4 — Duyệt: tạo chùa (backofficeService)

- [x] `backofficeService/domain`: `HeritagePriority` (bản chép, N7) + `heritage_priority_sync_test.go` đối chiếu với `appService/domain.HeritagePriority`; N6 đã chốt `temple_<region>_<NNN>`
- [x] `usecase`: `CreateTemple` — validate `NewTemple` AC18–AC20, suy `heritage_priority`; nhánh `rejected` chỉ cho `admin` (AC30); unit test từng AC
- [x] `repository`: `CreateTemple` — transaction C-4..C-7 (`FOR UPDATE`, advisory lock, `INSERT temples`, `UPDATE temple_suggestions` xoá `reject_reason`); test trên Postgres thật gồm: rollback giữ `pending`/`rejected`, hai request song song chỉ một thắng (AC22), mở lại từ `rejected`, CHECK của DB
- [x] Handler `CreateTemple` + test (201, 400, 403 mở lại bởi content_editor, 404, 409, 500 không lộ nguyên nhân)
- [x] `cmd/main.go`: 3 route `/v1/admin/temple-suggestions…` qua middleware + role; kiểm wiring
- [x] Cập nhật `docs/specs/api/openapi.yaml` (+ bản `web/docs/public/`) cho 4 route của bản đầu

## Phase 5 — Kiểm chứng (bản đầu)

- [x] E2E (`golang-e2e-verification`, `services/backofficeService/e2e/`): binary backoffice thật + testcontainers Postgres — hàng chờ (mã + đề xuất cũ), create-temple → chùa trả đúng bộ ba + `full_address` → create lần 2 trả 409, đua 4 request một thắng, mở lại `rejected` (admin ok, content_editor 403), 401/403 trên cả ba route. *Không* chứng minh `GET /v1/temples/{id}` của appService thấy chùa mới (hai binary khác nhau): AC25 dựa vào việc cả hai đọc cùng bảng `temples`, test repository xác nhận các cột được ghi
- [x] Audit AC1–AC27, AC30–AC31 ↔ test (AC9–AC15, AC18–AC24, AC26–AC27, AC30–AC31 có test; AC25 xem ghi chú ở mục E2E)
- [x] `go test -short ./...` và `go vet ./...` sạch
- [ ] Ghi vào changelog concept OKF cần tạo/làm mới (`docs/knowledge/features/`) — để `il-okf-writer` làm sau khi code đã kiểm chứng

## Phase 6 — Từ chối đề xuất (làm sau, hợp đồng API thiết kế khi đó)

- [ ] Chốt route và body của hành động từ chối (route tạm trong `requirements.md`)
- [ ] `usecase`: `Reject` — `reason` tuỳ chọn ≤500 ký tự (AC28–AC29); unit test
- [ ] `repository`: `Reject` — `UPDATE … WHERE id=$1 AND status='pending' RETURNING`; 0 dòng thì phân biệt 404/409; test trên Postgres thật
- [ ] Handler + route `POST …/reject` qua middleware + `RequireAnyRole`; test 200/400/401/403/404/409
- [ ] Sentinel `ErrSuggestionAlreadyHandled`; cập nhật `openapi.yaml`; thêm E2E: gửi → reject → list `rejected` → admin tạo chùa từ đề xuất đã từ chối

## Phase 7 — Địa chỉ có cấu trúc cho đề xuất (v3.0, N8)

Cascade: `requirements.md` AC1–AC3b, AC5, AC9, AC13 → `design.md` (Layers, Schema, Step S-1..S-4) → các mục dưới. Phase 2 đã `[x]` theo hợp đồng v2.x; mỗi mục dưới đây sửa lại phần đó nên đều `[ ]`.

- [x] `migrations/0030_temple_suggestions_address_codes.sql` — `ADD COLUMN IF NOT EXISTS province_code VARCHAR(16)`, `ward_code VARCHAR(16)`, `ward VARCHAR(255)` (nullable) + CHECK `chk_temple_suggestions_address_codes`; không FK, không backfill; cập nhật `migrations/README.md`, `docs/specs/db/erd.mmd`
- [ ] Sinh lại `docs/specs/db/erd.svg` bằng `mmdc` (máy này chưa cài `mmdc`; `erd.mmd` đã sửa)
- [x] appService `domain/temple_suggestion.go`: thêm `ProvinceCode`, `WardCode`, `Ward` vào entity; `TempleSuggestionInput{Name, ProvinceCode, WardCode, Address}`; port `WardResolver`; cập nhật mock
- [x] `domain/administrative_unit.go` + repository hành chính: `ResolveCurrentWard(ctx, provinceCode, wardCode)` (một SELECT join, `ErrAdministrativeUnitNotFound` khi không có/lệch tỉnh) + mock; test repository trên Postgres thật (khớp, xã thuộc tỉnh khác, mã hệ cũ không khớp)
- [x] `usecase/temple_suggestion_usecase.go`: bỏ kiểm `province` 100 ký tự; thêm kiểm rỗng `province_code`/`ward_code` và gọi `WardResolver` (AC2, AC3a, AC5; `details.field` đúng `province_code`/`ward_code`); unit test table-driven (hợp lệ, rỗng, mã không tồn tại, xã lệch tỉnh, tên chụp lấy từ dataset chứ không từ client)
- [x] `repository/postgres_temple_suggestion_repository.go`: `Create` ghi bốn cột mới; test trên Postgres thật (hàng mới có mã + tên chụp, CHECK từ chối mã đi lẻ)
- [x] `delivery/http/temple_suggestion_handler.go`: DTO request `{name, province_code, ward_code, address}`; test handler (201, từng nhánh 400, field `province` cũ bị từ chối — AC3b)
- [x] `cmd/main.go`: nối repository hành chính vào `NewTempleSuggestionUsecase`
- [x] Cập nhật `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`, `web/docs/src/content/docs/guides/catalogues/temple-suggestions.md`, changelog web) cho body mới và ví dụ lỗi `ward_code` lệch tỉnh
- [x] E2E (`golang-e2e-verification`): gửi với mã hợp lệ → 201; `province_code` lạ → 400; `ward_code` của tỉnh khác → 400; body cũ có `province` → 400
- [x] Backoffice Phase 3: `List`/`Get` trả `province_code`, `ward_code`, `province`, `ward` (null với dòng cũ — AC9, AC13); form `create-temple` tự điền sẵn từ các field này

## Phase 8 — Địa chỉ có cấu trúc trong `create-temple` (v3.2, N9)

Cascade: `requirements.md` AC18, AC18a–AC18b → `design.md` (Layers backoffice, Step C-3, C-6) → các mục dưới. Phase 4 là nơi `create-temple` được viết; các mục dưới đây bổ sung vào đó nên làm **cùng lúc hoặc sau** Phase 4, và phụ thuộc migration `0032` của `temple`. Đã làm cùng Phase 3–5 (v3.4).

- [x] Chờ `temple/tasks.md` Phase 10: migration `0032` đã tạo và áp (cột `province_code`, `ward_code`, `ward` của `temples`)
- [x] backofficeService `domain/temple_suggestion.go`: `NewTemple` thêm `ProvinceCode`, `WardCode`, `Province`, `Ward`; port `WardResolver` + `ResolvedWard` + sentinel `ErrAdministrativeUnitNotFound`/`ErrWardNotInProvince` (bản riêng, không import `appService/domain`); cập nhật mock
- [x] `usecase`: `CreateTemple` gọi `WardResolver` (Step C-3), lỗi → `TempleInputError{Field: province_code|ward_code}`; unit test: hợp lệ, rỗng, mã không tồn tại, xã lệch tỉnh, mã hệ cũ, tên chụp lấy từ dataset chứ không từ request (AC18a–AC18b)
- [x] `repository`: `ResolveCurrentWard` (một SELECT) + `CreateTemple` ghi bốn cột mới; test trên Postgres thật: mã hợp lệ → hàng chùa có đủ bộ ba, xã tỉnh khác → không tạo gì và đề xuất vẫn `pending`
- [x] Handler `CreateTemple`: DTO nhận `province_code`, `ward_code`, `address`; test 400 với `details.field` đúng
- [x] Response chùa của `create-temple` có `location.province_code`, `ward_code`, `ward`, `full_address` — cùng hình dạng `GET /v1/temples/{id}` (AC16); `FullAddress()` là bản chép tương đương của `appService/domain` (backoffice không import)
- [x] E2E: đề xuất có mã → create-temple → `GET /v1/temples/{id}` trả cùng `province_code`/`ward_code` và `full_address` đúng; đề xuất cũ (mã `NULL`) tạo chùa được khi request mang mã; mã sai → 400 và đề xuất vẫn `pending`
- [x] Cập nhật `docs/specs/api/openapi.yaml` (+ `web/docs/public/`) cho body `create-temple` (gộp vào mục openapi của Phase 4)

## Deferred (separate feature, not blocking temple-suggestion)

- Upload ảnh thật qua GCS (theo dõi như `docs/tech-debt/memorial/TD-001`)
- Bảng lịch sử duyệt (giữ lý do từ chối cũ khi mở lại)
- ~~Cột `province_code`/`ward_code` cho `temples`~~ — thuộc feature `temple` v1.7 (migration `0032`, `temple/tasks.md` Phase 10); phần ghi từ `create-temple` ở Phase 8 bên dưới
