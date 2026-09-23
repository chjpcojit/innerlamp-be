# Changelog — jwt-token-uniqueness Spec

Tracks version history for `requirements.md`, `design.md`, and `tasks.md` in this folder. All three
files share one changelog since they change together as one spec unit. Newest version first.

## v1.3 (2026-09-22)

Sửa tài liệu, **không đổi code, không đổi AC**.

- `design.md`: header bảng "Test plan" ghi `pkg/jwt/jwt_test.go` → **`pkg/jwt/jti_test.go`**, khớp
  tiêu đề mục và bảng Layers (đã đúng từ v1.1).
- `requirements.md` / `tasks.md`: chỉ bump header cùng version.

## v1.2 (2026-09-21)

- `tasks.md`: Phase 3 mục hồi quy ghi thêm — ngoại lệ "3 test `pkg/utils` đã hỏng từ trước" không còn;
  `go test -short -count=1 ./...` xanh toàn repo. Status cập nhật tương ứng. Không item nào đổi trạng thái.
- `requirements.md` / `design.md`: không đổi nội dung, đã kiểm. Chỉ bump header.
- Reason: 3 test đó đã được sửa ở spec riêng `docs/specs/features/json-body-hardening/` (v1.2).

## v1.1 (2026-09-21)

Implement xong Phase 1–4 (TDD: test đỏ → sửa → hồi quy → tài liệu).

- Code: `pkg/jwt/jwt.go` (`newTokenID` + `jti` ở `GenerateAccessToken`/`GenerateRefreshToken`),
  `pkg/jwt/jti_test.go` (7 test, thêm 8 subtest HS256/RS256), `pkg/jwt/README.md`, `pkg/jwt/claims/README.md`.
  `claims/claims.go`, `mock.go`, `middleware/`, `Config`, `Service` **không đổi** (AC8).
- Bằng chứng đỏ → xanh: trước fix, `TestGenerate_SameSecondDistinct` fail cho cả refresh/access,
  HS256/RS256, và 5 test khác fail; sau fix cả 7 test pass. Chạy thêm `-race -count=3` — xanh.
- Hồi quy: `go build ./...` + `go vet ./...` sạch; `go test -short ./...` xanh trừ đúng 3 test
  `pkg/utils` đã hỏng từ trước (không liên quan, không bị chạm).
- `requirements.md` / `design.md`: chỉ đổi Status + version; hai chi tiết thực thi ghi lại (test nằm ở
  `jti_test.go` thay vì `jwt_test.go`; `newTokenID` khai báo trước, chưa nối, để có test đỏ hợp lệ).
- `tasks.md`: Phase 1–4 tick `[x]` kèm ghi chú verify. **Không** có item nào còn `[ ]` ngoài Deferred.
- Reason: yêu cầu của anh — bắt đầu bước 1 trong thứ tự implement (`jwt-token-uniqueness` → `onboarding`
  Phase 7 → `login`).

### Cross-file follow-ups

- `docs/specs/features/login/tasks.md` Phase 0 — mục `jwt-token-uniqueness` đã tick; còn `onboarding` Phase 7.
- `docs/knowledge/` — không có concept `pkg/jwt` nào để refresh.
- `docs/ARCHITECTURE.md` — không cần sửa (đã kiểm tra).
- Token phát hành **trước** fix (không `jti`) vẫn hợp lệ tới khi hết hạn (≤ 7 ngày với refresh) — chủ ý (AC4).

## v1.0 (initial — 2026-09-20)

Spec đầu tiên, prospective. **Tách ra từ** `docs/specs/features/login/` v1.0 theo yêu cầu của anh:
lỗi nằm ở package dùng chung `pkg/jwt` nên phải có tài liệu và vòng implement riêng.

- `requirements.md`: 8 AC (EARS) — token duy nhất cho cùng đầu vào trong cùng giây, `jti` 128 bit từ
  `crypto/rand`, lỗi entropy không được ký token, token cũ không `jti` vẫn hợp lệ, `extra` không ghi đè
  được `jti`, interface không đổi. Có mục "Bằng chứng" với kết quả tái hiện thật.
- `design.md`: một điểm sửa (`pkg/jwt/jwt.go`), bốn phương án đã cân nhắc, map AC → test.
- `tasks.md`: 4 Phase theo TDD (test đỏ trước, sửa, hồi quy, tài liệu), mọi item `[ ]`.
- Reason: khi đặc tả login phát hiện `GenerateRefreshToken` không sinh `jti` và `iat` chỉ chính xác 1
  giây → hai refresh token cùng user trong cùng giây trùng byte, vỡ `UNIQUE(refresh_token_hash)`.
  Tái hiện bằng test tạm qua `go test -overlay` (không đổi repo): refresh trùng `true`, access trùng `true`.

### Cross-file follow-ups

- `docs/specs/features/login/` — **phải cập nhật** (đã làm ở login v1.1): bỏ phương án `sid`/`ver` trong
  `extra`, dùng lại `domain.TokenIssuer` của US-001, thêm phụ thuộc vào spec này.
- `pkg/jwt/README.md`, `pkg/jwt/claims/README.md` — cập nhật khi implement (tasks.md Phase 4).
- `docs/specs/api/openapi.yaml`, `migrations/`, `docs/specs/db/` — **không ảnh hưởng**.
- `docs/knowledge/` — chưa có concept nào về `pkg/jwt`; không có gì để refresh.
