# Temple catalogue — API contract (for iOS)

Version: 1.3
Last updated: 2026-10-11
Status: implemented. v1.3 adds `GET /v1/temples/options` and the structured address (`location.province_code`, `ward_code`, `ward`, `full_address`); the structured address is verified by an automated end-to-end test against the real binary and Postgres (`services/appService/e2e/temple_e2e_test.go`), the rest by the manual run of v1.2 (dev DB, then 179 temples; the seed file now has 93)
Code: `services/appService/delivery/http/temple_handler.go`, `usecase/temple_usecase.go`, `repository/postgres_temple_repository.go`
Data: `assets/data/temple.json` → table `temples` via `go run ./services/appService/cmd/seedtemples`

Base URL (dev): `http://localhost:8080` (simulator). A physical device needs the Mac's LAN IP instead of `localhost`.
All four endpoints are **public** — no `Authorization` header, no login. Method `GET` only.

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
| `thumbnail_url` | string? | Absolute URL of a small JPEG, served by the same server (`/static/images/temples/thumb/{id}.jpg`). `null` when a temple has no image (the seed file currently gives all 93 one) — keep a placeholder for that case. |
| `image_url` | string? | Same, large JPEG (`/static/images/temples/full/{id}.jpg`). |
| `sect` | string | `BẮC_TÔNG` `TRUC_LAM_YEN_TU` `KHAT_SI` `NAM_TONG_KHMER` `HOA_TONG` `NAM_TONG` `TIN_NGUONG_DANG_GIAN` `MINH_SU_DAO` |
| `heritage_rank` | string | see order above |
| `established_year` | int | |
| `location` | `Location` (below) | |

### `Location`

| Field | Type | Notes |
|---|---|---|
| `lat`, `lng` | double | |
| `address` | string | The part **below ward level** (house number, street, hamlet…); **may be `""`** when a temple is known only by its ward. Until a temple has its structured address (`ward_code` is `null`) this is still the old full string. **Do not display it on its own — use `full_address`.** |
| `province` | string | Short province/city name without prefix (`Hồ Chí Minh`, `Huế`); one of the 34 current province-level units. |
| `province_code` | string? | Current-scheme province code (`GET /v1/administrative-units/new/provinces`). `null` until the temple has its structured address. |
| `ward_code` | string? | Current-scheme ward code, a ward of `province_code`. Same `null` rule. |
| `ward` | string? | Ward's full name with its type prefix (`Phường Tân Định`). Same `null` rule. |
| `full_address` | string | **Show this.** Non-empty parts of `address`, `ward`, `province` joined by `, `, e.g. `73 Mai Thị Lựu, Phường Tân Định, Hồ Chí Minh`. Never `null`. |

`province_code`, `ward_code` and `ward` are all `null` together or all set together. Codes only mean something in the current scheme — never mix them with legacy-scheme codes.

Treat enum-like strings as open: decode unknown values to an "other" case rather than failing.

## 1. GET /v1/temples/top

Top 10 temples. No parameters.

```json
{"success": true, "data": {"items": [
  {"id": "temple_north_011", "name": "Chùa Bích Động", "type": "CHUA", "region": "north",
   "thumbnail_url": null, "image_url": null, "sect": "BẮC_TÔNG",
   "heritage_rank": "DI_SAN_THE_GIOI", "established_year": 1428,
   "location": {"lat": 20.2185, "lng": 105.9177, "address": "", "province": "Ninh Bình",
                "province_code": "37", "ward_code": "14359", "ward": "Phường Nam Hoa Lư",
                "full_address": "Phường Nam Hoa Lư, Ninh Bình"}}
]}}
```
`items` has 10 entries (`temple.top_limit` in server config).

## 2. GET /v1/temples?page=N

Paged list, **20 per page** (`temple.page_size`). `page` is 1-based, optional, default 1.

```json
{"success": true, "data": {"items": [ /* TempleSummary x20 */ ],
  "page": 1, "page_size": 20, "total": 93, "total_pages": 5}}
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

The three content fields are never `null`. All 93 temples in the seed file have them. Content is compiled from public web sources (Vietnamese Wikipedia, tourism and Buddhist news sites) and should be treated as editorial text, not authoritative history — years in `established_year` come from the seed file and can differ from what the text says.

Unknown id → **404** `TEMPLE_NOT_FOUND`.

## 4. GET /v1/temples/options?page=N&q=

Compact pick-list for the registration screen. **Public, no `X-API-Key`.** Same ranking as the other lists. Page size is server config (`temple.options_page_size`, dev 100) — use `page_size` and `total_pages` from the response, do not assume a number.

- `q` (optional): search in the temple **name**, case- and diacritic-insensitive (`bich dong` finds `Chùa Bích Động`). To not search, **omit** the parameter: `q=` empty or blank is **400** `INVALID_FILTER` (`details.field = "q"`); more than 100 characters likewise.
- `page`: as for the list (`INVALID_PAGE`, past-the-end is `200` with `items: []`).

```json
{"success": true, "data": {"items": [
  {"id": "temple_south_002", "name": "Chùa Ngọc Hoàng", "province": "Hồ Chí Minh", "province_code": "79"}
], "page": 1, "page_size": 100, "total": 93, "total_pages": 1}}
```

Each item has exactly four fields: `id` (send it as `ordination_temple_id` at registration), `name`, `province` (short name) and `province_code` (string, `null` until the temple has its structured address). For the full address, image, coordinates or detail, call the list or `GET /v1/temples/{id}`.

## Errors summary

| HTTP | code | When |
|---|---|---|
| 400 | `INVALID_PAGE` | bad `page` on list or options |
| 400 | `INVALID_FILTER` | empty/blank or over-long `q` on options |
| 404 | `TEMPLE_NOT_FOUND` | unknown id on detail |
| 404 | `NOT_FOUND` | wrong path |
| 405 | `METHOD_NOT_ALLOWED` | non-GET |
| 500 | `INTERNAL_ERROR` | server fault, no detail leaked |

## Running the server locally

```bash
make db-up                                        # Postgres + migrations (0005_temples.sql, 0011_temple_content.sql)
go run ./services/appService/cmd/seedadminunits  # administrative-units dataset (required before seedtemples)
go run ./services/appService/cmd/seedtemples      # load 93 temples, idempotent
go run ./services/appService/cmd                  # :8080
```
⚠️ The repository integration tests (`go test ./services/appService/repository`) truncate the dev DB, including `temples`. Re-run the seeder afterwards.

## Changelog

- **1.3** (2026-10-11) — Added `GET /v1/temples/options` (pick-list, search by name). `location` gains `province_code`, `ward_code`, `ward`, `full_address`; `location.address` is now only the part below ward level. Image fields are described as URLs (the seed file has an image for all 93 temples), counts updated from 179 to 93.

- **1.2** (2026-09-30) — Sync with `assets/data/temple.json` after the data update: top-list example now uses Chùa Bích Động's real `location`; the local-run note names `0011_temple_content.sql` (the migration that adds `introduction` / `architecture_heritage` / `visiting_notes`). No change to routes, fields or error codes.
