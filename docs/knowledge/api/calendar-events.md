---
type: API Endpoint
title: GET /v1/calendar/events
description: Returns the Buddhist, Vietnamese and folk events that fall in one solar year, each occurrence with both its solar and lunar date, for the calendar screen of a logged-in user.
resource: services/appService/delivery/http/calendar_event_handler.go
tags: [api, calendar, events, lunar-calendar]
status: stable
generated: { by: claude-code, at: 2026-10-11T04:45:00+07:00 }
sources:
  - path: docs/specs/features/calendar/requirements.md
    kind: spec
  - path: docs/specs/features/calendar/api-contract.md
    kind: spec
  - path: docs/specs/api/openapi.yaml
    kind: openapi
---

# Summary

Owned by the events half of [Calendar](/features/calendar.md). The client calls it once per solar year it shows and
calls again with `year + 1` for "load more"; it filters by month itself from `occurrences`. Success is the year's
groups, events and occurrences; failure is `400 INVALID_YEAR` or the auth middleware's `401`.

> ⚠️ **Contract status**: Confirmed against a running binary on 2026-10-11 (`curl` against the dev server, and
> `e2e/calendar_event_e2e_test.go` against the real binary and a fresh Postgres). `status: stable` describes the
> **contract**; the event *content* it returns is still a draft (see the feature's Pending work).

# Request

```
GET /v1/calendar/events?year=2026
Authorization: Bearer <access_token>
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `year` (query) | integer as string | no | Plain digits, 1900..2100. Omitted/empty = the current year in `calendar.timezone` (`Asia/Ho_Chi_Minh` in dev); the response `year` says which. |

**Headers**: `Authorization: Bearer` with an `access` token (not a refresh token), enforced by
`authMiddleware.AuthenticateFunc` in `cmd/main.go`. No web API key header is required. CORS headers come from the
default `pkg/middleware` chain. There is no request body.

# Response

## Success

`200 OK`, `Content-Type: application/json`, the standard envelope. Real response (year 2026, trimmed to one event and
two occurrences; the live answer has 51 events and 75 occurrences):

```json
{"success": true, "data": {
  "year": 2026,
  "groups": [
    {"code": "buddhist", "name": "Phật giáo"},
    {"code": "vietnam",  "name": "Việt Nam"},
    {"code": "folk",     "name": "Dân gian"}
  ],
  "events": [
    {"code": "tet-nguyen-dan", "name": "Tết Nguyên Đán",
     "short_description": "Mùng 1 Tết, ngày đầu năm âm lịch.",
     "long_description": "Tết Nguyên Đán là lễ tết lớn nhất của người Việt, …",
     "guidance": "Cúng gia tiên, chúc Tết, lễ chùa đầu năm; …",
     "image_url": null, "groups": ["vietnam", "folk"], "is_public_holiday": true}
  ],
  "occurrences": [
    {"event_code": "tet-duong-lich", "solar_date": "2026-01-01", "lunar_date": {"day": 13, "month": 11, "year": 2025, "is_leap_month": false}},
    {"event_code": "tet-nguyen-dan", "solar_date": "2026-02-17", "lunar_date": {"day": 1,  "month": 1,  "year": 2026, "is_leap_month": false}}
  ]
}}
```

| Field | Type | Notes |
|---|---|---|
| `year` | int | The solar year used. |
| `groups[]` | `{code, name}` | Every group, by display order. The client must not hard-code the three codes: back-office may add more. |
| `events[]` | object | Each event **once**, only if it occurs in the year. `image_url` is `null` today, always present. `groups` lists codes; a code unknown to the catalogue is dropped. |
| `occurrences[]` | object | Sorted by `solar_date`, then event display order, then code. `solar_date` is `YYYY-MM-DD` and always inside `year`; `lunar_date.year` is the **lunar** year and can be `year - 1` (tháng Chạp events in January–February). `is_leap_month` is true in a leap month. |

`is_public_holiday` is true for the event's own day only; there are no substitute-day occurrences. Mùng 1 and Rằm
(`ram-am-lich`) appear once in `events` and ~12–13 times in `occurrences`. Same `year`, same data → byte-identical body.

**Caching**: explicit `year` → `Cache-Control: private, max-age=<calendar.events_cache_max_age>` (86400 in dev);
omitted `year` → `Cache-Control: no-cache`. Error responses set neither.

## Error cases

| Case | Trigger | Sentinel | Status |
|---|---|---|---|
| `INVALID_YEAR` | `year` not plain digits, or outside 1900..2100 (`abc`, `2026.5`, `+2026`, `1899`, `2101`) | `domain.ErrInvalidYear` | 400, `details.field = "year"` |
| `UNAUTHORIZED` | no/garbage/expired token, or a refresh token | auth middleware (`WriteAuthError`) | 401 |
| `INTERNAL_ERROR` | repository failure | any other error | 500, generic message |

Mapping: `writeDomainError` in `services/appService/delivery/http/errors.go`.

# Call sites

| Layer | File | Notes |
|---|---|---|
| Handler | `services/appService/delivery/http/calendar_event_handler.go` | `CalendarEventHandler.Events`; consumer-defined `calendarEventUsecase` interface |
| Usecase | `services/appService/usecase/calendar_event_usecase.go` | `EventsForYear` |
| Repository | `services/appService/repository/postgres_calendar_event_repository.go` | `ListGroups`, `ListActiveEvents` |
| Wiring | `services/appService/cmd/main.go` | `mux.HandleFunc("GET /v1/calendar/events", authMiddleware.AuthenticateFunc(...))` |

# Related concepts

- [Calendar](/features/calendar.md) — the feature, the repeat-rule model and what is still pending.
- [Login](/features/login.md) — where the access token comes from.
- [0031 Calendar Events](/migrations/0031-calendar-events.md) — the tables behind it.
