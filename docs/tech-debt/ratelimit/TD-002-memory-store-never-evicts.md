---
id: TD-002
feature: ratelimit
title: "MemoryStore never evicts keys / MemoryStore không bao giờ dọn key"
severity: medium
status: open
created: 2026-10-14
owner: chjpcojit
code: pkg/ratelimit/store/memory.go (MemoryStore.Allow)
---

# TD-002 — MemoryStore never evicts keys / MemoryStore không bao giờ dọn key

## 1. Debt / Nợ gì

**EN** — `MemoryStore` adds one map entry per new key (client IP or user id) and never removes it, even after its window has passed. Every check also takes one store-wide `sync.Mutex`. appService now uses it in two places: as the whole store when `rate_limit_store.backend` is `memory`, and as the per-instance fallback while Redis is unreachable.

**VI** — `MemoryStore` thêm một entry vào map cho mỗi key mới (IP client hoặc user id) và không bao giờ xoá, kể cả khi window đã qua. Mỗi lần kiểm tra cũng giữ một `sync.Mutex` chung cho cả store. appService hiện dùng nó ở hai chỗ: là store chính khi `rate_limit_store.backend` là `memory`, và là fallback theo instance khi Redis không kết nối được.

## 2. Why accepted / Vì sao chấp nhận

**EN** — With `backend: redis` the memory store only fills during a Redis outage, and an instance restart clears it. The user asked for this item to be proposed, not fixed, in this change.

**VI** — Với `backend: redis`, memory store chỉ đầy lên khi Redis bị sự cố, và restart instance là sạch. Anh Đức yêu cầu chỗ này chỉ đề xuất sửa, chưa sửa trong thay đổi này.

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — During a long Redis outage, or with `backend: memory`, a client rotating source addresses (trivial with IPv6) grows the map without bound until the instance runs out of memory.

**VI** — Khi Redis sự cố lâu, hoặc khi dùng `backend: memory`, một client xoay địa chỉ nguồn (rất dễ với IPv6) làm map phình không giới hạn cho đến khi instance hết RAM.

## 4. Exit criteria / Điều kiện trả nợ

- [ ] Drop a key once its window has passed (lazy delete in `Allow` plus a periodic sweep), with a hard cap on the number of keys / Xoá key khi window đã qua (xoá lười trong `Allow` cộng một vòng quét định kỳ), có trần cứng số key
- [ ] Shard the mutex, or lock per key, so one hot key does not serialise every check / Chia nhỏ mutex hoặc lock theo key để một key nóng không chặn mọi lần kiểm tra
- [ ] Unit test: after the window passes the map is empty / Unit test: sau khi window qua thì map rỗng

## 5. Estimate / Ước lượng

**EN** — About half a day with tests.

**VI** — Khoảng nửa ngày kèm test.
