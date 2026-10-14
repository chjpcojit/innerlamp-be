# ADR-0001: In-app notifications on every platform, push via FCM on iOS and Android

| Field | Value |
|---|---|
| Status | Proposed |
| Date | 2026-10-14 |
| Deciders | anh Đức (decides) · Claude (`adr-writer`, proposes) |
| Features | cross-cutting — first user: `memorial` (reminders, ADR-0002); later temple announcements/events |
| Platforms | Backend, Website, iOS, Android |
| Supersedes | — |
| Related | ADR-0002 · `docs/product/memorial/product.md@1.0` (FR-MEM-007, FR-MEM-008, BR-MEM-11) · `/mnt/project-files/reports/infra/2026-10-14-thiet-ke-gcp.md` |

## Context

The product owner decided on 2026-10-14 that the backend notifies users instead of each app
scheduling its own local notifications:

- A notification is shown **in the app's notification list (activity) on all three platforms** —
  website, iOS, Android.
- iOS and Android **also** receive it as a **push notification**. The website gets no push.
- Notifications are stored on the server and stay until the user has read them; the list loads the
  most recent month first and loads older ones on demand.

What exists today (checked 2026-10-14 on `main` @ 5f4ca56):

- No notification table, no push provider, no device-token storage, no background job runner in
  the backend (`migrations/0001…0036`, `services/appService/cmd/main.go` routes, `pkg/` has no
  push package).
- Neither app has Firebase or push set up (no `FirebaseMessaging`, `aps-environment` entitlement or
  `POST_NOTIFICATIONS` in `innerlamp-iOS` / `innerlamp-Android`).
- `user_devices` (`migrations/0002_login_devices_and_rate_limit.sql`) already holds one row per
  (user, device), upserted at login — a natural owner for a device's push token.
- Target infrastructure is GCP Cloud Run + Cloud SQL Postgres, no queue and no Redis by default
  ("add Pub/Sub or Cloud Tasks when an async flow appears") — infra design v0.3.

Constraints (confirmed 2026-10-14 unless marked): under 10k users in the first year; a personal
project, so monthly cost and operational simplicity weigh heavily; the owner registers the Firebase
project. Inferred: notification volume is low (a few per user per month).

### Why this needs an ADR

It adds a new external provider (push), a new account-owned data model every platform reads, and a
client-side registration contract that ships inside released app versions — all expensive to change
later.

## Decision drivers

- One notification source of truth, identical on website, iOS and Android (the owner's requirement).
- Push on mobile only; the website relies on the in-app list.
- Unread notifications are kept; history is paged (one month at a time).
- Lowest operational cost at < 10k users; reuse Postgres and the existing Go service.
- Reusable by later features (temple announcements, events), not memorial-specific.
- Privacy: notification text may contain a deceased relative's name (NFR-MEM-02).

## Options considered

### Option 0: Keep the status quo

No server notifications. Each app schedules local notifications (product §2.7 today); the website
shows nothing.

| Dimension | Assessment |
|---|---|
| Complexity | Low on backend; each app builds its own scheduler (iOS still a stub, Android none) |
| Cost | None |
| Reversibility | High |
| Cross-platform impact | Logic duplicated in 2 apps; website has no notifications at all |
| Security & privacy | Nothing leaves the device |
| Reliability | Device-dependent; a deleted entry keeps firing on other devices until they reload |
| Fit with current stack | Fits |
| Familiarity | High |

**Pros:** no new infrastructure. **Cons:** contradicts the owner's decision (website notifications,
server-driven reminders); no shared activity list.

### Option A: Postgres notification inbox + direct FCM HTTP v1 from the Go service

- Table `notifications` (account-owned): type, payload (title/body/deep-link target), created_at,
  read_at, plus a feature-supplied **idempotency key** with a UNIQUE constraint so a producer can
  never create the same notification twice.
- Read API for all three platforms: list newest first with cursor paging (one month per page by
  default), unread count, mark one / all as read.
- Device push tokens stored against `user_devices` (or a child table keyed by it): registered by
  the app after login and on token refresh, removed on logout, pruned when FCM answers
  `UNREGISTERED`/`INVALID_ARGUMENT`.
- After a producer inserts a notification, the backend sends it to every push token of that user
  through **FCM HTTP v1** (one provider for both iOS — via the APNs key uploaded to Firebase — and
  Android), authenticated with Application Default Credentials. A new `pkg/push` package following
  the repo's interface + driver pattern (`pkg/push/fcm`).
- Push failure never removes the in-app notification; the inbox is the source of truth, push is
  best-effort.

| Dimension | Assessment |
|---|---|
| Complexity | Medium — 1 table, 1 token table/column, 3–4 endpoints, 1 small `pkg/push` driver |
| Cost | FCM free; storage negligible at < 10k users |
| Reversibility | Medium — the inbox API and token registration ship in app releases |
| Cross-platform impact | BE: new module. Web: notification list. iOS/Android: list + FCM SDK + token registration + permission prompt |
| Security & privacy | Tokens are per device, deleted on logout; payload kept minimal (see Open questions); FCM is a Google processor |
| Reliability | Push is synchronous best-effort; a failed send is logged, not retried beyond FCM's own delivery |
| Fit with current stack | High — Postgres, Clean Architecture module, `pkg/` driver pattern, ADC like `pkg/secrets/gcp` |
| Familiarity | FCM is new to the project |

**Pros:** smallest moving parts; no queue; reuses DB and service. **Cons:** a send burst runs inside
the producer (job or request); no automatic retry of failed pushes.

### Option B: Postgres inbox + Pub/Sub (or Cloud Tasks) fan-out to a push worker

Same inbox and token model as A, but producers publish a message and a separate worker sends pushes
with retries and dead-lettering.

| Dimension | Assessment |
|---|---|
| Complexity | High — topic/queue, worker service, IAM, DLQ, idempotent consumer |
| Cost | Low but non-zero; another service to deploy and watch |
| Reversibility | Medium |
| Cross-platform impact | Same as A for clients |
| Security & privacy | Same as A, plus one more service identity |
| Reliability | Better: retries, backpressure |
| Fit with current stack | Lower — the infra design has no queue yet |
| Familiarity | Low |

**Pros:** scales and retries. **Cons:** over-built for < 10k users and a few notifications per user
per month.

### Option C: Third-party notification service (e.g. OneSignal)

Hosted inbox + push. Rejected on a first pass: user data and relatives' names leave to another
processor, vendor lock-in for the inbox the website also reads, and paid tiers for features A gets
from Postgres.

## Trade-off analysis

A and B share the data model and client contract, which is the expensive part to change; they differ
only in how pushes are sent. At the confirmed scale, B's retries buy little — the in-app inbox is
already the durable record, so a lost push degrades to "seen next time the app opens". A keeps one
service and no queue, matching the infra design's "no queue until needed". Moving A → B later is a
backend-only change (producers publish instead of calling `pkg/push` directly), so choosing A now is
reversible where it matters. Option 0 does not meet the owner's requirements; C fails on privacy and
lock-in.

## Decision

**Recommended:** Option A — Postgres inbox as the single source of truth on all three platforms, push
sent directly through FCM HTTP v1 for iOS and Android, because it meets every requirement with the
fewest new parts and can move to a queue later without touching clients.

## Consequences

- **Easier:** any feature can notify a user by inserting one row with an idempotency key; the
  website, iOS and Android show the same list.
- **Harder:** both apps must integrate the FCM SDK, ask for notification permission and keep the
  token registered; a Firebase project per environment (dev, prod) must be maintained.
- **Revisit when:** sends per run exceed what one job execution handles comfortably (order of 10k
  pushes), push failures become user-visible, or a second producer needs retries → move to Option B.

## Follow-ups

| Layer | Follow-up |
|---|---|
| Product spec (`ba-writer`) | Memorial product spec cites ADR-0001 for where reminders appear (FR-MEM-008 becomes "in-app on all platforms + push on apps"); notification copy, tap target, permission-denied behavior |
| API contract (`spec-writer` BE) | New: list notifications (cursor, newest first), unread count, mark read / mark all read, register / delete push token |
| Backend spec / code | New `notification` module + migration; `pkg/push` + `pkg/push/fcm`; token cleanup on logout and on FCM invalid-token answers; never log notification bodies |
| Website | Notification list + unread badge (no push) |
| iOS / Android | Notification list, FCM SDK, permission prompt, token registration/refresh/removal, deep link on tap |
| Infrastructure | Firebase project per env (owner registers); APNs auth key uploaded to Firebase; service account of `app-api` gets FCM send permission; update infra design v0.3 |

## Open questions

- ⚠️ Not confirmed: may a push show the relative's name on the lock screen, or only the relationship ("Ngày mai là giỗ cha")? — blocks the push payload rule; product decision for `ba-writer` (NFR-MEM-02).
- ⚠️ Not confirmed: is there an account-wide switch to turn all notifications (or all push) off? — blocks the settings model; product decision for `ba-writer`.

### Resolved

- 2026-10-14 — Which platforms show notifications? → In-app on website, iOS, Android; push on iOS and Android only (anh Đức).
- 2026-10-14 — Retention? → Kept until read and beyond; list loads one month at a time with "load more" (anh Đức).
- 2026-10-14 — Scale? → Under 10k users in the first year (anh Đức).
- 2026-10-14 — Push provider? → FCM; anh Đức registers the Firebase project (anh Đức).

## Decision record

- 2026-10-14 — Proposed by Claude (`adr-writer`).
