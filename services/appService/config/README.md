# Config — services/appService/config

## Mục đích
Load cấu hình runtime của appService (server, database, JWT, storage) từ YAML qua `pkg/config`, rồi resolve các trường nhạy cảm (mật khẩu DB, JWT signing key) qua một `SecretGetter` thay vì lưu plaintext trong YAML — YAML chỉ giữ *tên* secret (`password_secret`, `secret_key_secret`), giá trị thật lấy ở bước `Load`.

## Khi nào dùng / không dùng
- Dùng khi: khởi động appService (`cmd/main.go`) cần đọc `config/config.dev.yaml` (hoặc nguồn `s3://`/`gs://` khác) và build `*AppConfig` để cấu hình `pkg/database`, `pkg/jwt`, `pkg/storage`.
- Không dùng khi: cần một cơ chế secret-resolving tổng quát cho service khác — package này là config struct + wiring riêng của appService, không phải abstraction tái dùng được (đó là việc của `pkg/config` + `pkg/secrets`).
- **`MockSecretGetter` (mock_secrets.go) chỉ dùng cho local/dev** — không dùng ở staging/production, xem phần Lưu ý bên dưới.

## API chính

### `Load(ctx context.Context, source pkgconfig.LoadSource, sg SecretGetter) (*AppConfig, error)`
Đọc YAML từ `source` (qua `pkg/config.LoadAndUnmarshal` — hỗ trợ local file, `s3://`, `gs://`), unmarshal vào `AppConfig`, rồi resolve `database.password_secret` và `jwt.secret_key_secret` qua `sg`. Lỗi resolve secret hoặc parse YAML đều được wrap và trả về nguyên (`errors.Is` hoạt động được với lỗi gốc, ví dụ `secrets.ErrSecretNotFound`).
```go
cfg, err := config.Load(ctx, pkgconfig.LoadSource("services/appService/config/config.dev.yaml"), secretGetter)
```

### `SecretGetter` interface
```go
type SecretGetter interface {
    Get(ctx context.Context, name string, opts *secrets.GetOptions) (*secrets.SecretValue, error)
}
```
Chỉ đúng 1 method — khớp chữ ký của `secrets.Client` (pkg/secrets), nên bất kỳ provider thật nào (GCP/AWS, xem `pkg/secrets/gcp`, `pkg/secrets/aws`) tự động thoả interface này mà không cần adapter.

### `AppConfig` và các struct con
`AppConfig{Server, Database, JWT, Storage}` — map trực tiếp từ YAML:
- `ServerConfig{Addr}`
- `DatabaseConfig{Host, Port, Username, PasswordSecret, Password, Database, SSLMode}` — `Password` có tag `yaml:"-"`, chỉ được set sau khi `Load` resolve `PasswordSecret`.
- `JWTConfig{Algorithm, SecretKeySecret, SecretKey, AccessTokenExpiration, RefreshTokenExpiration, Issuer, Audience}` — `SecretKey` cũng `yaml:"-"`, tương tự `Password`.
- `StorageConfig{Backend, ProjectID, DefaultBucket}`

### `Duration` (wrapper `time.Duration`)
Cho phép ghi duration dạng chuỗi trong YAML (`"15m"`, `"168h"`) vì `yaml.v3` không hỗ trợ `time.Duration` sẵn. Có `UnmarshalYAML(value *yaml.Node) error` và `Duration() time.Duration` để lấy giá trị gốc.
```go
d := cfg.JWT.AccessTokenExpiration.Duration() // time.Duration
```

### `NewMockSecretGetter(values map[string]string) *MockSecretGetter`
Tạo `SecretGetter` dev-only, tra cứu trực tiếp trong map `name -> value` truyền vào, dùng trong test hoặc khi cần seed thủ công.

### `NewMockSecretGetterFromYAML(data []byte) (*MockSecretGetter, error)`
Parse section `dev_secrets` (map phẳng `name: value`) từ cùng nội dung YAML config, build `MockSecretGetter` từ đó — đây là cách `cmd/main.go` đang dùng để lấy secret dev mà không cần secret manager thật.
```go
raw, _ := pkgconfig.LoadYAML(ctx, configSource)
sg, err := config.NewMockSecretGetterFromYAML(raw)
```

## Cấu hình / dependency
- Phụ thuộc `pkg/config` (đọc YAML, hỗ trợ local/S3/GCS) và `pkg/secrets` (chỉ dùng type `secrets.GetOptions`, `secrets.SecretValue`, `secrets.ErrSecretNotFound` — không tự chọn provider).
- Phụ thuộc thư viện ngoài `gopkg.in/yaml.v3` (trực tiếp trong `mock_secrets.go` để parse `dev_secrets`, gián tiếp qua `pkg/config` cho phần còn lại).
- Import path đúng trong repo này: `chjpcoj.com.innerlamp/services/appService/config`.
- File cấu hình mẫu: `config.dev.yaml` — các field non-sensitive để plain value, field sensitive (`password_secret`, `secret_key_secret`) chỉ ghi *tên* secret, giá trị thật nằm ở `dev_secrets:` cuối file (plaintext, chỉ để dev dùng với `MockSecretGetter`).

## Lưu ý / lỗi hay gặp
- **Tình trạng hardcoded dev trong `cmd/main.go` hiện tại**: `main.go` có comment đầu file `⚠️ DEV CONFIG - NOT FOR PRODUCTION` và `TODO before deploying`. Cụ thể: `defaultConfigPath` hardcode `"services/appService/config/config.dev.yaml"` (ghi đè được qua biến môi trường `CONFIG_PATH`, nhưng mặc định là file dev); secret được resolve qua `appConfig.NewMockSecretGetterFromYAML` (đọc plaintext từ `dev_secrets` trong YAML) thay vì `pkg/secrets/gcp`/`pkg/secrets/aws`; storage GCS dùng `UseDefaultCredentials: true` (Application Default Credentials, không phải service account thật). Việc gọi `config.Load` bản thân nó **không** hardcode gì — nó nhận `SecretGetter` từ ngoài truyền vào, nên đổi sang production chỉ cần đổi `CONFIG_PATH` trỏ `s3://`/`gs://` và đổi `MockSecretGetter` thành provider thật ở `cmd/main.go`, không cần sửa package `config` này.
- Đã kiểm tra: chuỗi literal `MOCK_DATA_MARKER` (được `CLAUDE.md` gốc repo nhắc tới) **không còn tồn tại** trong `cmd/main.go` hiện tại — theo `scripts/report/check-result.json` nó từng ở `main.go:10` ("dbConfig, jwtConfig placeholder values") nhưng bản hiện tại đã thay bằng cơ chế load YAML + `MockSecretGetter` thật sự (không phải giá trị placeholder cứng trong code Go), chỉ còn đánh dấu bằng comment `⚠️ DEV ONLY`/`DEV CONFIG`. Coi phần "Cấu hình / dependency" và mục này ở trên là nguồn đúng, không phải suy luận lại từ `MOCK_DATA_MARKER`.
- `Password`/`SecretKey` có tag `yaml:"-"` — set giá trị trực tiếp vào YAML cho 2 field này sẽ bị `yaml.Unmarshal` bỏ qua âm thầm; luôn phải đi qua `password_secret`/`secret_key_secret` + `SecretGetter`.
- Nếu YAML không có `password_secret`/`secret_key_secret` (rỗng), `resolveSecrets` **không gọi** `SecretGetter.Get` — `Password`/`SecretKey` giữ nguyên chuỗi rỗng, không lỗi (xem `TestLoad_NoSecretRefsSkipsLookup`). Dễ nhầm là bug nhưng là hành vi thiết kế.
- `Duration.UnmarshalYAML` với chuỗi rỗng `""` set về `0`, không lỗi; chuỗi không parse được (`time.ParseDuration` fail) mới trả lỗi — đừng nhầm `""` là hợp lệ với ý nghĩa "không hết hạn", nó chỉ là zero-value.
- `MockSecretGetter.Get` trả `secrets.ErrSecretNotFound` (wrap bằng `%w`) khi thiếu key — dùng `errors.Is` để phân biệt với lỗi khác khi test.
