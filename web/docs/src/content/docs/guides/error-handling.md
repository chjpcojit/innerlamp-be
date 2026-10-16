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
| 401 | `INVALID_API_KEY` | `X-API-Key` sai hoặc rỗng (chỉ client web). Lỗi cấu hình server web, không retry. |
| 404 | `ARTICLE_NOT_FOUND` | Không có bài viết với id này. |
| 404 | `SUTRA_NOT_FOUND` | Không có kinh với id này. |
| 404 | `TRANSLATION_NOT_FOUND` | Kinh có nhưng không có bản dịch với `key` này. |
| 404 | `GALLERY_IMAGE_NOT_FOUND` | Không có ảnh với id này. |
| 404 | `DHARMA_TALK_NOT_FOUND` | Không có pháp thoại với id này, hoặc bài đang chờ duyệt. |
| 404 | `MEMORIAL_NOT_FOUND` | Mục sổ giỗ không tồn tại hoặc thuộc người khác. |
| 400 | `INVALID_FULL_NAME` | Họ tên trống hoặc quá 300 ký tự (`details.field = "full_name"`). Dùng ở đăng ký và ở `PATCH /v1/users/me`. |
| 400 | `INVALID_DHARMA_NAME` | Pháp danh quá 255 ký tự (`details.field = "dharma_name"`). Dùng ở đăng ký và ở `PATCH /v1/users/me`. |
| 400 | `INVALID_AVATAR` | Ảnh đã upload có loại không được phép hoặc quá 5 MB; ảnh đại diện không đổi. *(Hồ sơ cá nhân: đã có spec, chưa có ở server.)* |
| 404 | `AVATAR_UPLOAD_NOT_FOUND` | Chưa có ảnh nào được upload với `upload_id` này. *(Hồ sơ cá nhân: đã có spec, chưa có ở server.)* |
| 409 | `MEMORIAL_LIMIT_REACHED` | Sổ giỗ đã đủ 100 mục. |
| 403 | `GENEALOGY_FORBIDDEN` | Người xem (`viewer`) gọi route chỉ dành cho chủ gia phả; không có gì bị đổi. |
| 404 | `GENEALOGY_NOT_FOUND` | Gia phả không tồn tại **hoặc người gọi không có vai trò nào** (cùng một phản hồi). |
| 404 | `MEMBER_NOT_FOUND`, `VIEWER_NOT_FOUND`, `INVITATION_NOT_FOUND` | Không có thành viên, người xem hoặc lời mời đó (với người gọi). |
| 409 | `GENEALOGY_LIMIT_REACHED`, `MEMBER_LIMIT_REACHED`, `VIEWER_LIMIT_REACHED` | Đã đủ giới hạn gia phả, thành viên hoặc người xem. |
| 409 | `MEMBER_HAS_DESCENDANTS`, `FOUNDER_NOT_DELETABLE`, `FOUNDER_NOT_MOVABLE` | Xoá người còn con (xoá từ đời dưới lên), xoá hoặc di chuyển thuỷ tổ. |
| 409 | `MEMBER_PHONE_ALREADY_USED` | Số điện thoại đã thuộc một thành viên khác của gia phả (`details.field`). |
| 409 | `VIEWER_ALREADY_EXISTS`, `INVITATION_ALREADY_PENDING`, `INVITATION_NOT_PENDING` | Số đã có quyền, đã có lời mời chờ, hoặc lời mời không còn chờ. |
| 409 | `LAMP_LIMIT_REACHED` | Đã có đủ 20 đèn đang sáng; thắp được lại khi một ngọn tắt. |
| 409 | `PHONE_TAKEN` | Báo người dùng đổi số điện thoại hoặc chuyển sang đăng nhập. |
| 409 | `REGISTRATION_IN_PROGRESS` | Email đang có mã đăng ký còn sống với dữ liệu khác. Dùng mã đã gửi hoặc chờ `details.retry_after_seconds`. |
| 429 | `OTP_LOCKED`, `LOGIN_LOCKED`, `OTP_RESEND_TOO_SOON`, `OTP_SEND_LIMITED`, `RATE_LIMITED` | Không retry tự động. Dùng `details` để hiện đếm ngược. |
| 5xx | `INTERNAL_ERROR`, `EMAIL_SEND_FAILED` | Cho phép thử lại thủ công. `message` không có thông tin nội bộ. |

## Giới hạn tần suất (429)

`details` cho biết bao giờ được thử lại:

- `OTP_LOCKED`, `LOGIN_LOCKED`, `OTP_SEND_LIMITED`: có `details.locked_until` (thời điểm hết khoá).
- `OTP_RESEND_TOO_SOON`, `RATE_LIMITED`, `REGISTRATION_IN_PROGRESS`: có `details.retry_after_seconds`. `RATE_LIMITED` còn kèm header `Retry-After`.

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
| 400 | `INVALID_PHONE` | `phone_country` ngoài danh sách hoặc `phone_number` sai ký tự / sai số chữ số (đăng ký). |
| 400 | `INVALID_EMAIL` | Email không hợp lệ. |
| 400 | `INVALID_PASSWORD` | Mật khẩu ngoài 8 đến 64 ký tự (đăng ký, đặt lại mật khẩu), hoặc rỗng (đăng nhập). |
| 400 | `INVALID_FULL_NAME`, `INVALID_DHARMA_NAME`, `INVALID_TEMPLE` | Họ tên trống hoặc quá 300 ký tự, pháp danh quá 255 ký tự, hoặc `ordination_temple_id` không khớp chùa nào (đăng ký). |
| 400 | `INVALID_DEVICE` | Thiếu `device.platform` hoặc `device.device_model`. |
| 400 | `OTP_INVALID` | Mã sai (hoặc, ở đăng ký, email đã có tài khoản: mọi mã đều sai). |
| 400 | `OTP_EXPIRED` | Mã đã hết hạn hoặc đã dùng (ở đặt lại mật khẩu: cũng khi email không có tài khoản hoặc chưa phát mã). |
| 400 | `PENDING_REGISTRATION_EXPIRED` | Không có lượt đăng ký chờ cho email này, hoặc đã quá 24 giờ: bắt đầu lại từ bước 1. |
| 400 | `ATTESTATION_CHALLENGE_INVALID` | Challenge không tồn tại, hết hạn hoặc đã dùng. |
| 400 | `INVALID_PAGE` | `page` không phải số nguyên dương. Ở [thông báo của chùa](/guides/catalogues/temple-announcements/) cũng áp dụng cho `page_size`; `details.field` luôn là `page`. |
| 400 | `INVALID_PAGE_SIZE` | `page_size` (đọc dòng kinh) không phải số nguyên dương. |
| 400 | `INVALID_FILTER` | Bộ lọc, `q` hoặc `role` của kinh, `q` của danh sách xã (hành chính), hoặc `lamp_type` của vườn đèn không hợp lệ (`details.field`). |
| 400 | `INVALID_LIMIT` | `limit` của vườn đèn không phải số nguyên dương. |
| 400 | `INVALID_CURSOR` | `cursor` của vườn đèn không do server cấp; bỏ `cursor` và tải lại từ đầu. |
| 400 | `INVALID_TRANSLATION` | Thiếu `translation` khi đọc dòng kinh. |
| 400 | `INVALID_GALLERY_TYPE` | `type` của thư viện ảnh không hợp lệ. |
| 400 | `INVALID_DATE` | `date` của ảnh theo ngày, hoặc `from`/`to` của [sự kiện chùa](/guides/catalogues/temple-events/), không hợp lệ. `details.field` nêu tham số sai. |
| 400 | `INVALID_DATE_RANGE` | `to` trước `from` ở [sự kiện chùa](/guides/catalogues/temple-events/). |
| 400 | `DATE_RANGE_TOO_LARGE` | Cửa sổ ngày của [sự kiện chùa](/guides/catalogues/temple-events/) dài hơn mức cho phép (3 tháng). |
| 400 | `INVALID_YEAR` | `year` của sự kiện lịch không phải số nguyên hoặc ngoài 1900..2100 (spec, chưa có ở server). |
| 401 | `UNAUTHORIZED` | Access token thiếu, sai, hết hạn hoặc sai loại. |
| 401 | `INVALID_CREDENTIALS` | Sai email hoặc mật khẩu (cùng một lỗi cho email chưa có tài khoản hay chưa xác minh). |
| 401 | `REFRESH_TOKEN_INVALID` | Refresh token sai, hết hạn, đã thu hồi hoặc bị dùng lại. |
| 401 | `INVALID_API_KEY` | Header `X-API-Key` sai hoặc rỗng. Không gửi header thì không gặp lỗi này (coi là mobile). |
| 404 | `TEMPLE_NOT_FOUND` | Không có chùa với id này. |
| 400 | `INVALID_ANNOUNCEMENT` | *(backoffice)* Đăng/sửa thông báo của chùa sai: `title`, `body`, `issuing_body_id` hoặc `expires_at` thiếu, quá dài, hoặc `expires_at` không ở tương lai / quá xa. `details.field` nêu field sai. |
| 400 | `INVALID_ISSUING_BODY` | *(backoffice)* `issuing_body_id` không thuộc chùa này. |
| 404 | `ANNOUNCEMENT_NOT_FOUND` | *(backoffice)* Không có thông báo với id này. |
| 409 | `ANNOUNCEMENT_REVOKED` | *(backoffice)* Thông báo đã bị thu hồi nên không sửa được. |
| 409 | `ISSUING_BODY_EXISTS` | *(backoffice)* Chùa đã có đơn vị phát hành cùng tên (không phân biệt hoa thường). |
| 404 | `ADMINISTRATIVE_UNIT_NOT_FOUND` | Không có tỉnh/quận/huyện/xã với `code` này trong hệ (`new`/`old`) đã chỉ định. |
| 404 | `DEITY_NOT_FOUND` | Không có mục với id này. |
| 403 | `GENEALOGY_FORBIDDEN` | Người xem gọi route ghi của gia phả (chỉ chủ gia phả được ghi). |
| 404 | `GENEALOGY_NOT_FOUND` | Gia phả không tồn tại, id không phải số nguyên dương, hoặc người gọi không có vai trò (cùng một phản hồi). |
| 404 | `MEMBER_NOT_FOUND` | Thành viên không có trong gia phả này (kể cả thuộc gia phả khác). |
| 404 | `VIEWER_NOT_FOUND` | Người đó không phải người xem của gia phả. |
| 404 | `INVITATION_NOT_FOUND` | Lời mời không tồn tại, gửi tới số khác, hoặc không còn `pending` (khi owner huỷ). |
| 409 | `GENEALOGY_LIMIT_REACHED` | Đã đủ số gia phả tối đa của một người. |
| 409 | `MEMBER_LIMIT_REACHED` | Gia phả đã đủ số thành viên tối đa. |
| 409 | `MEMBER_HAS_DESCENDANTS` | Xoá người còn con. |
| 409 | `FOUNDER_NOT_DELETABLE` | Xoá thuỷ tổ (muốn bỏ cả cây thì xoá gia phả). |
| 409 | `FOUNDER_NOT_MOVABLE` | Di chuyển thuỷ tổ. |
| 409 | `MEMBER_PHONE_ALREADY_USED` | `phone_number` đã thuộc một thành viên khác của gia phả. |
| 409 | `VIEWER_ALREADY_EXISTS` | Số thuộc chủ gia phả hoặc người đã xem. |
| 409 | `VIEWER_LIMIT_REACHED` | Đủ số người xem, tính cả lời mời đang chờ. |
| 409 | `INVITATION_ALREADY_PENDING` | Đã có lời mời chờ cho số này. |
| 409 | `INVITATION_NOT_PENDING` | Lời mời đã trả lời, đã huỷ hoặc đã hết hạn. |
| 409 | `PHONE_TAKEN` | Số điện thoại đã thuộc một tài khoản. |
| 409 | `REGISTRATION_IN_PROGRESS` | Email đang có mã đăng ký còn sống với dữ liệu khác; `details.retry_after_seconds`. |
| 413 | `PAYLOAD_TOO_LARGE` | Body vượt 1 MB. |
| 429 | `OTP_LOCKED` | Nhập sai mã quá số lần. |
| 429 | `LOGIN_LOCKED` | Sai mật khẩu quá số lần (khoá theo email). |
| 429 | `OTP_RESEND_TOO_SOON` | Gửi lại mã trước khi hết cooldown. |
| 429 | `OTP_SEND_LIMITED` | Hết lượt gửi mã. |
| 429 | `RATE_LIMITED` | Gửi quá nhanh theo IP. Áp dụng cho mọi `/v1/auth/*` trừ `logout` (theo nhóm `send`, `verify`, `login`, `refresh`, xem [Đăng ký và đăng nhập](/guides/register-and-login/#giới-hạn-tần-suất)) và hết ngân sách thư mỗi ngày; ngoài ra cho `POST /v1/temple-suggestions`: 5 lần mỗi giờ, và `GET /v1/administrative-units/*`: 60 lần mỗi phút). Nhóm hồ sơ cá nhân `/v1/users/me/*` giới hạn **theo tài khoản**: 10 request mỗi phút, đọc cũng tính *(đã có spec, chưa có ở server)*. [Thông báo của chùa](/guides/catalogues/temple-announcements/) có bộ đếm theo tài khoản **riêng**: 60 request mỗi phút (dev). |
| 500 | `INTERNAL_ERROR` | Lỗi nội bộ. |
| 500 | `EMAIL_SEND_FAILED` | Nhà cung cấp email lỗi (đăng ký). Lượt gửi không bị tính. |
| 503 | `SERVICE_UNAVAILABLE` | Route Bearer không kiểm tra được phiên (mất kết nối DB); thử lại sau. |
| 502 | `ATTESTATION_UPSTREAM_ERROR` | Google/Apple không phản hồi. Thử lại được. |

:::tip
Server cố ý không cho dò email: đăng ký trả cùng `200` cho email mới và email đã có tài khoản, đăng nhập trả cùng `INVALID_CREDENTIALS`, quên mật khẩu trả cùng `200` cho mọi email hợp lệ.
:::
