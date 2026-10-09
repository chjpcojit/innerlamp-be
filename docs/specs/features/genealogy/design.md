# Design — genealogy

Version: 3.0
Last updated: 2026-10-08
Status: not started — thiết kế dự kiến, chưa có code. Marker `⚠️ Not confirmed` còn mở ở `requirements.md`; Step G-1 (trần số gia phả), M-1 (trần thành viên/số đời), C-2 (trần 20 có tính lời mời chờ), I-1 (thời hạn lời mời), bảng xưng hô v1 (Step P-3) bị ảnh hưởng.
Module: `genealogy`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Hai mươi bốn route đều **user-scoped** — mười chín route `/v1/genealogies/*` cộng năm route lời mời (hai route `/v1/genealogies/{id}/invitations*` của owner, ba route `/v1/invitations*` của người được mời) —, bọc `authMiddleware.AuthenticateFunc` (`RequireTokenType: "access"`) như `POST /v1/auth/logout` và `/v1/memorials`. Khác `memorial`, quyền không chỉ là "chủ dòng": mỗi route đầu tiên **phân giải vai trò** của user trong gia phả (Step A) rồi mới chạy nghiệp vụ. Chạm đủ bốn layer (domain/usecase/repository/delivery), sở hữu migration `0020_genealogy.sql` (sáu bảng, gồm `genealogy_invitations`; `0019` đã bị `0019_sutra_audio_search.sql` dùng). Hình dạng gần nhất: `memorial` (404 đồng nhất, trần số lượng kiểm trong transaction có khoá) cộng lớp phân quyền.

Từ v2.0, thông tin hiển thị khi chọn một người (tuổi, họ hàng, đường từ thuỷ tổ, chi, xưng hô với "tôi") **tính ở server** trong một hàm thuần của usecase (`genealogy_kinship.go`) — tải cả cây (≤ 5000 hàng) một lần rồi tính trong bộ nhớ (Step P), không đệ quy SQL.

Từ v3.0, cấp quyền `editor` **luôn đi qua lời mời** (Step C, Step I): owner tạo lời mời theo số điện thoại (đã hoặc chưa có tài khoản — cùng một response, không lộ số đã đăng ký); người được mời xem lời mời ở hộp lời mời `GET /v1/invitations` rồi chấp nhận/từ chối. Lời mời khớp người **chỉ bằng số điện thoại** nên không đổi luồng register. Ba route `/v1/invitations*` không thuộc một gia phả mà người gọi chưa có vai trò, nên **không qua Step A** — kiểm bằng số điện thoại của chính người gọi.

Vai trò và quyền:

| Hành động | owner | clan_head | editor | không có vai trò |
|---|---|---|---|---|
| Xem gia phả, cây, phả ký | ✅ | ✅ | ✅ | 404 |
| Sửa tên/quê gốc, phả ký, thành viên (thêm/sửa/xoá) | ✅ | ✅ | ✅ | 404 |
| Xoá gia phả, mời/huỷ lời mời/thu quyền, chỉ định trưởng tộc | ✅ | 403 | 403 | 404 |
| Tự rời gia phả | ❌ | ❌ | ✅ | 404 |
| Chọn/bỏ "tôi là ai", xem hồ sơ thành viên (`profile`) | ✅ | ✅ | ✅ | 404 |

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/genealogy.go` | `Genealogy` (id, ownerUserID, name, ancestralHome, `ClanHeadUserID *int64`), `Chronicle` (các field AC39, con trỏ cho field tuỳ chọn), `GenealogyMember` (kind, gender, `BirthYear/DeathYear *int`, `IsAlive`, `ParentID/OtherParentID/SpouseOfID *int64`, `BirthOrder *int`, `BurialPlace *string`, `DeathLunarDay/DeathLunarMonth *int`, `DeathLunarLeap *bool`, `Generation`...), `Role` (`owner`/`clan_head`/`editor`), `Collaborator`, `MemberProfile`/`MemberRef`/`Lineage`/`Chi`/`Kinship` (kết quả tính ra). Chỉ field nghiệp vụ, không JSON tag. Hằng giới hạn (`MaxGenealogiesPerUser=10`, `MaxMembersPerGenealogy=5000`, `MaxGeneration=60`, `MaxSpousesPerMember=5`, `MaxEditors=20`) — ⚠️ giá trị đề xuất. |
| Domain / Errors | `domain/genealogy.go` | `ErrGenealogyNotFound`, `ErrGenealogyForbidden`, `ErrGenealogyLimitReached`, `ErrMemberNotFound`, `ErrMemberLimitReached`, `ErrMemberHasDescendants`, `ErrFounderNotDeletable`, `ErrUserNotFound`, `ErrCollaboratorExists`, `ErrCollaboratorLimitReached`, `ErrCollaboratorNotFound`; `ErrGenealogyInvalid` + `*GenealogyValidationError{Field, Message}` (theo mẫu `*MemorialValidationError`). Handler đọc `Field` để trả `details.field`. |
| Domain / Repository interface | `domain/genealogy.go` | `GenealogyRepository`: `Create(ctx, g, c, founder, spouses, maxPerUser)` (một tx), `ListByUser`, `ResolveRole(ctx, genealogyID, userID)` (trả `Role` hoặc `ErrGenealogyNotFound`), `FindDetail`, `UpdateInfo`, `Delete`, `UpsertChronicle`. `GenealogyMemberRepository`: `List`, `FindByID(ctx, genealogyID, memberID)`, `Add(ctx, genealogyID, relation, relatedID, otherParentID *int64, m, limits)` (một tx), `Update`, `Delete(ctx, genealogyID, memberID)` (một tx), `SetMyMember(ctx, genealogyID, userID, memberID)` (upsert; `ErrMemberNotFound` nếu không thuộc gia phả), `ClearMyMember`, `FindMyMemberID(ctx, genealogyID, userID) (*int64, error)`. `CollaboratorRepository`: `List`, `Add(ctx, genealogyID, userID, grantedBy, max)`, `Remove`, `SetClanHead(ctx, genealogyID, userID *int64)` (một tx); `Remove` và `SetClanHead` xoá luôn hàng `genealogy_user_members` của user mất vai trò **trong cùng tx** (AC63). `FindDetail(ctx, id, userID)` trả thêm `my_member_id`. Định nghĩa ở domain, không ở `repository/`. |
| Domain / Invitation | `domain/genealogy.go` (hoặc `domain/genealogy_invitation.go` nếu file vượt ~500 dòng) | Entity `Invitation` (id, genealogyID, phoneNumber, invitedBy, status, createdAt, expiresAt, respondedAt, không JSON tag), `InvitationStatus` (`pending`/`accepted`/`declined`/`cancelled`/`expired`), hằng `InvitationTTL = 30 ngày` (⚠️ AC75). Sentinel: `ErrInvitationNotFound`, `ErrInvitationNotPending`, `ErrInvitationAlreadyPending`. Interface `InvitationRepository`. Mở rộng `UserPhoneLookup` thêm `FindPhoneByID(ctx, userID)`. |
| Domain / User lookup port | `domain/genealogy.go` | `UserPhoneLookup.FindIDByPhone(ctx, normalizedPhone) (int64, error)` — port tối thiểu để usecase không import repo user cụ thể; ưu tiên tái dùng hàm có sẵn nếu repo user đã có (xem Ghi chú của `requirements.md`). |
| Domain / Mock | `domain/mock_genealogy_repository.go` | Mock viết tay cho ba repository + `UserPhoneLookup`, dùng cho test usecase. |
| Usecase | `services/appService/usecase/genealogy_usecase.go` | `Create`, `List`, `Get`, `UpdateInfo`, `Delete`, `UpdateChronicle`. Validation AC3–AC9, AC38–AC40; `time.LoadLocation("Asia/Ho_Chi_Minh")` cho "năm hiện tại"; mặc định giới tính phối ngẫu (AC9); tính `stats`/`founder_name` từ repo. Mọi hàm (trừ `Create`, `List`) mở đầu bằng `ResolveRole`. |
| Usecase | `usecase/genealogy_member_usecase.go` | `List`, `Get`, `Add`, `Update`, `Delete`. Validation AC4–AC8, AC18–AC27, AC30–AC37; tính `in_law_label`. Phân biệt `child`/`spouse`. |
| Usecase | `usecase/genealogy_access_usecase.go` | `ListCollaborators`, `Invite`, `ListInvitations`, `CancelInvitation`, `Revoke`, `Leave`, `SetClanHead`, `ClearClanHead`. Chuẩn hoá số điện thoại (tái dùng hàm của register), kiểm owner-only (403). `Invite` **không** tra user (Step C-1); `SetClanHead` vẫn tra user qua `UserPhoneLookup.FindIDByPhone` (AC51). |
| Usecase | `usecase/genealogy_invitation_usecase.go` | `ListMine`, `Accept`, `Decline` (AC70–AC72). **Không** qua Step A: lấy số điện thoại của người gọi bằng `UserPhoneLookup.FindPhoneByID`, rồi so với `phone_number` của lời mời. Id không phải số dương → `404 INVITATION_NOT_FOUND` không chạm DB. |
| Usecase | `usecase/genealogy_profile_usecase.go` | `GetProfile`, `SetMyMember`, `ClearMyMember`. Mở đầu bằng `ResolveRole` (Step A); `GetProfile` chạy Step P: tải cả cây + con trỏ rồi gọi các hàm thuần ở `genealogy_kinship.go`. Validation AC59–AC61. |
| Usecase | `usecase/genealogy_kinship.go` | **Hàm thuần, không I/O** (TDD bảng test): `BuildIndex(members)` (map id→member, con theo cha), `Age(member, now)` (AC65), `Relatives(index, id)` (AC66), `Lineage(index, id)` (AC67), `Chi(index, id)` (AC68), `Kinship(index, meID, targetID)` (AC69, theo "Bảng xưng hô v1"). Mọi sắp xếp theo `(birth_order, id)`; `now` truyền vào để test không phụ thuộc đồng hồ. |
| Repository | `services/appService/repository/postgres_genealogy_repository.go` | SQL sáu bảng; `Create` và `Add`/`Delete` member/`SetClanHead` chạy một tx; `ResolveRole` là **một** truy vấn; mọi truy vấn member có `WHERE genealogy_id = $1`. `NULL` ↔ `nil`. Map vi phạm unique/FK/CHECK → sentinel error. Sửa tách file `postgres_genealogy_member_repository.go`, `postgres_genealogy_access_repository.go` nếu file vượt ~500 dòng. |
| Repository | `services/appService/repository/postgres_genealogy_invitation_repository.go` | `Create` (Step C-2, một tx), `ListMine`, `ListByGenealogy`, `Cancel`, `Accept` (Step I-2, một tx — `INSERT genealogy_collaborators` nằm trong tx này, nên không còn `CollaboratorRepository.Add` riêng), `Decline`. Thứ tự khoá luôn **`genealogies` trước, `genealogy_invitations` sau** để `Create` và `Accept` không deadlock. |
| Delivery / Handler | `services/appService/delivery/http/genealogy_handler.go` (+ `genealogy_member_handler.go`, `genealogy_access_handler.go`, `genealogy_profile_handler.go`, `genealogy_invitation_handler.go`) | Parse path/body → gọi usecase → map sentinel sang HTTP. `DecodeJSONBodyStrict`. `user_id` chỉ từ context của middleware. Không logic nghiệp vụ. |
| Delivery / DTO | cùng các file handler | DTO request/response (JSON tag chỉ ở đây). Trả `null` tường minh; `items: []` khi rỗng. |
| Delivery / Errors | `services/appService/delivery/http/errors.go` | Thêm map: `GENEALOGY_NOT_FOUND`/`MEMBER_NOT_FOUND`/`USER_NOT_FOUND`/`COLLABORATOR_NOT_FOUND`/`INVITATION_NOT_FOUND` → 404, `GENEALOGY_FORBIDDEN` → 403 (chưa có mã 403 nào trong file — thêm helper mới), các `*_LIMIT_REACHED`/`MEMBER_HAS_DESCENDANTS`/`FOUNDER_NOT_DELETABLE`/`COLLABORATOR_ALREADY_EXISTS`/`INVITATION_ALREADY_PENDING`/`INVITATION_NOT_PENDING` → 409. |
| Wiring | `services/appService/cmd/main.go` | Đăng ký 24 route, **bọc `authMiddleware.AuthenticateFunc`** từng route (verify bằng E2E, không chỉ tài liệu). Router hiện dùng `ServeMux` kiểu nào thì theo đó cho path có `{id}`/`{memberId}`/`{userId}` (xem cách `memorial` đăng ký). Không chạm khối `⚠️ DEV ONLY`. |
| Migration | `migrations/0020_genealogy.sql` | Sáu bảng, CHECK, index (mục "Schema" bên dưới). Số `0020` vì `0019` đã dùng — kiểm lại bằng `ls migrations/` ngay trước khi tạo file. Mirror vào `docs/specs/db/erd.mmd` + `erd.svg`. |
| API spec | `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) | Thêm tag Genealogies, 24 route, các schema (`GenealogyDetail`, `Member`, `Chronicle`, `Collaborator`, `Invitation`, `MyInvitation`, `MemberProfile`, `MemberRef`), bảo mật Bearer. |
| E2E | `services/appService/e2e/genealogy_e2e_test.go` | Binary thật + Postgres thật: tạo → thêm → phân quyền hai user → 404/403 → chọn "tôi là ai" → `profile` (xưng hô, chi, đường từ thuỷ tổ). |

## Schema (`migrations/0020_genealogy.sql`)

```
genealogies            id BIGSERIAL PK, owner_user_id → users(id) ON DELETE RESTRICT,
                       clan_head_user_id → users(id) ON DELETE SET NULL,
                       name VARCHAR(100), ancestral_home VARCHAR(200), created_at, updated_at

genealogy_chronicles   genealogy_id PK → genealogies(id) ON DELETE CASCADE,
                       surname, current_location, since_year, origin, history,
                       hall_name, hall_address, chap_day, chap_month, chap_leap_month,
                       hoanh_phi, hoanh_phi_reading, hoanh_phi_meaning,
                       couplet_upper, couplet_lower, teaching, updated_at
                       CHECK ((chap_day IS NULL) = (chap_month IS NULL))

genealogy_members      id BIGSERIAL PK, genealogy_id → genealogies(id) ON DELETE CASCADE,
                       kind ('blood'|'in_law'), full_name, gender ('male'|'female'),
                       birth_year, is_alive, death_year, residence, dharma_name,
                       tablet_location, burial_place,
                       death_lunar_day SMALLINT, death_lunar_month SMALLINT, death_lunar_leap BOOLEAN,
                       phone_number, note,
                       parent_id, other_parent_id, spouse_of_id, birth_order SMALLINT,
                       generation SMALLINT, created_at, updated_at
                       UNIQUE (id, genealogy_id)
                       FK (parent_id, genealogy_id)    → members(id, genealogy_id) ON DELETE RESTRICT
                       FK (spouse_of_id, genealogy_id) → members(id, genealogy_id) ON DELETE CASCADE
                       FK (other_parent_id, genealogy_id) → members(id, genealogy_id)
                          ON DELETE SET NULL (other_parent_id)   -- cột con trỏ riêng, PG ≥ 15 (AC58)
                       CHECK other_parent_id IS NULL OR kind='blood'
                       CHECK kind='in_law' ⇒ birth_order IS NULL;  parent_id IS NULL ⇒ birth_order IS NULL
                       CHECK is_alive ⇒ burial_place, death_lunar_* đều NULL
                       CHECK (death_lunar_day IS NULL) = (death_lunar_month IS NULL);
                             death_lunar_day 1..30, death_lunar_month 1..12, birth_order 1..1000
                       CHECK death_lunar_leap IS NOT NULL ⇒ death_lunar_day IS NOT NULL
                       CHECK kind='blood'  ⇒ spouse_of_id IS NULL
                       CHECK kind='in_law' ⇒ spouse_of_id IS NOT NULL AND parent_id IS NULL
                       CHECK is_alive ⇒ death_year IS NULL AND tablet_location IS NULL
                       CHECK death_year >= birth_year (khi cả hai có)
                       UNIQUE INDEX (genealogy_id) WHERE kind='blood' AND parent_id IS NULL  -- đúng một thuỷ tổ
                       INDEX (genealogy_id, generation, id), INDEX (parent_id), INDEX (spouse_of_id)

genealogy_collaborators  PK (genealogy_id, user_id), genealogy_id → genealogies ON DELETE CASCADE,
                       user_id → users(id) ON DELETE CASCADE, granted_by → users(id), created_at
                       INDEX (user_id)  -- cho GET /v1/genealogies của một user

genealogy_invitations  id BIGSERIAL PK, genealogy_id → genealogies ON DELETE CASCADE,
                       phone_number VARCHAR(20) NOT NULL  -- đã chuẩn hoá như register; KHÔNG có user_id người được mời (AC75)
                       invited_by → users(id) ON DELETE CASCADE,
                       status VARCHAR(10) NOT NULL DEFAULT 'pending'
                         CHECK status IN ('pending','accepted','declined','cancelled','expired'),
                       created_at, expires_at NOT NULL (= created_at + 30 ngày), responded_at NULL
                       CHECK (status = 'pending') = (responded_at IS NULL)  -- 'expired' do Create đánh dấu lười (Step C-2) nên có responded_at
                       UNIQUE INDEX (genealogy_id, phone_number) WHERE status='pending'  -- AC46: một lời mời chờ cho mỗi số
                       INDEX (phone_number) WHERE status='pending'  -- cho GET /v1/invitations
                       INDEX (genealogy_id, created_at DESC)  -- cho GET .../invitations của owner

genealogy_user_members PK (genealogy_id, user_id), genealogy_id → genealogies ON DELETE CASCADE,
                       user_id → users(id) ON DELETE CASCADE,
                       member_id NOT NULL,
                       FK (member_id, genealogy_id) → members(id, genealogy_id) ON DELETE CASCADE  -- AC63
                       updated_at
                       -- không unique member_id: con trỏ riêng tư, nhiều user có thể chọn cùng một nút
```

> Ghi chú `other_parent_id`: việc "in-law này là vợ/chồng của `parent_id`" (`spouse_of_id = parent_id`) **không** ép được bằng CHECK/FK một hàng; usecase/repository kiểm trong tx (Step M-2) — AC57. `other_parent_id` chỉ gán lúc tạo, không có đường đổi (AC31) nên không lệch về sau, trừ khi in-law bị xoá (đã `SET NULL`).

## Request flow

> Xem `flow.md` cho sơ đồ sequence Mermaid (mirror 1:1 của sáu Step-by-step Flow bên dưới). Các route đọc/ghi đơn giản (`GET` danh sách, `PUT` chronicle/thông tin chung, `PUT`/`DELETE` "tôi là ai") chỉ gồm Step A + một truy vấn nên không có flow riêng.

## Step A — Phân giải vai trò (dùng chung mọi route `/v1/genealogies/{id}/...`)

- Trigger: handler đã qua `authMiddleware` và parse `{id}` thành số dương (không phải số dương → `404 GENEALOGY_NOT_FOUND`, không chạm DB).
- Pre-condition: `userID` có trong context.
- Call: `GenealogyRepository.ResolveRole(ctx, id, userID)` — **một** truy vấn:
  `SELECT CASE WHEN g.owner_user_id=$2 THEN 'owner' WHEN g.clan_head_user_id=$2 THEN 'clan_head' WHEN c.user_id IS NOT NULL THEN 'editor' END FROM genealogies g LEFT JOIN genealogy_collaborators c ON c.genealogy_id=g.id AND c.user_id=$2 WHERE g.id=$1`
    - Success (có vai trò) → nghiệp vụ của route, role được truyền xuống usecase.
    - Failure (không có hàng, hoặc role `NULL`) → `404 GENEALOGY_NOT_FOUND`, dừng (AC15).
    - Failure (route owner-only mà role ≠ owner) → `403 GENEALOGY_FORBIDDEN`, dừng.
    - Failure (lỗi DB) → `500`.

## Step-by-step Flow — Tạo gia phả (POST /v1/genealogies)

Step G-1 — Xác thực + validate request
- Trigger: handler nhận `POST /v1/genealogies`.
- Pre-condition: access token hợp lệ (nếu không `authMiddleware` đã trả `401`).
- Call: `DecodeJSONBodyStrict` + usecase validate AC3–AC9, AC11 (tên, quê gốc, founder, spouses, năm theo `Asia/Ho_Chi_Minh`, mặc định giới tính phối ngẫu).
    - Success → Step G-2
    - Failure → `400 VALIDATION_ERROR` kèm `details.field`; không chạm DB.

Step G-2 — Ghi tất cả trong một transaction
- Trigger: validate xong.
- Pre-condition: dữ liệu đã chuẩn hoá.
- Call: `GenealogyRepository.Create` trong **một transaction**:
  1. `SELECT id FROM users WHERE id=$owner FOR UPDATE` (khoá để hai request tạo song song không vượt trần)
  2. `SELECT count(*) FROM genealogies WHERE owner_user_id=$owner`
  3. `INSERT INTO genealogies ...` RETURNING id
  4. `INSERT INTO genealogy_chronicles (genealogy_id) VALUES (...)` (rỗng)
  5. `INSERT INTO genealogy_members` thuỷ tổ (`kind='blood'`, `generation=1`, `parent_id=NULL`) RETURNING id
  6. `INSERT INTO genealogy_members` cho từng spouse (`kind='in_law'`, `spouse_of_id=<thuỷ tổ>`, `generation=1`)
  7. `COMMIT`
    - Success → Step G-3
    - Failure (count ≥ 10) → `ErrGenealogyLimitReached` → rollback → `409 GENEALOGY_LIMIT_REACHED` (⚠️ trần chưa chốt)
    - Failure (lỗi DB / CHECK) → rollback toàn bộ (AC12) → `500`

Step G-3 — Dựng response
- Trigger: transaction đã commit.
- Pre-condition: có `genealogy_id`.
- Call: `FindDetail` (cùng kết nối/đọc lại) trả `GenealogyDetail` với `my_role = owner`, `founder_name`, `stats` (`generations_count = 1`, `members_count = 1 + len(spouses)`).
    - Success → `201 GenealogyDetail`
    - Failure (lỗi DB) → `500`; gia phả đã được tạo (idempotency nằm ở client; không tự rollback vì đã commit).

## Step-by-step Flow — Thêm thành viên (POST /v1/genealogies/{id}/members)

Step M-1 — Phân giải vai trò + validate request
- Trigger: handler nhận `POST /v1/genealogies/{id}/members`.
- Pre-condition: access token hợp lệ.
- Call: Step A (mọi vai trò đều được), rồi `DecodeJSONBodyStrict` + validate `relation`, `related_member_id`, thông tin thành viên (AC4–AC8, AC18, AC21, AC26).
    - Success → Step M-2
    - Failure (không có vai trò) → `404 GENEALOGY_NOT_FOUND`; (validation) → `400 VALIDATION_ERROR`.

Step M-2 — Ghi trong một transaction
- Trigger: validate xong.
- Pre-condition: `role` ≠ none.
- Call: `GenealogyMemberRepository.Add` trong **một transaction**:
  1. `SELECT id FROM genealogies WHERE id=$g FOR UPDATE` (tuần tự hoá các lần thêm vào cùng gia phả để trần thành viên không bị vượt)
  2. `SELECT kind, generation, gender FROM genealogy_members WHERE id=$related AND genealogy_id=$g`
  3. `SELECT count(*) FROM genealogy_members WHERE genealogy_id=$g`
  4. nếu `relation=spouse`: `SELECT count(*) FROM genealogy_members WHERE spouse_of_id=$related`
  4b. nếu `relation=child` và có `other_parent_id`: `SELECT 1 FROM genealogy_members WHERE id=$other AND genealogy_id=$g AND kind='in_law' AND spouse_of_id=$related` (AC57)
  4c. nếu `relation=child` và không gửi `birth_order`: `SELECT coalesce(max(birth_order),0)+1 FROM genealogy_members WHERE parent_id=$related` (AC56; an toàn vì đã giữ khoá ở bước 1)
  5. `INSERT INTO genealogy_members ...` (child: `kind='blood'`, `parent_id=$related`, `other_parent_id`, `birth_order`, `generation=related.generation+1`; spouse: `kind='in_law'`, `spouse_of_id=$related`, `generation=related.generation`) RETURNING `id, created_at, updated_at`
  6. `COMMIT`
    - Success → Step M-3
    - Failure (bước 2 không có hàng) → `ErrMemberNotFound` → rollback → `404 MEMBER_NOT_FOUND` (AC22)
    - Failure (related là in-law, đã đủ 5 phối ngẫu, đời > 60) → `400 VALIDATION_ERROR` `details.field=related_member_id` (AC19, AC20, AC25)
    - Failure (bước 4b không có hàng, hoặc `other_parent_id` kèm `relation=spouse`) → rollback → `400 VALIDATION_ERROR` `details.field=other_parent_id` (AC57)
    - Failure (count ≥ 5000) → `409 MEMBER_LIMIT_REACHED` (AC24, ⚠️ trần chưa chốt)
    - Failure (lỗi DB) → rollback → `500`

Step M-3 — Dựng response
- Trigger: transaction đã commit.
- Pre-condition: có `member.id` và `generation`.
- Call: usecase tính `in_law_label` (từ `gender` của người được gắn — đã đọc ở M-2.2) và trả `Member`.
    - Success → `201 Member`
    - Failure → không có truy vấn thêm nên không có nhánh lỗi mới.

## Step-by-step Flow — Mời người cùng sửa bằng số điện thoại (POST /v1/genealogies/{id}/collaborators)

Step C-1 — Phân giải vai trò + validate số điện thoại
- Trigger: handler nhận `POST /v1/genealogies/{id}/collaborators`.
- Pre-condition: access token hợp lệ.
- Call: Step A, yêu cầu role = `owner`; validate + chuẩn hoá `phone_number` (cùng hàm của register). **Không tra user ở bước này** — để response không phụ thuộc số đã đăng ký hay chưa (AC45).
    - Success → Step C-2
    - Failure (không phải owner) → `403 GENEALOGY_FORBIDDEN` (hoặc `404` nếu không có vai trò)
    - Failure (số sai định dạng) → `400 VALIDATION_ERROR` `details.field=phone_number`

Step C-2 — Ghi lời mời
- Trigger: số điện thoại hợp lệ.
- Pre-condition: role = owner.
- Call: `InvitationRepository.Create` trong **một transaction**: `SELECT id FROM genealogies WHERE id=$g FOR UPDATE`; đánh dấu lời mời cũ của đúng số này đã quá hạn: `UPDATE genealogy_invitations SET status='expired', responded_at=now() WHERE genealogy_id=$g AND phone_number=$p AND status='pending' AND expires_at <= now()`; kiểm số này không thuộc `owner_user_id`/`clan_head_user_id`/`genealogy_collaborators` (join `users` theo `phone_number`); kiểm chưa có lời mời `pending` chưa hết hạn cho số này; `SELECT (SELECT count(*) FROM genealogy_collaborators WHERE genealogy_id=$g) + (SELECT count(*) FROM genealogy_invitations WHERE genealogy_id=$g AND status='pending' AND expires_at > now()) < 20`; `INSERT INTO genealogy_invitations (genealogy_id, phone_number, invited_by, expires_at)`; `COMMIT`.
    - Success → `201 Invitation` (`status = "pending"`; **chưa** cấp quyền; response giống nhau dù số đã hay chưa có tài khoản)
    - Failure (đã có vai trò) → `409 COLLABORATOR_ALREADY_EXISTS` (AC46)
    - Failure (đã có lời mời chờ) → `409 INVITATION_ALREADY_PENDING` (AC46)
    - Failure (editor + lời mời chờ ≥ 20) → `409 COLLABORATOR_LIMIT_REACHED` (AC47, ⚠️ trần và việc tính lời mời chờ chưa chốt)
    - Failure (lỗi DB) → rollback → `500`
      > ⚠️ Not confirmed: thời hạn lời mời 30 ngày (AC75) — hằng `InvitationTTL`, đổi một chỗ.

## Step-by-step Flow — Chấp nhận lời mời (POST /v1/invitations/{id}/accept)

Step I-1 — Xác định người được mời
- Trigger: handler nhận `POST /v1/invitations/{id}/accept`.
- Pre-condition: access token hợp lệ; `{id}` parse được thành số dương (không → `404 INVITATION_NOT_FOUND`, không chạm DB). **Không qua Step A** (người gọi chưa có vai trò).
- Call: `UserPhoneLookup.FindPhoneByID(userID)` lấy số điện thoại của chính người gọi.
    - Success → Step I-2
    - Failure (lỗi DB) → `500`

Step I-2 — Chấp nhận trong một transaction
- Trigger: có số điện thoại của người gọi.
- Pre-condition: không.
- Call: `InvitationRepository.Accept` trong **một transaction**: đọc `genealogy_id` của lời mời `WHERE id=$i AND phone_number=$phone` (không có hàng → `404 INVITATION_NOT_FOUND`); `SELECT id FROM genealogies WHERE id=$g FOR UPDATE` (luôn khoá gia phả **trước**, như Step C-2); `SELECT status, expires_at FROM genealogy_invitations WHERE id=$i FOR UPDATE` và kiểm lại; kiểm người gọi chưa là `owner`/`clan_head`/`editor`; `SELECT count(*) FROM genealogy_collaborators WHERE genealogy_id=$g` < 20; `INSERT INTO genealogy_collaborators (genealogy_id, user_id, granted_by)` với `granted_by = invited_by`; `UPDATE genealogy_invitations SET status='accepted', responded_at=now()`; `COMMIT`.
    - Success → `200 {genealogy_id, role: "editor"}` (hiệu lực ngay ở request kế tiếp, AC54)
    - Failure (không có / không gửi tới số của người gọi) → `404 INVITATION_NOT_FOUND`
    - Failure (không còn `pending` hoặc đã hết hạn) → `409 INVITATION_NOT_PENDING`
    - Failure (người gọi đã có vai trò) → `409 COLLABORATOR_ALREADY_EXISTS`
    - Failure (≥ 20 editor) → `409 COLLABORATOR_LIMIT_REACHED`
    - Failure (lỗi DB) → rollback → `500`

Route `POST /v1/invitations/{id}/decline` chỉ là Step I-1 + một `UPDATE ... SET status='declined'` có điều kiện (cùng các mã `404`/`409` của Step I-2) nên không có flow riêng; `GET /v1/invitations`, `GET`/`DELETE` lời mời của owner là một truy vấn (cộng Step A cho hai route của owner).

## Step-by-step Flow — Xoá thành viên (DELETE /v1/genealogies/{id}/members/{memberId})

Step D-1 — Phân giải vai trò
- Trigger: handler nhận `DELETE /v1/genealogies/{id}/members/{memberId}`.
- Pre-condition: access token hợp lệ; `{memberId}` parse được thành số dương (không → `404 MEMBER_NOT_FOUND`).
- Call: Step A (mọi vai trò đều được).
    - Success → Step D-2
    - Failure → `404 GENEALOGY_NOT_FOUND`.

Step D-2 — Kiểm tra và xoá trong một transaction
- Trigger: có vai trò.
- Pre-condition: không.
- Call: `GenealogyMemberRepository.Delete` trong **một transaction**:
  1. `SELECT kind, parent_id FROM genealogy_members WHERE id=$m AND genealogy_id=$g FOR UPDATE`
  2. nếu `kind='blood'`: `SELECT EXISTS (SELECT 1 FROM genealogy_members WHERE parent_id=$m)`
  3. nếu `kind='blood'`: `DELETE FROM genealogy_members WHERE id=$m AND genealogy_id=$g` — các in-law gắn với nó bị xoá theo bằng `ON DELETE CASCADE` của `spouse_of_id`
  4. nếu `kind='in_law'`: chỉ `DELETE` đúng hàng đó
  5. `COMMIT`
    - Success → `200 {"success": true}`
    - Failure (bước 1 không có hàng) → `404 MEMBER_NOT_FOUND` (AC29, AC37)
    - Failure (bước 2 có con) → rollback → `409 MEMBER_HAS_DESCENDANTS` (AC35, ⚠️ chính sách xoá nhánh chưa chốt)
    - Failure (`parent_id IS NULL` và `kind='blood'`) → `409 FOUNDER_NOT_DELETABLE` (AC36)
    - Failure (lỗi DB, kể cả FK `RESTRICT` bắt được con phát sinh song song) → rollback → `500` hoặc `409 MEMBER_HAS_DESCENDANTS` nếu map được vi phạm FK

## Step-by-step Flow — Hồ sơ thành viên (GET /v1/genealogies/{id}/members/{memberId}/profile)

Step P-1 — Phân giải vai trò + parse
- Trigger: handler nhận `GET /v1/genealogies/{id}/members/{memberId}/profile`.
- Pre-condition: access token hợp lệ; `{memberId}` parse được thành số dương (không → `404 MEMBER_NOT_FOUND`).
- Call: Step A (mọi vai trò đều được).
    - Success → Step P-2
    - Failure → `404 GENEALOGY_NOT_FOUND`.

Step P-2 — Tải cây và con trỏ (chỉ đọc)
- Trigger: có vai trò.
- Pre-condition: không.
- Call: `GenealogyMemberRepository.List(ctx, genealogyID)` (tất cả thành viên, ≤ 5000) rồi `FindMyMemberID(ctx, genealogyID, userID)`, **trong cùng một transaction `READ ONLY` mức `REPEATABLE READ`** để hai đọc nhìn cùng một ảnh chụp (không thấy nửa vời khi người khác sửa đúng lúc).
    - Success → Step P-3
    - Failure (lỗi DB) → `500`.

Step P-3 — Tính trong bộ nhớ (hàm thuần, không I/O)
- Trigger: đã có danh sách thành viên và `my_member_id`.
- Pre-condition: không.
- Call: `BuildIndex`; kiểm `memberId` có trong chỉ mục; rồi `Age` (AC65), `Relatives` (AC66), `Lineage` (AC67), `Chi` (AC68); `Kinship` (AC69) **chỉ khi** `my_member_id` ≠ null.
    - Success → `200 MemberProfile`
    - Failure (`memberId` không có trong chỉ mục) → `404 MEMBER_NOT_FOUND` (không lộ thành viên của gia phả khác, vì chỉ mục chỉ chứa gia phả này)
    - Failure (`my_member_id` trỏ tới nút không còn) → coi như `null` (không xảy ra do FK CASCADE; chỉ phòng thủ), không lỗi.

### Quy tắc chi (`Chi` trong `genealogy_kinship.go`, AC68)

Chi nằm ở chỗ **cây rẽ nhánh** và lồng nhau, không cố định theo đời. Thuật toán (một lượt đi ngược `parent_id`, ≤ 60 bước, dùng chỉ mục đã dựng ở Step P-3):

1. Với in-law, thay bằng người dòng họ được gắn (`spouse_of_id`) rồi tính như bên dưới.
2. Dựng đường `root → ... → X` bằng cách đi ngược `parent_id` từ `X`.
3. Với mỗi nút `N` trên đường (trừ thuỷ tổ) có cha/mẹ `P`: `N` là **đầu chi** khi `N.gender = male` và `P` có ≥ 2 con trai (đếm trên chỉ mục: các thành viên blood có `parent_id = P` và `gender = male`).
4. `chi_path` = các đầu chi theo thứ tự đời tăng dần, mỗi phần tử `{ancestor = N, rank, is_eldest}` với `rank` = vị trí của `N` trong các con trai của `P` sắp theo `(birth_order, id)`. `chi` = phần tử cuối, hoặc `null` khi `chi_path` rỗng.

Ví dụ: thuỷ tổ T có hai con trai Lộc và Nghĩa; Lộc có hai con trai A và B; A có một con C. Khi đó `chi_path(C) = [Lộc, A]` và `chi(C) = A`; `chi_path(Nghĩa) = [Nghĩa]`; `chi_path(T) = []`. Bảng test cần các ca: chuỗi con một (không rẽ nhánh), rẽ nhánh nhiều tầng, con gái (không đầu chi, thuộc chi của cha), in-law, thuỷ tổ, thêm con trai thứ hai làm nhãn đổi (hành vi đã nêu ở AC68).

### Bảng xưng hô v1 (đầu vào của `Kinship`)

> ⚠️ Not confirmed: toàn bộ bảng dưới đây là **đề xuất** theo cách gọi phổ biến, chưa được anh Đức duyệt — chặn việc cài `Kinship` (xem AC69); phải duyệt trước khi viết bảng test.

Gọi "tôi" = `M`, người được chọn = `T`. Một người dòng họ (blood) là điểm neo; với in-law (dâu/rể) thì lấy quan hệ của người dòng họ được gắn rồi áp quy tắc "vợ/chồng của". Tổ tiên chung gần nhất là `A`; `up(M)` = số đời từ `M` lên `A`, `up(T)` = số đời từ `T` lên `A` (bằng `generation` hiệu số). Chỉ đi qua `parent_id` (bên nội); `mother` của `X` là `other_parent_id` của `X`.

| Điều kiện | `code` (đề xuất) | `label` |
|---|---|---|
| `T = M` | `SELF` | Bản thân |
| `T` là in-law gắn với `M` | `SPOUSE` | Vợ / Chồng (theo `T.gender`) |
| `T` là `other_parent_id` của `M` | `MOTHER` | Mẹ (hoặc Cha nếu nam) |
| `up(M)=1, up(T)=0` (T là cha) | `FATHER` | Cha |
| `up(M)=2, up(T)=0` | `PATERNAL_GRANDPARENT` | Ông nội / Bà nội (bà nội = vợ của ông nội) |
| `up(M)=3, up(T)=0` | `GREAT_GRANDPARENT` | Cụ ông / Cụ bà |
| `up(M)>=4, up(T)=0` | `ANCESTOR` | Cố / Kỵ / "Tổ đời N" (tên gọi theo đời, chưa chốt) |
| cùng cha (`up=1,1`), `T` lớn hơn `M` (theo `birth_order`) | `ELDER_SIBLING` | Anh / Chị (theo `T.gender`) |
| cùng cha, `T` nhỏ hơn | `YOUNGER_SIBLING` | Em |
| `up(M)=2, up(T)=1` (anh/em của cha), `T` lớn hơn cha | `FATHER_ELDER_BROTHER` | Bác |
| anh/em của cha, `T` nhỏ hơn cha, nam | `FATHER_YOUNGER_BROTHER` | Chú |
| anh/chị/em của cha, `T` là nữ | `FATHER_SISTER` | Cô |
| `up(M)=2, up(T)=2` | `COUSIN` | Anh / Chị / Em họ (lớn/nhỏ theo `birth_order` của hai cha) |
| `T` là con của `M` | `CHILD` | Con |
| `T` là con của anh/chị/em của `M` | `NEPHEW_NIECE` | Cháu |
| `T` là cháu nội của `M` | `GRANDCHILD` | Cháu nội |
| `T` là in-law gắn với `X` (≠ M) | suy từ `X` | vợ của chú → Thím; vợ của bác → Bác gái; chồng của cô → Dượng; vợ của anh → Chị dâu; chồng của chị → Anh rể; vợ của con → Con dâu; chồng của con gái → Con rể |
| không rơi vào dòng nào (kể cả bên ngoại) | `UNKNOWN` | `null` |

Thứ tự lớn/nhỏ giữa hai người cùng cha: so `birth_order` rồi `id`. Giữa `T` (anh/em của cha) và cha: cùng cách so sánh. `birth_order` trùng/không có thì dùng `id`.

## Deliberately-stubbed-for-now

Không có route nào bị stub: toàn bộ 24 route đều có contract ở `requirements.md`. Phần hoãn:

- Ảnh hoành phi/ảnh thành viên — chờ GCS, không có route/cột trong v1.
- Push notification/SMS báo lời mời — chờ trả lời AC70; v1 chỉ có hộp lời mời `GET /v1/invitations` (client tự gọi).
- Route đổi cha/mẹ của một thành viên (AC31) và chống ghi đè đồng thời (`Pending`) — chưa chốt, chưa thiết kế.
- Xưng hô bên ngoại (cậu, dì, ông bà ngoại): cần cho phép dâu/rể có cha mẹ trong cây — hiện `Kinship` trả `UNKNOWN`.
- Route sắp xếp lại thứ tự anh em (AC56): v1 chỉ sửa `birth_order` từng người.
- Ngày giỗ trên thành viên hay liên kết `memorial` (AC55): v1 lưu `death_lunar_*` ngay trên thành viên.
