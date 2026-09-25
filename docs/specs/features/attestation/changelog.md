# Changelog — attestation Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version first.

## v1.4 — thêm `flow.md` (Mermaid mirror của Step-by-step Flow) (2026-09-25)

Bổ sung `flow.md` còn thiếu — bản vẽ visual của 5 section "Step-by-step Flow" trong `design.md`.

- `flow.md`: thêm mới — 5 sequence diagram Mermaid mirror 1:1 năm section "Step-by-step Flow" của
  `design.md` (cấp challenge attest; cấp challenge assert; verify Android; verify iOS attest; assertion
  middleware `RequireAttestation`). Transaction boundary vẽ bằng `rect` (`SaveKey`+`MarkChallengeUsed`
  và `UpdateKeyCounter`+`MarkChallengeUsed`); bước crypto thuộc `pkg/attestation` đánh `Note` "ngoài
  scope feature này"; marker `⚠️ Not confirmed` (`ios.reject_development_keys`) vẽ thành `Note`, không
  vẽ như đã chốt.
- `design.md`: thay section "Request flow" (bản trace ASCII) bằng con trỏ sang `flow.md` — giữ một
  bản vẽ duy nhất, tránh drift (quy ước `flow_diagram.md`). Không đổi bất kỳ Step-by-step Flow nào.
- `requirements.md`: no change needed — chỉ bump version header (nội dung giữ nguyên).
- `tasks.md`: no change needed — chỉ bump version header (nội dung giữ nguyên).
- Reason: flow của attestation có nhiều lời gọi tuần tự ra ngoài (DB, Google) nên theo rule 6 của
  skill bắt buộc phải có `flow.md` mirror 1:1 các Step-by-step Flow — file này còn thiếu từ trước.

## v1.3 — implemented: real playintegrity + appattest drivers (2026-09-24)

Thêm hai driver thật vào `pkg/attestation`, hoàn tất Phase 5.

- **`pkg/attestation/playintegrity/`** — driver Android server-to-server: gọi
  `playintegrity/v1.DecodeIntegrityToken` bằng service account (scope `playintegrity`). Auth qua
  `credentials_json` / `credentials_file` (fallback ADC). `mapPayload` tách riêng để test không cần
  mạng. iOS → `ErrPlatformNotSupported`.
- **`pkg/attestation/appattest/`** — driver iOS local crypto: decode CBOR (attestation object +
  assertion), verify chuỗi `x5c` lên Apple App Attest Root (nhúng, override `root_cert_pem`),
  nonce = `SHA256(authData || SHA256(challenge))`, RP ID hash = `SHA256(app_id)`, `counter == 0`,
  aaguid → environment (`appattest`/`appattestdevelop`), hash public key == `key_id`, và verify chữ
  ký ES256 assertion over `authenticatorData || SHA256(clientData)`. Android → `ErrPlatformNotSupported`.
- **Dependency mới**: `github.com/fxamacker/cbor/v2` (CBOR) — đã `go mod tidy` + `go mod vendor`;
  `google.golang.org/api/playintegrity/v1` đã vendor (cùng module `api` sẵn có).

### Deviations from v1.2 (sửa đúng khi implement driver thật)

- **Port `VerifyIOSAssertion` nhận `publicKey []byte`, không phải `keyID string`** — keyID chỉ là
  hash của khoá, không đủ để verify chữ ký ES256. Usecase truyền `key.PublicKey` (đã có từ
  `FindKey`). Sửa ở `domain` port, `pkg/attestation` interface, adapter, mock và các test.
- **Config đổi từ một `attestation.provider` sang per-platform**: `attestation.android.provider`
  (`mock` | `playintegrity`) và `attestation.ios.provider` (`mock` | `appattest`) — hai backend bất
  đối xứng nên cần hai driver. Adapter `NewAttestationVerifier(android, ios)` nhận hai verifier.
- **`AttestationAndroidConfig` thêm `credentials_secret`** (tên secret service-account, resolve qua
  `SecretGetter` thành `Credentials`) — truyền vào driver qua `credentials_json` option.

### Cross-file follow-ups

- `docs/knowledge/features/` — concept `attestation` (mới) cần `bdd-okf-docs` tạo sau khi flow chạy
  end-to-end với driver thật (mock đã chạy; chưa viết knowledge bundle — thuộc `il-okf-writer`).
- `docs/specs/db/erd.mmd` đã thêm `attestation_challenges` + `attestation_keys` (v1.2); regenerate
  `erd.svg` qua `mermaid-cli` nếu cần.

## v1.2 — implemented: clean-arch layers + mock verifier (2026-09-24)

Build toàn bộ feature theo clean architecture + TDD, với `pkg/attestation` driver `mock` là impl
duy nhất (dev/test) — driver thật `playintegrity`/`appattest` deferred theo quyết định "mock trước".

- **Phase 1 — domain**: `attestation.go` (entity + enum + `AttestationRepository` port +
  `AttestationVerifier` port + input/output), sentinel trong `errors.go`, mock tay cho repo + verifier.
- **Phase 2 — usecase**: `attestation_usecase.go` (`RequestChallenge`, `Verify` rẽ nhánh android/ios,
  `VerifyAssertion`) + test table-driven phủ từng nhánh verdict.
- **Phase 3 — repository**: `migrations/0004_attestation.sql` (2 bảng + CHECK + index), `postgres_attestation_repository.go`
  (tx cho SaveKey/UpdateKeyCounter + challenge), adapter `attestation_verifier.go`, test Postgres thật.
- **Phase 4 — delivery**: `attestation_handler.go` (3 route), `attestation_middleware.go`
  (`RequireAttestation`), wire trong `cmd/main.go`, config `attestation:` + `openapi.yaml` (3 path +
  schema + header). Handler/middleware test đầy đủ.
- **Phase 5 — pkg/attestation**: interface + registry (`attestation.go`) + driver `mock` + test.
- **Phase 6 — integration**: `go build`/`vet`/`test -short` sạch, `check-architecture.sh` +
  `check_dependency_rule.sh` pass, E2E qua `curl` với Postgres thật (challenge → verify android/ios →
  re-use challenge bị 400).

### Deviations from the v1.1 shorthand (ghi để không drift)

- `AttestationRepository.SaveChallenge` nhận thêm `expiresAt` (usecase tính từ `ChallengeTTL`, như
  pattern OTP `IssueOTP(..., expiresAt)`). `SaveKey`/`UpdateKeyCounter` nhận thêm `challenge` để
  đóng challenge cùng MỘT tx (design.md flow iOS Step 4 / assertion Step 4) — giữ đúng 6 tên method.
- `AttestationConfig` thêm `android.required_app_recognition_verdict` (ngưỡng verdict config-driven,
  quyết định #11) bên cạnh `challenge_ttl` / `freshness_window` / `android.package_name` / `ios.app_id`.

### Cross-file follow-ups

- `docs/knowledge/features/` — concept `attestation` (mới) cần `bdd-okf-docs` tạo sau khi flow chạy
  end-to-end (đã chạy với mock; chưa viết knowledge bundle — thuộc `il-okf-writer`).
- `go.mod`/`vendor/` — dependency CBOR (`github.com/fxamacker/cbor/v2`) khi làm `pkg/attestation/appattest`.

## v1.1 — reconciliation với iOS client (2026-09-24)

Đối chiếu với design doc phía iOS client (`DeviceAttestation`, repo riêng), chốt lại mô hình
assertion cho khớp chuẩn App Attest (per-request):

- **Bỏ `POST /v1/attestation/verify {mode: assert}`** → thay bằng middleware **`RequireAttestation`**:
  assertion gắn vào header của request nhạy cảm, server verify inline và pass/`401` — thay vì "verify
  trả verdict" (verdict không ký, không gắn request thì không bảo vệ được gì).
- **Thêm `POST /v1/assertion/challenge`** — cấp nonce riêng cho assertion; cả `/attestation/challenge`
  lẫn `/assertion/challenge` đều là **`POST`** (mint nonce = side effect + không idempotent, không
  dùng `GET` như doc client).
- **Thêm request binding** — `clientData = challenge || method || path || SHA256(body)`; server dựng
  lại từ request thật (không parse JSON do client tự nhồi).
- **Bỏ trường `mode`** — `verify` giờ chỉ còn hai nhánh: Android (token) + iOS `attest` (attestation).
- **Schema** — thêm cột `purpose` (`attest` | `assert`) vào `attestation_challenges`.
- **Khẳng định lại:** iOS server verify **local** (không gọi Apple); sửa lại chữ "verifies against
  Apple" trong doc client.

Các quyết định v1.0 giữ nguyên: standalone verify service cho phần attestation; Android decode qua
Google server + classic nonce; verdict fail trả `200 {attested:false}`; mock verifier theo pattern
`logsms`; key iOS device-scoped + cho phép nhiều key; iOS fraud-risk receipt deferred.

### Cross-file follow-ups

- `services/appService/domain/attestation.go` — thêm enum `ChallengePurpose`, bỏ `IOSMode`.
- `migrations/0004_attestation.sql` — thêm cột `purpose`.
- `docs/specs/api/openapi.yaml` — 3 path (thêm `/assertion/challenge`) + header assertion.

## v1.0 (initial) (2026-09-24)

Initial spec written **before implementation** (Prospective mode), as the work order for building the
`attestation` feature — a standalone verify service for Android Play Integrity + iOS App Attest.

- `requirements.md`: user story + 14 EARS acceptance criteria (challenge; Android verify; iOS
  attest/assert; validation/error cases) + the confirmed API contract (2 public routes) + out of
  scope + technical notes + resolved open questions.
- `design.md`: Clean Architecture layers table (domain/usecase/repository/delivery + `pkg/attestation`
  framework), schema (`attestation_challenges` + `attestation_keys`), request flow, and 4
  step-by-step flows (challenge, android verify, ios attest, ios assert) with transaction boundaries.
- `tasks.md`: Phases 1–6 (Domain → Usecase → Repository → Delivery → `pkg/attestation` → Integration),
  all `[ ]`, plus a Deferred list.
- Reason: anh yêu cầu đặc tả backend hỗ trợ attestation cho Android + iOS. Các quyết định chốt qua
  phỏng vấn ngày 2026-09-24: standalone verify service; 2 endpoint (`challenge` + 1 `verify` tổng rẽ
  nhánh `platform`/`mode`); public; Android decode qua Google server + classic nonce; iOS defer
  fraud-risk receipt (chỉ lưu receipt); Postgres cho cả challenge lẫn khoá; verdict fail trả
  `200 {attested:false}`; ngưỡng Android mặc định `PLAY_RECOGNIZED`; mock verifier theo pattern
  `logsms`; key iOS device-scoped + cho phép nhiều key.

### Cross-file follow-ups

- `docs/specs/api/openapi.yaml` — thêm 2 path `POST /v1/attestation/challenge` + `POST /v1/attestation/verify`
  (chỉ viết sau khi handler chạy thật, đúng convention repo).
- `services/appService/config/config.go` + `config.dev.yaml` — block `attestation:` mới.
- `migrations/` — file `0004_attestation.sql` mới; ERD `docs/specs/db/` nếu muốn giữ đồng bộ.
- `docs/knowledge/features/` — concept `attestation` (mới) cần `bdd-okf-docs` tạo sau khi flow chạy
  end-to-end.
- `go.mod`/`vendor/` — dependency mới cho CBOR (iOS App Attest) khi làm `pkg/attestation/appattest`.

### ⚠️ Not confirmed (không chặn Phase 1–5, cần quyết định sau)

- `design.md` flow "iOS attest Step 3": production có từ chối key `environment=development` không
  (config `ios.reject_development_keys`)? — chỉ là một cờ config thêm sau, không đổi cấu trúc flow.
