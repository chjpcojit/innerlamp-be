---
id: A-007
feature: sutra
title: "The four sutra routes are public (no login) / Bốn route kinh là công khai (không cần đăng nhập)"
status: unvalidated
created: 2026-10-03
made-by: AI (Claude), while implementing the sutra feature
code: services/appService/cmd/main.go (route registration)
---

# A-007 — The four sutra routes are public (no login) / Bốn route kinh là công khai (không cần đăng nhập)

## 1. Assumption / Giả định

**EN** — The sutra routes are registered like the article, temple and deity routes: plain `mux.HandleFunc`, no auth middleware and no rate limit. The user asked for APIs the client can call to display the texts, and did not state an auth model.

**VI** — Các route kinh được đăng ký như route article, temple và deity: `mux.HandleFunc` thường, không middleware xác thực và không giới hạn tần suất. Người dùng yêu cầu API để client hiển thị văn bản và không nêu mô hình xác thực.

## 2. Rationale / Lý do

**EN** — All three sibling catalogues are public and the content is meant to be read freely; the old content spec was contradictory on this point.

**VI** — Cả ba catalogue anh em đều công khai và nội dung nhằm để đọc tự do; spec content cũ tự mâu thuẫn ở điểm này.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — If reading must require an account, the four routes need `RequireAuth` and the E2E tests need tokens; without a rate limit a scraper can pull the whole library (~11 MB) cheaply.

**VI** — Nếu đọc kinh phải có tài khoản thì bốn route cần `RequireAuth` và test E2E cần token; không có giới hạn tần suất thì kẻ cào dữ liệu lấy được cả thư viện (~11 MB) rất rẻ.

## 4. How to validate / Cách xác nhận

**EN** — Confirm with the product owner whether reading is public and whether the public routes need rate limiting.

**VI** — Xác nhận với người phụ trách sản phẩm: đọc kinh có công khai không và các route công khai có cần giới hạn tần suất không.
