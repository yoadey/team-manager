## Why

`helm/team-manager` currently exposes almost all backend configuration
through a single flat `values.yaml` `env:` string map (rendered via
`{{- range $key, $val := .Values.env }}`) and a single flat `existingSecret`
name that must contain *every* sensitive key the backend uses
(`DATABASE_URL`, `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY`, `S3_ACCESS_KEY_ID`/
`S3_SECRET_ACCESS_KEY`, `COOKIE_ENCRYPTION_KEY(S)`, `SENTRY_DSN`,
`METRICS_TOKEN`, `PAGINATION_HMAC_KEY`) — a "God Secret" that mixes
unrelated blast radii (e.g. rotating the S3 access key means touching the
same Secret object as the JWT signing key) and gives every consumer of that
Secret access to everything in it.

Concretely, this surfaced first as: **SMTP is entirely unwired.**
`backend/internal/config/config.go`'s `loadSMTPConfig` hard-requires
`SMTP_HOST`/`SMTP_FROM_ADDRESS` whenever `COOKIE_SECURE=true` (the chart's
default, and every shipped overlay) — `config.Load()` runs before the
`--migrate-only` branch in `main.go`, so this gates the migrate
initContainer too — and `cmd/server/main.go` wires `SMTP_USERNAME`/
`SMTP_PASSWORD` into the real mailer. None of `SMTP_HOST`/`SMTP_PORT`/
`SMTP_FROM_ADDRESS`/`SMTP_USERNAME`/`SMTP_PASSWORD` exist anywhere in the
chart, so every real (`COOKIE_SECURE=true`) deployment crash-loops on
`config.ErrSMTPConfigRequired`, with no supported way to add SMTP
credentials once that's fixed.

But the flat `env`/single-`existingSecret` design is also a broader
usability and safety gap: nothing in the chart validates that `env` keys
are spelled correctly, are the right type, or fall in a sane range — a
typo'd key (`S3_ENPOINT`) or a boolean passed where a string was expected
silently does nothing, discovered only at pod-startup crash-loop time, if
at all. Since the chart has not shipped a release yet, there is no
backward-compatibility constraint stopping a proper fix.

## What Changes

- **Replace the flat `env:` map** with typed, nested `values.yaml` sections
  per functional area: `server` (port, allowed origins, trusted proxy
  CIDRs, log level, rate limiting, error-type URI, deprecation date),
  `session` (TTL, cookie name/secure), `s3`, `smtp`, `selfRegistration`,
  `retention`, `pagination`, `metrics`, `observability` (OTEL + Sentry).
- **Replace the single `existingSecret`** with one secret reference per
  functional area (`database.secret`, `jwt.secret`, `cookieEncryption.secret`,
  `s3.secret`, `smtp.secret`, `pagination.secret`, `observability.sentry.secret`,
  `metrics.secret`) — no more God Secret. Each area's `secret` block supports
  either `existingSecret: <name>` (references a Secret the deployer manages
  externally, keyed by the same names as today) **or** `create: true` with
  plaintext fields, in which case the chart renders and manages that one
  narrowly-scoped Secret itself (useful for local/CI/test deployments via
  `--set`).
- **Add `values.schema.json`**, so `helm install`/`upgrade`/`lint`/`template`
  structurally validate every value (types, enums, patterns,
  `additionalProperties: false` to catch typo'd/unknown keys) before
  anything is rendered or applied.
- Wire `SMTP_HOST`/`SMTP_PORT`/`SMTP_FROM_ADDRESS`/`SMTP_USERNAME`/
  `SMTP_PASSWORD` end to end (both the migrate initContainer and the main
  container) as part of the above restructuring — this is what unblocks the
  original SMTP gap.
- Add a `networkPolicy.egress.smtp` rule (mirroring the existing
  `networkPolicy.egress.s3` rule), gated on `smtp.host` being set.
- Update `templates/NOTES.txt`'s deploy-time warnings and
  `docs/operations.md` for the new value paths, and add an "Outbound email
  (SMTP)" section mirroring the existing "Object storage (image uploads)"
  one.

## Capabilities

### New Capabilities
- `helm-deployment`: the Helm chart exposes every backend configuration
  surface through typed, schema-validated values, with per-area Secret
  references (create-or-reference) instead of one combined Secret, and
  matching NetworkPolicy egress.

### Modified Capabilities
<!-- none -->

## Impact

- All of `helm/team-manager/`: `values.yaml`, new `values.schema.json`,
  `templates/_helpers.tpl`, `templates/deployment.yaml`,
  `templates/backup-cronjob.yaml`, `templates/networkpolicy.yaml`,
  `templates/servicemonitor.yaml`, `templates/NOTES.txt`, new per-area
  `templates/*-secret.yaml`, `values-staging.yaml`, `values-prod.yaml`.
- `docs/operations.md` (existingSecret / value-path references throughout,
  new SMTP section).
- No application code, API, or database schema change; no migration.
  Chart-only. Every `values.yaml` key changes shape — acceptable since the
  chart has never been released (`Chart.yaml` `version: 0.1.0`, no tagged
  release), so there is no deployed values file to migrate.
