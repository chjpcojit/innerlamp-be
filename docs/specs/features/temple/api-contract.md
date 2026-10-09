# Temple catalogue — API contract (for iOS)

Version: 1.2
Last updated: 2026-09-30
Status: implemented + verified against a running server (dev DB, 179 temples)
Code: `services/appService/delivery/http/temple_handler.go`, `usecase/temple_usecase.go`, `repository/postgres_temple_repository.go`
Data: `assets/data/temple.json` → table `temples` via `go run ./services/appService/cmd/seedtemples`

Base URL (dev): `http://localhost:8080` (simulator). A physical device needs the Mac's LAN IP instead of `localhost`.
All three endpoints are **public** — no `Authorization` header, no login. Method `GET` only.

## Envelope (same as every other route)

Success: `{"success": true, "data": <payload>}`
Error: `{"success": false, "error": {"code": "UPPER_SNAKE", "message": "...", "type": "business|validation|system"}, "details"?: {...}}`
Branch on `error.code`, never on `message`.

## Ordering

Temples are ranked by **heritage_rank** (most prominent first), then **established_year** ascending (oldest first), then `id`. Top and list use the same order, so page 1 of the list starts with the top-10.

Rank order: `DI_SAN_THE_GIOI` > `DI_TICH_QUOC_GIA_DAC_BIET` > `KHU_DU_LICH_QUOC_GIA` > `DI_TICH_QUOC_GIA` > `DANH_LAM_THANG_CANH` > `DI_TICH_CAP_TINH` > `DI_TICH_CAP_THANH_PHO` > `CHUA_XEP_HANG`.

## Shared type: `TempleSummary`

| Field | Type | Notes |
|---|---|---|
| `id` | string | stable, e.g. `temple_north_001` |
| `name` | string | |
| `type` | string | `CHUA` `THIEN_VIEN` `DAO_TRANG` `TINH_XA` `THIEN_XA` |
| `region` | string | `north` `central` `south` |
| `thumbnail_url` | string? | **null today for every temple** — no images in the data yet. Show a placeholder. |
| `image_url` | string? | same as above |
| `sect` | string | `BẮC_TÔNG` `TRUC_LAM_YEN_TU` `KHAT_SI` `NAM_TONG_KHMER` `HOA_TONG` `NAM_TONG` `TIN_NGUONG_DANG_GIAN` `MINH_SU_DAO` |
| `heritage_rank` | string | see order above |
| `established_year` | int | |
| `location` | `{lat: double, lng: double, address: string}` | |

Treat enum-like strings as open: decode unknown values to an "other" case rather than failing.

## 1. GET /v1/temples/top

Top 10 temples. No parameters.

```json
{"success": true, "data": {"items": [
  {"id": "temple_north_011", "name": "Chùa Bích Động", "type": "CHUA", "region": "north",
   "thumbnail_url": null, "image_url": null, "sect": "BẮC_TÔNG",
   "heritage_rank": "DI_SAN_THE_GIOI", "established_year": 1428,
   "location": {"lat": 20.2185, "lng": 105.9177, "address": "Xã Ninh Hải, Huyện Hoa Lư, Tỉnh Ninh Bình"}}
]}}
```
`items` has 10 entries (`temple.top_limit` in server config).

## 2. GET /v1/temples?page=N

Paged list, **20 per page** (`temple.page_size`). `page` is 1-based, optional, default 1.

```json
{"success": true, "data": {"items": [ /* TempleSummary x20 */ ],
  "page": 1, "page_size": 20, "total": 179, "total_pages": 9}}
```
- A page past the end (`page=10`) is **200 with `items: []`** — not an error. Stop paging when `page >= total_pages` (or items is empty).
- `page=0`, negative, non-numeric, or an overflowing number → **400** `INVALID_PAGE`, `details.field = "page"`.

## 3. GET /v1/temples/{id}

`data` = `TempleSummary` **plus**:

| Field | Type |
|---|---|
| `aliases` | [string] (may be empty) |
| `founder` | string |
| `current_abbot` | string |
| `operating_hours` | string, e.g. `"07:30 - 18:00"` |
| `amenities` | `{parking_car, parking_motorcycle, vegetarian_meals, summer_retreat}` all bool |
| `introduction` | string — giới thiệu về chùa (`""` nếu chưa có) |
| `architecture_heritage` | string — kiến trúc và di sản nổi bật (`""` nếu chưa có) |
| `visiting_notes` | [string] — lưu ý khi viếng thăm, mỗi phần tử một ý (`[]` nếu chưa có) |

The three content fields are never `null`. All 179 temples have them. Content is compiled from public web sources (Vietnamese Wikipedia, tourism and Buddhist news sites) and should be treated as editorial text, not authoritative history — years in `established_year` come from the seed file and can differ from what the text says.

Unknown id → **404** `TEMPLE_NOT_FOUND`.

## Errors summary

| HTTP | code | When |
|---|---|---|
| 400 | `INVALID_PAGE` | bad `page` on list |
| 404 | `TEMPLE_NOT_FOUND` | unknown id on detail |
| 404 | `NOT_FOUND` | wrong path |
| 405 | `METHOD_NOT_ALLOWED` | non-GET |
| 500 | `INTERNAL_ERROR` | server fault, no detail leaked |

## Running the server locally

```bash
make db-up                                        # Postgres + migrations (0005_temples.sql, 0011_temple_content.sql)
go run ./services/appService/cmd/seedtemples      # load 179 temples, idempotent
go run ./services/appService/cmd                  # :8080
```
⚠️ The repository integration tests (`go test ./services/appService/repository`) truncate the dev DB, including `temples`. Re-run the seeder afterwards.

## Changelog

- **1.2** (2026-09-30) — Sync with `assets/data/temple.json` after the data update: top-list example now uses Chùa Bích Động's real `location`; the local-run note names `0011_temple_content.sql` (the migration that adds `introduction` / `architecture_heritage` / `visiting_notes`). No change to routes, fields or error codes.
