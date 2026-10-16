# Flow — article

Version: 3.0
Last updated: 2026-10-15
Status: in-progress — 1:1 mirror of the three Step-by-step Flows in `design.md`; every Step here traces to the Step with the same number. The related-articles flow (v3.0) is not implemented yet.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

`top` and `detail` are one SQL statement each, so they have no diagram — see "Request flow" in
`design.md`.

## Sequence — Paged list (`GET /v1/articles?page=N`)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as ArticleHandler<br/>(delivery/http)
    participant U as ArticleUsecase
    participant R as ArticleRepository
    participant DB as Postgres

    C->>H: GET /v1/articles?page=N
    H->>H: Step 1 — empty page → 1, else strconv.Atoi
    alt page is not an integer
        H--xC: 400 INVALID_PAGE {details.field: page}
    else page is an integer
        H->>U: List(ctx, page)
        U->>U: Step 2 — page < 1 or > MaxInt32?
        alt out of range
            U--xH: ErrInvalidPage
            H-->>C: 400 INVALID_PAGE
        else in range
            U->>R: List(ctx, offset, page_size)
            R->>DB: Step 3 — SELECT count(*) FROM articles
            DB-->>R: total
            Note over R,DB: Step 3 and Step 4 are two independent statements, NO shared transaction/snapshot
            R->>DB: Step 4 — SELECT … ORDER BY sort_order, id OFFSET $1 LIMIT $2
            DB-->>R: items (empty past the end)
            R-->>U: items, total
            U->>U: Step 5 — nil items → [], total_pages = ceil(total / page_size)
            U-->>H: ArticlePage
            H-->>C: 200 {items, page, page_size, total, total_pages}
        end
    end
    opt DB error in Step 3 or Step 4
        R--xU: error wrapped with %w
        U--xH: error
        H-->>C: 500 INTERNAL_ERROR
    end
```

## Sequence — Related articles (`GET /v1/articles/{id}/related`, v3.0)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as ArticleHandler<br/>(delivery/http)
    participant U as ArticleUsecase
    participant R as ArticleRepository
    participant DB as Postgres

    C->>H: GET /v1/articles/{id}/related
    H->>H: Step 1 — id = r.PathValue("id")
    H->>U: Related(ctx, id)
    U->>R: Step 2 — FindByID(ctx, id)
    R->>DB: SELECT … FROM articles WHERE id = $1
    alt no row
        R--xU: ErrArticleNotFound
        U--xH: ErrArticleNotFound
        H-->>C: 404 ARTICLE_NOT_FOUND
    else found
        DB-->>R: article (category)
        R-->>U: article
        U->>R: Step 3 — ListRelated(ctx, id, category, related_limit)
        Note over R,DB: Step 2 and Step 3 are independent reads, no transaction
        R->>DB: SELECT … WHERE id <> $1 ORDER BY (lower(btrim(category)) = lower(btrim($2))) DESC, sort_order, id LIMIT $3
        DB-->>R: 0..related_limit rows
        R-->>U: items
        U-->>H: items
        H->>H: Step 4 — nil → [], map to summary DTOs
        H-->>C: 200 {items}
    end
    opt DB error in Step 2 or Step 3
        R--xU: error wrapped with %w
        U--xH: error
        H-->>C: 500 INTERNAL_ERROR
    end
```

## Sequence — Seed with the command (`cmd/seedarticles`; the service never seeds)

```mermaid
sequenceDiagram
    autonumber
    participant M as cmd/seedarticles
    participant P as ParseArticleSeed<br/>(repository/article_seed.go)
    participant R as ArticleRepository
    participant DB as Postgres

    M->>P: Step 1 — parse embedded JSON (or -file)
    alt invalid content
        P--xM: error (bad JSON / no article / empty or duplicate id / empty title / bad date)
        M->>M: log.Fatalf, nothing written
    else valid content
        P-->>M: []Article (SortOrder = position in the file)
        M->>R: Step 2 — InsertMissing(ctx, articles)
        rect rgb(235, 240, 255)
            Note over R,DB: ONE transaction for every INSERT
            loop each article
                R->>DB: INSERT … ON CONFLICT (id) DO NOTHING
            end
            alt DB error
                R--xM: error → rollback, no row written
                M->>M: log.Fatalf
            else commit
                R-->>M: rows inserted
            end
        end
        opt -fill-images
            M->>R: Step 3 — FillMissingImages(ctx, articles)
            rect rgb(235, 240, 255)
                Note over R,DB: ONE transaction for every UPDATE
                R->>DB: UPDATE … SET image_url = COALESCE(image_url,$2), thumbnail_url = COALESCE(thumbnail_url,$3)
                alt DB error
                    R--xM: error → rollback
                    M->>M: log.Fatalf
                else commit
                    R-->>M: rows changed
                end
            end
        end
        M->>M: Step 4 — log counts, exit 0
    end
```
