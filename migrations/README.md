# Migrations — innerlamp

## Mục đích
Schema Postgres cho appService. Mỗi file là một bước thay đổi schema, đánh số tăng dần, áp theo thứ tự.

Hiện có **hai mươi mốt** file (`0001`–`0021`). `0001`–`0003` là phần identity của appService; `0004`–`0014` do các feature sau thêm vào và một số (`0006`, `0008`–`0010`) thuộc `backofficeService`, dùng chung cùng một Postgres:

| File | Feature | Nội dung |
|---|---|---|
| `0001_register_onboarding.sql` | US-001 | `users`, `otp_codes`, `pending_registration`, `user_sessions` |
| `0002_login_devices_and_rate_limit.sql` | US-002 | `user_devices`, `login_attempts`, `user_sessions.device_id` + `previous_refresh_token_hash` |
| `0003_otp_send_limits.sql` | US-001 v5.0 | `otp_send_limits` + drop `otp_codes.resend_count` |
| `0004_attestation.sql` | attestation | `attestation_challenges`, `attestation_keys` |
| `0005_temples.sql` | temple | `temples` + `idx_temples_ranking` |
| `0006_admins.sql` | admin | `admins` |
| `0007_deities.sql` | deity | `deities` + hai index |
| `0008_admin_must_change_password.sql` | admin | `ALTER TABLE admins` (cờ buộc đổi mật khẩu) |
| `0009_admin_tokens_valid_after.sql` | admin | `ALTER TABLE admins` (mốc thu hồi phiên `tokens_valid_after`) |
| `0010_admin_temp_password_expiry.sql` | admin | `ALTER TABLE admins` (hạn mật khẩu tạm) |
| `0011_temple_content.sql` | temple | `ALTER TABLE temples` thêm `introduction`, `architecture_heritage`, `visiting_notes` |
| `0012_articles.sql` | article | `articles` + `idx_articles_order (sort_order, id)` |
| `0016_article_thumbnail.sql` | article | `articles.thumbnail_url TEXT` (NULL) |
| `0013_sutras.sql` | sutra | `sutras`, `sutra_translations`, `sutra_lines`, `sutra_deities` + `idx_sutras_order`, `idx_sutras_tradition_kind`, `idx_sutras_purposes` (GIN), `idx_sutra_deities_deity` |
| `0017_gallery_images.sql` | gallery | `gallery_images` (CHECK `type`) + `idx_gallery_images_order (sort_order, id)` + `idx_gallery_images_type_order (type, sort_order, id)` |
| `0014_dharma_talks.sql` | dharmatalk | `dharma_talks` (+ CHECK nhất quán `media_type` ↔ `youtube_id`/`audio_url`) + `idx_dharma_talks_order (sort_order, id)` |
| `0018_memorials.sql` | memorial | `memorials` (FK `user_id` → `users` ON DELETE CASCADE; CHECK `relationship`, `death_calendar`, hai dạng ngày mất loại trừ nhau, reminder cùng-có-hoặc-cùng-NULL) + `idx_memorials_user (user_id, created_at DESC, id DESC)` |
| `0019_sutra_audio_search.sql` | sutra | `CREATE EXTENSION unaccent` (cho `GET /v1/sutras?q=`), `sutras.audio_url TEXT NOT NULL DEFAULT ''`, điền `duration_minutes` còn NULL (số dòng / 4) |
| `0020_sutra_reference_license_summary.sql` | sutra | `ALTER TABLE sutras` thêm `original_title`, `author`, `han_translator`, `han_translation_year` (CHECK ≥ 1, NULL = không rõ), `reference_note`, `summary`; `ALTER TABLE sutra_translations` thêm `license_url`, `license_note`. Chỉ thêm cột — không sửa dữ liệu đã có (seed không chạy lại khi bảng có dòng) |
| `0021_relative_image_urls.sql` | content | `UPDATE` `temples`, `deities`, `gallery_images`, `articles`: đổi `image_url`/`thumbnail_url` dạng `http(s)://<host>/static/images/...` thành path tương đối `/static/images/...` (API ghép host khi trả response). Idempotent; bỏ qua hotlink ngoài `/static/images/` |

`0002` là **tiến hoá** của `0001`, không phải bootstrap: nó `ALTER TABLE user_sessions ADD COLUMN` nên **phải chạy sau** `0001`. Chạy `0002` một mình trên DB trống sẽ lỗi `relation "user_sessions" does not exist`. Mọi file đều **idempotent**: mọi câu lệnh là `CREATE TABLE IF NOT EXISTS` / `ADD COLUMN IF NOT EXISTS` / `CREATE INDEX IF NOT EXISTS` / `DROP COLUMN IF EXISTS`, nên apply lại chỉ sinh `NOTICE ... skipping` chứ không lỗi (đã kiểm chứng: chạy `migrate` lần hai exit 0). Service `migrate` trong `docker-compose.yml` dựa vào tính chất này — xem `docs/development/local-stack.md`.

`0003` chạy sau `0001` (nó `ALTER TABLE otp_codes DROP COLUMN`), và **idempotent**: `CREATE TABLE IF NOT EXISTS` + `DROP COLUMN IF EXISTS`, nên apply lại không lỗi.

Các file sau `0003` cũng tuân theo quy ước idempotent (`IF NOT EXISTS`). `0008`–`0010` và `0011` là `ALTER TABLE` nên phải chạy sau `0006` và `0005` tương ứng.

Thứ tự apply (ba file đầu; trong local stack `make migrate` áp **mọi** file `*.sql` theo thứ tự tên nên không cần gõ tay):

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
- **`articles.id` là slug từ dữ liệu nguồn, không phải số tự tăng** (`0012`). Client cache id nên id phải sống qua mọi lần nạp lại; đổi sang surrogate key sẽ làm hỏng id đã cache.
- **`articles.sort_order` là vị trí trong `assets/data/buddhist_articles.json`** và cùng `id` tạo thứ tự cố định (`ORDER BY sort_order, id`, khớp `idx_articles_order`). `top` và phân trang đều theo thứ tự này vì nguồn không có ngày hay độ phổ biến đáng tin; `repository/postgres_article_repository.go` dùng cùng một hằng `ORDER BY` cho mọi truy vấn nên không lệch nhau.
- **`articles.published_date` và `articles.image_url` cho phép `NULL`** (`0012`). Nguồn không có giá trị cho phần lớn dòng và API trả JSON `null`; đừng đổi sang `NOT NULL DEFAULT ''` hay ngày giả.
- **Bảng `articles` phải có trước khi appService khởi động.** Service tự nạp 30 bài từ JSON nhúng trong binary khi bảng rỗng (`ArticleUsecase.SeedIfEmpty`); nếu `0012` chưa apply thì bước đó lỗi và service không lên (`failed to seed articles`). Lần khởi động sau không seed lại, nhưng xoá **hết** dòng rồi khởi động lại sẽ nạp lại đủ. Migration chỉ tạo schema, không chứa dữ liệu bài viết.
- **`dharma_talks.id` là slug từ dữ liệu nguồn** (`0014`), cùng lý do với `articles.id`. `sort_order` là vị trí trong `assets/data/dharma_talks.json` (`ORDER BY sort_order, id`, khớp `idx_dharma_talks_order`).
- **`dharma_talks.needs_review = true` là bài được lưu nhưng không bao giờ được trả ra** (`0014`): mọi route đọc lọc `needs_review = false`. Bước seed lại đếm **cả** bài bị ẩn, nên bảng mà mọi bài đều bị cờ không bị coi là rỗng và không bị nạp lại. Gỡ cờ bằng SQL tay sau khi duyệt.
- **`dharma_talks` ràng buộc phương tiện bằng CHECK** (`0014`): `youtube` ⇒ có `youtube_id`, không `audio_url`; `audio` ⇒ ngược lại; `text` ⇒ cả hai NULL. Ghi tay sai sẽ bị chặn. `duration_minutes`, `published_date`, `season`, `follow_up_meditation` cho phép `NULL` và API trả JSON `null`; đừng đổi sang `0`/chuỗi rỗng.
- **Bảng `dharma_talks` phải có trước khi appService khởi động**, như `articles`: nếu `0014` chưa apply thì seed lỗi và service không lên (`failed to seed dharma talks`).
- **`sutras.id` là slug từ dữ liệu nguồn** (`0013`, vd. `tam-kinh`), cùng lý do với `articles.id`: client cache id nên phải sống qua mọi lần nạp lại. `sutras.sort_order` là vị trí trong `assets/data/sutras/index.json` (`ORDER BY sort_order, id`, khớp `idx_sutras_order`).
- **`sutras.reference` là `TEXT`, không phải `VARCHAR`** (`0013`). Nó giữ nguyên văn trích dẫn của nguồn; `lang-nghiem` có `reference` dài 135 ký tự và nạp dữ liệu thật báo `value too long` với `VARCHAR(128)`.
- **`sutras.duration_minutes` cho phép `NULL`** (`0013`): `NULL` là chưa ước tính (hiện là 10 văn bản Kim Cang Thừa không có tiếng Việt), API trả JSON `null`. Đừng đổi sang `NOT NULL DEFAULT 0` — `0` sẽ bị đọc là "0 phút".
- **`sutra_translations (sutra_id, key)` là UNIQUE** (`uq_sutra_translations_key`): API chọn bản dịch theo `key`, vì `(lang, variant)` không đủ phân biệt (`tam-kinh` có hai bản `vi` + `meaning`). `sutra_translations.id` đổi mỗi khi một sutra được nạp lại nên **không bao giờ** lộ ra API.
- **`sutra_lines` PRIMARY KEY `(translation_id, n)`** và `n` chạy 1..`line_count` không hở trong một bản dịch: API phân trang dòng bằng khoảng `n > $from AND n <= $to` trên chính khoá này (không `OFFSET`, không `count(*)`). `line_count` được lưu sẵn trong `sutra_translations` để trả `total` mà không phải đếm; hai con số này phải luôn bằng nhau (test repository kiểm trên dữ liệu thật).
- **`sutra_deities.deity_id` là FK tới `deities` và KHÔNG `ON DELETE CASCADE`** (`0013`), còn `sutra_id` thì có. Hệ quả vận hành: `deities` phải được seed (`go run ./services/appService/cmd/seeddeities`) **trước** lần khởi động đầu của appService, nếu không service dừng với `failed to seed sutras: ... (run cmd/seeddeities first)`; và `TRUNCATE deities ... CASCADE` xoá luôn mọi liên kết `sutra_deities` — vì `sutras` vẫn còn dòng nên lần khởi động sau **không** nạp bù, phải seed lại cả hai.
- **Các bảng `sutra*` phải có trước khi appService khởi động.** Service tự nạp 153 văn bản (391 bản dịch, 64.009 dòng, 151 liên kết) từ JSON nhúng trong binary khi `sutras` rỗng (`SutraUsecase.SeedIfEmpty`, trong **một** transaction nên lỗi thì không để lại gì); nếu `0013` chưa apply thì bước đó lỗi và service không lên (`failed to seed sutras`). `go run ./services/appService/cmd/seedsutras` làm đúng việc đó bằng tay và cũng chỉ nạp khi bảng rỗng. Migration chỉ tạo schema, không chứa dữ liệu kinh.
- **`otp_codes.resend_count` đã bị xoá ở `0003`.** Trần gửi không còn carry-forward trên hàng OTP — đó chính là cơ chế gây bug "chặn vĩnh viễn". Đừng thêm lại cột này.

## Quan hệ với docs khác
- `docs/specs/features/onboarding/design.md` — flow và contract mà schema này phục vụ.
- `docs/specs/features/onboarding/requirements.md` — SQL trong đó dùng `ALTER TABLE users ADD COLUMN`, giả định bảng `users` đã tồn tại từ feature trước. File ở đây **thay thế** giả định đó bằng một `CREATE TABLE users` đầy đủ.
- `services/appService/domain/` — struct map từng bảng.
- `services/appService/repository/` — SQL thật, và là nơi tên constraint ở trên được tham chiếu.
- `docs/specs/features/article/` — spec của tính năng dùng `0012` (seed lần đầu, thứ tự, hình dạng `null`).
- `docs/specs/features/dharmatalk/` — spec của tính năng dùng `0014` (ba loại phương tiện, `needs_review`, seed lần đầu).
- `docs/specs/features/sutra/` — spec của tính năng dùng `0013` (seed lần đầu, phân trang dòng, liên kết với `deities`).
