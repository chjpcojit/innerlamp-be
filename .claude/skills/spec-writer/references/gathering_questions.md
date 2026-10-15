# Asking the user before writing a spec

Before generating or updating any spec file, ask the mandatory questions below with the user-question
tool (or equivalent) — never guess. This file defines the mandatory question set and the phrasing
rules that make the questions easy to answer.

**Language**: per the Language policy in `CLAUDE.md`, the questions are chat with the user, so they
are asked in Vietnamese ("anh" for the user, "em" for the agent), technical terms kept in English.
The quoted example questions below are written that way on purpose. What ends up in the spec —
including every `⚠️ Not confirmed` marker — is English.

## Phrasing rules

- **One topic per question.** Never merge two decisions into one question.
- **Concrete, not abstract.** Instead of "Thiết kế layer thế nào?", ask "Feature này có cần bảng DB
  mới không, hay chỉ đọc/ghi trên bảng đã có?"
- **Offer an example or a default** where possible, so the user only has to confirm instead of
  composing an answer:
  - Weak: "Tiêu chí chấp nhận nên là gì?"
  - Good: "Khi user gửi request với body sai định dạng — trả 400 kèm danh sách field lỗi (mặc định),
    hay trả 422 với error code riêng?"
- **Short.** A question answerable in one line beats a paragraph the user has to read.

## Read the product spec before asking

Read `docs/product/<feature>/product.md` first (see `upstream_product_spec.md`). Anything the product
spec already answers (user story, business rules, scope, edge cases) is **not asked again** — cite the
ID. Anything that is a product decision the product spec is missing is not settled here: record
`⚠️ Not confirmed (product)` and hand it back to `ba-writer`. The question set below covers only the
backend's technical part.

## Mandatory question set (before generating a new spec)

Ask in order, one at a time or combined into a short form — depending on the question tool's format.
Each item gives what to find out, then a suggested Vietnamese phrasing:

1. **Feature name** (kebab-case, matching the folder name) — e.g. `register`, `sutra-list`,
   `password-reset`. Ask: "Tên feature là gì? (kebab-case, ví dụ `register`, `password-reset`)"
2. **Is the feature already implemented in code?** → decides Prospective vs Retroactive
   (`modes.md`). Ask: "Feature này đã được implement trong code chưa?"
3. **User story in one sentence: who is the user and what do they want?** — or point to an existing
   US-XXX in the current `requirements.md` when continuing one. Ask: "Một câu: ai là người dùng và
   họ muốn làm gì?"
4. **Which endpoints/routes?** (`METHOD /v1/...`) — and does each need auth (`RequireAuth`
   middleware, or public read)? Ask: "Endpoint nào (`METHOD /v1/...`)? Có cần auth (`RequireAuth`)
   hay public read?"
5. **New DB table, or only existing tables?** If new: does the migration exist yet, or must one be
   added? Ask: "Có bảng DB mới không, hay chỉ dùng bảng đã có? Nếu có thì đã có migration chưa?"
6. **Does the main flow make more than one sequential external call** (several queries, a
   multi-table transaction, an external service call, a storage upload, sending mail)? Yes/No — if
   yes, a Step-by-step Flow section in `design.md` (see `step_by_step_flow.md`) **and** a Mermaid
   diagram in `flow.md` (see `flow_diagram.md`) are mandatory. Ask: "Flow chính có nhiều hơn một lời
   gọi ra ngoài theo thứ tự không (nhiều query, transaction nhiều bảng, external service, upload,
   gửi mail)?"
7. **For each call in question 6: is the contract settled** (request/response schema, error codes,
   HTTP status), or still unclear? Anything "unclear" becomes a `⚠️ Not confirmed` marker and must
   not be written as if it existed. Ask: "Với từng lời gọi đó, contract (schema, mã lỗi, HTTP status)
   đã chốt chưa?"
8. **Is there any implicit check/validation not obvious from the main action** — e.g. a duplicate
   email check, a rate limit, a resource-ownership check, a DB unique constraint, a secondary side
   effect (audit log, counter)? (This is the question that surfaces things like "email must be
   unique" before they get silently folded into a one-line user story.) Ask: "Có bước kiểm tra ngầm
   nào không — ví dụ check trùng email, rate limit, quyền sở hữu resource, unique constraint, audit
   log?"
9. **Any dependency on another module/service** — does the `journal` module's usecase call into
   `content`, or does the handler call a third-party external API directly? Ask: "Có phụ thuộc
   module/service khác không, hay gọi thẳng API bên thứ ba?"

## When updating an existing spec (not creating one)

Ask these instead:

1. **What changed?** — one sentence: a new requirement, a scope change (moving something from "Out
   of scope" into scope), or an answer to a `⚠️ Not confirmed` item? Ask: "Thay đổi gì ạ —
   requirement mới, đổi scope, hay trả lời một mục `⚠️ Not confirmed`?"
2. If it answers a `⚠️ Not confirmed` item: **what is the answer**, as simply as possible (e.g. "trả
   trong cùng response" vs "gọi API riêng" vs "tính ở client").
3. If it is new/changed behavior: **does it affect existing acceptance criteria, or is it entirely
   new?** Ask: "Có ảnh hưởng tới tiêu chí chấp nhận đã có không, hay hoàn toàn mới?"
4. If it is a new technical detail (schema field, error code, index): **does it live only in
   `design.md`, or does it change the behavior `requirements.md` describes?** Ask: "Chi tiết này chỉ
   nằm ở `design.md`, hay làm đổi behavior trong `requirements.md`?"

## The "never skip" rule

If the user does not answer a mandatory question (declines, or says "chưa chắc"), you must **not**:
- Drop it from the spec entirely.
- Pick a default and present it as if it were confirmed.
- Ask it again and again in the same session — ask once, then continue with the marker open.

Instead, record it in the spec in the exact format from rule 3 of `SKILL.md` (in English, since it is
committed):

```
> ⚠️ Not confirmed: <the question, one sentence> — <what it blocks, e.g. "blocks tasks.md Phase 3
> from being implemented">.
```

Place it in the most relevant section (`requirements.md` "Pending" for behavior/business questions,
`design.md` at the exact flow step for technical/API questions) — do not pile them all at the top of
the file, where they lose their context.
