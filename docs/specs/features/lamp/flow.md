# Flow — lamp

Version: 2.0
Last updated: 2026-10-17
Status: v1 flows implemented; v2.0 changes (blank recipient → `nil`, `IS NOT DISTINCT FROM` on the recipient, `is_mine` in the garden query) are design only. 1:1 mirror of `design.md` v2.0.
Module: `lamp`

> See `changelog.md` in this folder for the version history of this spec.

## Sequence — Light a lamp (POST /v1/lamps)

Redraws Step L-1 → L-4 of `design.md`.

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as authMiddleware
    participant H as LampHandler<br/>(delivery/http)
    participant U as LampUsecase
    participant R as LampRepository
    participant DB as Postgres

    C->>M: POST /v1/lamps (Bearer access token)
    alt valid token
        M->>H: userID in context
    else missing / expired / wrong token type
        M--xC: 401
    end

    H->>H: DecodeJSONBodyStrict
    H->>U: Light(ctx, userID, req)
    U->>U: trim + NFC, blank recipient → nil (v2.0), validate AC 3-8, now, litAt, endAt
    alt invalid
        U--xH: LampValidationError
        H--xC: 400 VALIDATION_ERROR + details.field
    end

    U->>R: Light(ctx, p{lamp, maxActive, dedupeWindow})

    rect rgb(235, 240, 255)
        Note over R,DB: ONE transaction
        R->>DB: SELECT dharma_name FROM users WHERE id FOR UPDATE
        R->>DB: SELECT duplicate lamp inside dedupeWindow (recipient and prayer IS NOT DISTINCT FROM)
        alt duplicate found
            R-->>U: existing lamp, created=false
        else not a duplicate
            R->>DB: SELECT count(*) WHERE user_id AND end_at > now
            alt count >= maxActive
                R--xU: ErrLampLimitReached → rollback
            else room left
                R->>DB: INSERT INTO lamps (...) RETURNING id
                R-->>U: new lamp, created=true
            end
        end
    end

    alt new lamp
        U-->>H: Lamp, created=true
        H-->>C: 201 Lamp
    else duplicate
        U-->>H: Lamp, created=false
        H-->>C: 200 Lamp
    else cap reached
        U--xH: ErrLampLimitReached
        H--xC: 409 LAMP_LIMIT_REACHED
    end
```

## Sequence — See the lamp garden (GET /v1/lamps)

Redraws Step G-1 → G-4 of `design.md`.

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as authMiddleware
    participant H as LampHandler<br/>(delivery/http)
    participant U as LampUsecase
    participant R as LampRepository
    participant DB as Postgres

    C->>M: GET /v1/lamps?lamp_type=&limit=&cursor=
    alt valid token
        M->>H: authenticated request, userID in context
    else missing / expired / wrong token type
        M--xC: 401
    end

    H->>U: Garden(ctx, userID, query)
    U->>U: validate lamp_type, limit, cursor, clamp limit, now, viewerID
    alt bad parameter
        U--xH: ErrInvalidLampFilter / ErrInvalidLimit / ErrInvalidCursor
        H--xC: 400 INVALID_FILTER / INVALID_LIMIT / INVALID_CURSOR
    end

    U->>R: Garden(ctx, q{lampType, limit, cursor, now, viewerID})

    rect rgb(235, 240, 255)
        Note over R,DB: ONE read-only transaction, REPEATABLE READ
        R->>DB: SELECT lamp_type, count(*) WHERE end_at > now GROUP BY lamp_type
        Note over R,DB: ⚠️ Not confirmed: does the counting meet the performance goal when lamps are many? No target or volume yet.
        R->>DB: SELECT keyset page, (user_id = viewer) AS is_mine, WHERE end_at > now [AND lamp_type] [AND (lit_at,id) < cursor] ORDER BY lit_at DESC, id DESC LIMIT limit+1
        alt success
            R-->>U: rows, total, counts
        else DB error
            R--xU: error → 500
        end
    end

    U->>U: trim to limit, has_more, next_cursor, counts for all three types
    U-->>H: GardenPage
    H->>H: map response, drop lit_at, lighter_name NULL → "Vô danh", recipient NULL → null, keep is_mine
    H-->>C: 200 {items, total, counts, has_more, next_cursor}
```
