# Flow — temple-announcement

Version: 1.2
Last updated: 2026-10-14
Status: implemented (code + test pass; OKF concept chưa viết, xem `tasks.md`)
Module: `temple-announcement`

> See `changelog.md` in this folder for the version history of this spec.

Bản vẽ của hai section "Step-by-step Flow" trong `design.md`; mỗi mũi tên trace 1:1 về một Step.

## Sequence — Đọc thông báo của tôi (appService)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as TempleAnnouncementHandler<br/>(delivery/http)
    participant U as TempleAnnouncementUsecase
    participant R as Repository
    participant DB as Postgres

    C->>H: GET /v1/users/me/temple-announcements?page&page_size (JWT access)
    H->>H: R-1 Parse page, page_size (mặc định + chặn max)
    alt không phải số, page hoặc page_size < 1
        H--xC: 400 INVALID_PAGE
    end
    H->>U: ListForUser(userID, page, pageSize)
    U->>R: R-2 GetSettings(userID)
    R->>DB: SELECT receive, ordination temple (id, name) FROM users LEFT JOIN temples
    DB-->>R: receive, templeID
    R-->>U: settings
    alt R-3 receive = false
        U-->>H: {receive:false, temple:null, items:[]}
    else R-4 không có chùa
        U-->>H: {receive:true, temple:null, items:[]}
    else R-5 có chùa
        U->>R: ListActiveByTemple(templeID, now, page)
        R->>DB: COUNT(*) rồi SELECT ... ORDER BY published_at DESC, id LIMIT/OFFSET
        DB-->>R: tổng + trang
        R-->>U: AnnouncementPage
        U-->>H: {receive:true, temple, items, page}
    end
    H-->>C: R-6 200 (DTO)
```

## Sequence — Đăng thông báo (backofficeService)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client (backoffice)
    participant H as TempleAnnouncementHandler<br/>(delivery/http)
    participant U as TempleAnnouncementUsecase
    participant R as AnnouncementRepository
    participant DB as Postgres

    C->>H: POST /v1/admin/temples/{id}/announcements (token admin)
    H->>H: P-1 Middleware: access token, session, role admin/content_editor
    alt không đủ quyền
        H--xC: 401 / 403
    end
    H->>H: P-2 Strict decode body
    alt body sai
        H--xC: 400 VALIDATION_ERROR
    end
    H->>U: Publish(templeID, input, adminID)
    U->>U: P-3 Kiểm title/body/expires_at theo config
    alt không hợp lệ
        U--xC: 400 INVALID_ANNOUNCEMENT (details.field)
    end
    U->>R: P-4 TempleExists(templeID)
    R->>DB: SELECT 1 FROM temples
    alt không có chùa
        U--xC: 404 TEMPLE_NOT_FOUND
    end
    U->>R: P-5 IssuingBodyBelongs(issuingBodyID, templeID)
    R->>DB: SELECT 1 FROM temple_issuing_bodies
    alt không thuộc chùa
        U--xC: 400 INVALID_ISSUING_BODY
    end
    U->>R: P-6 Insert(announcement, source=admin)
    R->>DB: INSERT INTO temple_announcements
    DB-->>R: ok
    U-->>H: announcement
    H-->>C: 201
```
