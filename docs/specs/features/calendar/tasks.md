# Tasks — calendar

Version: 2.1
Last updated: 2026-10-11
Status: US-CAL-001 và US-CAL-002 đã triển khai (Phase 1–13); dữ liệu seed là bản nháp chưa đối chiếu nguồn chính thức
Module: `calendar`

> See `changelog.md` in this folder for the version history of this spec.

## US-CAL-001 — ảnh theo ngày *(done)*

### Phase 1 — Domain

- [x] `domain/calendar.go`: `CalendarImage`, `ErrInvalidDate` (không tag JSON)

### Phase 2 — Repository (manifest parser) + Assets

- [x] `assets/embed.go`: nhúng `images/home-calendar/manifest.json`
- [x] `repository/calendar_manifest.go`: `ParseCalendarManifest` (AC10)
- [x] `repository/calendar_manifest_test.go`: bảng lỗi + manifest nhúng thật khớp file ảnh trên đĩa

### Phase 3 — Config + Usecase

- [x] `config/config.go` + `config.dev.yaml`: `CalendarConfig{Timezone, ImageBaseURL}`; cập nhật `config_test.go` nếu có kiểm tra khoá
- [x] `usecase/calendar_usecase.go`: `NewCalendarUsecase`, `ImageForDate` (AC1–AC5, AC10)
- [x] `usecase/calendar_usecase_test.go`: cùng ngày cùng ảnh, ngày liên tiếp khác ảnh, bỏ trống = hôm nay theo múi giờ, ngày sai, biên năm, năm < 1970, wrap

### Phase 4 — Delivery

- [x] `delivery/http/errors.go`: `ErrInvalidDate` → `400 INVALID_DATE`
- [x] `delivery/http/calendar_handler.go` (AC1, AC6–AC9)
- [x] `delivery/http/mock_calendar_usecase_test.go` + `calendar_handler_test.go`

### Phase 5 — Wiring & docs

- [x] `cmd/main.go`: parse manifest, dựng usecase/handler, route `GET /v1/calendar/image` công khai (AC8)
- [x] `docs/specs/api/openapi.yaml`: thêm route + schema
- [x] `api-contract.md` cho iOS (đã đối chiếu với server thật)
- [x] Chạy server thật + `curl` kiểm tra AC1–AC9; `go build ./... && go vet ./...`

## US-CAL-002 — sự kiện lịch *(done)*

Thứ tự theo `design.md`: logic thuần trước (không DB), rồi schema, repository, seed, usecase, delivery, wiring. Mỗi phase viết test cùng lúc (`golang-clean-arch-tdd`); đối chiếu E1–E21 ở `requirements.md`.

### Phase 6 — Lịch âm thuần (`domain/lunarcal`)

- [x] `domain/lunarcal/lunarcal.go`: `SolarToLunar`, `LunarToSolar`, `LeapMonth`, `DaysInLunarMonth`, `LunarMonths`; múi giờ UTC+7 là hằng có comment (E5, E7, E20)
- [x] `domain/lunarcal/solarterm.go`: `SolarTermDate(year, longitude)` — ngày dương (UTC+7) chứa thời điểm kinh độ đạt mốc (E10)
- [x] `domain/lunarcal/lunarcal_test.go`: bảng tham chiếu — mùng 1 Tết (vd 2024-02-10, 2025-01-29, 2026-02-17), năm có tháng nhuận (vd 2023 nhuận tháng 2, 2025 nhuận tháng 6), biên 1900 và 2100, tháng 29 vs 30 ngày, vòng khứ hồi `SolarToLunar(LunarToSolar(x)) == x` trên toàn 1900..2100; tiết khí đối chiếu bảng đã công bố (E20)

### Phase 7 — Domain: entity, quy tắc, resolver

- [x] `domain/calendar_event.go`: `CalendarEvent`, `RecurrenceRule` + kiểm tra tham số theo `kind`, `CalendarGroup`, `CalendarOccurrence`, `CalendarLunarDate` (tên khác `LunarDate` của genealogy), `ErrInvalidYear`, `ErrInvalidRecurrenceRule` (không tag JSON)
- [x] `domain/calendar_event_resolver.go`: `ResolveOccurrences` cho năm dương `Y`, năm âm `Y−1` và `Y` (E6–E12)
- [x] `domain/calendar_event_resolver_test.go`: mỗi `kind` happy path; 29/2 năm không nhuận bị bỏ; lễ âm tháng Chạp rơi vào tháng 1–2 dương (E11); `is_leap_month` có/không có tháng nhuận; `lunar_monthly` năm nhuận cho thêm một occurrence; `lunar_last_day` tháng 29 vs 30 ngày (E9); sắp xếp ổn định khi trùng ngày (E12); kind lạ bị bỏ, không panic

### Phase 8 — Migration + ERD

- [x] `migrations/0031_calendar_events.sql`: `calendar_groups`, `calendar_events` (CHECK theo `kind`), `calendar_event_translations`, `calendar_event_groups`; idempotent; ghi chú đầu file
- [x] `migrations/README.md`: thêm dòng `0031` (số `0030` đã bị `temple_suggestions` dùng)
- [x] `docs/specs/db/erd.mmd`: bốn bảng đã có (gỡ nhãn SPEC)
- [x] `docs/specs/db/erd.svg`: sinh lại bằng `@mermaid-js/mermaid-cli` (Chrome hệ thống)

### Phase 9 — Repository

- [x] `domain/calendar_event_repository.go` + `mock_calendar_event_repository.go`: interface `ListGroups`, `ListActiveEvents`, `ReplaceCatalogue`
- [x] `repository/postgres_calendar_event_repository.go`: `ListGroups`, `ListActiveEvents` (một SELECT, lọc `is_active`, `lang`), `ReplaceCatalogue` (một transaction, upsert, không DELETE) (E16, E18, E19)
- [x] `repository/postgres_calendar_event_repository_test.go`: chạy với `TEST_DB_NAME=innerlamp_test` (**không** chạy trần — xoá DB dev); upsert idempotent, ghi đè nội dung, rollback khi lỗi giữa chừng, sự kiện `is_active = false` không được trả, sự kiện vắng trong file không bị xoá

### Phase 10 — Seed CLI + dữ liệu

- [x] `cmd/seedcalendarevents/parse.go`: giải mã strict, `Verify` (E18), chuẩn hoá NFC nếu cần như `seedadminunits`
- [x] `cmd/seedcalendarevents/parse_test.go`: bảng lỗi — mã trùng, kind lạ, thiếu tham số, `solar_term_longitude` không bội của 15, nhóm không tồn tại, sự kiện không có nhóm, khoá lạ
- [x] `cmd/seedcalendarevents/main.go`: `-file`, `-dry-run`, kết nối DB, `ReplaceCatalogue`, báo cáo
- [x] `assets/data/calendar-events.json`: **team thu thập** — các sự kiện Phật giáo, Việt Nam, dân gian, mùng 1/Rằm, Giao thừa, tiết khí được chọn; mỗi sự kiện có `source`; ảnh để `null`; nội dung đầy đủ (tên, mô tả ngắn/dài, lễ nghi) tiếng Việt. Đối chiếu từng ngày trước khi nạp
- [x] Chạy `-dry-run` trên file thật, rồi nạp vào DB dev (`TEST_DB_NAME`/DB dev tách bạch)

### Phase 11 — Config + Usecase

- [x] `config/config.go` + `config.dev.yaml`: `CalendarConfig` thêm `DefaultLanguage`, `EventsCacheMaxAge`; rỗng/0 là lỗi; cập nhật `config_test.go`
- [x] `usecase/calendar_event_usecase.go`: `NewCalendarEventUsecase(repo, cfg, now)`, `EventsForYear` (E1–E3, E13–E15)
- [x] `usecase/calendar_event_usecase_test.go`: năm bỏ trống = năm hiện tại theo múi giờ; năm sai (`abc`, `2026.5`, `1899`, `2101`); biên 1900/2100; gom nhóm + sự kiện + occurrence; mã nhóm không có trong danh mục bị bỏ qua; lỗi repository truyền lên

### Phase 12 — Delivery

- [x] `delivery/http/errors.go`: `ErrInvalidYear` → `400 INVALID_YEAR`, `details.field = "year"` (E3)
- [x] `delivery/http/calendar_event_handler.go`: DTO (`lunar_date` object, `image_url` nullable), `Cache-Control` theo `year` rõ ràng hay bỏ trống (E17)
- [x] `delivery/http/mock_calendar_event_usecase_test.go` + `calendar_event_handler_test.go`

### Phase 13 — Wiring, E2E & docs

- [x] `cmd/main.go`: dựng repository/usecase/handler; `GET /v1/calendar/events` qua `authMiddleware.AuthenticateFunc` (E4)
- [x] `services/appService/e2e/`: kiểm tra qua binary thật — 401 không token / sai loại token, `INVALID_YEAR`, năm hợp lệ trả đủ `groups/events/occurrences`, Cache-Control hai trường hợp (skill `golang-e2e-verification`)
- [x] `docs/specs/api/openapi.yaml`: thêm `/v1/calendar/events`, schema, `INVALID_YEAR`
- [x] `api-contract.md`: thay ví dụ minh hoạ bằng response lấy từ server thật
- [x] `go build ./... && go vet ./... && go test -short ./...`; chạy server thật + `curl` kiểm E1–E17
- [x] Cập nhật `docs/specs/features/memorial/requirements.md`: bỏ/sửa câu "server không biết lịch âm" (xem follow-up changelog)

## Deferred

- Thumbnail ảnh ngày (chờ iOS xác nhận có cần).
- Ảnh theo ngày lễ cho route ảnh (nay đã có lịch âm ở BE; cần quyết định riêng nếu muốn).
- Back-office quản lý sự kiện / nhóm / cờ nghỉ lễ (SRS riêng).
- Ảnh cho sự kiện (nguồn, giấy phép, nơi lưu).
- Bảng dịch tên nhóm và `?lang=` cho API.
- Cache kết quả theo `(year, version danh mục)` nếu đo thấy cần.
- Guard chặn chạy seed đè dữ liệu BO ở môi trường có BO.
