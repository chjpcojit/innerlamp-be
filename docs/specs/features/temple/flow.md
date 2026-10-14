# Flow — temple

Version: 1.14
Last updated: 2026-10-11
Status: implemented (2026-09-30) — mirror 1:1 của ba Step-by-step Flow trong `design.md`, mỗi Step ở đây trace về đúng Step cùng số. **v1.7 (2026-10-11, chưa có code):** sơ đồ `options` có thêm `province_code`; sơ đồ seed có thêm Step 4 (đối chiếu mã) và đánh số lại Step 5–6.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

`top` và `detail` chỉ một câu SQL nên không có diagram — xem "Request flow" trong `design.md`.

## Sequence — Danh sách phân trang (`GET /v1/temples?page=N`)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as TempleHandler<br/>(delivery/http)
    participant U as TempleUsecase
    participant R as TempleRepository
    participant DB as Postgres

    C->>H: GET /v1/temples?page=N
    H->>H: Step 1 — page rỗng → 1, ngược lại strconv.Atoi
    alt page không phải số nguyên
        H--xC: 400 INVALID_PAGE {details.field: page}
    else page hợp lệ về kiểu
        H->>U: List(ctx, page)
        U->>U: Step 2 — page < 1 hoặc > MaxInt32?
        alt ngoài khoảng
            U--xH: ErrInvalidPage
            H-->>C: 400 INVALID_PAGE
        else trong khoảng
            U->>R: List(ctx, offset, page_size)
            R->>DB: Step 3 — SELECT count(*) FROM temples
            DB-->>R: total
            Note over R,DB: Step 3 và Step 4 là hai câu lệnh độc lập, KHÔNG chung transaction/snapshot
            R->>DB: Step 4 — SELECT … ORDER BY heritage_priority, established_year, id OFFSET $1 LIMIT $2
            DB-->>R: items (rỗng nếu vượt cuối)
            R-->>U: items, total
            U->>U: Step 5 — items nil → [], total_pages = ceil(total / page_size)
            U-->>H: TemplePage
            H-->>C: 200 {items, page, page_size, total, total_pages}
        end
    end
    opt lỗi DB ở Step 3 hoặc Step 4
        R--xU: error bọc %w
        U--xH: error
        H-->>C: 500 INTERNAL_ERROR
    end
```

## Sequence — Danh sách rút gọn phân trang, tìm theo tên (`GET /v1/temples/options?page=N&q=`) *(v1.5)*

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as TempleHandler<br/>(delivery/http)
    participant U as TempleUsecase
    participant R as TempleRepository
    participant DB as Postgres

    C->>H: GET /v1/temples/options?page=N&q=…
    H->>H: Step 1 — page rỗng → 1, ngược lại strconv.Atoi; đọc q (có mặt hay không)
    alt page không phải số nguyên
        H--xC: 400 INVALID_PAGE {details.field: page}
    else page hợp lệ về kiểu
        H->>U: Options(ctx, query, page)
        U->>U: Step 2 — page < 1 hoặc > MaxInt32? q có mặt: trim, rỗng hoặc > 100 ký tự?
        alt page ngoài khoảng
            U--xH: ErrInvalidPage
            H-->>C: 400 INVALID_PAGE
        else q không hợp lệ
            U--xH: ErrInvalidTempleQuery
            H-->>C: 400 INVALID_FILTER {details.field: q}
        else hợp lệ
            U->>R: ListOptions(ctx, query, offset, options_page_size)
            R->>DB: Step 3 — SELECT count(*) FROM temples [WHERE unaccent(lower(name)) LIKE …]
            DB-->>R: total (số khớp)
            Note over R,DB: Step 3 và Step 4 là hai câu lệnh độc lập, KHÔNG chung transaction/snapshot
            R->>DB: Step 4 — SELECT id, name, province, province_code … [cùng WHERE] ORDER BY … OFFSET $n LIMIT $m
            DB-->>R: dòng (rỗng nếu vượt cuối hoặc không khớp)
            R-->>U: items, total
            U->>U: Step 5 — items nil → [], total_pages = ceil(total / options_page_size)
            U-->>H: TempleOptionPage
            H-->>C: 200 {items [{id, name, province, province_code}], page, page_size, total, total_pages}
        end
    end
    opt lỗi DB ở Step 3 hoặc Step 4
        R--xU: error bọc %w
        U--xH: error
        H-->>C: 500 INTERNAL_ERROR
    end
```

## Sequence — Nạp dữ liệu bằng `cmd/seedtemples`

```mermaid
sequenceDiagram
    autonumber
    actor O as Operator
    participant M as seedtemples/main.go
    participant P as parseTemples<br/>(seedtemples/parse.go)
    participant R as TempleRepository
    participant W as WardResolver<br/>(AdministrativeUnitRepository)
    participant DB as Postgres

    O->>M: go run ./services/appService/cmd/seedtemples [-file path]
    M->>M: Step 1 — os.ReadFile(file)
    alt không đọc được file
        M--xO: log.Fatalf "read <file>" (chưa chạm DB)
    else đọc được
        M->>P: Step 2 — parseTemples(src)
        P->>P: stripBlockComments → json.Unmarshal → kiểm id / regionOf / HeritagePriority / province_code, ward_code, address không rỗng
        alt dữ liệu không hợp lệ
            P--xM: error (chú thích không đóng / JSON hỏng / không có chùa / id trùng / id sai dạng / hạng lạ / thiếu province_code, ward_code, address)
            M--xO: log.Fatalf "parse <file>" (chưa chạm DB)
        else hợp lệ
            P-->>M: []domain.Temple (Region từ id, HeritagePriority từ hạng)
            Note over M: ⚠️ DEV ONLY — config.dev.yaml + NewMockSecretGetterFromYAML
            M->>M: Step 3 — nạp config, secret dev
            M->>DB: database.New + Connect (timeout 30s)
            alt thiếu config/secret hoặc không kết nối được
                M--xO: log.Fatalf (chưa ghi gì)
            else kết nối được
                loop từng chùa
                    M->>W: Step 4 — ResolveCurrentWard(province_code, ward_code)
                    W->>DB: SELECT addr_new_provinces LEFT JOIN addr_new_wards
                    DB-->>W: tên tỉnh, tên xã (hoặc không có dòng)
                end
                Note over M,DB: Step 4 chỉ đọc, KHÔNG chung transaction với Step 5; Province/Ward lấy từ dataset
                alt mã tỉnh lạ / xã lạ hoặc lệch tỉnh / location.province lệch tên / dataset rỗng
                    W--xM: ErrAdministrativeUnitNotFound / ErrWardNotInProvince / lệch tên
                    M--xO: log.Fatalf nêu id chùa và field (chưa ghi gì)
                else lỗi DB khi đọc
                    W--xM: error
                    M--xO: log.Fatalf (chưa ghi gì)
                else mọi chùa có Province và Ward
                    W-->>M: Province, Ward
                end
                M->>R: Step 5 — Upsert(ctx, temples)
                rect rgb(235, 240, 255)
                    Note over R,DB: MỘT transaction cho mọi chùa
                    loop từng chùa
                        R->>DB: INSERT INTO temples (28 cột) … ON CONFLICT (id) DO UPDATE SET <mọi cột>, updated_at = now()
                    end
                    alt lỗi DB (CHECK type/region/hạng/địa chỉ, thiếu bảng/cột, mất kết nối)
                        R--xM: error → rollback, không dòng nào được ghi
                        M--xO: log.Fatalf "seed temples"
                    else commit
                        R-->>M: nil
                    end
                end
                M->>M: Step 6 — log "seeded N temples from <file>"
                M-->>O: thoát 0
            end
        end
    end
```
