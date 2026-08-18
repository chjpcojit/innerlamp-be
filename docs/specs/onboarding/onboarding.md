# Onboarding

---

# User Stories — Tâm Đăng (InnerLamp)

Tài liệu tổng hợp user story theo từng tính năng, viết theo thứ tự ưu tiên xây dựng (bắt đầu từ module `identity`).

---

## US-001: Đăng ký tài khoản

**Module:** `identity` **Priority:** P0 (bắt buộc, chặn mọi tính năng cần user context)

### User Story

> Là một **người dùng mới**, Tôi muốn **đăng ký tài khoản bằng email, mật khẩu và số điện thoại**, Để **có thể lưu và đồng bộ hành trình tu tập của mình qua nhiều thiết bị**.

### Các trường thông tin

|Trường|Bắt buộc?|Ghi chú|
|---|---|---|
|Email|Bắt buộc|Dùng để đăng nhập, phải unique|
|Mật khẩu|Bắt buộc|≥ 8 ký tự|
|Số điện thoại|Bắt buộc|Chuẩn hóa định dạng E.164 (`+84...`), không bắt buộc verify OTP ở MVP nhưng lưu sẵn cho tính năng sau (nhắc nhở qua SMS, khôi phục tài khoản...)|
|Pháp danh (dharma name)|Tùy chọn|Tên đạo, nếu đã quy y|
|Nơi quy y / xuất gia|Tùy chọn|Tên chùa/tổ đình nơi quy y hoặc xuất gia|

> **Giả định:** vì bạn tách riêng "thu thập thêm số điện thoại" khỏi nhóm "trường option", mình hiểu số điện thoại là **bắt buộc**, còn pháp danh và nơi quy y/xuất gia là **tùy chọn**. Nếu ý bạn là số điện thoại cũng optional, báo lại để mình chỉnh.

### Acceptance Criteria

1. **Given** tôi chưa có tài khoản, **when** tôi nhập email hợp lệ, số điện thoại hợp lệ, mật khẩu ≥ 8 ký tự và nhấn "Đăng ký", **then** tài khoản được tạo và tôi được đăng nhập ngay (nhận về access token + refresh token), không cần bước xác thực email/SMS riêng ở giai đoạn MVP.
2. **Given** email hoặc số điện thoại tôi nhập đã tồn tại trong hệ thống, **when** tôi nhấn "Đăng ký", **then** hệ thống báo lỗi tương ứng ("Email đã được sử dụng" / "Số điện thoại đã được sử dụng") và không tạo tài khoản trùng.
3. **Given** mật khẩu tôi nhập ngắn hơn 8 ký tự, **when** tôi nhấn "Đăng ký", **then** hệ thống báo lỗi ngay tại form, không gửi request lên server.
4. **Given** số điện thoại tôi nhập sai định dạng, **when** tôi nhấn "Đăng ký", **then** hệ thống báo lỗi ngay tại form.
5. **Given** tôi để trống trường Pháp danh và/hoặc Nơi quy y/xuất gia, **when** tôi đăng ký, **then** tài khoản vẫn được tạo bình thường — đây là các trường tùy chọn.
6. **Given** tôi đăng ký thành công, **when** hệ thống tạo tài khoản, **then** mật khẩu phải được hash (không lưu plaintext) trước khi lưu vào database.
7. **Given** tôi đăng ký thành công, **when** tôi mở lại app sau khi tắt hoàn toàn, **then** tôi vẫn ở trạng thái đã đăng nhập nhờ access token còn hạn (hoặc refresh token tự động cấp access token mới).

### Out of scope (giai đoạn MVP)

- Xác thực email hoặc OTP số điện thoại — để lại cho giai đoạn sau
- Đăng ký qua social login (Google/Apple) — để lại cho giai đoạn sau
- Quên mật khẩu / reset password — cần user story riêng, làm ngay sau US này vì là rủi ro UX nếu thiếu

### Ghi chú kỹ thuật liên quan

- Endpoint: `POST /v1/auth/register`
- Schema bảng `users` cập nhật thêm:
    
    ```sql
    ALTER TABLE users ADD COLUMN phone_number VARCHAR(20) UNIQUE NOT NULL;ALTER TABLE users ADD COLUMN ordination_place VARCHAR(255);
    ```
    
    (đặt `dharma_name` đã có sẵn từ thiết kế trước, giữ nguyên)
- Access token JWT sống 15 phút, refresh token sống 7 ngày (khớp `pkg/session.DefaultConfig().RefreshTokenTTL`), rotate mỗi lần dùng

---

## US-002: Đăng nhập

**Module:** `identity` **Priority:** P0 (bắt buộc, chặn mọi tính năng cần user context)

### User Story

> Là một **người dùng đã có tài khoản**, Tôi muốn **đăng nhập bằng email và mật khẩu**, Để **tiếp tục hành trình tu tập đã lưu trước đó trên thiết bị này hoặc thiết bị khác**.

### Acceptance Criteria

1. **Given** tôi đã có tài khoản, **when** tôi nhập đúng email và mật khẩu, **then** tôi được đăng nhập thành công và nhận về access token + refresh token.
2. **Given** tôi nhập sai mật khẩu, **when** tôi nhấn "Đăng nhập", **then** hệ thống báo lỗi chung "Email hoặc mật khẩu không đúng" — **không** tiết lộ cụ thể là sai email hay sai mật khẩu, tránh lộ thông tin email nào đã tồn tại trong hệ thống.
3. **Given** tôi nhập một email chưa từng đăng ký, **when** tôi nhấn "Đăng nhập", **then** hệ thống trả về **cùng một thông báo lỗi chung** như trường hợp sai mật khẩu (US-002 AC2) — cùng lý do bảo mật.
4. **Given** tôi đăng nhập thành công trên thiết bị mới, **when** tôi đã từng đăng nhập trên thiết bị khác, **then** cả hai thiết bị đều giữ phiên đăng nhập riêng biệt (mỗi thiết bị có refresh token riêng) — đăng nhập thiết bị mới không tự động đăng xuất thiết bị cũ ở giai đoạn MVP.
5. **Given** tôi đăng nhập thành công, **when** access token hết hạn sau 15 phút, **then** app tự động dùng refresh token để lấy access token mới mà không bắt tôi đăng nhập lại, miễn refresh token còn hạn và chưa bị revoke.
6. **Given** tôi nhập sai mật khẩu nhiều lần liên tiếp trong thời gian ngắn, **when** vượt ngưỡng cho phép, **then** hệ thống tạm khóa việc thử đăng nhập cho email đó trong một khoảng thời gian (rate limiting) — chống brute-force.

### Out of scope (giai đoạn MVP)

- Đăng nhập qua social login (Google/Apple) — để lại cho giai đoạn sau
- Đăng xuất tất cả thiết bị khác ("logout everywhere") — để lại cho giai đoạn sau
- Xác thực 2 lớp (2FA) — để lại cho giai đoạn sau

### Ghi chú kỹ thuật liên quan

- Endpoint: `POST /v1/auth/login`
- Dùng `bcrypt.CompareHashAndPassword` để so khớp mật khẩu, không so sánh plaintext
- Rate limiting (AC6) nên implement ở middleware layer hoặc API gateway, không phải trong usecase — tránh lẫn logic bảo mật hạ tầng vào logic nghiệp vụ
- Response giống hệt US-001 (`access_token`, `refresh_token`, thông tin user)
- 