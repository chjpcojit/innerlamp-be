---
id: A-001
feature: article
title: "Illustrative Wikimedia Commons images for 7 articles / Ảnh minh hoạ Wikimedia Commons cho 7 bài viết"
status: unvalidated
created: 2026-10-06
made-by: AI (Claude), while filling missing article images
code: assets/data/buddhist_articles.json (imageURL of 7 articles)
---

# A-001 — Illustrative Wikimedia Commons images for 7 articles / Ảnh minh hoạ Wikimedia Commons cho 7 bài viết

## 1. Assumption / Giả định

**EN** — Seven articles have no usable image on their source page (langmai.org pages with only the site logo, a text-only budsas.org page, one dead image host). The user asked to fill them "from the internet", so each got a topic-matching, openly licensed photo from Wikimedia Commons (1200 px thumbnail, hotlinked). They illustrate the theme, they are not the article's own picture; some are loose matches (the tea article uses a Japanese tea ceremony, the loving-speech article a photo of Thích Nhất Hạnh).

**VI** — Bảy bài không có ảnh dùng được ở trang nguồn (trang langmai.org chỉ có logo, trang budsas.org chỉ có chữ, một host ảnh đã chết). Người dùng yêu cầu lấy ảnh trên mạng cho đủ nên mỗi bài được gắn một ảnh đúng chủ đề, giấy phép mở, từ Wikimedia Commons (bản 1200 px, hotlink). Đây là ảnh minh hoạ chủ đề, không phải ảnh của bài; vài ảnh chỉ khớp tương đối (bài thiền trà dùng ảnh trà đạo Nhật, bài ái ngữ dùng ảnh Thiền sư Thích Nhất Hạnh).

| Article id | Commons file | Author | Licence |
|---|---|---|---|
| so-hai-chap-nhan-va-tha-thu | [A symbol of world peace.....the lotus](https://commons.wikimedia.org/wiki/File:A_symbol_of_world_peace.....the_lotus_.....(31779886760).jpg) | lensnmatter | CC BY 2.0 |
| an-com-chanh-niem | [People eating together… Buddhist temple in Laos](https://commons.wikimedia.org/wiki/File:People_eating_together_sitting_on_the_floor_in_a_Buddhist_temple_in_Laos.jpg) | Basile Morin | CC BY-SA 4.0 |
| dao-phat-va-chu-hieu | [Vu Lan Thang Hoi in 2015](https://commons.wikimedia.org/wiki/File:Vu_Lan_Thang_Hoi_in_2015.jpg) | Vuong Tri Binh | CC BY-SA 4.0 |
| gioi-thu-tu-ai-ngu-va-lang-nghe | [Thich Nhat Hanh in Vietnam 2](https://commons.wikimedia.org/wiki/File:Thich_Nhat_Hanh_in_Vietnam_2.jpg) | d nelson | CC BY 2.0 |
| thien-tra-lang-mai | [Japan tea ceremony 1165](https://commons.wikimedia.org/wiki/File:Japan_tea_ceremony_1165.jpg) | Ermell | CC BY-SA 4.0 |
| an-chay-de-nuoi-duong-long-tu-bi | [Saigon Vegetarian food](https://commons.wikimedia.org/wiki/File:Saigon_Vegetarian_food.jpg) | Takeaway | CC BY-SA 3.0 |
| chanh-niem-hanh-phuc-ngay-bay-gio | [Thich Nhat Hanh](https://commons.wikimedia.org/wiki/File:Thich_Nhat_Hanh.JPG) | Lưu Ly | Public domain |

## 2. Rationale / Lý do

**EN** — An empty thumbnail looks broken in the list; an honest topic illustration with a traceable licence is better than none, and better than a mismatched or unlicensed picture. Commons metadata gives author and licence per file.

**VI** — Thumbnail trống trông như lỗi; ảnh minh hoạ đúng chủ đề, có giấy phép truy vết được thì tốt hơn không có ảnh, và hơn ảnh sai chủ đề hay không rõ bản quyền. Commons có sẵn tác giả và giấy phép cho từng file.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — CC BY / CC BY-SA require attribution; the app does not show image credits yet, so this table is the only record. If the product wants a different image or no image, edit `imageURL` (null it) and run `seedarticles -fill-images` only for new rows — existing DB rows need an SQL `UPDATE`. Hotlinks to `thumb.wikimedia.org` can break if Wikimedia changes thumbnail paths.

**VI** — CC BY / CC BY-SA yêu cầu ghi công; app chưa hiển thị credit ảnh nên bảng này là nơi duy nhất lưu thông tin đó. Nếu product muốn ảnh khác hoặc không ảnh thì sửa `imageURL` (đặt `null`); `seedarticles -fill-images` chỉ điền dòng đang `NULL`, dòng đã có trong DB phải `UPDATE` bằng SQL. Hotlink `thumb.wikimedia.org` có thể hỏng nếu Wikimedia đổi đường dẫn thumbnail.

## 4. How to validate / Cách xác nhận

**EN** — Product reviews the 7 pictures and the attribution requirement (show a credit line, or copy the images to GCS with credits). Each URL returned HTTP 200 `image/jpeg` on 2026-10-06; re-check before release.

**VI** — Product xem lại 7 ảnh và yêu cầu ghi công (hiển thị dòng credit, hoặc chép ảnh về GCS kèm credit). Mỗi URL trả HTTP 200 `image/jpeg` ngày 2026-10-06; kiểm lại trước khi phát hành.
