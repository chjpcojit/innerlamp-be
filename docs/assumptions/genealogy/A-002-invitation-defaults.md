---
id: A-002
feature: genealogy
title: "Invitation defaults / Mặc định của lời mời cộng tác"
status: unvalidated
created: 2026-10-08
made-by: AI (Claude), while updating the genealogy spec to v3.0
code: (chưa có code — áp dụng cho docs/specs/features/genealogy/ AC44–AC47, AC70–AC75, Step C-1/C-2, I-1/I-2)
---

# A-002 — Invitation defaults / Mặc định của lời mời cộng tác

> Cập nhật 2026-10-10: anh Đức đã xác nhận mục (2) lời mời chờ tính vào trần 20 và mục (3) hạn 30 ngày. Mục (1) đã được chốt: thông báo push là tính năng riêng làm sau, không dùng SMS. Mục (7) không còn áp dụng: vai trò trưởng tộc đã bỏ ở spec v5.0; vai trò nhận lời mời giờ là `viewer` (chỉ xem). Các mục (1), (4), (5), (6) vẫn chưa được xác nhận.

## Assumption / Giả định

**EN** — (1) "Notification" in v1 is an in-app invitation inbox (`GET /v1/invitations`); no push/SMS. (2) Pending invitations count toward the 20-collaborator cap. (3) An invitation expires after 30 days. (4) Invitations are matched to a person by phone number only. (5) An owner may re-invite a number that declined. (6) The invitee sees only the genealogy name and the inviter's phone number before accepting. (7) Clan head is still assigned directly, not by invitation.

**VI** — (1) "Thông báo" ở v1 là hộp lời mời trong app (`GET /v1/invitations`), chưa có push/SMS. (2) Lời mời chờ tính vào trần 20 người cộng tác. (3) Lời mời hết hạn sau 30 ngày. (4) Lời mời khớp người chỉ bằng số điện thoại. (5) Owner mời lại được số đã từ chối. (6) Người được mời trước khi chấp nhận chỉ thấy tên gia phả và số điện thoại người mời. (7) Trưởng tộc vẫn chỉ định thẳng, không qua lời mời.

## Rationale / Lý do

**EN** — Smallest design that satisfies "invited users get notified and accept or decline": no new notification channel, no change to register, bounded growth of pending rows, and a window limit that reduces phone-number-recycling risk.

**VI** — Thiết kế nhỏ nhất đáp ứng "user nhận thông báo và đồng ý hoặc từ chối": không thêm kênh thông báo, không đổi luồng register, số lời mời chờ bị chặn trên, và hạn 30 ngày giảm rủi ro số điện thoại bị cấp lại cho người khác.

## Impact if wrong / Hệ quả nếu sai

**EN** — (1) Push/SMS needs a delivery channel and a device-token or SMS path. (2)/(3) are one constant each plus a count query. (4) A user-id based match would need a register hook. (5) A cooldown needs a column or a query on `declined` rows. (7) Invitation for clan head means another status flow on `genealogies.clan_head_user_id`.

**VI** — (1) Push/SMS cần kênh gửi và token thiết bị hoặc đường SMS. (2)/(3) chỉ đổi một hằng và một truy vấn đếm. (4) Khớp theo user id thì phải móc vào register. (5) Giới hạn mời lại cần thêm cột hoặc truy vấn trên các hàng `declined`. (7) Trưởng tộc qua lời mời cần thêm luồng trạng thái cho `genealogies.clan_head_user_id`.

## How to validate / Cách xác nhận

**EN** — Ask the user; resolve the matching `⚠️ Not confirmed` markers (AC47, AC51, AC70, AC72, AC75) in `requirements.md` and set `status` here.

**VI** — Hỏi anh Đức; chốt các marker `⚠️ Not confirmed` tương ứng (AC47, AC51, AC70, AC72, AC75) trong `requirements.md` và cập nhật `status` ở đây.
