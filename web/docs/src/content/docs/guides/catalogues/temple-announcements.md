---
title: Thông báo của chùa
description: "Endpoint xem thông báo của ngôi chùa mình quy y và bật/tắt việc nhận thông báo (cần đăng nhập)."
---

Mỗi ngôi chùa có thể đăng **thông báo** cho những người đang quy y tại chùa đó, ví dụ "ghi danh khoá tu mùa hè còn 10 ngày". Thông báo là dữ liệu có hạn hiển thị trong app/web: **không** phải push của hệ điều hành, không phải [sự kiện của chùa](/guides/catalogues/temple-events/) và không phải [thời khoá tu tập](/guides/catalogues/temple-schedule/).

Người dùng thấy thông báo của **chùa quy y hiện tại** (`ordination_temple_id` ở [Hồ sơ](/guides/catalogues/profile/)). Không có tham số nào chọn chùa hay người dùng: đổi hoặc gỡ chùa quy y thì từ lần đọc kế tiếp danh sách đổi theo.

| Thao tác | Endpoint | Đăng nhập |
| --- | --- | --- |
| Thông báo còn hiệu lực của chùa tôi | `GET /v1/users/me/temple-announcements?page=&page_size=` | **cần** |
| Xem công tắc nhận thông báo | `GET /v1/users/me/temple-announcement-settings` | **cần** |
| Bật/tắt nhận thông báo | `PUT /v1/users/me/temple-announcement-settings` | **cần** |

Cả ba route cần `Authorization: Bearer <access_token>` (refresh token trả `401 UNAUTHORIZED`) và luôn trả `Cache-Control: no-store`.

```bash
curl 'http://localhost:8080/v1/users/me/temple-announcements?page=1' \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

## Danh sách thông báo

Trả các thông báo **còn hiệu lực**, mới đăng nhất trước. Thông báo hết hạn (từ đúng lúc `expires_at`) hoặc đã bị chùa thu hồi không còn xuất hiện. Không có trạng thái đã đọc/chưa đọc và không có số chưa đọc.

```json
{
  "success": true,
  "data": {
    "receive": true,
    "temple": { "id": "temple_north_011", "name": "Chùa Bích Động" },
    "items": [
      {
        "id": "tann_5b0f4a52-3c1e-4a37-9f4e-0c6a7d3b1e21",
        "title": "Ghi danh khoá tu mùa hè",
        "body": "Chùa nhận ghi danh khoá tu mùa hè đến hết ngày 30/10.",
        "published_at": "2026-10-14T03:00:00Z",
        "expires_at": "2026-10-30T16:59:59Z",
        "issuing_body": { "id": "tib_0c2e8f1a-77d4-4e55-8a4d-3f6a1b9c2d10", "name": "Ban Trị sự" }
      }
    ],
    "page": { "page": 1, "page_size": 20, "total": 1 }
  }
}
```

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `receive` | bool | Công tắc đã lưu. Luôn có mặt để vẽ công tắc kể cả khi danh sách rỗng. |
| `temple` | object \| null | `{id, name}` của chùa quy y. `null` khi công tắc tắt hoặc người dùng chưa có chùa quy y. |
| `items` | array | Không bao giờ `null`; `[]` khi rỗng. |
| `items[].body` | string | Văn bản thuần, đừng render như HTML. |
| `items[].published_at`, `expires_at` | string | RFC 3339, UTC. |
| `items[].issuing_body` | object | Ban phát hành (`{id, name}`), vd "Ban Trị sự". Hiển thị như người gửi. |
| `page.page`, `page.page_size`, `page.total` | int | `total` là tổng thông báo còn hiệu lực của chùa. |

### Phân trang

- `page` bắt đầu từ 1, mặc định 1. `page_size` mặc định theo cấu hình server (dev: 20); giá trị lớn hơn mức tối đa (dev: 50) bị **chặn** về mức tối đa, không báo lỗi. Đọc `page.page_size` để biết kích thước đã dùng.
- Trang vượt cuối trả `200` với `items: []` và `total` đúng.
- `page` hoặc `page_size` không phải số nguyên, bằng `0` hoặc âm là `400 INVALID_PAGE` (`details.field` luôn là `page`, kể cả khi lỗi ở `page_size`).

### Ba trường hợp rỗng (đều là `200`, không phải lỗi)

| Trường hợp | `receive` | `temple` | `items` |
| --- | --- | --- | --- |
| Đã tắt nhận thông báo | `false` | `null` | `[]` |
| Chưa có chùa quy y | `true` | `null` | `[]` |
| Chùa chưa có thông báo nào còn hiệu lực | `true` | `{id, name}` | `[]` |

## Công tắc nhận thông báo

`GET` trả `{"success": true, "data": {"receive": true}}`; mặc định `true`. Có **một công tắc cho cả tài khoản**, dùng chung mọi thiết bị.

`PUT` nhận đúng một khoá:

```json
{ "receive": false }
```

- Trả giá trị đã lưu `{"data": {"receive": false}}`. Gửi lại cùng giá trị vẫn `200`.
- Tắt = ẩn hoàn toàn danh sách; thông báo không bị xoá và hiện lại khi bật.
- Đổi hoặc gỡ chùa quy y **không** đổi công tắc.
- Thiếu `receive` là `400 VALIDATION_ERROR` với `details.field = "receive"`; sai kiểu (không phải boolean), khoá lạ, body rỗng hoặc sai JSON cũng là `400 VALIDATION_ERROR` và công tắc không đổi.

## Giới hạn tần suất

Ba route này dùng chung **60 request mỗi phút mỗi tài khoản** (dev), tính cả đọc. Đây là bộ đếm **riêng**, không chung với nhóm [Hồ sơ](/guides/catalogues/profile/) (10 request/phút). Vượt giới hạn trả `429 RATE_LIMITED` kèm header `Retry-After` và `details.retry_after_seconds`; không có gì bị đọc hay đổi. Chỉ tải khi mở màn hình hoặc kéo để làm mới, đừng polling.

## Lỗi

| HTTP | `error.code` | Khi nào |
| --- | --- | --- |
| 400 | `INVALID_PAGE` | `page` hoặc `page_size` không hợp lệ ở danh sách. |
| 400 | `VALIDATION_ERROR` | Body của `PUT` sai (xem trên). |
| 401 | `UNAUTHORIZED` | Thiếu token, token sai, hết hạn hoặc không phải access token. |
| 413 | `PAYLOAD_TOO_LARGE` | Body của `PUT` quá 1 MB. |
| 429 | `RATE_LIMITED` | Vượt giới hạn tần suất của tài khoản. |
| 500 | `INTERNAL_ERROR` | Lỗi phía server. |

Xem thêm [Chùa](/guides/catalogues/temples/), [Sự kiện của chùa](/guides/catalogues/temple-events/), [Hồ sơ](/guides/catalogues/profile/) và [Xử lý lỗi](/guides/error-handling/).
