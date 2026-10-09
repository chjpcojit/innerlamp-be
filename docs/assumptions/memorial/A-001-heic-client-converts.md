---
id: A-001
feature: memorial
title: "App converts HEIC/HEIF photos to JPEG before upload / App đổi ảnh HEIC sang JPEG trước khi upload"
status: confirmed
created: 2026-10-07
made-by: AI (Claude), while resolving a spec question
code: (chưa có code — áp dụng cho Phase 6 của docs/specs/features/memorial/)
---

# A-001 — App converts HEIC/HEIF to JPEG before upload / App đổi HEIC sang JPEG trước khi upload

## 1. Assumption / Giả định

**EN** — Asked how to handle iPhone HEIC photos, the user answered "photos come from iPhone and Android". The spec reads that as: both platforms must work, and the **app** converts HEIC/HEIF to JPEG before upload. The backend keeps accepting only `image/jpeg`, `image/png`, `image/webp` (detected by content sniffing) and does not decode HEIC.

**VI** — Khi hỏi cách xử lý ảnh HEIC của iPhone, người dùng trả lời "ảnh từ iPhone và Android". Spec hiểu là: cả hai nền tảng đều phải dùng được, và **app** đổi HEIC/HEIF sang JPEG trước khi upload. Backend chỉ nhận `image/jpeg`, `image/png`, `image/webp` (nhận diện bằng nội dung) và không giải mã HEIC.

## 2. Rationale / Lý do

Decoding HEIC in Go needs a cgo/native library — a heavy dependency against the repo's "standard library first" rule. iOS converts to JPEG in one call. Not yet confirmed by the user that this is what they meant.

## 3. Implication / Hệ quả

If the user actually wants the backend to accept HEIC, `requirements.md` AC27, `design.md` Step P-2 and Phase 6 of `tasks.md` change, and a decoding dependency must be justified.

## 4. Validation / Cách kiểm chứng

**EN** — Confirmed by the user on 2026-10-07: the client converts HEIC/HEIF to JPEG before upload; the backend does not decode HEIC. What remains is client work, to be done with the photo feature (TD-001, memorial).

**VI** — Anh Đức xác nhận ngày 2026-10-07: client tự đổi HEIC/HEIF sang JPEG trước khi upload, backend không giải mã HEIC. Việc còn lại là phía client, làm cùng tính năng ảnh (TD-001, memorial).

Original plan / Kế hoạch ban đầu:

Ask the user (or the iOS client owner) when starting Phase 6 (photo), which is blocked on GCS anyway.
