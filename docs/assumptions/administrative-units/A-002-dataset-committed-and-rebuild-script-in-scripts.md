---
id: A-002
feature: administrative-units
title: "The dataset file is committed, and its rebuild script lives in scripts/ / File dữ liệu được commit, script dựng lại nằm ở scripts/"
status: unvalidated
created: 2026-10-11
made-by: AI (Claude), while implementing administrative-units
code: .gitignore (!assets/data/administrative-units.json), scripts/build_administrative_units.py
---

# A-002 — The dataset file is committed, and its rebuild script lives in scripts/ / File dữ liệu được commit, script dựng lại nằm ở scripts/

## 1. Assumption / Giả định

**EN** — `assets/data/administrative-units.json` is un-ignored (`.gitignore` blocks `*.json`) so it is committed with the repo, and the script that builds it is `scripts/build_administrative_units.py`, next to the repo's other data scripts.

**VI** — `assets/data/administrative-units.json` được bỏ ignore (`.gitignore` chặn `*.json`) để commit cùng repo, và script dựng file là `scripts/build_administrative_units.py`, cạnh các script dữ liệu khác của repo.

## 2. Rationale / Lý do

**EN** — The owner asked for one JSON file so the database can be re-seeded easily later; an ignored file cannot be shared or restored. It is public reference data, not a secret (AGENTS.md §5.3). The script reproduces the file byte for byte, so the corrections applied to the source are repeatable.

**VI** — Người dùng cần một file JSON để seed lại DB dễ dàng về sau; file bị ignore thì không chia sẻ hay khôi phục được. Đây là dữ liệu tham chiếu công khai, không phải bí mật (AGENTS.md §5.3). Script dựng lại ra file giống từng byte nên các sửa so với nguồn tái lập được.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — If the owner prefers the data out of git, remove the three `.gitignore` lines; the seed command and tests keep working (the real-file tests skip when the file is absent).

**VI** — Nếu muốn dữ liệu ngoài git thì xoá ba dòng `.gitignore`; lệnh seed và test vẫn chạy (test dùng file thật sẽ skip khi file vắng).

## 4. How to validate / Cách xác nhận

**EN** — Owner confirms or rejects; then set `status` and note it in `docs/specs/features/administrative-units/changelog.md`.

**VI** — Người dùng xác nhận hoặc bác bỏ; sau đó đổi `status` và ghi vào `docs/specs/features/administrative-units/changelog.md`.
