---
id: TD-005
feature: sutra
title: "Audio, delta sync and offline support were dropped with the old content spec / Âm thanh, đồng bộ delta và offline bị bỏ cùng spec content cũ"
severity: medium
status: open
created: 2026-10-03
owner: ducnguyen-wee
code: services/appService/delivery/http/sutra_handler.go
---

# TD-005 — Audio, delta sync and offline support were dropped with the old content spec / Âm thanh, đồng bộ delta và offline bị bỏ cùng spec content cũ

## 1. Debt / Nợ gì

**EN** — The removed `content` spec described `audio_url`, `version`, `updated_since` delta sync and offline-first caching. The sutra API has none of them: no `audio_url`, no ETag or `Cache-Control`, no conditional GET. The `practice` spec that relied on `audio_url` was deleted on 2026-10-03 as outdated.

**VI** — Spec `content` đã xoá có mô tả `audio_url`, `version`, đồng bộ delta `updated_since` và cache offline-first. API sutra không có cái nào: không `audio_url`, không ETag hay `Cache-Control`, không conditional GET. Spec `practice` từng dựa vào `audio_url` đã bị xoá ngày 2026-10-03 vì cũ.

## 2. Why accepted / Vì sao chấp nhận

**EN** — The user replaced the old spec with the sutra spec, where these are listed under Out of scope.

**VI** — Người dùng thay spec cũ bằng spec sutra, trong đó các mục này nằm ở Out of scope.

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — A future chanting player has no audio to play; clients re-download long texts (a page of 200 lines is ~21 KB, a whole Pháp Hoa ~740 KB) on every open; older users reading offline get nothing from the server's side.

**VI** — Trình phát tụng kinh sau này chưa có audio để phát; client tải lại các văn bản dài (một trang 200 dòng ~21 KB, cả Pháp Hoa ~740 KB) mỗi lần mở; người lớn tuổi đọc offline không được server hỗ trợ.

## 4. Exit criteria / Điều kiện trả nợ

- [ ] Decide where audio comes from and add `audio_url` per translation (migration + API + data) / Quyết định nguồn âm thanh và thêm `audio_url` theo từng bản dịch (migration + API + dữ liệu)
- [ ] Add `ETag`/`Last-Modified` with conditional GET, or `updated_since` / Thêm `ETag`/`Last-Modified` kèm conditional GET, hoặc `updated_since`

## 5. Estimate / Ước lượng

**EN** — 2–3 days (audio sourcing is the unknown).

**VI** — 2–3 ngày (nguồn âm thanh là ẩn số).

## 6. Decision / Quyết định (2026-10-03)

**EN** — Keep `audio_url` as a field to add to the sutra data later (the `practice` spec was deleted as outdated), so audio can be updated then. No column or API change now.

**VI** — Giữ `audio_url` là trường sẽ thêm vào dữ liệu kinh sau (spec `practice` đã xoá vì cũ) để cập nhật audio. Chưa đổi cột hay API.
