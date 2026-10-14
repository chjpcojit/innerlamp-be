---
id: TD-001
feature: config
title: "Dev secrets are plaintext in config.dev.yaml, resolved by a mock getter / Secret dev nằm plaintext trong config.dev.yaml, được giải bằng mock getter"
severity: medium
status: open
created: 2026-10-11
owner: ducnguyen-wee
code: services/appService/config/config.dev.yaml (dev_secrets), services/appService/config/ (NewMockSecretGetterFromYAML), services/appService/cmd/main.go and cmd/seed*/main.go (⚠️ DEV ONLY)
---

# TD-001 — Dev secrets are plaintext in config.dev.yaml / Secret dev nằm plaintext trong config.dev.yaml

## 1. Debt / Nợ gì

**EN** — Local runs resolve every `*_secret` name (database password, JWT signing key, web API key, ...) from a `dev_secrets:` block that holds literal values in `config.dev.yaml`, through `NewMockSecretGetterFromYAML`. The file is checked into git. All binaries share this path: `cmd/main.go` and every seed command (`seedtemples`, `seeddeities`, `seedsutras`, `seedgallery`, `seedarticles`, `seedadminunits`). The rest of the file is correctly non-sensitive config (limits, page sizes, rate limits, cache lifetime).

**VI** — Chạy local giải mọi tên `*_secret` (mật khẩu DB, khoá ký JWT, web API key, ...) từ khối `dev_secrets:` chứa giá trị thật trong `config.dev.yaml`, qua `NewMockSecretGetterFromYAML`. File nằm trong git. Mọi binary dùng chung đường này: `cmd/main.go` và mọi lệnh seed (`seedtemples`, `seeddeities`, `seedsutras`, `seedgallery`, `seedarticles`, `seedadminunits`). Phần còn lại của file đúng là cấu hình không nhạy cảm (giới hạn, kích thước trang, rate limit, thời hạn cache).

## 2. Why accepted / Vì sao chấp nhận

**EN** — Policy (AGENTS.md §5.3): config files hold non-sensitive data; sensitive values go to a secure store. In **dev** the shortcut is allowed provided it is recorded here and marked `⚠️ DEV ONLY` where used. The values are for a local container only and are not production values.

**VI** — Quy ước (AGENTS.md §5.3): file config chỉ chứa dữ liệu không nhạy cảm; giá trị nhạy cảm đưa lên nơi an toàn. Ở **dev** được phép đi tắt miễn là ghi ở đây và đánh dấu `⚠️ DEV ONLY` tại chỗ dùng. Giá trị chỉ cho container local, không phải giá trị production.

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — If the same file or the same pattern reaches staging or production, secrets sit in git and in the image. Seed commands have no production path at all (they cannot read a secret manager), so a production load would be done by hand with whatever config is at hand.

**VI** — Nếu cùng file hay cùng cách làm này lên staging/production thì secret nằm trong git và trong image. Các lệnh seed chưa có đường chạy production (không đọc được secret manager), nên nạp dữ liệu production sẽ làm tay bằng config nào có sẵn.

## 4. Exit criteria / Điều kiện trả nợ

- [ ] Real `secrets.Client` (`pkg/secrets/gcp`) wired in `cmd/main.go` for non-local environments, selected by config / `secrets.Client` thật (`pkg/secrets/gcp`) nối vào `cmd/main.go` cho môi trường không phải local, chọn qua config
- [ ] Seed commands share one config-loading helper that can use the same secret source / Các lệnh seed dùng chung một hàm nạp config có thể dùng cùng nguồn secret
- [ ] Production and staging keys generated and stored in the secret manager; `dev_secrets` absent from non-dev config files / Key production và staging được sinh và lưu trong secret manager; `dev_secrets` không có trong file config ngoài dev
- [ ] Related item closed: `docs/tech-debt/auth/TD-001` (web API key) / Đóng cùng mục liên quan: `docs/tech-debt/auth/TD-001` (web API key)

## 5. Estimate / Ước lượng

**EN** — 1–2 days including a documented seed procedure per environment.

**VI** — 1–2 ngày, gồm quy trình seed có tài liệu cho từng môi trường.
