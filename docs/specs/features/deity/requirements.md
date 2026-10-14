# Requirements — deity

Version: 2.3
Last updated: 2026-10-12
Status: implemented (2026-09-30) — 3 route đọc công khai (`top`, danh sách phân trang gộp Phật + Bồ tát, `detail`); v2.0 đổi danh sách phân trang từ `/v1/bodhisattvas` (chỉ Bồ tát) sang `/v1/deities` (cả hai loại), đã có code + unit test; chưa chạy lại trên binary + Postgres dev. **v2.2–2.3 (2026-10-12): tham số lọc `type` cho `GET /v1/deities` (AC 20–22) đã implement: code + unit test + repository test (DB `innerlamp_test`) + E2E `deity_e2e_test.go` pass.** Các route còn lại **chưa có** E2E. Còn 4 marker `⚠️ Not confirmed`.
Module: `content`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này được viết **retroactively** từ danh mục Phật / Bồ tát đã implement (Retroactive mode). Mọi AC dưới đây mô tả đúng hành vi code hiện có; thứ chưa làm nằm ở "Pending", không gộp vào AC. `api-contract.md` cùng thư mục là tài liệu **dành cho client iOS** (viết trước spec này, giữ nguyên); khi hai file lệch nhau thì code là nguồn sự thật và spec này được sửa theo code.

Đây là danh mục **chỉ đọc**, công khai, cùng hình dạng với `temple` và `article`, nhưng có một điểm riêng: danh sách "nổi bật" trộn cả Phật và Bồ tát, còn danh sách phân trang (`/v1/deities`) cũng trộn cả Phật và Bồ tát (cùng một bảng, phân biệt bằng trường `type`), nên Phật không còn chỉ xem được qua `top` hoặc trang chi tiết. Dữ liệu nạp bằng lệnh `cmd/seeddeities` chạy tay, không tự nạp khi service khởi động.

> ⚠️ Not confirmed: mã user story của tính năng này chưa được product chốt — spec tạm đánh `US-DTY-01..05` theo tiền lệ `US-ART-01` của `article`; đổi khi product chốt số.

---

## US-DTY-01: Xem Phật và Bồ tát nổi bật

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **thấy vài vị Phật và Bồ tát tiêu biểu ngay ở màn hình chủ**,
> Để **làm quen và bấm vào tìm hiểu thêm**.

## Acceptance Criteria (EARS notation)

1. WHEN a `GET /v1/deities/top` request arrives, THE SYSTEM SHALL return `200 OK` with `data.items` holding the first `deity.top_limit` deities of **any type** (dev: 10) ordered by `sort_order` ASC then `id` ASC, so Phật and Bồ tát may appear together. (Không cần `Authorization`.)
2. WHEN deities are returned by `top`, THE SYSTEM SHALL return each one in the **summary** shape only — `id`, `type`, `name_vi`, `name_sanskrit`, `name_chinese`, `title_honorific`, `thumbnail_url`, `image_url`, `core_virtue`, `pure_land` — and SHALL NOT include `canonical_source`, `lunar_memorial_days`, `iconography`, `biography_summary`, `great_vows` or `mantras`.
3. WHEN the table holds fewer deities than `deity.top_limit`, or none (chưa chạy seed), THE SYSTEM SHALL return all that exist, and SHALL return `items: []` (không phải `null`) for an empty table.

## US-DTY-02: Duyệt danh sách Phật và Bồ tát theo trang

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **cuộn qua toàn bộ các vị Phật và Bồ tát theo từng trang**,
> Để **tìm hiểu các vị ngoài phần nổi bật**.

## Acceptance Criteria (EARS notation)

4. WHEN a `GET /v1/deities` request arrives without `page` (or with an empty `page=`) and without `type` (or with an empty `type=`), THE SYSTEM SHALL serve page 1 of the deities of **any type** (`BUDDHA`, `BODHISATTVA` and `DEVA` together, no type filter), ordered by `sort_order` ASC then `id` ASC. Mỗi item mang trường `type` để client phân biệt.
5. WHEN a page is served, THE SYSTEM SHALL return `data` with `items` (summary shape), `page`, `page_size` (`deity.page_size`, dev: 20), `total` (số mục khớp bộ lọc `type` nếu có, ngược lại số mục của cả bảng) and `total_pages` = `ceil(total / page_size)`.
6. WHEN `page` is greater than `total_pages`, THE SYSTEM SHALL return `200 OK` with `items: []` and the same `page_size`/`total`/`total_pages` — không phải lỗi. Client dừng khi `page >= total_pages` hoặc `items` rỗng.
7. WHEN `page` is not a positive integer (`0`, negative, non-numeric, an integer that overflows, or larger than 2^31−1), THE SYSTEM SHALL return `400 INVALID_PAGE` with `details.field = "page"`.
20. WHEN a `GET /v1/deities?type=T` request arrives with `T` exactly one of `BUDDHA`, `BODHISATTVA`, `DEVA`, THE SYSTEM SHALL return only deities of that type, in the same order (`sort_order` ASC then `id` ASC), and `total` / `total_pages` SHALL count only that type.
21. WHEN `type` is present and non-empty but not exactly one of the three values (kể cả khác hoa/thường như `buddha`), THE SYSTEM SHALL return `400 INVALID_DEITY_TYPE` with `details.field = "type"`.
22. WHEN `type` is combined with `page`, THE SYSTEM SHALL page within the filtered list (offset = `(page-1) * page_size` trên danh sách đã lọc); WHEN both `type` and `page` are invalid, THE SYSTEM SHALL report `INVALID_PAGE` first (kiểm `page` trước `type`, cùng thứ tự với `gallery`).

> Ghi chú đánh số: AC 20–22 được thêm ở v2.2 và nối đuôi để các tham chiếu tới AC 1–19 ở nơi khác (vd. AC 3, 6, 13) không bị lệch; chúng thuộc US-DTY-02.

## US-DTY-03: Xem chi tiết một vị Phật hoặc Bồ tát

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **mở một vị Phật hay Bồ tát để đọc đầy đủ**,
> Để **biết hạnh nguyện, ngày vía, hình tượng và câu niệm của vị đó**.

## Acceptance Criteria (EARS notation)

8. WHEN a `GET /v1/deities/{id}` request arrives with an existing `id` of **either type**, THE SYSTEM SHALL return `200 OK` with the summary fields plus `canonical_source`, `lunar_memorial_days` (`lunar_day`, `lunar_month`, `event_name`), `iconography` (`posture`, `mount_or_throne`, `sacred_artifacts`, `symbolic_meaning`), `biography_summary`, `great_vows` and `mantras` (`short_mantra_sanskrit`, `short_mantra_sino_vietnamese`, `long_dharani_name`, `dharani_phonetic_snippet`).
9. WHEN `{id}` matches no deity, THE SYSTEM SHALL return `404 DEITY_NOT_FOUND`.
10. WHEN the literal path `/v1/deities/top` is requested, THE SYSTEM SHALL route it to the top list, not to detail (`ServeMux` ưu tiên segment literal). `id` của mục có dạng `buddha_*` / `bodhisattva_*` nên không trùng `top`.

## US-DTY-04: Hình dạng dữ liệu nhất quán

## User Story

> Là một **lập trình viên client**,
> Tôi muốn **các field thiếu dữ liệu luôn có dạng xác định**,
> Để **decode một lần mà không phải đoán `null` hay chuỗi rỗng**.

## Acceptance Criteria (EARS notation)

11. WHEN a deity has no value for a text field (`name_sanskrit`, `name_chinese`, `title_honorific`, `core_virtue`, `pure_land`, `canonical_source`, `biography_summary`, the iconography and mantra strings), THE SYSTEM SHALL return `""`, never `null`.
12. WHEN a deity has no memorial days, sacred artifacts or great vows, THE SYSTEM SHALL return `[]` for `lunar_memorial_days` / `iconography.sacred_artifacts` / `great_vows`, never `null`.
13. WHEN `thumbnail_url` / `image_url` are returned, THE SYSTEM SHALL pass the stored strings through **unchanged** — kể cả đường dẫn tương đối hoặc host giữ chỗ trong dữ liệu nguồn — and SHALL NOT convert an empty value to `null` (khác `temple` và `article`).
14. WHEN any deity route fails, THE SYSTEM SHALL answer in the shared envelope `{"success": false, "error": {code, message, type}, "details"?}`; a non-`GET` request returns `405 METHOD_NOT_ALLOWED` (kèm header `Allow`), an unknown path returns `404 NOT_FOUND`, and an infrastructure failure returns `500 INTERNAL_ERROR` without leaking the cause.

## US-DTY-05: Nạp dữ liệu Phật / Bồ tát bằng lệnh seed

## User Story

> Là một **người vận hành hệ thống**,
> Tôi muốn **nạp và cập nhật danh mục từ một file dữ liệu bằng một lệnh**,
> Để **đổi thứ tự hoặc nội dung trong file rồi chạy lại mà không tạo trùng và không làm hỏng dữ liệu nửa chừng**.

## Acceptance Criteria (EARS notation)

15. WHEN the operator runs `go run ./services/appService/cmd/seeddeities [-file <path>]`, THE SYSTEM SHALL read the file (mặc định `assets/data/phat_botat.json`, tính từ thư mục đang đứng), validate it, and upsert every deity in **one transaction** keyed by `id`, with `sort_order` equal to the deity's position in the file (0, 1, 2, …), then log `seeded N deities from <file>`.
16. WHEN the seed is validated, THE SYSTEM SHALL reject the whole file (write nothing) if it has no deity, an empty or duplicate `id`, a `type` other than `BUDDHA` / `BODHISATTVA` / `DEVA`, or an empty `name_vi`.
17. WHEN the seed is run again after the file was edited or reordered, THE SYSTEM SHALL update the existing rows in place by `id` — overwriting **every** column from the file, including `sort_order`, and setting `updated_at = now()` — and SHALL NOT create a second row for an existing `id`.
18. WHEN the file is invalid, the database is unreachable, or the database rejects a row (vd. migration `0007` chưa apply), THE SYSTEM SHALL exit non-zero with a fatal log and SHALL leave the `deities` table unchanged (transaction rolled back).
19. WHEN appService starts, THE SYSTEM SHALL NOT seed or otherwise write the `deities` table — một DB chưa chạy seed trả `items: []` (AC 3, 6), không phải lỗi.

## API contract (confirmed surface for this feature)

Đọc từ handler, `cmd/main.go`; đã có trong `docs/specs/api/openapi.yaml` (tag Deities). `api-contract.md` ghi 3 route này đã được xác nhận trên server dev với 18 mục.

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/deities/top` | public | `200` `{items: [DeitySummary]}` (Phật và Bồ tát trộn) | 404 `NOT_FOUND` (sai path), 405, 500 |
| GET | `/v1/deities?type=&page=N` | public | `200` `{items, page, page_size, total, total_pages}` (trộn các loại nếu không có `type`; lọc một loại nếu có) | 400 `INVALID_PAGE`, 400 `INVALID_DEITY_TYPE`, 405, 500 |
| GET | `/v1/deities/{id}` | public | `200` `DeityDetail` | 404 `DEITY_NOT_FOUND`, 405, 500 |

`DeitySummary`: `id` (vd. `buddha_amitabha`, `bodhisattva_avalokiteshvara`), `type` (`BUDDHA` \| `BODHISATTVA` \| `DEVA`), `name_vi`, `name_sanskrit`, `name_chinese`, `title_honorific`, `thumbnail_url` (string), `image_url` (string), `core_virtue`, `pure_land`.

`DeityDetail` = `DeitySummary` + `canonical_source`, `lunar_memorial_days` (`[{lunar_day, lunar_month, event_name}]`), `iconography` `{posture, mount_or_throne, sacred_artifacts: [string], symbolic_meaning}`, `biography_summary`, `great_vows` (`[string]`), `mantras` `{short_mantra_sanskrit, short_mantra_sino_vietnamese, long_dharani_name, dharani_phonetic_snippet}`.

## Out of scope (this feature boundary, not tracked here)

- Tạo/sửa/xoá mục qua API hoặc backoffice — hiện chỉ có `cmd/seeddeities` và SQL trực tiếp. `admin/requirements.md` ghi việc quản lý temple/deity qua backoffice sẽ có spec riêng.
- Tìm kiếm theo từ khoá, yêu thích; mọi bộ lọc khác ngoài `type` (AC 20–22); lọc nhiều `type` cùng lúc (`type=BUDDHA,DEVA`); tính ngày vía theo lịch âm (bảng chỉ lưu `lunar_day`/`lunar_month`, không có phép tính ngày dương).
- Lưu trữ/phân phối ảnh: chỉ có hai cột URL, trỏ vào dữ liệu nguồn (xem Pending).
- Đăng nhập: cả ba route công khai như `temple` và `article`.

## Pending (not yet implemented)

- Test E2E chạy binary thật + Postgres cho ba route — `tasks.md` Phase 5.
- OKF concept `docs/knowledge/features/deity.md` — chưa có; do `bdd-okf-docs` tạo sau khi có E2E, `tasks.md` Phase 5.
- Ảnh dùng được: `thumbnail_url` là đường dẫn tương đối (`assets/images/buddhas/thumbnails/<id>.webp`) và `image_url` là host giữ chỗ (`https://cdn.yourapp.com/images/buddhas/full/<id>.webp`) — cả hai chưa phân giải được — `tasks.md` Phase 3.

> ⚠️ Not confirmed: tiêu chí "nổi bật" — nguồn không có độ phổ biến, nên thứ tự trong `assets/data/phat_botat.json` (6 Phật, rồi Bồ tát theo mức nổi bật do biên tập) **chính là** xếp hạng; muốn đổi thì sửa file rồi seed lại. Chưa xác nhận với product rằng `top` 10 mục (6 Phật + 4 Bồ tát) là đúng ý.
> ⚠️ Not confirmed: nguồn ảnh và nơi lưu — chưa chốt ảnh thật nằm ở đâu (CDN nào, hay chép vào GCS), ai cấp file, quyền sử dụng; đến lúc đó phải thay hai URL trong file dữ liệu rồi seed lại.
> ⚠️ Not confirmed: quy trình nạp dữ liệu ở môi trường thật — `cmd/seeddeities` đang dùng config và secret **dev** (`config.dev.yaml`, `NewMockSecretGetterFromYAML`, đánh dấu `⚠️ DEV ONLY`), và file JSON đọc từ đĩa chứ không nhúng vào binary; chưa chốt ai chạy seed ở staging/production và bằng công cụ nào.
> ⚠️ Not confirmed: nguồn và độ chính xác nội dung tôn giáo — `canonical_source`, `great_vows`, `mantras`, `lunar_memorial_days` lấy từ file dữ liệu và chưa có bước rà soát chính thức (vd. sai một ngày vía hay một câu niệm là lỗi nội dung nhạy cảm); chưa chốt ai duyệt và theo quy trình nào.

## Ghi chú kỹ thuật liên quan

- Dữ liệu seed (`assets/data/phat_botat.json`, 18 mục): 6 `BUDDHA` (`buddha_shakyamuni`, `buddha_amitabha`, `buddha_bhaisajyaguru`, `buddha_maitreya`, `buddha_vairocana`, `buddha_dipankara`) đứng trước, rồi 12 `BODHISATTVA` (bắt đầu bằng `bodhisattva_avalokiteshvara`, `bodhisattva_manjushri`). Cả 18 mục đều có `thumbnail_url` và `image_url` khác rỗng, nhưng chưa phân giải được (xem Pending). `top` mặc định 10 mục là 6 Phật + 4 Bồ tát; `/v1/deities` có `total = 18` và với `page_size = 20` chỉ có 1 trang (6 Phật rồi 12 Bồ tát).
- `id` là chuỗi ổn định lấy từ dữ liệu nguồn nên id client đã cache vẫn đúng sau mọi lần nạp lại; seeder upsert theo `id`.
- `sort_order` là vị trí trong file (bắt đầu từ 0) và là **nguồn xếp hạng duy nhất**; `id` là tie-break cuối để hai dòng cùng `sort_order` không đổi chỗ giữa các trang. Index `idx_deities_order` phục vụ `top` và danh sách phân trang; `idx_deities_type_order (type, sort_order, id)` hiện chưa route nào dùng nhưng phục vụ bộ lọc `type` (AC 20).
- `lunar_memorial_days` lưu dưới dạng JSONB (danh sách ngắn, luôn đọc trọn), chưa có truy vấn nào nhìn vào bên trong nó; `sacred_artifacts` và `great_vows` là `TEXT[]`.
- `total` và trang được đọc bằng hai câu lệnh riêng (đếm cả bảng rồi lấy trang), không cùng một snapshot — xem `design.md`.
- Kích thước trang nằm ở block `deity:` của `config.<env>.yaml` (dev: `top_limit: 10`, `page_size: 20`); cả hai bắt buộc — giá trị 0 nghĩa là trang rỗng.
- Bảng `deities` phải tồn tại trước khi service chạy (`make db-up` áp `0007_deities.sql`).

## Open questions (resolved during implementation)

- 2026-10-12 (v2.2): **Có API lọc Phật / Bồ tát theo `type` không?** — chưa có (handler chỉ đọc `page`). Người dùng yêu cầu bổ sung. Các chi tiết người dùng chưa nêu được tự quyết theo tiền lệ `gallery` và ghi lại để duyệt: (a) tham số query `type` trên route sẵn có `GET /v1/deities`, không thêm route mới; (b) giá trị khớp chính xác, phân biệt hoa/thường, đúng ba giá trị `BUDDHA` / `BODHISATTVA` / `DEVA` (cùng giá trị với field `type` trả về); (c) bỏ trống hoặc `type=` rỗng = không lọc, nên client cũ không đổi hành vi; (d) giá trị lạ = `400 INVALID_DEITY_TYPE`, `details.field = "type"`; (e) một giá trị mỗi lần. Ảnh hưởng: `design.md` (thêm tham số cho usecase/repository, câu SQL có `WHERE type = $1`), `flow.md`, `tasks.md` Phase 7. Nếu product muốn nhận cả chữ thường hoặc nhiều giá trị, đổi AC 21 trước rồi cascade xuống.

- 2026-09-30 (v2.0): **Cách xem Phật** — trước đây chỉ có `top` và `detail`, không có route danh sách Phật. Người dùng yêu cầu API phân trang gộp cả Phật và Bồ tát (cùng một bảng), nên `GET /v1/bodhisattvas` được thay bằng `GET /v1/deities?page=N` (không lọc loại). Ảnh hưởng: `design.md` (bỏ `ListByType`, thay bằng `List`), `flow.md` (bỏ nhánh lọc theo loại), `tasks.md` (đổi tên/nội dung mục usecase, handler, repository, E2E). Route cũ bị xoá, không giữ alias — client iOS phải đổi sang `/v1/deities`.
