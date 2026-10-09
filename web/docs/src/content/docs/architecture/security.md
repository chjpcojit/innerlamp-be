---
title: Bảo mật
description: Các cơ chế bảo vệ đã có, để client biết hành vi của server và tích hợp đúng cách.
---

Trang này mô tả hành vi bảo mật **quan sát được từ phía client**, tổng hợp từ spec và code.

## Mật khẩu và OTP

- Mật khẩu và mã OTP đều được băm bằng **bcrypt**, không lưu dạng rõ.
- Mật khẩu tối thiểu 8 ký tự (kiểm ở use case, lỗi `INVALID_PASSWORD`).
- OTP sống 90 giây. Sai quá số lần cho phép thì khoá nhập OTP 30 phút (`OTP_LOCKED`).
- Số lần **gửi** OTP cũng có giới hạn và cooldown (`OTP_RESEND_TOO_SOON`, `OTP_SEND_LIMITED`), để chặn việc lạm dụng SMS.
- `verify-otp` trả cùng mã `OTP_INVALID` cho "sai mã" và "không có lượt đăng ký", để không ai dò được số điện thoại nào đang đăng ký dở.

## Đăng nhập

- Sai mật khẩu 5 lần trong 15 phút thì khoá 15 phút (`LOGIN_LOCKED`). Khi đang khoá, server dừng ngay, không so mật khẩu.
- Số điện thoại chưa đăng ký vẫn chạy qua một phép so bcrypt giả và trả `INVALID_CREDENTIALS`. Thời gian phản hồi và mã lỗi giống hệt sai mật khẩu, nên không dò được tài khoản nào tồn tại.
- Dữ liệu sai định dạng bị từ chối trước khi chạm DB và không tính vào số lần sai.

## Phiên và token

- Mỗi thiết bị có một phiên riêng. Đăng nhập lại trên cùng thiết bị thu hồi phiên cũ.
- Access token mang claim `sid` (id phiên). Logout thu hồi đúng phiên đó, client không thể chỉ định phiên của người khác.
- Server lưu **hash SHA-256** của refresh token, không lưu token gốc.
- **Xoay vòng refresh token**: mỗi lần refresh cấp token mới và vô hiệu token cũ. Thời hạn phiên gia hạn theo mỗi lần refresh.
- **Phát hiện dùng lại**: dùng một refresh token đã bị xoay sẽ thu hồi phiên của thiết bị đó ngay.
- Refresh token không dùng được làm Bearer, và ngược lại (server kiểm loại token).

## Request

- Decode JSON chặt: từ chối field lạ, body sai kiểu, body quá 1 MB.
- Lỗi hệ thống (`5xx`) không bao giờ lộ nguyên nhân nội bộ trong `message`.
- Lỗi xác thực không nói rõ lý do (hết hạn, giả mạo hay sai loại).
- IP client lấy từ kết nối, **không** tin header `X-Forwarded-For`. Giá trị này chỉ dùng làm metadata thiết bị, không dùng để quyết định bảo mật.

## Chống giả mạo app

Attestation (Play Integrity, App Attest) cho phép server kiểm tra request đến từ app thật. Challenge dùng một lần, chống replay. Với iOS, mỗi assertion có bộ đếm phải tăng dần. Xem [Attestation](/guides/attestation/).

:::caution[Trước khi lên production]
Các mục sau là bản dev và phải thay: OTP cố định `111111`, SMS chỉ ghi log, attestation dùng `mock`, secret đọc plaintext từ file cấu hình, và HTTP server chưa cấu hình timeout.
:::
