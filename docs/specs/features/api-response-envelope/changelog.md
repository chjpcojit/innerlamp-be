# Changelog — api-response-envelope Spec

Tracks version history for `requirements.md`, `design.md`, and `tasks.md` in this folder. All three
files share one changelog since they change together as one spec unit. Newest version first.

## v1.2 (2026-09-22)

Cross-spec follow-up từ `http-middleware-wiring`: preflight `OPTIONS` của `pkg/middleware.CORS` không còn là ngoại lệ
không body. Không đổi AC1 (vẫn ràng buộc "response that has a body"); không đổi code của spec này.

- `requirements.md`: bỏ preflight khỏi §Ngoại lệ; thêm ghi chú ngoại lệ còn lại (redirect `307`) + trỏ sang
  `http-middleware-wiring` AC4/AC10.
- `design.md`: bảng Layers dòng `pkg/middleware/middleware.go` ghi chéo thay đổi preflight; Request flow cập nhật
  chain thật `Recovery → CORS → Logger → WithAPIFallback → mux`.
- `tasks.md`: thêm Phase 6 (cross-spec follow-up) `[x]`.
- Reason: `http-middleware-wiring` wire `CORS` vào appService và AC4/AC10 đổi preflight `200` rỗng → `200`
  `application/json` `{"success": true}`.

### Thay đổi hành vi quan sát được ở client appService

- Preflight `OPTIONS` (khi `CORS` được wire): `200` rỗng → `200` `application/json` `{"success": true}`. Mọi
  response khác không đổi.

## v1.1 (2026-09-21)

Chốt marker cuối: logout thành công đổi từ `204 No Content` sang `200` `{"success": true}`. **Đổi code + hành vi
quan sát được ở client.**

- `requirements.md`: thêm AC11 (logout `200` envelope không `data`); AC10 thêm "trừ AC11"; bỏ logout khỏi mục
  Ngoại lệ; marker chuyển sang Open questions (resolved). Sửa tham chiếu sai "login AC12" → AC13/AC14.
- `design.md`: bảng Layers thêm dòng `login_handler.go`.
- `tasks.md`: Phase 5 — code, test, openapi, login spec, `pkg/utils/README.md` `[x]`; chạy binary thật còn `[ ]`.
- Reason: anh chốt đổi sang `200 {"success": true}` để client chỉ cần một bộ parse, không nhánh riêng cho body rỗng.

### Thay đổi hành vi quan sát được ở client appService

- `POST /v1/auth/logout` thành công (kể cả gọi lại, AC14): `204` rỗng → `200` `application/json` `{"success": true}`.

## v1.0 (initial, 2026-09-21)

Spec retroactive: code sửa + test trước, spec viết sau để ghi lại. Chuẩn hoá mọi response có body về một
envelope `utils.ApiResponse`.

- `requirements.md`: bảng bằng chứng 8 nguồn response lệch envelope; AC1–AC10; ngoại lệ không body (logout
  `204`, preflight, redirect); một marker `⚠️ Not confirmed` về logout.
- `design.md`: bảng Layers (4 file `pkg/`, `fallback.go` mới, `main.go`, examples/docs); lý do chọn
  `mux.Handler` thay route bắt-hết; 404/405 là `type: business`.
- `tasks.md`: Phase 1–4 đều `[x]` (đã verify — xem ghi chú từng mục).
- Reason: anh yêu cầu "chuẩn hoá lại 1 định dạng response cho client, từ tài liệu tới code, sửa hết" sau khi
  sửa docs envelope của `onboarding` v4.3 / `login` v1.7.

### Thay đổi hành vi quan sát được ở client appService

- `404` / `405` / panic giờ trả envelope JSON thay vì text / đóng kết nối. Mọi route đã có: không đổi.

### Cross-file follow-ups

- `docs/specs/api/openapi.yaml` — v3.6 (đã sửa).
- `docs/specs/features/login/` v1.8, `docs/specs/features/json-body-hardening/` v1.4 — đã sửa.
- `docs/knowledge/features/` — concept `register.md` chỉ nhắc `utils` envelope chung chung, không cần sửa.
  Concept mới cho envelope (Architecture pattern) nếu cần: tạo bằng `il-okf-writer`.
- `openwiki/` — generated, không sửa tay; sẽ refresh theo workflow.
