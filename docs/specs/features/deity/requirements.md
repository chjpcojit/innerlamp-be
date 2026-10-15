# Requirements — deity

Version: 3.0
Last updated: 2026-10-15
Status: in-progress — AC 1–22 (trừ phần đánh dấu **v3.0**) đã implement (top / danh sách phân trang có lọc `type` / chi tiết, seed `cmd/seeddeities`); E2E mới có cho bộ lọc `type`. **v3.0 (2026-10-15) chưa implement**: trang 12, ngày vía lấy từ lịch lễ (`calendar`) với lần vía kế tiếp do server tính, bỏ `lunar_memorial_days` lưu riêng, đưa file danh mục vào git (AC 5, 8, 12, 15, 16, 23–34; `tasks.md` Phase 8–10).
Module: `deity`
Code: `services/appService/`
Product spec: `docs/product/deity/product.md` @ v1.0
Contract: `docs/specs/api/openapi.yaml` (this repo)
Implements: BR-DTY-01, BR-DTY-02, BR-DTY-03, BR-DTY-05, BR-DTY-06, FR-DTY-001…007, FR-DTY-011, NFR-DTY-01, NFR-DTY-02

> See `changelog.md` in this folder for the version history of this spec.

Spec này ban đầu viết **retroactively** từ danh mục Phật / Bồ tát đã implement; từ v3.0 nguồn nghiệp vụ là product spec ở trên (Mixed mode: phần đã ship là Retroactive, phần **v3.0** là Prospective). Spec này không nhắc lại quy tắc của product spec — chỉ trích ID và nói backend làm thế nào. Khi spec này lệch code ở phần đã ship, code là nguồn sự thật và spec được sửa theo code. `api-contract.md` cùng thư mục là tài liệu client cũ, đã lỗi thời từ v3.0 (xem banner trong file đó); hợp đồng là `openapi.yaml`.

Danh mục **chỉ đọc**, công khai, cùng hình dạng với `temple` và `article`. Một bảng `deities` chứa cả ba loại (`BUDDHA`, `BODHISATTVA`, `DEVA`); `top` và danh sách phân trang trộn các loại theo cùng một thứ tự (BR-DTY-02). Dữ liệu nạp bằng lệnh `cmd/seeddeities` chạy tay, không tự nạp khi service khởi động.

Product IDs có cột Backend `n/a` (không thuộc spec này): BR-DTY-04, BR-DTY-07, BR-DTY-08, FR-DTY-008, FR-DTY-009, FR-DTY-010. Phía server của BR-DTY-04 chỉ là hình dạng dữ liệu xác định (US-DTY-04); việc ẩn trường rỗng là của client. BR-CAL-13 (đếm ngược) là của client: server trả ngày của lần vía, client tính "Hôm nay / Ngày mai / Còn n ngày".

---

## US-DTY-01: Xem Phật và Bồ tát nổi bật

## User Story

> Là một **người dùng app hoặc website (chưa cần đăng nhập)**,
> Tôi muốn **thấy vài vị Phật và Bồ tát tiêu biểu ngay ở màn hình chủ và thư viện**,
> Để **làm quen và bấm vào tìm hiểu thêm**.

## Acceptance Criteria (EARS notation)

1. WHEN a `GET /v1/deities/top` request arrives, THE SYSTEM SHALL return `200 OK` with `data.items` holding the first `deity.top_limit` deities of **any type** (`top_limit` = 10) ordered by `sort_order` ASC then `id` ASC. (Không cần `Authorization`.) (FR-DTY-001, BR-DTY-02)
2. WHEN deities are returned by `top` or by the paged list, THE SYSTEM SHALL return each one in the **summary** shape only — `id`, `type`, `name_vi`, `name_sanskrit`, `name_chinese`, `title_honorific`, `thumbnail_url`, `image_url`, `core_virtue`, `pure_land` and, **v3.0**, `next_via` (AC 24, 29) — and SHALL NOT include `canonical_source`, `lunar_memorial_days`, `via_events`, `iconography`, `biography_summary`, `great_vows` or `mantras`. (FR-DTY-001, FR-DTY-002, BR-DTY-06)
3. WHEN the table holds fewer deities than `deity.top_limit`, or none (chưa chạy seed), THE SYSTEM SHALL return all that exist, and SHALL return `items: []` (không phải `null`) for an empty table. (BR-DTY-02)

## US-DTY-02: Duyệt danh sách Phật và Bồ tát theo trang

## User Story

> Là một **người dùng app hoặc website (chưa cần đăng nhập)**,
> Tôi muốn **duyệt toàn bộ các vị Phật và Bồ tát theo từng trang, lọc theo loại**,
> Để **tìm hiểu các vị ngoài phần nổi bật**.

## Acceptance Criteria (EARS notation)

4. WHEN a `GET /v1/deities` request arrives without `page` (or with an empty `page=`) and without `type` (or with an empty `type=`), THE SYSTEM SHALL serve page 1 of the deities of **any type** (`BUDDHA`, `BODHISATTVA` and `DEVA` together), ordered by `sort_order` ASC then `id` ASC. Mỗi item mang trường `type` để client phân biệt. (FR-DTY-002, BR-DTY-02)
5. WHEN a page is served, THE SYSTEM SHALL return `data` with `items` (summary shape), `page`, `page_size` (`deity.page_size`; **v3.0: 12**, hiện code đang 20), `total` (số mục khớp bộ lọc `type` nếu có, ngược lại số mục của cả bảng) and `total_pages` = `ceil(total / page_size)`. (FR-DTY-002, BR-DTY-03)
6. WHEN `page` is greater than `total_pages`, THE SYSTEM SHALL return `200 OK` with `items: []` and the same `page_size`/`total`/`total_pages` — không phải lỗi. (BR-DTY-03)
7. WHEN `page` is not a positive integer (`0`, negative, non-numeric, an integer that overflows, or larger than 2^31−1), THE SYSTEM SHALL return `400 INVALID_PAGE` with `details.field = "page"`. (platform)
20. WHEN a `GET /v1/deities?type=T` request arrives with `T` exactly one of `BUDDHA`, `BODHISATTVA`, `DEVA`, THE SYSTEM SHALL return only deities of that type, in the same order (`sort_order` ASC then `id` ASC), and `total` / `total_pages` SHALL count only that type. (Product không có chip "Chư Thiên"; `type=DEVA` vẫn hợp lệ ở API.) (FR-DTY-003, BR-DTY-01, BR-DTY-02)
21. WHEN `type` is present and non-empty but not exactly one of the three values (kể cả khác hoa/thường như `buddha`), THE SYSTEM SHALL return `400 INVALID_DEITY_TYPE` with `details.field = "type"`. (platform)
22. WHEN `type` is combined with `page`, THE SYSTEM SHALL page within the filtered list (offset = `(page-1) * page_size` trên danh sách đã lọc); WHEN several parameters are invalid, THE SYSTEM SHALL report `INVALID_PAGE` first, then `INVALID_DEITY_TYPE`, then (**v3.0**) `INVALID_DATE` (AC 26). (FR-DTY-003)

> Ghi chú đánh số: AC 20–22 thêm ở v2.2, AC 23–34 thêm ở v3.0; tất cả nối đuôi để các tham chiếu AC cũ không bị lệch.

## US-DTY-03: Xem chi tiết một vị Phật hoặc Bồ tát

## User Story

> Là một **người dùng app hoặc website (chưa cần đăng nhập)**,
> Tôi muốn **mở một vị Phật hay Bồ tát để đọc đầy đủ**,
> Để **biết hạnh nguyện, ngày vía, hình tượng và câu niệm của vị đó**.

## Acceptance Criteria (EARS notation)

8. WHEN a `GET /v1/deities/{id}` request arrives with an existing `id` of any type, THE SYSTEM SHALL return `200 OK` with the summary fields (incl. **v3.0** `next_via`) plus `canonical_source`, `iconography` (`posture`, `mount_or_throne`, `sacred_artifacts`, `symbolic_meaning`), `biography_summary`, `great_vows` (thứ tự lưu), `mantras` (`short_mantra_sanskrit`, `short_mantra_sino_vietnamese`, `long_dharani_name`, `dharani_phonetic_snippet`), **v3.0** `via_events` (AC 28) and `lunar_memorial_days` (**v3.0: deprecated**, dẫn xuất theo AC 30 thay vì đọc cột riêng). (FR-DTY-004, FR-DTY-005, FR-DTY-006)
9. WHEN `{id}` matches no deity, THE SYSTEM SHALL return `404 DEITY_NOT_FOUND`. (FR-DTY-004)
10. WHEN the literal path `/v1/deities/top` is requested, THE SYSTEM SHALL route it to the top list, not to detail (`ServeMux` ưu tiên segment literal). `id` có dạng `buddha_*` / `bodhisattva_*` / `deva_*` nên không trùng `top`. (platform)

## US-DTY-04: Hình dạng dữ liệu nhất quán

## User Story

> Là một **lập trình viên client**,
> Tôi muốn **các field thiếu dữ liệu luôn có dạng xác định**,
> Để **decode một lần mà không phải đoán `null` hay chuỗi rỗng**.

## Acceptance Criteria (EARS notation)

11. WHEN a deity has no value for a text field (`name_sanskrit`, `name_chinese`, `title_honorific`, `core_virtue`, `pure_land`, `canonical_source`, `biography_summary`, the iconography and mantra strings), THE SYSTEM SHALL return `""`, never `null`. (platform — client ẩn theo BR-DTY-04)
12. WHEN a deity has no sacred artifacts, great vows or (**v3.0**) linked vía events, THE SYSTEM SHALL return `[]` for `iconography.sacred_artifacts` / `great_vows` / `via_events` / `lunar_memorial_days`, never `null`; and WHEN a deity has no upcoming vía occurrence (AC 24), THE SYSTEM SHALL return `next_via: null`. (platform, BR-DTY-06)
13. WHEN `thumbnail_url` / `image_url` are returned, THE SYSTEM SHALL turn a stored root-relative path (`/static/images/buddhas/{thumbnails,full}/<id>.webp`, migration `0021`) into an absolute URL by prefixing `server.public_base_url`, or the request's own scheme and host when that is empty (dev); SHALL return an already absolute `http(s)://` value unchanged; and SHALL return an empty value as `""`, never `null` (khác `temple` và `article`). (platform)
14. WHEN any deity route fails, THE SYSTEM SHALL answer in the shared envelope `{"success": false, "error": {code, message, type}, "details"?}`; a non-`GET` request returns `405 METHOD_NOT_ALLOWED` (kèm header `Allow`), an unknown path returns `404 NOT_FOUND`, and an infrastructure failure returns `500 INTERNAL_ERROR` without leaking the cause. (platform)

## US-DTY-05: Nạp dữ liệu Phật / Bồ tát bằng lệnh seed

## User Story

> Là một **người vận hành hệ thống**,
> Tôi muốn **nạp và cập nhật danh mục từ một file dữ liệu bằng một lệnh**,
> Để **đổi thứ tự hoặc nội dung trong file rồi chạy lại mà không tạo trùng và không làm hỏng dữ liệu nửa chừng**.

## Acceptance Criteria (EARS notation)

15. WHEN the operator runs `go run ./services/appService/cmd/seeddeities [-file <path>]`, THE SYSTEM SHALL read the file (mặc định `assets/data/phat_botat.json`, tính từ thư mục đang đứng), validate it, and upsert every deity in **one transaction** keyed by `id`, with `sort_order` equal to the deity's position in the file (0, 1, 2, …) and, **v3.0**, replace each deity's vía links with the file's `via_event_codes` in the same transaction, then log `seeded N deities from <file>`. (FR-DTY-011, BR-DTY-02, BR-DTY-05)
16. WHEN the seed is validated, THE SYSTEM SHALL reject the whole file (write nothing) if it has no deity, an empty or duplicate `id`, a `type` other than `BUDDHA` / `BODHISATTVA` / `DEVA`, an empty `name_vi`, or, **v3.0**, the same vía event code twice on one deity, or a vía event code that does not exist in `calendar_events` (lệnh log danh sách mã thiếu). (FR-DTY-011, BR-DTY-01, BR-DTY-05)
17. WHEN the seed is run again after the file was edited or reordered, THE SYSTEM SHALL update the existing rows in place by `id` — overwriting every column from the file, including `sort_order`, and setting `updated_at = now()` — and SHALL NOT create a second row for an existing `id`. (FR-DTY-011)
18. WHEN the file is invalid, the database is unreachable, or the database rejects a row (vd. migration chưa apply), THE SYSTEM SHALL exit non-zero with a fatal log and SHALL leave the `deities` table and (**v3.0**) the vía links unchanged (transaction rolled back). (FR-DTY-011)
19. WHEN appService starts, THE SYSTEM SHALL NOT seed or otherwise write the `deities` table — một DB chưa chạy seed trả `items: []` (AC 3, 6), không phải lỗi. (platform)
33. **v3.0** — THE SYSTEM SHALL keep the catalogue file `assets/data/phat_botat.json` in version control (ngoại lệ `.gitignore` như `administrative-units.json`), so `make ensure-deities` on a fresh clone, the E2E harness and `TestParseDeities_RealDataFile` all read the same committed file. (FR-DTY-011)

## US-DTY-06: Ngày vía lấy từ lịch lễ (v3.0)

## User Story

> Là một **người dùng app hoặc website**,
> Tôi muốn **thấy lần vía kế tiếp của mỗi vị trên thẻ và mọi ngày vía trên trang chi tiết, đúng như lịch lễ**,
> Để **không phải tự đổi ngày âm và không bao giờ thấy hai nơi trong app lệch nhau**.

## Acceptance Criteria (EARS notation)

23. THE SYSTEM SHALL take a deity's vía days only from the festival-calendar events linked to that deity (bảng liên kết deity ↔ `calendar_events.code`, nạp từ file danh mục), with their solar and lunar dates computed by the calendar's own rules (`domain.ResolveOccurrences`, BR-CAL-01…08), and SHALL NOT store or compute vía dates of its own. (BR-DTY-05)
24. WHEN a deity is returned by any of the three routes, THE SYSTEM SHALL set `next_via` to the occurrence, among the occurrences of all its linked active events in solar years `Y` and `Y+1` (`Y` = năm của "hôm nay", AC 25; `Y+1` bỏ qua khi `Y = 2100`), with the earliest `solar_date` on or after today (today included), ties broken by the event's calendar `sort_order` then `code` (BR-CAL-12); and to `null` when there is none. (BR-DTY-06)
25. WHEN a deity route request carries `date` (`YYYY-MM-DD`, năm 1900..2100), THE SYSTEM SHALL use it as "today" for AC 24 and AC 28; WHEN `date` is absent or empty, THE SYSTEM SHALL use today's date in `calendar.timezone` (dev: `Asia/Ho_Chi_Minh`). (Client gửi ngày trên thiết bị để khớp BR-DTY-06 "today = device date".) (BR-DTY-06)
26. WHEN `date` is present and non-empty but not a real date in `YYYY-MM-DD` form or outside 1900..2100, THE SYSTEM SHALL return `400 INVALID_DATE` with `details.field = "date"` on `top`, the list and the detail alike (cùng quy tắc với `GET /v1/calendar/image`). (platform)
27. WHEN a linked calendar event is inactive (`is_active = false`), THE SYSTEM SHALL ignore it for `next_via`, `via_events` and `lunar_memorial_days`, as if it were not linked. (BR-DTY-05, BR-CAL-16)
28. WHEN a deity detail is returned, THE SYSTEM SHALL list in `via_events` every linked active event exactly once, each with its next occurrence on or after today in years `Y`..`Y+1` (`next_occurrence`, or `null` when it has none in that window), ordered by `next_occurrence.solar_date`, then calendar `sort_order`, then `code` (BR-CAL-12), events without an occurrence last; the first item with an occurrence is the same as `next_via`. (FR-DTY-006, BR-DTY-05, BR-DTY-06)
29. WHEN `next_via` or a `via_events` item is returned, THE SYSTEM SHALL give the event's `code` and `name` (ngôn ngữ `calendar.default_language`) and the occurrence as `solar_date` (`YYYY-MM-DD`) plus `lunar_date` (`day`, `month`, `year`, `is_leap_month`), in the same field shape as `GET /v1/calendar/events` occurrences, so no client converts lunar dates. (FR-DTY-002, FR-DTY-006, BR-CAL-08)
30. WHEN a deity detail is returned, THE SYSTEM SHALL fill the deprecated `lunar_memorial_days` (`[{lunar_day, lunar_month, event_name}]`) from `via_events`, one entry per item that has a `next_occurrence` (ngày/tháng âm của lần đó, tên sự kiện), in the same order, so it never disagrees with the calendar. (BR-DTY-05)
31. WHEN `top`, a list page or a detail is served, THE SYSTEM SHALL read the deities and their vía links in the page query itself and the calendar catalogue in one further query, so the number of queries per request does not grow with the catalogue or the page size. (NFR-DTY-01, NFR-DTY-02)
32. THE SYSTEM SHALL pass every case of `docs/product/deity/rules/deity-order.json` (BR-DTY-02, repository test trên Postgres) and the choice of `event` / `solar_date` / `days` in every case of `docs/product/deity/rules/deity-next-via.json` (BR-DTY-06, unit test hàm thuần), loading both files from `docs/product/`, not from a copy. (BR-DTY-02, BR-DTY-06)

## Kinh liên quan

34. WHEN a client shows the related sutras of a deity, THE SYSTEM SHALL serve them only through the existing `GET /v1/deities/{id}/sutras` (owned by `docs/specs/features/sutra/`, AC 20–24 there), and the deity routes SHALL NOT embed sutras. (FR-DTY-007)

> ⚠️ Not confirmed (product): FR-DTY-007 nói "in the sutra library's order", còn `GET /v1/deities/{id}/sutras` sắp theo vai trò (`SUBJECT`, `SPEAKER`, `MENTIONED`) rồi `sort_order` — có phải đó là thứ tự product muốn không, hay chỉ `sort_order`? — chặn việc client hiển thị đúng thứ tự (backend không đổi gì nếu thứ tự hiện tại là đúng); hand back to ba-writer in innerlamp-be.

## API contract (confirmed surface for this feature)

Đọc từ handler và `cmd/main.go`; ba route đã có trong `docs/specs/api/openapi.yaml` (tag Deities). Các phần **v3.0** dưới đây là thiết kế của spec này, **chưa** có trong `openapi.yaml` — cập nhật cùng lượt implement (`tasks.md` Phase 9).

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/deities/top?date=` | public | `200` `{items: [DeitySummary]}` (≤ 10, mọi loại) | 400 `INVALID_DATE` (v3.0), 404 `NOT_FOUND` (sai path), 405, 500 |
| GET | `/v1/deities?type=&page=N&date=` | public | `200` `{items, page, page_size (12, v3.0), total, total_pages}` | 400 `INVALID_PAGE`, 400 `INVALID_DEITY_TYPE`, 400 `INVALID_DATE` (v3.0), 405, 500 |
| GET | `/v1/deities/{id}?date=` | public | `200` `DeityDetail` | 400 `INVALID_DATE` (v3.0), 404 `DEITY_NOT_FOUND`, 405, 500 |
| GET | `/v1/deities/{id}/sutras` | public | thuộc spec `sutra` (AC 34) | thuộc spec `sutra` |

`DeitySummary`: `id`, `type` (`BUDDHA` \| `BODHISATTVA` \| `DEVA`), `name_vi`, `name_sanskrit`, `name_chinese`, `title_honorific`, `thumbnail_url`, `image_url` (string, tuyệt đối hoặc `""`), `core_virtue`, `pure_land`, **v3.0** `next_via` (`DeityVia` \| `null`).

`DeityVia` (v3.0): `{event_code, name, solar_date, lunar_date: {day, month, year, is_leap_month}}`.

`DeityDetail` = `DeitySummary` + `canonical_source`, `iconography` `{posture, mount_or_throne, sacred_artifacts: [string], symbolic_meaning}`, `biography_summary`, `great_vows` (`[string]`), `mantras` `{…4 chuỗi}`, **v3.0** `via_events` (`[{event_code, name, next_occurrence: {solar_date, lunar_date} | null}]`), `lunar_memorial_days` (`[{lunar_day, lunar_month, event_name}]`, **deprecated từ v3.0**).

Thay đổi so với hợp đồng đang chạy: `page_size` 20 → 12 (client nào tự giả định 20 phải đọc `page_size`); thêm `next_via`, `via_events`, tham số `date`, lỗi `INVALID_DATE` — đều là thêm, không breaking; `lunar_memorial_days` vẫn có nhưng đổi nguồn (có thể ít mục hơn hôm nay nếu một ngày vía cũ không có sự kiện lịch tương ứng).

> ⚠️ Not confirmed: khi nào xoá hẳn `lunar_memorial_days` khỏi `DeityDetail` (sau khi website, iOS, Android đã chuyển sang `via_events`, hay giữ tới một mốc phiên bản app tối thiểu) — chặn `tasks.md` Phase 10 mục xoá field; không chặn phần còn lại.

## Out of scope (this feature boundary, not tracked here)

- Tạo/sửa/xoá mục qua API hoặc backoffice — chỉ có `cmd/seeddeities` và SQL trực tiếp (product §1.2).
- Tìm kiếm, yêu thích, theo dõi, nhắc ngày vía, "Báo thông tin sai" (product §1.2, BR-DTY-08).
- Lọc nhiều `type` cùng lúc; bộ lọc khác ngoài `type`.
- Nội dung, quy tắc lặp và vị trí ngày của sự kiện lịch: thuộc `docs/specs/features/calendar/` — spec này chỉ đọc catalogue sự kiện và gọi bộ tính ngày của nó.
- Đếm ngược và nhãn "Hôm nay / Ngày mai / Còn n ngày" (BR-CAL-13), định dạng "Vía dd/MM âm lịch", tháng "GIÊNG/CHẠP", ẩn trường rỗng (BR-DTY-04), thu gọn đại nguyện (BR-DTY-07): client.
- Danh sách kinh liên quan: route và thứ tự thuộc spec `sutra` (AC 34).
- Lưu trữ ảnh trên CDN / object store: ảnh hiện phục vụ dev-only qua `/static/images/` (`server.static_images_dir`).

## Pending (not yet implemented)

- Mọi AC/phần đánh dấu **v3.0**: trang 12 (AC 5), `next_via` / `via_events` / `date` / `INVALID_DATE` (AC 2, 8, 12, 22–31), seed liên kết vía (AC 15, 16, 18), file danh mục vào git (AC 33), vector (AC 32) — `tasks.md` Phase 8–10.
- `TestParseDeities_RealDataFile` **fail** trong checkout này vì `assets/data/phat_botat.json` bị `.gitignore` (`*.json`) và không có trên đĩa; E2E harness cũng đọc file này — AC 33.
- E2E cho `top` và `detail` — `tasks.md` Phase 5.
- OKF concept `docs/knowledge/features/deity.md` — chưa có; `il-okf-writer` tạo sau khi v3.0 ship.

> ⚠️ Not confirmed (product): với từng vị trong 20 vị, những sự kiện nào của lịch lễ là ngày vía của vị đó, và một ngày vía đang có trong file danh mục mà lịch lễ chưa có sự kiện tương ứng thì lịch lễ thêm sự kiện mới hay bỏ ngày đó (BR-DTY-05 chỉ nói lịch là nguồn duy nhất) — chặn nội dung `via_event_codes` trong file danh mục (`tasks.md` Phase 8, mục dữ liệu); cả hai file seed (`phat_botat.json`, `calendar-events.json`) đều vắng trong checkout này nên spec chưa đối chiếu được; hand back to ba-writer in innerlamp-be.
> ⚠️ Not confirmed (product): tiêu chí "nổi bật" (product Appendix A mục 2) — backend giữ BR-DTY-02 (thứ tự file), chỉ nội dung file có thể đổi; không chặn task nào.
> ⚠️ Not confirmed (product): ai duyệt nội dung tôn giáo và quyền ảnh trước khi phát hành (product Appendix A mục 1) — chặn phát hành nội dung, không chặn task nào ở đây.
> ⚠️ Not confirmed (product): ai chạy seed ở staging/production và từ bản file nào (product Appendix A mục 4); `cmd/seeddeities` vẫn dùng config + secret dev (`⚠️ DEV ONLY`) — chặn `tasks.md` Phase 3 mục "seed ngoài dev".
> ⚠️ Not confirmed: ảnh ở production nằm ở đâu (CDN / object store nào) — hôm nay ảnh là file trong `assets/images/buddhas/` phục vụ dev-only; chỉ cần đặt `server.public_base_url` khi có host (AC 13), không phải seed lại.

## Ghi chú kỹ thuật liên quan

- Dữ liệu: 20 vị — 7 `BUDDHA` (gồm `buddha_seven_past`, Thất Phật Quá Khứ), 12 `BODHISATTVA`, 1 `DEVA` (`deva_sri_mahadevi`, Đại Cát Tường Thiên Nữ; migration `0015`). Số này đối chiếu từ `parse_test.go` (`require.Len(t, got, 20)`) và 20 ảnh trong `assets/images/buddhas/full/`; file danh mục vắng nên thứ tự từng mục chưa đối chiếu được. Với trang 12: 2 trang (12 + 8).
- Ảnh lưu dạng đường dẫn gốc tương đối `/static/images/buddhas/{thumbnails,full}/<id>.webp` (migration `0021`); handler ghép `server.public_base_url` (rỗng = origin của request, dev) qua `imageHost` dùng chung với `temple`, `gallery`, `article`, `sutra`.
- `id` là chuỗi ổn định, seeder upsert theo `id`; `sort_order` = vị trí trong file và là nguồn xếp hạng duy nhất; `id` là tie-break (BR-DTY-02).
- **v3.0 — liên kết vía**: một bảng nối `deity_id` ↔ `calendar_events.code` (migration mới), nằm trong file danh mục deity (product §3.5: "kept with the deity"). Seed deity vì vậy cần catalogue lịch đã nạp trước (thứ tự nạp: `calendar` → `deity` → `sutra`). Cột `deities.lunar_memorial_days` bị bỏ.
- **v3.0 — ai tính lần vía kế tiếp**: **server tính** (BR-DTY-06 để contract quyết). Lý do: client không phải đổi âm ↔ dương, không phải tải lịch cả năm cho một thẻ, và mọi nền tảng ra cùng kết quả; client chỉ đếm ngược (BR-CAL-13). "Hôm nay" là ngày thiết bị gửi qua `date`; thiếu `date` thì server dùng `calendar.timezone` — giống `GET /v1/calendar/image`. Vì kết quả phụ thuộc ngày, response không thêm header cache công khai.
- Cửa sổ `Y`..`Y+1` đủ cho mọi sự kiện lặp hằng năm; một sự kiện chỉ có trong năm có tháng nhuận (BR-CAL-04) có thể không có lần nào trong cửa sổ → `next_occurrence: null`.
- Kích thước trang ở block `deity:` của `config.<env>.yaml` (dev hiện `top_limit: 10`, `page_size: 20` → v3.0 `12`).

## Open questions (resolved during implementation)

- 2026-10-15 (v3.0): **Mã user story** — product spec v1.0 cấp ID `BR-DTY-*` / `FR-DTY-*`; mọi AC gắn ID đó. `US-DTY-01..06` giữ làm tiêu đề nhóm nội bộ của spec này, không phải mã product. Ảnh hưởng: không đổi `design.md`/`tasks.md`.
- 2026-10-15 (v3.0): **Ảnh không phân giải được** (marker v1.0) — đã giải quyết trong code bằng migration `0021_relative_image_urls.sql` + `imageHost` + 20 file ảnh trong `assets/images/buddhas/`; AC 13 viết lại theo code. Câu hỏi còn lại (host production, quyền ảnh) giữ marker riêng ở Pending.
- 2026-10-15 (v3.0): **Cỡ trang, nguồn ngày vía, lần vía kế tiếp, kinh liên quan, file danh mục** — product spec v1.0 đã chốt (BR-DTY-03, BR-DTY-05, BR-DTY-06, FR-DTY-007, Appendix B). Quyết định kỹ thuật của spec này (chờ duyệt qua PR): server tính `next_via`; tham số `date` cho "hôm nay"; bảng nối deity ↔ sự kiện lịch nằm trong file danh mục deity; `lunar_memorial_days` giữ tạm, dẫn xuất từ `via_events`. Ảnh hưởng: `design.md` thêm Step gắn vía cho cả ba flow đọc, seed thêm bước kiểm mã sự kiện; `tasks.md` Phase 8–10.
- 2026-10-12 (v2.2): **Có API lọc Phật / Bồ tát theo `type` không?** — thêm tham số `type` trên `GET /v1/deities`, khớp chính xác ba giá trị, rỗng = không lọc, giá trị lạ = `400 INVALID_DEITY_TYPE`, một giá trị mỗi lần (theo tiền lệ `gallery`). Product v1.0 FR-DTY-003 xác nhận bộ lọc.
- 2026-09-30 (v2.0): **Cách xem Phật** — `GET /v1/bodhisattvas` thay bằng `GET /v1/deities?page=N` gộp mọi loại.
