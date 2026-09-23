# Changelog — json-body-hardening Spec

Tracks version history for `requirements.md`, `design.md`, and `tasks.md` in this folder. All three
files share one changelog since they change together as one spec unit. Newest version first.

## v1.5 (2026-09-22)

Dọn tham chiếu tới `docs/ARCHITECTURE.md` (file đã bị xoá vì stale). **Không đổi code, không đổi AC.**

- `requirements.md`: mục "Giới hạn kích thước ở tầng server" bỏ câu "xem `docs/ARCHITECTURE.md` §4".
- `design.md`: known gap `ReadTimeout`/`MaxHeaderBytes` bỏ câu trỏ §4.
- `tasks.md`: mục Deferred về `ReadTimeout` bỏ câu trỏ §4.

## v1.4 (2026-09-21)

- `tasks.md`: Phase 5 mục `openapi.yaml` — ghi chú `ErrorResponse` của US-001 đã được sửa. Không tick/untick.
- `requirements.md` / `design.md`: không đổi nội dung, đã kiểm. Chỉ bump header.
- Reason: follow-up "3 path US-001 vẫn trỏ `ErrorResponse`" của v1.3 đã xử lý ở `onboarding` v4.3 / `openapi.yaml`
  v3.5; toàn repo chuẩn hoá envelope ở `../api-response-envelope/`.

## v1.3 (2026-09-21)

- `tasks.md`: tick toàn bộ Phase 5, Status → done.
  - `pkg/utils/README.md`: mô tả 6 cổng kiểm và cách map `ErrBodyTooLarge` → 413.
  - `docs/specs/api/openapi.yaml` v3.4: `413 PAYLOAD_TOO_LARGE` ở 5 path có body (schema `ApiErrorResponse`).
  - `login` v1.6 và `jwt-token-uniqueness` v1.2: tick / cập nhật mục `go test -short ./...`, changelog ghi
    3 defect `pkg/utils` đã sửa.
  - Concept OKF `register.md`: không bị ảnh hưởng (không mô tả decode chi tiết).
- `requirements.md` / `design.md`: không đổi nội dung, đã kiểm. Bump header, Status → done.
- Reason: hoàn tất Phase 5 theo yêu cầu của anh.

### Follow-ups ngoài spec này

- `docs/specs/features/onboarding/` — 3 path US-001 trong `openapi.yaml` vừa có thêm `413`, nhưng changelog
  onboarding chưa ghi. Ngoài ra 3 path đó vẫn trỏ `ErrorResponse` (lồng `details` trong `error`, thiếu
  `success`/`type`) — sai so với envelope thật `utils.ApiResponse`. Đã ghi nhận từ login v1.5, chưa sửa.

## v1.2 (2026-09-21)

- `requirements.md`: không đổi nội dung, đã kiểm. Status → implemented.
- `design.md`: không đổi nội dung, đã kiểm. Chỉ bump header.
- `tasks.md`: đã vá xong 3 lỗ test nêu ở v1.1, tick lại:
  - AC6: viết lại `TestDecodeJSONBody_TruncatedAtValidBoundary` thành 3 subtest. `MaxBodySize` bằng độ dài
    `{"email":"a@b.c"}`, phía sau là whitespace, object thứ hai, hoặc rác. Trên HEAD `0475281` cả 3
    **decode thành công âm thầm** (`decoded {Email:a@b.c ...}`) — đây là lần đầu defect [D] được test
    tái hiện đúng. Code hiện tại trả `ErrBodyTooLarge`.
  - AC4: `_WrongContentType` thành table test, thêm `application/x-www-form-urlencoded`. Trên HEAD đỏ
    (`got nil`).
  - AC7: `login_handler_test.go` thêm subtest 413 cho `Login` và `Refresh`.
- Verify: `go test -count=1 ./pkg/utils/... ./services/appService/delivery/...` xanh; `go vet` sạch;
  `go test -short -count=1 ./...` xanh toàn repo.
- Còn `[ ]`: chỉ Phase 5 (README, `openapi.yaml`, tick bên `login`/`jwt-token-uniqueness`, concept OKF).
- Reason: v1.1 phát hiện test AC6 cũ không chứng minh được defect nặng nhất của spec. Sửa test thay vì
  tick dựa trên một test pass vì lý do khác.

## v1.1 (2026-09-21)

- `requirements.md`: không đổi nội dung, đã kiểm. Chỉ bump header và Status.
- `design.md`: không đổi nội dung, đã kiểm. Bảng Layers khớp code thật, kể cả dòng Delivery ghi
  `utils.WriteBusinessError`. Chỉ bump header và Status.
- `tasks.md`: đồng bộ với code đã implement (không có spec mới). Tick Phase 1–4 sau khi verify lại:
  `go build ./...`, `go vet ./...` sạch; `go test -short -count=1 ./...` xanh toàn repo lần đầu;
  `scripts/check-architecture.sh` → `ARCH_ERRORS=0`; `check_dependency_rule.sh` pass. Ghi bằng chứng đỏ
  tái hiện bằng cách chạy `http_test.go` hiện tại trên `http.go` ở HEAD `0475281`: 7 test fail. Tick
  mục doc comment `DecodeJSONBody` và mục `docs/ARCHITECTURE.md` (§4 không nhắc lỗi decode, không cần sửa).
  Sửa mô tả Phase 3 từ `WriteErrorResponse` thành `WriteBusinessError` cho khớp `design.md` và code.
- Còn `[ ]` sau khi verify:
  - AC4 thiếu case `application/x-www-form-urlencoded`.
  - AC6: `TestDecodeJSONBody_TruncatedAtValidBoundary` cắt ở `{"email": "ab"`, không phải JSON hợp lệ.
    Trên HEAD nó fail với `invalid JSON` chứ không bị nhận âm thầm, nên **chưa tái hiện đúng defect [D]**.
  - AC7 chưa có test 413 cho login.
  - Phase 5 tài liệu: `pkg/utils/README.md`, `openapi.yaml`, tick bên `login`/`jwt-token-uniqueness`,
    concept OKF.
- Reason: code đã implement xong trước khi `tasks.md` được cập nhật (v1.0 vẫn ghi "not started"). Bản
  này đưa spec về đúng trạng thái thật.

## v1.0 (initial — 2026-09-21)

Spec đầu tiên, prospective. **Tách ra từ** `docs/specs/features/login/` theo yêu cầu của anh: khi tick
Phase 5 của login, `go test -short ./...` không xanh được vì 3 test `pkg/utils` đỏ có sẵn; defect nằm
ở package dùng chung nên phải có tài liệu và vòng implement riêng — cùng tiền lệ
`docs/specs/features/jwt-token-uniqueness/`.

- `requirements.md`: 9 AC (EARS). Bốn defect, có mục "Bằng chứng" với kết quả tái hiện thật:
  `[A]` tên field lạ không được parse ra (`*errors.errorString`, không phải `*json.UnmarshalTypeError`);
  `[B]` rác/object thứ hai sau JSON không bị kiểm; `[C]` `Content-Type` không được kiểm;
  `[D]` body vượt `MaxBodySize` bị `io.LimitReader` cắt **âm thầm** — nếu cắt đúng ranh giới JSON hợp lệ
  thì request vẫn được nhận với dữ liệu thiếu, và `ErrBodyTooLarge` chưa bao giờ được trả về.
  `[D]` là defect nặng nhất và chưa từng được ghi nhận trước spec này.
- `design.md`: một file sửa chính (`pkg/utils/http.go`) + một nhánh map lỗi ở `delivery/http/errors.go`;
  bốn phương án đã cân nhắc ghi nội dung decode vào DTO rỗng; bảng map AC → test; thứ tự 5 cổng kiểm
  ghi thành **phần hợp đồng**, không phải chi tiết tự do.
- `tasks.md`: 5 Phase theo TDD (test đỏ trước → sửa → handler 413 → hồi quy → tài liệu), mọi item `[ ]`.
- Quyết định đã chốt qua hỏi anh (2026-09-21): **Content-Type** chấp nhận `application/json` và mọi
  media type có hậu tố `+json`; thiếu hẳn header vẫn lenient (giữ `TestDecodeJSONBody_NoContentType`).
  **Body vượt ngưỡng** → `413`, code `PAYLOAD_TOO_LARGE`.
- Reason: defect ở package dùng chung, mọi handler trong `services/appService` đều gọi
  `utils.DecodeJSONBodyStrict` → sửa ở gốc thay vì chỉnh test cho khớp hành vi sai (phương án B bị loại).

### Cross-file follow-ups

- `docs/specs/api/openapi.yaml` — thêm `413 PAYLOAD_TOO_LARGE` cho 5 path có body.
- `docs/specs/features/login/` — Phase 5 mục `go test -short ./...` tick được sau spec này; changelog
  login v1.5 đang ghi 3 test `pkg/utils` là "phát hiện ngoài phạm vi, chưa sửa" → cần cập nhật.
- `docs/specs/features/jwt-token-uniqueness/` — cùng mục `go test -short ./...`; tasks.md Phase 3 đang
  ghi "chấp nhận 3 test `pkg/utils` đỏ".
- `pkg/utils/README.md` — cập nhật hành vi decode.
- `docs/specs/db/`, `migrations/` — **không ảnh hưởng** (không đổi schema).
- `docs/knowledge/` — kiểm `docs/knowledge/features/register.md` xem có mô tả decode/body-size không;
  nếu có thì `bdd-okf-docs` refresh.
