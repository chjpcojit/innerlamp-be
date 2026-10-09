# Tech Debt / Nợ kỹ thuật

**EN** — One file per debt item: `docs/tech-debt/<feature>/TD-NNN-<slug>.md`. `NNN` increases per feature. Rule source: `AGENTS.md` §5.2.

**VI** — Mỗi khoản nợ một file: `docs/tech-debt/<feature>/TD-NNN-<slug>.md`. `NNN` tăng dần theo từng feature. Nguồn quy tắc: `AGENTS.md` §5.2.

## Frontmatter

```yaml
---
id: TD-001
feature: register
title: "<EN title> / <VI title>"
severity: high        # high | medium | low
status: open          # open | in-progress | resolved | wontfix
created: 2026-10-02
owner: <github-user>
code: <path/to/file.go (symbol)>
---
```

## Body sections (EN then VI under each heading / EN rồi VI dưới mỗi mục)

1. Debt / Nợ gì
2. Why accepted / Vì sao chấp nhận
3. Risk if unpaid / Rủi ro nếu không trả
4. Exit criteria / Điều kiện trả nợ (checklist)
5. Estimate / Ước lượng

## Lifecycle / Vòng đời

- **EN** — When paid, set `status: resolved`; never delete the file. Reference the ID in code: `// TD-001`.
- **VI** — Khi đã trả, đổi `status: resolved`, không xoá file. Ghi ID trong code: `// TD-001`.

Find open debt / Tìm nợ còn mở: `grep -rl "status: open" docs/tech-debt`
