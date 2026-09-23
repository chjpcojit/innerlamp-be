# Tasks — jwt-token-uniqueness

Version: 1.3
Last updated: 2026-09-22
Status: implemented — Phase 1–4 xong và đã verify (2026-09-21). `go test -short ./...` xanh toàn repo từ 2026-09-21 (3 test `pkg/utils` đỏ có sẵn đã sửa ở `json-body-hardening`). Không còn marker `⚠️ Not confirmed`.
Module: `pkg/jwt`

> See `changelog.md` in this folder for the version history of this spec.

<!--
Mọi item trace về một dòng trong bảng "Layers & components" hoặc "Test plan" của design.md.
Item `[x]` = code có thật + test pass, đã chạy lại 2026-09-21.
Theo TDD (skill golang-clean-arch-tdd): viết test AC1 trước, thấy nó FAIL, rồi mới sửa.
-->

## Phase 1 — Test trước (red)

- [x] `TestGenerate_SameSecondDistinct` (AC1) — nằm ở **`pkg/jwt/jti_test.go`** (file mới, thay vì `jwt_test.go`, để không phải đụng import block của file cũ). Đã chạy và thấy **FAIL** trước khi sửa: `Should not be: "eyJhbGci...` cho cả refresh và access, HS256 lẫn RS256 (bằng chứng bug). Để test biên dịch được mà chưa sửa logic, `newTokenID` được khai báo trước nhưng chưa nối vào `Generate*`.
- [x] `TestGenerate_HasJTI` (AC2), `TestGenerate_JTIEntropyFailure` (AC3), `TestValidateToken_LegacyTokenWithoutJTI` (AC4), `TestValidateToken_ExposesJTI` (AC5), `TestRefreshAccessToken_NewJTI` (AC6), `TestGenerate_ExtraCannotOverrideJTI` (AC7) — cùng file. Trước fix: 6/7 test fail; riêng AC4 **pass sẵn** (token cũ vốn hợp lệ — đúng kỳ vọng, test này là lưới an toàn hồi quy, không phải bằng chứng bug). AC1/AC2/AC5/AC6 chạy cho cả HS256 và RS256.

## Phase 2 — Sửa `pkg/jwt` (green)

- [x] `pkg/jwt/jwt.go` — `newTokenID` (16 byte `crypto/rand` → hex 32), biến cấp package để test bơm lỗi; hằng `tokenIDBytes = 16`
- [x] `GenerateAccessToken` và `GenerateRefreshToken` gán `RegisteredClaims.ID`; lỗi sinh `jti` → `fmt.Errorf("generate token id: %w", err)`, không ký token
- [x] Xác nhận `claims/claims.go` và `mock.go` **không cần sửa** (`git status` chỉ có `jwt.go`, 2 README, `jti_test.go`) (AC5, AC8); không thêm điều kiện `jti` vào `CustomClaims.Validate()` (AC4)
- [x] `go test ./pkg/jwt/...` xanh, gồm cả `pkg/jwt/middleware`; thêm `go test -race -count=3` trên các test mới — xanh, không flaky

## Phase 3 — Hồi quy toàn repo

- [x] `go build ./...`, `go vet ./...` sạch; `go test -short ./...` — mọi package xanh **trừ** đúng 3 test `pkg/utils` đã hỏng từ trước (`TestDecodeJSONBody_UnknownField_Disallowed`, `_MultipleJSONObjects`, `_WrongContentType`), không lỗi mới. **Cập nhật 2026-09-21:** 3 test đó đã xanh nhờ `docs/specs/features/json-body-hardening/`; `go test -short -count=1 ./...` giờ xanh **toàn repo**, không còn ngoại lệ.
- [x] `services/appService/usecase` (dùng `MockService`) xanh không sửa dòng nào — AC8

## Phase 4 — Tài liệu

- [x] `pkg/jwt/README.md` + `pkg/jwt/claims/README.md` — thêm `jti`, ghi token cũ không có `jti` vẫn hợp lệ
- [x] Ghi vào `changelog.md` (v1.1): OKF concept nào dưới `docs/knowledge/` bị ảnh hưởng (hiện không có concept `pkg/jwt` — ghi rõ "không có") và mở khoá `login/tasks.md` Phase 2–3
- [x] `docs/ARCHITECTURE.md` — đã grep `jti|CustomClaims|pkg/jwt`: chỉ có link tới README và ghi chú gap middleware, **không** mô tả claim → không cần sửa

## Deferred (separate feature, not blocking jwt-token-uniqueness)

- Denylist / tra session để thu hồi access token theo `jti` (chờ marker ở `login/requirements.md`)
- Cơ chế xoay khoá ký (`kid`)
