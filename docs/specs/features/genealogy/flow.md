# Flow — genealogy

Version: 3.0
Last updated: 2026-10-08
Status: not started — sơ đồ dự kiến, mirror `design.md` v3.0.
Module: `genealogy`

> See `changelog.md` in this folder for the version history of this spec.

Bản vẽ của sáu section "Step-by-step Flow" trong `design.md` (cùng tên, cùng thứ tự). Mỗi message trace 1:1 về một Step (kể cả Step A dùng chung); không thêm/bỏ bước.

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
    Note over R,DB: ⚠️ Not confirmed: trần 10 gia phả/người tạo là con số đề xuất

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
    R->>DB: SELECT role FROM genealogies LEFT JOIN genealogy_collaborators
    alt không có vai trò
        R--xU: ErrGenealogyNotFound
        H-->>C: 404 GENEALOGY_NOT_FOUND
    end
    H->>H: DecodeJSONBodyStrict + validate AC4–AC8, AC18, AC21, AC26 (Step M-1)
    alt validation lỗi
        H-->>C: 400 VALIDATION_ERROR {details.field}
    end
    U->>R: Add(ctx, genealogyID, relation, relatedID, member, limits)

    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction (Step M-2)
        R->>DB: BEGIN
        R->>DB: SELECT id FROM genealogies WHERE id=g FOR UPDATE
        R->>DB: SELECT kind, generation, gender FROM members WHERE id=related AND genealogy_id=g
        alt không có hàng
            R--xU: ErrMemberNotFound → ROLLBACK
        end
        R->>DB: SELECT count(*) FROM members WHERE genealogy_id=g
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
        alt vi phạm giới hạn / related là in-law / đời > 60
            R--xU: validation hoặc ErrMemberLimitReached → ROLLBACK
        else hợp lệ
            R->>DB: INSERT genealogy_members RETURNING id, created_at, updated_at
            R->>DB: COMMIT
        end
    end
    Note over R,DB: ⚠️ Not confirmed: trần 5000 thành viên, 60 đời; con của thành viên nữ

    R-->>U: Member
    U->>U: tính in_law_label từ gender người được gắn (Step M-3)
    alt thành công
        U-->>H: Member
        H-->>C: 201 Member
    else failure
        U--xH: sentinel error
        H-->>C: 400 / 404 MEMBER_NOT_FOUND / 409 MEMBER_LIMIT_REACHED / 500
    end
```

## Sequence — Mời người cùng sửa bằng số điện thoại (POST /v1/genealogies/{id}/collaborators)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as authMiddleware
    participant H as GenealogyAccessHandler<br/>(delivery/http)
    participant U as GenealogyAccessUsecase
    participant R as InvitationRepository
    participant DB as Postgres

    C->>M: POST /v1/genealogies/{id}/collaborators {phone_number} + Bearer
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
    U->>U: chuẩn hoá phone_number (hàm của register) (Step C-1)
    Note over U: Không tra user ở bước này — response không phụ thuộc số đã đăng ký (AC45)
    alt sai định dạng
        H-->>C: 400 VALIDATION_ERROR {phone_number}
    end
    U->>R: Create(ctx, genealogyID, phone, invitedBy, max=20)

    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction (Step C-2)
        R->>DB: BEGIN
        R->>DB: SELECT id FROM genealogies WHERE id=g FOR UPDATE
        R->>DB: UPDATE genealogy_invitations SET status='expired' (lời mời cũ của số này đã quá hạn)
        R->>DB: kiểm số này không thuộc owner/clan_head/editor (join users theo phone_number)
        R->>DB: kiểm chưa có lời mời pending chưa hết hạn cho số này
        R->>DB: SELECT count(editor) + count(lời mời pending chưa hết hạn)
        alt đã có vai trò
            R--xU: ErrCollaboratorExists → ROLLBACK
        else đã có lời mời chờ
            R--xU: ErrInvitationAlreadyPending → ROLLBACK
        else đủ 20 (editor + lời mời chờ)
            R--xU: ErrCollaboratorLimitReached → ROLLBACK
            Note over R,DB: ⚠️ Not confirmed: trần 20 và việc tính lời mời chờ (AC47)
        else hợp lệ
            R->>DB: INSERT genealogy_invitations (expires_at = now + 30 ngày)
            Note over R,DB: ⚠️ Not confirmed: thời hạn 30 ngày (AC75)
            R->>DB: COMMIT
        end
    end

    alt thành công
        U-->>H: Invitation
        H-->>C: 201 Invitation (chưa cấp quyền; giống nhau dù số đã hay chưa đăng ký)
    else failure
        H-->>C: 409 COLLABORATOR_ALREADY_EXISTS / INVITATION_ALREADY_PENDING / COLLABORATOR_LIMIT_REACHED / 500
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
        R->>DB: kiểm người gọi chưa là owner/clan_head/editor
        R->>DB: SELECT count(*) FROM genealogy_collaborators WHERE genealogy_id=g
        alt đã có vai trò
            R--xU: ErrCollaboratorExists → ROLLBACK
        else đủ 20 editor
            R--xU: ErrCollaboratorLimitReached → ROLLBACK
        else hợp lệ
            R->>DB: INSERT genealogy_collaborators (granted_by = invited_by)
            R->>DB: UPDATE genealogy_invitations SET status='accepted', responded_at=now()
            R->>DB: COMMIT
        end
    end

    alt thành công
        U-->>H: genealogyID
        H-->>C: 200 {genealogy_id, role: "editor"} (hiệu lực ở request kế tiếp)
    else failure
        H-->>C: 404 INVITATION_NOT_FOUND / 409 INVITATION_NOT_PENDING / COLLABORATOR_ALREADY_EXISTS / COLLABORATOR_LIMIT_REACHED / 500
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
            Note over R,DB: ⚠️ Not confirmed: chặn xoá hay xoá cả nhánh (AC35)
        else xoá được
            R->>DB: DELETE FROM members WHERE id=m AND genealogy_id=g (in-law gắn theo CASCADE)
            R->>DB: COMMIT
        end
    end

    alt thành công
        U-->>H: ok
        H-->>C: 200 {"success": true}
    else failure
        H-->>C: 404 MEMBER_NOT_FOUND / 409 FOUNDER_NOT_DELETABLE / 409 MEMBER_HAS_DESCENDANTS / 500
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
        U->>R: FindMyMemberID(ctx, genealogyID, userID)
        R->>DB: SELECT member_id FROM genealogy_user_members WHERE genealogy_id=g AND user_id=u
    end

    U->>K: BuildIndex(members) (Step P-3)
    alt memberId không có trong chỉ mục
        U--xH: ErrMemberNotFound
        H-->>C: 404 MEMBER_NOT_FOUND
    end
    U->>K: Age, Relatives, Lineage, Chi (Step P-3)
    opt my_member_id ≠ null
        U->>K: Kinship(meID, targetID)
        Note over U,K: ⚠️ Not confirmed: bảng xưng hô v1 chưa được duyệt (AC69)
    end
    K-->>U: MemberProfile
    alt thành công
        U-->>H: MemberProfile
        H-->>C: 200 MemberProfile
    else lỗi DB
        H-->>C: 500 INTERNAL_ERROR
    end
```
