# Mailer — pkg/mailer

## Mục đích
Gửi email giao dịch (transactional email) liên quan đến auth. Entry point chung là `Send(ctx, Message)` — caller tự dựng tiêu đề/nội dung (appService: `services/appService/repository/email_auth_mailer.go` dựng thư mã đăng ký / đặt lại mật khẩu / báo email đã có tài khoản, onboarding spec v8.0). Hai method cũ theo mục đích (`SendPasswordResetEmail`, `SendAccountInviteEmail`) giữ lại cho backofficeService.

## Khi nào dùng / không dùng
- Dùng khi: service cần gửi một email giao dịch (`Send`), hoặc backoffice gửi thư mời tài khoản (`SendAccountInviteEmail`).
- Không dùng khi: cần gửi email production thật — package hiện **chỉ có `LogMailer`** (dev-only, log ra thay vì gửi thật), chưa có provider thật nào (SendGrid, SES, ...). Đừng dùng `LogMailer` ở production.

## API chính

### `type Mailer interface`
Interface duy nhất của package, định nghĩa contract gửi email auth.
```go
type Message struct {
    From    string // địa chỉ gửi, caller lấy từ cấu hình (appService: mailer.from)
    To      string // một người nhận
    Subject string
    Text    string // bắt buộc
    HTML    string // tuỳ chọn, rỗng = chỉ text
}

type Mailer interface {
    Send(ctx context.Context, msg Message) error
    SendPasswordResetEmail(ctx context.Context, toEmail, resetToken string) error
    SendAccountInviteEmail(ctx context.Context, toEmail, name, tempPassword string) error
}
```
`From` không hardcode trong package — caller truyền giá trị cấu hình (appService: `mailer.from`).
Cũng có `SendAccountInviteEmail(ctx, toEmail, name, tempPassword)` — backofficeService dùng khi tạo tài khoản admin mới, gửi mật khẩu tạm (phải đổi ở lần đăng nhập đầu). `LogMailer` log cả mật khẩu tạm nên chỉ dùng cho dev.

`resetToken` là token thô (chưa hash) — caller chỉ được lưu bản hash của nó (xem `services/appService/domain/errors.go` / `password_reset.go`), không lưu token thô vào DB.

### `NewLogMailer(logger logging.Logger) *LogMailer`
Implementation duy nhất hiện có — **dev-only**, log lại **toàn bộ** nội dung email (kể cả mã xác minh, mật khẩu tạm) thay vì gửi thật.
```go
import "chjpcoj.com.innerlamp/pkg/mailer"

devMailer := mailer.NewLogMailer(logger) // dev only — thay bằng provider thật trước khi lên production
err := devMailer.Send(ctx, mailer.Message{To: "user@example.com", Subject: "...", Text: "..."})
```
Xem `services/appService/cmd/main.go` để biết cách wiring thực tế trong service.

### `MockMailer` (pkg/mailer/mock.go)
Mock `Mailer` bằng `testify/mock`, dùng trong unit test của code gọi `mailer.Mailer` (ví dụ usecase forgot-password).

## Cấu hình / dependency
- `LogMailer` chỉ cần một `logging.Logger` (`chjpcoj.com.innerlamp/pkg/logging`) — không có config riêng, không đọc biến môi trường.
- Không phụ thuộc thư viện gửi email ngoài nào (chưa có provider thật).
- `MockMailer` phụ thuộc `github.com/stretchr/testify/mock`.
- Import path đúng trong repo này: `chjpcoj.com.innerlamp/pkg/mailer`.

## Lưu ý / lỗi hay gặp
- appService chặn khởi động khi `mailer.provider: log` mà `app.environment` khác `dev` (onboarding AC21, `services/appService/config` `AppConfig.Validate`).
- **Package chưa hoàn thiện**: chỉ có 1 implementation (`LogMailer`, log-only). Không có gửi email thật — nếu deploy production mà quên swap `LogMailer` sang provider thật thì email reset mật khẩu sẽ không bao giờ đến tay user, chỉ nằm trong log.
- Package **cố tình không theo pattern provider-registry** (`Register`/`New`/`FactoryFunc`) như `pkg/database`, `pkg/secrets`, `pkg/storage` — vì mới chỉ có một implementation, thêm registry lúc này là premature abstraction. Khi có provider thật thứ hai (SendGrid, SES, ...) thì mới áp dụng pattern registry, không nên tự ý thêm registry sớm khi chưa cần.
- `resetToken` truyền vào `SendPasswordResetEmail` là giá trị thô (unhashed) — chỉ dùng để đưa vào nội dung email/link, không lưu thẳng vào DB; tầng gọi chịu trách nhiệm hash trước khi persist.
- Mọi method của `LogMailer` luôn trả `nil` — không mô phỏng lỗi gửi mail (timeout, địa chỉ không hợp lệ, ...), nên test chỉ dựa vào `LogMailer` sẽ không phát hiện được các lỗi đó khi chuyển qua provider thật.
