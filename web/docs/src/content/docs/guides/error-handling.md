---
title: Xử lý lỗi
description: Bảng mã lỗi và cách client nên phản ứng.
---

Mọi lỗi dùng chung [envelope](/guides/getting-started/#response-envelope). Rẽ nhánh theo `error.code`.

## Cách phản ứng theo nhóm

| Nhóm | Ví dụ | Client nên |
| --- | --- | --- |
| 400 validation | `VALIDATION_ERROR`, `INVALID_PHONE` | Hiển thị lỗi cạnh field trong `details.field`, không retry nguyên request. |
| 401 | `UNAUTHORIZED`, `REFRESH_TOKEN_INVALID`, `INVALID_CREDENTIALS` | Access token hết hạn thì refresh. Refresh hỏng hoặc sai mật khẩu thì về màn hình đăng nhập. |
| 404 | `ARTICLE_NOT_FOUND` | Không có bài viết với id này. |
| 404 | `SUTRA_NOT_FOUND` | Không có kinh với id này. |
| 404 | `TRANSLATION_NOT_FOUND` | Kinh có nhưng không có bản dịch với `key` này. |
| 404 | `GALLERY_IMAGE_NOT_FOUND` | Không có ảnh với id này. |
| 404 | `DHARMA_TALK_NOT_FOUND` | Không có pháp thoại với id này, hoặc bài đang chờ duyệt. |
| 404 | `MEMORIAL_NOT_FOUND` | Mục sổ giỗ không tồn tại hoặc thuộc người khác. |
| 409 | `MEMORIAL_LIMIT_REACHED` | Sổ giỗ đã đủ 100 mục. |
| 409 | `PHONE_TAKEN`, `EMAIL_TAKEN` | Báo người dùng đổi thông tin hoặc chuyển sang đăng nhập. |
| 429 | `OTP_LOCKED`, `LOGIN_LOCKED`, `OTP_RESEND_TOO_SOON` | Không retry tự động. Dùng `details` để hiện đếm ngược. |
| 5xx | `INTERNAL_ERROR`, `SMS_SEND_FAILED` | Cho phép thử lại thủ công. `message` không có thông tin nội bộ. |

## Giới hạn tần suất (429)

`details` cho biết bao giờ được thử lại:

- `OTP_LOCKED`, `LOGIN_LOCKED`, `OTP_SEND_LIMITED`: có `details.locked_until` (thời điểm hết khoá).
- `OTP_RESEND_TOO_SOON`: có `details.retry_after_seconds`.

```json
{
  "success": false,
  "error": { "code": "LOGIN_LOCKED", "message": "...", "type": "business" },
  "details": { "locked_until": "2026-09-29T08:15:00Z" }
}
```

## Bảng mã lỗi

| HTTP | code | Ý nghĩa |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | Body sai JSON, thiếu field, field lạ, sai Content-Type hoặc sai tag. `details` là `{field: tag}`. |
| 400 | `INVALID_PHONE` | Số điện thoại không hợp lệ. |
| 400 | `INVALID_EMAIL` | Email không hợp lệ. |
| 400 | `INVALID_PASSWORD` | Mật khẩu không đạt yêu cầu (đăng ký), hoặc rỗng (đăng nhập). |
| 400 | `INVALID_DEVICE` | Thiếu `device.platform` hoặc `device.device_model`. |
| 400 | `OTP_INVALID` | OTP sai, hoặc không có lượt đăng ký đang chờ. |
| 400 | `OTP_EXPIRED` | OTP đã hết hạn. |
| 400 | `PENDING_REGISTRATION_EXPIRED` | Lượt đăng ký chờ đã quá hạn, phải bắt đầu lại. |
| 400 | `UNSUPPORTED_OTP_PURPOSE` | `purpose` chưa được hỗ trợ. |
| 400 | `ATTESTATION_CHALLENGE_INVALID` | Challenge không tồn tại, hết hạn hoặc đã dùng. |
| 400 | `INVALID_PAGE` | `page` không phải số nguyên dương. |
| 400 | `INVALID_PAGE_SIZE` | `page_size` (đọc dòng kinh) không phải số nguyên dương. |
| 400 | `INVALID_FILTER` | Bộ lọc, `q` hoặc `role` của kinh không hợp lệ (`details.field`). |
| 400 | `INVALID_TRANSLATION` | Thiếu `translation` khi đọc dòng kinh. |
| 400 | `INVALID_GALLERY_TYPE` | `type` của thư viện ảnh không hợp lệ. |
| 400 | `INVALID_DATE` | `date` của ảnh theo ngày không hợp lệ. |
| 401 | `UNAUTHORIZED` | Access token thiếu, sai, hết hạn hoặc sai loại. |
| 401 | `INVALID_CREDENTIALS` | Sai số điện thoại hoặc mật khẩu. |
| 401 | `REFRESH_TOKEN_INVALID` | Refresh token sai, hết hạn, đã thu hồi hoặc bị dùng lại. |
| 404 | `TEMPLE_NOT_FOUND` | Không có chùa với id này. |
| 404 | `DEITY_NOT_FOUND` | Không có mục với id này. |
| 409 | `PHONE_TAKEN` | Số điện thoại đã đăng ký. |
| 409 | `EMAIL_TAKEN` | Email đã đăng ký. |
| 413 | `PAYLOAD_TOO_LARGE` | Body vượt 1 MB. |
| 429 | `OTP_LOCKED` | Nhập sai OTP quá số lần. |
| 429 | `LOGIN_LOCKED` | Sai mật khẩu quá số lần. |
| 429 | `OTP_RESEND_TOO_SOON` | Gửi lại OTP trước khi hết cooldown. |
| 429 | `OTP_SEND_LIMITED` | Hết lượt gửi OTP. |
| 500 | `INTERNAL_ERROR` | Lỗi nội bộ. |
| 500 | `SMS_SEND_FAILED` | Nhà cung cấp SMS lỗi. |
| 502 | `ATTESTATION_UPSTREAM_ERROR` | Google/Apple không phản hồi. Thử lại được. |

:::tip
Với lỗi `OTP_INVALID`, server cố ý không phân biệt "sai mã" và "không có lượt đăng ký" để người ngoài không dò được số điện thoại nào đang đăng ký dở.
:::
