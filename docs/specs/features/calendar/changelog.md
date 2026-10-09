# Changelog — calendar

## v1.1 (2026-10-07) — implemented

Code xong và kiểm tra trên server thật: domain, parser manifest, config, usecase, handler, wiring, openapi; mọi task `[x]`. Chỉ đổi `Status`/checkbox/`Pending` và ví dụ trong `api-contract.md` (lấy từ server thật); AC không đổi. Phát hiện khi code: `time.LoadLocation("")` âm thầm trả UTC nên usecase từ chối `calendar.timezone` rỗng (khớp AC10, không đổi spec).

Cross-file follow-ups:
- `docs/specs/api/openapi.yaml`: đã thêm `/v1/calendar/image` (done).
- `docs/knowledge/features/calendar.md`: chưa viết, gọi `il-okf-writer` khi được yêu cầu.

## v1.0 (2026-10-07) — initial

Spec đầu tiên cho `GET /v1/calendar/image` (ảnh Phật theo ngày cho màn Lịch âm), theo yêu cầu của session iOS. Prospective: mọi task `[ ]`.

Cross-file follow-ups:
- `docs/specs/api/openapi.yaml`: thêm route + schema (Phase 5).
- `docs/knowledge/features/calendar.md`: viết bằng `il-okf-writer` sau khi chạy end-to-end (chưa làm).
- Không có migration / ERD.
