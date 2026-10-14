# Design — temple-schedule

Version: 1.1
Last updated: 2026-10-13
Status: Phase 1–5 đã code (xem `tasks.md`); Phase 6 còn lại là tài liệu và kiểm tra tay
Module: `temple-schedule`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Một route đọc **cần đăng nhập** `GET /v1/temples/{id}/schedule`, chạm đủ bốn layer của `services/appService`, thêm hai bảng cha/con (`temple_schedules`, `temple_schedule_steps`) và một CLI seed. Hình dạng giống `temple-event` (kiểm tra chùa tồn tại bằng `TempleRepository.FindByID` có sẵn, đọc DB, trả về) nhưng **không có phần tính toán lịch**: không occurrence, không lịch âm, không múi giờ, không cửa sổ ngày, nên không cần khối config mới.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/temple_schedule.go` (new) | `TempleScheduleItem`, `TempleScheduleStep`, `TimeOfDay` (phút từ 00:00, parse/format `HH:MM`). Không tag JSON. |
| Domain / Errors | `services/appService/domain/temple_schedule.go` | `ErrInvalidTempleSchedule` (cho `Verify`). Dùng lại `ErrTempleNotFound`. |
| Domain / Ordering | `services/appService/domain/temple_schedule.go` | `SortTempleSchedule(items)` — thuần: `start_time`, `sort_order`, `id`; chuẩn hoá `weekdays` (bỏ trùng, sắp, đủ 7 → rỗng); sắp bước theo `position`. |
| Domain / Catalogue | `services/appService/domain/temple_schedule.go` | `TempleScheduleCatalogue{Source, Items}` + `Verify() error` (bọc `ErrInvalidTempleSchedule`, thông điệp liệt kê mọi lỗi; khác `CalendarCatalogue.Verify` trả `[]string`). |
| Domain / Repository interface | `services/appService/domain/temple_schedule.go` | `TempleScheduleRepository{ListActiveByTemple, ReplaceCatalogue}`. |
| Domain / Mock | `services/appService/domain/mock_temple_schedule_repository.go` (new) | Mock viết tay cho test usecase. |
| Usecase | `services/appService/usecase/temple_schedule_usecase.go` (new) | `ScheduleForTemple(ctx, templeID)`: kiểm tra chùa, đọc, sắp xếp. Phụ thuộc `TempleRepository` + `TempleScheduleRepository` (interface); không nhận logger (không có nhánh nào cần log: lỗi được bọc và handler ánh xạ). |
| Repository | `services/appService/repository/postgres_temple_schedule_repository.go` (new) | Đọc buổi active của một chùa + bước của chúng; `ReplaceCatalogue` upsert trong một transaction. |
| Delivery / Handler | `services/appService/delivery/http/temple_schedule_handler.go` (new) | Parse `{id}` → usecase → DTO. Không logic nghiệp vụ. |
| Delivery / DTO | `services/appService/delivery/http/temple_schedule_handler.go` | DTO response (`templeScheduleItemResponse`, `templeScheduleStepResponse`). Tag JSON chỉ ở đây. |
| Seed CLI | `services/appService/cmd/seedtempleschedules/` (new: `main.go`, `parse.go`, `parse_test.go`) | Đọc `assets/data/temple-schedules.json` (`{meta:{format:1, source}, items:[…]}`), strict decode, `Verify`, `ReplaceCatalogue`; cờ `-dry-run` như `seedcalendarevents`. |
| Data | `assets/data/temple-schedules.json` (new) | Dữ liệu **mock** dựa trên id trong `assets/data/temple.json`. |
| Wiring | `services/appService/cmd/main.go` | Dựng repo/usecase/handler; đăng ký route bọc `authMiddleware.AuthenticateFunc`. |
| Migration | `migrations/0034_temple_schedules.sql` | Hai bảng + index; là số kế tiếp thực tế khi viết (`0033` là lớn nhất). Nếu `temple-event` được viết sau thì dùng số kế tiếp của nó, không phải số này. |
| Build | `Makefile` | `ensure-temple-schedules`, `seed-temple-schedules`, thêm vào `ensure-data`. |
| API spec | `docs/specs/api/openapi.yaml` | Route + schema, cùng lúc với handler. |

Dependency rule: delivery → usecase → domain; repository → domain; domain không import tầng ngoài.

## Mô hình dữ liệu

Hai bảng cha/con. Một **buổi** có nhiều **bước**; mỗi bước thuộc đúng một buổi và bị xoá theo buổi.

### `temple_schedules` (buổi)

| Cột | Kiểu | Ghi chú |
|---|---|---|
| `id` | `VARCHAR(64)` PK | Mã ổn định do người soạn seed đặt (vd `tsch_north_001_cong_phu_sang`); seed upsert theo cột này. |
| `temple_id` | `VARCHAR(64)` NOT NULL, FK `temples(id)` `ON DELETE CASCADE` | Chùa sở hữu. |
| `name` | `VARCHAR(255)` NOT NULL | CHECK không rỗng sau `btrim`. |
| `start_time` | `TIME` NOT NULL | Giờ trong ngày, giờ Việt Nam, không ngày, không múi giờ. |
| `end_time` | `TIME` NULL | CHECK `end_time IS NULL OR end_time <> start_time`; được phép nhỏ hơn `start_time` (qua nửa đêm). |
| `weekdays` | `SMALLINT[]` NOT NULL DEFAULT `'{}'` | Rỗng = mỗi ngày. `0` = Chủ nhật … `6` = Thứ bảy (khớp `time.Weekday`). CHECK mỗi phần tử trong 0..6. |
| `description` | `TEXT` NOT NULL DEFAULT `''` | |
| `instructor` | `VARCHAR(255)` NOT NULL DEFAULT `''` | Văn bản tự do, áp cho cả buổi. |
| `location` | `VARCHAR(255)` NOT NULL DEFAULT `''` | Văn bản tự do, vd "Chánh điện". |
| `preparation` | `TEXT` NOT NULL DEFAULT `''` | Cần chuẩn bị gì. |
| `sort_order` | `INTEGER` NOT NULL DEFAULT 0 | Phá thế hoà khi cùng `start_time`. |
| `is_active` | `BOOLEAN` NOT NULL DEFAULT TRUE | Tắt = "huỷ thời khoá"; seed không ghi cột này. |
| `created_at`, `updated_at` | `TIMESTAMPTZ` NOT NULL DEFAULT `now()` | |

Index `idx_temple_schedules_temple (temple_id) WHERE is_active`.

### `temple_schedule_steps` (bước)

| Cột | Kiểu | Ghi chú |
|---|---|---|
| `schedule_id` | `VARCHAR(64)` NOT NULL, FK `temple_schedules(id)` `ON DELETE CASCADE` | Buổi sở hữu. |
| `position` | `SMALLINT` NOT NULL, CHECK `position >= 1` | Thứ tự do người nhập đặt. |
| `step_time` | `TIME` NULL | Giờ của bước, tuỳ chọn; **không** quyết định thứ tự. |
| `text` | `TEXT` NOT NULL, CHECK không rỗng sau `btrim` | |

Khoá chính `(schedule_id, position)` — chặn hai bước cùng vị trí trong một buổi (kịch bản "hai bước cùng vị trí").

## Quy ước thứ tự và chuẩn hoá

- `items`: `start_time` tăng dần, rồi `sort_order`, rồi `id`. Buổi qua nửa đêm xếp theo giờ bắt đầu của nó.
- `steps`: theo `position` tăng dần, **không** theo `step_time` (TS12).
- `weekdays`: bỏ trùng, sắp tăng dần; nếu đủ cả 7 thì trả `[]` giống "không chọn" (TS9). Việc chuẩn hoá ở `SortTempleSchedule` để nhất quán dù BO nhập kiểu nào.
- Giờ bước không bị ràng buộc nằm trong `start_time`–`end_time` của buổi (giả định SRS 2026-10-13).

## Response shape

```json
{
  "temple_id": "temple_north_001",
  "items": [
    {
      "id": "tsch_north_001_duoc_su",
      "name": "Lễ Dược Sư",
      "start_time": "08:00",
      "end_time": "11:00",
      "weekdays": [0],
      "description": "Lễ cầu an, tụng kinh Dược Sư.",
      "instructor": "Thượng toạ Thích Minh Tâm",
      "location": "Chánh điện",
      "preparation": "Mặc áo tràng, mang theo kinh sách.",
      "steps": [
        { "position": 1, "time": "08:00", "text": "Niệm Phật, ổn định tâm" },
        { "position": 2, "time": "08:30", "text": "Tụng kinh Dược Sư" },
        { "position": 3, "time": null, "text": "Hồi hướng" }
      ]
    }
  ]
}
```

`weekdays: []` nghĩa là mỗi ngày. Trường văn bản trống trả `""` (giống quy ước của `temples`), `end_time` và `time` của bước trả `null`, `steps` trống trả `[]`. Không đặt header cache riêng ở v1 (dữ liệu chỉ đổi khi vận hành viên nạp lại).

## Request flow

Có hai lần đọc DB nối tiếp (buổi, rồi bước), nên flow chi tiết ở **Step-by-step Flow** bên dưới và bản vẽ ở `flow.md`.

> Xem `flow.md` cho sơ đồ sequence Mermaid (mirror 1:1 của Step-by-step Flow bên dưới).

## Step-by-step Flow — Lấy thời khoá của một chùa (`GET /v1/temples/{id}/schedule`)

```
Step 1 — Xác thực (middleware)
- Trigger: request tới route
- Pre-condition: không
- Call: authMiddleware.AuthenticateFunc (RequireTokenType "access")
    - Success → Step 2
    - Failure (thiếu / hết hạn / sai loại token) → 401, dừng (TS2)

Step 2 — Parse request (handler)
- Trigger: Step 1 pass
- Pre-condition: path có {id}
- Call: handler đọc {id} → usecase.ScheduleForTemple
    - Success → Step 3
    - Failure → không có nhánh riêng (không có tham số query)

Step 3 — Kiểm tra chùa tồn tại
- Trigger: Step 2
- Pre-condition: không
- Call: TempleRepository.FindByID(ctx, id) → SELECT … FROM temples WHERE id = $1
    - Success → Step 4
    - Failure (không có dòng) → ErrTempleNotFound → 404 TEMPLE_NOT_FOUND (TS3)
    - Failure (lỗi DB) → 500, log
    - Transaction boundary: không; Step 3–5 là các lần đọc độc lập, chỉ đọc

Step 4 — Đọc các buổi active của chùa
- Trigger: Step 3 pass
- Pre-condition: chùa tồn tại
- Call: TempleScheduleRepository.ListActiveByTemple(ctx, templeID) →
  SELECT … FROM temple_schedules WHERE temple_id = $1 AND is_active
    - Success → Step 5 (danh sách rỗng → bỏ qua Step 5, sang Step 6)
    - Failure (lỗi DB) → 500, log

Step 5 — Đọc bước của các buổi đó
- Trigger: Step 4 trả ≥ 1 buổi
- Pre-condition: có danh sách id buổi
- Call: (trong cùng ListActiveByTemple) SELECT … FROM temple_schedule_steps
  WHERE schedule_id = ANY($1) ORDER BY schedule_id, position
    - Success → Step 6
    - Failure (lỗi DB) → 500, log

Step 6 — Ghép và sắp xếp (thuần, không I/O)
- Trigger: Step 4/5 pass
- Pre-condition: buổi và bước đã nạp
- Call: SortTempleSchedule(items): ghép bước vào buổi, sắp bước theo position, chuẩn hoá weekdays,
  sắp buổi theo start_time, sort_order, id (TS6, TS9, TS12–TS14)
    - Success → Step 7
    - Failure → không có nhánh lỗi (hàm thuần trên dữ liệu đã đọc)

Step 7 — Trả kết quả
- Trigger: Step 6
- Pre-condition: không
- Call: handler dựng DTO {temple_id, items}
    - Success → 200 (TS1, TS4, TS7, TS8)
```

## Step-by-step Flow — Nạp dữ liệu thời khoá (`cmd/seedtempleschedules`)

```
Step 1 — Đọc và giải mã file
- Trigger: người vận hành chạy `go run ./services/appService/cmd/seedtempleschedules` (hoặc make seed-temple-schedules)
- Pre-condition: có assets/data/temple-schedules.json; DB đã migrate và đã có bảng temples được seed
- Call: parse.go — strict decode (khoá lạ bị từ chối), NFC, `source` bắt buộc
    - Success → Step 2
    - Failure (JSON hỏng / khoá lạ / thiếu source) → thoát lỗi, không ghi gì (TS17)

Step 2 — Kiểm tra nội dung (thuần)
- Trigger: Step 1
- Pre-condition: không
- Call: TempleScheduleCatalogue.Verify() — id duy nhất; name không rỗng; start/end/step time đúng HH:MM;
  end ≠ start; weekdays trong 0..6; text bước không rỗng; position bước ≥ 1 và không trùng trong một buổi
    - Success → Step 3
    - Failure → thoát lỗi, nêu id buổi sai (TS17)

Step 3 — Kiểm tra chùa tồn tại và ghi (một transaction)
- Trigger: Step 2 pass (và không phải -dry-run)
- Pre-condition: kết nối DB sẵn sàng
- Call: TempleScheduleRepository.ReplaceCatalogue(ctx, catalogue): BEGIN;
  SELECT id FROM temples; mỗi temple_id của buổi phải có trong tập đó;
  INSERT INTO temple_schedules … ON CONFLICT (id) DO UPDATE (không ghi is_active);
  DELETE FROM temple_schedule_steps WHERE schedule_id = ANY(id các buổi trong file);
  INSERT bước mới; COMMIT
    - Success → in số buổi và số bước đã nạp (TS16, TS18)
    - Failure (temple_id không có trong temples) → ROLLBACK, lỗi nêu id buổi (TS17)
    - Failure (lỗi DB) → ROLLBACK, thoát lỗi
    - Transaction boundary: toàn bộ Step 3 là MỘT transaction; lỗi ở bất kỳ dòng nào huỷ cả file
```

## Known gaps carried by this feature

- **Seed không xoá** buổi có trong DB mà vắng trong file, và không ghi `is_active` — giữ những gì BO sẽ chỉnh tay (cùng lựa chọn với `calendar` và `temple-event`). Muốn bỏ một buổi mock thì xoá/tắt trong DB. Ngược lại, **bước của buổi có trong file bị thay hoàn toàn** (TS18): sửa tay bước của một buổi mock rồi nạp lại sẽ mất sửa đổi đó.
- **Dữ liệu mock gắn vào chùa thật**, phải thay bằng nội dung thật trước khi phát hành (quyết định 2026-10-13: không cần đánh dấu mock lúc develop). Khác `temple-event`, thời khoá không hết hạn theo ngày.
- **Người hướng dẫn và địa điểm là văn bản tự do ở mức buổi**: cùng một vị có thể gõ nhiều kiểu; không có trang vị sư; buổi có nhiều người dẫn khác nhau theo bước phải ghi gộp vào `description`.
- **Thứ trong tuần dùng `0` = Chủ nhật** (theo `time.Weekday`), khác ISO (`1` = Thứ hai); client iOS cần ánh xạ — ghi trong `api-contract.md` khi có.
- **Không có đa mùa**: một chùa chỉ có một thời khoá; chuyển mùa an cư thì BO sửa buổi.
- **Không có thay đổi một lần** ("nghỉ công phu chiều tuần này"): thuộc `temple-event`.
- **`TempleRepository.FindByID` đọc cả dòng chùa** chỉ để biết chùa có tồn tại (như `temple-event`); đổi sang `EXISTS` khi có số liệu đo.
- **Số migration phụ thuộc thứ tự**: `temple-schedule` được code trước nên lấy `0034`; `temple-event` (chưa code) sẽ lấy số kế tiếp, spec của nó đang ghi `0034` và cần sửa khi làm.
- **Response không kèm tên chùa**; client đã có từ chi tiết chùa.
- **Chưa có ngưỡng thời gian phản hồi** và chưa có header cache.
- Route này cần đăng nhập cùng tiền tố với các route `/v1/temples` công khai; cần ghi chú ở `cmd/main.go` (có trong tasks Phase 5).
