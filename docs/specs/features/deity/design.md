# Design — deity

Version: 2.1
Last updated: 2026-10-03
Status: implemented (2026-09-30) — 3 route đọc chạy thật qua binary + Postgres dev; có test usecase/handler/seed parser/repository. Thiếu: E2E, OKF concept, ảnh dùng được; seed chỉ chạy bằng CLI với config dev.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Ba route (`GET /v1/deities/top`, `GET /v1/deities`, `GET /v1/deities/{id}`) đều công khai, không đi qua middleware xác thực. Feature chạm đủ bốn layer Clean Architecture của `services/appService` (`domain` → `usecase` → `repository` / `delivery/http`, nối dây ở `cmd/main.go`) và sở hữu migration `0007_deities.sql`. Dữ liệu vào bảng bằng lệnh `cmd/seeddeities` (upsert theo `id`), **không** có seed tự động khi khởi động và file JSON **không** được nhúng vào binary. Cùng hình dạng với `temple` và `article`, nhưng khác ở một điểm: thứ tự dựa hoàn toàn vào `sort_order` (vị trí trong file) chứ không có cột xếp hạng dẫn xuất.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/deity.go` | `Deity` (mọi cột), `MemorialDay`, `Iconography`, `Mantras`, `DeityPage`, hằng `DeityBuddha`/`DeityBodhisattva`. Không JSON/db tag. |
| Domain / Errors | `domain/deity.go` | `ErrDeityNotFound`; `ErrInvalidPage` dùng chung, **định nghĩa ở `domain/temple.go`**. |
| Domain / Repository interface | `domain/deity.go` | `DeityRepository`: `ListTop`, `List`, `FindByID`, `Upsert`. Thứ tự `sort_order ASC, id ASC` là một phần của hợp đồng. |
| Domain / Mock | `domain/mock_deity_repository.go` | Mock viết tay cho test usecase. |
| Usecase | `services/appService/usecase/deity_usecase.go` | `Top`, `ListDeities` (mọi loại, không lọc; tính offset/`total_pages`, chặn `page < 1` hoặc `> MaxInt32`, `items` nil → `[]`), `Get`. Kích thước trang lấy từ `config.DeityConfig`. |
| Repository | `services/appService/repository/postgres_deity_repository.go` | SQL cho bảng `deities` với `deityColumns`/`deityOrder` dùng chung mọi lần đọc; `Upsert` chạy một transaction với `ON CONFLICT (id) DO UPDATE` ghi đè mọi cột; `lunar_memorial_days` marshal/unmarshal JSONB qua `memorialDayJSON`; `pq.Array` cho `sacred_artifacts`/`great_vows`; `sql.ErrNoRows` → `ErrDeityNotFound`. |
| Seed CLI | `services/appService/cmd/seeddeities/main.go`, `parse.go` | `main`: đọc file (`-file`), nạp config + secret dev, kết nối DB, gọi `Upsert` — cùng khung với `seedtemples`. `parseDeities`: giải mã JSON thuần, kiểm id rỗng/trùng, `type`, `name_vi`; gán `SortOrder` = vị trí trong file. |
| Data | `assets/data/phat_botat.json` | 18 mục (6 Phật, 12 Bồ tát), JSON thuần. Đọc từ đĩa lúc chạy seed (đường dẫn tính từ thư mục hiện tại) — không `go:embed`. |
| Delivery / Handler | `services/appService/delivery/http/deity_handler.go` | `Top`, `ListDeities`, `Get`; parse `page`; interface `deityUsecase` khai báo phía consumer. |
| Delivery / DTO | `delivery/http/deity_handler.go` | `deitySummaryResponse`, `deityDetailResponse` (nhúng summary), `memorialDayResponse`, `iconographyResponse`, `mantrasResponse`, `deityTopResponse`, `deityListResponse` — JSON tag chỉ nằm ở đây. URL ảnh là `string` (không `null`); `sacred_artifacts`/`great_vows`/`lunar_memorial_days` không bao giờ `null`. |
| Delivery / Error map | `delivery/http/errors.go` | `ErrDeityNotFound` → `404 DEITY_NOT_FOUND`; `ErrInvalidPage` → `400 INVALID_PAGE` (`details.field = "page"`); còn lại → `500 INTERNAL_ERROR`. |
| Delivery / Fallback | `delivery/http/fallback.go` | `WithAPIFallback` đổi 404/405 của `ServeMux` thành envelope `NOT_FOUND` / `METHOD_NOT_ALLOWED`. Dùng chung mọi route. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `DeityConfig{TopLimit, PageSize}`, block `deity:` (dev: 10 và 20). |
| Wiring | `services/appService/cmd/main.go` | Dựng repo → usecase → handler; đăng ký `GET /v1/deities/top`, `GET /v1/deities` (phân trang), `GET /v1/deities/{id}`. Không gọi seed. Không dính khối `⚠️ DEV ONLY` nào của `main.go` (riêng `cmd/seeddeities` có khối DEV ONLY của nó). |
| Migration | `migrations/0007_deities.sql` | Bảng `deities` (`CHECK` trên `type`, `lunar_memorial_days JSONB`, hai cột `TEXT[]`), index `idx_deities_type_order (type, sort_order, id)` và `idx_deities_order (sort_order, id)`. |
| API spec | `docs/specs/api/openapi.yaml` | **Đã có** 3 route (tag Deities) và schema; `web/docs/public/openapi.yaml` khớp. |
| ERD | `docs/specs/db/erd.mmd`, `erd.svg` | **Đã có** bảng `deities` (không FK tới bảng nào). |

Ghi chú về route: `mux.HandleFunc("GET /v1/deities/top", …)` là segment literal nên `ServeMux` ưu tiên nó hơn `GET /v1/deities/{id}`. Danh sách phân trang là `GET /v1/deities` (đúng path gốc, không segment thêm) nên không đụng `top` hay `{id}`; không có tham số lọc `type`.

## Request flow

`top` và `detail` chỉ một câu SQL nên giữ dạng ASCII; danh sách phân trang (đếm cả bảng rồi lấy trang) và seed có nhiều bước nên nằm ở Step-by-step Flow và `flow.md`.

```
GET /v1/deities/top
  └─ public
       └─ DeityHandler.Top                             (delivery/http)
            └─ DeityUsecase.Top(ctx)                   (usecase)
                 └─ DeityRepository.ListTop(limit)     (repository)
                      └─ SELECT <cột> FROM deities ORDER BY sort_order, id LIMIT $1   (mọi loại)

GET /v1/deities/{id}
  └─ public
       └─ DeityHandler.Get                             (delivery/http)
            └─ DeityUsecase.Get(ctx, id)               (usecase)
                 └─ DeityRepository.FindByID(id)       (repository)
                      └─ SELECT <cột> FROM deities WHERE id = $1   (Phật hoặc Bồ tát)
                           └─ sql.ErrNoRows → ErrDeityNotFound → 404 DEITY_NOT_FOUND
```

> Xem `flow.md` cho sơ đồ sequence Mermaid của hai flow có nhiều bước bên dưới (mirror 1:1).

## Step-by-step Flow — Danh sách Phật và Bồ tát phân trang (`GET /v1/deities?page=N`)

Hai lần truy vấn nối tiếp (đếm cả bảng, rồi lấy trang) nên flow này có Step-by-step riêng.

```
Step 1 — Đọc và kiểm tra `page` (handler)
- Trigger: request đến route công khai
- Pre-condition: —
- Call: r.URL.Query().Get("page"); rỗng → page = 1, ngược lại strconv.Atoi
    - Success → Step 2
    - Failure (không phải số nguyên, kể cả tràn int) → writeDomainError(ErrInvalidPage) → 400 INVALID_PAGE, details.field = "page"

Step 2 — Kiểm tra khoảng của `page` (usecase)
- Trigger: handler gọi DeityUsecase.ListDeities(ctx, page)
- Pre-condition: page là số nguyên
- Call: DeityUsecase.ListDeities — page < 1 hoặc page > math.MaxInt32
    - Success → Step 3 (không lọc loại, offset = (page-1) * page_size)
    - Failure → domain.ErrInvalidPage → 400 INVALID_PAGE

Step 3 — Đếm số deity (repository)
- Trigger: usecase gọi DeityRepository.List(offset, limit)
- Pre-condition: offset/limit hợp lệ
- Call: SELECT count(*) FROM deities
    - Success → total (cả hai loại), sang Step 4
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR

Step 4 — Lấy trang (repository)
- Trigger: Step 3 xong
- Pre-condition: có total
- Call: SELECT <cột> FROM deities ORDER BY sort_order ASC, id ASC OFFSET $1 LIMIT $2
    - Success → items (rỗng nếu offset vượt cuối) và total, sang Step 5
    - Failure (lỗi DB / scan / `lunar_memorial_days` không giải mã được) → error bọc %w → 500 INTERNAL_ERROR
    - Transaction boundary: không có — Step 3 và Step 4 là hai câu lệnh độc lập, không chung snapshot

Step 5 — Dựng trang và trả response (usecase + handler)
- Trigger: repository trả về
- Pre-condition: items, total
- Call: items nil → []; total_pages = (total + page_size - 1) / page_size; handler map sang DTO summary
    - Success → 200 {items, page, page_size, total, total_pages}
    - Failure → không có nhánh lỗi mới
```

## Step-by-step Flow — Nạp dữ liệu bằng `cmd/seeddeities`

Đọc file, kiểm tra, kết nối và ghi là các bước nối tiếp, trong đó bước ghi là side effect bền vững. Đây là lệnh chạy tay, **không** thuộc quá trình khởi động của appService.

```
Step 1 — Đọc file nguồn (seeddeities/main.go)
- Trigger: operator chạy `go run ./services/appService/cmd/seeddeities [-file <path>]`
- Pre-condition: thư mục hiện tại là gốc repo nếu dùng đường dẫn mặc định `assets/data/phat_botat.json`
- Call: os.ReadFile(*file)
    - Success → Step 2
    - Failure (không có file / không đọc được) → log.Fatalf "read <file>", thoát; chưa chạm DB

Step 2 — Kiểm tra dữ liệu (seeddeities/parse.go)
- Trigger: Step 1 xong
- Pre-condition: nội dung file trong bộ nhớ
- Call: parseDeities → json.Unmarshal (JSON thuần, không bỏ chú thích) → với mỗi mục: id không rỗng và không trùng, type ∈ {BUDDHA, BODHISATTVA}, name_vi không rỗng; SortOrder = chỉ số trong file
    - Success → danh sách domain.Deity, sang Step 3
    - Failure (JSON hỏng, không có mục, id rỗng/trùng, type lạ, name_vi rỗng) → log.Fatalf "parse <file>", thoát; chưa chạm DB

Step 3 — Nạp config và kết nối DB (seeddeities/main.go)
- Trigger: Step 2 xong
- Pre-condition: có danh sách hợp lệ; `CONFIG_PATH` (mặc định `services/appService/config/config.dev.yaml`)
- Call: pkgconfig.LoadSource/LoadYAML → appConfig.NewMockSecretGetterFromYAML (⚠️ DEV ONLY) → appConfig.Load → database.New + db.Connect (timeout 30 giây)
    - Success → Step 4
    - Failure (thiếu config/secret, không kết nối được DB) → log.Fatalf, thoát; chưa ghi gì

Step 4 — Upsert mọi mục (repository)
- Trigger: Step 3 xong
- Pre-condition: DB kết nối được; migration 0007 đã apply
- Call: DeityRepository.Upsert → với mỗi mục marshal lunar_memorial_days sang JSON rồi INSERT INTO deities (23 cột) … ON CONFLICT (id) DO UPDATE SET <mọi cột>, updated_at = now()
    - Success → commit, sang Step 5
    - Failure (lỗi marshal, lỗi DB: CHECK `type`, thiếu bảng, mất kết nối) → rollback toàn bộ → log.Fatalf "seed deities"
    - Transaction boundary: mọi INSERT/UPDATE của Step 4 chung MỘT transaction — hoặc đủ hết hoặc không dòng nào

Step 5 — Ghi log (seeddeities/main.go)
- Trigger: Upsert thành công
- Pre-condition: —
- Call: log.Printf "seeded %d deities from %s"
    - Success → thoát 0
    - Failure → không có nhánh lỗi
```

## Deliberately stubbed for now (if applicable)

Không có gì bị stub ở phía đọc: mọi method của `DeityRepository` đều có SQL thật. Riêng hai URL ảnh là **dữ liệu giữ chỗ** chứ không phải code stub — xem Known gaps.

## Known gaps carried by this feature

- **Không có E2E**: `services/appService/e2e/` không có gì về deity. Việc "đã chạy thật" là chạy tay trên server dev với 18 mục (theo `api-contract.md`); lượt viết spec này không gọi lại server.
- **Test repository cần Postgres và xoá dữ liệu dev**: `postgres_deity_repository_test.go` gọi `connectTestDB`, bỏ qua khi `-short`; helper `truncateAll` chạy `TRUNCATE … temples, deities, articles, users …` trên DB `innerlamp` (mặc định `localhost:5432`). Chạy xong bảng `deities` trống, phải seed lại bằng `cmd/seeddeities`. `go test -short` chỉ chạy usecase, handler và seed parser. Lượt viết spec này chỉ chạy `-short`, chưa chạy test repository.
- **Ảnh không phân giải được**: cả 18 mục có `thumbnail_url = assets/images/buddhas/thumbnails/<id>.webp` (đường dẫn tương đối, không có route nào phục vụ) và `image_url = https://cdn.yourapp.com/images/buddhas/full/<id>.webp` (host giữ chỗ). API trả nguyên xi (AC 13); client hiện không tải được ảnh nào.
- **Ảnh không theo quy ước `null` của `temple`/`article`**: hai URL luôn là chuỗi, một mục không có ảnh sẽ trả `""` thay vì `null`. Client cần biết khác biệt này giữa các danh mục.
- **Không có route danh sách Phật**: Phật chỉ tới được qua `top` (chỉ khi `deity.top_limit` đủ lớn để chứa 6 vị đầu file) hoặc `detail` khi đã biết `id`.
- **Seed không tự chạy và dùng config dev**: appService khởi động không đụng bảng `deities`; `cmd/seeddeities` gắn với `config.dev.yaml` và secret dev (`⚠️ DEV ONLY`) nên chưa dùng được cho staging/production (`⚠️ Not confirmed` trong `requirements.md`).
- **Upsert ghi đè mọi cột và không xoá**: chạy lại seed đưa mọi cột (kể cả `sort_order`) về đúng nội dung file, và mục bị xoá khỏi file **vẫn nằm trong bảng** — không có bước dọn.
- **Validation ở seeder chỉ một phần**: `parseDeities` kiểm id, `type`, `name_vi`; các trường còn lại (ngày âm, mantra…) không bị kiểm, ví dụ `lunar_day = 45` vẫn qua.
- **Một dòng có JSONB hỏng làm hỏng cả trang**: `scanDeity` trả lỗi khi `lunar_memorial_days` không giải mã được → `500` cho mọi request đọc dòng đó, kể cả `top` và danh sách. Chỉ xảy ra nếu ai đó sửa cột bằng SQL tay.
- **Đếm và lấy trang không cùng snapshot**: nếu có dòng đổi giữa Step 3 và Step 4 của flow phân trang, `total` có thể lệch `items`. Dữ liệu chỉ đổi khi chạy seed nên rủi ro thấp.
- **OKF chưa có deity**: `docs/knowledge/features/` chưa có `deity.md`. ERD đã có bảng `deities`.
