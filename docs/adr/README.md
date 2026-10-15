# docs/adr — Architecture Decision Records

**EN** — One file per architecture decision that is hard to reverse (infrastructure, a new or
reshaped data model, offline sync, auth/security, cross-cutting conventions, technology choices).
Shared by every InnerLamp platform: this backend, the website, iOS and Android. Written by the
`adr-writer` skill (`.claude/skills/adr-writer/`). Most features need no ADR; the skill's gate
(`references/when_to_write_adr.md`) decides.

**VI** — Mỗi quyết định kiến trúc khó đảo ngược là một file (hạ tầng, data model mới, offline sync,
auth/bảo mật, quy ước dùng chung, chọn công nghệ). Dùng chung cho cả 4 nền tảng. Skill `adr-writer`
viết; phần lớn feature không cần ADR.

## Where it sits / Vị trí trong luồng

```
idea → (optional) adr-writer → ba-writer → il-spec-be / -web / -ios / -android → implementation → CI
       docs/adr/NNNN-*.md       docs/product/<feature>/   platform specs
```

Product specs (`docs/product/`) and platform specs cite an ADR by ID (`ADR-0001`) and never copy
its content. `ba-writer` stops and asks when a feature needs an architecture decision that has no
Accepted ADR yet.

## Conventions / Quy ước

- File: `docs/adr/NNNN-<kebab-slug>.md`, `NNNN` increasing, never reused. Layout:
  `.claude/skills/adr-writer/assets/templates/adr.md.template`. Body in English.
- Status: `Proposed` → `Accepted` | `Rejected`; later `Superseded by ADR-NNNN` or `Deprecated`.
  Only the user accepts or rejects; the AI always writes `Proposed`.
- An Accepted ADR is never edited (typos/links aside). A changed decision is a new ADR that
  supersedes it; the old one only gets its status line updated.

## Index

| ID | Title | Status | Date | Features |
|---|---|---|---|---|
| [ADR-0001](0001-in-app-notifications-and-push.md) | In-app notifications on every platform, push via FCM on iOS and Android | Accepted | 2026-10-14 | cross-cutting (`memorial` first) |
| [ADR-0002](0002-server-side-memorial-reminders.md) | The backend computes and sends memorial reminders on an hourly schedule | Accepted | 2026-10-14 | `memorial` |
| [ADR-0003](0003-email-otp-account-verification-without-sms.md) | Accounts are identified by email and verified by an email OTP; SMS is removed | Accepted | 2026-10-14 | `onboarding`, `login`, `genealogy`, `profile` |
| [ADR-0004](0004-shared-links-are-website-urls-opened-by-the-apps.md) | Shared links are website URLs that the apps open as verified Universal Links / App Links | Proposed | 2026-10-15 | cross-cutting (`article` first) |
