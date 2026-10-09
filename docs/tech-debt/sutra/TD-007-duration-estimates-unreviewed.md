---
id: TD-007
feature: sutra
title: "durationMinutes is an unreviewed estimate with a known skew / durationMinutes là ước tính chưa duyệt và có độ lệch đã biết"
severity: low
status: open
created: 2026-10-03
owner: ducnguyen-wee
code: assets/data/sutras/*.json (durationMinutes)
---

# TD-007 — durationMinutes is an unreviewed estimate with a known skew / durationMinutes là ước tính chưa duyệt và có độ lệch đã biết

## 1. Debt / Nợ gì

**EN** — Values come from `ceil(syllables / 80)` of the preferred Vietnamese text (chant, then translation, then meaning, then poem), rounded for readability. Dược Sư is 80 minutes because only its translation (longer than the Hán-Việt chant) exists; Pháp Hoa and Lăng Nghiêm are 14 hours and Kim Quang Minh 15. The Phase 0 task to review and hand-edit them is still open (task list in the sutra spec).

**VI** — Giá trị lấy từ `ceil(số âm tiết / 80)` của bản tiếng Việt ưu tiên (tụng, rồi dịch, rồi dịch nghĩa, rồi thơ), làm tròn cho dễ đọc. Dược Sư ra 80 phút vì chỉ có bản dịch (dài hơn bản tụng Hán-Việt); Pháp Hoa và Lăng Nghiêm 14 giờ, Kim Quang Minh 15 giờ. Việc soát và chỉnh tay (Phase 0 trong danh sách task của spec sutra) vẫn còn mở.

## 2. Why accepted / Vì sao chấp nhận

**EN** — The user decided duration is an editorial estimate ("if we think a text takes 30 minutes, write 30"); the formula is only a first suggestion (A-005).

**VI** — Người dùng chốt thời lượng là ước tính biên tập ("nghĩ bài cần 30 phút thì ghi 30"); công thức chỉ là gợi ý ban đầu (A-005).

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — The client shows a wrong duration (45–60 minutes is more realistic for Dược Sư), and has no good answer for 14-hour texts.

**VI** — Client hiển thị thời lượng sai (45–60 phút thực tế hơn cho Dược Sư) và chưa có cách hiển thị tốt cho văn bản dài 14 giờ.

## 4. Exit criteria / Điều kiện trả nợ

- [ ] Review the texts under 15 minutes and over 60 minutes and edit the JSON (and `index.json`) / Soát các bài dưới 15 phút và trên 60 phút rồi sửa JSON (và `index.json`)
- [ ] Decide how the client presents very long texts (by phẩm/session rather than one number) / Quyết định client hiển thị văn bản rất dài thế nào (theo phẩm/buổi tụng thay vì một con số)

## 5. Estimate / Ước lượng

**EN** — 1 hour of editing; UI work separate.

**VI** — 1 giờ chỉnh sửa; phần UI tính riêng.
