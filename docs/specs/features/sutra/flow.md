# Flow — sutra

Version: 2.13
Last updated: 2026-10-07
Status: implemented (2026-10-03) — mirror 1:1 của năm Step-by-step Flow trong `design.md`, mỗi Step ở đây trace về đúng Step cùng số; cả năm flow đã được đối chiếu với code, và chạy thật qua binary + Postgres trong `e2e/sutra_e2e_test.go` (nạp dữ liệu, danh sách, chi tiết, đọc dòng, tra ngược).
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

Cả bốn route đọc đều có từ hai lời gọi DB nối tiếp trở lên nên đều có diagram. Sơ đồ vẽ đúng code đã implement (đối chiếu từng bước với `usecase/sutra_usecase.go`, `repository/postgres_sutra_repository.go` và `delivery/http/sutra_handler.go`).

## Sequence — Nạp dữ liệu lần đầu khi khởi động

```mermaid
sequenceDiagram
    autonumber
    participant M as cmd/main.go hoặc<br/>cmd/seedsutras
    participant U as SutraUsecase
    participant P as ParseSutraSeed<br/>(repository/sutra_seed.go)
    participant R as SutraRepository
    participant DB as Postgres

    M->>U: SeedIfEmpty(ctx, load)

    Note over R,DB: Step 1 — đếm số văn bản đang có
    Note over R,DB: đã chốt 2026-10-03 — cmd/seedsutras dùng chung luồng này, bảng đã có dòng thì bỏ qua ở cả hai cửa vào
    U->>R: Count(ctx, SutraFilter rỗng)
    R->>DB: SELECT count(*) FROM sutras
    alt lỗi DB (vd. bảng chưa tồn tại)
        R--xU: error bọc %w
        U--xM: lỗi
        M->>M: log.Fatalf "failed to seed sutras"
    else n > 0
        U-->>M: 0 văn bản được chèn, KHÔNG parse, KHÔNG ghi
    end

    Note over U,P: Step 2 — nạp và kiểm tra dữ liệu seed
    U->>P: load() = ParseSutraSeed(fs.Sub(assets.SutrasFS, data/sutras))
    P->>P: đọc index.json + từng id.json, kiểm điều kiện AC29, gán SortOrder, tính sections
    alt có kiểm tra nào fail
        P--xU: error nêu id + lý do
        U--xM: lỗi
        M->>M: log.Fatalf (chưa ghi gì)
    end

    Note over R,DB: Step 3 — kiểm tra Phật/Bồ tát tồn tại
    U->>R: MissingDeities(ctx, ids)
    R->>DB: SELECT id FROM deities WHERE id = ANY($1)
    Note over R,DB: đã chốt 2026-10-03 — thiếu deity thì abort startup, không bỏ qua liên kết thiếu
    alt thiếu deity_id
        R--xU: error nêu danh sách id thiếu
        U--xM: lỗi, nhắc chạy cmd/seeddeities
        M->>M: log.Fatalf (chưa ghi dòng nào)
    end

    Note over R,DB: Step 4 — chèn toàn bộ văn bản
    U->>R: InsertAll(ctx, seeds)
    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction cho toàn bộ
        loop mỗi văn bản theo thứ tự index.json
            R->>DB: INSERT INTO sutras ... ON CONFLICT (id) DO NOTHING RETURNING id
            opt chèn được (id chưa tồn tại)
                R->>DB: INSERT INTO sutra_translations ... RETURNING id
                R->>DB: INSERT INTO sutra_lines (lô nhiều hàng)
                R->>DB: INSERT INTO sutra_deities
            end
        end
        alt lỗi DB (kể cả vi phạm khoá ngoại)
            R--xU: rollback toàn bộ
            U--xM: lỗi
            M->>M: log.Fatalf
        end
    end
    R-->>U: số văn bản, bản dịch, dòng, liên kết đã chèn
    U-->>M: số văn bản đã chèn

    Note over M: Step 5 — ghi log
    M->>M: nếu số > 0 thì logger.Info "seeded sutras on first run" kèm số liệu
```

## Sequence — Danh sách và bộ lọc (`GET /v1/sutras`)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as SutraHandler<br/>(delivery/http)
    participant U as SutraUsecase
    participant R as SutraRepository
    participant DB as Postgres

    C->>H: GET /v1/sutras?page=&tradition=&kind=&collection=&purpose=&daily=&max_minutes=&lang=
    Note over H: Step 1 — đọc page và bộ lọc
    H->>H: page rỗng thì 1, ngược lại Atoi, đọc các bộ lọc
    alt page không phải số nguyên
        H--xC: 400 INVALID_PAGE (details.field = page)
    end
    H->>U: List(ctx, filter, page)

    Note over U: Step 2 — kiểm tra khoảng page và giá trị bộ lọc
    U->>U: page ∈ [1, MaxInt32], enum, daily, max_minutes ≥ 1, không rỗng
    alt page ngoài khoảng
        U--xH: ErrInvalidPage
        H--xC: 400 INVALID_PAGE
    else bộ lọc không hợp lệ
        U--xH: ErrInvalidSutraFilter{Field}
        H--xC: 400 INVALID_FILTER (details.field)
    end

    Note over R,DB: Step 3 — đếm văn bản khớp
    U->>R: Count(ctx, filter)
    R->>DB: SELECT count(*) FROM sutras WHERE (AND các bộ lọc)
    alt lỗi DB
        R--xU: error bọc %w
        H--xC: 500 INTERNAL_ERROR
    end

    Note over R,DB: Step 4 — lấy trang văn bản
    U->>R: List(ctx, filter, offset, limit)
    R->>DB: SELECT ... FROM sutras WHERE (AND các bộ lọc) ORDER BY sort_order, id OFFSET LIMIT
    alt lỗi DB
        R--xU: error bọc %w
        H--xC: 500 INTERNAL_ERROR
    end

    opt Step 5–6 — chỉ khi trang có ít nhất một văn bản
        Note over R,DB: Step 5 — nạp thông tin bản dịch
        R->>DB: SELECT key, lang, variant, translator, license, line_count FROM sutra_translations WHERE sutra_id = ANY($1)
        Note over R,DB: Step 6 — nạp liên kết Phật/Bồ tát
        R->>DB: SELECT ... FROM sutra_deities sd JOIN deities d WHERE sd.sutra_id = ANY($1)
        Note over R,DB: không transaction — Step 3 đến 6 không chung snapshot
    end
    R-->>U: []Sutra
    U-->>H: SutraPage

    Note over U,H: Step 7 — dựng trang và trả response
    H->>H: items nil → [], total_pages = ceil(total / page_size), mảng nil → [], duration nil → null
    H-->>C: 200 {items, page, page_size, total, total_pages}
```

## Sequence — Chi tiết một kinh (`GET /v1/sutras/{id}`)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as SutraHandler<br/>(delivery/http)
    participant U as SutraUsecase
    participant R as SutraRepository
    participant DB as Postgres

    C->>H: GET /v1/sutras/{id}
    H->>U: Get(ctx, id)

    Note over R,DB: Step 1 — tìm văn bản
    U->>R: FindByID(ctx, id)
    R->>DB: SELECT ... FROM sutras WHERE id = $1
    alt không có hàng
        R--xU: ErrSutraNotFound
        U--xH: ErrSutraNotFound
        H--xC: 404 SUTRA_NOT_FOUND
    else lỗi DB
        R--xU: error bọc %w
        H--xC: 500 INTERNAL_ERROR
    end

    Note over R,DB: Step 2 — nạp bản dịch kèm nguồn và mục lục
    R->>DB: SELECT key, lang, variant, translator, source_name, source_url, license, license_url, license_note, note, line_count, sections FROM sutra_translations WHERE sutra_id = $1
    alt lỗi DB / giải mã sections
        R--xU: error bọc %w
        H--xC: 500 INTERNAL_ERROR
    end

    Note over R,DB: Step 3 — nạp liên kết Phật/Bồ tát
    R->>DB: SELECT ... FROM sutra_deities sd JOIN deities d WHERE sd.sutra_id = $1
    Note over R,DB: không transaction — ba câu lệnh độc lập
    alt lỗi DB
        R--xU: error bọc %w
        H--xC: 500 INTERNAL_ERROR
    end
    R-->>U: Sutra
    U-->>H: Sutra

    Note over H: Step 4 — dựng response
    H->>H: map DTO detail, sections nil → []
    H-->>C: 200 SutraDetail
```

## Sequence — Đọc nội dung theo trang dòng (`GET /v1/sutras/{id}/lines`)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as SutraHandler<br/>(delivery/http)
    participant U as SutraUsecase
    participant R as SutraRepository
    participant DB as Postgres

    C->>H: GET /v1/sutras/{id}/lines?translation=&page=&page_size=
    Note over H: Step 1 — đọc và kiểm tra tham số
    H->>H: translation bắt buộc, page rỗng thì 1, Atoi page và page_size
    alt translation vắng / rỗng
        H--xC: 400 INVALID_TRANSLATION (details.field = translation)
    else page không phải số nguyên
        H--xC: 400 INVALID_PAGE
    else page_size không phải số nguyên
        H--xC: 400 INVALID_PAGE_SIZE
    end
    H->>U: Lines(ctx, id, key, page, pageSize)

    Note over U: Step 2 — kiểm tra khoảng và kẹp page_size
    U->>U: page ∈ [1, MaxInt32], page_size ≥ 1, page_size > max thì kẹp, vắng thì lines_page_size
    alt page / page_size ngoài khoảng
        U--xH: ErrInvalidPage / ErrInvalidPageSize
        H--xC: 400
    end

    Note over R,DB: Step 3 — kiểm tra kinh và bản dịch tồn tại
    U->>R: FindTranslation(ctx, id, key)
    R->>DB: SELECT 1 FROM sutras WHERE id = $1
    alt không có sutra
        R--xU: ErrSutraNotFound
        H--xC: 404 SUTRA_NOT_FOUND
    end
    R->>DB: SELECT id, line_count FROM sutra_translations WHERE sutra_id = $1 AND key = $2
    alt không có bản dịch
        R--xU: ErrSutraTranslationNotFound
        H--xC: 404 TRANSLATION_NOT_FOUND
    else lỗi DB
        R--xU: error bọc %w
        H--xC: 500 INTERNAL_ERROR
    end

    Note over R,DB: Step 4 — lấy trang dòng
    U->>R: ListLines(ctx, translationID, fromN, toN)
    R->>DB: SELECT n, text, section FROM sutra_lines WHERE translation_id = $1 AND n > fromN AND n <= toN ORDER BY n
    Note over R,DB: không transaction — Step 3 và Step 4 không chung snapshot
    alt lỗi DB / scan
        R--xU: error bọc %w
        H--xC: 500 INTERNAL_ERROR
    end
    R-->>U: []SutraLine
    U-->>H: SutraLinesPage

    Note over U,H: Step 5 — dựng trang và trả response
    H->>H: items nil → [], total = line_count, total_pages = ceil(total / page_size), section rỗng → null
    H-->>C: 200 {translation, items, page, page_size, total, total_pages}
```

## Sequence — Kinh theo Phật/Bồ tát (`GET /v1/deities/{id}/sutras`)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as SutraHandler<br/>(delivery/http)
    participant U as SutraUsecase
    participant D as DeityRepository<br/>(feature deity, có sẵn)
    participant R as SutraRepository
    participant DB as Postgres

    C->>H: GET /v1/deities/{id}/sutras?page=&role=
    Note over H: Step 1 — đọc page và bộ lọc vai
    H->>H: lấy {id}, page rỗng thì 1 ngược lại Atoi, role tuỳ chọn
    alt page không phải số nguyên
        H--xC: 400 INVALID_PAGE
    end
    H->>U: ListByDeity(ctx, deityID, role, page)

    Note over U: Step 2 — kiểm tra page và role
    U->>U: page ∈ [1, MaxInt32], role vắng hoặc ∈ SPEAKER | SUBJECT | MENTIONED
    alt page ngoài khoảng
        U--xH: ErrInvalidPage
        H--xC: 400 INVALID_PAGE
    else role không hợp lệ
        U--xH: ErrInvalidSutraFilter{Field: role}
        H--xC: 400 INVALID_FILTER
    end

    Note over U,D: Step 3 — kiểm tra Phật/Bồ tát tồn tại
    U->>D: FindByID(ctx, deityID)
    D->>DB: SELECT ... FROM deities WHERE id = $1
    alt không có
        D--xU: ErrDeityNotFound
        H--xC: 404 DEITY_NOT_FOUND
    else lỗi DB
        D--xU: error bọc %w
        H--xC: 500 INTERNAL_ERROR
    end

    Note over R,DB: Step 4 — đếm văn bản liên kết
    U->>R: CountByDeity(ctx, deityID, role)
    R->>DB: SELECT count(*) FROM sutra_deities WHERE deity_id = $1 [AND role = $2]
    alt lỗi DB
        R--xU: error bọc %w
        H--xC: 500 INTERNAL_ERROR
    end

    Note over R,DB: Step 5 — lấy trang văn bản liên kết
    U->>R: ListByDeity(ctx, deityID, role, offset, limit)
    R->>DB: SELECT ..., sd.role FROM sutras s JOIN sutra_deities sd WHERE sd.deity_id = $1 [AND sd.role = $2] ORDER BY role_rank, sort_order, id OFFSET LIMIT
    alt lỗi DB
        R--xU: error bọc %w
        H--xC: 500 INTERNAL_ERROR
    end

    opt Step 6–7 — chỉ khi trang có ít nhất một văn bản
        Note over R,DB: Step 6 — nạp bản dịch
        R->>DB: SELECT ... FROM sutra_translations WHERE sutra_id = ANY($1)
        Note over R,DB: Step 7 — nạp đủ liên kết Phật/Bồ tát của các văn bản đó
        R->>DB: SELECT ... FROM sutra_deities sd JOIN deities d WHERE sd.sutra_id = ANY($1)
        Note over R,DB: không transaction — Step 4 đến 7 không chung snapshot
    end
    R-->>U: []DeitySutra
    U-->>H: SutraPage kèm role

    Note over U,H: Step 8 — dựng trang và trả response
    H->>H: items nil → [], total_pages = ceil(total / page_size), map summary + role
    H-->>C: 200 {items, page, page_size, total, total_pages}
```
