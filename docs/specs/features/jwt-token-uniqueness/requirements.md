# Requirements — jwt-token-uniqueness

Version: 1.3
Last updated: 2026-09-22
Status: implemented (2026-09-21) — AC1–AC8 đều có test pass (`pkg/jwt/jti_test.go`). Không còn marker `⚠️ Not confirmed`.
Module: `pkg/jwt` (hạ tầng dùng chung — không thuộc `identity`)
Code: `pkg/jwt/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này được **tách ra từ** `docs/specs/features/login/` (phát hiện khi đặc tả US-002). Đây là bug của package dùng chung `pkg/jwt`, không phải của riêng login: mọi consumer đều dính, nên sửa ở gốc thay vì để mỗi feature tự vá bằng `extra`.

## Bằng chứng (tái hiện 2026-09-20)

Chạy test tạm (qua `go test -overlay`, không đổi file nào trong repo), HS256, cùng `userID`, hai lần gọi cách nhau vài micro-giây trong cùng một giây:

```
refresh identical (same user, same second): true
access  identical:                          true
refresh with distinct extra identical:      false
```

Nguyên nhân: `GenerateAccessToken` / `GenerateRefreshToken` (`pkg/jwt/jwt.go`) dựng claim chỉ gồm `user_id`, `token_type`, `iss`, `sub`, `aud` và ba mốc thời gian `exp`/`iat`/`nbf` (`jwt.NewNumericDate` → **độ chính xác 1 giây**), **không có `jti`**. HS256 và JSON encoding đều tất định → cùng đầu vào trong cùng một giây cho ra cùng từng byte.

Hệ quả cụ thể với `services/appService`:

- `user_sessions.refresh_token_hash` là `UNIQUE` (`migrations/0001`). Hai refresh token trùng byte → vi phạm ràng buộc → 500. Xảy ra khi login hai thiết bị cùng lúc, hoặc `refresh` ngay sau `login` (rotate cho ra đúng token cũ).
- Rotation vô nghĩa khi token mới ≡ token cũ: không phát hiện được reuse, không vô hiệu được token cũ.
- Access token cũng trùng — ít nguy hiểm hơn (chưa lưu hash) nhưng nghĩa là không có định danh nào để thu hồi/ghi log một token cụ thể.

## User Story

> Là một **developer của service dùng `pkg/jwt`**,
> Tôi muốn **mỗi token được sinh ra là duy nhất, dù sinh liên tiếp cho cùng một user**,
> Để **có thể lưu hash token, xoay vòng refresh token và phát hiện dùng lại mà không phải tự chèn nonce vào `extra`**.

## Acceptance Criteria (EARS notation)

1. WHEN `GenerateAccessToken` or `GenerateRefreshToken` is called twice with identical arguments within the same second, THE SYSTEM SHALL return two different token strings.
2. WHEN a token is generated, THE SYSTEM SHALL set the registered claim `jti` (`RegisteredClaims.ID`) to a value of at least 128 bits drawn from `crypto/rand`, encoded as a 32-character lowercase hex string, for both access and refresh tokens and for both HS256 and RS256.
3. WHEN the random source fails while generating the `jti`, THE SYSTEM SHALL return an error (wrapped with `%w`) and SHALL NOT return a token without a `jti`.
4. WHEN `ValidateToken` or `ParseToken` receives a token that has no `jti` (issued before this fix), THE SYSTEM SHALL still accept it exactly as before — `jti` is never required at validation time. (Không làm vô hiệu hàng loạt token đang sống.)
5. WHEN `ValidateToken` or `ParseToken` succeeds on a token that has a `jti`, THE SYSTEM SHALL expose it as `claims.CustomClaims.ID` without any new field or method.
6. WHEN `RefreshAccessToken` issues a new access token, THE SYSTEM SHALL give it a `jti` different from the refresh token's `jti` and from every previously issued token.
7. WHEN a caller passes an `extra` map containing a key named `jti`, `id` or `ID`, THE SYSTEM SHALL keep the generated `jti` untouched — `extra` lives under the `extra` claim and can never override a registered claim.
8. THE SYSTEM SHALL keep the public `Service` interface, `Config`, and every existing method signature unchanged, so that `MockService` and all current callers compile without modification.

## API contract (confirmed surface for this feature)

Không có HTTP endpoint. Bề mặt Go không đổi (AC8); thay đổi chỉ quan sát được ở nội dung token:

| Thứ | Trước | Sau |
|---|---|---|
| Claim `jti` trong token mới | vắng | luôn có, 32 ký tự hex |
| `claims.CustomClaims.ID` sau `ValidateToken` | `""` | giá trị `jti` (token mới) hoặc `""` (token cũ) |
| Token cũ (không `jti`) | hợp lệ | **vẫn hợp lệ** (AC4) |

## Out of scope (this feature boundary, not tracked here)

- **Denylist / thu hồi access token theo `jti`** — `jti` chỉ tạo *khả năng* nhận diện token; cơ chế tra `jti` mỗi request là quyết định của feature dùng nó (xem `login/requirements.md` Pending).
- Lưu `jti` xuống DB — `user_sessions` vẫn lưu hash của cả token; không thêm cột.
- Đổi thuật toán ký, khoá, hoặc `Config` (không thêm field cấu hình cho `jti`).
- Middleware (`pkg/jwt/middleware`) — không đổi hành vi.

## Pending (not yet implemented)

Không còn. AC1–AC8 đã đạt; việc này **không còn chặn** phần `jwt` của `login/tasks.md` Phase 0 (login vẫn còn bị chặn bởi `onboarding/tasks.md` Phase 7).

## Ghi chú kỹ thuật liên quan

- `jwt.RegisteredClaims.ID` là field chuẩn của `github.com/golang-jwt/jwt/v5`, đã được `claims.CustomClaims` nhúng sẵn (`claims/claims.go`) → **không cần sửa `claims/`**, chỉ set giá trị ở hai hàm `Generate*` (`RefreshAccessToken` đi qua `GenerateAccessToken` nên tự hưởng).
- Lý do 128 bit / hex: đủ để xác suất trùng ≈ 0 (birthday bound ~2⁶⁴ token) và không cần thư viện UUID mới — repo đề cao "Minimalism: ưu tiên stdlib".
- `RefreshAccessToken` hiện truyền `refreshClaims.Extra` sang access token mới (`jwt.go:170`) — hành vi giữ nguyên; chỉ lưu ý rằng nếu caller từng nhét `sid`/`ver` vào `extra` của refresh token thì chúng sẽ đi theo access token. Với fix này login **không** cần `extra` để chống trùng nữa.
- Callers hiện tại của `Generate*`: `services/appService/usecase/register_usecase.go` (2 chỗ) + test mock; `pkg/jwt/middleware/*_test.go`. Không chỗ nào so sánh chuỗi token cố định → không vỡ test do token đổi mỗi lần gọi.

## Open questions (resolved during implementation)

- ~~Sửa ở `pkg/jwt` hay để từng feature tự chèn nonce vào `extra`?~~ → **Sửa ở `pkg/jwt`** (yêu cầu của anh 2026-09-20: tách thành tài liệu riêng để implement fix). Resolved 2026-09-20.
