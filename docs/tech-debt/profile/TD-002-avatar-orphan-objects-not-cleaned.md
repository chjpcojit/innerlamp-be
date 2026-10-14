---
id: TD-002
feature: profile
title: "Orphaned avatar objects are never cleaned / Object avatar mồ côi không bao giờ được dọn"
severity: low
status: open
created: 2026-10-11
owner: ducnguyen-wee
code: services/appService/usecase/profile_usecase.go (ConfirmAvatar, RemoveAvatar), config profile.avatar_tmp_prefix
---

# TD-002 — Orphaned avatar objects are never cleaned / Object avatar mồ côi không bao giờ được dọn

## 1. Debt / Nợ gì

**EN** — Two kinds of avatar objects stay in the bucket forever: (a) objects in the temporary folder, because uploads that were never confirmed are not removed and confirmed ones are copied, not moved; (b) an old avatar whose deletion failed after the new one was accepted (the failure is only logged, with no retry).

**VI** — Hai loại object avatar nằm lại bucket mãi: (a) object ở thư mục tạm, vì upload không được xác nhận không bị xoá và upload đã xác nhận chỉ được copy chứ không di chuyển; (b) avatar cũ mà việc xoá lỗi sau khi avatar mới đã được nhận (chỉ ghi log, không thử lại).

## 2. Why accepted / Vì sao chấp nhận

**EN** — Decided by the user on 2026-10-11: "the temporary folder is not cleaned for now, to be decided later", and a failed delete of the old file is accepted as garbage. The user-facing behaviour is correct either way; only storage grows.

**VI** — Anh Đức quyết định ngày 2026-10-11: "folder phụ chưa tính đến việc dọn, anh sẽ tính sau", và việc xoá file cũ lỗi thì chấp nhận để lại rác. Hành vi với người dùng đúng trong cả hai trường hợp; chỉ dung lượng storage tăng.

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — Storage cost grows without bound, slowly. Unconfirmed uploads are user images that no record points to and that are never deleted, which matters for privacy and for the future account-deletion feature (it cannot find them by user row).

**VI** — Chi phí storage tăng không giới hạn, chậm. Upload không xác nhận là ảnh của người dùng mà không bản ghi nào trỏ tới và không bao giờ bị xoá, điều này quan trọng với quyền riêng tư và với tính năng xoá tài khoản sau này (không tìm được chúng qua dòng user).

## 4. Exit criteria / Điều kiện trả nợ

- [ ] Add a lifecycle rule to the bucket that deletes objects under `profile.avatar_tmp_prefix` after 1 day (no code change) / Thêm lifecycle rule cho bucket xoá object dưới `profile.avatar_tmp_prefix` sau 1 ngày (không cần sửa code)
- [ ] Decide what happens to old avatars whose deletion failed: sweep job, or a retry / Chốt cách xử lý avatar cũ xoá lỗi: job quét hoặc thử lại
- [ ] Before go-live, review the bucket size and the number of objects under the temporary prefix / Trước go-live, xem dung lượng bucket và số object dưới prefix tạm
- [ ] When account deletion is built, make it remove the user's objects under both prefixes / Khi làm xoá tài khoản, xoá cả object của user ở hai prefix

## 5. Estimate / Ước lượng

**EN** — The lifecycle rule is about an hour of configuration. A sweep job for the failed-delete case is about a day.

**VI** — Lifecycle rule khoảng một giờ cấu hình. Job quét cho trường hợp xoá lỗi khoảng một ngày.
