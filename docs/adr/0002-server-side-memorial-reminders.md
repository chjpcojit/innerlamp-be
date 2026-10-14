# ADR-0002: The backend computes and sends memorial (giỗ) reminders on an hourly schedule

| Field | Value |
|---|---|
| Status | Accepted |
| Date | 2026-10-14 |
| Deciders | anh Đức (decides) · Claude (`adr-writer`, proposes) |
| Features | `memorial` (sổ giỗ, memorial book) |
| Platforms | Backend, Website, iOS, Android |
| Supersedes | — (replaces the existing constraint recorded in `docs/product/memorial/product.md@1.0` §2.7, which had no ADR) |
| Related | ADR-0001 (delivery channel) · product spec `memorial@1.0`: BR-MEM-05…11, FR-MEM-007, FR-MEM-008, FR-MEM-009, §2.7, §3.6 |

## Context

Product spec v1.0 §2.7 records a constraint from 2026-10-06: the server only stores the date of
death and the reminder settings; each app computes the next anniversary and schedules its own local
notifications; the website has no reminders. On 2026-10-14 the product owner reversed it:

- Reminder settings are available on every platform (the owner adds the missing UI on the website).
- The **backend** decides when a reminder is due and creates the notification; delivery follows
  ADR-0001 (in-app on all three platforms, push on iOS and Android).
- "Add to the iOS Reminders app" (FR-MEM-009) is dropped.
- The schedule runs **once per hour** and compares each user's time zone so everyone is notified at
  their own chosen local time; the reminder choices stay as in BR-MEM-11 (on the day / 1 / 3 / 7 days
  before, at 06:00 / 07:00 / 12:00 / 19:00, default 1 day at 07:00).

What exists today (checked on `main` @ 5f4ca56):

- `memorials` table with `reminder_days_before` (0–30), `reminder_time` (any `HH:mm`),
  `reminder_timezone` (per entry) and `add_to_os_reminder` (`migrations/0018_memorials.sql`).
- A Vietnamese lunar converter in the backend, `services/appService/domain/lunarcal` (UTC+7), already
  the reference implementation of BR-MEM-10 and the source of `docs/product/memorial/rules/*.json`.
- No scheduler, no job binary, no notification table (ADR-0001).
- Infra design v0.3 already plans "Cloud Scheduler → Cloud Run Job" for a cleanup job.

Constraints: under 10k users (≤ 100 entries each, BR-MEM-02); cost must stay within free tiers where
possible (Cloud Scheduler: 3 jobs free per billing account); the owner accepted an hourly run.

### Why this needs an ADR

It moves anniversary computation from three clients to the server, adds a scheduled job and derived
scheduling state to the data model, and changes what released apps must do — expensive to undo.

## Decision drivers

- Reminders must be identical on every device and stop immediately when an entry is deleted or its
  reminder turned off (no stale local notifications on other devices).
- Each user is notified at their chosen local hour, wherever they live.
- Never notify twice for the same anniversary, even if a run is retried or overlaps.
- A missed run (outage, deploy) must not silently drop reminders.
- Reuse `lunarcal` and the shared vectors; no second lunar implementation on the server.
- Minimal infrastructure and cost.

## Options considered

### Option 0: Keep the status quo (client-side local reminders)

| Dimension | Assessment |
|---|---|
| Complexity | Low on BE; high overall — iOS and Android each build a scheduler, website none |
| Cost | None |
| Reversibility | High |
| Cross-platform impact | Duplicated logic; website without reminders |
| Security & privacy | Nothing leaves the device |
| Reliability | Stale reminders on other devices after delete/edit until they reload |
| Fit with current stack | Fits |
| Familiarity | High |

**Pros:** no server work. **Cons:** contradicts the owner's 2026-10-14 decision.

### Option A: Hourly scan computing due reminders on the fly

Every hour the job finds users whose local time (account time zone) equals one of the reminder hours,
loads their entries with a reminder, computes each next anniversary with `lunarcal`, and creates a
notification when `anniversary − lead days = local today` and the entry's hour matches. Duplicates
are blocked by an idempotency key `(memorial_id, anniversary_date)` (ADR-0001).

| Dimension | Assessment |
|---|---|
| Complexity | Low — stateless, no derived columns |
| Cost | Free tier; each run reads the entries of the users in that hour |
| Reversibility | High — no stored scheduling state |
| Cross-platform impact | Clients stop scheduling; same for A and B |
| Security & privacy | Same as B |
| Reliability | **A missed hour is lost** unless the scan also covers the gap since the last successful run, which needs a run log and re-derivation |
| Fit with current stack | High |
| Familiarity | High |

**Pros:** simplest. **Cons:** catch-up after a missed run is awkward; recomputes every entry each run.

### Option B: Precomputed `next_reminder_at` per entry + hourly due-sweep

- Each entry with a reminder stores `next_reminder_at` (UTC instant) and the `anniversary_date` it
  refers to. They are computed in Go with `lunarcal` from the entry's date, the lead and hour
  (BR-MEM-05…07, BR-MEM-11) and the **account time zone**, and recomputed when the entry is created or
  updated, when the user's time zone changes, and right after a reminder is sent (rolling to the
  following anniversary). Reminder off or entry deleted → nothing scheduled (row value cleared / row
  gone).
- Cloud Scheduler triggers a **Cloud Run Job** (same Go module, separate `cmd/` entry point) every
  hour. The job selects entries with `next_reminder_at <= now()` using `FOR UPDATE SKIP LOCKED` in
  small batches, inserts the notification with idempotency key `(memorial_id, anniversary_date)`
  (ADR-0001), sends push, then recomputes `next_reminder_at` — all per entry in one transaction for
  the DB part.
- **Catch-up:** a run after an outage picks up everything overdue; an overdue reminder whose
  anniversary has already passed is skipped and rolled forward (resolved 2026-10-14).
- **Time zone moves to the account:** `users.timezone` (IANA, default `Asia/Ho_Chi_Minh`), sent by
  the clients (device zone) on sign-in and when it changes. The per-entry `reminder_timezone` is
  retired. The lunar date itself is always the Vietnamese calendar (UTC+7, BR-MEM-10); the user's
  zone only decides "today" and the local hour.

| Dimension | Assessment |
|---|---|
| Complexity | Medium — two derived columns, recompute hooks, one job entry point |
| Cost | Free tier (1 scheduler job; job runs seconds); one indexed query per run |
| Reversibility | Medium — columns can be dropped; contract changes ship in apps |
| Cross-platform impact | Same as A |
| Security & privacy | Job logs ids only, never names/notes (NFR-MEM-02) |
| Reliability | Good — overdue rows are naturally caught up; SKIP LOCKED + UNIQUE key make retries and overlapping runs safe |
| Fit with current stack | High — Postgres, `lunarcal`, Cloud Run Job pattern from the infra design |
| Familiarity | High |

**Pros:** cheap indexed sweep, correct catch-up, easy to observe ("how many overdue"). **Cons:** derived
state must be recomputed on every relevant change; a bug there silently stops one entry's reminders
(mitigated by tests and a daily "overdue > 1h" metric).

### Option C: One scheduled task per reminder (Cloud Tasks with ETA)

Enqueue a task per entry at its exact due time; re-enqueue on change. Rejected on a first pass:
cancellation on edit/delete and re-enqueue after each send add a queue and its IAM for a problem an
hourly sweep solves; tasks also have a 30-day scheduling limit, shorter than many anniversaries.

## Trade-off analysis

A and B produce the same user-visible behavior; they differ on missed runs and per-run work. With a
fixed set of reminder hours, a missed hour under A means a missed reminder unless extra bookkeeping is
added — which turns A into B. B's derived columns cost a few recompute hooks, all inside the memorial
module the backend already owns. Both reuse `lunarcal` and the shared vectors, so the server's
computation is checked against the same cases as the clients'. C adds infrastructure without benefit
at this scale.

## Decision

**Decided:** Option B — store `next_reminder_at` per entry, sweep due entries hourly from a Cloud
Run Job triggered by Cloud Scheduler, deliver through ADR-0001, with the time zone on the account,
because it is the only option that is both cheap and correct after a missed run.

## Consequences

- **Easier:** reminders are identical everywhere and stop at once after delete/turn-off; the website
  gets reminders through its notification list; clients no longer need a scheduler.
- **Harder:** the backend now implements BR-MEM-05…07 and BR-MEM-11 (server must run
  `rules/next-anniversary.json` too); time zone handling moves to the account; one more deployable
  (job) to monitor.
- **Revisit when:** a single run handles more than ~10k due reminders, or the owner wants a reminder
  hour outside the fixed list (then the hourly granularity must change).

## Follow-ups

| Layer | Follow-up |
|---|---|
| Product spec (`ba-writer`) | MAJOR bump to v2.0: §2.7 cites ADR-0002; BR-MEM-05…07, 11 apply to Backend; FR-MEM-008 rewritten (in-app on all platforms + push on apps, server-driven); FR-MEM-009 removed; FR-MEM-007 and BR-MEM-11 on Website (no longer "later"); §3.6 rows updated; reminder time zone = account; catch-up rule; the 12 open product questions from `/mnt/project-files/reports/product/memorial/2026-10-14-cau-hoi-sau-spec-nen-tang.md` |
| API contract (`spec-writer` BE) | Memorial: narrow `reminder` to the product choices, drop `reminder.timezone` and `add_to_os_reminder` (versioning/compat to be decided there); account time zone on user profile / sign-in |
| Backend spec / code | Migration: `users.timezone`, `memorials.next_reminder_at` + `next_anniversary_date` + index, retire `reminder_timezone`/`add_to_os_reminder`; recompute hooks; `cmd/` job entry point; run all `rules/*.json` server-side; metrics: due, sent, overdue |
| Website | Reminder settings UI (owner adds), send device time zone, notification list (ADR-0001) |
| iOS / Android | Remove local scheduling and the Reminders-app toggle (iOS); send device time zone; FCM (ADR-0001) |
| Infrastructure | Cloud Scheduler job (hourly) → Cloud Run Job `memorial-reminders`, own service account; add to infra design v0.3 |

## Open questions

None.

### Resolved

- 2026-10-14 — Late run? → Send late as long as the anniversary day has not ended in the user's time zone; otherwise skip and roll forward (anh Đức).
- 2026-10-14 — Reminder moment already past when an entry is added or edited? → No immediate notification; schedule the following anniversary (anh Đức).

- 2026-10-14 — Who sends reminders? → The backend, via in-app notifications on all platforms and push on iOS/Android (anh Đức).
- 2026-10-14 — iOS "Add to Reminders app"? → Dropped (anh Đức).
- 2026-10-14 — Schedule granularity? → Hourly run comparing each user's time zone; stays within the GCP free tier (anh Đức).
- 2026-10-14 — Reminder choices? → Keep BR-MEM-11: 0/1/3/7 days × 06:00/07:00/12:00/19:00, default 1 day at 07:00 (anh Đức).
- 2026-10-14 — Time zone per entry or per account? → Per account, from the device (anh Đức, after explanation).

## Decision record

- 2026-10-14 — Proposed by Claude (`adr-writer`).
- 2026-10-14 — Open questions answered by anh Đức ("đồng ý" to both recommendations); merged as Proposed (be#8).
- 2026-10-14 — Accepted by anh Đức: "1. đồng ý" in reply to "Xác nhận 2 ADR là Accepted".
