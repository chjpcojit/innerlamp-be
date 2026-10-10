# Flow — lamp

Version: 1.2
Last updated: 2026-10-10
Status: implemented — mirror 1:1 của `design.md` v1.2 (flow thắp đèn và vườn đèn không đổi so với v1.1).
Module: `lamp`

> See `changelog.md` in this folder for the version history of this spec.

## Sequence — Thắp đèn (POST /v1/lamps)

Vẽ lại Step L-1 → L-4 của `design.md`.

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
    alt token hợp lệ
        M->>H: userID trong context
    else thiếu / hết hạn / sai loại token
        M--xC: 401
    end

    H->>H: DecodeJSONBodyStrict
    H->>U: Light(ctx, userID, req)
    U->>U: trim + NFC, validate AC3-AC8, now, litAt, endAt
    alt không hợp lệ
        U--xH: LampValidationError
        H--xC: 400 VALIDATION_ERROR + details.field
    end

    U->>R: Light(ctx, in{now, maxActive, dedupeWindow})

    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction
        R->>DB: SELECT dharma_name FROM users WHERE id FOR UPDATE
        R->>DB: SELECT đèn trùng nội dung trong dedupeWindow
        alt có đèn trùng
            R-->>U: đèn đã có, created=false
        else không trùng
            R->>DB: SELECT count(*) WHERE user_id AND end_at > now
            alt count >= maxActive
                R--xU: ErrLampLimitReached → rollback
            else còn chỗ
                R->>DB: INSERT INTO lamps (...) RETURNING id
                R-->>U: đèn mới, created=true
            end
        end
    end

    alt đèn mới
        U-->>H: Lamp, created=true
        H-->>C: 201 Lamp
    else đèn trùng
        U-->>H: Lamp, created=false
        H-->>C: 200 Lamp
    else đủ trần
        U--xH: ErrLampLimitReached
        H--xC: 409 LAMP_LIMIT_REACHED
    end
```

## Sequence — Xem vườn đèn (GET /v1/lamps)

Vẽ lại Step G-1 → G-4 của `design.md`.

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
    alt token hợp lệ
        M->>H: request đã xác thực
    else thiếu / hết hạn / sai loại token
        M--xC: 401
    end

    H->>U: Garden(ctx, query)
    U->>U: validate lamp_type, limit, cursor, kẹp limit, now
    alt tham số sai
        U--xH: ErrInvalidLampFilter / ErrInvalidLimit / ErrInvalidCursor
        H--xC: 400 INVALID_FILTER / INVALID_LIMIT / INVALID_CURSOR
    end

    U->>R: Garden(ctx, q{lampType, limit, cursor, now})

    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction chỉ đọc, REPEATABLE READ
        R->>DB: SELECT lamp_type, count(*) WHERE end_at > now GROUP BY lamp_type
        Note over R,DB: ⚠️ Not confirmed: cách đếm có đạt mục tiêu hiệu năng khi số đèn lớn? Chưa có mục tiêu/khối lượng.
        R->>DB: SELECT trang keyset WHERE end_at > now [AND lamp_type] [AND (lit_at,id) < cursor] ORDER BY lit_at DESC, id DESC LIMIT limit+1
        alt thành công
            R-->>U: dòng, total, counts
        else lỗi DB
            R--xU: lỗi → 500
        end
    end

    U->>U: cắt về limit, has_more, next_cursor, counts đủ ba loại
    U-->>H: GardenPage
    H->>H: map response, bỏ lit_at, lighter_name NULL → "Vô danh"
    H-->>C: 200 {items, total, counts, has_more, next_cursor}
```
