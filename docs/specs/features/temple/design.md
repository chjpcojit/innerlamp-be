# Design — temple

Version: 1.2
Last updated: 2026-10-03
Status: implemented (2026-09-30) — 3 route đọc chạy thật qua binary + Postgres dev; có test usecase/handler/seed parser/repository. Thiếu: E2E, OKF concept; seed chỉ chạy bằng CLI với config dev.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Ba route `GET /v1/temples/*` đều công khai, không đi qua middleware xác thực. Feature chạm đủ bốn layer Clean Architecture của `services/appService` (`domain` → `usecase` → `repository` / `delivery/http`, nối dây ở `cmd/main.go`) và sở hữu hai migration `0005_temples.sql` (bảng + index) và `0011_temple_content.sql` (ba cột nội dung biên tập). Dữ liệu vào bảng bằng lệnh `cmd/seedtemples` (upsert theo `id`), **không** có seed tự động khi service khởi động và file JSON **không** được nhúng vào binary. Cùng hình dạng với `deity` và `article` (top + list phân trang + detail, cấu hình `top_limit`/`page_size`); khác `article` ở cách seed (CLI, upsert ghi đè) và khác `deity` ở chỗ có một danh sách chung thay vì danh sách theo loại.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/temple.go` | `Temple` (mọi cột; `""` = chưa có ảnh), `TempleAmenities` (4 bool), `TemplePage`. Không JSON/db tag. |
| Domain / Ranking | `domain/temple.go` | Bảng `heritagePriority` (8 hạng → 1..8) và `HeritagePriority(rank) (int, bool)`; nguồn duy nhất cho thứ tự hạng, phải khớp `chk_temples_heritage_rank`. |
| Domain / Errors | `domain/temple.go` | `ErrTempleNotFound` và `ErrInvalidPage` — `ErrInvalidPage` **định nghĩa ở file này** nhưng dùng chung với `deity`/`article`; `domain/errors.go` không chứa chúng. |
| Domain / Repository interface | `domain/temple.go` | `TempleRepository`: `ListTop`, `List`, `FindByID`, `Upsert`. Thứ tự `heritage_priority, established_year, id` là một phần của hợp đồng. |
| Domain / Mock | `domain/mock_temple_repository.go` | Mock viết tay cho test usecase. |
| Usecase | `services/appService/usecase/temple_usecase.go` | `Top`, `List` (tính offset/`total_pages`, chặn `page < 1` hoặc `> MaxInt32`, `items` nil → `[]`), `Get`. Kích thước trang lấy từ `config.TempleConfig`. |
| Repository | `services/appService/repository/postgres_temple_repository.go` | SQL cho bảng `temples` với `templeColumns`/`templeOrder` dùng chung mọi lần đọc; `Upsert` chạy một transaction với `ON CONFLICT (id) DO UPDATE` ghi đè mọi cột; `pq.Array` cho `aliases`/`visiting_notes`; `sql.ErrNoRows` → `ErrTempleNotFound`. |
| Seed CLI | `services/appService/cmd/seedtemples/main.go`, `parse.go` | `main`: đọc file (`-file`), nạp config + secret dev, kết nối DB, gọi `Upsert`. `parseTemples`: bỏ chú thích `/* */` (biết chuỗi literal), giải mã, kiểm id/hạng/vùng. |
| Data | `assets/data/temple.json` | 179 chùa, JSONC. Đọc từ đĩa lúc chạy seed (đường dẫn tính từ thư mục hiện tại) — không `go:embed`. |
| Delivery / Handler | `services/appService/delivery/http/temple_handler.go` | `Top`, `List`, `Get`; parse `page`; interface `templeUsecase` khai báo phía consumer. |
| Delivery / DTO | `delivery/http/temple_handler.go` | `templeSummaryResponse`, `templeDetailResponse` (nhúng summary), `locationResponse`, `amenitiesResponse`, `templeTopResponse`, `templeListResponse` — JSON tag chỉ nằm ở đây. `nullable("")` → `nil` cho hai URL ảnh; `aliases`/`visiting_notes` nil → `[]`. |
| Delivery / Error map | `delivery/http/errors.go` | `ErrTempleNotFound` → `404 TEMPLE_NOT_FOUND`; `ErrInvalidPage` → `400 INVALID_PAGE` (`details.field = "page"`); còn lại → `500 INTERNAL_ERROR`. |
| Delivery / Fallback | `delivery/http/fallback.go` | `WithAPIFallback` đổi 404/405 của `ServeMux` thành envelope `NOT_FOUND` / `METHOD_NOT_ALLOWED`, giữ header `Allow`. Dùng chung mọi route, không riêng temple. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `TempleConfig{TopLimit, PageSize}`, block `temple:` (dev: 10 và 20). |
| Wiring | `services/appService/cmd/main.go` | Dựng repo → usecase → handler; đăng ký 3 route. Không gọi seed. Không dính khối `⚠️ DEV ONLY` nào của `main.go` (riêng `cmd/seedtemples` có khối DEV ONLY của nó). |
| Migration | `migrations/0005_temples.sql`, `0011_temple_content.sql` | Bảng `temples` (ba `CHECK`: `type`, `region`, `heritage_rank`), index `idx_temples_ranking (heritage_priority, established_year, id)`; `ALTER TABLE` thêm `introduction`, `architecture_heritage`, `visiting_notes`. |
| API spec | `docs/specs/api/openapi.yaml` | **Đã có** 3 route (tag Temples) và schema; `web/docs/public/openapi.yaml` khớp. |
| ERD | `docs/specs/db/erd.mmd`, `erd.svg` | **Đã có** bảng `temples` (không FK tới bảng nào). |

Ghi chú về route: `mux.HandleFunc("GET /v1/temples/top", …)` là segment literal nên `ServeMux` ưu tiên nó hơn `GET /v1/temples/{id}`.

## Request flow

`top` và `detail` chỉ một câu SQL nên giữ dạng ASCII; `list` (đếm rồi lấy trang) và seed có nhiều bước nên nằm ở Step-by-step Flow và `flow.md`.

```
GET /v1/temples/top
  └─ public
       └─ TempleHandler.Top                            (delivery/http)
            └─ TempleUsecase.Top(ctx)                  (usecase)
                 └─ TempleRepository.ListTop(limit)    (repository)
                      └─ SELECT <cột> FROM temples ORDER BY heritage_priority, established_year, id LIMIT $1

GET /v1/temples/{id}
  └─ public
       └─ TempleHandler.Get                            (delivery/http)
            └─ TempleUsecase.Get(ctx, id)              (usecase)
                 └─ TempleRepository.FindByID(id)      (repository)
                      └─ SELECT <cột> FROM temples WHERE id = $1
                           └─ sql.ErrNoRows → ErrTempleNotFound → 404 TEMPLE_NOT_FOUND
```

> Xem `flow.md` cho sơ đồ sequence Mermaid của hai flow có nhiều bước bên dưới (mirror 1:1).

## Step-by-step Flow — Danh sách phân trang (`GET /v1/temples?page=N`)

Hai lần truy vấn nối tiếp (đếm, rồi lấy trang) nên flow này có Step-by-step riêng.

```
Step 1 — Đọc và kiểm tra `page` (handler)
- Trigger: request đến route công khai
- Pre-condition: —
- Call: r.URL.Query().Get("page"); rỗng → page = 1, ngược lại strconv.Atoi
    - Success → Step 2
    - Failure (không phải số nguyên, kể cả tràn int) → writeDomainError(ErrInvalidPage) → 400 INVALID_PAGE, details.field = "page"

Step 2 — Kiểm tra khoảng của `page` (usecase)
- Trigger: handler gọi TempleUsecase.List(ctx, page)
- Pre-condition: page là số nguyên
- Call: TempleUsecase.List — page < 1 hoặc page > math.MaxInt32
    - Success → Step 3 (offset = (page-1) * page_size)
    - Failure → domain.ErrInvalidPage → 400 INVALID_PAGE

Step 3 — Đếm tổng số chùa (repository)
- Trigger: usecase gọi TempleRepository.List(offset, limit)
- Pre-condition: offset/limit hợp lệ
- Call: SELECT count(*) FROM temples
    - Success → total, sang Step 4
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR

Step 4 — Lấy trang (repository)
- Trigger: Step 3 xong
- Pre-condition: có total
- Call: SELECT <cột> FROM temples ORDER BY heritage_priority ASC, established_year ASC, id ASC OFFSET $1 LIMIT $2
    - Success → items (rỗng nếu offset vượt cuối) và total, sang Step 5
    - Failure (lỗi DB / scan) → error bọc %w → 500 INTERNAL_ERROR
    - Transaction boundary: không có — Step 3 và Step 4 là hai câu lệnh độc lập, không chung snapshot

Step 5 — Dựng trang và trả response (usecase + handler)
- Trigger: repository trả về
- Pre-condition: items, total
- Call: items nil → []; total_pages = (total + page_size - 1) / page_size; handler map sang DTO summary (URL ảnh rỗng → null)
    - Success → 200 {items, page, page_size, total, total_pages}
    - Failure → không có nhánh lỗi mới
```

## Step-by-step Flow — Nạp dữ liệu bằng `cmd/seedtemples`

Đọc file, kiểm tra, kết nối và ghi là các bước nối tiếp, trong đó bước ghi là side effect bền vững. Đây là lệnh chạy tay, **không** thuộc quá trình khởi động của appService.

```
Step 1 — Đọc file nguồn (seedtemples/main.go)
- Trigger: operator chạy `go run ./services/appService/cmd/seedtemples [-file <path>]`
- Pre-condition: thư mục hiện tại là gốc repo nếu dùng đường dẫn mặc định `assets/data/temple.json`
- Call: os.ReadFile(*file)
    - Success → Step 2
    - Failure (không có file / không đọc được) → log.Fatalf "read <file>", thoát; chưa chạm DB

Step 2 — Bỏ chú thích và kiểm tra dữ liệu (seedtemples/parse.go)
- Trigger: Step 1 xong
- Pre-condition: nội dung file trong bộ nhớ
- Call: parseTemples → stripBlockComments (bỏ `/* */`, bỏ qua chuỗi literal) → json.Unmarshal → với mỗi chùa: id không trùng, regionOf(id) hợp lệ, domain.HeritagePriority(rank) biết
    - Success → danh sách domain.Temple (Region từ id, HeritagePriority từ hạng), sang Step 3
    - Failure (chú thích không đóng, JSON hỏng, không có chùa, id trùng, id sai dạng hoặc vùng lạ, hạng lạ) → log.Fatalf "parse <file>", thoát; chưa chạm DB

Step 3 — Nạp config và kết nối DB (seedtemples/main.go)
- Trigger: Step 2 xong
- Pre-condition: có danh sách chùa hợp lệ; `CONFIG_PATH` (mặc định `services/appService/config/config.dev.yaml`)
- Call: pkgconfig.LoadSource/LoadYAML → appConfig.NewMockSecretGetterFromYAML (⚠️ DEV ONLY) → appConfig.Load → database.New + db.Connect (timeout 30 giây)
    - Success → Step 4
    - Failure (thiếu config/secret, không kết nối được DB) → log.Fatalf, thoát; chưa ghi gì

Step 4 — Upsert mọi chùa (repository)
- Trigger: Step 3 xong
- Pre-condition: DB kết nối được; migration 0005 và 0011 đã apply
- Call: TempleRepository.Upsert → INSERT INTO temples (24 cột) … ON CONFLICT (id) DO UPDATE SET <mọi cột> , updated_at = now() cho từng chùa
    - Success → commit, sang Step 5
    - Failure (lỗi DB: CHECK `type`/`region`/`heritage_rank`, thiếu bảng hoặc cột, mất kết nối) → rollback toàn bộ → log.Fatalf "seed temples"
    - Transaction boundary: mọi INSERT/UPDATE của Step 4 chung MỘT transaction — hoặc đủ hết hoặc không dòng nào

Step 5 — Ghi log (seedtemples/main.go)
- Trigger: Upsert thành công
- Pre-condition: —
- Call: log.Printf "seeded %d temples from %s"
    - Success → thoát 0
    - Failure → không có nhánh lỗi
```

## Deliberately stubbed for now (if applicable)

Không có gì bị stub ở phía đọc: mọi method của `TempleRepository` đều có SQL thật. Phần thuộc về vận hành nằm ở Known gaps (seed chỉ dùng config dev).

## Known gaps carried by this feature

- **Không có E2E**: `services/appService/e2e/` không có gì về temple. Việc "đã chạy thật" là chạy tay trên server dev với 179 chùa (theo `api-contract.md`); lượt viết spec này không gọi lại server.
- **Test repository cần Postgres và xoá dữ liệu dev**: `postgres_temple_repository_test.go` gọi `connectTestDB`, bỏ qua khi `-short`; helper `truncateAll` chạy `TRUNCATE … temples, deities, articles, users …` trên DB `innerlamp` (mặc định `localhost:5432`). Chạy xong bảng `temples` trống, phải seed lại bằng `cmd/seedtemples`; `deities` seed lại bằng `cmd/seeddeities`, `articles` tự nạp ở lần khởi động kế tiếp. `go test -short` chỉ chạy usecase, handler và seed parser. Lượt viết spec này chỉ chạy `-short`, chưa chạy test repository.
- **Seed không tự chạy và dùng config dev**: appService khởi động không đụng bảng `temples`; `cmd/seedtemples` gắn với `config.dev.yaml` và secret dev (`⚠️ DEV ONLY`) nên chưa dùng được cho staging/production (`⚠️ Not confirmed` trong `requirements.md`).
- **Upsert ghi đè mọi cột và không xoá**: chạy lại seed đưa mọi cột về đúng nội dung file (kể cả cột đã sửa tay trong DB), và chùa bị xoá khỏi file **vẫn nằm trong bảng** — không có bước dọn.
- **Validation ở seeder chỉ một phần**: `parseTemples` kiểm id, vùng, hạng; `type` chỉ bị chặn bởi `CHECK` ở DB (một `type` lạ làm rollback cả lần seed), còn `name`, `sect`, toạ độ, năm không bị kiểm — rỗng vẫn qua.
- **`sect` không có `CHECK`**: khác `type`/`heritage_rank`, giá trị mới của `sect` vào được DB mà không cần migration; client phải coi đây là tập mở.
- **Đếm và lấy trang không cùng snapshot**: nếu có dòng đổi giữa Step 3 và Step 4 của flow phân trang, `total` có thể lệch `items`. Dữ liệu chỉ đổi khi chạy seed nên rủi ro thấp.
- **Ảnh rỗng**: cả 179 chùa có `image_url`/`thumbnail_url` = `''` trong DB, API trả `null`.
- **OKF chưa có temple**: `docs/knowledge/features/` chưa có `temple.md`. ERD đã có bảng `temples`.
