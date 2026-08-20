# Validation — pkg/validation

## Mục đích
Validate và sanitize input dạng chuỗi: email, số điện thoại/CURP/postal code/INE key kiểu Mexico, phát hiện/loại bỏ XSS, cùng vài validator chuỗi chung (required, length, alpha, numeric...). Có `Validator` fluent để gom nhiều lỗi validate thành một lần trả về.

## Khi nào dùng / không dùng
- Dùng khi: cần validate format email hoặc sanitize/kiểm tra XSS cho text field do user nhập trước khi lưu/hiển thị; hoặc cần validate dữ liệu định danh/địa chỉ theo chuẩn Mexico (phone, CURP, postal code, INE key) — các hàm này hard-code theo format Mexico, không tổng quát cho quốc gia khác.
- Không dùng khi: cần validate struct bằng tag (`validate:"required,email"`) — package này không có reflection/tag-based validation, phải tự gọi từng hàm; cũng không dùng cho input số điện thoại/CMND/mã bưu chính của Việt Nam hay quốc gia khác — logic hiện tại chỉ đúng cho Mexico (`ValidateMexicanPhone`, `ValidateCURP`, `ValidatePostalCode`, `ValidateINEKey`).

## API chính

### `ValidateEmail(email string) error`
Kiểm tra format email (regex + độ dài ≤254 + local/domain part hợp lệ). Trả `*ValidationError` nếu sai.
```go
if err := validation.ValidateEmail("user@example.com"); err != nil {
    // xử lý lỗi
}
```

### `NormalizeEmail(email string) string`
Chuẩn hoá email về lowercase và trim khoảng trắng.
```go
email := validation.NormalizeEmail("  USER@Example.com  ") // "user@example.com"
```

### `ValidateMexicanPhone(phone string) error` / `NormalizeMexicanPhone(phone string) string`
Validate số điện thoại Mexico (10 số, hoặc có prefix `+52`/`52`), chấp nhận dấu `-`, ` `, `()`. `Normalize` trả về đúng 10 số cuối.
```go
err := validation.ValidateMexicanPhone("+52 55-1234-5678")
normalized := validation.NormalizeMexicanPhone("55-1234-5678") // "5512345678"
```

### `ValidateCURP(curp string) error` / `NormalizeCURP(curp string) string`
Validate CURP Mexico (18 ký tự, đúng format `AAAA######[H|M]AAAAA#0`, ký tự vị trí 10 phải là `H`/`M`). Chỉ check format, không check checksum thật.

### `ValidatePostalCode(postalCode string) error`
Postal code Mexico: đúng 5 chữ số.

### `ValidateINEKey(ineKey string) error`
Số thẻ cử tri (credential) Mexico: 13 hoặc 18 chữ số (sau khi bỏ khoảng trắng).

### `ValidateNoXSS(input string) error`
Check chuỗi có chứa pattern XSS phổ biến không (`<script>`, `<iframe>`, `javascript:`, `on*=`, `<embed>`, `<object>` — case-insensitive). Trả lỗi nếu match, không sửa input.
```go
if err := validation.ValidateNoXSS(comment); err != nil {
    // từ chối input, không lưu
}
```

### `SanitizeString(input string) string`
HTML-escape input (`html.EscapeString`) + loại null byte. Không trim, không loại control char khác.

### `SanitizeInput(input string) string`
Sanitize toàn diện hơn: trim whitespace, HTML-escape, loại control character (giữ lại `\n`, `\t`), loại null byte. Dùng trước khi lưu text field tự do (tên, địa chỉ, ghi chú...).
```go
clean := validation.SanitizeInput("  <script>alert(1)</script>  ")
```

### `ValidateRequired(value, field string) error` / `ValidateStringLength(value, field string, min, max int) error` / `ValidateAlphanumeric` / `ValidateAlpha` / `ValidateNumeric(value, field string) error`
Các validator chuỗi chung, không phụ thuộc quốc gia — required (sau trim), độ dài min/max, chỉ chữ+số, chỉ chữ (cho phép space), chỉ số.

### `Validator` (`NewValidator()`, `.ValidateField(err error) *Validator`, `.HasErrors() bool`, `.Errors() []*ValidationError`, `.Error() error`)
Fluent API gom nhiều lỗi validate lại thành một `error` duy nhất, thay vì return sớm ở lỗi đầu tiên.
```go
v := validation.NewValidator()
v.ValidateField(validation.ValidateRequired(email, "email"))
v.ValidateField(validation.ValidateEmail(email))
v.ValidateField(validation.ValidateNoXSS(name))

if v.HasErrors() {
    return v.Error() // "validation failed: email: ...; input: ..."
}
```

### `ValidationError` (`Field`, `Message string`) / `NewValidationError(field, message string) *ValidationError`
Kiểu lỗi trả về từ mọi hàm `Validate*` trong package, implement `error` với format `"<field>: <message>"`.

## Cấu hình / dependency
- Không có config, không có state ngoài các regex pre-compiled ở package scope (`MexicanPhoneRegex`, `MexicanPhoneWithCountryCodeRegex`, `EmailRegex`, `CURPRegex`, `XSSPatterns`) — các biến này exported nên có thể tái dùng trực tiếp nếu cần.
- Chỉ dùng thư viện chuẩn (`regexp`, `strings`, `unicode`, `html`, `fmt`), không có dependency ngoài, không phụ thuộc package nội bộ nào khác trong repo.
- Import path đúng trong repo này: `chjpcoj.com.innerlamp/pkg/validation` (README cũ của package này ghi `base_project/pkg/validation` — đó là tàn dư từ template, đừng copy).

## Lưu ý / lỗi hay gặp
- Các hàm `ValidateMexicanPhone`, `ValidateCURP`, `ValidatePostalCode`, `ValidateINEKey` hard-code theo chuẩn Mexico — dùng sai bối cảnh (ví dụ validate số điện thoại Việt Nam) sẽ luôn báo lỗi hoặc pass sai.
- `Validator.ValidateField` chỉ append lỗi nếu type-assert được về `*ValidationError` — nếu truyền vào một `error` thường (không phải do các hàm `Validate*` trong package này trả về) thì lỗi đó bị **âm thầm bỏ qua**, không có trong `Errors()`/`Error()`.
- `ValidateNoXSS` chỉ nhận diện các pattern trong `XSSPatterns` (script/iframe/javascript:/on*=/embed/object) — không phải bộ lọc XSS toàn diện, không thay thế được sanitize khi output ra HTML thật sự (dùng thêm `SanitizeString`/`SanitizeInput`, hoặc escape đúng context lúc render).
- `SanitizeString` chỉ escape HTML, không trim và không loại control character khác ngoài null byte — muốn sanitize đầy đủ hơn (trim + loại control char) thì dùng `SanitizeInput`.
- `ValidateCURP` chỉ check đúng format bằng regex, không tính checksum thật của CURP — chuỗi đúng định dạng nhưng vô nghĩa vẫn pass.
