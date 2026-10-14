# Changelog — administrative-units Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first. `SRS.md` (từ `il-ba`) có version riêng (hiện v0.5) và là đầu vào của bộ spec này.

## v1.10 — 2026-10-11

Chỉ là đồng bộ tài liệu; không đổi hành vi, hợp đồng API hay flow. Phát hiện khi rà soát SRS / spec / `openapi.yaml` / `web/docs` của phần metadata tỉnh thành.

- `design.md` (1.8 → 1.10): sửa mục "Deliberately stubbed" còn ghi "chưa có code" và "`?q=` chưa được hỏi" — `?q=` đã làm ở v1.8 (AC 25–26, FR-011); tiêu đề Schema bỏ "dự kiến" vì migration `0026` đã áp.
- `SRS.md` (0.4 → 0.5): cập nhật Version/Last updated/Status cho khớp (đã implement và verified; còn một mục mở ở Appendix A). Nội dung yêu cầu không đổi.
- `docs/specs/api/openapi.yaml` ↔ `web/docs/public/openapi.yaml`: giống hệt nhau; chín route khớp `cmd/main.go`.
- `web/docs`: thêm trang `guides/catalogues/administrative-units.md`, mục sidebar, thẻ ở `index.mdx`, dòng changelog; bổ sung `X-API-Key`, `ADMINISTRATIVE_UNIT_NOT_FOUND`, `INVALID_FILTER` (`q` của xã) và `RATE_LIMITED` của nhóm route này vào `authentication.md` / `error-handling.md`.

## v1.9 — 2026-10-11

Chỉ là tài liệu và bảo vệ test; không đổi hành vi, hợp đồng API hay flow.

- `requirements.md`, `design.md`, `flow.md`, `tasks.md`: không đổi (đã kiểm). Quy ước cấu hình/bí mật mới của repo (`AGENTS.md` §5.3) được feature này tuân thủ sẵn: `administrative_units.rate_limit` và `cache_max_age` là cấu hình không nhạy cảm nằm ở `config.yaml`; feature không thêm bí mật nào.
- Thêm `docs/assumptions/administrative-units/A-001..A-003` (thời hạn cache ở config; file dữ liệu được commit + script ở `scripts/`; cờ `-dry-run` và test skip khi file vắng).
- Thêm nợ kỹ thuật `docs/tech-debt/config/TD-001` (secret dev nằm plaintext trong `config.dev.yaml`, mọi lệnh seed dùng mock getter) và `TD-002` (test repository `TRUNCATE` database dev mặc định); tham chiếu trong `seedadminunits/main.go` và `config.dev.yaml`.
- Test repository của feature từ chối `TRUNCATE` các bảng `addr_*` khi database đang chứa một bộ dữ liệu thật (version dạng ngày), kèm hướng dẫn đặt `TEST_DB_NAME`.
- Áp migration `0026` và nạp dữ liệu vào database dev `innerlamp` (34 / 3.321 / 63 / 696 / 10.033 / 10.565).

### Sự cố đã xảy ra và đã xử lý

Khi chứng minh đoạn bảo vệ ở trên, em chạy test repository **không đặt `TEST_DB_NAME`**; `connectTestDB` của repo `TRUNCATE` các bảng nội dung của database dev (đúng như `TD-002` mô tả) trước khi tới đoạn bảo vệ của feature này. Đã nạp lại bằng lệnh seed của repo: deities 20, temples 93, gallery 86, articles 60, sutras 153 (391 bản dịch). **Không khôi phục được** dữ liệu do người dùng tạo ở dev: users, sessions, devices, memorials, lamps, genealogies. `dharma_talks` rỗng tới lần `make run` kế tiếp (server tự seed khi bảng rỗng). Bộ dữ liệu hành chính không bị ảnh hưởng.

## v1.8 — 2026-10-11

Quyết định 2026-10-11: **app chỉ cho người dùng chọn hệ mới**; thêm tìm xã theo tên và xếp lại ưu tiên.

- `requirements.md`: mục mới "Hướng sử dụng" ghi quyết định; thêm AC 25 (tìm `?q=` trong một tỉnh, không phân biệt hoa thường, dấu, vị trí dấu thanh, kiểu dấu nháy) và AC 26 (`q` có mặt nhưng rỗng/trắng/quá 100 ký tự → `400 INVALID_FILTER`); đóng marker `?q=`; "Open questions (resolved)" ghi hai quyết định. Các AC cũ không đổi.
- `design.md`: usecase có thêm `SearchCurrentWards`; repository có truy vấn `fold(full_name) LIKE fold($2)`; handler rẽ nhánh theo `q`; thêm trace ASCII cho route tìm; thêm một Known gap (phụ thuộc `unaccent`). **Flow nạp dữ liệu không đổi** (đã kiểm), nên `flow.md` chỉ đổi số phiên bản.
- `flow.md`: không đổi sơ đồ (route tìm là một câu SELECT, không thuộc Step-by-step Flow).
- `tasks.md`: thêm Phase 7 (6 mục, tất cả `[x]`); Status cập nhật.
- `SRS.md` (v0.4): FR-011 từ "Could, chưa xác nhận" thành "Should" có 5 kịch bản; hai quyết định ghi vào "resolved"; thêm "client policy".
- Reason: người dùng chốt UI chỉ cho chọn hệ mới. Hệ mới ít cấp và không có điểm mơ hồ, nhưng một tỉnh có tới 168 xã nên ô tìm là điều kiện để lựa chọn này dùng được.

### Kiểm chứng trước khi quyết định

Danh sách hệ mới (thứ UI sẽ dùng) được đối chiếu với `vietnamese-provinces-database` (nguồn độc lập, dựng từ danh mục Tổng cục Thống kê): 34/34 tỉnh, 3.321/3.321 mã xã trùng khít, 0 xã lệch tỉnh; 188 tên khác chữ thì 186 chỉ khác vị trí dấu thanh, 2 khác dấu nháy (`B’Lao`, `D’Ran`). Đó là lý do tìm kiếm phải chịu được hai khác biệt này.

### Kiểm chứng

- Test mới: usecase (6 ca), handler (5 ca), repository trên Postgres thật (8 ca), E2E trên dữ liệu thật (1 bước con); toàn bộ test của feature xanh.
- `openapi.yaml` (cả hai bản): YAML hợp lệ, mọi `$ref` resolve, hai file giống hệt nhau.
- Một lỗi em gây ra khi viết test: cú pháp Python lẫn vào mã Go và một kỳ vọng sai ("phuong thang" không là chuỗi con của "Phường Hòa Thắng"); `go vet` và test bắt được, đã sửa.
- Quy ước cần nhớ khi viết dữ liệu Unicode vào mã bằng công cụ: dấu thanh và dấu nháy cong trong fixture của test được dựng từ code point (`string(rune(0x00E0))`) để chắc chắn là dạng tổ hợp hay tách rời.

### Còn lại, có chủ đích (đã xếp lại ưu tiên)

- Thấp: chuyển đổi cấp huyện (AC 13), nạp `district_mappings`, đối chiếu `conversion-table.xlsx` và giấy phép dữ liệu, đánh dấu hết hiệu lực — đều chỉ ảnh hưởng hệ cũ và ánh xạ, mà client không còn dùng.
- Còn mở: nơi dùng đầu tiên của tra cứu (hồ sơ đăng ký, chùa, gia phả, sổ giỗ) và việc đặt lại tên nhãn `mapping_type`.
- Concept OKF `docs/knowledge/features/administrative-units.md` — `il-okf-writer`, chỉ viết khi được yêu cầu.

## v1.7 — 2026-10-11

Phase 6 (E2E) và các việc còn lại đã xong; tính năng được xác minh end-to-end.

- `requirements.md`: không đổi AC nào; Status chuyển thành implemented và verified end-to-end; "Pending" ghi hết phần implement; Ghi chú kỹ thuật thêm script dựng lại file và ngoại lệ `.gitignore`.
- `design.md`: hàng ERD (`erd.svg` đã sinh lại) và Data (thêm script, ngoại lệ `.gitignore`); Known gap về script sinh file sửa lại cho đúng (script nay nằm trong repo). Cấu trúc layer và flow không đổi.
- `flow.md`: không đổi sơ đồ; Status ghi flow nạp dữ liệu đã được E2E chứng minh (đã kiểm 1:1 với `design.md`).
- `tasks.md`: Phase 5 và 6 tick `[x]` nốt các mục còn lại (`erd.svg`, script vào repo, `.gitignore`, E2E, `go test`/`go vet`). Chỉ còn một mục `[ ]`: concept OKF, thuộc `il-okf-writer` và chỉ viết khi được yêu cầu.
- `test-coverage-report.md` (mới, ngoài chuỗi cascade): kiểm 24 AC — 23 có test chứng minh, AC 13 hoãn có chủ đích; không có lỗi wiring nào lọt qua.
- Reason: hoàn tất danh sách việc của `tasks.md` theo yêu cầu "làm đến hết".

### Quyết định em tự đưa ra (cần người dùng biết)

- **Thêm ngoại lệ `.gitignore`** `!assets/data/administrative-units.json` (repo chặn `*.json` trong khối "Sensitive file"). Lý do: file là dữ liệu tham chiếu công khai, và yêu cầu gốc là có một file JSON để seed lại DB về sau; một file bị ignore thì không chia sẻ được. Hoàn tác bằng cách xoá ba dòng đã thêm.
- **Đưa script sinh file vào `scripts/`** (không phải thư mục riêng của feature): theo thói quen của repo (`scripts/fetch_*.py`), có ghi vào `scripts/README.md`.
- **Không viết concept OKF**: lời nhắc của dự án ghi rõ chỉ viết OKF khi được yêu cầu, dù điều kiện "đã E2E" đã đủ.

### Kiểm chứng

- E2E: `TestAdministrativeUnits_E2E` (13 bước con) và `TestAdministrativeUnits_RateLimit_E2E` xanh; toàn bộ package `e2e` xanh (156 s), tức các sửa ở `main.go`, `config.go`, `errors.go` không làm hỏng test của feature khác.
- `go vet ./...` sạch; `go test -short ./...` toàn repo không có package nào fail; `check-architecture.sh` 0 lỗi.
- `erd.svg` sinh lại bằng mermaid-cli 11.4.2 chạy với Chrome cài sẵn (không tải trình duyệt); script sinh file dựng lại từ nguồn cho file giống từng byte.

### Còn lại, có chủ đích

- Chuyển đổi cấp huyện (AC 13) và nạp `district_mappings`; hành vi của 5 đặc khu đảo hiện là danh sách ứng viên rỗng và đã có test khoá lại.
- Đối chiếu bộ dữ liệu với `conversion-table.xlsx` chính thức và kiểm giấy phép dữ liệu; dòng "Kim Sơn → Kim Đông" nằm ở `meta.unresolved`.
- Đánh dấu hết hiệu lực cho đơn vị vắng trong bản dữ liệu mới.
- Ba marker `⚠️ Not confirmed` trong `requirements.md` "Pending": nơi dùng đầu tiên, tìm theo tên không dấu (`?q=`), và việc đặt lại tên nhãn `mapping_type`.
- Concept OKF `docs/knowledge/features/administrative-units.md`.

## v1.6 — 2026-10-11

Phase 5 (CLI nạp dữ liệu `cmd/seedadminunits`) đã implement và test; đã chạy CLI thật.

- `requirements.md`: không đổi AC nào; Status và "Pending" cập nhật; Ghi chú kỹ thuật thêm cú pháp CLI và cờ `-dry-run` (thêm ngoài danh sách AC, ghi rõ là bổ sung).
- `design.md`: hàng Seed CLI chuyển sang *(created)* và mô tả thật; Step 2 nêu các kiểm tra cấu trúc (strict, `meta.format`, `meta.counts`, trường rỗng); Step 3 thêm nhánh `-dry-run`; Step 7 sửa cho đúng cài đặt — số dòng từng bảng được in ngay sau `Verify` (vì `ReplaceDataset` không trả số dòng), còn bước cuối in version và thời gian. **Số bước vẫn 7.**
- `flow.md`: cập nhật sơ đồ cho khớp 1:1 với `design.md` v1.6 (thêm dòng in báo cáo và nhánh `opt -dry-run` sau `Verify`; nhãn Step 2 và Step 7); đã kiểm cú pháp Mermaid hợp lệ.
- `tasks.md`: Phase 5 tick `[x]` hai mục (parse + test, main); hai mục còn lại của Phase 5 là quyết định về script sinh file (mở). Header Status cập nhật.
- Reason: implement Phase 5. Hai quyết định cài đặt người dùng cần biết: (1) `-dry-run` — thêm ngoài spec vì rẻ và cho phép kiểm file mà không cần DB; (2) test file thật **skip** thay vì fail khi file vắng, vì `.gitignore` chặn `*.json` nên trên một checkout mới file này không có.

### Kiểm chứng

- 17 test mới ở `cmd/seedadminunits` xanh (kể cả file thật); `go build ./...`, `go vet` sạch.
- **Chạy CLI thật** với file dữ liệu thật trên database test: `-dry-run` in 34 / 3.321 / 63 / 696 / 10.033 / 10.565 (district mappings 5 đọc, không nạp; 4 sửa đã áp; 1 unresolved; 0 chuỗi phải chuẩn hoá NFC) và DB không đổi; nạp lần đầu 1,1 s; nạp lần hai 0,8 s, số dòng không đổi và đúng một dòng `addr_dataset_versions`; file bị làm hỏng (đặt lại lỗi nhãn gốc + bỏ một mapping) bị từ chối với 3 vi phạm và DB không đổi (dòng đánh dấu còn nguyên); file thiếu / JSON sai format báo lỗi rõ; nạp lại bản đúng sửa dòng bị đánh dấu về giá trị chuẩn.
- Không kiểm riêng mã thoát của tiến trình (lệnh pipe nuốt `$?`); mã nguồn dùng `log.Fatalf` / `os.Exit(1)` cho mọi nhánh lỗi.

### Follow-up cho các tài liệu/mã ngoài thư mục này

- `.gitignore`: `*.json` chặn `assets/data/administrative-units.json` nên file dữ liệu **không được commit** trừ khi thêm ngoại lệ `!assets/data/administrative-units.json` — chưa sửa vì là quyết định của người dùng.
- `docs/specs/db/erd.svg`: chưa sinh lại; `docs/knowledge/features/administrative-units.md`: chưa tạo (chưa có E2E).
- `go.mod`/CLAUDE.md của repo: không đổi gì.

## v1.5 — 2026-10-11

Phase 4 (config, handler, wiring `main.go`, `openapi.yaml`) đã implement và test; đã chạy binary thật.

- `requirements.md`: AC 24 đổi từ hằng `86400` sang "lấy từ `administrative_units.cache_max_age` (dev 24h)"; mục API contract ghi đường dẫn đã có trong `openapi.yaml`; ghi chú NFR cập nhật số đo thật. Không có hành vi mới nào đối với client — dev vẫn trả `max-age=86400`.
- `design.md`: các hàng Handler, DTO, Error map, Config, Wiring, API spec chuyển sang *(created)*/*(modified)*/*(updated)*; ghi tên DTO thật (`provinceResponse` dùng chung hai hệ, hai kiểu generic), một `ClientIPRateLimit` dùng chung cho cả chín route, và lý do cache lifetime nằm ở config. Request flow ASCII không đổi (đã khớp với mã).
- `flow.md`: không đổi — các route đọc không thuộc Step-by-step Flow nạp dữ liệu (đã kiểm).
- `tasks.md`: Phase 4 tick `[x]` cả bảy mục (đã kiểm); mục `mapping_type` được tách: phần cập nhật OpenAPI xong, phần "đặt lại tên nhãn" vẫn là marker mở ở `requirements.md`.
- Reason: implement Phase 4. Một quyết định cài đặt cần người dùng biết: lifetime của cache được đưa vào config (`cache_max_age`) thay vì hằng số 86400 trong handler, theo quy tắc "mọi giá trị đổi theo môi trường nằm ở `config/`"; đồng ý "24 giờ" của người dùng vẫn là giá trị dev.

### Kiểm chứng

- Test: config (2 test mới), delivery/http (10 test mới, gồm chuỗi lắp ráp thật `webAPIKey` → limiter → handler); `go build ./services/...`, `go vet`, `go test -short ./services/appService/{config,delivery/http,domain,usecase,cmd}/` xanh; `check-architecture.sh` 0 lỗi.
- **Chạy binary thật** (database tạm `innerlamp_smoke`, đã xoá sau khi dùng; file dữ liệu thật nạp vào): `new/provinces` → 34 tỉnh, `Cache-Control: public, max-age=86400`, header key vắng → 200, key sai → 401 `INVALID_API_KEY`, key đúng → 200; Hồ Chí Minh mới → 168 phường, "Phường An Đông" đầu tiên; xã cũ `26761` → Phường Cầu Ông Lãnh, `unique: true`; xã mới `26758` → bốn xã cũ (Cầu Kho, Cầu Ông Lãnh, Cô Giang, Nguyễn Cư Trinh), `unique: false`; Hồ Chí Minh mới ← ba tỉnh cũ; Tiền Giang cũ → chỉ Đồng Tháp; huyện đảo `318` → `items: []` 200; mã không tồn tại → 404 `ADMINISTRATIVE_UNIT_NOT_FOUND` không có `Cache-Control`; đường dẫn lạ 404, sai method 405; "Phường 1, 2, 4, 5, 7, 9, 11, 12, 13, 14" đúng thứ tự số.
- `openapi.yaml` (cả hai bản): YAML hợp lệ, 9 path và 13 schema `Administrative*`, mọi `$ref` resolve, hai file giống hệt nhau. Lần chèn đầu có `summary` chứa `: ` làm hỏng YAML; đã bắt được và sửa bằng cách đặt trong dấu nháy.

### Phát hiện khi implement

- `main()` không có test: đăng ký route ở `cmd/main.go` chỉ được xác minh bằng biên dịch, đọc lại và chạy binary thật. Đó là lý do E2E ở Phase 6 vẫn cần.
- Server lúc khởi động tự seed kinh (`sutras`) và cần bảng `deities` đã có dữ liệu; một database mới chỉ chạy migration thì binary thoát với "sutra seed references deities that are not seeded". Không liên quan đến feature này nhưng làm E2E phải seed deities trước.
- `config.go`, `config.dev.yaml`, `main.go`, `errors.go` là file dùng chung với phiên "lamp"/"genealogy" đang làm song song; em chỉ chèn theo điểm neo và đọc lại ngay trước khi ghi.

### Follow-up cho các tài liệu/mã ngoài thư mục này

- `docs/specs/db/erd.svg`: vẫn chưa sinh lại (không có `mmdc`).
- `docs/knowledge/features/administrative-units.md`: chưa tạo (chưa có E2E).

## v1.4 — 2026-10-10

Phase 3 (migration `0026` + repository) đã implement và test trên Postgres thật.

- `requirements.md`: không đổi hành vi nào; chỉ cập nhật Status và "Pending". Đã kiểm AC 1–12, 14, 15, 19, 20 bằng test repository (các AC của client còn lại chờ Phase 4).
- `design.md`: hàng Repository, Migration, ERD chuyển sang *(created)*/*(updated)*; ghi cách repository phân biệt "không tồn tại" với "có nhưng rỗng" (một câu `SELECT 1` chỉ khi kết quả rỗng); `ReplaceDataset` nạp theo lô 1.000 dòng và không tự `Verify`; mục Schema ghi rõ cột `type` cố ý không CHECK; thêm một Known gap về quy ước test chung của package. Cấu trúc 7 bước của flow **không đổi** — chỉ đổi cách thực hiện Step 4–6 (lô thay vì từng dòng), vẫn một transaction.
- `flow.md`: không đổi sơ đồ (Step 4–6 vẫn là các UPSERT trong một transaction; lô là chi tiết cài đặt). Đã kiểm lại 1:1 với `design.md`.
- `tasks.md`: Phase 3 tick `[x]` 4 mục; tách "cập nhật ERD" thành hai mục vì `erd.svg` chưa sinh lại được (không có `mmdc` trên máy), mục SVG giữ `[ ]`.
- Reason: implement Phase 3. Hai chi tiết do cài đặt định ra và đã ghi vào `design.md`: (1) nạp theo lô, vì nạp từng dòng mất 22 s cho file thật; (2) không CHECK cột `type`, theo cùng lý do với `mapping_type`.

### Kiểm chứng

- 11 test repository xanh trên Postgres 16 thật (database riêng `innerlamp_test`); `go build ./services/...`, `go vet`, `go test -short ./services/appService/...` xanh; `check-architecture.sh` 0 lỗi.
- Nạp **file dữ liệu thật** (test tạm, đã xoá): 34 / 3.321 / 63 / 696 / 10.033 / 10.565 dòng, lần đầu 1,3 s và lần hai 1,3 s với số dòng không đổi; truy vấn chuyển đổi 1–3 ms. Ví dụ thật: xã cũ `26761` → Phường Cầu Ông Lãnh; Hồ Chí Minh ← Hồ Chí Minh + Bình Dương + Bà Rịa - Vũng Tàu; Tiền Giang cũ → chỉ Đồng Tháp; Đặc khu Bạch Long Vĩ ← 0 ứng viên (đúng như đã ghi ở Known gaps).

### Phát hiện khi implement

- Quy ước test của package `repository` (`connectTestDB`) xoá dữ liệu trên database dev `innerlamp`. Em tạo database `innerlamp_test` trong container Postgres đang chạy và dùng `TEST_DB_NAME` để không đụng dữ liệu dev; database này nằm lại trong container.
- Trong lúc làm, một phiên khác thêm `postgres_lamp_repository.go` (feature lamp) làm package tạm thời không build vì trùng `rowScanner`; đã tự hết sau đó, em không đụng vào file của họ.

### Follow-up cho các tài liệu/mã ngoài thư mục này

- `docs/specs/db/erd.svg`: sinh lại từ `erd.mmd` (cần mermaid-cli); hiện stale so với `0026`.
- `docs/knowledge/features/administrative-units.md`: chưa tạo (chưa chạy được end-to-end).

## v1.3 — 2026-10-10

Phase 2 (usecase) đã implement và test; spec chỉ cần đánh dấu tiến độ.

- `requirements.md`: không đổi hành vi nào; chỉ cập nhật Status và dòng "Pending" (Phase 1–2 xong). Đã kiểm lại AC 1–12 và 14–15: usecase không thêm quy tắc nào ngoài các AC này.
- `design.md`: hàng Usecase đổi từ *(not yet created)* sang *(created)* và liệt kê đúng chín method, nêu rõ constructor không nhận config (lists không phân trang, giới hạn tần suất nằm ở delivery) và quy tắc "mã trống → not found, không gọi repository" mà v1.2 chưa ghi. Không đổi layers khác, không đổi flow.
- `flow.md`: không đổi — usecase đọc là một câu SELECT, không thuộc Step-by-step Flow nạp dữ liệu (đã kiểm).
- `tasks.md`: Phase 2 tick `[x]` cả ba mục; nội dung mục được viết lại theo thực tế (chín method, mã trống, bộ năm kiểm tra dùng chung); header Status cập nhật.
- Reason: implement Phase 2. Không có quyết định hành vi mới — hai chi tiết nhỏ do usecase tự định ra và đã ghi vào `design.md`: mã trống là not-found (route path không thể rỗng nhưng usecase không tin vào điều đó) và danh sách luôn là slice không-nil.

### Kiểm chứng

- `go build`, `go vet` sạch; `go test -short ./services/appService/...` xanh toàn bộ; `scripts/check-architecture.sh` 0 lỗi.
- File mới phủ 100% dòng; chỉ import `domain` và thư viện chuẩn.

### Follow-up cho các tài liệu/mã ngoài thư mục này

- `docs/knowledge/features/administrative-units.md`: chưa tạo (feature chưa chạy được end-to-end).

## v1.2 — 2026-10-10

Phase 1 (domain) đã implement và test; spec được chỉnh cho khớp với code.

- `requirements.md`: AC 21 mở rộng từ năm lên bảy quy tắc toàn vẹn — thêm "mã trùng trong một bảng", "đơn vị trỏ tới cha không có" và "tỉnh cũ có xã đi sang nhiều hơn một tỉnh mới" (quy tắc cuối làm AC 12 kiểm được ngay lúc nạp, và là thứ đã bắt được lỗi nhãn Tiền Giang); "Pending" ghi Phase 1 xong. Không thêm hay bỏ hành vi nào đối với client.
- `design.md`: đổi tên entity `New*`/`Old*` thành `Current*`/`Legacy*` (`NewX` đọc như constructor trong Go) ở bảng layers, DTO và interface; `Conversion[T]` có `Unique()` suy ra thay vì field `Unique`; `Dataset` có `SourceNote` thay `SourceDate`, và bảng `addr_dataset_versions` bỏ cột `source_date` (version chính là ngày bộ dữ liệu); Step 3 nêu bảy quy tắc; thêm một Known gap mới (xem dưới). Cấu trúc 7 bước **không đổi**.
- `flow.md`: chỉ đổi chữ "năm quy tắc" thành "bảy quy tắc" ở `Note` Step 3 để khớp `design.md`. Không đổi bước nào.
- `tasks.md`: Phase 1 tick `[x]` cả năm mục (đã kiểm: build, vet, test xanh); sửa nội dung mục entity và mục `Verify` theo thực tế; header Status cập nhật.
- Reason: implement Phase 1 làm lộ ba chỗ cần sửa spec — tên kiểu không idiomatic, một cột thừa, và một khoảng trống chuyển đổi cho năm đặc khu đảo.

### Phát hiện khi implement (cần người dùng quyết khi làm tiếp)

- Năm đặc khu đảo chỉ là đích của mapping **cấp huyện**, mà mapping cấp huyện chưa được nạp (hoãn). Hệ quả: chuyển đặc khu đó từ mới sang cũ trả danh sách rỗng chứ không phải một huyện cũ. Chấp nhận được trong phạm vi hiện tại, nhưng nên quyết khi làm chuyển đổi cấp huyện: nạp `district_mappings` và trả huyện cũ, hay giữ rỗng.
- `Dataset.Verify` đã chạy thật trên `assets/data/administrative-units.json`: 0 vi phạm (34 / 3.321 / 63 / 696 / 10.033 / 10.565 / 5). Khi đặt lại lỗi nhãn gốc của xã `27691` thì hai quy tắc cùng báo.

### Follow-up cho các tài liệu/mã ngoài thư mục này

- `services/appService/domain/README.md` không liệt kê entity `temple`/`deity` nên không thêm mục cho feature này; nếu muốn README liệt kê, cần một mục riêng.
- `docs/knowledge/features/administrative-units.md`: chưa tạo — feature chưa chạy được end-to-end, `il-okf-writer` chỉ viết sau khi có E2E.

## v1.1 — 2026-10-10

Trả lời đợt câu hỏi sau bản v1.0 (mười câu) và tạo file dữ liệu duy nhất để seed lại DB.

- `requirements.md`: thêm AC 23 (giới hạn tần suất theo IP) và AC 24 (`Cache-Control` 24 giờ); AC 1, 2, 4–6 chốt thứ tự chữ cái tiếng Việt; AC 13 (chuyển đổi cấp huyện) hoãn — giữ nguyên số để các tham chiếu khác không lệch (số AC là id ổn định, không phải thứ tự); AC 19 thêm "không xoá đơn vị vắng trong bản mới"; bỏ route chuyển đổi cấp huyện (còn 9 route); "Pending" còn 3 marker (nơi dùng đầu tiên, `?q=`, tên `mapping_type`) và đưa đối chiếu file chính thức vào mục hoãn có chủ đích; mười một câu trả lời ghi vào "Open questions (resolved)".
- `design.md`: Overview thêm limiter + `Cache-Control` + một file dữ liệu; bỏ `ConvertOldDistrict` và bảng `addr_district_mappings` (bảy bảng thay vì tám); thêm `sort_order` vào các bảng đơn vị; thêm hàng Config; Data trỏ `assets/data/administrative-units.json`; Step 1 đọc một file (`-file`), Step 2 đọc nhưng không nạp `district_mappings`, Step 3 và Step 4 từ marker mở thành "Confirmed" (file đã sửa; chỉ upsert, không DELETE); không đổi số bước (vẫn 7).
- `flow.md`: cập nhật sơ đồ cho khớp 1:1 với `design.md` v1.1 (một file, bỏ hai `Note` ⚠️, `Note` mới "chỉ upsert, không DELETE", `ON CONFLICT DO NOTHING` ở version); đã kiểm cú pháp Mermaid hợp lệ. Cấu trúc bước không đổi.
- `tasks.md`: bỏ mục huyện và đối chiếu xlsx (chuyển sang Deferred); thêm mục config rate limit và `Cache-Control`; thêm hai mục `[x]` ở Phase 5 (file dữ liệu đã tạo và đã tự kiểm; hai lỗi nhãn đã sửa trong file); thêm mục quyết định có đưa script sinh file vào repo; Phase 4 không còn bị chặn.
- Reason: người dùng trả lời toàn bộ câu hỏi chặn và yêu cầu một file JSON duy nhất để seed lại dễ dàng. Các câu trả lời **đơn giản hoá** thiết kế (bớt một route, một bảng, một nhánh transaction) chứ không làm đổi cấu trúc flow.

### File dữ liệu `assets/data/administrative-units.json`

- Nguồn: `vietnam-address-kit` commit `f584592` (2026-09-26), dữ liệu bản 2026-09-25; mã MIT; **chưa** đối chiếu với file chính thức.
- Đã sửa: xã "Phường 4" (mã cũ `27691`, huyện `794` Thành phố Tân An) từ tỉnh `82` sang `80`; xã "Xã Tiên Hải" (`30781`, huyện `900` Thành phố Hà Tiên) từ `89` sang `91`; loại hai dòng huyện nhân đôi. Dòng huyện Kim Sơn (`37/376`) nằm ở `meta.unresolved`, không nạp.
- Tự kiểm khi sinh: mọi xã cũ có mapping, mọi xã mới là đích, mapping không trỏ vào đơn vị không có, xã cùng tỉnh với huyện, mã huyện duy nhất, mỗi tỉnh cũ ứng đúng một tỉnh mới.
- Số lượng: 34 tỉnh mới, 3.321 xã mới, 63 tỉnh cũ, 696 huyện cũ, 10.033 xã cũ, 10.565 mapping cấp xã, 5 mapping cấp huyện (chưa nạp).

### Follow-up cho các tài liệu/mã ngoài thư mục này (chưa làm ở đây)

- `docs/specs/api/openapi.yaml` và `web/docs/public/openapi.yaml`: chín route khi implement handler.
- `migrations/0026_administrative_units.sql`, `migrations/README.md`, `docs/specs/db/erd.mmd`/`erd.svg`.
- `services/appService/config/config.go` + `config.dev.yaml`: block `administrative_units.rate_limit`.
- `docs/knowledge/features/administrative-units.md`: concept OKF, do `il-okf-writer` tạo sau khi code đã kiểm chứng.
- Không có file OKF hiện có nào bị lạc hậu bởi thay đổi này.

## v1.0 (initial) — 2026-10-10

Initial spec written before implementation, as the work order for building the administrative-units
feature (prospective mode). Derived from `SRS.md` v0.2.

- `requirements.md`: năm user story (US-ADU-01…05), 22 tiêu chí EARS cho chọn địa chỉ hai hệ, chuyển đổi cũ ↔ mới, truy cập công khai qua web API key, và nạp dữ liệu. Contract đề xuất (chưa có trong `openapi.yaml`); các marker `⚠️ Not confirmed` còn mở nằm ở "Pending" và hai chỗ trong mục API contract. Sáu câu đã chốt ghi ở "Open questions (resolved)".
- `design.md`: bảng layers & components cho `domain` / `usecase` / `repository` / `delivery` / `cmd/seedadminunits`, schema tám bảng `addr_*` dự kiến (migration `0026`), request flow ASCII cho route đọc, Step-by-step Flow 7 bước cho nạp dữ liệu.
- `flow.md`: một sequence diagram cho flow nạp dữ liệu, mirror 1:1 bảy bước của `design.md` (route đọc chỉ một câu SELECT nên không có sơ đồ).
- `tasks.md`: sáu Phase, mọi mục `[ ]`; các mục bị chặn bởi marker được ghi rõ.
- Reason: người dùng cần bộ metadata hành chính hai hệ (63 tỉnh cũ, 34 tỉnh mới) với bảng đối chiếu cấp xã để chọn địa chỉ theo hệ nào cũng được và suy ra hệ còn lại.

### Đã kiểm tra trên dữ liệu thật trong lúc viết (ghi lại để người đọc sau không phải làm lại)

- Bộ dữ liệu ứng viên `vietnam-address-kit` (2026-09-25) đủ ở cấp xã: mọi xã cũ có mapping, mọi xã mới là đích; đối chiếu chéo với `vietnamadminunits` khớp 9.470/9.532 xã khớp tên được (99,3%).
- Mỗi tỉnh cũ ứng đúng một tỉnh mới. Dòng "Tiền Giang → Tây Ninh" trong dữ liệu thô là lỗi nhãn của một xã thuộc Tân An (Long An), tương tự một xã của Hà Tiên; **không** phải tỉnh bị tách. Chú thích "không tỉnh cũ nào bị tách" của migration `0025_temple_province_34_units.sql` vì vậy vẫn đúng.
- 34 tên tỉnh của bộ dữ liệu trùng khít `domain/province.go`.

### Follow-up cho các tài liệu/mã ngoài thư mục này (chưa làm ở đây)

- `docs/specs/api/openapi.yaml` và `web/docs/public/openapi.yaml`: thêm mười route khi contract được chốt.
- `migrations/0026_administrative_units.sql` + `migrations/README.md` + `docs/specs/db/erd.mmd`/`erd.svg`.
- `docs/knowledge/features/administrative-units.md`: concept OKF, do `il-okf-writer` tạo sau khi code đã kiểm chứng.
- Không có file OKF hiện có nào bị lạc hậu bởi spec này.
