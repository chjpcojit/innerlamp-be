---
id: TD-001
feature: auth
title: "Web API key is a plaintext dev secret and does not stop callers who omit it / Web API key là secret dev dạng plaintext và không chặn được caller bỏ header"
severity: medium
status: open
created: 2026-10-09
owner: ducnguyen-wee
code: services/appService/config/config.dev.yaml (dev_secrets.appservice-web-api-key), services/appService/delivery/http/web_api_key_middleware.go (RequireWebAPIKey), services/appService/delivery/http/login_handler.go (Login platform handling)
---

# TD-001 — Web API key is a plaintext dev secret and does not stop callers who omit it / Web API key là secret dev dạng plaintext và không chặn được caller bỏ header

## 1. Debt / Nợ gì

**EN** — Two gaps in the `X-API-Key` mechanism added for the web frontend (spec `docs/specs/features/login/` v1.12, AC16–AC19):
1. The key used for local runs was generated once by the assistant and stored as plaintext in `config.dev.yaml` (`dev_secrets.appservice-web-api-key`), which is checked into git. There is no production key, no rotation, and only one key can be valid at a time.
2. The key only authenticates callers that *claim* to be the website. A request with no `X-API-Key` header is treated as mobile and passes, and a caller can send `device.platform` other than `web`. Nothing else yet protects the pre-login routes (`request-otp`, `resend-otp`, `login`) from abuse such as SMS pumping or password guessing from outside the website.

**VI** — Hai lỗ hổng trong cơ chế `X-API-Key` thêm cho web (spec `docs/specs/features/login/` v1.12, AC16–AC19):
1. Key chạy local do trợ lý sinh một lần và lưu plaintext trong `config.dev.yaml` (`dev_secrets.appservice-web-api-key`), file này nằm trong git. Chưa có key production, chưa có xoay key, và tại một thời điểm chỉ một key hợp lệ.
2. Key chỉ xác thực caller *tự nhận* là website. Request không có header `X-API-Key` được coi là mobile và đi qua, caller cũng có thể gửi `device.platform` khác `web`. Chưa có lớp nào khác bảo vệ các route trước login (`request-otp`, `resend-otp`, `login`) khỏi lạm dụng như SMS pumping hay dò mật khẩu từ ngoài website.

## 2. Why accepted / Vì sao chấp nhận

**EN** — Decided by the user on 2026-10-09: web uses a shared key held by the SvelteKit server, mobile sends no key and will be protected by something else (attestation) later. The dev key unblocks the web team now; the service is not production-ready anyway (OTP is hardcoded `111111`, SMS is log-only).

**VI** — Anh Đức quyết định ngày 2026-10-09: web dùng key dùng chung do server SvelteKit giữ, mobile không gửi key và sẽ được bảo vệ bằng cơ chế khác (attestation) sau này. Key dev giúp team web làm việc ngay; dù sao service chưa sẵn sàng production (OTP hardcode `111111`, SMS chỉ ghi log).

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — Anyone with repo access knows the dev key (acceptable for dev only). If this reaches production unchanged: the key is public, and the no-key path is an open door to SMS cost abuse and brute force. Per-IP rate limiting at the BE is also ineffective while the web server is the only caller seen.

**VI** — Ai có quyền vào repo đều biết key dev (chấp nhận được cho dev). Nếu lên production nguyên trạng: key bị lộ, và đường "không có key" là cửa mở cho lạm dụng chi phí SMS và dò mật khẩu. Rate limit theo IP ở BE cũng vô hiệu khi chỉ thấy IP của server web.

## 4. Exit criteria / Điều kiện trả nợ

- [ ] Production key generated (`openssl rand -hex 32`) and stored in the secret manager; `NewMockSecretGetterFromYAML` replaced by `pkg/secrets/gcp` (same item as the existing `⚠️ DEV ONLY` markers)
- [ ] Dev key removed from git history or rotated before the first production deploy
- [ ] Captcha (e.g. Turnstile) on web `request-otp` / `resend-otp` / `login`, verified by the SvelteKit server
- [ ] Mobile protected for the same routes (attestation wired via `RequireAttestation`)
- [ ] Decide whether to support two valid keys during rotation

## 5. Estimate / Ước lượng

**EN** — About 1–2 days including the attestation wiring decision; captcha is on the web side.

**VI** — Khoảng 1–2 ngày gồm quyết định gắn attestation; captcha nằm ở phía web.
