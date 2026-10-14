# Design — temple-event

Version: 1.1
Last updated: 2026-10-13
Status: đã code (xem `tasks.md`); còn kiểm tra tay và `api-contract.md`
Module: `temple-event`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Một route đọc **cần đăng nhập** `GET /v1/temples/{id}/events`, chạm đủ bốn layer của `services/appService`, thêm một bảng `temple_events`, một CLI seed và một khối config `temple_event`. Hình dạng giống `calendar` events (đọc DB rồi **tính** occurrence bằng hàm thuần, dùng lại `domain/lunarcal`) nhưng gắn với một chùa và hỏi theo cửa sổ ngày thay vì năm. Việc kiểm tra chùa tồn tại dùng lại `TempleRepository.FindByID` có sẵn.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/temple_event.go` (new) | `TempleEvent`, `TempleEventOccurrence`, enum `TempleEventKind` / `TempleEventStatus` / `TempleRecurrence`, `TempleEventWindow`. Không tag JSON. |
| Domain / Errors | `services/appService/domain/temple_event.go` | `ErrInvalidDateRange`, `ErrDateRangeTooLarge`, `ErrInvalidTempleEvent` (dùng cho `Verify`), `InvalidDateFieldError{Field}` (bọc `ErrInvalidDate`, để `details.field` là `from` hay `to`). Dùng lại `ErrInvalidDate` (calendar) và `ErrTempleNotFound` (temple). |
| Domain / Resolver | `services/appService/domain/temple_event_resolve.go` (new) | `ResolveTempleOccurrences(events, window, now, loc)` — hàm **thuần**, không I/O; dùng `lunarcal`. |
| Domain / Catalogue | `services/appService/domain/temple_event.go` | `TempleEventCatalogue{Source, Events}` + `Verify(loc) error` (bọc `ErrInvalidTempleEvent`, thông điệp liệt kê mọi lỗi; `loc` là múi giờ cấu hình để đọc sự kiện cả ngày). `TempleEvent.Validate(loc)` là kiểm tra từng sự kiện, dùng lại ở usecase và resolver. |
| Domain / Repository interface | `services/appService/domain/temple_event.go` | `TempleEventRepository{ListActiveByTemple, ReplaceCatalogue}`. Định nghĩa ở domain. |
| Domain / Mock | `services/appService/domain/mock_temple_event_repository.go` (new) | Mock viết tay cho test usecase. |
| Config | `services/appService/config/config.go` + `config.dev.yaml` | `TempleEventConfig{Timezone, DefaultWindowMonths, MaxWindowMonths}`, `Validate()`. Giá trị không nhạy cảm, đổi theo môi trường → config, không hằng trong usecase. |
| Usecase | `services/appService/usecase/temple_event_usecase.go` (new) | `EventsForTemple(ctx, templeID, from, to)`: dựng cửa sổ, kiểm tra chùa, đọc, gọi resolver. Phụ thuộc `TempleRepository` + `TempleEventRepository` (interface). |
| Repository | `services/appService/repository/postgres_temple_event_repository.go` (new) | SQL đọc sự kiện active của một chùa trong khoảng thô; `ReplaceCatalogue` upsert trong một transaction. |
| Delivery / Handler | `services/appService/delivery/http/temple_event_handler.go` (new) | Parse `{id}`, `from`, `to` → usecase → DTO. Không có logic nghiệp vụ. |
| Delivery / DTO | `services/appService/delivery/http/temple_event_handler.go` | Response DTO (`templeEventResponse`, `templeOccurrenceResponse`, …). Tag JSON chỉ ở đây. |
| Delivery / Errors | `services/appService/delivery/http/errors.go` | Map `ErrInvalidDateRange` → `400 INVALID_DATE_RANGE`, `ErrDateRangeTooLarge` → `400 DATE_RANGE_TOO_LARGE`. |
| Seed CLI | `services/appService/cmd/seedtempleevents/` (new: `main.go`, `parse.go`, `parse_test.go`) | Đọc `assets/data/temple-events.json` (`{meta:{format:1, source}, events:[…]}`, `starts_at`/`ends_at` RFC 3339 có offset, `recurrence_until` `YYYY-MM-DD`), strict-decode, `Verify`, gọi `ReplaceCatalogue`; cờ `-dry-run` như `seedcalendarevents`. |
| Data | `assets/data/temple-events.json` (new) | Dữ liệu **mock** dựa trên id trong `assets/data/temple.json`. |
| Wiring | `services/appService/cmd/main.go` | Dựng repo/usecase/handler; đăng ký route bọc `authMiddleware.AuthenticateFunc`. |
| Migration | `migrations/0035_temple_events.sql` | Bảng `temple_events` + index; `0034` đã dùng cho `temple_schedules` (code trước), nên đây là số kế tiếp thực tế. |
| Build | `Makefile` | `ensure-temple-events`, `seed-temple-events`, thêm vào `ensure-data`. |
| API spec | `docs/specs/api/openapi.yaml` | Route + schema + ba mã lỗi mới, cùng lúc với handler. |

Dependency rule: delivery → usecase → domain; repository → domain; domain không import tầng ngoài. Resolver nằm ở `domain` vì thuần và chỉ cần `lunarcal` (cũng ở domain).

## Mô hình dữ liệu

Một bảng `temple_events` (một bảng có cột lặp, quyết định 2026-10-13):

| Cột | Kiểu | Ghi chú |
|---|---|---|
| `id` | `VARCHAR(64)` PK | Mã ổn định do người soạn seed đặt (vd `tevt_north_001_khoa_tu_he`); seed upsert theo cột này. |
| `temple_id` | `VARCHAR(64)` NOT NULL, FK `temples(id)` `ON DELETE CASCADE` | Chùa sở hữu. Cascade giống `sutra_id`. |
| `title` | `VARCHAR(255)` NOT NULL | |
| `description` | `TEXT` NOT NULL DEFAULT `''` | |
| `kind` | `VARCHAR(32)` NOT NULL, CHECK | `KHOA_TU`, `PHAP_HOI`, `LE_CAU_AN`, `LE_CAU_SIEU`, `LE_HOI`, `THUYET_PHAP`, `KHAC`. Thêm kind = migration + đổi `domain`, như `temples.type`. |
| `starts_at` | `TIMESTAMPTZ` NOT NULL | Lần đầu tiên (sự kiện lặp) hoặc lần duy nhất. |
| `ends_at` | `TIMESTAMPTZ` NOT NULL, CHECK `ends_at >= starts_at` | Sự kiện lặp: `ends_at - starts_at` là thời lượng mỗi lần, **< 24 giờ**. |
| `is_all_day` | `BOOLEAN` NOT NULL DEFAULT FALSE | Xem "Quy ước cả ngày". |
| `status` | `VARCHAR(16)` NOT NULL DEFAULT `'SCHEDULED'`, CHECK | `SCHEDULED`, `POSTPONED`, `CANCELLED`. |
| `audience_note` | `TEXT` NOT NULL DEFAULT `''` | Vd "dành cho thiếu nhi 8–15 tuổi". |
| `contact_info` | `TEXT` NOT NULL DEFAULT `''` | Liên hệ/đường dẫn đăng ký, chỉ hiển thị. |
| `image_url` | `TEXT` NULL | `NULL` khi chưa có ảnh. |
| `recurrence` | `VARCHAR(16)` NOT NULL DEFAULT `'NONE'`, CHECK | `NONE`, `LUNAR_MONTHLY`, `WEEKLY`. |
| `recurrence_lunar_days` | `SMALLINT[]` NOT NULL DEFAULT `'{}'` | Bắt buộc không rỗng khi `LUNAR_MONTHLY`, mỗi phần tử 1..30. |
| `recurrence_weekday` | `SMALLINT` NULL, CHECK 0..6 | Bắt buộc khi `WEEKLY`; `0` = Chủ nhật … `6` = Thứ bảy (khớp `time.Weekday`). |
| `recurrence_until` | `DATE` NULL | Ngày cuối cùng (bao gồm) còn lặp; `NULL` = lặp mãi. |
| `is_active` | `BOOLEAN` NOT NULL DEFAULT TRUE | Seed không ghi cột này (xem Step-by-step Flow — Seed). |
| `created_at`, `updated_at` | `TIMESTAMPTZ` NOT NULL DEFAULT `now()` | |

CHECK nhất quán tham số lặp (một ràng buộc `chk_temple_events_recurrence`): `NONE` ⇒ không ngày âm, không thứ, không `until`; `LUNAR_MONTHLY` ⇒ có ngày âm, không thứ; `WEEKLY` ⇒ có thứ, không ngày âm.

Index `idx_temple_events_temple_start (temple_id, starts_at) WHERE is_active` — phục vụ truy vấn một chùa. Ngày âm **không lưu**.

### Quy ước cả ngày

`is_all_day = true`: `starts_at` và `ends_at` là 00:00 (giờ `temple_event.timezone`) của ngày đầu và ngày cuối; sự kiện bao trùm hết ngày cuối. Với resolver, thời điểm kết thúc hiệu lực = cuối ngày của `ends_at`. Sự kiện cả ngày chỉ hợp lệ khi `recurrence = NONE` hoặc thời lượng một ngày; `Verify` chặn tổ hợp khác.

## Sinh occurrence

`ResolveTempleOccurrences(events, window, now, loc)`; `window` = `[fromDate, toDate]` (ngày, bao gồm cả hai), `loc` = `temple_event.timezone`. Mọi so sánh theo giờ `loc`.

| `recurrence` | Cách sinh |
|---|---|
| `NONE` | Đúng một occurrence `[starts_at, ends_at]`. |
| `WEEKLY` | Mỗi ngày `d` trong cửa sổ có `weekday(d) = recurrence_weekday`, `d ≥ ngày(starts_at)`, `d ≤ recurrence_until` (nếu có). Giờ bắt đầu = giờ của `starts_at`; kết thúc = bắt đầu + (`ends_at − starts_at`). |
| `LUNAR_MONTHLY` | Với mỗi năm âm `y` từ `năm(from)−1` đến `năm(to)` (ngày 29/12 ÂL có thể rơi sang tháng 1 dương), lấy `lunarcal.LunarMonths(y)` (gồm tháng nhuận). Với mỗi tháng và mỗi `day` trong `recurrence_lunar_days`: `d' = min(day, month.Days)`; ngày dương = `month.Start + (d'−1)`. Loại trùng (29 và 30 trong tháng 29 ngày ⇒ một lần). Lọc `≥ ngày(starts_at)`, `≤ recurrence_until`. |

Sự kiện cả ngày: `ends_at` của occurrence là `23:59:59` của ngày cuối (`ends_at` trong DB là 00:00 của ngày cuối). Sau khi sinh, **giữ** occurrence nếu: (a) kết thúc `≥ now`; (b) giao với cửa sổ (`start ≤ cuối ngày toDate` và `end ≥ đầu ngày fromDate`). Rồi tính `lunar_date = lunarcal.SolarToLunar(ngày bắt đầu)`, sắp theo `starts_at`, rồi `event_id`.

Quyết định liên quan:

- **Ngày 29 giữ nguyên ngày 29** trong tháng 30 ngày (mặc định, `⚠️ Not confirmed` ở `requirements.md`); chỉ ngày 30 được kéo về cuối tháng.
- Tham số lặp sai trong DB (kind lạ, sự kiện lặp có thời lượng ≥ 24h, năm ngoài 1900..2100): bỏ sự kiện đó, log cảnh báo kèm `id`, không làm hỏng cả response. `Verify` đã chặn dữ liệu sai lúc seed; đây là lưới an toàn cho dữ liệu BO sửa sai sau này.

## Response shape

```json
{
  "temple_id": "temple_north_001",
  "from": "2026-10-13",
  "to": "2027-01-13",
  "events": [
    {
      "id": "tevt_north_001_sam_hoi",
      "title": "Lễ sám hối",
      "description": "…",
      "kind": "KHAC",
      "is_all_day": false,
      "status": "SCHEDULED",
      "audience_note": "",
      "contact_info": "",
      "image_url": null,
      "recurrence": { "kind": "LUNAR_MONTHLY", "lunar_days": [14, 30], "weekday": null, "until": null }
    }
  ],
  "occurrences": [
    {
      "event_id": "tevt_north_001_sam_hoi",
      "starts_at": "2026-11-23T19:00:00+07:00",
      "ends_at": "2026-11-23T21:00:00+07:00",
      "solar_date": "2026-11-23",
      "lunar_date": { "day": 14, "month": 10, "year": 2026, "is_leap_month": false }
    }
  ]
}
```

Mỗi `occurrence` trỏ về `events[].id`; chỉ trả các sự kiện có ít nhất một occurrence trong cửa sổ (tránh lặp mô tả của sự kiện lặp). `from`/`to` là cửa sổ hiệu lực sau khi áp mặc định. Response phụ thuộc "bây giờ" nên handler gửi `Cache-Control: no-cache`.

## Request flow

Hai lần đọc DB nối tiếp rồi một bước tính, nên flow chi tiết ở **Step-by-step Flow** bên dưới và bản vẽ ở `flow.md`.

> Xem `flow.md` cho sơ đồ sequence Mermaid (mirror 1:1 của Step-by-step Flow bên dưới).

## Step-by-step Flow — Lấy sự kiện của một chùa (`GET /v1/temples/{id}/events`)

```
Step 1 — Xác thực (middleware)
- Trigger: request tới route
- Pre-condition: không
- Call: authMiddleware.AuthenticateFunc (RequireTokenType "access")
    - Success → Step 2 (user_id trong context; usecase không dùng)
    - Failure (thiếu / hết hạn / sai loại token) → 401, dừng (TE2)

Step 2 — Parse request (handler)
- Trigger: Step 1 pass
- Pre-condition: path có {id}; query from, to tuỳ chọn
- Call: handler đọc {id}, from, to (chuỗi) → usecase.EventsForTemple
    - Success → Step 3
    - Failure → không có nhánh riêng; kiểm tra định dạng nằm ở Step 3

Step 3 — Dựng và kiểm tra cửa sổ (usecase, thuần)
- Trigger: Step 2
- Pre-condition: cấu hình temple_event hợp lệ (đã Validate lúc khởi động)
- Call: from rỗng → hôm nay (temple_event.timezone); to rỗng → from + default_window_months;
  parse YYYY-MM-DD; năm 1900..2100; to ≥ from; to − from ≤ max_window_months
    - Success → Step 4
    - Failure (định dạng/năm sai) → ErrInvalidDate → 400 INVALID_DATE (TE19)
    - Failure (to < from) → ErrInvalidDateRange → 400 INVALID_DATE_RANGE (TE19)
    - Failure (quá dài) → ErrDateRangeTooLarge → 400 DATE_RANGE_TOO_LARGE (TE18)

Step 4 — Kiểm tra chùa tồn tại
- Trigger: Step 3 pass
- Pre-condition: cửa sổ hợp lệ
- Call: TempleRepository.FindByID(ctx, id) → SELECT … FROM temples WHERE id = $1
    - Success → Step 5
    - Failure (không có dòng) → ErrTempleNotFound → 404 TEMPLE_NOT_FOUND (TE3)
    - Failure (lỗi DB) → 500, log
    - Transaction boundary: không; các bước 4–5 là hai lần đọc độc lập, chỉ đọc

Step 5 — Đọc sự kiện của chùa
- Trigger: Step 4 pass
- Pre-condition: chùa tồn tại
- Call: TempleEventRepository.ListActiveByTemple(ctx, templeID, window) →
  SELECT … FROM temple_events WHERE temple_id=$1 AND is_active
    AND starts_at < to + 2 ngày
    AND (recurrence <> 'NONE' OR ends_at >= from − 1 ngày)
    AND (recurrence_until IS NULL OR recurrence_until >= from − 1 ngày)
  — bộ lọc thô, rộng hơn cửa sổ một đến hai ngày để repository không cần biết múi giờ; lọc chính xác ở Step 6
    - Success → Step 6
    - Failure (lỗi DB) → 500, log

Step 6 — Sinh occurrence (thuần, không I/O)
- Trigger: Step 5 pass
- Pre-condition: danh sách sự kiện đã nạp; "bây giờ" lấy từ đồng hồ tiêm vào usecase
- Call: ResolveTempleOccurrences(events, window, now, loc) — mở rộng quy tắc lặp (mục "Sinh occurrence"),
  bỏ occurrence đã kết thúc, lọc theo cửa sổ, tính ngày âm, sắp xếp (TE8–TE14, TE20–TE22)
    - Success → Step 7
    - Failure (tham số lặp sai trong DB) → bỏ sự kiện đó, log cảnh báo kèm id; không hỏng response

Step 7 — Trả kết quả
- Trigger: Step 6
- Pre-condition: không
- Call: handler dựng DTO: events (chỉ sự kiện có occurrence) + occurrences
    - Success → 200 + Cache-Control: no-cache (TE1, TE4–TE7, TE15, TE16)
```

## Step-by-step Flow — Nạp dữ liệu sự kiện chùa (`cmd/seedtempleevents`)

```
Step 1 — Đọc và giải mã file
- Trigger: người vận hành chạy `go run ./services/appService/cmd/seedtempleevents` (hoặc make seed-temple-events)
- Pre-condition: có assets/data/temple-events.json; DB đã migrate và đã có bảng temples được seed
- Call: parse.go — strict decode (khoá lạ bị từ chối), NFC, `source` bắt buộc
    - Success → Step 2
    - Failure (JSON hỏng / khoá lạ / thiếu source) → thoát lỗi, không ghi gì (TE24)

Step 2 — Kiểm tra nội dung (thuần)
- Trigger: Step 1
- Pre-condition: không
- Call: TempleEventCatalogue.Verify(loc) — id duy nhất; kind/status/recurrence hợp lệ; tham số lặp đủ
  theo recurrence; lunar_days 1..30; weekday 0..6; ends_at ≥ starts_at; sự kiện lặp có thời lượng < 24h;
  recurrence_until không trước ngày đầu; mọi ngày trong 1900..2100; cả ngày chỉ NONE hoặc một ngày
    - Success → Step 3
    - Failure → thoát lỗi, liệt kê id sự kiện sai (TE24)

Step 3 — Kiểm tra chùa tồn tại và ghi (một transaction)
- Trigger: Step 2 pass (và không phải -dry-run)
- Pre-condition: kết nối DB sẵn sàng
- Call: TempleEventRepository.ReplaceCatalogue(ctx, catalogue): BEGIN;
  SELECT id FROM temples; mỗi temple_id của sự kiện phải có trong tập đó;
  INSERT … ON CONFLICT (id) DO UPDATE (không ghi is_active); COMMIT
    - Success → in số sự kiện đã upsert
    - Failure (temple_id không có trong temples) → ROLLBACK, lỗi nêu id sự kiện (TE24)
    - Failure (lỗi DB) → ROLLBACK, thoát lỗi
    - Transaction boundary: toàn bộ Step 3 là MỘT transaction; lỗi ở bất kỳ dòng nào huỷ cả file
```

## Known gaps carried by this feature

- **Dữ liệu mock hết hạn**: sự kiện một lần có ngày cố định sẽ thành "đã qua" và biến mất khỏi API (TE14). File mock nên chủ yếu là sự kiện lặp, cộng vài sự kiện một lần ở các tháng gần; cần đặt lại ngày khi cũ. Dữ liệu mock gắn vào chùa thật, **phải thay bằng nội dung thật** trước khi phát hành (quyết định 2026-10-13: không cần đánh dấu mock lúc develop).
- **Seed không xoá** sự kiện có trong DB mà vắng trong file, và không ghi `is_active` — giữ những gì BO sẽ chỉnh tay (cùng lựa chọn với `calendar`). Muốn bỏ một sự kiện mock thì xoá/tắt trong DB.
- **Chưa có hoãn/huỷ một lần** của sự kiện lặp; cần bảng ngoại lệ `temple_event_exceptions (event_id, date, status)` sau này.
- **Không trả ngày âm của ngày kết thúc** cho sự kiện nhiều ngày; chỉ `lunar_date` của ngày bắt đầu.
- **`TempleRepository.FindByID` đọc cả dòng chùa** chỉ để biết chùa có tồn tại; chấp nhận được vì chùa ít (hàng trăm) và đã có sẵn; đổi sang truy vấn `EXISTS` khi có số liệu đo.
- **Chưa có cột `is_active` trên `temples`**: SRS FR-010 "chùa bị ẩn" quy về "chùa không tồn tại".
- **Ngưỡng thời gian phản hồi (NFR-001) chưa chốt**, chưa có benchmark.
- **CLI seed đọc config** (`temple_event.timezone`) kể cả khi `-dry-run`, vì múi giờ là một phần của việc kiểm tra sự kiện cả ngày; khác `seedcalendarevents` chỉ đọc config khi ghi.
- Hai route `/v1/temples/…` công khai và route này cần đăng nhập cùng tiền tố; cần ghi chú ở `cmd/main.go` (đã có trong tasks Phase 5).
