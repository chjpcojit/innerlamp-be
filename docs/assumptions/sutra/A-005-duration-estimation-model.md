---
id: A-005
feature: sutra
title: "Duration is syllables ÷ 80 per minute, rounded / Thời lượng = số âm tiết ÷ 80 mỗi phút, làm tròn"
status: unvalidated
created: 2026-10-03
made-by: AI (Claude), while implementing the sutra feature
code: assets/data/sutras/*.json (durationMinutes); docs/specs/features/sutra/design.md
---

# A-005 — Duration is syllables ÷ 80 per minute, rounded / Thời lượng = số âm tiết ÷ 80 mỗi phút, làm tròn

## 1. Assumption / Giả định

**EN** — The first suggestion for `durationMinutes` assumes chanting runs at about 80 Vietnamese syllables per minute, counted on the preferred Vietnamese text (chant, then translation, meaning, poem), then rounded (≤10 as is; 11–60 up to a multiple of 5; 61–180 up to 10; above that 30). The user accepted it as an estimate.

**VI** — Gợi ý đầu tiên cho `durationMinutes` giả định tụng ở tốc độ khoảng 80 âm tiết tiếng Việt mỗi phút, đếm trên bản tiếng Việt ưu tiên (tụng, dịch, dịch nghĩa, thơ), rồi làm tròn (≤10 giữ nguyên; 11–60 lên bội của 5; 61–180 lên bội của 10; lớn hơn lên bội của 30). Người dùng chấp nhận như một ước tính.

## 2. Rationale / Lý do

**EN** — A sample of known durations gave Tâm Kinh ≈3 minutes and A Di Đà ≈24, which are plausible; no measured chanting data exists in the repo.

**VI** — Mẫu thời lượng đã biết cho Tâm Kinh ≈3 phút và A Di Đà ≈24, hợp lý; trong repo không có dữ liệu tụng đo thực tế.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — Texts whose only Vietnamese version is a translation (not a chant) come out too long — Dược Sư shows 80 minutes against a realistic 45–60.

**VI** — Văn bản chỉ có bản dịch (không có bản tụng) ra quá dài — Dược Sư hiện 80 phút so với thực tế 45–60.

## 4. How to validate / Cách xác nhận

**EN** — Review the short and long lists and edit by hand. Related debt: TD-007.

**VI** — Soát danh sách bài ngắn và dài rồi sửa tay. Nợ liên quan: TD-007.
