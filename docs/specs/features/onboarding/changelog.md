# Changelog — onboarding Spec

Tracks version history for `requirements.md`, `design.md`, and `tasks.md` in this folder. All three
files share one changelog since they change together as one spec unit. Newest version first.

## v5.2 (2026-09-22)

Sửa tài liệu cũ, **không đổi code, không đổi AC**. Không còn mục nào "chờ cho phép sửa".

- `requirements.md`: §Ghi chú kỹ thuật — thêm endpoint thứ ba `POST /v1/auth/resend-otp` (trước chỉ
  liệt 2 route); "refresh token sống 30 ngày" → **7 ngày** (`JWTConfig.RefreshTokenExpiration` =
  `168h`), khớp `config.dev.yaml`; bỏ ghi chú "tên path là đề xuất, chưa chốt với openapi" (cả ba
  path đã có trong openapi v3.7). Bỏ câu tồn đọng ở header Status.
- `design.md`: helper `hashRefreshToken()` của repository → **`domain.HashRefreshToken`**
  (`domain/session_token.go`); `TokenIssuer` ghi "sẽ đổi chữ ký" → chữ ký thật
  `func(userID, sessionID string)` (Phase 7/v4.2 đã xong); mục TTL refresh token ghi rõ requirements
  đã được sửa.
- `tasks.md`: Phase 1 item `OTPConfig` thêm ghi chú các key đã đổi ở v5.0/v5.1; khối unit test
  repository đổi `hashRefreshToken` → `TestHashRefreshToken` cho `domain.HashRefreshToken`; E2E Phase 8
  ghi rõ `TestRegisterFlow_ExpiredPendingCanRestart_E2E` đã bị Phase 10 thay bằng
  `TestRegisterFlow_RestartOverwritesPending_E2E`; mở rộng ghi chú supersession ở Phase 9.
- Follow-up đã đóng: `docs/knowledge/features/register.md` từng ghi "the fourth counted send" (chỉ 3
  lượt được tính) — đã sửa trong bundle.
- Dọn tham chiếu `docs/ARCHITECTURE.md` (§4 gap auth) trong `design.md`: file này đã bị xoá; các
  route `/v1/users/me` nay chỉ ghi là "chưa được triển khai".

## v5.1 (2026-09-22)

Tách `PendingRegistrationTTL` khỏi OTP TTL và mở lại "đường làm lại" cho US-001. Hệ quả của v5.0:
payload tạm chỉ sống 90 giây (bằng OTP TTL) nên sau khoá-gửi 30 phút không còn gì để `resend-otp`
bám vào, còn `request-otp` bị `409 PHONE_TAKEN` bởi chính lượt dở của mình. **Đổi hành vi quan sát
được: đổi AC6, thêm AC15.**

- `requirements.md`: **AC6 viết lại** — SĐT chỉ bị coi là "đã dùng" khi có tài khoản; hàng pending
  của chính SĐT không chặn; email vẫn phải quét `users.email` **và** `pending_registration.email`
  của một **SĐT khác**. **Thêm AC15** (làm lại/gửi lại sau khi hết khoá). Bảng tham số OTP + schema
  ghi `pending_registration.expires_at` = 24h.
- `design.md`: thêm §"Pending payload và đường làm lại"; `FindConflicts` còn 3 nguồn; `ResendOTP`
  không còn trả `OTP_EXPIRED` khi OTP cũ hết hạn; `ReissueOTP` không đụng `pending_registration`;
  `PendingRegistrationTTL` 90s → 24h; hết marker `⚠️ Not confirmed` (nhánh reset đã chốt).
- `tasks.md`: thêm **Phase 10**, tất cả `[x]`.
- Code (2026-09-22): `config/config.go` + `config.dev.yaml` (`pending_registration_ttl: "24h"`);
  `domain/register.go` (bỏ `PhoneInPending`), `domain/errors.go`; `repository`
  (`FindConflicts` 3 EXISTS + `phone_number <> $1`, `ReissueOTP` thôi gia hạn pending);
  `usecase/register_usecase.go` (`ResendOTP` chấp nhận `ErrOTPInvalid`); `docs/specs/api/openapi.yaml`;
  `services/appService/domain/README.md`.
- Tests: usecase (`TestResendOTP_AfterCodeExpired`), repository (`FindConflicts` 3 nhánh mới,
  `ReissueOTP` không đổi pending), config (24h), E2E (`TestRegisterFlow_RestartOverwritesPending_E2E`,
  `TestRegisterFlow_ResendAfterCodeExpired_E2E`). Verify: `go test -short ./...` + repository/E2E với
  Postgres thật xanh; `check-architecture.sh` + `check_dependency_rule.sh` sạch.
- Reason: anh chốt "pending để im đó, cron dọn sau 24h; request-otp ghi đè, resend-otp giữ payload" —
  ngày 2026-09-22.
- Trade-off: `password_hash` sống trên server tới 24h. Bù lại bằng job dọn hàng hết hạn (Deferred);
  TTL 24h là chốt chặn cuối nếu job chưa chạy.

## v5.0 (2026-09-22)

Thiết kế lại rate-limit gửi OTP cho US-001: bỏ "trần resend" chặn **vĩnh viễn** (known gap:
`resend_count` carry-forward không bao giờ reset), thay bằng **trần số lần gửi theo SĐT + khoá tạm
30 phút**. **Đổi code, đổi hành vi quan sát được, đổi AC5 và AC12.**

- `requirements.md`: AC5 viết lại theo mô hình "1 lần đầu miễn phí + 3 lượt tính = 4 lượt/lượt đăng
  ký, chạm trần khoá gửi 30 phút"; AC12 đổi khoá 15 phút → 30 phút; bảng tham số OTP + schema thêm
  `otp_send_limits`.
- `design.md`: thêm bảng `otp_send_limits` (migration `0003`) tách khỏi vòng đời hàng `otp_codes`;
  bỏ hẳn cơ chế carry-forward `resend_count`; `request-otp` nay **cũng** tiêu một lượt gửi; thêm mã
  lỗi `429 OTP_SEND_LIMITED` + typed error `OTPSendLimitedError`; bỏ `ErrOTPResendExhausted`;
  `OTPConfig` đổi (`send_cooldown` 60s, `max_send_attempts` 3, `send_lock_duration` 30m,
  `lock_duration` 30m; bỏ `max_resend`); flow 1 Step 4, flow 2 Step 3–4, flow 3 Step 3–4 cập nhật.
- `tasks.md`: thêm **Phase 9** — migration 0003, domain, usecase, repository, delivery, test, E2E —
  **đã xong 2026-09-22 (tất cả `[x]`)**.
- Code (2026-09-22): `migrations/0003_otp_send_limits.sql`; `config/OTPConfig` + `config.dev.yaml`;
  `domain/otp_send_limit.go`, `domain/errors.go`, `domain/register.go`, mock; `usecase/register_usecase.go`
  (xoá `checkOTPLimits`/`checkCooldown`/`resendsRemaining`, policy chuyển vào repository);
  `repository/postgres_registration_repository.go` (`applySendLimits`, `ReleaseSendLimit`, xoá
  `latestResendCount`/carry-forward); `delivery/http/errors.go`; `docs/specs/api/openapi.yaml`;
  `migrations/README.md`. Verify: `go test -short ./...` xanh; repository test + E2E chạy Postgres
  thật (testcontainers) xanh; `check-architecture.sh` + `check_dependency_rule.sh` sạch.
- Reason: bug "gửi lại 3 lần rồi chặn vĩnh viễn, không có đường gỡ" — ghi ở known gap
  `docs/knowledge/features/register.md`; anh chốt phương án "khoá 30 phút" ngày 2026-09-22.

### Thay đổi hành vi quan sát được ở client

- `POST /v1/auth/register/request-otp` lần đầu của một lượt: không đổi (200, gửi OTP).
- Sau đó, `request-otp` **và** `resend-otp` cùng chịu một bộ đếm theo SĐT: trong 60s kể từ lần gửi
  gần nhất → `429 OTP_RESEND_TOO_SOON` (kèm `retry_after_seconds`); qua 60s, mỗi lần gọi là một lượt.
- Lượt gửi thứ 4 → 200 (SMS vẫn gửi), đồng thời khoá gửi 30 phút. Từ lượt thứ 5 đến khi hết khoá →
  `429 OTP_SEND_LIMITED` kèm `locked_until`. Hết 30 phút, bộ đếm đặt lại.
- Khoá-gửi **không** chặn `verify-otp`: OTP hợp lệ cuối cùng vẫn nhập được trong 90s của nó.
- `verify-otp` sai OTP 3 lần → khoá xác thực **30 phút** (trước là 15 phút).
- `429 OTP_RESEND_EXHAUSTED` **không còn được trả** ở bất kỳ nhánh nào.
- `resend_remaining` trong response `resend-otp` đổi nghĩa thành số lượt gửi còn lại trong lượt
  (lần đầu = 3; chạm trần = 0) — xem design.md.

### Cross-file follow-ups

- `docs/knowledge/features/register.md` — cập nhật Known gap `resend_count` reset (concept này thuộc
  quy trình OKF, không sửa trong spec).
- `docs/specs/api/openapi.yaml` — **đã cập nhật**: thêm `OTP_SEND_LIMITED`, bỏ `OTP_RESEND_EXHAUSTED`,
  cập nhật mô tả `locked_until` / `retry_after_seconds` và `resend_remaining`.
- `login/` không bị ảnh hưởng (login không dùng OTP — `purpose=login` ngoài scope).

### ⚠️ Not confirmed (đã chốt ở v5.1)

- Bộ đếm gửi tự đặt lại khi lần gửi cuối đã quá 30 phút (người dùng bỏ ngang, chưa chạm trần), hay
  chỉ đặt lại khi khoá 30 phút hết? Design v5.0 chọn **"quá 30 phút không gửi thì đặt lại"** (dùng
  chung một con số `send_lock_duration`) — **anh đã xác nhận ngày 2026-09-22**: quá 30 phút thì lần
  gửi kế tiếp mở lại và là lần đầu **miễn phí** của cycle mới (xem v5.1).

## v4.5 (2026-09-21)

Fix bug: một lượt đăng ký bỏ dở chặn SĐT/email đó mãi mãi. **Đổi code và hành vi quan sát được.**

- `design.md`: flow 1 Step 2 — `FindConflicts` chỉ tính hàng `pending_registration` còn hạn
  (`expires_at > now()`); thêm mục Confirmed giải thích. Các flow khác không đổi.
- `tasks.md`: thêm Phase 8 (4 mục, đều `[x]`); ghi chú Deferred về dọn hàng hết hạn cập nhật lại —
  `pending_registration` hết hạn nay cũng vô hiệu về logic.
- `requirements.md`: chỉ bump. AC không đổi — code giờ khớp AC12 ("làm lại từ bước 1").
- Code: `repository/postgres_registration_repository.go` (`FindConflicts` SQL + doc), `domain/register.go`
  (doc interface), test repository + E2E `services/appService/e2e/register_e2e_test.go`, harness
  `e2e/harness.go` + `e2e/main_test.go`.
- Reason: phát hiện khi cập nhật concept OKF `register.md`; anh chốt "thêm kiểm tra expire".

### Thay đổi hành vi quan sát được ở client

- `POST /v1/auth/register/request-otp` với SĐT/email chỉ vướng một hàng pending **đã hết hạn**:
  `409 PHONE_TAKEN` / `EMAIL_TAKEN` → `200` (cấp OTP mới, payload mới ghi đè payload cũ).
- Pending còn hạn: không đổi, vẫn `409`.

### Cross-file follow-ups

- `docs/knowledge/features/register.md` — bỏ Known gap tương ứng (làm cùng lượt này).
- Chưa làm: reset `resend_count` / trần SMS theo SĐT — chờ anh chọn phương án A/B/C.

## v4.4 (2026-09-21)

`tasks.md` + header `design.md`. **Không đổi code, không đổi hành vi, không đổi flow.**

- `tasks.md`: Phase 6 mục dọn hàng hết hạn chuyển xuống Deferred — anh chốt chấp nhận ở MVP (2026-09-21).
- `tasks.md`: Phase 6 tick `go test -short ./...` sạch — 3 test `pkg/utils` từng đỏ đã được sửa ở
  `json-body-hardening`; chạy lại 2026-09-21 thì toàn repo xanh.
- `tasks.md` Deferred: sửa ghi chú "hàng hết hạn vô hiệu" — không đúng với `pending_registration` (hàng hết hạn
  vẫn chặn `request-otp` bằng `409 PHONE_TAKEN`); ghi là chờ quyết định sửa.
- `tasks.md` + `design.md`: Status → `done (US-001)`; bỏ các câu cũ về `pkg/utils` đỏ và cron chưa có; ghi rõ
  còn ngoài task list: audit E2E cho US-001 và placeholder OTP/SMS. `requirements.md` không đổi (US-003/004 vẫn
  `in progress`).

## v4.3 (2026-09-21)

Sửa mô tả envelope response của US-001 cho khớp code thật (`pkg/utils.ApiResponse`). **Không đổi code, không
đổi hành vi** — code đã đúng từ đầu; chỉ tài liệu mô tả sai.

- `requirements.md`: không đổi nội dung, đã kiểm (prose không nhắc shape envelope). Chỉ bump header.
- `design.md`: dòng "Delivery / Shared" (`errors.go`) bỏ `{error: {code, message, details}}`, ghi envelope thật:
  lỗi = `{success: false, error: {code, message, type}, details}` (`details` nằm cạnh `error`), thành công =
  `{success: true, data: {...}}`; thêm ghi chú các body `Success → ...` trong Step-by-step Flow là nội dung của
  `data`. Step-by-step Flow không đổi cấu trúc.
- `tasks.md`: Phase 4 mục `errors.go` sửa mô tả envelope + ghi chú; vẫn `[x]` (code không đổi). Không tick/untick mục khác.
- Reason: `login/` (changelog v1.5) phát hiện `ErrorResponse` của US-001 trong `openapi.yaml` mô tả sai envelope;
  kiểm lại thấy lỗi gốc nằm ở spec này, và cả 3 response thành công của US-001 cũng thiếu lớp `{success, data}`.
  Bằng chứng: `register_handler.go` dùng `writeDecodeError`/`writeDomainError`/`utils.WriteSuccessResponse*` y
  hệt `login_handler.go`; `register_handler_test.go` decode `{success, data}`.

### Cross-file follow-ups

- `docs/specs/api/openapi.yaml` — **đã sửa, v3.5**: mọi response lỗi của 3 route US-001 chuyển sang
  `ApiErrorResponse` (gộp error code + `details` của US-001 vào đó); xoá schema `ErrorResponse` sai; tách
  `RequestOTPData`/`ResendOTPData`/`VerifyOTPData` (payload) khỏi `*Response` (bọc `{success, data}`);
  `LoginResponse.data` trỏ `VerifyOTPData`. YAML parse OK, không còn `$ref` treo hay schema thừa.
- `docs/knowledge/features/` — concept identity (khi `il-okf-writer` viết) phải mô tả envelope thật.

## v4.2 (2026-09-21)

Phase 7 implement xong: access token của `verify-otp` mang claim `sid`. Không đổi hành vi ngoài của US-001.

- Code: `domain.TokenIssuer` → `func(userID, sessionID string)`; `usecase` closure gắn `extra{"sid": sessionID}` vào
  `GenerateAccessToken`; `repository.CreateUserAndSession` sinh `sessionID` trước `issueTokens` rồi INSERT đúng
  giá trị đó. Test: usecase pin exact `extra`; repository test mới chứng minh `sessionID` issuer nhận = `user_sessions.session_id`.
- Verification: `go build`/`go vet`/`go test ./services/appService/...` xanh (Postgres Docker tạm); e2e thật
  giải mã JWT → `sid` khớp `session_id` trong DB; hai script kiểm kiến trúc sạch.
- `design.md`: khối "Thay đổi kế hoạch" ở Step 6 flow 2 đổi thành "Đã implement"; dòng `GenerateAccessToken` cập nhật.
  `domain/README.md`, `usecase/README.md`: cập nhật chữ ký/`extra`.
- `tasks.md`: 6 mục Phase 7 tick `[x]` kèm ghi chú verify. `requirements.md`: chỉ bump version.
- Reason: điều kiện tiên quyết của `login/tasks.md` Phase 0 đã thoả.

### Cross-file follow-ups

- `docs/specs/features/login/tasks.md` Phase 0 — có thể tick/mở khoá; `login/design.md` "Phụ thuộc: sửa US-001" đã thực hiện.
- `docs/knowledge/features/` — khi viết concept identity qua `il-okf-writer`, phản ánh claim `sid`.

## v4.1 (2026-09-21)

Thêm một thay đổi kế hoạch cho US-001 do `login/` yêu cầu: access token của `verify-otp` phải mang claim
`sid`. Không đổi hành vi ngoài của US-001.

- `requirements.md`: **không đổi nội dung** (đã kiểm tra: status/body/error code của `verify-otp` giữ
  nguyên; claim `sid` là chi tiết kỹ thuật của token). Chỉ bump version cho khớp.
- `design.md`: Step 6 flow 2 thêm khối "Thay đổi kế hoạch (v4.1, chưa implement)" mô tả `sessionID` sinh
  trước khi mint và `TokenIssuer` đổi chữ ký; đoạn mô tả code hiện tại giữ nguyên vì Phase 7 chưa làm.
- `tasks.md`: **thêm Phase 7** (6 mục, tất cả `[ ]`). Không tick/untick task cũ — các item `[x]` của
  Phase 1–5 mô tả code ở thời điểm đó vẫn đúng cho tới khi Phase 7 được implement.
- Reason: `login/` chọn `logout` bằng access token + `sid` (2026-09-21); session do `verify-otp` tạo phải
  logout được (login AC12).

### Cross-file follow-ups

- `docs/specs/features/login/tasks.md` Phase 0 — chờ Phase 7 ở đây.
- `docs/knowledge/features/` — khi Phase 7 xong và concept identity được viết, phải phản ánh claim `sid`.

## v4.0 (2026-09-20)

US-002 (đăng nhập) được **chuyển ra** khỏi folder này sang `docs/specs/features/login/` (v1.0,
prospective). Đổi scope của spec unit → major bump. Các US còn lại (US-001 đã implement; US-003,
US-004 còn ở mức user story) **giữ nguyên tại đây**, sẽ được chuyển dần theo cùng cách.

- `requirements.md`: **xoá toàn bộ section US-002** (user story, 6 AC, Out of scope, Ghi chú kỹ thuật) và
  thay bằng một khối con trỏ sang `../login/requirements.md`. Sửa các tham chiếu chéo còn lại cho khớp:
  bảng "Gate email chưa verify" (dòng US-002 nay trỏ `login/` và ghi rằng login chỉ hỗ trợ SĐT);
  "Phụ thuộc chéo" (không còn "US-002 sẽ phải sửa" — đã viết lại); dòng `otp_codes.purpose = 'login'`
  (nay ghi rõ login bằng OTP ngoài scope); tham chiếu "US-002 AC2/AC3" và "US-002 AC6" trong US-003
  đổi sang `login/` AC2 / AC7. Prose của US-001 và US-004 không đổi.
- `design.md`: chỉ đổi con trỏ — Known gaps `user_devices` (nay trỏ `login/` + migration `0002`) và
  Known gaps `ResendOTP` (còn lại mỗi US-003 là nơi bổ sung nhánh `purpose` khác). **Đã kiểm tra, không
  cần đổi nội dung kỹ thuật**: các Step-by-step Flow, bảng Layers, cột `purpose` của `otp_codes` không
  bị ảnh hưởng vì `login/` không dùng `purpose = 'login'`.
- `tasks.md`: chỉ đổi con trỏ ở mục Deferred (device, US-002). **Không tick/untick task nào**; mọi trạng
  thái `[x]`/`[ ]` của US-001 giữ nguyên.
- Reason: anh muốn tách US-002 thành spec riêng (`login/`) và sẽ chuyển dần các user story sau; đây
  là bước đầu tiên của việc tách.

### Cross-file follow-ups

- Dòng "Access token JWT sống 15 phút, refresh token sống **30 ngày**" trong `requirements.md` §Ghi chú
  kỹ thuật lệch `config.dev.yaml` / `0001` (168h = 7 ngày). **Chưa sửa** ở lần này (chờ cho phép);
  `login/` đã theo 7 ngày. Dòng "Hai chỗ trong §Ghi chú kỹ thuật đã cũ" ở header Status vẫn còn hiệu lực.
- `docs/specs/features/login/` cần migration `0002`, `openapi.yaml` 3 path, OKF concept — xem
  `../login/changelog.md`.
- AC11 vế thứ hai của US-001 ("refresh token tự động cấp access token mới") vẫn **không thoả ở phạm vi
  US-001**; nay được theo dõi bởi `login/` AC9–AC11.

## v3.3 (2026-09-17)

Phase 4 (Delivery) and Phase 5 (Hạ tầng dùng chung) implemented and tested. US-001 is now
end-to-end runnable — request-otp → resend-otp → verify-otp exercised against a real Postgres
through the actual HTTP server, not just unit/integration tests per layer. One addition to the
Layers table that wasn't in the original design (an adapter), no reversed decision.

- Code:
  - `services/appService/delivery/http/register_handler.go` + `errors.go` — 3 handlers, DTOs,
    shared error mapping — plus `register_handler_test.go` (21 test funcs) and
    `mock_register_usecase_test.go` (hand-written mock for the consumer-defined `registerUsecase`
    interface).
  - `pkg/sms/sms.go` + `pkg/sms/logsms/log_sms.go` — provider-agnostic Sender + the log-only
    implementation — plus their test files.
  - `services/appService/repository/sms_otp_sender.go` — adapter from `pkg/sms.Sender` to
    `domain.OTPSender`, not in the original Layers table (see design.md note on that row): needed
    because `pkg/sms` is deliberately domain-agnostic (just "send text to a number"), so something
    has to turn an OTP code + purpose into message text, and that something has to import both
    `domain` and `pkg/sms` — which is exactly what `repository/` already does for every other port.
  - `services/appService/cmd/main.go` — composition root, rewritten from the pre-rewrite template
    (commit `e622180`) with the two-step flow's dependencies wired in and the old single-step
    login/profile/avatar/content wiring dropped (no usecase exists yet for any of those).
  - `services/appService/config/config.go` + `config.dev.yaml` — `AppConfig.SMS` (`SMSConfig{Provider}`).
  - `docs/specs/api/openapi.yaml` — written *after* the handlers ran for real over `curl`, so every
    status/code/field in it is checked against an actual response, not drafted ahead of the code.
  - `go.mod`/`go.sum`/`vendor/`: `go mod tidy` removed `github.com/DATA-DOG/go-sqlmock`, an unused
    dependency that predates this session (not introduced by Phase 3-5's own imports, which added
    nothing new beyond already-vendored modules). `go mod vendor` changed nothing else.
- Verification: `go build ./...`, `go vet ./...`, `scripts/check-architecture.sh`, and
  `.claude/skills/golang-clean-arch-tdd/scripts/check_dependency_rule.sh services/appService` all
  clean. `go test -short ./...` — every package this feature touches passes; the only remaining
  failures are 3 pre-existing, unrelated `pkg/utils` tests (tracked in tasks.md Phase 6, unchanged
  by this session). Full end-to-end run against a throwaway Docker Postgres (not part of the repo
  or CI) via `go run ./services/appService/cmd` + `curl`, confirming AC1/AC2/AC3/AC5(cooldown)/
  AC6/AC7 on real infrastructure: request-otp (200, OTP logged) → immediate resend (429
  OTP_RESEND_TOO_SOON, cooldown working) → verify-otp with a wrong code (400 OTP_INVALID) →
  verify-otp with the right code (201, tokens + user, `email_verified_at` NULL, session expiry
  +168h) → a second request-otp for the same number (409 PHONE_TAKEN).
- `design.md`:
  - Header: `Status:` now lists every layer as done except the two open Phase 6 items.
  - Layers table: drops `*(chưa tồn tại)*` from the delivery/cmd/pkg-sms/openapi rows; adds the
    `sms_otp_sender.go` row (see above) with its own rationale, since it wasn't anticipated in the
    original design; notes `pkg/sms.Config`'s extra `Logger` field and `userResponse`'s existence.
- `tasks.md`: Phase 4's 11 items and Phase 5's remaining 3 (pkg/sms, logsms, SMS config; the
  `NormalizePhone` pair were already done at v3.1) all ticked with verification notes. Phase 6's
  "chạy end-to-end" item ticked with the curl trace above. Phase 6's `go test -short ./...` item
  stays unticked — same 3 unrelated `pkg/utils` failures as before, nothing new. Header `Status:`
  updated.
- `requirements.md`: **no content change** — version bumped to keep the three-file convention only.
- Reason: continuing tasks.md's Phase 4/5 checklists per `.claude/skills/golang-clean-arch-tdd`'s
  inside-out build order, now that Phase 3 (Repository) is done.

### Cross-file follow-ups

- `docs/knowledge/features/` — the identity concept can now be written: the flow is genuinely
  exercisable end-to-end (this version's verification section), which is `il-okf-writer`'s
  precondition. Not written in this session — that skill owns `docs/knowledge/`, this one does not
  invoke it automatically. Next step for whoever picks this up: call `il-okf-writer` for the
  identity/US-001 concept.

## v3.2 (2026-09-17)

Phase 3 (Repository) implemented and tested. No new requirement, no reversed decision — spec updated
to match code, plus one behavior the code makes precise that flow 2 Step 3's literal SQL had left
ambiguous.

- Code: `services/appService/repository/postgres_registration_repository.go` (all nine
  `domain.RegistrationRepository` methods) + `repository/postgres_registration_repository_test.go`
  (10 integration test funcs against real Postgres, plus 3 DB-free unit tests for the helpers).
  `go build ./...` and `go vet ./...` clean; `scripts/check-architecture.sh` and
  `.claude/skills/golang-clean-arch-tdd/scripts/check_dependency_rule.sh services/appService` both
  pass. Integration tests skip cleanly under `go test -short` (no Postgres required) and were run
  once against a throwaway Docker Postgres container to verify the actual SQL — that container is
  not part of the repo or CI, so `-short` remains the reproducible baseline.
- `design.md`:
  - Header: `Status:` adds `repository/` to the "has real code" group.
  - Layers table: drops `*(chưa tồn tại)*` from the repository row; lists `hashRefreshToken()`
    alongside the other two internal helpers.
  - New note under the Layers table documenting `FindActiveOTP`'s actual query condition —
    `(locked_until > now() OR (used_at IS NULL AND expires_at > now()))` — versus flow 2 Step 3's
    literal `used_at IS NULL`. The literal version breaks two cases repository tests caught: an
    expired-but-never-marked-used row would still read "active" (no cron clears `otp_codes`, so this
    is the normal case, not an edge case — AC4), and a locked row (which `LockOTP` also marks
    `used_at`) would become invisible to the very lookup meant to report `OTP_LOCKED` (AC12). This is
    not a new decision — it is exactly what `domain.OTPCode`'s existing doc comment already said
    ("FindActiveOTP matches on UsedAt OR LockedUntil rather than UsedAt alone", written during
    Phase 1/2) — Step 3's SQL snippet just hadn't been updated to say the same thing literally.
  - Opening paragraph: repository/ added to the list of layers rewritten after commit `0475281`.
- `tasks.md`: Phase 3's nine items ticked `[x]` with verification notes (which test proves which
  behavior). Two Phase 2 items that were half-done pending Phase 3 — the AC3 "same tx" assertion and
  ResendOTP's "old OTP gets `used_at`" assertion — are now also `[x]`, since both are directly proven
  by the new repository tests. Header `Status:` updated.
- `requirements.md`: **no content change** — version bumped to keep the three-file convention only.
- Reason: continuing tasks.md's Phase 3 checklist per `.claude/skills/golang-clean-arch-tdd`'s
  inside-out build order, now that Phase 1 (Domain) and Phase 2 (Usecase) are done.

### Cross-file follow-ups

- `docs/knowledge/features/` — identity concept still not written. Repository is done but the flow
  is not exercisable end-to-end yet (no `delivery/http`, no route in `cmd/main.go`); `il-okf-writer`
  writes concepts only once a flow runs end-to-end, per its own rule. Unchanged from v3.1/v3.0 — next
  candidate trigger is Phase 4 (Delivery) + Phase 6's end-to-end run.

## v3.1 (2026-09-17)

Đồng bộ spec với code hiện có. Không có yêu cầu mới, không có quyết định thiết kế mới — chỉ sửa
những chỗ spec còn mô tả trạng thái cũ.

- `design.md`:
  - Header: `Status:` đổi từ `prospective — chưa có code` sang `in progress`, liệt kê rõ nhóm nào đã
    có code thật (`domain/`, `usecase/`, `migrations/0001`) và nhóm nào còn mở. Bỏ câu "Toàn bộ file
    trong bảng Layers bên dưới **chưa tồn tại**" — nay sai với 11 dòng.
  - Bảng Layers: bỏ marker `*(chưa tồn tại)*` ở 11 dòng đã có file thật (`user.go`,
    `pending_registration.go`, `otp.go`, `register.go` ×3, `errors.go`, mock ×2, `register_usecase.go`
    ×2, `0001_register_onboarding.sql`). Bổ sung field/method mà code có thêm so với bản thiết kế
    gốc — `PendingRegistration.ID`/`CreatedAt`, `RequestOTPResult`, `ResendOTPResult`,
    `RegistrationConflicts`, `ErrUnsupportedOTPPurpose`, `ErrSMSSendFailed`, hai typed error,
    method `DeletePendingRegistration`.
  - Bảng "Deliberately stubbed for now": dòng `migrations/` đổi từ "**Rỗng**" sang "**Một file**" —
    `0001_register_onboarding.sql` đã có.
  - Năm chỗ trong Step-by-step Flow còn dùng tên hằng cũ (`otpMaxResend`, `otpMaxAttempts`,
    `otpResendCooldown`) đổi sang tên field của `config.OTPConfig` (`MaxResend`, `MaxAttempts`,
    `ResendCooldown`).
- `tasks.md`:
  - Phase 2, item OTP policy: `[ ] ⚠️ CHƯA CHỐT` → `[x]`, nội dung viết lại theo `config.OTPConfig`
    như code đang làm.
  - Dòng cảnh báo đầu file (`⚠️ HAI XUNG ĐỘT CHƯA CHỐT`) bỏ — xung đột đã giải quyết.
- `requirements.md`: **no content change** — đã kiểm tra, không có tiêu chí nào (AC1..AC14) bị lần
  đồng bộ này làm sai. Hai sửa ở header: (a) bump version từ `1.8` lên `3.1` để khôi phục quy tắc
  "cả ba file dùng chung một version" (`references/versioning_and_changelog.md` §"Header format");
  việc giữ nguyên `1.8` từ v3.0 là ngoại lệ có ý thức, nhưng hệ quả là ba header lệch nhau và người
  đọc không còn biết chúng đã được review cùng nhau ở version nào — nên ngoại lệ đó được bỏ ở
  version này. (b) `Status:` đổi từ `draft` sang `in progress`. Câu biện minh cũ — "US-001 đủ quyết
  định để viết `design.md`" — viết từ lúc `design.md` chưa tồn tại, nay đã sai; và template của
  `il-spec-writer` (`assets/templates/design.md.template`, `tasks.md.template`) chỉ định `design.md`
  và `tasks.md` *mirror* `Status` của `requirements.md`, nên `requirements.md` là file phải sửa chứ
  không phải hai file kia.
- Reason: một lượt kiểm tra spec đối chiếu với code phát hiện `design.md` vẫn ở chế độ prospective
  trong khi `domain/`, `usecase/` và `migrations/0001` đã được viết lại từ commit `0475281`. Item
  OTP policy ở `tasks.md` mô tả thiết kế `const` đã bị bỏ — code đã refactor sang `config.OTPConfig`
  theo đúng quy tắc §"Configuration" của `golang-clean-arch-tdd`. Cả hai là spec đi sau code, không
  phải code đi sau spec.

## v3.0 (2026-09-17)

Năm câu hỏi mở của v2.0 đã được trả lời. `design.md` không còn marker `⚠️ Not confirmed` nào;
`tasks.md` bỏ hẳn mục "Blocked".

Các quyết định đã chốt (trước đó là câu hỏi mở):

1. **Chuẩn hoá số điện thoại.** Luôn chuẩn hoá về E.164 trước khi validate và trước khi lưu. Số
   local (`"0901234567"`) nhận country code `+84`; số đã có `+` với country code khác (`+1`, `+81`)
   giữ nguyên, không ép về +84; separator gõ tay (space, `-`, `.`, `(`, `)`) bị bỏ. Không có nhánh
   nào từ chối số local. Hàm `NormalizePhone(raw string) string` đặt ở **`pkg/validation`**, không
   trong `usecase` — US-002/US-003 sẽ cần đúng phép chuẩn hoá này, và hai bản copy lệch nhau sẽ làm
   `UNIQUE` constraint bắt hụt.
2. **`pending_registration.expires_at` = `created_at + 90 giây`** — bằng đúng TTL của OTP, không dài
   hơn. Hệ quả chấp nhận có ý thức: quá 90 giây thì payload tạm hết hạn và người dùng phải nhập lại
   từ đầu. Lý do: không giữ bản copy `password_hash` sống lâu trên server.
3. **Thêm endpoint thứ ba `POST /v1/auth/resend-otp`**, thiết kế dùng chung cho **mọi** `purpose`
   (`register` / `login` / `reset_password` / `verify_email`) để US-002/US-003 tái sử dụng, không
   phải thêm route mới. US-001 chỉ hiện thực `purpose='register'`; ba giá trị còn lại trả
   `ErrUnsupportedOTPPurpose` cho tới khi feature tương ứng wire vào. Hằng `OTPPurpose` và
   `chk_otp_codes_purpose` khai đủ bốn giá trị ngay từ `0001` để không phải `ALTER` lại.
4. **SMS lỗi thì API trả lỗi** — `500 SMS_SEND_FAILED`, không log-rồi-trả-200. Hàng OTP ở Step 4 đã
   commit nên vẫn hiệu lực; lần thử lại ngay bị chặn bởi cooldown 60 giây. Đánh đổi có chủ ý.
5. **Refresh token sống 7 ngày** — `user_sessions.expires_at = now() + 168h`, đọc từ
   `JWTConfig.RefreshTokenExpiration` (`config.dev.yaml` ghi `refresh_token_expiration: "168h"`),
   không hardcode trong usecase.

- `design.md`: bốn thay đổi cấu trúc, còn lại là chốt giá trị.
  - Thêm mục **"Step-by-step Flow — Gửi lại OTP"** (rule 5 bắt buộc — flow này có 2 lệnh ghi DB
    tuần tự). Bước 4(c) của nó — gia hạn `pending_registration.expires_at` thêm 90 giây — là khác
    biệt duy nhất so với flow 1; thiếu nó thì resend thành công vẫn để payload chết ở giây thứ 90.
  - `OTPSender` đổi từ `SendRegistrationOTP(ctx, phoneNumber, otpCode)` sang **một method duy nhất**
    `SendOTP(ctx, phoneNumber, otpCode, purpose)` — flow 1 và flow resend dùng chung một đường gửi.
  - `RegistrationRepository` thêm method `ReissueOTP`.
  - Bảng Layers, Request flow, wiring, Migration, API spec: 2 route → **3 route**; handler 2 → 3
    (`RequestOTP`, `ResendOTP`, `VerifyOTP`).
  - Step 6 flow 2: marker refresh TTL thay bằng khẳng định 7 ngày.
- `tasks.md`: bỏ mục "Blocked" (không còn gì chặn); Phase 1 thêm hằng `OTPPurpose` đủ bốn giá trị +
  sentinel `ErrUnsupportedOTPPurpose`; Phase 2 thêm item `ResendOTP` + test; Phase 3 thêm item
  `ReissueOTP` + test; Phase 4 thêm handler `ResendOTP` + route thứ ba; Phase 5 thêm item
  `NormalizePhone` + test; Phase 6 thêm item xác nhận resend không phá AC12.
- `requirements.md`: **không sửa** — người dùng yêu cầu giữ nguyên file này ở lần làm việc hiện tại,
  nên nó vẫn ở version 1.8. Hệ quả là các quyết định ở trên được ghi ở `design.md`/`changelog.md`
  thay vì ở "Open questions (resolved)" trong `requirements.md` như quy tắc 3 của skill yêu cầu.
  Đây là **vi phạm có chủ ý, có ý thức**, không phải bỏ sót.
- Reason: người dùng trả lời 5 câu hỏi chặn ở v2.0. Ba câu (1, 3, 4) mở rộng phạm vi hoặc đổi
  contract nên bump major; hai câu (2, 5) chỉ chốt giá trị.

Cross-file follow-ups recorded at this version:

- **`requirements.md` §Ghi chú kỹ thuật giờ đã cũ ở hai chỗ** — (a) nó liệt kê **2** endpoint, trong
  khi bản design chốt **3**; (b) nó ghi refresh token sống "30 ngày", đã bị thay bằng 7 ngày. Cả hai
  chờ người dùng cho phép sửa `requirements.md`.
- `docs/specs/api/openapi.yaml` — phải phủ **3** route (không phải 2), gồm cả `POST /v1/auth/resend-otp`
  với tham số `purpose`. tasks.md Phase 4.
- `migrations/0001_register_onboarding.sql` — `chk_otp_codes_purpose` phải liệt kê đủ **bốn** giá trị
  purpose ngay từ đầu, nếu không US-002/US-003 sẽ phải `ALTER` constraint.
- `services/appService/config/config.dev.yaml` — `refresh_token_expiration: "168h"` nay là **nguồn
  chân lý** cho lifetime của refresh token; `user_sessions.expires_at` đọc từ đó, không hardcode.

## v2.0 (2026-09-17)

- **Phạm vi thu hẹp còn US-001.** `design.md` và `tasks.md` chỉ đặc tả luồng đăng ký 2 bước; US-002
  (đăng nhập), US-003 (quên mật khẩu) và US-004 (hồ sơ) vẫn nằm trong `requirements.md` nhưng chưa
  được thiết kế ở version này.
- `requirements.md`: **không đổi gì** — không nội dung, không header. File đang ở version 1.8 và
  người dùng yêu cầu giữ nguyên; đây là **ngoại lệ có chủ ý** so với quy tắc "ba file cùng version"
  trong `references/versioning_and_changelog.md`, vì `requirements.md` còn phủ US-002/003/004 chưa
  được đặc tả ở version này nên bump nó sẽ sai lệch với nội dung thật. `design.md` và `tasks.md` đi
  trước ở 2.0; khi US-002/003/004 được đặc tả, `requirements.md` sẽ được chỉnh và cả ba hội tụ.
- `design.md`: viết lại từ đầu, **prospective** — không còn là bản retroactive của code cũ. Toàn bộ
  Layers & components trỏ tới file chưa tồn tại; Step-by-step Flow mô tả 2 luồng
  (request-otp, verify-otp) mà các bước đều là kế hoạch, không phải mô tả code đã có.
- `tasks.md`: viết lại từ đầu, toàn bộ `[ ]`. Thêm **Phase 5 — Hạ tầng dùng chung** cho `pkg/sms/`
  (package mới, chưa tồn tại trong repo).
- Reason: commit `0475281` xoá toàn bộ `services/appService/{cmd,domain,usecase,repository,delivery}`,
  cả 7 file `migrations/`, cùng `design.md`/`tasks.md`/`changelog.md` cũ — repo không còn code của
  feature này. Giữ lại bản `design.md` cũ (mô tả code đã bị xoá) sẽ khiến spec nói dối về trạng thái
  repo, nên nó được thay bằng bản prospective.

Các quyết định được chốt ở version này (trước đó là câu hỏi mở):

- File migration bắt đầu lại từ `0001` vì `migrations/` đã rỗng — không giữ số 0001-0007 cũ.
- Refresh token lưu ở bảng `user_sessions` (kế thừa thiết kế cũ, không đổi sang cột hash).
- `POST /v1/auth/refresh` **ngoài scope** US-001 — access token hết hạn trong phiên bản này không có
  đường tự gia hạn; đây là khoảng trống đã biết, ghi ở "Known gaps carried by this feature".
- Hai path `/v1/auth/register/request-otp` và `/v1/auth/register/verify-otp` được **chốt**, không còn
  là "đề xuất".

Cross-file follow-ups recorded at this version:

- `docs/specs/api/openapi.yaml` — `docs/specs/api/` hiện **rỗng**; cần dựng mới với 2 route của US-001
  (tasks.md Phase 4).
- `migrations/0001_register_otp.sql` + `0002_users_baseline.sql` — cần tạo; `docs/specs/db/` cũng đang
  rỗng nên ERD `register-erd.md` cũ sẽ phải viết lại nếu muốn giữ.
- `docs/knowledge/features/` — đang rỗng (6 concept content bị xoá cùng commit `0475281`). Identity
  là concept **mới** cần `bdd-okf-docs` tạo, không phải refresh.

## v1.2 (2026-09-16)

- **Breaking API change.** The phone field on the wire is renamed `phone_number` → `phone`, in both
  requests and responses, so one name is used end to end instead of request and response drifting
  apart. Affects `POST /v1/auth/register` (request and the `user` in its response) and
  `GET`/`PATCH /v1/users/me`. Existing clients sending `phone_number` now fail with
  `INVALID_PHONE` (the key is absent, so `required` fires); clients reading `phone_number` off a
  response get nothing.
- `requirements.md`: US-001 invalid-phone criterion now says `details {"field":"phone"}`; US-004's
  `GET /v1/users/me` field list and its `PATCH` ignore-list renamed to match.
- `design.md`: the register step-by-step flow's failure line renamed to `{field: phone}`.
- `tasks.md`: no content change, version bumped to stay in step with the other two files.
- **Not renamed:** the database column stays `users.phone_number`, along with the unique constraint
  `users_phone_number_key` and every SQL statement referencing them. The repository maps columns to
  struct fields by hand rather than by tag, so the wire name and the column name are free to differ
  — and renaming a `UNIQUE NOT NULL` column on a live table buys nothing here.

## v1.1 (2026-09-15)

- `requirements.md`: no behavior change. Fixed three stale cross-references in the "Pending" list —
  items 1, 2 and 3 pointed at `tasks.md` Phase 5, but Phase 5 is "Integration & verification" and
  the open work actually lives in Phase 6. Rewritten to name the specific item (`Phase 6 item 1`,
  `item 3`, `items 5-6`) so the pointer survives further renumbering. The `⚠️ Not confirmed`
  marker on US-004's 401 criterion was left open — the gap it flags is still real.
- `design.md`: no structural change to any flow. Added a "Known gaps" bullet recording that no
  refresh endpoint exists and `token_version` is never bumped past the hardcoded `1` in
  `insertSession` — the fact that makes US-001's "rotate mỗi lần dùng" note inaccurate. Steps 1-4
  of every flow, the Layers table and the transaction boundaries are unchanged.
- `tasks.md`: header bumped for consistency. Two new Phase 6 items — decide whether refresh-token
  rotation is in scope for this feature, and resolve the access/refresh token lifetime
  contradiction (config says `15m`/`168h`, the original user story says 15 minutes / 30 days).
  Also tightened the avatar item to a dimension/decompression-bomb check rather than a vague
  "content check". No item was checked off — nothing new was implemented.
- Reason: a verification pass against the real code found the new `tasks.md` Phase 6 had grown
  past what `requirements.md`'s "Pending" list described, and that the refresh-token gap — real in
  the code but only implied by a `⚠️` marker in `requirements.md` — had no task or design bullet
  backing it. This version closes both gaps; it is a correction of the v1.0 retroactive write, not
  a new requirement.

## v1.0 (initial)

- `requirements.md`: initial version. The file previously carried four Vietnamese user stories
  (US-001..US-004) in "Given/When/Then" prose, moved here from
  `docs/specs/features/onboarding/onboarding.md` when the four-file spec convention was adopted.
  The acceptance criteria were rewritten in EARS notation and given the API contract each one maps
  to (status code + error code). The original user-story prose, field table and "Ghi chú kỹ thuật
  liên quan" notes are preserved verbatim under `## Original user stories`, so nothing was lost in
  the rewrite.
- `design.md`: initial version, written retroactively from the shipped code.
- `tasks.md`: initial version, written retroactively. Phases 1-5 checked off against files that
  were read directly; Phase 6 lists what is genuinely still open, including the middleware gap
  that makes US-004 unroutable today.
- Reason: the repo adopted a four-file spec convention (`requirements.md` / `design.md` /
  `tasks.md` / `changelog.md`) under `docs/specs/features/<feature>/` to match the per-folder
  README convention and to give `design.md` somewhere to record the transaction boundaries and
  hidden steps that a user story cannot express. `onboarding` is the first feature converted; the
  original single-file user story was renamed to `requirements.md` and extended rather than
  replaced.

Cross-file follow-ups recorded at this version:

- `docs/specs/api/openapi.yaml` — verified to cover all six routes; must be re-checked whenever a
  handler changes.
- `docs/specs/db/register-erd.md` — the ERD sourced from the original onboarding user story; update
  if migrations 0001-0004 change.
- `docs/knowledge/features/` — has no identity/onboarding concept yet (it covers only the six
  content features). If one is added later, `bdd-okf-docs` owns it; this spec does not write there.
