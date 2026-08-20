# Storage — pkg/storage

## Mục đích
Abstraction layer thống nhất cho thao tác lưu trữ object (upload/download/delete/copy/list, quản lý bucket, signed URL) trên nhiều cloud storage backend — hiện có GCS (`pkg/storage/gcs`) và S3 (`pkg/storage/s3`) — thông qua một interface `Client` duy nhất.

## Khi nào dùng / không dùng
- Dùng khi: service cần lưu file/object (avatar, media, export file...) lên GCS hoặc S3 mà không muốn code phụ thuộc trực tiếp vào SDK của từng cloud, hoặc muốn có thể đổi backend qua config mà không sửa code gọi.
- Không dùng khi: chỉ cần đọc secret/credential (dùng `pkg/secrets`), hoặc cần một filesystem/CDN cục bộ không phải object storage — package này chỉ implement đúng tập operation của `Client` interface.

## API chính

### `New(config *Config) (Client, error)`
Tạo storage client theo `config.Backend` (`BackendGCS`/`BackendS3`) qua factory đã đăng ký. Validate config trước, trả `ErrInvalidConfig` nếu thiếu field bắt buộc, trả `ErrUnsupportedBackend` nếu chưa blank-import package driver tương ứng. **Không** mở kết nối mạng — chỉ dựng struct.
```go
import (
    "chjpcoj.com.innerlamp/pkg/storage"
    _ "chjpcoj.com.innerlamp/pkg/storage/gcs" // đăng ký backend GCS
)

client, err := storage.New(&storage.Config{
    Backend: storage.BackendGCS,
    Timeout: 30 * time.Second,
    GCS: &storage.GCSConfig{
        ProjectID:             "my-project-id",
        UseDefaultCredentials: true,
        DefaultBucket:         "my-bucket",
    },
})
```

### `Client.Connect(ctx context.Context) error` / `Close() error` / `IsConnected() bool`
`Connect` mới thực sự tạo kết nối tới GCS/S3 (đọc credentials, dựng SDK client). Gọi một lần lúc service khởi động, tái sử dụng client cho toàn bộ request; `Close` lúc shutdown.
```go
client, _ := storage.New(cfg)
if err := client.Connect(ctx); err != nil { log.Fatal(err) }
defer client.Close()
```

### `Client.Upload(ctx, bucket, objectName string, reader io.Reader, opts *UploadOptions) (*ObjectInfo, error)`
Upload object, `opts` có thể `nil` (dùng default). Hỗ trợ `ContentType`, `Metadata`, `StorageClass`, `ACL`, `CacheControl`, `Conditions` (vd. `DoesNotExist` để chặn ghi đè).
```go
info, err := client.Upload(ctx, "my-bucket", "avatars/u1.png", file, &storage.UploadOptions{
    ContentType: "image/png",
    ACL:         storage.ACLPrivate,
})
```

### `Client.Download(ctx, bucket, objectName string, writer io.Writer, opts *DownloadOptions) (*ObjectInfo, error)`
Ghi nội dung object vào `writer`. `opts.Range` cho phép tải một phần (`ByteRange{Start, End}`, `End: -1` = tới cuối file).

### `Client.Delete/GetObjectInfo/UpdateMetadata/Copy/Exists`
CRUD + metadata cho object đơn lẻ. `Copy` copy giữa hai bucket/object (kể cả cross-bucket), có thể override `ContentType`/`Metadata`/`StorageClass` qua `CopyOptions`.

### `Client.List(ctx, bucket string, opts *ListOptions) (*ListResult, error)` / `ListAll(...) ([]*ObjectInfo, error)`
`List` trả một trang (`ListResult.NextPageToken` để phân trang thủ công qua `opts.PageToken`), `ListAll` tự động lặp hết các trang. Dùng `Prefix`/`Delimiter: "/"` để liệt kê kiểu thư mục (`ListResult.Prefixes`).

### `Client.CreateBucket/DeleteBucket/GetBucketInfo/BucketExists/ListBuckets`
Quản lý bucket. `DeleteBucket` yêu cầu bucket rỗng (giới hạn của cả GCS lẫn S3, không phải giới hạn riêng của package).

### `Client.SignedURL(ctx, bucket, objectName string, opts *SignedURLOptions) (string, error)`
Sinh URL tạm thời để truy cập trực tiếp object (không qua service). `opts` **bắt buộc** phải có, và `opts.Method` + `opts.Expires > 0` là điều kiện validate cứng ở cả hai driver — thiếu sẽ trả `ErrInvalidConfig` trước khi gọi SDK.
```go
url, err := client.SignedURL(ctx, "my-bucket", "private-file.pdf", &storage.SignedURLOptions{
    Method:  storage.SignedURLMethodGET,
    Expires: 15 * time.Minute,
})
```

### `DetectContentType(objectName string) string`
Đoán MIME type từ đuôi file khi không truyền `ContentType` (map các đuôi phổ biến, fallback `application/octet-stream`).

### `ValidateBucketName(name string) error` / `ValidateObjectName(name string) error`
Validate tên trước khi gọi API — bucket 3-63 ký tự, object không rỗng, không quá 1024 ký tự, không có `//` liên tiếp. Cả hai driver tự gọi các hàm này ở đầu mỗi method, nhưng caller nên gọi trước để fail sớm và trả lỗi rõ ràng hơn cho người dùng.

### `CopyMetadata(map[string]string) map[string]string` / `MergeMetadata(maps ...map[string]string) map[string]string`
Helper copy/merge metadata map an toàn (không sửa map gốc).

## Cấu hình / dependency
- `Config{Backend, Timeout, GCS *GCSConfig, S3 *S3Config, Options map[string]string}` — `Backend` và `Timeout` (>0) luôn bắt buộc; đúng đúng một trong `GCS`/`S3` bắt buộc theo `Backend`. Không có default nào được áp — `Config.Validate()` từ chối config thiếu field.
  - `GCSConfig`: `ProjectID` bắt buộc; cần `UseDefaultCredentials: true` **hoặc** `CredentialsFile`/`CredentialsJSON`.
  - `S3Config`: `Region` bắt buộc; `AccessKeyID`/`SecretAccessKey` optional (dùng IAM role nếu bỏ trống); `Endpoint`+`UsePathStyle: true` để dùng dịch vụ S3-compatible (MinIO, DigitalOcean Spaces...).
- Không tự đọc biến môi trường — toàn bộ config truyền qua tham số (credentials thường lấy qua `pkg/secrets` rồi build `Config`).
- Driver phải được đăng ký bằng blank import trước khi gọi `storage.New`: `_ "chjpcoj.com.innerlamp/pkg/storage/gcs"` hoặc `_ "chjpcoj.com.innerlamp/pkg/storage/s3"`.
- Import path đúng trong repo này: `chjpcoj.com.innerlamp/pkg/storage...` (README của `s3/`, `gcs/`, `examples/` và `QUICK_START.md` trong thư mục này vẫn còn ghi `base_project/pkg/storage...` — đó là tàn dư từ template, đừng copy).

## Lưu ý / lỗi hay gặp
- `New()` chỉ validate + dựng struct, **không** mở kết nối — phải gọi `Connect(ctx)` riêng, nếu không mọi operation khác sẽ trả `ErrNotConnected`. Đúng convention connection-lifecycle chung của repo (constructor validate-only, `Connect` mới I/O thật).
- Quên blank-import package driver (`gcs`/`s3`) → `New()` trả `ErrUnsupportedBackend` dù config đúng, vì `init()` của driver chưa chạy để đăng ký factory.
- Tạo client một lần lúc service khởi động và tái sử dụng cho mọi request (đặc biệt trên Cloud Run/serverless) — đừng gọi `storage.New`/`Connect` trong từng request; SDK client bên dưới (`s3.Client`, `cloud.google.com/go/storage.Client`) tự quản lý connection pool.
- `SignedURL` yêu cầu `opts != nil`, `opts.Method` hợp lệ và `opts.Expires > 0` — thiếu sẽ lỗi `ErrInvalidConfig` ngay tại package này trước khi chạm SDK.
- `DeleteBucket` chỉ xóa được bucket rỗng — lỗi này đến từ giới hạn của GCS/S3, không phải bug của package.
- `StorageClass` là kiểu string chung cho cả hai backend nhưng tập giá trị khác nhau (GCS: `STANDARD`/`NEARLINE`/`COLDLINE`/`ARCHIVE`; S3: `STANDARD`/`INTELLIGENT_TIERING`/`STANDARD_IA`/`ONEZONE_IA`/`GLACIER`/`DEEP_ARCHIVE`) — truyền nhầm class của backend này cho backend kia sẽ bị driver từ chối hoặc SDK trả lỗi, package không tự map đổi qua lại.
