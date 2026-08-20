# Config — pkg/config

## Mục đích
Load nội dung config dạng YAML từ local file, AWS S3, hoặc Google Cloud Storage bằng một hàm duy nhất, rồi unmarshal vào struct Go. Không phải là một config-management framework đầy đủ (không watch file, không hot-reload, không merge phân cấp phức tạp) — chỉ là lớp I/O mỏng để đọc YAML từ nhiều nguồn khác nhau bằng cùng một API.

## Khi nào dùng / không dùng
- Dùng khi: service cần đọc file config YAML lúc khởi động và muốn cùng một code path chạy được cả khi config nằm trên máy local (dev) lẫn khi nằm trên S3/GCS (prod/staging), chỉ khác nhau ở giá trị `source` string.
- Không dùng khi: cần đọc secret riêng lẻ (API key, DB password...) — dùng `pkg/secrets` (env/aws/gcp) cho việc đó rồi tự resolve các trường `*_secret` trong config sau khi unmarshal (xem cách `services/appService/config` làm). `pkg/config` không tự tích hợp với `pkg/secrets`.

## API chính

### `LoadYAML(ctx context.Context, source LoadSource) ([]byte, error)`
Đọc raw bytes YAML từ nguồn được chỉ định. Tự nhận diện nguồn qua prefix của `source`:
- Không có prefix → đọc file local (`os.Open`).
- `s3://bucket/key` → đọc từ AWS S3 (dùng `awsconfig.LoadDefaultConfig`, tức là credential lấy từ env/shared config/IAM role mặc định của AWS SDK).
- `gs://bucket/object` → đọc từ Google Cloud Storage (dùng `storage.NewClient` với Application Default Credentials).
```go
data, err := config.LoadYAML(ctx, config.LoadSource("s3://my-bucket/configs/prod.yaml"))
```

### `UnmarshalYAML(data []byte, dest interface{}) error`
Wrapper mỏng quanh `yaml.Unmarshal` (gopkg.in/yaml.v3).

### `LoadAndUnmarshal(ctx context.Context, source LoadSource, dest interface{}) error`
Gộp `LoadYAML` + `UnmarshalYAML` — hàm dùng phổ biến nhất trong thực tế.
```go
var cfg AppConfig
err := config.LoadAndUnmarshal(ctx, config.LoadSource("config/configs.yaml"), &cfg)
```

### `MergeYAML(base []byte, overrides map[string]interface{}) ([]byte, error)`
Merge nông (shallow, chỉ ghi đè key cấp 1) một map override vào YAML gốc, trả về YAML bytes mới. Comment trong code ghi rõ đây là "simple implementation" — nếu cần deep merge thì tự thêm thư viện như `github.com/imdario/mergo`, package này không làm.

## Cấu hình / dependency
- `LoadSource` chỉ là `type LoadSource string` — không có struct config riêng, không đọc biến môi trường nào trực tiếp trong package này.
- S3: phụ thuộc `github.com/aws/aws-sdk-go-v2/...`, credential resolve theo cơ chế mặc định của AWS SDK (env var, `~/.aws/config`, IAM role) — package không nhận credential qua tham số.
- GCS: phụ thuộc `cloud.google.com/go/storage`, dùng Application Default Credentials (biến `GOOGLE_APPLICATION_CREDENTIALS` hoặc ADC trên GCP runtime).
- **Không phụ thuộc `pkg/secrets`** dù `CLAUDE.md` ở gốc repo có mô tả tổng quát là "config load... và cấu hình qua pkg/secrets" — với code hiện tại, `pkg/config` tự quản lý credential cloud độc lập; việc phối hợp với `pkg/secrets` (resolve secret field sau khi unmarshal) do code gọi (ví dụ `services/appService/config`) tự làm, không phải trách nhiệm của package này.
- Import path đúng trong repo này: `chjpcoj.com.innerlamp/pkg/config`.

## Lưu ý / lỗi hay gặp
- `source` không có prefix `s3://` hay `gs://` sẽ luôn được coi là local file path — gõ sai scheme (ví dụ `S3://` viết hoa, hoặc thiếu `//`) sẽ âm thầm bị coi là đường dẫn local rồi báo lỗi "failed to open config file" gây khó hiểu.
- URI S3/GCS phải có dạng `scheme://bucket/object` với cả bucket lẫn object non-empty — thiếu object (`s3://bucket-only`) trả lỗi `invalid S3 URI ... expected s3://bucket/object` từ `parseCloudURI`.
- `LoadYAML`/`LoadAndUnmarshal` tạo client S3/GCS mới và load default credential mỗi lần gọi — không cache client, không phù hợp gọi lặp lại tần suất cao (hot path); chỉ nên dùng lúc khởi động service.
- `MergeYAML` chỉ ghi đè key cấp 1 (top-level), không merge đệ quy vào nested map/struct — override một field lồng sâu sẽ thay thế nguyên cả nhánh cha, không merge từng field con.
