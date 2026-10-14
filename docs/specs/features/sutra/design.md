# Design — sutra

Version: 3.0
Last updated: 2026-10-11
Status: implemented (2026-10-03) — domain, usecase, repository, seed, handler, 4 route, `cmd/seedsutras`, migration `0013`, `openapi.yaml`, ERD, `migrations/README.md`, tài liệu dev local; đã chạy thật binary + Postgres và có E2E riêng cho sutra (7 hàm test, đã kiểm bằng đột biến). v2.12 (2026-10-07): tách trường + `summary` đã implement và kiểm chứng (Phase 6 của `tasks.md`), trừ việc áp lên DB đã seed. Còn mở: `sutraSeedTimeout` có nên ở `config/` không (câu hỏi cho người dùng). `durationMinutes` được giữ nguyên giá trị đã sinh theo quyết định của người dùng (2026-10-03).
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Bốn route `GET` đều công khai, không đi qua middleware xác thực: `/v1/sutras`, `/v1/sutras/{id}`, `/v1/sutras/{id}/lines` và `/v1/deities/{id}/sutras`. Feature chạm đủ bốn layer Clean Architecture của `services/appService` (`domain` → `usecase` → `repository` / `delivery/http`, nối dây ở `cmd/main.go`), sở hữu migration `0013_sutras.sql` (4 bảng) và mở rộng package `assets/` để `go:embed` thư mục dữ liệu. Cùng hình dạng với `article` (danh sách phân trang theo `page`, chi tiết, seed **tự động một lần khi khởi động**), khác ở ba điểm: mỗi văn bản có **nhiều bản dịch**, nội dung được **phân trang theo số dòng mỗi trang**, và có **khoá ngoại sang `deities`** nên lần khởi động đầu phụ thuộc `cmd/seeddeities`.

Dữ liệu được thu thập từ web nên JSON chỉ là **nguồn nạp một lần**: sau lần nạp đầu, Postgres là nguồn dữ liệu duy nhất và mọi route đọc từ DB. `ParseSutraSeed` nhận `fs.FS` nên cùng một parser dùng được cho dữ liệu nhúng trong binary (đường chính) lẫn một thư mục trên đĩa (`cmd/seedsutras -dir`, cửa vào thứ hai đã chốt).

Route `GET /v1/deities/{id}/sutras` thuộc feature này nhưng đăng ký dưới tiền tố `/v1/deities/`; `deityHandler` không đổi. Usecase tra ngược dùng lại `DeityRepository.FindByID` có sẵn để kiểm tra Phật/Bồ tát tồn tại.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/sutra.go` | `Sutra`, `SutraTranslation`, `SutraSection`, `SutraLine`, `SutraDeity`, `DeitySutra` (= `Sutra` + `Role`), `SutraPage`, `SutraLinesPage`, `SutraFilter`, `SutraSeed`; hằng enum `Tradition`, `Kind`, `Variant`, `DeityRole`, `Purpose`. `DurationMinutes *int`. Từ v2.12: `Sutra` thêm `OriginalTitle`, `Author`, `HanTranslator`, `HanTranslationYear *int`, `ReferenceNote`, `Summary`; `SutraTranslation` thêm `LicenseURL`, `LicenseNote`; hằng mã giấy phép (`LicenseCC0` = `CC0-1.0`, `LicenseCCBYNC`, `LicensePublicDomain`, `LicenseCopyAllowed`, `LicenseUnknown`) và `ValidLicense(code string) bool` — `License` vẫn là `string`. Không JSON/db tag. |
| Domain / Errors | `domain/sutra.go` | `ErrSutraNotFound`, `ErrSutraTranslationNotFound`, `ErrInvalidSutraFilter` (mang tên field), `ErrInvalidPageSize`, `ErrInvalidTranslation`. Tái dùng `ErrInvalidPage` (định nghĩa ở `domain/temple.go`) và `ErrDeityNotFound` (`domain/deity.go`). |
| Domain / Repository interface | `domain/sutra.go` | `SutraRepository`: `Count(ctx, filter)`, `List(ctx, filter, offset, limit)`, `FindByID` (từ v2.12 đọc thêm sáu cột chỉ có ở chi tiết), `FindTranslation(ctx, sutraID, key)`, `ListLines(ctx, translationID, fromN, toN)`, `CountByDeity`, `ListByDeity(ctx, deityID, role, offset, limit)`, `MissingDeities(ctx, ids)`, `InsertAll(ctx, []SutraSeed)`. Thứ tự `sort_order ASC, id ASC` là một phần của hợp đồng. |
| Domain / Mock | `domain/mock_sutra_repository.go` | Mock viết tay cho test usecase. |
| Usecase | `services/appService/usecase/sutra_usecase.go` | `List` (kiểm tra bộ lọc và `page`, tính offset/`total_pages`), `Get`, `Lines` (kiểm tra `page`/`page_size`, kẹp `page_size`, đổi sang khoảng `n`), `ListByDeity` (kiểm tra `role`/`page`, kiểm tra deity tồn tại), `SeedIfEmpty(load)`. Cỡ trang lấy từ `config.SutraConfig`. |
| Repository | `services/appService/repository/postgres_sutra_repository.go` | SQL cho 4 bảng; `List`/`Count` dựng `WHERE` từ bộ lọc rồi nạp bản dịch và liên kết bằng `= ANY($1)`; `ListLines` truy vấn khoảng khoá `(translation_id, n)`; `InsertAll` chạy MỘT transaction với `ON CONFLICT (id) DO NOTHING` cho `sutras`, chèn dòng theo lô nhiều hàng; `pq.Array` cho `purposes`; `NULL` ↔ `nil` cho `duration_minutes`/`section`. |
| Repository / Seed parser | `services/appService/repository/sutra_seed.go` | `ParseSutraSeed(fsys fs.FS)`: đọc `index.json` + từng `<id>.json`, kiểm toàn bộ điều kiện của AC29 và AC40 (`license` thuộc tập đóng, `hanTranslationYear` null hoặc ≥ 1; các trường mới còn lại tuỳ chọn, mặc định `""`), gán `SortOrder` = vị trí trong `index.json`, tính `sections`. Nhận `fs.FS` để test bằng `fstest.MapFS`. |
| Embedded data | `assets/embed.go`, `assets/data/sutras/*.json` | Thêm `SutrasFS embed.FS` với `//go:embed data/sutras/*.json` để binary tự mang dữ liệu seed, không phụ thuộc thư mục làm việc. Chỉ nhúng `*.json` (không nhúng `README.md`). |
| Seed command | `services/appService/cmd/seedsutras/main.go` | Cửa vào thứ hai cho việc nạp: nạp cấu hình và kết nối DB như `cmd/seeddeities`, dựng repository + usecase, gọi `SutraUsecase.SeedIfEmpty` với dữ liệu nhúng (hoặc `os.DirFS(-dir)`). Bảng đã có dòng thì log `sutras already seeded, skipping` và thoát 0 (không ghi); lỗi (dữ liệu sai, thiếu deity) thì thoát 1. Logic chọn nguồn nằm trong hàm `run`/`dataFS` để test được; không chứa logic nạp riêng. |
| Delivery / Handler | `services/appService/delivery/http/sutra_handler.go` | `List`, `Get`, `Lines`, `ListByDeity`; đọc query (`page`, `page_size`, `tradition`, `kind`, `collection`, `purpose`, `daily`, `max_minutes`, `lang`, `translation`, `role`); map entity → DTO; `nonNil` cho mảng. |
| Delivery / DTO | `delivery/http/sutra_handler.go` | `sutraSummaryResponse`, `sutraDetailResponse` (v2.12: thêm sáu trường của AC36), `sutraTranslationResponse` (chi tiết thêm `license_url`, `license_note`), `sutraLineResponse`, `sutraLinesResponse`, `sutraDeityResponse`, `sutraListResponse`, `sutraByDeityItemResponse`, `sutraByDeityListResponse` — JSON tag chỉ nằm ở đây. |
| Delivery / Error map | `delivery/http/errors.go` | `ErrSutraNotFound` → `404 SUTRA_NOT_FOUND`; `ErrSutraTranslationNotFound` → `404 TRANSLATION_NOT_FOUND`; `ErrInvalidSutraFilter` → `400 INVALID_FILTER` (`details.field`); `ErrInvalidPageSize` → `400 INVALID_PAGE_SIZE`; `ErrInvalidTranslation` → `400 INVALID_TRANSLATION`. `ErrInvalidPage` → `400 INVALID_PAGE` và `ErrDeityNotFound` → `404 DEITY_NOT_FOUND` đã có sẵn. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `SutraConfig{PageSize, LinesPageSize, LinesMaxPageSize}`, block `sutra:` (dev: 20, 200, 500). Khác `article`/`deity`: `SutraConfig.Validate()` từ chối giá trị < 1 (và `lines_page_size` lớn hơn `lines_max_page_size`), vì 0 sẽ khiến mọi trang rỗng và chia cho 0 ở `total_pages`. `Validate` được gọi từ `cmd/main.go` ngay sau `Load` (không nằm trong `Load`, vì `Load` hiện không kiểm tra giá trị nào và các test cũ của nó dùng YAML không có block catalogue). |
| Wiring | `services/appService/cmd/main.go` | Dựng repo → usecase → handler; **không seed lúc khởi động** (từ 2026-10-11; thư viện kinh nạp bằng `cmd/seedsutras`, sau khi `deities` đã có); đăng ký 4 route. Không dính khối `⚠️ DEV ONLY` nào. |
| E2E harness | `services/appService/e2e/harness.go` | Sau khi áp migration, seed `deities` từ `assets/data/phat_botat.json` (như `cmd/seeddeities` trong triển khai thật) vì service không khởi động lần đầu nếu thiếu `deities`; khi binary không lên, in stderr của nó thay vì chỉ báo hết giờ. Không có thay đổi này thì 3 test E2E của register/login hỏng. |
| Migration | `migrations/0013_sutras.sql` | 4 bảng + index (xem "Mô hình dữ liệu"). Idempotent (`IF NOT EXISTS`) như mọi file khác. |
| Migration (v2.12) | `migrations/0020_sutra_reference_license_summary.sql` | `ALTER TABLE … ADD COLUMN IF NOT EXISTS` cho 6 cột của `sutras` và 2 cột của `sutra_translations` (xem "Mô hình dữ liệu"); idempotent; không sửa dữ liệu đã có (xem marker ở `requirements.md`). |
| Data | `assets/data/sutras/*.json`, `index.json` | Thêm `key`, `dailyChant`, `purposes`, `durationMinutes` (xem "Dữ liệu cần bổ sung"). v2.12: tách `reference`/`license`, thêm `originalTitle`, `author`, `hanTranslator`, `hanTranslationYear`, `referenceNote`, `summary`, `licenseUrl`, `licenseNote` (đã làm 2026-10-07, 153 file + `index.json`). |
| API spec | `docs/specs/api/openapi.yaml`, `web/docs/public/openapi.yaml` | 4 route (tag Sutras) và schema `SutraSummary`/`SutraDetail`/`Line`. |
| DB spec | `docs/specs/db/erd.mmd`, `erd.svg` | Thêm 4 bảng và các quan hệ (đã cập nhật 2026-10-03, SVG tạo bằng `mermaid-cli`). |

Ghi chú về route: `mux.HandleFunc("GET /v1/sutras/{id}/lines", …)` và `"GET /v1/sutras/{id}"` là hai pattern khác độ dài nên không xung đột; `"GET /v1/deities/{id}/sutras"` khác pattern với `"GET /v1/deities/{id}"` đã có.

## Mô hình dữ liệu

```
sutras
  id               VARCHAR(128) PRIMARY KEY            -- slug ổn định
  sort_order       INTEGER NOT NULL                    -- vị trí trong index.json
  tradition        VARCHAR(16) NOT NULL CHECK (IN 'THERAVADA','MAHAYANA','VAJRAYANA')
  kind             VARCHAR(16) NOT NULL CHECK (IN 'SUTRA','VERSE','DHARANI')
  collection       VARCHAR(128) NOT NULL DEFAULT ''
  reference        TEXT NOT NULL DEFAULT ''                -- chỉ mã trích dẫn (v2.12), vd. 'Taishō T0450', 'MN 86'
  original_title   TEXT NOT NULL DEFAULT ''                -- v2.12: tên Hán-Việt/tên gốc đầy đủ (hoặc kinh gốc của bài chú)
  author           TEXT NOT NULL DEFAULT ''                -- v2.12: tác giả (không phải người dịch)
  han_translator   TEXT NOT NULL DEFAULT ''                -- v2.12: người dịch sang Hán
  han_translation_year INTEGER CHECK (han_translation_year >= 1)  -- v2.12: năm dịch (CN); NULL = không rõ
  reference_note   TEXT NOT NULL DEFAULT ''                -- v2.12: phần còn lại của ghi chú nguồn
  summary          TEXT NOT NULL DEFAULT ''                -- v2.12: tóm tắt tiếng Việt, '' nếu chưa có
  titles           JSONB NOT NULL                      -- {"vi": "...", "en": "..."}; luôn có "vi"
  daily_chant      BOOLEAN NOT NULL DEFAULT false
  purposes         TEXT[] NOT NULL DEFAULT '{}'
  duration_minutes INTEGER CHECK (duration_minutes >= 1)   -- NULL = không ước tính được
  review_status    VARCHAR(16) NOT NULL DEFAULT 'UNREVIEWED'
  created_at, updated_at TIMESTAMPTZ

sutra_translations
  id           BIGSERIAL PRIMARY KEY
  sutra_id     VARCHAR(128) NOT NULL REFERENCES sutras(id) ON DELETE CASCADE
  key          VARCHAR(64) NOT NULL                    -- vd. vi-chant, vi-meaning-2
  ordinal      INTEGER NOT NULL                        -- thứ tự trong file
  lang         VARCHAR(8) NOT NULL
  variant      VARCHAR(16) NOT NULL CHECK (IN 'translation','chant','meaning','poem')
  translator, source_name, source_url, license, note   TEXT NOT NULL DEFAULT ''   -- license (v2.12) là mã: CC0-1.0 | CC-BY-NC-4.0 | PUBLIC_DOMAIN | COPY_ALLOWED | UNKNOWN
  license_url, license_note                            TEXT NOT NULL DEFAULT ''   -- v2.12: URL và câu nguyên văn của nguồn
  line_count   INTEGER NOT NULL
  sections     JSONB NOT NULL DEFAULT '[]'             -- [{name, start_n, line_count}]
  UNIQUE (sutra_id, key)

sutra_lines
  translation_id BIGINT NOT NULL REFERENCES sutra_translations(id) ON DELETE CASCADE
  n              INTEGER NOT NULL
  text           TEXT NOT NULL
  section        TEXT                                  -- NULL nếu không thuộc phẩm nào
  PRIMARY KEY (translation_id, n)

sutra_deities
  sutra_id  VARCHAR(128) NOT NULL REFERENCES sutras(id) ON DELETE CASCADE
  deity_id  VARCHAR(64)  NOT NULL REFERENCES deities(id)
  role      VARCHAR(16)  NOT NULL CHECK (IN 'SPEAKER','SUBJECT','MENTIONED')
  PRIMARY KEY (sutra_id, deity_id)
```

Index: `idx_sutras_order (sort_order, id)`; `idx_sutras_tradition_kind (tradition, kind, sort_order)`; `idx_sutras_purposes` GIN trên `purposes`; `idx_sutra_deities_deity (deity_id, role)` để tra ngược. Khoá chính `(translation_id, n)` của `sutra_lines` chính là chỉ mục cho truy vấn khoảng `n > $from AND n <= $to`.

Lý do chọn:
- **Tách trường (v2.12)**: `reference` và `license` từng chứa nhiều thứ trong một chuỗi (mã Taishō + người dịch Hán + năm; mã giấy phép + URL + câu giải thích) nên client không hiển thị riêng được. Mỗi thứ nay là một cột; `license` là mã trong tập đóng để client ánh xạ nhãn, câu nguyên văn của nguồn giữ nguyên ở `license_note` (không mất thông tin). Không dùng `CHECK` cho mã `license` ở DB — tập này do parser kiểm (AC40) và có thể thêm mã mà không cần migration. `original_title` cũng dùng cho tên của kinh gốc mà bài chú được trích ra (vd. chú Dược Sư → `Kinh Dược Sư`), vì nguồn không phân biệt hai trường hợp.
- **`reference` là `TEXT`**: lúc đầu thiết kế là `VARCHAR(128)`, nhưng `lang-nghiem` có `reference` dài 135 ký tự (cả tên kinh) và nạp dữ liệu thật báo `value too long`; trích dẫn nguyên văn của nguồn không có độ dài chặn trên hợp lý.
- **Mỗi dòng một hàng** (`sutra_lines`, ~64.000 hàng) thay vì mảng `TEXT[]` trong bản dịch: trang dòng đọc bằng khoá chính, không phải đọc và giải nén cả mảng 6.967 dòng cho mỗi trang.
- **Trang dòng bằng khoảng `n`, không `OFFSET`**: vì `n` liên tục 1..N trong mỗi bản dịch (AC19), trang `p` cỡ `s` là `n ∈ ((p−1)·s, p·s]`; không phải bỏ qua hàng.
- **`sections` là JSONB tính sẵn** ở bước seed: mục lục `[{name, start_n, line_count}]` trả ngay trong chi tiết kinh mà không phải quét `sutra_lines`, và client đổi `start_n` thành trang bằng `floor((start_n−1)/page_size)+1`.
- **`line_count` lưu sẵn**: `total` và `total_pages` của trang dòng không cần câu `count(*)`.
- **`sutra_deities` khoá `(sutra_id, deity_id)`**: dữ liệu hiện không có văn bản nào gắn cùng một vị hai vai; parser kiểm tra ràng buộc này.

## Dữ liệu cần bổ sung

File JSON (`assets/data/sutras/<id>.json`, `index.json`) cần bốn trường dưới đây. **Đã thêm `key` và `dailyChant` ngày 2026-10-03** (153 file, `dailyChant = true` ở 34 văn bản); **`purposes` và `durationMinutes` được thêm cùng ngày** (6 văn bản có `purposes = []`, 10 văn bản Kim Cang Thừa có `durationMinutes = null`); seed lúc khởi động sẽ coi dữ liệu thiếu chúng là không hợp lệ và abort. Đây là việc của `tasks.md` Phase 0, không phải của bước seed.

| Trường JSON | Cột | Cách có giá trị |
|---|---|---|
| `translations[].key` | `sutra_translations.key` | `"<lang>-<variant>"`; từ bản thứ hai trùng cặp thêm hậu tố `-2`, `-3`. Hiện chỉ `tam-kinh` trùng (hai bản `vi` + `meaning`). |
| `dailyChant` | `sutras.daily_chant` | Gán theo nhóm kinh (đề xuất bên dưới). |
| `purposes` | `sutras.purposes` | Gán theo nhóm kinh (đề xuất bên dưới), giá trị thuộc bảng `purposes`. |
| `durationMinutes` | `sutras.duration_minutes` | **Do mình ghi** (ước tính biên tập: nghĩ bài cần 30 phút thì ghi 30). Script chỉ gợi ý giá trị ban đầu: `ceil(số âm tiết / 80)`, tối thiểu 1, từ bản `vi` ưu tiên `chant` > `translation` > `meaning` > `poem`; sau đó làm tròn (≤10 giữ nguyên; 11–60 lên bội của 5; 61–180 lên bội của 10; >180 lên bội của 30) và/hoặc chỉnh tay. `null` khi chưa có ước tính (vd. không có bản `vi`). |

**Trường thêm ở v2.12** (đã ghi vào dữ liệu 2026-10-07):

| Trường JSON | Cột | Cách có giá trị |
|---|---|---|
| `reference` (đã thu gọn) | `sutras.reference` | Chỉ mã trích dẫn; tách phần còn lại sang các trường dưới theo bảng ánh xạ viết tay cho 48 văn bản có chuỗi gộp (153 văn bản: 21 có `originalTitle`, 13 `hanTranslator`, 4 `hanTranslationYear`, 8 `author`, 13 `referenceNote`); tiêu đề Phạn của `kim-cang`, `tam-kinh` bỏ vì đã có ở `titles.sa`. |
| `originalTitle`, `author`, `hanTranslator`, `hanTranslationYear`, `referenceNote` | `sutras.*` | Lấy nguyên văn từ chuỗi `reference` cũ (chuẩn hoá chính tả tên người dịch Hán thành một dạng, vd. `Cưu Ma La Thập`); không thêm thông tin ngoài nguồn. |
| `summary` | `sutras.summary` | **Do em (Claude) soạn** cho 153 văn bản, 2–4 câu tiếng Việt từ tên bài, bộ, ghi chú và phần đầu/cuối bản dịch (A-011); chưa ai duyệt. |
| `translations[].license` (đã thu gọn) | `sutra_translations.license` | Mã trong tập đóng: `CC0-1.0` 207, `UNKNOWN` 107, `COPY_ALLOWED` 38 (trang ghi "được sao chép tự do" — không phải giấy phép chuẩn), `CC-BY-NC-4.0` 27, `PUBLIC_DOMAIN` 12 bản dịch. |
| `translations[].licenseUrl`, `licenseNote` | `license_url`, `license_note` | URL (chỉ khi nguồn cho, 3 dạng CC0) và câu nguyên văn còn lại của nguồn. |

> Đã chốt (2026-10-03, làm rõ ở v2.2): thời lượng là **ước tính biên tập**, không phải số đo; công thức 80 âm tiết/phút chỉ gợi ý ban đầu và có thể lệch thực tế (Tâm Kinh ra ~3 phút, A Di Đà ~24 phút, nhưng Dược Sư ra ~75 phút vì bản `translation` dài hơn bản tụng) nên chỉnh tay khi cần.

**Đề xuất khởi tạo cho `dailyChant` / `purposes`** (theo nhóm kinh; chưa ai duyệt):

| Nhóm / văn bản | `daily_chant` | `purposes` |
|---|---|---|
| `Nhật tụng / *`, `Kinh nhật tụng`, `Kinh / Cứu khổ` | `true` | theo tên nhóm: `Hồi hướng` → `HOI_HUONG`, `Phát nguyện` → `PHAT_NGUYEN`, `Tam quy`/`Năm giới` → `QUY_Y_THO_GIOI`, còn lại `CAU_AN` |
| `Sám hối` | `true` | `SAM_HOI` |
| Mọi `kind = DHARANI` (Chú, Thập Chú) | `true` cho Chú Đại Bi và Thập Chú, còn lại `false` | `TRI_CHU` (cộng `CAU_AN` với Chú Đại Bi) |
| Tâm Kinh, A Di Đà, Phổ Môn, Bát Đại Nhân Giác | `true` | Tâm Kinh `HOC_GIAO_LY`; A Di Đà `CAU_SIEU`; Phổ Môn `CAU_AN` |
| Dược Sư | `false` | `CAU_AN` |
| Địa Tạng, Vô Lượng Thọ, Quán Vô Lượng Thọ | `false` | `CAU_SIEU` |
| Vu Lan, Báo Phụ Mẫu | `false` | `BAO_HIEU`, `CAU_SIEU` |
| Niệm Xứ, Anapanasati, Thiền | `false` | `THIEN_DINH` |
| Kinh Từ Bi, Kinh Hạnh Phúc (Nam Tông) | `true` | `CAU_AN` |
| Còn lại (Nikāya, Pháp Cú, Pháp Hoa, Kim Cang, Lăng Nghiêm, Hoa Nghiêm, …) | `false` | `HOC_GIAO_LY` |

> Đã chốt (2026-10-03): bảng trên là giá trị khởi tạo do em (Claude) gán vì không có người am hiểu duyệt; bảng dựa theo tên nhóm kinh, **không** có nguồn tôn giáo. `review_status` giữ `UNREVIEWED` để sau này có người duyệt thì đổi.
> Giá trị đã ghi vào dữ liệu (2026-10-03) lệch bảng ở các chỗ sau, đều dựa theo tên bài: Kệ khai kinh, Lễ Phật, Tán Phật, Tán lễ → `[]` (bảng `purposes` không có mục đích phù hợp, và `CAU_AN` sẽ sai nghĩa); `bat-dai-nhan-giac` → `HOC_GIAO_LY`; Thập Chú theo tên (`chu-vang-sanh` thêm `CAU_SIEU`, `chu-tieu-tai-cat-tuong` thêm `CAU_AN`, `chu-that-phat-diet-toi` thêm `SAM_HOI`, `chu-duoc-su-quan-dinh` thêm `CAU_AN`); `pho-hien-hanh-nguyen` thêm `PHAT_NGUYEN`; ba kinh Niệm Xứ của Tương Ưng Bộ → `THIEN_DINH`; Kim Cang Thừa: hai nguyện văn → `PHAT_NGUYEN`, `tara-21-praise` → `[]`.

## Request flow

> Xem `flow.md` cho sơ đồ sequence Mermaid của cả năm flow bên dưới (mirror 1:1). Cả bốn route đọc đều có từ hai lời gọi DB nối tiếp trở lên, nên không giữ bản ASCII riêng.

## Step-by-step Flow — Nạp dữ liệu bằng lệnh `cmd/seedsutras` (service không tự nạp)

```
Step 1 — Đếm số văn bản đang có (usecase → repository)
- Trigger: cmd/seedsutras (`make ensure-sutras` hoặc chạy tay) gọi SutraUsecase.SeedIfEmpty(ctx, load) sau khi kết nối DB. Từ 2026-10-11 cmd/main.go không còn gọi SeedIfEmpty.
- Pre-condition: DB kết nối được; migration 0013 đã apply
- Call: SutraRepository.Count(ctx, SutraFilter{}) → SELECT count(*) FROM sutras
    - Success (n = 0) → Step 2
    - Success (n > 0) → dừng: trả 0 văn bản đã chèn, KHÔNG gọi load, KHÔNG ghi gì (restart không đụng dữ liệu); cmd/seedsutras log "sutras already seeded, skipping" và thoát 0
    - Failure (vd. bảng chưa tồn tại) → error bọc %w → main log.Fatalf "failed to seed sutras"
  Đã chốt (2026-10-03): thêm script cmd/seedsutras làm cửa vào thứ hai, dùng chung SeedIfEmpty; bảng đã có dữ liệu thì bỏ qua ở cả hai cửa vào.

Step 2 — Nạp và kiểm tra dữ liệu seed (repository)
- Trigger: n = 0
- Pre-condition: —
- Call: load() = repository.ParseSutraSeed(fs.Sub(assets.SutrasFS, "data/sutras")) — đọc index.json và từng <id>.json; kiểm mọi điều kiện của AC29 (file thiếu, id rỗng/trùng/khác tên file, enum ngoài tập cho phép, titles.vi rỗng, dailyChant/durationMinutes sai kiểu, key thiếu/trùng, n có hở, deity_id lặp, license ngoài tập đóng, hanTranslationYear không phải null/số nguyên ≥ 1 — AC40); gán SortOrder = vị trí trong index.json; tính sections
    - Success → danh sách domain.SutraSeed, sang Step 3
    - Failure (bất kỳ kiểm tra nào) → error nêu id và lý do → log.Fatalf, chưa ghi gì

Step 3 — Kiểm tra Phật/Bồ tát tồn tại (repository)
- Trigger: Step 2 xong
- Pre-condition: có tập deity_id được tham chiếu
- Call: SutraRepository.MissingDeities → SELECT id FROM deities WHERE id = ANY($1), rồi lấy các id được tham chiếu mà không có trong kết quả
    - Success (không thiếu) → Step 4
    - Failure (thiếu deity_id) → error nêu danh sách id thiếu và nhắc chạy cmd/seeddeities → log.Fatalf, chưa ghi dòng nào
    - Failure (lỗi DB) → error bọc %w → log.Fatalf
  Đã chốt (2026-10-03): thiếu deity thì abort startup, không bỏ qua liên kết thiếu (bỏ qua sẽ mất liên kết vĩnh viễn vì seed không chạy lại khi bảng đã có dòng). Thứ tự triển khai: migration → cmd/seeddeities → service.

Step 4 — Chèn toàn bộ văn bản (repository)
- Trigger: Step 3 xong
- Pre-condition: dữ liệu hợp lệ, deity đủ
- Call: SutraRepository.InsertAll — trong MỘT transaction, với mỗi văn bản theo thứ tự index.json:
    (a) INSERT INTO sutras … ON CONFLICT (id) DO NOTHING RETURNING id; không trả hàng (id đã tồn tại) → bỏ qua (b)–(d) của văn bản đó
    (b) INSERT INTO sutra_translations … RETURNING id cho từng bản
    (c) INSERT INTO sutra_lines theo lô nhiều hàng cho từng bản
    (d) INSERT INTO sutra_deities cho từng liên kết
    - Success → số văn bản thực sự được chèn (153 ở lần đầu) và số bản dịch/dòng/liên kết, sang Step 5
    - Failure (lỗi DB ở bất kỳ câu nào, kể cả vi phạm khoá ngoại) → rollback toàn bộ → error → log.Fatalf
    - Transaction boundary: mọi câu lệnh của Step 4 chung MỘT transaction — hoặc đủ hết hoặc không dòng nào

Step 5 — Ghi log (cmd/main.go hoặc cmd/seedsutras)
- Trigger: SeedIfEmpty trả về số văn bản đã chèn
- Pre-condition: —
- Call: nếu số > 0 → logger.Info("seeded sutras on first run", "sutras", n, "translations", t, "lines", l, "deity_links", d)
    - Success → tiếp tục khởi động, đăng ký route
    - Failure → không có nhánh lỗi
```

## Step-by-step Flow — Danh sách và bộ lọc (`GET /v1/sutras`)

```
Step 1 — Đọc page và bộ lọc (handler)
- Trigger: request đến route công khai
- Pre-condition: —
- Call: r.URL.Query() lấy page (rỗng → 1, ngược lại strconv.Atoi), tradition, kind, collection, purpose, daily, max_minutes, lang; tham số có mặt nhưng rỗng vẫn được coi là có mặt
    - Success → Step 2
    - Failure (page không phải số nguyên, kể cả tràn int) → writeDomainError(ErrInvalidPage) → 400 INVALID_PAGE, details.field = "page"

Step 2 — Kiểm tra khoảng của page và giá trị bộ lọc (usecase)
- Trigger: handler gọi SutraUsecase.List(ctx, filter, page)
- Pre-condition: page là số nguyên
- Call: page < 1 hoặc page > math.MaxInt32; tradition/kind/purpose thuộc tập cho phép; daily ∈ {true,false}; max_minutes là số nguyên ≥ 1 (và ≤ MaxInt32); không tham số nào rỗng
    - Success → Step 3 (offset = (page-1) × page_size, tính bằng int64)
    - Failure (page) → domain.ErrInvalidPage → 400 INVALID_PAGE
    - Failure (bộ lọc) → domain.ErrInvalidSutraFilter{Field} → 400 INVALID_FILTER, details.field

Step 3 — Đếm văn bản khớp (repository)
- Trigger: Step 2 xong
- Pre-condition: filter hợp lệ
- Call: SutraRepository.Count → SELECT count(*) FROM sutras WHERE <điều kiện AND>; max_minutes → duration_minutes <= $n (NULL tự bị loại); purpose → $p = ANY(purposes); lang → EXISTS (SELECT 1 FROM sutra_translations …)
    - Success → total, sang Step 4
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR

Step 4 — Lấy trang văn bản (repository)
- Trigger: Step 3 xong
- Pre-condition: có total
- Call: SutraRepository.List → SELECT <cột sutras> FROM sutras WHERE <điều kiện AND> ORDER BY sort_order ASC, id ASC OFFSET $1 LIMIT $2
    - Success → items (rỗng nếu offset vượt cuối), sang Step 5
    - Failure (lỗi DB / scan) → error bọc %w → 500 INTERNAL_ERROR

Step 5 — Nạp thông tin các bản dịch (repository)
- Trigger: Step 4 trả về ít nhất một văn bản (rỗng → bỏ qua Step 5–6, sang Step 7)
- Pre-condition: có tập id của trang
- Call: SELECT key, lang, variant, translator, license, line_count FROM sutra_translations WHERE sutra_id = ANY($1) ORDER BY sutra_id, ordinal
    - Success → gắn vào từng văn bản, sang Step 6
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR

Step 6 — Nạp liên kết Phật/Bồ tát (repository)
- Trigger: Step 5 xong
- Pre-condition: có tập id của trang
- Call: SELECT sd.sutra_id, sd.deity_id, sd.role, d.name_vi, d.thumbnail_url FROM sutra_deities sd JOIN deities d ON d.id = sd.deity_id WHERE sd.sutra_id = ANY($1) ORDER BY sd.sutra_id, role_rank, sd.deity_id
    - Success → gắn vào từng văn bản, sang Step 7
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR
    - Transaction boundary: không có — Step 3 đến 6 là các câu lệnh độc lập, không chung snapshot

Step 7 — Dựng trang và trả response (usecase + handler)
- Trigger: repository trả về
- Pre-condition: items, total
- Call: items nil → []; total_pages = (total + page_size - 1) / page_size; handler map sang DTO summary (mảng nil → []; duration_minutes nil → null)
    - Success → 200 {items, page, page_size, total, total_pages}
    - Failure → không có nhánh lỗi mới
```

## Step-by-step Flow — Chi tiết một kinh (`GET /v1/sutras/{id}`)

```
Step 1 — Tìm văn bản (repository)
- Trigger: handler đọc {id} rồi gọi SutraUsecase.Get(ctx, id)
- Pre-condition: —
- Call: SutraRepository.FindByID → SELECT <cột sutras, gồm summary từ v2.13> + original_title, author, han_translator, han_translation_year, reference_note FROM sutras WHERE id = $1 (v2.12: năm cột cuối chỉ đọc ở chi tiết; danh sách không đọc; v2.13: `summary` nằm trong cột chung `sutraColumns` nên danh sách và by-deity cũng đọc)
    - Success → Step 2
    - Failure (sql.ErrNoRows) → domain.ErrSutraNotFound → 404 SUTRA_NOT_FOUND, dừng
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR

Step 2 — Nạp bản dịch kèm nguồn và mục lục (repository)
- Trigger: Step 1 xong
- Pre-condition: văn bản tồn tại
- Call: SELECT key, lang, variant, translator, source_name, source_url, license, license_url, license_note, note, line_count, sections FROM sutra_translations WHERE sutra_id = $1 ORDER BY ordinal
    - Success → gắn vào văn bản, sang Step 3
    - Failure (lỗi DB / giải mã sections) → error bọc %w → 500 INTERNAL_ERROR

Step 3 — Nạp liên kết Phật/Bồ tát (repository)
- Trigger: Step 2 xong
- Pre-condition: —
- Call: cùng câu JOIN `deities` như Step 6 của danh sách, cho một sutra_id
    - Success → gắn vào văn bản, sang Step 4
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR
    - Transaction boundary: không có — ba câu lệnh độc lập

Step 4 — Dựng response (handler)
- Trigger: usecase trả về
- Pre-condition: văn bản đủ bản dịch và liên kết
- Call: map sang sutraDetailResponse (v2.12: thêm sáu trường của AC36 và license_url/license_note mỗi bản dịch; han_translation_year nil → null); sections nil → []
    - Success → 200 SutraDetail
    - Failure → không có nhánh lỗi mới
```

## Step-by-step Flow — Đọc nội dung theo trang dòng (`GET /v1/sutras/{id}/lines`)

```
Step 1 — Đọc và kiểm tra tham số (handler)
- Trigger: request đến route công khai
- Pre-condition: —
- Call: lấy translation (bắt buộc), page (rỗng → 1), page_size (rỗng/vắng → mặc định lấy từ config); strconv.Atoi cho page và page_size
    - Success → Step 2
    - Failure (translation vắng hoặc rỗng) → ErrInvalidTranslation → 400 INVALID_TRANSLATION, details.field = "translation"
    - Failure (page không phải số nguyên, tràn int) → ErrInvalidPage → 400 INVALID_PAGE
    - Failure (page_size không phải số nguyên) → ErrInvalidPageSize → 400 INVALID_PAGE_SIZE

Step 2 — Kiểm tra khoảng và kẹp page_size (usecase)
- Trigger: handler gọi SutraUsecase.Lines(ctx, id, key, page, pageSize)
- Pre-condition: page, page_size là số nguyên
- Call: page < 1 hoặc > math.MaxInt32 → lỗi; page_size < 1 → lỗi; page_size > sutra.lines_max_page_size → kẹp xuống giá trị đó; vắng → sutra.lines_page_size
    - Success → Step 3 (fromN = (page-1) × page_size, toN = page × page_size, tính bằng int64)
    - Failure → ErrInvalidPage / ErrInvalidPageSize → 400, dừng

Step 3 — Kiểm tra kinh và bản dịch tồn tại (repository)
- Trigger: Step 2 xong
- Pre-condition: tham số hợp lệ
- Call: SutraRepository.FindTranslation → (a) SELECT 1 FROM sutras WHERE id = $1; (b) SELECT id, line_count FROM sutra_translations WHERE sutra_id = $1 AND key = $2
    - Success → translation_id và line_count, sang Step 4
    - Failure ((a) không có hàng) → ErrSutraNotFound → 404 SUTRA_NOT_FOUND, dừng
    - Failure ((b) không có hàng) → ErrSutraTranslationNotFound → 404 TRANSLATION_NOT_FOUND, dừng
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR

Step 4 — Lấy trang dòng (repository)
- Trigger: Step 3 xong
- Pre-condition: có translation_id
- Call: SutraRepository.ListLines → SELECT n, text, section FROM sutra_lines WHERE translation_id = $1 AND n > $2 AND n <= $3 ORDER BY n ASC
    - Success → các dòng (rỗng nếu fromN ≥ line_count), sang Step 5
    - Failure (lỗi DB / scan) → error bọc %w → 500 INTERNAL_ERROR
    - Transaction boundary: không có — Step 3 và Step 4 là các câu lệnh độc lập, không chung snapshot

Step 5 — Dựng trang và trả response (usecase + handler)
- Trigger: repository trả về
- Pre-condition: items, line_count
- Call: items nil → []; total = line_count; total_pages = (total + page_size - 1) / page_size; section rỗng → null
    - Success → 200 {translation, items, page, page_size, total, total_pages}
    - Failure → không có nhánh lỗi mới
```

## Step-by-step Flow — Kinh theo Phật/Bồ tát (`GET /v1/deities/{id}/sutras`)

```
Step 1 — Đọc page và bộ lọc vai (handler)
- Trigger: request đến route công khai
- Pre-condition: —
- Call: lấy {id}, page (rỗng → 1, ngược lại strconv.Atoi) và role (tuỳ chọn; có mặt nhưng rỗng vẫn tính là có mặt)
    - Success → Step 2
    - Failure (page không phải số nguyên, tràn int) → ErrInvalidPage → 400 INVALID_PAGE

Step 2 — Kiểm tra page và role (usecase)
- Trigger: handler gọi SutraUsecase.ListByDeity(ctx, deityID, role, page)
- Pre-condition: page là số nguyên
- Call: page ∈ [1, MaxInt32]; role vắng hoặc ∈ {SPEAKER, SUBJECT, MENTIONED}
    - Success → Step 3
    - Failure (page) → ErrInvalidPage → 400 INVALID_PAGE
    - Failure (role) → ErrInvalidSutraFilter{Field: "role"} → 400 INVALID_FILTER

Step 3 — Kiểm tra Phật/Bồ tát tồn tại (repository của feature deity)
- Trigger: Step 2 xong
- Pre-condition: —
- Call: DeityRepository.FindByID(ctx, deityID) — có sẵn trong feature `deity`, không sửa
    - Success → Step 4
    - Failure (không có) → ErrDeityNotFound → 404 DEITY_NOT_FOUND, dừng
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR

Step 4 — Đếm văn bản liên kết (repository)
- Trigger: Step 3 xong
- Pre-condition: deity tồn tại
- Call: SutraRepository.CountByDeity → SELECT count(*) FROM sutra_deities WHERE deity_id = $1 [AND role = $2]
    - Success → total, sang Step 5
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR

Step 5 — Lấy trang văn bản liên kết (repository)
- Trigger: Step 4 xong
- Pre-condition: có total
- Call: SutraRepository.ListByDeity → SELECT <cột sutras>, sd.role FROM sutras s JOIN sutra_deities sd ON sd.sutra_id = s.id WHERE sd.deity_id = $1 [AND sd.role = $2] ORDER BY role_rank, s.sort_order, s.id OFFSET $3 LIMIT $4
    - Success → items (có thể rỗng), sang Step 6
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR

Step 6 — Nạp bản dịch (repository)
- Trigger: Step 5 trả về ít nhất một văn bản (rỗng → bỏ qua Step 6–7, sang Step 8)
- Pre-condition: có tập id của trang
- Call: cùng câu như Step 5 của Danh sách và bộ lọc
    - Success → sang Step 7
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR

Step 7 — Nạp liên kết Phật/Bồ tát của các văn bản đó (repository)
- Trigger: Step 6 xong
- Pre-condition: có tập id của trang
- Call: cùng câu JOIN như Step 6 của Danh sách và bộ lọc — trả đủ mọi vị của mỗi văn bản, không chỉ vị đang tra
    - Success → sang Step 8
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR
    - Transaction boundary: không có — Step 4 đến 7 độc lập, không chung snapshot

Step 8 — Dựng trang và trả response (usecase + handler)
- Trigger: usecase trả về
- Pre-condition: items, total
- Call: items nil → []; total_pages = (total + page_size - 1) / page_size; map sang sutraByDeityItemResponse = SutraSummary + role
    - Success → 200 {items, page, page_size, total, total_pages}
    - Failure → không có nhánh lỗi mới
```

## Deliberately stubbed for now (if applicable)

Không có gì bị stub: mọi method của `SutraRepository` có SQL thật và đã chạy trên Postgres. Phần không làm mà có chủ đích nằm ở "Out of scope" của `requirements.md` (lệnh seed thủ công, đồng bộ delta, `audio_url`, tìm kiếm toàn văn, bookmark).

## Known gaps carried by this feature

- **Nhãn tôn giáo và bản Chú Đại Bi chưa có người duyệt**: `daily_chant`, `purposes` do em (Claude) gán theo bảng đề xuất và bản Chú Đại Bi giữ bản phatgiao.org.vn 84 câu (đã chốt 2026-10-03: không có người am hiểu duyệt); `review_status = UNREVIEWED` được giữ trong dữ liệu để đổi khi có người duyệt; liên kết `deities` do người thu thập dữ liệu gán, không lấy từ nguồn.
- **Giấy phép chưa rõ**: phần lớn bản tiếng Việt `license = unknown`; bản Kim Cang Thừa chỉ cho phép phi thương mại (CC BY-NC 4.0). Người dùng tạm không xử lý giấy phép vì ứng dụng miễn phí (đã chốt 2026-10-03): dữ liệu được `go:embed` nên mọi văn bản đi cùng binary và API trả hết; các trường `license`, `source_name`, `source_url` vẫn được lưu và trả trong chi tiết. Cần xem lại nếu ứng dụng đổi sang thu phí.
- **Kích thước binary và thời gian khởi động** (đã đo 2026-10-03): `go:embed` ~10,4 MB JSON làm binary lớn thêm 10,25 MB (70,91 MB so với 60,66 MB, +17%); lần khởi động đầu parse và chèn ~64.000 dòng trong một transaction nên mất median 1,82 s tới khi nhận request đầu tiên (1,73–2,93 s), các lần sau 0,13 s. Đo trên Postgres cục bộ; chưa đo trên DB từ xa — đó là lý do `sutraSeedTimeout` có ngân sách riêng.
- **Khởi động lần đầu phụ thuộc `deities`**: nếu `deities` chưa được seed thì service abort (đã chốt, xem Step 3); thứ tự triển khai là migration → `cmd/seeddeities` → service.
- **Dữ liệu tách trường và `summary` chưa tới DB có sẵn (v2.12)**: migration `0020` chỉ thêm cột; DB đã seed vẫn giữ `reference`/`license` cũ và `summary = ''` tới khi người dùng quyết định cách áp (xem marker ở `requirements.md`). `summary` do em soạn, chưa duyệt (A-011).
- **Sửa JSON không cập nhật DB có sẵn**: seed chỉ chạy khi bảng `sutras` rỗng và không còn lệnh seed thủ công; nội dung chỉ đổi được bằng SQL tay như `article` (đã chốt 2026-10-03, lệnh reseed và backoffice hoãn).
- **Bảng bị xoá sạch thì được nạp lại**: điều kiện "không hồi sinh văn bản đã xoá" chỉ đúng khi còn ít nhất một dòng; xoá toàn bộ `sutras` rồi khởi động lại sẽ nạp lại đủ 153 văn bản.
- **Hai instance khởi động cùng lúc**: cả hai có thể cùng thấy bảng rỗng và cùng parse seed; transaction thứ hai chờ rồi gặp `ON CONFLICT DO NOTHING` nên bỏ qua toàn bộ, an toàn nhưng tốn công parse.
- **Spec `content` cũ đã bị xoá**: `docs/specs/features/content/requirements.md` (US-005/006 mô tả `GET /v1/content/sutras` với `updated_since`, `slug`, `audio_url`, `version`) đã xoá ngày 2026-10-03; đồng bộ delta, `version`, offline-first và `audio_url` chỉ còn trong lịch sử git và chưa có spec nào đảm nhận. Spec `practice` từng tham chiếu `audio_url` đã bị xoá vì cũ.
- **Các câu truy vấn không chung snapshot**: danh sách, chi tiết, trang dòng và tra ngược đọc bằng nhiều câu lệnh độc lập; dữ liệu chỉ đổi bằng SQL tay nên rủi ro thấp.
- **`page_size` của trang dòng do client chọn**: đổi `page_size` giữa hai lần gọi làm các trang dịch chỗ; client phải giữ một `page_size` cho cả lần đọc, hoặc tính lại trang từ `n`.
- **Giới hạn của E2E hiện có** (`test-coverage-report.md` liệt kê đủ): E2E không thể gây lỗi hạ tầng trên route đọc nên `500` không lộ nguyên nhân chỉ được chứng minh ở tầng handler; dữ liệu nhúng hợp lệ nên việc binary chính abort khi dữ liệu sai chỉ được chứng minh qua parser và `cmd/seedsutras -dir`; dòng log `seeded sutras on first run` và việc hai instance khởi động đồng thời chưa có test tự động. Test repository dùng chung helper `connectTestDB` (chạy `TRUNCATE` trên DB dev) giống các feature khác.
- **`sutraSeedTimeout` đang là hằng số trong `cmd/main.go` (2 phút), không ở `config/`**: `connectCtx` của `main.go` chỉ có 10 giây và dùng chung cho nạp cấu hình, kết nối DB và seed `article`, trong khi seed sutra (~64.000 dòng trong một transaction) mất khoảng 1,2 giây trên Postgres cục bộ nhưng có thể lâu hơn trên DB từ xa; nên nó có ngân sách riêng. Đây là giá trị có thể thay đổi theo môi trường, nên có thể thuộc `config/` — chưa hỏi người dùng. `cmd/seedsutras` có hằng `seedTimeout` cùng giá trị.
- **Dev local cần `cmd/seeddeities` trước lần chạy đầu — đã có target Makefile**: `make run` chạy `ensure-deities` (chỉ seed `deities` khi bảng rỗng, nên không ghi đè chỉnh sửa tay); `make seed-sutras` và `make seed` nạp thư viện kinh bằng tay (không làm gì nếu bảng `sutras` đã có dòng); `make seed-deities` upsert và ghi đè chỉnh sửa tay, chỉ chạy khi chủ ý. Đã ghi vào `docs/development/local-stack.md`, `migrations/README.md`.
- **Dựng khoá ngoại nên `TRUNCATE deities` cũng xoá `sutra_deities`**: `TRUNCATE ... deities CASCADE` (như `truncateAll` trong test, hoặc SQL tay) xoá luôn các liên kết sutra–deity, và vì `sutras` vẫn còn dòng nên seed không bù lại; phải seed lại cả hai.
