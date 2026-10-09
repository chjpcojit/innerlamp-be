# Tasks — sutra

Version: 2.13
Last updated: 2026-10-07
Status: v2.12 (2026-10-07) thêm Phase 6 (tách `reference`/`license` + `summary`): mọi mục đã `[x]`, gồm "Áp lên DB đã seed" (kiểm 2026-10-08). Phase 1–5 xong và đã kiểm chứng (2026-10-03): `go build`, `go vet`, `go test -short` trên toàn repo; test repository trên Postgres thật; bộ E2E đầy đủ gồm 7 hàm test mới cho sutra (đã kiểm bằng 6 đột biến); số đo khởi động, binary và response; ERD, `migrations/README.md`, tài liệu dev local. Không còn mục `[ ]` nào; mục soát `durationMinutes` được đóng ngày 2026-10-03 theo quyết định giữ nguyên giá trị đã sinh. Không còn marker `⚠️ Not confirmed`.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

Mọi mục dưới đây trace về một hàng của bảng "Layers & components" hoặc một Step của "Step-by-step Flow" trong `design.md`. Một mục chỉ được `[x]` khi đã kiểm chứng: mã đã viết, `go build`/`go vet` sạch và test của mục đó pass (test repository chạy trên Postgres thật, không phải mock). Bằng chứng (số đo, tên test) được ghi ngay trong từng mục; báo cáo độ phủ theo tiêu chí ở `test-coverage-report.md`.

## Phase 0 — Dữ liệu (`assets/data/sutras/`, bảng "Dữ liệu cần bổ sung")

- [x] Thêm `translations[].key` cho mọi bản dịch của 153 văn bản (`<lang>-<variant>`, hậu tố `-2`, `-3` khi trùng; hiện chỉ `tam-kinh` trùng) — đã thêm 2026-10-03, kiểm tra 153 file: `key` duy nhất trong từng văn bản (`tam-kinh` có `vi-meaning` và `vi-meaning-2`)
- [x] Thêm `dailyChant` cho 153 văn bản theo bảng đề xuất trong `design.md` (em gán, không có người duyệt; giữ `review_status = UNREVIEWED`) — đã thêm 2026-10-03, `true` ở 34 văn bản (Nhật tụng, Sám hối, Thập Chú, Chú Đại Bi, Tâm Kinh, A Di Đà, Phổ Môn, Bát Đại Nhân Giác, Kinh Cứu Khổ, Năm Giới, Kinh Từ Bi, Kinh Hạnh Phúc)
- [x] Thêm `purposes` cho 153 văn bản theo bảng đề xuất trong `design.md` (em gán; giá trị thuộc bảng `purposes` đã chốt) — đã thêm 2026-10-03, kiểm tra mọi giá trị thuộc bảng; lệch bảng ở một số chỗ theo tên bài (liệt kê trong `design.md`)
- [x] Thêm `durationMinutes` cho 153 văn bản: script gợi ý giá trị ban đầu (`ceil(âm tiết / 80)`, tối thiểu 1; `null` khi không có bản `vi`) rồi làm tròn cho dễ đọc — đã ghi 2026-10-03; 10 văn bản Kim Cang Thừa là `null`, các phân bố: ≤5 phút 63, ≤15 phút 27, ≤30 phút 13, ≤60 phút 21, >60 phút 19
- [x] Soát và chỉnh tay `durationMinutes` theo ước tính của mình (nghĩ bài cần 30 phút thì ghi 30); ca nghi lệch: `duoc-su` (80 phút, bản dịch dài hơn bản tụng) và các bộ rất dài (Pháp Hoa, Lăng Nghiêm 14 giờ, Kim Quang Minh 15 giờ) — đóng 2026-10-03: người dùng quyết định **giữ nguyên giá trị đã sinh**, không chỉnh tay; độ lệch đã biết được chấp nhận (TD-007, A-005); dữ liệu không đổi
- [x] Cập nhật `assets/data/sutras/index.json` (đưa `dailyChant`, `purposes`, `durationMinutes`, `key` vào metadata) và `assets/data/sutras/README.md` (mô tả bốn trường mới) — đã làm 2026-10-03; `index.json` khớp từng file (153/153)

## Phase 1 — Domain

- [x] `Sutra`, `SutraTranslation`, `SutraSection`, `SutraLine`, `SutraDeity`, `DeitySutra`, `SutraPage`, `SutraLinesPage`, `SutraFilter`, `SutraSeed` và các hằng enum trong `services/appService/domain/sutra.go` (không tag; `DurationMinutes *int`)
- [x] Sentinel error `ErrSutraNotFound`, `ErrSutraTranslationNotFound`, `ErrInvalidSutraFilter` (mang tên field), `ErrInvalidPageSize`, `ErrInvalidTranslation` trong `domain/sutra.go`; tái dùng `ErrInvalidPage` (`domain/temple.go`) và `ErrDeityNotFound`
- [x] Interface `SutraRepository` (`Count`, `List`, `FindByID`, `FindTranslation`, `ListLines`, `CountByDeity`, `ListByDeity`, `MissingDeities`, `InsertAll`; thứ tự `sort_order ASC, id ASC` nằm trong hợp đồng) trong `domain/sutra.go`
- [x] `domain/mock_sutra_repository.go` cho test usecase

## Phase 2 — Usecase

- [x] `usecase/sutra_usecase.go`: constructor nhận `SutraRepository`, `DeityRepository` và `config.SutraConfig`
- [x] `List` — kiểm tra `page` ∈ [1, MaxInt32] và các bộ lọc (`tradition`/`kind`/`purpose` thuộc tập cho phép, `daily` ∈ {true,false}, `max_minutes` ≥ 1, không tham số rỗng) → `ErrInvalidPage` / `ErrInvalidSutraFilter{Field}`; offset bằng int64; `total_pages` (Step 2, 7 của Danh sách)
- [x] `Get` — gọi `FindByID` và trả `ErrSutraNotFound` (Step 1–3 của Chi tiết)
- [x] `Lines` — kiểm tra `page`/`page_size`, kẹp `page_size` theo `lines_max_page_size`, mặc định `lines_page_size`; đổi sang `fromN`/`toN`; tính `total_pages` (Step 2, 5 của Đọc nội dung theo trang dòng)
- [x] `ListByDeity` — kiểm tra `page` và `role`, gọi `DeityRepository.FindByID`, rồi `CountByDeity` và `ListByDeity`; `total_pages` (Step 2–5, 8 của Kinh theo Phật/Bồ tát)
- [x] `SeedIfEmpty(ctx, load)` — `Count` → `load()` → `MissingDeities` → `InsertAll`; không gọi `load` khi bảng đã có dòng (Step 1–4 của Nạp dữ liệu)
- [x] Ánh xạ lỗi repository sang sentinel error (tương thích `errors.Is`)
- [x] Unit test `usecase/sutra_usecase_test.go`: table-driven, `testify/assert` + `testify/require`, mock viết tay — mỗi nhánh lỗi của năm flow, kẹp `page_size`, `total_pages` làm tròn lên, `page > total_pages`, `SeedIfEmpty` bỏ qua khi bảng đã có dòng và abort khi thiếu deity

## Phase 3 — Repository, seed và nhúng dữ liệu

- [x] `migrations/0013_sutras.sql` — bốn bảng `sutras`, `sutra_translations`, `sutra_lines`, `sutra_deities`, các `CHECK`, khoá ngoại (cascade trừ `deities`) và index theo "Mô hình dữ liệu"; idempotent (đã áp lại nhiều lần trong test mà không lỗi); `reference` là `TEXT` chứ không phải `VARCHAR(128)` vì `lang-nghiem` có `reference` dài 135 ký tự
- [x] `repository/postgres_sutra_repository.go` — `Count`/`List` (dựng `WHERE` từ bộ lọc, `= ANY($1)` cho bản dịch và liên kết), `FindByID`, `FindTranslation` (hai truy vấn: sutra rồi bản dịch), `ListLines` (khoảng `n` trên khoá `(translation_id, n)`), `CountByDeity`, `ListByDeity`
- [x] Cột liệt kê tường minh, `context` xuyên suốt, bọc lỗi `%w`; `NULL` ↔ `nil` cho `duration_minutes`/`section`; `pq.Array` cho `purposes`
- [x] `MissingDeities` và `InsertAll` — MỘT transaction, `ON CONFLICT (id) DO NOTHING RETURNING id` cho `sutras` (không trả hàng → bỏ qua phần con), chèn dòng theo lô nhiều hàng (Step 3–4 của Nạp dữ liệu)
- [x] `repository/sutra_seed.go` — `ParseSutraSeed(fsys fs.FS)` kiểm toàn bộ theo AC29, gán `SortOrder`, tính `sections` (Step 2)
- [x] `assets/embed.go` — thêm `SutrasFS embed.FS` với `//go:embed data/sutras/*.json`
- [x] `cmd/seedsutras/main.go` — nạp cấu hình và kết nối DB như `cmd/seeddeities`, dựng repository + usecase, gọi `SeedIfEmpty` với dữ liệu nhúng (hoặc `os.DirFS` khi có `-dir`); bảng đã có dòng thì log "sutras already seeded, skipping" và thoát 0; lỗi thì thoát khác 0 (Step 1–5 của Nạp dữ liệu, AC35); test logic chọn nguồn và "không đọc dữ liệu khi đã seed" ở `cmd/seedsutras/main_test.go`; đã chạy thật: DB đã seed → thoát 0, DB trống → nạp 153/391/64.009/151, thiếu deity → thoát 1 kèm danh sách id
- [x] Unit test `repository/sutra_seed_test.go` bằng `fstest.MapFS` — từng điều kiện của AC29 (file thiếu, id lặp/khác tên file, enum sai, `n` có hở, `key` thiếu/lặp, deity lặp, `durationMinutes` = 0) và dữ liệu hợp lệ cho `SortOrder`/`sections`
- [x] Test repository `repository/postgres_sutra_repository_test.go` (cần Postgres; bỏ qua với `-short`; đã chạy trên Postgres 16 thật ngày 2026-10-03, 12 test pass, kể cả nạp cả thư viện thật) — `InsertAll` một transaction, rollback khi lỗi, `ON CONFLICT` bỏ qua văn bản đã có, khoảng `n` của `ListLines`, bộ lọc và `total`

## Phase 4 — Delivery

- [x] DTO (`sutraSummaryResponse`, `sutraDetailResponse`, `sutraTranslationResponse`, `sutraLineResponse`, `sutraLinesResponse`, `sutraDeityResponse`, `sutraListResponse`, `sutraByDeityItemResponse`, `sutraByDeityListResponse`) trong `delivery/http/sutra_handler.go` (JSON tag chỉ ở đây)
- [x] Handler `List`, `Get`, `Lines`, `ListByDeity`: đọc query → gọi usecase → map entity sang DTO (`nonNil` cho mảng; `null` cho `duration_minutes`, `section`)
- [x] Ánh xạ lỗi trong `delivery/http/errors.go`: `ErrSutraNotFound` → `404 SUTRA_NOT_FOUND`, `ErrSutraTranslationNotFound` → `404 TRANSLATION_NOT_FOUND`, `ErrInvalidSutraFilter` → `400 INVALID_FILTER`, `ErrInvalidPageSize` → `400 INVALID_PAGE_SIZE`, `ErrInvalidTranslation` → `400 INVALID_TRANSLATION`
- [x] `SutraConfig{PageSize, LinesPageSize, LinesMaxPageSize}` và block `sutra:` trong `config/config.go`, `config.dev.yaml`; `SutraConfig.Validate()` báo lỗi khi giá trị < 1 hoặc `lines_page_size` > `lines_max_page_size`, và `cmd/main.go` gọi nó ngay sau `Load` (không đặt trong `Load` vì `Load` hiện không kiểm tra giá trị nào và 4 test cũ của nó dùng YAML không có block catalogue); test: `config/config_test.go` (`TestSutraConfig_Validate`, `TestLoad_DevYAMLSutraConfig`)
- [x] `cmd/main.go` — gọi `SutraUsecase.SeedIfEmpty` sau khi kết nối DB và sau khi `deities` sẵn sàng (lỗi → `log.Fatalf`, thành công → log `seeded sutras on first run`); đăng ký bốn route (`GET /v1/sutras`, `/v1/sutras/{id}`, `/v1/sutras/{id}/lines`, `/v1/deities/{id}/sutras`); xác nhận đã chạy thật rằng không route nào bị bọc `RequireAuth` và `/v1/deities/{id}` không đổi hành vi
- [x] Cập nhật `docs/specs/api/openapi.yaml` và `web/docs/public/openapi.yaml` trong cùng thay đổi với handler
- [x] Test handler `delivery/http/sutra_handler_test.go` (cùng `mock_sutra_usecase_test.go`): thành công, từng nhánh lỗi (400/404/405/500), mảng rỗng là `[]` không `null`, `duration_minutes` và `section` là `null` khi phù hợp, `page > total_pages` trả `items: []`
- [x] `services/appService/e2e/harness.go` — seed `deities` (từ `assets/data/phat_botat.json`) sau khi áp migration, vì service không khởi động lần đầu nếu thiếu `deities`; và khi binary không lên thì in stderr của nó thay vì chỉ báo hết giờ. Không có thay đổi này thì 3 test E2E của register/login (không liên quan tới kinh) hỏng; sau khi sửa, cả bộ E2E hiện có pass lại (đã chạy thật trên testcontainers Postgres)

## Phase 5 — Integration và tài liệu

- [x] Chạy thật trên Postgres: áp migration, `cmd/seeddeities`, khởi động service lần đầu, ghi số liệu thật, khởi động lần hai, rồi chạy `cmd/seedsutras` trên DB đã có dữ liệu — đã làm ngày 2026-10-03 trên **một Postgres 16-alpine tạm (cùng image với `docker-compose.yml`, DB mới, đã dọn sau khi đo), không phải stack dev của người dùng nên không đụng dữ liệu của họ**: 153/391/64.009/151 đúng số; khởi động lần hai không nạp lại (không có dòng log seed, `total` vẫn 153); `cmd/seedsutras` trên DB đã seed thoát 0 và không ghi, trên DB trống nạp đủ, thiếu `deities` thoát 1 kèm danh sách id; thiếu block `sutra:` trong config thì service thoát 1. Cũng được test tự động ở `e2e/sutra_e2e_test.go`
- [x] Đo thời gian khởi động lần đầu (seed ~64.000 dòng) và kích thước binary trước/sau khi nhúng dữ liệu — đo ngày 2026-10-03 trên Postgres cục bộ (container), 5 lần mỗi loại: từ lúc chạy process tới `200` đầu tiên của `/v1/sutras` mất **1,73–2,93 s (median 1,82 s) ở lần chạy đầu** (seed 153 sutras / 64.009 dòng) và **0,12–0,13 s ở các lần sau**; binary **70,91 MB** khi nhúng thư viện so với **60,66 MB** khi cùng code chỉ nhúng `index.json` — chênh **10,25 MB (+17%)**, bằng đúng dung lượng các file JSON được nhúng (10,40 MB trừ `index.json` 0,24 MB)
- [x] Đo thời gian và kích thước response của `GET /v1/sutras` (một trang) và của một trang dòng của bản `phap-hoa/vi` (6.967 dòng, trang 35 với 200 dòng/trang) — đo ngày 2026-10-03 trên loopback, mỗi route 40 request sau 5 request làm nóng, kết nối mới cho mỗi request (JSON, chưa nén; gzip là ước tính tính cục bộ vì server không tự nén): danh sách trang 1 (20 mục) 2,49 ms (p95 2,74), 28,4 KB, gzip 2,0 KB; chi tiết `tam-kinh` 1,73 ms, 3,9 KB; chi tiết `phap-hoa` (27 phẩm) 2,08 ms, 7,5 KB; dòng `phap-hoa/vi` trang 1 (200 dòng) 1,81 ms, 21,2 KB, gzip 5,3 KB; `page_size=500` (mức kẹp) 1,94 ms, 49,6 KB, gzip 10,7 KB; trang 35 (167 dòng) 1,85 ms, 36,6 KB; tra ngược `buddha_shakyamuni` (81 mục, trang 1) 3,47 ms, 29,0 KB. Chưa đo dưới tải đồng thời hay qua mạng thật
- [x] Test E2E trong `services/appService/e2e/` (binary thật + Postgres ephemeral) cho bốn route và seed lần đầu — `e2e/sutra_e2e_test.go`, 7 hàm test: `TestSutraAPI_E2E` (bốn route, 13 nhóm tiêu chí, kỳ vọng tính từ `index.json` chứ không gõ số cứng), `TestSutraSeed_FirstRunFillsEveryTable_E2E`, `TestSutraSeed_SecondStartLeavesTheDatabaseAlone_E2E`, `TestSutraSeed_RefusesToStartWithoutDeities_E2E`, `TestSutraSeed_FailedInsertRollsBackEverything_E2E`, `TestSutraConfig_InvalidValuesRefuseToStart_E2E`, `TestSeedSutrasCommand_E2E`; harness `e2e/harness.go` mở rộng (Postgres không seed `deities`, `openDB`/`countRows`/`execSQL`, `runUntilExit`, build thêm `seedsutras`). Đã kiểm chứng bằng 6 đột biến cố ý trên code thật, mỗi cái đều làm test đỏ (xem `test-coverage-report.md`)
- [x] `go build ./...`, `go vet ./...`, `go test -short ./...` sạch — chạy ngày 2026-10-03 trên **toàn repo** (`./...`), cả ba thoát 0; bộ E2E đầy đủ (cần Docker) cũng pass
- [x] Cập nhật ERD `docs/specs/db/erd.mmd` và `erd.svg` (thêm `sutras`, `sutra_translations`, `sutra_lines`, `sutra_deities` và 4 quan hệ; SVG tạo bằng `@mermaid-js/mermaid-cli` — công cụ này tái tạo bản cũ gần như y hệt bản `erd.svg` có sẵn, 628.408 so với 628.442 byte, cùng viewBox và font — nên cùng toolchain với bản gốc), `migrations/README.md` (liệt kê `0013` và 7 bất biến mới) và `docs/development/local-stack.md` (khung cảnh báo "seed `deities` trước lần chạy đầu" kèm dòng xử lý sự cố)
- [x] `Makefile`: `ensure-deities` (nối vào `run`), `seed-deities`, `seed-sutras`, `seed` và cập nhật `docs/development/local-stack.md` — đã chạy thử trên compose project riêng (DB trống → seed deities rồi sutras; chạy lại không đổi gì)
- [x] Tạo OKF `docs/knowledge/features/sutra.md` (draft), cập nhật `index.md`, `log.md`; ghi nợ kỹ thuật `docs/tech-debt/sutra/TD-001…010` và giả định `docs/assumptions/sutra/A-001…008`
- [x] Ghi trong changelog đoạn OKF `docs/knowledge/features/sutra.md` cần được `il-okf-writer` tạo — đã ghi ở `changelog.md` v2.8; hiện đã đủ điều kiện (có E2E chạy binary thật), chưa tạo

## Phase 6 — Tách `reference` / `license` và thêm `summary` (v2.12, AC36–AC40)

- [x] Data: tách `reference` và `license`, thêm `originalTitle`, `author`, `hanTranslator`, `hanTranslationYear`, `referenceNote`, `summary`, `licenseUrl`, `licenseNote` cho 153 file + `index.json` (bảng "Dữ liệu cần bổ sung" của `design.md`); cập nhật `assets/data/sutras/README.md` — đã làm 2026-10-07: 153 file + `index.json`; `TestParseSutraSeed_ShippedDataIsSplit` (mọi văn bản có `summary`, `reference` không còn `;`/`dịch Hán`, mọi `license` thuộc tập đóng, 4 văn bản có năm dịch Hán) và `TestSutraRepository_ShippedData` pass
- [x] `migrations/0020_sutra_reference_license_summary.sql` — 6 cột mới của `sutras`, 2 cột mới của `sutra_translations`; idempotent (áp lại không lỗi); số `0020` kiểm bằng `ls migrations/` ngay trước khi tạo file — áp trong test Postgres thật (`applyMigration` đã thêm file), áp lại không lỗi
- [x] `domain/sutra.go` — các trường mới của `Sutra` / `SutraTranslation`, hằng mã giấy phép + `ValidLicense` — build/vet sạch
- [x] `repository/sutra_seed.go` — đọc các trường mới (vẫn `DisallowUnknownFields`), kiểm `license` thuộc tập đóng và `hanTranslationYear` (AC40); test `sutra_seed_test.go` cho từng nhánh lỗi mới và giá trị mặc định `""` — `TestParseSutraSeed_InvalidSutra` thêm 6 ca (license thiếu/ngoài tập, năm 0/âm/chuỗi/lẻ) và `TestParseSutraSeed_OptionalFieldsDefaultToEmpty`
- [x] `repository/postgres_sutra_repository.go` — `InsertAll` ghi cột mới; `FindByID` đọc thêm sáu cột; `loadTranslations` ở chế độ chi tiết đọc `license_url`, `license_note`; test repository trên Postgres thật (chèn rồi đọc lại đủ các trường, `han_translation_year` NULL ↔ `nil`; danh sách không mang sáu trường) — `TestSutraRepository_InsertAllAndFindByID` đọc lại đủ trường, `List` không mang sáu trường; chạy trên Postgres thật
- [x] `delivery/http/sutra_handler.go` — DTO chi tiết thêm sáu trường và `license_url`/`license_note`, `han_translation_year` là `null` khi không có; test handler cho `null`/`""` và rằng danh sách không có các trường này — `TestSutraHandler_Get` (năm `null`, chuỗi rỗng không phải `null`) và `TestSutraHandler_List_Success` (danh sách không có các trường chi tiết)
- [x] `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml` — schema `SutraDetail`, `TranslationDetail`, mô tả `reference` và `license` — hai file giống nhau từng byte, đọc lại bằng YAML parser (Ruby) thấy đủ trường
- [x] `docs/specs/db/erd.mmd`, `erd.svg`, `migrations/README.md` — cột mới và bất biến mới của `0020` — `erd.svg` sinh lại bằng `@mermaid-js/mermaid-cli` cùng toolchain trước đó
- [x] `e2e/sutra_e2e_test.go` — chi tiết `a-di-da` trả `han_translator`, `han_translation_year`, `summary` khác rỗng; danh sách không có `summary` — subtest `AC36-AC38` chạy binary thật + Postgres ephemeral, toàn bộ nhóm test Sutra pass
- [x] Áp lên DB đã seed — đã kiểm 2026-10-08 trên Postgres dev: 153/153 `sutras` khớp JSON (`summary`, `original_title`, `reference`), không còn `reference` kiểu cũ, `license` của 411 bản dịch đều thuộc tập đóng; `GET /v1/sutras/a-di-da` trả `summary`, `han_translator`, `han_translation_year`=402, danh sách không có `summary` (AC36–AC38)
- [x] v2.13 (AC41): `summary` lên thẻ danh sách và by-deity — `sutraColumns` đọc `s.summary`, `sutraHeadResponse` có `summary` (detail không còn trường riêng), OpenAPI (hai file) chuyển `summary` sang `SutraSummary`; test repository/handler/E2E đổi từ "danh sách không có summary" sang "có summary"; `go build`, `go vet`, `go test -short`, test repository và E2E nhóm Sutra pass (2026-10-08)

## Deferred (separate feature, not blocking sutra)

- Cập nhật nội dung kinh sau lần seed đầu bằng lệnh reseed hoặc backoffice CRUD — đã chốt hoãn, hiện dùng SQL tay như `article`
- Đồng bộ delta (`updated_since`), `audio_url`, `version` — từng có trong spec `content` cũ (đã xoá 2026-10-03), nay chưa có spec nào
- Bookmark, kinh yêu thích, lịch sử tụng, đếm biến tụng — module `journal`
- Tìm kiếm toàn văn trong nội dung kinh
