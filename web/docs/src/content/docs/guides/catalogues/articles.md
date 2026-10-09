---
title: Bài viết Phật pháp
description: "Endpoint đọc bài viết Phật pháp: nổi bật, danh sách phân trang và chi tiết."
---

Danh mục bài viết **chỉ đọc**, không cần đăng nhập.

| Nổi bật | Danh sách phân trang | Chi tiết |
| --- | --- | --- |
| `GET /v1/articles/top` | `GET /v1/articles?page=N` | `GET /v1/articles/{id}` |

## Luồng dùng thường gặp

1. Màn hình chủ: gọi `top` để hiển thị mục nổi bật (5 bài).
2. Màn hình danh sách: gọi danh sách với `page`, cuộn tới đâu tải trang kế tiếp tới đó.
3. Màn hình chi tiết: gọi `{id}` với id lấy từ `top` hoặc danh sách. Danh sách và `top` trả bản tóm tắt, chi tiết trả đầy đủ hơn.

## `ArticleSummary` (mỗi phần tử trong `items`)

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | string | Slug ổn định, vd. `vipassana-nghe-thuat-song`. Dùng cho `GET /v1/articles/{id}`. |
| `title` | string | |
| `category` | string | Chủ đề, vd. `Thiền định`. |
| `author` | string | |
| `author_initials` | string | Chữ viết tắt, dùng cho ảnh đại diện. |
| `published_date` | string \| null | `"YYYY-MM-DD"`. **`null` ở khoảng một nửa số bài** (nguồn không có ngày). |
| `read_minutes` | int | Số phút đọc ước tính. |
| `summary` | string | Đoạn tóm tắt. |
| `tags` | string[] | `[]` nếu không có. |
| `image_url` | string \| null | `null` ở 22/60 bài; còn lại là URL ảnh từ trang nguồn. |

`top` trả 5 bài, danh sách trả 10 bài mỗi trang. Thứ tự là thứ tự biên tập của dữ liệu nguồn, không theo ngày hay độ phổ biến, nên đừng sắp xếp lại theo `published_date`.

## `GET /v1/articles/{id}` thêm

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `body_paragraphs` | string[] | Nội dung bài, mỗi phần tử một đoạn. |
| `quote` | string | Câu trích nổi bật. |
| `quote_source` | string | Người hoặc nguồn của câu trích. |
| `language` | string | Hiện là `vi` hoặc `en`. |
| `source_name` | string | Tên nguồn bài gốc. |
| `source_url` | string | Đường dẫn tới bài gốc. |
| `license` | string | Hiện là `all-rights-reserved-summarized`: nội dung là bản tóm tắt bài của bên khác. |

Mảng không bao giờ là `null`. Vì mỗi bài đều trỏ tới nguồn gốc, hãy quyết định với bên sản phẩm cách hiển thị `source_name` và `source_url` trước khi phát hành.

## Phân trang

```bash
curl 'http://localhost:8080/v1/articles?page=2'
```

`data` có dạng `{ "items": [...], "page", "page_size", "total", "total_pages" }`.

- `page` bắt đầu từ 1 và mặc định là 1. Giá trị không phải số nguyên dương trả `400 INVALID_PAGE`.
- Mỗi trang 10 mục (cấu hình dev). Dùng `total_pages` để biết khi nào dừng, đừng hard-code.
- Trang vượt quá cuối trả `200` với `items: []`, không phải lỗi.

## Ảnh

`image_url` là `null` khi chưa có ảnh, hiện là `null` ở 22/60 bài. Hãy xử lý dạng Optional và hiển thị ảnh thay thế.

## Không tìm thấy

Id không tồn tại trả `404` với `ARTICLE_NOT_FOUND`.

Xem cấu trúc đầy đủ của response trong [API Reference](/api-reference/). Envelope và mã lỗi chung: [Xử lý lỗi](/guides/error-handling/).
