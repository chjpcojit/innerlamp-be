# Changelog — genealogy Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first.

## v3.0 (2026-10-08)

Chốt AC44/AC45 theo hướng **lời mời**: mọi lần cấp quyền editor đều qua lời mời (số đã hay chưa đăng ký đều tạo lời mời chờ), người được mời nhận trong app rồi chấp nhận hoặc từ chối. Bump major vì thêm bảng/migration, thêm một flow, thêm 5 route.

- `requirements.md`: viết lại AC44–AC47 (tạo lời mời, response đồng nhất không lộ số đã đăng ký, `409` vai trò/lời mời trùng, trần 20 tính cả lời mời chờ); thêm AC70–AC75 (hộp lời mời, chấp nhận, từ chối, owner xem/huỷ lời mời, khớp theo số điện thoại + hạn 30 ngày); AC51 giữ cấp thẳng cho trưởng tộc và thêm marker; AC53 thêm AC73/AC74; thêm story "người được mời", 5 route + schema `Invitation`/`MyInvitation`; bỏ marker AC44/AC45 (chuyển "Open questions (resolved)"), thêm marker AC51/70/72/75, AC45 không còn chặn.
- `design.md`: Step C-1/C-2 viết lại (không tra user, ghi lời mời trong một tx, đánh dấu `expired` lười); thêm Step I-1/I-2 (chấp nhận, khoá `genealogies` trước `genealogy_invitations` để không deadlock); thêm bảng `genealogy_invitations`, usecase/repo/handler mới, 24 route, sáu bảng; bỏ `CollaboratorRepository.Add` (insert editor nằm trong tx `Accept`).
- `flow.md`: viết lại sơ đồ "Mời người cùng sửa" (mirror Step C-1/C-2), thêm sơ đồ "Chấp nhận lời mời" (mirror Step I-1/I-2).
- `tasks.md`: không thêm Phase mới; thêm/sửa mục ở Phase 1–5 (entity, usecase, repo, handler lời mời, 24 route, unit/repo/E2E). Bỏ mục "chặn bởi AC45". Mọi mục mới là `[ ]`.
- Reason: anh Đức chốt "số có hoặc chưa đăng ký đều đưa vào lời mời, user nhận thông báo và đồng ý hoặc từ chối" — thay giả định A-001 (cấp thẳng) và giải marker AC45 (lộ số đã đăng ký).
- Follow-up cho các file ngoài spec (chưa làm ở bước này): `migrations/0020_genealogy.sql` — số `0020` và `0021` đã bị migration khác dùng (`ls migrations/` ngay trước khi tạo, hiện số trống kế tiếp là `0022`) và `docs/specs/db/erd.mmd` + `erd.svg` (bảng `genealogy_invitations`); `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) — 24 route và schema `Invitation`/`MyInvitation`; `docs/assumptions/genealogy/` — A-001 đã bị thay, thêm A-002; chưa có `docs/knowledge/features/genealogy.md` nên không cần refresh OKF.

## v2.2 (2026-10-07)

- `requirements.md`: chốt AC68 — rẽ nhánh để chia chi chỉ đếm con trai; bỏ marker, chuyển sang "Open questions (resolved)", bỏ AC68 khỏi danh sách Pending.
- `design.md`: no change needed, verified — "Quy tắc chi" đã đếm con trai.
- `flow.md`: no change needed, verified.
- `tasks.md`: no change needed, verified.
- Reason: anh Đức trả lời "tính trên con trai thôi".

## v2.1 (2026-10-07)

Sửa định nghĩa "chi" theo phản hồi của anh Đức: chi bắt đầu từ chỗ cây rẽ nhánh và lồng nhau, không phải "tổ tiên đời 2" như v2.0 hiểu nhầm.

- `requirements.md`: viết lại AC68 — `chi` (chi hiện tại) và `chi_path` (các chi từ ngoài vào trong); đầu chi là con trai của một người có ≥ 2 con trai; `rank`/`is_eldest` tính trong các con trai của cha/mẹ đó. Sửa quyết định nền tảng, mục Open questions (resolved) và bảng `MemberProfile` (thêm `chi_path`). Marker AC68 viết lại: rẽ nhánh chỉ đếm con trai hay cả con gái, và nhãn chi đổi khi có thêm con trai thứ hai.
- `design.md`: thêm mục "Quy tắc chi" (thuật toán một lượt đi ngược `parent_id`, ví dụ Lộc/Nghĩa/A/B và các ca test); không đổi schema, Step hay layer.
- `flow.md`: no change needed, verified — Step P-3 vẫn gọi `Chi` như cũ, chỉ thay đổi quy tắc bên trong hàm thuần.
- `tasks.md`: sửa mô tả bảng test của `genealogy_kinship.go` cho khớp quy tắc chi mới; không đổi Phase.
- Reason: phản hồi của anh Đức — chọn người ở nhánh A thì là chi A, không phải chi cụ đời 2.

## v2.0 (2026-10-07)

Bổ sung theo các quyết định chốt với anh Đức sau khi đối chiếu spec v1.0 với nhu cầu của người dùng cuối (xem hồ sơ một người, biết mình ở đâu trong cây). Bump major vì thêm bảng/migration, thêm một flow mới và thêm route.

- `requirements.md`: thêm US-GEN-07..09 (AC55–AC69) — thông tin mở rộng của thành viên (`other_parent_id`, `birth_order`, `burial_place`, ngày giỗ âm `death_lunar_*`), con trỏ "tôi là ai" riêng tư, route hồ sơ `profile` tính ở server (tuổi, họ hàng, đường từ thuỷ tổ, chi, xưng hô). Sửa AC30/AC31 (field sửa được, `other_parent_id` không đổi được). Chốt AC48 (chỉ `owner` quản lý quyền và trưởng tộc) — bỏ marker. Thêm bốn quyết định nền tảng (chỉ owner đổi trưởng tộc, "tôi là ai" do client đặt qua API, tính ở server, "chi" = tổ tiên đời 2) và sáu mục "Open questions (resolved)". Route 16 → 19; thêm schema `MemberProfile`, `my_member_id` trong `GenealogyDetail`. Các marker mới: AC55, 56, 63, 65, 66, 68, 69 (AC69 chặn hàm xưng hô).
- `design.md`: thêm bảng `genealogy_user_members` và cột mới của `genealogy_members` (FK `ON DELETE SET NULL (other_parent_id)`, CHECK); Step M-2 thêm bước 4b/4c; thêm Step-by-step Flow P (hồ sơ thành viên, tải cả cây một lần trong tx `READ ONLY REPEATABLE READ` rồi tính ở hàm thuần `genealogy_kinship.go`) và "Bảng xưng hô v1" (đề xuất, chưa duyệt); thêm usecase/handler mới, bảng quyền thêm một dòng. **Sửa số migration `0019` → `0020`** vì `0019_sutra_audio_search.sql` đã tồn tại (v1.0 ghi sai).
- `flow.md`: thêm sơ đồ "Hồ sơ thành viên" (mirror Step P-1..P-3); sơ đồ "Thêm thành viên" thêm hai nhánh `opt` (kiểm `other_parent_id`, gán `birth_order`).
- `tasks.md`: không thêm Phase mới; thêm mục vào Phase 1–5 (entity/interface mới, `genealogy_profile_usecase.go`, `genealogy_kinship.go`, repo của con trỏ, handler `profile`, 19 route, E2E cho `profile`). Mục xưng hô ghi rõ bị chặn bởi AC69. Mọi mục mới là `[ ]`. Migration đổi tên `0020_genealogy.sql`.
- Reason: spec v1.0 chưa đủ để dựng màn hình chi tiết một người (thiếu mẹ, thứ tự con, giỗ, an táng, xưng hô, chi, vị trí của "tôi" trong cây); blocker "con gắn cha hay mẹ" trong handoff được giải bằng `parent_id` + `other_parent_id`.
- Follow-up cho các file ngoài spec (chưa làm ở bước này): `migrations/0020_genealogy.sql` và `docs/specs/db/erd.mmd` + `erd.svg`; `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) — 19 route và các schema mới; `docs/assumptions/genealogy/` — thêm giả định về "con trỏ riêng tư, cho phép nhiều user cùng chọn một nút" (AC63) nếu anh muốn lưu; concept OKF `docs/knowledge/features/genealogy.md` chỉ viết sau khi code được kiểm chứng.

## v1.0 (2026-10-07)

Initial spec written **before implementation**, as the work order for building the gia phả feature.

- `requirements.md`: US-GEN-01..06, 54 AC (tạo gia phả + thuỷ tổ, xem/sửa/xoá, thêm/sửa/xoá thành viên, phả ký, cấp quyền + trưởng tộc), contract 16 route. Bảy quyết định nền tảng đã chốt với anh Đức hôm nay (ba vai trò owner/clan_head/editor, trưởng tộc là vai trò gắn user, cấp quyền bằng số điện thoại, không có viewer, con chọn 1 cha/mẹ, nhiều vợ/chồng ghi dâu/rể, chỉ lưu năm). Các giới hạn số và nhiều chi tiết biên còn là `⚠️ Not confirmed` (AC10, 19, 24, 25, 27, 31, 35, 39, 40, 44, 45, 47, 48, 54, và ghi đè đồng thời ở Pending).
- `design.md`: Step A (phân giải vai trò, dùng chung), bốn Step-by-step Flow (G tạo, M thêm thành viên, C cấp quyền, D xoá thành viên), schema bốn bảng với FK ghép cùng-gia-phả, bảng quyền theo vai trò.
- `flow.md`: bốn sequence diagram mirror 1:1 các flow trên; các `⚠️ Not confirmed` xuất hiện dưới dạng `Note`.
- `tasks.md`: Phase 1–5 + Deferred; mọi mục `[ ]`.
- Reason: yêu cầu mới của anh Đức về tính năng gia phả.
- Follow-up cho các file ngoài spec (chưa làm ở bước này): `migrations/0019_genealogy.sql` và `docs/specs/db/erd.mmd` + `erd.svg`; `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`); concept OKF `docs/knowledge/features/genealogy.md` chỉ viết sau khi code được kiểm chứng.
- Giả định của AI ghi ở `docs/assumptions/genealogy/A-001-grant-takes-effect-immediately.md`.
