# Requirements — attestation

Version: 1.4
Last updated: 2026-09-25
Status: implemented — đủ clean-arch layers + cả 3 driver (mock, playintegrity, appattest); fraud-risk receipt + gate nghiệp vụ deferred
Module: `attestation`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này được viết ở chế độ **Prospective** — chưa có dòng code nào cho feature `attestation`; đây là
work order để build. Mọi route mô tả dưới đây đều **chưa tồn tại** trong
`services/appService/cmd/main.go` và `docs/specs/api/openapi.yaml` (xem "Pending").

## User Story

> Là một **client app InnerLamp (Android / iOS)**,
> Tôi muốn **chứng minh với backend rằng tôi là bản app thật, chạy trên thiết bị thật, chưa bị
> sửa đổi (tampered) hay chạy trong môi trường giả lập**,
> Để **backend có thể tin cậy và cho phép tôi truy cập các tính năng nhạy cảm, chống gian lận,
> bot và app giả mạo**.

Feature này gồm hai phần:

1. **Verify service** (`challenge` + `verify`): xác minh bằng chứng attestation rồi trả **verdict**
   — cho Android (Play Integrity) lẫn iOS (App Attest, pha `attest` đăng ký khoá).
2. **Assertion (iOS)**: cấp nonce assertion + middleware `RequireAttestation` để verify assertion
   gắn vào **từng request nhạy cảm** (request binding + chống replay) — đây là phần "gate" thật sự,
   không phải trả verdict.

Quyết định *gate feature nào* (route nào cần `RequireAttestation`, xử lý fail ra sao) vẫn thuộc client
/ một feature tương lai — feature này chỉ cung cấp cơ chế (xem "Out of scope").

## Acceptance Criteria (EARS notation)

### Cấp challenge — attestation

1. WHEN a `POST /v1/attestation/challenge` request arrives with a valid `platform` (`android` |
   `ios`), THE SYSTEM SHALL generate a cryptographically-random nonce of at least 16 bytes, persist a
   challenge row (nonce, platform, `purpose = 'attest'`, `expires_at = now() + challenge_ttl`), and
   return `200` with `{challenge, expires_in_seconds}`.

2. WHEN the challenge request body is malformed, carries an unknown field, or has a `platform` other
   than `android`/`ios`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` and SHALL NOT persist a
   challenge.

### Cấp challenge — assertion (iOS)

3. WHEN a `POST /v1/assertion/challenge` request arrives with `platform=ios`, THE SYSTEM SHALL
   generate a nonce (≥ 16 bytes), persist a challenge row (`purpose = 'assert'`), and return `200`
   with `{challenge, expires_in_seconds}`.

4. WHEN the assertion challenge body is malformed, carries an unknown field, or has a `platform`
   other than `ios`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` and SHALL NOT persist a
   challenge.

### Verify Android (Play Integrity)

5. WHEN a `POST /v1/attestation/verify` request with `platform=android`, a valid `token`, and a known,
   unused, unexpired challenge (nonce) arrives, THE SYSTEM SHALL decode the integrity token on Google's
   servers (`playintegrity.googleapis.com/v1/{packageName}:decodeIntegrityToken`), then either:
   - on success (the decoded `requestPackageName` matches the configured package, the `nonce` in the
     decoded token matches the challenge the caller presented, `timestampMillis` is within the allowed
     freshness window, and
     `appRecognitionVerdict == PLAY_RECOGNIZED`): return `200` with `attested: true` plus the Android
     signals, and SHALL mark the challenge used;
   - on failure (any of those checks fail): return `200` with `attested: false` plus a `reason`, and
     SHALL mark the challenge used.

6. WHEN a verify request references a challenge (nonce) that is unknown, expired, or already used, THE
   SYSTEM SHALL return `400 ATTESTATION_CHALLENGE_INVALID` (one code for all three, so the caller
   cannot probe which state) and SHALL NOT call Google.

7. WHEN Google's decode service is unreachable or times out, THE SYSTEM SHALL return
   `502 ATTESTATION_UPSTREAM_ERROR` and SHALL NOT mark the challenge used (so the caller may retry
   the same challenge).

### Verify iOS (App Attest) — pha `attest` (đăng ký khoá, 1 lần)

8. WHEN a `POST /v1/attestation/verify` request with `platform=ios`, a valid `key_id` and
   `attestation` object, and a known, unused, unexpired challenge (nonce) arrives, THE SYSTEM SHALL
   verify the attestation object (CBOR decode; `x5c` chain against Apple's App Attest root
   certificate; `nonce == SHA256(authData || SHA256(challenge))`; RP ID hash == `SHA256(app_id)`;
   `counter == 0`; `aaguid` environment; public-key hash == `key_id`), then either:
   - on success: return `200` with `attested: true`, persist the key (`key_id`, public key, receipt,
     counter = 0, environment), and mark the challenge used;
   - on failure: return `200` with `attested: false` plus a `reason`, mark the challenge used, and
     SHALL NOT persist a key.

9. WHEN an iOS attestation object fails any verification step (invalid cert chain, nonce mismatch,
   `counter != 0`, wrong RP ID, or a public-key hash that does not equal `key_id`), THE SYSTEM SHALL
   return `200` with `attested: false` plus a `reason`, and SHALL NOT persist a key.

10. WHEN a `key_id` that is already registered is attested again (an idempotent retry, e.g. after a
    transient failure), THE SYSTEM SHALL update the existing key row (upsert by `key_id`) rather than
    return a duplicate error, and return `200` with `attested: true`.

### Assertion iOS (App Attest) — middleware `RequireAttestation` (verify từng request)

> Không phải endpoint — là middleware bọc route nhạy cảm. Client gắn assertion vào header của chính
> request thật; server verify inline và cho đi tiếp hoặc từ chối.

11. WHEN a request protected by `RequireAttestation` arrives with headers `X-Attestation-Key-Id`,
    `X-Attestation-Assertion`, `X-Attestation-Challenge`, THE SYSTEM SHALL:
    (a) look up the challenge (nonce, `purpose = 'assert'`) — must be unused and unexpired;
    (b) look up the key by `key_id` — must exist;
    (c) rebuild `clientData = challenge || method || path || SHA256(body)` from the actual request;
    (d) verify the ES256 assertion signature over `authenticatorData || SHA256(clientData)` using the
        stored public key;
    (e) verify `SHA256(app_id) == RP ID hash` in `authenticatorData`;
    (f) verify `counter > stored counter`;
    then, on success: update the stored counter, mark the challenge used, and pass the request to the
    next handler; on failure: reject with `401` and SHALL NOT update the counter.

12. WHEN the assertion's counter is not strictly greater than the stored counter (replay or rollback),
    THE SYSTEM SHALL reject with `401 ATTESTATION_FAILED` (`reason = counter_not_increased`), and
    SHALL NOT update the stored counter.

13. WHEN the `key_id` is unknown, THE SYSTEM SHALL reject with `401 ATTESTATION_FAILED`
    (`reason = unknown_key`).

14. WHEN the assertion challenge is unknown, expired, or already used, THE SYSTEM SHALL reject with
    `401 ATTESTATION_CHALLENGE_INVALID` (one code for all three, so the caller cannot probe which
    state).

### Chung

15. WHEN a verify request body is malformed, carries an unknown field, or has an invalid `platform`,
    THE SYSTEM SHALL return `400 VALIDATION_ERROR` without calling Google or Apple.

16. WHEN `platform=android` but `token` is missing, or `platform=ios` but `key_id`/`attestation` is
    missing, THE SYSTEM SHALL return `400 VALIDATION_ERROR`.

17. WHEN the database is unavailable while issuing or verifying a challenge, THE SYSTEM SHALL return
    `500 INTERNAL_ERROR`.

## API contract (confirmed surface for this feature)

> Chưa có trong `docs/specs/api/openapi.yaml` — đây là hợp đồng dự kiến; cột mã lỗi là cam kết thiết
> kế, sẽ được đưa vào openapi **sau** khi handler chạy thật qua `curl` (đúng convention repo, xem
> "Ghi chú kỹ thuật liên quan").

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| POST | `/v1/attestation/challenge` | public | 200 `{challenge, expires_in_seconds}` | 400 VALIDATION_ERROR; 413 PAYLOAD_TOO_LARGE; 500 INTERNAL_ERROR |
| POST | `/v1/attestation/verify` | public | 200 `{attested, platform, signals?, reason?}` | 400 VALIDATION_ERROR / ATTESTATION_CHALLENGE_INVALID; 413 PAYLOAD_TOO_LARGE; 502 ATTESTATION_UPSTREAM_ERROR; 500 INTERNAL_ERROR |
| POST | `/v1/assertion/challenge` | public | 200 `{challenge, expires_in_seconds}` | 400 VALIDATION_ERROR; 413 PAYLOAD_TOO_LARGE; 500 INTERNAL_ERROR |
| (middleware) | `RequireAttestation` | assertion headers | pass-through (next handler) | 401 ATTESTATION_FAILED / ATTESTATION_CHALLENGE_INVALID |

Shape request `verify` (rẽ nhánh theo `platform`):

```jsonc
// Android
{ "platform": "android", "challenge": "<nonce>", "token": "<integrity token>" }

// iOS — đăng ký khoá (1 lần)
{ "platform": "ios", "challenge": "<nonce>", "key_id": "…", "attestation": "<base64>" }
```

Shape request `assertion/challenge`:

```jsonc
{ "platform": "ios" }
```

Assertion (không qua endpoint — gắn header vào request nhạy cảm được `RequireAttestation` bọc):

```text
X-Attestation-Key-Id:     <key_id base64>
X-Attestation-Assertion:  <assertion base64>
X-Attestation-Challenge:  <nonce>
```

Với `clientData = challenge || method || path || SHA256(body)` — client tính `SHA256(clientData)`
trước khi gọi `generateAssertion`, server dựng lại `clientData` từ request thật để verify chữ ký.

Shape response `verify` (luôn 200, kể cả khi verdict âm tính — quyết định #10):

```jsonc
{ "attested": true,  "platform": "android", "signals": { "app_recognition_verdict": "PLAY_RECOGNIZED", "device_recognition_verdict": "MEETS_DEVICE_INTEGRITY", "app_licensing_verdict": "LICENSED" } }
{ "attested": false, "platform": "android", "reason": "app_not_recognized" }
```

## Out of scope (this feature boundary, not tracked here)

- **Gọi server-to-server tới Apple để lấy fraud-risk metric** (risk metric từ receipt) — **deferred**.
  Feature này vẫn **lưu receipt** khi attestation thành công để làm được bước đó về sau (quyết định #7).
- **Play Integrity giải mã local** (bằng decryption key từ Play Console) — ngoài scope, đã chọn Google
  server decode (quyết định #5).
- **Standard request (requestHash)** của Play Integrity — ngoài scope, đã chọn classic + nonce
  (quyết định #6).
- **DeviceCheck `DCDevice`** (2 bit per-device data, phục vụ promo/fraud flag khác) — không thuộc App
  Attest, ngoài scope.
- **Chính sách gate nghiệp vụ** (feature nào cần `RequireAttestation`, xử lý fail thế nào) — thuộc
  client / một feature tương lai; feature này chỉ cung cấp middleware, không tự gắn vào route hiện có.

## Pending (not yet implemented)

Đã build (v1.3): domain + usecase + repository + handler + middleware + migration `0004` +
`pkg/attestation` (interface + 3 driver: `mock`, `playintegrity`, `appattest`) + route trong
`cmd/main.go` + `openapi.yaml`.

Còn lại:

- **Bật driver thật ở config** — đổi `attestation.android.provider = playintegrity` /
  `attestation.ios.provider = appattest` khi app được publish; cần service-account JSON cho
  Play Integrity và `app_id` thật (`{team_id}.{bundle_id}`). Các giá trị này là cấu hình điền khi
  publish, không phải chỗ trống trong thiết kế (xem "Ghi chú kỹ thuật liên quan").

## Ghi chú kỹ thuật liên quan

- **Luồng tổng quát (server-side), tách từ tài liệu Google/Apple:**
  - *Cả hai nền tảng*: server cấp challenge (nonce ≥ 16 byte, dùng-một-lần, có TTL) → client lấy
    bằng chứng attestation gắn challenge đó → gửi về server → server verify và trả verdict.
  - *Android (Play Integrity, classic)*: client gọi `IntegrityManager.requestIntegrityToken(nonce)`
    nhận integrity token (JWE của JWS); server gọi `decodeIntegrityToken` trên Google server bằng
    service account (scope `playintegrity`), rồi kiểm `requestPackageName`, `nonce`,
    `timestampMillis`, `appRecognitionVerdict`, (tuỳ chọn) `deviceRecognitionVerdict`,
    `appLicensingVerdict`.
  - *iOS (App Attest)*: client `generateKey()` → `keyId`; `attestKey(keyId, SHA256(challenge))` →
    attestation object (CBOR `apple-appattest`, `x5c`, receipt, `authData`); server verify chuỗi
    cert, nonce, RP ID, counter=0, aaguid, hash public key = keyId, rồi lưu public key + receipt +
    counter=0. Về sau, mỗi request nhạy cảm: client gọi `POST /v1/assertion/challenge` lấy nonce,
    dựng `clientData = challenge || method || path || SHA256(body)`, gọi
    `generateAssertion(keyId, SHA256(clientData))`, gắn assertion vào header; middleware
    `RequireAttestation` verify chữ ký ES256 bằng public key đã lưu, kiểm counter tăng + request
    binding + challenge dùng-một-lần, chống replay.
- **Hai nền tảng bất đối xứng:** Android **không có khoá bền** — mỗi lần verify là một token mới gắn
  challenge; iOS **có khoá bền per-device** (`keyId`) — một lần `attest` (đăng ký) + nhiều lần
  `assert` (mỗi request). Vì vậy `attestation_keys` chỉ có ý nghĩa cho iOS; Android chỉ cần bảng
  challenge. **Verify khác bản chất:** Android = server gọi Google `decodeIntegrityToken`
  (server-to-server); iOS = server verify **local** (crypto), **không** gọi Apple.
- **Thư viện CBOR:** repo hiện **chưa có** thư viện CBOR/COSE (cần cho decode attestation/assertion
  object và parse COSE public key của iOS). Đây là **dependency bên thứ ba mới** (đề xuất
  `github.com/fxamacker/cbor/v2`) — xem `tasks.md` Phase 5.
- **Play Integrity SDK phía server:** `google.golang.org/api` (đã có trong `go.mod`) chứa
  `playintegrity/v1` với `DecodeIntegrityToken` — không cần dependency mới cho Android, chỉ cần
  service account credential (secret) và scope `playintegrity`.
- **Pattern provider-agnostic:** `pkg/attestation` theo đúng pattern `pkg/sms` — interface `Verifier`
  ở root + `Register(type, FactoryFunc)` / `New(config)`, driver con tự đăng ký qua `init()`. Driver
  `mock` (chỉ verify giả, luôn/không bao giờ pass theo config) dùng cho dev/test theo đúng tinh thần
  `pkg/sms/logsms` và các block `⚠️ DEV ONLY` trong `cmd/main.go`.
- **Verdict policy trong config:** ngưỡng verdict bắt buộc (hiện chỉ `PLAY_RECOGNIZED`, quyết định
  #11) để trong `attestation` config — không hardcode trong usecase — để sau nâng ngưỡng
  (`MEETS_DEVICE_INTEGRITY`, `LICENSED`, …) chỉ cần sửa config, không đổi code.
- **Request binding (iOS assertion):** `clientData = challenge || method || path || SHA256(body)`.
  Server **dựng lại** `clientData` từ request thật + challenge đã lưu (KHÔNG parse JSON do client tự
  nhồi) để hash lại và verify chữ ký — assertion bị ràng buộc vào đúng request (chống replay theo
  request, mạnh hơn nonce đơn thuần). Nếu client gửi `clientData` chứa challenge/method/path/body
  khác với request thật → chữ ký không khớp → bị từ chối.
- **Giá trị deployment chưa có (điền khi publish, không phải gap thiết kế):**
  - `attestation.android.package_name` — package name thật của app trên Google Play.
  - `attestation.ios.app_id` — `{team_id}.{bundle_id}` dùng tính RP ID hash = `SHA256(app_id)`.
  - Service account JSON (hoặc secret) cho `decodeIntegrityToken`.
  - Apple App Attest root certificate (nhúng hoặc tải về, dùng verify chuỗi `x5c`).
- **Envelope:** mọi response dùng `utils.ApiResponse` (`{success, data}` / `{success:false, error, details}`)
  như phần còn lại của service (xem `docs/specs/features/api-response-envelope/`). Verdict object nằm
  trong `data`. Middleware `RequireAttestation` khi từ chối cũng dùng envelope với status `401`.

## Open questions (resolved during implementation)

- ~~Feature này gate những endpoint nào?~~ → **Không gate gì ở verify service**; gate thật sự qua
  middleware `RequireAttestation` (quyết định #2, cập nhật v1.1). Resolved 2026-09-24.
- ~~iOS assert qua endpoint verify hay middleware?~~ → **Middleware `RequireAttestation`** (assertion
  gắn header vào request, verify inline) — thay cho `POST /verify {mode:assert}` ở v1.0. Resolved
  2026-09-24.
- ~~Android verify token bằng Google server hay local?~~ → **Google server decode** (quyết định #5).
  Resolved 2026-09-24.
- ~~iOS có làm fraud-risk receipt ngay không?~~ → **Defer, chỉ lưu receipt** (quyết định #7).
  Resolved 2026-09-24.
- ~~Lưu challenge/khoá ở đâu?~~ → **Postgres cả hai** (quyết định #8). Resolved 2026-09-24.
