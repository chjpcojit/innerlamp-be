# Tasks — genealogy

Version: 5.5
Last updated: 2026-10-10
Status: in progress — config, Phase 1 domain (kèm mock) và Phase 2 usecase xong, có test; còn Phase 3 repository + migration, Phase 4 handler + wiring + openapi, Phase 5 E2E.
Module: `genealogy`

> See `changelog.md` in this folder for the version history of this spec.

Mỗi mục trace về một dòng của bảng "Layers & components" hoặc một Step trong `design.md`. Làm theo TDD (skill `golang-clean-arch-tdd`), kiểm E2E cuối (skill `golang-e2e-verification`). Các mục bị chặn bởi marker `⚠️ Not confirmed` ghi rõ.

## Phase 1 — Domain

- [x] `config/config.go`: `GenealogyConfig` (`max_per_user`, `max_members`, `max_generation`, `max_spouses`, `max_viewers`, `invitation_ttl`) + `Validate()` (từng giá trị ≥ 1, TTL > 0) + khối `genealogy:` trong `config.dev.yaml` + `cfg.Genealogy.Validate()` ở `cmd/main.go` (test: `TestGenealogyConfig_Validate`, `TestGenealogyConfig_ZeroValueIsInvalid`, `TestLoad_DevYAMLGenealogyConfig`)
- [x] `domain/genealogy.go`: entity `Genealogy`, `Chronicle`, `GenealogyMember` (gồm `PhoneNumber`, `IsClanHead`, `OtherParentID`, `BirthOrder`, `BurialPlace`, `DeathMonth/DeathDay/DeathEnteredAsLunar`), `MemberWarning`, `Role` (`owner`/`viewer`), `Viewer`, `Invitation`/`InvitationStatus`/`InvitationTTL` (30 ngày, đã chốt), `MemberProfile`/`MemberRef`/`Lineage`/`Chi`/`Kinship` (không JSON tag) (giới hạn số và hạn lời mời **không** là hằng ở domain: truyền vào qua `MemberLimits` và tham số `maxPerUser`/`maxViewers`/`ttl`, nguồn là `config.GenealogyConfig`)
- [x] Sentinel errors (gồm `ErrFounderNotMovable`) + `*GenealogyValidationError{Field, Message}` (có `Unwrap`)
- [x] Interface `GenealogyRepository`, `GenealogyMemberRepository` (gồm `Move`, `FindMemberIDByPhone`), `ViewerRepository` (`List` kèm `member_id` theo số điện thoại, `Remove`), `InvitationRepository`, `UserPhoneLookup` (gồm `FindPhoneByID` cho lời mời và cho `my_member_id`)
- [x] `domain/mock_genealogy_repository.go` cho test usecase (năm mock: `MockGenealogyRepository`, `MockGenealogyMemberRepository`, `MockViewerRepository`, `MockInvitationRepository`, `MockUserPhoneLookup`)

## Phase 2 — Usecase (TDD, bảng test, mock — không DB)

- [x] `usecase/genealogy_usecase.go`: `Create`, `List`, `Get`, `UpdateInfo`, `Delete` (owner-only), `UpdateChronicle`; validation AC3–AC9, AC38–AC40, năm theo `Asia/Ho_Chi_Minh` (Step G-1, Step A)
- [x] `usecase/genealogy_member_usecase.go`: `List`, `Get`, `Add` (child/spouse, `other_parent_id`, `birth_order`), `Update`, `Delete`, `Move`; validation AC4–AC8, AC18–AC27, AC30–AC37, AC55–AC58, AC76–AC80; con của nữ dòng họ bị chặn, thuỷ tổ phải nam; `in_law_label` (AC33); `warnings` (AC27) (Step M-1/M-3, D-1, R-1/R-3)
- [x] `usecase/genealogy_access_usecase.go`: `ListViewers`, `Invite(memberID, phone)` (Step C-1), `ListInvitations`, `CancelInvitation`, `RemoveViewer` (owner gỡ người xem, viewer tự rời, owner không tự rời); owner-only → 403 (viewer); chuẩn hoá số điện thoại bằng hàm của register; `Invite` không tra user (Step C-1)
- [x] `usecase/genealogy_invitation_usecase.go`: `ListMine`, `Accept`, `Decline` — không qua Step A, so số điện thoại của người gọi với lời mời (Step I-1); id không phải số dương → 404 không chạm DB
- [x] `usecase/genealogy_profile_usecase.go`: `GetProfile` — lấy số điện thoại người gọi, so khớp `phone_number` để ra `my_member_id` (AC59), cả owner lẫn viewer; không còn `SetMyMember`/`ClearMyMember` (Step A, Step P-1/P-2)
- [x] `usecase/genealogy_lunar.go`: hàm thuần `SolarToLunar` (Hồ Ngọc Đức, UTC+7, `1900..2199`) — bảng test: Tết (mùng 1 tháng 1) nhiều năm, tháng nhuận (ví dụ 2023 nhuận tháng 2), ngày cuối tháng thiếu/đủ, ranh giới năm, ngoài khoảng → `ok=false`; dùng để dựng `chap_lunar_*`/`death_lunar_*` (AC40, AC55)
- [x] `usecase/genealogy_kinship.go`: hàm thuần `BuildIndex`, `Age`, `Relatives`, `Lineage`, `Chi` (AC65–AC68, Step P-3) — bảng test: cây mẫu thuỷ tổ → cụ Lộc/cụ Nghĩa → Lộc tách chi A/B (`chi_path = [Lộc, A]`), chi trưởng/thứ, chuỗi con một không rẽ nhánh, con gái không đầu chi, in-law, nhiều vợ, anh em cùng cha khác mẹ (theo "Quy tắc chi" ở `design.md`)
- [x] `usecase/genealogy_kinship.go`: `Kinship` theo "Bảng xưng hô v1" + "Quy tắc vai vế" (AC69, đã duyệt) — bảng test: ngang vai theo `birth_order` (con bác là anh dù nhỏ tuổi hơn), chú nhỏ tuổi hơn cháu vẫn là chú, thím/dượng/chị dâu/anh rể/con dâu/con rể, hậu tố "họ", ông bác/ông chú, cụ/kỵ/tổ đời N, chắt/chút, bên ngoại → `UNKNOWN`, `birth_year` không ảnh hưởng kết quả
- [x] Unit tests usecase (mock) — `genealogy_validation_test.go`, `genealogy_usecase_test.go`, `genealogy_member_usecase_test.go`, `genealogy_access_usecase_test.go`: validate từng trường (AC3–AC9, AC38–AC40, AC55, AC56, AC82), thuỷ tổ phải nam, con của nữ/in-law bị từ chối, `other_parent_id` phải là vợ/chồng của cha, trần 60 đời, `warnings` (lệch năm, bằng năm không cảnh báo), trưởng tộc (nữ/đã mất/dâu rể → 400; đổi giới tính hay `is_alive` của người đang giữ cờ mà không bỏ cờ → 400; vắng `is_clan_head` giữ cờ; `false` bỏ cờ), đổi giới tính nữ khi đã có con → 400, viewer → `403`/stranger → `404` trên mọi route ghi, id không dương → 404 không chạm DB, `RemoveViewer` (owner không tự rời), inbox/accept/decline dùng số của chính người gọi, lỗi từ repository được truyền nguyên vẹn
- [ ] Phần còn lại của các nhóm test trên thuộc về repository (Phase 3, cần DB thật): số đã/chưa đăng ký cùng một response (AC45), mời ghi số lên nút chưa có số và `409 MEMBER_PHONE_ALREADY_USED`, lời mời tự huỷ khi số của nút đổi/xoá (AC30), dời `generation` cả nhánh khi đổi quan hệ + chống vòng + trần 60 đời + `FOUNDER_NOT_MOVABLE` (AC76–AC79), đặt trưởng tộc gỡ cờ người cũ (AC82), `Accept` song song không vượt trần 20

## Phase 3 — Repository

- [ ] `migrations/0020_genealogy.sql` — năm bảng (`genealogies`, `genealogy_chronicles`, `genealogy_members`, `genealogy_viewers`, `genealogy_invitations`; **không** có `genealogy_user_members` và `clan_head_user_id`; `genealogy_invitations` với CHECK `status`, partial unique `(genealogy_id, phone_number) WHERE status='pending'`, index `(phone_number) WHERE status='pending'`), cột mới của `genealogy_members` (`other_parent_id`, `birth_order`, `burial_place`, `death_month`, `death_day`, `death_entered_as_lunar`), chronicle (`chap_year/month/day` dương lịch + `chap_entered_as_lunar`, `couplet_*_reading/meaning`; **tên cột chap chờ marker AC40 nếu đổi cách tính lại**), FK ghép cùng-gia-phả, `ON DELETE SET NULL (other_parent_id)`, CHECK, partial unique một thuỷ tổ, **partial unique `(genealogy_id, phone_number) WHERE phone_number IS NOT NULL` (AC60)**, **`genealogy_members.is_clan_head` + partial unique `(genealogy_id) WHERE is_clan_head` + CHECK blood/nam/còn sống (AC82)**, **`genealogy_invitations.member_id` + FK ghép ON DELETE CASCADE (AC44)**, index (mục "Schema"); kiểm `ls migrations/` ngay trước khi tạo (số `0019` đã dùng); cập nhật `docs/specs/db/erd.mmd` + `erd.svg`; cập nhật `migrations/README.md`
- [ ] `repository/postgres_genealogy_repository.go`: `Create` (Step G-2, một tx + khoá `users`), `ResolveRole` (Step A, một truy vấn), `ListByUser`, `FindDetail` (+ stats, founder_name, `my_member_id`), `UpdateInfo`, `Delete`, `UpsertChronicle`
- [ ] `repository/postgres_genealogy_member_repository.go`: `List`, `FindByID`, `Add` (Step M-2 gồm 4b/4c, khoá `genealogies` FOR UPDATE), `Update`, `Delete` (Step D-2), `Move` (Step R-2, một tx: khoá `genealogies`, tải cả cây, kiểm trong bộ nhớ, dời `generation` cả nhánh), `FindMemberIDByPhone`; `List` + `FindPhoneByID` dùng được trong một tx `READ ONLY REPEATABLE READ` (Step P-2)
- [ ] `repository/postgres_genealogy_access_repository.go`: `List` (viewers kèm `member_id` theo số điện thoại), `Remove`
- [ ] `repository/postgres_genealogy_invitation_repository.go`: `Create` (Step C-2, một tx, đánh dấu `expired` lười), `ListMine`, `ListByGenealogy`, `Cancel`, `Accept` (Step I-2, một tx gồm `INSERT genealogy_viewers`; khoá `genealogies` trước `genealogy_invitations`), `Decline`
- [ ] Adapter `UserPhoneLookup`: kiểm repo user đã có hàm tìm theo `phone_number` chưa; chưa có thì thêm `FindIDByPhone`, và `FindPhoneByID` (số điện thoại của người gọi, cho lời mời)
- [ ] Repository tests (Postgres thật, giống `postgres_registration_repository_test.go`): tx rollback khi vượt trần, lời mời: `Create` và `Accept` song song không vượt trần 20 và không deadlock, partial unique chặn hai lời mời chờ cho cùng số, lời mời quá hạn được đánh dấu `expired` rồi mời lại được, tạo song song không vượt trần, FK chặn gắn cha/mẹ từ gia phả khác, CHECK `is_alive`, partial unique thuỷ tổ, CASCADE xoá in-law, xoá in-law thì `other_parent_id` của con về `NULL`, `birth_order` tự tăng khi thêm con song song, unique `(genealogy_id, phone_number)` chặn hai nút cùng số trong một gia phả nhưng cho phép ở gia phả khác, `my_member_id` đổi/mất khi sửa hoặc xoá nút có số

## Phase 4 — Delivery

- [ ] DTO + handler `genealogy_handler.go`, `genealogy_member_handler.go`, `genealogy_access_handler.go`, `genealogy_profile_handler.go` (`DecodeJSONBodyStrict`, `user_id` chỉ từ ctx, `items: []`, `null` tường minh)
- [ ] `delivery/http/errors.go`: map các sentinel mới (gồm `FOUNDER_NOT_MOVABLE` 409, `INVITATION_NOT_FOUND` 404, `INVITATION_NOT_PENDING`/`INVITATION_ALREADY_PENDING` 409); thêm helper `403 GENEALOGY_FORBIDDEN` (chưa có)
- [ ] Handler lời mời `genealogy_invitation_handler.go` (`GET /v1/invitations`, `POST .../accept`, `POST .../decline`) và các route của owner (`POST`/`GET`/`DELETE /v1/genealogies/{id}/invitations*`, `GET`/`DELETE /v1/genealogies/{id}/viewers*`) trong `genealogy_access_handler.go`; `POST .../invitations` trả `201 Invitation`
- [ ] Handler tests: success, từng nhánh lỗi, body sai/unknown field, phương thức sai (405)
- [ ] `cmd/main.go`: đăng ký 21 route, bọc `authMiddleware.AuthenticateFunc` từng route; không chạm khối `⚠️ DEV ONLY`
- [ ] `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`): tag Genealogies, 21 route, các schema (`GenealogyDetail`, `Member`, `Chronicle`, `Viewer`, `Invitation`, `MyInvitation`, `MemberProfile`, `MemberRef`), Bearer

## Phase 5 — Integration & documentation

- [ ] E2E (binary thật + Postgres thật): hai user — chủ tạo gia phả, ghi `phone_number` cho nút của người thân rồi mời bằng đúng số đó (cả số đã và chưa đăng ký cho cùng response); người được mời thấy lời mời (kèm tên nút) ở `GET /v1/invitations` rồi chấp nhận, owner thấy trạng thái chuyển `pending` → `accepted` ở `GET .../invitations`, → thành `viewer`, đọc được cây/`profile` và thấy `my_member_id` đúng nút; `viewer` gọi route ghi nhận `403`; từ chối thì không có quyền; user đăng ký sau vẫn thấy lời mời (AC75); user thứ ba nhận 404; thu quyền rồi 404 ngay (AC54); hai nút cùng số trong một gia phả bị `409`; cùng số ở gia phả thứ hai được; `profile` đúng chi/đường từ thuỷ tổ (AC64–AC68); owner thấy `member_id` của viewer ở `GET .../viewers`, viewer không thấy của người khác (AC62)
- [ ] E2E đổi quan hệ: chuyển một nhánh sang cha/mẹ khác rồi `GET members` thấy `generation` mới, `profile` đúng `chi`/`lineage`; chuyển vào hậu duệ → 400; chuyển thuỷ tổ → 409
- [ ] Audit AC ↔ test (skill `golang-e2e-verification`)
- [ ] `go build ./...`, `go vet ./...`, `go test -short ./...` sạch
- [ ] Ghi vào `changelog.md`: concept OKF `docs/knowledge/features/genealogy.md` cần tạo bởi `il-okf-writer` sau khi code được kiểm chứng (không viết trước)
- [ ] Chốt các marker `⚠️ Not confirmed` và chuyển sang "Open questions (resolved)"; cập nhật `docs/assumptions/genealogy/A-001`/`A-002`

## Deferred (separate feature, not blocking genealogy)

- Ảnh hoành phi/thành viên — chờ GCS (`docs/tech-debt/memorial/TD-001`)
- Nhắc giỗ hằng năm (thuộc `memorial`; gia phả không liên kết sang `memorial`, chốt 2026-10-10)
- Thông báo push (FCM/APNs) — tính năng riêng, làm sau, không dùng SMS; vai trò `editor` (bỏ ở v5.0, để sau nếu cần)
- Chống ghi đè đồng thời; đổi loại thành viên dòng họ ↔ dâu/rể (AC80); chuyển quyền owner; audit lịch sử sửa
- Xưng hô bên ngoại (cần dâu/rể có cha mẹ trong cây); route sắp xếp lại thứ tự anh em; liên kết ngày giỗ với `memorial`; "mỗi nút một tài khoản" (unique)
