-- Gallery: beautiful Buddhist photos (temples, Buddha statues, meditation, festivals)
-- for the iOS browse screens: top, paged list with an optional type filter, detail.
-- Populated from assets/data/buddhist_gallery.json by
-- `go run ./services/appService/cmd/seedgallery`.
--
-- id is the stable string id from the source data (gallery_001, ...), so the seeder
-- upserts on it and an id an iOS client cached survives reseeds.
--
-- sort_order is assigned by the seeder: the source file groups images by type, so it
-- interleaves the four types (CHUA, TUONG_PHAT, THIEN, LE_HOI, CHUA, ...) keeping file
-- order within a type. "top" and paging both order by (sort_order, id).
--
-- image_url / thumbnail_url are URLs of the locally served copies (assets/images/gallery,
-- LAN IP in dev; see scripts/download_gallery_images.py); author, license and
-- source_url carry the attribution the CC licences require.
--
-- Idempotent (IF NOT EXISTS) like every other file here.
CREATE TABLE IF NOT EXISTS gallery_images (
    id            VARCHAR(64) PRIMARY KEY,
    type          VARCHAR(16) NOT NULL
                  CONSTRAINT chk_gallery_images_type CHECK (type IN ('CHUA', 'TUONG_PHAT', 'THIEN', 'LE_HOI')),
    sort_order    INTEGER NOT NULL,
    title         VARCHAR(255) NOT NULL,
    location      VARCHAR(255) NOT NULL DEFAULT '',
    thumbnail_url TEXT NOT NULL,
    image_url     TEXT NOT NULL,
    author        VARCHAR(255) NOT NULL DEFAULT '',
    license       VARCHAR(64) NOT NULL DEFAULT '',
    source_url    TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Serves top and the unfiltered list: ORDER BY sort_order, id.
CREATE INDEX IF NOT EXISTS idx_gallery_images_order ON gallery_images (sort_order, id);

-- Serves the list filtered by type.
CREATE INDEX IF NOT EXISTS idx_gallery_images_type_order ON gallery_images (type, sort_order, id);
