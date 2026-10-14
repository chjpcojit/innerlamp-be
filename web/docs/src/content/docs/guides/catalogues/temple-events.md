---
title: Sự kiện của chùa
description: "Endpoint xem sự kiện do một ngôi chùa tổ chức (khoá tu, pháp hội, lễ, thuyết pháp…) trong một khoảng ngày, kèm ngày âm và sự kiện lặp (cần đăng nhập)."
---

Mỗi ngôi chùa có **sự kiện của riêng mình**: khoá tu, pháp hội, lễ cầu an, cầu siêu, thuyết pháp, lễ hội… Đây là dữ liệu của từng chùa, **không liên quan** sự kiện Phật giáo chung của [Lịch âm](/guides/catalogues/calendar/) (Vu Lan, Phật Đản… dùng chung cho mọi chùa).

Cần phân biệt với [Thời khoá tu tập](/guides/catalogues/temple-schedule/): thời khoá là nếp sinh hoạt thường ngày (công phu sáng, thọ trai) và không có ngày cụ thể; sự kiện là chương trình có ngày giờ cụ thể, có thể bị hoãn hoặc huỷ.

| Thao tác | Endpoint | Đăng nhập |
| --- | --- | --- |
| Sự kiện của một chùa trong một khoảng ngày | `GET /v1/temples/{id}/events?from=&to=` | **cần** |

:::caution[Khác các route chùa còn lại]
`GET /v1/temples`, `/top`, `/options` và `/{id}` là công khai. Route `/events` nằm dưới cùng tiền tố nhưng **bắt buộc** `Authorization: Bearer <access_token>`. Thiếu token, token hết hạn hoặc là refresh token trả `401 UNAUTHORIZED`.
:::

```bash
curl 'http://localhost:8080/v1/temples/temple_north_001/events?from=2026-10-13&to=2027-01-13' \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

## Tham số

| Tham số | Bắt buộc | Ghi chú |
| --- | --- | --- |
| `from` | không | Ngày đầu `YYYY-MM-DD`, năm 1900..2100. Bỏ trống là **hôm nay** theo giờ Việt Nam. |
| `to` | không | Ngày cuối `YYYY-MM-DD` (bao gồm). Bỏ trống là `from` + 3 tháng. |

- Cửa sổ dài nhất **3 tháng** (cấu hình `temple_event.max_window_months`). Muốn xem tiếp thì gọi lại với `from` mới ("load more").
- `data.from` và `data.to` trả cửa sổ đã dùng.
- **Sự kiện đã kết thúc không được trả.** Sự kiện đang diễn ra hoặc còn lại trong hôm nay vẫn có. Vì phụ thuộc thời điểm hiện tại, response luôn là `Cache-Control: no-cache`.

## Response

`data` có dạng `{ "temple_id", "from", "to", "events": [...], "occurrences": [...] }`.

Nội dung mỗi sự kiện xuất hiện **một lần** trong `events`; mỗi lần diễn ra là một phần tử của `occurrences`, trỏ về `events[].id`. Sự kiện lặp mỗi tháng vì thế không lặp lại mô tả. Chỉ trả sự kiện có ít nhất một lần diễn ra trong cửa sổ.

### `events[]`

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | string | Vd. `tevt_north_001_sam_hoi`. |
| `title` | string | |
| `description` | string | `""` nếu không có. |
| `kind` | string | `KHOA_TU`, `PHAP_HOI`, `LE_CAU_AN`, `LE_CAU_SIEU`, `LE_HOI`, `THUYET_PHAP`, `KHAC`. Nên decode kiểu mở (giá trị lạ về "khác"). |
| `is_all_day` | bool | `true` thì không có giờ cụ thể. |
| `status` | string | `SCHEDULED`, `POSTPONED`, `CANCELLED`. **Sự kiện hoãn/huỷ vẫn được trả** để app hiển thị đúng trạng thái. |
| `audience_note` | string | Đối tượng tham dự, vd "thiếu nhi 8–15 tuổi". `""` nếu không có. |
| `contact_info` | string | Liên hệ/đăng ký, chỉ để hiển thị (app không đăng ký hộ). `""` nếu không có. |
| `image_url` | string \| null | Hiện là `null`. |
| `recurrence` | object | Xem bên dưới. |

### `events[].recurrence`

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `kind` | string | `NONE` (một lần), `WEEKLY` (mỗi tuần), `LUNAR_MONTHLY` (mỗi tháng âm). |
| `lunar_days` | int[] | Chỉ với `LUNAR_MONTHLY`: các ngày âm 1..30; còn lại `[]`. |
| `weekday` | int \| null | Chỉ với `WEEKLY`: `0` = Chủ nhật … `6` = Thứ bảy (không phải ISO). |
| `until` | string \| null | Ngày cuối còn lặp `YYYY-MM-DD` (bao gồm). `null` = lặp mãi. |

### `occurrences[]`

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `event_id` | string | Trỏ về `events[].id`. |
| `starts_at` | string | RFC 3339 theo giờ `Asia/Ho_Chi_Minh`, vd. `2026-11-23T19:00:00+07:00`. |
| `ends_at` | string | Như trên. Sự kiện cả ngày kết thúc `23:59:59` của ngày cuối. |
| `solar_date` | string | Ngày dương của **ngày bắt đầu** `YYYY-MM-DD`. |
| `lunar_date` | object | Ngày âm của ngày bắt đầu: `{day, month, year, is_leap_month}`. |

```json
{
  "success": true,
  "data": {
    "temple_id": "temple_north_001",
    "from": "2026-10-13",
    "to": "2027-01-13",
    "events": [
      {
        "id": "tevt_north_001_sam_hoi",
        "title": "Lễ sám hối",
        "description": "Lễ sám hối vào các ngày 14 và 30 (cuối tháng) âm lịch.",
        "kind": "KHAC",
        "is_all_day": false,
        "status": "SCHEDULED",
        "audience_note": "Phật tử",
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
}
```

## Quy ước client cần biết

- **Server tính sự kiện lặp, client không tự tính.** Không có "lặp mỗi tháng âm" ở phía app: chỉ việc hiển thị `occurrences`.
- **Ngày 30 âm trong tháng thiếu** (tháng chỉ có 29 ngày) rơi vào **ngày cuối tháng**; hai ngày 29 và 30 cùng đặt không sinh hai lần. Tháng nhuận được tính.
- **Sự kiện nhiều ngày** hiện ở mọi ngày nó trải qua: một khoá tu 3 ngày có một occurrence bắt đầu ngày đầu và kết thúc ngày cuối. **Sự kiện qua nửa đêm** là một occurrence, `solar_date` là ngày bắt đầu.
- **Hoãn/huỷ áp cho cả sự kiện**, chưa có hoãn/huỷ riêng một lần của sự kiện lặp.
- `occurrences` đã sắp theo `starts_at`, rồi `event_id`: hai lần gọi cùng cửa sổ cho cùng thứ tự.
- Chùa chưa có sự kiện nào trong cửa sổ trả `200` với `events: []` và `occurrences: []`, không phải lỗi.
- Dữ liệu hiện là **mock** do đội phát triển soạn; sự kiện một lần có ngày cố định sẽ biến mất khỏi API khi qua ngày.

## Lỗi

| HTTP | `error.code` | Khi nào |
| --- | --- | --- |
| 400 | `INVALID_DATE` | `from` hoặc `to` không phải ngày thật theo `YYYY-MM-DD` hoặc năm ngoài 1900..2100. `details.field` là `from` hoặc `to`. |
| 400 | `INVALID_DATE_RANGE` | `to` trước `from`. |
| 400 | `DATE_RANGE_TOO_LARGE` | Cửa sổ dài hơn 3 tháng. |
| 401 | `UNAUTHORIZED` | Thiếu token, token sai, hết hạn hoặc không phải access token. |
| 404 | `TEMPLE_NOT_FOUND` | Không có chùa với `id` này. |
| 500 | `INTERNAL_ERROR` | Lỗi phía server. |

Xem thêm [Chùa](/guides/catalogues/temples/), [Thời khoá tu tập](/guides/catalogues/temple-schedule/), [Lịch âm](/guides/catalogues/calendar/) và [Xử lý lỗi](/guides/error-handling/).
