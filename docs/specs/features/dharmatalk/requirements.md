# Requirements — dharmatalk

Version: 1.1
Last updated: 2026-10-03
Status: implemented (2026-10-03) — 2 route đọc công khai + seed lần đầu chạy thật qua binary + Postgres (testcontainers, 55 bài, 36 hiển thị được); có test usecase, handler, seed parser, repository và E2E. `openapi.yaml` và ERD đã cập nhật. Còn 11 marker `⚠️ Not confirmed` (mọi quyết định sản phẩm vẫn mở).
Module: `content`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này được viết **prospectively** (trước khi có code; từ v1.1 đã implement) cho màn "Pháp thoại" của app (mockup: tiêu đề, giảng sư, trình phát, mô tả, "Các phần", thẻ thiền sau khi nghe, nguồn ở cuối). **Pháp thoại khác kinh**: kinh (`sutra`) là văn bản kinh điển; pháp thoại là bài giảng của một vị thầy cụ thể, nên có thêm giảng sư, thời lượng, phương tiện nghe/xem. Hai danh mục tách bảng, tách route.

Đây là danh mục **chỉ đọc**, công khai, cùng hình dạng với `article`: danh sách phân trang + trang chi tiết, và seed một lần khi service khởi động lần đầu từ file JSON nhúng trong binary.

Mỗi bài thuộc **đúng một trong ba loại phương tiện** (`media_type`): `youtube` (nhúng video), `audio` (phát file âm thanh) hoặc `text` (chỉ đọc). Màn hình đổi biến thể theo loại này; backend chỉ cung cấp dữ liệu, không quyết định giao diện.

> ⚠️ Not confirmed: mã user story chưa được product chốt — spec tạm đánh `US-DT-01..05` theo tiền lệ `US-ART-01`.

---

## US-DT-01: Duyệt danh sách pháp thoại theo trang

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **cuộn qua các bài pháp thoại theo từng trang**,
> Để **chọn một bài để nghe, xem hoặc đọc**.

## Acceptance Criteria (EARS notation)

1. WHEN a `GET /v1/dharma-talks` request arrives without `page` (or with an empty `page=`), THE SYSTEM SHALL serve page 1, ordered by `sort_order` ASC then `id` ASC. (Không cần `Authorization`.)
2. WHEN a page is served, THE SYSTEM SHALL return `data` with `items` (summary shape), `page`, `page_size` (`dharma_talk.page_size`, dev: 10), `total` (số bài **được hiển thị**, xem AC 4) and `total_pages` = `ceil(total / page_size)`.
3. WHEN `page` is greater than `total_pages`, THE SYSTEM SHALL return `200 OK` with `items: []` and the same `page_size`/`total`/`total_pages` — không phải lỗi.
4. WHEN listing or reading, THE SYSTEM SHALL only expose talks whose `needs_review` is `false`; talks flagged for review are stored but never returned by any route. (Mặc định tạm — xem `⚠️ Not confirmed` bên dưới.)
5. WHEN `page` is not a positive integer (`0`, negative, non-numeric, overflows, or larger than 2^31−1), THE SYSTEM SHALL return `400 INVALID_PAGE` with `details.field = "page"`.

## US-DT-02: Mở một bài pháp thoại

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **mở một bài pháp thoại và nghe, xem hoặc đọc nó**,
> Để **tiếp nhận bài giảng và biết nó đến từ nguồn nào**.

## Acceptance Criteria (EARS notation)

6. WHEN a `GET /v1/dharma-talks/{id}` request arrives with an existing, displayable `id`, THE SYSTEM SHALL return `200 OK` with the summary fields plus `key_points`, `quote`, `quote_source`, `language`, `youtube_id`, `audio_url`, `chapters`, `follow_up_meditation`, `source_name`, `source_url` and `license`.
7. WHEN `{id}` matches no talk, or matches a talk with `needs_review = true`, THE SYSTEM SHALL return `404 DHARMA_TALK_NOT_FOUND`.
8. WHEN a talk has `media_type = "youtube"`, THE SYSTEM SHALL return a non-empty 11-character `youtube_id` and `audio_url: null`; WHEN `media_type = "audio"`, a non-empty `audio_url` (`http`/`https`) and `youtube_id: null`; WHEN `media_type = "text"`, both `null`. (Không bao giờ trả URL YouTube đầy đủ — client tự ghép khi nhúng.)

## US-DT-03: Các phần của bài giảng

## User Story

> Là một **người dùng app**,
> Tôi muốn **nhảy tới một phần của bài giảng dài**,
> Để **nghe lại đúng đoạn tôi quan tâm**.

## Acceptance Criteria (EARS notation)

9. WHEN a talk has chapters, THE SYSTEM SHALL return `chapters` as an array of `{start_seconds, title}` ordered by `start_seconds` ASC, with `start_seconds` a non-negative integer; WHEN it has none, THE SYSTEM SHALL return `[]` (không phải `null`), để client ẩn mục "Các phần".
10. WHEN a talk has `media_type = "text"`, THE SYSTEM SHALL return `chapters: []` (không có mốc thời gian để nhảy tới).

## US-DT-04: Hình dạng dữ liệu nhất quán

## User Story

> Là một **lập trình viên client**,
> Tôi muốn **các field thiếu dữ liệu luôn có dạng xác định**,
> Để **decode một lần và ẩn đúng phần giao diện không có dữ liệu**.

## Acceptance Criteria (EARS notation)

11. WHEN a talk has a publication date, THE SYSTEM SHALL return `published_date` as `"YYYY-MM-DD"`; otherwise JSON `null`.
12. WHEN a talk has a known duration, THE SYSTEM SHALL return `duration_minutes` as a positive integer; otherwise JSON `null` — không trả `0`.
13. WHEN a talk has no `season` (dịp như Vu Lan, Tết…), THE SYSTEM SHALL return `season: null`; WHEN it has no `follow_up_meditation`, THE SYSTEM SHALL return `follow_up_meditation: null`.
14. WHEN a talk has no tags or no key points, THE SYSTEM SHALL return `[]` for `tags` / `key_points`, never `null`.
15. WHEN any dharma-talk route fails, THE SYSTEM SHALL answer in the shared envelope `{"success": false, "error": {code, message, type}, "details"?}`; a non-`GET` request returns `405 METHOD_NOT_ALLOWED` (kèm `Allow: GET, HEAD`) and an infrastructure failure returns `500 INTERNAL_ERROR` without leaking the cause.

## US-DT-05: Nạp dữ liệu lần đầu khi khởi động

## User Story

> Là một **người triển khai hệ thống**,
> Tôi muốn **service tự có sẵn bộ pháp thoại ngay lần chạy đầu**,
> Để **môi trường mới chạy được ngay và các lần khởi động sau không đụng vào dữ liệu đã chỉnh**.

## Acceptance Criteria (EARS notation)

16. WHEN appService starts AND the `dharma_talks` table has no row, THE SYSTEM SHALL parse the JSON embedded in the binary (`assets/data/dharma_talks.json`) and insert every talk in **one transaction**, with `sort_order` equal to the talk's position in the file, then log `seeded dharma talks on first run` with the count.
17. WHEN the table already has at least one row, THE SYSTEM SHALL NOT parse the seed and SHALL NOT write any row.
18. WHEN a seed insert meets an `id` that already exists, THE SYSTEM SHALL leave that row untouched (`ON CONFLICT (id) DO NOTHING`).
19. WHEN the seed is invalid, THE SYSTEM SHALL abort startup with a fatal log and leave the table unchanged. Invalid means any of: no talk; empty or duplicate `id`; empty `title` or `speaker`; `mediaType` not one of `youtube`/`audio`/`text`; `youtube` without a valid 11-character id, `audio` without an `http(s)` URL, or `text` carrying either; `durationMinutes` ≤ 0; a `publishedDate` that is not `YYYY-MM-DD`; a chapter with a negative `startSeconds`, an empty title, or chapters out of order.

## API contract (implemented)

Đọc từ handler và `cmd/main.go`, đã kiểm bằng E2E chạy binary thật. Đã có trong `docs/specs/api/openapi.yaml` (tag DharmaTalks) và `web/docs/public/openapi.yaml`.

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/dharma-talks?page=N` | public | `200` `{items, page, page_size, total, total_pages}` | 400 `INVALID_PAGE`, 405, 500 |
| GET | `/v1/dharma-talks/{id}` | public | `200` `DharmaTalkDetail` | 404 `DHARMA_TALK_NOT_FOUND`, 405, 500 |

`DharmaTalkSummary`: `id` (slug), `title`, `speaker`, `speaker_initials`, `category`, `season` (string | `null`), `published_date` (`"YYYY-MM-DD"` | `null`), `duration_minutes` (int | `null`), `summary`, `tags` (`[string]`), `media_type` (`youtube` | `audio` | `text`).

`DharmaTalkDetail` = `DharmaTalkSummary` + `key_points` (`[string]`), `quote`, `quote_source`, `language` (`vi`), `youtube_id` (string | `null`), `audio_url` (string | `null`), `chapters` (`[{start_seconds: int, title: string}]`), `follow_up_meditation` (`{title: string, minutes: int, meditation_id: string | null}` | `null`), `source_name`, `source_url`, `license`.

## Out of scope (this feature boundary, not tracked here)

- Tạo/sửa/xoá pháp thoại qua API hoặc backoffice — chỉ SQL trực tiếp trên bảng.
- Trích xuất audio từ video YouTube hoặc tự lưu/phân phối lại file media: YouTube chỉ được **nhúng** qua `youtube_id`; điều khoản YouTube và bản quyền không cho tách audio. Không có file media trong repo hay GCS.
- Tìm kiếm, lọc theo `category`/`tag`/`season`, bookmark, đánh dấu đã nghe.
- Route `top` (nổi bật) — màn hình mockup không cần; thêm khi màn chủ cần.
- Đăng nhập: cả hai route công khai như `article`.

## Pending (not yet implemented)

- Khái niệm OKF `docs/knowledge/features/dharmatalk.md` (`il-okf-writer`) — `tasks.md` Phase 5.
- Trạng thái "lỗi nhúng" của YouTube là việc của client (video bị gỡ hoặc tắt nhúng); backend không kiểm tra `embeddable`.

> ⚠️ Not confirmed: lượt nghe ("2.4k lượt nghe") — tin nhắn của người dùng bị cụt ở ý này. Backend **chưa** có cột, trường hay route ghi lượt nghe; client ẩn khi không có. Cần chốt có đếm lượt nghe không, đếm khi nào, và route ghi có cần đăng nhập không.
> ⚠️ Not confirmed: thẻ "Nghe xong, ngồi thiền" — app chưa có entity bài thiền để `meditation_id` trỏ tới nên `follow_up_meditation` luôn `null` trong dữ liệu seed; cần chốt nguồn dữ liệu và ánh xạ pháp thoại → bài thiền.
> ⚠️ Not confirmed: "Các phần" — dữ liệu seed hiện **không có** mốc thời gian nào (`chapters` rỗng ở cả 55 bài); nguồn có thể là chapter của video YouTube hoặc nghe và ghi tay; chưa có người/ngày làm.
> ⚠️ Not confirmed: `season` (Mùa Vu Lan…) — danh sách dịp hợp lệ và cách gán chưa chốt; hiện là chuỗi tự do, `null` ở mọi bài.
> ⚠️ Not confirmed: cờ `needs_review` — mặc định tạm là ẩn bài đang bị cờ khỏi mọi route (19/55 bài, trong đó có 3 bài Thông Triệt chưa xác minh nội dung); cần người dùng quyết định giữ hay loại từng bài, và quy trình gỡ cờ (hiện chỉ SQL tay).
> ⚠️ Not confirmed: audio — chỉ 8/55 bài có file mp3 hợp lệ (Làng Mai); nguồn audio hợp lệ khác (Pháp Âm, archive.org, xin phép chùa) chưa tìm; 29 bài còn là `text`. Cũng chưa chốt tốc độ phát, nhớ vị trí nghe, nút lưu/chia sẻ — thuộc client, không ảnh hưởng hợp đồng này.
> ⚠️ Not confirmed: bản quyền — cả 55 bài có `license` là `unknown` hoặc chỉ có dòng copyright của nguồn; chưa xác nhận app được hiển thị tóm tắt/trích dẫn và nhúng video theo điều kiện nào, và có bắt buộc hiển thị nguồn không (UI đã có thẻ nguồn ở cuối).
> ⚠️ Not confirmed: số lượng — mục tiêu ban đầu 60 bài, hiện 55 (thiếu 5 bài Thích Chân Quang không có nguồn kiểm chứng được; đã bỏ).
> ⚠️ Not confirmed: tên hiển thị giảng sư — `speaker` hiện giữ tiền tố gốc ("Hòa thượng Thích …", "Sư …"); màn hình mockup ghi "Thầy …"; chưa chốt app tự rút gọn hay backend thêm `speaker_display_name`.
> ⚠️ Not confirmed: quy trình cập nhật nội dung sau lần seed đầu — như `article`, sửa file JSON rồi build lại không đổi DB có sẵn; cần chốt SQL tay, lệnh reseed hay backoffice.

## Ghi chú kỹ thuật liên quan

- Dữ liệu seed hiện có: 55 bài — 18 `youtube`, 8 `audio`, 29 `text`; 38 bài có ngày; 16 bài có `durationMinutes`; 19 bài `needs_review` nên chỉ 36 bài hiển thị được. `publishedDate` của bài YouTube thường là ngày upload, không phải ngày giảng.
- `id` là slug ổn định để client cache id vẫn đúng sau mọi lần nạp lại.
- `media_type` quyết định biến thể giao diện; seed hiện dùng `video` cho YouTube — parser đổi thành `youtube` (xem `design.md`).
- Kích thước trang nằm ở block `dharma_talk:` của `config.<env>.yaml` (dev: `page_size: 10`), bắt buộc.
- Bảng `dharma_talks` phải tồn tại trước khi service chạy (`make migrate`), nếu không bước seed lỗi và service không khởi động.

## Open questions (resolved during implementation)

Chưa có câu hỏi nào được người dùng trả lời. Các quyết định tạm của phiên bản này (tên feature `dharmatalk`, route `/v1/dharma-talks`, bỏ route `top`, ẩn bài `needs_review`) nằm trong `docs/assumptions/dharmatalk/` chờ xác nhận.
