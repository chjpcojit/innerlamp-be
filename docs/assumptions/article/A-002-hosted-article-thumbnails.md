---
id: A-002
feature: article
title: "Hosted square thumbnails derived from hotlinked article images / Ảnh nhỏ vuông do app lưu, tạo từ ảnh hotlink của bài viết"
status: unvalidated
created: 2026-10-06
made-by: AI (Claude), while adding article thumbnail_url at the iOS client's request
code: assets/images/articles/thumb/ (60 files); assets/data/buddhist_articles.json (thumbnailURL); migrations/0016_article_thumbnail.sql
---

# A-002 — Hosted square thumbnails derived from hotlinked article images / Ảnh nhỏ vuông do app lưu, tạo từ ảnh hotlink của bài viết

## 1. Assumption / Giả định

**EN** — The user chose "add `thumbnail_url`, the backend hosts the small image". For each of the 60 articles the source `imageURL` was downloaded, centre-cropped to a square (bias 10% upward) and saved as a 360x360 WebP (3-39 KB, ~15 KB average, 1.0 MB in total) under `assets/images/articles/thumb/<id>.webp`, served by the dev static route `/static/images/`. The square crop is an assumption from the client's 84x84 card; a different card shape needs a different crop. `image_url` is unchanged (still the hotlink).

**VI** — Người dùng chọn "thêm `thumbnail_url`, backend host ảnh nhỏ". Với mỗi bài trong 60 bài, ảnh nguồn `imageURL` được tải về, cắt vuông ở giữa (lệch lên 10%) và lưu thành WebP 360x360 (3-39 KB, trung bình ~15 KB, tổng 1,0 MB) tại `assets/images/articles/thumb/<id>.webp`, phục vụ bởi route tĩnh dev `/static/images/`. Cắt vuông là giả định theo thẻ 84x84 của client; thẻ hình khác cần cách cắt khác. `image_url` giữ nguyên (vẫn là hotlink).

## 2. Rationale / Lý do

**EN** — The hero image is typically 1200x630: too heavy for a list card, and a hotlink the source site may block or change. A small hosted copy fixes both for the card, the same way temples and deities already work.

**VI** — Ảnh gốc thường 1200x630: quá nặng cho thẻ danh sách, lại là hotlink mà trang nguồn có thể chặn hoặc đổi. Một bản nhỏ do app lưu giải quyết cả hai cho thẻ, giống cách temple và deity đang làm.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — The thumbnails are derivative copies of images owned by other sites (`license: all-rights-reserved-summarized`, plus CC BY / CC BY-SA images from Commons that need attribution). Fine for development; before any release the product must either obtain permission, use its own images, or keep hotlinking. The URL host baked into the data is the dev Mac's LAN IP (`make lan-urls` rewrites it, and for rows already in the DB it uses SQL because `seedarticles` never overwrites a stored value). The files are not in git (`assets/` is untracked, TD-001).

**VI** — Các ảnh nhỏ là bản phái sinh của ảnh thuộc trang khác (`license: all-rights-reserved-summarized`, cộng ảnh CC BY / CC BY-SA từ Commons cần ghi công). Dùng khi phát triển thì được; trước khi phát hành, product phải xin phép, dùng ảnh riêng hoặc quay lại hotlink. Host trong URL của dữ liệu là IP LAN của Mac dev (`make lan-urls` ghi lại, và với dòng đã có trong DB thì dùng SQL vì `seedarticles` không ghi đè giá trị đã lưu). Các file chưa nằm trong git (`assets/` chưa được theo dõi, TD-001).

## 4. How to validate / Cách xác nhận

- [ ] Product / legal confirms whether re-hosting thumbnails of these images is acceptable / Product hoặc pháp lý xác nhận có được lưu lại ảnh nhỏ của các ảnh này không
- [ ] iOS confirms the 360x360 square crop suits the 84x84 card / iOS xác nhận ảnh vuông 360x360 hợp với thẻ 84x84
