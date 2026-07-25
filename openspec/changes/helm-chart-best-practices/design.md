## Context

`helm-chart-structured-config` (archived 2026-07-24) restructured the chart
from a flat `env:` map / single "God Secret" into typed `values.yaml`
sections with one `create`-or-`existingSecret` Secret per functional area.
That change's own design.md explicitly *rejected* configurable secret key
names ("no known use case needs it, and it would only add schema surface
without adding safety"). This change revisits that call: it's now an
explicit requirement, and an independent best-practices research pass
(official Helm `chart_best_practices` guide, Artifact Hub conventions,
Bitnami/community chart conventions) confirms it's standard practice for
exactly the scenario that motivates it here — an operator whose
`existingSecret` is populated by something they don't control the key
naming of (`ExternalSecrets` synced from Vault/AWS Secrets Manager/etc.).

That same research pass also surfaced a handful of other concrete,
non-cosmetic gaps against current guidance — covered in `proposal.md`'s
"What Changes". This document covers the design decisions for all of them,
since they touch the same files.

## Goals / Non-Goals

**Goals:**
- Every `<area>.secret` block's key name(s) inside its `existingSecret` are
  overridable, defaulting to today's literal names (zero behavior change
  for every values file that doesn't set the new field).
- The same configured key name is used whether the Secret is externally
  managed (`existingSecret`) or chart-managed (`create: true`) — one source
  of truth per field, not two independently-driftable ones.
- A chart-managed secret's plaintext value change on `helm upgrade`
  actually reaches running pods (rollout triggered), matching the
  standard `checksum/*` annotation pattern.
- Standard forward-compatibility escape hatches
  (`extraEnv`/`extraVolumes`/`extraVolumeMounts`/`podLabels`) exist so a
  one-off need doesn't require forking the chart.
- The packaged `.tgz` this repo already builds and OCI-pushes
  (`release.yml`) contains a `README.md` and `LICENSE`, like any
  reasonably-distributed chart.
- Commonly-flagged production-readiness knobs
  (`priorityClassName`, `topologySpreadConstraints`, main-image `digest`
  pinning) are present, matching what this chart already does for its
  backup-job images.
- CI catches what it currently doesn't: WARNING-level lint findings
  (`--strict`), and Kubernetes-schema-invalid rendered output
  (`kubeconform`) — both cheap, both catch a real class of bug the existing
  extensive `helm template --set ...` matrix doesn't (that matrix proves
  *conditional branches render*, not that what they render is
  *structurally valid Kubernetes*).

**Non-Goals:**
- `chart-testing` (`ct lint`/`ct install`) and `helm-unittest`. Both are
  genuinely valuable (per the research pass) but `ct install` needs a live
  cluster (`kind`) in CI — a meaningfully larger CI-infrastructure change —
  and `helm-unittest` needs a test suite written from scratch covering the
  chart's ~20 conditional templates. Both are sized for their own follow-up
  change, not a "best practices hygiene" pass.
- Changing `NetworkPolicy`'s default posture (port-restricted, source/
  destination-open unless `networkPolicy.ingress.from`/`egress.*.to` is
  set) to default-deny. Already a deliberate trade-off in the existing
  chart, called out in its own `values.yaml` comments, to avoid silently
  breaking any deployment that hasn't set those fields yet. Flipping the
  default is a behavior change with its own migration story, not a
  best-practices-hygiene fix.
- Chart signing/provenance for OCI publishing — already implemented
  (`release.yml`'s keyless `cosign sign`). No gap, no work here.
- `app.kubernetes.io/part-of`, `commonLabels`/`commonAnnotations`,
  `extraObjects` (arbitrary manifest injection), and VPA support. All
  reasonable asks from the wider checklist, but lower-value for a
  single-chart (no subcharts, no umbrella) deployment than the items above;
  left for a future change if a concrete need arises rather than added
  speculatively (this repo's own `CLAUDE.md`: "Don't add features... beyond
  what the task requires").
- Rewriting `values.yaml`'s existing grouping into strict alphabetical
  order. The current grouping (by functional area, matching
  `backend/internal/config/config.go`'s own section boundaries) already
  satisfies the best-practices guidance's "sorted logically *or*
  alphabetically" — re-sorting alphabetically would scatter related keys
  (`s3.endpoint`/`s3.secret`) for no readability gain.

## Decisions

### Secret key-name overrides

- **Single-key areas** (`database`, `push`, `pagination`,
  `observability.sentry`, `metrics`) get one new field,
  `secret.existingSecretKey`, defaulting to the area's current literal key
  name (e.g. `database.secret.existingSecretKey: "DATABASE_URL"`) — same
  shape `monitoring.scrapeToken.existingSecretKey` already uses, so this is
  actually completing an existing-but-inconsistently-applied pattern
  rather than introducing a new one.
- **Multi-key areas** (`jwt`, `cookieEncryption`, `s3`, `smtp`) get
  `secret.existingSecretKeys: { <field>: <default literal key> }` — a map
  keyed by the same field names already used for `create: true`'s
  plaintext values (e.g. `jwt.secret.existingSecretKeys.privateKey`
  defaults to `"JWT_PRIVATE_KEY"`), so the plaintext-field name and its
  corresponding Secret key name are visibly paired in `values.yaml`.
- `_env.tpl`'s `secretKeyRef.key` for every area switches from a hardcoded
  literal (`key: DATABASE_URL`) to
  `{{ .Values.database.secret.existingSecretKey }}` (or the
  `existingSecretKeys.<field>` equivalent) — this is evaluated
  unconditionally (not just under `existingSecret`), so it applies equally
  when `create: true`.
- Each `<area>-secret.yaml` template (the `create: true` renderer) writes
  its `data` key(s) using that same value instead of the hardcoded literal
  — e.g. `templates/database-secret.yaml`'s `data:` becomes
  `{{ .Values.database.secret.existingSecretKey }}: {{ .Values.database.secret.url | b64enc }}`.
  This is the "one source of truth" goal: a values override that changes
  `existingSecretKey` changes both what the chart writes (when it manages
  the Secret) and what it reads (when it doesn't), so the two paths can
  never disagree.
- Defaults for every new field match today's hardcoded literals exactly —
  no values file in this repo (`values.yaml`, `values-staging.yaml`,
  `values-prod.yaml`) needs to change for this alone, and no
  externally-managed Secret an operator already created needs re-keying.
- `monitoring.scrapeToken.existingSecretKey` is left exactly as-is (already
  correct, already defaults to `"token"`).

### `checksum/secrets` annotation

- One combined annotation (`checksum/secrets`), not one per area — a new
  `team-manager.secretsChecksum` helper in `_helpers.tpl` concatenates the
  rendered output of every `templates/*-secret.yaml` file (via
  `include (print $.Template.BasePath "/<file>") .` per area, matching the
  documented Helm pattern for this) and `sha256sum`s the result once.
  Simpler than eight separate annotations, and every one of those templates
  already renders to nothing (empty string) when its area's `create` is
  `false` — so toggling `create` on/off, not just editing a plaintext
  value, also changes the checksum and correctly triggers a rollout.
- Deliberately does **not** attempt to checksum `existingSecret`-referenced
  Secrets' actual *content* — the chart only ever sees that Secret's name,
  never its data (same limitation `templates/NOTES.txt` already documents
  for why it can't verify e.g. `METRICS_TOKEN` is non-empty). Rotating an
  externally-managed Secret's value is already the deployer's own
  responsibility/tooling (e.g. Reloader, or the two-step
  `COOKIE_ENCRYPTION_KEYS` rotation runbook in `CLAUDE.md`) — out of scope
  here, and not a regression since no such mechanism existed before this
  change either.

### Escape hatches

- `extraEnv: []` — raw `EnvVar` objects (`{name, value}` or
  `{name, valueFrom}`), appended via `toYaml` after
  `team-manager.env`'s output in both the migrate initContainer and the
  main container (same list, same order dependency Kubernetes itself
  applies: a later-declared `name` masking an earlier one is standard env
  var precedence, not something this chart needs to special-case).
- `extraVolumes: []` / `extraVolumeMounts: []` — raw `Volume`/
  `VolumeMount` objects, `toYaml`'d into the pod spec's `volumes:` (which
  today is deliberately an explicit `[]` for the IRSA webhook JSON-patch
  reason already documented in `_helpers.tpl`/`deployment.yaml` — appending
  here preserves that array-exists guarantee) and the main container's
  `volumeMounts:` (a key that doesn't exist on the container today; added
  only when non-empty via `{{- with }}`, so the common case renders
  identically to before).
- `podLabels: {}` — merged into the pod template's `labels:` alongside
  `team-manager.labels`, mirroring the existing `podAnnotations` pattern
  exactly (map, `toYaml`'d, additive).
- Not adding `extraContainers` or `extraInitContainers` in this pass —
  no concrete need surfaced (unlike volumes/env, which are the two escape
  hatches every "mount a CA bundle" / "add a debug env var" ask actually
  needs), and each is meaningfully more values-schema surface for a
  chart-defaults reviewer to reason about. Left for a future change if a
  real need shows up.

### Production-readiness values

- `priorityClassName: ""` — wired into both `deployment.yaml`'s pod spec
  and `backup-cronjob.yaml`'s pod spec (only rendered when non-empty, via
  `{{- with }}`).
- `topologySpreadConstraints: []` — wired into `deployment.yaml`'s pod spec
  only (not the backup CronJob, which runs a single Job pod at a time —
  spread constraints are meaningless there), alongside (not replacing) the
  existing `affinity.podAntiAffinity` default; both can be set
  simultaneously since they're independent pod-spec fields.
- `image.digest: ""` — when set, the main app image reference becomes
  `{{ .Values.image.repository }}@{{ .Values.image.digest }}` instead of
  `{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}`,
  mirroring exactly how `backup.postgresImageDigest`/
  `backup.s3.awsCliImage`'s digest suffix already work. Left empty by
  default (unlike the backup images, the main app's tag is the repo's own
  release artifact, resolved per-deploy per `templates/NOTES.txt`'s
  existing warning — a chart-shipped default digest would immediately go
  stale the same way a default `image.tag` would).

### `app.kubernetes.io/component: backend`

- Added to `team-manager.labels` (used by every non-backup resource:
  Deployment, Service, ServiceAccount, every Secret, PDB, HPA,
  NetworkPolicy, ServiceMonitor, PrometheusRule, monitoring ConfigMap/
  Secret) as a static `backend` value — not templated/configurable, since
  this chart only ever manages one component split (main app vs. backup
  job, the latter already labeled `backup` directly in
  `backup-cronjob.yaml`'s pod template rather than via the shared
  `labels` helper).
- Not added to `team-manager.selectorLabels` — selectors are immutable
  once a Deployment exists, and `name`+`instance` is already the complete,
  correct selector; adding a third label there would be a breaking change
  for zero benefit (nothing needs to select on `component` today; `pdb.yaml`
  already excludes `component: backup` via `matchExpressions`, which reads
  from the *pod template's* labels — a broader set than the selector —
  so this addition doesn't require any change there).

### CI

- `helm lint --strict` in both `ci.yml`'s `helm-lint` job and
  `release.yml`'s `helm-chart` job (three `helm lint` invocations across
  both files, per current values files each already lints against).
- `kubeconform` (not `kubeval`, which the research pass flagged as no
  longer actively maintained) installed the same
  download-and-verify-checksum way `ci.yml` already installs Helm itself,
  then run against every `helm template` invocation already present in
  that job — piping each render (`helm template ... | kubeconform -strict
  -summary -kubernetes-version <matches kubeVersion floor>`) rather than
  adding a separate render pass, so this doesn't duplicate the job's
  existing "exercise every conditional branch" `--set` matrix.
- `templates/tests/test-connection.yaml` (a `helm.sh/hook: test` Pod
  curling the Service's `/healthz`) is added as chart content and covered
  by the existing job's client-side `helm template` render (it's an
  ordinary conditional-free template, so no new `--set` branch is needed).
  Actually *executing* `helm test` needs a live release, which needs a
  cluster — consistent with the `ct install`/`helm-unittest` Non-Goal
  above, standing up a `kind` cluster in CI for this one hook is left as
  future work rather than bundled in; an operator can already run
  `helm test` themselves against any real install today once this hook
  ships.

## Risks / Trade-offs

- `existingSecretKey(s)` fields widen `values.schema.json`'s surface
  per area (one or more new string properties each) — mechanical, low-risk,
  but touches every area's schema block, so the schema diff for this change
  is large even though behaviorally most of it is "add an optional string
  field with today's literal as its documented default".
- `checksum/secrets` being one combined annotation (not per-area) means
  changing *any* chart-managed secret's plaintext value restarts *all*
  replicas, even if only one area actually changed — acceptable, since a
  rolling restart is already what any Deployment spec change does chart-
  wide, and per-area checksums would need eight separate annotations for a
  marginal benefit (avoiding restarts a plaintext-secret-value change was
  going to require anyway, just not atomically-per-area).
