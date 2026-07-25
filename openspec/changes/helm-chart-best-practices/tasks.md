## 1. Configurable secret key names

- [x] 1.1 `values.yaml`: add `database.secret.existingSecretKey: "DATABASE_URL"`,
      `push.secret.existingSecretKey: "VAPID_PRIVATE_KEY"`,
      `pagination.secret.existingSecretKey: "PAGINATION_HMAC_KEY"`,
      `observability.sentry.secret.existingSecretKey: "SENTRY_DSN"`,
      `metrics.secret.existingSecretKey: "METRICS_TOKEN"` (single-key areas).
- [x] 1.2 `values.yaml`: add `jwt.secret.existingSecretKeys: {privateKey:
      "JWT_PRIVATE_KEY", publicKey: "JWT_PUBLIC_KEY"}`,
      `cookieEncryption.secret.existingSecretKeys: {key:
      "COOKIE_ENCRYPTION_KEY", keys: "COOKIE_ENCRYPTION_KEYS"}`,
      `s3.secret.existingSecretKeys: {accessKeyId: "S3_ACCESS_KEY_ID",
      secretAccessKey: "S3_SECRET_ACCESS_KEY"}`,
      `smtp.secret.existingSecretKeys: {username: "SMTP_USERNAME", password:
      "SMTP_PASSWORD"}` (multi-key areas).
- [x] 1.3 `templates/_env.tpl`: every `secretKeyRef.key` for the ten areas
      above switches from a hardcoded literal to the corresponding
      `.Values.<area>.secret.existingSecretKey[s...]` value.
- [x] 1.4 Every `templates/<area>-secret.yaml` (database, jwt,
      cookie-encryption, s3, smtp, pagination, sentry, metrics): `data:` key
      name(s) switch from hardcoded literals to the same values-driven key
      name(s) used in 1.3, so `create: true` and `existingSecret` always
      agree on the key name.
- [x] 1.5 `templates/backup-cronjob.yaml`: `DATABASE_URL` secretKeyRef's
      `key:` switches to `.Values.database.secret.existingSecretKey`.
- [x] 1.6 `values.schema.json`: add the new `existingSecretKey`/
      `existingSecretKeys` properties to each of the ten areas'
      `secret` object schemas (strings, `additionalProperties: false` on
      each `existingSecretKeys` map with its area's fixed field names).
- [x] 1.7 `docs/operations.md`: update each area's "existingSecret Secret
      key(s): ..." mention to note the key name(s) are overridable via
      `<area>.secret.existingSecretKey[s]`.

## 2. `checksum/secrets` annotation

- [x] 2.1 `templates/_helpers.tpl`: new `team-manager.secretsChecksum`
      named template concatenating
      `include (print $.Template.BasePath "/<file>-secret.yaml") .` for
      every one of the eight `<area>-secret.yaml` templates, `sha256sum`d.
- [x] 2.2 `templates/deployment.yaml`: pod template annotations gain
      `checksum/secrets: {{ include "team-manager.secretsChecksum" . }}`.

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
