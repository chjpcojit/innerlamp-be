# Changelog — onboarding Spec

Tracks version history for `requirements.md`, `design.md`, and `tasks.md` in this folder. All three
files share one changelog since they change together as one spec unit. Newest version first.

## v1.2 (2026-09-16)

- **Breaking API change.** The phone field on the wire is renamed `phone_number` → `phone`, in both
  requests and responses, so one name is used end to end instead of request and response drifting
  apart. Affects `POST /v1/auth/register` (request and the `user` in its response) and
  `GET`/`PATCH /v1/users/me`. Existing clients sending `phone_number` now fail with
  `INVALID_PHONE` (the key is absent, so `required` fires); clients reading `phone_number` off a
  response get nothing.
- `requirements.md`: US-001 invalid-phone criterion now says `details {"field":"phone"}`; US-004's
  `GET /v1/users/me` field list and its `PATCH` ignore-list renamed to match.
- `design.md`: the register step-by-step flow's failure line renamed to `{field: phone}`.
- `tasks.md`: no content change, version bumped to stay in step with the other two files.
- **Not renamed:** the database column stays `users.phone_number`, along with the unique constraint
  `users_phone_number_key` and every SQL statement referencing them. The repository maps columns to
  struct fields by hand rather than by tag, so the wire name and the column name are free to differ
  — and renaming a `UNIQUE NOT NULL` column on a live table buys nothing here.

## v1.1 (2026-09-15)

- `requirements.md`: no behavior change. Fixed three stale cross-references in the "Pending" list —
  items 1, 2 and 3 pointed at `tasks.md` Phase 5, but Phase 5 is "Integration & verification" and
  the open work actually lives in Phase 6. Rewritten to name the specific item (`Phase 6 item 1`,
  `item 3`, `items 5-6`) so the pointer survives further renumbering. The `⚠️ Not confirmed`
  marker on US-004's 401 criterion was left open — the gap it flags is still real.
- `design.md`: no structural change to any flow. Added a "Known gaps" bullet recording that no
  refresh endpoint exists and `token_version` is never bumped past the hardcoded `1` in
  `insertSession` — the fact that makes US-001's "rotate mỗi lần dùng" note inaccurate. Steps 1-4
  of every flow, the Layers table and the transaction boundaries are unchanged.
- `tasks.md`: header bumped for consistency. Two new Phase 6 items — decide whether refresh-token
  rotation is in scope for this feature, and resolve the access/refresh token lifetime
  contradiction (config says `15m`/`168h`, the original user story says 15 minutes / 30 days).
  Also tightened the avatar item to a dimension/decompression-bomb check rather than a vague
  "content check". No item was checked off — nothing new was implemented.
- Reason: a verification pass against the real code found the new `tasks.md` Phase 6 had grown
  past what `requirements.md`'s "Pending" list described, and that the refresh-token gap — real in
  the code but only implied by a `⚠️` marker in `requirements.md` — had no task or design bullet
  backing it. This version closes both gaps; it is a correction of the v1.0 retroactive write, not
  a new requirement.

## v1.0 (initial)

- `requirements.md`: initial version. The file previously carried four Vietnamese user stories
  (US-001..US-004) in "Given/When/Then" prose, moved here from
  `docs/specs/features/onboarding/onboarding.md` when the four-file spec convention was adopted.
  The acceptance criteria were rewritten in EARS notation and given the API contract each one maps
  to (status code + error code). The original user-story prose, field table and "Ghi chú kỹ thuật
  liên quan" notes are preserved verbatim under `## Original user stories`, so nothing was lost in
  the rewrite.
- `design.md`: initial version, written retroactively from the shipped code.
- `tasks.md`: initial version, written retroactively. Phases 1-5 checked off against files that
  were read directly; Phase 6 lists what is genuinely still open, including the middleware gap
  that makes US-004 unroutable today.
- Reason: the repo adopted a four-file spec convention (`requirements.md` / `design.md` /
  `tasks.md` / `changelog.md`) under `docs/specs/features/<feature>/` to match the per-folder
  README convention and to give `design.md` somewhere to record the transaction boundaries and
  hidden steps that a user story cannot express. `onboarding` is the first feature converted; the
  original single-file user story was renamed to `requirements.md` and extended rather than
  replaced.

Cross-file follow-ups recorded at this version:

- `docs/specs/api/openapi.yaml` — verified to cover all six routes; must be re-checked whenever a
  handler changes.
- `docs/specs/db/register-erd.md` — the ERD sourced from the original onboarding user story; update
  if migrations 0001-0004 change.
- `docs/knowledge/features/` — has no identity/onboarding concept yet (it covers only the six
  content features). If one is added later, `bdd-okf-docs` owns it; this spec does not write there.
