## 1. Chart

- [x] 1.1 Delete `templates/backup-cronjob.yaml` and
      `templates/backup-serviceaccount.yaml`
- [x] 1.2 Remove the `backup:` block from `values.yaml` and its overrides
      from `values-prod.yaml`
- [x] 1.3 Remove the `backup` section from `values.schema.json` — every
      remaining top-level key in both values files was re-checked against
      the schema
- [x] 1.4 Remove `team-manager.backupServiceAccountName` from `_helpers.tpl`
- [x] 1.5 NetworkPolicy's S3 egress rule now keys off `s3.endpoint` alone
      (the app's own image storage still needs it)
- [x] 1.6 Remove the PodDisruptionBudget's `component NotIn [backup]`
      exclusion — with no backup pods it excludes nothing
- [x] 1.7 Remove NOTES.txt's backup retention reminder
- [x] 1.8 Remove the `BackupCronJobFailed`/`BackupCronJobStale` alerts from
      `files/prometheus-rules.yaml`
- [x] 1.9 Remove the `backup.*` rows from the chart README's values table
      and `backup.s3` from its per-area Secret list
- [x] 1.10 Update the comments in `Chart.yaml`, `_helpers.tpl`,
      `values.yaml` and `tests/test-connection.yaml` that referred to the
      CronJob

## 2. CI

- [x] 2.1 Delete the `security-container-backup` job from `ci.yml` and
      `container-backup` from `scheduled-security-scan.yml` (including its
      `needs:` entry on the failure-notifier)
- [x] 2.2 Delete `.github/trivyignore-postgres-backup.txt` and
      `.github/trivyignore-awscli-backup.txt`
- [x] 2.3 Narrow the pinned-digest drift check: `postgres:17` between
      `docker-compose.yml` and CI's service containers, `node:22-alpine`
      between `docker-compose.yml` and `frontend/Dockerfile`; the aws-cli
      group is gone entirely
- [x] 2.4 Replace the helm-lint job's three `--set backup.*` renders with
      one covering `s3.endpoint` (networkpolicy.yaml's S3 egress branch,
      otherwise uncovered)

## 3. Docs

- [x] 3.1 `docs/operations.md`: replace the CronJob description with
      standalone guidance (verify dumps, restore on a schedule, enforce
      retention at the bucket) plus an explicit note that the chart ships
      no backup job and why
- [x] 3.2 `docs/operations.md`: drop the backup alerts from the alerting
      section

## 4. Verification

- [x] 4.1 No `.Values.backup` reference remains in any template; no
      `backup` key in either values file; no `backup` entry in the schema
- [x] 4.2 Both workflows parse as YAML and contain no backup job
- [x] 4.3 `files/prometheus-rules.yaml` parses; no `Backup*` alert remains
- [x] 4.4 Every template's `{{ if/range/with/define }}`/`{{ end }}` pairs
      balance after the edits
- [x] 4.5 The narrowed drift check re-run by hand — postgres, node and the
      Go version pins all in sync
- [x] 4.6 `helm lint --strict` + `helm template` + kubeconform, and the
      chart's own `helm template rejects unknown values keys` step —
      Helm's download host is blocked by the environment this was prepared
      in, so these could not run locally. Confirmed instead by CI's
      `Helm · Lint & template` job, green on the PR's final head
      (f847993) and on every head after the removal landed
