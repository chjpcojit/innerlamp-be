# claims

`CustomClaims` — claims của mọi token do `pkg/jwt` sinh ra: `user_id`, `email`, `role`, `token_type` (`access` | `refresh`), `extra`, cộng `jwt.RegisteredClaims` nhúng sẵn.

## Contents

- `claims.go` — `CustomClaims`, `TokenType`, `Validate()`.

## Notes

- `jti` (JWT ID) là **`RegisteredClaims.ID`** — không có field riêng. `pkg/jwt` luôn set nó khi sinh token (32 ký tự hex, 128 bit `crypto/rand`) để mỗi token duy nhất; xem [`../README.md`](../README.md). Token cũ không có `jti` (`ID == ""`) vẫn hợp lệ, nên `CustomClaims.Validate()` **không** được đòi `jti`.
- `Extra` là map do caller tuỳ ý điền, nằm dưới claim `extra`; không thể ghi đè các registered claim (kể cả `jti`).
- Update this README when the folder structure or responsibilities change.
