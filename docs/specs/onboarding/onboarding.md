# Onboarding

---

# User Stories — Tâm Đăng (InnerLamp)

Tài liệu tổng hợp user story theo từng tính năng, viết theo thứ tự ưu tiên xây dựng (bắt đầu từ module `identity`).

---

## US-001: Đăng ký tài khoản

**Module:** `identity`
**Priority:** P0 (bắt buộc, chặn mọi tính năng cần user context)

### User Story

> Là một **người dùng mới**,
> Tôi muốn **đăng ký tài khoản bằng email, mật khẩu và số điện thoại**,
> Để **có thể lưu và đồng bộ hành trình tu tập của mình qua nhiều thiết bị**.

### Các trường thông tin

| Trường | Bắt buộc? | Ghi chú |
|---|---|---|
| Email | Bắt buộc | Dùng để đăng nhập, phải unique |
| Mật khẩu | Bắt buộc | ≥ 8 ký tự |
| Số điện thoại | Bắt buộc | Chuẩn hóa định dạng E.164 (`+84...`), không bắt buộc verify OTP ở MVP nhưng lưu sẵn cho tính năng sau (nhắc nhở qua SMS, khôi phục tài khoản...) |
| Pháp danh (dharma name) | Tùy chọn | Tên đạo, nếu đã quy y |
| Nơi quy y / xuất gia | Tùy chọn | Tên chùa/tổ đình nơi quy y hoặc xuất gia |

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
  ALTER TABLE users ADD COLUMN phone_number VARCHAR(20) UNIQUE NOT NULL;
  ALTER TABLE users ADD COLUMN ordination_place VARCHAR(255);
  ```
  (đặt `dharma_name` đã có sẵn từ thiết kế trước, giữ nguyên)
- Access token JWT sống 15 phút, refresh token sống 30 ngày, rotate mỗi lần dùng

---

## US-002: Đăng nhập

**Module:** `identity`
**Priority:** P0 (bắt buộc, chặn mọi tính năng cần user context)

### User Story

> Là một **người dùng đã có tài khoản**,
> Tôi muốn **đăng nhập bằng email và mật khẩu**,
> Để **tiếp tục hành trình tu tập đã lưu trước đó trên thiết bị này hoặc thiết bị khác**.

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

---

## US-003: Quên mật khẩu

**Module:** `identity`
**Priority:** P1 (không chặn MVP đầu tiên, nhưng cần sớm — rủi ro UX nếu thiếu)

### User Story

> Là một **người dùng đã có tài khoản nhưng quên mật khẩu**,
> Tôi muốn **đặt lại mật khẩu thông qua email đã đăng ký**,
> Để **có thể tiếp tục đăng nhập mà không cần tạo tài khoản mới**.

### Acceptance Criteria

1. **Given** tôi đã có tài khoản, **when** tôi nhập email và nhấn "Quên mật khẩu", **then** hệ thống gửi một email chứa link/mã đặt lại mật khẩu, có hiệu lực trong thời gian giới hạn (ví dụ 15 phút).
2. **Given** tôi nhập một email chưa từng đăng ký, **when** tôi nhấn "Quên mật khẩu", **then** hệ thống vẫn hiển thị thông báo thành công giống hệt trường hợp email tồn tại ("Nếu email tồn tại, chúng tôi đã gửi hướng dẫn") — không tiết lộ email nào đã đăng ký, cùng lý do bảo mật như US-002 AC2/AC3.
3. **Given** tôi mở link/nhập mã đặt lại mật khẩu còn hiệu lực, **when** tôi nhập mật khẩu mới hợp lệ (≥ 8 ký tự) và xác nhận, **then** mật khẩu được cập nhật, mọi refresh token hiện có của tài khoản đó bị revoke ngay lập tức (buộc đăng nhập lại trên mọi thiết bị).
4. **Given** link/mã đặt lại mật khẩu đã hết hạn hoặc đã được dùng một lần, **when** tôi cố dùng lại, **then** hệ thống báo lỗi và yêu cầu thực hiện lại từ bước "Quên mật khẩu".
5. **Given** tôi yêu cầu đặt lại mật khẩu nhiều lần liên tiếp trong thời gian ngắn, **when** vượt ngưỡng cho phép, **then** hệ thống tạm giới hạn số lần gửi email cho địa chỉ đó — tránh bị lợi dụng làm spam email người khác.

### Out of scope (giai đoạn này)

- Đặt lại mật khẩu qua SMS/OTP số điện thoại — có thể cân nhắc thêm sau vì US-001 đã thu thập số điện thoại, nhưng chưa cần ở bản đầu
- Xác thực 2 lớp khi đặt lại mật khẩu — để lại cho giai đoạn sau

### Ghi chú kỹ thuật liên quan

- Endpoint gợi ý:
  ```
  POST /v1/auth/forgot-password   { email }
  POST /v1/auth/reset-password    { token, new_password }
  ```
- Cần thêm bảng lưu reset token (tương tự pattern `refresh_tokens` — lưu hash, có `expires_at`, có thể đánh dấu đã dùng thay vì revoke):
  ```sql
  CREATE TABLE password_reset_tokens (
      id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      token_hash  VARCHAR(255) NOT NULL,
      expires_at  TIMESTAMPTZ NOT NULL,
      used_at     TIMESTAMPTZ,
      created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  ```
- Cần dịch vụ gửi email (SendGrid, AWS SES, hoặc GCP tương đương) — đây là dependency ngoại vi mới, chưa có trong kiến trúc hiện tại, cần bổ sung vào `platform/` (ví dụ `platform/mailer`)
- Rate limiting (AC5) cùng cơ chế với US-002 AC6, áp dụng theo email thay vì theo IP

---

## US-004: Cập nhật hồ sơ cá nhân

**Module:** `identity`
**Priority:** P1

### User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **cập nhật pháp danh, nơi quy y/xuất gia và ảnh đại diện của mình**,
> Để **hồ sơ tu tập phản ánh đúng thông tin cá nhân, đặc biệt khi tôi mới quy y sau khi đã tạo tài khoản**.

### Acceptance Criteria

1. **Given** tôi đã đăng nhập, **when** tôi mở màn hình hồ sơ, **then** tôi thấy đầy đủ thông tin hiện có: email, số điện thoại, pháp danh, nơi quy y/xuất gia, ảnh đại diện (các trường trống hiển thị rõ là chưa cập nhật, không hiện "null" hay lỗi).
2. **Given** tôi đang ở màn hình hồ sơ, **when** tôi sửa pháp danh và/hoặc nơi quy y/xuất gia rồi lưu, **then** thông tin được cập nhật thành công và phản ánh ngay trên UI, không cần tải lại app.
3. **Given** tôi tải lên một ảnh đại diện mới, **when** tôi lưu, **then** ảnh cũ (nếu có) được thay thế, ảnh mới hiển thị ở mọi nơi có avatar trong app.
4. **Given** tôi cố sửa email hoặc số điện thoại từ màn hình hồ sơ, **when** tôi thao tác, **then** hệ thống **không cho sửa trực tiếp ở đây** — đổi email/số điện thoại cần luồng xác thực riêng (US riêng, chưa làm ở giai đoạn này) vì đây là thông tin định danh đăng nhập, đổi tùy tiện có rủi ro bảo mật.
5. **Given** tôi để trống pháp danh hoặc nơi quy y/xuất gia và lưu, **when** hệ thống xử lý, **then** trường đó được xóa (set về rỗng) — cho phép người dùng "gỡ bỏ" thông tin đã nhập trước đó nếu muốn.
6. **Given** tôi chưa đăng nhập (access token không hợp lệ hoặc hết hạn), **when** tôi cố gọi API cập nhật hồ sơ, **then** hệ thống trả về lỗi 401 và không cho phép thao tác.

### Out of scope (giai đoạn này)

- Đổi email hoặc số điện thoại — cần luồng xác thực lại (verify email mới/OTP) riêng, chưa làm ở giai đoạn này
- Đổi mật khẩu khi đã đăng nhập (khác với "quên mật khẩu" ở US-003) — cần user story riêng
- Xóa tài khoản (account deletion) — cần user story riêng, có thể liên quan yêu cầu pháp lý (GDPR-like) nếu mở rộng quốc tế sau này

### Ghi chú kỹ thuật liên quan

- Endpoint: `PATCH /v1/users/me` (đã có sẵn trong thiết kế API ban đầu), yêu cầu `RequireAuth` middleware
- Ảnh đại diện: upload lên Cloud Storage, lưu `avatar_url` trong bảng `users` — cần endpoint upload riêng (`POST /v1/users/me/avatar` hoặc dùng signed upload URL) vì `PATCH /v1/users/me` chỉ nên nhận JSON, không nhận file trực tiếp
- Thuộc `UserService` trong `usecase/` (tách khỏi `AuthService` vì đây là CRUD hồ sơ, không phải nghiệp vụ xác thực — theo naming convention đã thống nhất)