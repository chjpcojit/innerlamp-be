---
id: TD-008
feature: sutra
title: "Seed timeout is hardcoded and config validation is inconsistent / Timeout nạp dữ liệu viết cứng và kiểm tra config chưa nhất quán"
severity: low
status: open
created: 2026-10-03
owner: ducnguyen-wee
code: services/appService/cmd/seedsutras/main.go (seedTimeout); services/appService/cmd/seeddharmatalks/main.go (seedTimeout); services/appService/config/config.go (SutraConfig.Validate)
---

# TD-008 — Seed timeout is hardcoded and config validation is inconsistent / Timeout nạp dữ liệu viết cứng và kiểm tra config chưa nhất quán

## 1. Debt / Nợ gì

**EN** — *(Update 2026-10-11: appService no longer seeds on startup, so the constant in `cmd/main.go` is gone; the number now lives only in the seed commands.)* `seedTimeout = 2 * time.Minute` is a constant in `cmd/seedsutras/main.go` (and in `cmd/seeddharmatalks/main.go`); neither is in `config/`, although a different environment (a remote database) may need another value. `SutraConfig.Validate()` is called by hand in `cmd/main.go` only (not by `seedsutras`, not inside `Load`), while the other catalogue configs (temple, deity, article) are not validated at all and a zero there silently means empty pages.

**VI** — *(Cập nhật 2026-10-11: appService không còn seed lúc khởi động nên hằng số ở `cmd/main.go` đã bỏ; con số nay chỉ còn ở các lệnh seed.)* `seedTimeout = 2 * time.Minute` là hằng số trong `cmd/seedsutras/main.go` (và `cmd/seeddharmatalks/main.go`); cả hai đều không nằm trong `config/`, dù môi trường khác (DB từ xa) có thể cần giá trị khác. `SutraConfig.Validate()` chỉ được gọi tay ở `cmd/main.go` (không phải `seedsutras`, không nằm trong `Load`), còn config của các catalogue khác (temple, deity, article) không được kiểm tra gì và giá trị 0 ở đó âm thầm có nghĩa là trang rỗng.

## 2. Why accepted / Vì sao chấp nhận

**EN** — `Load` validates nothing today and four existing tests feed it YAML without catalogue blocks, so validating inside `Load` would have broken them; whether the timeout belongs in config was asked and not yet answered.

**VI** — Hiện `Load` không kiểm tra gì và bốn test có sẵn nạp YAML không có block catalogue, nên kiểm tra trong `Load` sẽ làm hỏng chúng; câu hỏi timeout có thuộc config hay không đã được nêu và chưa được trả lời.

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — A slow remote database can exceed 2 minutes and the service then refuses to start on first run; tuning needs a code change; a mistyped `page_size: 0` for another catalogue goes unnoticed.

**VI** — DB từ xa chậm có thể vượt 2 phút và service từ chối khởi động ở lần chạy đầu; chỉnh giá trị cần sửa code; gõ nhầm `page_size: 0` cho catalogue khác không ai phát hiện.

## 4. Exit criteria / Điều kiện trả nợ

- [ ] Answer whether the timeout is configuration; if yes add `SutraConfig.SeedTimeout` and use it in both entry points / Trả lời timeout có phải cấu hình không; nếu có thì thêm `SutraConfig.SeedTimeout` và dùng ở cả hai điểm vào
- [ ] Validate every catalogue config in one place (inside `Load`) and update the four old `Load` test fixtures / Kiểm tra mọi config catalogue ở một chỗ (trong `Load`) và sửa bốn fixture test cũ của `Load`

## 5. Estimate / Ước lượng

**EN** — 2 hours.

**VI** — 2 giờ.
