---
id: A-001
feature: genealogy
title: "Editor grant takes effect immediately / Cấp quyền sửa có hiệu lực ngay, không cần chấp nhận"
status: invalidated (2026-10-08) — anh Đức chốt phải chấp nhận lời mời; thay bằng lời mời, xem requirements.md v3.0 AC44–AC47, AC70–AC75
created: 2026-10-07
made-by: AI (Claude), while writing the genealogy spec
code: (chưa có code — áp dụng cho docs/specs/features/genealogy/ AC44, Step C-2)
---

# A-001 — Editor grant takes effect immediately / Cấp quyền sửa có hiệu lực ngay

## Assumption / Giả định

**EN** — When the owner grants edit access by phone number, the target user becomes `editor` immediately; there is no invitation to accept.

**VI** — Khi chủ gia phả cấp quyền sửa theo số điện thoại, người đó thành `editor` ngay, không có bước chấp nhận lời mời.

## Rationale / Lý do

**EN** — The user said "người tạo hoặc cấp quyền có thể sửa" and "mời theo số điện thoại"; "cấp quyền" reads as a direct grant. Skipping acceptance keeps v1 to one table and no notification channel.

**VI** — Anh Đức nói "cấp quyền" và chọn "mời theo số điện thoại"; "cấp quyền" hiểu là cấp thẳng. Bỏ bước chấp nhận giúp v1 chỉ cần một bảng và không cần kênh thông báo.

## Impact if wrong / Hệ quả nếu sai

**EN** — If acceptance is required, add a pending-invitation state/table, a notification path and accept/decline routes; AC44, Step C-2 and `flow.md` change.

**VI** — Nếu cần chấp nhận thì phải thêm trạng thái/bảng lời mời chờ, kênh thông báo và route chấp nhận/từ chối; AC44, Step C-2 và `flow.md` đổi theo.

## How to validate / Cách xác nhận

**EN** — Ask the user; on answer, resolve the `⚠️ Not confirmed` at AC44 in `requirements.md` and set `status` here.

**VI** — Hỏi anh Đức; khi có câu trả lời, chốt marker `⚠️ Not confirmed` ở AC44 trong `requirements.md` và cập nhật `status` ở đây.

## Resolution / Kết quả (2026-10-08)

**VI** — Giả định sai: anh Đức chốt người được mời phải nhận thông báo và chấp nhận hoặc từ chối; số chưa đăng ký cũng đi qua lời mời. Spec đổi sang v3.0 (bảng `genealogy_invitations`, route chấp nhận/từ chối). Các mặc định mới nằm ở `A-002`.
