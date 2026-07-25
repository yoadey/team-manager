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

### No chart-managed Secret, anywhere (supersedes an earlier iteration)

An earlier iteration of this change kept `<area>.secret.create: true` (the
chart renders and manages that area's Secret from plaintext `values.yaml`
fields) alongside `existingSecret`, plus per-area `existingSecretKey`/
`existingSecretKeys` overrides defaulting to the backend's own env var
names (`DATABASE_URL`, `JWT_PRIVATE_KEY`, ...). Review feedback rejected
both:

- **`create: true` is removed entirely.** `<area>.secret.existingSecret`
  is now the only way to supply every credential in the chart. This closes
  the actual risk `create: true` posed — secret material passing through
  `values.yaml`/`--set`/a committed overlay/`helm get values`/Helm's own
  release-Secret history — at the cost of the local/CI/test convenience
  it offered (spinning up a fully working deployment from one `--set`
  invocation with no external Secret required). That convenience was never
  worth the risk once weighed directly against it: this chart's own CI
  already proves every `existingSecret` code path with a plain
  `--set <area>.secret.existingSecret=dummy-name` (no Secret object needs
  to actually exist for `helm template`/`lint` to validate a
  `secretKeyRef` pointing at it), so `create: true` bought nothing CI
  didn't already have another way to get.
- **Per-area `existingSecretKey`/`existingSecretKeys` become a uniform
  `secret.keys: {<field>: "<key>"}` map** on every area, single-key or
  multi-key alike (e.g. `database.secret.keys.password`,
  `s3.secret.keys.accessKeyId`/`secretAccessKey`) — one shape everywhere,
  not two depending on how many keys an area happens to need.
- **Default key values switch from the backend's env var names to
  lowercase, dash-separated names** (`password`, `access-key-id`,
  `private-key`, ...) — the Kubernetes Secret key convention, and
  deliberately decoupled from this app's internal env var naming. An
  operator's Secret (hand-created, or synced by External Secrets Operator
  from Vault/AWS Secrets Manager/etc.) is under no obligation to key its
  contents after `team-manager`'s own env vars; forcing that coupling was
  itself a design smell the review feedback correctly flagged.
- `_env.tpl`'s `secretKeyRef.key` for every area now reads
  `{{ $root.Values.<area>.secret.keys.<field> }}` directly — no more
  `team-manager.secretName` create-or-reference resolution helper (deleted
  entirely, along with every per-area `templates/*-secret.yaml` and
  `templates/monitoring-scrape-token-secret.yaml`, all dead code once
  nothing renders a chart-managed Secret).
- **The `checksum/secrets` pod annotation this change originally added is
  removed too** — it existed solely to trigger a rollout when a
  chart-managed secret's plaintext value changed on `helm upgrade`; with
  no chart-managed secret left to checksum, the annotation has nothing to
  do. Rotating an `existingSecret`-referenced credential's value still
  needs a manual `kubectl rollout restart` (unchanged from before this
  change, and now documented uniformly across every area rather than only
  the ones that happened to reference an external Secret already —
  see `docs/operations.md`'s JWT-rotation section).

### `database` split into structural fields, `DATABASE_URL` composed by the chart

Also raised in the same review feedback: `database.secret` held the
*entire* `DATABASE_URL` connection string as one opaque Secret-backed
value, rather than following the same "structural fields plain, only the
actual secret Secret-backed" shape the S3/SMTP/push areas already used.

- `database.host`/`port`/`name`/`username`/`sslmode` become plain values;
  only `database.secret.keys.password` is Secret-backed.
- The backend (`backend/internal/config/config.go`) still needs a single
  `DATABASE_URL` env var — `net/url.Parse`-validated, `postgres://`/
  `postgresql://` scheme required — so this chart must compose one from
  the pieces above. **The backend image is distroless**
  (`gcr.io/distroless/static-debian12`, see `backend/Dockerfile` —
  confirmed by inspection before choosing this approach), so there is no
  shell available to build the string with a wrapper script the way a
  Bitnami-style chart typically would.
- Instead, `templates/_helpers.tpl`'s new `team-manager.databaseEnv` named
  template uses **Kubernetes' own `$(VAR_NAME)` env-var expansion**
  (kubelet-native, not shell — resolves references to *earlier* entries in
  the same container's `env` list, including ones sourced via
  `secretKeyRef`): `DB_HOST`/`DB_PORT`/`DB_NAME`/`DB_USERNAME` (plain) and
  `DB_PASSWORD` (`secretKeyRef`) are declared first, then `DATABASE_URL`'s
  `value` references all five as `postgres://$(DB_USERNAME):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)`.
  Shared via `include` between the main Deployment/migrate initContainer
  (`_env.tpl`) and the backup CronJob's pg-dump container
  (`backup-cronjob.yaml`), so the composition logic exists in exactly one
  place.
- **Known limitation, documented in `values.yaml`'s `database` comment and
  this design doc rather than silently accepted**: Kubernetes' `$(VAR_NAME)`
  expansion does no URL-encoding. A password containing a URL-reserved
  character (`@ : / ? # %`, ...) breaks the composed `DATABASE_URL`'s
  parsing (ambiguous delimiter, or an outright `ErrInvalidDatabaseURL`
  crash-loop) — there is no shell here to percent-encode it, and
  Kubernetes' expansion mechanism has no encoding mode. Mitigation:
  generate the password from an alphanumeric-only charset (e.g.
  `openssl rand -hex 24`), which is already standard practice for a
  generated database credential and sidesteps the whole class of problem.
  This is a real, accepted trade-off, not an oversight — the alternative
  (keeping `DATABASE_URL` as one opaque Secret value, unsplit) was
  rejected by the same review feedback that asked for the split in the
  first place; a shell-based encode-and-wrap approach was considered and
  rejected because the backend image has no shell to run one in.

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

- `secret.keys` maps widen `values.schema.json`'s surface per area (one or
  more new string properties each) — mechanical, low-risk, but touches
  every area's schema block, so the schema diff for this change is large.
- Removing `create: true` is a genuine loss of convenience for local/CI/
  test deployments that used to spin up a fully working release from
  `--set` alone with no external Secret required — now every credential
  needs a real Secret to exist in-cluster before `helm install` (not
  before `helm template`/`lint`, which never touch a live cluster).
  Accepted deliberately (see the "No chart-managed Secret" decision above)
  rather than mitigated, since re-adding any chart-rendered-Secret path
  would reopen the exact risk removing it closed.
- The `database.secret.keys.password` URL-encoding limitation (see the
  `database` decision above) is a real, documented constraint on what
  characters a generated password may contain — not eliminated, only
  mitigated by recommending alphanumeric-only generation. A password from
  an existing external system that already contains a reserved character
  would need to be rotated to a compliant one before it can be used here.
- This is another round of **breaking** changes to `values.yaml`'s shape
  on top of the ones this same change already made (per-area Secrets,
  `existingSecretKey(s)`) — acceptable per the same "no tagged release
  exists yet" rationale as before, but worth noting that this chart's
  `values.yaml` shape has now changed twice within one still-open change;
  `values-staging.yaml`/`values-prod.yaml` are rewritten again alongside
  it so nothing in this repo is left on the intermediate shape.
