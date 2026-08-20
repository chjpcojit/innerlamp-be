# Mailer — pkg/mailer

## Mục đích
Gửi email giao dịch (transactional email) liên quan đến auth. Hiện tại chỉ có một use case: gửi email đặt lại mật khẩu (US-003 — quên mật khẩu). Package được thêm vào vì spec onboarding cần forgot-password nhưng kiến trúc gốc chưa có dependency gửi email nào.

## Khi nào dùng / không dùng
- Dùng khi: service cần gửi email reset mật khẩu cho luồng forgot-password (`SendPasswordResetEmail`).
- Không dùng khi: cần gửi email production thật — package hiện **chỉ có `LogMailer`** (dev-only, log ra thay vì gửi thật), chưa có provider thật nào (SendGrid, SES, ...). Đừng dùng `LogMailer` ở production. Cũng không dùng cho các loại email khác ngoài password-reset — interface hiện chưa hỗ trợ.

## API chính

### `type Mailer interface`
Interface duy nhất của package, định nghĩa contract gửi email auth.
```go
type Mailer interface {
    SendPasswordResetEmail(ctx context.Context, toEmail, resetToken string) error
}
```
`resetToken` là token thô (chưa hash) — caller chỉ được lưu bản hash của nó (xem `services/appService/domain/errors.go` / `password_reset.go`), không lưu token thô vào DB.

### `NewLogMailer(logger logging.Logger) *LogMailer`
Implementation duy nhất hiện có — **dev-only**, log lại nội dung email thay vì gửi thật.
```go
import "chjpcoj.com.innerlamp/pkg/mailer"

devMailer := mailer.NewLogMailer(logger) // dev only — thay bằng provider thật trước khi lên production
err := devMailer.SendPasswordResetEmail(ctx, "user@example.com", rawResetToken)
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
- **Package chưa hoàn thiện**: chỉ có 1 method (`SendPasswordResetEmail`) và 1 implementation (`LogMailer`, log-only). Không có gửi email thật — nếu deploy production mà quên swap `LogMailer` sang provider thật thì email reset mật khẩu sẽ không bao giờ đến tay user, chỉ nằm trong log.
- Package **cố tình không theo pattern provider-registry** (`Register`/`New`/`FactoryFunc`) như `pkg/database`, `pkg/secrets`, `pkg/storage` — vì mới chỉ có một implementation, thêm registry lúc này là premature abstraction. Khi có provider thật thứ hai (SendGrid, SES, ...) thì mới áp dụng pattern registry, không nên tự ý thêm registry sớm khi chưa cần.
- `resetToken` truyền vào `SendPasswordResetEmail` là giá trị thô (unhashed) — chỉ dùng để đưa vào nội dung email/link, không lưu thẳng vào DB; tầng gọi chịu trách nhiệm hash trước khi persist.
- `LogMailer.SendPasswordResetEmail` luôn trả `nil` — không mô phỏng lỗi gửi mail (timeout, địa chỉ không hợp lệ, ...), nên test chỉ dựa vào `LogMailer` sẽ không phát hiện được các lỗi đó khi chuyển qua provider thật.
