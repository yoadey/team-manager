## Why

The chart ships a daily `pg_dump` CronJob (`templates/backup-cronjob.yaml`,
~80 values keys, a dedicated ServiceAccount, two Prometheus alerts, an S3
upload path and a NetworkPolicy egress rule) for a database the chart does
not deploy. `database.host` points at a PostgreSQL the operator runs
elsewhere, so backing it up belongs with whatever operates it — a managed
offering does it better than a CronJob can (continuous archiving,
point-in-time recovery, provider-enforced retention), and a self-hosted
Postgres has its operator's own tooling.

Carrying it has a running cost the project keeps paying. Two CI jobs exist
solely to scan `postgres:17` and `amazon/aws-cli` — images the project
references but does not build — and they fail whenever Debian discloses
anything in either. They are failing now, on `main`, with no fix available:
7 Go stdlib CVEs in the `gosu` binary baked into `postgres:17` (no rebuilt
upstream tag exists), plus perl (3× CRITICAL), sqlite, pcre2 and openssl
findings whose fixes Debian has published but the image has not picked up.
Keeping the gate green means an ever-growing allowlist of suppressions for
third-party images, re-litigated on every disclosure.

Removing the feature removes the reason for both scans, and is honest about
what the chart is: a working example deployment of *this application*, not a
production database operations story.

## What Changes

- Delete `templates/backup-cronjob.yaml` and `templates/backup-serviceaccount.yaml`,
  the `backup:` values block, its `values.schema.json` section, its
  `values-prod.yaml` overrides and the `team-manager.backupServiceAccountName`
  helper.
- Drop the two `BackupCronJob*` Prometheus alerts, the NetworkPolicy S3-egress
  condition on `backup.*` (the rule stays, now driven by `s3.endpoint` alone),
  the PodDisruptionBudget's backup-pod exclusion, and the NOTES.txt retention
  reminder.
- Delete the `security-container-backup` CI job, its mirror in
  `scheduled-security-scan.yml`, and the two `trivyignore-*-backup.txt`
  files they consumed. Narrow the pinned-digest drift check to what remains
  (`postgres:17` between `docker-compose.yml` and CI's service containers;
  `node:22-alpine` between `docker-compose.yml` and `frontend/Dockerfile`).
- Replace `docs/operations.md`'s chart-CronJob description with guidance that
  stands on its own: verify dumps rather than exit codes, restore on a
  schedule, and enforce retention at the bucket — plus an explicit statement
  that the chart ships no backup job and why.

## Capabilities

### Modified Capabilities
- `helm-deployment`: the chart no longer ships a database backup CronJob;
  backing up the external database is the operator's responsibility.

## Impact

- Deleted: `helm/team-manager/templates/backup-cronjob.yaml`,
  `templates/backup-serviceaccount.yaml`,
  `.github/trivyignore-postgres-backup.txt`,
  `.github/trivyignore-awscli-backup.txt`.
- Edited: `helm/team-manager/values.yaml`, `values-prod.yaml`,
  `values.schema.json`, `README.md`, `Chart.yaml`,
  `templates/{_helpers.tpl,networkpolicy.yaml,pdb.yaml,NOTES.txt,tests/test-connection.yaml}`,
  `files/prometheus-rules.yaml`, `.github/workflows/{ci.yml,scheduled-security-scan.yml}`,
  `docs/operations.md`.
- **Breaking for anyone running `backup.enabled: true`**: the CronJob is gone
  and `backup.*` keys are now rejected by `values.schema.json`. Upgrading
  requires removing those values and arranging backups by other means.
  Nothing else in the chart changes; no application behavior is affected.
