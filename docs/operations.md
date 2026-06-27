# Operations Runbook

Operational guidance for running Teamverwaltung in production. Pairs with the
environment-variable reference in `CLAUDE.md`.

## Database backup & restore

All durable state lives in PostgreSQL (`postgres:17`). The container's data
volume alone is **not** a backup — take logical dumps on a schedule and store
them off-host.

### Scheduled logical backups

```bash
# Nightly compressed custom-format dump (retain off-host, e.g. object storage).
pg_dump --format=custom --no-owner --dbname="$DATABASE_URL" \
  --file="teammanager-$(date +%F).dump"
```

Run it from a cron job / Kubernetes CronJob against the production DSN. Keep at
least 7 daily + 4 weekly copies; encrypt at rest and verify restores regularly
(a backup you have never restored is not a backup).

### Restore

```bash
# Into a fresh, empty database.
createdb teammanager_restore
pg_restore --no-owner --clean --if-exists \
  --dbname="postgres://USER:PASS@HOST:5432/teammanager_restore" \
  teammanager-2026-06-26.dump
```

Application migrations are idempotent (goose); after restore, the backend runs
any pending migrations automatically on startup.

### Point-in-time recovery (PITR)

For tighter RPO than nightly dumps, enable WAL archiving (`archive_mode=on` +
`archive_command`) or use a managed Postgres offering with continuous backup.
Logical dumps remain the simplest portable baseline.

### What is safe to lose

- `river_*` job-queue tables: in-flight background jobs (notifications). Losing
  these drops pending notifications but not core data.
- Session rows: users simply re-authenticate.

## Metrics endpoint

`/metrics` (Prometheus) is unauthenticated by default for in-cluster scraping
over a private network. To expose it on an untrusted network, set
`METRICS_TOKEN` and configure the scraper with
`Authorization: Bearer <token>`.

## Container images & releases

Tagging a release (`vX.Y.Z`) triggers `.github/workflows/release.yml`, which
builds and pushes versioned backend and frontend images to GHCR. Deploy by
pinning the image to the released tag; roll back by redeploying the previous
tag (images are immutable per digest).
