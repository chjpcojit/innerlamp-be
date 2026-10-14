---
id: TD-001
feature: ratelimit
title: "Redis connection has no TLS option / Kết nối Redis chưa có tuỳ chọn TLS"
severity: medium
status: open
created: 2026-10-14
owner: chjpcojit
code: services/appService/cmd/ratelimit_store.go (newRateLimitStore), config rate_limit_store.redis
---

# TD-001 — Redis connection has no TLS option / Kết nối Redis chưa có tuỳ chọn TLS

## 1. Debt / Nợ gì

**EN** — `newRateLimitStore` builds a plain-TCP `redis.Client`. The `rate_limit_store.redis` block has address, password secret, db, prefix and timeouts, but no TLS settings (CA bundle, server name). The dev password also comes from `dev_secrets` (see `docs/tech-debt/config/TD-001`).

**VI** — `newRateLimitStore` tạo `redis.Client` qua TCP thường. Block `rate_limit_store.redis` có địa chỉ, tên secret mật khẩu, db, prefix và timeout nhưng chưa có cấu hình TLS (CA bundle, server name). Mật khẩu dev vẫn lấy từ `dev_secrets` (xem `docs/tech-debt/config/TD-001`).

## 2. Why accepted / Vì sao chấp nhận

**EN** — The ask was a working Redis on docker compose for local use. Compose Redis runs on localhost with no TLS, and there is no deployment target yet (no Memorystore instance, no IaC) to test TLS against.

**VI** — Yêu cầu là có Redis chạy được trên docker compose để dùng local. Redis của compose chạy trên localhost không TLS, và chưa có môi trường deploy (chưa có Memorystore, chưa có IaC) để kiểm thử TLS.

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — Deployed as is, the Redis AUTH password and the rate-limit keys (which contain client IPs and user ids) travel in clear text inside the VPC, and a Memorystore instance with in-transit encryption enabled cannot be reached at all.

**VI** — Nếu deploy nguyên trạng, mật khẩu AUTH của Redis và các key rate limit (chứa IP client và user id) đi dạng plaintext trong VPC, và không kết nối được Memorystore đã bật in-transit encryption.

## 4. Exit criteria / Điều kiện trả nợ

- [ ] Add `tls_enabled` and `ca_cert_secret` (or a CA file path) to `rate_limit_store.redis`, build `redis.Options.TLSConfig` from them / Thêm `tls_enabled` và `ca_cert_secret` (hoặc đường dẫn CA) vào `rate_limit_store.redis`, dựng `redis.Options.TLSConfig` từ đó
- [ ] Resolve the password from Secret Manager, not `dev_secrets`, outside dev / Ngoài dev, lấy mật khẩu từ Secret Manager thay vì `dev_secrets`
- [ ] Verify against a Memorystore instance with AUTH + in-transit encryption / Kiểm chứng trên Memorystore bật AUTH + in-transit encryption

## 5. Estimate / Ước lượng

**EN** — Half a day of code and tests; verification depends on the GCP environment existing.

**VI** — Nửa ngày code và test; phần kiểm chứng phụ thuộc việc có môi trường GCP.
