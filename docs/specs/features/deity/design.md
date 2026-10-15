# Design — deity

Version: 3.0
Last updated: 2026-10-15
Status: in-progress — 3 route đọc + seed đã chạy thật (dev); bộ lọc `type` có E2E. **v3.0 chưa implement**: các hàng / Step đánh dấu **(v3.0)** là thiết kế, chưa có code (trang 12, liên kết vía với lịch lễ, `next_via` / `via_events`, tham số `date`, file danh mục vào git).
Module: `deity`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Ba route (`GET /v1/deities/top`, `GET /v1/deities`, `GET /v1/deities/{id}`) đều công khai, không đi qua middleware xác thực. Feature chạm đủ bốn layer Clean Architecture của `services/appService` (`domain` → `usecase` → `repository` / `delivery/http`, nối dây ở `cmd/main.go`) và sở hữu migration `0007_deities.sql`, `0015_deities_deva_type.sql` và (v3.0) migration liên kết vía. Dữ liệu vào bảng bằng lệnh `cmd/seeddeities` (upsert theo `id`), **không** seed khi khởi động, file JSON **không** nhúng vào binary. Cùng hình dạng với `temple` và `article`; khác ở chỗ thứ tự dựa hoàn toàn vào `sort_order` (vị trí trong file).

**(v3.0)** Ngày vía không còn là dữ liệu của deity: mỗi deity liên kết tới các sự kiện của lịch lễ (`calendar_events.code`), và mọi lần đọc deity tính lần vía kế tiếp bằng chính bộ tính ngày của `calendar` (`domain.ResolveOccurrences`) — server tính, client chỉ đếm ngược. Deity usecase vì vậy phụ thuộc thêm `domain.CalendarEventRepository` (chỉ đọc `ListActiveEvents`) và một đồng hồ; nó không gọi `CalendarEventUsecase` và không đổi gì của feature `calendar`.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/deity.go` | `Deity`, `Iconography`, `Mantras`, `DeityPage`, hằng `DeityBuddha`/`DeityBodhisattva`/`DeityDeva`, `IsValidDeityType`. Không JSON/db tag. **(v3.0)** `Deity` bỏ `LunarMemorialDays` và kiểu `MemorialDay`; thêm `ViaEventCodes []string` (mã sự kiện lịch liên kết, nội dung danh mục). |
| Domain / Vía (v3.0, chưa có) | `services/appService/domain/deity_via.go` | Hàm thuần, không clock, không I/O. `DeityVia{Event CalendarEvent; Next *CalendarOccurrence}`. `PickNextVia(occurrences []CalendarOccurrence, sortOrder map[string]int, today time.Time) *CalendarOccurrence` — lần sớm nhất có `SolarDate >= today`, hoà thì `sortOrder` rồi `EventCode` (BR-DTY-06, BR-CAL-12); đây là hàm mà `rules/deity-next-via.json` chạy trên. `ResolveDeityVias(events []CalendarEvent, today time.Time) []DeityVia` — gọi `ResolveOccurrences(events, Y)` và `(…, Y+1)` (bỏ `Y+1` khi `Y = 2100`), lấy lần đầu `>= today` cho từng sự kiện, sắp theo AC 28 (sự kiện không có lần nào xếp cuối). `NextVia([]DeityVia) *DeityVia` = phần tử đầu có `Next != nil`. |
| Domain / Errors | `domain/deity.go` | `ErrDeityNotFound`, `ErrInvalidDeityType`; tái dùng `ErrInvalidPage` (`domain/temple.go`). **(v3.0)** tái dùng `ErrInvalidDate` (`domain/calendar.go`). |
| Domain / Repository interface | `domain/deity.go` | `DeityRepository`: `ListTop`, `List(ctx, deityType, offset, limit)`, `FindByID`, `Upsert`. Thứ tự `sort_order ASC, id ASC` là một phần của hợp đồng. **(v3.0)** mọi lần đọc trả `ViaEventCodes` (sắp theo mã); `Upsert` ghi cả liên kết vía; thêm `MissingCalendarEvents(ctx, codes []string) ([]string, error)` cho seed. |
| Domain / Mock | `domain/mock_deity_repository.go` | Mock viết tay cho test usecase. **(v3.0)** thêm `MissingCalendarEvents`. Usecase dùng lại `domain/mock_calendar_event_repository.go` có sẵn. |
| Usecase | `services/appService/usecase/deity_usecase.go` | `Top`, `ListDeities` (offset, `total_pages`, chặn `page < 1` / `> MaxInt32`, kiểm `type`, `items` nil → `[]`), `Get`. **(v3.0)** constructor nhận thêm `domain.CalendarEventRepository`, `config.CalendarConfig` (`Timezone`, `DefaultLanguage`) và `now func() time.Time`; ba method nhận thêm `date string`: rỗng → hôm nay trong `calendar.timezone`, ngược lại parse `YYYY-MM-DD` năm 1900..2100 như `CalendarUsecase` (lỗi → `ErrInvalidDate`), kiểm sau `page` và `type`. Sau khi đọc deity: một lần `ListActiveEvents(lang)`, lọc theo `ViaEventCodes` của từng deity (mã không có trong danh sách active bị bỏ qua — BR-CAL-16), `ResolveDeityVias`, `NextVia`. Trả `domain.DeityWithVia{Deity; Vias []DeityVia; Next *DeityVia}` (hoặc trang của nó). |
| Repository | `services/appService/repository/postgres_deity_repository.go` | SQL cho `deities` với `deityColumns`/`deityOrder` dùng chung; `List` thêm `WHERE type = $n` khi lọc, cho cả câu đếm và câu lấy trang; `Upsert` một transaction `ON CONFLICT (id) DO UPDATE` ghi đè mọi cột; `pq.Array` cho `sacred_artifacts`/`great_vows`; `sql.ErrNoRows` → `ErrDeityNotFound`. **(v3.0)** bỏ cột `lunar_memorial_days` và `memorialDayJSON`; `deityColumns` thêm `ARRAY(SELECT event_code FROM deity_via_events v WHERE v.deity_id = d.id ORDER BY event_code)` (không thêm câu truy vấn); `Upsert` trong cùng transaction: `DELETE FROM deity_via_events WHERE deity_id = ANY($1)` rồi `INSERT` lại liên kết của file; `MissingCalendarEvents` = mã trong `codes` không có ở `calendar_events` (kể cả inactive — mã inactive vẫn hợp lệ để liên kết). |
| Seed CLI | `services/appService/cmd/seeddeities/main.go`, `parse.go` | `main`: đọc file (`-file`), nạp config + secret dev (`⚠️ DEV ONLY`), kết nối DB, gọi `Upsert`. `parseDeities`: JSON thuần, kiểm id rỗng/trùng, `type` ∈ 3 loại, `name_vi`; `SortOrder` = vị trí trong file. **(v3.0)** `sourceDeity` bỏ `lunar_memorial_days`, thêm `via_event_codes: [string]` (mỗi mã không rỗng, không trùng trong một deity); `main` gọi `MissingCalendarEvents` với mọi mã trước khi `Upsert`, có mã thiếu → `log.Fatalf` liệt kê mã, không ghi gì. |
| Data | `assets/data/phat_botat.json` | 20 mục (7 Phật, 12 Bồ tát, 1 Chư Thiên), JSON thuần, đọc từ đĩa lúc seed. **Đang bị `.gitignore` (`*.json`) và vắng trong checkout này.** **(v3.0)** đưa vào git bằng ngoại lệ `!assets/data/phat_botat.json`; mỗi mục có `via_event_codes` thay cho `lunar_memorial_days` (nội dung chặn bởi `⚠️ Not confirmed (product)` về ánh xạ vía trong `requirements.md`). |
| Delivery / Handler | `services/appService/delivery/http/deity_handler.go` | `Top`, `ListDeities`, `Get`; parse `page`; đọc `type` thô. Interface `deityUsecase` khai báo phía consumer. **(v3.0)** đọc `date` thô (`r.URL.Query().Get("date")`) cho cả ba route, chuyển cho usecase, không tự kiểm. |
| Delivery / DTO | `delivery/http/deity_handler.go` | `deitySummaryResponse`, `deityDetailResponse`, `iconographyResponse`, `mantrasResponse`, `deityTopResponse`, `deityListResponse` — JSON tag chỉ ở đây. URL ảnh qua `imageHost.url` (gốc tương đối → tuyệt đối; `""` giữ `""`). **(v3.0)** summary thêm `next_via *deityViaResponse` (`{event_code, name, solar_date, lunar_date}`, `lunar_date` cùng shape với occurrence của `GET /v1/calendar/events`); detail thêm `via_events []deityViaEventResponse` (`{event_code, name, next_occurrence *{solar_date, lunar_date}}`) và `lunar_memorial_days` dẫn xuất từ `via_events` (AC 30, `memorialDayResponse` giữ làm DTO thuần, không còn domain counterpart). Mảng không bao giờ `null`. |
| Delivery / Error map | `delivery/http/errors.go` | `ErrDeityNotFound` → `404 DEITY_NOT_FOUND`; `ErrInvalidPage` → `400 INVALID_PAGE`; `ErrInvalidDeityType` → `400 INVALID_DEITY_TYPE`; còn lại → `500`. **(v3.0)** `ErrInvalidDate` đã map sẵn sang `400 INVALID_DATE`, `details.field` mặc định `"date"` (đã đọc `errors.go`) — không cần sửa file này. |
| Delivery / Images | `delivery/http/image_url.go` | `imageHost` dùng chung: giá trị bắt đầu bằng `/` được ghép `server.public_base_url` (rỗng = origin của request); giá trị khác trả nguyên. |
| Delivery / Fallback | `delivery/http/fallback.go` | `WithAPIFallback` đổi 404/405 của `ServeMux` thành envelope. Dùng chung. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `DeityConfig{TopLimit, PageSize}` (dev: 10, 20). **(v3.0)** `page_size: 12`. Deity usecase đọc thêm `cfg.Calendar.Timezone` và `cfg.Calendar.DefaultLanguage` (không thêm khoá mới). |
| Wiring | `services/appService/cmd/main.go` | Dựng repo → usecase → handler (`imageBase` = `WithImageBase(cfg.Server.PublicBaseURL)`); đăng ký 3 route. **(v3.0)** truyền `calendarEventRepo` (đã dựng cho calendar), `cfg.Calendar`, `time.Now` vào `NewDeityUsecase`. Không dính khối `⚠️ DEV ONLY` của `main.go`. `GET /v1/deities/{id}/sutras` thuộc `sutra`, không đổi. |
| Migration | `migrations/0007_deities.sql`, `0015_deities_deva_type.sql`, `0021_relative_image_urls.sql` | Bảng `deities`, `CHECK` ba loại, index `idx_deities_order`, `idx_deities_type_order`; ảnh tương đối. **(v3.0, chưa có)** `migrations/00NN_deity_via_events.sql` (số trống kế tiếp lúc implement, hôm nay `0037`): `CREATE TABLE IF NOT EXISTS deity_via_events (deity_id VARCHAR(64) REFERENCES deities(id) ON DELETE CASCADE, event_code VARCHAR(64) REFERENCES calendar_events(code), PRIMARY KEY (deity_id, event_code))`, index `(event_code)`; `ALTER TABLE deities DROP COLUMN IF EXISTS lunar_memorial_days`. Phụ thuộc `0031_calendar_events.sql`. |
| Make / dữ liệu | `Makefile` | `ensure-deities` / `seed-deities`. **(v3.0)** `ensure-data` chạy `ensure-calendar-events` trước `ensure-deities` (FK + kiểm mã), `ensure-deities` phụ thuộc `ensure-calendar-events`. |
| E2E harness | `services/appService/e2e/harness.go` | Seed `deities` từ `assets/data/phat_botat.json` (cột tối thiểu). **(v3.0)** đọc file đã commit; test vía cần thêm seed sự kiện lịch tối thiểu và liên kết. |
| API spec | `docs/specs/api/openapi.yaml`, `web/docs/public/openapi.yaml` | Có 3 route + schema (tag Deities). **(v3.0) chưa cập nhật**: `page_size` 12, `date`, `INVALID_DATE`, `next_via`, `via_events`, `lunar_memorial_days` deprecated, ví dụ ảnh (đang là `http://192.168.1.186:8080/...`). Sửa cùng lượt với handler. |
| ERD | `docs/specs/db/erd.mmd`, `erd.svg` | Có bảng `deities`. **(v3.0)** thêm `deity_via_events` + hai FK, bỏ `lunar_memorial_days`. |

Ghi chú về route: `GET /v1/deities/top` là segment literal nên `ServeMux` ưu tiên nó hơn `GET /v1/deities/{id}`; `GET /v1/deities/{id}/sutras` (spec `sutra`) dài hơn nên không đụng detail. Tham số mới (`date`) là query, không đổi bảng route.

## Request flow

Từ v3.0 cả ba route đọc đều có nhiều bước (đọc deity → đọc catalogue lịch → tính vía), nên mỗi route có Step-by-step Flow riêng bên dưới.

> Xem `flow.md` cho sơ đồ sequence Mermaid (mirror 1:1 của các Step-by-step Flow bên dưới).

## Step-by-step Flow — Danh sách Phật và Bồ tát phân trang (`GET /v1/deities?type=&page=N&date=`)

```
Step 1 — Đọc `page`, `type`, `date` (handler)
- Trigger: request đến route công khai
- Pre-condition: —
- Call: r.URL.Query().Get("page"); rỗng → page = 1, ngược lại strconv.Atoi. deityType = Get("type") (chuỗi thô). (v3.0) date = Get("date") (chuỗi thô)
    - Success → Step 2
    - Failure (page không phải số nguyên, kể cả tràn int) → ErrInvalidPage → 400 INVALID_PAGE, details.field = "page"

Step 2 — Kiểm tra `page`, rồi `type`, rồi `date` (usecase)
- Trigger: handler gọi DeityUsecase.ListDeities(ctx, deityType, page, date)
- Pre-condition: page là số nguyên
- Call: page < 1 hoặc > math.MaxInt32; rồi deityType != "" và !IsValidDeityType; rồi (v3.0) date rỗng → today = now() trong calendar.timezone, ngược lại parse YYYY-MM-DD, năm 1900..2100
    - Success → Step 3 (offset = (page-1) * page_size trên danh sách đã lọc; page_size = 12 từ v3.0)
    - Failure (page) → ErrInvalidPage → 400 INVALID_PAGE
    - Failure (type lạ) → ErrInvalidDeityType → 400 INVALID_DEITY_TYPE, details.field = "type"
    - Failure (v3.0, date sai) → ErrInvalidDate → 400 INVALID_DATE, details.field = "date"; chưa chạm DB

Step 3 — Đếm số deity (repository)
- Trigger: usecase gọi DeityRepository.List(ctx, deityType, offset, limit)
- Pre-condition: tham số đã hợp lệ
- Call: SELECT count(*) FROM deities [WHERE type = $1]
    - Success → total (khớp bộ lọc), sang Step 4
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR

Step 4 — Lấy trang (repository)
- Trigger: Step 3 xong
- Pre-condition: có total
- Call: SELECT <cột> (v3.0: kèm ARRAY(SELECT event_code FROM deity_via_events …) AS via_event_codes) FROM deities d [WHERE type = $1] ORDER BY sort_order, id OFFSET $n LIMIT $m
    - Success → items (rỗng nếu offset vượt cuối) và total, sang Step 5
    - Failure (lỗi DB / scan) → error bọc %w → 500 INTERNAL_ERROR
    - Transaction boundary: không có — Step 3 và Step 4 là hai câu độc lập, không chung snapshot

Step 5 — Đọc catalogue lịch lễ (v3.0, repository của calendar)
- Trigger: Step 4 trả về; bỏ qua khi items rỗng hoặc không deity nào có via_event_codes
- Pre-condition: lang = calendar.default_language
- Call: CalendarEventRepository.ListActiveEvents(ctx, lang) — một câu, chỉ sự kiện is_active (BR-CAL-16), có rule, tên, sort_order
    - Success → Step 6
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR (không trả trang thiếu vía)

Step 6 — Tính lần vía kế tiếp (v3.0, domain thuần)
- Trigger: Step 5 xong
- Pre-condition: today (Step 2), danh sách sự kiện active
- Call: với từng deity: lọc sự kiện theo ViaEventCodes (mã không active → bỏ) → ResolveDeityVias(events, today) → NextVia
    - Success → Step 7
    - Failure → không có nhánh lỗi (rule sai bị ResolveOccurrences bỏ qua như ở calendar)

Step 7 — Dựng trang và trả response (usecase + handler)
- Trigger: Step 6 xong (hoặc Step 4/5 khi bỏ qua)
- Pre-condition: items, total
- Call: items nil → []; total_pages = (total + page_size - 1) / page_size; handler map sang DTO summary (+ next_via, null khi không có)
    - Success → 200 {items, page, page_size, total, total_pages}
    - Failure → không có nhánh lỗi mới
```

## Step-by-step Flow — Phật và Bồ tát nổi bật (`GET /v1/deities/top?date=`) (v3.0)

Trước v3.0 là một câu SQL; v3.0 thêm đọc catalogue lịch và tính vía nên thành nhiều bước.

```
Step 1 — Đọc `date` (handler)
- Trigger: request đến route công khai
- Pre-condition: —
- Call: date = r.URL.Query().Get("date") (chuỗi thô) → DeityUsecase.Top(ctx, date)
    - Success → Step 2
    - Failure → không có (handler không tự kiểm)

Step 2 — Xác định "hôm nay" (usecase)
- Trigger: Step 1
- Pre-condition: —
- Call: date rỗng → today trong calendar.timezone; ngược lại parse như danh sách Step 2
    - Success → Step 3
    - Failure (date sai) → ErrInvalidDate → 400 INVALID_DATE, details.field = "date"; chưa chạm DB

Step 3 — Lấy deity nổi bật (repository)
- Trigger: Step 2 xong
- Pre-condition: limit = deity.top_limit (10)
- Call: SELECT <cột> + via_event_codes FROM deities ORDER BY sort_order, id LIMIT $1
    - Success → items (có thể rỗng), sang Step 4
    - Failure (lỗi DB / scan) → 500 INTERNAL_ERROR

Step 4 — Đọc catalogue lịch lễ (repository của calendar)
- Trigger: Step 3 xong; bỏ qua khi không deity nào có via_event_codes
- Call: CalendarEventRepository.ListActiveEvents(ctx, lang)
    - Success → Step 5
    - Failure (lỗi DB) → 500 INTERNAL_ERROR

Step 5 — Tính lần vía kế tiếp (domain thuần)
- Trigger: Step 4 xong
- Call: như danh sách Step 6
    - Success → Step 6
    - Failure → không có nhánh lỗi

Step 6 — Trả response (handler)
- Trigger: Step 5 xong
- Call: map sang DTO summary (+ next_via); items nil → []
    - Success → 200 {items}
    - Failure → không có nhánh lỗi mới
```

## Step-by-step Flow — Chi tiết một vị (`GET /v1/deities/{id}?date=`) (v3.0)

```
Step 1 — Đọc `id`, `date` (handler)
- Trigger: request đến route công khai
- Pre-condition: —
- Call: r.PathValue("id"), r.URL.Query().Get("date") → DeityUsecase.Get(ctx, id, date)
    - Success → Step 2
    - Failure → không có

Step 2 — Xác định "hôm nay" (usecase)
- Trigger: Step 1
- Call: như top Step 2
    - Success → Step 3
    - Failure (date sai) → 400 INVALID_DATE; chưa chạm DB (date được kiểm trước khi tìm id)

Step 3 — Tìm deity (repository)
- Trigger: Step 2 xong
- Call: SELECT <cột> + via_event_codes FROM deities WHERE id = $1
    - Success → deity, sang Step 4
    - Failure (sql.ErrNoRows) → ErrDeityNotFound → 404 DEITY_NOT_FOUND
    - Failure (lỗi DB khác) → 500 INTERNAL_ERROR

Step 4 — Đọc catalogue lịch lễ (repository của calendar)
- Trigger: Step 3 xong; bỏ qua khi via_event_codes rỗng
- Call: CalendarEventRepository.ListActiveEvents(ctx, lang)
    - Success → Step 5
    - Failure (lỗi DB) → 500 INTERNAL_ERROR

Step 5 — Tính mọi ngày vía và lần kế tiếp (domain thuần)
- Trigger: Step 4 xong
- Call: ResolveDeityVias(events của deity, today) → vias (mỗi sự kiện active một lần, Next hoặc nil, sắp theo AC 28); NextVia(vias)
    - Success → Step 6
    - Failure → không có nhánh lỗi

Step 6 — Trả response (handler)
- Trigger: Step 5 xong
- Call: map sang DTO detail: next_via, via_events, lunar_memorial_days (dẫn xuất từ via_events có next_occurrence: lunar day/month của lần đó + tên sự kiện), mảng nil → []
    - Success → 200 DeityDetail
    - Failure → không có nhánh lỗi mới
```

## Step-by-step Flow — Nạp dữ liệu bằng `cmd/seeddeities`

Lệnh chạy tay, **không** thuộc quá trình khởi động của appService. (v3.0) Cần catalogue lịch đã nạp trước (`make ensure-calendar-events`).

```
Step 1 — Đọc file nguồn (seeddeities/main.go)
- Trigger: operator chạy `go run ./services/appService/cmd/seeddeities [-file <path>]`
- Pre-condition: thư mục hiện tại là gốc repo nếu dùng đường dẫn mặc định `assets/data/phat_botat.json` (v3.0: file có trong git)
- Call: os.ReadFile(*file)
    - Success → Step 2
    - Failure (không có file / không đọc được) → log.Fatalf "read <file>", thoát; chưa chạm DB

Step 2 — Kiểm tra dữ liệu (seeddeities/parse.go)
- Trigger: Step 1 xong
- Pre-condition: nội dung file trong bộ nhớ
- Call: parseDeities → json.Unmarshal → mỗi mục: id không rỗng/không trùng, type ∈ {BUDDHA, BODHISATTVA, DEVA}, name_vi không rỗng; (v3.0) via_event_codes: mỗi mã không rỗng, không trùng trong cùng mục; SortOrder = chỉ số trong file
    - Success → []domain.Deity, sang Step 3
    - Failure (JSON hỏng, không có mục, id rỗng/trùng, type lạ, name_vi rỗng, v3.0 mã vía rỗng/trùng) → log.Fatalf "parse <file>"; chưa chạm DB

Step 3 — Nạp config và kết nối DB (seeddeities/main.go)
- Trigger: Step 2 xong
- Pre-condition: `CONFIG_PATH` (mặc định `services/appService/config/config.dev.yaml`)
- Call: pkgconfig.LoadSource/LoadYAML → appConfig.NewMockSecretGetterFromYAML (⚠️ DEV ONLY) → appConfig.Load → database.New + Connect (timeout 30 giây)
    - Success → Step 4
    - Failure (thiếu config/secret, không kết nối được DB) → log.Fatalf; chưa ghi gì

Step 4 — Kiểm mã sự kiện lịch tồn tại (v3.0, repository)
- Trigger: Step 3 xong
- Pre-condition: DB kết nối được; migration 0031 + migration liên kết vía đã apply; catalogue lịch đã nạp
- Call: DeityRepository.MissingCalendarEvents(ctx, mọi mã via_event_codes) — SELECT các mã không có trong calendar_events
    - Success (không thiếu) → Step 5
    - Failure (có mã thiếu) → log.Fatalf liệt kê mã thiếu; không ghi gì
    - Failure (lỗi DB) → log.Fatalf; không ghi gì

Step 5 — Upsert mọi mục và liên kết vía (repository)
- Trigger: Step 4 xong
- Pre-condition: dữ liệu hợp lệ, mọi mã tồn tại
- Call: DeityRepository.Upsert → INSERT INTO deities (<cột>) … ON CONFLICT (id) DO UPDATE SET <mọi cột>, updated_at = now(); (v3.0) DELETE FROM deity_via_events WHERE deity_id = ANY($ids); INSERT INTO deity_via_events (deity_id, event_code) cho từng liên kết
    - Success → commit, sang Step 6
    - Failure (lỗi DB: CHECK type, FK event_code do mã bị xoá giữa Step 4 và 5, thiếu bảng, mất kết nối) → rollback toàn bộ → log.Fatalf "seed deities"
    - Transaction boundary: mọi INSERT/UPDATE/DELETE của Step 5 chung MỘT transaction — đủ hết hoặc không gì

Step 6 — Ghi log (seeddeities/main.go)
- Trigger: Upsert thành công
- Call: log.Printf "seeded %d deities from %s"
    - Success → thoát 0
    - Failure → không có nhánh lỗi
```

Ghi chú: deity bị xoá khỏi file vẫn nằm trong bảng (không có bước dọn); liên kết vía của nó giữ nguyên vì `DELETE` chỉ chạm các `id` có trong file.

## Deliberately stubbed for now (if applicable)

Không có code stub. Toàn bộ phần **(v3.0)** là chưa làm (không phải stub) — `tasks.md` Phase 8–10.

## Known gaps carried by this feature

- **File danh mục vắng và bị ignore**: `assets/data/phat_botat.json` khớp `*.json` trong `.gitignore`; checkout này không có file nên `TestParseDeities_RealDataFile` fail (`open ../../../../assets/data/phat_botat.json: no such file or directory`, chạy 2026-10-15) và E2E harness không seed được. Sửa ở Phase 8 (AC 33).
- **Catalogue lịch cũng chưa vào git** (`assets/data/calendar-events.json`, spec `calendar` E26 chưa làm) — v3.0 của deity phụ thuộc nó cho cả seed (FK) lẫn E2E.
- **E2E chỉ có bộ lọc `type`** (`e2e/deity_e2e_test.go`); `top`, `detail` chưa có.
- **Test repository dùng DB thật**: `postgres_deity_repository_test.go` cần Postgres (`TEST_DB_NAME=innerlamp_test` để khỏi `TRUNCATE` dữ liệu dev), bỏ qua khi `-short`.
- **Seed dùng config dev** (`⚠️ DEV ONLY`), chưa dùng được cho staging/production (product Appendix A mục 4).
- **Upsert ghi đè mọi cột và không xoá** mục đã bỏ khỏi file.
- **Validation seeder chỉ một phần**: mantra, đại nguyện… không được kiểm.
- **Đếm và lấy trang không cùng snapshot**: dữ liệu chỉ đổi khi seed, rủi ro thấp.
- **(v3.0) Cửa sổ hai năm**: sự kiện chỉ có trong năm có tháng nhuận có thể không có lần nào trong `Y`..`Y+1` → `next_occurrence: null` dù deity có liên kết.
- **(v3.0) Mỗi request đọc cả catalogue lịch** (~51 sự kiện) để lọc theo mã: rẻ ở cỡ hiện tại; nếu catalogue lớn lên, thay bằng truy vấn theo danh sách mã.
- **(v3.0) Response phụ thuộc "hôm nay"**: không thêm `Cache-Control` công khai; khi thiếu `date` kết quả đổi theo ngày ở `calendar.timezone`.
- **OKF chưa có deity** (`docs/knowledge/features/deity.md`).
