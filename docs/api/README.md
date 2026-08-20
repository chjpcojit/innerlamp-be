# docs/api

`openapi.yaml` — OpenAPI 3.0 spec for every HTTP endpoint in
`services/appService/delivery/http/`: auth (US-001/002/003), users/profile
(US-004), and content — sutras/mantras/bodhisattvas (US-005..US-010).

Generated from the handlers themselves (request/response shapes, status
codes, error codes) — when a handler changes, update this file in the same
change rather than letting it drift.

Validate after editing:

```bash
npx @redocly/cli lint docs/api/openapi.yaml
```
