# Design — temple

Version: 2.1
Last updated: 2026-10-15
Status: **v2.0 (2026-10-15): thiết kế mới cho product spec `temple` v1.1, chưa có code** — các hàng/bước đánh dấu *(v2.0)*: trang 12, `q` (tên + `aliases`) và `region` ở danh sách, bốn nhóm trường mới ở chi tiết (migration `0037`), seed kiểm định dạng giờ/liên hệ, `temple.json` vào git. Phần còn lại (đến v1.7) đã implement và kiểm chứng.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Bốn route `GET /v1/temples/*` (`top`, danh sách phân trang, `options`, chi tiết) đều công khai, không đi qua middleware xác thực. Feature chạm đủ bốn layer Clean Architecture của `services/appService` (`domain` → `usecase` → `repository` / `delivery/http`, nối dây ở `cmd/main.go`) và sở hữu ba migration `0005_temples.sql` (bảng + index), `0011_temple_content.sql` (ba cột nội dung biên tập) `0024_temple_province.sql` (cột `province`, v1.5) và `0025_temple_province_34_units.sql` (đổi tên tỉnh cũ sang 34 đơn vị mới, v1.6). Dữ liệu vào bảng bằng lệnh `cmd/seedtemples` (upsert theo `id`), **không** có seed tự động khi service khởi động và file JSON **không** được nhúng vào binary. `options` (v1.4, thêm `?q=` ở v1.5) là danh sách **rút gọn, có phân trang, tìm được theo tên** (`id`, `name`, `province`) dành cho màn đăng ký chọn nơi quy y/xuất gia; nó dùng chung bảng, thứ tự và repository với ba route kia, chỉ khác tập cột `SELECT`, kích thước trang (`temple.options_page_size`) và DTO. `province` là cột riêng `temples.province` (v1.5, migration `0024`); `?q=` tìm theo `name` không dấu bằng `unaccent` (extension đã bật ở migration `0019`). Từ v1.7 địa chỉ chùa có cấu trúc: `province_code` + `ward_code` (mã hệ mới của `administrative-units`) + `ward` (tên chụp) + `address` (chỉ phần dưới cấp xã), ghép thành `full_address` ở tầng domain khi đọc; seed kiểm mã bằng `WardResolver` đã có của `temple-suggestion` (một bước đọc DB thêm giữa kết nối và upsert). Cùng hình dạng với `deity` và `article` (top + list phân trang + detail, cấu hình `top_limit`/`page_size`); khác `article` ở cách seed (CLI, upsert ghi đè) và khác `deity` ở chỗ có một danh sách chung thay vì danh sách theo loại.

## Layers & components

> v2.0: product spec `docs/product/temple/product.md` @ v1.1. Hàng *(v2.0)* là thiết kế mới, chưa có code.

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/temple.go` | `Temple` (mọi cột, gồm `Province` từ v1.5; `""` = chưa có ảnh), `TempleAmenities` (4 bool), `TemplePage`. Không JSON/db tag. |
| Domain / Address *(v1.7)* | `domain/temple.go` | `Temple` thêm `ProvinceCode`, `WardCode`, `Ward` (`""` = chưa nạp lại, cùng quy ước với ảnh) và `func (t Temple) FullAddress() string` (AC 33: ghép `Address`, `Ward`, `Province` không rỗng bằng `", "` khi có `Ward`, ngược lại trả `Address`) — hàm thuần, có test bảng. `Province`/`Ward` do seed lấy từ dataset (AC 35), không nhận từ file. |
| Domain / Detail fields *(v2.0)* | `domain/temple.go` | `Temple` thêm `HolidayOperatingHours`, `SanghaBody`, `Contacts TempleContacts{Hotline, Fanpage, Website, Email}` (chuỗi, `""` = chưa có) và `ContentUpdatedMonth time.Time` (zero = chưa có; chỉ năm + tháng có nghĩa). Hàm thuần `ValidOperatingHours(s string) bool` — `""` hợp lệ, ngược lại đúng `HH:MM - HH:MM`, 00:00–23:59, giờ mở < giờ đóng (AC 46); test bảng. |
| Domain / Filter *(v2.0)* | `domain/temple.go` | `TempleFilter{Query string; HasQuery bool; Region string}` (`Region` `""` = mọi miền) và `ErrInvalidTempleRegion`; `IsTempleRegion(r)` nhận đúng `north`/`central`/`south` (cùng tập với `CHECK` `region` của `0005`). |
| Domain / Option | `domain/temple.go` | `TempleOption{ID, Name, Province, ProvinceCode}` *(v1.7: `ProvinceCode`)* và `TempleOptionPage{Items, Page, PageSize, Total, TotalPages}` — chiếu rút gọn cho `options`; không JSON/db tag. *(v1.3; `Province` + page ở v1.4)* |
| Domain / Province | `domain/province.go` | Danh sách 34 đơn vị cấp tỉnh (Nghị quyết 202/2025/QH15, viết không tiền tố) và `IsProvince(name)` — nguồn duy nhất cho giá trị hợp lệ của `temples.province`; là dữ liệu cố định theo luật, không phải config và không phải `CHECK` ở DB. *(v1.6; **v1.7: đã gỡ** — giá trị hợp lệ của `province` giờ do dataset `addr_new_provinces` quyết định qua mã (AC 35); `domain/province.go` và test của nó bị xoá 2026-10-11 sau khi `grep` xác nhận không còn nơi dùng)* |
| Domain / Ranking | `domain/temple.go` | Bảng `heritagePriority` (8 hạng → 1..8) và `HeritagePriority(rank) (int, bool)`; nguồn duy nhất cho thứ tự hạng, phải khớp `chk_temples_heritage_rank`. |
| Domain / Errors | `domain/temple.go` | `ErrTempleNotFound` và `ErrInvalidPage` — `ErrInvalidPage` **định nghĩa ở file này** nhưng dùng chung với `deity`/`article`; `domain/errors.go` không chứa chúng. |
| Domain / Repository interface | `domain/temple.go` | `TempleRepository`: `ListTop`, `List` *(v2.0: `List(ctx, filter TempleFilter, offset, limit)`)*, `ListOptions(ctx, query, offset, limit)` *(v1.3; offset/limit + total ở v1.4; `query` ở v1.5)*, `FindByID`, `Upsert`. Thứ tự `heritage_priority, established_year, id` là một phần của hợp đồng. |
| Domain / Mock | `domain/mock_temple_repository.go` | Mock viết tay cho test usecase; thêm `ListOptions` ở v1.3; v2.0 đổi chữ ký `List`. |
| Usecase | `services/appService/usecase/temple_usecase.go` | `Top`, `List` (tính offset/`total_pages`, chặn `page < 1` hoặc `> MaxInt32`, `items` nil → `[]`), `Options(ctx, query, page)` *(v1.5 — cùng quy tắc `page`/`total_pages` như `List`, kích thước `cfg.OptionsPageSize`; `query` nil = không tìm, rỗng/quá 100 ký tự → `ErrInvalidTempleQuery`; `nil` → `[]`)*, `Get`. Kích thước trang lấy từ `config.TempleConfig`. *(v2.0)* `List(ctx, filter, page)`: kiểm `region` trước (`ErrInvalidTempleRegion`), rồi `q` theo đúng quy tắc của `Options` (dùng chung hàm chuẩn hoá `q`), rồi `page`. |
| Repository | `services/appService/repository/postgres_temple_repository.go` | SQL cho bảng `temples` với `templeColumns`/`templeOrder` dùng chung mọi lần đọc; `Upsert` chạy một transaction với `ON CONFLICT (id) DO UPDATE` ghi đè mọi cột; v1.7: `templeColumns`/`scanTemple`/`Upsert` thêm `province_code`, `ward_code`, `ward` (NULL ↔ `""`), `ListOptions` đọc thêm `province_code`; `ListOptions` *(v1.5)*: `SELECT count(*)` rồi `SELECT id, name, province FROM temples` + `WHERE unaccent(lower(name)) LIKE unaccent(lower($1)) ESCAPE '\'` (chỉ khi có `q`; `%`, `_`, `\` được escape) + `templeOrder` + `OFFSET/LIMIT`; `templeColumns` có thêm `province`; `pq.Array` cho `aliases`/`visiting_notes`; `sql.ErrNoRows` → `ErrTempleNotFound`. *(v2.0)* Một hàm `templeFilterWhere(filter) (string, []any)` dựng `WHERE` dùng chung cho câu đếm và câu lấy trang của `List`: `region = $n` khi có miền; khi có `q`: `(unaccent(lower(name)) LIKE unaccent(lower($m)) ESCAPE '\' OR EXISTS (SELECT 1 FROM unnest(aliases) a WHERE unaccent(lower(a)) LIKE unaccent(lower($m)) ESCAPE '\'))` với cùng hàm escape của `ListOptions`. `templeColumns`/`scanTemple`/`Upsert` thêm 7 cột mới (`content_updated_month` NULL ↔ zero time). |
| Seed CLI | `services/appService/cmd/seedtemples/main.go`, `parse.go` | `main`: đọc file (`-file`), nạp config + secret dev, kết nối DB, gọi `Upsert`. `parseTemples`: bỏ chú thích `/* */` (biết chuỗi literal), giải mã, kiểm id/hạng/vùng và `location.province` (bắt buộc, phải thuộc 34 đơn vị qua `domain.IsProvince` — v1.6). **v1.7:** `parseTemples` kiểm `location.province_code`, `location.ward_code` không rỗng và `location.address` có mặt (được rỗng, AC 34) thay cho kiểm `province`; `main` thêm bước đối chiếu mã qua `domain.WardResolver` (`ResolveCurrentWard`, cài bằng `AdministrativeUnitRepository`) sau khi kết nối DB và trước `Upsert` (Step 4), gán `Province`/`Ward` từ dataset, từ chối cả file khi mã sai/lệch tỉnh/`location.province` lệch tên, hoặc dataset rỗng (AC 35–36). *(v2.0)* `parseTemples` đọc thêm các khoá tuỳ chọn `holiday_operating_hours`, `sangha_body`, `contacts{hotline, fanpage, website, email}`, `content_updated_month` và kiểm theo AC 47 (`domain.ValidOperatingHours` cho cả hai giờ; tháng parse bằng `time.Parse("2006-01")`; email một `@`; website/fanpage tiền tố `https://`). |
| Data | `assets/data/temple.json` | *(v2.0)* đưa vào git bằng một dòng `!assets/data/temple.json` trong `.gitignore` (AC 50); bổ sung dần các khoá mới. 93 chùa, JSONC. Đọc từ đĩa lúc chạy seed (đường dẫn tính từ thư mục hiện tại) — không `go:embed`. *(v1.7: `location` đổi thành `{lat, lng, address, province_code, ward_code}`; `address` chỉ còn phần dưới cấp xã; bỏ `location.province` hoặc để đúng tên ngắn của mã. Dữ liệu mới do script đề xuất rồi người duyệt — `tasks.md` Phase 10, không do spec này sinh.)* |
| Delivery / Handler | `services/appService/delivery/http/temple_handler.go` | `Top`, `List`, `Options` *(v1.3; phân trang từ v1.4, dùng chung bước parse `page` với `List`)*, `Get`; parse `page`; interface `templeUsecase` khai báo phía consumer. *(v2.0)* `List` đọc thêm `q` (có mặt hay không, kể cả rỗng) và `region` (có mặt hay không) bằng `r.URL.Query()` rồi dựng `TempleFilter`; `Top`/`Options` không đọc `region` (AC 44). |
| Delivery / DTO | `delivery/http/temple_handler.go` | `templeSummaryResponse`, `templeDetailResponse` (nhúng summary), `locationResponse`, `amenitiesResponse`, `templeTopResponse`, `templeListResponse`, `templeOptionResponse{id, name, province}` và `templeOptionsResponse{items, page, page_size, total, total_pages}` *(v1.3; `province` và phân trang từ v1.4)*; `locationResponse` có thêm `province` *(v1.5)*, rồi `province_code`, `ward_code`, `ward` (`nullable("")` → `null`) và `full_address` (`Temple.FullAddress()`) *(v1.7)*; `templeOptionResponse` thêm `province_code` *(v1.7)* — JSON tag chỉ nằm ở đây. `nullable("")` → `nil` cho hai URL ảnh; `aliases`/`visiting_notes` nil → `[]`. *(v2.0)* `templeDetailResponse` thêm `holiday_operating_hours`, `sangha_body`, `contacts` (`contactsResponse{hotline, fanpage, website, email}`) và `content_updated_month` (`*string` `"2006-01"`, zero → `null`); summary và option **không** đổi (AC 45). |
| Delivery / Error map | `delivery/http/errors.go` | `ErrTempleNotFound` → `404 TEMPLE_NOT_FOUND`; `ErrInvalidPage` → `400 INVALID_PAGE` (`details.field = "page"`); `ErrInvalidTempleQuery` → `400 INVALID_FILTER` (`details.field = "q"`, cùng mã với `GET /v1/sutras`) *(v1.5)*; *(v2.0)* `ErrInvalidTempleRegion` → `400 INVALID_FILTER` (`details.field = "region"`); `ErrInvalidTempleQuery` của `List` dùng lại ánh xạ `q`; còn lại → `500 INTERNAL_ERROR`. |
| Delivery / Fallback | `delivery/http/fallback.go` | `WithAPIFallback` đổi 404/405 của `ServeMux` thành envelope `NOT_FOUND` / `METHOD_NOT_ALLOWED`, giữ header `Allow`. Dùng chung mọi route, không riêng temple. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `TempleConfig{TopLimit, PageSize, OptionsPageSize}` *(`OptionsPageSize` mới ở v1.4)*, block `temple:` (dev: 10, 20 và 100; *v2.0: `page_size: 12`* ở mọi `config.<env>.yaml` có block này); `TempleConfig.Validate` *(v1.4)* từ chối giá trị `< 1` ở cả ba khoá, gọi ở `cmd/main.go` lúc khởi động. |
| Wiring | `services/appService/cmd/main.go` | Dựng repo → usecase → handler; đăng ký 3 route; v1.3 thêm `mux.HandleFunc("GET /v1/temples/options", templeHandler.Options)` . Không gọi seed. Không dính khối `⚠️ DEV ONLY` nào của `main.go` (riêng `cmd/seedtemples` có khối DEV ONLY của nó). |
| Migration *(v2.0)* | `migrations/0037_temple_contact_hours.sql` *(chưa tạo)* | `ADD COLUMN IF NOT EXISTS` `holiday_operating_hours VARCHAR(64)`, `sangha_body VARCHAR(255)`, `hotline VARCHAR(32)`, `fanpage VARCHAR(512)`, `website VARCHAR(512)`, `email VARCHAR(254)` — tất cả `NOT NULL DEFAULT ''` — và `content_updated_month DATE` (nullable, ngày 1 của tháng); không đổi cột cũ, không backfill, idempotent (AC 48). Không thêm index: tìm/lọc quét tuần tự (Known gaps). |
| Migration *(v1.7)* | `migrations/0032_temple_address_codes.sql` *(chưa tạo)* | `ADD COLUMN IF NOT EXISTS province_code VARCHAR(16)`, `ward_code VARCHAR(16)`, `ward VARCHAR(255)` (nullable) + CHECK `chk_temples_address_codes` (cùng NULL hoặc cùng có — như `chk_temple_suggestions_address_codes` ở `0030`); không FK, không backfill, không đụng `address`/`province`; idempotent. Phụ thuộc: `0026` (bảng `addr_*`, để seed tra mã) và `0030` (tiền lệ). |
| Migration | `migrations/0005_temples.sql`, `0011_temple_content.sql`, `0024_temple_province.sql` *(v1.5: `ADD COLUMN IF NOT EXISTS province VARCHAR(100) NOT NULL DEFAULT ''` + `UPDATE … WHERE province = ''` backfill từ `address`)*, `0025_temple_province_34_units.sql` *(v1.6: `UPDATE` đổi tên tỉnh đã sáp nhập sang đơn vị mới theo bảng 63 → 34, idempotent, không đụng `address`)* | Bảng `temples` (ba `CHECK`: `type`, `region`, `heritage_rank`), index `idx_temples_ranking (heritage_priority, established_year, id)`; `ALTER TABLE` thêm `introduction`, `architecture_heritage`, `visiting_notes`. |
| API spec | `docs/specs/api/openapi.yaml` | **Đã có** 3 route (tag Temples) và schema; v1.3 thêm `/v1/temples/options` + `TempleOption`/`TempleOptionsEnvelope` (v1.4: `province`, `page`/`page_size`/`total`/`total_pages`); `web/docs/public/openapi.yaml` là bản đồng bộ (cùng nội dung); v1.5: `q` ở `options`, `location.province` ở `TempleSummary`. *(v2.0, chưa làm)*: `q`, `region` + lỗi `INVALID_FILTER` ở `GET /v1/temples`, `page_size` 12 trong ví dụ, 4 nhóm trường ở `TempleDetail`; đồng bộ `web/docs/public/openapi.yaml` và trang `web/docs/.../catalogues/temples.md`. |
| ERD | `docs/specs/db/erd.mmd`, `erd.svg` | **Đã có** bảng `temples` (không FK tới bảng nào); *(v2.0)* thêm 7 cột vào `erd.mmd`. |

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

## Step-by-step Flow — Danh sách phân trang, tìm và lọc miền (`GET /v1/temples?page=N&q=&region=`) *(v2.0: `q`, `region`, trang 12)*

Hai lần truy vấn nối tiếp (đếm, rồi lấy trang) cùng một điều kiện lọc nên flow này có Step-by-step riêng.

```
Step 1 — Đọc tham số (handler)
- Trigger: request đến route công khai
- Pre-condition: —
- Call: r.URL.Query(): `page` (rỗng → 1, ngược lại strconv.Atoi); `q` và `region` — ghi nhận có mặt hay không, kể cả rỗng (v2.0)
    - Success → Step 2 với TempleFilter{Query, HasQuery, Region (chưa kiểm)}
    - Failure (page không phải số nguyên, kể cả tràn int) → writeDomainError(ErrInvalidPage) → 400 INVALID_PAGE, details.field = "page"

Step 2 — Kiểm `region`, `q`, khoảng của `page` (usecase)
- Trigger: handler gọi TempleUsecase.List(ctx, filter, page)
- Pre-condition: page là số nguyên
- Call: theo thứ tự: region có mặt mà không thuộc north/central/south (kể cả rỗng) → lỗi; q có mặt: trim, rỗng hoặc > 100 ký tự → lỗi; page < 1 hoặc > math.MaxInt32 → lỗi
    - Success → Step 3 (offset = (page-1) * page_size, page_size = 12)
    - Failure (region) → domain.ErrInvalidTempleRegion → 400 INVALID_FILTER, details.field = "region"
    - Failure (q) → domain.ErrInvalidTempleQuery → 400 INVALID_FILTER, details.field = "q"
    - Failure (page) → domain.ErrInvalidPage → 400 INVALID_PAGE

Step 3 — Đếm số chùa khớp (repository)
- Trigger: usecase gọi TempleRepository.List(filter, offset, limit)
- Pre-condition: filter đã kiểm; offset/limit hợp lệ
- Call: SELECT count(*) FROM temples [WHERE templeFilterWhere(filter)] — region = $n; q khớp name hoặc một phần tử aliases, không dấu, `%term%` đã escape
    - Success → total (số khớp, hoặc cả bảng khi không lọc), sang Step 4
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR

Step 4 — Lấy trang (repository)
- Trigger: Step 3 xong
- Pre-condition: có total
- Call: SELECT <cột summary> FROM temples [cùng WHERE] ORDER BY heritage_priority ASC, established_year ASC, id ASC OFFSET $a LIMIT $b
    - Success → items (rỗng nếu offset vượt cuối hoặc không khớp) và total, sang Step 5
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
- Call: SELECT id, name, province, province_code (v1.7) FROM temples [cùng WHERE] ORDER BY heritage_priority ASC, established_year ASC, id ASC OFFSET $n LIMIT $m
    - Success → items (rỗng nếu offset vượt cuối hoặc không khớp) và total, sang Step 5
    - Failure (lỗi DB / scan) → error bọc %w → 500 INTERNAL_ERROR
    - Transaction boundary: không có — Step 3 và Step 4 là hai câu lệnh độc lập, không chung snapshot

Step 5 — Dựng trang và trả response (usecase + handler)
- Trigger: repository trả về
- Pre-condition: items, total
- Call: items nil → []; total_pages = (total + options_page_size - 1) / options_page_size; handler map sang DTO option (id, name, province, province_code — `null` khi chưa nạp lại, v1.7)
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
- Call: parseTemples → stripBlockComments (bỏ `/* */`, bỏ qua chuỗi literal) → json.Unmarshal → với mỗi chùa: id không trùng, regionOf(id) hợp lệ, domain.HeritagePriority(rank) biết, `location.province_code`, `location.ward_code` có và không rỗng sau trim, `location.address` có mặt (được phép rỗng — A-004) (v1.7, AC 34; thay cho kiểm `province` thuộc 34 đơn vị của v1.6); v2.0: `operating_hours`/`holiday_operating_hours` rỗng hoặc đúng `HH:MM - HH:MM` mở < đóng, `content_updated_month` là tháng thật `YYYY-MM`, `contacts.email` có đúng một `@`, `contacts.website`/`fanpage` bắt đầu `https://` (AC 47)
    - Success → danh sách domain.Temple (Region từ id, HeritagePriority từ hạng; ProvinceCode/WardCode/Address từ file, Province/Ward còn trống), sang Step 3
    - Failure (chú thích không đóng, JSON hỏng, không có chùa, id trùng, id sai dạng hoặc vùng lạ, hạng lạ, thiếu/rỗng `province_code`/`ward_code`, thiếu field `address`, v2.0: giờ sai dạng, tháng sai, email/URL sai) → log.Fatalf "parse <file>", thoát; chưa chạm DB

Step 3 — Nạp config và kết nối DB (seedtemples/main.go)
- Trigger: Step 2 xong
- Pre-condition: có danh sách chùa hợp lệ; `CONFIG_PATH` (mặc định `services/appService/config/config.dev.yaml`)
- Call: pkgconfig.LoadSource/LoadYAML → appConfig.NewMockSecretGetterFromYAML (⚠️ DEV ONLY) → appConfig.Load → database.New + db.Connect (timeout 30 giây)
    - Success → Step 4
    - Failure (thiếu config/secret, không kết nối được DB) → log.Fatalf, thoát; chưa ghi gì

Step 4 — Đối chiếu mã với metadata hành chính (seedtemples/main.go, đọc) *(v1.7)*
- Trigger: Step 3 xong
- Pre-condition: DB kết nối được; migration 0026 đã apply và dataset đã nạp bằng `cmd/seedadminunits`
- Call: với mỗi chùa, WardResolver.ResolveCurrentWard(ctx, ProvinceCode, WardCode) — cùng một SELECT `addr_new_provinces LEFT JOIN addr_new_wards` mà S-3 của temple-suggestion dùng; gán Province = province.name, Ward = ward.full_name; so `location.province` (nếu file có) với province.name
    - Success → mọi chùa có Province và Ward từ dataset, sang Step 5
    - Failure (ErrAdministrativeUnitNotFound: mã tỉnh lạ; ErrWardNotInProvince: xã lạ hoặc thuộc tỉnh khác; `location.province` lệch tên; bảng `addr_new_provinces` rỗng → thông báo nêu `cmd/seedadminunits`) → log.Fatalf nêu id chùa và field, thoát; chưa ghi gì (AC 35–36)
    - Failure (lỗi DB) → log.Fatalf, thoát; chưa ghi gì
    - Ghi chú: bước này chỉ đọc và không chung transaction với Step 5; dataset chỉ đổi khi vận hành nạp bản mới, cửa sổ lệch giữa Step 4 và Step 5 là chấp nhận được (cùng lập luận S-3/S-4 của temple-suggestion)

Step 5 — Upsert mọi chùa (repository)
- Trigger: Step 4 xong
- Pre-condition: DB kết nối được; migration 0005, 0011, 0032 (v1.7) và 0037 (v2.0) đã apply
- Call: TempleRepository.Upsert → INSERT INTO temples (28 cột ở v1.7; 35 cột ở v2.0) … ON CONFLICT (id) DO UPDATE SET <mọi cột> , updated_at = now() cho từng chùa
    - Success → commit, sang Step 6
    - Failure (lỗi DB: CHECK `type`/`region`/`heritage_rank`/`chk_temples_address_codes`, thiếu bảng hoặc cột, mất kết nối) → rollback toàn bộ → log.Fatalf "seed temples"
    - Transaction boundary: mọi INSERT/UPDATE của Step 5 chung MỘT transaction — hoặc đủ hết hoặc không dòng nào

Step 6 — Ghi log (seedtemples/main.go)
- Trigger: Upsert thành công
- Pre-condition: —
- Call: log.Printf "seeded %d temples from %s"
    - Success → thoát 0
    - Failure → không có nhánh lỗi
```

## Deliberately stubbed for now (if applicable)

Không có gì bị stub ở phía đọc: mọi method của `TempleRepository` đều có SQL thật. Phần thuộc về vận hành nằm ở Known gaps (seed chỉ dùng config dev).

## Known gaps carried by this feature

- **Không có E2E**: `services/appService/e2e/` không có gì về temple. Việc "đã chạy thật" là lần chạy tay trên server dev được ghi ở `api-contract.md` cũ (đã gỡ ở v2.1, xem lịch sử git; dữ liệu lúc đó khác file seed hiện có 93 chùa); lượt viết spec này không gọi lại server.
- **Test repository cần Postgres và xoá dữ liệu dev**: `postgres_temple_repository_test.go` gọi `connectTestDB`, bỏ qua khi `-short`; helper `truncateAll` chạy `TRUNCATE … temples, deities, articles, users …` trên DB `innerlamp` (mặc định `localhost:5432`). Chạy xong bảng `temples` trống, phải seed lại bằng `cmd/seedtemples`; `deities` seed lại bằng `cmd/seeddeities`, `articles` tự nạp ở lần khởi động kế tiếp. `go test -short` chỉ chạy usecase, handler và seed parser. Lượt viết spec này chỉ chạy `-short`, chưa chạy test repository.
- **Seed không tự chạy và dùng config dev**: appService khởi động không đụng bảng `temples`; `cmd/seedtemples` gắn với `config.dev.yaml` và secret dev (`⚠️ DEV ONLY`). v2.0: product đã chốt dùng file JSON ở mọi môi trường (FR-TPL-010); việc cho seed đọc config/secret của môi trường thật là việc kỹ thuật, `tasks.md` Phase 3.
- **Upsert ghi đè mọi cột và không xoá**: chạy lại seed đưa mọi cột về đúng nội dung file (kể cả cột đã sửa tay trong DB), và chùa bị xoá khỏi file **vẫn nằm trong bảng** — không có bước dọn.
- **Validation ở seeder chỉ một phần**: `parseTemples` kiểm id, vùng, hạng; `type` chỉ bị chặn bởi `CHECK` ở DB (một `type` lạ làm rollback cả lần seed), còn `name`, `sect`, toạ độ, năm không bị kiểm — rỗng vẫn qua.
- **`sect` không có `CHECK`**: khác `type`/`heritage_rank`, giá trị mới của `sect` vào được DB mà không cần migration; client phải coi đây là tập mở.
- **Đếm và lấy trang không cùng snapshot**: nếu có dòng đổi giữa Step 3 và Step 4 của flow phân trang, `total` có thể lệch `items`. Dữ liệu chỉ đổi khi chạy seed nên rủi ro thấp.
- **(v1.7) Chùa chưa nạp lại có địa chỉ cũ và mã `NULL`.** Migration `0032` không điền mã (AC 37); `assets/data/temple.json` đã có mã cho cả 93 chùa, nhưng một DB chỉ có địa chỉ theo dạng này sau khi áp `0032` rồi chạy `seedadminunits` → `seedtemples`. Trước đó API trả `province_code`/`ward_code`/`ward` = `null` và `full_address` = `address` cũ. Các gap "`province` và `address` có thể lệch nhau" và "`address` chưa theo đơn vị mới" bên dưới chỉ còn đúng cho những dòng chưa nạp lại.
- **(v1.7) 6 chùa có xã mới chưa chắc chắn.** 5 chùa độ tin cậy trung bình (xã cũ tách cho nhiều xã mới, không có văn bản ranh giới cấp đường) và 1 chưa xác minh (`temple_south_021`): `docs/specs/features/temple/address-decisions.md`, `docs/assumptions/temple/A-005`. Tỉnh của các chùa này không bị ảnh hưởng.
- **(v1.7) `temple.json` không nằm trong git** (`.gitignore` có `*.json`): chỉ `assets/data/temple.json.pre-v1.7.bak` được theo dõi. *(v2.0: sửa bằng AC 50 — Phase 13.)*
- **(v1.7) Seed phụ thuộc dataset hành chính.** `seedtemples` phải chạy sau `seedadminunits` (AC 36); dataset đổi phiên bản thì mã trong `temple.json` có thể không còn khớp và seed sẽ từ chối cả file (chủ ý, không ghi dở).
- **(v1.7) Gán mã từ chuỗi địa chỉ cũ là suy đoán.** Xã trùng tên, một xã cũ tách thành nhiều xã mới, hoặc `address` thiếu cấp xã không có đáp án duy nhất; script chỉ đề xuất, người duyệt quyết (`tasks.md` Phase 10). Dataset hành chính ghi `data_provenance` là "chưa đối chiếu với file chính thức", nên "chính thức" ở đây là theo dataset đó.
- **`province` và `address` có thể lệch nhau**: `address` vẫn là chuỗi đầy đủ có chứa tên tỉnh, còn `province` là cột riêng do người nạp dữ liệu điền; không có kiểm tra chéo giữa hai cột (cố ý — nếu tên tỉnh đổi, `province` cập nhật mà không bắt buộc sửa `address`). Migration `0024` backfill `province` từ `address` đúng một lần cho dòng đã có.
- **`address` chưa theo đơn vị hành chính mới**: `province` đã theo 34 đơn vị (v1.6) nhưng `address` vẫn là chuỗi cũ nêu tỉnh và huyện cũ (vd. `Huyện Hoa Lư, Tỉnh Ninh Bình`, `Tỉnh Thừa Thiên Huế`). Cố ý không sửa: đây là chuỗi hiển thị do người biên soạn dữ liệu cung cấp, không có bảng ánh xạ cấp huyện/xã. Hệ quả: với một số chùa `province` (vd. `Huế`, `Đà Nẵng`) khác tên đơn vị nêu trong `address`.
- **Danh sách 34 đơn vị nằm trong code**: đổi khi đơn vị hành chính đổi là một lần sửa `domain/province.go` cộng một migration dữ liệu (như `0025`); `backofficeService` chưa dùng danh sách này (form `create-temple` chưa có `province`) và có `domain` riêng nên sẽ cần bản sao hoặc chỗ dùng chung — việc của spec `temple-suggestion`.
- **Tìm không dấu không có index**: `unaccent(lower(name)) LIKE …` (và từ v2.0 `EXISTS (… unnest(aliases) …)` ở danh sách) quét tuần tự bảng; ổn với vài trăm chùa. Lên hàng chục nghìn thì thêm cột chuẩn hoá + index trigram. `options` chỉ khớp `name`; danh sách (v2.0) khớp `name` + `aliases`; không route nào khớp `province`. `unaccent` cần extension (đã bật ở `0019`); môi trường mới phải áp `0019` trước.
- **`options` có hai câu lệnh không cùng snapshot** (đếm rồi lấy trang), giống danh sách phân trang. Thứ tự giữ theo ranking (chùa nổi bật trước) chứ không theo bảng chữ cái, vì sắp xếp theo `name` phụ thuộc collation của DB với tiếng Việt; client cần thứ tự khác thì tự sắp trong từng trang.
- **Ảnh**: file seed có `image_url`/`thumbnail_url` cho cả 93 chùa (đường dẫn `/static/images/temples/...`); repo không chứa thư mục `static/images/` nên chưa kiểm được file tồn tại. Chùa không có đường dẫn thì API trả `null`.
- **OKF**: `docs/knowledge/features/temple.md` đã có; sau khi implement v2.0 phải làm mới (`il-okf-writer`).
