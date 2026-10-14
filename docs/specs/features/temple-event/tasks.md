# Tasks — temple-event

Version: 1.1
Last updated: 2026-10-13
Status: Phase 1–6 và E2E đã code + test; còn kiểm tra `curl` bằng tay và `api-contract.md`
Module: `temple-event`

> See `changelog.md` in this folder for the version history of this spec.

Thứ tự theo `design.md`: logic thuần trước (không DB), rồi config, schema, repository, seed + dữ liệu, usecase, delivery, wiring. Mỗi phase viết test cùng lúc (`golang-clean-arch-tdd`); đối chiếu TE1–TE25 ở `requirements.md`. Đường dẫn tính từ `services/appService/` trừ khi có ghi khác.

## Phase 1 — Domain

- [x] `domain/temple_event.go`: `TempleEvent`, `TempleEventOccurrence`, `TempleEventWindow`, enum `TempleEventKind` / `TempleEventStatus` / `TempleRecurrence` (không tag JSON)
- [x] `domain/temple_event.go`: `ErrInvalidDateRange`, `ErrDateRangeTooLarge`, `ErrInvalidTempleEvent`; dùng lại `ErrInvalidDate`, `ErrTempleNotFound`
- [x] `domain/temple_event.go`: `TempleEventRepository{ListActiveByTemple(ctx, templeID, window), ReplaceCatalogue}` và `TempleEventCatalogue{Source, Events}`
- [x] `domain/temple_event.go`: `TempleEventCatalogue.Verify(loc) error` bọc `ErrInvalidTempleEvent`, nhận múi giờ cấu hình để đọc sự kiện cả ngày (TE24); `TempleEvent.Validate(loc)` dùng chung cho usecase/resolver; `InvalidDateFieldError{Field}` để `INVALID_DATE` nêu `from` hay `to`
- [x] `domain/mock_temple_event_repository.go`: mock viết tay cho test usecase

## Phase 2 — Resolver thuần

- [x] `domain/temple_event_resolve.go`: `ResolveTempleOccurrences(events, window, now, loc)` (TE8–TE14, TE20–TE23)
- [x] `domain/temple_event_resolve_test.go` (bảng, đồng hồ cố định): một lần, nhiều ngày, qua nửa đêm, cả ngày; weekly; lunar-monthly 14+30 và 15; tháng nhuận; ngày 30 trong tháng 29 ngày; 29+30 trong tháng 29 ngày ra một lần; `recurrence_until`; đã kết thúc bị bỏ, đang diễn ra được giữ; cửa sổ trong quá khứ; biên năm âm (29/12 ÂL rơi sang tháng 1 dương); thứ tự ổn định
- [x] `domain/temple_event_test.go`: bảng lỗi cho `Verify` (id trùng, kind/status lạ, thiếu tham số lặp, `ends_at` < `starts_at`, `until` trước ngày đầu, thời lượng lặp ≥ 24h, ngoài 1900..2100, cả ngày + lặp tuần)

## Phase 3 — Config

- [x] `config/config.go`: `TempleEventConfig{Timezone, DefaultWindowMonths, MaxWindowMonths}` + `Validate()` (múi giờ IANA hợp lệ, `1 ≤ default ≤ max`); thêm vào `Config` (`yaml:"temple_event"`)
- [x] `config/config.dev.yaml` (và các file env khác): `temple_event.timezone: Asia/Ho_Chi_Minh`, `default_window_months: 3`, `max_window_months: 3`
- [x] `config/config_test.go`: khoá bắt buộc, múi giờ sai, `default > max`
- [x] `cmd/main.go`: gọi `Validate()` lúc khởi động, `log.Fatalf` khi sai (fail fast)

## Phase 4 — Schema, repository, seed, dữ liệu

- [x] `migrations/0035_temple_events.sql`: bảng + CHECK (`kind`, `status`, `recurrence`, nhất quán tham số lặp, `ends_at >= starts_at`, `weekday` 0..6) + index; idempotent (`IF NOT EXISTS`); xác nhận số migration kế tiếp trước khi tạo
- [x] `repository/postgres_temple_event_repository.go`: `ListActiveByTemple` (cột liệt kê rõ, lọc thô theo Step 5), `ReplaceCatalogue` (một transaction, kiểm tra `temple_id`, upsert không ghi `is_active`); `%w` bọc lỗi
- [x] `repository/postgres_temple_event_repository_test.go`: **đặt `TEST_DB_NAME=innerlamp_test`** (test repository xoá DB dev); upsert hai lần không trùng; `is_active` do tay chỉnh được giữ; temple_id lạ rollback cả file; `ListActiveByTemple` lọc theo chùa, bỏ `is_active=false`, cắt theo khoảng thô
- [x] `cmd/seedtempleevents/main.go` + `parse.go` + `parse_test.go`: strict decode, `-dry-run` (mẫu `seedcalendarevents`)
- [x] `assets/data/temple-events.json`: dữ liệu mock cho ~6–8 chùa lấy id từ `assets/data/temple.json`; phủ mọi dạng: một lần, nhiều ngày, qua nửa đêm, cả ngày, weekly, lunar-monthly 14+30 và 15, `POSTPONED`, `CANCELLED`, có/không `recurrence_until`; `source` ghi rõ là mock
- [x] `Makefile`: `ensure-temple-events`, `seed-temple-events`, thêm vào `ensure-data` (mẫu `ensure-calendar-events`)
- [x] `docs/specs/db/erd.mmd`, `migrations/README.md`: thêm `temple_events` (`erd.svg` không vẽ lại: `erd.mmd` là nguồn)

## Phase 5 — Usecase

- [x] `usecase/temple_event_usecase.go`: `NewTempleEventUsecase(templeRepo, eventRepo, cfg, now, logger) (*Usecase, error)`, `EventsForTemple` (Step 3–6); bọc lỗi để `errors.Is` hoạt động
- [x] `usecase/temple_event_usecase_test.go`: cửa sổ mặc định, `from` có/`to` rỗng, định dạng sai, năm ngoài 1900..2100, `to < from`, quá `max_window_months`, chùa không tồn tại, chùa không có sự kiện, lỗi DB ở mỗi lần đọc, đồng hồ cố định cho "đã qua"

## Phase 6 — Delivery & wiring

- [x] `delivery/http/temple_event_handler.go`: parse `{id}`, `from`, `to`; DTO response; `Cache-Control: no-cache`
- [x] `delivery/http/errors.go`: `ErrInvalidDateRange` → `400 INVALID_DATE_RANGE`, `ErrDateRangeTooLarge` → `400 DATE_RANGE_TOO_LARGE`; kiểm tra `ErrInvalidDate` đã map đúng với `details.field`
- [x] `delivery/http/mock_temple_event_usecase_test.go` + `temple_event_handler_test.go`: 200, mỗi nhánh lỗi, DTO `null` cho `image_url`
- [x] `cmd/main.go`: dựng repo/usecase/handler; `mux.HandleFunc("GET /v1/temples/{id}/events", authMiddleware.AuthenticateFunc(templeEventHandler.Events))`; **comment nêu route này cần đăng nhập, khác các route `/v1/temples` công khai**
- [x] `docs/specs/api/openapi.yaml`: route + schema `TempleEventsResponse` / `TempleEvent` / `TempleOccurrence` + ba mã lỗi mới

## Phase 7 — Integration & documentation

- [x] `e2e/temple_event_e2e_test.go` (binary thật + Postgres thật, theo `golang-e2e-verification`): 401 không token; 404 chùa lạ; 200 rỗng; sự kiện lặp ra đúng ngày; cửa sổ quá dài 400; route thật sự có middleware
- [ ] Chạy server thật + `curl` kiểm tra TE1–TE23 với dữ liệu seed
- [x] `go build ./... && go vet ./... && go test -short ./...` sạch (test repository chỉ chạy với `TEST_DB_NAME=innerlamp_test`)
- [ ] `api-contract.md` cho iOS (đối chiếu với server thật)
- [x] Ghi vào `changelog.md` rằng `docs/knowledge/features/temple-event.md` cần tạo bằng `il-okf-writer` sau khi code được kiểm chứng

## Deferred (separate feature, not blocking temple-event)

- Admin/backoffice tạo, sửa, tắt sự kiện chùa và phân quyền (đại diện chùa).
- Bảng ngoại lệ `temple_event_exceptions` để hoãn/huỷ một lần của sự kiện lặp.
- Theo dõi chùa (follow temple), nhắc lịch, push notification, đăng ký tham dự trong app.
- Upload ảnh sự kiện (cần object storage — xem `docs/tech-debt/memorial/TD-001`).
- Đa ngôn ngữ cho nội dung sự kiện.
- Thay dữ liệu mock bằng nội dung thật do chùa cung cấp.
