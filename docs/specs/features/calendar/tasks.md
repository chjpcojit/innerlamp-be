# Tasks — calendar

Version: 3.2
Last updated: 2026-10-17
Status: Phase 1–18 đã làm (v3.0 xong ở v3.2); còn bước đối chiếu GHPGVN, `docs/knowledge/features/calendar.md` và cột Backend của product spec (`ba-writer`)
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

- [x] `cmd/main.go`: dựng repository/usecase/handler; `GET /v1/calendar/events` qua `authMiddleware.AuthenticateFunc` (E4 của v2.x; v3.0 đổi ở Phase 17)
- [x] `services/appService/e2e/`: kiểm tra qua binary thật — 401 không token / sai loại token, `INVALID_YEAR`, năm hợp lệ trả đủ `groups/events/occurrences`, Cache-Control hai trường hợp (skill `golang-e2e-verification`)
- [x] `docs/specs/api/openapi.yaml`: thêm `/v1/calendar/events`, schema, `INVALID_YEAR`
- [x] `api-contract.md`: thay ví dụ minh hoạ bằng response lấy từ server thật
- [x] `go build ./... && go vet ./... && go test -short ./...`; chạy server thật + `curl` kiểm E1–E17
- [x] Cập nhật `docs/specs/features/memorial/requirements.md`: bỏ/sửa câu "server không biết lịch âm" (xem follow-up changelog)

## US-CAL-002 v3.0 — theo product spec `calendar` v1.1 *(done v3.2)*

Thứ tự theo `design.md` v3.0. Viết test cùng lúc (`golang-clean-arch-tdd`).

### Phase 14 — Migration + domain + repository: `is_major`

- [x] `migrations/0039_calendar_events_is_major.sql` (`0037`/`0038` đã dùng); `migrations/README.md`; `docs/specs/db/erd.mmd` (`erd.svg` chưa vẽ lại) (E19, E24)
- [x] `domain/calendar_event.go`: `CalendarEvent.IsMajor` (E13, E19)
- [x] `repository/postgres_calendar_event_repository.go`: `ListActiveEvents` đọc `is_major`; `ReplaceCatalogue` ghi đè `is_major` (E24)
- [x] `repository/postgres_calendar_event_repository_test.go` (với `TEST_DB_NAME=innerlamp_test`): `is_major` được lưu, ghi đè khi seed lại, mặc định `false`

### Phase 15 — Seed CLI

- [x] `cmd/seedcalendarevents/parse.go`: khoá `is_major` tuỳ chọn; `CalendarCatalogue.Verify()` thêm luật `lunar_monthly` có `sort_order` lớn nhất (E24, E25)
- [x] `cmd/seedcalendarevents/parse_test.go`: thiếu `is_major` = `false`; `is_major` không phải bool bị từ chối; Rằm có `sort_order` nhỏ hơn một sự kiện có tên bị từ chối

### Phase 16 — Dữ liệu seed vào git

- [x] `.gitignore`: không cần ngoại lệ, file đã được git theo dõi (E26)
- [x] `assets/data/calendar-events.json`: lấy từ `/mnt/project-files/data/calendar/calendar-events.json`; Rằm / Mùng 1 `sort_order` 52 / 53 và dồn các sự kiện có tên về 1…51 giữ thứ tự cũ; `guidance` của mọi sự kiện viết lại một việc một dòng; `is_major: true` cho 10 sự kiện của BR-CAL-10; `meta.version` mới; anh Đức duyệt nội dung trong PR (BR-CAL-05, BR-CAL-10, BR-CAL-14)
- [x] `-dry-run` trên file mới, rồi `make seed-calendar-events` ở DB dev; kiểm lại `rules/ordering.json` và `rules/home-block.json` khớp dữ liệu thật
- [x] Giữ nhãn `DRAFT` tới khi có người đối chiếu GHPGVN (⚠️ ở `requirements.md`; việc đối chiếu vẫn mở)

### Phase 17 — Config, delivery, wiring: route công khai

- [x] `config/config.go` + `config.dev.yaml`: `calendar.events_rate_limit` (`max`, `window`, `trust_forwarded_for`), bắt buộc; cập nhật `config_test.go` (E23)
- [x] `delivery/http/calendar_event_handler.go`: DTO thêm `is_major`, `sort_order`, `recurrence`; `Cache-Control` `public, max-age` khi có `year` (E13, E17)
- [x] `delivery/http/calendar_event_handler_test.go`: ba trường mới; header `public`
- [x] `cmd/main.go`: `calendarEventLimit` (`rlScope("calendar_events:ip")`); route qua `webAPIKey` + `calendarEventLimit`, bỏ `authMiddleware`; sửa comment (E4, E22, E23)
- [x] Bỏ qua `calendarEventLimit` khi `X-API-Key` đúng: middleware cần biết Step 1a đã khớp khoá (cờ trong context do `RequireWebAPIKey` đặt, hoặc một wrapper riêng cho route này); test cả hai nhánh (E23 v3.1)
- [x] `services/appService/e2e/calendar_event_e2e_test.go`: thay case E4 cũ; thêm không token → 200, sai `X-API-Key` → 401, vượt hạn mức không có key → 429, có key đúng thì không bị 429, `Cache-Control: public`, `is_major` (skill `golang-e2e-verification`)

### Phase 18 — Test vectors, contract, kiểm tra

- [x] `domain/calendar_event_vectors_test.go`: chạy mọi case của `docs/product/calendar/rules/event-placement.json` và `rules/ordering.json`, đọc thẳng từ `docs/product/` (E27)
- [x] `docs/specs/api/openapi.yaml`: `/v1/calendar/events` bỏ `security`, thêm `X-API-Key` tuỳ chọn, `401`, `429`; `CalendarEvent` thêm `is_major`, `sort_order`, `recurrence`; mô tả `Cache-Control: public`
- [x] `go build ./... && go vet ./... && go test -short ./...` + `go test ./services/appService/e2e/`; chạy server thật + `curl` không token
- [ ] Cập nhật cột Backend của product spec §4.2 (FR-CAL-001, BR-CAL-10 → `done`) qua `ba-writer`
- [ ] `docs/knowledge/features/calendar.md`: gọi `il-okf-writer` sau khi kiểm chứng (khi anh yêu cầu)

## Deferred

- Thumbnail ảnh ngày (chờ iOS xác nhận có cần).
- Ảnh theo ngày lễ cho route ảnh (nay đã có lịch âm ở BE; cần quyết định riêng nếu muốn).
- Back-office quản lý sự kiện / nhóm / cờ nghỉ lễ (SRS riêng).
- Ảnh cho sự kiện (nguồn, giấy phép, nơi lưu).
- Bảng dịch tên nhóm và `?lang=` cho API.
- Cache kết quả theo `(year, version danh mục)` nếu đo thấy cần.
- Guard chặn chạy seed đè dữ liệu BO ở môi trường có BO.
