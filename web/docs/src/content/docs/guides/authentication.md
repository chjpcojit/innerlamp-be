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

- Đăng ký: `register/request-otp` rồi `register/verify-otp` (tạo tài khoản và đăng nhập luôn).
- Đăng nhập: `POST /v1/auth/login`.

Xem chi tiết ở [Đăng ký và đăng nhập](/guides/register-and-login/).

## Gọi API cần đăng nhập

```http
POST /v1/auth/logout HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIs…
```

`logout` và toàn bộ năm endpoint sổ giỗ chạp (`/v1/memorials`) cần token. Các endpoint còn lại (đăng ký, đăng nhập, refresh, attestation, danh mục) đều công khai.

Refresh token **không** dùng được làm Bearer. Gửi nhầm loại sẽ nhận `401 UNAUTHORIZED`.

## `expires_in`

`verify-otp`, `login` và `refresh` trả thêm `expires_in`: thời gian sống của `access_token` tính bằng **giây** (dev: 900). Lên lịch `refresh` theo giá trị này, không cần giải mã JWT. Nó không nói về hạn của `refresh_token`.

Các response này luôn có `Cache-Control: no-store` và `Pragma: no-cache`, nên không để proxy hay lớp cache của client giữ token.

## `X-API-Key` (client web)

Server của website gửi header `X-API-Key` (secret dùng chung của ứng dụng web, không phải token của user) trên các route `/v1/auth/*` và `POST /v1/temple-suggestions`.

- Header sai hoặc rỗng: `401 INVALID_API_KEY`.
- Không gửi header: coi là mobile, xử lý bình thường.
- `device.platform = web` ở `login` chỉ được chấp nhận khi có key đúng.
- Key đặt ở server web, **không** đưa xuống trình duyệt. Mobile không có key.

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

`POST /v1/auth/logout` thu hồi phiên hiện tại. Không có body: phiên được lấy từ claim `sid` trong access token. Gọi lại khi phiên đã bị thu hồi vẫn trả `200`.

Sau khi logout, xoá cả hai token khỏi thiết bị.

## Lưu token ở đâu

- **iOS**: Keychain.
- **Android**: `EncryptedSharedPreferences` hoặc Keystore.
- **Web**: tránh `localStorage` cho refresh token nếu có thể.
