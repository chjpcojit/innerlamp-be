# Design — calendar

Version: 1.1
Last updated: 2026-10-07
Status: implemented + verified against a running server (curl, port tạm)
Module: `calendar`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Một route công khai, chỉ đọc: `GET /v1/calendar/image`. Chạm đủ bốn layer của `services/appService` nhưng **không có DB**: "repository" ở đây là bộ phân tích `manifest.json` nhúng sẵn, danh sách ảnh nằm trong bộ nhớ, bất biến sau khi khởi động. Cùng hình dạng public-read với `gallery`. Không có migration.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/calendar.go` | `CalendarImage` (id, file, author, license, license_url, source_url), `ErrInvalidDate`. Không có tag JSON. |
| Repository (parser) | `services/appService/repository/calendar_manifest.go` | `ParseCalendarManifest([]byte)` đọc `manifest.json`, kiểm tra (không rỗng, tên file hợp lệ, không trùng, có giấy phép), sắp theo tên file. |
| Usecase | `services/appService/usecase/calendar_usecase.go` | `CalendarUsecase.ImageForDate(date string)`: bỏ trống → hôm nay theo `calendar.timezone`; kiểm tra ngày; chọn ảnh theo `dayNumber mod N`. Đồng hồ được tiêm vào để test. |
| Delivery | `services/appService/delivery/http/calendar_handler.go` | `GET /v1/calendar/image`; DTO riêng; dựng `image_url` tuyệt đối; đặt `Cache-Control`. |
| Delivery / errors | `services/appService/delivery/http/errors.go` | `ErrInvalidDate` → `400 INVALID_DATE`, `details.field = "date"`. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `CalendarConfig{Timezone, ImageBaseURL}`. |
| Assets | `assets/embed.go` | Nhúng `images/home-calendar/manifest.json` (`CalendarManifestJSON`). |
| Wiring | `services/appService/cmd/main.go` | Parse manifest, dựng usecase + handler, đăng ký route công khai. |
| API spec | `docs/specs/api/openapi.yaml` | Thêm `/v1/calendar/image`, tag `Calendar`. |

## Request flow

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

Không có Step-by-step Flow: không có hai lời gọi ngoài hay tác dụng phụ nào nối tiếp nhau (chỉ tính toán trong bộ nhớ), nên không có `flow.md`.

## Chọn ảnh theo ngày

- `dayNumber` = số ngày nguyên từ 1970-01-01 đến `date` (UTC, `floor(unix/86400)`); `index = ((dayNumber % N) + N) % N` để năm < 1970 vẫn dương.
- Ảnh sắp theo tên file (`buddha_001.jpg` …) để kết quả không phụ thuộc thứ tự trong manifest.
- Ngày liên tiếp → ảnh liên tiếp (AC3), chu kỳ N ngày. Thêm/bớt ảnh làm đổi ánh xạ từ lúc đó — chấp nhận.

## Known gaps carried by this feature

- `image_url` mặc định dựng từ `Host` của request: tiện cho dev (IP LAN đổi liên tục) nhưng **không được dùng cho production** — production đặt `calendar.image_base_url` trỏ CDN. Ảnh phục vụ qua `/static/images/` vẫn là dev-only.
- Chưa có thumbnail (xem `requirements.md` — Pending).
