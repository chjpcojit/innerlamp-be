# Temple announcements — API contract (for web and app clients)

Version: 1.0
Last updated: 2026-10-14
Status: implemented (build, vet, unit tests and both services' E2E against real Postgres pass). Not yet checked by a human in a running client.
Code: `services/appService/delivery/http/temple_announcement_handler.go`, `services/appService/cmd/temple_announcement_routes.go`, `usecase/temple_announcement_usecase.go`, `repository/postgres_temple_announcement_repository.go`
Spec: `requirements.md` (TA1-TA33), `design.md`. Machine-readable: `docs/specs/api/openapi.yaml` (tag `TempleAnnouncements`).

Base URL (dev): `http://localhost:8080`. Web sends `X-API-Key` like every other route; mobile does not.
All three routes need `Authorization: Bearer <access_token>` (a refresh token is rejected). There is no route that takes a temple id or user id: the user and the temple are always the token's user and that user's **current** ordination temple (`PATCH /v1/users/me` `ordination_temple_id`).

## Envelope

Success: `{"success": true, "data": <payload>}`
Error: `{"success": false, "error": {"code": "UPPER_SNAKE", "message": "...", "type": "business|validation|system"}, "details"?: {...}}`
Branch on `error.code`, never on `message`. All three routes answer with `Cache-Control: no-store`.

## What an announcement is

A notice a temple publishes (e.g. "ghi danh khoa tu mua he con 10 ngay") to the people whose ordination temple it is. It has an expiry, an issuing body (ban) and plain-text `body`. It is **not** an OS push, not a temple event and not a schedule. There is no read/unread state and no unread count: the client must not invent one from this API.

## 1. GET /v1/users/me/temple-announcements?page=N&page_size=M

The **active** announcements of the caller's ordination temple, newest `published_at` first (then `id`). Active = not revoked, `published_at <= now`, `now < expires_at` (an announcement is expired from the exact moment `expires_at`).

```json
{"success": true, "data": {
  "receive": true,
  "temple": {"id": "temple_north_011", "name": "Chùa Bích Động"},
  "items": [
    {"id": "tann_5b0f4a52-3c1e-4a37-9f4e-0c6a7d3b1e21",
     "title": "Ghi danh khoá tu mùa hè",
     "body": "Chùa nhận ghi danh khoá tu mùa hè đến hết ngày 30/10.",
     "published_at": "2026-10-14T03:00:00Z",
     "expires_at": "2026-10-30T16:59:59Z",
     "issuing_body": {"id": "tib_0c2e8f1a-77d4-4e55-8a4d-3f6a1b9c2d10", "name": "Ban Trị sự"}}
  ],
  "page": {"page": 1, "page_size": 20, "total": 1}
}}
```

| Field | Type | Notes |
|---|---|---|
| `receive` | bool | Stored switch (section 2). Always present, so the UI can draw the switch even when the list is empty. |
| `temple` | `{id, name}`? | `null` when `receive = false` or the user has no ordination temple. Present (even with `items: []`) when `receive = true` and the user has a temple. |
| `items` | array | Never `null`; `[]` when empty. |
| `items[].body` | string | Plain text. Do not render as HTML. |
| `items[].published_at`, `expires_at` | string | RFC 3339, UTC. |
| `items[].issuing_body.name` | string | Show it as the sender. |
| `page.page_size` | int | The size actually used (see below). |
| `page.total` | int | Total active announcements of the temple (0 when `temple` is null). |

Paging:
- `page` is 1-based, default 1. `page_size` default is server config `temple_announcement.page_size` (dev 20); a larger value is **capped** to `max_page_size` (dev 50), not an error: read `page.page_size` back.
- A page past the end is `200` with `items: []` and the real `total`.
- `page` or `page_size` that is non-numeric, `0` or negative is **400 `INVALID_PAGE`** (with `details.field = "page"` even when the bad one is `page_size`).

Three cases that all return `200` with `items: []`: switch off (`temple: null`), no ordination temple (`temple: null`), temple has nothing active (`temple` set). Changing or clearing the ordination temple takes effect on the next call; nothing is copied per user.

## 2. GET /v1/users/me/temple-announcement-settings

`{"success": true, "data": {"receive": true}}`. Default `true`. One switch per account, shared by every device.

## 3. PUT /v1/users/me/temple-announcement-settings

Body is exactly `{"receive": true|false}`. Response is the stored value: `{"success": true, "data": {"receive": false}}`.

- Off hides the whole list (route 1 returns `items: []`); nothing is deleted, turning it on shows them again.
- Changing the ordination temple does not change the switch.
- `receive` missing -> **400 `VALIDATION_ERROR`**, `details.field = "receive"`. A non-boolean, an extra key, an empty or malformed body, or a wrong `Content-Type` is also `400 VALIDATION_ERROR` and changes nothing. Body over 1 MB is `413 PAYLOAD_TOO_LARGE`.
- Writing the same value again is `200`.

## Rate limit

60 requests per minute per account for these three routes together (dev: `temple_announcement.rate_limit`). It is **its own budget**: it neither spends nor is spent by the 10/min of `/v1/users/me*` profile routes. Over the limit: `429 RATE_LIMITED` with header `Retry-After` and `details.retry_after_seconds`; nothing is read or changed. Do not poll; fetch on screen open and pull-to-refresh.

## Errors summary

| HTTP | code | When |
|---|---|---|
| 400 | `INVALID_PAGE` | bad `page` or `page_size` on route 1 |
| 400 | `VALIDATION_ERROR` | route 3 body invalid (see above) |
| 401 | `UNAUTHORIZED` | missing/invalid/expired token, or refresh token used as Bearer |
| 401 | `INVALID_API_KEY` | web only: wrong `X-API-Key` |
| 413 | `PAYLOAD_TOO_LARGE` | route 3 body over 1 MB |
| 429 | `RATE_LIMITED` | per-account budget spent |
| 500 | `INTERNAL_ERROR` | server fault, no detail leaked |

## Backoffice (not for end-user clients)

Publishing, editing, revoking and listing announcements, and managing issuing bodies, are six routes on backofficeService (`:8081`, role `admin` or `content_editor`): `POST/GET /v1/admin/temples/{id}/announcements`, `PATCH /v1/admin/temple-announcements/{id}`, `POST /v1/admin/temple-announcements/{id}/revoke`, `POST/GET /v1/admin/temples/{id}/issuing-bodies`. Error codes there: `INVALID_ANNOUNCEMENT`, `INVALID_ISSUING_BODY`, `INVALID_PAGE`, `ANNOUNCEMENT_NOT_FOUND`, `ANNOUNCEMENT_REVOKED`, `ISSUING_BODY_EXISTS`, `TEMPLE_NOT_FOUND`, `VALIDATION_ERROR`. See `openapi.yaml`.

## Changelog

- **1.0** (2026-10-14) — First version, written from the implemented handlers.
