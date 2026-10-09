---
id: A-008
feature: sutra
title: "Default page sizes: 20 sutras, 200 lines, cap 500 / Cỡ trang mặc định: 20 kinh, 200 dòng, tối đa 500"
status: unvalidated
created: 2026-10-03
made-by: AI (Claude), while implementing the sutra feature
code: services/appService/config/config.dev.yaml (sutra:); services/appService/config/config.go (SutraConfig)
---

# A-008 — Default page sizes: 20 sutras, 200 lines, cap 500 / Cỡ trang mặc định: 20 kinh, 200 dòng, tối đa 500

## 1. Assumption / Giả định

**EN** — `sutra.page_size: 20`, `lines_page_size: 200` and `lines_max_page_size: 500` were chosen by the AI. They are config values, so another environment can change them.

**VI** — `sutra.page_size: 20`, `lines_page_size: 200` và `lines_max_page_size: 500` do AI chọn. Chúng là giá trị config nên môi trường khác đổi được.

## 2. Rationale / Lý do

**EN** — 20 matches the temple list; 200 lines is about 21 KB and a screen or two of reading; 500 bounds a response at about 50 KB. Measured on loopback: 1.8–2.5 ms per request.

**VI** — 20 khớp danh sách chùa; 200 dòng khoảng 21 KB và một hai màn hình đọc; 500 giới hạn một response ở khoảng 50 KB. Đo trên loopback: 1,8–2,5 ms mỗi request.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — On mobile networks a different size may feel better; only config and docs would change, but clients may have hard-coded assumptions.

**VI** — Trên mạng di động cỡ khác có thể tốt hơn; chỉ config và tài liệu đổi, nhưng client có thể đã viết cứng giả định.

## 4. How to validate / Cách xác nhận

**EN** — Try the reader screen on a real device and adjust the three values in `config.<env>.yaml`.

**VI** — Thử màn hình đọc trên thiết bị thật và chỉnh ba giá trị trong `config.<env>.yaml`.
