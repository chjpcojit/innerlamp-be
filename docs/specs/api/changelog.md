# Changelog — InnerLamp API contract

Tracks the history of `info.version` in `openapi.yaml` (and its copy under `web/docs/public/`).
Newest first. Platform specs cite the version they were written against in their `Contract:` header.

## 2026-11-15 — documentation only, no version bump

- `DharmaTalkDetail.audio_url` is now documented as an **absolute** public `http(s)` URL with `HEAD`,
  byte-range and CORS delivery; clients pass it straight to their player and never prefix a host.
- Described the dev `/static/audio/` route and the production `server.audio_base_url` / object-storage
  path. No path, schema field or error code changed, so `info.version` stays **3.0.0**.

## 2026-11-14 — documentation only, no version bump

- `servers:` and every example URL now use the local development ports **8082** (appService) and
  **8083** (backofficeService), matching `config.dev.yaml` and `make urls`. Before this entry both
  files still said 8080/8081.
- No path, schema, field or error code changed, so `info.version` stays **3.0.0**.

## v3.0.0 — 2026-10-27

- **Lamp**: `POST /v1/lamps` `recipient` becomes optional and nullable; a blank or absent recipient is
  stored as `null` ("all beings"). `GET /v1/lamps` items gain `is_mine`. MAJOR because `recipient`
  can now be `null`.
- Spec: `docs/specs/features/lamp/` v2.1, product `docs/product/lamp/product.md` v1.0.
- Commit: `e90169e`.

## v2.2.0 — 2026-10-17

- **Calendar**: public `GET /v1/calendar/events`, plus `is_major`, `sort_order` and `recurrence` on a
  calendar event; deity "vía" events are derived from the calendar. Backward-compatible, MINOR.
- Spec: `docs/specs/features/calendar/` v3.2, product `docs/product/calendar/product.md` v1.2.
- Commit: `3af0844`.

## v2.1.1 — 2026-10-16

- `info.version` pinned at 2.1.1 and the contract versioning rule written down (MAJOR for a breaking
  change, MINOR for an addition). Commit: `7214104`.

## Before v2.1.1

Not reconstructed here. The contract predates this changelog; use the git history of
`docs/specs/api/openapi.yaml` and the feature changelogs under `docs/specs/features/*/changelog.md`.
