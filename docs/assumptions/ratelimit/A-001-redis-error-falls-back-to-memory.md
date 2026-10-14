---
id: A-001
feature: ratelimit
title: "A Redis error falls back to a per-instance count / Redis lỗi thì chuyển sang đếm theo instance"
status: unvalidated
created: 2026-10-14
made-by: AI (Claude), while wiring the rate limiters to Redis
code: services/appService/cmd/ratelimit_store.go (newRateLimitStore), pkg/ratelimit/store/fallback.go
---

# A-001 — A Redis error falls back to a per-instance count / Redis lỗi thì chuyển sang đếm theo instance

## 1. Assumption / Giả định

**EN** — When Redis cannot answer a limit check (down, timeout past `op_timeout`, pool exhausted), the check is answered by an in-memory count on that instance instead of letting the request through (the middleware's fail-open) or refusing it (fail-closed). At startup, an unreachable Redis stops the service.

**VI** — Khi Redis không trả lời được lần kiểm tra limit (down, quá `op_timeout`, hết pool), lần kiểm tra đó được trả lời bằng bộ đếm memory của instance, thay vì cho request đi qua (fail-open của middleware) hoặc từ chối (fail-closed). Lúc khởi động, Redis không kết nối được thì service dừng.

## 2. Rationale / Lý do

**EN** — Fail-open turns every limit off exactly when an attacker could cause the outage; fail-closed turns a cache outage into a full API outage. A per-instance count keeps a limit in force (weaker by the number of instances) and keeps the API up. Failing at startup matches the repo's connect-at-start convention.

**VI** — Fail-open tắt mọi limit đúng lúc attacker có thể gây ra sự cố; fail-closed biến sự cố cache thành sự cố toàn bộ API. Đếm theo instance vẫn giữ limit (yếu đi theo số instance) và API vẫn chạy. Dừng lúc khởi động khớp quy ước connect-at-start của repo.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — If the product wants auth routes (login, OTP) to fail closed during an outage, those limiters need their own policy. If instances should start without Redis, the startup ping must become a warning.

**VI** — Nếu sản phẩm muốn các route auth (login, OTP) fail-closed khi sự cố, các limiter đó cần policy riêng. Nếu instance cần khởi động được khi không có Redis, ping lúc khởi động phải đổi thành cảnh báo.

## 4. How to validate / Cách xác nhận

**EN** — Confirm with the user. Verified behaviour on 2026-10-14: with Redis stopped mid-run, a 60/min route kept answering 200 and returned 429 on the 61st request, and one warning was logged.

**VI** — Xác nhận với anh Đức. Đã kiểm chứng ngày 2026-10-14: tắt Redis khi service đang chạy, route 60/phút vẫn trả 200 và trả 429 ở request thứ 61, có một dòng cảnh báo trong log.
