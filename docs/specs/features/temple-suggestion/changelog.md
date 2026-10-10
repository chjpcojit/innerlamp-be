# Changelog — temple-suggestion Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first.

## v2.2 (2026-10-10)

- `requirements.md`: chốt N7 — backoffice chép bảng ánh xạ `heritage_rank → heritage_priority` kèm test đối chiếu, không đưa vào `pkg/`. Không còn marker.
- `design.md`: ghi bản chép + test đối chiếu vào bảng domain của backoffice; thêm Known gap "ánh xạ priority có hai bản".
- `flow.md`: no change needed, verified — chỉ cập nhật header (flow không nhắc N7).
- `tasks.md`: Phase 4 đổi mục "Quyết N7" thành việc cụ thể (bản chép + test đối chiếu).
- Reason: anh Đức chọn phương án chép. Bump minor vì không đổi hành vi.

## v2.1 (2026-10-10)

- `requirements.md`: chốt N6 (id chùa mới `temple_<region>_<NNN>`) và mã user story `US-TSG-01..04` (anh Đức tự chốt); chuyển sang "Open questions (resolved)". Còn 1 marker: N7.
- `design.md`: Step C-4 bỏ nhãn chưa chốt; Known gaps ghi lại rủi ro trùng id với seed. Số bước không đổi.
- `flow.md`: bỏ Note "Not confirmed" ở C-4; số bước không đổi.
- `tasks.md`: Phase 4 chỉ còn N7 cần quyết.
- Reason: anh Đức đồng ý đề xuất N6 và chốt mã US. Bump minor vì không đổi hành vi.

## v2.0 (2026-10-10)

Đồng bộ với `SRS.md` v1.0 (IEEE 29148, đã duyệt) sau khi anh Đức chốt các câu mở.

- `requirements.md`: chốt N1 (rate-limit theo IP, 5 đề xuất/IP/giờ; rủi ro IP dùng chung được chấp nhận), N2 (tỉnh/thành chuỗi tự do), N3 (không dò trùng), N4 (**thêm trạng thái `rejected`**, lý do tuỳ chọn ≤500 ký tự), N5 (role: `admin` + `content_editor`; mở lại đề xuất đã từ chối chỉ `admin`); đề xuất lưu vĩnh viễn, không lưu IP. Thêm US-TSG-04 và AC28–AC31; sửa AC7, AC10, AC13, AC15; thêm route `reject` (phase sau). Còn 2 marker: N6, N7.
- `design.md`: `status` thêm `rejected`, cột `reject_reason`, CHECK mở rộng; Step C-1 ghi rõ `RequireAnyRole`; Step C-3 thêm nhánh `rejected` + role ≠ admin → 403; Step C-6 xoá `reject_reason`; request flow thêm route reject (một câu SQL nên không thành Step-by-step Flow); cập nhật Known gaps.
- `flow.md`: mirror C-3 (nhánh `ErrReopenForbidden`) và C-6 (`reject_reason=NULL`); số bước không đổi.
- `tasks.md`: bỏ nhãn "chặn bởi N1/N5"; Phase 3–4 cập nhật cho `rejected`; **thêm Phase 6 — Từ chối đề xuất** (làm sau); bỏ N2/N3/N4 khỏi Deferred.
- Reason: SRS bổ sung trạng thái `rejected` và chuyển `rejected` → `created`, đổi phạm vi nên bump major.
- Follow-ups (chưa làm): `migrations/0022_temple_suggestions.sql` phải có `rejected` + `reject_reason`; `docs/specs/db/erd.mmd`/`erd.svg`; `docs/specs/api/openapi.yaml` (+ `web/`) cho 4 route (route reject khi tới Phase 6); concept OKF chỉ tạo sau khi code kiểm chứng.

## v1.0 (2026-10-10)

Initial spec written before implementation (Prospective mode), as the work order for letting a client suggest a temple and the backoffice turn that suggestion into an official temple.

- `requirements.md`: 3 user story (gửi đề xuất / xem hàng chờ / tạo chùa), 27 AC. Chốt theo anh Đức: gửi công khai, ảnh nhập bằng đường dẫn, admin điền đủ form khi tạo chùa; chỉ hai trạng thái `pending` → `created`. 7 marker `⚠️ Not confirmed` (N1–N7).
- `design.md`: bảng `temple_suggestions` (migration dự kiến `0022`); 1 route ở appService, 3 route ở backofficeService; một Step-by-step Flow (C-1..C-6) cho `create-temple` với một transaction khoá dòng đề xuất.
- `flow.md`: một sequence diagram mirror C-1..C-6.
- `tasks.md`: 5 Phase, mọi mục `[ ]`; N1/N5/N6 chặn các mục tương ứng.
- Reason: yêu cầu mới — người dùng app đề xuất chùa chưa có, ban quản trị duyệt, bổ sung thông tin và đưa vào danh mục.
- Follow-ups (chưa làm, không tự sửa ở đây): `migrations/0022_temple_suggestions.sql`, `docs/specs/db/erd.mmd`/`erd.svg`, `docs/specs/api/openapi.yaml` (+ `web/`), concept OKF `docs/knowledge/features/` chỉ tạo sau khi code kiểm chứng. `docs/specs/features/temple/requirements.md` có thể thêm một dòng trỏ tới spec này — chưa sửa vì chưa được yêu cầu.
