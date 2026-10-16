# Changelog — login Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this
folder. All four files share one changelog since they change together as one spec unit. Newest
version first.

## v2.6 (2026-10-16)

Fixes a contradiction found by the website spec review. No code change.

- `requirements.md`: AC4 said a missing `platform` was `400 INVALID_DEVICE`, while AC17 and `openapi.yaml` 2.1.1 say an empty platform is stored as `mobile`. The code agrees with AC17 (the handler defaults `platform` before the use case), so AC4 now names only `device` and `device_model`; the API table marks `platform` optional.

## v2.5 (2026-10-16)

**`openapi.yaml` khớp hợp đồng đăng nhập v2** (tasks Phase 7, item 1). Không đổi AC, không đổi code.

- `docs/specs/api/openapi.yaml`: `login` theo email (`INVALID_EMAIL`, message "Email hoặc mật khẩu không đúng", `LOGIN_LOCKED` chỉ có `locked_until`, `RATE_LIMITED`), `refresh` (+ `429`), `logout` (idempotent, không giới hạn IP), thêm `password-reset/request-otp` và `password-reset/confirm` (luôn `200` ở bước xin mã, thu hồi mọi phiên khi đổi). Tag `Auth` mô tả bốn nhóm giới hạn theo IP, `Cache-Control: no-store`, `X-InnerLamp-Client-IP`; `bearerAuth` mô tả kiểm phiên (`401` ngay request kế tiếp, `503` khi không kiểm được). Profile: `email_verified` không còn "luôn false".
- Code thắng spec ở hai chỗ (chưa sửa code): (1) `Cache-Control: no-store` được đặt cho **mọi** route `/v1/auth/*` có giới hạn IP (cả `request-otp`, `resend-otp`), spec AC18 chỉ liệt kê các route trả token; (2) `RATE_LIMITED` kèm header `Retry-After`, spec chỉ nêu `retry_after_seconds`. Ngoài ra `password-reset/confirm` có thể trả `VALIDATION_ERROR` (`otp_code` sai dạng), bảng API chưa liệt kê.
- Còn mở: `docs/knowledge/features/login.md` (il-okf-writer), product spec §4.2 (ba-writer).
- Product spec cites moved to login `product.md` @ v1.4 and onboarding `product.md` @ v1.6 (matrix changes and the `REGISTRATION_IN_PROGRESS` screen; no login AC affected).

## v2.4 (2026-10-16)

Triển khai Phase 1–6. Một quyết định đổi spec: **đếm giới hạn ở Redis** thay vì bảng Postgres `auth_rate_limits`.

- `requirements.md`: AC32 viết lại (store dùng chung, Redis); Open questions đảo "Đếm giới hạn ở đâu" (anh Đức chọn Redis 2026-10-16); Pending không còn AC nào.
- `design.md`: bảng Layers đổi hàng rate-limit (dùng `ClientIPRateLimit` có sẵn, `BudgetedAuthMailer` bọc `domain.AuthMailer`); migration là `0038_login_attempts_email.sql`, bỏ `auth_rate_limits`; `ResetLoginAttempts` không có method riêng (xoá trong `CreateLoginSession`); mục "Giới hạn tần suất" viết lại; Known gaps thêm: hết ngân sách thư lộ nhánh có/không tài khoản ở request-otp.
- `flow.md`: chỉ bump version và Status (Step không đổi).
- `tasks.md`: Phase 1–6 tick; item `auth_rate_limits`/`postgres_rate_limit_store.go` đánh dấu bỏ.
- Cross-file follow-ups: `docs/knowledge/features/login.md` (il-okf-writer, Phase 7); `openapi.yaml`; onboarding `design.md` "Migration 0037" bước 5 → `0038`, bước 7 bỏ (đã sửa ở migrations/README).
- Reason: anh Đức chọn Redis khi tới cổng Phase 5.

## v2.3 (2026-10-16)

Theo sau onboarding v8.1. Không đổi AC.

- `tasks.md`: Phase 1 thêm item migration `login_attempts` theo email (bước 5 của "Migration 0037", tách khỏi `0037` vì login đang chạy vẫn khoá theo SĐT).
- `requirements.md` / `design.md` / `flow.md`: no change needed — chỉ bump version.
- Code login hiện tại (SĐT) đã so mật khẩu qua `domain.PasswordHasher` (cùng hash với đăng ký v8.1); item `login_usecase.go` ở Phase 1 chưa tick vì phần email chưa làm.
- Mở: giới hạn theo IP (Phase 5) cần chốt Redis (`fc31897`) hay bảng `auth_rate_limits` trên Postgres.

## v2.2 (2026-10-14)

Giới hạn tần suất theo IP của khách website (website gọi backend từ server SvelteKit, nên trước đây mọi khách web chung một IP).

- `requirements.md`: AC34 viết lại — thứ tự lấy IP khách: `X-InnerLamp-Client-IP` khi `X-API-Key` hợp lệ → `X-Forwarded-For` (trusted proxy) → kết nối; cùng IP cho metadata thiết bị của phiên. Thêm vào Open questions (resolved).
- `design.md`: dòng `auth_rate_limit.go` nêu `clientIPFor` và thứ tự middleware (sau `RequireWebAPIKey`).
- `flow.md`: không đổi nội dung (không có bước mới), chỉ bump version.
- `tasks.md`: thêm task `clientIPFor` + test, vẫn `[ ]`.
- Hand-offs: website login v2.0 AC19 gửi `X-InnerLamp-Client-IP`. `openapi.yaml`: thêm header này khi handler chạy thật.
- Reason: rà spec website (2026-10-14), anh Đức đồng ý.

## v2.1 (2026-10-14)

Chốt câu hỏi product còn mở: đặt lại mật khẩu thành công thì **gỡ khoá đăng nhập** của email đó.

- `requirements.md`: AC28 thêm xoá `login_attempts`; bỏ marker `⚠️ Not confirmed (product)`; header trích product login v1.2; ghi vào Open questions (resolved).
- `design.md`: Step C-4 thêm một câu lệnh trong cùng transaction; bỏ marker.
- `flow.md`: sequence "Đặt mật khẩu mới" thay Note ⚠️ bằng bước xoá `login_attempts`.
- `tasks.md`: Phase 4 thành task cụ thể, vẫn `[ ]`.
- Reason: anh Đức "Gỡ như em đề xuất" (2026-10-14); product login v1.2 BR-LGN-05.

## v2.0 (2026-10-14)

**Viết lại spec** theo ADR-0003 (Accepted): đăng nhập bằng **email**, khoá theo email, **gộp quên mật khẩu** (trước ở `password-reset/`, chưa từng có code) thành US-003 bằng mã qua email, thêm giới hạn tần suất cho mọi `/v1/auth/*` và kiểm phiên còn sống trên mọi route Bearer. Spec nay trích `docs/product/login/product.md` @ v1.1 bằng ID.

- `requirements.md`: viết lại; header `Product spec`/`Contract`/`Implements`; AC1–AC35 gắn `FR-LGN-*`/`BR-*`/`NFR-*`; AC6–AC7, AC9–AC17, AC19 giữ hành vi v1.13; bảng API (login `email`, hai route `password-reset/*`, `429 RATE_LIMITED`); một `⚠️ Not confirmed (product)` (gỡ khoá đăng nhập sau khi đặt lại mật khẩu).
- `design.md`: viết lại; `ReserveLoginAttempt` nguyên tử; middleware `activeSession` (đảo quyết định "stateless" 2026-09-21 vì product FR-LGN-009 cần đăng xuất ở request kế tiếp); `PasswordResetUsecase`; thời gian trả lời tối thiểu cho xin mã đặt lại; bảng `auth_rate_limits` + decorator ngân sách thư.
- `flow.md`: vẽ lại bốn sequence (Login, Kiểm phiên, Xin mã đặt lại, Đặt mật khẩu mới). Sơ đồ refresh/logout v1.13 bỏ khỏi file vì không đổi.
- `tasks.md`: thay bằng Phase 1–7 của v2.0, tất cả `[ ]`.
- Reason: anh Đức bỏ SMS (2026-10-14); product login v1.1. Các lựa chọn kỹ thuật là đề xuất của Claude, anh Đức đồng ý 2026-10-14, duyệt chi tiết qua PR.
- Cần theo sau: `docs/specs/api/openapi.yaml`, `docs/knowledge/features/login.md` (sau khi code xong); `SRS.md`, `test-coverage-report.md` và folder `password-reset/` bị xoá ở bước dọn tài liệu cũ.

## v1.13 (2026-10-10)

Hợp đồng response cấp token: thêm `expires_in` và header chống cache. **Có đổi code** — đã làm và verify trong cùng ngày (`tasks.md` Phase 7 toàn `[x]`).

- `requirements.md`: thêm AC20–AC22 (nhóm "Response cấp token"); bảng API contract thêm `expires_in` cho `login`, `refresh`; Out of scope ghi rõ không có `refresh_expires_in` và BE không đặt cookie. Phạm vi gồm cả `register/verify-otp` của `onboarding/` (không bump `onboarding/`; con trỏ ở đây, như v1.12).
- `design.md`: thêm §Cross-cutting: response cấp token (Step-by-step Flow 3 bước); Known gaps thêm mục BE không chỉ định nơi FE lưu token và CORS `*`.
- `flow.md`: thêm sequence diagram vẽ lại 1:1 ba bước.
- `tasks.md`: thêm Phase 7; sau khi code xong đánh `[x]` kèm bằng chứng test (unit handler, `go build`/`vet`, E2E `TestLoginFlow_E2E` AC20–AC22).
- Code: `token_response.go` (mới), `login_handler.go`, `register_handler.go`, `cmd/main.go`, `e2e/login_e2e_test.go`, `token_response_test.go`. `openapi.yaml` đã cập nhật cùng lúc.
- Quyết định (anh Đức, 2026-10-10, sau khi rà soát response): thêm `Cache-Control: no-store` và `expires_in`. `expires_in` là TTL access token tính bằng giây (cấu hình, không đọc từ `exp`); không thêm hạn refresh token (YAGNI). Header đặt cho cả response lỗi của ba route.
- Cross-file follow-ups còn lại: `docs/knowledge/features/login.md` và `register.md` cần `il-okf-writer` làm mới (chưa gọi — chỉ viết OKF khi được yêu cầu); `onboarding/requirements.md` mục `verify-otp` có thể cần con trỏ tới AC20–AC21 ở đây.
- Reason: FE web cần biết khi nào refresh mà không giải mã JWT, và response cấp token không được phép bị cache.

## v1.12 (2026-10-09)

Thêm cơ chế `X-API-Key` cho client web (SvelteKit) và quy ước `device.platform`. **Có đổi code** (đã làm trước, spec viết bổ sung ngay sau khi verify bằng unit test).

- `requirements.md`: thêm AC16–AC19 (nhóm "Web client"); bảng API contract thêm `INVALID_API_KEY` cho cả ba route. Phạm vi thực tế là cả sáu route `/v1/auth/*`, gồm ba route của `onboarding/` (không bump `onboarding/`; con trỏ ở đây).
- `design.md`: thêm §Cross-cutting: web `X-API-Key` (Step-by-step Flow 4 bước); Known gaps thêm hai mục (key không chặn caller bỏ header; IP/UA của thiết bị web).
- `flow.md`: thêm sơ đồ flowchart `X-API-Key` vẽ lại 1:1 bốn bước.
- `tasks.md`: thêm Phase 6, các item đã verify; còn một item `[ ]` (E2E).
- Quyết định (anh Đức, 2026-10-09): web gửi `X-API-Key` (secret dùng chung, giữ ở server SvelteKit); mobile không có key; `platform` vắng = `mobile`. Key chỉ bắt khi header có mặt (AC18) nên không chặn được kẻ bỏ header — ghi nợ `docs/tech-debt/auth/TD-001`.
- Key dev sinh ngẫu nhiên, lưu plaintext trong `config.dev.yaml` (nợ TD-001).
- Reason: client web cần xác định "website nào đang gọi" tách khỏi token của user.

## v1.11 (2026-09-29)

Thêm `flow.md` — file này còn thiếu từ khi spec được viết (rule 6 của skill `il-spec-writer`: mọi
"Step-by-step Flow" trong `design.md` bắt buộc có Mermaid diagram tương ứng trong `flow.md`).
**Không đổi code, không đổi AC.**

- `flow.md`: file mới. Ba sequence diagram Mermaid (Login, Refresh, Logout) vẽ lại 1:1 ba section
  "Step-by-step Flow" của `design.md`, gồm cả các sub-step `(a)`–`(f)` của Step 6 (Login) và Step 3
  (Refresh), nhánh reuse Step 2b, và `FindUserByID` (bước đọc thêm trong `RotateSession`).
- `design.md`: §Request flow — bản ASCII cũ thay bằng con trỏ sang `flow.md`, tránh giữ song song
  hai bản vẽ cùng một thứ (nguồn sự thật vẫn là §Step-by-step Flow bằng chữ, không đổi).
- `requirements.md`: không đổi nội dung — bump version theo quy ước "cả bốn file cùng version".
- `tasks.md`: không đổi nội dung, không có task mới — `flow.md` chỉ vẽ lại flow đã implement và đã
  verify (Phase 0–5 done), không phải công việc chưa làm.
- Reason: người dùng phát hiện `docs/specs/features/login/` thiếu `flow.md` dù `design.md` có đủ ba
  Step-by-step Flow (điều kiện bắt buộc theo rule 5/6 của skill). Đây là hoàn thiện tài liệu, không
  phải thay đổi hành vi hệ thống.

## v1.10 (2026-09-22)

Sửa tài liệu cũ, **không đổi code, không đổi AC**.

- `design.md`: `repository.hashRefreshToken()` → **`domain.HashRefreshToken`**
  (`domain/session_token.go`) ở phần "tái dùng từ US-001", bảng Layers (Repository) và pseudocode
  `FindSessionByRefreshHash`; mục rủi ro "bất đồng TTL refresh token" đóng lại (onboarding v5.2 đã
  sửa dòng 30 ngày).
- `requirements.md`: xác nhận cả 3 path đã có trong `openapi.yaml` v3.7 (bỏ câu "chưa có"); dòng
  follow-up "refresh token 30 ngày" → đã sửa ở onboarding v5.2.
- `tasks.md`: bỏ "khoá ở lần thứ 6" (khoá ghi nhận ở lần sai thứ 5; lần thứ 6 trả 429); mục curl
  end-to-end giữ log `204` nhưng chú thích rõ từ v1.9 logout trả `200` `{"success": true}`; item
  handler `Logout` ghi `200` cho nhánh verified.
- Dọn tham chiếu `docs/ARCHITECTURE.md` (đã xoá): câu "gap của US-004" cho `/v1/users/me` trong
  `requirements.md` đổi thành "chưa được triển khai".

## v1.9 (2026-09-21)

Theo `api-response-envelope` v1.1 (AC11): logout thành công đổi `204 No Content` → `200` `{"success": true}`.
**Đổi code và AC13/AC14** (chỉ phần status/body; logic revoke, idempotent, 401 không đổi).

- `requirements.md`: AC13, AC14 và bảng API contract: `204` → `200` `{"success": true}`.
- `design.md`: Logout flow Step 3 — hai nhánh success trả `200` `{"success": true}`.
- `tasks.md`: mục handler `Logout` ghi chú đã đổi ở v1.9. Mục curl end-to-end (ghi `204`) giữ nguyên làm lịch sử
  verify ngày đó — chưa chạy lại binary với `200`.
- Code: `delivery/http/login_handler.go` (`utils.WriteSuccessResponse(w, nil)`), `login_handler_test.go`.
- `docs/specs/api/openapi.yaml` v3.7.
- `tasks.md` Deferred: job dọn `login_attempts`/session — anh chốt chấp nhận ở MVP.
- `tasks.md` Phase 5 tick audit `golang-e2e-verification` (anh chốt thêm `testcontainers-go`); Status → done.
  `requirements.md`/`design.md` Status cập nhật. Report mới `test-coverage-report.md` (audit log, ngoài cascade).
- Cùng ngày, đồng bộ trạng thái: `requirements.md` §còn mở bỏ mục `go test -short` (đã xanh) và ghi concept OKF
  `docs/knowledge/features/login.md` đã viết (`draft`); `tasks.md` Status tương tự. Curl lại toàn bộ flow trên binary
  thật (login, refresh rotate/reuse, supersede, lock, logout `200` ×2) — khớp AC.

## v1.8 (2026-09-21)

Đồng bộ với spec mới `api-response-envelope` v1.0. **Không đổi code login, không đổi AC.**

- `requirements.md`: §Auth model — bỏ câu "mặc định của middleware trả `{"error":"..."}` phẳng" (không còn đúng);
  ghi lý do vẫn giữ `ErrorHandler` riêng: message luôn `unauthorized`, không lộ lý do.
- `design.md`: mục rủi ro "401 của middleware phải khớp envelope" — ghi rõ tình trạng cũ/mới.
- `tasks.md`: không đổi nội dung, đã kiểm. Chỉ bump header.
- Reason: `pkg/jwt/middleware` mặc định giờ dùng envelope chuẩn; `cmd/main.go` bọc `Recovery` + `WithAPIFallback`
  nên 404/405/panic của mọi route (gồm 3 route login) cũng trả envelope — xem `../api-response-envelope/`.

## v1.7 (2026-09-21)

Dọn hai câu còn ghi envelope cũ `{error:{code,message}}`. **Không đổi code, không đổi hành vi**, AC1–AC15 giữ nguyên.

- `requirements.md`: §Auth model — 401 của middleware có cùng envelope `utils.ApiResponse`
  (`{success, error{code,message,type}, details}`) với các route khác, thay cho `{error:{code,message}}`.
- `design.md`: mục rủi ro "401 của middleware phải khớp envelope" — sửa cùng chỗ đó. Dòng `errors.go` ở bảng
  Layers (đã ghi đúng từ v1.5) không đổi.
- `tasks.md`: Phase 5 mục `openapi.yaml` — ghi chú `ErrorResponse` đổi từ "chưa sửa, ngoài phạm vi" sang đã sửa.
  Không tick/untick mục nào. Mục Phase 4 `errors.go` đã tự sửa ở v1.5, không đổi.
- Reason: v1.5 phát hiện envelope thật khác spec nhưng chỉ sửa một phần; phần US-001 (`openapi.yaml` v3.5,
  `onboarding/` v4.3) vừa được sửa nên đồng bộ nốt ở đây.

## v1.6 (2026-09-21)

- `tasks.md`: tick Phase 5 mục `go build` / `go vet` / `go test -short ./...` / 2 script kiến trúc —
  `go test -short -count=1 ./...` giờ xanh toàn repo. Cập nhật ghi chú Phase 4 handler test: case "hai
  object liền nhau" do `pkg/utils` đảm bảo; thêm test 413 cho `Login`/`Refresh`. Status bỏ mục
  `pkg/utils`. Còn mở ở Phase 5: audit AC bằng `golang-e2e-verification` và concept OKF.
- `requirements.md` / `design.md`: không đổi nội dung, đã kiểm. Chỉ bump header.
- `docs/specs/api/openapi.yaml` (v3.4): `/auth/login` và `/auth/refresh` có thêm `413 PAYLOAD_TOO_LARGE`;
  mô tả `400 VALIDATION_ERROR` ghi thêm rác sau JSON và Content-Type sai.
- Reason: mục "Phát hiện ngoài phạm vi (chưa sửa)" của v1.5 về 3 test `pkg/utils` **đã được sửa** ở spec
  riêng `docs/specs/features/json-body-hardening/` (v1.2). Hệ quả thật nêu ở v1.5 — handler chấp nhận
  object thứ hai và Content-Type sai — không còn.

## v1.5 (2026-09-21)

Implement Phase 3–5: repository, delivery, wiring, E2E, OpenAPI. Không đổi hành vi so với spec — chỉ sửa
các chỗ spec mô tả sai thực tế lúc code.

- `tasks.md`: tick Phase 3, Phase 4 toàn bộ, Phase 5 (curl E2E, OpenAPI, ghi OKF
  follow-up). **Còn hai mục Phase 5 chưa tick:** `go test -short ./...` sạch (3 test `pkg/utils` đỏ có sẵn)
  và audit AC bằng `golang-e2e-verification` (cần `testcontainers-go`).
- `design.md`: (1) `LoginRepository` là **tám** method — thêm `FindUserByID` (`RotateSession` cần email cho
  access token); (2) `writeAuthError` → **`WriteAuthError`** xuất khẩu (`cmd/main.go` ở package khác);
  (3) envelope 401/lỗi thật là `utils.ApiResponse` `{success, error{code,message,type}, details}`, không
  phải `{error:{code,message}}`; (4) `dummyPasswordHash` là biến package-level, không phải `func init()`;
  (5) Login Step 7 ghi đúng access token mang `extra{"sid"}`; (6) bảng "Deliberately stubbed" viết lại
  (code, migration, OpenAPI đã có; ERD và test E2E tự động chưa).
- `requirements.md`: chỉ đổi Status + Pending. AC1–AC15 không đổi.
- Reason: hoàn thành implement theo `tasks.md`.

### Quyết định khi implement (không có trong spec)

- `INVALID_DEVICE` trả `details: {"field": "device"}` — usecase chỉ trả sentinel, không nói thiếu
  `platform` hay `device_model`; AC5 nói `{field}` nên chọn tên chung.
- Reuse detection: `FindSessionByRefreshHash` trả `ErrRefreshTokenInvalid` cho session đã revoked/hết hạn
  **kể cả khi** khớp `previous_refresh_token_hash` — không revoke lần hai, không ghi đè `revoked_reason`.
- `clientIP` lấy từ `RemoteAddr`, không đọc `X-Forwarded-For` (header client tự đặt được); giá trị chỉ dùng
  làm metadata thiết bị.
- `Logout` mà `user_id` claim không phải số cũng → 401 `UNAUTHORIZED` (không 500).

### Phát hiện ngoài phạm vi (chưa sửa)

- `pkg/utils` — 3 test đỏ có sẵn: `TestDecodeJSONBody_UnknownField_Disallowed` (`DecodeError.Field` rỗng),
  `..._MultipleJSONObjects` (body `{} {}` được chấp nhận), `..._WrongContentType` (không kiểm Content-Type).
  Hệ quả thật: mọi handler chấp nhận object thứ hai và Content-Type sai.
- `docs/specs/api/openapi.yaml` — schema `ErrorResponse` của US-001 mô tả sai envelope thật (lồng `details`
  trong `error`, thiếu `success`/`type`); 3 path login dùng `ApiErrorResponse` mới, đúng thực tế.
  → **Đã sửa 2026-09-21** ở `onboarding/` v4.3 + `openapi.yaml` v3.5 (xoá `ErrorResponse`, mọi route dùng
  `ApiErrorResponse`; response thành công US-001 bọc `{success, data}`).

### Cross-file follow-ups

- `docs/specs/db/` — vẫn rỗng: **schema `0002` (`user_devices`, `login_attempts`, cột mới của
  `user_sessions`) chưa có tài liệu ERD**; nguồn duy nhất là `migrations/0002_*.sql` + `migrations/README.md`.
- `docs/knowledge/features/` — concept OKF cần `il-okf-writer` tạo **sau khi audit AC xong**: `login`
  (feature, gồm session + device + brute-force lock), endpoint `POST /v1/auth/login`, `.../refresh`,
  `.../logout`, entity `Session`, migration `0002`. `status: draft`.
- `docs/specs/api/openapi.yaml` — v3.3, đã thêm 3 path + `bearerAuth`.
- `docs/specs/features/onboarding/` — không đổi.

## v1.4 (2026-09-21)

`jwt-token-uniqueness` implement xong; mở khoá một nửa Phase 0.

- `tasks.md`: Phase 0 mục `jwt-token-uniqueness` tick `[x]` (verified: `go test ./pkg/jwt/...`, 7 test `jti`).
  Phase 0 vẫn còn mục `onboarding` Phase 7 → Phase 2–3 vẫn chưa được bắt đầu.
- `requirements.md` / `design.md`: chỉ đổi Status + bỏ chữ "bị chặn bởi jwt". **Không đổi nội dung kỹ thuật**
  (đã kiểm tra: login vốn dùng lại `TokenIssuer`, không phụ thuộc `extra` để chống trùng token).
- Reason: hoàn thành bước 1 của thứ tự implement.

## v1.3 (2026-09-21)

Chốt câu cuối: `logout` xác định session bằng **access token trong header `Authorization: Bearer` + claim
`sid`** (phương án B). Không còn `⚠️ Not confirmed` nào.

- `requirements.md`: AC8/AC9 thêm "access token mang claim `sid`"; AC12 nối với việc sửa US-001; AC13–AC14
  viết lại theo access token (tra theo cả `user_id` lẫn `session_id`); **thêm AC15** (401 khi thiếu/sai/hết
  hạn/refresh token/không có `sid`); bảng API contract: logout là `RequireAuth`, không có body; Auth model
  và Pending viết lại; thêm bullet về claim `sid` và middleware (`RequireTokenType = "access"`, `ErrorHandler`
  riêng vì mặc định trả `{"error":"..."}` phẳng). Marker logout chuyển sang Open questions (resolved).
- `design.md`: `TokenIssuer` đổi chữ ký (`func(userID, sessionID)`) thay vì thêm type mới; `LogoutInput`
  thành `{UserID, SessionID}`; Logout flow viết lại (3 bước: middleware → lấy `sid` từ claims → revoke),
  bỏ `FindSessionByRefreshHash` khỏi logout; `RevokeSession` đổi khoá thành `(user_id, session_id)`; Login
  Step 6 sinh `sessionID = uuid.NewString()` trước khi mint; thêm mục "Phụ thuộc: sửa US-001"; Known gaps
  thêm hai mục (logout cần access token còn hạn; envelope 401 của middleware). Số bước của Login/Refresh
  **không đổi**; Logout đổi cấu trúc (2 → 3 bước).
- `tasks.md`: Phase 0 thêm điều kiện tiên quyết (onboarding Phase 7); các mục `TokenIssuer`, `Logout`,
  `RevokeSession`, handler, wiring, e2e cập nhật. Không tick task nào.
- Reason: anh chọn B — đúng chuẩn, refresh token chỉ lộ ở `/refresh`, và middleware cần được wire cho
  US-004 dù sao. Đánh đổi có ý thức: sửa code đã ship của US-001.

### Cross-file follow-ups (bổ sung)

- `docs/specs/features/onboarding/` — v4.1: design + tasks (Phase 7) mô tả việc sửa `TokenIssuer` /
  `CreateUserAndSession`; requirements không đổi nội dung (hành vi ngoài của US-001 giữ nguyên).
- `docs/specs/api/openapi.yaml` — logout ghi `security: bearerAuth`, không request body, response `204` /
  `401`.

## v1.2 (2026-09-21)

Chốt bốn trong năm điểm mở sau phỏng vấn vòng hai; còn lại **logout** (đang được thảo luận).

- `requirements.md`: AC7 thêm điều kiện "liên tiếp = mỗi lần sai cách lần trước ≤ 15 phút"; bốn marker
  (path/shape, sliding, access token sau logout, cửa sổ đếm) chuyển sang "Open questions (resolved)"; ba
  quyết định ghi vào Ghi chú kỹ thuật. Marker logout viết lại với ba phương án (A body, A′ header
  refresh, B access token + `sid`).
- `design.md`: Login Step 5 và Refresh Step 3 chuyển từ `⚠️ Not confirmed` sang `Confirmed`; Known gaps
  ghi access token sau logout là chấp nhận; Logout Step 1 ghi hệ quả của phương án B. **Hình dạng các
  flow không đổi** (đã kiểm tra: cả bốn quyết định trùng mặc định đề xuất).
- `tasks.md`: bỏ ghi chú "bị chặn" ở 4 mục (usecase test, migration, `RotateSession`, config); vẫn
  chặn mục `Logout`. Không tick task nào.
- Reason: anh đồng ý câu 2–5.

## v1.1 (2026-09-20)

Tách lỗi JWT ra spec riêng theo yêu cầu của anh; login không tự vá bằng `extra` nữa.

- `requirements.md`: bullet "Refresh token phải duy nhất" viết lại thành phụ thuộc cứng vào
  `../jwt-token-uniqueness/`; header Status ghi bị chặn. Không đổi AC nào.
- `design.md`: bỏ `SessionTokenIssuer` và mục "Refresh token uniqueness"; thay bằng mục "Phụ thuộc:
  `jwt-token-uniqueness`". Login/Refresh dùng lại `domain.TokenIssuer` (`issuer(userID)`), bỏ mọi chỗ
  nhắc `sid`/`ver`. Hình dạng các Step-by-step Flow **không đổi** (chỉ đổi tham số issuer).
- `tasks.md`: thêm **Phase 0** (điều kiện tiên quyết: fix `pkg/jwt` xong); mục `SessionTokenIssuer`
  bị gạch bỏ; các mục dùng nó đổi sang `TokenIssuer`. Không tick task nào.
- Reason: `pkg/jwt` là package dùng chung, lỗi không thuộc riêng login; sửa ở gốc (`jti`) đúng hơn và
  giúp cả access token.

## v1.0 (initial — 2026-09-20)

Spec đầu tiên của folder `login/`, viết ở chế độ **prospective** (chưa có code login). US-002 được
**chuyển từ** `docs/specs/features/onboarding/requirements.md` (v3.3, mức user story) và viết lại đủ
bộ ba file; folder `onboarding/` bump lên v4.0 để bỏ US-002 (xem `../onboarding/changelog.md`).

- `requirements.md`: US-002 viết lại từ "đăng nhập bằng email + mật khẩu" thành **SĐT + mật khẩu**
  (khớp US-001: định danh chính là SĐT đã verify OTP; email chưa verify bị gate). 14 AC theo EARS
  cho ba endpoint `login` / `refresh` / `logout`. `refresh` và `logout` **được đưa vào scope** (US-001
  đã đẩy chúng ra ngoài). Đổi so với US-002 cũ: bỏ lỗi "Email hoặc mật khẩu không đúng" → "Số điện
  thoại hoặc mật khẩu không đúng"; AC "tạm khoá sau nhiều lần sai" cụ thể hoá thành 5 lần / 15 phút
  lưu ở DB; thêm AC thiết bị (upsert + revoke session cũ của device), reuse-detection, logout
  idempotent, tương thích session US-001 (không device).
- `design.md`: 3 Step-by-step Flow (Login 7 bước, Refresh 3 bước + 2b, Logout 2 bước); bảng Layers;
  schema `0002` (`user_devices`, `login_attempts`, `user_sessions.device_id` +
  `previous_refresh_token_hash`, partial unique index); mục "Refresh token uniqueness".
- `tasks.md`: 5 Phase, mọi item `[ ]`; item bị chặn bởi marker được ghi rõ tại chỗ.
- Reason: yêu cầu viết spec US-002 và phỏng vấn edge case; US-002 cũ đã lệch với US-001.

### Quyết định từ phỏng vấn (2026-09-20)

| Câu hỏi | Trả lời | Ảnh hưởng |
|---|---|---|
| Định danh đăng nhập | SĐT + mật khẩu | Bỏ email khỏi login; thoả gate US-001 AC13 mặc nhiên |
| `refresh` trong US-002? | Có | Thêm endpoint + rotate; khép AC11 vế 2 của US-001 |
| Device | Client gửi, `platform` + `device_model` bắt buộc | Thêm `user_devices` + `user_sessions.device_id` (migration `0002`) |
| Login lặp cùng device | Upsert device, revoke session cũ của device đó | AC4; partial unique index |
| Rate limit | Bảng DB theo SĐT, 5 lần / 15 phút | Bảng `login_attempts`; không dùng `pkg/ratelimit` in-memory |
| Reuse token đã rotate | 401 + revoke session của device đó | AC11; `previous_refresh_token_hash` |
| Refresh TTL | 7 ngày | Khớp config/`0001`; dòng "30 ngày" của US-001 thành follow-up |
| Logout | Có `POST /v1/auth/logout` | AC13–AC14 |
| Vị trí spec | Folder `login/` riêng | Chuyển US-002 khỏi `onboarding/` |

### Phát hiện khi đọc code (không phải câu hỏi cho user)

- `pkg/jwt.GenerateRefreshToken` không sinh `jti`, `iat` độ chính xác 1 giây → hai refresh token cùng
  user trong cùng một giây giống hệt nhau, vỡ `UNIQUE(refresh_token_hash)` của `0001`. Design chốt
  nhét `sid` + `ver` vào `extra` (không sửa `pkg/jwt`). **Đã bị thay ở v1.1**: lỗi được tách sang `../jwt-token-uniqueness/` và sửa ở gốc bằng `jti`.
- `domain/errors.go` hiện **chưa có** `ErrUserNotFound`; login cần một sentinel nội bộ.

### Còn mở (`⚠️ Not confirmed`)

1. *(đã chốt ở v1.2 — path/shape được duyệt.)*
2–5. *(đã chốt ở v1.2 và v1.3 — logout bằng access token + `sid`; sau logout access token stateless; cửa sổ đếm 15 phút; refresh sliding.)*

Chưa hỏi lại user lần thứ hai theo quy tắc "hỏi một lần, rồi tiếp tục với marker mở"; mặc định đề xuất
của từng mục ghi cạnh marker.

### Cross-file follow-ups

- `docs/specs/api/openapi.yaml` — thêm `POST /v1/auth/login`, `/refresh`, `/logout` **sau khi** handler chạy thật.
- `migrations/` — tạo `0002_login_devices_and_rate_limit.sql`; cập nhật `migrations/README.md`.
- `docs/specs/db/` — đang rỗng; ERD cho `user_devices` / `login_attempts` / cột mới của `user_sessions` chưa có nơi ghi.
- `docs/knowledge/features/` — cần concept mới về login/session; chỉ tạo bởi `il-okf-writer` **sau** khi code implement và verify.
- `docs/specs/features/onboarding/requirements.md` — dòng "refresh token sống 30 ngày" (§Ghi chú kỹ thuật) lệch config 168h; **chưa sửa**, cần anh cho phép.
- `docs/ARCHITECTURE.md` §4 — nếu sau này chọn phương án logout dùng `RequireAuth`, thêm vào danh sách gap auth.
