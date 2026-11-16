# ADR-0004: The backend runs on Cloud Run with Cloud SQL, provisioned by Terraform

| Field | Value |
|---|---|
| Status | Accepted (dev) |
| Date | 2026-11-16 |
| Deciders | anh Đức (requested the deployment) · ♛ Chưởng Môn ♛ (proposes) |
| Features | infrastructure only |
| Platforms | Backend |
| Supersedes | — |
| Related | `infra/terraform/README.md` · ADR-0003 (secret handling in config) |

## Context

The backend had only ever run on a developer machine: `docker compose` brought up
PostgreSQL, and appService ran from the host with `config.dev.yaml`, whose
`dev_secrets` block holds plaintext passwords. The owner asked for a real
deployment on GCP under the personal account `ngthanhducit@gmail.com`, with the
infrastructure described in Terraform rather than clicked in the console.

Constraints that shaped the decision:

- One service (`appService`); a backoffice service exists but is not deployed yet.
- The backend is a long-running HTTP server with a Postgres dependency, not a
  request-scoped function.
- This is a dev environment: cheapest tier, no HA, no production hardening.
- The workflow rule for Dev applies: the simplest thing that runs and passes its
  acceptance criteria; anything only needed for Prod/Beta is proposed separately.

## Decision

- **Cloud Run (v2)** runs `appService` from a distroless image, scale-to-zero,
  listening on 8080. `max_instances = 2` while rate limiting uses the in-memory
  store.
- **Cloud SQL for PostgreSQL 16**, `db-f1-micro`, zonal, reached through the
  Cloud SQL connector over the unix socket `/cloudsql/<connection>`; no
  authorized networks, no proxy sidecar.
- **Secret Manager** holds the database password, the JWT signing key, the web
  API key and the rendered `config.cloud.yaml`. The runtime service account gets
  `secretAccessor` **per secret**, not project-wide.
- **GCS** bucket for avatars, private with uniform access; the service signs V4
  URLs through `iam.serviceAccounts.signBlob` as itself.
- **Artifact Registry** stores two images: `app` (the service) and `migrate`
  (`postgres:16-alpine` plus `migrations/*.sql` and a small runner that records
  applied files in `schema_migrations`).
- **One service account**, `innerlamp-run`, is the only identity workloads use:
  `cloudsql.client`, `cloudtrace.agent`, `logging.logWriter`,
  `monitoring.metricWriter`, the four secrets, `objectAdmin` on the avatars
  bucket and `serviceAccountTokenCreator` on itself.
- **Terraform** in `infra/terraform/` owns all of the above. State is local for
  now (single operator, dev).
- `main.go` chooses its secret resolver: `SECRETS_PROVIDER=gcp` reads Secret
  Manager with the runtime identity; otherwise the dev plaintext map is used.

## Alternatives considered

- **GKE Autopilot** — closest to a production target, but a cluster, ingress and
  release plumbing are more than a dev deployment needs.
- **A Compute Engine VM** — cheapest to reason about, but patching, TLS and
  process supervision become the project's problem.
- **Cloud Functions** — fits request-scoped handlers, not an existing HTTP server
  with background work and shared middleware.
- **Deploying backofficeService too** — possible with the same module, but it
  bootstraps a default `admin` account in dev and nothing needs it yet.

## Consequences

- Cost while idle is dominated by Cloud SQL (~9–10 USD/month); Cloud Run scales
  to zero.
- The deployment runs with `app.environment: "dev"`: OTP is the fixed `111111`,
  mail is logged, attestation is mocked. It must not take real users.
- Audio for dharma talks is not served from this environment yet (no GCS/CDN
  path); the relative `/static/audio/...` URLs 404 here.
- Seeding stays manual: the service does not seed on startup, and no seed job is
  in Terraform yet.
- Local Terraform state means two operators cannot apply concurrently. Moving to
  a GCS backend is the first step before a second environment.
- The container must be built for `linux/amd64`; an arm64 image fails with
  `exec format error` at startup.
