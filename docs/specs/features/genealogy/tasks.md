# Tasks — genealogy

Version: 5.12
Last updated: 2026-10-11
Status: implemented and documented — code, test, E2E, OKF, OpenAPI và `web/docs` xong; không còn marker nào mở.
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

- [x] `migrations/0028_genealogy.sql` — năm bảng (`genealogies`, `genealogy_chronicles`, `genealogy_members`, `genealogy_viewers`, `genealogy_invitations`; **không** có `genealogy_user_members` và `clan_head_user_id`; `genealogy_invitations` với CHECK `status`, partial unique `(genealogy_id, phone_number) WHERE status='pending'`, index `(phone_number) WHERE status='pending'`), cột mới của `genealogy_members` (`other_parent_id`, `birth_order`, `burial_place`, `death_month`, `death_day`, `death_entered_as_lunar`), chronicle (`chap_year/month/day` dương lịch + `chap_entered_as_lunar`, `couplet_*_reading/meaning`; **tên cột chap chờ marker AC40 nếu đổi cách tính lại**), FK ghép cùng-gia-phả, `ON DELETE SET NULL (other_parent_id)`, CHECK, partial unique một thuỷ tổ, **partial unique `(genealogy_id, phone_number) WHERE phone_number IS NOT NULL` (AC60)**, **`genealogy_members.is_clan_head` + partial unique `(genealogy_id) WHERE is_clan_head` + CHECK blood/nam/còn sống (AC82)**, **`genealogy_invitations.member_id` + FK ghép ON DELETE CASCADE (AC44)**, index (mục "Schema"); số `0028` (đã kiểm `ls migrations/`); đã cập nhật `docs/specs/db/erd.mmd` và `migrations/README.md` (chưa render lại `erd.svg`)
- [x] `repository/postgres_genealogy_repository.go`: `Create` (Step G-2, một tx + khoá `users`), `ResolveRole` (Step A, một truy vấn), `ListByUser`, `FindDetail` (+ stats, founder_name, `my_member_id`), `UpdateInfo`, `Delete`, `UpsertChronicle`
- [x] `repository/postgres_genealogy_member_repository.go`: `List`, `FindByID`, `Add` (Step M-2 gồm 4b/4c, khoá `genealogies` FOR UPDATE), `Update`, `Delete` (Step D-2), `Move` (Step R-2, một tx: khoá `genealogies`, tải cả cây, kiểm trong bộ nhớ, dời `generation` cả nhánh), `FindMemberIDByPhone`; `List` + `FindPhoneByID` dùng được trong một tx `READ ONLY REPEATABLE READ` (Step P-2)
- [x] `repository/postgres_genealogy_access_repository.go` — `genealogyViewerRepository` (`List` kèm `member_id` theo số điện thoại, `Remove`) và `genealogyInvitationRepository` (`Create`, `ListByGenealogy`, `ListMine`, `Cancel`, `Accept`, `Decline`) cùng nằm trong file này; `NewUserPhoneLookup` cũng ở đây
- [x] (đã gộp) invitation repository: xem mục trên — `Create` (Step C-2, một tx, đánh dấu `expired` lười), `Accept` (Step I-2, khoá `genealogies` trước `genealogy_invitations`)
- [x] Adapter `UserPhoneLookup`: không có sẵn hàm tìm theo id nên thêm `NewUserPhoneLookup` (`SELECT phone_number FROM users WHERE id = $1`; không có hàng → `domain.ErrUserNotFound`)
- [x] Repository tests (Postgres thật, `postgres_genealogy_repository_test.go`): tạo nguyên tử (lỗi giữa chừng không để lại cây/phả ký/thành viên), tạo song song không vượt trần, thêm song song không vượt trần thành viên và không trùng `birth_order`, `Accept` song song không vượt trần và không deadlock, FK ghép chặn quan hệ chéo cây, unique số điện thoại (trùng trong cây bị chặn, khác cây được), đặt trưởng tộc gỡ cờ người cũ, xoá in-law thì `other_parent_id` của con về `NULL`, xoá lá kèm in-law, dời cả nhánh dịch `generation` của hậu duệ và dâu/rể, chống vòng, trần 60 đời, `FOUNDER_NOT_MOVABLE`, in-law chuyển người được gắn, hộp lời mời, lời mời hết hạn không chấp nhận được và không chặn lời mời mới, xoá số của nút huỷ lời mời chờ

## Phase 4 — Delivery

- [x] DTO + handler: `genealogy_handler.go` (cây + phả ký), `genealogy_member_handler.go` (thành viên + hồ sơ), `genealogy_access_handler.go` (lời mời của owner, người xem, và hộp lời mời của người được mời) — `DecodeJSONBodyStrict`, `user_id` chỉ từ ctx, `items: []`, `null` tường minh, id sai định dạng đi xuống use case thành 0 (404 sau khi đã resolve vai trò)
- [x] `delivery/http/errors.go`: map `GenealogyValidationError` → `VALIDATION_ERROR` + `details.field`, và 15 sentinel (`GENEALOGY_NOT_FOUND`/`GENEALOGY_FORBIDDEN` 403/`GENEALOGY_LIMIT_REACHED`, `MEMBER_*`, `FOUNDER_NOT_DELETABLE`/`FOUNDER_NOT_MOVABLE`, `VIEWER_*`, `INVITATION_*`, `MEMBER_PHONE_ALREADY_USED` kèm `details.field = phone_number`)
- [x] (đã gộp vào mục handler ở trên) route lời mời: `GET /v1/invitations`, `POST .../accept`, `POST .../decline`; route của owner: `POST`/`GET`/`DELETE /v1/genealogies/{id}/invitations*`, `GET`/`DELETE /v1/genealogies/{id}/viewers*`; `POST .../invitations` trả `201 Invitation`
- [x] Handler tests (`genealogy_handler_test.go`, mock usecase): success, từng mã lỗi, body sai/unknown field/field do server giữ, không có token → 401, id sai định dạng, danh sách không bao giờ null, hộp lời mời không lộ gì khác của cây, `my_member_id`/`*_lunar_*` có trong response, `warnings` chỉ ở ghi, phương thức sai → 405
- [x] `cmd/main.go`: khởi tạo năm repository + năm usecase + bốn handler, đăng ký 21 route bọc `authMiddleware.AuthenticateFunc` từng route; không chạm khối `⚠️ DEV ONLY`. **Chưa kiểm bằng E2E** (Phase 5): wiring mới được kiểm bằng biên dịch và test handler
- [x] `docs/specs/api/openapi.yaml`: ba tag (`Genealogies`, `GenealogyMembers`, `GenealogyAccess`), 21 operation, 15 schema + envelope, 5 parameter, Bearer; đã kiểm YAML hợp lệ, không trùng `operationId`, mọi `$ref` đều có đích. `web/docs/public/openapi.yaml` là bản sinh (gitignore), `sync-openapi.mjs` chép khi build nên không sửa tay

## Phase 5 — Integration & documentation

- [x] E2E (binary thật + Postgres thật, `e2e/genealogy_e2e_test.go`): wiring 21 route (không token / token rác / refresh token → 401); tạo → xem → đổi tên → phả ký có ngày âm → xoá; thêm con/vợ chồng, cảnh báo lệch năm, con của nữ bị chặn, xoá lá/có con/thuỷ tổ, dời nhánh, vòng, trưởng tộc, ngày giỗ có ngày âm, trùng số; profile `Bác` dù sinh sau; mời → hộp lời mời → chấp nhận → viewer đứng đúng nút → viewer bị `403` ở 10 route ghi → owner thấy `accepted` → xoá số mất vị trí → gỡ có hiệu lực ngay; người lạ nhận 404 **giống hệt từng byte** với id không tồn tại; từ chối/huỷ/mời lại/tự rời; người đã mất không mời được; trần và hạn lời mời lấy từ config chạy thật (`max_per_user`, `max_members`, `max_viewers`, `invitation_ttl`)
- [x] (đã gộp vào mục E2E ở trên) đổi quan hệ: dời nhánh → `generation` mới, vòng → 400, thuỷ tổ → 409
- [x] Audit AC ↔ test (skill `golang-e2e-verification`): xem `ac-audit.md` — AC1–AC82, không còn AC nào chưa có test; ba điểm chứng minh yếu hơn đã nêu rõ lý do
- [x] `go build ./...`, `go vet ./...`, `go test -short ./...` sạch (toàn repo, 2026-10-11); `go test ./services/appService/repository/` và `./services/appService/e2e/ -run TestGenealogy` chạy trên Postgres/Docker thật cũng pass
- [x] Ghi vào `changelog.md`: concept OKF `docs/knowledge/features/genealogy.md` — luồng đã chạy end-to-end nên được tạo bằng `il-okf-writer` (v5.8)
- [x] Chốt toàn bộ marker `⚠️ Not confirmed`: 7 marker ngày 2026-10-11 theo anh Đức, AC65 chốt hiển thị cả tuổi tây và tuổi mụ; đã thêm `TraditionalAge` + trường `traditional_age` (test: `TestTraditionalAge`, `TestProfileUsecase`, `TestGenealogyMemberHandler/profile…`)
- [x] Tài liệu hạ nguồn: `docs/knowledge/features/genealogy.md` (OKF, `draft`, đã vào `index.md`/`log.md`); `web/docs` — trang `guides/catalogues/genealogies.md`, sidebar "Gia phả", bảng mã lỗi, changelog (`npm run build` dựng 72 trang, có trang mới và trang OKF); `docs/specs/db/erd.svg` đã phản ánh `0028` (dựng từ `erd.mmd`)

## Deferred (separate feature, not blocking genealogy)

- Ảnh hoành phi/thành viên — chờ GCS (`docs/tech-debt/memorial/TD-001`)
- Nhắc giỗ hằng năm (thuộc `memorial`; gia phả không liên kết sang `memorial`, chốt 2026-10-10)
- Thông báo push (FCM/APNs) — tính năng riêng, làm sau, không dùng SMS; vai trò `editor` (bỏ ở v5.0, để sau nếu cần)
- Chống ghi đè đồng thời; đổi loại thành viên dòng họ ↔ dâu/rể (AC80); chuyển quyền owner; audit lịch sử sửa
- Xưng hô bên ngoại (cần dâu/rể có cha mẹ trong cây); route sắp xếp lại thứ tự anh em; liên kết ngày giỗ với `memorial`; "mỗi nút một tài khoản" (unique)
