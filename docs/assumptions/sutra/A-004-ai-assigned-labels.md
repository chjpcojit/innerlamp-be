---
id: A-004
feature: sutra
title: "dailyChant, purposes and deity links assigned by the AI from collection names / dailyChant, purposes và liên kết Phật/Bồ tát do AI gán từ tên nhóm kinh"
status: unvalidated
created: 2026-10-03
made-by: AI (Claude), while implementing the sutra feature
code: assets/data/sutras/*.json (dailyChant, purposes, deities); docs/specs/features/sutra/design.md (table "Đề xuất khởi tạo")
---

# A-004 — dailyChant, purposes and deity links assigned by the AI from collection names / dailyChant, purposes và liên kết Phật/Bồ tát do AI gán từ tên nhóm kinh

## 1. Assumption / Giả định

**EN** — With no religious reviewer available the user told the AI to assign the labels. They were derived from collection and sutra names by the table in the design, with deviations by title (e.g. praise/opening verses get `[]` instead of `CAU_AN`; Vãng Sanh gets `CAU_SIEU`; `chu-lang-nghiem` is not daily; Khandha/Dhajagga paritta are not daily). Deity roles (SPEAKER/SUBJECT/MENTIONED) were set from titles and `canonical_source`.

**VI** — Không có người duyệt về tôn giáo nên người dùng giao AI gán nhãn. Nhãn suy ra từ tên nhóm và tên kinh theo bảng trong design, có lệch theo tên bài (vd. bài tán/kệ khai kinh được `[]` thay vì `CAU_AN`; Vãng Sanh được `CAU_SIEU`; `chu-lang-nghiem` không phải nhật tụng; paritta Khandha/Dhajagga không phải nhật tụng). Vai trò Phật/Bồ tát (SPEAKER/SUBJECT/MENTIONED) đặt từ tên bài và `canonical_source`.

## 2. Rationale / Lý do

**EN** — Names are the only evidence available; the labels make the library filterable now and every text keeps `reviewStatus: UNREVIEWED`.

**VI** — Tên bài là bằng chứng duy nhất có sẵn; nhãn giúp lọc được thư viện ngay và mọi văn bản vẫn giữ `reviewStatus: UNREVIEWED`.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — A text could be offered for the wrong occasion (cầu siêu vs cầu an), or a deity linked wrongly; users may take the label as authoritative.

**VI** — Một văn bản có thể được gợi ý cho sai dịp (cầu siêu so với cầu an), hoặc gắn sai Phật/Bồ tát; người dùng có thể coi nhãn là chuẩn.

## 4. How to validate / Cách xác nhận

**EN** — A knowledgeable reviewer checks the labels. Related debt: TD-002.

**VI** — Người am hiểu duyệt nhãn. Nợ liên quan: TD-002.
