---
title: Changelog
description: Lịch sử thay đổi của API.
---

## v1 (hiện tại)

**Cập nhật 2026-10-16 — phiên bản hợp đồng (`openapi.yaml` 2.1.1):**

- `openapi.yaml` có `info.version` **2.1.1** để client trích dẫn. Từ nay mỗi thay đổi của file bump số này theo semver (MAJOR: phá vỡ client; MINOR: thêm tương thích ngược; PATCH: chỉ sửa chữ), kèm một mục ở changelog này.
- 2.1.1: `device.platform` của `login` là **tuỳ chọn** (vắng hoặc rỗng thì server lưu `mobile`); chỉ `device.device_model` bắt buộc, `INVALID_DEVICE` chỉ khi thiếu `device` hoặc `device_model`. Hành vi của code không đổi; tài liệu trước đó ghi mâu thuẫn. Client vẫn nên gửi `platform`.

**Cập nhật 2026-10-16 — đăng ký, đăng nhập và quên mật khẩu bằng email:**

Thay đổi hợp đồng, **không tương thích ngược** cho mọi client dùng `/v1/auth/*`. Xác minh tài khoản chuyển từ SMS sang mã gửi qua **email**; đăng nhập bằng email thay cho số điện thoại.

- `POST /v1/auth/register/request-otp`: body thêm `phone_country` (`VN`, `US`, `JP`, `KR`, `TH`, `SG`, `MY`, `TW`, `AU`, `DE`, `FR`, `CA`) và `phone_number` là số người dùng gõ (không kèm mã quốc gia, không `+`, đúng số chữ số của nước đó; server trả `user.phone_number` dạng E.164). Mật khẩu 8 đến 64 ký tự. Mỗi field có mã lỗi riêng: field thiếu trả `INVALID_EMAIL` / `INVALID_PHONE` / `INVALID_PASSWORD` / `INVALID_FULL_NAME`, không còn `VALIDATION_ERROR`. Mã sống 600 giây (trước 90).
- **Bỏ `409 EMAIL_TAKEN`.** Email đã có tài khoản nhận về `200` y hệt email mới (thư gửi đi là thư báo thay cho mã), nên không dò được email nào đã đăng ký. `409 PHONE_TAKEN` chỉ khi số thuộc một tài khoản. Thêm `409 REGISTRATION_IN_PROGRESS` (+ `details.retry_after_seconds`) khi email đang có mã còn sống với dữ liệu khác. Thêm `500 EMAIL_SEND_FAILED` thay `SMS_SEND_FAILED`.
- **Đổi đường dẫn gửi lại mã:** `POST /v1/auth/resend-otp` (`{phone_number, purpose}`) bị **xoá**, thay bằng `POST /v1/auth/register/resend-otp` với `{email}`. Bỏ mã lỗi `UNSUPPORTED_OTP_PURPOSE`.
- `POST /v1/auth/register/verify-otp`: body `{email, otp_code}` thay `{phone_number, otp_code}`. Đồng thời trả `Cache-Control: no-store`.
- `POST /v1/auth/login`: body `{email, password, device}` thay `{phone_number, ...}`; lỗi `INVALID_CREDENTIALS` có message "Email hoặc mật khẩu không đúng"; khoá `LOGIN_LOCKED` tính theo email (kể cả email chưa có tài khoản). Sai định dạng email trả `INVALID_EMAIL`.
- Thêm **quên mật khẩu** (2 endpoint, công khai): `POST /v1/auth/password-reset/request-otp {email}` luôn trả `200` cùng một body cho mọi email hợp lệ; `POST /v1/auth/password-reset/confirm {email, otp_code, new_password}` đặt mật khẩu mới, thu hồi mọi phiên, gỡ khoá đăng nhập, **không** trả token (đăng nhập lại).
- **Giới hạn theo IP** cho mọi `/v1/auth/*` trừ `logout`: nhóm `send` (10/giờ), `verify` (30/giờ), `login` (30/15 phút), `refresh` (120/15 phút) ở dev; vượt thì `429 RATE_LIMITED` kèm `details.retry_after_seconds` và header `Retry-After`. Hết ngân sách thư mỗi ngày cũng trả `RATE_LIMITED`. Website gửi IP của khách ở header `X-InnerLamp-Client-IP` (chỉ được tin khi có `X-API-Key` đúng).
- Mọi response của `register/*`, `login`, `refresh` và `password-reset/*` (kể cả lỗi) có `Cache-Control: no-store`.
- Các route Bearer (trừ `logout`) từ chối access token của **phiên đã thu hồi** (`401 UNAUTHORIZED`) ngay ở request kế tiếp, và trả `503` nếu không kiểm tra được phiên. Trước đây access token dùng được tới khi hết hạn.
- `GET /v1/users/me`: `email_verified` là `true` với tài khoản đăng ký bằng mã email.
- `openapi.yaml`: viết lại sáu operation Auth cho khớp code, thêm hai operation `password-reset/*`, response `OtpVerifyRateLimited` / `AuthRateLimited` / header `CacheControlNoStore`, schema `PasswordReset*`; `User` thêm `full_name`; `RequestOtpRequest` thêm `phone_country`.
- Xem [Đăng ký và đăng nhập](/guides/register-and-login/).

**Cập nhật 2026-10-14 — thông báo của chùa:**

- Thêm `GET /v1/users/me/temple-announcements?page=&page_size=` (**cần đăng nhập**): thông báo còn hiệu lực của chùa quy y hiện tại của người dùng, mới nhất trước, kèm `receive`, `temple` và phân trang `page`. Chùa do server suy ra từ hồ sơ, không có tham số chọn chùa. Công tắc tắt hoặc chưa có chùa quy y trả `200` với `items: []`.
- Thêm `GET` và `PUT /v1/users/me/temple-announcement-settings`: công tắc nhận thông báo, một cái cho cả tài khoản (mặc định bật), body `{"receive": bool}`.
- Ba route có giới hạn tần suất **riêng** theo tài khoản (dev: 60 request/phút), tách khỏi nhóm hồ sơ; vượt thì `429 RATE_LIMITED`. Response luôn `Cache-Control: no-store`. Xem [Thông báo của chùa](/guides/catalogues/temple-announcements/).
- `INVALID_PAGE` dùng cho cả `page_size` của route này (`details.field` luôn là `page`); `page_size` lớn hơn mức tối đa bị chặn, không báo lỗi.
- `openapi.yaml`: thêm ba route trên, sáu route quản trị `/v1/admin/temples/{id}/announcements`, `/v1/admin/temple-announcements/{id}`, `/v1/admin/temple-announcements/{id}/revoke`, `/v1/admin/temples/{id}/issuing-bodies` (backoffice) và các mã lỗi `INVALID_ANNOUNCEMENT`, `INVALID_ISSUING_BODY`, `ANNOUNCEMENT_NOT_FOUND`, `ANNOUNCEMENT_REVOKED`, `ISSUING_BODY_EXISTS`.

**Cập nhật 2026-10-13 — sự kiện của chùa:**

- Thêm `GET /v1/temples/{id}/events?from=&to=` (**cần đăng nhập**): sự kiện do một ngôi chùa tổ chức (khoá tu, pháp hội, lễ cầu an/cầu siêu, thuyết pháp, lễ hội), độc lập với sự kiện Phật giáo chung của `calendar`. Server tính sự kiện lặp (mỗi tuần, mỗi tháng âm, kể cả ngày 30 trong tháng thiếu và tháng nhuận) thành `occurrences` kèm ngày dương và ngày âm của ngày bắt đầu. Cửa sổ mặc định và tối đa 3 tháng; sự kiện đã kết thúc không được trả; sự kiện hoãn/huỷ vẫn được trả kèm `status`. Response luôn `Cache-Control: no-cache`. Xem [Sự kiện của chùa](/guides/catalogues/temple-events/).
- Thêm mã lỗi `INVALID_DATE_RANGE` và `DATE_RANGE_TOO_LARGE`. `INVALID_DATE` của route này có `details.field` là `from` hoặc `to`.
- `openapi.yaml`: thêm `/v1/temples/{id}/events` và các schema `TempleEventsEnvelope`, `TempleEvent`, `TempleEventRecurrence`, `TempleEventOccurrence`.
- Dữ liệu hiện là mock (12 sự kiện, 5 chùa, nạp bằng `seedtempleevents`).

**Cập nhật 2026-10-13 — thời khoá tu tập của chùa:**

- Thêm `GET /v1/temples/{id}/schedule` (**cần đăng nhập**, khác các route `/v1/temples` công khai): thời khoá sinh hoạt thường ngày của một chùa. Mỗi buổi có `start_time`, `end_time` (nhỏ hơn `start_time` là qua nửa đêm), `weekdays` (`[]` = mỗi ngày, `0` = Chủ nhật), `description`, `instructor`, `location`, `preparation` và `steps` (sắp theo `position`). Chùa chưa có buổi nào trả `items: []`. Xem [Thời khoá tu tập của chùa](/guides/catalogues/temple-schedule/).
- Không thêm mã lỗi mới: dùng `TEMPLE_NOT_FOUND` và `UNAUTHORIZED` sẵn có.
- `openapi.yaml`: thêm `/v1/temples/{id}/schedule` và các schema `TempleScheduleEnvelope`, `TempleScheduleItem`, `TempleScheduleStep`.
- Dữ liệu hiện là mock (6 chùa, nạp bằng `seedtempleschedules`), chưa phải thời khoá thật.

**Cập nhật 2026-10-11 — họ tên (`full_name`):**

- `POST /v1/auth/register/request-otp` **bắt buộc** `full_name` (tối đa 300 ký tự). Thiếu field trả `400 VALIDATION_ERROR`; trống hoặc quá dài trả `400 INVALID_FULL_NAME`.
- `user.full_name` có trong response của `register/verify-otp` và `login` (vắng mặt với tài khoản cũ chưa có tên).
- `GET /v1/users/me` trả `full_name`; `PATCH /v1/users/me` sửa được `full_name` nhưng không xoá được (`null` hoặc rỗng trả `400 INVALID_FULL_NAME`).

69 endpoint appService đang có (cộng 5 endpoint hồ sơ cá nhân và 1 endpoint sự kiện lịch mới ở dạng spec, chưa có ở server):

- **Auth**: `register/request-otp`, `register/resend-otp`, `register/verify-otp`, `login`, `refresh`, `logout`, `password-reset/request-otp`, `password-reset/confirm`.
- **Attestation**: `attestation/challenge`, `assertion/challenge`, `attestation/verify`.
- **Chùa**: `temples/top`, `temples`, `temples/{id}`.
- **Thời khoá chùa** (cần đăng nhập): `temples/{id}/schedule`.
- **Sự kiện của chùa** (cần đăng nhập): `temples/{id}/events`.
- **Đề xuất chùa**: `POST /v1/temple-suggestions` (công khai, chỉ ghi).
- **Phật và Bồ tát**: `deities/top`, `deities`, `deities/{id}`.
- **Bài viết Phật pháp**: `articles/top`, `articles`, `articles/{id}`.
- **Sổ giỗ chạp** (cần đăng nhập): `POST/GET /v1/memorials`, `GET/PUT/DELETE /v1/memorials/{id}`.
- **Thắp đèn** (cần đăng nhập): `POST /v1/lamps`, `GET /v1/lamps` (vườn đèn).
- **Hồ sơ cá nhân** (cần đăng nhập, 5 endpoint): `GET/PATCH /v1/users/me`, `POST /v1/users/me/avatar/upload-url`, `POST /v1/users/me/avatar/confirm`, `DELETE /v1/users/me/avatar`.
- **Gia phả** (cần đăng nhập, 21 endpoint): `genealogies` (tạo, danh sách, xem, đổi tên, xoá), `genealogies/{id}/chronicle`, `genealogies/{id}/members` và `members/{memberId}` (+ `relation`, `profile`), `genealogies/{id}/invitations`, `genealogies/{id}/viewers`, `invitations` (hộp lời mời), `invitations/{id}/accept`, `invitations/{id}/decline`.
- **Kinh, kệ, chú**: `sutras`, `sutras/{id}`, `sutras/{id}/lines`, `deities/{id}/sutras`.
- **Ảnh đẹp Phật giáo**: `gallery/top`, `gallery`, `gallery/{id}`.
- **Pháp thoại**: `dharma-talks`, `dharma-talks/{id}`.
- **Lịch âm**: `calendar/image` (công khai). Cộng `calendar/events` (cần đăng nhập) ở dạng spec.
- **Đơn vị hành chính** (công khai, 9 endpoint): `administrative-units/new/provinces`, `new/provinces/{code}/wards` (có `q`), `new/provinces/{code}/conversion`, `new/wards/{code}/conversion`, `old/provinces`, `old/provinces/{code}/districts`, `old/districts/{code}/wards`, `old/provinces/{code}/conversion`, `old/wards/{code}/conversion`.
- **Ảnh tĩnh** (dev): `static/images/{path}`.

### 2026-10-11

- Thêm **sự kiện lịch** (cần đăng nhập, 1 endpoint `GET /v1/calendar/events?year=`): các sự kiện Phật giáo, ngày lễ Việt Nam và lễ tết dân gian của một năm dương, mỗi lần xảy ra có cả `solar_date` và `lunar_date` (kèm cờ tháng nhuận), nhóm theo `buddhist` / `vietnam` / `folk`, cờ `is_public_holiday`. Response chuẩn hoá thành `groups` + `events` + `occurrences`; "load more" là gọi năm kế tiếp; năm hợp lệ 1900..2100, bỏ trống là năm hiện tại. **Đang là spec**: chưa có code, route chưa được đăng ký ở server. Xem [Lịch âm](/guides/catalogues/calendar/#sự-kiện-lịch). Thêm mã lỗi `INVALID_YEAR`.
- `openapi.yaml`: thêm `/v1/calendar/events` và các schema `CalendarEventsEnvelope`, `CalendarEventsPage`, `CalendarGroup`, `CalendarEvent`, `CalendarOccurrence`, `CalendarLunarDate`; mô tả tag `Calendar` nói rõ hai endpoint.
- Thêm **hồ sơ cá nhân** (cần đăng nhập, 5 endpoint dưới `/v1/users/me`): xem hồ sơ, sửa pháp danh và chùa quy y theo kiểu cập nhật từng phần (`null` hoặc rỗng = xoá), đặt/đổi/gỡ ảnh đại diện bằng signed URL (xin vé, PUT thẳng lên storage, xác nhận). Địa chỉ ảnh sống 15 phút; giới hạn 10 request mỗi phút mỗi tài khoản, đọc cũng tính. Route đã được đăng ký ở server; luồng upload ảnh mới chạy thử trên emulator storage, chưa kiểm tra trên bucket thật (`TD-001`). CORS của server cho phép thêm method `PATCH`. Xem [Hồ sơ cá nhân](/guides/catalogues/profile/).
- Thêm **đơn vị hành chính** (công khai, chỉ đọc, 9 endpoint `GET /v1/administrative-units/*`): tỉnh/thành và xã/phường theo hệ hai cấp mới (34 tỉnh) và hệ ba cấp cũ (63 tỉnh), tìm xã theo tên không dấu bằng `q`, chuyển đổi cấp xã và cấp tỉnh giữa hai hệ. Xem [Đơn vị hành chính](/guides/catalogues/administrative-units/). Thêm mã lỗi `ADMINISTRATIVE_UNIT_NOT_FOUND`; `RATE_LIMITED` áp dụng cho nhóm route này (60 lần mỗi phút mỗi IP).
- **Thay đổi hợp đồng** `POST /v1/temple-suggestions`: body đổi từ `{name, province, address}` sang `{name, province_code, ward_code, address}`. `province_code` và `ward_code` là mã **hệ mới** lấy từ [Đơn vị hành chính](/guides/catalogues/administrative-units/); `address` chỉ còn phần dưới cấp xã; field `province` chuỗi tự do bị bỏ (gửi lên là `400 VALIDATION_ERROR`). Mã không tồn tại hoặc xã không thuộc tỉnh trả `400` với `details.field` là `province_code` hoặc `ward_code`. Xem [Đề xuất thêm chùa](/guides/catalogues/temple-suggestions/).
- `openapi.yaml`: sửa ví dụ của schema `AdministrativeProvince`, `AdministrativeLegacyDistrict`, `AdministrativeLegacyWard` (trước đó dùng chung ví dụ phường "Tân Định"); sửa lỗi thụt lề ở mô tả `genealogyMembersProfile` khiến cả file không parse được.

- Thêm **duyệt đề xuất chùa** ở backofficeService (cổng `:8081`, 3 endpoint, role `admin` hoặc `content_editor`): `GET /v1/admin/temple-suggestions` (hàng chờ, lọc `status`, phân trang `page`), `GET /v1/admin/temple-suggestions/{id}`, `POST /v1/admin/temple-suggestions/{id}/create-temple`. Tạo chùa nhận `province_code`, `ward_code`, `address` cùng các field chùa; tên tỉnh/xã lấy từ bộ dữ liệu hành chính, không lấy từ body; chèn chùa và đánh dấu đề xuất trong một transaction. Đề xuất đã `rejected` chỉ `admin` mở lại được. Chưa có route từ chối. Xem [Đề xuất thêm chùa](/guides/catalogues/temple-suggestions/#dành-cho-ban-quản-trị-backoffice).
- `openapi.yaml`: thêm 3 route trên (tag `Admin` và `TempleSuggestions`) và schema `TempleSuggestionItem`, `TempleSuggestion`, `TempleSuggestionEnvelope`, `TempleSuggestionPageEnvelope`, `CreateTempleRequest`, `CreateTempleEnvelope`.
- Tài liệu **đơn vị hành chính**: thêm mục "Dùng ở những nơi khác" nói rõ bộ ba `province_code` + `ward_code` + `address` và cách hiển thị bằng `full_address`.

- **Thay đổi hợp đồng** chùa (`GET /v1/temples/top`, `/v1/temples`, `/v1/temples/{id}`, `/v1/temples/options`): địa chỉ chùa có cấu trúc theo metadata hành chính hệ mới.
  - `location` thêm `province_code`, `ward_code`, `ward` (cùng `null` khi chùa chưa có địa chỉ có cấu trúc) và `full_address` (chuỗi để hiển thị, không bao giờ `null`).
  - **`location.address` đổi nghĩa**: chỉ còn phần dưới cấp xã (số nhà, đường, thôn…), có thể rỗng; hiển thị bằng `full_address`, không dùng `address` riêng lẻ.
  - `options` thêm `province_code` cho mỗi phần tử (nay có bốn field: `id`, `name`, `province`, `province_code`).
  - Đã nạp lại cả 93 chùa trong file seed; mã xã được gán qua ánh xạ xã cũ → mới của bộ dữ liệu hành chính và kiểm tra thủ công; một số ít chùa nằm trên ranh giới xã cũ đã tách có thể cần rà lại.
- `openapi.yaml`: schema `Location` và `TempleOption` thêm các trường trên; `address` được mô tả lại.
- Tài liệu **chùa**: thêm mục [Chọn chùa](/guides/catalogues/temples/#chọn-chùa-options) mô tả `GET /v1/temples/options` (danh sách rút gọn `{id, name, province}`, phân trang theo `page`, tìm theo tên không dấu bằng `q`, công khai, không cần `X-API-Key`); thêm `location.province` vào bảng `TempleSummary`. Trang [Đăng ký và đăng nhập](/guides/register-and-login/) trỏ `ordination_temple_id` tới `options` thay vì danh sách đầy đủ.
- `openapi.yaml`: sửa ví dụ `province` của `options` (`Thừa Thiên Huế` → `Huế`, theo 34 đơn vị cấp tỉnh), bỏ câu "có thể vẫn là tên tỉnh cũ", sửa mô tả `total` khi có `q`, thêm ví dụ `INVALID_PAGE`.

- Thêm **gia phả** (cần đăng nhập, 21 endpoint). Hai vai trò: `owner` (người tạo) xem và sửa; `viewer` (người được mời) chỉ xem. Người không có vai trò nhận `404 GENEALOGY_NOT_FOUND` giống gia phả không tồn tại.
  - Dựng cây: thuỷ tổ (nam) và vợ/chồng khi tạo; thêm con / vợ-chồng; sửa; xoá từ đời dưới lên; `PUT .../relation` sửa nhầm quan hệ (dời cả nhánh). Con của nữ dòng họ không vào cây. Lệch năm sinh con/cha chỉ trả `warnings`.
  - Phả ký thay thế toàn bộ, gồm hoành phi và hai vế câu đối đều có phiên âm và nghĩa. Ngày chạp họ và ngày giỗ lưu **dương lịch** kèm cờ `*_entered_as_lunar`; server trả thêm ngày âm (`chap_lunar_*`, `death_lunar_*`).
  - Vị trí "tôi" (`my_member_id`) là nút có `phone_number` trùng số tài khoản; một số chỉ thuộc một nút trong gia phả.
  - Mời: owner chọn một thành viên còn sống và mời bằng số điện thoại (`pending`); người được mời xem ở `GET /v1/invitations`, chấp nhận hoặc từ chối; owner thấy trạng thái đổi. Không có SMS/push ở v1.
  - Hồ sơ thành viên tính ở server: tuổi (`age` tây và `traditional_age` mụ), họ hàng, đường từ thuỷ tổ, chi, xưng hô theo đời (không theo tuổi).
  - Mã lỗi mới: `GENEALOGY_NOT_FOUND`, `GENEALOGY_FORBIDDEN`, `GENEALOGY_LIMIT_REACHED`, `MEMBER_NOT_FOUND`, `MEMBER_LIMIT_REACHED`, `MEMBER_HAS_DESCENDANTS`, `FOUNDER_NOT_DELETABLE`, `FOUNDER_NOT_MOVABLE`, `MEMBER_PHONE_ALREADY_USED`, `VIEWER_ALREADY_EXISTS`, `VIEWER_LIMIT_REACHED`, `VIEWER_NOT_FOUND`, `INVITATION_NOT_FOUND`, `INVITATION_NOT_PENDING`, `INVITATION_ALREADY_PENDING`.
  - Tài liệu: thêm trang [Gia phả](/guides/catalogues/genealogies/); bảng Xử lý lỗi cập nhật.

- Thêm **thắp đèn và vườn đèn** (cần đăng nhập):
  - `POST /v1/lamps` thắp một ngọn đèn (`lamp_type`, `intention`, `recipient`, `prayer`, `duration_days` 1/7/21/49, `is_anonymous`); trả `201 Lamp`. Gửi lại đúng nội dung trong 5 giây trả `200` với đèn đã có. Tối đa 20 đèn đang sáng mỗi người (`409 LAMP_LIMIT_REACHED`).
  - `GET /v1/lamps` trả đèn đang sáng của mọi người, lọc theo `lamp_type`, tối đa 12 đèn mỗi trang. **Phân trang bằng `cursor`/`next_cursor`/`has_more`**, không dùng `page`/`total_pages` như các danh sách khác. Kèm `total` và `counts` (số đèn từng loại) của toàn bộ đèn đang sáng, không đổi theo bộ lọc.
  - Tâm nguyện và lời nguyện chỉ trả cho người thắp; vườn đèn chỉ có `id`, `lamp_type`, `recipient`, `lighter_name`, `end_at`. Tên người thắp là pháp danh lúc thắp, hoặc `Vô danh`.
  - Mã lỗi mới: `LAMP_LIMIT_REACHED`, `INVALID_LIMIT`, `INVALID_CURSOR`; `INVALID_FILTER` thêm trường hợp `lamp_type`.
- Tài liệu: thêm trang [Thắp đèn và vườn đèn](/guides/catalogues/lamps/); bảng Xử lý lỗi cập nhật.

### 2026-10-10

- Thêm `POST /v1/temple-suggestions`: người dùng đề xuất một ngôi chùa chưa có (`name`, `province`, `address`); trả `201 {id, status: "pending"}`. Công khai, giới hạn 5 đề xuất mỗi IP mỗi giờ (`429 RATE_LIMITED`, kèm `Retry-After`). Không có route đọc lại đề xuất. Duyệt đề xuất do backoffice làm, chưa có route.
- `verify-otp`, `login`, `refresh` trả thêm `expires_in` (giây, thời gian sống của access token) và luôn có `Cache-Control: no-store`.
- Header `X-API-Key` cho client web (`401 INVALID_API_KEY` khi sai); `device.platform = web` ở `login` yêu cầu key đúng. Không gửi header thì xử lý như mobile.
- Tài liệu: thêm trang [Đề xuất thêm chùa](/guides/catalogues/temple-suggestions/); bảng Xử lý lỗi thêm `INVALID_API_KEY` và `RATE_LIMITED`.

### 2026-10-07

- Tài liệu: thêm hướng dẫn tích hợp cho Sổ giỗ chạp, Kinh/kệ/chú, Ảnh đẹp, Pháp thoại và Ảnh theo ngày; bổ sung mã lỗi mới vào bảng Xử lý lỗi. Trang Xác thực ghi rõ `/v1/memorials` cần token.

### 2026-09-30

- **Thay đổi không tương thích**: `GET /v1/bodhisattvas?page=N` (chỉ Bồ tát) được thay bằng `GET /v1/deities?page=N`, danh sách phân trang gồm cả Phật và Bồ tát (mỗi mục có `type`, `total` là số mục của cả hai loại; dữ liệu hiện tại `total = 18`). Route cũ bị xoá, không có alias. Lỗi và cách phân trang không đổi (`400 INVALID_PAGE`, trang vượt cuối trả `items: []`).
- Thêm 3 endpoint bài viết Phật pháp (công khai, chỉ đọc): `GET /v1/articles/top` (5 bài), `GET /v1/articles?page=N` (10 bài mỗi trang), `GET /v1/articles/{id}`. Lỗi mới: `404 ARTICLE_NOT_FOUND`; `page` sai vẫn là `400 INVALID_PAGE`. `published_date` và `image_url` là `null` khi chưa có (hiện `image_url` là `null` ở mọi bài).
- `GET /v1/temples/{id}`: thêm 3 field nội dung `introduction`, `architecture_heritage` (chuỗi) và `visiting_notes` (mảng chuỗi). Chùa chưa có nội dung trả `""` / `[]`, không trả `null`.
- Tài liệu: nhóm Attestation được viết lại theo cùng mẫu (bảng field, example cho từng verdict, mã lỗi theo endpoint, `413`). Sửa tên key trong `signals` của Android: đúng là `app_recognition_verdict`, `device_recognition_verdict`, `app_licensing_verdict` (tài liệu cũ ghi `app_recognition`). Hướng dẫn Attestation thêm định dạng `clientData` và bảng lỗi `401` của assertion.
- Tài liệu: nhóm Auth trong API Reference được viết lại cho ngang với nhóm danh mục: bảng field của request và response, example cho response thành công, mã lỗi liệt kê theo từng endpoint kèm example, và bổ sung `413 PAYLOAD_TOO_LARGE`. Hướng dẫn Đăng ký và đăng nhập thêm bảng field và bảng mã lỗi. Không đổi route hay field nào.
- Tài liệu: API Reference liệt kê đầy đủ field trong `items` của `top`/danh sách và response chi tiết cho cả chùa lẫn Phật/Bồ tát, kèm example lấy từ server thật. Sửa example `type` của Phật/Bồ tát thành `BUDDHA` / `BODHISATTVA`.

## Backoffice (`:8081`)

5 endpoint admin: `admin/auth/login`, `admin/auth/change-password`, `admin/me`, `admin/users`, `admin/users/{id}/reset-password`. Xem mục Admin trong API Reference.

### Chưa có

Đã có spec nhưng chưa có route, nên chưa được tài liệu hoá: hồ sơ và avatar, đặt lại mật khẩu, nhật ký và thực hành.

:::note
Khi thêm hoặc đổi endpoint, cập nhật `docs/specs/api/openapi.yaml` (nguồn duy nhất; `public/openapi.yaml` được chép từ đó khi chạy `npm run dev`/`build`) và ghi một mục vào đây.
:::
