# services/appService/storage

Adapters from `domain` storage ports onto `pkg/storage`. This is the one place that knows which
bucket the service's objects live in; `usecase` sees only the narrow `domain.AvatarStorage`.

| File | What |
|---|---|
| `avatar_storage.go` | `AvatarStorage` implements `domain.AvatarStorage` for one bucket (`profile.bucket`): `SignUpload` (PUT, signed `Content-Type` + `x-goog-content-length-range: 0,<max>`), `SignRead` (GET), `Stat`, `Copy`, `Delete`. |
| `avatar_storage_test.go` | Tests against a fake `pkg/storage.Client`: right method, bucket, TTL and signed headers reach the client; emulator-mode URLs; not-found maps to `domain.ErrObjectNotFound`; deleting a missing key is not an error. |
| `avatar_storage_emulator_test.go` | Integration test against a real `fake-gcs-server` container (testcontainers; skipped under `-short` or without Docker): ticket → upload → Stat → Copy → read URL → Delete. |

Spec: `docs/specs/features/profile/` (design.md "Infrastructure / Storage adapter", Phase 4).

## Two modes

| | Real bucket (`storage.emulator_host` empty) | Emulator (`storage.emulator_host` set) |
|---|---|---|
| `SignUpload` | V4-signed `PUT`, signed `Content-Type` and `x-goog-content-length-range: 0,<max>` | unsigned `POST` to `/upload/storage/v1/b/{bucket}/o?uploadType=media&name={key}` (the emulator has no PUT upload) |
| `SignRead` | V4-signed `GET` | plain `/download/storage/v1/b/{bucket}/o/{key}?alt=media` |
| `Stat` / `Copy` / `Delete` | pkg/storage client (ADC) | same client, reaching the emulator via `STORAGE_EMULATOR_HOST` (set by `main` from the config) |

The use case never assumes `PUT`: the method travels in `domain.SignedUpload`.

## Known limits

- **TD-001** — real signing has never run: the emulator does not verify V4 signatures, signed
  headers, length ranges or expiry, so the 5 MB / content-type / lifetime limits are only enforced by
  the use case on confirm (recorded type and size), not by the store. Check ticket → upload → confirm
  on a real bucket before the first deploy that serves the avatar routes (`docs/tech-debt/profile/`).
- **TD-002** — nothing cleans up the temporary folder or an old avatar whose delete failed.

## Local run

`make up` / `make run` start the `gcs` service from `docker-compose.yml` (port 4443); `config.dev.yaml`
points `storage.emulator_host` at it and `main` creates the bucket if it is missing. If the emulator is
not up the service still starts (it logs a warning) and only the avatar routes fail. E2E tests start
their own container and never use the one on `localhost:4443`.
