# Tasks — genealogy

Version: 3.0
Last updated: 2026-10-08
Status: not started — chưa có code; mọi mục đều `[ ]`.
Module: `genealogy`

> See `changelog.md` in this folder for the version history of this spec.

Mỗi mục trace về một dòng của bảng "Layers & components" hoặc một Step trong `design.md`. Làm theo TDD (skill `golang-clean-arch-tdd`), kiểm E2E cuối (skill `golang-e2e-verification`). Các mục bị chặn bởi marker `⚠️ Not confirmed` ghi rõ.

## Phase 1 — Domain

- [ ] `domain/genealogy.go`: entity `Genealogy`, `Chronicle`, `GenealogyMember` (gồm `OtherParentID`, `BirthOrder`, `BurialPlace`, `DeathLunar*`), `Role`, `Collaborator`, `Invitation`/`InvitationStatus`/`InvitationTTL` (⚠️ AC75), `MemberProfile`/`MemberRef`/`Lineage`/`Chi`/`Kinship` (không JSON tag) + hằng giới hạn (⚠️ giá trị đề xuất, AC10/24/25/47)
- [ ] Sentinel errors + `*GenealogyValidationError{Field, Message}` (có `Unwrap`)
- [ ] Interface `GenealogyRepository`, `GenealogyMemberRepository` (gồm `SetMyMember`, `ClearMyMember`, `FindMyMemberID`), `CollaboratorRepository` (`List`, `Remove`), `InvitationRepository`, `UserPhoneLookup` (gồm `FindIDByPhone` cho `SetClanHead`, `FindPhoneByID` cho lời mời)
- [ ] `domain/mock_genealogy_repository.go` cho test usecase

## Phase 2 — Usecase (TDD, bảng test, mock — không DB)

- [ ] `usecase/genealogy_usecase.go`: `Create`, `List`, `Get`, `UpdateInfo`, `Delete` (owner-only), `UpdateChronicle`; validation AC3–AC9, AC38–AC40, năm theo `Asia/Ho_Chi_Minh` (Step G-1, Step A)
- [ ] `usecase/genealogy_member_usecase.go`: `List`, `Get`, `Add` (child/spouse, `other_parent_id`, `birth_order`), `Update`, `Delete`; validation AC4–AC8, AC18–AC27, AC30–AC37, AC55–AC58; `in_law_label` (AC33) (Step M-1/M-3, D-1)
- [ ] `usecase/genealogy_access_usecase.go`: `ListCollaborators`, `Invite`, `ListInvitations`, `CancelInvitation`, `Revoke`, `Leave`, `SetClanHead`, `ClearClanHead`; owner-only → 403; chuẩn hoá số điện thoại bằng hàm của register; `Invite` không tra user (Step C-1)
- [ ] `usecase/genealogy_invitation_usecase.go`: `ListMine`, `Accept`, `Decline` — không qua Step A, so số điện thoại của người gọi với lời mời (Step I-1); id không phải số dương → 404 không chạm DB
- [ ] `usecase/genealogy_profile_usecase.go`: `GetProfile`, `SetMyMember`, `ClearMyMember`; validation AC59–AC61 (Step A, Step P-1/P-2)
- [ ] `usecase/genealogy_kinship.go`: hàm thuần `BuildIndex`, `Age`, `Relatives`, `Lineage`, `Chi` (AC65–AC68, Step P-3) — bảng test: cây mẫu thuỷ tổ → cụ Lộc/cụ Nghĩa → Lộc tách chi A/B (`chi_path = [Lộc, A]`), chi trưởng/thứ, chuỗi con một không rẽ nhánh, con gái không đầu chi, in-law, nhiều vợ, anh em cùng cha khác mẹ (theo "Quy tắc chi" ở `design.md`)
- [ ] `usecase/genealogy_kinship.go`: `Kinship` theo "Bảng xưng hô v1" — **chặn bởi marker AC69** (phải duyệt bảng xưng hô trước khi viết bảng test); phần còn lại của `profile` không bị chặn, `relation_to_me` tạm trả `UNKNOWN`
- [ ] Unit tests lời mời: số đã/chưa đăng ký cho cùng một response (AC45), 409 vai trò / lời mời trùng / trần 20 gồm lời mời chờ, `Accept`/`Decline` sai số điện thoại → `404 INVITATION_NOT_FOUND`, lời mời hết hạn / đã trả lời / đã huỷ → `409 INVITATION_NOT_PENDING`, `Accept` khi đã có vai trò, `ListMine` chỉ trả lời mời còn hạn
- [ ] Unit tests: ma trận vai trò × hành động (bảng ở `design.md` Overview), `403` vs `404`, mỗi nhánh validation, spouse mặc định giới tính, con của in-law bị từ chối, `other_parent_id` phải là vợ/chồng của cha, `birth_order` mặc định

## Phase 3 — Repository

- [ ] `migrations/0020_genealogy.sql` — sáu bảng (thêm `genealogy_user_members`, `genealogy_invitations` với CHECK `status`, partial unique `(genealogy_id, phone_number) WHERE status='pending'`, index `(phone_number) WHERE status='pending'`), cột mới của `genealogy_members` (`other_parent_id`, `birth_order`, `burial_place`, `death_lunar_*`), FK ghép cùng-gia-phả, `ON DELETE SET NULL (other_parent_id)`, CHECK, partial unique một thuỷ tổ, index (mục "Schema"); kiểm `ls migrations/` ngay trước khi tạo (số `0019` đã dùng); cập nhật `docs/specs/db/erd.mmd` + `erd.svg`; cập nhật `migrations/README.md`
- [ ] `repository/postgres_genealogy_repository.go`: `Create` (Step G-2, một tx + khoá `users`), `ResolveRole` (Step A, một truy vấn), `ListByUser`, `FindDetail` (+ stats, founder_name, `my_member_id`), `UpdateInfo`, `Delete`, `UpsertChronicle`
- [ ] `repository/postgres_genealogy_member_repository.go`: `List`, `FindByID`, `Add` (Step M-2 gồm 4b/4c, khoá `genealogies` FOR UPDATE), `Update`, `Delete` (Step D-2), `SetMyMember`/`ClearMyMember`/`FindMyMemberID`; `List` + `FindMyMemberID` dùng được trong một tx `READ ONLY REPEATABLE READ` (Step P-2)
- [ ] `repository/postgres_genealogy_access_repository.go`: `List`, `Remove`, `SetClanHead` (xoá editor row cùng tx, AC51; xoá hàng `genealogy_user_members` của user mất vai trò cùng tx, AC63)
- [ ] `repository/postgres_genealogy_invitation_repository.go`: `Create` (Step C-2, một tx, đánh dấu `expired` lười), `ListMine`, `ListByGenealogy`, `Cancel`, `Accept` (Step I-2, một tx gồm `INSERT genealogy_collaborators`; khoá `genealogies` trước `genealogy_invitations`), `Decline`
- [ ] Adapter `UserPhoneLookup`: kiểm repo user đã có hàm tìm theo `phone_number` chưa; chưa có thì thêm `FindIDByPhone`, và `FindPhoneByID` (số điện thoại của người gọi, cho lời mời)
- [ ] Repository tests (Postgres thật, giống `postgres_registration_repository_test.go`): tx rollback khi vượt trần, lời mời: `Create` và `Accept` song song không vượt trần 20 và không deadlock, partial unique chặn hai lời mời chờ cho cùng số, lời mời quá hạn được đánh dấu `expired` rồi mời lại được, tạo song song không vượt trần, FK chặn gắn cha/mẹ từ gia phả khác, CHECK `is_alive`, partial unique thuỷ tổ, CASCADE xoá in-law, xoá in-law thì `other_parent_id` của con về `NULL`, `birth_order` tự tăng khi thêm con song song, con trỏ "tôi là ai" bị xoá khi xoá thành viên / thu quyền

## Phase 4 — Delivery

- [ ] DTO + handler `genealogy_handler.go`, `genealogy_member_handler.go`, `genealogy_access_handler.go`, `genealogy_profile_handler.go` (`DecodeJSONBodyStrict`, `user_id` chỉ từ ctx, `items: []`, `null` tường minh)
- [ ] `delivery/http/errors.go`: map các sentinel mới (gồm `INVITATION_NOT_FOUND` 404, `INVITATION_NOT_PENDING`/`INVITATION_ALREADY_PENDING` 409); thêm helper `403 GENEALOGY_FORBIDDEN` (chưa có)
- [ ] Handler lời mời `genealogy_invitation_handler.go` (`GET /v1/invitations`, `POST .../accept`, `POST .../decline`) và hai route của owner (`GET`/`DELETE /v1/genealogies/{id}/invitations*`) trong `genealogy_access_handler.go`; `POST .../collaborators` trả `201 Invitation` (marker AC45 đã chốt ở v3.0 — không còn chặn)
- [ ] Handler tests: success, từng nhánh lỗi, body sai/unknown field, phương thức sai (405)
- [ ] `cmd/main.go`: đăng ký 24 route, bọc `authMiddleware.AuthenticateFunc` từng route; không chạm khối `⚠️ DEV ONLY`
- [ ] `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`): tag Genealogies, 24 route, các schema (`GenealogyDetail`, `Member`, `Chronicle`, `Collaborator`, `Invitation`, `MyInvitation`, `MemberProfile`, `MemberRef`), Bearer

## Phase 5 — Integration & documentation

- [ ] E2E (binary thật + Postgres thật): hai user — chủ tạo, mời bằng số điện thoại (cả số đã và chưa đăng ký cho cùng response), người được mời thấy lời mời ở `GET /v1/invitations` rồi chấp nhận → thành editor, editor sửa được; từ chối thì không có quyền; user đăng ký sau vẫn thấy lời mời (AC75); user thứ ba nhận 404, editor gọi route owner-only nhận 403, thu quyền rồi 404 ngay (AC54); chọn "tôi là ai" rồi `profile` đúng chi/đường từ thuỷ tổ (AC59–AC68), con trỏ của user khác không lộ (AC62), thu quyền xoá con trỏ (AC63)
- [ ] Audit AC ↔ test (skill `golang-e2e-verification`)
- [ ] `go build ./...`, `go vet ./...`, `go test -short ./...` sạch
- [ ] Ghi vào `changelog.md`: concept OKF `docs/knowledge/features/genealogy.md` cần tạo bởi `il-okf-writer` sau khi code được kiểm chứng (không viết trước)
- [ ] Chốt các marker `⚠️ Not confirmed` và chuyển sang "Open questions (resolved)"; cập nhật `docs/assumptions/genealogy/A-001`/`A-002`

## Deferred (separate feature, not blocking genealogy)

- Ảnh hoành phi/thành viên — chờ GCS (`docs/tech-debt/memorial/TD-001`)
- Liên kết thành viên gia phả với mục `memorial`; nhắc giỗ
- Push notification/SMS báo lời mời (chờ trả lời AC70); chỉ định trưởng tộc qua lời mời (chờ trả lời AC51)
- Route đổi cha/mẹ; chống ghi đè đồng thời; chuyển quyền owner; audit lịch sử sửa
- Xưng hô bên ngoại (cần dâu/rể có cha mẹ trong cây); route sắp xếp lại thứ tự anh em; liên kết ngày giỗ với `memorial`; "mỗi nút một tài khoản" (unique)
