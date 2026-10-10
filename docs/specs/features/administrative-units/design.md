# Design — administrative-units

Version: 1.4
Last updated: 2026-10-10
Status: Phase 1 (domain), 2 (usecase) và 3 (repository + migration `0026`) đã implement và test trên Postgres thật; delivery, config rate limit, CLI chưa làm. File dữ liệu `assets/data/administrative-units.json` đã tạo và qua `Dataset.Verify`. Thiết kế theo `requirements.md` v1.2.
Module: `administrative-units`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Chín route `GET /v1/administrative-units/*` (danh sách theo hai hệ và chuyển đổi cấp xã/tỉnh) đều công khai — không đi qua JWT — nhưng **bọc `webAPIKey.WrapFunc`** như `POST /v1/temple-suggestions` và `/v1/auth/*` (header `X-API-Key` đúng hoặc vắng thì phục vụ, sai thì `401 INVALID_API_KEY`), rồi qua **giới hạn tần suất theo IP**, và trả `Cache-Control: public, max-age=86400`. Feature chạm đủ bốn layer Clean Architecture của `services/appService` (`domain` → `usecase` → `repository` / `delivery/http`, nối dây ở `cmd/main.go`), sở hữu một migration (dự kiến `migrations/0026_administrative_units.sql`), một block config và một lệnh seed `cmd/seedadminunits`. Dữ liệu là **tĩnh giữa hai lần nạp**: mọi route chỉ đọc DB, không gọi dịch vụ ngoài. Nguồn nạp là **một file JSON duy nhất**, `assets/data/administrative-units.json`, nên seed lại DB bất cứ lúc nào chỉ cần chạy lại lệnh. Hình dạng gần nhất với `temple`/`deity` (danh mục đọc công khai, nạp bằng CLI upsert), khác ở chỗ có hai cây dữ liệu riêng và một bảng đối chiếu N:M, và route chuyển đổi không phân trang.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/administrative_unit.go` *(created)* | `CurrentProvince`, `CurrentWard`, `LegacyProvince`, `LegacyDistrict`, `LegacyWard` (mã + `Name` + `FullName` + `Type` + khoá cha + `SortOrder`), `WardMapping`, `DistrictMapping`, `CurrentWardCandidate`/`LegacyWardCandidate` (xã đích + tỉnh/huyện cha + `MappingType` + `Note`), `Conversion[T]` với `Unique()` suy ra từ số ứng viên. Không JSON/db tag. Tên `Current*`/`Legacy*` (không `New*`/`Old*`) vì `NewX` đọc như constructor trong Go. Tách khỏi `domain/province.go` hiện có (danh sách 34 tên cho `temples.province`). |
| Domain / Errors | `domain/administrative_unit.go` *(created)* | `ErrAdministrativeUnitNotFound` — handler map `404 ADMINISTRATIVE_UNIT_NOT_FOUND`. Đặt cùng file như `ErrTempleNotFound` nằm ở `domain/temple.go`. |
| Domain / Dataset | `domain/administrative_unit.go` *(created)* | `Dataset` (hai hệ + mapping + `Version`/`SourceNote`), `Counts()` và `(Dataset).Verify() []IntegrityIssue` — hàm thuần, không I/O, nguồn duy nhất của các quy tắc toàn vẹn AC 21; kết quả sắp theo (quy tắc, đối tượng) để báo cáo ổn định; không sửa dữ liệu. |
| Domain / Repository interface | `domain/administrative_unit.go` *(created)* | `AdministrativeUnitRepository`: `ListCurrentProvinces`, `ListCurrentWards(provinceCode)`, `ListLegacyProvinces`, `ListLegacyDistricts(provinceCode)`, `ListLegacyWards(districtCode)`, `ConvertLegacyWard(code)`, `ConvertCurrentWard(code)`, `ConvertLegacyProvince(code)`, `ConvertCurrentProvince(code)`, `ReplaceDataset(ctx, Dataset)`. Repository trả slice ứng viên; usecase bọc thành `Conversion[T]`. Định nghĩa ở domain, không ở `repository/`. |
| Domain / Mock | `domain/mock_administrative_unit_repository.go` *(created)* | Mock viết tay cho test usecase. |
| Usecase | `services/appService/usecase/administrative_unit_usecase.go` *(created)* | `AdministrativeUnitUsecase` (constructor chỉ nhận repository, không config): chín method một mỗi route — `ListCurrentProvinces`, `ListCurrentWards`, `ListLegacyProvinces`, `ListLegacyDistricts`, `ListLegacyWards`, `ConvertLegacyWard`, `ConvertCurrentWard`, `ConvertLegacyProvince`, `ConvertCurrentProvince`. Trim `code`; mã trống → `ErrAdministrativeUnitNotFound` mà không gọi repository; danh sách nil → `[]`; bọc kết quả chuyển đổi vào `Conversion[T]` (`Unique()` do domain suy ra); bọc lỗi bằng `%w`. Không tự chọn ứng viên (AC 11). Không phân trang nên không cần config. |
| Repository | `services/appService/repository/postgres_administrative_unit_repository.go` *(created)* | `NewAdministrativeUnitRepository(db)`. SQL cho bảy bảng `addr_*`; mỗi read là **một** câu SELECT (join cho chuyển đổi), sắp theo `sort_order, code`; khi kết quả rỗng mới chạy thêm một câu `SELECT 1` để phân biệt "cha không tồn tại" (→ `ErrAdministrativeUnitNotFound`) với "có nhưng rỗng" (huyện đảo không có xã; đặc khu chưa có mapping cấp xã). Chuyển đổi cấp tỉnh là `SELECT DISTINCT` qua `addr_ward_mappings`. `ReplaceDataset` chạy một transaction, INSERT nhiều dòng theo lô 1.000 với `ON CONFLICT … DO UPDATE`, không DELETE, từ chối `Version` rỗng; **không** tự `Verify` (seeder gọi). |
| Delivery / Handler | `services/appService/delivery/http/administrative_unit_handler.go` *(not yet created)* | Parse `{code}` từ path → gọi usecase → map lỗi domain sang status; đặt header `Cache-Control` khi thành công. Interface `administrativeUnitUsecase` khai báo phía consumer. Không logic nghiệp vụ. |
| Delivery / DTO | `delivery/http/administrative_unit_handler.go` *(not yet created)* | `currentProvinceResponse`, `currentWardResponse`, `legacyProvinceResponse`, `legacyDistrictResponse`, `legacyWardResponse`, `wardCandidateResponse`, `conversionResponse{unique, candidates}` (JSON vẫn dùng khoá `new`/`old` trong path và `unique`/`candidates` trong body); JSON tag chỉ nằm ở đây. |
| Delivery / Error map | `delivery/http/errors.go` *(needs modification)* | Thêm `ErrAdministrativeUnitNotFound` → `404 ADMINISTRATIVE_UNIT_NOT_FOUND`; còn lại → `500 INTERNAL_ERROR` như hiện có. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` *(needs modification)* | Block `administrative_units.rate_limit` (`max`, `window`, `trust_forwarded_for`) theo hình dạng `TempleSuggestionRateLimit`, có `Validate` từ chối giá trị `< 1`; dev đề xuất 60 yêu cầu/phút. |
| Wiring | `services/appService/cmd/main.go` *(needs modification)* | Dựng repo → usecase → handler; đăng ký 9 route dạng `webAPIKey.WrapFunc(limit.WrapFunc(h.X))` (key chạy trước để key sai không tiêu hao ngân sách của người gọi — cùng thứ tự với `temple-suggestions`). Không dính khối `⚠️ DEV ONLY` nào của `main.go`. |
| Seed CLI | `services/appService/cmd/seedadminunits/main.go`, `parse.go` *(not yet created)* | `main`: đọc file (`-file`, mặc định `assets/data/administrative-units.json`), nạp config + secret dev (như `seedtemples`), kết nối DB, gọi `Dataset.Verify` rồi `ReplaceDataset`, in số dòng đã nạp. `parse.go`: JSON → `Dataset`, chuẩn hoá NFC, **không** sửa dữ liệu im lặng. Có khối `⚠️ DEV ONLY` của riêng nó, giống `seedtemples`. |
| Data | `assets/data/administrative-units.json` *(created 2026-10-10)* | Một file JSON gồm `meta` + tám mảng (xem `requirements.md` Ghi chú kỹ thuật). Đã áp 2 sửa lỗi nhãn và loại 2 dòng huyện nhân đôi; dòng Kim Sơn nằm ở `meta.unresolved`; 5 mapping cấp huyện nằm sẵn trong file nhưng chưa nạp. Đọc từ đĩa lúc chạy seed, không `go:embed`. |
| Migration | `migrations/0026_administrative_units.sql` *(created)* | Bảy bảng ở mục "Schema" bên dưới; idempotent (`IF NOT EXISTS`); không CHECK trên cột `type`. `0027` do feature khác (lamp) đã lấy sau đó nên không còn là migration cuối. |
| API spec | `docs/specs/api/openapi.yaml` *(needs modification)* | Chín route + schema, cập nhật cùng thay đổi với handler; `web/docs/public/openapi.yaml` là bản đồng bộ. |
| ERD | `docs/specs/db/erd.mmd` *(updated)*, `erd.svg` *(stale — chưa sinh lại)* | Bảy bảng `addr_*` và năm quan hệ đã có trong `erd.mmd`; `erd.svg` cần sinh lại bằng mermaid-cli. |

## Schema (dự kiến cho migration `0026`)

Hai hệ ở bảng riêng vì mã có thể trùng số nhưng khác nghĩa (AC 8); tên bảng số nhiều như `temples`. Cột là đề xuất, chốt khi viết migration. Mọi bảng đơn vị có thêm `sort_order INT NOT NULL` lấy từ file dữ liệu (thứ tự chữ cái tiếng Việt trong phạm vi cha), nên `ORDER BY sort_order, code` không phụ thuộc collation của Postgres.

| Bảng | Khoá | Cột chính | Ghi chú |
|---|---|---|---|
| `addr_dataset_versions` | `version` | `source_note`, `loaded_at` | Một dòng mỗi `version` mới nạp thành công; `version` chính là ngày của bộ dữ liệu (`2026-09-25`) nên không có cột `source_date` riêng. |
| `addr_new_provinces` | `code` | `name`, `full_name`, `type`, `sort_order` | 34 dòng. |
| `addr_new_wards` | `code` | `province_code` FK, `name`, `full_name`, `type`, `sort_order` | 3.321 dòng; index `(province_code, sort_order)`. |
| `addr_old_provinces` | `code` | `name`, `full_name`, `type`, `sort_order` | 63 dòng. |
| `addr_old_districts` | `code` | `province_code` FK, `name`, `full_name`, `type`, `sort_order` | 696 dòng; `code` là khoá chính **vì** file dữ liệu đã loại 2 dòng nhân đôi. Cột `type` không CHECK ở mọi bảng: là nhãn của bộ dữ liệu, một nhãn mới không được làm hỏng cả lần nạp. |
| `addr_old_wards` | `code` | `district_code` FK, `name`, `full_name`, `type`, `sort_order` | 10.033 dòng; tỉnh suy ra qua huyện (không lưu lặp). |
| `addr_ward_mappings` | `(old_ward_code, new_ward_code)` | `mapping_type`, `note` | 10.565 dòng; N:M; cả hai cột là FK; index theo `new_ward_code` cho chiều mới → cũ. |

Chuyển đổi cấp tỉnh **không có bảng riêng**: cũ → mới là `SELECT DISTINCT nw.province_code` qua `addr_ward_mappings` từ các xã của tỉnh cũ; mới → cũ là hướng ngược lại (AC 12). Quy tắc "xã cũ phải cùng tỉnh với huyện của nó" kiểm ở `Dataset.Verify` vì schema chuẩn hoá (tỉnh suy ra qua huyện) không biểu diễn được vi phạm này — `old_wards[].province_code` trong file dữ liệu chỉ để kiểm tra, không lưu.

## Request flow

Mọi route đọc chỉ một câu SELECT nên giữ dạng ASCII; riêng flow nạp dữ liệu có nhiều bước nên nằm ở Step-by-step Flow và `flow.md`.

```
GET /v1/administrative-units/new/provinces/{code}/wards
  └─ webAPIKey.WrapFunc    (header vắng/đúng → đi qua; sai → 401 INVALID_API_KEY)
       └─ clientIPRateLimit.WrapFunc   (quá ngân sách → 429 RATE_LIMITED)
            └─ AdministrativeUnitHandler.NewWards                          (delivery/http)
                 └─ AdministrativeUnitUsecase.ListNewWards(ctx, code)      (usecase)
                      └─ AdministrativeUnitRepository.ListNewWards(code)   (repository)
                           ├─ SELECT … FROM addr_new_wards WHERE province_code = $1 ORDER BY sort_order, code
                           └─ rỗng → kiểm tỉnh tồn tại: không có → ErrAdministrativeUnitNotFound → 404
                      thành công → 200 + Cache-Control: public, max-age=86400

GET /v1/administrative-units/old/wards/{code}/conversion
  └─ webAPIKey.WrapFunc → clientIPRateLimit.WrapFunc
       └─ AdministrativeUnitHandler.ConvertOldWard                    (delivery/http)
            └─ AdministrativeUnitUsecase.ConvertOldWard(ctx, code)    (usecase)
                 └─ AdministrativeUnitRepository.ConvertOldWard(code) (repository)
                      └─ SELECT nw.*, np.*, m.mapping_type, m.note
                         FROM addr_ward_mappings m
                         JOIN addr_new_wards nw ON nw.code = m.new_ward_code
                         JOIN addr_new_provinces np ON np.code = nw.province_code
                         WHERE m.old_ward_code = $1 ORDER BY nw.sort_order, nw.code
                           └─ không có dòng và xã cũ không tồn tại → 404; Unique = (số dòng == 1)
```

## Step-by-step Flow — Nạp bộ dữ liệu hành chính (`cmd/seedadminunits`)

Đọc file, kiểm tra toàn vẹn rồi ghi nhiều bảng trong một transaction — nhiều bước tuần tự có side effect nên bắt buộc có section này (rule 5).

```
Step 1 — Đọc file dữ liệu (seed CLI)
- Trigger: operator chạy `go run ./services/appService/cmd/seedadminunits -file assets/data/administrative-units.json`
- Pre-condition: file tồn tại và đọc được
- Call: os.ReadFile
    - Success → Step 2
    - Failure (không có file / không đọc được) → in đường dẫn, exit khác 0; chưa chạm DB

Step 2 — Giải mã và dựng Dataset (seed CLI, parse.go)
- Trigger: Step 1 pass
- Pre-condition: nội dung đọc được
- Call: json.Unmarshal; chuẩn hoá chuỗi NFC; dựng Dataset gồm hai hệ + ward_mappings + Version/SourceDate từ `meta`; `district_mappings` được đọc và kiểm nhưng không nạp (hoãn)
  Confirmed: parse không sửa, không bỏ, không "đoán" dòng nào — mọi bất thường để Step 3 báo.
    - Success → Step 3
    - Failure (JSON hỏng / thiếu khoá bắt buộc) → in vị trí, exit khác 0; chưa chạm DB

Step 3 — Kiểm tra toàn vẹn (domain, hàm thuần)
- Trigger: Step 2 pass
- Pre-condition: Dataset đã dựng
- Call: Dataset.Verify() — bảy quy tắc của AC 21 (trùng mã trong một bảng, kể cả mã huyện cũ do hai tỉnh cùng nhận; đơn vị trỏ tới cha không có; xã cũ khác tỉnh với huyện của nó; xã cũ thiếu mapping; xã mới không là đích của mapping nào; mapping trỏ tới đơn vị không có; tỉnh cũ có xã đi sang nhiều hơn một tỉnh mới)
  Confirmed: file dữ liệu hiện tại qua cả bảy quy tắc — đã chạy `Dataset.Verify` thật trên file (2026-10-10, 0 vi phạm); đặt lại lỗi nhãn gốc của xã `27691` thì bị bắt bởi hai quy tắc (tỉnh khác huyện, tỉnh cũ tách đôi).
    - Success (không có vi phạm) → Step 4
    - Failure (có vi phạm) → in báo cáo từng dòng, exit khác 0; chưa mở transaction nên không ghi gì

Step 4 — Mở transaction và upsert hai hệ (repository.ReplaceDataset)
- Trigger: Step 3 pass
- Pre-condition: kết nối DB sống (config + secret dev, như seedtemples)
- Call: BEGIN; INSERT … ON CONFLICT (code) DO UPDATE cho `addr_new_provinces`, `addr_new_wards`, `addr_old_provinces`, `addr_old_districts`, `addr_old_wards` (cha trước con)
  Confirmed: chỉ upsert, **không DELETE** — đơn vị có trong DB mà vắng trong file được giữ nguyên (chạy tay, người dùng chốt 2026-10-10).
    - Success → Step 5
    - Failure (lỗi DB, vi phạm FK) → ROLLBACK, in lỗi, exit khác 0; DB giữ nguyên dữ liệu trước

Step 5 — Upsert bảng đối chiếu (repository.ReplaceDataset, cùng transaction)
- Trigger: Step 4 pass
- Pre-condition: mọi đơn vị cha/con đã có trong transaction
- Call: INSERT … ON CONFLICT (old_ward_code, new_ward_code) DO UPDATE cho `addr_ward_mappings`
    - Success → Step 6
    - Failure (FK/lỗi DB) → ROLLBACK như Step 4

Step 6 — Ghi phiên bản và commit (repository.ReplaceDataset, cùng transaction)
- Trigger: Step 5 pass
- Pre-condition: mọi bảng đã ghi trong transaction
- Call: INSERT INTO addr_dataset_versions (version, source_note, loaded_at) ON CONFLICT (version) DO NOTHING; COMMIT
  Confirmed: chạy lại cùng bộ dữ liệu không đổi đơn vị hay mapping nào (upsert đồng nhất) và không thêm dòng version thứ hai.
    - Success → Step 7
    - Failure (COMMIT lỗi) → ROLLBACK, in lỗi, exit khác 0

Step 7 — Báo cáo (seed CLI)
- Trigger: Step 6 pass
- Pre-condition: đã commit
- Call: in số dòng đã nạp mỗi bảng và phiên bản
    - Success → exit 0 (operator so với `meta.counts` của file: 34 / 3.321 / 63 / 696 / 10.033 / 10.565)
```

> Xem `flow.md` cho sơ đồ sequence Mermaid của flow nạp dữ liệu (mirror 1:1).

## Deliberately stubbed for now

Chưa có code nào được implement nên không có stub. Phần dưới đây **cố ý chưa làm**:

- **Chuyển đổi cấp huyện** và bảng `addr_district_mappings`: hoãn theo người dùng (2026-10-10). Dữ liệu 5 huyện đảo nằm sẵn trong file (`district_mappings`) để thêm sau không phải dựng lại file.
- **Đánh dấu hết hiệu lực cho đơn vị vắng trong bản mới**: lần nạp chỉ upsert nên không xoá; thêm cột trạng thái là việc sau.
- **Tìm theo tên không dấu (`?q=`)**: chưa được hỏi; không có trong chín route.

## Known gaps carried by this feature

- **Web API key không ngăn người gọi bỏ header.** Header vắng được coi là mobile và vẫn được phục vụ, nên key chỉ xác thực nơi nhận mình là website (comment `web_api_key_middleware.go`); giới hạn theo IP là lớp chặn lạm dụng duy nhất còn lại.
- **Dữ liệu chưa đối chiếu với file chính thức.** Nguồn là repo cộng đồng `vietnam-address-kit` (phần điều chỉnh 2026 do tác giả chép tay); người dùng hoãn đối chiếu `conversion-table.xlsx` và giấy phép dữ liệu. Dòng "Huyện Kim Sơn → Xã Kim Đông" nằm ở `meta.unresolved` chứ không bị bỏ im lặng.
- **File dữ liệu đã bị sửa so với nguồn** (2 nhãn tỉnh, 2 dòng huyện bị loại): được ghi trong `meta.corrections`; cập nhật nguồn sau này phải áp lại các sửa này hoặc kiểm xem nguồn đã tự sửa chưa. Script sinh file hiện **không** nằm trong repo.
- **Hai nguồn "danh sách 34 tỉnh" tồn tại song song:** `domain/province.go` (hằng số cho `temples.province`) và bảng `addr_new_provinces`. Tên hiện trùng khít (đã so sánh); chưa chốt hợp nhất — một thay đổi hành chính sau này phải sửa cả hai. Comment của `province.go` ghi "28 tỉnh và 6 thành phố" còn dữ liệu 2026-09-25 ghi 25 + 9; chỉ lệch loại, không lệch tên.
- **`cmd/seedadminunits` dùng config và secret dev** như `seedtemples` (`⚠️ DEV ONLY`); quy trình nạp ở staging/production chưa có (người dùng: chạy tay khi có cập nhật).
- **Không có thay đổi nào cho địa chỉ đang lưu**: `temples.province` vẫn là chuỗi tự do, chưa có cột mã.
- **Repository test của cả package dùng chung một quy ước cũ**: `connectTestDB` chạy `TRUNCATE` trên database `innerlamp` mặc định (xoá dữ liệu dev đã seed). Test của feature này chỉ xoá các bảng `addr_*`, và đã chạy trên database riêng `innerlamp_test` (`TEST_DB_NAME=innerlamp_test`); các test khác trong package vẫn xoá `temples`, `users`… ở DB mặc định.
- **Năm đặc khu đảo chỉ là đích của mapping cấp huyện** (Bạch Long Vĩ, Cồn Cỏ, Hoàng Sa, Lý Sơn, Côn Đảo). Vì mapping cấp huyện chưa được nạp (hoãn), chuyển một đặc khu này từ mới sang cũ trả danh sách **rỗng** (`Unique() = false`, không có ứng viên), không phải 404. `Verify` vẫn coi chúng là đã có đích để file qua kiểm tra. Cần quyết định khi làm tiếp chuyển đổi cấp huyện.
