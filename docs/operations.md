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

This repo does not ship a CronJob/scheduler manifest, since the scheduling
mechanism is deployment-topology-specific (Kubernetes CronJob, systemd timer,
managed-Postgres provider backup, ...). Whichever mechanism is used, wire up
a periodic *restore* test (e.g. restore the latest dump into a scratch
database and run a trivial query) — a backup pipeline that only ever writes
and never restores can silently produce unusable dumps for months.

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

## JWT key rotation

Sessions are signed with `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY` (RS256). Unlike
`COOKIE_ENCRYPTION_KEYS`, there is no built-in dual-key rotation for these —
rotating them invalidates every existing session immediately (all holders
must re-authenticate). To rotate:

1. Generate a new RSA-2048 key pair.
2. Deploy the new `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY` during a maintenance
   window (accept that all active sessions are invalidated).
3. Communicate the forced re-login to users ahead of time if possible.

Rotate on a suspected key compromise, or on a routine schedule aligned with
your organization's key-management policy.

## Trace sampling

`OTEL_EXPORTER_OTLP_ENDPOINT` enables tracing with the SDK's default sampler
(parent-based, always-on), i.e. 100% of requests are traced when enabled. For
production traffic beyond low volume, configure a probabilistic sampler via
the standard OpenTelemetry SDK environment variables, e.g.:

```
OTEL_TRACES_SAMPLER=traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1   # sample 10% of requests
```

Keep sampling at 100% in staging/low-traffic environments where full
visibility matters more than collector load.

## Rate limiting

The global (`RATE_LIMIT_RPS`) and login brute-force (`LOGIN_RATE_LIMIT_PER_MIN`)
limiters key on the client's IP address. By default (`TRUSTED_PROXY_CIDRS`
unset) that is the raw TCP peer address of the connection — client-supplied
`X-Forwarded-For`/`X-Real-IP`/`True-Client-IP` headers are ignored, so a
direct client cannot bypass rate limiting by spoofing them.

**If the backend runs behind a reverse proxy or load balancer**, every real
client will appear to share the proxy's IP unless you set
`TRUSTED_PROXY_CIDRS` to the proxy's address range (e.g. your cluster's
internal CIDR or the load balancer's known egress range). Only once the
immediate TCP peer falls within that range are the forwarded-IP headers
honored — this keeps the bypass protection while still supporting the common
deployment topology. Get the CIDR wrong (too broad) and you reopen the
spoofing bypass; get it wrong (too narrow or unset) and all clients behind
the proxy share one rate-limit bucket.

Rate limiting is also per-instance (in-memory, not shared across replicas).
In a multi-replica deployment the effective limit scales with replica count
— size `RATE_LIMIT_RPS`/`LOGIN_RATE_LIMIT_PER_MIN` accordingly, or put a
rate limiter in front (API gateway, WAF) if you need a hard global cap.

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
