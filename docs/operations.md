# Operations Runbook

Operational guidance for running Teamverwaltung in production. Pairs with the
environment-variable reference in `CLAUDE.md`.

## Legal setup before going public

Read this before pointing a public domain at a real deployment. Each instance
is run by exactly one operator (the club or whoever hosts it for a club).
Operator identity/contact data is resolved at **container start**, the same
mechanism the frontend already uses for `API_BASE_URL`/`SENTRY_DSN`/
`VAPID_PUBLIC_KEY` (see "Frontend image: pointing it at a backend" below) —
so the same built image can carry any operator's own legal-notice data
without a rebuild. The generic legal boilerplate (liability, dispute
resolution, GDPR purposes/rights/retention text) stays build-time source in
`frontend/src/features/legal/content.ts`, since it doesn't vary per operator
and is a poor fit for an env var either way. See
`openspec/changes/operator-data-runtime-config/design.md` for the full
reasoning (supersedes the build-time-only approach in
`openspec/changes/archive/2026-07-24-webapp-legal-compliance/design.md`
Decision 1).

**This is engineering's best-effort translation of publicly known German/EU
requirements into shipped defaults and this checklist — it is not legal
advice.** Get real legal review before a genuine public launch.

1. **Set the `OPERATOR_*` environment variables on the frontend container.**
   No rebuild required — set these alongside `API_BASE_URL` per "Frontend
   image: pointing it at a backend" below. These pages are reachable without
   login (footer on the login/registration screen) and from the profile
   sheet ("Rechtliches") inside the app, per `§5 DDG`'s "leicht erkennbar,
   unmittelbar erreichbar" requirement.

   Always-required — leaving any of these unset renders an explicit
   `[BETREIBER: ...]`/`[OPERATOR: ...]` placeholder marker on the legal-notice
   page (intentional: a loud, obvious gap beats a page that looks complete
   but is legally empty — don't deploy publicly with markers still showing):
   - `OPERATOR_NAME`, `OPERATOR_STREET`, `OPERATOR_POSTAL_CODE`,
     `OPERATOR_CITY` — the `§5 DDG` name/address block.
   - `OPERATOR_PHONE`, `OPERATOR_EMAIL` — the "Kontakt"/"Contact" section.

   Optional — leaving any of these unset omits the corresponding
   section/list item entirely (not a placeholder, since these genuinely
   don't apply to every operator/deployment):
   - `OPERATOR_LEGAL_FORM` (e.g. "Eingetragener Verein (e. V.)") —
     appended to the name/address block.
   - `OPERATOR_REPRESENTED_BY` — "Vertreten durch"/"Represented by" section.
   - `OPERATOR_REGISTER_COURT` + `OPERATOR_REGISTER_NUMBER` (both required
     together) — "Registereintrag"/"Register entry" section.
   - `OPERATOR_VAT_ID` — "Umsatzsteuer-Identifikationsnummer"/"VAT
     identification number" section.
   - `OPERATOR_DATA_PROTECTION_EMAIL` — the privacy policy's "Verantwortlicher"/
     "Controller" contact line; falls back to `OPERATOR_EMAIL` if unset.

2. **Data-processing agreements (Art. 28 GDPR) for enabled integrations,
   and the matching `OPERATOR_*_PROVIDER` disclosure.** Each of these is
   optional and off by default; if you turn one on for your deployment, you
   need a signed AVV/DPA with that provider before going live, **and** you
   must set the matching frontend env var below or the privacy policy's
   "Empfänger und Auftragsverarbeiter"/"Recipients and processors" section
   silently omits that processor (it does not infer disclosure from these
   backend vars being set — the actual provider/region has to be stated
   explicitly):
   - `S3_ENDPOINT` (+ `S3_BUCKET`/credentials) — object storage for photo/logo
     uploads → set `OPERATOR_S3_PROVIDER` (e.g. "self-hosted on our own
     infrastructure" or the vendor name + region).
   - `SMTP_HOST` (+ `SMTP_FROM_ADDRESS`) — outbound self-registration
     verification email → set `OPERATOR_SMTP_PROVIDER`.
   - `SENTRY_DSN` (backend) / `VITE_SENTRY_DSN` (frontend, set at container
     start per "Frontend image: pointing it at a backend" below) — error
     tracking → set `OPERATOR_SENTRY_PROVIDER`. See
     `frontend/src/monitoring.ts`'s comment above `initMonitoring` for the
     current cookie/storage determination (no consent banner needed for the
     integrations actually enabled there, re-verify if that changes) before
     you decide whether a consent banner is also needed for your deployment.
   - `OTEL_EXPORTER_OTLP_ENDPOINT` — tracing/telemetry collector, if it's a
     third-party or otherwise external service → set `OPERATOR_OTEL_PROVIDER`.

3. **Cross-check the data-subject-rights and retention documentation.**
   `SECURITY.md`'s "Data Protection (GDPR)" section and
   `docs/gdpr-data-subject-rights.md` describe what's already implemented
   (Art. 15 export, Art. 17 erasure, retention windows). Raise
   `RETENTION_AUDIT_LOG_DAYS` and friends (see `CLAUDE.md`'s environment
   table) if your organization's retention policy requires longer than the
   shipped defaults.

4. **Self-registration age gate.** `POST /auth/register` requires the
   registering person to confirm they're at least 16 (GDPR Art. 8; Germany
   kept the national threshold at 16) before the form submits. Younger club
   members must be added by a team admin via the invite flow instead — make
   sure your club's onboarding process for youth members actually does this.

5. **Accessibility (BFSG / EN 301 549) — assess, don't assume.** Whether
   Germany's Barrierefreiheitsstärkungsgesetz applies to a club-internal
   management tool (as opposed to consumer e-commerce/banking/media-access
   services, which are squarely in its Anlage 2 scope) is genuinely unclear
   without a real legal opinion specific to your deployment and audience —
   get one if accessibility compliance is a hard requirement for you. The
   existing UI relies on MUI's accessible primitives, keyboard navigation,
   and `vitest-axe` + Lighthouse CI in the test suite as a baseline, but that
   is not the same as a certified BFSG/EN 301 549 conformance statement.

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
Before uploading, the pg-dump container runs `pg_restore --list` against the
dump and fails the Job if it has fewer than `backup.minDumpEntries` (default
10) table-of-contents entries — this catches the case where `pg_dump` exits
0 but produced a near-empty/corrupt dump (e.g. `DATABASE_URL` momentarily
pointing at the wrong database) *before* it reaches S3 looking legitimate.
It is not a substitute for an actual restore test, though: wire up a
periodic *restore* test too (e.g. restore the latest dump into a scratch
database and run a trivial query) — a backup pipeline that only ever writes
and never restores can silently produce unusable dumps for months even when
every individual dump passes the TOC-entry-count check above.

`backup.retentionDays` is **informational only** — the chart does not
enforce it. Configure a matching S3 (or S3-compatible) bucket lifecycle rule
separately, or backups accumulate in `backup.s3.bucket` indefinitely.

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

### Disaster recovery: restoring into production

The steps above verify a dump is restorable into a scratch database — the
actual DR cutover, if the primary database is lost or corrupted, needs more
care:

1. **Stop writes first.** Scale the backend Deployment to 0
   (`kubectl scale deploy/<release> --replicas=0`) before restoring. A
   restore racing concurrent application writes can corrupt the target or
   have the app fail mid-request against a half-restored schema.
2. **Restore into the real target**, not a scratch DB this time —
   `pg_restore --no-owner --clean --if-exists --dbname=<production DSN>
   <dump>`. If the target host/database name is changing (e.g. failing over
   to a new Postgres instance), update `database.host`/`database.name`
   (plain `values.yaml` fields the chart composes `DATABASE_URL` from —
   see `helm/team-manager/README.md`'s "Secrets" section) and/or the
   password key in the Secret referenced by `database.secret.existingSecret`,
   then `helm upgrade` *before* the next step.
3. **Restart every pod**, not just scale back up — `DATABASE_URL` (like
   `JWT_PRIVATE_KEY`/`COOKIE_ENCRYPTION_KEY(S)` elsewhere in this doc) is
   read once via `config.Load()` at process start
   (`backend/internal/config/config.go`), so an already-running pod (if any
   survived) won't pick up a changed value without a restart —
   `helm upgrade` triggers one automatically for the plain `database.host`/
   `name`/`username`/`sslmode` fields (they're rendered directly into the
   pod spec), but changing only the Secret's password key still needs a
   manual `kubectl rollout restart` (same reasoning as the JWT key rotation
   section above).
4. **Mind the schema-version gap.** The restored dump reflects whatever
   goose migration state existed at backup time, which may be *behind* the
   currently-deployed app version (if migrations shipped between the backup
   and the incident) — the migrate initContainer applies anything pending
   automatically on the next pod start, same as a normal deploy, so this is
   usually transparent. It can matter in reverse too, if you're
   deliberately rolling back the app alongside the restore: see "Rolling
   upgrades & schema-changing migrations" below for why an old binary
   against a newer schema can be the more dangerous direction.
5. **Verify before scaling back up.** Check `SELECT version_id, is_applied
   FROM goose_db_version ORDER BY id DESC LIMIT 5;` looks sane, and
   spot-check a couple of core tables (`teams`, `memberships`) for
   plausible row counts before serving traffic again.
6. **Scale the backend back up** once the above checks pass.

Practice this end-to-end against a real (non-production) cluster at least
once — a restore procedure that's only ever been read, never run, is not a
tested procedure.

### Point-in-time recovery (PITR)

For tighter RPO than nightly dumps, enable WAL archiving (`archive_mode=on` +
`archive_command`) or use a managed Postgres offering with continuous backup.
Logical dumps remain the simplest portable baseline.

### What is safe to lose

- `river_*` job-queue tables: in-flight background jobs (notifications). Losing
  these drops pending notifications but not core data.
- Session rows: users simply re-authenticate.

### Rolling upgrades & schema-changing migrations

The Helm chart's Deployment leaves `strategy:` unset by default
(`deploymentStrategy: {}` in values.yaml), so it uses Kubernetes' own default
`RollingUpdate`: with `replicaCount > 1` (the default and prod values), old-
and new-version pods run concurrently for the whole rollout window, each
applying migrations via their own `initContainer` (the migration runner
itself is safe under this — concurrent execution across replicas is
serialized via a Postgres session-level advisory lock). CI's
`backend-migration-safety` job only checks *lock-duration* safety
(unindexed `CREATE INDEX`, `ALTER COLUMN ... TYPE`, unvalidated `CHECK`
constraints) — it does not, and cannot, check whether a migration is
*semantically* backward-incompatible with the old binary still serving
traffic during that window. A migration that converts a column's stored
representation in place (e.g. changing a monetary column from a float type
to integer cents) is a concrete example of the shape to watch for: if that
shipped under a live rolling upgrade with replicas > 1, the still-running
old-version pods would read/write the new column expecting the old type for
the duration of the rollout. For any future migration that changes a
column's *meaning* (not just its lock duration), either use the standard
expand/contract pattern (add the new column, dual-write
from both binary versions, backfill, then drop the old column in a later
release) or, for that one deploy, scale to a single replica (`--set
replicaCount=1`) *and* switch to `Recreate`
(`--set deploymentStrategy.type=Recreate`) so the old pod is fully terminated
before the new one starts — a `RollingUpdate` surge pod (default `maxSurge:
25%`) would otherwise still briefly run the new binary alongside the
still-terminating old one even at `replicaCount=1`, which is the exact
concurrent-old/new-binary condition this mitigation exists to avoid.

### Recovering from a migration killed mid-flight

The initial-setup migration (`00001_init.sql`) wraps every `CREATE TABLE`
in a single `-- +goose StatementBegin`/`StatementEnd` block, which Postgres
executes as one implicit transaction: if the `migrate` initContainer is
killed mid-flight (OOM, node eviction, `kubectl delete pod`, a deploy
timeout) while that block is running, none of it persists, and a retry
starts clean. Its `CREATE INDEX CONCURRENTLY` statements run afterward,
each outside that transaction (required for `CONCURRENTLY`); every one uses
`IF NOT EXISTS`, so re-running the migration after an interruption there is
also safe — whatever indexes already landed are simply skipped.

Future migrations that add tables/indexes outside this pattern should keep
using `CREATE TABLE IF NOT EXISTS`/`CREATE INDEX CONCURRENTLY IF NOT
EXISTS` for the same reason, since goose only marks a migration as applied
once it returns cleanly, and an interrupted run gets retried from the top.

## Object storage (image uploads)

Team/user photos and team logos are stored in an S3-compatible object store
(`internal/storage`), not in Postgres — the DB only holds an object key
(`users.photo_object_key`, `teams.photo_object_key`, `teams.logo_object_key`).
GET endpoints (`/auth/me/photo`, `/teams/{teamId}/photo`,
`/teams/{teamId}/logo`, `/teams/{teamId}/members/{membershipId}/photo`) verify
team membership, then respond `302` with a `Location` header pointing at a
short-lived (15 minute) presigned URL — the application server never streams
image bytes itself.

**Configuration** (see CLAUDE.md's env var table for the full reference):
`S3_ENDPOINT`, `S3_BUCKET`, `S3_ACCESS_KEY_ID`, `S3_SECRET_ACCESS_KEY` are
**required** when `COOKIE_SECURE=true` (production) — startup fails loudly
(`os.Exit(1)`) without them, mirroring the JWT/cookie-key hard-gating.
`S3_REGION` and `S3_USE_PATH_STYLE` default to AWS-friendly values; set
`S3_USE_PATH_STYLE=true` for most self-hosted S3-compatible stores (MinIO).
Set `S3_PUBLIC_BASE_URL` when the backend's S3 endpoint (e.g. an in-cluster
MinIO service DNS name) differs from the endpoint a browser can actually
reach — it rewrites the scheme+host of presigned URLs after signing (the
signature doesn't cover the host, so this is safe).

**Local dev**: `docker compose up` runs a MinIO container plus a one-shot
`minio-init` job that creates the configured bucket; the backend's
`S3_PUBLIC_BASE_URL` is set to `http://localhost:9000` since the backend
container reaches MinIO via the in-network `minio:9000` hostname but a
browser on the host cannot. If `S3_ENDPOINT` is left unset entirely (e.g.
running `go run ./cmd/server` directly, bypassing Compose) the backend falls
back to an in-memory fake object store with a startup warning — fine for a
quick manual smoke test, but uploaded images vanish on restart and aren't
shared across replicas, so never rely on it beyond that.

**Kubernetes**: set the plaintext `s3.endpoint`/`s3.region`/`s3.bucket`/
`s3.usePathStyle`/`s3.publicBaseUrl` values in your overlay, and populate
`s3.secret.existingSecret` with a Secret keyed per `s3.secret.keys`
(defaults `access-key-id`/`secret-access-key`; override to match your
Secret's actual key names). The chart's NetworkPolicy
(`networkPolicy.egress.s3`, `templates/networkpolicy.yaml`) already opens
egress to the S3 endpoint whenever `s3.endpoint` is set — override
`networkPolicy.egress.s3.port`/`.to` to match a self-hosted endpoint's
actual port/destination (AWS S3 needs no override; it's covered by the
chart's general HTTPS egress rule too, but the dedicated S3 rule exists for
self-hosted endpoints on non-443 ports).

## Outbound email (SMTP)

Self-registration verification email (`internal/mailer`) is sent via SMTP
(`internal/mailer/smtp.go`, stdlib `net/smtp` with explicit STARTTLS).

**Configuration** (see CLAUDE.md's env var table for the full reference):
`SMTP_HOST` and `SMTP_FROM_ADDRESS` are **required** when `COOKIE_SECURE=true`
(production) — `config.Load()` fails startup loudly (`os.Exit(1)`) without
them, and since `config.Load()` runs before the `--migrate-only` branch in
`main.go`, this gates the migrate initContainer too, not just the main
container. `SMTP_PORT` defaults to `587` (STARTTLS). `SMTP_USERNAME`/
`SMTP_PASSWORD` may both be blank for an open relay.

**Local dev**: leaving `SMTP_HOST` unset (the chart's default) falls back to
a logging fake mailer (`internal/mailer/fake.go`) — the verification link is
only written to the server log, fine for manual testing but obviously not a
real delivery path.

**Kubernetes**: set the plaintext `smtp.host`/`smtp.port`/`smtp.fromAddress`
values in your overlay, and optionally populate `smtp.secret.existingSecret`
with a Secret keyed per `smtp.secret.keys` (defaults `username`/`password`,
both may be blank for an open relay). The chart's NetworkPolicy
(`networkPolicy.egress.smtp`, `templates/networkpolicy.yaml`) already opens
egress to the SMTP relay whenever `smtp.host` is set — override
`networkPolicy.egress.smtp.port`/`.to` to match your relay's actual
port/destination (SMTP relays commonly listen on 587/465/25, not 443, so
this is a separate rule from the chart's general HTTPS egress rule).
`templates/NOTES.txt` warns at deploy time if `session.cookie.secure` is
`true` and `smtp.host`/`smtp.fromAddress` aren't both set.

## Web Push (VAPID keys)

Push notifications (`internal/push`) are delivered directly from the backend
to each browser's push service (Mozilla, FCM, etc.) using VAPID
(`internal/push/webpush.go`, `github.com/SherClockHolmes/webpush-go`) — no
additional push-relay service is involved.

**Configuration** (see CLAUDE.md's env var table for the full reference):
`VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, and `VAPID_SUBJECT` are **required**
when `COOKIE_SECURE=true` (production) — `config.Load()` fails startup
loudly (`os.Exit(1)`) without them, and since `config.Load()` runs before
the `--migrate-only` branch in `main.go`, this gates the migrate
initContainer too, not just the main container. `VAPID_SUBJECT` must be a
`mailto:` (or `https:`) contact URI identifying the sender to push services,
per the VAPID spec. `VAPID_PUBLIC_KEY` is not secret — it also has to reach
the frontend as `VITE_VAPID_PUBLIC_KEY`/`VAPID_PUBLIC_KEY`, see "Frontend
image: pointing it at a backend" below.

**Local dev**: leaving `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY` unset falls
back to a logging fake pusher (`internal/push/fake.go`) — push payloads are
only written to the server log, fine for manual testing but no real
delivery. Generate a real keypair with `npx web-push generate-vapid-keys`.

**Kubernetes**: set the plaintext `push.publicKey`/`push.subject` values in
your overlay, and populate `push.secret.existingSecret` with a Secret keyed
per `push.secret.keys.privateKey` (default `private-key`). The chart also
composes `frontend.vapidPublicKey` from `push.publicKey` automatically when
the frontend is deployed via this same chart (`frontend.enabled: true`) —
see `helm/team-manager/README.md`. `templates/NOTES.txt` warns at deploy time if
`session.cookie.secure` is `true` and `push.publicKey`/`push.subject` aren't
both set. No dedicated NetworkPolicy egress rule is needed — push services
are reached over HTTPS/443, already covered by the chart's general HTTPS
egress rule.

**Troubleshooting a `status 401`/`status 403` in `jobs.PushDeliveryWorker`
logs**: the error includes a snippet of the push service's own response
body (`internal/push/webpush.go`), which usually names the problem
directly. If that body names a known VAPID key-mismatch signature —
Mozilla autopush's `"VAPID public key mismatch"` (errno 109), or FCM's
`"...do not correspond to the credentials used to create the
subscription"` — the affected `push_subscriptions` row is pruned
automatically, the same as a 404/410; no ops action is needed beyond the
affected user re-enabling push in their browser, since that specific
subscription was created against a VAPID public key this server no longer
signs with (e.g. after a key rotation, or environment data seeded/restored
across a key change) and no amount of retrying fixes that for it. Any
other 401/403 — e.g. a malformed `VAPID_SUBJECT` (not a `mailto:`/`https:`
URI), or a genuinely server-wide key misconfiguration not yet reflected in
any subscription's stored key — is **not** scoped to one subscription, so
it recurs for every delivery until fixed. The most common cause there is a
`VAPID_PUBLIC_KEY` mismatch: the frontend's
`VITE_VAPID_PUBLIC_KEY`/`VAPID_PUBLIC_KEY` must be byte-for-byte identical
to the backend's `VAPID_PUBLIC_KEY` (see "Frontend image: pointing it at a
backend" below), and — unlike `COOKIE_ENCRYPTION_KEYS` or
`JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY` — there is no rotation mechanism for
VAPID keys today: rotating the backend's keypair invalidates every
existing browser subscription (they were created against the old public
key) until each browser unsubscribes and re-subscribes against the new
one — those will show up pruned automatically per the above rather than
recurring in the logs.

**Troubleshooting a `status 413` ("Payload Too Large") in
`jobs.PushDeliveryWorker` logs**: the push service rejected the encrypted
message as too big for a push notification (commonly a ~4 KB limit). The
notification body sent to `push.WebPusher.Send` is truncated to 200
characters before being enqueued (`jobs.pushPayloadForNotification`), so
this should be rare going forward regardless of how long the underlying
content is (e.g. an unbounded poll question). The affected job is
cancelled rather than retried — the payload is fixed once enqueued, so
retrying it changes nothing — and, unlike a 401/403 key mismatch, the
`push_subscriptions` row is left in place: a 413 says nothing about
whether the subscription itself is still valid, only that this one
message was too big. No ops action is needed.

## Cookie encryption key rotation

`COOKIE_ENCRYPTION_KEYS` supports zero-downtime rotation, but — same
one-read-at-process-start caveat as `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY` below —
updating the Secret alone does nothing until pods restart, and a *single*
rolling restart straight to `<new>,<old>` is not actually zero-downtime
(mid-rollout, already-restarted pods encrypt with `<new>` while
not-yet-restarted pods have never loaded it and reject those cookies,
forcing re-login for anyone whose requests land on both pod generations).
The safe sequence needs **two** rolling restarts — see CLAUDE.md's
`COOKIE_ENCRYPTION_KEYS` entry in the backend env var table for the full
step-by-step.

## JWT key rotation

Sessions are signed with `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY` (RS256). Unlike
`COOKIE_ENCRYPTION_KEYS`, there is no built-in dual-key rotation for these —
rotating them invalidates every existing session immediately (all holders
must re-authenticate). To rotate:

1. Generate a new RSA-2048 key pair.
2. Update the keys named by `jwt.secret.keys.privateKey`/`publicKey`
   (defaults `private-key`/`public-key`) in the Secret referenced by
   `jwt.secret.existingSecret` during a maintenance window (accept that
   all active sessions are invalidated).
3. **Restart every backend pod**: `kubectl rollout restart deployment/<fullname>
   -n <namespace>`. This step is not optional and easy to miss — editing a
   Kubernetes Secret does not restart pods that reference it via
   `secretKeyRef` (this chart never creates/manages the Secret itself, only
   references it — there is no chart-side mechanism to trigger a rollout
   automatically), and `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY` are only read
   once, at process start (`loadJWTKeys`). Skipping this step doesn't do
   nothing — it's worse than that: already-running replicas
   keep validating/issuing tokens with the *old* keypair indefinitely, while
   any replica that happens to restart on its own for an unrelated reason
   (HPA scale-out, node reschedule) silently picks up the new key and starts
   rejecting old-key sessions — producing confusing, non-deterministic
   session invalidation split across replicas instead of the clean "everyone
   re-authenticates now" step 2 sets up.
4. Communicate the forced re-login to users ahead of time if possible.

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
`METRICS_TOKEN` is empty in that case. Either populate
`metrics.secret.existingSecret` with a Secret keyed per
`metrics.secret.keys.token` (default `token`), or set
`metrics.allowOpen=true` if `/metrics` is already restricted at the network
layer and you accept it being unauthenticated. The Helm chart's
`templates/NOTES.txt` prints a reminder about this at deploy time, since
referencing an `existingSecret` doesn't let the chart verify the Secret's
actual contents from here. Note this is a separate Secret from
`monitoring.scrapeToken` (the Prometheus *scraper's* own copy of the same
token value, kept separate because Prometheus Operator resolves a
ServiceMonitor's `bearerTokenSecret` in the ServiceMonitor's own namespace —
see the comment on `monitoring.scrapeToken` in `values.yaml`); the two must
be kept in sync by hand.

## Alerting & dashboards

`helm/team-manager/files/prometheus-rules.yaml` defines the alert rules for
this service (availability, error rate/latency, rate-limit spikes, login
failure/bulk-deletion anomalies, DB pool exhaustion, retention job health,
notification job health, backup job health, memory/disk pressure). The
backup CronJob's two rules
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

**`RetentionJobFailing` around deploys:** the daily retention job (runs once
every 24h via a River periodic job) is allowed up to ~150s to complete
(`RetentionWorker.Timeout()`, 4 phases × 30s + margin), but a SIGTERM during
that window — a rolling deploy, node drain, or HPA scale-down landing on the
replica currently running it — cancels the job after only `jobs.SoftStopTimeout`
(8s), not its own full budget; see `cmd/server/main.go`'s graceful-shutdown
sequence. The cancelled phase increments `retention_job_failures_total` and
can trip `RetentionJobFailing`. River automatically retries on the next
scheduled run, so a single occurrence coinciding with a deploy is expected and
self-healing, not a persistent failure — cross-check `RetentionJobStale`
(fires only after 36h with no successful run) before treating this as a real
incident.

## Container images & releases

Tagging a release (`vX.Y.Z`) triggers `.github/workflows/release.yml`, which
builds and pushes versioned backend and frontend images to GHCR. Deploy by
pinning the image to the released tag; roll back by redeploying the previous
tag (images are immutable per digest).

### Frontend image: pointing it at a backend

The frontend image is built once per release and is environment-agnostic —
which backend it talks to (and which Sentry project, if any, it reports
errors to, and which VAPID keypair it uses for Web Push, and whose
legal-notice/privacy-policy data it shows) is resolved at **container
start**, not baked in at build time, so the same image tag can be deployed
to staging and production, for any operator, unchanged. Set the
`API_BASE_URL` and (optionally) `SENTRY_DSN`/`VAPID_PUBLIC_KEY`/`OPERATOR_*`
(see "Legal setup before going public" above) environment variables on the
container:

```
docker run -e API_BASE_URL=https://api.example.com -e SENTRY_DSN=https://key@o0.ingest.sentry.io/1 \
  -e VAPID_PUBLIC_KEY=<same value as the backend's VAPID_PUBLIC_KEY> \
  -e OPERATOR_NAME="Stefan May" -e OPERATOR_STREET="Robensstraße 56" \
  -e OPERATOR_POSTAL_CODE=52070 -e OPERATOR_CITY=Aachen \
  -e OPERATOR_EMAIL=info@example.com \
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

`SENTRY_DSN` and `VAPID_PUBLIC_KEY` have no build-time equivalent that
reaches the release image — the release workflow only ever passes
`VITE_BUILD_VERSION`/`VITE_BUILD_COMMIT` as build args — so these runtime
env vars are the *only* way to enable Sentry error tracking, or Web Push,
in a released frontend image. Leaving `SENTRY_DSN` unset disables Sentry,
matching today's default; leaving `VAPID_PUBLIC_KEY` unset hides the push
opt-in toggle entirely (there's nothing to subscribe against). Set it to
the *same* value as the backend's `VAPID_PUBLIC_KEY` — a mismatch fails
`PushManager.subscribe()` client-side with a benign-looking error, not a
clear "wrong key" message.

**Kubernetes**: `helm/team-manager` can deploy the frontend alongside the
backend — set `frontend.enabled=true`, `frontend.image.tag`, and
`frontend.apiBaseUrl` (the backend's public URL, i.e. wherever this
chart's own `ingress` — or your own separately-managed backend — is
reachable), plus `frontend.sentryDsn`/`frontend.vapidPublicKey`/
`frontend.operator.*` as needed (all plain values, no Secret — see
`helm/team-manager/README.md`'s "Frontend" section and values table). It's
off by default and renders its own independent `frontend.ingress` — the
backend's OpenAPI routes have no shared path prefix, so the two Ingresses
are normally on separate hostnames (e.g. `team-manager.example.com` for
the frontend, `api.team-manager.example.com` for the backend). If you'd
rather deploy the frontend container by some other means (a static host
that proxies to the image, etc.), that still works the same way, setting
`API_BASE_URL` as above.

### Helm chart

The same tag also packages `helm/team-manager` and pushes it as a versioned
OCI artifact to GHCR (`oci://ghcr.io/<org>/charts/team-manager`), with
`Chart.yaml`'s `version`/`appVersion` set to the release version at package
time and the pushed digest signed with keyless cosign, same as the container
images above. Install or upgrade directly from the registry instead of a
local checkout:

```
helm upgrade --install team-manager oci://ghcr.io/<org>/charts/team-manager \
  --version X.Y.Z -f values-prod.yaml
```

Roll back by re-running the same command with the previous version — chart
versions are immutable per digest, same as the images.
