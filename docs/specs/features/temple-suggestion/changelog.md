# Changelog — temple-suggestion Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first.

## v3.4 (2026-10-11)

Implement phần duyệt ở backoffice (Phase 3, 4, 5 và Phase 8). Theo yêu cầu của anh Đức: "tiếp tục đi em", "đồng ý" (đề xuất của em: giải quyết điểm nghẽn rồi làm backoffice). Không đổi yêu cầu; thông tin triển khai được ghi lại.

- Điểm nghẽn "shared file conflicts (config.go, main.go, errors.go)" của HANDOFF hoá ra không phải xung đột git (không có marker) — `go build ./...` đã xanh; backoffice chỉ cần thêm vào ba file đó. Không cần giải quyết gì thêm.
- Code (`services/backofficeService/`): `domain/temple_suggestion.go` (`ProvinceCode`/`WardCode`/`Ward` cho đề xuất; `NewTemple` thêm mã + tên chụp; `Temple.FullAddress()` bản chép của appService; `WardResolver`, `ResolvedWard`, sentinel); `usecase/temple_suggestion_usecase.go` (`List`/`Get`/`CreateTemple`); `repository/postgres_temple_suggestion_repository.go` (`List`, `FindByID`, `CreateTemple` một transaction `FOR UPDATE` + advisory lock theo region + `INSERT temples` + `UPDATE`, `ResolveCurrentWard`); `delivery/http/temple_suggestion_handler.go` (+`RequireAnyRole`, map lỗi); `config` (`temple_suggestion.page_size`, `server.public_base_url`); `cmd/main.go` (3 route qua `RequireActiveSession` + `RequireAnyRole(admin, content_editor)`).
- Test: unit usecase/handler/config; repository trên Postgres thật (danh sách, thứ tự, tạo, id theo region, mở lại `rejected`, rollback, đua 4 request, `ResolveCurrentWard`) — chỉ chạy khi `TEST_DB_NAME` là DB scratch khác `innerlamp`; E2E mới `services/backofficeService/e2e/` (harness riêng, binary thật + testcontainers).
- `openapi.yaml` (+ `web/docs/public/`): 3 route admin, schema `TempleSuggestionItem`/`TempleSuggestion`/`CreateTempleRequest`/envelope.
- Quyết định kèm theo (spec chưa nêu, em chọn): `RequireAnyRole(next, roles...)` nhận handler trước (khớp `RequireRole`); `{id}` sai dạng trả `404 TEMPLE_SUGGESTION_NOT_FOUND` như id không tồn tại; số bắt buộc trong body (`established_year`, `latitude`, `longitude`) thiếu thì `400` với `details.field` đúng tên (con trỏ trong DTO); đề xuất `rejected` mở lại bởi `content_editor` trả `403 FORBIDDEN`; response chùa của `create-temple` ghép ảnh bằng `server.public_base_url` của backoffice (phải trùng appService khi triển khai).
- Chưa làm: Phase 6 (từ chối); `erd.svg` (chưa có `mmdc`/Chrome); concept OKF; E2E chéo hai binary cho AC25.
- `requirements.md`, `design.md`, `flow.md`, `tasks.md` (3.3 → 3.4): header/Status; `tasks.md` đánh `[x]` các mục đã kiểm chứng. Spec `temple` (AC 38) được đánh dấu xong ở `temple` v1.14.

## v3.3 (2026-10-11)

Cập nhật `SRS.md` theo v3.2 và sửa số liệu chùa. Theo yêu cầu của anh Đức: "sửa SRS của temple-suggestion", "sửa số liệu lại 93". Không đổi AC.

- `SRS.md` (1.1 → 1.2): **Amendment A2** — chùa chính thức do ban quản trị tạo cũng có địa chỉ có cấu trúc: FR-009 yêu cầu tỉnh và xã chọn từ metadata hành chính (xã thuộc tỉnh), tên lấy từ metadata, thêm 4 Gherkin scenario (11 tổng); mục 2.7 (delta 7), 3.4, bảng truy vết FR-009 và Appendix A (hai dòng quyết định, gồm "không cần tiền tố Thành phố").
- `requirements.md` (3.2 → 3.3): ghi chú N6 không còn nói "179 chùa" mà nói theo dạng id; AC không đổi.
- `design.md`, `flow.md`, `tasks.md` (3.2 → 3.3): no change needed, verified — chỉ cập nhật header.
- Reason: SRS là nguồn nghiệp vụ của spec này và phải phản ánh quyết định A2; con số 179 là số cũ so với `temple.json` (93 chùa).

## v3.2 (2026-10-11)

`create-temple` nhận và ghi địa chỉ có cấu trúc (N9). Theo yêu cầu của anh Đức: "khi đề xuất thêm chùa thì cũng có địa chỉ và nó sẽ được lấy từ metadata" — cùng đợt với spec `temple` v1.7. Chưa có code — Phase 8.

- `requirements.md` (3.1 → 3.2): AC18 thêm `province_code`, `ward_code` vào các field bắt buộc; **thêm AC18a** (mã không tồn tại / xã lệch tỉnh → 400, kiểm trước transaction) và **AC18b** (ghi mã như gửi, tên lấy từ dataset, một quy tắc cho cả đề xuất cũ lẫn mới); bảng API contract, Pending, Ghi chú kỹ thuật; thêm N9 vào "Open questions (resolved)".
- `design.md` (3.1 → 3.2): bảng backoffice (entity, `WardResolver`, usecase, repository), dòng Migration `0032`; **thêm Step C-3 "đối chiếu mã"** và đánh số lại C-3..C-6 → C-4..C-7 (các entry cũ của changelog này vẫn dùng số cũ); Known gaps: gap "`create-temple` chưa dùng mã" chuyển sang trạng thái đã thiết kế, thêm gap "bản chép SQL tra mã thứ hai".
- `flow.md` (3.1 → 3.2): sơ đồ `create-temple` thêm participant `WardResolver` và C-3, mirror 1:1 `design.md`; sơ đồ gửi đề xuất không đổi.
- `tasks.md` (3.1 → 3.2): thêm **Phase 8** (mọi mục `[ ]`); sửa tham chiếu "C-3..C-6" của Phase 4 thành "C-4..C-7"; mục Deferred về cột `temples` trỏ sang `temple` v1.7.
- `SRS.md`: không đổi ở lượt này (do `il-ba` sở hữu); amendment cho SRS nếu cần là việc riêng. Ghi nhận, chưa làm.
- Quyết định kèm theo (người dùng chưa chỉ định, spec chọn): body luôn mang mã, không có nhánh riêng cho đề xuất gửi trước v3.0 (admin chọn mã khi đề xuất không có); backoffice giữ bản `WardResolver` riêng thay vì import `appService/domain` (cùng cách xử lý `HeritagePriority`, N7).
- Việc cần làm ngoài thư mục này (chưa làm): `migrations/0032` (spec `temple`), `docs/specs/api/openapi.yaml` + `web/docs`, `docs/specs/db/erd.*`, concept OKF (`il-okf-writer`).

## v3.1 (2026-10-11)

Implement Phase 7 (hợp đồng gửi đề xuất v3.0). Không đổi yêu cầu; thông tin triển khai được ghi lại.

- `tasks.md` (3.0 → 3.1): Phase 7 đánh `[x]` — migration, domain, usecase, repository, handler, wiring, `openapi.yaml`, `web/docs`, E2E; còn `[ ]`: sinh lại `erd.svg` (chưa có `mmdc`) và mục backoffice Phase 3.
- `design.md` (3.0 → 3.1): bỏ dấu "chưa làm/cần sửa" ở bảng Layers; mô tả `WardResolver` đúng như code (`ResolvedWard`, `ErrWardNotInProvince`, một câu `LEFT JOIN`); Step S-3 sửa theo đó.
- `requirements.md`, `flow.md` (3.0 → 3.1): chỉ cập nhật Status/Pending; nội dung AC và sơ đồ không đổi.
- Số migration: **`0030`**, không phải `0029` như bản nháp v3.0 (`0029` đã dùng cho `users_avatar`). `docs/specs/db/erd.mmd` còn ghi `0030_calendar_events.sql` là migration dự kiến của feature `calendar`; feature đó cần đổi số khi làm.
- Kiểm chứng: `go test -short ./...` (appService, backofficeService); `TEST_DB_NAME=innerlamp_test go test ./repository/ ./e2e/` — pass.
- Quyết định khi implement: `ResolveCurrentWard` thêm vào `AdministrativeUnitRepository` (và mock), usecase nhận port hẹp `domain.WardResolver`; kiểm mã tỉnh trước, xã sau (một câu SQL).

## v3.0 (2026-10-11)

Đổi hợp đồng gửi đề xuất: địa chỉ có cấu trúc theo metadata hành chính hệ mới (N8). Chưa có code — Phase 7.

- `requirements.md` (2.4 → 3.0): AC1 body `{name, province_code, ward_code, address}`; AC2 thêm `province_code`/`ward_code` bắt buộc; AC3 bỏ giới hạn `province`; **thêm AC3a** (mã không tồn tại / xã lệch tỉnh → 400, chỉ nhận mã hệ mới) và **AC3b** (field `province` cũ bị từ chối, không chạy song song); AC5 lưu mã kèm tên chụp từ dataset; AC9/AC13/AC26 thêm `province_code`, `ward_code`, `ward`; bảng API contract, Pending, Ghi chú kỹ thuật; N2 bị N8 thay thế.
- `design.md` (2.4 → 3.0): bảng Layers (entity, `WardResolver`, usecase, repository, DTO, wiring, migration `0030`), Schema (ba cột nullable + CHECK, không FK), Request flow, **thêm Step-by-step Flow S-1..S-4**, Known gaps (bỏ field cũ, `create-temple` chưa dùng mã, S-3/S-4 không chung transaction).
- `flow.md` (2.4 → 3.0): thêm sơ đồ sequence gửi đề xuất, mirror S-1..S-4; sơ đồ C-1..C-6 không đổi.
- `tasks.md` (2.4 → 3.0): thêm **Phase 7** (mọi mục `[ ]`; Phase 2 vẫn `[x]` theo hợp đồng v2.x, không đánh dấu lại).
- `SRS.md` (1.0 → 1.1): Amendment A1 — địa chỉ có cấu trúc thay cho "tỉnh là chuỗi tự do"; sửa dòng giả định tương ứng.
- Quyết định (anh Đức, 2026-10-11): bỏ hẳn `province` chuỗi, không chạy song song; `ward_code` bắt buộc. Ghi ở `docs/assumptions/temple-suggestion/A-001..A-002` các giả định em tự thêm (không FK; tên chụp; không backfill).
- Việc tiếp theo ngoài thư mục này (chưa làm): migration `0030`, `docs/specs/api/openapi.yaml` + `web/docs`, `docs/specs/db/erd.*`, concept OKF (`il-okf-writer` sau khi code kiểm chứng).

## v2.4 (2026-10-10)

Dọn tài liệu sau khi rà soát trước khi làm tiếp; không đổi hành vi hay phạm vi.

- `requirements.md`: Status/mở đầu không còn "not started" (US-TSG-01 đã có code); ghi chú AC7 không còn nói rate-limit chưa wire; "API contract" ghi route gửi đề xuất đã có trong `openapi.yaml`; "Pending" chỉ còn phần duyệt; ghi chú schema bỏ chữ "dự kiến/cần cập nhật" (migration, ERD, README đã có).
- `design.md`, `tasks.md`: no change needed, verified — chỉ cập nhật header.
- `flow.md`: Status nói rõ sơ đồ này chỉ cho phần duyệt (chưa có code).
- Reason: các dòng trạng thái còn mô tả trạng thái trước khi implement Phase 1–2 (v2.3).
- Kiểm chứng: `go vet ./services/appService/...`, `go test -short ./services/appService/...` sạch.
- Ghi nhận, chưa sửa: `web/docs/public/openapi.yaml` lệch `docs/specs/api/openapi.yaml` ở phần auth (`X-API-Key`, `expires_in`, `no-store`) từ trước, ngoài phạm vi feature này.

## v2.3 (2026-10-10)

Implement Phase 1 và Phase 2 (gửi đề xuất, appService) theo `golang-clean-arch-tdd`.

- `requirements.md`: no change needed, verified — hành vi đã implement khớp AC1–AC8; chỉ cập nhật header.
- `design.md`: bảng layer đánh dấu phần đã làm; `CreateTemple` có thêm tham số `priority` (usecase suy ra từ hạng rồi truyền xuống, repository không tự tính) và trả `CreatedTemple`; cập nhật Known gap rate-limit (đã wire nhưng bộ đếm theo từng instance, production phải bật `trust_forwarded_for`). Số bước của flow không đổi.
- `flow.md`: chữ ký `CreateTemple` trên sơ đồ có sẵn `role`; thêm tham số `priority` không đổi số bước — no change needed ngoài header.
- `tasks.md`: tick Phase 1 (4 mục) và Phase 2 (6 mục) sau khi kiểm chứng; mục "Quyết N7" của Phase 4 tick vì bản chép + test đối chiếu đã có. Phase 3–6 vẫn `[ ]`.
- Reason: bắt đầu code sau khi mọi câu hỏi đã chốt. Bump minor vì không đổi phạm vi.
- Kiểm chứng: usecase (table-driven, mock), handler, rate-limit, config (dev yaml), repository trên Postgres thật (gồm CHECK), E2E trên binary thật + testcontainers Postgres (route đã wire, 201, trùng vẫn 201, validation, field lạ, 429 ở lần thứ 6, không có route đọc, sai `X-API-Key` 401 không tốn hạn mức). `go vet ./...`, `go test -short ./...` và `check_dependency_rule.sh` cho cả hai service đều sạch.
- Ghi nhận khi làm: bản chép `HeritagePriority` ở backoffice được test đối chiếu với appService; test import `appService/domain` chỉ trong file `_test.go` nên không tạo phụ thuộc runtime.
- Follow-ups (chưa làm): `openapi.yaml` (+ `web/`) mới có route gửi đề xuất, ba route duyệt thêm ở Phase 3–4; concept OKF `docs/knowledge/features/temple-suggestion.md` chỉ tạo sau khi cả luồng gửi → duyệt chạy end-to-end; `config.<prod>.yaml` (khi có) phải đặt `temple_suggestion.rate_limit.trust_forwarded_for: true`.

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
