---
title: Bảo mật
description: Các cơ chế bảo vệ đã có, để client biết hành vi của server và tích hợp đúng cách.
---

Trang này mô tả hành vi bảo mật **quan sát được từ phía client**, tổng hợp từ spec và code.

## Mật khẩu và mã xác minh (email)

- Mã xác minh gửi qua **email** (không dùng SMS). Mật khẩu và mã đều được băm bằng **bcrypt**, không lưu dạng rõ; mã chỉ xuất hiện trong thư. Mật khẩu được băm SHA-256 trước khi qua bcrypt, nên đủ 64 ký tự đều được tính bất kể độ dài byte.
- Mật khẩu 8 đến 64 ký tự (kiểm ở use case, lỗi `INVALID_PASSWORD`), không có luật thành phần.
- Mã sống 10 phút (dev). Sai 3 lần thì khoá mã 30 phút (`OTP_LOCKED`); mỗi lượt nhập được đếm trước khi so mã nên đoán song song không vượt được số lần.
- Số lần **gửi** mã cũng có giới hạn và cooldown theo email (`OTP_RESEND_TOO_SOON`, `OTP_SEND_LIMITED`), cộng giới hạn theo IP và ngân sách thư mỗi ngày (`RATE_LIMITED`), để chặn việc lạm dụng gửi thư.
- **Không lộ email đã đăng ký**: `register/request-otp` trả cùng `200` cho email mới và email đã có tài khoản (với email đã có, thư gửi đi là thư báo thay cho mã), `verify-otp` trả `OTP_INVALID` cho mọi mã trong trường hợp đó, và `password-reset/request-otp` trả cùng `200` cho mọi email hợp lệ. Ngoại lệ chấp nhận: `409 PHONE_TAKEN` khi số điện thoại thuộc tài khoản khác.
- **Chống chiếm lượt đăng ký**: khi email đang có mã còn sống, một yêu cầu đăng ký với dữ liệu khác bị từ chối `409 REGISTRATION_IN_PROGRESS` và không đổi gì, nên người lạ không thay được mật khẩu của người đang đăng ký dở.
- Đặt lại mật khẩu (`password-reset/confirm`) thu hồi mọi phiên của tài khoản và gỡ khoá đăng nhập của email.

## Đăng nhập

- Đăng nhập bằng **email**. Sai mật khẩu 5 lần liên tiếp (mỗi lần cách lần trước tối đa 15 phút) thì khoá email đó 15 phút (`LOGIN_LOCKED`). Khi đang khoá, server dừng ngay, không so mật khẩu. Bộ đếm khoá giữ cả cho email chưa có tài khoản, nên khoá không lộ email nào đã đăng ký.
- Email chưa đăng ký, email mới đăng ký dở và tài khoản chưa xác minh vẫn chạy qua một phép so bcrypt giả và trả `INVALID_CREDENTIALS` ("Email hoặc mật khẩu không đúng"). Thời gian phản hồi và mã lỗi giống hệt sai mật khẩu, nên không dò được tài khoản nào tồn tại.
- Dữ liệu sai định dạng bị từ chối trước khi chạm DB và không tính vào số lần sai.

## Phiên và token

- Mỗi thiết bị có một phiên riêng. Đăng nhập lại trên cùng thiết bị thu hồi phiên cũ.
- Access token mang claim `sid` (id phiên). Logout thu hồi đúng phiên đó, client không thể chỉ định phiên của người khác.
- Server lưu **hash SHA-256** của refresh token, không lưu token gốc.
- **Xoay vòng refresh token**: mỗi lần refresh cấp token mới và vô hiệu token cũ. Thời hạn phiên gia hạn theo mỗi lần refresh.
- **Phát hiện dùng lại**: dùng một refresh token đã bị xoay sẽ thu hồi phiên của thiết bị đó ngay.
- Refresh token không dùng được làm Bearer, và ngược lại (server kiểm loại token).
- Mọi route Bearer (trừ `logout`) kiểm phiên của access token còn sống ở mỗi request: đăng xuất hoặc đặt lại mật khẩu có hiệu lực ở request kế tiếp, không đợi access token hết hạn. Không kiểm tra được phiên thì trả `503`, không cho qua.

## Request

- Decode JSON chặt: từ chối field lạ, body sai kiểu, body quá 1 MB.
- Lỗi hệ thống (`5xx`) không bao giờ lộ nguyên nhân nội bộ trong `message`.
- Lỗi xác thực không nói rõ lý do (hết hạn, giả mạo hay sai loại).
- Mọi route `/v1/auth/*` trừ `logout` bị giới hạn **theo IP** (`429 RATE_LIMITED` kèm `Retry-After`), đếm trong store dùng chung nên đúng khi chạy nhiều instance. Nếu store lỗi thì đếm theo từng instance, không bao giờ bỏ giới hạn.
- IP client lấy theo thứ tự: header `X-InnerLamp-Client-IP` khi request có `X-API-Key` đúng (website chuyển IP khách vào đây); nếu không và cấu hình tin proxy thì mục **cuối cùng** của `X-Forwarded-For` (do proxy tin cậy thêm vào, các mục trước do client tự khai nên bị bỏ); nếu không thì địa chỉ kết nối. Header không kèm key đúng không bao giờ được đọc.
- Mọi response của `register/*`, `login`, `refresh` và `password-reset/*` (kể cả lỗi) có `Cache-Control: no-store`.

## Chống giả mạo app

Attestation (Play Integrity, App Attest) cho phép server kiểm tra request đến từ app thật. Challenge dùng một lần, chống replay. Với iOS, mỗi assertion có bộ đếm phải tăng dần. Xem [Attestation](/guides/attestation/).

:::caution[Trước khi lên production]
Các mục sau là bản dev và phải thay: mã xác minh cố định `111111` (`otp.dev_fixed_code`), mailer `log` chỉ ghi thư vào log (cả hai bị chặn khởi động ngoài môi trường `dev`), attestation dùng `mock`, secret đọc plaintext từ file cấu hình, và HTTP server chưa cấu hình timeout.
:::
