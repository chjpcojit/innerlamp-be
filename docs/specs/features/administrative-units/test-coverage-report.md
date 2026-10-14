# Test Coverage Report — administrative-units

AC coverage audits and E2E wiring findings for this feature, run by `golang-e2e-verification`.
Newest run first. This file is an audit log, not part of the requirements→design→tasks cascade —
it needs no version number and no `changelog.md` entry of its own.

## 2026-10-11 (bổ sung — tìm xã theo tên)

Sau quyết định "app chỉ cho chọn hệ mới", thêm `GET …/new/provinces/{code}/wards?q=` (AC 25–26, `requirements.md` v1.8).

### AC coverage

| AC | Requirement (short) | Covered by | Status |
|----|---------------------|-----------|--------|
| 25 | `q` narrows the wards of that province; tolerant of case, diacritics, tone-mark position, apostrophes; no match → 200 `[]`; unknown province → 404; no `q` = AC 2 | E `…/ward_search_inside_a_province…` (dữ liệu thật: `Hóa`/`Hoá`/`HÓA`/`hoá` cùng một tập ở Thanh Hóa, `b'lao`/`b’lao` tìm ra "B’Lao" ở Lâm Đồng, `phuong tan dinh`, không khớp, tỉnh lạ, không `q`); R `TestAdministrativeUnitRepository_SearchCurrentWards` (Postgres thật: `Hòa`↔`Hoà` hai chiều, dấu nháy cong/thẳng hai chiều, `đ`↔`d`, `a_b` nghĩa đen, tiền tố loại, ràng buộc tỉnh, thứ tự `sort_order`); H `CurrentWardsSearch` (q có mặt → Search, vắng → List, không khớp, tỉnh lạ); U `SearchCurrentWards` (trim, NFC) | ✅ covered |
| 26 | Present-but-blank or over-long `q` → 400 `INVALID_FILTER`, field `q`, no cache header, no query | E (`""`, `"   "`, 101 ký tự → 400, `details.field = q`, không `Cache-Control`); H `CurrentWardsSearch/an_empty_q…`; U (rỗng, trắng, 101 rune → lỗi và `AssertNotCalled`; đúng 100 rune được nhận, đếm ký tự không phải byte) | ✅ covered |

### E2E findings

None this run. Một lỗi do chính em gây ra ở lớp test, đã bắt và sửa trước khi tính là xong: kỳ vọng sai trong test repository
("phuong thang" không phải chuỗi con của "Phường Hòa Thắng") — là lỗi của test, không phải của mã.

### Follow-ups

- Tìm là *chứa chuỗi con*: không xếp theo độ liên quan, không sửa lỗi gõ sai, chỉ tìm trong một tỉnh. Nếu client cần tìm xuyên tỉnh thì là yêu cầu mới.
- Phụ thuộc `unaccent` (migration `0019`); đã có ở mọi môi trường vì `sutra` và `temple` đã dùng.

## 2026-10-11

Spec `requirements.md` v1.6 (AC 1–24, AC 13 hoãn). Thứ tự tìm: `usecase` → `repository` → `delivery/http` → `e2e`.
Ký hiệu: **U** usecase, **R** repository (Postgres thật), **H** handler, **E** E2E (binary thật + Postgres testcontainers),
**D** domain, **C** config, **P** parse (`cmd/seedadminunits`).

### AC coverage

| AC | Requirement (short) | Covered by | Status |
|----|---------------------|-----------|--------|
| 1 | New-scheme provinces: all 34, fields, stable alphabetical order | E `administrative_unit_e2e_test.go` `…/new_scheme:_provinces…` (34, "An Giang" đầu, không có "Bình Dương"); H `TestAdministrativeUnitHandler_Lists` (fields, no `sort_order`); R `TestAdministrativeUnitRepository_ListsCurrentScheme` (order by `sort_order`, not code) | ✅ covered |
| 2 | Wards of a province in one response, only that province | E (168 wards of Hồ Chí Minh, all `province_code` = 79, "Thủ Dầu Một", "Vũng Tàu"); R `ListsCurrentScheme` | ✅ covered |
| 3 | Unknown new province → 404 `ADMINISTRATIVE_UNIT_NOT_FOUND`, no partial list | E (404 + no `Cache-Control`); H `NotFoundOnEveryListRoute`; R `ListsCurrentScheme` | ✅ covered |
| 4 | Old-scheme provinces: all 63 | E `…/old_scheme:…` (63); R `ListsLegacyScheme` | ✅ covered |
| 5 | Districts of an old province, none of another | E (22 districts of Hồ Chí Minh, "Bình Chánh" đầu); R `ListsLegacyScheme` | ✅ covered |
| 6 | Wards of a district; district without wards → 200 `[]` | E (district 318 → `{"items":[]}`); H `EmptyListIs200WithEmptyArray`; R `ListLegacyWards_DistrictWithoutWardsIsEmptyNotMissing`; U `ListLegacyWards` | ✅ covered |
| 7 | Unknown old province/district → 404 | E (district `no-such`); H `NotFoundOnEveryListRoute`; U `CodeMethods/not_found…` | ✅ covered |
| 8 | Two schemes keep separate namespaces | E `…/the_same_code_in_both_schemes…` (26737 = Đa Kao cũ và Tân Định mới); R `SameCodeInBothSchemesStaysSeparate` | ✅ covered |
| 9 | Old ward → new ward(s) with province, `mapping_type`, `note` | E `…/conversion_old_→_new…`; H `ConvertLegacyWard`; R `ConvertLegacyWard` | ✅ covered |
| 10 | New ward → old wards with district and province | E `…/conversion_new_→_old…` (4 xã cũ của Cầu Ông Lãnh); H `ConvertCurrentWard`; R `ConvertCurrentWard` | ✅ covered |
| 11 | Several candidates → all returned, `unique=false`, none chosen | E (26737 → 2 `ambiguous`); U `ConversionUniqueness`, `ConversionKeepsCandidateDetail`; H `ConvertLegacyWard/several_candidates…` | ✅ covered |
| 12 | Province conversion derived from wards: old → exactly one, new → list | E `…/province_conversion_is_derived…` (Tiền Giang → chỉ Đồng Tháp; Hồ Chí Minh ← 3); R `ProvinceConversionIsDerivedFromWards`; D `Verify` rule `legacy_province_split` | ✅ covered |
| 13 | District-level conversion | — | ⏸ deferred by the owner (2026-10-10); not built |
| 14 | Unknown unit in a conversion → 404 | E (ward and province `no-such`); H `ConvertLegacyWard/not_found…`; U `CodeMethods` | ✅ covered |
| 15 | Conversions read only, no write, no external call | E `…/reading_and_converting_write_nothing…` (row counts unchanged after every read); R/H are SELECT-only by construction | ✅ covered (no-external-call is by construction: no HTTP client in the layers) |
| 16 | No access token needed | E `…/access…` (no `Authorization` header sent on any call); H `ComposedLikeMain/needs_no_access_token` | ✅ covered |
| 17 | Web key: correct / absent served, wrong → 401 `INVALID_API_KEY` | E `…/access…` (the dev key read from config; wrong key → 401, no `Cache-Control`); H `ComposedLikeMain` (5 cases) | ✅ covered |
| 18 | Every response in the shared envelope | E `getKeyed` decodes the envelope on every call, incl. 404/405/429; H asserts `success` | ✅ covered |
| 19 | Seeder loads all tables + version in one transaction; never deletes | E `…/CLI:_load_fills_every_table…` (counts 34/3.321/63/696/10.033/10.565, 1 version); R `ReplaceDataset_IsAtomic`, `UpdatesInPlaceAndNeverDeletes` | ✅ covered |
| 20 | Loading the same dataset again changes nothing | E (second CLI run, same counts, still 1 version); R `ReplaceDataset_IsIdempotent` | ✅ covered |
| 21 | Dataset violating an integrity rule refused whole, report, nothing written | E `…/CLI:_a_corrupted_file…` (exit 1, names `ward_province_mismatch` + `27691`, DB empty); D `Verify_ReportsEachRule` (12 cases, one per rule); P `RefusesAMalformedFile` (structure) | ✅ covered |
| 22 | Seeder prints per-table counts | E dry-run asserts the "34 provinces" line and the load asserts the success line; counts themselves asserted against the DB | ✅ covered (the printed per-table numbers are asserted only for the provinces line — see follow-ups) |
| 23 | Per-IP limit, 429 `RATE_LIMITED`, key check first | E `TestAdministrativeUnits_RateLimit_E2E` (config budget 3 shared across routes, refused keys spend nothing, 429 has no `Cache-Control`); H `ComposedLikeMain`; C `TestAdministrativeUnitsConfig_Validate`, `TestLoad_DevYAMLAdministrativeUnitsConfig` | ✅ covered |
| 24 | `Cache-Control: public, max-age=<config>` on success, not on errors | E (`max-age=86400` on 200, absent on 401/404/429); H `Lists`, `CacheControlFollowsConfig` (90 s), 404/500 without header; C | ✅ covered |

### E2E findings

Không có lỗi wiring nào lọt qua. Hai bài E2E (`TestAdministrativeUnits_E2E`, `TestAdministrativeUnits_RateLimit_E2E`)
xanh ngay lần chạy đầu với binary thật `cmd/` và `cmd/seedadminunits` trên Postgres 16 testcontainers:
chín route đều được đăng ký, web key và limiter đứng trước handler đúng thứ tự, `cache_max_age` và
`rate_limit` của config tới được tiến trình đang chạy.

Những gì per-layer test **không** chứng minh được và E2E đã chứng minh: `main.go` thật sự đăng ký đủ chín
đường dẫn và bọc `webAPIKey` rồi limiter; ngân sách giới hạn **dùng chung** cho cả nhóm route; dữ liệu do CLI
nạp là dữ liệu mà route đọc; CLI từ chối file hỏng mà không ghi gì (kiểm trên DB thật, không mock).

Nhắc lại giới hạn: cả hai bài **skip** khi `assets/data/administrative-units.json` không có trong checkout
(bài chính) hoặc khi không có Docker (cả hai). Tệp dữ liệu nay đã được `.gitignore` cho phép (`!assets/data/administrative-units.json`)
nên trên một checkout đầy đủ bài chạy thật.

### Follow-ups

- AC 22: assertion hiện chỉ khớp dòng "34 provinces" và dòng thành công; số liệu từng bảng được kiểm trực tiếp
  trên DB. Muốn chặt hơn có thể assert cả chuỗi "3321 wards" / "10565" trong stderr — mức độ thấp, thuộc test CLI.
- AC 13 (chuyển đổi cấp huyện) hoãn; khi làm, thêm test ở cả `repository` và E2E và quyết định lại hành vi của
  5 đặc khu đảo (hiện trả `candidates: []`, đã có test khoá hành vi này ở R, H và E).
- Không có gap nào thuộc về lớp khác cần chuyển cho `golang-clean-arch-tdd`.
