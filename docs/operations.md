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

`helm/team-manager/templates/backup-cronjob.yaml` ships exactly this: a
`--format=custom` dump on `backup.schedule` (disabled by default — set
`backup.enabled=true`), uploaded to S3-compatible object storage when
`backup.s3.enabled=true` (otherwise the job intentionally fails with a
warning, since an unpersisted dump discarded with the pod isn't a backup).
Whichever mechanism is used, wire up a periodic *restore* test (e.g. restore
the latest dump into a scratch database and run a trivial query) — a backup
pipeline that only ever writes and never restores can silently produce
unusable dumps for months.

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

### Rolling upgrades & schema-changing migrations

The Helm chart's Deployment has no explicit `strategy:`, so it uses
Kubernetes' default `RollingUpdate`: with `replicaCount > 1` (the default and
prod values), old- and new-version pods run concurrently for the whole
rollout window, each applying migrations via their own `initContainer` (the
migration runner itself is safe under this — concurrent execution across
replicas is serialized via a Postgres session-level advisory lock). CI's
`backend-migration-safety` job only checks *lock-duration* safety
(unindexed `CREATE INDEX`, `ALTER COLUMN ... TYPE`, unvalidated `CHECK`
constraints) — it does not, and cannot, check whether a migration is
*semantically* backward-incompatible with the old binary still serving
traffic during that window. `00008_amount_cents.sql` (converting
`transactions`/`penalties`/`contributions.amount` from euro floats to integer
cents in place) is a concrete example of the shape to watch for: had that
migration shipped under a live rolling upgrade with replicas > 1, the
still-running old-version pods would have read/written the new column
expecting the old type for the duration of the rollout. For any future
migration that changes a column's *meaning* (not just its lock duration),
either use the standard expand/contract pattern (add the new column, dual-write
from both binary versions, backfill, then drop the old column in a later
release) or scale to a single replica / use `Recreate` strategy for that one
deploy so no two binary versions serve traffic against the changed schema
simultaneously.

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

**`METRICS_TOKEN` is not merely a recommendation once `COOKIE_SECURE=true`**
(the production default): the backend fails startup outright
(`os.Exit(1)`, every replica crash-loops, not just a logged warning) if
`METRICS_TOKEN` is empty in that case. Either set `METRICS_TOKEN`, or set
`METRICS_ALLOW_OPEN=true` if `/metrics` is already restricted at the network
layer and you accept it being unauthenticated. The Helm chart's
`templates/NOTES.txt` prints a reminder about this at deploy time, since
`values.yaml`'s `existingSecret` doesn't create the Secret's contents for
you.

## Alerting & dashboards

`helm/team-manager/files/prometheus-rules.yaml` defines the alert rules for
this service (availability, error rate/latency, rate-limit spikes, login
failure/bulk-deletion anomalies, DB pool exhaustion, retention job health,
backup job health, memory/disk pressure). The backup CronJob's two rules
(`BackupCronJobFailed`, `BackupCronJobStale`) rely on kube-state-metrics
(`>= 2.6.0` for `kube_cronjob_status_last_successful_time`) and match job
names by suffix (`.+-backup.*`) rather than the chart's templated fullname,
since this file is embedded verbatim, not Helm-templated — adjust the
regexes if you set `fullnameOverride`/`nameOverride`. When `monitoring.enabled: true` and Prometheus
Operator is installed, the chart applies these automatically via a
`PrometheusRule` (`templates/prometheusrule.yaml`) alongside the
`ServiceMonitor` that sets up scraping — no extra step needed. If you run a
standalone Prometheus without the Operator, load the same file directly via
its `rule_files:` config instead.

`helm/team-manager/files/grafana-dashboard.json` is a starter Grafana
dashboard covering the same signals. Set `monitoring.grafanaDashboard.enabled:
true` to have the chart render it as a `ConfigMap` labeled
`grafana_dashboard: "1"` for the standard kube-prometheus-stack Grafana
sidecar to auto-import; otherwise import the JSON file manually.

## Container images & releases

Tagging a release (`vX.Y.Z`) triggers `.github/workflows/release.yml`, which
builds and pushes versioned backend and frontend images to GHCR. Deploy by
pinning the image to the released tag; roll back by redeploying the previous
tag (images are immutable per digest).

### Frontend image: pointing it at a backend

The frontend image is built once per release and is environment-agnostic —
which backend it talks to (and which Sentry project, if any, it reports
errors to) is resolved at **container start**, not baked in at build time, so
the same image tag can be deployed to staging and production unchanged. Set
the `API_BASE_URL` and (optionally) `SENTRY_DSN` environment variables on the
container:

```
docker run -e API_BASE_URL=https://api.example.com -e SENTRY_DSN=https://key@o0.ingest.sentry.io/1 \
  ghcr.io/<org>/team-manager-frontend:vX.Y.Z
```

An entrypoint script (`frontend/docker/docker-entrypoint-runtime-config.sh`)
regenerates `config.js` from these env vars (and the page's CSP
`connect-src` from `API_BASE_URL`) before nginx starts. Leaving
`API_BASE_URL` unset serves the app against its built-in in-memory mock
backend (useful for a quick demo/preview, but not a real deployment) —
always set it in staging/production. If the backend is reachable on a
different origin than the frontend, that origin also needs
`ALLOWED_ORIGINS` on the backend to include the frontend's origin (see the
environment variable table in `CLAUDE.md`) so the browser's CORS preflight
succeeds.

`SENTRY_DSN` has no build-time equivalent that reaches the release image —
the release workflow only ever passes `VITE_BUILD_VERSION`/
`VITE_BUILD_COMMIT` as build args — so this runtime env var is the *only*
way to enable Sentry error tracking in a released frontend image. Leaving it
unset disables Sentry, matching today's default.

Note: there is currently no Helm/Kubernetes manifest for deploying the
frontend image itself (only the backend has one under `helm/team-manager/`);
until one exists, deploy the frontend container by whatever means fits your
infrastructure (a plain Deployment/Service, a static host that proxies to the
image, etc.), setting `API_BASE_URL` as above.
