# Utils — pkg/utils

## Mục đích
Tập hợp các hàm tiện ích dùng chung cho HTTP handler: decode/validate JSON body, trả `ApiResponse` chuẩn hoá, lấy thông tin resource GCP cho logging/metrics, và một số helper bảo mật nhỏ (giới hạn body size, sanitize user agent).

## Khi nào dùng / không dùng
- Dùng khi: viết HTTP handler trong `services/appService/delivery/http` cần decode request body, trả response JSON đồng nhất (`success/data/error`), hoặc validate struct request bằng tag `validate:"..."`.
- Không dùng khi: cần xác thực/authorization (dùng `pkg/jwt`), cần rate limit (dùng `pkg/ratelimit`), hoặc cần structured logging (dùng `pkg/logging`) — package này không thay thế các package hạ tầng chuyên biệt đó.

## API chính

### `DecodeJSONBody(r *http.Request, dst any, opts *DecodeOptions) error`
Đọc và decode JSON body vào `dst`, giới hạn kích thước body (mặc định 1MB qua `DefaultDecodeOptions()`), trả `*DecodeError` (bọc các sentinel error `ErrEmptyBody`, `ErrInvalidJSON`, `ErrInvalidFieldType`, `ErrUnknownField`) khi có lỗi.
```go
var req RegisterRequest
if err := utils.DecodeJSONBody(r, &req, nil); err != nil {
    field, msg := utils.GetDecodeErrorDetails(err)
    utils.WriteValidationError(w, "invalid_body", msg, map[string]any{"field": field})
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
utils.WriteValidationError(w, "email_invalid", "Email không hợp lệ", nil)
utils.WriteBusinessError(w, http.StatusConflict, "email_taken", "Email đã tồn tại", nil)
utils.WriteSystemError(w, "internal_error", "Đã có lỗi xảy ra", nil)
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
- `LimitRequestBodySize` chỉ set `MaxBytesReader` trên `r.Body` — không tự động ghi response lỗi khi vượt giới hạn; phải tự kiểm tra bằng `IsRequestTooLarge(err)` sau khi đọc body thất bại rồi map sang response phù hợp.
- `WriteErrorResponse` và các hàm `Write*Error` không trả lỗi (không kiểm tra lỗi từ `json.NewEncoder(...).Encode(...)`) — nếu response ghi lỗi (hiếm khi xảy ra vì response đã set header), lỗi sẽ bị bỏ qua âm thầm.
- `GetResourceLabels` trả set field khác nhau tuỳ môi trường (Cloud Run/GKE/generic) — không giả định key cố định khi dùng map này, luôn kiểm tra tồn tại trước khi đọc.
