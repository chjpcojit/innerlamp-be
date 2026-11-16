# ADR-0005: The dev database moves from Cloud SQL to Neon

| Field | Value |
|---|---|
| Status | Accepted |
| Date | 2026-11-16 |
| Deciders | anh Đức (requested) · ♛ Chưởng Môn ♛ (implements) |
| Features | infrastructure only |
| Platforms | Backend |
| Supersedes | The database part of ADR-0004 |
| Related | ADR-0004 (Cloud Run + Terraform) · `infra/terraform/README.md` |

## Context

ADR-0004 put Cloud SQL PostgreSQL 16 (`db-f1-micro`, zonal) behind appService.
The stack worked, but the instance cost about 10.3 USD/month running 24/7, and
that was the only real cost in an otherwise free stack. Stopping the instance
would not have removed the charge: Cloud SQL keeps billing the IPv4 address
while an instance is stopped (~0.014 USD/hour).

The owner asked for a free SQL database and approved moving to Neon's free plan.
The dev database held schema only — no seed data had been loaded — so the move
carried no data to migrate.

## Decision

- **PostgreSQL on Neon Free**, region Singapore (`asia-southeast1`), database
  `neondb`, role `neondb_owner`, TLS required (`sslmode=require`).
- The password lives in Secret Manager (`innerlamp-db-password`, fed from
  `var.neon_password`, a git-ignored tfvars value); it is never in the repo or
  the image.
- Terraform no longer creates a database. `infra/terraform/sql.tf`, the
  `cloudsql.client` grant and the Cloud SQL socket mounts are removed; the
  service and the migration job connect over TCP.
- The first apply added Neon alongside Cloud SQL, the migration job was run
  against Neon, the API was verified, and only then was `innerlamp-pg` deleted.

## Alternatives considered

- **Stop the Cloud SQL instance** — does not remove the idle IPv4 charge, and
  breaks the API while the migration is not done.
- **Supabase Free** — 500 MB and projects pause after a week of inactivity;
  Neon allows 1 GB and scales compute to zero without a pause-then-resume step.
- **Keep paying ~10 USD/month** — rejected: the owner asked for zero cost in dev.

## Consequences

- Monthly infrastructure cost is now ~0 USD at dev usage.
- The database is outside GCP, so Cloud Run reaches it over the public internet
  with TLS. Latency to Singapore is a few milliseconds.
- Neon free suspends compute after idle; the first query after that adds about a
  second. `max_instances = 2` keeps connection pressure low.
- No automated backups and a 1 GB cap: acceptable for dev, not for real data.
- Deleting Cloud SQL is irreversible. There was no application data in it.
