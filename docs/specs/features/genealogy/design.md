# Design — genealogy

Version: 5.12
Last updated: 2026-10-11
Status: implemented — đã cài đặt và kiểm chứng end-to-end (xem `ac-audit.md`); không còn marker `⚠️ Not confirmed` nào.
Module: `genealogy`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Hai mươi mốt route đều **user-scoped** — mười tám route `/v1/genealogies/*` (trong đó ba route `.../invitations*` và hai route `.../viewers*` của owner) cộng ba route `/v1/invitations*` của người được mời —, bọc `authMiddleware.AuthenticateFunc` (`RequireTokenType: "access"`) như `POST /v1/auth/logout` và `/v1/memorials`. Khác `memorial`, quyền không chỉ là "chủ dòng": mỗi route đầu tiên **phân giải vai trò** của user trong gia phả (Step A) rồi mới chạy nghiệp vụ. Chạm đủ bốn layer (domain/usecase/repository/delivery), sở hữu migration `0028_genealogy.sql` (năm bảng, gồm `genealogy_invitations`; `0019` đã bị `0019_sutra_audio_search.sql` dùng). Hình dạng gần nhất: `memorial` (404 đồng nhất, trần số lượng kiểm trong transaction có khoá) cộng lớp phân quyền.

Từ v2.0, thông tin hiển thị khi chọn một người (tuổi, họ hàng, đường từ thuỷ tổ, chi, xưng hô với "tôi") **tính ở server** trong một hàm thuần của usecase (`genealogy_kinship.go`) — tải cả cây (≤ 5000 hàng) một lần rồi tính trong bộ nhớ (Step P), không đệ quy SQL.

Từ v4.0: (1) thêm route đổi quan hệ thành viên `PUT .../members/{memberId}/relation` (Step R) — tải cả cây trong một tx, kiểm chống vòng và trần 60 đời trong bộ nhớ rồi dời `generation` cả nhánh; (2) con của thành viên nữ dòng họ bị chặn, thuỷ tổ phải là nam (Step M-2, G-1); (3) lệch năm sinh cha/con chỉ trả `warnings` (Step M-3, R-3); (4) ngày chạp họ và ngày giỗ lưu **dương lịch** kèm cờ `*_entered_as_lunar`, server quy đổi sang âm lịch bằng một hàm thuần (`genealogy_lunar.go`) và trả `chap_lunar_*`/`death_lunar_*` tính ra (không lưu).

Từ v3.0, cấp quyền xem (`viewer`) **luôn đi qua lời mời** (Step C, Step I): owner tạo lời mời theo số điện thoại (đã hoặc chưa có tài khoản — cùng một response, không lộ số đã đăng ký); người được mời xem lời mời ở hộp lời mời `GET /v1/invitations` rồi chấp nhận/từ chối. Lời mời khớp người **chỉ bằng số điện thoại** nên không đổi luồng register. Ba route `/v1/invitations*` không thuộc một gia phả mà người gọi chưa có vai trò, nên **không qua Step A** — kiểm bằng số điện thoại của chính người gọi.

Từ v5.0 chỉ còn **hai vai trò** `owner` (sửa) và `viewer` (chỉ xem); bỏ `editor` và `clan_head`. Vị trí "tôi" trong cây **tính ra khi đọc** bằng cách so `phone_number` của thành viên với số điện thoại tài khoản người gọi (Step P-2/P-3) — không có bảng con trỏ, không có route đặt "tôi là ai"; lời mời luôn gắn một thành viên (`genealogy_invitations.member_id`) và ghi số điện thoại lên nút đó, nên người chấp nhận có vị trí ngay; trưởng tộc là cờ `is_clan_head` trên thành viên blood, tối đa một nút/gia phả (unique index một phần), không phải vai trò; owner gắn một người vào nút bằng cách ghi `phone_number` của nút (một số chỉ thuộc một nút trong gia phả, ràng buộc ở DB).

Vai trò và quyền:

| Hành động | owner | viewer | không có vai trò |
|---|---|---|---|
| Xem gia phả, cây, phả ký, hồ sơ thành viên (`profile`) | ✅ | ✅ | 404 |
| Sửa tên/quê gốc, phả ký, thành viên (thêm/sửa/xoá/đổi quan hệ) | ✅ | 403 | 404 |
| Xoá gia phả, mời/huỷ lời mời, xem người xem, thu quyền xem | ✅ | 403 | 404 |
| Tự rời gia phả | ❌ | ✅ | 404 |

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/genealogy.go` | `Genealogy` (id, ownerUserID, name, ancestralHome), `Chronicle` (các field AC39, con trỏ cho field tuỳ chọn), `GenealogyMember` (kind, gender, `BirthYear/DeathYear *int`, `IsAlive`, `PhoneNumber *string`, `IsClanHead bool`, `ParentID/OtherParentID/SpouseOfID *int64`, `BirthOrder *int`, `BurialPlace *string`, `DeathMonth/DeathDay *int`, `DeathEnteredAsLunar *bool`, `Generation`...), `MemberWarning` (`Code`, `MemberID`, `RelatedMemberID`), `Role` (`owner`/`viewer`), `Viewer` (userID, grantedBy, createdAt), `MemberProfile`/`MemberRef`/`Lineage`/`Chi`/`Kinship` (kết quả tính ra). Chỉ field nghiệp vụ, không JSON tag. Giới hạn số và hạn lời mời **không** là hằng ở domain: nằm ở `config.GenealogyConfig` (`max_per_user=10`, `max_members=5000`, `max_generation=60`, `max_spouses=5`, `max_viewers=20`, `invitation_ttl=720h`; chốt 2026-10-10, quy ước "mọi giá trị không nhạy cảm đều ở config"), truyền xuống repository qua `domain.MemberLimits` và tham số `maxPerUser`/`maxViewers`/`ttl`. |
| Domain / Errors | `domain/genealogy.go` | `ErrGenealogyNotFound`, `ErrGenealogyForbidden`, `ErrGenealogyLimitReached`, `ErrMemberNotFound`, `ErrMemberLimitReached`, `ErrMemberHasDescendants`, `ErrFounderNotDeletable`, `ErrFounderNotMovable`, `ErrMemberPhoneInUse`, `ErrViewerExists`, `ErrViewerLimitReached`, `ErrViewerNotFound`; `ErrGenealogyInvalid` + `*GenealogyValidationError{Field, Message}` (theo mẫu `*MemorialValidationError`). Handler đọc `Field` để trả `details.field`. |
| Domain / Repository interface | `domain/genealogy.go` | `GenealogyRepository`: `Create(ctx, g, c, founder, spouses, maxPerUser)` (một tx), `ListByUser`, `ResolveRole(ctx, genealogyID, userID)` (trả `Role` hoặc `ErrGenealogyNotFound`), `FindDetail`, `UpdateInfo`, `Delete`, `UpsertChronicle`. `GenealogyMemberRepository`: `List`, `FindByID(ctx, genealogyID, memberID)`, `Add(ctx, genealogyID, relation, relatedID, otherParentID *int64, m, limits)` (một tx), `Update`, `Delete(ctx, genealogyID, memberID)` (một tx), `Move(ctx, genealogyID, memberID, relation, relatedID, otherParentID *int64, birthOrder *int, limits)` (một tx, Step R-2), `FindMemberIDByPhone(ctx, genealogyID, phone) (*int64, error)` (cho `my_member_id`, AC59). `ViewerRepository`: `List` (kèm `member_id` theo số điện thoại), `Remove`. `FindDetail(ctx, id, userID)` trả thêm `my_member_id` (so `phone_number` với số điện thoại của userID). Định nghĩa ở domain, không ở `repository/`. |
| Domain / Invitation | `domain/genealogy.go` (hoặc `domain/genealogy_invitation.go` nếu file vượt ~500 dòng) | Entity `Invitation` (id, genealogyID, `MemberID`, phoneNumber, invitedBy, status, createdAt, expiresAt, respondedAt, không JSON tag), `InvitationStatus` (`pending`/`accepted`/`declined`/`cancelled`/`expired`); hạn lời mời lấy từ `config.GenealogyConfig.InvitationTTL`. Sentinel: `ErrInvitationNotFound`, `ErrInvitationNotPending`, `ErrInvitationAlreadyPending`. Interface `InvitationRepository`. Mở rộng `UserPhoneLookup` thêm `FindPhoneByID(ctx, userID)`. |
| Domain / User lookup port | `domain/genealogy.go` | `UserPhoneLookup.FindIDByPhone(ctx, normalizedPhone) (int64, error)` — port tối thiểu để usecase không import repo user cụ thể; ưu tiên tái dùng hàm có sẵn nếu repo user đã có (xem Ghi chú của `requirements.md`). |
| Domain / Mock | `domain/mock_genealogy_repository.go` | Mock viết tay cho ba repository + `UserPhoneLookup`, dùng cho test usecase. |
| Usecase | `services/appService/usecase/genealogy_usecase.go` | `Create`, `List`, `Get`, `UpdateInfo`, `Delete`, `UpdateChronicle`. Validation AC3–AC9, AC38–AC40; `time.LoadLocation("Asia/Ho_Chi_Minh")` cho "năm hiện tại"; mặc định giới tính phối ngẫu (AC9); tính `stats`/`founder_name` từ repo. Mọi hàm (trừ `Create`, `List`) mở đầu bằng `ResolveRole`. |
| Usecase | `usecase/genealogy_member_usecase.go` | `List`, `Get`, `Add`, `Update`, `Delete`, `Move`. Validation AC4–AC8, AC18–AC27, AC30–AC37, AC76–AC80; tính `in_law_label`; ghi `is_clan_head` (AC82: chỉ blood + nam + còn sống, kiểm cả trạng thái sau ghi của người đang giữ cờ nếu `gender`/`is_alive` đổi; vắng ở `PUT` = giữ nguyên; khi `true`, `UPDATE ... SET is_clan_head=false` người cũ **trước** rồi mới đặt người mới, cùng tx — unique index một phần kiểm theo từng câu lệnh); khi `phone_number` đổi/xoá thì huỷ lời mời `pending` của nút đó cùng tx (AC30); tính `warnings` (AC27) **sau commit** bằng hai đọc nhẹ (cha/mẹ và các con của người vừa ghi) — không cần nằm trong tx vì chỉ là cảnh báo. Con của thành viên nữ bị chặn (AC19); `founder.gender` phải `male` (AC4). Phân biệt `child`/`spouse`. |
| Usecase | `usecase/genealogy_access_usecase.go` | `ListViewers`, `Invite`, `ListInvitations`, `CancelInvitation`, `RemoveViewer` (một hàm cho "owner gỡ viewer" và "viewer tự rời"; owner gọi với chính mình → `ErrViewerNotFound`). Chuẩn hoá số điện thoại (tái dùng hàm của register), kiểm owner-only (403). `Invite(ctx, userID, genealogyID, memberID, phone)` — **không** tra user (Step C-1). |
| Usecase | `usecase/genealogy_invitation_usecase.go` | `ListMine`, `Accept`, `Decline` (AC70–AC72). **Không** qua Step A: lấy số điện thoại của người gọi bằng `UserPhoneLookup.FindPhoneByID`, rồi so với `phone_number` của lời mời. Id không phải số dương → `404 INVITATION_NOT_FOUND` không chạm DB. |
| Usecase | `usecase/genealogy_profile_usecase.go` | `GetProfile`. Mở đầu bằng `ResolveRole` (Step A, cả owner lẫn viewer); chạy Step P: lấy số điện thoại của người gọi (`UserPhoneLookup.FindPhoneByID`), tải cả cây, rồi gọi các hàm thuần ở `genealogy_kinship.go`. Không còn `SetMyMember`/`ClearMyMember` (AC61). |
| Usecase | `usecase/genealogy_lunar.go` | **Hàm thuần, không I/O** (TDD bảng test): `SolarToLunar(year, month, day) (LunarDate, bool)` — thuật toán Hồ Ngọc Đức, múi giờ UTC+7, hợp lệ năm `1900..2199` (ngoài khoảng → `ok=false`, field tính ra là `null`); `LunarDate{Day, Month int; Leap bool}`. Được gọi khi dựng `Member` (`death_lunar_*`) và `chronicle` (`chap_lunar_*`) ở mọi response (Step G-3, M-3, R-3, `GET`). |
| Usecase | `usecase/genealogy_kinship.go` | **Hàm thuần, không I/O** (TDD bảng test): `BuildIndex(members)` (map id→member, con theo cha), `Age(member, now)` (AC65), `Relatives(index, id)` (AC66), `Lineage(index, id)` (AC67), `Chi(index, id)` (AC68), `Kinship(index, meID, targetID)` (AC69, theo "Bảng xưng hô v1"). Mọi sắp xếp theo `(birth_order, id)`; `now` truyền vào để test không phụ thuộc đồng hồ. |
| Repository | `services/appService/repository/postgres_genealogy_repository.go` | SQL năm bảng; `Create` và `Add`/`Delete`/`Move` member chạy một tx; `ResolveRole` là **một** truy vấn; mọi truy vấn member có `WHERE genealogy_id = $1`. `NULL` ↔ `nil`. Map vi phạm unique/FK/CHECK → sentinel error (unique `(genealogy_id, phone_number)` → `ErrMemberPhoneInUse`). Sửa tách file `postgres_genealogy_member_repository.go`, `postgres_genealogy_access_repository.go` nếu file vượt ~500 dòng. |
| Repository | `services/appService/repository/postgres_genealogy_invitation_repository.go` | `Create` (Step C-2, một tx), `ListMine`, `ListByGenealogy`, `Cancel`, `Accept` (Step I-2, một tx — `INSERT genealogy_viewers` nằm trong tx này, nên không có `ViewerRepository.Add` riêng), `Decline`. Thứ tự khoá luôn **`genealogies` trước, `genealogy_invitations` sau** để `Create` và `Accept` không deadlock. |
| Delivery / Handler | `services/appService/delivery/http/genealogy_handler.go` (+ `genealogy_member_handler.go`, `genealogy_access_handler.go`, `genealogy_profile_handler.go`, `genealogy_invitation_handler.go`) | Parse path/body → gọi usecase → map sentinel sang HTTP. `DecodeJSONBodyStrict`. `user_id` chỉ từ context của middleware. Không logic nghiệp vụ. |
| Delivery / DTO | cùng các file handler | DTO request/response (JSON tag chỉ ở đây). Trả `null` tường minh; `items: []` khi rỗng. |
| Delivery / Errors | `services/appService/delivery/http/errors.go` | Thêm map: `GENEALOGY_NOT_FOUND`/`MEMBER_NOT_FOUND`/`VIEWER_NOT_FOUND`/`INVITATION_NOT_FOUND` → 404, `GENEALOGY_FORBIDDEN` → 403 (chưa có mã 403 nào trong file — thêm helper mới), các `*_LIMIT_REACHED`/`MEMBER_HAS_DESCENDANTS`/`FOUNDER_NOT_DELETABLE`/`FOUNDER_NOT_MOVABLE`/`MEMBER_PHONE_ALREADY_USED`/`VIEWER_ALREADY_EXISTS`/`INVITATION_ALREADY_PENDING`/`INVITATION_NOT_PENDING` → 409. |
| Wiring | `services/appService/cmd/main.go` | Đăng ký 21 route, **bọc `authMiddleware.AuthenticateFunc`** từng route (verify bằng E2E, không chỉ tài liệu). Router hiện dùng `ServeMux` kiểu nào thì theo đó cho path có `{id}`/`{memberId}`/`{userId}` (xem cách `memorial` đăng ký). Không chạm khối `⚠️ DEV ONLY`. |
| Migration | `migrations/0028_genealogy.sql` | Năm bảng, CHECK, index (mục "Schema" bên dưới). Số `0028` (các số `0019`–`0027` đã được feature khác dùng) — kiểm lại bằng `ls migrations/` ngay trước khi tạo file. Mirror vào `docs/specs/db/erd.mmd` + `erd.svg`. |
| API spec | `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) | Thêm tag Genealogies, 21 route, các schema (`GenealogyDetail`, `Member`, `Chronicle`, `Viewer`, `Invitation`, `MyInvitation`, `MemberProfile`, `MemberRef`), bảo mật Bearer. |
| E2E | `services/appService/e2e/genealogy_e2e_test.go` | Binary thật + Postgres thật: tạo → thêm → phân quyền hai user → 404/403 → chọn "tôi là ai" → `profile` (xưng hô, chi, đường từ thuỷ tổ). |

## Schema (`migrations/0028_genealogy.sql`, đã viết và áp lên Postgres 16)

```
genealogies            id BIGSERIAL PK, owner_user_id → users(id) ON DELETE RESTRICT,
                       name VARCHAR(100), ancestral_home VARCHAR(200), created_at, updated_at

genealogy_chronicles   genealogy_id PK → genealogies(id) ON DELETE CASCADE,
                       surname, current_location, since_year, origin, history,
                       hall_name, hall_address,
                       chap_year SMALLINT, chap_month SMALLINT, chap_day SMALLINT,  -- DƯƠNG lịch (AC40)
                       chap_entered_as_lunar BOOLEAN NOT NULL DEFAULT false,
                       hoanh_phi, hoanh_phi_reading, hoanh_phi_meaning,
                       couplet_upper, couplet_upper_reading, couplet_upper_meaning,
                       couplet_lower, couplet_lower_reading, couplet_lower_meaning,
                       teaching, updated_at
                       CHECK (chap_year, chap_month, chap_day: cả ba NULL hoặc cả ba NOT NULL)
                       CHECK chap_month 1..12, chap_day 1..31 (ngày tối đa theo tháng kiểm ở usecase)
                       CHECK chap_entered_as_lunar ⇒ chap_year IS NOT NULL

genealogy_members      id BIGSERIAL PK, genealogy_id → genealogies(id) ON DELETE CASCADE,
                       kind ('blood'|'in_law'), full_name, gender ('male'|'female'),
                       birth_year, is_alive, death_year, residence, dharma_name,
                       is_clan_head BOOLEAN NOT NULL DEFAULT false,  -- cờ trưởng tộc (AC82)
                       tablet_location, burial_place,
                       death_month SMALLINT, death_day SMALLINT,   -- DƯƠNG lịch, năm là death_year (AC55)
                       death_entered_as_lunar BOOLEAN,
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
                       CHECK is_alive ⇒ burial_place, death_month, death_day, death_entered_as_lunar đều NULL
                       CHECK (death_day IS NULL) = (death_month IS NULL);
                             death_day 1..31, death_month 1..12, birth_order 1..1000
                       CHECK death_day IS NOT NULL ⇒ death_year IS NOT NULL
                       CHECK death_entered_as_lunar IS NOT NULL ⇒ death_day IS NOT NULL
                       CHECK is_clan_head ⇒ (kind='blood' AND gender='male' AND is_alive)  -- AC82
                       UNIQUE INDEX (genealogy_id) WHERE is_clan_head  -- tối đa một trưởng tộc/gia phả
                       CHECK kind='blood'  ⇒ spouse_of_id IS NULL
                       CHECK kind='in_law' ⇒ spouse_of_id IS NOT NULL AND parent_id IS NULL
                       CHECK is_alive ⇒ death_year IS NULL AND tablet_location IS NULL
                       CHECK death_year >= birth_year (khi cả hai có)
                       UNIQUE INDEX (genealogy_id) WHERE kind='blood' AND parent_id IS NULL  -- đúng một thuỷ tổ
                       UNIQUE INDEX (genealogy_id, phone_number) WHERE phone_number IS NOT NULL  -- AC60: một số chỉ thuộc một nút/gia phả
                       INDEX (genealogy_id, generation, id), INDEX (parent_id), INDEX (spouse_of_id)

genealogy_viewers  PK (genealogy_id, user_id), genealogy_id → genealogies ON DELETE CASCADE,
                       user_id → users(id) ON DELETE CASCADE, granted_by → users(id), created_at
                       INDEX (user_id)  -- cho GET /v1/genealogies của một user

genealogy_invitations  id BIGSERIAL PK, genealogy_id → genealogies ON DELETE CASCADE,
                       member_id BIGINT NOT NULL,
                       FK (member_id, genealogy_id) → members(id, genealogy_id) ON DELETE CASCADE  -- nút được mời (AC44)
                       phone_number VARCHAR(20) NOT NULL  -- đã chuẩn hoá như register; KHÔNG có user_id người được mời (AC75)
                       invited_by → users(id) ON DELETE CASCADE,
                       status VARCHAR(10) NOT NULL DEFAULT 'pending'
                         CHECK status IN ('pending','accepted','declined','cancelled','expired'),
                       created_at, expires_at NOT NULL (= created_at + 30 ngày), responded_at NULL
                       CHECK (status = 'pending') = (responded_at IS NULL)  -- 'expired' do Create đánh dấu lười (Step C-2) nên có responded_at
                       UNIQUE INDEX (genealogy_id, phone_number) WHERE status='pending'  -- AC46: một lời mời chờ cho mỗi số
                       INDEX (phone_number) WHERE status='pending'  -- cho GET /v1/invitations
                       INDEX (genealogy_id, created_at DESC)  -- cho GET .../invitations của owner

-- (đã bỏ ở v5.0: genealogy_user_members; vị trí "tôi" tính ra theo phone_number, AC59)
```

> Ghi chú `other_parent_id`: việc "in-law này là vợ/chồng của `parent_id`" (`spouse_of_id = parent_id`) **không** ép được bằng CHECK/FK một hàng; usecase/repository kiểm trong tx (Step M-2) — AC57. `other_parent_id` gán lúc tạo hoặc qua Step R (AC76, kiểm lại trong tx), không có đường đổi khác nên không lệch về sau, trừ khi in-law bị xoá hoặc bị chuyển (đã `SET NULL`). Bất biến "con chỉ gắn vào nam dòng họ" (AC19) ép ở usecase/Step M-2/R-2, không ép bằng CHECK vì phải đọc hàng khác.

## Request flow

> Xem `flow.md` cho sơ đồ sequence Mermaid (mirror 1:1 của bảy Step-by-step Flow bên dưới). Các route đọc/ghi đơn giản (`GET` danh sách, `PUT` chronicle/thông tin chung, `PUT`/`DELETE` "tôi là ai") chỉ gồm Step A + một truy vấn nên không có flow riêng.

## Step A — Phân giải vai trò (dùng chung mọi route `/v1/genealogies/{id}/...`)

- Trigger: handler đã qua `authMiddleware` và parse `{id}` thành số dương (không phải số dương → `404 GENEALOGY_NOT_FOUND`, không chạm DB).
- Pre-condition: `userID` có trong context.
- Call: `GenealogyRepository.ResolveRole(ctx, id, userID)` — **một** truy vấn:
  `SELECT CASE WHEN g.owner_user_id=$2 THEN 'owner' WHEN v.user_id IS NOT NULL THEN 'viewer' END FROM genealogies g LEFT JOIN genealogy_viewers v ON v.genealogy_id=g.id AND v.user_id=$2 WHERE g.id=$1`
    - Success (có vai trò: `owner` hoặc `viewer`) → nghiệp vụ của route, role được truyền xuống usecase.
    - Failure (không có hàng, hoặc role `NULL`) → `404 GENEALOGY_NOT_FOUND`, dừng (AC15).
    - Failure (route owner-only hoặc route ghi mà role ≠ owner, tức `viewer`) → `403 GENEALOGY_FORBIDDEN`, dừng (AC81).
    - Failure (lỗi DB) → `500`.

## Step-by-step Flow — Tạo gia phả (POST /v1/genealogies)

Step G-1 — Xác thực + validate request
- Trigger: handler nhận `POST /v1/genealogies`.
- Pre-condition: access token hợp lệ (nếu không `authMiddleware` đã trả `401`).
- Call: `DecodeJSONBodyStrict` + usecase validate AC3–AC9, AC11 (tên, quê gốc, founder — `gender` phải `male` (AC4), spouses, năm theo `Asia/Ho_Chi_Minh`, mặc định giới tính phối ngẫu).
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
  5. `INSERT INTO genealogy_members` thuỷ tổ (`kind='blood'`, `generation=1`, `parent_id=NULL`, `is_clan_head` theo body, AC82) RETURNING id
  6. `INSERT INTO genealogy_members` cho từng spouse (`kind='in_law'`, `spouse_of_id=<thuỷ tổ>`, `generation=1`)
     (số điện thoại trùng giữa thuỷ tổ và spouses đã bị chặn ở Step G-1 → `400`; unique index `(genealogy_id, phone_number)` là lưới cuối, AC60)
  7. `COMMIT`
    - Success → Step G-3
    - Failure (count ≥ 10) → `ErrGenealogyLimitReached` → rollback → `409 GENEALOGY_LIMIT_REACHED`
    - Failure (lỗi DB / CHECK) → rollback toàn bộ (AC12) → `500`

Step G-3 — Dựng response
- Trigger: transaction đã commit.
- Pre-condition: có `genealogy_id`.
- Call: `FindDetail` (cùng kết nối/đọc lại) trả `GenealogyDetail` với `my_role = owner`, `founder_name`, `chap_lunar_*` (`null` vì phả ký còn rỗng), `stats` (`generations_count = 1`, `members_count = 1 + len(spouses)`).
    - Success → `201 GenealogyDetail`
    - Failure (lỗi DB) → `500`; gia phả đã được tạo (idempotency nằm ở client; không tự rollback vì đã commit).

## Step-by-step Flow — Thêm thành viên (POST /v1/genealogies/{id}/members)

Step M-1 — Phân giải vai trò + validate request
- Trigger: handler nhận `POST /v1/genealogies/{id}/members`.
- Pre-condition: access token hợp lệ.
- Call: Step A, yêu cầu role = `owner`; rồi `DecodeJSONBodyStrict` + validate `relation`, `related_member_id`, thông tin thành viên (AC4–AC8, AC18, AC21, AC26).
    - Success → Step M-2
    - Failure (không có vai trò) → `404 GENEALOGY_NOT_FOUND`; (`viewer`) → `403 GENEALOGY_FORBIDDEN`; (validation) → `400 VALIDATION_ERROR`.

Step M-2 — Ghi trong một transaction
- Trigger: validate xong.
- Pre-condition: `role` = `owner`.
- Call: `GenealogyMemberRepository.Add` trong **một transaction**:
  1. `SELECT id FROM genealogies WHERE id=$g FOR UPDATE` (tuần tự hoá các lần thêm vào cùng gia phả để trần thành viên không bị vượt)
  2. `SELECT kind, generation, gender FROM genealogy_members WHERE id=$related AND genealogy_id=$g` — nếu `relation=child` và `kind='blood'` mà `gender='female'` → từ chối (AC19)
  3. `SELECT count(*) FROM genealogy_members WHERE genealogy_id=$g`
  4. nếu `relation=spouse`: `SELECT count(*) FROM genealogy_members WHERE spouse_of_id=$related`
  4b. nếu `relation=child` và có `other_parent_id`: `SELECT 1 FROM genealogy_members WHERE id=$other AND genealogy_id=$g AND kind='in_law' AND spouse_of_id=$related` (AC57)
  4c. nếu `relation=child` và không gửi `birth_order`: `SELECT coalesce(max(birth_order),0)+1 FROM genealogy_members WHERE parent_id=$related` (AC56; an toàn vì đã giữ khoá ở bước 1)
  4d. nếu `is_clan_head=true` (chỉ khi thêm `child`; dâu/rể bị chặn ở Step M-1): `UPDATE genealogy_members SET is_clan_head=false WHERE genealogy_id=$g AND is_clan_head` (AC82)
  5. `INSERT INTO genealogy_members ...` (child: `kind='blood'`, `parent_id=$related`, `other_parent_id`, `birth_order`, `generation=related.generation+1`; spouse: `kind='in_law'`, `spouse_of_id=$related`, `generation=related.generation`) RETURNING `id, created_at, updated_at`
  6. `COMMIT`
    - Success → Step M-3
    - Failure (bước 2 không có hàng) → `ErrMemberNotFound` → rollback → `404 MEMBER_NOT_FOUND` (AC22)
    - Failure (related là in-law, related là nữ dòng họ khi `child`, đã đủ 5 phối ngẫu, đời > 60) → `400 VALIDATION_ERROR` `details.field=related_member_id` (AC19, AC20, AC25)
    - Failure (bước 4b không có hàng, hoặc `other_parent_id` kèm `relation=spouse`) → rollback → `400 VALIDATION_ERROR` `details.field=other_parent_id` (AC57)
    - Failure (count ≥ 5000) → `409 MEMBER_LIMIT_REACHED` (AC24)
    - Failure (vi phạm unique `(genealogy_id, phone_number)`) → rollback → `409 MEMBER_PHONE_ALREADY_USED` `details.field=phone_number` (AC60)
    - Failure (lỗi DB) → rollback → `500`

Step M-3 — Dựng response
- Trigger: transaction đã commit.
- Pre-condition: có `member.id` và `generation`.
- Call: usecase tính `in_law_label` (từ `gender` của người được gắn — đã đọc ở M-2.2) và `death_lunar_*` (`SolarToLunar`, AC55); rồi tính `warnings` (AC27): nếu `member.birth_year` có, so với `birth_year` của `parent_id` (đã đọc ở M-2.2 — thêm cột `birth_year` vào truy vấn đó) và trả `Member` + `warnings`.
    - Success → `201 Member` (`warnings: []` khi không lệch)
    - Failure → không có truy vấn thêm nên không có nhánh lỗi mới; lệch năm không bao giờ là lỗi.

## Step-by-step Flow — Mời người xem bằng số điện thoại (POST /v1/genealogies/{id}/invitations)

Step C-1 — Phân giải vai trò + validate request
- Trigger: handler nhận `POST /v1/genealogies/{id}/invitations` với `{member_id, phone_number?}`.
- Pre-condition: access token hợp lệ.
- Call: Step A, yêu cầu role = `owner`; `DecodeJSONBodyStrict`; validate `member_id` (số dương) và, nếu có, chuẩn hoá `phone_number` (cùng hàm của register). **Không tra user ở bước này** — để response không phụ thuộc số đã đăng ký hay chưa (AC45).
    - Success → Step C-2
    - Failure (không phải owner) → `403 GENEALOGY_FORBIDDEN` (hoặc `404` nếu không có vai trò)
    - Failure (`member_id` thiếu/không phải số dương, số sai định dạng) → `400 VALIDATION_ERROR` `details.field=member_id`/`phone_number`

Step C-2 — Ghi lời mời (và ghi số lên nút nếu chưa có)
- Trigger: request hợp lệ.
- Pre-condition: role = owner.
- Call: `InvitationRepository.Create` trong **một transaction**:
  1. `SELECT id FROM genealogies WHERE id=$g FOR UPDATE`
  2. `SELECT is_alive, phone_number FROM genealogy_members WHERE id=$member AND genealogy_id=$g` — không có hàng → `404 MEMBER_NOT_FOUND`; `is_alive=false` → `400 member_id`; xác định số mời = body hoặc `phone_number` của nút (thiếu cả hai, hoặc body khác số đã có trên nút → `400 phone_number`)
  3. đánh dấu lời mời cũ của đúng số này đã quá hạn: `UPDATE genealogy_invitations SET status='expired', responded_at=now() WHERE genealogy_id=$g AND phone_number=$p AND status='pending' AND expires_at <= now()`
  4. kiểm số này không thuộc `owner_user_id`/`genealogy_viewers` (join `users` theo `phone_number`); kiểm chưa có lời mời `pending` chưa hết hạn cho số này
  5. nếu nút chưa có số: kiểm không nút nào khác trong gia phả đang giữ số này, rồi `UPDATE genealogy_members SET phone_number=$p WHERE id=$member` (vi phạm unique `(genealogy_id, phone_number)` → `409 MEMBER_PHONE_ALREADY_USED`, AC60)
  6. `SELECT (SELECT count(*) FROM genealogy_viewers WHERE genealogy_id=$g) + (SELECT count(*) FROM genealogy_invitations WHERE genealogy_id=$g AND status='pending' AND expires_at > now()) < 20`
  7. `INSERT INTO genealogy_invitations (genealogy_id, member_id, phone_number, invited_by, expires_at)`
  8. `COMMIT`
    - Success → `201 Invitation` (`status = "pending"`; **chưa** cấp quyền; response giống nhau dù số đã hay chưa có tài khoản)
    - Failure (đã có vai trò) → `409 VIEWER_ALREADY_EXISTS` (AC46)
    - Failure (đã có lời mời chờ) → `409 INVITATION_ALREADY_PENDING` (AC46)
    - Failure (viewer + lời mời chờ ≥ 20) → `409 VIEWER_LIMIT_REACHED` (AC47)
    - Failure (số đã thuộc nút khác) → `409 MEMBER_PHONE_ALREADY_USED` (AC60)
    - Failure (lỗi DB) → rollback → `500`
      > Thời hạn lời mời 30 ngày (AC75, chốt 2026-10-10) — `config.GenealogyConfig.InvitationTTL` (`invitation_ttl: "720h"`).

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
- Call: `InvitationRepository.Accept` trong **một transaction**: đọc `genealogy_id` của lời mời `WHERE id=$i AND phone_number=$phone` (không có hàng → `404 INVITATION_NOT_FOUND`); `SELECT id FROM genealogies WHERE id=$g FOR UPDATE` (luôn khoá gia phả **trước**, như Step C-2); `SELECT status, expires_at FROM genealogy_invitations WHERE id=$i FOR UPDATE` và kiểm lại; kiểm người gọi chưa là `owner`/`viewer` của gia phả; `SELECT count(*) FROM genealogy_viewers WHERE genealogy_id=$g` < 20; `INSERT INTO genealogy_viewers (genealogy_id, user_id, granted_by)` với `granted_by = invited_by`; `UPDATE genealogy_invitations SET status='accepted', responded_at=now()`; `COMMIT`.
    - Success → `200 {genealogy_id, member_id, role: "viewer"}` (`member_id` = nút của lời mời; số trên nút đã khớp số tài khoản nên `my_member_id` đúng ngay) (hiệu lực ngay ở request kế tiếp, AC54)
    - Failure (không có / không gửi tới số của người gọi) → `404 INVITATION_NOT_FOUND`
    - Failure (không còn `pending` hoặc đã hết hạn) → `409 INVITATION_NOT_PENDING`
    - Failure (người gọi đã có vai trò) → `409 VIEWER_ALREADY_EXISTS`
    - Failure (≥ 20 viewer) → `409 VIEWER_LIMIT_REACHED`
    - Failure (lỗi DB) → rollback → `500`

Route `POST /v1/invitations/{id}/decline` chỉ là Step I-1 + một `UPDATE ... SET status='declined'` có điều kiện (cùng các mã `404`/`409` của Step I-2) nên không có flow riêng; `GET /v1/invitations` (kèm `member_full_name` bằng JOIN), `GET`/`DELETE` lời mời của owner (liệt kê `pending`/`accepted`/`declined`/`expired`) là một truy vấn (cộng Step A, yêu cầu owner, cho các route của owner).

## Step-by-step Flow — Xoá thành viên (DELETE /v1/genealogies/{id}/members/{memberId})

Step D-1 — Phân giải vai trò
- Trigger: handler nhận `DELETE /v1/genealogies/{id}/members/{memberId}`.
- Pre-condition: access token hợp lệ; `{memberId}` parse được thành số dương (không → `404 MEMBER_NOT_FOUND`).
- Call: Step A, yêu cầu role = `owner`.
    - Success → Step D-2
    - Failure → `404 GENEALOGY_NOT_FOUND`; (`viewer`) → `403 GENEALOGY_FORBIDDEN`.

Step D-2 — Kiểm tra và xoá trong một transaction
- Trigger: role = `owner`.
- Pre-condition: không.
- Call: `GenealogyMemberRepository.Delete` trong **một transaction**:
  1. `SELECT kind, parent_id FROM genealogy_members WHERE id=$m AND genealogy_id=$g FOR UPDATE`
  2. nếu `kind='blood'`: `SELECT EXISTS (SELECT 1 FROM genealogy_members WHERE parent_id=$m)`
  3. nếu `kind='blood'`: `DELETE FROM genealogy_members WHERE id=$m AND genealogy_id=$g` — các in-law gắn với nó bị xoá theo bằng `ON DELETE CASCADE` của `spouse_of_id`
  4. nếu `kind='in_law'`: chỉ `DELETE` đúng hàng đó
  5. `COMMIT`
    - Success → `200 {"success": true}`
    - Failure (bước 1 không có hàng) → `404 MEMBER_NOT_FOUND` (AC29, AC37)
    - Failure (bước 2 có con) → rollback → `409 MEMBER_HAS_DESCENDANTS` (AC35, chốt 2026-10-10: chặn, không xoá cả nhánh)
    - Failure (`parent_id IS NULL` và `kind='blood'`) → `409 FOUNDER_NOT_DELETABLE` (AC36)
    - Failure (lỗi DB, kể cả FK `RESTRICT` bắt được con phát sinh song song) → rollback → `500` hoặc `409 MEMBER_HAS_DESCENDANTS` nếu map được vi phạm FK

## Step-by-step Flow — Đổi quan hệ thành viên (PUT /v1/genealogies/{id}/members/{memberId}/relation)

Step R-1 — Phân giải vai trò + validate request
- Trigger: handler nhận `PUT /v1/genealogies/{id}/members/{memberId}/relation`.
- Pre-condition: access token hợp lệ; `{memberId}` parse được thành số dương (không → `404 MEMBER_NOT_FOUND`).
- Call: Step A, yêu cầu role = `owner`; rồi `DecodeJSONBodyStrict` + validate `relation` (`child`|`spouse`), `related_member_id` (số dương), `other_parent_id`/`birth_order` (kiểu, range AC56) (AC80).
    - Success → Step R-2
    - Failure (không có vai trò) → `404 GENEALOGY_NOT_FOUND`; (`viewer`) → `403 GENEALOGY_FORBIDDEN`; (validation) → `400 VALIDATION_ERROR`.

Step R-2 — Kiểm tra và đổi trong một transaction
- Trigger: validate xong.
- Pre-condition: `role` ≠ none.
- Call: `GenealogyMemberRepository.Move` trong **một transaction**:
  1. `SELECT id FROM genealogies WHERE id=$g FOR UPDATE` (tuần tự hoá với Add/Delete/Move cùng gia phả)
  2. `SELECT ... FROM genealogy_members WHERE genealogy_id=$g` — tải cả cây (≤ 5000) để kiểm trong bộ nhớ
  3. Trong bộ nhớ: tìm `target` và `related` (thiếu → `ErrMemberNotFound`); `target` là thuỷ tổ → `ErrFounderNotMovable`; `relation` không khớp `kind` của `target` → validation `relation`; `related` là chính `target` hoặc nằm trong hậu duệ của `target` → validation `related_member_id`; với `child`: `related` phải là blood và `gender='male'` (AC19); với `spouse`: `related` là blood và số phối ngẫu hiện có (không tính `target`) < 5 (AC20); `other_parent_id` (nếu có) là in-law có `spouse_of_id = related` (AC57); `delta = related.generation + 1 − target.generation` và `max(generation hậu duệ) + delta ≤ 60` (AC78)
  4. `UPDATE genealogy_members SET parent_id / spouse_of_id, other_parent_id, birth_order, generation, updated_at WHERE id=$m AND genealogy_id=$g`
  5. nếu `target` là blood và `delta ≠ 0`: `UPDATE genealogy_members SET generation = generation + $delta, updated_at = now() WHERE id = ANY($ids)` với `$ids` = hậu duệ + dâu/rể của `target` và của hậu duệ (tính ở bước 3)
  6. nếu `target` là in-law: `UPDATE genealogy_members SET other_parent_id = NULL WHERE other_parent_id = $m AND genealogy_id=$g` (AC76)
  7. `COMMIT`
    - Success → Step R-3
    - Failure (bước 3 không thấy `target`/`related`) → rollback → `404 MEMBER_NOT_FOUND` (AC79)
    - Failure (`ErrFounderNotMovable`) → rollback → `409 FOUNDER_NOT_MOVABLE` (AC79)
    - Failure (kiểm ở bước 3: sai `relation`, vòng, `related` là nữ/in-law, đủ 5 phối ngẫu, `other_parent_id` sai, đời > 60) → rollback → `400 VALIDATION_ERROR` với `details.field` tương ứng (AC76–AC78)
    - Failure (lỗi DB) → rollback toàn bộ → `500`

Step R-3 — Dựng response
- Trigger: transaction đã commit.
- Pre-condition: có `member` đã cập nhật.
- Call: usecase tính `in_law_label`, `death_lunar_*` và `warnings` như Step M-3 (với người vừa chuyển: so với cha/mẹ mới và các con của nó).
    - Success → `200 Member` + `warnings`
    - Failure → không có truy vấn ghi thêm.

## Step-by-step Flow — Hồ sơ thành viên (GET /v1/genealogies/{id}/members/{memberId}/profile)

Step P-1 — Phân giải vai trò + parse
- Trigger: handler nhận `GET /v1/genealogies/{id}/members/{memberId}/profile`.
- Pre-condition: access token hợp lệ; `{memberId}` parse được thành số dương (không → `404 MEMBER_NOT_FOUND`).
- Call: Step A (mọi vai trò đều được).
    - Success → Step P-2
    - Failure → `404 GENEALOGY_NOT_FOUND`.

Step P-2 — Tải cây và số điện thoại của người gọi (chỉ đọc)
- Trigger: có vai trò (`owner` hoặc `viewer`).
- Pre-condition: không.
- Call: `GenealogyMemberRepository.List(ctx, genealogyID)` (tất cả thành viên, ≤ 5000) rồi `UserPhoneLookup.FindPhoneByID(ctx, userID)` (số điện thoại tài khoản người gọi), **trong cùng một transaction `READ ONLY` mức `REPEATABLE READ`** để hai đọc nhìn cùng một ảnh chụp (không thấy nửa vời khi người khác sửa đúng lúc).
    - Success → Step P-3
    - Failure (lỗi DB) → `500`.

Step P-3 — Tính trong bộ nhớ (hàm thuần, không I/O)
- Trigger: đã có danh sách thành viên và số điện thoại của người gọi.
- Pre-condition: không.
- Call: `BuildIndex`; `my_member_id` = id thành viên có `phone_number` trùng số của người gọi, hoặc `null` (AC59); kiểm `memberId` có trong chỉ mục; rồi `Age` và `TraditionalAge` (AC65, tuổi tây và tuổi mụ), `Relatives` (AC66), `Lineage` (AC67), `Chi` (AC68); `Kinship` (AC69) **chỉ khi** `my_member_id` ≠ null.
    - Success → `200 MemberProfile`
    - Failure (`memberId` không có trong chỉ mục) → `404 MEMBER_NOT_FOUND` (không lộ thành viên của gia phả khác, vì chỉ mục chỉ chứa gia phả này)
    - Không có thành viên nào trùng số của người gọi → `my_member_id = null`, `relation_to_me = null`, không lỗi (AC63).

### Quy tắc chi (`Chi` trong `genealogy_kinship.go`, AC68)

Chi nằm ở chỗ **cây rẽ nhánh** và lồng nhau, không cố định theo đời. Thuật toán (một lượt đi ngược `parent_id`, ≤ 60 bước, dùng chỉ mục đã dựng ở Step P-3):

1. Với in-law, thay bằng người dòng họ được gắn (`spouse_of_id`) rồi tính như bên dưới.
2. Dựng đường `root → ... → X` bằng cách đi ngược `parent_id` từ `X`.
3. Với mỗi nút `N` trên đường (trừ thuỷ tổ) có cha/mẹ `P`: `N` là **đầu chi** khi `N.gender = male` và `P` có ≥ 2 con trai (đếm trên chỉ mục: các thành viên blood có `parent_id = P` và `gender = male`).
4. `chi_path` = các đầu chi theo thứ tự đời tăng dần, mỗi phần tử `{ancestor = N, rank, is_eldest}` với `rank` = vị trí của `N` trong các con trai của `P` sắp theo `(birth_order, id)`. `chi` = phần tử cuối, hoặc `null` khi `chi_path` rỗng.

Ví dụ: thuỷ tổ T có hai con trai Lộc và Nghĩa; Lộc có hai con trai A và B; A có một con C. Khi đó `chi_path(C) = [Lộc, A]` và `chi(C) = A`; `chi_path(Nghĩa) = [Nghĩa]`; `chi_path(T) = []`. Bảng test cần các ca: chuỗi con một (không rẽ nhánh), rẽ nhánh nhiều tầng, con gái (không đầu chi, thuộc chi của cha), in-law, thuỷ tổ, thêm con trai thứ hai làm nhãn đổi (hành vi đã nêu ở AC68).

### Bảng xưng hô v1 (đầu vào của `Kinship`)

> Đã được anh Đức duyệt 2026-10-10 (kèm quy tắc vai vế và các ca xa bên dưới).

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

### Quy tắc vai vế (`Kinship`, AC69)

Vai vế theo **đời**, không theo năm sinh (người nhỏ tuổi nhưng đời trên vẫn là vai trên). Gọi `A` = tổ tiên chung gần nhất của `M` và `T` (là chính `M` hoặc `T` nếu người này là tổ tiên của người kia); `dM = M.generation − A.generation`, `dT = T.generation − A.generation`. Với dâu/rể thì thay bằng người dòng họ được gắn rồi áp "vợ/chồng của".

- **Vai**: `dT < dM` → vai trên (bậc `dM − dT`); `dT = dM` → ngang vai; `dT > dM` → vai dưới.
- **Lớn/nhỏ**: chỉ so khi cần phân biệt anh/em hay bác/chú. Lấy hai người con trực tiếp của `A` nằm trên đường tới `M` và tới `T` (khi `A` là chính `M` hoặc `T`, không cần so), so `birth_order` rồi `id` của hai người đó. Không dùng `birth_year`. Vì vậy con của bác là "anh/chị" dù nhỏ tuổi hơn con của chú.
- **"Họ"**: thêm hậu tố "họ" vào `label` (và `_HO` vào `code`) khi `min(dM, dT) ≥ 2` và `T` không phải tổ tiên/hậu duệ trực hệ — tức tổ tiên chung cách **cả hai** bên từ hai đời trở lên (anh/em họ, chú/bác/cô họ, cháu họ). Anh/chị/em ruột, chú bác cô ruột, cháu ruột (`min = 1`) không có "họ". (Làm rõ ở v5.3 khi cài đặt; trước đó ghi `dT ≥ 2`, sẽ gán "họ" cho cả cháu ruột.)
- **Ca xa**: vai trên bậc 1 với `dT ≥ 1` → bác/chú/cô (theo giới tính và so với người nhánh `M`); bậc 2 với `dT ≥ 1` → ông bác/ông chú/bà cô; bậc ≥ 3 với `dT ≥ 1` → `UNKNOWN`. Tổ tiên trực hệ (`dT = 0`): bậc 1 cha, 2 ông nội, 3 cụ, 4 kỵ, ≥ 5 "Tổ đời N". Vai dưới: `dM = 0` → con (1), cháu nội (2), chắt (3), chút (4), xa hơn `UNKNOWN`; `dM ≥ 1` → cháu (`dT − dM ≤ 2`, thêm "họ" khi `dM ≥ 2`), xa hơn `UNKNOWN`.
- Bên ngoại (đi qua dâu/rể lên cha mẹ của họ) vẫn `UNKNOWN` (Out of scope).

Thứ tự lớn/nhỏ giữa hai người cùng cha: so `birth_order` rồi `id`. Giữa `T` (anh/em của cha) và cha: cùng cách so sánh. `birth_order` trùng/không có thì dùng `id`.

## Deliberately-stubbed-for-now

Không có route nào bị stub: toàn bộ 21 route đều có contract ở `requirements.md`. Phần hoãn:

- Ảnh hoành phi/ảnh thành viên — chờ GCS, không có route/cột trong v1.
- Thông báo (push notification) — **tính năng riêng của ứng dụng, làm sau**; không dùng SMS. v1 chỉ có hộp lời mời `GET /v1/invitations` (client tự gọi).
- Chống ghi đè đồng thời (`Pending`) và đổi **loại** thành viên dòng họ ↔ dâu/rể (AC80) — chưa chốt, chưa thiết kế.
- Nhắc ngày chạp/ngày giỗ hằng năm và tính ngày dương của năm sau (cần quy đổi âm → dương; thuộc `memorial`) — server chỉ quy đổi dương → âm cho ngày đã lưu.
- Xưng hô bên ngoại (cậu, dì, ông bà ngoại): cần cho phép dâu/rể có cha mẹ trong cây — hiện `Kinship` trả `UNKNOWN`.
- Route sắp xếp lại thứ tự anh em (AC56): v1 chỉ sửa `birth_order` từng người.
- Liên kết ngày giỗ với `memorial`: **không làm** (chốt 2026-10-10) — ngày giỗ chỉ lưu `death_month/death_day/death_entered_as_lunar` ngay trên thành viên (dương lịch + cờ, AC55).
