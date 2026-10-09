-- Articles ("bài viết Phật pháp hay"): curated reading list for the iOS home
-- (top 5), the paged articles screen and the article detail screen. Seeded once
-- from assets/data/buddhist_articles.json when the table is empty on service
-- start (see usecase.ArticleUsecase.SeedIfEmpty); after that the DB is the
-- source of truth and restarts never overwrite it.
--
-- id is the stable slug from the source data (vipassana-nghe-thuat-song, ...),
-- so an id an iOS client cached survives any reseed.
--
-- sort_order is the position in the source file. The source carries no
-- reliable date (most publishedDate are null) and no popularity metric, so
-- "top" and paging both order by the curated file order instead of guessing.
--
-- published_date and image_url are NULLable: the source has no value for most
-- rows and the API returns them as JSON null, not a fake date / empty string.
--
-- Idempotent (IF NOT EXISTS) like every other file here.
CREATE TABLE IF NOT EXISTS articles (
    id              VARCHAR(128) PRIMARY KEY,
    sort_order      INTEGER NOT NULL,
    title           TEXT NOT NULL,
    category        VARCHAR(128) NOT NULL DEFAULT '',
    author          VARCHAR(255) NOT NULL DEFAULT '',
    author_initials VARCHAR(8) NOT NULL DEFAULT '',
    published_date  DATE,
    read_minutes    INTEGER NOT NULL DEFAULT 0,
    summary         TEXT NOT NULL DEFAULT '',
    body_paragraphs TEXT[] NOT NULL DEFAULT '{}',
    quote           TEXT NOT NULL DEFAULT '',
    quote_source    TEXT NOT NULL DEFAULT '',
    tags            TEXT[] NOT NULL DEFAULT '{}',
    language        VARCHAR(8) NOT NULL DEFAULT 'vi',
    image_url       TEXT,
    source_name     TEXT NOT NULL DEFAULT '',
    source_url      TEXT NOT NULL DEFAULT '',
    license         VARCHAR(64) NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Serves top-N and the paged list (same ORDER BY as the repository).
CREATE INDEX IF NOT EXISTS idx_articles_order ON articles (sort_order, id);
