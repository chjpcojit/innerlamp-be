# Utils — pkg/utils

## Mục đích
Tập hợp các hàm tiện ích dùng chung cho HTTP handler: decode/validate JSON body, trả `ApiResponse` chuẩn hoá, lấy thông tin resource GCP cho logging/metrics, và một số helper bảo mật nhỏ (giới hạn body size, sanitize user agent).

## Khi nào dùng / không dùng
- Dùng khi: viết HTTP handler trong `services/appService/delivery/http` cần decode request body, trả response JSON đồng nhất (`success/data/error`), hoặc validate struct request bằng tag `validate:"..."`.
- Không dùng khi: cần xác thực/authorization (dùng `pkg/jwt`), cần rate limit (dùng `pkg/ratelimit`), hoặc cần structured logging (dùng `pkg/logging`) — package này không thay thế các package hạ tầng chuyên biệt đó.

## Envelope response chuẩn (một định dạng cho mọi response có body)

Mọi response JSON gửi cho client — từ handler, từ middleware (`pkg/jwt/middleware`, `pkg/ratelimit/middleware`, `pkg/middleware.Recovery`), và 404/405 của router — đều đi qua các hàm `Write*` bên dưới. Không tự `json.NewEncoder(w)`, không `http.Error`, không định nghĩa struct lỗi riêng.

```jsonc
// thành công — WriteSuccessResponse / WriteSuccessResponseWithStatus
{"success": true, "data": { ... }}

// lỗi — WriteValidationError / WriteBusinessError / WriteSystemError
{"success": false,
 "error": {"code": "UPPER_SNAKE_CASE", "message": "...", "type": "validation|business|system"},
 "details": { ... }}   // tuỳ chọn, nằm CẠNH "error", không lồng bên trong
```

- `code`: `UPPER_SNAKE_CASE`, client rẽ nhánh theo `code`, không theo `message`.
- `type`: `validation` (400 do input sai hình dạng), `business` (mọi 4xx còn lại do nghiệp vụ/xác thực/route), `system` (5xx — `message` không bao giờ lộ nguyên nhân nội bộ).
- `details`: chỉ có khi mang thông tin client dùng được — `{"field": ...}`, `{"locked_until": ...}`, `{"retry_after_seconds": N}`, hoặc một cặp `field: tag` cho mỗi field lỗi validate.
- Ngoại lệ duy nhất: response không có body (preflight `OPTIONS`, redirect) — không có envelope vì không có body. Thành công mà không có payload vẫn dùng envelope: `WriteSuccessResponse(w, nil)` → `{"success": true}` (`data` bị `omitempty` bỏ đi), ví dụ logout của appService.

## API chính

### `DecodeJSONBody(r *http.Request, dst any, opts *DecodeOptions) error`
Đọc và decode JSON body vào `dst`, giới hạn kích thước body (mặc định 1MB qua `DefaultDecodeOptions()`), trả `*DecodeError` (bọc các sentinel error `ErrEmptyBody`, `ErrInvalidJSON`, `ErrInvalidFieldType`, `ErrUnknownField`, `ErrBodyTooLarge`) khi có lỗi.

Các cổng kiểm chạy theo thứ tự cố định — thứ tự này là **hợp đồng**, vì đảo hai cổng sẽ đổi lỗi trả về cho cùng một request (spec: `docs/specs/features/json-body-hardening/`):

1. **Content-Type** — chấp nhận `application/json` và mọi media type có hậu tố `+json` (vd. `application/vnd.api+json`), không phân biệt hoa/thường, cho phép tham số (`; charset=utf-8`). Thiếu hẳn header → cho qua (lenient). Media type khác, hoặc header không parse được → `ErrInvalidJSON` với message `unsupported content type`.
2. **Đọc body**, tối đa `MaxBodySize+1` byte (`MaxBodySize <= 0` = không giới hạn).
3. **Vượt ngưỡng** → `ErrBodyTooLarge`, kể cả khi phần đọc được tình cờ là JSON hợp lệ — body không bao giờ bị cắt âm thầm rồi decode phần còn lại.
4. **Rỗng / chỉ có khoảng trắng** → `ErrEmptyBody`.
5. **Decode** → `ErrUnknownField` (`Field` = tên field lạ), `ErrInvalidFieldType` (`Field` = field sai kiểu), hoặc `ErrInvalidJSON`.
6. **Dữ liệu sau giá trị JSON đầu tiên** (object thứ hai, rác, dấu phẩy) → `ErrInvalidJSON`. Khoảng trắng đuôi vẫn hợp lệ.
```go
var req RegisterRequest
if err := utils.DecodeJSONBody(r, &req, nil); err != nil {
    field, msg := utils.GetDecodeErrorDetails(err)
    utils.WriteValidationError(w, "VALIDATION_ERROR", msg, map[string]any{"field": field})
    return
}
```

### `DecodeJSONBodyStrict(r *http.Request, dst any) error`
Giống `DecodeJSONBody` nhưng bật `DisallowUnknownFields` — reject request có field lạ.

### `WriteSuccessResponse(w http.ResponseWriter, data any)` / `WriteSuccessResponseWithStatus(w, statusCode, data)`
Ghi response `ApiResponse{Success: true, Data: data}` dạng JSON.
```go
utils.WriteSuccessResponse(w, userDTO) // 200 OK
utils.WriteSuccessResponseWithStatus(w, http.StatusCreated, userDTO)
```

### `WriteValidationError` / `WriteBusinessError` / `WriteSystemError` (wrap `WriteErrorResponse`)
Ghi response lỗi chuẩn hoá theo loại: validation (400), business (status tuỳ chọn), system (500).
```go
utils.WriteValidationError(w, "INVALID_EMAIL", "Email không hợp lệ", map[string]any{"field": "email"})
utils.WriteBusinessError(w, http.StatusConflict, "EMAIL_TAKEN", "Email đã tồn tại", nil)
utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil) // không lộ nguyên nhân nội bộ
```

### `GetValidator() *Validator`
Trả singleton `*Validator` bọc `go-playground/validator/v10`.

### `(*Validator) Validate(data any, customValidators map[string]func(any) string) error`
Validate struct theo tag `validate:"..."`, trả `ValidationErrors` (implement `error`, chứa danh sách `ValidationError{Field, Message, Tag, Value}`) nếu có lỗi. Tham số `customValidators` hiện là placeholder — logic custom validator chưa được implement đầy đủ trong `runCustomValidators` (xem phần Lưu ý).
```go
if err := utils.GetValidator().Validate(req, nil); err != nil {
    utils.WriteValidationError(w, "invalid_body", err.Error(), utils.GetValidationErrorDetails(err))
    return
}
```

### `IsValidationError(err error) bool` / `GetValidationErrorDetails(err error) map[string]any`
Kiểm tra và trích chi tiết lỗi từ `ValidationErrors` để đưa vào response.

### `GetEnv(key, defaultValue string) string`
Đọc biến môi trường, trả `defaultValue` nếu chưa set hoặc rỗng.

### `GetResourceType() string` / `GetResourceLabels() map[string]string`
Tự phát hiện môi trường chạy (Cloud Run qua `K_SERVICE`, GKE qua `KUBERNETES_SERVICE_HOST`, GCE qua `GCE_INSTANCE_ID`, mặc định generic task) và trả resource type/labels tương ứng cho GCP logging/monitoring. Dùng nội bộ bởi các package như `pkg/logging`, `pkg/metrics`, `pkg/tracing`.

### `SanitizeUserAgent(userAgent string) string`
Cắt bớt user agent quá dài (> `MaxUserAgentLength` = 200 ký tự) và loại bỏ `\n`/`\r` để log an toàn, chống log injection.

### `LimitRequestBodySize(w http.ResponseWriter, r *http.Request)`
Bọc `r.Body` bằng `http.MaxBytesReader` với giới hạn `MaxRequestSize` (1MB) — gọi ở đầu handler/middleware trước khi đọc body.

### `IsRequestTooLarge(err error) bool`
Kiểm tra error message có chứa "request body too large" — dùng để phân biệt lỗi do vượt giới hạn body size sau khi đọc body thất bại.

## Cấu hình / dependency
- Không có config struct riêng — các hằng số giới hạn (`MaxRequestSize`, `MaxUserAgentLength`) là hardcoded trong `utils.go`.
- `GetResourceType`/`GetResourceLabels` đọc trực tiếp biến môi trường (`K_SERVICE`, `KUBERNETES_SERVICE_HOST`, `GCE_INSTANCE_ID`, `GCP_PROJECT_ID`, `GOOGLE_CLOUD_PROJECT`, `K_REVISION`, `CLOUD_RUN_REGION`, `K_CONFIGURATION`, `GKE_CLUSTER_LOCATION`, `GKE_CLUSTER_NAME`, `KUBERNETES_NAMESPACE`, `HOSTNAME`, `LOCATION`) — không nhận config truyền vào.
- Phụ thuộc thư viện ngoài: `github.com/go-playground/validator/v10` (cho `Validator`).
- Không phụ thuộc package nội bộ nào khác trong `pkg/` — đây là package "lá" (leaf), an toàn để import từ bất kỳ layer nào mà không tạo cycle.
- Import path: `chjpcoj.com.innerlamp/pkg/utils`.

## Lưu ý / lỗi hay gặp
- `Validator.Validate` nhận tham số `customValidators map[string]func(any) string` nhưng `runCustomValidators` hiện là **placeholder chưa implement** — vòng lặp gọi `validatorFunc` nhưng bỏ qua kết quả trả về, không sinh `ValidationError` nào. Đừng trông chờ custom validator hoạt động cho tới khi phần này được implement lại.
- `GetValidator()` trả về **singleton** — không đăng ký thêm validator tuỳ biến (`validate.RegisterValidation`) trong runtime của một service cụ thể vì sẽ ảnh hưởng toàn bộ process dùng chung `pkg/utils`.
- `DecodeJSONBody` với `opts == nil` dùng `DefaultDecodeOptions()` (giới hạn 1MB, cho phép unknown field) — muốn strict mode phải gọi `DecodeJSONBodyStrict` hoặc tự truyền `opts`.
- `ErrBodyTooLarge` là lỗi về **kích thước**, không phải cú pháp — handler nên map nó sang `413`, không phải `400`. `services/appService/delivery/http/errors.go` (`writeDecodeError`) làm vậy qua `WriteBusinessError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", ...)`; lưu ý `WriteValidationError` hardcode `400` nên không dùng được cho nhánh này.
- `LimitRequestBodySize` chỉ set `MaxBytesReader` trên `r.Body` — không tự động ghi response lỗi khi vượt giới hạn; phải tự kiểm tra bằng `IsRequestTooLarge(err)` sau khi đọc body thất bại rồi map sang response phù hợp.
- Các hàm `Write*` không trả lỗi và bỏ qua lỗi của `json.NewEncoder(...).Encode(...)` — status line đã gửi đi trước khi encode nên không thể đổi sang response khác; ghi thêm body text (như bản cũ dùng `http.Error`) chỉ làm hỏng JSON client đang đọc. Lỗi encode thực tế chỉ xảy ra khi kết nối đứt hoặc `data` không marshal được.
- `GetResourceLabels` trả set field khác nhau tuỳ môi trường (Cloud Run/GKE/generic) — không giả định key cố định khi dùng map này, luôn kiểm tra tồn tại trước khi đọc.
