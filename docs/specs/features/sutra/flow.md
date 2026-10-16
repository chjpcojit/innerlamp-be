# Flow — sutra

Version: 4.0
Last updated: 2026-10-16
Status: implemented — 1:1 mirror of the five Step-by-step Flow sections of `design.md`; each Step here traces to the Step with the same number there. The v4.0 part (Step 2 of the load also refuses a missing `durationMinutes`, AC 38) is drawn as a `Note` and is not implemented.
Module: `sutra`

> See `changelog.md` in this folder for the version history of this spec.

All four read routes make two or more sequential DB calls, so every one has a diagram. The diagrams follow the implemented code (checked step by step against `usecase/sutra_usecase.go`, `repository/postgres_sutra_repository.go` and `delivery/http/sutra_handler.go`).

## Sequence — Load the library (cmd/seedsutras; the service does not seed)

```mermaid
sequenceDiagram
    autonumber
    participant M as cmd/seedsutras
    participant U as SutraUsecase
    participant P as ParseSutraSeed<br/>(repository/sutra_seed.go)
    participant R as SutraRepository
    participant DB as Postgres

    M->>U: SeedIfEmpty(ctx, load)

    Note over R,DB: Step 1 — count the texts that exist
    U->>R: Count(ctx, empty SutraFilter)
    R->>DB: SELECT count(*) FROM sutras
    alt DB error (e.g. the table does not exist)
        R--xU: error wrapped with %w
        U--xM: error
        M->>M: fatal log, exit non-zero
    else n > 0
        U-->>M: 0 texts inserted, NO parse, NO write
        M->>M: log "sutras already seeded, skipping", exit 0
    end

    Note over U,P: Step 2 — load and validate the seed data
    U->>P: load() = ParseSutraSeed(fs.Sub(assets.SutrasFS, data/sutras) or os.DirFS(-dir))
    P->>P: read index.json + each id.json, check the conditions of AC 37, assign SortOrder, compute sections
    Note over P: v4.0, not implemented — also refuse a missing or null durationMinutes (AC 38)
    alt any check fails
        P--xU: error naming the id and the reason
        U--xM: error
        M->>M: fatal log (nothing written)
    end

    Note over R,DB: Step 3 — check that the deities exist
    U->>R: MissingDeities(ctx, ids)
    R->>DB: SELECT id FROM deities WHERE id = ANY($1)
    alt deity_id missing
        R--xU: error listing the missing ids
        U--xM: error, tells to run cmd/seeddeities
        M->>M: fatal log (nothing written)
    end

    Note over R,DB: Step 4 — insert every text
    U->>R: InsertAll(ctx, seeds)
    rect rgb(235, 240, 255)
        Note over R,DB: ONE transaction for everything
        loop each text in index.json order
            R->>DB: INSERT INTO sutras ... ON CONFLICT (id) DO NOTHING RETURNING id
            opt inserted (the id did not exist)
                R->>DB: INSERT INTO sutra_translations ... RETURNING id
                R->>DB: INSERT INTO sutra_lines (multi-row batches)
                R->>DB: INSERT INTO sutra_deities
            end
        end
        alt DB error (including a foreign-key violation)
            R--xU: whole rollback
            U--xM: error
            M->>M: fatal log
        end
    end
    R-->>U: counts of texts, versions, lines and links inserted
    U-->>M: number of texts inserted

    Note over M: Step 5 — log
    M->>M: if the number > 0, log "seeded N sutras, ..." with the counts, exit 0
```

## Sequence — List, filter and search (`GET /v1/sutras`)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as SutraHandler<br/>(delivery/http)
    participant U as SutraUsecase
    participant R as SutraRepository
    participant DB as Postgres

    C->>H: GET /v1/sutras?page=&tradition=&kind=&collection=&purpose=&daily=&max_minutes=&lang=&q=
    Note over H: Step 1 — read page and filters
    H->>H: empty page → 1, otherwise Atoi; read the filters and q
    alt page is not an integer
        H--xC: 400 INVALID_PAGE (details.field = page)
    end
    H->>U: List(ctx, filter, page)

    Note over U: Step 2 — validate the page range and the filter values
    U->>U: page in [1, MaxInt32], enums, daily, max_minutes ≥ 1, no empty value, q trimmed to 1–100 runes
    alt page out of range
        U--xH: ErrInvalidPage
        H--xC: 400 INVALID_PAGE
    else invalid filter (including q)
        U--xH: InvalidSutraFilterError{Field}
        H--xC: 400 INVALID_FILTER (details.field)
    end

    Note over R,DB: Step 3 — count the matching texts
    U->>R: Count(ctx, filter)
    R->>DB: SELECT count(*) FROM sutras WHERE (filters ANDed, q via unaccent LIKE on titles, collection, reference)
    alt DB error
        R--xU: error wrapped with %w
        H--xC: 500 INTERNAL_ERROR
    end

    Note over R,DB: Step 4 — fetch the page of texts
    U->>R: List(ctx, filter, offset, limit)
    R->>DB: SELECT ... FROM sutras WHERE (same conditions) ORDER BY sort_order, id OFFSET LIMIT
    alt DB error
        R--xU: error wrapped with %w
        H--xC: 500 INTERNAL_ERROR
    end

    opt Steps 5–6 — only when the page has at least one text
        Note over R,DB: Step 5 — load the versions
        R->>DB: SELECT key, lang, variant, translator, license, line_count FROM sutra_translations WHERE sutra_id = ANY($1)
        Note over R,DB: Step 6 — load the deity links
        R->>DB: SELECT ... FROM sutra_deities sd JOIN deities d WHERE sd.sutra_id = ANY($1)
        Note over R,DB: no transaction — Steps 3 to 6 do not share a snapshot
    end
    R-->>U: []Sutra
    U-->>H: SutraPage

    Note over U,H: Step 7 — build the page and respond
    H->>H: nil items → [], total_pages = ceil(total / page_size), nil arrays → [], nil duration → null (until v4.0)
    H-->>C: 200 {items, page, page_size, total, total_pages}
```

## Sequence — Sutra detail (`GET /v1/sutras/{id}`)

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

    Note over R,DB: Step 1 — find the text
    U->>R: FindByID(ctx, id)
    R->>DB: SELECT ... FROM sutras WHERE id = $1
    alt no row
        R--xU: ErrSutraNotFound
        U--xH: ErrSutraNotFound
        H--xC: 404 SUTRA_NOT_FOUND
    else DB error
        R--xU: error wrapped with %w
        H--xC: 500 INTERNAL_ERROR
    end

    Note over R,DB: Step 2 — load the versions with sources and sections
    R->>DB: SELECT key, lang, variant, translator, source_name, source_url, license, license_url, license_note, note, line_count, sections FROM sutra_translations WHERE sutra_id = $1
    alt DB error / sections decode
        R--xU: error wrapped with %w
        H--xC: 500 INTERNAL_ERROR
    end

    Note over R,DB: Step 3 — load the deity links
    R->>DB: SELECT ... FROM sutra_deities sd JOIN deities d WHERE sd.sutra_id = $1
    Note over R,DB: no transaction — three independent statements
    alt DB error
        R--xU: error wrapped with %w
        H--xC: 500 INTERNAL_ERROR
    end
    R-->>U: Sutra
    U-->>H: Sutra

    Note over H: Step 4 — build the response
    H->>H: map to the detail DTO, nil sections → []
    H-->>C: 200 SutraDetail
```

## Sequence — Read a version page by page (`GET /v1/sutras/{id}/lines`)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as SutraHandler<br/>(delivery/http)
    participant U as SutraUsecase
    participant R as SutraRepository
    participant DB as Postgres

    C->>H: GET /v1/sutras/{id}/lines?translation=&page=&page_size=
    Note over H: Step 1 — read and check the parameters
    H->>H: translation required, empty page → 1, Atoi for page and page_size
    alt translation absent / empty
        H--xC: 400 INVALID_TRANSLATION (details.field = translation)
    else page is not an integer
        H--xC: 400 INVALID_PAGE
    else page_size is not an integer
        H--xC: 400 INVALID_PAGE_SIZE
    end
    H->>U: Lines(ctx, id, key, page, pageSize)

    Note over U: Step 2 — validate the range and clamp page_size
    U->>U: page in [1, MaxInt32], page_size ≥ 1, above the max → clamp, absent → lines_page_size
    alt page / page_size out of range
        U--xH: ErrInvalidPage / ErrInvalidPageSize
        H--xC: 400
    end

    Note over R,DB: Step 3 — check that the text and the version exist
    U->>R: FindTranslation(ctx, id, key)
    R->>DB: SELECT 1 FROM sutras WHERE id = $1
    alt no text
        R--xU: ErrSutraNotFound
        H--xC: 404 SUTRA_NOT_FOUND
    end
    R->>DB: SELECT id, line_count FROM sutra_translations WHERE sutra_id = $1 AND key = $2
    alt no version
        R--xU: ErrSutraTranslationNotFound
        H--xC: 404 TRANSLATION_NOT_FOUND
    else DB error
        R--xU: error wrapped with %w
        H--xC: 500 INTERNAL_ERROR
    end

    Note over R,DB: Step 4 — fetch the page of lines
    U->>R: ListLines(ctx, translationID, fromN, toN)
    R->>DB: SELECT n, text, section FROM sutra_lines WHERE translation_id = $1 AND n > fromN AND n <= toN ORDER BY n
    Note over R,DB: no transaction — Steps 3 and 4 do not share a snapshot
    alt DB error / scan
        R--xU: error wrapped with %w
        H--xC: 500 INTERNAL_ERROR
    end
    R-->>U: []SutraLine
    U-->>H: SutraLinesPage

    Note over U,H: Step 5 — build the page and respond
    H->>H: nil items → [], total = line_count, total_pages = ceil(total / page_size), empty section → null
    H-->>C: 200 {translation, items, page, page_size, total, total_pages}
```

## Sequence — Sutras of a Buddha or bodhisattva (`GET /v1/deities/{id}/sutras`)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as SutraHandler<br/>(delivery/http)
    participant U as SutraUsecase
    participant D as DeityRepository<br/>(deity feature, existing)
    participant R as SutraRepository
    participant DB as Postgres

    C->>H: GET /v1/deities/{id}/sutras?page=&role=
    Note over H: Step 1 — read page and role filter
    H->>H: read {id}, empty page → 1 otherwise Atoi, role optional
    alt page is not an integer
        H--xC: 400 INVALID_PAGE
    end
    H->>U: ListByDeity(ctx, deityID, role, page)

    Note over U: Step 2 — validate page and role
    U->>U: page in [1, MaxInt32], role absent or SPEAKER | SUBJECT | MENTIONED
    alt page out of range
        U--xH: ErrInvalidPage
        H--xC: 400 INVALID_PAGE
    else invalid role
        U--xH: InvalidSutraFilterError{Field: role}
        H--xC: 400 INVALID_FILTER
    end

    Note over U,D: Step 3 — check that the deity exists
    U->>D: FindByID(ctx, deityID)
    D->>DB: SELECT ... FROM deities WHERE id = $1
    alt none
        D--xU: ErrDeityNotFound
        H--xC: 404 DEITY_NOT_FOUND
    else DB error
        D--xU: error wrapped with %w
        H--xC: 500 INTERNAL_ERROR
    end

    Note over R,DB: Step 4 — count the linked texts
    U->>R: CountByDeity(ctx, deityID, role)
    R->>DB: SELECT count(*) FROM sutra_deities WHERE deity_id = $1 [AND role = $2]
    alt DB error
        R--xU: error wrapped with %w
        H--xC: 500 INTERNAL_ERROR
    end

    Note over R,DB: Step 5 — fetch the page of linked texts
    U->>R: ListByDeity(ctx, deityID, role, offset, limit)
    R->>DB: SELECT ..., sd.role FROM sutras s JOIN sutra_deities sd WHERE sd.deity_id = $1 [AND sd.role = $2] ORDER BY role_rank, sort_order, id OFFSET LIMIT
    alt DB error
        R--xU: error wrapped with %w
        H--xC: 500 INTERNAL_ERROR
    end

    opt Steps 6–7 — only when the page has at least one text
        Note over R,DB: Step 6 — load the versions
        R->>DB: SELECT ... FROM sutra_translations WHERE sutra_id = ANY($1)
        Note over R,DB: Step 7 — load every deity link of those texts
        R->>DB: SELECT ... FROM sutra_deities sd JOIN deities d WHERE sd.sutra_id = ANY($1)
        Note over R,DB: no transaction — Steps 4 to 7 do not share a snapshot
    end
    R-->>U: []DeitySutra
    U-->>H: SutraPage with role

    Note over U,H: Step 8 — build the page and respond
    H->>H: nil items → [], total_pages = ceil(total / page_size), map card + role
    H-->>C: 200 {items, page, page_size, total, total_pages}
```
