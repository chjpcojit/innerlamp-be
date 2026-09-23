# Design — jwt-token-uniqueness

Version: 1.3
Last updated: 2026-09-22
Status: implemented (2026-09-21) — các dòng bảng Layers đều đã có code thật. Không còn marker `⚠️ Not confirmed`. (v1.3: sửa header bảng Test plan — test nằm ở `pkg/jwt/jti_test.go`, không phải `jwt_test.go`. Không đổi code/AC.)
Module: `pkg/jwt`
Code: `pkg/jwt/`
Feature folder: `pkg/jwt/` + `docs/specs/features/jwt-token-uniqueness/`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Thêm claim `jti` (JWT ID, RFC 7519 §4.1.7) ngẫu nhiên vào mọi token do `pkg/jwt` sinh. Một điểm sửa duy nhất trong `pkg/jwt/jwt.go`; `Service` interface, `Config`, `claims/`, `middleware/` không đổi.

Flow chỉ có **một** bước side-effect (đọc `crypto/rand`, không I/O ra ngoài, không transaction) nên không cần Step-by-step Flow — dùng "Data flow" ngắn bên dưới (ngoại lệ được `references/step_by_step_flow.md` cho phép).

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Package logic | `pkg/jwt/jwt.go` | Thêm helper không export `newTokenID() (string, error)`: đọc 16 byte từ `crypto/rand`, trả `hex.EncodeToString` (32 ký tự). `GenerateAccessToken` và `GenerateRefreshToken` gọi nó **trước** khi dựng claim, gán vào `RegisteredClaims.ID`; lỗi đọc → `fmt.Errorf("generate token id: %w", err)`, không ký token. `RefreshAccessToken` không đổi (đi qua `GenerateAccessToken`). |
| Package logic | `pkg/jwt/jwt.go` | `newTokenID` là biến cấp package kiểu `func() (string, error)` (hoặc nhận `io.Reader`) để test bơm lỗi cho AC3 mà không phải giả lập `crypto/rand` toàn cục. |
| Claims | `pkg/jwt/claims/claims.go` | **Không đổi.** `CustomClaims` đã nhúng `jwt.RegisteredClaims` nên `ID` có sẵn; `Validate()` không được thêm điều kiện "phải có `jti`" (AC4). |
| Mock | `pkg/jwt/mock.go` | **Không đổi.** `Service` không đổi chữ ký (AC8). |
| Tests | `pkg/jwt/jti_test.go` *(file mới; ban đầu dự kiến `jwt_test.go`)* | Test AC1–AC7 (bảng bên dưới), AC1/2/5/6 chạy cho cả HS256 và RS256. |
| Docs | `pkg/jwt/README.md` + `pkg/jwt/claims/README.md` | Ghi `jti` vào danh sách claim và mục `Generate*`; ghi rõ "token cũ không có `jti` vẫn hợp lệ". |

Dependency rule: không thêm import ngoài stdlib (`crypto/rand`, `encoding/hex`); không thêm dependency vào `go.mod`.

## Data flow

```
GenerateRefreshToken(userID, extra)
  ├─ userID == "" → error (như cũ)
  ├─ newTokenID()  ── crypto/rand.Read(16 bytes) ──► hex(32)     [MỚI]
  │     └─ lỗi → "generate token id: %w", KHÔNG ký token           (AC3)
  ├─ dựng CustomClaims{ ..., RegisteredClaims{ ID: jti, ExpiresAt, IssuedAt, NotBefore, ... } }
  └─ signToken(claims)  (HS256 hoặc RS256 — chung một đường, AC2)
```

## Các phương án đã cân nhắc

| Phương án | Kết luận |
|---|---|
| **A. `jti` ngẫu nhiên 128 bit ở `pkg/jwt` (chọn)** | Sửa một chỗ, mọi consumer được lợi (kể cả access token), đúng chuẩn RFC 7519, không phụ thuộc kỷ luật của caller. |
| B. Login tự chèn `sid`/`ver` vào `extra` (phương án tạm ở `login/design.md` v1.0) | Chỉ vá cho refresh token của login; access token vẫn trùng; feature kế tiếp lại dính lại. Bị thay bằng A. |
| C. Tăng độ chính xác `iat` lên nano-giây (`jwt.TimePrecision`) | `TimePrecision` là biến toàn cục của thư viện (ảnh hưởng mọi nơi), vẫn có thể trùng khi hai lời gọi rơi vào cùng tick, và đổi ngữ nghĩa `iat`/`exp` mà client có thể đọc. Loại. |
| D. Dùng UUID v4 (`github.com/google/uuid`) | Thêm dependency chỉ để có 128 bit ngẫu nhiên; `crypto/rand` + hex đủ. Loại vì nguyên tắc Minimalism. |

## Deliberately stubbed for now

Không có. Không phần nào được để trống có chủ ý.

## Known gaps carried by this feature

- **`jti` không tự thu hồi được token.** Fix này chỉ làm token duy nhất và nhận diện được; chặn access token đã revoke cần denylist/tra session — ngoài scope, theo dõi ở `login/requirements.md` Pending.
- **Token cũ (không `jti`) tồn tại tới khi hết hạn.** Refresh token cũ sống tối đa 7 ngày sau khi fix deploy; trong thời gian đó chúng vẫn hợp lệ (AC4) nhưng không có `jti`. Chấp nhận — không làm logout hàng loạt.
- **`RefreshAccessToken` copy `Extra` của refresh token sang access token** (`jwt.go:170`). Không đổi trong spec này; ghi lại để consumer không nhét dữ liệu nhạy cảm vào `extra` của refresh token.

## Test plan (map AC → test) — tất cả nằm ở `pkg/jwt/jti_test.go`

| AC | Test (trong `pkg/jwt/jti_test.go`) |
|---|---|
| 1 | `TestGenerate_SameSecondDistinct` — gọi 2 lần liên tiếp mỗi loại token, HS256 và RS256, assert `!=`. Test này **fail trước fix** (đã tái hiện: `identical: true`). |
| 2 | `TestGenerate_HasJTI` — `ParseToken` rồi assert `ID` khớp `^[0-9a-f]{32}$`, cho access + refresh, HS256 + RS256. |
| 3 | `TestGenerate_JTIEntropyFailure` — thay `newTokenID` bằng hàm trả lỗi, assert `err != nil` và `errors.Is`, token rỗng. |
| 4 | `TestValidateToken_LegacyTokenWithoutJTI` — dựng token tay không có `ID`, `ValidateToken` thành công. |
| 5 | `TestValidateToken_ExposesJTI` — `claims.ID` bằng `jti` đã sinh. |
| 6 | `TestRefreshAccessToken_NewJTI` — `ID` access mới ≠ `ID` refresh gốc. |
| 7 | `TestGenerate_ExtraCannotOverrideJTI` — `extra{"jti":"x","id":"y"}` → `claims.ID` vẫn là giá trị sinh. |
| 8 | Biên dịch: `go build ./...` + `go vet ./...` + toàn bộ test `services/appService` với `MockService` không sửa. |
