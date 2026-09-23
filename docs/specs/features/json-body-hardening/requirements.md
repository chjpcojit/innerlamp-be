# Requirements — json-body-hardening

Version: 1.5
Last updated: 2026-09-22
Status: done (2026-09-21) — code sửa xong, cả 9 AC có test chứng minh, tài liệu cross-file đã cập nhật. Nội dung requirements không đổi từ v1.0. (v1.5: dọn tham chiếu `docs/ARCHITECTURE.md` đã bị xoá; không đổi code/AC.)
Module: `pkg/utils` (hạ tầng dùng chung — không thuộc `identity`)
Code: `pkg/utils/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này **tách ra từ** `docs/specs/features/login/` (phát hiện khi tick Phase 5 của login: `go test -short ./...` không xanh được vì 3 test `pkg/utils` đỏ có sẵn). Giống tiền lệ [`../jwt-token-uniqueness/`](../jwt-token-uniqueness/requirements.md): defect nằm ở package dùng chung, mọi handler trong `services/appService` đều gọi `utils.DecodeJSONBodyStrict`, nên sửa ở gốc thay vì để từng handler tự vá hoặc chỉnh test cho khớp hành vi sai.

Spec này **không** thuộc `identity` và không đổi AC nào của `login` hay `onboarding`.

## Bằng chứng (tái hiện 2026-09-21)

Chạy trực tiếp `DecodeJSONBody` / `DecodeJSONBodyStrict` trong `pkg/utils` (không sửa file nào trong repo):

```
[A] unknown field với DisallowUnknownFields
    err  = `json: unknown field "nope"`
    type = *errors.errorString          ← KHÔNG phải *json.UnmarshalTypeError
    ⇒ DecodeError.Field = ""            (tên field có trong message nhưng không được parse ra)

[B] rác phía sau JSON (không được kiểm)
    `{"email":"x"}`                 Decode lần 2 = io.EOF   → hợp lệ   (đúng)
    `{"email":"x"}{"email":"y"}`    Decode lần 2 = nil      → ĐƯỢC CHẤP NHẬN  (sai)
    `{"email":"x"}garbage`          Decode lần 2 = syntax   → ĐƯỢC CHẤP NHẬN  (sai)

[C] Content-Type (không được kiểm)
    `text/plain`                        → err = <nil>   (sai)
    `application/x-www-form-urlencoded`  → err = <nil>   (sai)
    `application/json; charset=utf-8`    → err = <nil>   (đúng)
    thiếu hẳn header                     → err = <nil>   (đúng — giữ lenient, xem AC4)

[D] body vượt MaxBodySize (io.LimitReader cắt âm thầm)
    cắt giữa chừng, JSON hỏng     → ErrInvalidJSON   (khớp test hiện có)
    cắt đúng ranh giới JSON hợp lệ → err = <nil>, dst nhận dữ liệu ĐÃ MẤT  (sai)
```

Case `[D]` dòng thứ hai là defect nặng nhất và chưa từng được ghi nhận: `ErrBodyTooLarge` (`pkg/utils/http.go:25`) được khai báo nhưng **không bao giờ được trả về** — nó là hằng số chết. Body bị cắt không báo, nên request quá lớn vẫn có thể được xử lý với payload thiếu.

## User Story

> Là một **developer viết HTTP handler trong repo này**,
> Tôi muốn **`utils.DecodeJSONBody` thật sự từ chối đúng những gì doc của nó nói là từ chối, và báo đúng field cùng lý do**,
> Để **không phải tự kiểm Content-Type, kích thước body và rác phía sau JSON trong từng handler, và không phải chịu rủi ro JSON parser differential khi service ra sau proxy/WAF**.

## Acceptance Criteria (EARS notation)

1. WHEN `DecodeJSONBody` is called with `DisallowUnknownFields` and the body contains a field that is not present in `dst`, THE SYSTEM SHALL return a `*DecodeError` whose `Field` is that field's **JSON name as sent by the client** and whose `Err` is `ErrUnknownField`. (Tên field nằm sẵn trong message `json: unknown field "x"`; hiện code type-assert sang `*json.UnmarshalTypeError` nên luôn ra chuỗi rỗng.)
2. WHEN the body fails to decode because a value has the wrong type for its target field, THE SYSTEM SHALL return `ErrInvalidFieldType` with `Field` set to that field's JSON name — hành vi hiện có, giữ nguyên (dùng `errors.As` thay cho type assert trực tiếp).
3. WHEN the body contains a complete JSON value followed by any further non-whitespace content (a second object, a second array, trailing garbage), THE SYSTEM SHALL return `ErrInvalidJSON` and SHALL NOT return decoded data to the caller. (Whitespace đuôi vẫn hợp lệ.)
4. WHEN the request carries a `Content-Type` header whose media type is neither `application/json` nor a `+json` suffix type (`application/vnd.api+json`, …), THE SYSTEM SHALL return `ErrInvalidJSON` with the message `unsupported content type` and SHALL NOT decode the body. (So sánh trên media type đã chuẩn hoá qua `mime.ParseMediaType`, nên `APPLICATION/JSON` và `application/json; charset=utf-8` đều được chấp nhận; header hỏng cú pháp bị coi là không hợp lệ.)
5. WHEN the request carries **no** `Content-Type` header at all, THE SYSTEM SHALL decode the body exactly as before — thiếu header vẫn lenient. (Giữ nguyên hành vi `TestDecodeJSONBody_NoContentType` đang chốt.)
6. WHEN the request body is longer than `DecodeOptions.MaxBodySize` bytes (default 1 MB), THE SYSTEM SHALL return `ErrBodyTooLarge` and SHALL NOT return decoded data — kể cả khi phần đã đọc được cắt đúng ranh giới của một JSON hợp lệ. (`MaxBodySize <= 0` tiếp tục nghĩa là không giới hạn.)
7. WHEN a handler under `services/appService/delivery/http` receives a decode failure whose `Err` is `ErrBodyTooLarge`, THE SYSTEM SHALL respond `413` with error code `PAYLOAD_TOO_LARGE` in the shared `utils.ApiResponse` envelope. Mọi decode failure khác giữ nguyên `400` / `VALIDATION_ERROR`.
8. THE SYSTEM SHALL keep the exported surface of `pkg/utils` unchanged — `DecodeJSONBody`, `DecodeJSONBodyStrict`, `DecodeOptions`, `DecodeError`, `DefaultDecodeOptions`, `IsDecodeError`, `GetDecodeErrorDetails` giữ nguyên chữ ký và ngữ nghĩa; không thêm sentinel mới.
9. WHEN the body is empty or whitespace-only, malformed, `null`, an array, a map, or a nested struct, THE SYSTEM SHALL behave exactly as today — `ErrEmptyBody` / `ErrInvalidJSON` / decode thành công như cũ. Không có nhánh nào đang xanh bị đổi kết quả.

## API contract (confirmed surface for this feature)

Feature này không thêm endpoint. Nó đổi response của **mọi** route có body trong `services/appService` (3 route US-001 + 2 route US-002 có body; `logout` không có body):

| Method | Path | Trước | Sau |
|---|---|---|---|
| POST | `/v1/auth/register/request-otp` | 400 `VALIDATION_ERROR` cho mọi lỗi decode | thêm `413` `PAYLOAD_TOO_LARGE`; `400` giữ nguyên |
| POST | `/v1/auth/resend-otp` | nt | nt |
| POST | `/v1/auth/register/verify-otp` | nt | nt |
| POST | `/v1/auth/login` | nt | nt |
| POST | `/v1/auth/refresh` | nt | nt |

Bốn thay đổi quan sát được ở client, tất cả đều trên envelope `utils.ApiResponse` sẵn có:

1. Body vượt 1 MB → `413` `PAYLOAD_TOO_LARGE` (trước: `400` hoặc — với ranh giới "may mắn" — `200` với dữ liệu thiếu).
2. Tên field lạ xuất hiện trong `details.field` (trước: `details` vắng).
3. `Content-Type` không phải JSON → `400` `VALIDATION_ERROR` (trước: được xử lý bình thường).
4. Rác sau JSON → `400` `VALIDATION_ERROR` (trước: được xử lý bình thường).

## Out of scope (this feature boundary, not tracked here)

- **Hạ tầng chống JSON parser differential** (proxy/WAF/CDN đồng bộ cách parse) — spec này chỉ đóng lỗ hổng phía service; phần hạ tầng không thuộc repo.
- **CSRF** — việc `text/plain` / `form-urlencoded` từ chối được một vector simple-request, nhưng mọi route hiện dùng `Authorization: Bearer` chứ không cookie, nên CSRF vẫn ngoài scope.
- **Giới hạn body riêng cho từng endpoint** — `MaxBodySize` vẫn là một hằng số 1 MB dùng chung; không thêm cấu hình theo route.
- **`encoding/json/v2` (`GOEXPERIMENT=jsonv2`)** — chưa bật mặc định ở Go 1.26 và sẽ đổi ngữ nghĩa decode toàn repo.
- **Giới hạn kích thước ở tầng server** (`http.MaxBytesReader`, `Server.ReadTimeout`) — `cmd/main.go` hiện chưa cấu hình timeout nào (`#nosec G114`); vẫn là gap riêng của server.
- Không sửa `pkg/jwt`, `pkg/validation` hay bất kỳ package nào khác ngoài `pkg/utils`.

## Pending (not yet implemented)

Toàn bộ — spec prospective. Theo `tasks.md`:

- Phase 1 — bốn test đỏ mới trong `pkg/utils/http_test.go` (AC1–AC6).
- Phase 2 — sửa `pkg/utils/http.go` (AC1–AC6, AC8, AC9).
- Phase 3 — map `413 PAYLOAD_TOO_LARGE` ở `services/appService/delivery/http/errors.go` + test handler (AC7).
- Phase 4 — hồi quy toàn repo: `go test -short ./...` phải xanh **toàn bộ**, lần đầu tiên.
- Phase 5 — tài liệu: `pkg/utils/README.md`, `docs/specs/api/openapi.yaml`.

## Ghi chú kỹ thuật liên quan

- **Vì sao không thêm sentinel mới:** `ErrBodyTooLarge` **đã** tồn tại sẵn trong `pkg/utils` và đã có test chờ nó — chỉ chưa bao giờ được trả về. AC6 chỉ nối dây. Với lỗi Content-Type, `TestDecodeJSONBody_WrongContentType` đã chốt kỳ vọng là `ErrInvalidJSON`, và ở tầng HTTP mọi decode failure (trừ 413) đều map về cùng `400 VALIDATION_ERROR` — nên một `Message` riêng (`unsupported content type`) đủ để client phân biệt, không cần mở rộng bề mặt lỗi export (AC8).
- **Nguyên tắc phân biệt lỗi:** chỉ tách sentinel khi **response HTTP phải khác**. Đó là lý do duy nhất `ErrBodyTooLarge` cần nhận diện riêng ở handler (413 vs 400).
- **Thứ tự các cổng kiểm trong `DecodeJSONBody` là một phần hợp đồng**, không phải chi tiết tự do: Content-Type → đọc body (cap `MaxBodySize + 1`) → kiểm vượt ngưỡng → kiểm rỗng → decode → kiểm rác đuôi. Đảo thứ tự sẽ đổi lỗi trả về cho cùng một request (ví dụ body rỗng + Content-Type sai hiện trả `ErrEmptyBody`).
- **Vì sao `MaxBodySize + 1`:** `io.LimitReader(r.Body, n)` cắt âm thầm và không cho biết đã cắt. Đọc thêm đúng một byte là cách phân biệt "vừa đủ ngưỡng" với "vượt ngưỡng" mà không cần `http.MaxBytesReader` (thứ sẽ đóng connection và đổi hành vi transport).
- **Vì sao kiểm rác đuôi bằng `Decode` lần hai vào `&struct{}{}` rồi so `io.EOF`,** không dùng `decoder.More()`: `More()` được thiết kế cho phần tử trong array/object, không phải cho stream ở top level. Đo thực tế: body hợp lệ một object → lần decode thứ hai trả `io.EOF`; body có thêm giá trị → trả `nil` hoặc lỗi cú pháp.
- **Phạm vi ảnh hưởng thật:** 5 call site `utils.DecodeJSONBodyStrict` (`register_handler.go:117/148/178`, `login_handler.go:69/102`). Không call site nào ở ngoài `services/appService`; `pkg/utils` không có consumer nào khác trong repo.
- **Quan hệ với hai spec khác:** spec này là mục `go test -short ./...` sạch của cả `login/tasks.md` Phase 5 lẫn `jwt-token-uniqueness/tasks.md` Phase 3 — cả hai đang ghi rõ "chấp nhận 3 test `pkg/utils` đỏ có sẵn". Sau khi spec này xong, cả hai mục đó tick được.
- Dependency rule không bị ảnh hưởng: `pkg/utils` là leaf, không import layer nào của `services/`.

## Open questions (resolved during implementation)

- ~~`Content-Type` nào được chấp nhận cho request có body JSON?~~ → **`application/json` và mọi media type có hậu tố `+json`**; thiếu hẳn header vẫn cho qua (giữ `TestDecodeJSONBody_NoContentType`). Resolved 2026-09-21.
- ~~Body vượt `MaxBodySize` nên trả status nào?~~ → **`413`**, code `PAYLOAD_TOO_LARGE`. Resolved 2026-09-21.
