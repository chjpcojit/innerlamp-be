-- US-007/US-008/US-009/US-010: Xem danh sách/chi tiết thần chú và Chư Phật
-- & Bồ Tát — creates bodhisattvas and mantras tables.
-- Schema per docs/db/content-erd.md. bodhisattvas must exist first since
-- mantras.bodhisattva_id references it. updated_at is indexed on both
-- tables for delta sync (US-007 AC5, US-009 AC4); mantras.bodhisattva_id is
-- indexed for the "thần chú của vị này" filter (US-007 AC3) and for
-- GetBodhisattvaUsecase's related-mantras lookup (US-010 AC1).
-- bodhisattva_id is nullable: some mantras (e.g. Om Mani Padme Hum in
-- general recitation) aren't gắn với a specific vị — AC2 of US-007/US-008.
-- gen_random_uuid() requires the pgcrypto extension (same convention as
-- 0005_content_sutras.sql).

CREATE TABLE IF NOT EXISTS bodhisattvas (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        VARCHAR(255) NOT NULL UNIQUE,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    image_url   TEXT,
    version     INT NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_bodhisattvas_updated_at ON bodhisattvas(updated_at);

CREATE TABLE IF NOT EXISTS mantras (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            VARCHAR(255) NOT NULL UNIQUE,
    title           VARCHAR(255) NOT NULL,
    body            TEXT NOT NULL,
    audio_url       TEXT,
    bodhisattva_id  UUID REFERENCES bodhisattvas(id),
    version         INT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mantras_bodhisattva_id ON mantras(bodhisattva_id);
CREATE INDEX IF NOT EXISTS idx_mantras_updated_at ON mantras(updated_at);
