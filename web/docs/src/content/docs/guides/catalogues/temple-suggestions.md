---
title: Đề xuất thêm chùa
description: "Gửi đề xuất một ngôi chùa chưa có trong danh mục để ban quản trị xem xét."
---

Người dùng có thể đề xuất một ngôi chùa chưa có trong [danh mục chùa](/guides/catalogues/temples/). Đề xuất được lưu ở trạng thái chờ duyệt; ban quản trị xem xét, bổ sung thông tin rồi mới đưa vào danh mục.

Endpoint **công khai**, không cần đăng nhập, và **chỉ ghi**: không có route nào đọc lại đề xuất.

| Gửi đề xuất |
| --- |
| `POST /v1/temple-suggestions` |

## Body

| Field | Kiểu | Bắt buộc | Ghi chú |
| --- | --- | --- | --- |
| `name` | string | có | Tên chùa, tối đa 255 ký tự. |
| `province` | string | có | Tỉnh/thành phố, chuỗi tự do, tối đa 100 ký tự. |
| `address` | string | có | Địa chỉ, tối đa 500 ký tự. |

- Giá trị được cắt khoảng trắng đầu cuối, độ dài đếm theo ký tự Unicode, và được lưu đúng như người dùng nhập.
- Field lạ (kể cả `id`, `status`) bị từ chối `400 VALIDATION_ERROR`.
- Web gửi thêm header `X-API-Key` như các route công khai khác, xem [Xác thực](/guides/authentication/#x-api-key-client-web).

```bash
curl -X POST http://localhost:8080/v1/temple-suggestions \
  -H 'Content-Type: application/json' \
  -d '{"name":"Chùa Mới","province":"Quảng Ninh","address":"Thôn 1, xã A"}'
```

## Response

`201 Created`:

```json
{ "success": true, "data": { "id": 12, "status": "pending" } }
```

`id` chỉ để tham chiếu khi cần; client không có cách nào xem trạng thái đề xuất sau đó. Cùng một đề xuất gửi lại vẫn nhận `201`, vì server không dò trùng với chùa hay đề xuất đã có.

## Lỗi

| HTTP | `error.code` | Khi nào | Client nên làm gì |
| --- | --- | --- | --- |
| 400 | `VALIDATION_ERROR` | Thiếu hoặc rỗng một field, quá dài, JSON sai, field lạ. `details.field` là field sai đầu tiên. | Báo lỗi cạnh field đó. |
| 401 | `INVALID_API_KEY` | `X-API-Key` sai hoặc rỗng. | Kiểm tra cấu hình server web. |
| 413 | `PAYLOAD_TOO_LARGE` | Body vượt 1 MB. | Sửa request. |
| 429 | `RATE_LIMITED` | IP này đã gửi đủ số đề xuất cho phép (5 mỗi giờ, cấu hình dev). | Không retry tự động; dùng header `Retry-After` hoặc `details.retry_after_seconds`. |
| 500 | `INTERNAL_ERROR` | Lỗi nội bộ. | Cho phép thử lại thủ công. |

```json
{
  "success": false,
  "error": { "code": "RATE_LIMITED", "message": "too many requests", "type": "business" },
  "details": { "retry_after_seconds": 1800 }
}
```

:::note
Giới hạn tính theo IP và chung cho mọi người dùng cùng IP (mạng nhà mạng, văn phòng). IP chỉ dùng tạm cho bộ đếm, không lưu cùng đề xuất.
:::

Envelope và mã lỗi chung: [Xử lý lỗi](/guides/error-handling/).
