# Tasks — attestation

Version: 1.4
Last updated: 2026-09-25
Status: implemented — đủ clean-arch layers + cả 3 driver (mock, playintegrity, appattest); còn deferred là fraud-risk receipt + gate nghiệp vụ + bật driver thật ở config (chờ credential)
Module: `attestation`

> See `changelog.md` in this folder for the version history of this spec.

<!--
Mọi item dưới đây trace về một dòng trong bảng "Layers & components" của design.md hoặc một step
trong "Step-by-step Flow" của nó. Các item đã [x] là code + test đã chạy xanh.
-->

## Phase 1 — Domain

- [x] `domain/attestation.go` — entity `AttestationChallenge` (`ID`, `Challenge`, `Platform`, `Purpose`, `UsedAt`, `ExpiresAt`), `AttestationKey` (`KeyID`, `Platform`, `PublicKey`, `Receipt`, `SignCounter`, `Environment`, `UserID *int64`), `Verdict` (`Attested`, `Platform`, `Signals`, `Reason`), enum `Platform`/`ChallengePurpose` (`attest`/`assert`). Thuần nghiệp vụ, không JSON tag.
- [x] `domain/errors.go` — sentinel `ErrAttestationChallengeInvalid`, `ErrAttestationUpstreamError`, `ErrAttestationKeyNotFound`, `ErrAttestationCounterNotIncreased`, `ErrAttestationAssertionInvalid`. Nhánh "verdict fail" của verify KHÔNG phải error (là `Verdict{Attested:false}`); middleware map lỗi → `401`.
- [x] `domain/attestation.go` — interface `AttestationRepository` (`SaveChallenge`, `FindChallenge`, `MarkChallengeUsed`, `SaveKey`, `FindKey`, `UpdateKeyCounter`). *Note: `SaveChallenge` nhận thêm `expiresAt` (usecase tính từ TTL, như pattern OTP); `SaveKey`/`UpdateKeyCounter` nhận thêm `challenge` để đóng challenge cùng 1 tx (design.md flow iOS Step 4 / assertion Step 4).*
- [x] `domain/attestation.go` — port `AttestationVerifier` (`VerifyAndroid`, `VerifyIOSAttestation`, `VerifyIOSAssertion`).
- [x] `domain/mock_attestation_repository.go` + `domain/mock_attestation_verifier.go` — mock tay cho usecase test.

## Phase 2 — Usecase

- [x] `usecase/attestation_usecase.go` — `RequestChallenge(ctx, platform, purpose)`: sinh nonce `crypto/rand` (≥ 16 byte) → base64 → `SaveChallenge` → trả DTO. Nonce KHÔNG dùng `math/rand`.
- [x] `usecase/attestation_usecase.go` — `Verify` rẽ nhánh `platform`: android → `VerifyAndroid` (Google) → `MarkChallengeUsed`; ios → `VerifyIOSAttestation` (crypto local) → `SaveKey` + `MarkChallengeUsed`. Không còn `mode`.
- [x] `usecase/attestation_usecase.go` — `VerifyAssertion(ctx, keyID, assertion, clientData, challenge)`: `FindChallenge(challenge, purpose=assert)` → `FindKey` → `VerifyIOSAssertion` → `UpdateKeyCounter` + `MarkChallengeUsed`. Map `ErrAttestationChallengeInvalid` / `ErrAttestationUpstreamError` / assertion sentinels đúng chỗ; verdict fail trả `Verdict{Attested:false}`.
- [x] `usecase/attestation_usecase.go` — nhận `config.AttestationConfig` qua constructor (TTL, ngưỡng verdict, `package_name`/`app_id`, freshness window) — không hardcode.
- [x] Unit test `usecase/attestation_usecase_test.go` — table-driven, mock repository + verifier, phủ: challenge (attest + assert) thành công/validation fail; verify android pass/fail từng nhánh verdict; challenge invalid; upstream error không đánh dấu used; ios attest pass/verify-fail; ios assert counter tăng/counter không tăng/unknown key/challenge invalid/signature fail.

## Phase 3 — Repository

- [x] `migrations/0004_attestation.sql` — `attestation_challenges` (kèm cột `purpose`) + `attestation_keys` (theo §Schema của design.md), idempotent (`CREATE ... IF NOT EXISTS`).
- [x] `repository/postgres_attestation_repository.go` — implement `AttestationRepository`: `SaveChallenge` (lưu `purpose`), `FindChallenge` (lọc `purpose` + `used_at IS NULL AND expires_at > now()`), `MarkChallengeUsed` (UPDATE … WHERE `used_at IS NULL`, trả số hàng để bắt race), `SaveKey` (UPSERT theo `key_id`), `FindKey`, `UpdateKeyCounter`.
- [x] `repository/attestation_verifier.go` — adapter bọc `pkg/attestation.Verifier` thành `domain.AttestationVerifier`.
- [x] Repository test `repository/postgres_attestation_repository_test.go` — chạy Postgres thật (skip khi `-short`): race hai `MarkChallengeUsed` cùng challenge → chỉ một thành công; `FindChallenge` lọc theo `purpose`; upsert key idempotent; counter update.

## Phase 4 — Delivery

- [x] `delivery/http/attestation_handler.go` — DTO `challengeRequest` / `verifyRequest` (+ response types); JSON tag chỉ sống ở đây. Không có `Mode`.
- [x] `delivery/http/attestation_handler.go` — handler `Challenge`, `AssertionChallenge`, `Verify`: decode strict → validate (platform oneof; field theo platform) → call usecase → map domain error → status (`ErrAttestationChallengeInvalid` → 400, `ErrAttestationUpstreamError` → 502).
- [x] `delivery/http/attestation_middleware.go` — `RequireAttestation`: đọc 3 header + tee body, dựng `clientData = challenge || method || path || SHA256(body)`, gọi usecase `VerifyAssertion`, pass/`401`.
- [x] Route trong `services/appService/cmd/main.go`: `POST /v1/attestation/challenge`, `POST /v1/assertion/challenge`, `POST /v1/attestation/verify` — **cả ba public**, blank import `pkg/attestation/mock` (⚠️ DEV ONLY). Middleware wrap cho route nhạy cảm là việc gate nghiệp vụ (ngoài scope build này).
- [x] `services/appService/config/config.go` + `config.dev.yaml` — block `attestation:` (`challenge_ttl`, `freshness_window`, `android.package_name`, `android.required_app_recognition_verdict`, `ios.app_id`, driver/provider chọn mock ở dev).
- [x] `docs/specs/api/openapi.yaml` — thêm 3 path + schema + header assertion, viết SAU khi handler chạy thật qua `curl`.
- [x] Handler test `delivery/http/attestation_handler_test.go` + middleware test — success mỗi nhánh, từng mã lỗi, body sai/malformed, field iOS gửi lên cho android (và ngược lại) bị từ chối; middleware: pass khi assertion hợp lệ, `401` khi thiếu header / counter không tăng / unknown key / challenge invalid / chữ ký sai.

## Phase 5 — Hạ tầng dùng chung (`pkg/attestation`)

- [x] `pkg/attestation/attestation.go` — interface `Verifier` + `Register(type, FactoryFunc)` / `New(config)` theo pattern `pkg/sms`.
- [x] `pkg/attestation/mock/` — driver mock tự đăng ký `init()`: trả verdict theo config (luôn pass / luôn fail / upstream error / theo assertion mode), dùng cho dev.
- [x] `pkg/attestation/playintegrity/` — driver Android: `playintegrity/v1.DecodeIntegrityToken` bằng service account (scope `playintegrity`), trả `AndroidResult`. Auth qua `credentials_json`/`credentials_file` (fallback ADC); `mapPayload` tách riêng để test không cần mạng.
- [x] `pkg/attestation/appattest/` — driver iOS: decode CBOR, verify `x5c` (Apple root nhúng, override qua `root_cert_pem`), nonce/RP ID/counter/aaguid, verify chữ ký ES256 assertion. Dependency CBOR `github.com/fxamacker/cbor/v2` đã thêm + vendor.
- [x] Test từng driver + `pkg/attestation/attestation_test.go` (registry: New nil config / provider thiếu / provider chưa đăng ký / factory lỗi).

## Phase 6 — Integration & verification

- [x] `go build ./...`, `go vet ./...`, `go test -short ./...` sạch (driver thật bị skip khi thiếu credential).
- [x] Chạy end-to-end với driver `mock` + Postgres thật: challenge → verify (android + ios attest) → assertion/challenge qua `curl`.
- [x] `check-architecture.sh` + `check_dependency_rule.sh services/appService` sạch.
- [ ] Ghi vào changelog concept OKF nào dưới `docs/knowledge/features/` cần `bdd-okf-docs` tạo (concept mới `attestation`). **Ghi nhận trong changelog.md "Cross-file follow-ups" — chưa viết vì flow end-to-end đã chạy nhưng knowledge bundle do `il-okf-writer` sở hữu.**

## Deferred (không chặn attestation)

- Fraud-risk receipt — gọi `data-*.devicecheck.apple.com/v1/attestationData` lấy risk metric (cần JWT auth bằng khoá Apple cấp). `attestation_keys.receipt` đã lưu sẵn để làm sau.
- Job dọn hàng hết hạn `attestation_challenges` (gộp vào job dọn chung khi có).
- Chính sách vòng đời khoá iOS (re-attest định kỳ, thu hồi khoá).
- Chính sách gate nghiệp vụ — chọn route nào bọc `RequireAttestation` và xử lý fail ra sao (thuộc client / feature tương lai).
- Play Integrity standard request (requestHash) + giải mã local — ngoài scope (đã chọn classic + Google server).
- Bật driver thật ở config (`attestation.android.provider = playintegrity`, `ios.provider = appattest`) — chờ service-account credential và `app_id` thật khi app được publish.
