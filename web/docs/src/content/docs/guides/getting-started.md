---
title: Kết nối
description: Base URL, định dạng request và response, và lần gọi API đầu tiên.
---

## Base URL

| Môi trường | URL |
| --- | --- |
| Dev (chạy local) | `http://localhost:8080` |
| Staging / Production | Chưa có. Cập nhật khi triển khai. |

Mọi endpoint bắt đầu bằng `/v1`.

## Gọi thử

Các endpoint danh mục không cần đăng nhập, nên dùng chúng để kiểm tra kết nối:

```bash
curl http://localhost:8080/v1/temples/top
```

## Request

- Body là JSON, header `Content-Type: application/json`, tối đa 1 MB.
- Server **từ chối field lạ** trong body (`400 VALIDATION_ERROR`). Chỉ gửi đúng các field đã được tài liệu hoá.
- Server bật CORS nên trình duyệt gọi trực tiếp được.

## Response envelope

Mọi response có body đều dùng một khuôn duy nhất.

```json title="Thành công"
{ "success": true, "data": { "...": "..." } }
```

```json title="Lỗi"
{
  "success": false,
  "error": { "code": "OTP_INVALID", "message": "...", "type": "business" },
  "details": { "field": "phone_number" }
}
```

- `error.code` viết `UPPER_SNAKE_CASE`. **Rẽ nhánh theo `code`, không theo `message`.**
- `error.type` là `validation` (400 do sai định dạng), `business` (các 4xx còn lại) hoặc `system` (5xx, không lộ nguyên nhân nội bộ).
- `details` chỉ có khi mang thông tin dùng được, ví dụ `{"field": "..."}`, `{"locked_until": "..."}`, `{"retry_after_seconds": 30}`.
- Thành công không có dữ liệu trả về (như logout) vẫn dùng envelope: `{"success": true}`.

Cả route không tồn tại (404) và sai method (405) cũng trả đúng envelope này, nên client chỉ cần một bộ parser.

## Bước tiếp theo

- [Xác thực](/guides/authentication/): cách dùng access token và refresh token.
- [Xử lý lỗi](/guides/error-handling/): bảng mã lỗi và cách phản ứng.
- [Đăng ký và đăng nhập](/guides/register-and-login/): luồng đầy đủ kèm code mẫu.
