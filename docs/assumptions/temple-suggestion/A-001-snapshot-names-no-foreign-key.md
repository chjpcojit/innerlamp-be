---
id: A-001
feature: temple-suggestion
title: "Store unit codes with a snapshot of their names, no foreign key / Lưu mã kèm tên chụp, không đặt khoá ngoại"
status: unvalidated
created: 2026-10-11
made-by: AI (Claude), while specifying temple-suggestion v3.0
code: docs/specs/features/temple-suggestion/design.md (Schema, Step S-3..S-4)
---

# A-001 — Store unit codes with a snapshot of their names, no foreign key / Lưu mã kèm tên chụp, không đặt khoá ngoại

## 1. Assumption / Giả định

**EN** — `temple_suggestions` stores `province_code`, `ward_code` and also the unit names (`province`, `ward`) as they were at submission time. No foreign key points at `addr_*`; validity is checked in the use case when the suggestion is submitted.

**VI** — `temple_suggestions` lưu `province_code`, `ward_code` và cả tên đơn vị (`province`, `ward`) tại thời điểm gửi. Không đặt khoá ngoại sang `addr_*`; tính hợp lệ được kiểm ở usecase lúc gửi.

## 2. Rationale / Lý do

**EN** — The dataset is reloaded by upsert and codes only mean something together with their scheme. A snapshot keeps the back-office view readable if the dataset changes, and `temples.province` already uses the same short names.

**VI** — Dataset được nạp lại theo kiểu upsert và mã chỉ có nghĩa cùng hệ. Tên chụp giúp backoffice vẫn đọc được khi dataset đổi, và `temples.province` đã dùng cùng tên ngắn.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — If the owner wants referential integrity or live names, a follow-up migration adds the FK and the read path joins `addr_*`; stored snapshots would then be redundant but harmless.

**VI** — Nếu chủ sản phẩm muốn toàn vẹn tham chiếu hoặc tên luôn mới, một migration sau thêm FK và đường đọc join `addr_*`; tên chụp lúc đó thừa nhưng vô hại.

## 4. How to validate / Cách xác nhận

**EN** — Owner confirms or rejects; then set `status` and note it in `docs/specs/features/temple-suggestion/changelog.md`.

**VI** — Người dùng xác nhận hoặc bác bỏ; sau đó đổi `status` và ghi vào `docs/specs/features/temple-suggestion/changelog.md`.
