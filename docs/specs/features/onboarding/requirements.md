# User Stories — Tâm Đăng (InnerLamp)

Version: 5.2
Last updated: 2026-09-22
Status: in progress — US-001 đã được đặc tả đầy đủ ở `design.md`/`tasks.md` (luồng 2 bước, `pending_registration`, bảng OTP dùng chung `otp_codes` có `purpose`, tham số OTP, SMS log-only, gate email chưa verify); US-003/004 còn ở mức user story; **US-002 đã chuyển sang `docs/specs/features/login/`** (v4.0). (v5.2: sửa hai chỗ tồn đọng ở §Ghi chú kỹ thuật — thêm endpoint thứ ba `POST /v1/auth/resend-otp`; refresh token 30 ngày → **7 ngày** (`168h`), khớp `config.dev.yaml`. v5.1: **đổi AC6, thêm AC15** — tách `PendingRegistrationTTL` khỏi OTP TTL (24h thay 90s), `request-otp` ghi đè payload của chính SĐT để làm lại, `resend-otp` cấp OTP mới khi OTP cũ hết hạn; v5.0: **đổi AC5 và AC12** — thay trần resend chặn vĩnh viễn bằng trần số lần gửi theo SĐT + khoá gửi tạm 30 phút, thêm bảng `otp_send_limits`; v4.5: chỉ bump theo design/tasks — fix bug pending hết hạn chặn đăng ký lại, không đổi AC; v4.3: chỉ bump theo design/tasks — sửa mô tả envelope, không đổi prose; v4.2: chỉ bump theo Phase 7 — không đổi prose, `sid` là chi tiết token; v4.1: chỉ bump theo design/tasks, không đổi prose; v4.0: US-002 bị xoá khỏi file này và chuyển sang `login/` — đổi scope nên bump major; prose của US-001/003/004 không đổi ngoài các con trỏ tham chiếu chéo tới `login/`.)

Tài liệu tổng hợp user story theo từng tính năng, viết theo thứ tự ưu tiên xây dựng (bắt đầu từ module `identity`).

---

## US-001: Đăng ký tài khoản

**Module:** `identity`
**Priority:** P0 (bắt buộc, chặn mọi tính năng cần user context)

### User Story

> Là một **người dùng mới**,
> Tôi muốn **đăng ký tài khoản bằng số điện thoại (xác thực OTP qua SMS), email và mật khẩu**,
> Để **có thể lưu và đồng bộ hành trình tu tập của mình qua nhiều thiết bị**.

### Các trường thông tin

| Trường | Bắt buộc? | Ghi chú |
|---|---|---|
| Số điện thoại | Bắt buộc | Chuẩn hóa định dạng E.164 (`+84...`), unique. **Phải verify OTP qua SMS** — đây là định danh chính ở MVP. Đã verify thì lưu thẳng vào `users.phone_number` |
| Mật khẩu | Bắt buộc | ≥ 8 ký tự |
| Email | Bắt buộc nhập | Unique toàn hệ thống. Client gửi lên ở bước 1, lưu tạm ở `pending_registration`; sau khi verify OTP mới copy sang `users.email`. Vì chưa verify nên `users.email_verified_at` = `NULL` |
| Pháp danh (dharma name) | Tùy chọn | Tên đạo, nếu đã quy y |
| Nơi quy y / xuất gia | Tùy chọn | Tên chùa/tổ đình nơi quy y hoặc xuất gia |

> Số điện thoại **bắt buộc**; email **bắt buộc nhập**. Thiết kế bảng: `pending_registration` có cột `email`; bảng `users` cũng có cột `email` nhưng thêm cột `email_verified_at`. Client gửi email lên ở bước 1, hệ thống lưu tạm vào `pending_registration`; **sau khi verify OTP xong** mới copy sang `users.email`. Tại thời điểm copy email vẫn **chưa được verify**, nên `users.email_verified_at` để `NULL`. Không có bảng `pending_email` thường trú — email chưa verify nằm chung trong `pending_registration` với phần payload còn lại. Verify email (gửi mail xác thực + set `email_verified_at`) **không nằm trong phạm vi US-001** (xem [Out of scope](#out-of-scope-giai-đoạn-mvp)).

### Luồng đăng ký (2 bước)

Thứ tự: **verify OTP trước, tạo account sau** — không có tài khoản nào ở trạng thái "chưa verify" tồn tại trong `users`.

1. Người dùng nhập số điện thoại + email + mật khẩu (+ pháp danh / nơi quy y nếu có) → hệ thống lưu tạm toàn bộ payload vào bảng `pending_registration`, **gửi OTP qua SMS**, chưa ghi gì vào `users`.
2. Người dùng nhập OTP → hệ thống xác thực, **rồi mới** tạo hàng `users` (lấy dữ liệu từ `pending_registration`) và cấp token; hàng tạm bị xoá sau khi tạo xong.

Dữ liệu đăng ký tạm (kể cả password **đã hash**) giữ ở **server**, trong bảng `pending_registration` — không bắt client giữ plaintext password rồi gửi lại ở bước 2. Hệ quả: request bước 2 chỉ cần `phone_number` + `otp_code`, và password không đi qua mạng lần thứ hai.

### Acceptance Criteria

1. **Given** tôi chưa có tài khoản, **when** tôi nhập số điện thoại hợp lệ, email hợp lệ, mật khẩu ≥ 8 ký tự và nhấn "Đăng ký", **then** hệ thống gửi một OTP qua SMS tới số điện thoại đó và **chưa** tạo tài khoản.
2. **Given** tôi đã nhận được OTP, **when** tôi nhập đúng OTP còn hiệu lực, **then** tài khoản được tạo với `phone_number` = số điện thoại vừa verify, `email` = email tôi đã nhập, và `email_verified_at` = `NULL`, đồng thời tôi được đăng nhập ngay **bằng số điện thoại vừa verify** (nhận về access token + refresh token). Email chưa verify **không** dùng được để đăng nhập — xem AC13.
3. **Given** email tôi nhập hợp lệ nhưng chưa verify, **when** tài khoản được tạo, **then** email đó được ghi vào `users.email` và `users.email_verified_at` để `NULL`; không hàng nào của dữ liệu đăng ký tạm được giữ lại sau khi tạo account.
4. **Given** OTP tôi nhập sai, đã hết hạn (quá 90 giây kể từ lúc gửi), hoặc đã dùng một lần rồi, **when** tôi xác nhận, **then** hệ thống báo lỗi và không tạo tài khoản.
5. **Given** tôi vừa yêu cầu OTP lần đầu thành công, **when** tôi yêu cầu gửi lại (hoặc yêu cầu OTP mới) trong vòng 60 giây, **then** hệ thống từ chối và yêu cầu tôi chờ. Qua mốc 60 giây đó, mỗi lần **yêu cầu OTP mới hoặc gửi lại** đều tính là một lượt gửi; hệ thống cho tổng cộng **4 lượt gửi** cho một số điện thoại trong một lượt đăng ký (1 lần đầu + 3 lượt tính), mỗi lượt tạo OTP mới và vô hiệu hóa OTP cũ. Khi lượt gửi thứ 4 được thực hiện, hệ thống **khóa việc gửi cho số điện thoại đó trong 30 phút**; lượt thứ 5 trở đi bị từ chối. Hết 30 phút, bộ đếm đặt lại và người dùng có thể bắt đầu lại. Việc khóa gửi **không** ngăn người dùng nhập OTP hợp lệ cuối cùng trong thời hạn 90 giây của nó.
6. **Given** số điện thoại hoặc email tôi nhập đã tồn tại trong hệ thống, **when** tôi nhấn "Đăng ký", **then** hệ thống báo lỗi tương ứng ("Số điện thoại đã được sử dụng" / "Email đã được sử dụng") và không tạo tài khoản trùng. Số điện thoại chỉ bị coi là "đã dùng" khi đã có tài khoản; một lượt đăng ký dở của **chính** số đó không chặn — người dùng được bắt đầu lại (AC15). Việc kiểm tra trùng email phải truy vấn **cả hai** nguồn: `users.email` (email của tài khoản đã tạo) **và** `pending_registration.email` đang chờ verify của một **số điện thoại khác** — vì email chưa verify **chưa** nằm ở `users`.
7. **Given** mật khẩu tôi nhập ngắn hơn 8 ký tự, **when** tôi nhấn "Đăng ký", **then** hệ thống báo lỗi ngay tại form, không gửi request lên server.
8. **Given** số điện thoại hoặc email tôi nhập sai định dạng, **when** tôi nhấn "Đăng ký", **then** hệ thống báo lỗi ngay tại form.
9. **Given** tôi để trống trường Pháp danh và/hoặc Nơi quy y/xuất gia, **when** tôi đăng ký, **then** tài khoản vẫn được tạo bình thường — đây là các trường tùy chọn.
10. **Given** tôi đăng ký thành công, **when** hệ thống tạo tài khoản, **then** mật khẩu phải được hash (không lưu plaintext) trước khi lưu vào database.
11. **Given** tôi đăng ký thành công, **when** tôi mở lại app sau khi tắt hoàn toàn, **then** tôi vẫn ở trạng thái đã đăng nhập nhờ access token còn hạn (hoặc refresh token tự động cấp access token mới).
12. **Given** tôi đã nhập sai OTP 3 lần, **when** tôi thử lần thứ 4, **then** hệ thống khóa xác thực OTP cho số điện thoại đó trong 30 phút, vô hiệu hóa OTP hiện tại, và buộc tôi chờ hết thời gian khóa rồi làm lại từ bước 1 — chống brute-force OTP.
13. **Given** email của tôi có `email_verified_at IS NULL`, **when** tôi cố đăng nhập bằng email đó, **then** hệ thống từ chối và báo lỗi "Email chưa được xác thực" — email chưa verify **không** phải phương thức đăng nhập hợp lệ. Chỉ số điện thoại đã verify OTP ở bước 2 mới đăng nhập được.
14. **Given** email của tôi có `email_verified_at IS NULL`, **when** tôi cố dùng bất kỳ tính năng nào gửi thư tới email đó (quên mật khẩu, đổi email, nhận thông báo qua email), **then** hệ thống từ chối và hướng tôi sang luồng xác thực email trước — **không** gửi mail tới một địa chỉ chưa được xác thực.
15. **Given** tôi có một lượt đăng ký dở (payload tạm ở `pending_registration`), **when** tôi bắt đầu lại — gửi lại yêu cầu đăng ký cùng số điện thoại (`request-otp`, ghi đè email/password) hoặc bấm gửi lại OTP (`resend-otp`, giữ nguyên payload) — **then** hệ thống cho tiếp tục khi số điện thoại đó **không** đang trong khoá-gửi 30 phút (AC5); payload tạm sống **24 giờ**, và `resend-otp` vẫn cấp OTP mới kể cả khi OTP trước đã hết hạn 90 giây, miễn payload chưa hết hạn.

### Gate email chưa verify

Luật chung áp cho mọi tính năng hiện tại và sau này: **`users.email_verified_at IS NULL` ⇒ email đó bị coi là chưa sở hữu hợp lệ** — không dùng được làm định danh đăng nhập, không dùng được làm đích nhận thư, không dùng được để khôi phục tài khoản. Lý do: ở bước 2, email được copy sang `users` mà chưa hề có bằng chứng người đăng ký thực sự sở hữu hộp thư đó.

Hệ quả trực tiếp lên các US khác (chưa sửa trong lần này, ghi lại để lần sau khớp):

| US | Ảnh hưởng |
|---|---|
| US-002 (đăng nhập — nay ở [`../login/`](../login/requirements.md)) | Đăng nhập bằng email chỉ hợp lệ khi `email_verified_at IS NOT NULL`. Định danh chính là số điện thoại. **Đã áp dụng:** login spec chỉ hỗ trợ SĐT + mật khẩu, email login ngoài scope. |
| US-003 (quên mật khẩu) | Gửi link reset qua email **chỉ** khi email đã verify. User chỉ có SĐT đã verify mà email chưa verify thì phải có đường khôi phục qua SMS OTP. |
| US-004 (hồ sơ) | Màn hình hồ sơ phải hiển thị rõ trạng thái email (chưa xác thực / đã xác thực) và có nút "Xác thực email". |
| Tính năng tương lai dùng email | Phải kiểm tra gate trước khi gửi thư hoặc dùng email làm định danh. |

Ghi chú: chính tính năng **xác thực email** (gửi mail xác thực + set `email_verified_at`) và luồng **đổi email** nằm ngoài phạm vi US-001 — đây mới chỉ là luật gate.

_Không còn mục "Not confirmed" nào cho US-001._

### Out of scope (giai đoạn MVP)

- **Verify email sau khi đăng ký** — US-001 copy email từ `pending_registration` sang `users.email` nhưng để `email_verified_at` = `NULL`, chưa gửi mail xác thực và chưa có luồng set `email_verified_at`. Để lại user story riêng, làm ngay sau US này (cùng dịp với US-003 vì cả hai đều cần dịch vụ gửi email).
- Đăng ký qua social login (Google/Apple) — để lại cho giai đoạn sau
- Quên mật khẩu / reset password — cần user story riêng, làm ngay sau US này vì là rủi ro UX nếu thiếu

### Ghi chú kỹ thuật liên quan

- Endpoint (2 bước, khớp [Luồng đăng ký](#luồng-đăng-ký-2-bước)):
  ```
  POST /v1/auth/register/request-otp   { phone_number, email, password, dharma_name?, ordination_place? }
  POST /v1/auth/register/verify-otp    { phone_number, otp_code }
  POST /v1/auth/resend-otp             { phone_number, purpose }   -- dùng chung mọi purpose; US-001 chỉ hiện thực purpose=register
  ```
  Bước 1 **chưa ghi gì** vào `users` — chỉ ghi `pending_registration` + `otp_codes` (purpose = `register`); bước 2 mới tạo hàng `users` + cấp token.
  *(Cả ba path đã có trong `docs/specs/api/openapi.yaml` v3.7.)*
- Schema:
  ```sql
  -- email vẫn nằm thẳng ở users: unique + NOT NULL giữ nguyên như thiết kế cũ.
  -- Chỉ thêm cột đánh dấu trạng thái verify.
  ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMPTZ;
  ALTER TABLE users ADD COLUMN phone_number VARCHAR(20) UNIQUE NOT NULL;
  ALTER TABLE users ADD COLUMN ordination_place VARCHAR(255);
  -- (dharma_name đã có sẵn từ thiết kế trước, giữ nguyên)

  -- OTP dùng chung cho mọi mục đích (register / login / reset_password / verify_email),
  -- phân biệt bằng cột purpose. Bước 1 ghi vào đây với purpose = 'register'.
  CREATE TABLE otp_codes (
      id           BIGSERIAL PRIMARY KEY,
      phone_number VARCHAR(20) NOT NULL,
      purpose      VARCHAR(32) NOT NULL,          -- 'register' | 'login' | 'reset_password' | 'verify_email'
      otp_hash     VARCHAR(255) NOT NULL,
      attempts     INT NOT NULL DEFAULT 0,
      locked_until TIMESTAMPTZ,
      expires_at   TIMESTAMPTZ NOT NULL,
      used_at      TIMESTAMPTZ,
      created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  -- v5.0: bỏ cột resend_count — trần gửi không còn đếm trên hàng OTP nữa mà
  -- nằm ở bảng otp_send_limits bên dưới (bộ đếm không được phụ thuộc hàng OTP
  -- 90 giây, và khoá-gửi phải tách khỏi khoá-nhập-sai để không chặn verify).
  -- Tra cứu luôn theo cặp (phone_number, purpose): một SĐT có thể có OTP
  -- đang hiệu lực cho nhiều mục đích khác nhau cùng lúc.
  CREATE INDEX idx_otp_codes_phone_purpose ON otp_codes(phone_number, purpose);
  -- Ràng buộc giá trị hợp lệ của purpose (chặn typo, không phải enum cứng của Postgres
  -- để thêm mục đích mới chỉ cần sửa constraint, không cần ALTER TYPE).
  ALTER TABLE otp_codes ADD CONSTRAINT chk_otp_codes_purpose
      CHECK (purpose IN ('register', 'login', 'reset_password', 'verify_email'));

  -- v5.0: rate limit theo SĐT cho việc GỬI OTP. Tách khỏi hàng otp_codes vì
  -- (a) bộ đếm phải sống lâu hơn hàng OTP 90 giây — gốc của bug "chặn vĩnh viễn"
  -- trước đây là resend_count carry-forward trên hàng OTP; (b) khoá-gửi không
  -- được chặn việc nhập OTP hợp lệ cuối cùng (khác khoá-nhập-sai ở otp_codes).
  CREATE TABLE otp_send_limits (
      id           BIGSERIAL PRIMARY KEY,
      phone_number VARCHAR(20) NOT NULL,
      purpose      VARCHAR(32) NOT NULL,
      send_count   INT NOT NULL DEFAULT 0,   -- số lượt đã tính (không tính lần đầu)
      last_sent_at TIMESTAMPTZ,              -- mốc cooldown 60s; hết 30 phút thì đặt lại bộ đếm
      locked_until TIMESTAMPTZ,              -- khoá-gửi 30 phút khi chạm trần
      created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
      updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
      UNIQUE (phone_number, purpose)
  );

  -- dữ liệu đăng ký tạm giữa bước 1 và bước 2 (giữ ở server)
  CREATE TABLE pending_registration (
      id               BIGSERIAL PRIMARY KEY,
      phone_number     VARCHAR(20) NOT NULL UNIQUE,
      email            VARCHAR(255) NOT NULL,
      password_hash    VARCHAR(255) NOT NULL,
      dharma_name      VARCHAR(255),
      ordination_place VARCHAR(255),
      expires_at       TIMESTAMPTZ NOT NULL,  -- now() + PendingRegistrationTTL (24h, tách khỏi OTP TTL — v5.1)
      created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  CREATE INDEX idx_pending_registration_phone ON pending_registration(phone_number);
  ```
- **Không có bảng `pending_email` riêng.** Email chưa verify nằm chung trong `pending_registration` cùng phần payload còn lại, rồi mới copy sang `users.email` khi verify OTP xong. Trạng thái "chưa verify" ở `users` chỉ là `email_verified_at IS NULL`.
- **Email xuất hiện ở hai nơi trong khoảng giữa 2 bước**, nên check trùng phải quét cả hai: `users.email` (tài khoản đã tạo — unique constraint giữ nguyên, là chốt chặn cuối) và `pending_registration.email` (lượt đăng ký đang chờ verify).
- **Bảng `pending_registration`:** giữ tạm toàn bộ payload đăng ký ở **server** trong khoảng giữa bước 1 và bước 2. Ở bước 2, hệ thống copy **cả `email` lẫn `password_hash`** sang `users` — email ở đây vẫn **chưa verify**, nên `users.email_verified_at` để `NULL`. Password đã **hash** sẵn từ bước 1, bước 2 chỉ copy chứ không hash lại. Hàng tạm bị xoá ngay sau khi tạo `users` thành công; hàng hết hạn (`expires_at`) dọn định kỳ cùng chỗ với `otp_codes`.
  - Lưu ý: `pending_registration.email` **không** đặt unique — trùng email phải được phát hiện bằng check tường minh khi tạo `users` (và qua unique constraint của `users.email`), tránh chặn nhầm hai người cùng nhập một email chưa verify.
  - **TTL của payload tạm = 24 giờ** (v5.1; trước là 90 giây, bằng đúng OTP TTL). Payload phải sống lâu hơn khoá-gửi 30 phút thì người dùng mới "gửi lại / đăng ký lại" được sau khi khoá hết (AC15). Đánh đổi có ý thức: `password_hash` nằm trên server tới 24h; job dọn hàng hết hạn (đã hoãn ở MVP) xoá định kỳ, còn `expires_at` là chốt chặn cuối. Một lượt đăng ký dở của **chính** số điện thoại đó **không** chặn `request-otp` (ghi đè payload).
- OTP lưu dạng **hash**, không plaintext — cùng pattern với reset token ở US-003.
- **Bảng `otp_codes` dùng chung, phân biệt bằng `purpose`.** Cột `purpose` cho biết OTP phục vụ luồng nào: `register` (US-001), `login` (đăng nhập bằng SĐT + OTP — **ngoài scope** của US-002 ở `login/`, giữ chỗ cho sau), `reset_password` (khôi phục mật khẩu qua SMS, dự kiến ở US-003 bản sửa), `verify_email` (xác thực email sau đăng ký). Lý do gộp một bảng thay vì bảng riêng cho từng luồng: cùng một cơ chế sinh/lưu/hash/đếm `attempts`/`expires_at`/khóa tạm — tách bảng sẽ nhân bản logic verify. Tra cứu luôn theo cặp `(phone_number, purpose)` để một SĐT có thể có OTP đang hiệu lực cho nhiều mục đích cùng lúc mà không đè lên nhau. US-001 hiện chỉ dùng `purpose = 'register'`; các giá trị khác đã có trong `CHECK` constraint sẵn để lần sau không phải `ALTER TABLE`. Không đặt unique trên `(phone_number, purpose)` — nhiều hàng cùng cặp vẫn hợp lệ (các OTP cũ đã `used_at`/hết hạn), hàng đang hiệu lực xác định bằng `used_at IS NULL AND expires_at > now()`.
- **Tham số OTP:** TTL 90 giây; sau lần gửi đầu tiên, chặn gửi trong 60 giây; qua 60 giây, mỗi lần yêu cầu OTP mới **hoặc** gửi lại tính một lượt, tổng **4 lượt gửi** cho một lượt đăng ký (1 lần đầu + 3 lượt tính); khi thực hiện lượt thứ 4 thì **khóa gửi 30 phút** cho số điện thoại đó (khoá-gửi không chặn việc nhập OTP); nhập sai tối đa 3 lần thì khóa xác thực cho số điện thoại đó **30 phút**. Payload tạm `pending_registration` sống **24 giờ** (v5.1, tách khỏi TTL 90 giây của OTP); sau khi khoá-gửi hết, `request-otp` (ghi đè payload) và `resend-otp` (giữ payload) đều mở lại, và `resend-otp` cấp OTP mới kể cả khi OTP cũ đã hết hạn.
- **SMS gateway chưa có provider thật** ở giai đoạn này: tạo package `pkg/sms` theo pattern provider-agnostic interface + `Register`/`New` factory của repo, trong đó file `logSMS.go` là implementation **chỉ ghi log** nội dung SMS (không gửi thật). Provider thật (Twilio / AWS SNS / GCP tương đương) thêm sau bằng một subpackage `init()` tự đăng ký theo đúng interface đã định nghĩa — không phải đổi interface.
- **OTP giai đoạn này là hằng số `111111`** (6 chữ số) để dev/test, chưa ngẫu nhiên. Khi chạy thật đổi sang sinh ngẫu nhiên bằng `crypto/rand` — tách riêng chỗ sinh OTP để chỉ đổi một điểm, không rải rác trong usecase.
- Access token JWT sống 15 phút, refresh token sống **7 ngày** (`JWTConfig.RefreshTokenExpiration` = `168h`), rotate mỗi lần dùng
- **Phụ thuộc chéo:** US-002 đã được viết lại và chuyển sang [`docs/specs/features/login/requirements.md`](../login/requirements.md) (v1.0, 2026-09-20): đăng nhập bằng **số điện thoại + mật khẩu**, kèm `refresh` và `logout`. Login bằng email và login bằng OTP để ngoài scope.

---

## US-002: Đăng nhập — đã chuyển sang `login/`

> US-002 (đăng nhập, kèm refresh token và logout) được đặc tả tại [`docs/specs/features/login/`](../login/requirements.md) từ v4.0 của thư mục này (2026-09-20). Không còn nội dung US-002 ở file này; các tham chiếu tới US-002 ở những chỗ khác trong file là tham chiếu chéo có chủ ý.

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
2. **Given** tôi nhập một email chưa từng đăng ký, **when** tôi nhấn "Quên mật khẩu", **then** hệ thống vẫn hiển thị thông báo thành công giống hệt trường hợp email tồn tại ("Nếu email tồn tại, chúng tôi đã gửi hướng dẫn") — không tiết lộ email nào đã đăng ký, cùng lý do bảo mật như `login/` AC2 (một thông báo lỗi chung).
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
- Rate limiting (AC5) cùng cơ chế với `login/` AC7 (khoá theo SĐT, lưu ở DB), áp dụng theo email thay vì theo IP

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