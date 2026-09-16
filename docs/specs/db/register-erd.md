# ERD — Register (US-001)

Source: `docs/specs/onboarding/onboarding.md` (US-001, plus US-002 AC4 for device scope).

Register (`POST /v1/auth/register`) creates a `users` row, records the device it was registered from (US-002 AC4: each device keeps its own login state — "mỗi thiết bị có refresh token riêng"), and issues a session holding the JWT access+refresh token pair (AC1/AC7: register logs the user in immediately). All three tables live in PostgreSQL.

```mermaid
erDiagram
    USERS ||--o{ USER_DEVICES : "registers from"
    USERS ||--o{ USER_SESSIONS : "has (one per device)"
    USER_DEVICES ||--o{ USER_SESSIONS : "issued for"

    USERS {
        bigint id PK
        varchar email UK "unique, login identifier"
        varchar password_hash "bcrypt, never plaintext (AC6)"
        varchar phone_number UK "E.164 format, e.g. +84..., no OTP verify at MVP"
        varchar dharma_name "optional, pre-existing column"
        varchar ordination_place "optional, chùa/tổ đình nơi quy y"
        timestamp created_at
        timestamp updated_at
    }

    USER_DEVICES {
        bigint id PK
        bigint user_id FK
        varchar device_id "device fingerprint hash"
        varchar platform "iOS, Android, Web"
        varchar os_version
        varchar app_version
        varchar device_model
        varchar device_name "user-assigned name"
        varchar ip_address
        varchar user_agent
        timestamp created_at "device first seen / registered at"
    }

    USER_SESSIONS {
        bigint id PK
        bigint user_id FK
        bigint device_id FK
        varchar session_id UK
        text access_token "current JWT access token"
        text refresh_token UK "current refresh token, rotates on use"
        int token_version "incremented on rotation"
        boolean is_active
        boolean is_revoked
        timestamp revoked_at "nullable"
        varchar revoked_reason "nullable"
        timestamp last_activity_at
        timestamp expires_at
        timestamp created_at
        timestamp updated_at
    }
```

## Field notes / open items

- `email`, `phone_number`: both unique — AC2 requires separate "already used" errors for each, so uniqueness must be checked (and violated) independently, not via a combined constraint.
- `dharma_name`: spec says this column "đã có sẵn" (pre-existing) — not part of this migration, listed here only because it's optional data collected at register.
- `password_hash`: bcrypt per AC6 and per US-002's technical note (`bcrypt.CompareHashAndPassword`).
- `USER_DEVICES`: one row per device a user has registered/logged in from — device fingerprint/metadata only, no token state.
- `USER_SESSIONS`: PostgreSQL table (not the Mongo-backed `pkg/session.Session` — that package's `Manager` interface is Mongo-only today; a Postgres-backed implementation of the same interface would need to be added, or this table consumed directly). Mirrors that struct's fields 1:1 minus the embedded `DeviceInfo` (now a separate `USER_DEVICES` row referenced by `device_id`).
- `expires_at`: refresh token TTL is 7 days, matching `pkg/session.DefaultConfig().RefreshTokenTTL` (`pkg/session/session.go:116-124`) — onboarding spec's technical note was updated to match this default instead of the config being changed to match the spec.
