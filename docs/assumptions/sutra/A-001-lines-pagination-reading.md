---
id: A-001
feature: sutra
title: ""Page by number of lines" means page + page_size / "Phân trang theo số dòng" nghĩa là page + page_size"
status: unvalidated
created: 2026-10-03
made-by: AI (Claude), while implementing the sutra feature
code: services/appService/delivery/http/sutra_handler.go (Lines); services/appService/usecase/sutra_usecase.go (Lines)
---

# A-001 — "Page by number of lines" means page + page_size / "Phân trang theo số dòng" nghĩa là page + page_size

## 1. Assumption / Giả định

**EN** — The user asked to load a long text "page by page by number of lines". It was read as `GET /v1/sutras/{id}/lines?translation=&page=N&page_size=M`, where `page_size` is the number of lines per page and is optional (default 200, capped at 500) — instead of a cursor (`after`/`limit`).

**VI** — Người dùng yêu cầu tải văn bản dài "từng page theo số dòng". Điều này được hiểu là `GET /v1/sutras/{id}/lines?translation=&page=N&page_size=M`, trong đó `page_size` là số dòng mỗi trang và là tuỳ chọn (mặc định 200, tối đa 500) — thay vì con trỏ (`after`/`limit`).

## 2. Rationale / Lý do

**EN** — It matches the other catalogues (`page`) and, because `n` is contiguous, a line's page is `floor((n-1)/page_size)+1`, so a table-of-contents entry can be turned into a page.

**VI** — Khớp các catalogue khác (`page`) và vì `n` liên tục nên trang của một dòng là `floor((n-1)/page_size)+1`, nhờ đó một mục trong mục lục đổi được thành số trang.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — If the client wants a cursor or fixed pages, the endpoint and its tests change; clients that vary `page_size` between calls would see shifted pages.

**VI** — Nếu client muốn con trỏ hoặc trang cố định thì endpoint và test phải đổi; client đổi `page_size` giữa các lần gọi sẽ thấy các trang dịch chỗ.

## 4. How to validate / Cách xác nhận

**EN** — Ask the iOS developer whether `page` + `page_size` fits the reader screen; recorded in the sutra changelog v2.0.

**VI** — Hỏi lập trình viên iOS xem `page` + `page_size` có hợp màn hình đọc không; đã ghi ở changelog sutra v2.0.
