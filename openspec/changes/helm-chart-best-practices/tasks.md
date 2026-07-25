## 1. Configurable secret key names (superseded — see 1a)

- [x] 1.1-1.7 Implemented an intermediate `existingSecretKey`/
      `existingSecretKeys` design (kept alongside `create: true`), then
      **replaced by section 1a below** per review feedback before this
      change was merged. Left unstruck in the history for an accurate
      record of what actually happened; the checked-in chart matches 1a,
      not this section.

## 1a. Revision: no chart-managed Secret + kebab-case `keys` map + `database` split (supersedes 1 & 2)

- [x] 1a.1 `values.yaml`: remove `<area>.secret.create` and every
      plaintext-value field it gated, on all ten areas
      (`database`/`jwt`/`cookieEncryption`/`s3`/`smtp`/`push`/`pagination`/
      `observability.sentry`/`metrics`/`monitoring.scrapeToken`) plus
      `backup.s3.credentialsSecretName` -> `backup.s3.secret`. Replace
      `existingSecretKey`/`existingSecretKeys` with a uniform
      `secret.keys: {<field>: "<kebab-case-key>"}` map on every area,
      defaults lowercase/dash-separated (`password`, `access-key-id`,
      `private-key`, `hmac-key`, `dsn`, `token`, `key`/`keys`,
      `username`/`password`, `private-key`/`public-key`) -- not the
      backend's env var names.
- [x] 1a.2 `values.yaml`: split `database` into plain `host`/`port`/`name`/
      `username`/`sslmode` fields; `database.secret.keys.password` is the
      only Secret-backed field. Document the URL-encoding limitation
      inline (see design.md).
- [x] 1a.3 `templates/_helpers.tpl`: delete `team-manager.secretName` and
      `team-manager.secretsChecksum` (both moot with no chart-managed
      Secret left). Add `team-manager.databaseEnv`, composing
      `DATABASE_URL` from `DB_HOST`/`DB_PORT`/`DB_NAME`/`DB_USERNAME`
      (plain) + `DB_PASSWORD` (`secretKeyRef`) via Kubernetes' native
      `$(VAR_NAME)` env-var expansion -- confirmed no shell exists in the
      backend's distroless runtime image before choosing this over a
      shell-wrapper approach.
- [x] 1a.4 `templates/_env.tpl`: every `secretKeyRef.key` now reads
      `$root.Values.<area>.secret.keys.<field>` directly (no more
      create-or-reference resolution); database env vars come from
      `{{ include "team-manager.databaseEnv" $root }}`.
- [x] 1a.5 `templates/backup-cronjob.yaml`: pg-dump's `DATABASE_URL` env
      vars come from the same `team-manager.databaseEnv` include; AWS
      credentials read `backup.s3.secret.existingSecret`/`secret.keys`.
- [x] 1a.6 `templates/servicemonitor.yaml`: `monitoring.scrapeToken`
      restructured to the same `secret.{existingSecret,keys}` shape
      (was `create`/`token`/`existingSecretName`/`existingSecretKey`).
- [x] 1a.7 Delete every now-dead `templates/<area>-secret.yaml` (database,
      jwt, cookie-encryption, s3, smtp, push, pagination, sentry, metrics)
      and `templates/monitoring-scrape-token-secret.yaml`.
- [x] 1a.8 `templates/deployment.yaml`: remove the `checksum/secrets` pod
      annotation (moot -- nothing left to checksum).
- [x] 1a.9 `values.schema.json`: rewrite every area's `secret` schema to
      `{existingSecret: string, keys: {additionalProperties:false, ...}}`;
      add `database.host`/`port`/`name`/`username`/`sslmode`.
- [x] 1a.10 `templates/NOTES.txt`: rewrite every `.secret.create=true`
      mention; add a `database.*` completeness warning (5 separate fields
      now, easier to partially misconfigure than one opaque
      `DATABASE_URL`).
- [x] 1a.11 `docs/operations.md`: rewrite every `.secret.create=true`/
      `existingSecretKey(s)` mention across Object storage/SMTP/VAPID/JWT
      rotation/metrics/DR-restore sections.
- [x] 1a.12 `README.md`: rewrite the "Secrets" section and every affected
      values-table row.
- [x] 1a.13 `values-staging.yaml`/`values-prod.yaml`: add
      `database.host`/`name`/`username`(/`sslmode` in prod); update every
      area's comment referencing the old field names.
- [x] 1a.14 `.github/workflows/ci.yml`: rewrite the `helm-lint` job's
      `--set` matrix -- every `.secret.create=true` combination becomes
      `.secret.existingSecret=<dummy>` (+ non-default `.secret.keys.*`
      overrides proving the override is actually read), `database.host`/
      `name`/`username` added, `backup.s3.credentialsSecretName` ->
      `backup.s3.secret.existingSecret`, `monitoring.scrapeToken.create`/
      `.existingSecretName` -> `monitoring.scrapeToken.secret.existingSecret`.

## 2. `checksum/secrets` annotation (removed -- see 1a.8)

- [x] 2.1-2.2 Implemented, then removed per 1a.8 once `create: true` (the
      only thing it was tracking) was removed. Left unstruck for an
      accurate record.

## 3. Escape hatches

- [x] 3.1 `values.yaml`: add `extraEnv: []`, `extraVolumes: []`,
      `extraVolumeMounts: []`, `podLabels: {}`.
- [x] 3.2 `templates/_env.tpl` (or `deployment.yaml` directly after each
      `team-manager.env` include): append `{{- with .Values.extraEnv
      }}{{ toYaml . | nindent 12 }}{{- end }}` in both the migrate
      initContainer's and the main container's `env:` blocks.
- [x] 3.3 `templates/deployment.yaml`: main container gains a
      `volumeMounts:` key (`{{- with .Values.extraVolumeMounts }}` only,
      absent when empty); the pod spec's `volumes: []` gains
      `{{- with .Values.extraVolumes }}{{ toYaml . | nindent 8 }}{{- end
      }}` appended after the explicit `[]`.
- [x] 3.4 `templates/deployment.yaml`: pod template `labels:` gains
      `{{- with .Values.podLabels }}{{ toYaml . | nindent 8 }}{{- end }}`
      after `team-manager.labels`, mirroring `podAnnotations`.
- [x] 3.5 `values.schema.json`: add `extraEnv`/`extraVolumes`/
      `extraVolumeMounts` (loosely typed arrays, matching this schema's
      existing treatment of raw K8s spec fragments) and `podLabels`
      (`additionalProperties: {type: string}`, matching `podAnnotations`).

## 4. Chart packaging & metadata

- [x] 4.1 New `helm/team-manager/README.md`: description, prerequisites,
      install/upgrade instructions, full values table (name/type/default/
      description) covering every `values.yaml` key.
- [x] 4.2 New `helm/team-manager/LICENSE`: copy of the repo-root
      `LICENSE` (Apache 2.0) — `helm package` only bundles the chart
      directory, so the packaged `.tgz` this repo's `release.yml` already
      OCI-pushes currently ships with no license text.
- [x] 4.3 New `helm/team-manager/.helmignore` (standard `helm create`
      defaults: `.git/`, OS/editor cruft).
- [x] 4.4 `Chart.yaml`: add `sources: ["https://github.com/yoadey/team-manager"]`.

## 5. Production-readiness values

- [x] 5.1 `values.yaml`: add `priorityClassName: ""`.
- [x] 5.2 `templates/deployment.yaml` and `templates/backup-cronjob.yaml`:
      pod spec gains `{{- with .Values.priorityClassName
      }}priorityClassName: {{ . }}{{- end }}`.
- [x] 5.3 `values.yaml`: add `topologySpreadConstraints: []`.
- [x] 5.4 `templates/deployment.yaml`: pod spec gains
      `{{- with .Values.topologySpreadConstraints }}topologySpreadConstraints:
      {{ toYaml . | nindent 8 }}{{- end }}` alongside the existing
      `affinity` block.
- [x] 5.5 `values.yaml`: add `image.digest: ""`.
- [x] 5.6 `templates/deployment.yaml`: both the migrate initContainer's and
      the main container's `image:` line become
      `"{{ .Values.image.repository }}{{ if .Values.image.digest }}@{{
      .Values.image.digest }}{{ else }}:{{ .Values.image.tag | default
      .Chart.AppVersion }}{{ end }}"`.
- [x] 5.7 `values.schema.json`: add `priorityClassName` (string),
      `topologySpreadConstraints` (loosely-typed array), `image.digest`
      (string) to the appropriate schema blocks.

## 6. `app.kubernetes.io/component: backend` label

- [x] 6.1 `templates/_helpers.tpl`: `team-manager.labels` gains
      `app.kubernetes.io/component: backend`. `team-manager.selectorLabels`
      unchanged (stays `name`+`instance` only).

## 7. `helm test` hook

- [x] 7.1 New `templates/tests/test-connection.yaml`: a Pod annotated
      `helm.sh/hook: test`, `helm.sh/hook-weight: "0"`,
      `helm.sh/hook-delete-policy: before-hook-creation,hook-succeeded`,
      running a minimal image (`curlimages/curl`, digest-pinned) that curls
      `http://{{ include "team-manager.fullname" . }}:{{
      .Values.service.port }}/healthz` and exits non-zero on failure —
      matching this chart's existing non-root/readOnlyRootFilesystem/
      capabilities-drop-ALL security posture.

## 8. `values-staging.yaml` / `values-prod.yaml`

- [x] 8.1 No functional changes needed (every new field defaults to
      today's behavior). Add a one-line comment near each overlay's
      `existingSecret` block pointing at the new `existingSecretKey[s]`
      override, so the capability is discoverable where it's most likely
      to be used.

## 9. CI

- [x] 9.1 `.github/workflows/ci.yml`'s `helm-lint` job: all three
      `helm lint` invocations gain `--strict`.
- [x] 9.2 `.github/workflows/release.yml`'s `helm-chart` job: its
      `helm lint` invocation gains `--strict`.
- [x] 9.3 `.github/workflows/ci.yml`'s `helm-lint` job: pipe every existing
      `helm template` invocation in that job through
      `kubeconform -strict -summary -ignore-missing-schemas -kubernetes-version
      1.23.0` (matching `Chart.yaml`'s `kubeVersion` floor), via a
      digest-pinned `ghcr.io/yannh/kubeconform` image run with
      `docker run --rm -i` (consistent with this repo's existing
      digest-pinned-third-party-image convention, and avoiding a separate
      checksum-verified binary download step). Also added `set -o pipefail`
      to that step (a `helm template | kubeconform` pipe otherwise masks a
      failing `helm template` behind kubeconform's own exit code — the same
      bug class `backup-cronjob.yaml`'s pg_dump comment already documents),
      and discovered/fixed a pre-existing latent bug while testing this
      locally: none of this job's `helm template` invocations passed
      `--kube-version`, so every one of them was already failing against
      Helm's hardcoded offline-fallback `v1.20.0` vs. `Chart.yaml`'s
      `kubeVersion: >=1.23.0-0` — fixed by adding `--kube-version 1.23.0` to
      the shared `render()` helper.
- [x] 9.4 `.github/workflows/ci.yml`'s `helm-lint` job: add `--set`
      coverage (following the job's existing per-branch-comment style) for:
      `extraEnv`/`extraVolumes`/`extraVolumeMounts`/`podLabels` all
      non-empty; `priorityClassName` set; `topologySpreadConstraints`
      non-empty; `image.digest` set; `database.secret.existingSecretKey`/
      `jwt.secret.existingSecretKeys.privateKey` overridden to non-default
      values (proving the create-side Secret and the env-var secretKeyRef
      agree on the overridden name, not just the default).

## 10. Verification

- [x] 10.1 `helm lint --strict helm/team-manager` (and against both
      `values-staging.yaml`/`values-prod.yaml`) passes.
- [x] 10.2 `helm template team-manager helm/team-manager` (default values,
      both overlays, and every new `--set` combination from task 9.4)
      renders without error.
- [x] 10.3 A rendered manifest with a non-default `existingSecretKey`
      override is inspected to confirm both the `create: true` Secret's
      `data` key and the corresponding `secretKeyRef.key` use the
      overridden name, not the old literal.
- [x] 10.4 A rendered Deployment's pod template is inspected to confirm
      `checksum/secrets` changes when a `create: true` secret's plaintext
      value changes between two `helm template` invocations.
- [x] 10.5 CI green: `helm-lint` job (including the new `kubeconform` step)
      and `backend-openapi-drift`/other unaffected jobs unaffected.
