---
id: A-001
feature: temple
title: "Temples store unit codes with name snapshots, no foreign key, all-or-none / Chùa lưu mã kèm tên chụp, không khoá ngoại, đủ bộ hoặc không có"
status: unvalidated
created: 2026-10-11
made-by: AI (Claude), while specifying temple v1.7
code: docs/specs/features/temple/design.md (Layers: Migration, Seed CLI)
---

# A-001 — Temples store unit codes with name snapshots, no foreign key, all-or-none / Chùa lưu mã kèm tên chụp, không khoá ngoại, đủ bộ hoặc không có

## 1. Assumption / Giả định

**EN** — `temples` gets `province_code`, `ward_code` and `ward` (full name snapshot) next to the existing `province` (short name). No foreign key to `addr_new_*`. A CHECK requires the three new columns to be all NULL or all NOT NULL. Rows not re-seeded yet keep NULL and the API returns `null` for them.

**VI** — `temples` thêm `province_code`, `ward_code` và `ward` (tên đầy đủ chụp) cạnh `province` (tên ngắn) đã có. Không FK sang `addr_new_*`. CHECK bắt ba cột mới cùng NULL hoặc cùng có giá trị. Dòng chưa nạp lại giữ NULL và API trả `null`.

## 2. Rationale / Lý do

**EN** — Same reasoning as `temple-suggestion` A-001: the dataset is reloaded by upsert and a code only means something with its scheme. A snapshot keeps `options` a single-table read (no join) and an official temple copied from a suggestion carries the exact names the admin saw.

**VI** — Cùng lý do với `temple-suggestion` A-001: dataset nạp lại theo kiểu upsert và mã chỉ có nghĩa cùng hệ. Tên chụp giữ `options` là một lần đọc một bảng (không join) và chùa chính thức tạo từ đề xuất mang đúng tên admin đã thấy.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — If the owner wants live names or referential integrity, a later migration adds the FK and the read path joins `addr_new_*`; the snapshot columns become redundant but harmless.

**VI** — Nếu chủ sản phẩm muốn tên luôn mới hoặc toàn vẹn tham chiếu, một migration sau thêm FK và đường đọc join `addr_new_*`; cột tên chụp lúc đó thừa nhưng vô hại.

## 4. How to validate / Cách xác nhận

**EN** — Owner confirms or rejects; then set `status` and note it in `docs/specs/features/temple/changelog.md`.

**VI** — Người dùng xác nhận hoặc bác bỏ; sau đó đổi `status` và ghi vào `docs/specs/features/temple/changelog.md`.
