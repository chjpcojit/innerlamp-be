# Requirements — article

Version: 1.6
Last updated: 2026-10-06
Status: implemented (2026-09-30) — 3 route đọc công khai + seed lần đầu chạy thật qua binary + Postgres (dev, 60 bài); có test usecase, handler, seed parser và repository. **Chưa có** E2E tự động, `openapi.yaml` và ERD đã cập nhật. Còn 4 marker `⚠️ Not confirmed`.
Module: `content`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này được viết **retroactively** từ tính năng "Bài viết Phật pháp hay" đã được implement (Retroactive mode). Mọi AC dưới đây mô tả đúng hành vi code hiện có; thứ chưa làm nằm ở "Pending", không gộp vào AC.

Đây là danh mục **chỉ đọc**, công khai, cùng hình dạng với danh mục chùa (`temple`) và Phật/Bồ tát (`deity`): một danh sách "nổi bật", một danh sách phân trang, một trang chi tiết. Khác hai danh mục kia ở chỗ dữ liệu **không** nạp bằng lệnh seed thủ công mà tự nạp một lần khi service khởi động lần đầu, từ file JSON nhúng trong binary.

> ⚠️ Not confirmed: mã user story của tính năng này chưa được product chốt — spec tạm đánh `US-ART-01..05` theo tiền lệ `US-BO-01` của `admin`; đổi khi product chốt số.

---

## US-ART-01: Xem bài viết nổi bật

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **thấy vài bài viết Phật pháp được chọn lọc ngay ở màn hình chủ**,
> Để **có thứ đọc ngay mà không phải tìm kiếm**.

## Acceptance Criteria (EARS notation)

1. WHEN a `GET /v1/articles/top` request arrives, THE SYSTEM SHALL return `200 OK` with `data.items` holding the first `article.top_limit` articles (dev: 5) ordered by `sort_order` ASC then `id` ASC. (Không cần `Authorization`.)
2. WHEN articles are returned by `top`, THE SYSTEM SHALL return each one in the **summary** shape only — `id`, `title`, `category`, `author`, `author_initials`, `published_date`, `read_minutes`, `summary`, `tags`, `image_url`, `thumbnail_url` — and SHALL NOT include `body_paragraphs`, `quote`, `quote_source`, `language`, `source_name`, `source_url` or `license`.
3. WHEN the table holds fewer articles than `article.top_limit`, or none, THE SYSTEM SHALL return all that exist, and SHALL return `items: []` (không phải `null`) for an empty table.

## US-ART-02: Duyệt danh sách bài viết theo trang

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **cuộn qua toàn bộ bài viết theo từng trang**,
> Để **đọc tiếp những bài ngoài phần nổi bật**.

## Acceptance Criteria (EARS notation)

4. WHEN a `GET /v1/articles` request arrives without `page` (or with an empty `page=`), THE SYSTEM SHALL serve page 1. Thứ tự giống hệt `top`, nên trang 1 bắt đầu bằng đúng các bài của `top`.
5. WHEN a page is served, THE SYSTEM SHALL return `data` with `items` (summary shape), `page`, `page_size` (`article.page_size`, dev: 10), `total` (số bài trong toàn bảng) and `total_pages` = `ceil(total / page_size)`.
6. WHEN `page` is greater than `total_pages`, THE SYSTEM SHALL return `200 OK` with `items: []` and the same `page_size`/`total`/`total_pages` — không phải lỗi. Client dừng khi `page >= total_pages` hoặc `items` rỗng.
7. WHEN `page` is not a positive integer (`0`, negative, non-numeric, an integer that overflows, or larger than 2^31−1), THE SYSTEM SHALL return `400 INVALID_PAGE` with `details.field = "page"`.

## US-ART-03: Đọc một bài viết

## User Story

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **mở một bài viết để đọc đầy đủ**,
> Để **đọc nội dung, câu trích và biết bài này từ nguồn nào**.

## Acceptance Criteria (EARS notation)

8. WHEN a `GET /v1/articles/{id}` request arrives with an existing `id`, THE SYSTEM SHALL return `200 OK` with the summary fields plus `body_paragraphs`, `quote`, `quote_source`, `language`, `source_name`, `source_url` and `license`.
9. WHEN `{id}` matches no article, THE SYSTEM SHALL return `404 ARTICLE_NOT_FOUND`.
10. WHEN the literal path `/v1/articles/top` is requested, THE SYSTEM SHALL route it to the top list, not to detail — nên một bài có `id = "top"` sẽ không đọc được qua route chi tiết.

## US-ART-04: Hình dạng dữ liệu nhất quán

## User Story

> Là một **lập trình viên client**,
> Tôi muốn **các field thiếu dữ liệu luôn có dạng xác định**,
> Để **decode một lần mà không phải đoán `null` hay chuỗi rỗng**.

## Acceptance Criteria (EARS notation)

11. WHEN an article has a publication date, THE SYSTEM SHALL return `published_date` as a `"YYYY-MM-DD"` string; WHEN it has none, THE SYSTEM SHALL return JSON `null` — không trả chuỗi rỗng hay ngày giả.
12. WHEN an article has an image, THE SYSTEM SHALL return `image_url` as a string; WHEN it has none, THE SYSTEM SHALL return JSON `null`.
13. WHEN an article has a thumbnail, THE SYSTEM SHALL return `thumbnail_url` as a string pointing at a small image hosted by this service; WHEN it has none, THE SYSTEM SHALL return JSON `null` (the key is always present, in the summary and in the detail). The client falls back to `image_url` when it is `null`.
13. WHEN an article has no tags or no body paragraphs, THE SYSTEM SHALL return `[]` for `tags` / `body_paragraphs`, never `null`.
14. WHEN any article route fails, THE SYSTEM SHALL answer in the shared envelope `{"success": false, "error": {code, message, type}, "details"?}`; a non-`GET` request returns `405 METHOD_NOT_ALLOWED` (kèm header `Allow: GET, HEAD`) and an infrastructure failure returns `500 INTERNAL_ERROR` without leaking the cause.

## US-ART-05: Nạp dữ liệu lần đầu khi khởi động

## User Story

> Là một **người triển khai hệ thống**,
> Tôi muốn **service tự có sẵn bộ bài viết ngay lần chạy đầu mà không cần lệnh seed riêng**,
> Để **môi trường mới (kể cả container không có mã nguồn) chạy được ngay và các lần khởi động sau không đụng vào dữ liệu đã chỉnh**.

## Acceptance Criteria (EARS notation)

15. WHEN appService starts AND the `articles` table has no row, THE SYSTEM SHALL parse the JSON embedded in the binary (`assets/data/buddhist_articles.json`) and insert every article in **one transaction**, with `sort_order` equal to the article's position in the file (0, 1, 2, …), then log `seeded articles on first run` with the count.
16. WHEN appService starts AND the `articles` table already has at least one row, THE SYSTEM SHALL NOT parse the seed and SHALL NOT write any row — so a restart never overwrites an edited article and never brings back an article that was deleted (miễn là bảng còn ít nhất một dòng).
17. WHEN a seed insert meets an `id` that already exists, THE SYSTEM SHALL leave that row untouched (`ON CONFLICT (id) DO NOTHING`), so two instances starting at the same time cannot overwrite each other or later edits.
18. WHEN the seed is invalid (no article, an empty or duplicate `id`, an empty `title`, a `publishedDate` that is not `YYYY-MM-DD`) or the database rejects it (kể cả khi migration `0012` chưa được apply), THE SYSTEM SHALL abort startup with a fatal log and SHALL leave the table unchanged (transaction rolled back).

## API contract (confirmed surface for this feature)

Đọc từ handler, `cmd/main.go` và chạy thử trên server dev. Đã có trong `docs/specs/api/openapi.yaml` (tag Articles) và `web/docs/public/openapi.yaml`.

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/articles/top` | public | `200` `{items: [ArticleSummary]}` | 405, 500 |
| GET | `/v1/articles?page=N` | public | `200` `{items, page, page_size, total, total_pages}` | 400 `INVALID_PAGE`, 405, 500 |
| GET | `/v1/articles/{id}` | public | `200` `ArticleDetail` | 404 `ARTICLE_NOT_FOUND`, 405, 500 |

`ArticleSummary`: `id` (slug, string), `title`, `category`, `author`, `author_initials`, `published_date` (`"YYYY-MM-DD"` | `null`), `read_minutes` (int), `summary`, `tags` (`[string]`), `image_url` (string | `null`), `thumbnail_url` (string | `null`).

`ArticleDetail` = `ArticleSummary` + `body_paragraphs` (`[string]`), `quote`, `quote_source`, `language` (`vi` | `en` trong dữ liệu hiện có), `source_name`, `source_url`, `license`.

## Out of scope (this feature boundary, not tracked here)

- Tạo/sửa/xoá bài viết qua API hoặc backoffice — hiện chỉ có SQL trực tiếp trên bảng `articles`.
- Tìm kiếm, lọc theo `category`/`tag`, bookmark, đánh dấu đã đọc.
- Lưu trữ/phân phối ảnh gốc: `image_url` vẫn là URL hotlink (trang nguồn của bài, hoặc Wikimedia Commons cho 7 bài trang nguồn không có ảnh), app không lưu bản sao ảnh gốc. Chỉ có ảnh nhỏ `thumbnail_url` được app lưu và phục vụ (`assets/images/articles/thumb/`, xem A-002); chưa có GCS/CDN.
- Đăng nhập: cả ba route công khai như `temple` và `deity`.

## Pending (not yet implemented)

- Test E2E chạy binary thật + Postgres cho ba route và seed lần đầu — `tasks.md` Phase 5.

> ⚠️ Not confirmed: quy trình cập nhật nội dung sau lần seed đầu — file JSON nhúng chỉ được đọc khi bảng rỗng, nên sửa file rồi build lại **không** đổi dữ liệu trên DB đã có; cần chốt dùng SQL tay, một lệnh reseed hay CRUD backoffice.
> ⚠️ Not confirmed: quyền dùng và độ bền của ảnh — `image_url` lấy từ thẻ `og:image` hoặc ảnh minh hoạ đầu thân bài của `source_url` (nguồn ảnh đã chốt: theo bài gốc), và với 7 bài mà trang nguồn không có ảnh thì lấy ảnh minh hoạ chủ đề trên Wikimedia Commons (giấy phép mở; ghi công ở `docs/assumptions/article/A-001-commons-illustrative-images.md`); nên là hotlink tới server của bên khác (phatgiao.org.vn, giacngo.vn, langmai.org, tricycle.org, plumvillage.org, upload/thumb.wikimedia.org); chưa xác nhận quyền hiển thị ảnh, và ảnh có thể hỏng nếu nguồn đổi đường dẫn hoặc chặn hotlink. Phương án dự phòng: chép về GCS (chưa làm).
> ⚠️ Not confirmed: điều kiện hiển thị và cách ghi công — mọi bài mang `license = "all-rights-reserved-summarized"` và `source_name`/`source_url` trỏ tới bài gốc của bên khác; chưa xác nhận app được hiển thị bản tóm tắt này theo điều kiện nào và có bắt buộc hiển thị nguồn hay không.
> ⚠️ Not confirmed: tiêu chí "nổi bật" — hiện là thứ tự biên tập trong file (nguồn không có ngày hay độ phổ biến đáng tin); chưa xác nhận với product rằng thứ tự này là đủ.

## Ghi chú kỹ thuật liên quan

- Dữ liệu seed: 60 bài, 58 tiếng Việt và 2 tiếng Anh; 29 bài có `publishedDate`, 60/60 bài có `imageURL`: 38 bài từ `og:image` của `source_url`, 15 bài từ ảnh trong trang nguồn (giacngo.vn, langmai.org; bỏ logo chung `logo_share.png`), 7 bài từ Wikimedia Commons (ảnh minh hoạ chủ đề, không phải ảnh của bài). Nghĩa là `published_date` là `null` ở 31 bài và `image_url` không còn `null` trong dữ liệu seed (API vẫn trả `null` nếu một dòng DB thiếu ảnh).
- `id` là slug ổn định lấy từ dữ liệu nguồn (vd. `vipassana-nghe-thuat-song`) nên client cache id vẫn đúng sau mọi lần nạp lại.
- Thứ tự `sort_order ASC, id ASC` là thứ tự biên tập trong file JSON. `id` là tie-break cuối để hai dòng cùng `sort_order` không đổi chỗ giữa các trang.
- `total` và trang được đọc bằng hai câu lệnh riêng (đếm rồi lấy trang), không cùng một snapshot — xem `design.md`.
- Bảng `articles` phải tồn tại trước khi service chạy (`make migrate` áp `migrations/0012_articles.sql`), nếu không bước seed lỗi và service không khởi động.
- Kích thước trang nằm ở block `article:` của `config.<env>.yaml` (dev: `top_limit: 5`, `page_size: 10`); cả hai bắt buộc — giá trị 0 nghĩa là trang rỗng.

## Open questions (resolved during implementation)

Chưa có câu hỏi nào được người dùng trả lời trong phiên bản này; các câu còn mở nằm ở "Pending" phía trên.
