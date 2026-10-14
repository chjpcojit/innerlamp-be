---
id: TD-001
feature: memorial
title: "Memorial photo upload is not built; blocked on GCS / Chưa làm upload ảnh sổ giỗ; bị chặn bởi GCS"
severity: medium
status: open
created: 2026-10-07
owner: ducnguyen-wee
code: services/appService/cmd/main.go (memorial routes), services/appService/delivery/http/memorial_handler.go (memorialResponse.PhotoURL), migrations/0018_memorials.sql (photo_object_key)
---

# TD-001 — Memorial photo upload is not built; blocked on GCS / Chưa làm upload ảnh sổ giỗ; bị chặn bởi GCS

## 1. Debt / Nợ gì

**EN** — US-MEM-05 (upload, replace and remove one photo per memorial) is not implemented. v1 ships the five CRUD routes only: `photo_url` is always JSON `null`, `memorials.photo_object_key` is always `NULL`, and `PUT/DELETE /v1/memorials/{id}/photo` do not exist. The design (multipart through the backend into a private GCS bucket, 15-minute signed URLs, DB-first-then-object ordering) is already specced in `docs/specs/features/memorial/` and tracked as `tasks.md` Phase 6. The repo has never called `pkg/storage` for any feature, and the only configured bucket (`storage.default_bucket: innerlamp-avatars-dev`) is not a decided home for memorial photos. Whether the Cloud Run service account may sign URLs (`iam.serviceAccounts.signBlob`) is unknown.

**VI** — US-MEM-05 (tải, thay và gỡ một ảnh cho mỗi mục sổ giỗ) chưa được implement. v1 chỉ có năm route CRUD: `photo_url` luôn là JSON `null`, cột `memorials.photo_object_key` luôn `NULL`, và `PUT/DELETE /v1/memorials/{id}/photo` chưa tồn tại. Thiết kế (multipart qua backend vào bucket GCS riêng tư, signed URL 15 phút, ghi DB trước rồi mới đụng object) đã có trong `docs/specs/features/memorial/` và được theo dõi ở `tasks.md` Phase 6. Repo chưa từng gọi `pkg/storage` cho feature nào, bucket duy nhất đang cấu hình (`storage.default_bucket: innerlamp-avatars-dev`) chưa được chốt làm chỗ chứa ảnh sổ giỗ. Chưa rõ service account của Cloud Run có quyền ký URL (`iam.serviceAccounts.signBlob`) hay không.

## 2. Why accepted / Vì sao chấp nhận

**EN** — Decided by the user on 2026-10-07: skip GCS for now because it cannot be done yet. The photo is optional, so the book is fully usable without it. The column and the always-`null` response field are kept so adding photos later needs no migration and no client contract change.

**VI** — Anh Đức quyết định ngày 2026-10-07: tạm bỏ qua GCS vì chưa làm được. Ảnh là tuỳ chọn nên sổ giỗ vẫn dùng đầy đủ khi không có ảnh. Cột và field `photo_url` luôn `null` được giữ lại để sau này thêm ảnh không cần migration và không đổi hợp đồng với client.

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — Users cannot attach a portrait, which the original request asked for. Clients that build the photo UI against `photo_url` will see `null` forever. When photos arrive, orphaned objects (failed deletes) have no cleanup mechanism yet — deferred by the user as "tính sau" and tracked with this item.

**VI** — Người dùng không gắn được di ảnh, trong khi yêu cầu ban đầu có. Client dựng giao diện ảnh theo `photo_url` sẽ thấy `null` mãi. Khi có ảnh, object mồ côi (xoá thất bại) chưa có cơ chế dọn — anh Đức để "tính sau" và được theo dõi chung với khoản nợ này.

## 4. Exit criteria / Điều kiện trả nợ

- [ ] Decide the bucket (shared `innerlamp-avatars-dev` with a `memorials/` prefix, or a dedicated one) and confirm the Cloud Run service account can sign URLs / Chốt bucket (dùng chung `innerlamp-avatars-dev` với prefix `memorials/` hay bucket riêng) và xác nhận service account Cloud Run ký được URL
- [ ] Do `docs/specs/features/memorial/tasks.md` Phase 6 (port, GCS adapter, usecase, two photo routes wrapped by `authMiddleware`, config, openapi) / Làm `tasks.md` Phase 6
- [ ] E2E: user B gets 404 on user A's photo routes, and an upload for someone else's memorial writes nothing to the bucket / E2E: user B nhận 404 trên route ảnh của user A và upload cho mục của người khác không ghi gì vào bucket
- [ ] Decide how orphaned objects are cleaned up (bucket lifecycle rule or a sweep job) / Chốt cách dọn object mồ côi (lifecycle rule hoặc job quét)
- [x] Decision: HEIC is converted to JPEG by the client, backend accepts jpeg/png/webp only — confirmed 2026-10-07 (now product spec memorial FR-MEM-011; the assumption file A-001 was deleted 2026-10-14) / Quyết định: client đổi HEIC sang JPEG, backend chỉ nhận jpeg/png/webp — đã xác nhận 2026-10-07
- [ ] iOS and Android clients implement the HEIC → JPEG conversion before calling the photo route / Client iOS và Android cài đặt việc đổi HEIC → JPEG trước khi gọi route ảnh

## 5. Estimate / Ước lượng

**EN** — About 1–1.5 days once the bucket and signing permission are settled; the bucket decision itself is outside the code.

**VI** — Khoảng 1–1,5 ngày sau khi chốt bucket và quyền ký URL; việc chốt bucket nằm ngoài code.
