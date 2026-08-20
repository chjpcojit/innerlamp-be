# Database — pkg/database

## Mục đích
Lớp trừu tượng database provider-agnostic: một interface `Database` chung cho MySQL, PostgreSQL, MongoDB, chọn driver lúc runtime qua factory/registry (`Register`/`New`) thay vì import cứng một driver cụ thể vào code nghiệp vụ.

## Khi nào dùng / không dùng
- Dùng khi: service cần kết nối MySQL/PostgreSQL/MongoDB nhưng muốn code gọi (repository, usecase) không phụ thuộc trực tiếp vào driver cụ thể — đổi DB type chỉ cần đổi `Config.Type` + blank import.
- Không dùng khi: cần ORM/query builder cấp cao (package này chỉ bọc `database/sql`/`mongo.Client` thô, không có model mapping); hoặc cần các API đặc thù nâng cao của một DB (transaction MongoDB qua session, connection stats MySQL/PostgreSQL...) — khi đó phải lấy connection gốc qua `GetSQLDB()`/`GetMongoClient()` hoặc type-assert sang struct driver cụ thể (`*mysql.MySQL`, `*postgresql.PostgreSQL`, `*mongodb.MongoDB`) để dùng thêm các method riêng (`GetStats`, `HealthCheck`, `Collection`, `StartSession`...).

## API chính

### `database.New(config *Config) (Database, error)`
Tạo instance DB theo `config.Type`, dựa trên factory đã đăng ký qua `Register` (driver subpackage tự đăng ký trong `init()`). Điền default cho các field còn thiếu (port, pool size, timeout...) ngay trên `config` truyền vào — **không** tự mở kết nối.
```go
import (
    "chjpcoj.com.innerlamp/pkg/database"
    _ "chjpcoj.com.innerlamp/pkg/database/postgresql" // blank import để driver tự Register trong init()
)

config := database.DefaultConfig(database.PostgreSQL)
config.Username = "postgres"
config.Password = "secret"
config.Database = "innerlamp"

db, err := database.New(config)
if err != nil {
    // config.Type rỗng/không hỗ trợ → ErrInvalidConfig / ErrUnsupportedDriver
}

ctx := context.Background()
if err := db.Connect(ctx); err != nil { // kết nối thật sự diễn ra ở đây
    // ErrConnectionFailed
}
defer db.Close()
```
Nếu quên blank import driver tương ứng, `New` trả `ErrUnsupportedDriver` (factory chưa được đăng ký).

### `database.Register(dbType DBType, factory FactoryFunc)`
Đăng ký một factory cho `DBType`. Được gọi trong `init()` của mỗi driver subpackage (`mysql.init()`, `postgresql.init()`, `mongodb.init()`) — bản thân code dùng package không cần gọi trực tiếp, chỉ cần blank import driver muốn dùng.

### `database.DefaultConfig(dbType DBType) *Config`
Trả `*Config` với default hợp lý: port theo `dbType` (MySQL 3306, PostgreSQL 5432, MongoDB 27017), `MaxOpenConns=25`, `MaxIdleConns=5`, `ConnMaxLifetime=5m`, `ConnMaxIdleTime=10m`, `Timeout=10s`, `SSLMode="disable"`.

### `Database` interface (pkg/database/database.go)
Interface chung mà cả 3 driver implement:
```go
type Database interface {
    Connect(ctx context.Context) error
    Close() error
    Ping(ctx context.Context) error
    IsConnected() bool

    Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    QueryRow(ctx context.Context, query string, args ...any) *sql.Row
    Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
    Begin(ctx context.Context) (*sql.Tx, error)

    GetSQLDB() (*sql.DB, error)
    GetMongoClient() (any, error)

    GetType() DBType
    GetConfig() *Config
}
```
- Với MySQL/PostgreSQL: `Query`/`QueryRow`/`Exec`/`Begin`/`GetSQLDB` hoạt động bình thường, `GetMongoClient` trả `ErrOperationNotSupported`.
- Với MongoDB: ngược lại — `Query`/`QueryRow`/`Exec`/`Begin`/`GetSQLDB` trả `ErrOperationNotSupported`, phải lấy client qua `GetMongoClient()` rồi type-assert `*mongo.Client`, hoặc dùng thêm method riêng của struct `mongodb.MongoDB` (`Collection`, `StartSession`, `RunCommand`, `ListCollections`...).

### `database.NewConnectionPool() *ConnectionPool`
Quản lý nhiều connection cùng lúc theo tên (`Add`, `Get`, `Remove`, `CloseAll`, `List`) — hữu ích khi service cần đồng thời một DB chính và một DB analytics chẳng hạn.
```go
pool := database.NewConnectionPool()
pool.Add("primary", db)
conn, err := pool.Get("primary")
defer pool.CloseAll()
```

### Driver constructors (`mysql.NewMySQL`, `postgresql.NewPostgreSQL`, `mongodb.NewMongoDB`)
Không cần gọi trực tiếp trong code nghiệp vụ bình thường — `database.New` sẽ gọi qua factory đã `Register`. Chỉ cần biết: constructor chỉ validate `config != nil` rồi trả struct, **chưa** mở kết nối mạng.

## Cấu hình / dependency
- `database.Config`: `Type` (`MySQL`/`PostgreSQL`/`MongoDB`), `Host`, `Port`, `Username`, `Password`, `Database`, `SSLMode` (chỉ PostgreSQL dùng), `MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime`, `ConnMaxIdleTime`, `Timeout`, `Options map[string]string`.
- `Options["dsn"]` (MySQL/PostgreSQL) hoặc `Options["uri"]` (MongoDB) cho phép ghi đè hoàn toàn chuỗi kết nối tự build từ các field trên; các key khác trong `Options` được nối thêm vào DSN/URI làm param bổ sung.
- Không tự đọc biến môi trường — `Config` truyền trực tiếp vào code (secret/mật khẩu thường lấy qua `pkg/secrets` rồi build `Config`).
- Import path đúng trong repo này: `chjpcoj.com.innerlamp/pkg/database`, `chjpcoj.com.innerlamp/pkg/database/{mysql,postgresql,mongodb}` — README cũ/rễ template còn ghi `base_project/pkg/database`, đó là tàn dư, đừng copy.
- Phụ thuộc thư viện ngoài: `github.com/go-sql-driver/mysql`, `github.com/lib/pq`, `go.mongodb.org/mongo-driver/mongo`.
- Muốn thêm driver mới: implement `Database` interface trong subpackage mới, gọi `database.Register(database.DBType("..."), NewXxx)` trong `init()`, không block I/O trong constructor (theo đúng convention `Connect(ctx)` tách biệt).

## Lưu ý / lỗi hay gặp
- **Constructor không kết nối** — `database.New()`/`NewMySQL`/`NewPostgreSQL`/`NewMongoDB` chỉ validate config, kết nối thật sự chỉ xảy ra khi gọi `db.Connect(ctx)` (fail-fast tách biệt khỏi khởi tạo). Quên gọi `Connect` thì mọi `Query`/`Exec`/`Ping` đều trả `ErrNotConnected`.
- **Quên blank import driver** → `database.New` trả `ErrUnsupportedDriver` dù `Config.Type` hợp lệ, vì factory chưa được `Register` trong `init()`.
- **`QueryRow` khi chưa connect trả `&sql.Row{}` rỗng, không trả lỗi qua kênh error** — gọi `.Scan(...)` trên đó sẽ panic (rows nội bộ là nil) thay vì trả lỗi gọn gàng. Luôn kiểm tra `IsConnected()`/gọi `Connect` trước khi `QueryRow`.
- **`New()` mutate ngay trên con trỏ `*Config` truyền vào** (điền default cho `Port`, `MaxOpenConns`, `Timeout`...) — đừng dùng lại cùng một `*Config` cho hai lần `New()` với kỳ vọng giá trị gốc không đổi.
- **MongoDB dùng nhầm field cho pool idle time**: `ConnMaxLifetime` được map sang `SetMaxConnIdleTime` của driver Mongo, còn `ConnMaxIdleTime` của `Config` không được dùng ở driver này — khác với MySQL/PostgreSQL (dùng cả `ConnMaxLifetime` lẫn `ConnMaxIdleTime` đúng nghĩa qua `database/sql`).
- **`ConnectionPool` không thread-safe** — `Add`/`Get`/`Remove`/`CloseAll`/`List` thao tác trực tiếp trên map không có mutex; nếu dùng pool từ nhiều goroutine phải tự đồng bộ hoá bên ngoài.
- **`GetSQLDB()`/`GetMongoClient()` trả `ErrOperationNotSupported` chéo nhau** — gọi `GetMongoClient()` trên MySQL/PostgreSQL hoặc `GetSQLDB()` trên MongoDB đều lỗi, không phải bug, đúng theo thiết kế "một interface, DB nào không hỗ trợ thì trả lỗi rõ ràng thay vì panic".
- `GetConfig()` trả về con trỏ `*Config` gốc (không phải bản copy) — sửa qua con trỏ này sẽ ảnh hưởng trực tiếp đến config đang dùng của connection.
