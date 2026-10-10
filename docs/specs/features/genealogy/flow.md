# Flow — genealogy

Version: 5.5
Last updated: 2026-10-10
Status: not started — sơ đồ dự kiến, mirror `design.md` v5.5.
Module: `genealogy`

> See `changelog.md` in this folder for the version history of this spec.

Bản vẽ của bảy section "Step-by-step Flow" trong `design.md` (cùng tên, cùng thứ tự). Mỗi message trace 1:1 về một Step (kể cả Step A dùng chung); không thêm/bỏ bước.

## Sequence — Tạo gia phả (POST /v1/genealogies)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as authMiddleware
    participant H as GenealogyHandler<br/>(delivery/http)
    participant U as GenealogyUsecase
    participant R as GenealogyRepository
    participant DB as Postgres

    C->>M: POST /v1/genealogies + Bearer access token
    alt token không hợp lệ
        M--xC: 401
    end
    M->>H: userID trong ctx
    H->>H: DecodeJSONBodyStrict (Step G-1)
    H->>U: Create(ctx, userID, input)
    U->>U: validate AC3–AC9, AC11 (Step G-1)
    alt validation lỗi
        U--xH: validation error
        H-->>C: 400 VALIDATION_ERROR {details.field}
    end
    U->>R: Create(ctx, g, founder, spouses, max=10)

    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction (Step G-2)
        R->>DB: BEGIN
        R->>DB: SELECT id FROM users WHERE id=owner FOR UPDATE
        R->>DB: SELECT count(*) FROM genealogies WHERE owner_user_id=owner
        alt count >= 10
            R--xU: ErrGenealogyLimitReached → ROLLBACK
        else còn chỗ
            R->>DB: INSERT genealogies RETURNING id
            R->>DB: INSERT genealogy_chronicles (rỗng)
            R->>DB: INSERT genealogy_members thuỷ tổ (blood, đời 1)
            R->>DB: INSERT genealogy_members mỗi spouse (in_law, spouse_of = thuỷ tổ)
            R->>DB: COMMIT
        end
    end

    R->>DB: SELECT detail + stats (Step G-3)
    R-->>U: GenealogyDetail
    alt thành công
        U-->>H: GenealogyDetail
        H-->>C: 201 GenealogyDetail (my_role = owner)
    else đủ 10 gia phả
        H-->>C: 409 GENEALOGY_LIMIT_REACHED
    else lỗi DB
        H-->>C: 500 INTERNAL_ERROR
    end
```

## Sequence — Thêm thành viên (POST /v1/genealogies/{id}/members)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as authMiddleware
    participant H as GenealogyMemberHandler<br/>(delivery/http)
    participant U as GenealogyMemberUsecase
    participant R as GenealogyRepository /<br/>GenealogyMemberRepository
    participant DB as Postgres

    C->>M: POST /v1/genealogies/{id}/members + Bearer
    alt token không hợp lệ
        M--xC: 401
    end
    M->>H: userID trong ctx
    H->>U: Add(ctx, userID, genealogyID, input)
    U->>R: ResolveRole(genealogyID, userID) (Step A / M-1)
    R->>DB: SELECT role FROM genealogies LEFT JOIN genealogy_viewers
    alt không có vai trò
        R--xU: ErrGenealogyNotFound
        H-->>C: 404 GENEALOGY_NOT_FOUND
    else role = viewer
        H-->>C: 403 GENEALOGY_FORBIDDEN (AC81)
    end
    H->>H: DecodeJSONBodyStrict + validate AC4–AC8, AC18, AC21, AC26 (Step M-1)
    alt validation lỗi
        H-->>C: 400 VALIDATION_ERROR {details.field}
    end
    U->>R: Add(ctx, genealogyID, relation, relatedID, member, limits)
    Note over H,R: thuỷ tổ phải nam (AC4, Step G-1); con của nữ dòng họ bị chặn (AC19, Step M-2.2)

    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction (Step M-2)
        R->>DB: BEGIN
        R->>DB: SELECT id FROM genealogies WHERE id=g FOR UPDATE
        R->>DB: SELECT kind, generation, gender FROM members WHERE id=related AND genealogy_id=g
        alt không có hàng
            R--xU: ErrMemberNotFound → ROLLBACK
        end
        R->>DB: SELECT count(*) FROM members WHERE genealogy_id=g
        opt is_clan_head = true
            R->>DB: UPDATE members SET is_clan_head=false WHERE genealogy_id=g AND is_clan_head (AC82)
        end
        opt relation = spouse
            R->>DB: SELECT count(*) FROM members WHERE spouse_of_id=related
        end
        opt relation = child và có other_parent_id
            R->>DB: SELECT 1 FROM members WHERE id=other AND genealogy_id=g AND kind='in_law' AND spouse_of_id=related
            alt không có hàng
                R--xU: validation other_parent_id → ROLLBACK
            end
        end
        opt relation = child và không gửi birth_order
            R->>DB: SELECT coalesce(max(birth_order),0)+1 FROM members WHERE parent_id=related
        end
        alt vi phạm giới hạn / related là in-law / related là nữ dòng họ (child) / đời > 60
            R--xU: validation hoặc ErrMemberLimitReached → ROLLBACK
        else hợp lệ
            R->>DB: INSERT genealogy_members RETURNING id, created_at, updated_at
            R->>DB: COMMIT
        end
    end

    R-->>U: Member
    U->>U: tính in_law_label từ gender người được gắn và death_lunar_* bằng SolarToLunar (Step M-3, AC55)
    U->>U: tính warnings: birth_year con < birth_year cha/mẹ → CHILD_BORN_BEFORE_PARENT (Step M-3, AC27; không bao giờ là lỗi)
    alt thành công
        U-->>H: Member + warnings
        H-->>C: 201 Member + warnings
    else failure
        U--xH: sentinel error
        H-->>C: 400 / 403 / 404 MEMBER_NOT_FOUND / 409 MEMBER_LIMIT_REACHED / MEMBER_PHONE_ALREADY_USED / 500
    end
```

## Sequence — Mời người xem bằng số điện thoại (POST /v1/genealogies/{id}/invitations)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as authMiddleware
    participant H as GenealogyAccessHandler<br/>(delivery/http)
    participant U as GenealogyAccessUsecase
    participant R as InvitationRepository
    participant DB as Postgres

    C->>M: POST /v1/genealogies/{id}/invitations {member_id, phone_number?} + Bearer
    alt token không hợp lệ
        M--xC: 401
    end
    M->>H: userID trong ctx
    H->>U: Invite(ctx, userID, genealogyID, phone)
    U->>R: ResolveRole (Step A / C-1)
    R->>DB: SELECT role ...
    alt không có vai trò
        H-->>C: 404 GENEALOGY_NOT_FOUND
    else role ≠ owner
        H-->>C: 403 GENEALOGY_FORBIDDEN
    end
    U->>U: validate member_id + chuẩn hoá phone_number nếu có (hàm của register) (Step C-1)
    Note over U: Không tra user ở bước này — response không phụ thuộc số đã đăng ký (AC45)
    alt sai định dạng
        H-->>C: 400 VALIDATION_ERROR {member_id / phone_number}
    end
    U->>R: Create(ctx, genealogyID, memberID, phone, invitedBy, max=20)

    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction (Step C-2)
        R->>DB: BEGIN
        R->>DB: SELECT id FROM genealogies WHERE id=g FOR UPDATE
        R->>DB: SELECT is_alive, phone_number FROM members WHERE id=member AND genealogy_id=g
        alt không có nút / nút đã mất / số thiếu hoặc khác số trên nút
            R--xU: ErrMemberNotFound / validation → ROLLBACK
        end
        R->>DB: UPDATE genealogy_invitations SET status='expired' (lời mời cũ của số này đã quá hạn)
        R->>DB: kiểm số này không thuộc owner/viewer (join users theo phone_number)
        R->>DB: kiểm chưa có lời mời pending chưa hết hạn cho số này
        opt nút chưa có phone_number
            R->>DB: UPDATE members SET phone_number=p WHERE id=member (unique (genealogy_id, phone_number))
        end
        R->>DB: SELECT count(viewer) + count(lời mời pending chưa hết hạn)
        alt đã có vai trò
            R--xU: ErrViewerExists → ROLLBACK
        else đã có lời mời chờ
            R--xU: ErrInvitationAlreadyPending → ROLLBACK
        else đủ 20 (viewer + lời mời chờ)
            R--xU: ErrViewerLimitReached → ROLLBACK
        else hợp lệ
            R->>DB: INSERT genealogy_invitations (member_id, expires_at = now + 30 ngày)
            R->>DB: COMMIT
        end
    end

    alt thành công
        U-->>H: Invitation
        H-->>C: 201 Invitation (chưa cấp quyền; giống nhau dù số đã hay chưa đăng ký)
    else failure
        H-->>C: 400 / 404 MEMBER_NOT_FOUND / 409 VIEWER_ALREADY_EXISTS / INVITATION_ALREADY_PENDING / VIEWER_LIMIT_REACHED / MEMBER_PHONE_ALREADY_USED / 500
    end
```

## Sequence — Chấp nhận lời mời (POST /v1/invitations/{id}/accept)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as authMiddleware
    participant H as GenealogyInvitationHandler<br/>(delivery/http)
    participant U as GenealogyInvitationUsecase
    participant L as UserPhoneLookup
    participant R as InvitationRepository
    participant DB as Postgres

    C->>M: POST /v1/invitations/{id}/accept + Bearer
    alt token không hợp lệ
        M--xC: 401
    end
    M->>H: userID trong ctx
    alt {id} không phải số dương
        H-->>C: 404 INVITATION_NOT_FOUND (không chạm DB)
    end
    H->>U: Accept(ctx, userID, invitationID)
    U->>L: FindPhoneByID(userID) (Step I-1)
    L->>DB: SELECT phone_number FROM users WHERE id=$1
    Note over U,R: Không qua Step A — người gọi chưa có vai trò trong gia phả
    U->>R: Accept(ctx, invitationID, phone, userID, max=20)

    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction (Step I-2) — khoá gia phả trước, lời mời sau
        R->>DB: BEGIN
        R->>DB: SELECT genealogy_id FROM genealogy_invitations WHERE id=i AND phone_number=phone
        alt không có hàng
            R--xU: ErrInvitationNotFound → ROLLBACK
        end
        R->>DB: SELECT id FROM genealogies WHERE id=g FOR UPDATE
        R->>DB: SELECT status, expires_at FROM genealogy_invitations WHERE id=i FOR UPDATE
        alt không còn pending hoặc hết hạn
            R--xU: ErrInvitationNotPending → ROLLBACK
        end
        R->>DB: kiểm người gọi chưa là owner/viewer của gia phả
        R->>DB: SELECT count(*) FROM genealogy_viewers WHERE genealogy_id=g
        alt đã có vai trò
            R--xU: ErrViewerExists → ROLLBACK
        else đủ 20 viewer
            R--xU: ErrViewerLimitReached → ROLLBACK
        else hợp lệ
            R->>DB: INSERT genealogy_viewers (granted_by = invited_by)
            R->>DB: UPDATE genealogy_invitations SET status='accepted', responded_at=now()
            R->>DB: COMMIT
        end
    end

    alt thành công
        U-->>H: genealogyID
        H-->>C: 200 {genealogy_id, member_id, role: "viewer"} (hiệu lực ở request kế tiếp)
    else failure
        H-->>C: 404 INVITATION_NOT_FOUND / 409 INVITATION_NOT_PENDING / VIEWER_ALREADY_EXISTS / VIEWER_LIMIT_REACHED / 500
    end
```

## Sequence — Xoá thành viên (DELETE /v1/genealogies/{id}/members/{memberId})

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as authMiddleware
    participant H as GenealogyMemberHandler<br/>(delivery/http)
    participant U as GenealogyMemberUsecase
    participant R as GenealogyMemberRepository
    participant DB as Postgres

    C->>M: DELETE /v1/genealogies/{id}/members/{memberId} + Bearer
    alt token không hợp lệ
        M--xC: 401
    end
    M->>H: userID trong ctx
    H->>U: Delete(ctx, userID, genealogyID, memberID)
    U->>R: ResolveRole (Step A / D-1)
    R->>DB: SELECT role ...
    alt không có vai trò
        H-->>C: 404 GENEALOGY_NOT_FOUND
    else role = viewer
        H-->>C: 403 GENEALOGY_FORBIDDEN (AC81)
    end
    U->>R: Delete(ctx, genealogyID, memberID)

    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction (Step D-2)
        R->>DB: BEGIN
        R->>DB: SELECT kind, parent_id FROM members WHERE id=m AND genealogy_id=g FOR UPDATE
        alt không có hàng
            R--xU: ErrMemberNotFound → ROLLBACK
        else blood và parent_id IS NULL
            R--xU: ErrFounderNotDeletable → ROLLBACK
        end
        opt kind = blood
            R->>DB: SELECT EXISTS (con có parent_id = m)
        end
        alt có con
            R--xU: ErrMemberHasDescendants → ROLLBACK
        else xoá được
            R->>DB: DELETE FROM members WHERE id=m AND genealogy_id=g (in-law gắn theo CASCADE)
            R->>DB: COMMIT
        end
    end

    alt thành công
        U-->>H: ok
        H-->>C: 200 {"success": true}
    else failure
        H-->>C: 403 / 404 MEMBER_NOT_FOUND / 409 FOUNDER_NOT_DELETABLE / 409 MEMBER_HAS_DESCENDANTS / 500
    end
```

## Sequence — Đổi quan hệ thành viên (PUT /v1/genealogies/{id}/members/{memberId}/relation)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as authMiddleware
    participant H as GenealogyMemberHandler<br/>(delivery/http)
    participant U as GenealogyMemberUsecase
    participant R as GenealogyMemberRepository
    participant DB as Postgres

    C->>M: PUT /v1/genealogies/{id}/members/{memberId}/relation + Bearer
    alt token không hợp lệ
        M--xC: 401
    end
    M->>H: userID trong ctx
    H->>U: Move(ctx, userID, genealogyID, memberID, input)
    U->>R: ResolveRole (Step A / R-1)
    R->>DB: SELECT role ...
    alt không có vai trò
        H-->>C: 404 GENEALOGY_NOT_FOUND
    else role = viewer
        H-->>C: 403 GENEALOGY_FORBIDDEN (AC81)
    end
    H->>H: DecodeJSONBodyStrict + validate relation, related_member_id, other_parent_id, birth_order (Step R-1, AC80)
    alt validation lỗi
        H-->>C: 400 VALIDATION_ERROR {details.field}
    end
    U->>R: Move(ctx, genealogyID, memberID, relation, relatedID, otherParentID, birthOrder, limits)

    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction (Step R-2)
        R->>DB: BEGIN
        R->>DB: SELECT id FROM genealogies WHERE id=g FOR UPDATE
        R->>DB: SELECT ... FROM members WHERE genealogy_id=g (tải cả cây)
        R->>R: kiểm trong bộ nhớ: target/related tồn tại, không phải thuỷ tổ, relation khớp kind, không vòng, related là nam dòng họ (child) / ≤ 5 phối ngẫu (spouse), other_parent_id hợp lệ, đời ≤ 60
        alt không thấy target/related
            R--xU: ErrMemberNotFound → ROLLBACK
        else target là thuỷ tổ
            R--xU: ErrFounderNotMovable → ROLLBACK
        else vi phạm quy tắc
            R--xU: validation {details.field} → ROLLBACK
        else hợp lệ
            R->>DB: UPDATE members SET parent_id/spouse_of_id, other_parent_id, birth_order, generation WHERE id=m
            opt target là blood và delta generation ≠ 0
                R->>DB: UPDATE members SET generation = generation + delta WHERE id = ANY(hậu duệ + dâu/rể)
            end
            opt target là in-law
                R->>DB: UPDATE members SET other_parent_id = NULL WHERE other_parent_id = m
            end
            R->>DB: COMMIT
        end
    end

    R-->>U: Member
    U->>U: tính in_law_label + death_lunar_* + warnings (Step R-3, AC27, AC55)
    alt thành công
        U-->>H: Member + warnings
        H-->>C: 200 Member + warnings
    else failure
        H-->>C: 400 VALIDATION_ERROR / 403 / 404 MEMBER_NOT_FOUND / 409 FOUNDER_NOT_MOVABLE / 500
    end
```

## Sequence — Hồ sơ thành viên (GET /v1/genealogies/{id}/members/{memberId}/profile)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as authMiddleware
    participant H as GenealogyProfileHandler<br/>(delivery/http)
    participant U as GenealogyProfileUsecase
    participant K as genealogy_kinship<br/>(hàm thuần)
    participant R as GenealogyRepository /<br/>GenealogyMemberRepository
    participant DB as Postgres

    C->>M: GET /v1/genealogies/{id}/members/{memberId}/profile + Bearer
    alt token không hợp lệ
        M--xC: 401
    end
    M->>H: userID trong ctx
    H->>H: parse {id}, {memberId} (Step P-1)
    alt memberId không phải số dương
        H-->>C: 404 MEMBER_NOT_FOUND
    end
    H->>U: GetProfile(ctx, userID, genealogyID, memberID)
    U->>R: ResolveRole (Step A / P-1)
    R->>DB: SELECT role ...
    alt không có vai trò
        H-->>C: 404 GENEALOGY_NOT_FOUND
    end

    rect rgb(235, 240, 255)
        Note over R,DB: Một transaction READ ONLY, REPEATABLE READ (Step P-2)
        U->>R: List(ctx, genealogyID)
        R->>DB: SELECT * FROM genealogy_members WHERE genealogy_id=g
        U->>R: FindPhoneByID(ctx, userID) (UserPhoneLookup)
        R->>DB: SELECT phone_number FROM users WHERE id=u
    end

    U->>K: BuildIndex(members) (Step P-3); my_member_id = thành viên có phone_number trùng số của người gọi, hoặc null (AC59)
    alt memberId không có trong chỉ mục
        U--xH: ErrMemberNotFound
        H-->>C: 404 MEMBER_NOT_FOUND
    end
    U->>K: Age, Relatives, Lineage, Chi (Step P-3)
    opt my_member_id ≠ null
        U->>K: Kinship(meID, targetID)
    end
    K-->>U: MemberProfile
    alt thành công
        U-->>H: MemberProfile
        H-->>C: 200 MemberProfile
    else lỗi DB
        H-->>C: 500 INTERNAL_ERROR
    end
```
