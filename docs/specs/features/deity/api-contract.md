# Phật / Bồ tát catalogue — API contract (for iOS)

Version: 1.3
Last updated: 2026-10-12
Status: implemented + verified against a running server (dev DB, 18 deities: 6 BUDDHA, 12 BODHISATTVA)
Code: `services/appService/delivery/http/deity_handler.go`, `usecase/deity_usecase.go`, `repository/postgres_deity_repository.go`
Data: `assets/data/phat_botat.json` → table `deities` (`migrations/0007_deities.sql`) via `go run ./services/appService/cmd/seeddeities` (idempotent)

All endpoints are **public**, `GET` only. Envelope and error format are the same as the temple catalogue (`docs/specs/api/openapi.yaml`, tag Temples).

## Ordering

By `sort_order` = position in `phat_botat.json`, then `id`. The source has no popularity metric, so the file order (6 Buddhas, then bodhisattvas) *is* the ranking; reorder the file and reseed to change it. ⚠️ Not confirmed: whether "top 10" should use another criterion.

## Endpoints

| Route | Returns |
|---|---|
| `GET /v1/deities/top` | `{items: [DeitySummary x10]}` — Buddhas and bodhisattvas mixed (`deity.top_limit`) |
| `GET /v1/deities?page=N` | `{items, page, page_size, total, total_pages}` — Buddhas and bodhisattvas together (each item has `type`), 20/page (`deity.page_size`), page 1-based, default 1. **Replaces `GET /v1/bodhisattvas` (removed in v1.1)** |
| `GET /v1/deities/{id}` | `DeityDetail` — works for a Buddha or a bodhisattva id |

`DeitySummary`: `id, type (BUDDHA|BODHISATTVA), name_vi, name_sanskrit, name_chinese, title_honorific, thumbnail_url, image_url, core_virtue, pure_land`.

`DeityDetail` = summary + `canonical_source`, `lunar_memorial_days [{lunar_day, lunar_month, event_name}]`, `iconography {posture, mount_or_throne, sacred_artifacts [string], symbolic_meaning}`, `biography_summary`, `great_vows [string]`, `mantras {short_mantra_sanskrit, short_mantra_sino_vietnamese, long_dharani_name, dharani_phonetic_snippet}`. Arrays are never null.

A page past the end is `200` with `items: []`. `page=0`/negative/non-numeric → `400 INVALID_PAGE`. Unknown id → `404 DEITY_NOT_FOUND`.

### Filter by type — `GET /v1/deities?type=<T>&page=N` (implemented 2026-10-12)

- `type` is optional. Allowed values, exact and case-sensitive: `BUDDHA`, `BODHISATTVA`, `DEVA`. Omitted or empty = all types (current behavior, unchanged).
- With `type`, only that type is returned, same order; `total` / `total_pages` count only that type; `page` applies within the filtered list.
- Any other non-empty value (including `buddha`) → `400 INVALID_DEITY_TYPE`, `details.field = "type"`. `INVALID_PAGE` is reported before `INVALID_DEITY_TYPE`.
- One value per request; multiple types are not supported.
- Clients that filtered on `item.type` locally can switch to this parameter; omitting it keeps the old behavior.

⚠️ Image URLs are passed through as in the source: `thumbnail_url` is a relative path (`assets/images/buddhas/thumbnails/...`) and `image_url` a placeholder host (`cdn.yourapp.com`). Neither resolves yet.
