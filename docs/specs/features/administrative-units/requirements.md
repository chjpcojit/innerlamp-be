# Requirements — administrative-units

Version: 1.8
Last updated: 2026-10-11
Status: implemented và verified end-to-end (2026-10-11), gồm tìm xã theo tên (`?q=`, AC 25–26); mọi AC (trừ AC 13 đã hoãn) có test chứng minh, xem `test-coverage-report.md`. Còn 2 marker `⚠️ Not confirmed` (xem "Pending").
Module: `administrative-units`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này sinh ra từ [`SRS.md`](./SRS.md) (v0.3, cùng thư mục): SRS nói **cái gì** theo ngôn ngữ nghiệp vụ, file này chuyển nó thành tiêu chí chấp nhận EARS cho API. Không có phần nào trong thư mục này đã được implement.

## US-ADU-01: Chọn địa chỉ theo hệ mới (tỉnh → xã)

## User Story

> Là một **người dùng app đang nhập địa chỉ** (kể cả khi đang đăng ký, chưa có tài khoản),
> Tôi muốn **chọn một tỉnh/thành rồi chọn xã/phường/đặc khu thuộc tỉnh đó theo đơn vị hành chính mới**,
> Để **địa chỉ của tôi đúng với hệ thống hai cấp đang áp dụng từ 01/07/2025**.

## Acceptance Criteria (EARS notation)

1. WHEN a `GET` request for the new-scheme provinces arrives, THE SYSTEM SHALL return `200` with `data.items` holding all 34 provinces, each with `code`, `name`, `full_name` and `type` (`province` | `city`), and SHALL return them in Vietnamese alphabetical order of `name` (digits compared as numbers), the same order on every call.
2. WHEN a `GET` request for the wards of a new-scheme province arrives with a `code` that exists, THE SYSTEM SHALL return `200` with `data.items` holding **every** ward of that province in one response (no pagination), each with `code`, `name`, `full_name`, `type` (`ward` | `commune` | `special_zone`) and `province_code`, and SHALL NOT include a ward of any other province; the wards SHALL be in Vietnamese alphabetical order of `name`, digits compared as numbers ("Phường 2" before "Phường 10").
3. WHEN a request names a new-scheme province `code` that does not exist, THE SYSTEM SHALL return `404` with error code `ADMINISTRATIVE_UNIT_NOT_FOUND` and SHALL NOT return a partial list.
25. WHEN a request for the wards of a new-scheme province carries a search term `q`, THE SYSTEM SHALL return `200` with only the wards **of that province** whose full name contains the term, in the same order as without `q`, matching regardless of letter case, Vietnamese diacritics (`đ` = `d`), the position of a tone mark (`Hòa` = `Hoà`, `Thủy` = `Thuỷ`) and the kind of apostrophe (`B’Lao` = `B'Lao`); WHEN nothing matches, THE SYSTEM SHALL return `200` with `data.items = []`; WHEN the province does not exist, it SHALL return `404` as in AC 3, whatever the term; and a request without `q` SHALL behave exactly as in AC 2.
26. WHEN `q` is present but empty, blank after trimming, or longer than 100 characters, THE SYSTEM SHALL return `400` with error code `INVALID_FILTER` and `details.field = "q"`, SHALL NOT treat it as "no search", and SHALL NOT query wards; the error response SHALL NOT carry `Cache-Control`.

## US-ADU-02: Chọn địa chỉ theo hệ cũ (tỉnh → huyện → xã)

## User Story

> Là một **người dùng quen với địa chỉ cũ** (hoặc đang nhập địa chỉ ghi theo giấy tờ trước 01/07/2025),
> Tôi muốn **chọn tỉnh, rồi quận/huyện, rồi xã/phường theo đơn vị hành chính cũ**,
> Để **nhập địa chỉ theo cách tôi biết, không bị ép sang tên đơn vị mới**.

## Acceptance Criteria (EARS notation)

4. WHEN a `GET` request for the old-scheme provinces arrives, THE SYSTEM SHALL return `200` with `data.items` holding all 63 old provinces in Vietnamese alphabetical order of `name`, each with `code`, `name`, `full_name`, `type`.
5. WHEN a `GET` request for the districts of an old-scheme province arrives with a `code` that exists, THE SYSTEM SHALL return `200` with every district of that province (`code`, `name`, `full_name`, `type`, `province_code`) and none of another province, in Vietnamese alphabetical order of `name`.
6. WHEN a `GET` request for the wards of an old-scheme district arrives with a `code` that exists, THE SYSTEM SHALL return `200` with every ward of that district (`code`, `name`, `full_name`, `type`, `district_code`, `province_code`); WHEN the district has no wards (island districts such as Bạch Long Vĩ), THE SYSTEM SHALL return `200` with `data.items = []`, not an error.
7. WHEN a request names an old-scheme province or district `code` that does not exist, THE SYSTEM SHALL return `404` with error code `ADMINISTRATIVE_UNIT_NOT_FOUND`.
8. THE SYSTEM SHALL keep the two schemes in separate namespaces, so a numeric `code` that denotes an old ward and a different new ward is never confused; every route names its scheme in the path.

## US-ADU-03: Suy ra địa chỉ ở hệ còn lại (cũ ↔ mới)

## User Story

> Là một **client của app**,
> Tôi muốn **từ một xã/phường cũ biết xã/phường mới tương ứng, và ngược lại**,
> Để **hiển thị và lưu cả hai dạng địa chỉ mà người dùng không phải nhập hai lần**.

## Acceptance Criteria (EARS notation)

9. WHEN a conversion request arrives for an old-scheme ward `code` that exists, THE SYSTEM SHALL return `200` with `data.candidates` holding every new-scheme ward that old ward maps to; each candidate SHALL carry the new ward (`code`, `name`, `full_name`, `type`), its new province (`code`, `name`, `full_name`), a `mapping_type` and a `note`.
10. WHEN a conversion request arrives for a new-scheme ward `code` that exists, THE SYSTEM SHALL return `200` with `data.candidates` holding every old-scheme ward that maps to it; each candidate SHALL carry the old ward, its district and its province, plus `mapping_type` and `note`.
11. WHEN a conversion has more than one candidate, THE SYSTEM SHALL return all of them, SHALL set `data.unique = false`, and SHALL NOT choose one for the caller; WHEN it has exactly one, THE SYSTEM SHALL set `data.unique = true`.
12. WHEN a conversion request arrives for an old-scheme province, THE SYSTEM SHALL return exactly one new-scheme province; WHEN it arrives for a new-scheme province, THE SYSTEM SHALL return every old-scheme province that was merged into it. Both are derived from the ward mappings, not stored separately.
13. *(Deferred 2026-10-10 — conversion at district level is not part of this version; the number is kept so other references stay valid. The five district mappings travel in the data file but are not loaded.)*
14. WHEN a conversion request names a unit `code` that does not exist in the named scheme, THE SYSTEM SHALL return `404` with error code `ADMINISTRATIVE_UNIT_NOT_FOUND`.
15. THE SYSTEM SHALL answer every conversion from the loaded dataset only, without calling any external service and without writing to the database.

## US-ADU-04: Gọi được khi chưa đăng nhập, qua web API key

## User Story

> Là một **client web hoặc mobile đang ở bước đăng ký**,
> Tôi muốn **gọi các route địa chỉ mà không cần access token**,
> Để **màn đăng ký chọn được địa chỉ ngay khi người dùng chưa có tài khoản**.

## Acceptance Criteria (EARS notation)

16. WHEN a request to any `administrative-units` route carries no access token, THE SYSTEM SHALL serve it as it does a request that carries one; THE SYSTEM SHALL NOT apply the JWT authentication middleware to these routes.
17. WHEN a request carries the header `X-API-Key` with the configured web API key, THE SYSTEM SHALL serve it; WHEN the header is absent, THE SYSTEM SHALL also serve it (the caller is treated as mobile); WHEN the header is present but wrong or empty, THE SYSTEM SHALL respond `401` with error code `INVALID_API_KEY` and SHALL NOT return any unit.
18. THE SYSTEM SHALL write every response in the project envelope (`{"success": true, "data": …}` / `{"success": false, "error": {…}}`).
23. WHEN a client IP exceeds the configured request budget on these routes, THE SYSTEM SHALL respond `429` with error code `RATE_LIMITED`, `details.retry_after_seconds` and the `Retry-After` header (as in `api-response-envelope`), and the wrong-key check SHALL run before the limiter so a refused key does not spend the caller's budget. The budget and window are configuration, not constants.
24. WHEN a request succeeds, THE SYSTEM SHALL send `Cache-Control: public, max-age=<seconds>` where the lifetime is the configured `administrative_units.cache_max_age` (dev: 24 hours = 86400), so clients and intermediaries may reuse the answer for that long; a `401`, `404`, `429` or `5xx` SHALL NOT carry it.

## US-ADU-05: Nạp bộ dữ liệu hành chính bằng lệnh seed

## User Story

> Là một **operator vận hành**,
> Tôi muốn **nạp một phiên bản bộ dữ liệu hành chính (hai hệ + bảng đối chiếu) bằng một lệnh, chạy lại được**,
> Để **dữ liệu có sẵn trong DB trước khi API phục vụ và cập nhật được khi có nghị quyết mới**.

## Acceptance Criteria (EARS notation)

19. WHEN the operator runs the seed command with a dataset, THE SYSTEM SHALL store both schemes' provinces and wards, the old scheme's districts, every ward mapping and district mapping, and the dataset version with its source date, all inside one transaction, so a failure leaves the previous data unchanged; the load only adds or updates units and SHALL NOT delete a unit that is absent from the new dataset.
20. WHEN the operator runs the seed command a second time with the same dataset, THE SYSTEM SHALL leave every unit and mapping unchanged (idempotent).
21. WHEN the dataset violates an integrity rule — an old ward with no mapping, a new ward that is the target of no mapping, a mapping that points to an unknown unit, an old ward whose province differs from its district's province, a code that appears twice in one table (for old districts: claimed by two provinces), a unit that names a parent not in the dataset, or an old province whose wards map into more than one new province — THE SYSTEM SHALL refuse the whole load, print a report naming each offending row, and SHALL NOT write anything.
22. WHEN the seed command finishes successfully, THE SYSTEM SHALL print the number of units and mappings loaded per table, so the operator can compare them with the expected counts of the dataset.

## API contract (confirmed surface for this feature)

Đường dẫn đã được người dùng chốt ngày 2026-10-10 ("đồng ý") và đã có trong `docs/specs/api/openapi.yaml` (tag `AdministrativeUnits`, bản đồng bộ `web/docs/public/openapi.yaml`, v1.5). Đã chạy binary thật và gọi đủ chín route với file dữ liệu thật.

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/administrative-units/new/provinces` | public + web API key | `200` `{items: [NewProvince]}` | 401 `INVALID_API_KEY`, 405, 429 `RATE_LIMITED`, 500 |
| GET | `/v1/administrative-units/new/provinces/{code}/wards` | public + web API key | `200` `{items: [NewWard]}` | 401 `INVALID_API_KEY`, 404 `ADMINISTRATIVE_UNIT_NOT_FOUND`, 405, 500 |
| GET | `/v1/administrative-units/old/provinces` | public + web API key | `200` `{items: [OldProvince]}` | 401, 405, 500 |
| GET | `/v1/administrative-units/old/provinces/{code}/districts` | public + web API key | `200` `{items: [OldDistrict]}` | 401, 404 `ADMINISTRATIVE_UNIT_NOT_FOUND`, 405, 429 `RATE_LIMITED`, 500 |
| GET | `/v1/administrative-units/old/districts/{code}/wards` | public + web API key | `200` `{items: [OldWard]}` (có thể rỗng) | 401, 404 `ADMINISTRATIVE_UNIT_NOT_FOUND`, 405, 429 `RATE_LIMITED`, 500 |
| GET | `/v1/administrative-units/old/wards/{code}/conversion` | public + web API key | `200` `{unique, candidates: [NewWardCandidate]}` | 401, 404 `ADMINISTRATIVE_UNIT_NOT_FOUND`, 405, 429 `RATE_LIMITED`, 500 |
| GET | `/v1/administrative-units/new/wards/{code}/conversion` | public + web API key | `200` `{unique, candidates: [OldWardCandidate]}` | 401, 404 `ADMINISTRATIVE_UNIT_NOT_FOUND`, 405, 429 `RATE_LIMITED`, 500 |
| GET | `/v1/administrative-units/old/provinces/{code}/conversion` | public + web API key | `200` `{unique: true, candidates: [NewProvince]}` | 401, 404, 405, 500 |
| GET | `/v1/administrative-units/new/provinces/{code}/conversion` | public + web API key | `200` `{unique, candidates: [OldProvince]}` | 401, 404, 405, 500 |

Hình dạng dữ liệu (đề xuất): `NewProvince` `{code, name, full_name, type}`; `NewWard` thêm `province_code`; `OldProvince` như `NewProvince`; `OldDistrict` thêm `province_code`; `OldWard` thêm `district_code`, `province_code`. `*Candidate` gồm unit tương ứng (kèm tỉnh/huyện cha), `mapping_type`, `note`. `name` là tên ngắn không tiền tố ("Tân Định"), `full_name` có tiền tố loại ("Phường Tân Định").

> ⚠️ Not confirmed: tập giá trị `mapping_type` trả ra client — dữ liệu nguồn dùng sáu giá trị (`merged`, `province_changed`, `ambiguous`, `unchanged`, `split_population`, `renamed`); chưa chốt có đưa nguyên xi hay đặt lại tên cho client — chặn DTO ở `tasks.md` Phase 4.

## Hướng sử dụng (quyết định 2026-10-11)

Ý định sản phẩm: **app chỉ cho người dùng chọn địa chỉ hệ mới** (tỉnh → xã). Hệ cũ và các route chuyển đổi vẫn nằm trong API để tham chiếu và dùng về sau (đọc địa chỉ cũ, gắn dữ liệu cũ vào mã mới), nhưng client không gọi chúng. Lý do: hệ mới ít cấp hơn, không có điểm mơ hồ khi chọn, danh sách xã đã đối chiếu khớp với một nguồn độc lập (3.321/3.321 mã, 0 xã lệch tỉnh), còn các sai sót đã biết nằm ở ánh xạ cũ ↔ mới và hệ cũ. Vì danh sách xã của một tỉnh có thể dài tới 168 mục, tìm theo tên (AC 25–26) là phần bắt buộc của luồng này.

## Out of scope (this feature boundary, not tracked here)

- Mọi thứ dưới cấp xã: ấp, thôn, bản, tổ dân phố, số nhà, tên đường là chuỗi tự do do người dùng gõ; thuộc feature dùng địa chỉ (không có metadata cho cấp này).
- Lưu địa chỉ của người dùng/chùa/gia phả/sổ giỗ và chuyển `temples.province` (chuỗi tự do) sang mã đơn vị — mỗi feature dùng địa chỉ có spec riêng.
- Toạ độ, ranh giới, mã bưu chính, geocoding.
- Chuyển đổi **cấp huyện** (người dùng hoãn, 2026-10-10: "sau này tính tiếp"); dữ liệu 5 huyện đảo vẫn nằm trong file nhưng không được nạp và không có route.
- Gắn tra cứu vào dữ liệu đang có (`temples.province`, hồ sơ người dùng, gia phả, sổ giỗ) — đã chốt chỉ làm API + dữ liệu.
- Màn hình backoffice sửa đơn vị bằng tay; đơn vị chỉ đổi khi nạp phiên bản dữ liệu mới.

## Pending (not yet implemented)

Không còn phần implement nào chưa làm trong phạm vi hiện tại (`tasks.md` Phase 1–6 xong). Còn lại: concept OKF `docs/knowledge/features/administrative-units.md` (`il-okf-writer`, chỉ viết khi được yêu cầu) và các mục hoãn có chủ đích (chuyển đổi cấp huyện, đối chiếu file chính thức, đánh dấu hết hiệu lực).

> ⚠️ Not confirmed: màn/field nào dùng tra cứu trước (hồ sơ đăng ký, chùa, gia phả, sổ giỗ) và dòng địa chỉ tự do có bắt buộc/giới hạn độ dài không — quyết định yêu cầu nào chuyển sang feature đó; không chặn feature này.
> ⚠️ Not confirmed: tập giá trị `mapping_type` trả ra client — file dữ liệu giữ nguyên sáu giá trị của nguồn (`merged`, `province_changed`, `ambiguous`, `unchanged`, `split_population`, `renamed`); chưa chốt có đưa nguyên xi hay đặt lại tên cho client — chặn DTO ở `tasks.md` Phase 4.

Đã hoãn có chủ đích (không còn là câu hỏi): đối chiếu với `conversion-table.xlsx` chính thức và dòng "Huyện Kim Sơn → Xã Kim Đông" — người dùng nói "sau này sửa lại cũng được" (2026-10-10); tình trạng ghi trong `meta.source.data_provenance` và `meta.unresolved` của file dữ liệu. Đơn vị bị bỏ khỏi bản dữ liệu mới: lần nạp chỉ upsert nên không xoá; có đánh dấu hết hiệu lực hay không là việc làm sau.

## Ghi chú kỹ thuật liên quan

- Số liệu của bộ dữ liệu ứng viên `vietnam-address-kit` (bản 2026-09-25), đã kiểm tra trong phiên viết SRS: hệ mới 34 tỉnh (25 tỉnh + 9 thành phố), 3.321 xã (2.598 xã, 710 phường, 13 đặc khu), tỉnh lớn nhất Hồ Chí Minh 168 xã, nhỏ nhất 38; hệ cũ 63 tỉnh, 696 huyện (nguồn có 698 dòng, 2 dòng là bản sao do lỗi nhãn đã được loại), 10.033 xã, huyện lớn nhất 47 xã; 10.565 mapping cấp xã, và 5 mapping cấp huyện cho huyện không có xã (nguồn có 6 dòng cấp huyện, dòng thứ sáu là bất thường Kim Sơn); 441 xã cũ ứng với nhiều hơn một xã mới (~4,4%); 5 huyện không có xã (Bạch Long Vĩ, Cồn Cỏ, Hoàng Sa, Lý Sơn, Côn Đảo).
- Tên 34 tỉnh của bộ dữ liệu trùng khít `domain/province.go` (đã so sánh), nên `temples.province` hiện có thể nối sang `code` mới mà không đổi tên. Lưu ý: bộ dữ liệu ghi 25 tỉnh + 9 thành phố (sau điều chỉnh 2026), còn comment của `province.go` ghi 28 tỉnh + 6 thành phố — chỉ lệch loại, không lệch tên.
- Mã (`code`) hai hệ có thể trùng số nhưng khác nghĩa (vd. cùng một mã là phường Đa Kao cũ và phường Tân Định mới), nên lưu hai hệ ở bảng riêng và route luôn mang tên hệ.
- Middleware web API key (`delivery/http/web_api_key_middleware.go`): header vắng → đi qua và coi là mobile; header đúng → đi qua kèm cờ đã xác thực; header sai/rỗng → `401 INVALID_API_KEY`. Đây là bí mật dùng chung cho ứng dụng, không phải xác thực người dùng, và không chặn người gọi bỏ header (`docs/tech-debt/auth/TD-001`).
- Cùng hình dạng danh mục công khai với `temple`/`deity`, khác ở chỗ route chuyển đổi không phân trang và nằm sau `webAPIKey`.
- **File dữ liệu duy nhất:** `assets/data/administrative-units.json` (khoảng 3,2 MB, JSON thuần, mỗi phần tử một dòng để diff được). Khoá cấp một: `meta`, `new_provinces`, `new_wards`, `old_provinces`, `old_districts`, `old_wards`, `ward_mappings`, `district_mappings`. `meta` ghi nguồn (repo, commit, giấy phép MIT của mã), số lượng, `corrections` (2 xã gắn sai tỉnh được sửa + 2 dòng huyện nhân đôi bị loại) và `unresolved` (dòng Kim Sơn). Mỗi đơn vị có `sort_order` đã tính sẵn theo thứ tự chữ cái tiếng Việt nên không phụ thuộc collation của Postgres. Seed lại bằng cách chạy lại `cmd/seedadminunits -file assets/data/administrative-units.json`.
- Dựng lại file dữ liệu: `scripts/build_administrative_units.py` (cách dùng ở docstring); `.gitignore` cho phép `assets/data/administrative-units.json` nên file được commit cùng repo.
- CLI nạp dữ liệu: `go run ./services/appService/cmd/seedadminunits [-file assets/data/administrative-units.json] [-dry-run]`. `-dry-run` (thêm ở v1.6, ngoài danh sách AC) chỉ đọc, kiểm cấu trúc và toàn vẹn rồi in số dòng, không cần database — để kiểm một file trước khi nạp. Cấu hình DB lấy từ `CONFIG_PATH` (mặc định `config.dev.yaml`, `⚠️ DEV ONLY` như `seedtemples`).
- Chỉ tiêu phi chức năng đã chốt (2026-10-10): p95 ≤ 200 ms (đo thực tế 1–3 ms cho chuyển đổi trên Postgres local); client cache được 24 giờ (AC 24, `administrative_units.cache_max_age`); giới hạn tần suất theo IP (AC 23, `administrative_units.rate_limit`, dev 60 lượt/phút, **một ngân sách chung** cho cả chín route).

## Open questions (resolved during implementation)

Đã chốt ở v1.0 (2026-10-10), từ phiên viết SRS:

- ~~Route tra cứu công khai hay cần đăng nhập?~~ → **Công khai**, cho cả màn đăng ký lẫn trong app (người dùng, 2026-10-10). Ảnh hưởng: AC 16.
- ~~Có cần API key không?~~ → **Có, web API key** như route đăng ký và `temple-suggestions`: header đúng hoặc vắng thì phục vụ, sai thì `401` (người dùng, 2026-10-10). Ảnh hưởng: AC 17; `cmd/main.go` bọc `webAPIKey.WrapFunc`.
- ~~Chọn tỉnh thì ra danh sách xã để chọn?~~ → **Có** (người dùng, 2026-10-10). Ảnh hưởng: AC 2, 6.
- ~~Dưới cấp xã có metadata không?~~ → **Không**, là chuỗi người dùng nhập (2026-10-10). Ảnh hưởng: phạm vi "Out of scope".
- ~~Tỉnh cũ có thể ứng với hai tỉnh mới không (Tiền Giang)?~~ → **Không.** Dòng "Tiền Giang → Tây Ninh" là lỗi nhãn của một xã thuộc Tân An (Long An); mỗi tỉnh cũ ứng đúng một tỉnh mới (2026-10-10, kiểm tra trên dữ liệu). Ảnh hưởng: AC 12; chú thích "không tỉnh nào bị tách" của migration `0025` vẫn đúng.
- ~~Chuyển đổi cấp tỉnh có cần cờ "không chắc chắn"?~~ → **Không**: cũ → mới luôn một kết quả, mới → cũ luôn là danh sách (2026-10-10). Ảnh hưởng: AC 12.

Đã chốt ở v1.1 (2026-10-10), trả lời đợt câu hỏi sau khi có bản v1.0:

- ~~Đường dẫn route?~~ → **Đồng ý** như đề xuất `/v1/administrative-units/...`. Ảnh hưởng: bảng API contract; marker đường dẫn được đóng.
- ~~Hai xã bị gắn sai tỉnh xử lý thế nào?~~ → **Cách A**: sửa trong file dữ liệu và ghi chú vào `meta.corrections` (Phường 4 thuộc Thành phố Tân An → Long An; Xã Tiên Hải thuộc Thành phố Hà Tiên → Kiên Giang). Ảnh hưởng: `Dataset.Verify` không cần vá gì; `tasks.md` Phase 5 mục lỗi nhãn đã xong.
- ~~Có đối chiếu với file chính thức trước production không?~~ → **Không cần bây giờ**, "sau này sửa lại cũng được". Ảnh hưởng: marker được đóng; ghi trạng thái "chưa đối chiếu" trong `meta`; dòng Kim Sơn không được nạp.
- ~~Giới hạn tần suất?~~ → **Đồng ý theo đề xuất** (theo IP như `temple-suggestions`). Ảnh hưởng: AC 23; `design.md` (limiter sau `webAPIKey`, block config); `tasks.md` Phase 4.
- ~~Thứ tự danh sách?~~ → **Đồng ý theo đề xuất** (bảng chữ cái tiếng Việt). Ảnh hưởng: AC 1, 2, 4–6; cột `sort_order` trong file dữ liệu và bảng.
- ~~Nhiều ứng viên thì hiển thị thế nào?~~ → **Đồng ý**: client hiện tất cả để người dùng chọn. Ảnh hưởng: AC 11 (đã đúng); marker được đóng.
- ~~Phạm vi?~~ → **A**: chỉ API + dữ liệu. Ảnh hưởng: "Out of scope".
- ~~Ai nạp bản mới?~~ → **Chạy tay khi có cập nhật**. Ảnh hưởng: không cần cơ chế tự động; đơn vị vắng không bị xoá (chỉ upsert).
- ~~Độ trễ và cache?~~ → **Đồng ý** p95 ≤ 200 ms, cache 24 giờ. Ảnh hưởng: AC 24.
- ~~Chuyển đổi cấp huyện?~~ → **Tạm bỏ qua**. Ảnh hưởng: AC 13 hoãn; bỏ route và bảng `addr_district_mappings` khỏi phiên bản này.
- Người dùng yêu cầu **một file JSON duy nhất** để seed lại DB dễ dàng. Ảnh hưởng: `design.md` (cờ `-file`, một file thay cho sáu); file đã tạo.

Đã chốt ở v1.8 (2026-10-11):

- ~~App cho người dùng chọn hệ nào?~~ → **Chỉ hệ mới.** Hệ cũ và chuyển đổi giữ trong API, client không dùng (người dùng, 2026-10-11). Ảnh hưởng: mục "Hướng sử dụng"; ưu tiên tìm theo tên lên trước, đối chiếu file chính thức và chuyển đổi cấp huyện xuống thấp.
- ~~Có cần tìm xã theo tên không dấu, phạm vi?~~ → **Có, trong một tỉnh của hệ mới**, không phân biệt hoa thường, dấu, vị trí dấu thanh và kiểu dấu nháy (người dùng, 2026-10-11). Ảnh hưởng: AC 25–26; `GET …/new/provinces/{code}/wards?q=`; `unaccent` (migration `0019`) + `translate` cho dấu nháy; marker `?q=` được đóng.
