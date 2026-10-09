---
id: TD-006
feature: sutra
title: "Stale references to the removed content spec / Tham chiếu cũ tới spec content đã xoá"
severity: low
status: resolved
created: 2026-10-03
owner: ducnguyen-wee
code: docs/specs/features/practice/requirements.md; AGENTS.md:95; README.md:159,350; CLAUDE.md:9; docs/specs/features/{article,deity,temple}/requirements.md:15
---

# TD-006 — Stale references to the removed content spec / Tham chiếu cũ tới spec content đã xoá

## 1. Debt / Nợ gì

**EN** — After `docs/specs/features/content/requirements.md` was deleted: `practice/requirements.md` (lines 3, 10, 20, 26, 33, 38) still cites its US-005…US-010 and the `sutras`/`mantras` tables; `AGENTS.md` line 95 lists sutras/mantras/bodhisattvas as US-003–US-010 "without code"; `README.md` lines 159 and 350 plan a `content` module (CRUD + delta sync); `CLAUDE.md` line 9 mentions "the content user stories"; the `⚠️ Not confirmed` markers in the article, deity and temple specs say "`content/requirements.md` has no US for …".

**VI** — Sau khi `docs/specs/features/content/requirements.md` bị xoá: `practice/requirements.md` (dòng 3, 10, 20, 26, 33, 38) vẫn trích US-005…US-010 và các bảng `sutras`/`mantras`; `AGENTS.md` dòng 95 liệt kê sutras/mantras/bodhisattvas là US-003–US-010 "chưa có code"; `README.md` dòng 159 và 350 vẫn lên kế hoạch module `content` (CRUD + delta sync); `CLAUDE.md` dòng 9 nhắc "the content user stories"; các marker `⚠️ Not confirmed` trong spec article, deity, temple ghi "`content/requirements.md` chưa có US cho …".

## 2. Why accepted / Vì sao chấp nhận

**EN** — Out of scope of the request that removed the spec; changing other features' specs needs a version bump and a changelog entry each.

**VI** — Nằm ngoài phạm vi yêu cầu xoá spec; sửa spec của feature khác cần bump version và ghi changelog từng cái.

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — A reader or an agent follows a pointer to a file that no longer exists, or believes sutras are still unimplemented.

**VI** — Người đọc hoặc agent đi theo con trỏ tới file không còn tồn tại, hoặc tin rằng kinh vẫn chưa được làm.

## 4. Exit criteria / Điều kiện trả nợ

- [ ] Update the references above (specs via `il-spec-writer` so versions and changelogs move together) / Cập nhật các tham chiếu trên (spec qua `il-spec-writer` để version và changelog đi cùng nhau)
- [ ] `grep -rn "features/content" docs README.md AGENTS.md CLAUDE.md` returns nothing / `grep -rn "features/content" docs README.md AGENTS.md CLAUDE.md` không còn kết quả

## 5. Estimate / Ước lượng

**EN** — 1 hour.

**VI** — 1 giờ.

## 6. Decision / Quyết định (2026-10-03)

**EN** — Removed: the `practice` spec (deleted entirely, outdated), `AGENTS.md` line 95, `README.md` lines 159 and 350, and the markers in the article, deity and temple specs (each bumped a minor version). `CLAUDE.md` no longer names the content stories as pending.

**VI** — Đã xoá: spec `practice` (xoá hẳn vì cũ), `AGENTS.md` dòng 95, `README.md` dòng 159 và 350, và các marker trong spec article, deity, temple (mỗi spec tăng một minor). `CLAUDE.md` không còn nêu story content là việc chờ.
