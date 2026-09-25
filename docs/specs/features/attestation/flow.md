# Flow — attestation

Version: 1.4
Last updated: 2026-09-25
Status: implemented — đủ clean-arch layers + cả 3 driver (mock, playintegrity, appattest); fraud-risk receipt + gate nghiệp vụ deferred
Module: `attestation`

> See `changelog.md` in this folder for the version history of this spec.

<!--
File này là **bản vẽ** (visual) của các section "Step-by-step Flow" trong `design.md`. Nó không
thêm bước mới, không bỏ bước — mỗi mũi tên/message ở đây phải trace 1:1 về một Step trong
`design.md`. Ngược lại, mỗi Step trong `design.md` phải xuất hiện ở đây. Thấy chỗ lệch là bug của
chính file này. Quy ước đầy đủ: xem `references/flow_diagram.md`.

Quy tắc của file này:
- Một section `###` cho MỖI section "Step-by-step Flow — ..." trong `design.md` (cùng tên, cùng thứ
  tự). Không gộp nhiều flow vào một diagram, không tách một flow ra nhiều diagram.
- Mặc định là **sequence diagram**. Chỉ thêm **flowchart** cho nhánh rẽ logic mà sequence vẽ gượng.
- Nhãn message/participant viết tiếng Việt ngắn; keyword Mermaid (`participant`, `alt`, `loop`, ...)
  giữ tiếng Anh — khớp quy ước mixed-language của spec và comment `%%` của `erd.mmd`.
- Mọi `⚠️ Not confirmed` ở một bước trong `design.md` xuất hiện ở đây dưới dạng `Note` trên đúng
  message tương ứng — không được vẽ nó như thể đã chốt.
-->

## Sequence — Cấp challenge (attestation)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as AttestationHandler<br/>(delivery/http)
    participant U as AttestationUsecase
    participant R as AttestationRepository
    participant DB as Postgres

    C->>H: POST /v1/attestation/challenge
    H->>H: DecodeJSONBodyStrict → challengeRequest{platform}
    alt platform ∈ {android, ios}
        H->>U: RequestChallenge(ctx, platform, purpose=attest)
    else validation fail
        H--xC: 400 VALIDATION_ERROR (không chạm DB)
    end

    U->>U: crypto/rand sinh nonce ≥ 16 byte → base64
    U->>R: SaveChallenge(ctx, challenge, platform, purpose=attest)
    R->>DB: INSERT attestation_challenges (purpose='attest', expires_at = now() + ttl)
    alt success
        R-->>U: ok
        U-->>H: { challenge, expires_in_seconds }
        H-->>C: 200 { challenge, expires_in_seconds }
    else lỗi DB
        R--xU: err
        U--xH: err
        H--xC: 500 INTERNAL_ERROR
    end
```

## Sequence — Cấp challenge (assertion, iOS)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as AttestationHandler<br/>(delivery/http)
    participant U as AttestationUsecase
    participant R as AttestationRepository
    participant DB as Postgres

    C->>H: POST /v1/assertion/challenge
    H->>H: DecodeJSONBodyStrict → challengeRequest{platform}
    alt platform = ios
        H->>U: RequestChallenge(ctx, platform=ios, purpose=assert)
    else validation fail (platform ≠ ios)
        H--xC: 400 VALIDATION_ERROR (không chạm DB)
    end

    U->>U: crypto/rand sinh nonce ≥ 16 byte → base64
    U->>R: SaveChallenge(ctx, challenge, platform, purpose=assert)
    R->>DB: INSERT attestation_challenges (purpose='assert', expires_at = now() + ttl)
    alt success
        R-->>U: ok
        U-->>H: { challenge, expires_in_seconds }
        H-->>C: 200 { challenge, expires_in_seconds }
    else lỗi DB
        R--xU: err
        U--xH: err
        H--xC: 500 INTERNAL_ERROR
    end
```

## Sequence — Verify Android (Play Integrity)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as AttestationHandler<br/>(delivery/http)
    participant U as AttestationUsecase
    participant R as AttestationRepository
    participant DB as Postgres
    participant V as AttestationVerifier<br/>(adapter → pkg/attestation)
    participant G as Google Play Integrity<br/>(pkg/attestation/playintegrity)

    C->>H: POST /v1/attestation/verify (platform=android)
    H->>H: DecodeJSONBodyStrict → verifyRequest{platform, challenge, token}
    alt platform=android, token + challenge đủ, cấm key_id/attestation
        H->>U: Verify(ctx, input)
    else validation fail
        H--xC: 400 VALIDATION_ERROR (không gọi Google, không chạm DB)
    end

    U->>R: FindChallenge(challenge, purpose=attest)
    R->>DB: SELECT ... FROM attestation_challenges WHERE challenge=$1 AND purpose='attest'
    alt còn hạn, used_at IS NULL
        R-->>U: challenge hợp lệ
    else không có / hết hạn / đã dùng
        R--xU: ErrAttestationChallengeInvalid
        U--xH: err
        H--xC: 400 ATTESTATION_CHALLENGE_INVALID (không gọi Google)
    end

    Note over U,G: Step 3 — decode token, thuộc pkg/attestation, ngoài scope feature này
    U->>V: VerifyAndroid(ctx, token, challenge)
    V->>G: DecodeIntegrityToken(packageName, {integrity_token}) bằng service account
    alt Google 2xx
        G-->>V: TokenPayloadExternal
        V-->>U: AndroidVerdict{packageName, nonce, timestampMillis, ...verdict}
    else Google lỗi / timeout / HTTP != 2xx
        G--xV: err
        V--xU: ErrAttestationUpstreamError
        U--xH: err
        H--xC: 502 ATTESTATION_UPSTREAM_ERROR (KHÔNG đánh dấu used → retry được)
    end

    U->>U: đối chiếu: packageName == config / nonce == challenge / freshness window / appRecognitionVerdict == PLAY_RECOGNIZED
    U->>R: MarkChallengeUsed(challenge)
    R->>DB: UPDATE attestation_challenges SET used_at=now() WHERE challenge=$1 AND used_at IS NULL
    alt mọi check pass ∧ 1 hàng được đánh dấu
        R-->>U: ok
        U-->>H: { attested: true, platform, signals }
        H-->>C: 200 { attested: true, ... }
    else một check không pass (verdict fail)
        Note over U,R: vẫn MarkChallengeUsed — nonce one-time bất kể verdict
        R-->>U: ok
        U-->>H: { attested: false, platform, reason }
        H-->>C: 200 { attested: false, ... }
    else 0 hàng bị đánh dấu (race)
        R--xU: 0 hàng — challenge đã bị request khác dùng
        U--xH: err
        H--xC: 400 ATTESTATION_CHALLENGE_INVALID
    end
```

## Sequence — Verify iOS pha `attest` (đăng ký khoá)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as AttestationHandler<br/>(delivery/http)
    participant U as AttestationUsecase
    participant R as AttestationRepository
    participant DB as Postgres
    participant V as AttestationVerifier<br/>(adapter → pkg/attestation)

    C->>H: POST /v1/attestation/verify (platform=ios)
    H->>H: DecodeJSONBodyStrict → verifyRequest{platform, challenge, key_id, attestation}
    alt platform=ios, key_id + attestation đủ, cấm token
        H->>U: Verify(ctx, input)
    else validation fail
        H--xC: 400 VALIDATION_ERROR (không chạm DB)
    end

    U->>R: FindChallenge(challenge, purpose=attest)
    R->>DB: SELECT ... FROM attestation_challenges WHERE challenge=$1 AND purpose='attest'
    alt còn hạn, chưa dùng
        R-->>U: challenge hợp lệ
    else không có / hết hạn / đã dùng
        R--xU: ErrAttestationChallengeInvalid
        U--xH: err
        H--xC: 400 ATTESTATION_CHALLENGE_INVALID
    end

    Note over U,V: Step 3 — verify crypto, thuộc pkg/attestation, ngoài scope feature này (local, KHÔNG gọi Apple)
    U->>V: VerifyIOSAttestation(ctx, key_id, attestation, challenge)
    V->>V: decode CBOR / verify x5c / nonce / RP ID / counter==0 / aaguid→environment / SHA256(pubkey)==key_id
    Note over V: ⚠️ Not confirmed: production có từ chối key environment=development không (config ios.reject_development_keys)?
    alt mọi bước crypto pass
        V-->>U: IOSAttestationResult{publicKey, receipt, environment}
    else crypto sai
        V--xU: err nội bộ
        U-->>H: { attested: false, reason } (không phải 4xx)
        H-->>C: 200 { attested: false, ... } (KHÔNG lưu key)
    end

    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction — SaveKey + MarkChallengeUsed
        U->>R: SaveKey(key_id, publicKey, receipt, environment) + MarkChallengeUsed(challenge)
        R->>DB: UPSERT attestation_keys ... ON CONFLICT (key_id) DO UPDATE
        R->>DB: UPDATE attestation_challenges SET used_at=now() WHERE challenge=$1 AND used_at IS NULL
        alt success
            R-->>U: ok
        else lỗi DB
            R--xU: err → rollback
        end
    end

    alt success
        U-->>H: { attested: true, platform: ios, signals: { environment } }
        H-->>C: 200 { attested: true, ... }
    else lỗi DB (sau khi đã verify)
        U--xH: err
        H--xC: 500 INTERNAL_ERROR
    end
```

## Sequence — Assertion iOS (middleware `RequireAttestation`)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant MW as RequireAttestation<br/>(delivery/http middleware)
    participant U as AttestationUsecase
    participant R as AttestationRepository
    participant DB as Postgres
    participant V as AttestationVerifier<br/>(adapter → pkg/attestation)

    C->>MW: request nhạy cảm (đã bọc RequireAttestation)
    MW->>MW: đọc 3 header X-Attestation-Key-Id / -Assertion / -Challenge + tee body
    alt đủ 3 header
        MW->>MW: clientData = challenge || method || path || SHA256(body)
        MW->>U: VerifyAssertion(ctx, keyID, assertion, clientData, challenge)
    else thiếu header
        MW--xC: 401 ATTESTATION_FAILED (thiếu bằng chứng)
    end

    U->>R: FindChallenge(challenge, purpose=assert)
    R->>DB: SELECT ... WHERE challenge=$1 AND purpose='assert'
    alt có + còn hạn + chưa dùng
        R-->>U: challenge hợp lệ
    else không có / hết hạn / đã dùng
        R--xU: ErrAttestationChallengeInvalid
        U--xMW: err
        MW--xC: 401 ATTESTATION_CHALLENGE_INVALID
    end

    U->>R: FindKey(key_id)
    R->>DB: SELECT ... FROM attestation_keys WHERE key_id=$1
    alt có key
        R-->>U: key{publicKey, storedCounter}
    else không có
        R--xU: ErrAttestationKeyNotFound
        U--xMW: err
        MW--xC: 401 ATTESTATION_FAILED (reason=unknown_key)
    end

    Note over U,V: Step 3 — verify assertion, crypto thuộc pkg/attestation (local, KHÔNG gọi Apple)
    U->>V: VerifyIOSAssertion(ctx, publicKey, assertion, clientData, storedCounter)
    V->>V: decode CBOR / SHA256(clientData) / verify ES256 / RP ID / counter > storedCounter
    alt chữ ký + RP ID đúng, counter tăng
        V-->>U: newCounter
    else chữ ký / RP ID sai
        V--xU: ErrAttestationAssertionInvalid
        U--xMW: err
        MW--xC: 401 ATTESTATION_FAILED (reason=assertion_invalid, KHÔNG cập nhật counter)
    else counter <= storedCounter
        V--xU: ErrAttestationCounterNotIncreased
        U--xMW: err
        MW--xC: 401 ATTESTATION_FAILED (reason=counter_not_increased, chống replay)
    end

    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction — UpdateKeyCounter + MarkChallengeUsed
        U->>R: UpdateKeyCounter(key_id, newCounter) + MarkChallengeUsed(challenge)
        R->>DB: UPDATE attestation_keys SET sign_counter=$1 WHERE key_id=$2
        R->>DB: UPDATE attestation_challenges SET used_at=now() WHERE challenge=$1 AND used_at IS NULL
        alt success
            R-->>U: ok
        else lỗi DB
            R--xU: err → rollback
        end
    end

    alt success
        U-->>MW: ok
        MW->>MW: pass sang next handler (request đi tiếp)
    else lỗi DB
        U--xMW: err
        MW--xC: 500 INTERNAL_ERROR
    end
```
