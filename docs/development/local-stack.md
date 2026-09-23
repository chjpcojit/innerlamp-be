# Local stack — `docker-compose.yml`

Tài liệu này mô tả ý nghĩa từng thành phần trong `docker-compose.yml`, cách chạy, và các điểm dev-only cần biết. Phần thao tác với Postgres qua CLI nằm ở [`database-cli.md`](./database-cli.md).

> Nguồn sự thật là chính `docker-compose.yml` và các file dưới `observability/`. Tài liệu này giải thích *tại sao*, không thay thế việc đọc file.

---

## 1. Thành phần

| Service | Image | Port (host→container) | Volume | Phụ thuộc | Vai trò |
|---|---|---|---|---|---|
| `postgres` | `postgres:16-alpine` | `5432:5432` | `postgres-data:/var/lib/postgresql/data` | — | DB cho appService; khớp nguyên `services/appService/config/config.dev.yaml` |
| `migrate` | `postgres:16-alpine` | — (không publish) | `./migrations:/migrations:ro` | `postgres` healthy | One-shot: apply `migrations/*.sql` rồi thoát |
| `loki` | `grafana/loki:3.7.8` | `3100:3100` | `loki-data:/loki` | — | Nhận và lưu log; config `observability/loki/config.yml` |
| `alloy` | `grafana/alloy:v1.19.2` | `12345:12345` | `./observability/alloy/config.alloy` (ro), `./observability/logs:/var/log/innerlamp:ro`, `alloy-data` | `loki` | Tail file log trên host → push vào Loki |
| `grafana` | `grafana/grafana:13.2.2` | `3000:3000` | `./observability/grafana/provisioning` (ro), `grafana-data` | `loki` | UI truy vấn log; datasource Loki được provision sẵn |

### `postgres`

Bốn giá trị `POSTGRES_USER=postgres`, `POSTGRES_PASSWORD=postgres`, `POSTGRES_DB=innerlamp` khớp **chính xác** `database.*` trong `config.dev.yaml`, nên appService chạy `go run` trên host kết nối được qua `localhost:5432` mà không phải sửa config.

`healthcheck` dùng `pg_isready`. Nó tồn tại vì hai lý do: `migrate` cần biết DB đã sẵn sàng, và `db.Connect()` trong `cmd/main.go` có timeout cứng 10 giây — không có healthcheck thì server có thể đua với DB lúc mới khởi động.

### `migrate`

One-shot, `restart: "no"`: chạy xong là thoát, không chiếm port. Nó dùng chính image `postgres`, tận dụng `psql` có sẵn nên **không cần cài PostgreSQL client trên máy host**.

Logic:

```sh
for f in /migrations/*.sql; do
  case "$f" in
    *.down.sql) echo "skipping $f (down migration)"; continue ;;
  esac
  psql ... -v ON_ERROR_STOP=1 -f "$f"
done
```

- Thứ tự áp là **thứ tự glob** (`0001` < `0002` < `0003` theo lexicographic, trùng thứ tự số).
- `ON_ERROR_STOP=1`: một file lỗi thì dừng ngay, không để schema nửa vời.
- Bỏ qua `*.down.sql`: các file migration hiện tại là bare `000N_*.sql`, không có cặp `.up/.down`, nhưng nếu sau này có thì glob sẽ khớp cả hai và `x.down.sql` sort **trước** `x.up.sql` — tức rollback chạy trước migration. Nhánh `case` chặn sẵn cái đó.
- Không dùng `scripts/migrate.sh` / golang-migrate: công cụ đó chỉ nhận cặp `.up.sql`/`.down.sql` và **không nhìn thấy** file bare. Xem `migrations/README.md`.

Chạy lại `migrate` là **idempotent** (đã kiểm chứng: exit 0), vì mọi câu lệnh là `CREATE TABLE IF NOT EXISTS` / `ADD COLUMN IF NOT EXISTS` / `CREATE INDEX IF NOT EXISTS` / `DROP COLUMN IF EXISTS`.

### Log stack (`loki` + `alloy` + `grafana`)

Luồng log:

```
go run (host) --tee--> observability/logs/*.log --tail--> Alloy --> Loki --> Grafana
```

- **Loki** chạy single-binary, filesystem-backed, `auth_enabled: false`. Đây là cấu hình cho laptop, không phải topology production (không object storage, không replication, không auth). `reject_old_samples_max_age: 168h` để Alloy replay file từ đầu không bị Loki từ chối vì timestamp cũ.
- **Alloy** là bản kế nhiệm Promtail (Promtail đã EOL). Nó đọc file qua `local.file_match` + `loki.source.file`, gắn label `service="appservice"`, `env="local"`. `tail_from_end` cố ý để mặc định (`false`): khi Alloy restart, cả file được đọc lại từ đầu — đúng cho dev. Đặt `true` sẽ bỏ qua toàn bộ log đã ghi.
- **Grafana** tự nạp datasource Loki từ `observability/grafana/provisioning/datasources/loki.yml`, nên `up -d` xong là query được ngay, không phải bấm tay.

---

## 2. Cách chạy

### Đường nhanh — `make`

```bash
make            # liệt kê target
make up         # dựng cả stack
make run        # chạy appService trên host (tự gọi `make up` trước), tee log cho Alloy
```

Sau đó: appService `http://localhost:8080`, Grafana `http://localhost:3000`.

Danh sách target:

| Target | Việc nó làm |
|---|---|
| `make help` | Liệt kê target |
| `make urls` | In các endpoint local |
| `make up` / `make down` / `make restart` | Dựng / dừng (giữ data) / dựng lại |
| `make reset` | `down -v` rồi `up` — **xoá sạch** data Postgres + Loki + Grafana |
| `make status` / `make logs` | `docker compose ps` / `logs -f` mọi container |
| `make run` | `go run ./services/appService/cmd`, tee vào `observability/logs/appservice.log` |
| `make build` / `test` / `vet` | `go build ./...` / `go test -short ./...` / `go vet ./...` |
| `make db-up` | Chỉ dựng Postgres + apply migrations (không Loki/Grafana/Alloy) |
| `make migrate` | Apply lại `migrations/` vào Postgres đang chạy |
| `make psql` | Mở shell `psql` tương tác |
| `make db-tables` / `db-counts` | Liệt kê bảng / số hàng ước lượng |
| `make db-accounts` / `db-sessions` / `db-otp` | Xem users / sessions / OTP + send limits |
| `make db-reset-phone PHONE=+84901234567` | Xoá toàn bộ state của một số để test lại flow |
| `make db-dump` | In `pg_dump` ra stdout (`make db-dump > backup.sql`) |
| `make log-query` | In 20 dòng log mới nhất của appService từ Loki |

### Chạy thủ công (không dùng make)

```bash
# Từ repo root — bắt buộc, vì cmd/main.go đọc config theo đường dẫn tương đối
docker compose up -d
go run ./services/appService/cmd 2>&1 | tee -a observability/logs/appservice.log
```

Apply lại migration sau khi thêm file mới:

```bash
docker compose run --rm migrate
```

Xoá sạch để chạy lại từ đầu:

```bash
docker compose down -v && docker compose up -d
```

---

## 3. Vì sao appService không nằm trong compose

`postgres` publish `5432:5432`, nên process trên host vẫn kết nối được qua `localhost:5432` — đúng như `config.dev.yaml` khai báo. Việc app nằm trong Docker hay không **không** quyết định nó có chạm được Postgres hay không; port publishing là cầu nối.

Giữ app trên host vì:

1. **Vòng lặp sửa–chạy nhanh.** `go run` restart trong vài giây; đóng gói image thì mỗi lần sửa phải rebuild.
2. **Debug trực tiếp.** Breakpoint, log ra terminal, `go test` ngay tại chỗ.
3. **Repo chưa có Dockerfile nào.** Đưa app vào compose đòi hỏi thêm Dockerfile + một config riêng (`database.host` phải là `postgres` chứ không phải `localhost`), tức thêm artifact mới chưa ai kiểm chứng.

Hệ quả duy nhất: app không phải container nên Alloy **không** đọc được stdout của nó như log container — phải `tee` ra `observability/logs/appservice.log` cho Alloy tail.

Đây cũng là mô hình `services/appService/e2e/` đang dùng: Postgres trong container + binary chạy trên host.

---

## 4. Giới hạn dev-only

Các điểm sau **chỉ dùng local**, tuyệt đối không copy sang môi trường thật:

- Grafana bật **anonymous admin**: `GF_AUTH_ANONYMOUS_ENABLED=true` + `GF_AUTH_ANONYMOUS_ORG_ROLE=Admin` + `GF_AUTH_DISABLE_LOGIN_FORM=true`. Ai chạm được port 3000 là có full admin — đừng expose port này.
- Loki `auth_enabled: false`, không retention, lưu trên filesystem.
- Postgres dùng `postgres/postgres`, không TLS.
- Bên phía app (không thuộc compose): secret đọc từ `dev_secrets` trong YAML (`⚠️ DEV ONLY`), SMS chỉ log (`pkg/sms/logsms`), OTP hardcode `111111`.

---

## 5. Xử lý sự cố

| Hiện tượng | Nguyên nhân / cách xử lý |
|---|---|
| `failed to connect to database` khi `go run` | Postgres chưa lên. Chạy `make up` (hoặc `make db-up`) trước. |
| `bind: address already in use` khi `up` | Port 5432/3000/3100/12345 đã bị chiếm. Kiểm tra `lsof -nP -iTCP:<port> -sTCP:LISTEN`, hoặc dừng Postgres local. |
| `loki /ready` trả `503` ngay sau `up` | Bình thường trong vài giây đầu (ring chưa sẵn sàng). Chờ rồi thử lại. |
| Log không thấy trong Grafana | Alloy chỉ thấy file **sau khi** file tồn tại. Đảm bảo app chạy bằng lệnh có `tee` (target `make run`), và thư mục `observability/logs/` do user sở hữu (đã commit `.gitkeep`). |
| `migrate` báo lỗi schema | Xem log: `docker compose logs migrate`. Nếu schema đang dở dang, `make reset` để làm lại từ đầu. |
| `0002 ... column already exists` | Không còn xảy ra với file hiện tại. Nếu gặp, bạn đang dùng bản migration cũ. |
| Muốn xoá sạch mọi thứ | `make reset` (= `docker compose down -v && docker compose up -d`). |

---

## 6. Liên quan

- [`database-cli.md`](./database-cli.md) — truy cập Docker bằng CLI, xem dữ liệu Postgres.
- [`../../README.md`](../../README.md) §7 — bản tóm tắt ngắn.
- [`../../migrations/README.md`](../../migrations/README.md) — thứ tự và bất biến schema.
- [`../../services/appService/config/config.dev.yaml`](../../services/appService/config/config.dev.yaml) — giá trị cấu hình app khớp với compose.
