---
id: A-002
feature: sutra
title: "Only the sutra and mantra stories of the old content spec were replaced / Chỉ các story kinh và thần chú của spec content cũ bị thay"
status: rejected
created: 2026-10-03
made-by: AI (Claude), while implementing the sutra feature
code: docs/specs/features/content/ (deleted)
---

# A-002 — Only the sutra and mantra stories of the old content spec were replaced / Chỉ các story kinh và thần chú của spec content cũ bị thay

## 1. Assumption / Giả định

**EN** — When the user said the old `content` spec was incomplete and could be dropped, it was assumed that only US-005…US-008 (sutras, mantras) were replaced by the sutra spec and that US-009/US-010 (Phật & Bồ Tát) stayed in the file.

**VI** — Khi người dùng nói spec `content` cũ không đầy đủ và có thể bỏ, đã giả định rằng chỉ US-005…US-008 (kinh, thần chú) được spec sutra thay và US-009/US-010 (Phật & Bồ Tát) vẫn ở lại trong file.

## 2. Rationale / Lý do

**EN** — The file was described as being about sutras, and the deity catalogue was a separate feature with its own spec.

**VI** — File được mô tả là về kinh, và danh mục Phật/Bồ tát là một feature riêng có spec riêng.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — A story might be removed that still mattered (US-009/010) — or a stale one kept.

**VI** — Có thể xoá một story vẫn còn cần (US-009/010) — hoặc giữ lại một story đã cũ.

## 4. How to validate / Cách xác nhận

**EN** — Rejected on 2026-10-03: the user deleted the whole `content` folder because sutras, mantras and Phật/Bồ tát each have their own feature. Recorded in the sutra changelog v2.4; the leftovers are TD-006.

**VI** — Bị bác bỏ ngày 2026-10-03: người dùng xoá cả thư mục `content` vì kinh, thần chú và Phật/Bồ tát đều đã có feature riêng. Đã ghi ở changelog sutra v2.4; phần còn sót là TD-006.
