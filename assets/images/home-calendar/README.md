# Home-calendar Image Assets (ảnh Phật cho khối lịch âm)

Bộ 100 ảnh Phật/Bồ Tát dùng cho khối hiển thị **ngày âm lịch + hình Phật** trên trang chủ app
Tâm Đăng (InnerLamp).

## Nguồn & bản quyền

Tất cả ảnh được tải về từ **[Wikimedia Commons](https://commons.wikimedia.org)** (truy vấn qua
`commons.wikimedia.org/w/api.php`), chỉ giữ lại ảnh có giấy phép **tự do** (không có ảnh
`NC`/`ND`/`All rights reserved`):

| Giấy phép | Số ảnh | Ghi chú |
|---|---|---|
| CC BY-SA 4.0 | 41 | Ghi công tác giả |
| CC BY-SA 3.0 | 14 | Ghi công tác giả |
| CC BY-SA 2.0 | 7 | Ghi công tác giả |
| CC BY 2.0 | 12 | Ghi công tác giả |
| CC BY 4.0 | 5 | Ghi công tác giả |
| CC0 | 9 | Không cần ghi công |
| Public domain | 12 | Không cần ghi công |

> ⚠️ **Bắt buộc ghi công (attribution)** cho ảnh `CC BY` / `CC BY-SA`: tên tác giả + liên kết
> giấy phép. Danh sách đầy đủ nằm ở [`manifest.json`](manifest.json).

## Quy cách ảnh

- Định dạng: JPEG.
- Cạnh dài tối đa **1200px** (đủ sắc nét cho màn iPhone 3x, banner full-width), cạnh ngắn ≥ 500px.
- Nén quality ~80 → trung bình ~340 KB/ảnh, tổng ~33 MB cho 100 ảnh.

## Cách dùng

Tham chiếu trong Flutter:

```yaml
flutter:
  assets:
    - assets/images/home-calendar/
```

Đọc metadata (tác giả + giấy phép) từ `manifest.json` để render attribution nếu cần.

## Tái tạo / bổ sung

Script thu thập nằm ngoài repo (chạy một lần): tìm kiếm nhiều từ khoá trên Wikimedia Commons,
lọc theo giấy phép tự do + độ phân giải, tải thumbnail 1200px và chuẩn hoá bằng `sips`.
Muốn thêm ảnh mới thì liên hệ người tạo bộ ảnh này.
