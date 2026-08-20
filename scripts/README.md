# scripts Directory

Static-check scripts for the `golang-clean-arch-tdd` workflow, plus DB tooling for `appService`. Run individually or all together via `run-all.sh`.

## Contents

Clean-architecture / TDD static checks (`check-*.sh`):

- `check-api-standards.sh`, `check-architecture.sh`, `check-compile.sh`, `check-completeness.sh`, `check-context-propagation.sh`, `check-coverage.sh`, `check-ctx-deadline.sh`, `check-db-transactions.sh`, `check-dependencies.sh`, `check-error-wrapping.sh`, `check-gcp-config.sh`, `check-grpc-errors.sh`, `check-idempotency.sh`, `check-mock-data.sh`, `check-mongo-indexes.sh`, `check-pubsub-patterns.sh`, `check-rate-limiting.sh`, `check-security.sh`

Domain/business-rule checks specific to this project's user stories:

- `check-ba-br-err-coverage.sh`, `check-ba-sensitive-fields.sh`, `check-business-rules.sh`, `check-cnbv-data-retention.sh`

Migrations:

- `check-migrations.sh` — validates `../migrations/`
- `cleanup-obsolete-migrations.sh`
- `migrate.sh` — golang-migrate runner, see `../migrations/README.md`

Other:

- `import-mx-postcodes.sh` — one-off data import, unrelated to InnerLamp's VN domain (leftover from template — verify before running)
- `run-all.sh` — runs all `check-*.sh` scripts, writes to `report/`
- `setup-harness.sh` — one-time environment setup
- `report/` — output of the last `run-all.sh` run (`check-result.json`, `error-report.md`), regenerated each run, not hand-edited

## Notes

Update this README when the folder structure or responsibilities change — this list must match `ls scripts/*.sh`, not a template checklist. (It previously listed several `.sh`/`.js` files that don't exist in this repo; verify against `ls`/`find` before trusting it, same caution as the generated `openwiki/` index.)
