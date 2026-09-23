# Tasks — json-body-hardening

Version: 1.5
Last updated: 2026-09-22
Status: done — Phase 1–5 implement và verify xong hết (2026-09-21). Mọi AC có test chứng minh (đỏ trên HEAD `0475281`, xanh sau khi sửa). Không còn marker `⚠️ Not confirmed`. Chỉ còn mục Deferred (feature riêng). (v1.5: dọn tham chiếu `docs/ARCHITECTURE.md` đã bị xoá; không đổi code/AC.)
Module: `pkg/utils`

> See `changelog.md` in this folder for the version history of this spec.

<!--
Mọi item dưới đây trace về một dòng trong bảng "Layers & components" của design.md hoặc một bước trong
"Step-by-step Flow" của nó. Không có task mồ côi.

Item `[x]` = đã verify (code có thật + test tương ứng pass). Viết spec KHÔNG phải là implement.

Thứ tự build theo TDD (`.claude/skills/golang-clean-arch-tdd`): viết test đỏ trước, thấy nó FAIL,
rồi mới sửa code. Ở đây `pkg/utils` là package lá (không layer), nên không có Phase Domain/Usecase —
thay bằng Phase Test-đỏ → Phase Sửa → Phase Handler → Phase Hồi quy.
-->

## Phase 1 — Test trước (red)

- [x] `pkg/utils/http_test.go` — AC1: `TestDecodeJSONBody_UnknownField_Disallowed` giữ nguyên assertion
      `field == "unknown_field"`; đỏ trên HEAD (`expected field 'unknown_field', got ''`).
- [x] `pkg/utils/http_test.go` — AC3: `TestDecodeJSONBody_MultipleJSONObjects` giữ nguyên; thêm
      `TestDecodeJSONBody_TrailingGarbage` (subtest `second object`, `trailing garbage`, `trailing comma`)
      và `TestDecodeJSONBody_TrailingWhitespace` (whitespace đuôi vẫn hợp lệ).
- [x] `pkg/utils/http_test.go` — AC4: `_WrongContentType` (subtest `text/plain`,
      `application/x-www-form-urlencoded` → `ErrInvalidJSON`), `_ContentTypeJSONSuffix`
      (`application/vnd.api+json`), `_ContentTypeCaseInsensitive` (`APPLICATION/JSON`),
      `_MalformedContentType` (`application/json; charset=` → `ErrInvalidJSON`). Cả 2 subtest sai đều đỏ
      trên HEAD (`got nil`).
- [x] `pkg/utils/http_test.go` — AC5: `TestDecodeJSONBody_NoContentType` vẫn xanh sau khi sửa.
- [x] `pkg/utils/http_test.go` — AC6: `TestDecodeJSONBody_TruncatedAtValidBoundary` tái hiện đúng defect [D]:
      `MaxBodySize` = `len({"email":"a@b.c"})`, body là object đó + (`trailing whitespace` | `second object`
      | `trailing garbage`) → phần đọc được là JSON hợp lệ trọn vẹn. Trên HEAD cả 3 subtest **decode thành
      công âm thầm** (`decoded {Email:a@b.c ...}`); code mới trả `ErrBodyTooLarge`. (v1.1 cắt ở
      `{"email": "ab"` — không phải JSON hợp lệ, không tái hiện defect — đã thay.) Kèm
      `_BodyExactlyAtLimit` (đúng bằng ngưỡng → hợp lệ) và `_BodyTooLarge` chỉ chấp nhận `ErrBodyTooLarge`.
- [x] Bằng chứng đỏ (tái hiện 2026-09-21 bằng cách chạy `http_test.go` hiện tại trên `http.go` ở HEAD
      `0475281`, worktree tạm): `FAIL chjpcoj.com.innerlamp/pkg/utils` — 7 test fail:
      `_UnknownField_Disallowed` (`got ''`), `_BodyTooLarge` (`got invalid JSON`),
      `_MultipleJSONObjects` (`got nil`), `_WrongContentType` (`got nil`), `_MalformedContentType`
      (`got nil`), `_TrailingGarbage` (cả 3 subtest `got nil`), `_TruncatedAtValidBoundary`
      (`got invalid JSON`). Sau khi sửa: `ok chjpcoj.com.innerlamp/pkg/utils`.
      Bổ sung v1.2 (cùng cách, HEAD `0475281`): `_WrongContentType/application/x-www-form-urlencoded`
      (`got nil`), `_TruncatedAtValidBoundary` 3 subtest (`decoded {Email:a@b.c ...}`).

## Phase 2 — Sửa `pkg/utils/http.go` (green)

- [x] `pkg/utils/http.go` — AC4: cổng Content-Type chạy **đầu tiên** (`isJSONContentType`, dùng
      `mime.ParseMediaType`): thiếu header → cho qua; không phải `application/json` và không có hậu tố
      `+json`, hoặc header không parse được → `ErrInvalidJSON` message `unsupported content type`.
- [x] `pkg/utils/http.go` — AC6: đọc qua `io.LimitReader(r.Body, opts.MaxBodySize+1)`, rồi
      `len(data) > opts.MaxBodySize` → `ErrBodyTooLarge`. `MaxBodySize <= 0` vẫn = không giới hạn.
- [x] `pkg/utils/http.go` — AC1: `unknownFieldName` parse tên field từ message `json: unknown field "x"`.
      `ErrUnknownField` vẫn là sentinel.
- [x] `pkg/utils/http.go` — AC2: nhánh "cannot unmarshal" dùng `errors.As` sang `*json.UnmarshalTypeError`.
- [x] `pkg/utils/http.go` — AC3: sau `decoder.Decode(dst)` gọi `decoder.Decode(&struct{}{})`; khác `io.EOF`
      → `ErrInvalidJSON`.
- [x] Thứ tự cổng kiểm đúng design.md: Content-Type → đọc body → vượt ngưỡng → rỗng → decode → rác đuôi
      (ghi trong doc comment `DecodeJSONBody`).
- [x] AC8: `go build ./...` sạch. `git diff pkg/utils/http.go` chỉ thêm 2 hàm unexported
      (`isJSONContentType`, `unknownFieldName`) — không đổi chữ ký public, không thêm sentinel.
- [x] AC9 hồi quy: `go test -count=1 ./pkg/utils/...` xanh toàn package, gồm các test cũ.

## Phase 3 — Handler mapping 413 (AC7)

- [x] `services/appService/delivery/http/errors.go` — `writeDecodeError`: nhánh
      `errors.Is(de.Err, utils.ErrBodyTooLarge)` → `utils.WriteBusinessError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", de.Message, nil)`,
      đặt trước nhánh 400. (v1.0 của file này ghi `WriteErrorResponse` — sai so với `design.md`, đã sửa.)
- [x] `services/appService/delivery/http/errors.go` — comment đầu `writeDecodeError` khớp code: 6 case,
      5 case → `400 VALIDATION_ERROR`, body quá lớn → `413`.
- [x] Test `413 PAYLOAD_TOO_LARGE` (body > 1 MB, usecase không bị gọi): `register_handler_test.go`
      (`TestRegisterHandler_RequestOTP_BodyTooLarge`), `login_handler_test.go` (subtest 413 trong
      `TestLoginHandler_Login` và `TestLoginHandler_Refresh`). Các nhánh decode failure khác vẫn
      `400 VALIDATION_ERROR` (test "body errors are 400" / "bad bodies are 400" giữ nguyên, xanh).
- [x] `go test -count=1 ./services/appService/...` xanh.

## Phase 4 — Hồi quy toàn repo

- [x] `go build ./...`, `go vet ./...` sạch.
- [x] `go test -short -count=1 ./...` **xanh toàn bộ, lần đầu tiên** (2026-09-21) — 3 test `pkg/utils`
      đỏ có sẵn đã xanh. Đây là mục mở của cả `login/tasks.md` Phase 5 lẫn `jwt-token-uniqueness/tasks.md`
      Phase 3 (việc tick bên đó nằm ở Phase 5 dưới đây).
- [x] `go test ./services/appService/...` xanh.
- [x] `scripts/check-architecture.sh` → `ARCH_ERRORS=0`;
      `.claude/skills/golang-clean-arch-tdd/scripts/check_dependency_rule.sh services/appService` →
      `All dependency-direction checks passed.`

## Phase 5 — Tài liệu & cross-file

- [x] `pkg/utils/http.go` doc comment của `DecodeJSONBody` — liệt kê các cổng kiểm và thứ tự (6 bước:
      tách "đọc body" và "kiểm vượt ngưỡng" thành 2 bước).
- [x] `pkg/utils/README.md` — mục `DecodeJSONBody` liệt kê 6 cổng kiểm theo thứ tự (Content-Type chấp nhận
      `application/json` + `+json`, thiếu header cho qua; vượt ngưỡng → `ErrBodyTooLarge` kể cả khi phần đọc
      được là JSON hợp lệ; rác đuôi bị từ chối); thêm `ErrBodyTooLarge` vào danh sách sentinel; mục "Lưu ý"
      thêm: map `ErrBodyTooLarge` sang `413` qua `WriteBusinessError`, không dùng `WriteValidationError`.
- [x] `docs/specs/api/openapi.yaml` (v3.3 → v3.4) — `413` `PAYLOAD_TOO_LARGE` ở cả 5 path có body, đặt đúng
      thứ tự mã status; `ApiErrorResponse.error.code` thêm `PAYLOAD_TOO_LARGE`; mô tả `400` của `/auth/login`
      và `/auth/refresh` ghi thêm rác sau JSON và Content-Type sai. Parse YAML bằng Ruby để kiểm: hợp lệ.
      `ApiErrorResponse` vs `ErrorResponse`: cả 5 khối `413` dùng `ApiErrorResponse` (envelope thật). 3 path
      US-001 vẫn dùng `ErrorResponse` (sai envelope) cho các mã khác — **không sửa ở đây**, là lỗi riêng của
      spec onboarding, ghi thành follow-up trong `changelog.md`. → Đã sửa 2026-09-21 ở `onboarding` v4.3 +
      `openapi.yaml` v3.5: `ErrorResponse` bị xoá, mọi path dùng `ApiErrorResponse`.
- [x] `docs/specs/features/login/tasks.md` Phase 5 + `docs/specs/features/jwt-token-uniqueness/tasks.md`
      Phase 3 — mục `go test -short ./...` đã tick/cập nhật, ghi rõ xanh nhờ spec này (login v1.6, jwt v1.2).
- [x] `docs/specs/features/login/changelog.md` (v1.6) + `jwt-token-uniqueness/changelog.md` (v1.2) — ghi
      3 defect `pkg/utils` đã được sửa ở spec này.
- [x] `docs/ARCHITECTURE.md` — đã kiểm §4: không liệt kê defect decode của `pkg/utils`, không cần sửa.
- [x] Concept OKF: `docs/knowledge/features/register.md` chỉ nhắc "decode + validate" chung chung, không mô tả
      giới hạn body, Content-Type hay mã lỗi decode → **không bị ảnh hưởng**, không cần refresh.

## Deferred (separate feature, not blocking json-body-hardening)

- Giới hạn kích thước ở tầng server (`http.MaxBytesReader`, `Server.ReadTimeout`) — `cmd/main.go` chưa
  cấu hình timeout (`#nosec G114`); gap riêng của server.
- `encoding/json/v2` (`GOEXPERIMENT=jsonv2`) — chưa bật mặc định ở Go 1.26.
- `MaxBodySize` theo từng endpoint — hiện một hằng số 1 MB dùng chung.
- Denylist/thu hồi access token, rate limit theo IP — xem Deferred của `login/tasks.md`.
