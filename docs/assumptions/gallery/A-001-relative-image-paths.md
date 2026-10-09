---
id: A-001
feature: gallery
title: "Catalogue images stored as root-relative paths / Ảnh danh mục lưu path tương đối"
status: unvalidated
created: 2026-10-08
made-by: AI (Claude), while fixing images breaking after a network change
code: services/appService/delivery/http/image_url.go
---

## 1. Assumption / Giả định

**EN** — `image_url` / `thumbnail_url` of temples, deities, gallery, articles (and the sutra deity thumbnail) are stored as `/static/images/...`. The API prepends `server.public_base_url`, else the request's origin (`X-Forwarded-Proto` + `Host`). Values not starting with `/` (absolute URLs, bare names, empty) are returned unchanged.

**VI** — Các cột ảnh lưu dạng `/static/images/...`; API ghép `server.public_base_url`, nếu rỗng thì dùng origin của request. Giá trị không bắt đầu bằng `/` giữ nguyên.

## 2. Rationale / Lý do

**EN** — Seed data carried `http://<dev IP>:8080/...`, so every IP change broke all images. Calendar already resolved URLs per request. Mobile clients still receive absolute URLs (no contract change).

**VI** — Seed cũ hardcode IP LAN nên đổi mạng là hỏng toàn bộ ảnh. Client vẫn nhận URL tuyệt đối.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — Behind a proxy that does not forward `Host`/`X-Forwarded-Proto`, derived URLs would be wrong: set `server.public_base_url` in production. Migration 0021 is not reversible.

**VI** — Sau proxy không chuyển tiếp Host/Proto thì URL suy ra sai: production phải đặt `server.public_base_url`.

## 4. How to validate / Cách xác nhận

**EN** — Restart appService, `curl -H 'Host: x:1' localhost:8080/v1/gallery/top` → URLs start with `http://x:1/static/images/`; set `public_base_url` → the configured origin wins.

**VI** — Khởi động lại appService rồi curl với header Host khác; URL phải theo Host, hoặc theo `public_base_url` nếu có.
