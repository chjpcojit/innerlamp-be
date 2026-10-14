# Design process (system-design + architecture, adapted to InnerLamp)

This is the process of the generic `engineering:system-design` and `engineering:architecture`
skills, narrowed to what InnerLamp needs. Use only the sections that fit the decision.

## 1. Context that always applies (verify, then state in the ADR)

Facts to check against the repo before relying on them — `CLAUDE.md` is the starting point, the
code is the authority:

- Backend: Go service(s) in Clean Architecture layout under `services/`, shared infrastructure
  packages under `pkg/` using the interface + factory/registry pattern; Postgres; containerized for
  GCP (Cloud Run / GKE) with AWS abstractions kept but secondary.
- Clients: SvelteKit website (server talks to backend, browser does not by default), native iOS,
  native Android (Kotlin/Compose). Each has its own architecture doc in its repo.
- Team and budget: a personal project with a very small team — operational simplicity and monthly
  cost weigh heavily. Treat these as constraints, but confirm the actual numbers with the user.

## 2. Questions to ask before writing (Vietnamese, prioritized)

Ask only what applies; one topic per question, with a recommended default.

1. **Scope** — which platforms the decision affects now and later.
2. **Scale** — expected users / records / requests (order of magnitude is enough) now and in a year.
3. **Availability & latency** — what the user may tolerate when the backend is down or slow.
4. **Cost** — monthly budget ceiling for new infrastructure.
5. **Security & privacy** — personal data involved, who may access it, encryption, deletion.
6. **Offline** — must it work without network, on which platforms.
7. **Timeline** — what must ship first; can a simpler option ship now with a planned migration?
8. **Existing data** — data already in production or on devices that must be migrated.

Unanswered → `⚠️ Not confirmed` inline and in "Open questions".

## 3. High-level design (per option)

Components and data flow in a few lines or an ASCII/Mermaid diagram; where the data lives; which
platform owns which step; what changes in the API contract (named, not specified — the contract is
written later by `il-spec-be`).

## 4. Assessment dimensions (same table for every option, status quo included)

| Dimension | What to assess |
|---|---|
| Complexity | Build + operate effort for a very small team |
| Cost | Expected monthly cost at the confirmed scale |
| Reversibility | What it takes to undo: data migration, app release, contract change |
| Cross-platform impact | Work required on backend / website / iOS / Android |
| Security & privacy | Trust boundary, data exposure, compliance with the project's security baseline |
| Reliability | Failure modes, behavior when a dependency is down |
| Fit with current stack | Reuses existing `pkg/` abstractions and patterns, or adds a new one |
| Familiarity | Whether the team already runs this technology |

Rate Low / Med / High (or a short phrase) and add pros/cons. The recommendation must follow from the
table, not contradict it.

## 5. Deep-dive checklists by decision type

**Infrastructure** — managed vs self-hosted; environment isolation (dev/staging/prod); IaC;
secrets via the secrets abstraction; observability (logs, metrics, tracing); failure and retry;
cost at 10× scale; exit path to another provider.

**Data model** — entities and ownership (account vs device); identifiers (server vs client-generated,
format); invariants and constraints the DB enforces; idempotency of writes; migration of existing
data and rollback; soft vs hard delete and account deletion; indexes for the main queries.

**Offline sync** — source of truth; change tracking (timestamps, versions, op log); conflict policy
(last-write-wins, merge, user choice) per field if needed; deletes/tombstones; client-generated IDs
and idempotent upload; clock skew; initial sync and re-install; behavior across two devices;
what each client stores locally.

**Auth / security** — token types and lifetimes, rotation and revocation; device/session binding;
where secrets live on each client (Keychain / Keystore / httpOnly cookie); brute-force and replay
protection; audit logging; least privilege; impact on existing sessions when changed.

## 6. Consequences

List what becomes easier, what becomes harder, what must be revisited (with a trigger — "when users
exceed N", "before public launch"), and the follow-ups per layer: product spec, contract, each
platform spec, infrastructure.
