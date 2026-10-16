---
title: Xác thực
description: Access token, refresh token, và cách giữ phiên đăng nhập.
---

API dùng cặp JWT: **access token** để gọi API và **refresh token** để lấy cặp mới.

| | Access token | Refresh token |
| --- | --- | --- |
| Dùng để | Gọi API cần đăng nhập | Gọi `POST /v1/auth/refresh` |
| Gửi ở đâu | Header `Authorization: Bearer <token>` | Body của `refresh` |
| Thời hạn (cấu hình dev) | 15 phút | 7 ngày, gia hạn theo mỗi lần refresh |

:::caution
Thời hạn lấy từ cấu hình server, môi trường thật có thể khác. Đừng hard-code, hãy xử lý `401` khi token hết hạn.
:::

## Lấy token

Có hai đường, đều trả `access_token`, `refresh_token`, `expires_in` và `user`:

- Đăng ký: `register/request-otp` rồi `register/verify-otp` bằng mã gửi qua email (tạo tài khoản và đăng nhập luôn).
- Đăng nhập: `POST /v1/auth/login` bằng email và mật khẩu.

Quên mật khẩu: `password-reset/request-otp` rồi `password-reset/confirm`. Route này **không** trả token, người dùng đăng nhập lại.

Xem chi tiết ở [Đăng ký và đăng nhập](/guides/register-and-login/).

## Gọi API cần đăng nhập

```http
POST /v1/auth/logout HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIs…
```

`logout`, toàn bộ năm endpoint sổ giỗ chạp (`/v1/memorials`) và hai endpoint thắp đèn (`/v1/lamps`) cần token. Năm endpoint hồ sơ cá nhân (`/v1/users/me`, đã có spec, chưa có ở server) cũng sẽ cần token và luôn thao tác trên chính người dùng trong token. Các endpoint còn lại (đăng ký, đăng nhập, refresh, attestation, danh mục) đều công khai.

Refresh token **không** dùng được làm Bearer. Gửi nhầm loại sẽ nhận `401 UNAUTHORIZED`.

Mọi route Bearer (trừ `logout`) còn kiểm **phiên** của access token sau khi kiểm chữ ký. Phiên đã đăng xuất, bị thu hồi (đặt lại mật khẩu, phát hiện dùng lại refresh token) hoặc hết hạn thì request kế tiếp nhận `401 UNAUTHORIZED`, không phải đợi access token hết hạn 15 phút. Nếu server không kiểm tra được phiên (mất kết nối DB) thì trả `503 SERVICE_UNAVAILABLE`, không bao giờ cho qua. Khi gặp `401` sau khi đã refresh bất thành, đưa người dùng về màn hình đăng nhập.

## `expires_in`

`register/verify-otp`, `login` và `refresh` trả thêm `expires_in`: thời gian sống của `access_token` tính bằng **giây** (dev: 900). Lên lịch `refresh` theo giá trị này, không cần giải mã JWT. Nó không nói về hạn của `refresh_token`.

Mọi response của `register/*`, `login`, `refresh` và `password-reset/*` (kể cả lỗi) luôn có `Cache-Control: no-store` và `Pragma: no-cache`, nên không để proxy hay lớp cache của client giữ token.

## `X-API-Key` (client web)

Server của website gửi header `X-API-Key` (secret dùng chung của ứng dụng web, không phải token của user) trên các route `/v1/auth/*`, `POST /v1/temple-suggestions` và `GET /v1/administrative-units/*`.

- Header sai hoặc rỗng: `401 INVALID_API_KEY`.
- Không gửi header: coi là mobile, xử lý bình thường.
- `device.platform = web` ở `login` chỉ được chấp nhận khi có key đúng.
- Key đặt ở server web, **không** đưa xuống trình duyệt. Mobile không có key.
- Vì website gọi API từ server, giới hạn tần suất theo IP sẽ thấy IP của server web. Server web gửi thêm header `X-InnerLamp-Client-IP` với IP của khách; backend **chỉ đọc** header này khi request mang `X-API-Key` đúng, và bỏ qua nếu không phải đúng một địa chỉ IP. Mobile bị tính theo địa chỉ kết nối.

## Làm mới token

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    C->>S: Gọi API với access token
    S-->>C: 401 UNAUTHORIZED (hết hạn)
    C->>S: POST /v1/auth/refresh {refresh_token}
    alt refresh hợp lệ
        S-->>C: 200 {access_token mới, refresh_token mới, expires_in}
        C->>C: Lưu cặp mới, bỏ refresh token cũ
        C->>S: Gọi lại API với access token mới
    else refresh không hợp lệ
        S-->>C: 401 REFRESH_TOKEN_INVALID
        C->>C: Xoá token, đưa người dùng về màn hình đăng nhập
    end
```

Quy tắc quan trọng:

1. **Refresh token xoay vòng.** Mỗi lần refresh nhận về một refresh token mới, token cũ không dùng lại được. Luôn lưu token mới.
2. **Dùng lại token cũ bị coi là dấu hiệu bị đánh cắp.** Server thu hồi luôn phiên của thiết bị đó và trả `401 REFRESH_TOKEN_INVALID`. Người dùng phải đăng nhập lại.
3. **Đừng gọi refresh song song.** Nếu nhiều request cùng gặp 401, chỉ cho một request refresh, các request còn lại chờ kết quả. Hai lần refresh cùng lúc với cùng một token sẽ làm một lần bị coi là dùng lại.

## Đăng xuất

`POST /v1/auth/logout` thu hồi phiên hiện tại. Không có body: phiên được lấy từ claim `sid` trong access token. Gọi lại khi phiên đã bị thu hồi vẫn trả `200`, nên có thể thử lại an toàn.

| Tình huống | Kết quả |
| --- | --- |
| Access token hợp lệ, phiên còn hoạt động | `200` `{"success": true}`, refresh token của phiên đó không dùng được nữa |
| Phiên đã bị thu hồi hoặc không tồn tại | `200` `{"success": true}`, không đổi gì |
| Thiếu token, token sai hoặc hết hạn, gửi refresh token, hoặc token thiếu `sid` | `401 UNAUTHORIZED` |
| Web gửi `X-API-Key` sai hoặc rỗng | `401 INVALID_API_KEY` |

Cần biết:

- Chỉ phiên của thiết bị đang gọi bị thu hồi. Các thiết bị khác của cùng tài khoản vẫn đăng nhập.
- Sau khi logout, các route Bearer khác từ chối access token của phiên đó ở request kế tiếp (`401 UNAUTHORIZED`), vì server kiểm phiên còn sống ở mỗi request.
- Logout không bị giới hạn theo IP và không kiểm phiên còn sống, nên luôn trả `200` cho phiên đã thu hồi.
- Access token đã hết hạn thì logout trả `401`. Có thể `refresh` rồi logout; hoặc chỉ cần xoá token cục bộ.

Sau khi logout, xoá cả hai token khỏi thiết bị, bất kể response là gì.

## Lưu token ở đâu

- **iOS**: Keychain.
- **Android**: `EncryptedSharedPreferences` hoặc Keystore.
- **Web**: tránh `localStorage` cho refresh token nếu có thể.
