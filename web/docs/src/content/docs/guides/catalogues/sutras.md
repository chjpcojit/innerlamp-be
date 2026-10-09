---
title: Kinh, kệ, chú
description: "Endpoint đọc thư viện kinh, kệ, chú: danh mục có bộ lọc và tìm kiếm, chi tiết, đọc nội dung theo trang dòng, kinh liên quan đến Phật/Bồ tát."
---

Thư viện kinh, kệ, chú **chỉ đọc**, không cần đăng nhập. Khác với [pháp thoại](/guides/catalogues/dharma-talks/) (bài giảng của một vị thầy).

| Danh mục | Chi tiết | Nội dung | Theo Phật/Bồ tát |
| --- | --- | --- | --- |
| `GET /v1/sutras` | `GET /v1/sutras/{id}` | `GET /v1/sutras/{id}/lines?translation=` | `GET /v1/deities/{id}/sutras` |

## Luồng dùng thường gặp

1. Màn thư viện: gọi `GET /v1/sutras?page=N`, thêm bộ lọc hoặc `q` khi người dùng lọc/tìm.
2. Màn chi tiết: gọi `GET /v1/sutras/{id}` để lấy metadata, danh sách bản dịch (`translations[].key`) và mục lục phẩm (`sections`).
3. Màn đọc: chọn một bản dịch, gọi `GET /v1/sutras/{id}/lines?translation=<key>` và tải từng trang dòng khi cuộn.
4. Màn Phật/Bồ tát: gọi `GET /v1/deities/{id}/sutras` để hiện kinh, chú liên quan.

## Danh mục và bộ lọc

Phân trang 20 mục mỗi trang, theo thứ tự biên tập. Danh mục chỉ có metadata, **không có nội dung dòng**. Các bộ lọc kết hợp bằng AND:

| Tham số | Giá trị | Ghi chú |
| --- | --- | --- |
| `page` | số nguyên ≥ 1 | Mặc định 1. Sai trả `400 INVALID_PAGE`. |
| `tradition` | `THERAVADA`, `MAHAYANA`, `VAJRAYANA` | Tông phái. |
| `kind` | `SUTRA`, `VERSE`, `DHARANI` | Kinh, kệ, chú (chú đọc theo âm). |
| `collection` | vd. `Bát Nhã`, `Thập Chú`, `Trung Bộ` | Khớp chính xác. |
| `purpose` | `CAU_AN`, `CAU_SIEU`, `TRI_CHU`, `SAM_HOI`, `HOI_HUONG`, `PHAT_NGUYEN`, `BAO_HIEU`, `THIEN_DINH`, `HOC_GIAO_LY`, `QUY_Y_THO_GIOI` | Mục đích tụng đọc. Nhãn tiếng Việt do client tự ánh xạ. |
| `daily` | `true` / `false` | Chỉ kinh nhật tụng / chỉ kinh không phải nhật tụng. |
| `max_minutes` | số nguyên ≥ 1 | Chỉ lấy văn bản có `duration_minutes` ≤ giá trị này. |
| `lang` | vd. `vi` | Có ít nhất một bản dịch ngôn ngữ này. |
| `q` | 1–100 ký tự | Tìm trong `titles`, `collection`, `reference`. Khớp một phần, **không phân biệt hoa thường và dấu tiếng Việt** (`bat nha` = `Bát Nhã`). Thứ tự vẫn là thứ tự biên tập. |

Giá trị không hợp lệ (kể cả tham số có mặt nhưng để trống) trả `400 INVALID_FILTER`, `details.field` là tên tham số. Trang vượt cuối trả `200` với `items: []`.

### `SutraSummary` (mỗi phần tử trong `items`)

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | string | Slug, vd. `tam-kinh`. |
| `tradition`, `kind`, `collection`, `reference` | string | |
| `titles` | object | Tiêu đề theo ngôn ngữ: `vi`, `zh`, `sa`, `en`... |
| `daily_chant` | bool | Kinh nhật tụng. |
| `purposes` | string[] | Các mục đích tụng đọc. |
| `duration_minutes` | int | Ước tính thời gian tụng/đọc. |
| `audio_url` | string | Có thể rỗng. |
| `deities` | object[] | `{deity_id, role, name_vi, thumbnail_url}`. `role`: `SPEAKER` (thuyết kinh), `SUBJECT` (văn bản nói về vị này), `MENTIONED` (được nhắc đến). |
| `translations` | object[] | `{key, lang, variant, translator, license, line_count}`. `variant`: `translation`, `chant` (âm để tụng), `meaning` (dịch nghĩa), `poem`. |

## Chi tiết

`GET /v1/sutras/{id}` trả toàn bộ field của `SutraSummary`, thêm:

| Field | Ghi chú |
| --- | --- |
| `review_status` | `UNREVIEWED`: nội dung chưa được người am hiểu duyệt. |
| `translations[].source_name`, `source_url`, `note` | Nguồn và ghi chú bản dịch. `note` là `""` nếu không có. |
| `translations[].sections` | Mục lục `{name, start_n, line_count}`. `[]` nếu không có. |

Một văn bản có thể có nhiều bản dịch cùng `lang`: phân biệt bằng `key` (duy nhất trong văn bản) và `variant`. Hãy hiển thị `translator`, `license` và nguồn cho người đọc.

## Đọc nội dung theo trang dòng

```bash
curl 'http://localhost:8080/v1/sutras/kinh-tu-bi/lines?translation=vi-translation&page=1&page_size=200'
```

`data` có dạng `{ "translation", "items": [{"n", "text", "section"}], "page", "page_size", "total", "total_pages" }`.

- `translation` **bắt buộc**, là `translations[].key`.
- `n` bắt đầu từ 1 và liên tục **trong từng bản dịch** (các bản dịch không khớp dòng với nhau). Dòng `n` nằm ở trang `floor((n - 1) / page_size) + 1`.
- `page_size` mặc định 200, tối đa 500. Lớn hơn 500 được phục vụ ở cỡ 500 (không báo lỗi); `page_size` trong response là cỡ thực tế. Không phải số nguyên dương trả `400 INVALID_PAGE_SIZE`.
- Đừng đổi `page_size` giữa hai lần gọi khi đang đọc một bản dịch: các trang sẽ dịch chỗ.
- Dừng khi `page >= total_pages` hoặc `items` rỗng. Bản dài nhất có 6.967 dòng nên cần tải từng trang.
- `section` là tên phẩm, hoặc `null` nếu dòng không thuộc phẩm nào.
- Mở một phẩm: đổi `start_n` thành trang bằng `floor((start_n - 1) / page_size) + 1`.

## Kinh theo Phật/Bồ tát

`GET /v1/deities/{id}/sutras` trả `SutraSummary` kèm `role` là vai của vị đó trong văn bản, phân trang như danh mục. Thứ tự: `SUBJECT`, rồi `SPEAKER`, rồi `MENTIONED`. Tham số `role` tuỳ chọn để lọc; giá trị khác trả `400 INVALID_FILTER`. Vị có tồn tại nhưng chưa có văn bản nào trả `200` với `items: []`.

## Lỗi

| HTTP | code | Khi nào |
| --- | --- | --- |
| 400 | `INVALID_FILTER` | Bộ lọc, `q` hoặc `role` không hợp lệ (`details.field`). |
| 400 | `INVALID_PAGE`, `INVALID_PAGE_SIZE` | `page` / `page_size` không phải số nguyên dương. |
| 400 | `INVALID_TRANSLATION` | Thiếu `translation` ở `/lines`. |
| 404 | `SUTRA_NOT_FOUND` | Không có văn bản với id này. |
| 404 | `TRANSLATION_NOT_FOUND` | Văn bản có nhưng không có bản dịch với `key` này (văn bản được kiểm tra trước). |
| 404 | `DEITY_NOT_FOUND` | Ở `/v1/deities/{id}/sutras`: không có Phật/Bồ tát với id này. |

Cấu trúc đầy đủ: [API Reference](/api-reference/). Mã lỗi chung: [Xử lý lỗi](/guides/error-handling/).
