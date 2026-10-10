# Flow — temple-suggestion

Version: 2.4
Last updated: 2026-10-10
Status: sơ đồ cho phần duyệt (backoffice) — chưa có code; phần gửi đề xuất ở appService đã implement và không có sơ đồ. Không còn marker `⚠️ Not confirmed`.
Module: `content` + `backoffice`

> See `changelog.md` in this folder for the version history of this spec.

File này là bản vẽ của section "Step-by-step Flow" duy nhất trong `design.md`; mỗi mũi tên trace 1:1 về một Step C-1..C-6. Route gửi đề xuất và hai route đọc chỉ một câu SQL nên không có sơ đồ (xem "Request flow" trong `design.md`).

## Sequence — Tạo chùa từ đề xuất (POST /v1/admin/temple-suggestions/{id}/create-temple)

```mermaid
sequenceDiagram
    autonumber
    participant C as Admin (backoffice client)
    participant M as authMiddleware +<br/>RequireActiveSession + RequireAnyRole
    participant H as TempleSuggestionHandler<br/>(backofficeService/delivery/http)
    participant U as TempleSuggestionUsecase
    participant R as TempleSuggestionRepository
    participant DB as Postgres

    C->>M: POST /v1/admin/temple-suggestions/{id}/create-temple
    Note over M: C-1 — xác thực token + role
    alt token sai / bị thu hồi
        M--xC: 401
    else role không được phép
        M--xC: 403
    end
    M->>H: claims (admin id) trong context
    H->>H: C-2 — parse {id}, DecodeJSONBodyStrict
    H->>U: CreateTemple(ctx, suggestionID, adminID, input)
    U->>U: C-2 — validate NewTemple, suy heritage_priority
    alt {id} sai / body sai
        U--xH: 404 / 400 VALIDATION_ERROR (chưa chạm DB)
        H-->>C: 404 / 400
    end
    U->>R: CreateTemple(ctx, suggestionID, adminID, role, newTemple)

    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction (C-3 → C-6)
        R->>DB: C-3 — SELECT … FROM temple_suggestions WHERE id=$1 FOR UPDATE
        alt không có dòng
            R--xU: ErrSuggestionNotFound → rollback
        else status = created
            R--xU: ErrSuggestionAlreadyCreated → rollback
        else status = rejected và role ≠ admin
            R--xU: ErrReopenForbidden → rollback
        end
        R->>DB: C-4 — pg_advisory_xact_lock(region) + max(n) → id mới
        Note over R,DB: id dạng temple_<region>_<NNN>
        R->>DB: C-5 — INSERT INTO temples (...)
        alt vi phạm constraint / lỗi DB
            R--xU: lỗi → rollback (đề xuất vẫn pending)
        end
        R->>DB: C-6 — UPDATE temple_suggestions SET status='created', temple_id, reviewed_by, reviewed_at, reject_reason=NULL
        R->>DB: COMMIT
        R-->>U: temple + suggestion
    end

    alt thành công
        U-->>H: {suggestion, temple}
        H-->>C: 201 {suggestion, temple}
    else ErrSuggestionNotFound
        U--xH: 404 TEMPLE_SUGGESTION_NOT_FOUND
        H-->>C: 404
    else ErrSuggestionAlreadyCreated
        U--xH: 409 SUGGESTION_ALREADY_CREATED
        H-->>C: 409
    else ErrReopenForbidden
        U--xH: 403 FORBIDDEN
        H-->>C: 403
    else lỗi hạ tầng
        U--xH: error
        H-->>C: 500 INTERNAL_ERROR
    end
```
