# Changelog — administrative-units Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first. `SRS.md` (từ `il-ba`) có version riêng (hiện v0.2) và là đầu vào của bộ spec này.

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
