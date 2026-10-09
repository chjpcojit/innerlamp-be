---
title: Pháp thoại
description: "Endpoint đọc pháp thoại (bài giảng của một vị thầy): danh sách phân trang và chi tiết theo loại media."
---

Pháp thoại là bài giảng của một vị thầy, **chỉ đọc**, không cần đăng nhập. Khác với [kinh](/guides/catalogues/sutras/).

| Danh sách phân trang | Chi tiết |
| --- | --- |
| `GET /v1/dharma-talks?page=N` | `GET /v1/dharma-talks/{id}` |

Phân trang 10 bài mỗi trang theo thứ tự biên tập. `total` chỉ đếm bài được hiển thị: bài đang chờ duyệt không bao giờ được trả ra. Trang vượt cuối trả `200` với `items: []`.

## `DharmaTalkSummary` (mỗi phần tử trong `items`)

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | string | Slug. Dùng cho `GET /v1/dharma-talks/{id}`. |
| `title`, `speaker`, `speaker_initials`, `category` | string | |
| `season` | string \| null | |
| `published_date` | string \| null | `YYYY-MM-DD`. |
| `duration_minutes` | int | |
| `summary` | string | |
| `tags` | string[] | |
| `media_type` | string | `youtube`, `audio` hoặc `text`: quyết định giao diện chi tiết. |

## Chi tiết

`GET /v1/dharma-talks/{id}` trả toàn bộ field của `DharmaTalkSummary`, thêm:

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `key_points` | string[] | Ý chính. `[]` nếu không có. |
| `quote`, `quote_source` | string | Câu trích ngắn và nguồn. |
| `language` | string | Hiện là `vi`. |
| `youtube_id` | string \| null | Mã 11 ký tự; chỉ có khi `media_type = youtube`. Client tự ghép khi nhúng player. Không phải URL đầy đủ. |
| `audio_url` | string \| null | Chỉ có khi `media_type = audio`. |
| `chapters` | object[] | `{start_seconds, title}` tăng dần. `[]` thì ẩn mục "Các phần"; luôn `[]` với `text`. |
| `follow_up_meditation` | object \| null | `{title, minutes, meditation_id}`. Hiện là `null` ở mọi bài. |
| `source_name`, `source_url`, `license` | string | Hiển thị thẻ nguồn ở cuối trang. |

## Giao diện theo `media_type`

- `youtube`: nhúng player từ `youtube_id`.
- `audio`: trình phát âm thanh từ `audio_url`.
- `text`: chỉ đọc, không có media.

## Lỗi

| HTTP | code | Khi nào |
| --- | --- | --- |
| 400 | `INVALID_PAGE` | `page` không phải số nguyên dương. |
| 404 | `DHARMA_TALK_NOT_FOUND` | Không có bài với id này, hoặc bài đang chờ duyệt. |

Cấu trúc đầy đủ: [API Reference](/api-reference/). Mã lỗi chung: [Xử lý lỗi](/guides/error-handling/).
