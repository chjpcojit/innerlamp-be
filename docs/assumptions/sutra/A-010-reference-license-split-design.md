---
id: A-010
feature: sutra
title: "How reference and license were split / Cách tách reference và license"
status: unvalidated
created: 2026-10-07
made-by: AI (Claude), after the user asked to split the packed fields
code: assets/data/sutras/ (153 files + index.json); migrations/0020_sutra_reference_license_summary.sql; services/appService/repository/sutra_seed.go
---

# A-010 — Reference / license split / Tách reference và license

## 1. Assumption / Giả định

**EN** — (a) `reference` keeps only the citation code; the Hán/original title, author, Hán translator, year (CE) and the leftover remark become `original_title`, `author`, `han_translator`, `han_translation_year`, `reference_note`. For a dharani whose old `reference` was the name of its source scripture ("Kinh Dược Sư"), that name goes to `original_title`. (b) `license` becomes a code in a closed set (`CC0-1.0`, `CC-BY-NC-4.0`, `PUBLIC_DOMAIN`, `COPY_ALLOWED`, `UNKNOWN`); `license_url` holds a URL only when the source gave one; `license_note` holds the source's own words. A page that only says "copying is welcome" (phatgiao.org.vn) is `COPY_ALLOWED`, not a standard licence. (c) The six new sutra fields appear in the detail only; the list keeps its card shape and only `license` (the code) reaches it. (d) `note` of a translation is left as is.

**VI** — (a) `reference` chỉ còn mã trích dẫn; tên Hán/tên gốc, tác giả, người dịch Hán, năm (CN) và ghi chú còn lại thành `original_title`, `author`, `han_translator`, `han_translation_year`, `reference_note`. Với bài chú mà `reference` cũ là tên kinh gốc ("Kinh Dược Sư") thì tên đó vào `original_title`. (b) `license` thành mã trong tập đóng; `license_url` chỉ có khi nguồn cho URL; `license_note` giữ nguyên lời của nguồn. Trang chỉ ghi "được sao chép" (phatgiao.org.vn) là `COPY_ALLOWED`, không phải giấy phép chuẩn. (c) Sáu trường mới của kinh chỉ ở chi tiết; danh sách giữ dạng thẻ, chỉ thêm mã `license`. (d) `note` của bản dịch giữ nguyên.

## 2. Rationale / Lý do

**EN** — The old strings mixed several facts, so a client could not show them separately. The mapping of 48 texts was written by hand from the old strings without adding facts, so nothing is lost; the one exception is that the Sanskrit titles of `kim-cang` and `tam-kinh` were dropped because `titles.sa` already holds them.

**VI** — Chuỗi cũ gộp nhiều thông tin nên client không hiển thị riêng được. Bảng ánh xạ 48 văn bản viết tay từ chuỗi cũ, không thêm dữ kiện nên không mất gì; ngoại lệ duy nhất là tiêu đề Phạn của `kim-cang` và `tam-kinh` bị bỏ vì `titles.sa` đã có.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — A field filled in the wrong place (e.g. `dia-tang`: the source names "Pháp Đăng" as the Hán translator while also naming Śikṣānanda; the data keeps Śikṣānanda and the original words in `reference_note`) shows wrongly on screen; fix by editing the JSON and re-seeding. If iOS wants the code set renamed, it is a data + parser change.

**VI** — Trường đặt sai chỗ (vd. `dia-tang`: nguồn ghi "Pháp Đăng" là người dịch Hán nhưng lại nêu Śikṣānanda; dữ liệu giữ Śikṣānanda và lời gốc ở `reference_note`) sẽ hiện sai; sửa bằng chỉnh JSON rồi seed lại. Nếu iOS muốn đổi tên mã thì đổi dữ liệu và parser.
