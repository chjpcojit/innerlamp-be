# Flow — temple-schedule

Version: 1.1
Last updated: 2026-10-13
Status: đã code; mirror `design.md`
Module: `temple-schedule`

> See `changelog.md` in this folder for the version history of this spec.

File này vẽ lại 1:1 hai section "Step-by-step Flow" của `design.md`. Số bước và thứ tự khớp nhau.

## Sequence — Lấy thời khoá của một chùa (`GET /v1/temples/{id}/schedule`)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant A as authMiddleware<br/>(jwt/middleware)
    participant H as TempleScheduleHandler<br/>(delivery/http)
    participant U as TempleScheduleUsecase
    participant TR as TempleRepository
    participant R as TempleScheduleRepository
    participant DB as Postgres

    Note over C,A: Step 1 — xác thực
    C->>A: GET /v1/temples/{id}/schedule
    alt token access hợp lệ
        A->>H: Schedule (user_id trong context)
    else thiếu / hết hạn / sai loại
        A--xC: 401 (middleware)
    end

    Note over H: Step 2 — parse request
    H->>U: ScheduleForTemple(ctx, id)

    Note over U,DB: Step 3 đến 5 là các lần đọc độc lập, KHÔNG bọc transaction (chỉ đọc)
    Note over U,DB: Step 3 — kiểm tra chùa tồn tại
    U->>TR: FindByID(ctx, id)
    TR->>DB: SELECT … FROM temples WHERE id = $1
    alt không có dòng
        TR--xU: ErrTempleNotFound
        U--xH: ErrTempleNotFound
        H-->>C: 404 TEMPLE_NOT_FOUND
    else lỗi DB
        TR--xU: error
        U--xH: error
        H-->>C: 500
    end

    Note over U,DB: Step 4 — đọc các buổi active của chùa
    U->>R: ListActiveByTemple(ctx, templeID)
    R->>DB: SELECT … FROM temple_schedules WHERE temple_id AND is_active
    alt lỗi DB
        R--xU: error
        U--xH: error
        H-->>C: 500
    end

    Note over R,DB: Step 5 — đọc bước của các buổi đó (bỏ qua nếu không có buổi nào)
    opt có ≥ 1 buổi
        R->>DB: SELECT … FROM temple_schedule_steps WHERE schedule_id = ANY(...) ORDER BY schedule_id, position
        alt lỗi DB
            R--xU: error
            U--xH: error
            H-->>C: 500
        end
    end
    R-->>U: items kèm steps

    Note over U: Step 6 — ghép và sắp xếp (thuần, không I/O)
    U->>U: SortTempleSchedule: bước theo position, weekdays chuẩn hoá, buổi theo start_time/sort_order/id

    Note over H,C: Step 7 — trả kết quả
    U-->>H: items
    H-->>C: 200 {temple_id, items}
```

## Sequence — Nạp dữ liệu thời khoá (`cmd/seedtempleschedules`)

```mermaid
sequenceDiagram
    autonumber
    participant Op as Người vận hành
    participant S as seedtempleschedules<br/>(main.go + parse.go)
    participant D as TempleScheduleCatalogue<br/>(domain)
    participant R as TempleScheduleRepository
    participant DB as Postgres

    Note over Op,S: Step 1 — đọc và giải mã file
    Op->>S: go run ./services/appService/cmd/seedtempleschedules
    S->>S: đọc temple-schedules.json, strict decode, NFC, `source` bắt buộc
    alt JSON hỏng / khoá lạ / thiếu source
        S--xOp: lỗi, không ghi gì
    end

    Note over S,D: Step 2 — kiểm tra nội dung (thuần)
    S->>D: Verify()
    D->>D: id duy nhất, name, giờ HH:MM, end ≠ start, weekdays 0..6, bước: text và position
    alt không hợp lệ
        D--xS: ErrInvalidTempleSchedule (nêu id buổi)
        S--xOp: lỗi, không ghi gì
    end

    Note over S,DB: Step 3 — kiểm tra chùa tồn tại và ghi
    S->>R: ReplaceCatalogue(ctx, catalogue)
    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction
        R->>DB: BEGIN
        R->>DB: SELECT id FROM temples
        alt temple_id không có trong temples
            R--xS: lỗi nêu id buổi → ROLLBACK
        else
            R->>DB: INSERT temple_schedules … ON CONFLICT (id) DO UPDATE (không ghi is_active)
            R->>DB: DELETE bước của các buổi trong file, INSERT bước mới
            alt lỗi DB
                R--xS: error → ROLLBACK
            else
                R->>DB: COMMIT
            end
        end
    end
    S-->>Op: in số buổi và số bước đã nạp
```
