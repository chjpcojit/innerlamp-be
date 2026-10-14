# Design — calendar

Version: 2.1
Last updated: 2026-10-11
Status: US-CAL-001 và US-CAL-002 đã triển khai; US-CAL-002 verified bằng test từng tầng và E2E (binary thật + Postgres thật). Dữ liệu seed là bản nháp chưa đối chiếu nguồn chính thức
Module: `calendar`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Module có hai route, hình dạng khác nhau:

- **`GET /v1/calendar/image`** (US-CAL-001, đã triển khai): công khai, chỉ đọc, **không có DB** — "repository" là bộ phân tích `manifest.json` nhúng sẵn, danh sách ảnh nằm trong bộ nhớ. Không đổi trong phiên bản này.
- **`GET /v1/calendar/events`** (US-CAL-002, mới): cần JWT `access`, chỉ đọc, **có DB** (bốn bảng `calendar_*`). Mỗi request đọc danh mục sự kiện (một vài dòng, đổi được qua BO) rồi **tính** ngày của năm được hỏi bằng bộ chuyển đổi âm ↔ dương và công thức tiết khí — kết quả không lưu. Một seed CLI nạp danh mục từ file JSON.

US-CAL-002 chạm đủ bốn layer của `services/appService`, thêm một package thuần tính toán `domain/lunarcal`, một migration dự kiến `migrations/0031_calendar_events.sql`, một block config và một lệnh `cmd/seedcalendarevents`. Hình dạng gần nhất: `administrative-units` / `temple` (danh mục đọc + nạp bằng CLI upsert).

## Layers & components

### US-CAL-001 — ảnh theo ngày *(implemented, không đổi)*

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/calendar.go` | `CalendarImage` (id, file, author, license, license_url, source_url), `ErrInvalidDate`. Không có tag JSON. |
| Repository (parser) | `services/appService/repository/calendar_manifest.go` | `ParseCalendarManifest([]byte)` đọc `manifest.json`, kiểm tra (không rỗng, tên file hợp lệ, không trùng, có giấy phép), sắp theo tên file. |
| Usecase | `services/appService/usecase/calendar_usecase.go` | `CalendarUsecase.ImageForDate(date string)`: bỏ trống → hôm nay theo `calendar.timezone`; kiểm tra ngày; chọn ảnh theo `dayNumber mod N`. Đồng hồ được tiêm vào để test. |
| Delivery | `services/appService/delivery/http/calendar_handler.go` | `GET /v1/calendar/image`; DTO riêng; dựng `image_url` tuyệt đối; đặt `Cache-Control`. |
| Delivery / errors | `services/appService/delivery/http/errors.go` | `ErrInvalidDate` → `400 INVALID_DATE`, `details.field = "date"`. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `CalendarConfig{Timezone, ImageBaseURL}`. |
| Assets | `assets/embed.go` | Nhúng `images/home-calendar/manifest.json` (`CalendarManifestJSON`). |

### US-CAL-002 — sự kiện lịch

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/calendar_event.go` | `CalendarEvent` (`Code`, `Name`, `ShortDescription`, `LongDescription`, `Guidance`, `ImageURL *string`, `Groups []string`, `IsPublicHoliday`, `SortOrder`, `Rule`), `RecurrenceRule` (`Kind` ∈ `solar` / `lunar` / `lunar_monthly` / `lunar_last_day` / `solar_term`; `SolarMonth`, `SolarDay`, `LunarMonth`, `LunarDay`, `IsLeapMonth`, `SolarTermLongitude`), `CalendarGroup` (`Code`, `Name`), `CalendarOccurrence` (`EventCode`, `SolarDate`, `LunarDate`), `LunarDate` (`Day`, `Month`, `Year`, `IsLeapMonth`), `ErrInvalidYear`. Không có tag JSON. |
| Domain / Repository interface | `services/appService/domain/calendar_event_repository.go`, `mock_calendar_event_repository.go` | `CalendarEventRepository`: `ListActiveEvents(ctx, lang)`, `ListGroups(ctx)`, `ReplaceCatalogue(ctx, Catalogue)` (seed). Mock theo mẫu `mock_administrative_unit_repository.go`. |
| Domain / Pure logic | `services/appService/domain/lunarcal/` | Không phụ thuộc gì ngoài `math`/`time`. `SolarToLunar(y,m,d)`, `LunarToSolar(lunarYear, month, day, isLeap)`, `LeapMonth(lunarYear)`, `DaysInLunarMonth(lunarYear, month, isLeap)`, `LunarMonths(lunarYear)`, `SolarTermDate(year, longitudeDeg)`. Múi giờ cố định **UTC+7** (định nghĩa của lịch âm Việt Nam, không phải cấu hình theo môi trường — ghi bằng hằng + comment). Giữ trong `domain/` vì mới một service dùng; nâng lên `pkg/` nếu service khác cần. |
| Domain / Pure logic | `services/appService/domain/calendar_event_resolver.go` | `ResolveOccurrences(events, year) []CalendarOccurrence`: áp từng `RecurrenceRule` cho `year` (xem "Sinh ngày theo quy tắc"), lọc theo năm dương, sắp xếp. Thuần, không I/O, không đọc đồng hồ. |
| Repository | `services/appService/repository/postgres_calendar_event_repository.go` | `NewCalendarEventRepository(db)`. `ListActiveEvents`: **một** SELECT `calendar_events` JOIN `calendar_event_translations` (`lang = $1`) với `is_active`, kèm danh sách nhóm của từng sự kiện (`array_agg` theo `calendar_groups.sort_order`). `ListGroups`: SELECT `calendar_groups` sắp theo `sort_order, code`. `ReplaceCatalogue`: **một** transaction, `INSERT … ON CONFLICT … DO UPDATE` (nhóm, sự kiện, bản dịch), thay liên kết nhóm của từng sự kiện; **không DELETE** sự kiện vắng mặt trong file (BO tắt bằng `is_active`). |
| Usecase | `services/appService/usecase/calendar_event_usecase.go` | `CalendarEventUsecase.EventsForYear(ctx, year string)`: bỏ trống → năm hiện tại theo `calendar.timezone`; parse số nguyên, kiểm tra 1900..2100 → `ErrInvalidYear`; đọc nhóm + sự kiện từ repository; `ResolveOccurrences`; trả `{Year, Groups, Events, Occurrences}`. Đồng hồ tiêm vào để test (như `CalendarUsecase`). Tách khỏi `CalendarUsecase` để không đổi chữ ký `NewCalendarUsecase` của route ảnh. |
| Delivery | `services/appService/delivery/http/calendar_event_handler.go` | `GET /v1/calendar/events`; DTO riêng (`year`, `groups`, `events`, `occurrences`); `lunar_date` là object; `image_url` có thể `null`; đặt `Cache-Control` (E17). |
| Delivery / errors | `services/appService/delivery/http/errors.go` | Thêm `ErrInvalidYear` → `400 INVALID_YEAR`, `details.field = "year"`. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `CalendarConfig` thêm `DefaultLanguage` (`vi`), `EventsCacheMaxAge` (cùng kiểu với `administrative_units.cache_max_age`). Giá trị không nhạy cảm, đổi theo môi trường → để trong config, không để hằng trong usecase/domain. Cả hai bắt buộc: rỗng/0 là lỗi cấu hình (fail ở khởi động). |
| Migration | `migrations/0031_calendar_events.sql` | Bốn bảng, xem "Mô hình dữ liệu". Idempotent (`IF NOT EXISTS`) như các file khác. |
| Seed CLI | `services/appService/cmd/seedcalendarevents/main.go`, `parse.go` | `-file` (mặc định `assets/data/calendar-events.json`), `-dry-run`. Giải mã **strict**, `Verify`, nạp qua `ReplaceCatalogue`. Cùng khung `seedadminunits` (`⚠️ DEV ONLY` config/secret). |
| Data | `assets/data/calendar-events.json` *(team thu thập, bản nháp)* | `meta` + `groups` + `events`; mỗi sự kiện có `source` (ghi nguồn để review, không nạp vào DB). Đọc từ đĩa lúc chạy seed, không `go:embed`. |
| Wiring | `services/appService/cmd/main.go` | Dựng repository/usecase/handler; đăng ký `mux.HandleFunc("GET /v1/calendar/events", authMiddleware.AuthenticateFunc(calendarEventHandler.Events))` (cùng cách `GET /v1/lamps`). |
| API spec | `docs/specs/api/openapi.yaml` | Thêm `/v1/calendar/events` (security bearer), schema `CalendarEventsResponse`, `CalendarOccurrence`, `LunarDate`, code `INVALID_YEAR`. |

## Mô hình dữ liệu

Bốn bảng (chi tiết cột/ràng buộc chốt khi viết migration; ERD `docs/specs/db/` cần cập nhật):

| Bảng | Khoá | Cột chính | Ghi chú |
|---|---|---|---|
| `calendar_groups` | `code` | `name` (vi), `sort_order` | Nhóm là **dữ liệu**: seed ban đầu `buddhist`, `vietnam`, `folk`; BO thêm được (E14). |
| `calendar_events` | `code` | `kind`, `solar_month`, `solar_day`, `lunar_month`, `lunar_day`, `is_leap_month`, `solar_term_longitude`, `is_public_holiday`, `image_url` (null), `sort_order`, `is_active` | Mỗi `kind` bắt buộc đúng tập cột của nó (CHECK theo `kind`); cột còn lại NULL. `is_public_holiday`, `is_active` mặc định `false`/`true`. |
| `calendar_event_translations` | `(event_code, lang)` | `name`, `short_description`, `long_description`, `guidance` | v1 chỉ có `lang = 'vi'`. Thêm ngôn ngữ = thêm dòng, không đổi `calendar_events`. |
| `calendar_event_groups` | `(event_code, group_code)` | — | Mỗi sự kiện ≥ 1 nhóm (kiểm ở seed/BO, không CHECK được ở DB). |

Không bảng nào lưu từng năm; không có bảng ngoại lệ (nghỉ bù ngoài phạm vi).

### Bảng tham số theo `kind`

| `kind` | Tham số bắt buộc | Ví dụ (nội dung seed do team soạn, chỉ minh hoạ) |
|---|---|---|
| `solar` | `solar_month` 1..12, `solar_day` 1..31 (29/2 cho phép, kiểm tra ngày tối đa theo tháng) | 30/4, 25/12 |
| `lunar` | `lunar_month` 1..12, `lunar_day` 1..30, `is_leap_month` | 15/4 ÂL, 10/3 ÂL, 1/1 ÂL |
| `lunar_monthly` | `lunar_day` 1..30 (không có tháng) | mùng 1, Rằm |
| `lunar_last_day` | `lunar_month` 1..12 | Giao thừa (tháng 12) |
| `solar_term` | `solar_term_longitude` ∈ {0, 15, …, 345} | Thanh minh 15°, Đông chí 270° |

## Sinh ngày theo quy tắc

`ResolveOccurrences(events, Y)` với `Y` là năm dương được hỏi:

- **`solar`**: `time.Date(Y, m, d)`; nếu ngày không tồn tại (29/2 năm không nhuận) thì bỏ. Ngày âm = `SolarToLunar`.
- **`lunar`**: một ngày âm có thể rơi vào năm dương `Y` từ **năm âm `Y−1`** (tháng Chạp rơi vào tháng 1–2 dương) hoặc `Y`. Với mỗi `L ∈ {Y−1, Y}`: tháng đích là tháng thường `m` (hoặc tháng nhuận `m` nếu `is_leap_month`; bỏ qua nếu năm âm `L` không có tháng nhuận đó) → `LunarToSolar` → giữ nếu năm dương == `Y`. Ngày dương lấy từ chính phép đổi; ngày âm trả về chính là `(day, month, L, is_leap)`.
- **`lunar_monthly`**: với `L ∈ {Y−1, Y}`, duyệt mọi tháng âm của `L` (kể cả tháng nhuận), nếu tháng đủ ngày `lunar_day` thì `LunarToSolar`, giữ nếu năm dương == `Y`.
- **`lunar_last_day`**: với `L ∈ {Y−1, Y}`, ngày = `DaysInLunarMonth(L, lunar_month, false)` (29 hoặc 30), rồi `LunarToSolar`, giữ nếu năm dương == `Y`.
- **`solar_term`**: `SolarTermDate(Y, longitude)` = ngày dương (UTC+7) **chứa thời điểm** kinh độ Mặt Trời đạt `longitude`; ngày âm = `SolarToLunar`.

Một sự kiện lunar-stored có thể cho 0, 1 hoặc nhiều occurrence trong một năm dương (`lunar_monthly` cho ~12–13). Sắp xếp: `solar_date`, `sort_order`, `code` (E12).

Độ chính xác: bộ chuyển đổi dựa trên thuật toán Hồ Ngọc Đức (kinh độ Mặt Trời + Sóc tính theo ngày Julian, múi giờ 7) — đúng cho 1900..2100. Kiểm thử đối chiếu bảng tham chiếu (E20, NFR-002).

## Response shape

Chuẩn hoá theo sự kiện để không lặp mô tả dài của sự kiện lặp hàng tháng (mùng 1/Rằm ~25 occurrence/năm):

```
data: {
  year: 2026,
  groups:      [ {code, name} … ],
  events:      [ {code, name, short_description, long_description, guidance,
                  image_url|null, groups:[code…], is_public_holiday} … ],   // mỗi sự kiện một lần
  occurrences: [ {event_code, solar_date:"YYYY-MM-DD",
                  lunar_date:{day, month, year, is_leap_month}} … ]          // đã sắp xếp
}
```

Mỗi `occurrence` trỏ về `events[].code`. Client ghép theo `event_code`. Chỉ trả các sự kiện có ít nhất một occurrence trong năm đó. `image_url` là giá trị lưu trong DB (tuyệt đối hoặc `null`); khác route ảnh, không dựng từ `Host` của request — ảnh sự kiện chưa có (xem Known gaps).

## Request flow

Vì US-CAL-002 có hai lần đọc DB nối tiếp rồi một bước tính, flow chi tiết nằm ở **Step-by-step Flow** bên dưới và bản vẽ ở `flow.md`. Route ảnh (US-CAL-001) vẫn giữ trace ASCII:

```
GET /v1/calendar/image?date=YYYY-MM-DD
  └─ public (không middleware xác thực)
       └─ CalendarHandler.Image              (delivery/http)
            ├─ CalendarUsecase.ImageForDate(date)   (usecase)
            │    ├─ date rỗng → hôm nay theo calendar.timezone
            │    ├─ parse YYYY-MM-DD + kiểm tra năm 1900..2100 → ErrInvalidDate
            │    └─ images[dayNumber mod N]         (danh sách trong bộ nhớ)
            ├─ dựng image_url tuyệt đối (calendar.image_base_url, hoặc scheme+host của request)
            └─ Cache-Control + utils.WriteSuccessResponse
```

Route ảnh không có Step-by-step Flow (chỉ tính trong bộ nhớ, không lời gọi ngoài).

## Step-by-step Flow — Lấy sự kiện của một năm (`GET /v1/calendar/events`)

```
Step 1 — Xác thực (middleware)
- Trigger: request tới GET /v1/calendar/events
- Pre-condition: route được bọc authMiddleware.AuthenticateFunc, RequireTokenType = "access"
- Call: jwt/middleware kiểm tra chữ ký, hạn, loại token; đặt user_id vào context
    - Success → Step 2
    - Failure (thiếu/hết hạn/sai loại token) → 401 theo middleware hiện có, dừng; không chạm DB

Step 2 — Xác định năm (handler → usecase)
- Trigger: Step 1 pass
- Pre-condition: query `year` là chuỗi (có thể rỗng)
- Call: CalendarEventUsecase.EventsForYear: rỗng → năm hiện tại theo calendar.timezone; ngược lại
  strconv.Atoi nghiêm ngặt (không "2026.5", không khoảng trắng), kiểm tra 1900..2100
    - Success → Step 3
    - Failure (không phải số nguyên / ngoài 1900..2100) → domain.ErrInvalidYear → 400 INVALID_YEAR,
      details.field = "year", dừng; không chạm DB

Step 3 — Đọc nhóm (repository)
- Trigger: Step 2 pass
- Pre-condition: năm hợp lệ
- Call: SELECT code, name FROM calendar_groups ORDER BY sort_order, code
    - Success → Step 4
    - Failure (lỗi DB) → 500, log; chưa có gì được tính

Step 4 — Đọc sự kiện đang hoạt động (repository)
- Trigger: Step 3 pass
- Pre-condition: ngôn ngữ = calendar.default_language
- Call: SELECT sự kiện is_active JOIN bản dịch (lang) kèm mảng mã nhóm — một câu SELECT
  Confirmed: hai lần đọc (Step 3, 4) không bọc transaction; chúng chỉ đọc và dữ liệu hiếm đổi,
  nên một thay đổi BO chen giữa hai câu chỉ có thể làm một mã nhóm của sự kiện vắng trong `groups`
  — usecase bỏ qua mã nhóm không có trong danh mục thay vì lỗi (xem Known gaps).
    - Success → Step 5
    - Failure (lỗi DB) → 500, log

Step 5 — Sinh occurrence (thuần, không I/O)
- Trigger: Step 4 pass
- Pre-condition: danh sách sự kiện đã nạp
- Call: ResolveOccurrences(events, year) — áp từng RecurrenceRule (mục "Sinh ngày theo quy tắc"),
  tính ngày âm/dương còn lại, lọc theo năm dương, sắp xếp (E12)
    - Success → Step 6
    - Failure (tham số quy tắc không hợp lệ trong DB, ví dụ kind lạ) → bỏ sự kiện đó, log cảnh báo
      kèm code; không làm hỏng cả response. Seed Verify đã chặn dữ liệu sai, nên đây là lưới an toàn
      cho dữ liệu BO sửa sai.

Step 6 — Dựng response và Cache-Control (handler)
- Trigger: Step 5 pass
- Pre-condition: có {year, groups, events, occurrences}
- Call: giữ chỉ những sự kiện có occurrence; map sang DTO; đặt Cache-Control: `year` rõ ràng →
  `private, max-age=<calendar.events_cache_max_age>`, bỏ trống → `no-cache`;
  utils.WriteSuccessResponse
    - Success → 200
    - Failure (lỗi ghi response) → log, không thể đổi status
```

## Step-by-step Flow — Nạp danh mục sự kiện (`cmd/seedcalendarevents`)

```
Step 1 — Đọc file seed (seed CLI)
- Trigger: operator chạy `go run ./services/appService/cmd/seedcalendarevents -file assets/data/calendar-events.json`
- Pre-condition: file tồn tại
- Call: os.ReadFile
    - Success → Step 2
    - Failure (không đọc được) → in lỗi, exit 1; không chạm DB

Step 2 — Giải mã strict và kiểm tra (seed CLI, parse.go)
- Trigger: Step 1 pass
- Pre-condition: nội dung file là JSON
- Call: giải mã strict (khoá lạ là lỗi, dữ liệu thừa sau JSON là lỗi); Verify: `code` duy nhất,
  tên (vi) không rỗng, `kind` hợp lệ và có đủ tham số theo "Bảng tham số theo kind" (khoảng giá trị,
  `solar_term_longitude` bội của 15, tháng/ngày hợp lệ), mỗi sự kiện ≥ 1 nhóm và nhóm tồn tại trong
  `groups` của file, `sort_order` ≥ 1; in tối đa 50 vi phạm
    - Success → Step 3 (`-dry-run` dừng ở đây, không cần DB)
    - Failure (vi phạm) → in danh sách, exit 1; không chạm DB

Step 3 — Kết nối DB (seed CLI)
- Trigger: Step 2 pass, không `-dry-run`
- Pre-condition: config + secret dev (⚠️ DEV ONLY, như seedadminunits)
- Call: nạp config, mở kết nối Postgres
    - Success → Step 4
    - Failure → in lỗi, exit 1

Step 4 — Nạp catalogue trong một transaction (repository)
- Trigger: Step 3 pass
- Pre-condition: dữ liệu đã Verify
- Call: BEGIN; upsert `calendar_groups`; upsert `calendar_events`; upsert `calendar_event_translations`
  (lang = vi); với mỗi sự kiện: xoá rồi chèn lại `calendar_event_groups`; COMMIT
  Confirmed: ghi đè nội dung đã có theo `code` (quyết định 2026-10-11: ở dev chạy lại thì đè); không
  DELETE sự kiện vắng trong file; không guard production (đang develop).
    - Success → Step 5
    - Failure (lỗi DB giữa chừng) → ROLLBACK, in lỗi, exit 1; không có gì bị ghi một phần

Step 5 — Báo cáo (seed CLI)
- Trigger: Step 4 pass
- Call: in số nhóm / sự kiện / bản dịch đã nạp và thời gian
    - Success → exit 0
```

## Known gaps carried by this feature

- US-CAL-001: `image_url` mặc định dựng từ `Host` của request: tiện cho dev (IP LAN đổi liên tục) nhưng **không được dùng cho production** — production đặt `calendar.image_base_url` trỏ CDN. Ảnh phục vụ qua `/static/images/` vẫn là dev-only. Chưa có thumbnail.
- US-CAL-002 ảnh sự kiện: `image_url` lưu nguyên giá trị trong DB, seed để `null`. Khi có ảnh phải chốt nơi lưu/CDN và ghi công (giống ảnh ngày) — việc riêng.
- Tên nhóm chỉ có một ngôn ngữ (`calendar_groups.name`); thêm ngôn ngữ cần bảng dịch cho nhóm. Sự kiện đã có bảng dịch.
- Hai lần đọc DB không trong transaction (Step 3–4): chấp nhận vì chỉ đọc và dữ liệu hiếm đổi; mã nhóm lạ bị bỏ qua.
- Tính toán mỗi request, không cache trong bộ nhớ: ~vài chục sự kiện, vài trăm phép tính — dư sức với mục tiêu 200 ms (NFR-001). Nếu cần, cache kết quả theo `(year, version danh mục)` sau.
- Cơ chế quản trị (BO) chưa có: tới lúc đó seed chỉ chạy ở dev; không có guard chặn chạy seed đè dữ liệu BO ở môi trường khác.
- Tiết khí nào là sự kiện do người soạn seed chọn; cơ chế hỗ trợ cả 24 tiết (bội của 15°).
