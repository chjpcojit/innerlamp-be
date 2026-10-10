# Changelog — genealogy Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first.

## v5.5 (2026-10-10)

Cài đặt xong tầng usecase. Bump minor: không đổi hành vi, ghi tiến độ và hai điều chỉnh thiết kế nhỏ.

- `tasks.md`: tick mock repository (năm mock), năm usecase (`genealogy`, `genealogy_member`, `genealogy_access`, `genealogy_invitation`, `genealogy_profile`) và gộp các mục unit test; phần cần DB thật chuyển rõ sang Phase 3.
- `design.md`: `Revoke` + `Leave` gộp thành `RemoveViewer` (một quy tắc: owner gỡ bất kỳ viewer, viewer chỉ gỡ chính mình, owner tự gỡ → `ErrViewerNotFound`).
- `requirements.md`, `flow.md`: chỉ bump header.
- Điều chỉnh khi cài đặt: `MemberProfile.Member` là `MemberView` (để profile mang `in_law_label` và ngày âm); `GenealogyDetail`/`Chronicle` mang `ChapLunar` và `GenealogyMember` mang `DeathLunar` (tính ra khi đọc, không lưu); thêm `MemberView`/`MemberResult` ở domain.
- Code mới: `domain/mock_genealogy_repository.go`; `usecase/genealogy_validation.go`, `genealogy_usecase.go`, `genealogy_member_usecase.go`, `genealogy_access_usecase.go`, `genealogy_invitation_usecase.go`, `genealogy_profile_usecase.go` (+ test).
- Còn lại: repository + migration (Phase 3), handler + wiring + `openapi.yaml` (Phase 4), E2E (Phase 5).

## v5.4 (2026-10-10)

Chốt nơi đặt các giới hạn số. Bump minor: không đổi hành vi.

- Quyết định (anh Đức): **mọi giá trị không nhạy cảm đều đặt ở config**. Các giới hạn số (10 gia phả/người, 5000 thành viên, 60 đời, 5 phối ngẫu, 20 viewer) và hạn lời mời (720h = 30 ngày) là `config.GenealogyConfig`, không phải hằng trong domain như `design.md` v5.3 và trước đó ghi.
- `design.md`: sửa dòng Domain/Entity và Domain/Invitation, ghi chú Step C-2 trỏ tới `config.GenealogyConfig.InvitationTTL`.
- `tasks.md`: thêm và tick mục `GenealogyConfig` (config + Validate + yaml + khởi động), sửa mục domain.
- `requirements.md`, `flow.md`: chỉ bump header.
- Code: `config/config.go` (`GenealogyConfig`, `Validate`), `config/config.dev.yaml` (khối `genealogy:`), `cmd/main.go` (dừng service khi cấu hình thiếu/sai), test ở `config/config_test.go`.
- Không đưa vào config (cố ý): phạm vi quy đổi âm lịch 1900–2199 (giới hạn thuật toán, không đổi theo môi trường) và các giới hạn độ dài trường (CHECK của schema).

## v5.3 (2026-10-10)

Bắt đầu cài đặt. Bump minor: không đổi hành vi, chỉ làm rõ một quy tắc và ghi tiến độ.

- `tasks.md`: tick Phase 1 (entity, sentinel errors, interfaces) và các hàm thuần của Phase 2 (`SolarToLunar`, `BuildIndex`/`Age`/`RelativesOf`/`LineageOf`/`ChiOf`/`KinshipOf`); mock repository chưa làm nên chưa tick.
- `design.md`: làm rõ quy tắc "họ" là `min(dM, dT) ≥ 2` (bản cũ ghi `dT ≥ 2` sẽ gán "họ" cho cháu ruột).
- `requirements.md`, `flow.md`: không đổi nội dung, chỉ bump header.
- Code mới: `domain/genealogy.go`; `usecase/genealogy_lunar.go`, `usecase/genealogy_kinship.go` (+ test).
- Quyết định đang chờ anh Đức: các giới hạn số (10 gia phả, 5000 thành viên, 60 đời, 5 phối ngẫu, 20 viewer) và hạn lời mời 30 ngày nên là **`config/`** (đổi theo môi trường, theo quy ước của skill `golang-clean-arch-tdd`) hay **hằng** trong code như `design.md` đang ghi. Domain/repository hiện nhận chúng qua tham số nên chưa bị chặn.
- Phần mở rộng em thêm khi cài kinship (ngoài bảng đã duyệt, cần anh liếc lại): "Cháu dâu/Cháu rể" cho vợ/chồng của cháu, "Anh rể họ/Em rể họ/Chị dâu họ/Em dâu họ", "Bà nội/Cụ bà/Kỵ" cho vợ của tổ tiên trực hệ, và nếu "tôi" là dâu/rể thì chỉ gọi tên chồng/vợ và con của mình (còn lại `UNKNOWN`).
- Follow-up ngoài spec: `openapi.yaml`, ERD, migration — chưa làm.

## v5.2 (2026-10-10)

Chốt điều kiện trưởng tộc và cách thông báo. Bump minor: không thêm route/bảng/flow.

- `requirements.md`: AC82 — trưởng tộc phải **nam, còn sống, thuộc dòng họ**; đổi `gender`/`is_alive` của người đang giữ cờ mà không bỏ cờ thì `400`; vắng `is_clan_head` ở `PUT` = giữ nguyên; AC70 gỡ marker (thông báo push là tính năng riêng làm sau, không dùng SMS); Out of scope, Pending, Open questions (resolved) cập nhật.
- `design.md`: CHECK `is_clan_head ⇒ (kind='blood' AND gender='male' AND is_alive)`; ghi chú usecase; Deliberately-stubbed ghi push là tính năng riêng.
- `flow.md`: chỉ đổi header/version, verified không cần đổi sơ đồ.
- `tasks.md`: thêm ca test điều kiện trưởng tộc; Deferred ghi push (FCM/APNs) làm sau.
- Reason: anh Đức: "người chết không được mời", "tự huỷ khi xoá số của nút", "trưởng tộc là nam và còn sống", "thông báo làm tính năng riêng, push notification chứ không xài SMS".
- Đã có sẵn từ v5.1 nên không đổi: không mời người đã mất (AC44), huỷ lời mời chờ khi số của nút đổi hoặc bị xoá (AC30).
- Sửa hai câu cũ còn sót ở `requirements.md`: dòng "Ngày giỗ ... thuộc về `memorial`, không nhân đôi" (mâu thuẫn AC55) và "`tasks.md` Phase 1–6" (thực tế Phase 1–5). Thêm `SRS.md` (tài liệu BA, IEEE 29148) cùng thư mục; không đổi hành vi.
- Hệ quả cần biết: không có SMS nên người **chưa có tài khoản** chỉ thấy lời mời sau khi đăng ký bằng đúng số đó; owner tự báo họ ngoài ứng dụng.

## v5.1 (2026-10-10)

Giữ trưởng tộc như một cờ trên thành viên, và gắn lời mời với một thành viên cụ thể. Bump minor: không thêm route/bảng; thêm cột và field.

- `requirements.md`: thêm AC82 (`is_clan_head` trên thành viên blood, tối đa một/gia phả, người mới thì người cũ mất cờ, `clan_head_member_id` trong `GenealogyDetail`); AC44 lời mời nhận `member_id` + `phone_number?`, ghi số lên nút nếu chưa có, chỉ mời người còn sống; AC30 huỷ lời mời chờ khi số của nút đổi; AC70 thêm `member_full_name`; AC71 trả thêm `member_id`; AC73 liệt kê cả `accepted` để owner thấy trạng thái cập nhật; Member thêm `is_clan_head`; marker AC63 viết lại.
- `design.md`: schema `genealogy_members.is_clan_head` + unique một phần, `genealogy_invitations.member_id` + FK ghép; Step C-1/C-2 viết lại theo nút; Step G-2/M-2 xử lý cờ; Step I-2 trả `member_id`.
- `flow.md`: sơ đồ mời có `member_id`, ghi số lên nút; sơ đồ thêm thành viên có bước gỡ cờ cũ; sơ đồ chấp nhận trả `member_id`.
- `tasks.md`: thêm cột/index/FK, test trưởng tộc và mời theo nút, E2E trạng thái chuyển.
- Reason: anh Đức: "giữ trưởng tộc đó chỉ là 1 cờ nhớ trong db, 1 gia phả chỉ có 1 trưởng tộc" và "tôi chọn 1 người và invite họ bằng số điện thoại, ở trạng thái chờ; người đó nhận lời mời ở thông báo, đồng ý thì cập nhật lại trạng thái".
- Giả định em suy ra, cần anh xác nhận: (1) không mời được người đã mất; (2) mời ghi luôn số lên nút, và sửa/xoá số của nút sẽ tự huỷ lời mời chờ; (3) trưởng tộc không bị ràng buộc còn sống hay giới tính; (4) chỉ có hộp thông báo trong app (`GET /v1/invitations`), chưa có push/SMS (marker AC70 còn mở).
- Follow-up ngoài spec: như v5.0, thêm `openapi.yaml` schema `Invitation` (có `member_id`) và `is_clan_head`; migration/ERD.

## v5.0 (2026-10-10)

Đổi mô hình quyền theo anh Đức: chỉ còn `owner` (sửa) và `viewer` (chỉ xem); vị trí "tôi" trong cây tính theo số điện thoại. Bump major vì đổi vai trò, bỏ một bảng, bỏ 4 route, đổi tên route, thêm ràng buộc DB.

- `requirements.md`: bỏ vai trò `editor` và `clan_head` (AC51–AC52 bỏ); thêm `viewer` chỉ xem và AC81 (viewer gọi route ghi → `403 GENEALOGY_FORBIDDEN`); các route ghi (AC16, AC18, AC30, AC34, AC38, AC76) chỉ `owner`; đổi tên route `collaborators` → `viewers` và mời qua `POST .../invitations` (AC44–AC50, AC71); viết lại US-GEN-08: `my_member_id` **tính ra** theo `phone_number` của thành viên trùng số tài khoản (AC59), duy nhất trong gia phả (AC60, `409 MEMBER_PHONE_ALREADY_USED`), bỏ route `me/member` (AC61), người được mời không tự đổi được; `GET .../viewers` trả kèm `member_id` cho owner (AC48); 25 → 21 route; trần 20 giữ nguyên nhưng áp cho `viewer`.
- `design.md`: Step A chỉ còn owner/viewer; schema bỏ `clan_head_user_id` và `genealogy_user_members`, đổi `genealogy_collaborators` → `genealogy_viewers`, thêm unique `(genealogy_id, phone_number) WHERE phone_number IS NOT NULL`; Step M/D/R yêu cầu role = owner; Step C-1/C-2 và I-1/I-2 đổi tên + `viewer`; Step P-2/P-3 so khớp số điện thoại thay cho `FindMyMemberID`; 21 route, năm bảng.
- `flow.md`: bỏ marker/`editor`/`clan_head`; thêm nhánh `403` cho viewer ở các sơ đồ ghi; sơ đồ mời đổi path; sơ đồ hồ sơ dùng `FindPhoneByID`.
- `tasks.md`: bỏ `SetClanHead`, `SetMyMember`/`ClearMyMember`; thêm `ViewerRepository`, `FindMemberIDByPhone`, unique index, E2E viewer chỉ xem.
- Reason: anh Đức chốt "chỉ cho xem", "bỏ vai trò khác giữ xem thôi", "1 nút 1 tài khoản trong 1 gia phả", "họ không được đổi vì đó là gia phả do tôi tạo ra", tìm theo số điện thoại (một tài khoản một số, một nút một số).
- Giả định em suy ra, cần anh xác nhận: (1) bỏ luôn vai trò trưởng tộc (không còn route/field `clan_head`); (2) giữ trần 20 `viewer` (con số cũ của editor); (3) hai thành viên cùng gia phả **không** được cùng một số điện thoại (hệ quả của "một nút một tài khoản"), kể cả cha con dùng chung số; (4) owner muốn gắn người thân vào nút thì ghi `phone_number` của nút rồi mời cùng số đó (chưa có route "mời kèm gán nút").
- Follow-up ngoài spec (chưa làm): `migrations/0020_genealogy.sql` (năm bảng, không còn `genealogy_user_members`) + `docs/specs/db/erd.mmd`/`erd.svg`; `openapi.yaml` (+ `web/docs/public/openapi.yaml`) 21 route, schema `Viewer`, `GenealogyDetail` không còn `clan_head`; `docs/assumptions/genealogy/A-001`/`A-002` cập nhật; chưa có `docs/knowledge/features/genealogy.md` nên không cần refresh OKF.

## v4.2 (2026-10-10)

Duyệt bảng xưng hô v1 (AC69) — marker chặn cuối cùng. Bump minor: không thêm route/bảng/flow.

- `requirements.md`: AC69 bỏ marker, nêu rõ vai vế theo đời không theo năm sinh; cập nhật Pending (không còn marker chặn) và Open questions (resolved).
- `design.md`: bỏ marker ở "Bảng xưng hô v1", thêm mục "Quy tắc vai vế" (vai trên/ngang/dưới theo `generation` so với tổ tiên chung, so lớn/nhỏ bằng `birth_order` của hai nhánh con trực tiếp, hậu tố "họ", ca xa).
- `flow.md`: bỏ `Note` ⚠️ AC69 ở sơ đồ hồ sơ thành viên.
- `tasks.md`: bỏ "chặn bởi marker AC69", bổ sung ca bảng test (con bác lớn hơn dù nhỏ tuổi, chú nhỏ tuổi hơn cháu, "họ", ông bác/ông chú, chắt/chút).
- Reason: anh Đức đồng ý bảng xưng hô và quy tắc "vai vế theo đời, không theo tuổi; thêm họ/ông bác/ông chú cho ca xa".
- Phần mở rộng em thêm theo quy tắc chung (anh đã đồng ý "quy tắc họ và ông bác/ông chú"; **chắt/chút và "cháu họ"** là suy ra từ quy tắc, cần anh liếc lại khi viết test).
- Không đổi: schema, route, Step A–R.

## v4.1 (2026-10-10)

Chốt thêm ba câu hỏi mở theo anh Đức. Bump minor vì không thêm route/bảng/flow, chỉ thêm field tính ra và một hàm thuần.

- `requirements.md`: AC40/AC55 — server quy đổi dương → âm và trả field tính ra `chap_lunar_day/month/leap` (trong `chronicle`) và `death_lunar_day/month/leap` (trong `Member`), `null` ngoài năm `1900..2199`; AC51 chốt cấp thẳng; AC54 chốt không có chuyển quyền `owner`; cập nhật Out of scope, Pending, Open questions (resolved).
- `design.md`: thêm `usecase/genealogy_lunar.go` (hàm thuần `SolarToLunar`), Step G-3/M-3/R-3 dựng field âm lịch; bỏ mục "server tự quy đổi" khỏi Deliberately-stubbed (đổi thành: nhắc hằng năm/quy đổi âm → dương thuộc `memorial`).
- `flow.md`: hai message tính `death_lunar_*` ở sơ đồ thêm thành viên và đổi quan hệ.
- `tasks.md`: thêm task `genealogy_lunar.go` + bảng test; bỏ hai mục chờ AC40/AC51 khỏi Deferred.
- AC55 chốt: ngày giỗ lưu ngay trên thành viên, không ghi vào `memorial` (gỡ marker; bỏ mục liên kết `memorial` khỏi `design.md`/`tasks.md`/Out of scope).
- Reason: anh Đức trả lời "server tính và quy đổi luôn", "trưởng tộc cấp thẳng", "không có chuyển quyền owner".
- Không đổi: schema/migration (field âm lịch chỉ tính ra), 25 route, thiết kế Step A–R. `design.md` đã cấp thẳng trưởng tộc nên AC51 không phát sinh thay đổi.
- Follow-up ngoài spec: `openapi.yaml` (+ `web/docs/public/openapi.yaml`) thêm `chap_lunar_*`/`death_lunar_*` (readOnly) vào schema `Chronicle`/`Member`; `docs/assumptions/genealogy/A-002` mục (7) đã được xác nhận.

## v4.0 (2026-10-10)

Chốt các câu hỏi mở theo anh Đức và thêm route đổi quan hệ thành viên. Bump major vì thêm một route + một flow (Step R), đổi cột trong migration (chronicle, member).

- `requirements.md`: chốt AC10/24/25/47 (10 gia phả, 5000 thành viên, 60 đời, 20 editor tính cả lời mời chờ) và AC75 (30 ngày); AC19 con của nữ dòng họ không vào cây và AC4/AC30 thuỷ tổ phải nam; AC35 giữ chặn xoá người còn hậu duệ; AC27 chuyển từ "không kiểm" sang cảnh báo `warnings` (không chặn); AC31 trỏ sang route mới; AC39 thêm `couplet_{upper,lower}_{reading,meaning}`; AC40/AC55 lưu dương lịch kèm cờ `*_entered_as_lunar`, thêm `chap_year`, bỏ `chap_leap_month`, đổi `death_lunar_*` → `death_month/death_day/death_entered_as_lunar`; thêm US-GEN-10 + AC76–AC80 (đổi quan hệ), route thứ 25, `warnings` trong contract, mục Open questions (resolved), cập nhật Pending.
- `design.md`: 25 route; thêm Step R-1..R-3 (đổi quan hệ trong một tx: khoá gia phả, tải cả cây, kiểm vòng/đời/giới tính trong bộ nhớ, dời `generation` cả nhánh); Step M-2 chặn con của nữ, Step M-3 và R-3 tính `warnings`; `ErrFounderNotMovable`, `Move`; schema chronicle và member đổi cột; bỏ các cờ "⚠️ đề xuất" của trần số.
- `flow.md`: thêm sơ đồ "Đổi quan hệ thành viên" (mirror Step R-1..R-3), sơ đồ thêm thành viên có `warnings` và check giới tính, bỏ các `Note` ⚠️ đã chốt.
- `tasks.md`: thêm mục `Move`, `warnings`, test đổi quan hệ + E2E, 25 route, cột migration mới; bỏ "route đổi cha/mẹ" khỏi Deferred.
- Reason: anh Đức chốt các số trần, "con thành viên nữ không thêm vào", "không tạo thuỷ tổ nữ", "chặn xoá", "cảnh báo năm sinh", "có thể đổi quan hệ", "âm dương: lưu dương, có cờ nhớ âm", "phiên âm/nghĩa cho cả hai".
- Giả định em tự suy ra, cần anh xác nhận: (1) "chặn xoá, phải xoá theo cây" = giữ 409, xoá từ lá lên; (2) quy tắc dương+cờ áp cho cả ngày giỗ thành viên; (3) thêm `chap_year`/yêu cầu `death_year` đi kèm ngày giỗ để có thể quy đổi lại; (4) route đổi quan hệ không cho đổi loại dòng họ ↔ dâu/rể (AC80).
- Follow-up cho các file ngoài spec (chưa làm ở bước này): `migrations/` (chưa có file genealogy; số trống kế tiếp kiểm bằng `ls migrations/` ngay trước khi tạo) + `docs/specs/db/erd.mmd`/`erd.svg`; `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) — 25 route, `warnings`, `FOUNDER_NOT_MOVABLE`; `docs/assumptions/genealogy/A-002` mục (2)(3) đã được anh xác nhận; chưa có `docs/knowledge/features/genealogy.md` nên không cần refresh OKF.

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
