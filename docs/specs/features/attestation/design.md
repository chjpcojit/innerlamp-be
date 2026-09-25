# Design — attestation

Version: 1.4
Last updated: 2026-09-25
Status: implemented — đủ clean-arch layers + cả 3 driver (mock, playintegrity, appattest); fraud-risk receipt + gate nghiệp vụ deferred
Module: `attestation`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

`attestation` gồm hai phần, cùng một khối Clean Architecture riêng:

1. **Verify service** — ba route **public** (không `RequireAuth`): cấp nonce (attestation lẫn
   assertion) và xác minh bằng chứng attestation của Android (Play Integrity) lẫn iOS (App Attest,
   pha `attest`), rồi trả **verdict** (`{attested: bool, signals}`).
2. **Middleware `RequireAttestation`** — bọc route nhạy cảm; verify assertion iOS gắn vào **từng
   request** (request binding + chống replay) và cho đi tiếp hoặc từ chối `401`. Đây là phần "gate"
   thật sự, thay thế cho `POST /verify {mode: assert}` của bản v1.0.

Hai nền tảng bất đối xứng nên chia rõ hai nhánh verify:
- **Android** (không khoá bền): `challenge` → client lấy integrity token → `verify` (decode qua
  Google server).
- **iOS** (khoá bền per-device): `challenge` → `verify` (đăng ký khoá, 1 lần) → mỗi request sau đó
  dùng `assertion/challenge` + assertion gắn header → middleware verify inline.

Mọi flow đều có ≥ 2 lời gọi tuần tự ra ngoài (nonce → DB; token → Google → DB), nên bắt buộc có
**Step-by-step Flow** (rule 5 của skill). Phần crypto/CBOR nặng (verify chuỗi cert, chữ ký ES256,
decode CBOR/COSE) nằm ở `pkg/attestation` — infrastructure dùng chung — còn usecase chỉ phối hợp
challenge + gọi verifier + đọc/ghi trạng thái qua repository. Middleware chỉ phối hợp đọc request →
dựng `clientData` → gọi usecase, không đụng crypto trực tiếp.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/attestation.go` | `AttestationChallenge` (`ID`, `Challenge`, `Platform`, `Purpose` (`attest`/`assert`), `UsedAt`, `ExpiresAt`), `AttestationKey` (`KeyID`, `Platform`, `PublicKey`, `Receipt`, `SignCounter`, `Environment`, `UserID *int64`), `Verdict` (`Attested bool`, `Platform`, `Signals`, `Reason`), enum `Platform` (`android`/`ios`), enum `ChallengePurpose` (`attest`/`assert`). Thuần nghiệp vụ, không JSON tag. |
| Domain / Errors | `services/appService/domain/errors.go` | Sentinels: `ErrAttestationChallengeInvalid` (unknown/expired/used), `ErrAttestationUpstreamError` (Google lỗi/timeout), `ErrAttestationKeyNotFound`, `ErrAttestationCounterNotIncreased`, `ErrAttestationAssertionInvalid` (chữ ký/RP ID sai). Verify endpoint map 2 sentinel đầu sang 400/502; middleware map toàn bộ sang `401`. Nhánh "verdict fail" của verify **không** là error — là `Verdict{Attested:false, Reason:...}`. |
| Domain / Repository interface | `services/appService/domain/attestation.go` | `AttestationRepository`: `SaveChallenge(ctx, challenge, platform, purpose)`, `FindChallenge(ctx, challenge, purpose)` (kèm trạng thái used/expired), `MarkChallengeUsed`, `SaveKey` (upsert), `FindKey`, `UpdateKeyCounter`. Khai ở `domain/`, implement ở `repository/`. |
| Domain / Port | `services/appService/domain/attestation.go` | `AttestationVerifier` interface (seam tới `pkg/attestation`): `VerifyAndroid(ctx, token, challenge string) (AndroidVerdict, error)`, `VerifyIOSAttestation(ctx, keyID string, attestation []byte, challenge string) (IOSAttestationResult, error)`, `VerifyIOSAssertion(ctx, publicKey []byte, assertion, clientData []byte, storedCounter int64) (newCounter int64, err error)`. usecase không import `pkg/attestation` trực tiếp. *Note v1.3: `VerifyIOSAssertion` nhận `publicKey` (khoá đã lưu từ FindKey), không phải `keyID` — keyID chỉ là hash của khoá, không đủ để verify chữ ký.* |
| Domain / Mock | `services/appService/domain/mock_attestation_repository.go` | Mock tay cho repository + verifier, dùng cho test usecase — không DB/không Google/Apple thật. |
| Usecase | `services/appService/usecase/attestation_usecase.go` | `RequestChallenge(ctx, platform, purpose)` (sinh nonce `crypto/rand` → `SaveChallenge` → trả DTO), `Verify(ctx, input)` (rẽ nhánh `platform`: android → Google decode; ios → attest local + lưu key), `VerifyAssertion(ctx, keyID, assertion, clientData, challenge)` (cho middleware: tra challenge/key → verify → `UpdateKeyCounter` + `MarkChallengeUsed`). Sở hữu TTL/ngưỡng verdict qua `config.AttestationConfig` (import `services/appService/config` — ngoại lệ có chủ ý, như `OTPConfig`). |
| Repository | `services/appService/repository/postgres_attestation_repository.go` | SQL + transaction: `SaveChallenge`, `FindChallenge` (có `FOR UPDATE` khi `MarkChallengeUsed` để chống race), `SaveKey` (UPSERT theo `key_id`), `FindKey`, `UpdateKeyCounter`. |
| Repository / Port adapter | `services/appService/repository/attestation_verifier.go` | Adapter bọc `pkg/attestation.Verifier` thành `domain.AttestationVerifier` (đổi shape chung của pkg thành shape có `challenge`/`storedCounter` mà domain cần). Sống ở `repository/` — cùng lý do `sms_otp_sender.go` bọc `pkg/sms.Sender`. |
| Delivery / Handler | `services/appService/delivery/http/attestation_handler.go` | DTO request/response + 3 handler (`Challenge`, `AssertionChallenge`, `Verify`), decode strict, map domain error → status. JSON tag chỉ sống ở đây. |
| Delivery / Middleware | `services/appService/delivery/http/attestation_middleware.go` | `RequireAttestation(next)` — đọc 3 header + request body (tee để vừa hash vừa chuyển cho handler), dựng `clientData`, gọi usecase `VerifyAssertion`, pass/`401`. Sống cạnh `pkg/middleware` nhưng là nghiệp vụ attestation nên đặt ở `delivery/http`. |
| Delivery / DTO | `services/appService/delivery/http/attestation_handler.go` | `challengeRequest{Platform}` (attest lẫn assertion), `verifyRequest{Platform, Challenge, Token?, KeyID?, Attestation?}` + response types. Không còn `Mode` — iOS assert đã chuyển sang middleware. |
| Wiring | `services/appService/cmd/main.go` | Constructor injection + `mux.HandleFunc` cho 3 route public + bọc middleware cho route nhạy cảm (tương lai). Blank import driver `pkg/attestation/mock` (dev) — chạm block `⚠️ DEV ONLY`. |
| External package | `pkg/attestation/attestation.go` | Interface `Verifier` + `Register(type, FactoryFunc)` / `New(config)` theo pattern provider-agnostic (giống `pkg/sms`). |
| External package | `pkg/attestation/playintegrity/` | Driver Android: gọi `playintegrity/v1.DecodeIntegrityToken` bằng service account, trả `TokenPayloadExternal`. Tự đăng ký qua `init()`. |
| External package | `pkg/attestation/appattest/` | Driver iOS: decode CBOR, verify `x5c` với Apple root, verify nonce/RP ID/counter/aaguid, verify chữ ký ES256 assertion. Tự đăng ký qua `init()`. |
| External package | `pkg/attestation/mock/` | Driver mock (dev): trả verdict theo config (luôn pass / luôn fail / pass theo nonce). Tự đăng ký qua `init()`. |
| Migration | `migrations/0004_attestation.sql` | `attestation_challenges` + `attestation_keys` (xem §Schema). |
| API spec | `docs/specs/api/openapi.yaml` | 3 path + schema + header assertion. Viết sau khi handler chạy thật qua `curl`. |

Dependency rule: `delivery → usecase → domain`; `repository → domain`; `domain` không import layer
ngoài. DTO HTTP chỉ sống trong `delivery/http/attestation_handler.go`. Crypto/CBOR nằm ở
`pkg/attestation`, **không** rò vào `domain`. Middleware `RequireAttestation` ở `delivery/http` được
phép import `usecase` (cùng chiều dependency `delivery → usecase`).

## Schema

```sql
-- 0004_attestation.sql (idempotent, toàn CREATE ... IF NOT EXISTS)

CREATE TABLE attestation_challenges (
    id           BIGSERIAL PRIMARY KEY,
    challenge    VARCHAR(255) NOT NULL UNIQUE,  -- nonce base64 (>= 16 byte thô), là lookup key
    platform     VARCHAR(16) NOT NULL,      -- 'android' | 'ios'
    purpose      VARCHAR(16) NOT NULL,      -- 'attest' | 'assert'
    used_at      TIMESTAMPTZ,               -- NULL = chưa dùng
    expires_at   TIMESTAMPTZ NOT NULL,      -- now() + challenge_ttl
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_attestation_challenges_expires ON attestation_challenges(expires_at);

CREATE TABLE attestation_keys (
    id           BIGSERIAL PRIMARY KEY,
    key_id       VARCHAR(255) NOT NULL,     -- keyId base64 của iOS
    platform     VARCHAR(16) NOT NULL,      -- 'ios' (android không có khoá bền)
    public_key   BYTEA NOT NULL,            -- COSE ES256 public key (dạng CBOR/X9.62)
    receipt      BYTEA,                     -- lưu lại cho fraud-risk (deferred), không gửi Apple ở bản này
    sign_counter BIGINT NOT NULL DEFAULT 0, -- counter lớn nhất đã thấy, chống replay
    environment  VARCHAR(16) NOT NULL,      -- 'development' | 'production' (từ aaguid)
    user_id      BIGINT REFERENCES users(id) ON DELETE SET NULL,  -- nullable, device-scoped
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_attestation_keys_key_id ON attestation_keys(key_id);
```

Chú thích:
- `attestation_challenges.purpose` phân biệt nonce cho `verify` (attestation) và nonce cho assertion
  (middleware). Cùng một bảng vì cả hai đều là nonce dùng-một-lần có TTL; `FindChallenge` lọc theo
  `purpose` để nonce của luồng này không bị dùng ở luồng kia.
- `attestation_keys.key_id` là hash của public key (credentialId), nên **unique tự nhiên** — upsert
  theo `key_id` làm cho việc đăng ký lại (retry) là idempotent (AC10), và "cho nhiều key" (quyết định
  #14) vẫn được vì mỗi reinstall sinh `key_id` khác.
- Không có bảng key riêng cho Android — Android không giữ trạng thái khoá (mỗi request một token).

## Request flow

> Bản vẽ chi tiết từng luồng nằm ở `flow.md` trong cùng thư mục — mỗi section "Step-by-step Flow"
> dưới đây có một sequence diagram Mermaid tương ứng, mirror 1:1 (không thêm/bớt bước). Bản trace
> ASCII trước đây đã được thay bằng `flow.md` để tránh hai bản vẽ cùng một thứ bị drift (quy ước
> `flow_diagram.md`). Nguồn sự thật của luồng vẫn là các section "Step-by-step Flow" ngay dưới.

## Step-by-step Flow — Cấp challenge (attestation)

```
Step 1 — Validate request (feature này)
- Trigger: AttestationHandler nhận POST /v1/attestation/challenge
- Pre-condition: DecodeJSONBodyStrict pass (field lạ bị từ chối, cap 1 MB)
- Call: validator trên challengeRequest (platform required, oneof=android ios)
    - Success → Step 2
    - Failure → 400 VALIDATION_ERROR, không chạm DB

Step 2 — Sinh nonce + lưu challenge (feature này)
- Trigger: Step 1 pass
- Pre-condition: platform hợp lệ
- Call: crypto/rand sinh 16+ byte ngẫu nhiên → base64; AttestationRepository.SaveChallenge
        INSERT attestation_challenges (challenge, platform, purpose='attest', expires_at = now() + ttl)
    - Success → 200 { challenge, expires_in_seconds }
    - Failure (lỗi DB) → 500 INTERNAL_ERROR
  Nonce KHÔNG được sinh bằng math/rand — phải crypto/rand, kẻ đoán được nonce sẽ forge verdict.
```

## Step-by-step Flow — Cấp challenge (assertion, iOS)

```
Step 1 — Validate request (feature này)
- Trigger: AttestationHandler nhận POST /v1/assertion/challenge
- Pre-condition: DecodeJSONBodyStrict pass
- Call: validator (platform required, oneof=ios)
    - Success → Step 2
    - Failure → 400 VALIDATION_ERROR, không chạm DB

Step 2 — Sinh nonce + lưu challenge (feature này)
- Call: crypto/rand sinh 16+ byte → base64; SaveChallenge(...) với purpose='assert'
    - Success → 200 { challenge, expires_in_seconds }
    - Failure (lỗi DB) → 500 INTERNAL_ERROR
```

## Step-by-step Flow — Verify Android (Play Integrity)

```
Step 1 — Validate request (feature này)
- Trigger: AttestationHandler nhận POST /v1/attestation/verify với platform=android
- Pre-condition: DecodeJSONBodyStrict pass
- Call: validator (platform=android; token required; challenge required; cấm field ios như
        key_id/attestation)
    - Success → Step 2
    - Failure → 400 VALIDATION_ERROR, không gọi Google, không chạm DB

Step 2 — Tra challenge (feature này)
- Trigger: Step 1 pass
- Pre-condition: challenge (nonce) có trong body
- Call: AttestationRepository.FindChallenge(challenge, purpose=attest)
    - Success (còn hạn, used_at IS NULL) → Step 3
    - Failure (không có / hết hạn / đã dùng) → domain.ErrAttestationChallengeInvalid
      → 400 ATTESTATION_CHALLENGE_INVALID (MỘT code cho cả ba, tránh dò trạng thái). Dừng, không gọi Google.

Step 3 — Decode integrity token trên Google server (thuộc pkg/attestation, ngoài scope feature này)
- Trigger: Step 2 pass
- Pre-condition: token + nonce đã lưu có sẵn
- Call: AttestationVerifier.VerifyAndroid(ctx, token, challenge) → pkg/attestation/playintegrity
        → playintegrity/v1.DecodeIntegrityToken(packageName, {integrity_token}) bằng service account
    - Success → trả AndroidVerdict (packageName, nonce, timestampMillis, appRecognitionVerdict,
      deviceRecognitionVerdict, appLicensingVerdict) → Step 4
    - Failure (Google lỗi/timeout/HTTP != 2xx) → domain.ErrAttestationUpstreamError → 502
      ATTESTATION_UPSTREAM_ERROR. KHÔNG đánh dấu challenge đã dùng (caller retry được).

Step 4 — Đối chiếu verdict + đóng challenge (feature này)
- Trigger: Step 3 pass
- Pre-condition: AndroidVerdict đã có
- Call: so khớp ngay trong usecase —
        (a) requestPackageName == config.AttestationConfig.Android.PackageName
        (b) nonce == challenge đã lưu (Step 2)
        (c) now - timestampMillis <= config.AttestationConfig.FreshnessWindow
        (d) appRecognitionVerdict == "PLAY_RECOGNIZED"  (ngưỡng mặc định; config-driven)
        → AttestationRepository.MarkChallengeUsed(challenge)  (UPDATE ... SET used_at = now()
          WHERE used_at IS NULL; nếu 0 hàng bị đổi nghĩa là race — đã bị request khác dùng)
    - Success (mọi check pass, mark used thành công) → 200 { attested: true, platform, signals }
    - Failure (một check không pass) → 200 { attested: false, platform, reason }, vẫn MarkChallengeUsed
    - Failure (mark used race — 0 hàng) → coi như challenge đã bị dùng: 400 ATTESTATION_CHALLENGE_INVALID
  Confirmed (quyết định thiết kế): verdict fail KHÔNG phải HTTP error — luôn 200 + {attested:false}
  (quyết định #10). Chỉ lỗi trạng thái challenge (Step 2) và lỗi upstream (Step 3) mới là 4xx/5xx.
  Challenge được đóng (used) ngay lần verify đầu tiên bất kể verdict — nonce là one-time.
```

## Step-by-step Flow — Verify iOS pha `attest` (đăng ký khoá)

```
Step 1 — Validate request (feature này)
- Trigger: POST /v1/attestation/verify, platform=ios
- Pre-condition: DecodeJSONBodyStrict pass
- Call: validator (platform=ios; key_id + attestation required; cấm token)
    - Success → Step 2
    - Failure → 400 VALIDATION_ERROR

Step 2 — Tra challenge (feature này)
- Trigger: Step 1 pass
- Call: AttestationRepository.FindChallenge(challenge, purpose=attest)
    - Success (còn hạn, chưa dùng) → Step 3
    - Failure → 400 ATTESTATION_CHALLENGE_INVALID, dừng

Step 3 — Verify attestation object (thuộc pkg/attestation, ngoài scope feature này)
- Trigger: Step 2 pass
- Pre-condition: key_id + attestation (base64) + challenge đã có
- Call: AttestationVerifier.VerifyIOSAttestation(ctx, key_id, attestation, challenge)
        → pkg/attestation/appattest:
        (a) decode CBOR, kiểm fmt == "apple-appattest"
        (b) verify x5c: credCert + intermediate CA lên Apple App Attest root (chữ ký + thời hạn)
        (c) clientDataHash = SHA256(challenge); nonce = SHA256(authData || clientDataHash);
            so nonce trong credCert (OID 1.2.840.113635.100.8.2)
        (d) SHA256(app_id) == RP ID hash trong authData
        (e) counter == 0
        (f) aaguid → environment ('appattestdevelop' = development, 'appattest' = production)
        (g) SHA256(public key trong credCert, dạng X9.62) == key_id
    - Success → trả IOSAttestationResult{publicKey, receipt, environment} → Step 4
    - Failure (mọi bước crypto sai) → trả lỗi nội bộ cho usecase → 200 { attested: false, reason }
      (không phải 4xx — quyết định #10). Dừng, KHÔNG lưu key.
  ⚠️ Not confirmed: môi trường production có từ chối key `environment=development` không
  (config `ios.reject_development_keys`)? — không chặn Phase 1–5, chỉ là một cờ config thêm sau.

Step 4 — Lưu key + đóng challenge (feature này)
- Trigger: Step 3 success
- Pre-condition: publicKey, receipt, environment đã có
- Call: AttestationRepository.SaveKey (UPSERT theo key_id: INSERT ... ON CONFLICT (key_id) DO UPDATE
        SET public_key, receipt, environment, updated_at)  — idempotent (AC10)
        + MarkChallengeUsed(challenge)
    - Success → 200 { attested: true, platform: ios, signals: { environment } }
    - Failure (lỗi DB) → 500 INTERNAL_ERROR (không rollback được việc đã verify, nhưng key chưa lưu
      thì client phải attest lại với challenge mới)
  Transaction boundary: SaveKey + MarkChallengeUsed nằm cùng MỘT tx để không có key lưu mà challenge
  chưa đóng (tránh dùng lại challenge cấp ra một key thứ hai).
```

## Step-by-step Flow — Assertion iOS (middleware `RequireAttestation`)

```
Step 1 — Đọc headers + body (feature này, delivery/http)
- Trigger: request nhạy cảm (được bọc RequireAttestation) đến
- Pre-condition: route đã được wrap; body được tee (vừa đọc để hash vừa chuyển nguyên vẹn cho handler)
- Call: đọc headers X-Attestation-Key-Id, X-Attestation-Assertion, X-Attestation-Challenge
    - Thiếu header nào → 401 ATTESTATION_FAILED (thiếu bằng chứng)
    - Success → Step 2

Step 2 — Tra challenge + key (feature này)
- Trigger: Step 1 pass
- Call: AttestationRepository.FindChallenge(challenge, purpose=assert)
        → không có / hết hạn / dùng rồi → 401 ATTESTATION_CHALLENGE_INVALID (MỘT code cho cả ba), dừng
        AttestationRepository.FindKey(key_id) → không có → 401 ATTESTATION_FAILED (reason=unknown_key), dừng
    - Success → Step 3

Step 3 — Dựng clientData + verify assertion (crypto ở pkg/attestation)
- Trigger: Step 2 pass
- Pre-condition: public key + storedCounter đã có; method/path/body của request thật có sẵn
- Call: clientData = challenge || method || path || SHA256(body)   (dựng từ request thật + nonce đã lưu)
        AttestationVerifier.VerifyIOSAssertion(ctx, public_key, assertion, clientData, storedCounter)
        → pkg/attestation/appattest:
        (a) decode CBOR assertion (signature + authenticatorData)
        (b) clientDataHash = SHA256(clientData)
        (c) verify chữ ký ES256 bằng public key đã lưu trên (authenticatorData || clientDataHash)
        (d) SHA256(app_id) == RP ID hash trong authenticatorData
        (e) đọc counter trong authenticatorData; yêu cầu counter > storedCounter
    - Success → trả counter mới → Step 4
    - Failure (chữ ký sai, RP ID sai) → 401 ATTESTATION_FAILED (reason=assertion_invalid), dừng,
      KHÔNG cập nhật counter
    - Failure (counter <= storedCounter) → 401 ATTESTATION_FAILED (reason=counter_not_increased),
      KHÔNG cập nhật counter (AC12 — chống replay/rollback)

Step 4 — Cập nhật counter + đóng challenge (feature này)
- Trigger: Step 3 success
- Call: AttestationRepository.UpdateKeyCounter(key_id, newCounter) + MarkChallengeUsed(challenge)
    - Success → pass sang next handler (request đi tiếp)
    - Failure (lỗi DB) → 500 INTERNAL_ERROR
  Transaction boundary: UpdateKeyCounter + MarkChallengeUsed cùng MỘT tx.
  Request binding: server KHÔNG parse JSON client_data do client gửi — nó DỰNG LẠI clientData từ
  request thật (method/path/body) + challenge đã lưu, rồi hash để verify chữ ký. Nếu client gắn
  assertion sinh cho một request khác (method/path/body khác) → chữ ký không khớp → bị từ chối. Đây
  là chống replay theo request, mạnh hơn nonce đơn thuần.
```

## Deliberately stubbed for now

| Thứ | Trạng thái | Ghi chú |
|---|---|---|
| `pkg/attestation` + `pkg/attestation/mock` | **Đã implement** (v1.2) | Tạo theo pattern `pkg/sms`. Driver `mock` dùng cho dev — verify giả theo config (pass / fail / upstream error / assertion mode). |
| `pkg/attestation/playintegrity` | **Đã implement** (v1.3) | Gọi `playintegrity/v1.DecodeIntegrityToken` bằng service account (scope `playintegrity`); auth qua `credentials_json`/`credentials_file` (fallback ADC). |
| `pkg/attestation/appattest` | **Đã implement** (v1.3) | Verify CBOR/x5c/nonce/RP ID/counter/aaguid + chữ ký ES256 assertion; Apple root nhúng (override `root_cert_pem`). Dependency `fxamacker/cbor/v2`. |
| Fraud-risk receipt | **Defer** | `attestation_keys.receipt` được lưu nhưng chưa gọi `data-*.devicecheck.apple.com` lấy risk metric (quyết định #7). |

## Known gaps carried by this feature

- **Không có cơ chế dọn hàng hết hạn** cho `attestation_challenges` (hàng `expires_at <= now()` tích
  tụ). Cùng tình trạng `otp_codes`/`pending_registration` đã chốt "chấp nhận ở MVP" — hàng hết hạn
  vô hiệu về logic (bị `FindChallenge` lọc). Không cần endpoint nội bộ riêng; khi làm job dọn chung
  thì gộp bảng này vào.
- **`attestation_keys` không tự hết hạn/vô hiệu khoá** — khoá iOS sống mãi tới khi bị ghi đè hoặc
  user xoá. Chính sách vòng đời khoá (re-attest sau N ngày, thu hồi khoá) ngoài scope bản này.
- **Middleware chưa gắn vào route nào** — feature này chỉ *cung cấp* `RequireAttestation`; việc chọn
  route nào bọc nó (và xử lý fail) là quyết định gate nghiệp vụ, ngoài scope.
