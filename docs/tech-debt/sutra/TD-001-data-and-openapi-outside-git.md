---
id: TD-001
feature: sutra
title: "Sutra dataset and openapi.yaml are not under version control / Dữ liệu kinh và openapi.yaml chưa nằm trong git"
severity: high
status: open
created: 2026-10-03
owner: ducnguyen-wee
code: assets/ (assets/data/sutras/, assets/embed.go); .gitignore:64 (*.yaml) hides docs/specs/api/openapi.yaml and web/docs/public/openapi.yaml
---

# TD-001 — Sutra dataset and openapi.yaml are not under version control / Dữ liệu kinh và openapi.yaml chưa nằm trong git

## 1. Debt / Nợ gì

**EN** — The 11 MB curated library (153 texts, collected from the web) and `assets/embed.go`, which the binary needs in order to compile, exist only on this disk: `git status` shows `?? assets/`. Both `openapi.yaml` copies are ignored by `.gitignore` line 64 (`*.yaml`), so API contract changes (the sutra routes, and the earlier article ones) never show up in a diff. The scripts that fetched and segmented the sources were scratch files and no longer exist; each translation's `sourceUrl` is the only provenance left.

**VI** — Thư viện kinh 11 MB (153 văn bản, thu thập từ web) và `assets/embed.go` (binary cần file này để biên dịch) chỉ tồn tại trên đĩa này: `git status` hiện `?? assets/`. Cả hai bản `openapi.yaml` bị `.gitignore` dòng 64 (`*.yaml`) bỏ qua nên thay đổi hợp đồng API (các route sutra, và cả article trước đó) không hiện trong diff. Các script tải và tách câu nguồn chỉ là file tạm và không còn; `sourceUrl` của từng bản dịch là dấu vết nguồn gốc duy nhất còn lại.

## 2. Why accepted / Vì sao chấp nhận

**EN** — Not a decision: the working tree was never committed (the AI commits only when asked), and the `*.yaml` ignore rule predates this work.

**VI** — Không phải quyết định có chủ ý: working tree chưa từng được commit (AI chỉ commit khi được yêu cầu) và quy tắc ignore `*.yaml` có từ trước.

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — A lost disk or a fresh clone cannot build (`go:embed data/sutras/*.json` matches nothing, a compile error) and the dataset is gone with no history; the web docs and clients cannot diff the API contract.

**VI** — Mất đĩa hoặc clone mới sẽ không build được (`go:embed data/sutras/*.json` không khớp file nào, lỗi biên dịch) và dữ liệu mất hẳn, không có lịch sử; web docs và client không diff được hợp đồng API.

## 4. Exit criteria / Điều kiện trả nợ

- [ ] Commit `assets/` (plain JSON, ~11 MB; Git LFS is not needed) / Commit `assets/` (JSON thuần, ~11 MB; không cần Git LFS)
- [~] Un-ignore the two `openapi.yaml` files (e.g. `!docs/specs/api/openapi.yaml` after the `*.yaml` rule) and commit them / Bỏ ignore hai file `openapi.yaml` (vd. `!docs/specs/api/openapi.yaml` sau quy tắc `*.yaml`) rồi commit — un-ignore DONE 2026-10-07 (`.gitignore` lines 66–67, both files now show as untracked); still to do: commit / đã bỏ ignore xong, còn commit
- [ ] Prove a clean clone builds: `git clone … && go build ./...` / Chứng minh clone sạch build được: `git clone … && go build ./...`

## 5. Estimate / Ước lượng

**EN** — 0.5 hour (the decision to commit is the user's).

**VI** — 0,5 giờ (quyết định commit thuộc về người dùng).

## 6. Decision / Quyết định (2026-10-03)

**EN** — The user will add `assets/` and the OpenAPI files to git themselves. Stays open until that commit exists and a clean clone builds.

**VI** — Anh sẽ tự add `assets/` và các file OpenAPI vào git. Giữ mở cho tới khi có commit và clone sạch build được.
