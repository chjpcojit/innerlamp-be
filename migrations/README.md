# Migrations — innerlamp

## Mục đích
Schema Postgres cho appService. Mỗi file là một bước thay đổi schema, đánh số tăng dần, áp theo thứ tự.

Hiện có **ba** file:

| File | Feature | Nội dung |
|---|---|---|
| `0001_register_onboarding.sql` | US-001 | `users`, `otp_codes`, `pending_registration`, `user_sessions` |
| `0002_login_devices_and_rate_limit.sql` | US-002 | `user_devices`, `login_attempts`, `user_sessions.device_id` + `previous_refresh_token_hash` |
| `0003_otp_send_limits.sql` | US-001 v5.0 | `otp_send_limits` + drop `otp_codes.resend_count` |

`0002` là **tiến hoá** của `0001`, không phải bootstrap: nó `ALTER TABLE user_sessions ADD COLUMN` nên **phải chạy sau** `0001`. Chạy `0002` một mình trên DB trống sẽ lỗi `relation "user_sessions" does not exist`. Cả ba file đều **idempotent**: mọi câu lệnh là `CREATE TABLE IF NOT EXISTS` / `ADD COLUMN IF NOT EXISTS` / `CREATE INDEX IF NOT EXISTS` / `DROP COLUMN IF EXISTS`, nên apply lại chỉ sinh `NOTICE ... skipping` chứ không lỗi (đã kiểm chứng: chạy `migrate` lần hai exit 0). Service `migrate` trong `docker-compose.yml` dựa vào tính chất này — xem `docs/development/local-stack.md`.

`0003` chạy sau `0001` (nó `ALTER TABLE otp_codes DROP COLUMN`), và **idempotent**: `CREATE TABLE IF NOT EXISTS` + `DROP COLUMN IF EXISTS`, nên apply lại không lỗi.

Thứ tự apply:

```bash
psql "$DATABASE_URL" -f migrations/0001_register_onboarding.sql
psql "$DATABASE_URL" -f migrations/0002_login_devices_and_rate_limit.sql
psql "$DATABASE_URL" -f migrations/0003_otp_send_limits.sql
```

## ⚠️ Apply bằng `psql -f`, KHÔNG qua `scripts/migrate.sh`

Đây là quyết định có chủ ý, không phải sót việc:

- File đặt tên **bare** `0001_register_onboarding.sql` — khớp tên gọi trong `docs/specs/features/onboarding/design.md` và `tasks.md`.
- `scripts/migrate.sh` dùng **golang-migrate**, công cụ này chỉ nhận cặp `{version}_{name}.up.sql` + `{version}_{name}.down.sql`. File bare **không** được nó nhìn thấy.
- Hệ quả kéo theo: `scripts/check-migrations.sh` glob `*.up.sql`, tìm thấy **0 file**, và **pass rỗng**. Không phải bug — nhưng đừng đọc kết quả đó là bằng chứng migration hợp lệ.

Đường apply chính thức:

```bash
psql "$DATABASE_URL" -f migrations/0001_register_onboarding.sql
```

Trong local stack, service `migrate` của `docker-compose.yml` chạy đúng các lệnh `psql -f` này cho mọi file `*.sql` theo thứ tự tên (`make migrate`).

File dùng `CREATE TABLE IF NOT EXISTS` nên chạy lại không lỗi, nhưng cũng **không** migrate dữ liệu đã có — nó là bootstrap schema, không phải migration tiến hoá.

Nếu sau này cần golang-migrate thật (rollback, version tracking, CI apply), đổi tên thành cặp `.up.sql`/`.down.sql` và viết file `down` — chưa làm vì chưa cần.

## Bất biến schema phải giữ

- **`users_email_key` / `users_phone_number_key`** — tên constraint do Postgres tự đặt cho `UNIQUE` inline. `repository/postgres_registration_repository.go` dịch `pq` error `23505` theo **tên constraint** để trả 409 đúng (`EMAIL_TAKEN`/`PHONE_TAKEN`). Đổi tên constraint, hoặc tạo constraint tương đương dưới tên khác, âm thầm biến 409 của người thua race thành 500. Đây là ràng buộc thật giữa SQL và Go, không phải chi tiết trang trí.
- **`chk_otp_codes_purpose`** liệt kê **đủ bốn** purpose (`register`, `login`, `reset_password`, `verify_email`) ngay từ đầu, dù US-001 chỉ hiện thực `register`. Khai đủ để US-002/US-003 không phải `ALTER` constraint. Tập hằng tương ứng ở `domain/otp.go`.
- **`pending_registration.email` cố ý KHÔNG unique.** Chỉ hàng `users` đã hoàn tất mới có thẩm quyền về unique email; hai người có thể cùng giữ pending cho một địa chỉ chưa verify, và `CreateUserAndSession` re-check bên trong transaction của nó. Đặt UNIQUE ở đây sẽ chặn người thứ hai bằng một lỗi DB họ không xử lý được.
- **`pending_registration.phone_number` CÓ unique** — đây là target của `ON CONFLICT (phone_number) DO UPDATE`, cơ chế khiến một lần đăng ký dở không để lại hai password cạnh tranh.
- **`user_sessions.refresh_token_hash`** lưu SHA-256 hex của refresh token, **không** lưu token thô. Access token không lưu: nó ngắn hạn và verify bằng chữ ký.
- **`user_sessions.expires_at`** = `now() + JWTConfig.RefreshTokenExpiration` (`168h` trong `config.dev.yaml`), do use-case truyền vào. Không default trong SQL — nếu default ở đây thì con số trong DB và con số trong refresh token có hai nguồn và sẽ lệch.
- **`users.email_verified_at` NULL** lúc tạo. US-001 verify **số điện thoại**, không verify email — cột này ghi nhận bước chưa tồn tại.
- **`user_devices_user_id_device_id_key`** — UNIQUE `(user_id, device_id)`, target của `ON CONFLICT` trong `CreateLoginSession`. `device_id` là **fingerprint server-side** (`SHA-256(platform|os_version|device_model|user_agent)`, hex 64 ký tự), không phải giá trị client gửi lên — client không thể tự khai một `device_id` để ghi đè thiết bị người khác.
- **`uq_user_sessions_active_device`** — partial unique index trên `(device_id) WHERE is_revoked = false AND device_id IS NOT NULL`. Đây là ràng buộc *"mỗi thiết bị tối đa một session active"* (AC4) được **DB** cưỡng chế, không chỉ bằng logic: hai request login đồng thời cùng một thiết bị thì một thắng, một nhận unique violation. `IS NOT NULL` là phần bắt buộc — session của US-001 (`device_id` NULL) không được tính vào ràng buộc này, nếu không mọi session device-less sẽ đụng nhau ở một hàng NULL duy nhất.
- **`user_sessions.device_id` nullable + `ON DELETE SET NULL`.** Session US-001 không có thiết bị (AC12 phải refresh/logout được như mọi session khác); `SET NULL` thay vì `CASCADE` để xoá một thiết bị không âm thầm xoá session của người dùng.
- **`login_attempts` không có FK tới `users`.** Bộ đếm phải hoạt động với số điện thoại **chưa có tài khoản** — có FK nghĩa là chỉ số đã đăng ký mới bị khoá, và như vậy chính cái khoá lại tiết lộ số nào đã đăng ký (AC7).
- **`login_attempts.phone_number` là PRIMARY KEY** (không phải `id` surrogate) — mọi truy vấn đều theo số điện thoại, và PK chính là target của `ON CONFLICT (phone_number) DO UPDATE` trong `RecordLoginFailure`.
- **`user_sessions.previous_refresh_token_hash` + `idx_user_sessions_previous_refresh_hash`** — lưu hash của refresh token **liền trước** để phát hiện reuse (AC11). Có index vì `FindSessionByRefreshHash` tra theo **cả hai** cột; không index cột này thì mỗi lần refresh là một seq scan.

- **`otp_send_limits` tách khỏi `otp_codes`** (US-001 v5.0). Bộ đếm gửi phải sống lâu hơn hàng OTP (TTL 90s), và `locked_until` ở đây là **khoá-gửi** — chặn `request-otp`/`resend-otp` nhưng **không** chặn `verify-otp`. Khoá-nhập-sai vẫn ở `otp_codes.locked_until` và **có** chặn verify. Gộp hai khoá làm một sẽ khiến OTP hợp lệ cuối cùng vừa gửi không nhập được.
- **`otp_send_limits UNIQUE (phone_number, purpose)`** — target của `INSERT ... ON CONFLICT (phone_number, purpose) DO NOTHING` trong `applySendLimits`, đồng thời là index tra cứu. Cùng tập `purpose` với `chk_otp_codes_purpose` (`chk_otp_send_limits_purpose`).
- **`otp_codes.resend_count` đã bị xoá ở `0003`.** Trần gửi không còn carry-forward trên hàng OTP — đó chính là cơ chế gây bug "chặn vĩnh viễn". Đừng thêm lại cột này.

## Quan hệ với docs khác
- `docs/specs/features/onboarding/design.md` — flow và contract mà schema này phục vụ.
- `docs/specs/features/onboarding/requirements.md` — SQL trong đó dùng `ALTER TABLE users ADD COLUMN`, giả định bảng `users` đã tồn tại từ feature trước. File ở đây **thay thế** giả định đó bằng một `CREATE TABLE users` đầy đủ.
- `services/appService/domain/` — struct map từng bảng.
- `services/appService/repository/` — SQL thật, và là nơi tên constraint ở trên được tham chiếu.
