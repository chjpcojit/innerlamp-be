# Database CLI — truy cập Docker & xem dữ liệu Postgres

Hướng dẫn thao tác với Postgres của local stack (xem [`local-stack.md`](./local-stack.md)) bằng CLI: mở shell, xem schema, truy vấn dữ liệu, reset để test lại flow, dump/restore.

> Postgres chạy trong container; `psql` **có sẵn trong image `postgres:16-alpine`**, nên không cần cài PostgreSQL client trên máy host. Mọi lệnh dưới đây chạy từ **repo root**.

---

## 1. Mở shell `psql`

```bash
docker compose exec postgres psql -U postgres -d innerlamp
```

Hoặc dùng Make: `make psql` (tự đảm bảo Postgres + migrations đã chạy).

Trong shell:

```
\dt              -- liệt kê bảng
\d users         -- mô tả cấu trúc + index + FK của một bảng
\d+ user_sessions -- kèm storage/comment
\x auto          -- bật/tắt chế độ hiển thị dọc cho hàng nhiều cột
\du              -- liệt kê role
\q               -- thoát
```

---

## 2. Chạy lệnh một lần (không vào shell)

```bash
docker compose exec postgres psql -U postgres -d innerlamp -c '\dt'
docker compose exec postgres psql -U postgres -d innerlamp -c 'SELECT count(*) FROM users;'
```

- Thêm `-T` khi output bị pipe/redirect hoặc chạy trong Make/CI (tắt cấp phát TTY để không lẫn ký tự điều khiển):
  ```bash
  docker compose exec -T postgres psql -U postgres -d innerlamp -c 'SELECT 1;'
  ```
- Nhiều lệnh trong một lần gọi: lặp `-c`:
  ```bash
  docker compose exec -T postgres psql -U postgres -d innerlamp \
    -c 'SELECT count(*) FROM users;' \
    -c 'SELECT count(*) FROM user_sessions;'
  ```

### `docker compose exec` vs `docker exec`

| Cách | Khi nào dùng |
|---|---|
| `docker compose exec postgres psql ...` | **Ưu tiên.** Dùng tên service, không phụ thuộc tên container. Container phải đang chạy. |
| `docker exec -it innerlamp-postgres-1 psql ...` | Khi cần gọi thẳng container (tên do compose sinh, có thể khác nếu bạn đổi project name). Cách này cũng ổn nhưng cứng hơn. |
| `docker compose run --rm migrate` | Chạy một **container mới** theo định nghĩa service rồi xoá. Dùng cho tác vụ one-shot (migrate), không dùng cho psql vì dễ nhầm container. |

Kiểm tra container nào đang chạy:

```bash
docker compose ps
docker compose ps -a          # kể cả container đã thoát (ví dụ migrate)
```

---

## 3. Schema: bảng nào ở đâu

Schema do `migrations/0001`–`0003` tạo. Đây là 7 bảng và ý nghĩa ngắn gọn — chi tiết bất biến ở [`migrations/README.md`](../../migrations/README.md).

| Bảng | Khoá | Ghi chú |
|---|---|---|
| `users` | `id` BIGSERIAL | Tài khoản **đã verify**. `email`/`phone_number` UNIQUE inline (constraint `users_email_key` / `users_phone_number_key`). `email_verified_at` NULL vì verify là số điện thoại. |
| `pending_registration` | `id` | Payload giữa bước 1 và 2. `phone_number` UNIQUE (target `ON CONFLICT`), `email` **không** unique. Hết hạn thì bỏ. |
| `otp_codes` | `id` | OTP theo `purpose` (`register`/`login`/`reset_password`/`verify_email`). Lưu `otp_hash` (bcrypt), `attempts`, `locked_until`, `expires_at`, `used_at`. |
| `otp_send_limits` | `id` | Bộ đếm **gửi** theo `(phone_number, purpose)`. `send_count`, `last_sent_at`, `locked_until`. Tách khỏi `otp_codes` từ v5.0. |
| `user_sessions` | `id` | Session. `refresh_token_hash` là SHA-256 hex (**không** phải token thô). `device_id` nullable, `previous_refresh_token_hash` để phát hiện reuse. |
| `user_devices` | `id` | Một hàng / `(user_id, device_id)`. `device_id` là fingerprint server-side (SHA-256), không phải client gửi. |
| `login_attempts` | `phone_number` (PK) | Bộ đếm sai mật khẩu. **Không** FK tới `users` (để khoá được cả số chưa đăng ký). |

Ghi chú về dữ liệu nhạy cảm:

- `password_hash` / `otp_hash` là **bcrypt**; `refresh_token_hash` là **SHA-256 hex**. Không bao giờ thấy plaintext trong DB — điều này đúng thiết kế, không phải lỗi.
- Access token **không** lưu ở DB (ngắn hạn, verify bằng chữ ký).

---

## 4. Truy vấn mẫu

Các lệnh dưới đây tương ứng target Make (`make db-accounts`, `make db-sessions`, `make db-otp`).

### Bảng & số hàng

```bash
docker compose exec -T postgres psql -U postgres -d innerlamp -c '\dt'
docker compose exec -T postgres psql -U postgres -d innerlamp -c \
  "SELECT relname AS table_name, n_live_tup AS row_estimate
     FROM pg_stat_user_tables ORDER BY relname;"
```

### Tài khoản đã đăng ký

```bash
docker compose exec -T postgres psql -U postgres -d innerlamp -c \
  "SELECT id, phone_number, email, dharma_name, created_at
     FROM users ORDER BY id;"
```

### Đăng ký dở (chưa verify)

```bash
docker compose exec -T postgres psql -U postgres -d innerlamp -c \
  "SELECT phone_number, email, expires_at, created_at
     FROM pending_registration ORDER BY created_at DESC;"
```

### OTP đang tồn tại + giới hạn gửi

```bash
docker compose exec -T postgres psql -U postgres -d innerlamp -c \
  "SELECT id, phone_number, purpose, attempts, expires_at, used_at, locked_until
     FROM otp_codes ORDER BY id DESC LIMIT 10;"

docker compose exec -T postgres psql -U postgres -d innerlamp -c \
  "SELECT phone_number, purpose, send_count, last_sent_at, locked_until
     FROM otp_send_limits ORDER BY phone_number;"
```

### Session (đang hoạt động / đã thu hồi)

```bash
docker compose exec -T postgres psql -U postgres -d innerlamp -c \
  "SELECT id, user_id, device_id, is_active, is_revoked, revoked_reason,
          last_activity_at, expires_at
     FROM user_sessions ORDER BY id DESC LIMIT 20;"
```

### Bộ đếm sai mật khẩu / khoá

```bash
docker compose exec -T postgres psql -U postgres -d innerlamp -c \
  "SELECT phone_number, failed_count, locked_until, last_failed_at
     FROM login_attempts ORDER BY last_failed_at DESC;"
```

---

## 5. Reset state của một số điện thoại để test lại flow

Trong lúc test register/login, một số có thể bị vướng: tài khoản đã tồn tại, đang cooldown gửi OTP, hoặc bị khoá do sai quá nhiều lần. Có hai cách.

### Cách nhanh — target Make

```bash
make db-reset-phone PHONE=+84901234567
```

Target này xoá `users` (kéo theo `user_sessions` và `user_devices` nhờ `ON DELETE CASCADE`), rồi xoá `pending_registration`, `otp_codes`, `otp_send_limits`, `login_attempts` — bốn bảng keyed theo `phone_number` chứ không theo `user_id`, nên phải xoá riêng.

### Tương đương bằng SQL

```bash
docker compose exec -T postgres psql -U postgres -d innerlamp -v ON_ERROR_STOP=1 \
  -c "DELETE FROM users WHERE phone_number = '+84901234567';" \
  -c "DELETE FROM pending_registration WHERE phone_number = '+84901234567';" \
  -c "DELETE FROM otp_codes WHERE phone_number = '+84901234567';" \
  -c "DELETE FROM otp_send_limits WHERE phone_number = '+84901234567';" \
  -c "DELETE FROM login_attempts WHERE phone_number = '+84901234567';"
```

`ON DELETE CASCADE` chỉ tồn tại ở `user_sessions.user_id` và `user_devices.user_id`; `user_sessions.device_id` là `ON DELETE SET NULL` (xoá device không xoá session). Vì vậy xoá `users` là đủ cho session/device, nhưng **không** đủ cho OTP/limit/attempts.

### Xoá sạch toàn bộ DB

```bash
docker compose down -v && docker compose up -d     # hoặc: make reset
```

`down -v` xoá luôn volume Loki/Grafana, không chỉ Postgres.

---

## 6. Dump & restore

```bash
# Dump ra file
make db-dump > backup.sql
# hoặc: docker compose exec -T postgres pg_dump -U postgres -d innerlamp > backup.sql

# Restore từ file (DB phải đang chạy)
cat backup.sql | docker compose exec -T postgres psql -U postgres -d innerlamp
```

---

## 7. Xem log của Postgres

```bash
docker compose logs -f postgres          # theo dõi liên tục
docker compose logs --tail=100 postgres  # 100 dòng cuối
```

Lưu ý: đây là log của **container Postgres**. Log của **appService** đi theo đường khác (`go run` → file → Alloy → Loki), xem [`local-stack.md`](./local-stack.md) §1 và §2.

---

## 8. Lưu ý

- **`DATABASE_URL` và `scripts/migrate.sh` không áp dụng cho bộ migration hiện tại.** Script đó dùng golang-migrate, chỉ nhận cặp `.up.sql`/`.down.sql`, nên **không nhìn thấy** các file bare `000N_*.sql` — chạy nó sẽ tìm thấy 0 file và "pass rỗng". Đường apply đúng là service `migrate` trong compose (`make migrate`) hoặc `psql -f`. Chi tiết ở `migrations/README.md`.
- **Dữ liệu nằm trong volume `postgres-data`**, không nằm trong container. `docker compose down` giữ nguyên; `docker compose down -v` (hay `make reset`) mới xoá.
- **Mật khẩu/secret trong DB là giá trị dev** (`postgres`), khớp `dev_secrets` trong `config.dev.yaml`. Không dùng cho môi trường thật.
