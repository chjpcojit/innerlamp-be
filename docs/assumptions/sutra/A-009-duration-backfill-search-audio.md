---
id: A-009
feature: sutra
title: "Duration backfill at 4 lines/min, search semantics, audio_url empty string / Điền duration 4 dòng/phút, ngữ nghĩa tìm kiếm, audio_url rỗng"
status: unvalidated
created: 2026-10-07
made-by: AI (Claude), answering an iOS request (3 additions to /v1/sutras)
code: migrations/0019_sutra_audio_search.sql; assets/data/sutras/ (10 files + index.json); services/appService/repository/postgres_sutra_repository.go (sutraWhere, likeContains)
---

# A-009 — Duration backfill, search semantics, audio_url / Điền duration, ngữ nghĩa tìm kiếm, audio_url

## 1. Assumption / Giả định

**EN** — (a) The ten texts with `duration_minutes = null` (Vajrayana, no Vietnamese version) get `ceil(lines of the first translation / 4)`, at least 1; 4 lines a minute is the median of the 143 hand-made estimates. (b) `q` is a substring match, ignoring case and diacritics (`unaccent`), over every title, `collection` and `reference`, ANDed with the other filters, in curated order (no relevance ranking), 1–100 characters after trimming. (c) `audio_url` is a plain `TEXT NOT NULL DEFAULT ''` column; no audio exists yet, so every value is `""`.

**VI** — (a) Mười văn bản có `duration_minutes = null` (Kim Cang Thừa, không có bản tiếng Việt) được điền `ceil(số dòng bản dịch đầu / 4)`, tối thiểu 1; 4 dòng mỗi phút là trung vị của 143 ước tính làm tay. (b) `q` khớp một phần chuỗi, không phân biệt hoa thường và dấu (`unaccent`), trên mọi title, `collection`, `reference`, kết hợp AND với bộ lọc khác, theo thứ tự biên tập (không xếp theo độ liên quan), 1–100 ký tự sau khi trim. (c) `audio_url` là cột `TEXT NOT NULL DEFAULT ''`; chưa có audio nên mọi giá trị là `""`.

## 2. Rationale / Lý do

**EN** — iOS asked for "a rough estimate, not exact" and for no nulls. The hand-made estimates vary from 1.5 to 19 lines a minute, so a single rate is crude; the median is the least surprising. 153 rows make an unindexed scan trivial, and `unaccent` is built into Postgres and Cloud SQL. `""` instead of null follows the iOS request.

**VI** — iOS xin "ước tính chung chung, không cần chính xác" và không còn null. Các ước tính làm tay dao động 1,5–19 dòng/phút nên một tốc độ chung là thô; trung vị ít gây bất ngờ nhất. 153 dòng nên quét không index là rẻ, `unaccent` có sẵn trong Postgres và Cloud SQL. `""` thay vì null theo yêu cầu của iOS.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — A long Vajrayana text may show a duration far from the real chanting time; fix by editing `duration_minutes` (data only). If the library grows to thousands of rows, `q` needs a trigram index. A `max_minutes` filter now includes these ten texts.

**VI** — Một văn bản Kim Cang Thừa dài có thể hiện thời lượng lệch xa thực tế; sửa bằng chỉnh `duration_minutes` (chỉ dữ liệu). Nếu thư viện lên hàng nghìn dòng, `q` cần index trigram. Bộ lọc `max_minutes` giờ bao gồm cả mười văn bản này.

## 4. How to validate / Cách xác nhận

**EN** — Ask the content owner to review the ten values; have iOS confirm the search feels right on real titles; load audio into `sutras.audio_url` when recordings exist.

**VI** — Nhờ người phụ trách nội dung soát mười giá trị; nhờ iOS xác nhận tìm kiếm đúng ý trên title thật; nạp audio vào `sutras.audio_url` khi có bản ghi.
