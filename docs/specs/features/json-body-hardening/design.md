# Design — json-body-hardening

Version: 1.5
Last updated: 2026-09-22
Status: implemented (2026-09-21) — bảng Layers dưới đây khớp code thật (`pkg/utils/http.go`, `delivery/http/errors.go`); nội dung design không đổi từ v1.0. Done. (v1.5: dọn tham chiếu `docs/ARCHITECTURE.md` đã bị xoá; không đổi code/AC.)
Module: `pkg/utils`
Code: `pkg/utils/`
Feature folder: `pkg/utils/` + `docs/specs/features/json-body-hardening/`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Bịt bốn lỗ trong `utils.DecodeJSONBody` — hàm mà **mọi** route có body trong `services/appService` gọi qua `DecodeJSONBodyStrict` (5 call site). Ba lỗ đầu là doc nói một đằng code làm một nẻo (comment ở `delivery/http/errors.go:16-22` hứa năm case, code làm ba); lỗ thứ tư (`ErrBodyTooLarge` là hằng số chết, body bị cắt âm thầm) chưa từng được ghi nhận và là lỗ nặng nhất.

Sửa **một** file logic (`pkg/utils/http.go`) cộng một nhánh map lỗi ở `delivery/http/errors.go`. Bề mặt export của `pkg/utils` không đổi (AC8) — không thêm sentinel, không đổi chữ ký. Kiểu sửa giống tiền lệ [`../jwt-token-uniqueness/`](../jwt-token-uniqueness/design.md): defect ở package dùng chung thì sửa ở gốc, không để từng handler tự vá.

`pkg/utils` là leaf của dependency graph — chỉ import stdlib, không import layer nào của `services/`. Dependency rule vì thế không bị ảnh hưởng.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Package logic | `pkg/utils/http.go` | `DecodeJSONBody` chia thành các cổng kiểm **theo thứ tự cố định** (xem Step-by-step Flow): kiểm `Content-Type` → đọc body với cap `MaxBodySize + 1` → kiểm vượt ngưỡng → kiểm rỗng → decode → kiểm rác đuôi (AC3–AC6). |
| Package logic | `pkg/utils/http.go` | Helper mới không export `isJSONContentType(header string) bool`: `mime.ParseMediaType` → so media type với `application/json` hoặc hậu tố `+json`; header rỗng → `true` (AC5); parse lỗi → `false` (AC4). |
| Package logic | `pkg/utils/http.go` | Nhánh unknown field: bỏ type assert sang `*json.UnmarshalTypeError` (không bao giờ đúng với `DisallowUnknownFields`), thay bằng helper `unknownFieldName(msg string) string` parse `json: unknown field "x"` → `x` (AC1). |
| Package logic | `pkg/utils/http.go` | Nhánh sai kiểu field: đổi type assert trực tiếp sang `errors.As` để vẫn lấy được `Field` (AC2) — giữ nguyên kết quả hiện có, chỉ đổi cơ chế. |
| Tests | `pkg/utils/http_test.go` | Bốn test đỏ mới cho AC1–AC6 (bảng "Test plan" bên dưới). **Không sửa** các test đang xanh — chúng là lưới hồi quy cho AC9. |
| Delivery | `services/appService/delivery/http/errors.go` | `writeDecodeError`: thêm nhánh `errors.Is(de.Err, utils.ErrBodyTooLarge)` → `utils.WriteBusinessError(w, 413, "PAYLOAD_TOO_LARGE", …)` **trước** nhánh `400` hiện có; mọi decode failure khác giữ `400 VALIDATION_ERROR` (AC7). Cập nhật comment đầu hàm cho khớp năm case thật. |
| Delivery tests | `services/appService/delivery/http/login_handler_test.go` + `register_handler_test.go` | Thêm case 413 (mock usecase trả `&utils.DecodeError{Err: utils.ErrBodyTooLarge}`) và case `details.field` có tên field lạ. |
| Docs | `pkg/utils/README.md` | Ghi năm cổng kiểm, thứ tự của chúng, và việc thiếu `Content-Type` vẫn lenient. |
| Docs | `docs/specs/api/openapi.yaml` | Thêm response `413` cho 5 path POST có body (US-001 ×3, US-002 ×2). `logout` không có body → không đổi. |

Dependency rule: `delivery → usecase → domain`, `repository → domain` không đổi. `pkg/utils` vẫn là leaf, chỉ thêm import stdlib (`mime`).

## Request flow

```
POST /v1/<path>  (body JSON)
  └─ <RequireAuth | public>
       └─ <Feature>Handler.<Action>                    (delivery/http)
            ├─ utils.DecodeJSONBodyStrict(r, &req)
            │    └─ utils.DecodeJSONBody(r, dst, opts)  (pkg/utils)  ← sửa ở đây
            │         ├─ 1. isJSONContentType(Content-Type)  → ErrInvalidJSON
            │         ├─ 2. io.ReadAll(io.LimitReader(body, Max+1))
            │         ├─ 3. len > Max                        → ErrBodyTooLarge   [MỚI]
            │         ├─ 4. rỗng / toàn khoảng trắng         → ErrEmptyBody
            │         ├─ 5. decoder.Decode(dst)              → ErrUnknownField / ErrInvalidFieldType / ErrInvalidJSON
            │         └─ 6. Decode(&struct{}{}) != io.EOF    → ErrInvalidJSON     [MỚI]
            │    └─ writeDecodeError(w, err)                (delivery/http)
            │         ├─ errors.Is(de.Err, ErrBodyTooLarge)  → 413 PAYLOAD_TOO_LARGE   [MỚI]
            │         └─ còn lại                             → 400 VALIDATION_ERROR
            └─ usecase…  (không tới được nếu decode fail)
```

## Step-by-step Flow — thứ tự cổng kiểm trong `DecodeJSONBody`

Thứ tự này **là một phần hợp đồng**, không phải chi tiết tự do: đảo hai cổng sẽ đổi lỗi trả về cho cùng một request. Cả sáu cổng nằm trong một lời gọi hàm, không có I/O ngoài (không DB, không network), nên **không có transaction boundary** — dừng ở cổng nào là trả lỗi ở cổng đó, `dst` không được đảm bảo nguyên vẹn.

```
Cổng 1 — Content-Type (feature này)
- Trigger: mọi lời gọi DecodeJSONBody
- Pre-condition: —
- Call: isJSONContentType(r.Header.Get("Content-Type")) → mime.ParseMediaType
    - Success (rỗng → lenient theo AC5) → Cổng 2
    - Success (media type == "application/json" hoặc có hậu tố "+json") → Cổng 2
    - Failure (media type khác, hoặc header không parse được) → ErrInvalidJSON,
      Message = "unsupported content type", Err = ErrInvalidJSON; KHÔNG đọc body
  Chốt 2026-09-21: so sánh trên media type đã chuẩn hoá, nên `APPLICATION/JSON` và
  `application/json; charset=utf-8` đều qua (mime.ParseMediaType lowercase hoá và tách tham số).
  Hệ quả có ý thức: body RỖNG + Content-Type SAI giờ trả lỗi Content-Type, không còn ErrEmptyBody.
  Không test nào đang xanh phủ tổ hợp đó (mọi test body rỗng đều set application/json), nên AC9
  không bị vi phạm — nhưng đây là thay đổi hành vi thật, ghi lại để không ai bất ngờ.

Cổng 2 — đọc body có cap (feature này)
- Trigger: Cổng 1 pass
- Pre-condition: —
- Call: io.ReadAll(io.LimitReader(r.Body, MaxBodySize + 1))   (MaxBodySize > 0)
        io.ReadAll(r.Body)                                    (MaxBodySize <= 0 — không giới hạn)
    - Success → Cổng 3
    - Failure (lỗi đọc) → DecodeError{Message: "failed to read request body", Err: err}
  Vì sao `Max+1` chứ không `Max`: io.LimitReader cắt ÂM THẦM và không cho biết đã cắt. Đọc thêm
  đúng một byte là cách duy nhất phân biệt "vừa đủ ngưỡng" với "vượt ngưỡng" mà không cần
  http.MaxBytesReader (thứ sẽ đóng connection và đổi hành vi transport).

Cổng 3 — kiểm vượt ngưỡng (feature này, AC6)
- Trigger: Cổng 2 xong
- Pre-condition: MaxBodySize > 0
- Call: len(data) > MaxBodySize
    - Success (không vượt) → Cổng 4
    - Failure (vượt, DÙ dữ liệu đã đọc cắt đúng ranh giới một JSON hợp lệ) →
      DecodeError{Err: ErrBodyTooLarge}; KHÔNG decode, KHÔNG trả dữ liệu
  Đây là defect `[D]` trong requirements.md: trước fix, nhánh này không tồn tại nên body bị cắt
  vẫn được decode và trả về thành công với dữ liệu thiếu.

Cổng 4 — rỗng / toàn khoảng trắng (giữ nguyên)
- Trigger: Cổng 3 pass
- Call: strings.TrimSpace(string(data)) == ""
    - Success (có nội dung) → Cổng 5
    - Failure → DecodeError{Message: ErrEmptyBody.Error(), Err: ErrEmptyBody}

Cổng 5 — decode (sửa cách lấy Field, AC1–AC2)
- Trigger: Cổng 4 pass
- Pre-condition: opts.DisallowUnknownFields áp dụng nếu được bật
- Call: decoder.Decode(dst)
    - Success → Cổng 6
    - Failure (message chứa "unknown field") → DecodeError{Field: unknownFieldName(msg),
      Message: ErrUnknownField.Error(), Err: ErrUnknownField}                         (AC1)
    - Failure (message chứa "cannot unmarshal") → DecodeError{Field: <errors.As sang
      *json.UnmarshalTypeError>, Message: ErrInvalidFieldType.Error(), Err: ErrInvalidFieldType}  (AC2)
    - Failure (khác) → DecodeError{Message: ErrInvalidJSON.Error(), Err: ErrInvalidJSON}
  Vì sao helper parse message ở nhánh unknown field: decoder.DisallowUnknownFields() trả
  errors.New(...) — kiểu *errors.errorString, KHÔNG phải *json.UnmarshalTypeError (đã đo:
  type=*errors.errorString, msg=`json: unknown field "nope"`). Type assert cũ vì thế luôn fail →
  Field luôn rỗng → delivery/http/errors.go bỏ luôn details. Tên field CHỈ có trong message, nên
  parse message là cách duy nhất không phải tự viết lại decoder.

Cổng 6 — rác đuôi (feature này, AC3)
- Trigger: Cổng 5 thành công
- Pre-condition: dst đã được gán
- Call: decoder.Decode(&struct{}{}) — decode lần hai vào một struct rỗng
    - Success (err == io.EOF — hết sạch input, kể cả khoảng trắng đuôi) → trả nil
    - Failure (err == nil: còn một JSON value nữa; hoặc lỗi cú pháp: còn rác) →
      DecodeError{Message: ErrInvalidJSON.Error(), Err: ErrInvalidJSON}
  Vì sao không dùng decoder.More(): More() được thiết kế cho phần tử trong array/object, không cho
  stream ở top level. Đo thực tế: body một object hợp lệ → decode lần hai = io.EOF; body có thêm
  giá trị → nil hoặc lỗi cú pháp; `{"email":"x"}  ` (khoảng trắng đuôi) → io.EOF (hợp lệ, AC3).
  Lưu ý: lỗi ở cổng này xảy ra SAU khi dst đã bị ghi — hàm trả lỗi nên caller không được dùng dst.
```

## Test plan (map AC → test) — `pkg/utils/http_test.go` + handler tests

| AC | Test |
|---|---|
| 1 | `TestDecodeJSONBody_UnknownField_Disallowed` (đang đỏ) — assert `Field == "unknown_field"`. Thêm case field lồng trong object con. |
| 2 | `TestDecodeJSONBody_InvalidFieldType` (đang xanh) — assert `Field == "age"`; là lưới hồi quy cho việc đổi sang `errors.As`. |
| 3 | `TestDecodeJSONBody_MultipleJSONObjects` (đang đỏ) — `{} {}` → `ErrInvalidJSON`. Thêm case `{"a":1}garbage` và case khoảng trắng đuôi vẫn hợp lệ. |
| 4 | `TestDecodeJSONBody_WrongContentType` (đang đỏ) — `text/plain` và `application/x-www-form-urlencoded` → `ErrInvalidJSON`, `Message == "unsupported content type"`. Thêm case `application/vnd.api+json` hợp lệ. |
| 5 | `TestDecodeJSONBody_NoContentType` + `TestDecodeJSONBody_ContentTypeWithCharset` (đang xanh) — lưới hồi quy. |
| 6 | `TestDecodeJSONBody_BodyTooLarge` (đang xanh) — thắt lại thành `IsDecodeError(err, ErrBodyTooLarge)` đơn nhất, bỏ nhánh "hoặc ErrInvalidJSON". Thêm case body bị cắt đúng ranh giới JSON hợp lệ (defect `[D]`) → vẫn phải là `ErrBodyTooLarge`; và case `MaxBodySize <= 0` vẫn không giới hạn. |
| 7 | `TestLoginHandler_Login` / handler test US-001 — case decode trả `ErrBodyTooLarge` → `413` `PAYLOAD_TOO_LARGE`; case khác vẫn `400` `VALIDATION_ERROR`. |
| 8 | Biên dịch: `go build ./...` + `go vet ./...`; 5 call site `DecodeJSONBodyStrict` và các test đang xanh không phải sửa dòng nào. |
| 9 | Toàn bộ test hiện có của `pkg/utils` xanh, không sửa kỳ vọng — trừ đúng ba test đang đỏ vốn là mục tiêu của AC1/AC3/AC4. |

## Deliberately stubbed for now

Không có. Không phần nào được để trống có chủ ý — cả bốn defect đều sửa được bằng stdlib, không cần dependency mới, không cần `go.mod` đổi.

## Known gaps carried by this feature

- **Chỉ từ chối SAU khi đã đọc.** Body vượt ngưỡng bị phát hiện ở Cổng 3, tức sau khi đã đọc `Max+1` byte vào bộ nhớ. Phần còn lại của request body không được đọc; `net/http` sẽ tự drain tối đa 256 KB rồi đóng connection. Chặn sớm hơn cần `http.MaxBytesReader` (đổi hành vi transport) hoặc kiểm `Content-Length` trước khi đọc (header có thể thiếu khi chunked, và có thể nói dối) — cả hai đều ngoài scope, ghi ở requirements.md "Out of scope".
- **`MaxBodySize` là một hằng số dùng chung cho mọi route.** Route nào cần trần khác phải tự truyền `DecodeOptions`; `DecodeJSONBodyStrict` giữ 1 MB.
- **Content-Type thiếu vẫn lenient** (AC5, giữ theo `TestDecodeJSONBody_NoContentType`). Nghĩa là một client gửi JSON mà quên header vẫn qua — chấp nhận có ý thức, vì đó là hành vi đang chạy và đã có test chốt.
- **Body rỗng + Content-Type sai giờ trả lỗi Content-Type** thay vì `ErrEmptyBody` (hệ quả của thứ tự cổng, đã ghi ở Cổng 1). Không test nào phủ tổ hợp này; nếu sau này muốn ưu tiên `ErrEmptyBody`, đảo Cổng 1 xuống sau Cổng 4 — nhưng đó là đổi hợp đồng, phải sửa spec trước.
- **Không giải quyết JSON parser differential ở tầng hạ tầng.** Cổng 6 đóng lỗ phía service; nếu proxy/WAF/CDN phía trước parse khác đi thì vẫn cần cấu hình đồng bộ — ngoài repo.
- **`cmd/main.go` vẫn chưa cấu hình `ReadTimeout`/`MaxHeaderBytes`** (`#nosec G114`) — gap riêng của server. Feature này không chạm.
