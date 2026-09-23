# Validation — pkg/validation

## Mục đích
Bọc `github.com/go-playground/validator/v10` trong struct `Validator`: đây là nơi **khởi tạo và giữ** instance của thư viện. Không chỗ nào ngoài package này gọi `validator.New()`.

Package **không quyết định tên field trong lỗi**, và **không tự đăng ký rule nào**. Tag có sẵn của thư viện (`required`, `email`, `min`, `max`, `e164`, `oneof`, ...) dùng thẳng làm struct tag ở delivery layer — package này **không** bọc lại. Rule thư viện không có thì đăng ký qua `Register(tag, fn)`; tên field trong lỗi lấy theo struct tag nào thì caller khai qua `SetFieldNameTag("json")`. Cả tập tag lẫn wire format đều là quyết định của composition root — `New()` không tự làm gì.

Bốn file code:
- `validation.go` — struct `Validator` (giữ instance), `New()`, `Register`, `SetFieldNameTag`, `Struct`/`Var`, và helper không phải validator (`NormalizeEmail`).
- `phone.go` — rule `phone`: `PhoneTag` + `PhoneRule` + `IsValidPhone`, và helper `NormalizePhone` (canonical hoá về E.164). Đây là **ví dụ duy nhất package ship sẵn một custom tag**, không tự động đăng ký; muốn dùng thì `Register` ở main.
- `email.go` — `IsValidEmail`, kiểm tra hình dạng email phía **trong** (use-case), bổ sung cho tag `email` của thư viện ở delivery. Không phải custom tag, không cần `Register`.
- `phone_test.go` / `email_test.go` / `validation_test.go` — pin hợp đồng của các helper và của ba kiểu lỗi đăng ký.

Thêm custom tag mới: viết hàm `validator.Func` (đặt trong package nào cũng được — service cần thì để ở service), rồi thêm một dòng `Register` ở main. Rule nào thuộc nghiệp vụ riêng của một service thì **đặt ở service đó**, không thêm vào package dùng chung này.

## Ai khởi tạo
`main.go` (composition root) là nơi duy nhất gọi `New()`, khai wire format (`SetFieldNameTag`) và custom field nào được đăng ký (`Register(tag, fn)` với rule của chính service), rồi truyền instance xuống các layer cần. Delivery handler nhận instance qua constructor, **không** tự dựng `validator.New()` — nếu tự dựng thì tag `phone` sẽ không tồn tại và request mang tag đó fail toàn bộ.

```go
// services/appService/cmd/main.go
validate := validation.New()
validate.SetFieldNameTag("json")
if err := validate.Register(validation.PhoneTag, validation.PhoneRule); err != nil {
    log.Fatalf("failed to register phone validation: %v", err)
}
requestValidator := deliveryhttp.NewRequestValidator(validate)
```

## Khi nào dùng / không dùng
- Dùng khi: service dựng validator instance cho delivery layer (dù có custom tag hay không); hoặc cần chuẩn hoá email/số điện thoại trước khi so trùng/tra cứu; hoặc dùng luôn rule `phone` mẫu ở `phone.go`.
- Rule **dùng chung** (nhiều service cần, không gắn nghiệp vụ của riêng ai) thì để trong package này; rule gắn với một service thì đặt ở service đó — cả hai đều đăng ký qua `Register` ở composition root.
- Không dùng cho rule theo thị trường cụ thể (bắt buộc mã quốc gia nào, độ dài số nội địa bao nhiêu) — đó là business rule, thuộc use-case, không phải struct tag.

## API chính

### `type Validator struct{ ... }`
Struct giữ instance của thư viện — thứ được truyền giữa các layer. Field bên trong **không export**: mọi thứ đi qua `Struct`/`Var`/`Register`/`SetFieldNameTag`, nên bề mặt của thư viện chạm vào repo qua đúng một cửa thay vì lan ra mọi chỗ nhận `*validator.Validate`.

Thư viện tự mô tả instance của nó là **thread-safe và nên dùng như singleton** (nó cache tag/struct đã parse sau lock riêng), nên mỗi service dựng một `Validator` rồi chia sẻ. Ngoại lệ là cấu hình: thư viện ghi rõ các hàm `Register*` của nó (gồm cả `RegisterTagNameFunc`) **không** thread-safe và phải chạy hết **trước** lần validate đầu tiên — lý do `Register` và `SetFieldNameTag` thuộc về composition root.

### `New() *Validator`
Dựng `Validator`. **Không** đăng ký custom tag nào và **không** chọn wire format — cả hai là quyết định của caller. Trước khi composition root gọi `Register`/`SetFieldNameTag`: chỉ tag built-in chạy được, và lỗi mang **tên field Go**.

### `(*Validator) Register(tag string, fn validator.Func) error`
Bind một tên struct-tag với hàm thực thi nó, cho rule thư viện không có:

```go
v.Register(validation.PhoneTag, validation.PhoneRule)
// struct tag: `validate:"required,phone"`
```

Hàm là của caller — package này không giữ danh sách "service nên có tag gì", và `New()` không tự đăng ký gì.

Thư viện báo lỗi đăng ký sai theo **ba kiểu khác nhau** (đã verify với v10, không phải suy đoán):

| Trường hợp | Thư viện làm gì |
|---|---|
| Tag rỗng, hoặc hàm `nil` | Trả `error` → `RegisterXxx()` bọc lại và trả về |
| Tag trùng tên tag có sẵn (`required`, `dive`, `omitempty`, ...) hoặc chứa ký tự dành riêng ``.[],\|=+()`~!@#$%^&*\"/?<>{}`` | **Panic**, không trả error → `Register` không chuyển thành error được |
| Tag đã đăng ký rồi | Không error, không panic — **âm thầm ghi đè** hàm cũ |

Cả ba đều là lỗi lập trình và đều lộ lúc khởi động (main), nên đặt các lời gọi `Register` ở composition root.

`SetFieldNameTag` không trả lỗi — thư viện chỉ nhận hàm đặt tên, không validate gì. Truyền tên tag không tồn tại thì mọi field lặng lẽ rơi về tên Go.

Wrapper **không** truyền tham số `callValidationEvenIfNull` của thư viện → giữ mặc định: field con trỏ `nil` không chạy vào hàm rule. Muốn nói "thiếu giá trị nghĩa là gì" thì ghép tag với `required`/`omitempty`, đừng xử lý `nil` bên trong rule.

### `(*Validator) SetFieldNameTag(tag string)`
Cho lỗi validate gọi field theo struct tag chỉ định, thay vì tên field Go.

```go
v.SetFieldNameTag("json")
// PhoneNumber string `json:"phone_number"`  →  lỗi báo "phone_number"
```

⚠️ Thư viện **không** tự đọc tag `json`. Mặc định nó báo tên field Go (`PhoneNumber`) — `json:"..."` chỉ có nghĩa với `encoding/json`, không package nào hiểu tag của package khác. Không gọi hàm này thì lỗi mang tên Go.

Truyền tag nào là quyết định của delivery: `"json"`, `"form"`, `"xml"`... Package chỉ áp dụng thứ được đưa, không ôm sẵn thiên vị nào.

Quy tắc lấy tên: cắt ở dấu phẩy đầu tiên, nên `json:"nickname,omitempty"` → `nickname`. Không có tag đó, tên rỗng (`json:",omitempty"`), hoặc `json:"-"` → **giữ tên field Go**, để response luôn có key chứ không rỗng.

Giống `Register`, hàm này **không** thread-safe, và thư viện cache tên field ngay lần validate struct đầu tiên → gọi ở composition root, trước khi validate lần đầu.

### `(*Validator) Struct(s any) error` / `(*Validator) Var(field any, tag string) error`
`Struct` validate theo tag `validate:"..."` của struct; `Var` validate một giá trị lẻ không nằm trong struct (path/query param). Lỗi trả về là lỗi gốc của thư viện — `validator.ValidationErrors` khi field fail — nên caller vẫn `errors.As` để lấy field/tag nào hỏng.

### `IsValidEmail(raw string) bool`
Kiểm email có dùng được làm địa chỉ tài khoản không. Tồn tại vì `NormalizeEmail` **cố ý** không kiểm hình dạng (chỉ lowercase), còn use-case phải loại email hỏng **trước** khi chạm DB (US-001 AC7/AC8). Tag `email` của thư viện ở delivery là cửa ngoài; đây là cửa trong, nên caller đi thẳng vào use-case mà không qua handler vẫn bị chặn.

Strict hơn `mail.ParseAddress` ở ba điểm:
- từ chối display name / angle-addr (`"Lamp <a@b.com>"`) — giá trị lưu phải đúng chuỗi người dùng gõ, không phải hình dạng khác parse ra cùng địa chỉ;
- domain phải có dấu chấm ⇒ `"user@localhost"` không phải địa chỉ tài khoản. Đây là **chỗ duy nhất lệch với tag `email`** của thư viện (tag đó chấp nhận domain không chấm) — khi hai bên lệch, hàm này là bên chặt hơn, nên use-case không bao giờ nhận thứ delivery đã từ chối;
- domain có dấu chấm ở đầu/cuối hoặc hai dấu chấm liền → từ chối.

Chỉ kiểm hình dạng: không DNS lookup, không MX record, không chính sách plus-address. Caller truyền giá trị **đã chuẩn hoá** (`NormalizeEmail`) để kiểm đúng chuỗi sẽ được lưu.

### `NormalizeEmail(email string) string`
Lowercase và trim khoảng trắng. Mọi thao tác so trùng/tra cứu email **phải** đi qua hàm này để `User@Example.com` và `user@example.com` là cùng một tài khoản. Hàm không kiểm tra email có đúng định dạng không — đó là việc của tag `email` ở delivery layer.

```go
email := validation.NormalizeEmail("  USER@Example.com  ") // "user@example.com"
```

### `IsValidEmail(raw string) bool`
Kiểm tra email có **hình dạng** dùng được làm địa chỉ tài khoản không. Hàm này tồn tại vì `NormalizeEmail` cố ý **không** kiểm tra hình dạng (nó chỉ lowercase), mà use-case phải từ chối địa chỉ hỏng **trước khi** chạm DB (US-001 AC7/AC8). Tag `email` của thư viện ở delivery là cổng ngoài; đây là cổng trong, nên caller đi thẳng vào use-case mà bỏ qua handler vẫn bị chặn.

Nghiêm hơn `mail.ParseAddress` ở ba điểm:
- từ chối display name / angle-addr (`"Lamp <a@b.com>"`) — giá trị lưu phải đúng chuỗi user gõ, không phải hình dạng khác parse ra cùng địa chỉ;
- domain **phải** có dấu chấm, nên `user@localhost` không phải địa chỉ tài khoản (đây là điểm duy nhất lệch với tag `email` của thư viện — tag đó nhận domain không chấm);
- từ chối dấu chấm ở đầu/cuối hoặc lặp (`..`) trong domain.

Chỉ kiểm tra hình dạng: không DNS, không MX, không chính sách plus-address. Caller truyền giá trị **đã chuẩn hoá** (`NormalizeEmail`) để kiểm tra đúng chuỗi sẽ được lưu.

```go
if !validation.IsValidEmail(email) {
    return nil, domain.ErrInvalidEmail
}
```

## Cấu hình / dependency
- Không có config struct.
- Phụ thuộc ngoài duy nhất: `github.com/go-playground/validator/v10`.
- Không import package nội bộ nào khác trong `pkg/` — package "lá" (leaf).
- Import path: `chjpcoj.com.innerlamp/pkg/validation`.
- Caller hiện tại: `services/appService/usecase/` (`RegisterUsecase` gọi `NormalizeEmail` + `IsValidEmail` + `NormalizePhone` để validate **trước** khi chạm DB). `services/appService/cmd/main.go` — nơi duy nhất gọi `New()`/`SetFieldNameTag`/`Register` — chưa tồn tại ở phạm vi US-001 hiện tại.

## Lưu ý / lỗi hay gặp
- Thư viện **không** báo lỗi khi đăng ký trùng tên tag — nó âm thầm ghi đè hàm cũ, và `register` không phát hiện được. Vì vậy hãy gọi `New()` **một lần cho mỗi service** rồi truyền `*Validator` đi, đừng dựng lại ở từng layer.
- Đặt tên tag mới tránh trùng tag built-in và tránh ký tự dành riêng — trùng thì **panic** ngay lúc đăng ký chứ không trả error.
- `New()` gọi `validator.New()` **không kèm option**. Đáng chú ý là `WithRequiredStructEnabled()`: upstream khuyến nghị bật và sẽ là mặc định ở v11 — nó làm tag `required` áp lên field struct lồng (non-pointer) thay vì bị bỏ qua. Hiện các request DTO đều phẳng nên chưa ảnh hưởng; nếu thêm DTO có struct lồng thì cân nhắc bật.
- Quên gọi `RegisterXxx()` mà struct vẫn mang tag đó → thư viện **panic** ("Undefined validation function"), không phải trả lỗi. Lỗi này lộ ngay ở request đầu tiên chứ không âm thầm bỏ qua rule.
- Ba kiểu lỗi đăng ký ở trên, cộng với các fallback của `SetFieldNameTag`, được pin bằng test trong `validation_test.go` — thư viện đổi hành vi thì test gãy chứ README không âm thầm sai.
- Quên gọi `SetFieldNameTag("json")` → lỗi trả về tên field Go (`PhoneNumber`). `requestFieldError` phía delivery switch theo tên JSON (`phone`) nên sẽ rơi hết vào nhánh `default` `INVALID_REQUEST_BODY` thay vì mã lỗi đúng.
