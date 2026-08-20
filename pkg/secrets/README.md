# Secrets — pkg/secrets

## Mục đích
Interface thống nhất để tạo/đọc/cập nhật/xoá secret (mật khẩu DB, API key...) qua nhiều provider — GCP Secret Manager, AWS Secrets Manager, hoặc biến môi trường cho local dev — mà code gọi (usecase, `pkg/config`, ...) không cần biết đang chạy trên cloud nào.

## Khi nào dùng / không dùng
- Dùng khi: service cần đọc secret lúc khởi động (DB password, JWT signing key, API key bên thứ 3) từ GCP Secret Manager / AWS Secrets Manager, hoặc cần một implementation giả lập bằng env var khi chạy local/test.
- Không dùng khi: chỉ cần đọc config không nhạy cảm (dùng `pkg/config`), hoặc cần rotate/mount secret tự động kiểu Kubernetes Secret/Vault agent — package này không quản lý vòng đời secret ở tầng hạ tầng, chỉ là client gọi API của provider.

## API chính

### `secrets.New(config *Config) (Client, error)`
Factory tạo client theo `config.Provider` (`ProviderGCP` hoặc `ProviderAWS`). Provider phải được đăng ký trước qua blank import, nếu không trả `ErrUnsupportedProvider`.
```go
import (
    "chjpcoj.com.innerlamp/pkg/secrets"
    _ "chjpcoj.com.innerlamp/pkg/secrets/gcp" // đăng ký provider GCP
)

client, err := secrets.New(&secrets.Config{
    Provider: secrets.ProviderGCP,
    Timeout:  30 * time.Second,
    GCP: &secrets.GCPConfig{
        ProjectID:             "your-project-id",
        UseDefaultCredentials: true, // ADC — Cloud Run/GKE tự cấp
    },
})
```

### `Client.Connect(ctx) / Close() / IsConnected() bool`
`New()` chỉ validate config, **không** mở kết nối. Phải gọi `Connect(ctx)` riêng để service fail-fast lúc khởi động thay vì giữa request. `Connect` gọi lần 2 khi đã kết nối trả `ErrAlreadyConnected`.
```go
if err := client.Connect(ctx); err != nil { log.Fatal(err) }
defer client.Close()
```

### `Client.Get(ctx, name string, opts *GetOptions) (*SecretValue, error)`
Lấy giá trị secret (mặc định version "latest"). `SecretValue.String()` trả giá trị dạng string.
```go
value, err := client.Get(ctx, "database-password", nil)
dbPass := value.String()
```

### `Client.Create / Update / Delete / List`
CRUD đầy đủ: `Create(ctx, name, value, *CreateOptions)`, `Update(ctx, name, value)` (thêm version mới), `Delete(ctx, name, *DeleteOptions)`, `List(ctx, *ListOptions)`.

### `Client.GetVersion / ListVersions / DisableVersion / EnableVersion / DestroyVersion`
Thao tác theo version cụ thể — hành vi khác nhau giữa provider (xem phần "Lưu ý" bên dưới).

### `Client.Exists(ctx, name string) (bool, error)`
Kiểm tra secret tồn tại mà không cần đọc giá trị.

### `secrets.Register(provider Provider, factory FactoryFunc)`
Đăng ký một provider mới — provider con (aws/gcp) tự gọi trong `init()`. Chỉ cần khi viết thêm driver mới cho package này.

### `env.New() secrets.Client` (pkg/secrets/env)
Implementation đọc secret từ biến môi trường, dùng cho local dev/test. Tên secret được chuyển đổi: `-` và `/` → `_`, rồi viết hoa — ví dụ `elcora-mongodb-uri` → `ELCORA_MONGODB_URI`. Chỉ hỗ trợ `Get`/`Exists`; mọi thao tác ghi (`Create`, `Update`, `Delete`, version ops...) trả `ErrUnsupportedProvider`.
```go
import secretsenv "chjpcoj.com.innerlamp/pkg/secrets/env"

client := secretsenv.New() // không cần Connect, không cần Config
value, err := client.Get(ctx, "elcora-mongodb-uri", nil)
```

## Cấu hình / dependency
- `Config{Provider, Timeout, GCP *GCPConfig, AWS *AWSConfig}` — mọi giá trị phải truyền tường minh, không đọc env var ngầm, không có default. `Config.Validate()` được `New()` gọi tự động.
- `GCPConfig{ProjectID, CredentialsFile, CredentialsJSON, UseDefaultCredentials}`: khi `UseDefaultCredentials: true`, dùng Application Default Credentials (Cloud Run/GKE tự cấp qua service account gắn kèm) và bỏ qua `CredentialsFile`/`CredentialsJSON`.
- `AWSConfig{Region, AccessKeyID, SecretAccessKey, SessionToken, Endpoint, UseDefaultCredentials}`: khi `UseDefaultCredentials: false`, package **phải** dùng static credentials từ `AccessKeyID`/`SecretAccessKey`/`SessionToken` được truyền vào (`credentials.NewStaticCredentialsProvider`) — không tự đọc từ file/env như GCP; khi `true` thì dùng default credential chain của AWS SDK (IAM role trên Lambda/ECS/Fargate).
- Provider AWS/GCP tự đăng ký qua `init()` trong `pkg/secrets/aws` và `pkg/secrets/gcp` — cần blank import (`_ "chjpcoj.com.innerlamp/pkg/secrets/aws"`) để `secrets.New()` nhận ra. Provider `env` **không** tự đăng ký (không có `init()` gọi `secrets.Register`) — gọi `env.New()` trực tiếp, không đi qua `secrets.New()`.
- Import path đúng trong repo này: `chjpcoj.com.innerlamp/pkg/secrets`, `.../pkg/secrets/aws`, `.../pkg/secrets/gcp`, `.../pkg/secrets/env`.
- Dependency ngoài: `cloud.google.com/go/secretmanager` (GCP), `github.com/aws/aws-sdk-go-v2/...` (AWS).

## Lưu ý / lỗi hay gặp
- `New()` không mở kết nối — quên gọi `Connect(ctx)` thì mọi call khác trả `ErrNotConnected`.
- Gọi `Connect` lần thứ hai trên cùng client trả `ErrAlreadyConnected` — không tự "reconnect", phải `Close()` trước nếu muốn kết nối lại.
- Provider AWS/GCP đăng ký qua blank import; quên import `_ "chjpcoj.com.innerlamp/pkg/secrets/aws"` (hoặc `/gcp`) thì `secrets.New()` trả `ErrUnsupportedProvider` dù `config.Provider` hợp lệ.
- Khác biệt giữa provider ở version operations — đừng giả định hành vi giống nhau:
  - `DestroyVersion`: GCP hỗ trợ đầy đủ (xoá vĩnh viễn); AWS **không hỗ trợ** xoá từng version riêng lẻ, luôn trả `ErrDestroyVersionFailed`.
  - `DisableVersion`: GCP disable đúng nghĩa; AWS chỉ chuyển version sang stage `AWSPREVIOUS` (không có khái niệm disable thật).
  - `EnableVersion`: AWS không hỗ trợ trực tiếp (hạn chế của Secrets Manager API).
- Provider `env` là read-only và chỉ dành cho local dev — gọi `Create`/`Update`/`Delete`/mọi version op đều trả `ErrUnsupportedProvider`; không dùng cho production.
- `Delete` mặc định: GCP xoá ngay lập tức, AWS có recovery window (mặc định 30 ngày, cấu hình qua `DeleteOptions.RecoveryWindowDays`, `Force: true` để xoá ngay).
- Tên secret phải qua `ValidateSecretName` (chỉ `[a-zA-Z0-9_-/+=.@]`, tối đa 255 ký tự) — ký tự lạ hoặc secret name rỗng bị từ chối trước khi gọi provider.
- Client được thiết kế tạo một lần lúc service khởi động và tái sử dụng qua các goroutine (`sync.RWMutex` bảo vệ state) — không tạo client mới cho mỗi request.
