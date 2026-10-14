# When an ADR is needed

An ADR is for a decision that is **hard to reverse** — undoing it later means migrating data,
re-releasing mobile apps, changing a public contract, or rebuilding infrastructure. Most features
need none. Apply this gate before doing any design work.

## Needs an ADR (any one is enough)

| Area | Examples in InnerLamp |
|---|---|
| **Infrastructure** | Adding a cloud service (queue, cache, object storage, CDN, scheduler, search), changing where/how a service runs (Cloud Run ↔ GKE), a new external provider (SMS, push, maps, payment), a new database or a second datastore |
| **New or reshaped data model** | A new aggregate shared by several features, changing ownership of data (device-local ↔ account), splitting/merging tables that already hold user data, a model that must be identical on all platforms, retention/deletion rules that shape the schema |
| **Offline sync** | Any data the user edits while offline on a device: conflict resolution, IDs generated client-side, tombstones, sync protocol, what happens on two devices |
| **Auth / identity / security** | Token model or lifetime, session/device binding, a new auth factor or sign-in method, roles and permissions, encryption of user data, anything that changes the trust boundary between clients and backend |
| **Cross-cutting conventions** | Pagination/versioning/error envelope for the whole API, a new architectural layer or service boundary, a new cross-platform library or framework every client must adopt |
| **Technology choice** | "X or Y" between technologies when the loser would be expensive to switch to later |

## Does not need an ADR

- A feature that reuses the existing stack and patterns unchanged (a new CRUD endpoint on the
  existing Postgres + Clean Architecture layout, a new screen on an existing pattern).
- A product decision — what the user sees, a business rule, a limit (→ `ba-writer`, `BR-*`).
- A local implementation choice inside one platform that can be changed in one PR without data
  migration or a client release (→ that platform's `design.md`).
- A decision already settled by an Accepted ADR (cite it instead).
- A known shortcut (→ `docs/tech-debt/`) or a guess made while coding (→ `docs/assumptions/`).

## Borderline — ask the user

If the decision is reversible today but will become expensive once real users have data (e.g. the
first version of a sync format), say so and ask whether to write the ADR now. Recommend writing it
when the feature ships to mobile apps, because a released app version cannot be recalled.
