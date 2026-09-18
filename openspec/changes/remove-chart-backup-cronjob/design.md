## Context

The chart deploys the application and points it at an external PostgreSQL
(`database.host`); it has never deployed the database itself. The backup
CronJob was the one piece of the chart that took responsibility for
something outside its own deployment.

## Goals

- Stop the chart from owning a database-operations concern it cannot do
  well for a database it does not run.
- Remove the two CI gates that exist only to scan third-party images,
  which cannot be kept green without an open-ended suppression list.

## Non-Goals

- Changing how the application connects to its database.
- Removing the general backup guidance from `docs/operations.md`. The
  advice is sound and worth keeping; only the parts describing the chart's
  own CronJob go.

## Decisions

- **Remove the feature rather than only the CI scans.** Deleting the scans
  while keeping the CronJob would leave an unscanned container that holds
  `DATABASE_URL` and, when S3 upload is enabled, bucket write credentials —
  strictly worse than today. The scans exist because the CronJob does.
- **Remove rather than suppress.** The alternative was ~15 more entries in
  `trivyignore-postgres-backup.txt`. Every one would have been defensible
  in isolation (the container runs only `pg_dump`, so perl, sqlite and
  pcre2 are unreachable, and CVE-2026-14456 is an OpenSSL QUIC *server*
  DoS while `pg_dump` is a TLS client) — but the list would keep growing
  for a feature whose value was already marginal.
- **`values.schema.json` will now reject `backup.*`.** That is the point:
  a silently-ignored key is worse than a failed upgrade that says exactly
  what to remove. Called out as breaking in the proposal.
- **The NetworkPolicy S3-egress rule stays.** It was conditional on
  `backup.s3` *or* `s3.endpoint`; the app's own image storage still needs
  it, so only the backup half of the condition goes.
- **The PodDisruptionBudget's `component NotIn [backup]` exclusion goes.**
  It existed solely so backup pods could not inflate `currentHealthy`. With
  no backup pods, it excludes nothing; leaving it would be a selector whose
  reason no longer exists.

## Risks

- **Operators running `backup.enabled: true` lose their backup on upgrade**,
  and the upgrade fails on the now-unknown values keys rather than silently
  dropping the CronJob. The loud failure is deliberate — a silent one would
  leave them believing backups still run. Documented in the proposal's
  Impact section and in `docs/operations.md`.
- **`docs/operations.md`'s restore runbook now describes restoring a dump
  the chart never produced.** That is correct: the runbook was always about
  restoring a logical dump, whatever produced it.
