# Assumptions / Giả định

**EN** — One file per assumption the AI makes while coding: `docs/assumptions/<feature>/A-NNN-<slug>.md`. `NNN` increases per feature. Rule source: `AGENTS.md` §5.2.

**VI** — Mỗi giả định AI đưa ra khi code một file: `docs/assumptions/<feature>/A-NNN-<slug>.md`. `NNN` tăng dần theo từng feature. Nguồn quy tắc: `AGENTS.md` §5.2.

## Frontmatter

```yaml
---
id: A-001
feature: login
title: "<EN title> / <VI title>"
status: unvalidated   # unvalidated | confirmed | rejected
created: 2026-10-02
made-by: AI (Claude), while implementing US-002
code: <path/to/file.go>
---
```

## Body sections (EN then VI under each heading / EN rồi VI dưới mỗi mục)

1. Assumption / Giả định
2. Rationale / Lý do
3. Impact if wrong / Hệ quả nếu sai
4. How to validate / Cách xác nhận

## Lifecycle / Vòng đời

- **EN** — Once confirmed or rejected, update `status` and the related spec under `docs/specs/features/` (resolve the `⚠️ Not confirmed` marker). Never delete the file. Reference the ID in code: `// A-001`.
- **VI** — Khi đã xác nhận hoặc bác bỏ, cập nhật `status` và spec liên quan trong `docs/specs/features/` (gỡ marker `⚠️ Not confirmed`). Không xoá file. Ghi ID trong code: `// A-001`.

## Assumption vs tech debt / Giả định và nợ kỹ thuật

- **EN** — An assumption is something guessed because the spec is silent. Debt is a known shortcut. If an assumption leads to a shortcut, write both and cross-link them.
- **VI** — Giả định là điều đoán vì spec không nói rõ. Nợ là đường tắt đã biết. Nếu giả định dẫn tới đường tắt thì viết cả hai và link chéo.

Find unvalidated / Tìm giả định chưa xác nhận: `grep -rl "status: unvalidated" docs/assumptions`
