# Design — temple

Version: 1.6
Last updated: 2026-10-10
Status: 3 route đọc đã implement (2026-09-30), chạy thật qua binary + Postgres dev; v1.5 thêm `GET /v1/temples/options` với `?q=` không dấu, cột `temples.province` (migration `0024`) và `location.province` ở summary/detail — trạng thái implement ghi ở `tasks.md` Phase 7–8. Thiếu: E2E, OKF concept; seed chỉ chạy bằng CLI với config dev.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Bốn route `GET /v1/temples/*` (`top`, danh sách phân trang, `options`, chi tiết) đều công khai, không đi qua middleware xác thực. Feature chạm đủ bốn layer Clean Architecture của `services/appService` (`domain` → `usecase` → `repository` / `delivery/http`, nối dây ở `cmd/main.go`) và sở hữu ba migration `0005_temples.sql` (bảng + index), `0011_temple_content.sql` (ba cột nội dung biên tập) `0024_temple_province.sql` (cột `province`, v1.5) và `0025_temple_province_34_units.sql` (đổi tên tỉnh cũ sang 34 đơn vị mới, v1.6). Dữ liệu vào bảng bằng lệnh `cmd/seedtemples` (upsert theo `id`), **không** có seed tự động khi service khởi động và file JSON **không** được nhúng vào binary. `options` (v1.4, thêm `?q=` ở v1.5) là danh sách **rút gọn, có phân trang, tìm được theo tên** (`id`, `name`, `province`) dành cho màn đăng ký chọn nơi quy y/xuất gia; nó dùng chung bảng, thứ tự và repository với ba route kia, chỉ khác tập cột `SELECT`, kích thước trang (`temple.options_page_size`) và DTO. `province` là cột riêng `temples.province` (v1.5, migration `0024`); `?q=` tìm theo `name` không dấu bằng `unaccent` (extension đã bật ở migration `0019`). Cùng hình dạng với `deity` và `article` (top + list phân trang + detail, cấu hình `top_limit`/`page_size`); khác `article` ở cách seed (CLI, upsert ghi đè) và khác `deity` ở chỗ có một danh sách chung thay vì danh sách theo loại.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/temple.go` | `Temple` (mọi cột, gồm `Province` từ v1.5; `""` = chưa có ảnh), `TempleAmenities` (4 bool), `TemplePage`. Không JSON/db tag. |
| Domain / Option | `domain/temple.go` | `TempleOption{ID, Name, Province}` và `TempleOptionPage{Items, Page, PageSize, Total, TotalPages}` — chiếu rút gọn cho `options`; không JSON/db tag. *(v1.3; `Province` + page ở v1.4)* |
| Domain / Province | `domain/province.go` | Danh sách 34 đơn vị cấp tỉnh (Nghị quyết 202/2025/QH15, viết không tiền tố) và `IsProvince(name)` — nguồn duy nhất cho giá trị hợp lệ của `temples.province`; là dữ liệu cố định theo luật, không phải config và không phải `CHECK` ở DB. *(v1.6)* |
| Domain / Ranking | `domain/temple.go` | Bảng `heritagePriority` (8 hạng → 1..8) và `HeritagePriority(rank) (int, bool)`; nguồn duy nhất cho thứ tự hạng, phải khớp `chk_temples_heritage_rank`. |
| Domain / Errors | `domain/temple.go` | `ErrTempleNotFound` và `ErrInvalidPage` — `ErrInvalidPage` **định nghĩa ở file này** nhưng dùng chung với `deity`/`article`; `domain/errors.go` không chứa chúng. |
| Domain / Repository interface | `domain/temple.go` | `TempleRepository`: `ListTop`, `List`, `ListOptions(ctx, query, offset, limit)` *(v1.3; offset/limit + total ở v1.4; `query` ở v1.5)*, `FindByID`, `Upsert`. Thứ tự `heritage_priority, established_year, id` là một phần của hợp đồng. |
| Domain / Mock | `domain/mock_temple_repository.go` | Mock viết tay cho test usecase; thêm `ListOptions` ở v1.3. |
| Usecase | `services/appService/usecase/temple_usecase.go` | `Top`, `List` (tính offset/`total_pages`, chặn `page < 1` hoặc `> MaxInt32`, `items` nil → `[]`), `Options(ctx, query, page)` *(v1.5 — cùng quy tắc `page`/`total_pages` như `List`, kích thước `cfg.OptionsPageSize`; `query` nil = không tìm, rỗng/quá 100 ký tự → `ErrInvalidTempleQuery`; `nil` → `[]`)*, `Get`. Kích thước trang lấy từ `config.TempleConfig`. |
| Repository | `services/appService/repository/postgres_temple_repository.go` | SQL cho bảng `temples` với `templeColumns`/`templeOrder` dùng chung mọi lần đọc; `Upsert` chạy một transaction với `ON CONFLICT (id) DO UPDATE` ghi đè mọi cột; `ListOptions` *(v1.5)*: `SELECT count(*)` rồi `SELECT id, name, province FROM temples` + `WHERE unaccent(lower(name)) LIKE unaccent(lower($1)) ESCAPE '\'` (chỉ khi có `q`; `%`, `_`, `\` được escape) + `templeOrder` + `OFFSET/LIMIT`; `templeColumns` có thêm `province`; `pq.Array` cho `aliases`/`visiting_notes`; `sql.ErrNoRows` → `ErrTempleNotFound`. |
| Seed CLI | `services/appService/cmd/seedtemples/main.go`, `parse.go` | `main`: đọc file (`-file`), nạp config + secret dev, kết nối DB, gọi `Upsert`. `parseTemples`: bỏ chú thích `/* */` (biết chuỗi literal), giải mã, kiểm id/hạng/vùng và `location.province` (bắt buộc, phải thuộc 34 đơn vị qua `domain.IsProvince` — v1.6). |
| Data | `assets/data/temple.json` | 179 chùa, JSONC. Đọc từ đĩa lúc chạy seed (đường dẫn tính từ thư mục hiện tại) — không `go:embed`. |
| Delivery / Handler | `services/appService/delivery/http/temple_handler.go` | `Top`, `List`, `Options` *(v1.3; phân trang từ v1.4, dùng chung bước parse `page` với `List`)*, `Get`; parse `page`; interface `templeUsecase` khai báo phía consumer. |
| Delivery / DTO | `delivery/http/temple_handler.go` | `templeSummaryResponse`, `templeDetailResponse` (nhúng summary), `locationResponse`, `amenitiesResponse`, `templeTopResponse`, `templeListResponse`, `templeOptionResponse{id, name, province}` và `templeOptionsResponse{items, page, page_size, total, total_pages}` *(v1.3; `province` và phân trang từ v1.4)*; `locationResponse` có thêm `province` *(v1.5)* — JSON tag chỉ nằm ở đây. `nullable("")` → `nil` cho hai URL ảnh; `aliases`/`visiting_notes` nil → `[]`. |
| Delivery / Error map | `delivery/http/errors.go` | `ErrTempleNotFound` → `404 TEMPLE_NOT_FOUND`; `ErrInvalidPage` → `400 INVALID_PAGE` (`details.field = "page"`); `ErrInvalidTempleQuery` → `400 INVALID_FILTER` (`details.field = "q"`, cùng mã với `GET /v1/sutras`) *(v1.5)*; còn lại → `500 INTERNAL_ERROR`. |
| Delivery / Fallback | `delivery/http/fallback.go` | `WithAPIFallback` đổi 404/405 của `ServeMux` thành envelope `NOT_FOUND` / `METHOD_NOT_ALLOWED`, giữ header `Allow`. Dùng chung mọi route, không riêng temple. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `TempleConfig{TopLimit, PageSize, OptionsPageSize}` *(`OptionsPageSize` mới ở v1.4)*, block `temple:` (dev: 10, 20 và 100); `TempleConfig.Validate` *(v1.4)* từ chối giá trị `< 1` ở cả ba khoá, gọi ở `cmd/main.go` lúc khởi động. |
| Wiring | `services/appService/cmd/main.go` | Dựng repo → usecase → handler; đăng ký 3 route; v1.3 thêm `mux.HandleFunc("GET /v1/temples/options", templeHandler.Options)` . Không gọi seed. Không dính khối `⚠️ DEV ONLY` nào của `main.go` (riêng `cmd/seedtemples` có khối DEV ONLY của nó). |
| Migration | `migrations/0005_temples.sql`, `0011_temple_content.sql`, `0024_temple_province.sql` *(v1.5: `ADD COLUMN IF NOT EXISTS province VARCHAR(100) NOT NULL DEFAULT ''` + `UPDATE … WHERE province = ''` backfill từ `address`)*, `0025_temple_province_34_units.sql` *(v1.6: `UPDATE` đổi tên tỉnh đã sáp nhập sang đơn vị mới theo bảng 63 → 34, idempotent, không đụng `address`)* | Bảng `temples` (ba `CHECK`: `type`, `region`, `heritage_rank`), index `idx_temples_ranking (heritage_priority, established_year, id)`; `ALTER TABLE` thêm `introduction`, `architecture_heritage`, `visiting_notes`. |
| API spec | `docs/specs/api/openapi.yaml` | **Đã có** 3 route (tag Temples) và schema; v1.3 thêm `/v1/temples/options` + `TempleOption`/`TempleOptionsEnvelope` (v1.4: `province`, `page`/`page_size`/`total`/`total_pages`); `web/docs/public/openapi.yaml` là bản đồng bộ (cùng nội dung); v1.5: `q` ở `options`, `location.province` ở `TempleSummary`. |
| ERD | `docs/specs/db/erd.mmd`, `erd.svg` | **Đã có** bảng `temples` (không FK tới bảng nào). |

Ghi chú về route: `mux.HandleFunc("GET /v1/temples/top", …)` và `GET /v1/temples/options` là segment literal nên `ServeMux` ưu tiên chúng hơn `GET /v1/temples/{id}`.

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

> Xem `flow.md` cho sơ đồ sequence Mermaid của ba flow có nhiều bước bên dưới (mirror 1:1).

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

## Step-by-step Flow — Danh sách rút gọn phân trang, tìm theo tên (`GET /v1/temples/options?page=N&q=`) *(v1.5)*

Cùng khung với danh sách phân trang ở trên (đếm rồi lấy trang), khác ở cột đọc, kích thước trang, bước kiểm `q` và điều kiện tìm không dấu, nên có Step-by-step riêng.

```
Step 1 — Đọc và kiểm tra `page` (handler)
- Trigger: request đến route công khai (không webAPIKey, không JWT)
- Pre-condition: —
- Call: cùng bước parse với `List` — r.URL.Query().Get("page"); rỗng → page = 1, ngược lại strconv.Atoi; đồng thời đọc `q` (có mặt hay không, kể cả rỗng)
    - Success → Step 2
    - Failure (page không phải số nguyên, kể cả tràn int) → writeDomainError(ErrInvalidPage) → 400 INVALID_PAGE, details.field = "page"

Step 2 — Kiểm tra `page` và `q` (usecase)
- Trigger: handler gọi TempleUsecase.Options(ctx, query, page)
- Pre-condition: page là số nguyên
- Call: page < 1 hoặc page > math.MaxInt32; nếu `q` có mặt: trim, rỗng hoặc > 100 ký tự Unicode
    - Success → Step 3 (offset = (page-1) * options_page_size; `q` đã trim hoặc không có)
    - Failure (page) → domain.ErrInvalidPage → 400 INVALID_PAGE
    - Failure (q) → domain.ErrInvalidTempleQuery → 400 INVALID_FILTER, details.field = "q"

Step 3 — Đếm số chùa khớp (repository)
- Trigger: usecase gọi TempleRepository.ListOptions(query, offset, limit)
- Pre-condition: offset/limit hợp lệ
- Call: SELECT count(*) FROM temples [WHERE unaccent(lower(name)) LIKE unaccent(lower($1)) ESCAPE '\']; `$1` = `%term%` đã escape `\`, `%`, `_`
    - Success → total (số khớp), sang Step 4
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR

Step 4 — Lấy trang rút gọn (repository)
- Trigger: Step 3 xong
- Pre-condition: có total
- Call: SELECT id, name, province FROM temples [cùng WHERE] ORDER BY heritage_priority ASC, established_year ASC, id ASC OFFSET $n LIMIT $m
    - Success → items (rỗng nếu offset vượt cuối hoặc không khớp) và total, sang Step 5
    - Failure (lỗi DB / scan) → error bọc %w → 500 INTERNAL_ERROR
    - Transaction boundary: không có — Step 3 và Step 4 là hai câu lệnh độc lập, không chung snapshot

Step 5 — Dựng trang và trả response (usecase + handler)
- Trigger: repository trả về
- Pre-condition: items, total
- Call: items nil → []; total_pages = (total + options_page_size - 1) / options_page_size; handler map sang DTO option (id, name, province)
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
- Call: parseTemples → stripBlockComments (bỏ `/* */`, bỏ qua chuỗi literal) → json.Unmarshal → với mỗi chùa: id không trùng, regionOf(id) hợp lệ, domain.HeritagePriority(rank) biết, `location.province` có và thuộc 34 đơn vị (v1.6)
    - Success → danh sách domain.Temple (Region từ id, HeritagePriority từ hạng), sang Step 3
    - Failure (chú thích không đóng, JSON hỏng, không có chùa, id trùng, id sai dạng hoặc vùng lạ, hạng lạ, thiếu/rỗng hoặc ngoài danh sách `province`) → log.Fatalf "parse <file>", thoát; chưa chạm DB

Step 3 — Nạp config và kết nối DB (seedtemples/main.go)
- Trigger: Step 2 xong
- Pre-condition: có danh sách chùa hợp lệ; `CONFIG_PATH` (mặc định `services/appService/config/config.dev.yaml`)
- Call: pkgconfig.LoadSource/LoadYAML → appConfig.NewMockSecretGetterFromYAML (⚠️ DEV ONLY) → appConfig.Load → database.New + db.Connect (timeout 30 giây)
    - Success → Step 4
    - Failure (thiếu config/secret, không kết nối được DB) → log.Fatalf, thoát; chưa ghi gì

Step 4 — Upsert mọi chùa (repository)
- Trigger: Step 3 xong
- Pre-condition: DB kết nối được; migration 0005 và 0011 đã apply
- Call: TempleRepository.Upsert → INSERT INTO temples (25 cột) … ON CONFLICT (id) DO UPDATE SET <mọi cột> , updated_at = now() cho từng chùa
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
- **`province` và `address` có thể lệch nhau**: `address` vẫn là chuỗi đầy đủ có chứa tên tỉnh, còn `province` là cột riêng do người nạp dữ liệu điền; không có kiểm tra chéo giữa hai cột (cố ý — nếu tên tỉnh đổi, `province` cập nhật mà không bắt buộc sửa `address`). Migration `0024` backfill `province` từ `address` đúng một lần cho dòng đã có.
- **`address` chưa theo đơn vị hành chính mới**: `province` đã theo 34 đơn vị (v1.6) nhưng `address` vẫn là chuỗi cũ nêu tỉnh và huyện cũ (vd. `Huyện Hoa Lư, Tỉnh Ninh Bình`, `Tỉnh Thừa Thiên Huế`). Cố ý không sửa: đây là chuỗi hiển thị do người biên soạn dữ liệu cung cấp, không có bảng ánh xạ cấp huyện/xã. Hệ quả: với một số chùa `province` (vd. `Huế`, `Đà Nẵng`) khác tên đơn vị nêu trong `address`.
- **Danh sách 34 đơn vị nằm trong code**: đổi khi đơn vị hành chính đổi là một lần sửa `domain/province.go` cộng một migration dữ liệu (như `0025`); `backofficeService` chưa dùng danh sách này (form `create-temple` chưa có `province`) và có `domain` riêng nên sẽ cần bản sao hoặc chỗ dùng chung — việc của spec `temple-suggestion`.
- **Tìm không dấu không có index**: `unaccent(lower(name)) LIKE …` quét tuần tự bảng; ổn với vài trăm chùa. Chỉ khớp `name`, không khớp `aliases`/`province`. `unaccent` cần extension (đã bật ở `0019`); môi trường mới phải áp `0019` trước.
- **`options` có hai câu lệnh không cùng snapshot** (đếm rồi lấy trang), giống danh sách phân trang. Thứ tự giữ theo ranking (chùa nổi bật trước) chứ không theo bảng chữ cái, vì sắp xếp theo `name` phụ thuộc collation của DB với tiếng Việt; client cần thứ tự khác thì tự sắp trong từng trang.
- **Ảnh rỗng**: cả 179 chùa có `image_url`/`thumbnail_url` = `''` trong DB, API trả `null`.
- **OKF chưa có temple**: `docs/knowledge/features/` chưa có `temple.md`. ERD đã có bảng `temples`.
