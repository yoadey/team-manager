## Why

`helm/team-manager` is already a mature, actively-hardened chart (typed
`values.yaml` sections, `values.schema.json`, per-area `create`-or-
`existingSecret` Secrets, `NetworkPolicy`, `PodDisruptionBudget`, HPA v2,
digest-pinned backup images, cosign-signed OCI chart publishing in
`release.yml`), but it was never audited end-to-end against the current
(2025/2026) Helm chart best-practices literature (official Helm
`chart_best_practices` guide, `chart-testing`/Artifact Hub conventions,
Kubernetes label/Pod Security Standards guidance). An independent research
pass against those sources (official Helm docs, `helm/chart-testing`,
Artifact Hub, `kubernetes.io` labels docs, community/vendor writeups)
surfaced concrete, actionable gaps grouped below.

**The most concrete gap, and the one explicitly requested**: every
`<area>.secret.existingSecret` reference in this chart hard-codes the key
name it expects inside that externally-managed Secret (e.g. `jwt.secret`
always reads `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY`, `s3.secret` always reads
`S3_ACCESS_KEY_ID`/`S3_SECRET_ACCESS_KEY`). Only
`monitoring.scrapeToken.existingSecretKey` breaks from this pattern. An
operator whose Secret is populated by a tool they don't control the key
naming of (e.g. an `ExternalSecrets` sync from Vault/AWS Secrets Manager
using that provider's own key convention) currently cannot point this chart
at it without first re-keying the Secret to match — exactly the gap the
Helm best-practices literature calls out ("secret key names inside
`existingSecret` are themselves overridable, not hardcoded to one literal
key").

> **Revision (post-review)**: this change originally also added a
> `<area>.secret.create: true` chart-managed-Secret mode (plaintext values
> rendered into a chart-created Secret, for local/CI/test convenience) and
> a `checksum/secrets` pod annotation to trigger a rollout when that
> plaintext value changed. Review feedback on the resulting PR asked for a
> stricter design instead: **no chart-managed Secret at all** —
> `existingSecret` is the only path, so secret material never transits
> `values.yaml`/`--set`/Helm's release history — plus a uniform
> `secret.keys: {<field>: "<kebab-case-key>"}` map (not a singular
> `existingSecretKey`/`existingSecretKeys`, and not the backend's own env
> var names as defaults) across *every* secret-bearing area, and splitting
> `database` into plain `host`/`port`/`name`/`username`/`sslmode` fields
> with only the password Secret-backed (composed into `DATABASE_URL` — see
> "What Changes" below). The `create`/plaintext-value mode and the
> `checksum/secrets` annotation it needed are gone entirely as a result —
> see design.md's "Secret key-name overrides" and "No chart-managed
> Secret" decisions for the full rationale. The bullets below describe the
> **final** design, not the superseded intermediate one.

Beyond the secret-structure work, the research pass found:

- **No forward-compatibility escape hatches** (`extraEnv`,
  `extraVolumes`/`extraVolumeMounts`, `podLabels`) — any one-off need (an
  extra sidecar volume mount, an extra env var for a feature this chart
  doesn't model yet) currently requires forking the chart.
- **Missing chart packaging/metadata hygiene**: no `README.md` or `LICENSE`
  inside `helm/team-manager` (so neither is bundled into the `.tgz` this
  repo's `release.yml` already packages and pushes to `ghcr.io` via OCI),
  no `Chart.yaml` `sources`, no `.helmignore`.
- **Missing production-readiness knobs** flagged repeatedly in current
  survey data as commonly absent even from well-maintained charts:
  `priorityClassName`, `topologySpreadConstraints`, and (for the main app
  image specifically, mirroring the digest-pinning this chart already does
  for its backup-job images) an optional `image.digest`.
- **Missing `app.kubernetes.io/component` label** on the main app's
  resources — the backup CronJob's pod template already carries
  `component: backup` (used by `pdb.yaml`'s selector exclusion), but
  nothing marks the main Deployment/Service/Secrets as `component: backend`
  for symmetry and for tools that group by that label.
- **CI gaps**: `helm lint` (both `ci.yml` and `release.yml`) runs without
  `--strict` (silently passes on WARNING-level findings), and rendered
  manifests are never schema-validated against the real Kubernetes OpenAPI
  schema (`kubeconform`) — `helm lint`/`helm template` catch Helm-level
  and chart-schema-level problems but not "this field doesn't exist on this
  Kubernetes resource kind" typos.
- **No `helm test` hook** — nothing exercises a live release's actual
  health endpoint post-install/upgrade the way `helm test` is meant to.

## What Changes

- **No chart-managed Secrets, anywhere**: `<area>.secret.create` and every
  plaintext-value field it gated (`database.secret.url`,
  `jwt.secret.privateKey`, ...) are gone. `<area>.secret.existingSecret` is
  the only way to supply credentials — the chart never renders a Secret
  object itself, so secret material never has to pass through
  `values.yaml`/`--set`/a committed overlay/Helm's own release-Secret
  history. Applies uniformly to every credential in the chart:
  `database`, `jwt`, `cookieEncryption`, `s3`, `smtp`, `push`,
  `pagination`, `observability.sentry`, `metrics`,
  `monitoring.scrapeToken`, and `backup.s3`.
- **Configurable, kebab-case secret key names**: every `<area>.secret`
  block gains a `keys: {<field>: "<kebab-case-key>"}` map (e.g.
  `database.secret.keys.password`, `s3.secret.keys.accessKeyId` →
  `access-key-id`), replacing the single-area `existingSecretKey`/
  `existingSecretKeys` naming from an earlier iteration of this change.
  Defaults are lowercase, dash-separated (the Kubernetes Secret key
  convention) and deliberately **not** the backend's own env var names
  (`DATABASE_URL`, `JWT_PRIVATE_KEY`, ...) — an operator's `existingSecret`
  is under no obligation to key its contents after this app's internal
  env var naming.
- **`database` split into structural fields**: `host`, `port`, `name`,
  `username`, `sslmode` are now plain values (not folded into one opaque
  connection string), with only the password Secret-backed
  (`database.secret.keys.password`). The backend still needs a single
  `DATABASE_URL` connection string — this chart composes it from those
  pieces at container start using Kubernetes' own `$(VAR_NAME)` env-var
  expansion (no shell involved: the backend image is distroless, see
  design.md), with one documented limitation around URL-reserved
  characters in the password (see values.yaml's `database` comment).
- **Escape hatches**: `extraEnv` (list, appended after generated env vars
  in both the migrate initContainer and the main container),
  `extraVolumes`/`extraVolumeMounts`, and `podLabels`.
- **Chart packaging/metadata**: `helm/team-manager/README.md` (values
  table + prerequisites/install instructions), `helm/team-manager/LICENSE`
  (copied from the repo root Apache-2.0 `LICENSE`, since `helm package`
  only bundles the chart directory), `.helmignore`, and `Chart.yaml`
  `sources`.
- **Production-readiness values**: `priorityClassName` (string, wired into
  the Deployment and backup CronJob pod specs), `topologySpreadConstraints`
  (list, wired into the Deployment pod spec alongside the existing
  `affinity`), `image.digest` (optional, appended to the main app image
  reference the same way `backup.postgresImageDigest`/
  `backup.s3.awsCliImage`'s digest already are).
- **`app.kubernetes.io/component: backend`** added to `team-manager.labels`
  (non-selector, so `selectorLabels` stays minimal/immutable).
- **CI**: `helm lint --strict` (both `ci.yml` and `release.yml`),
  `kubeconform` validation of every `helm template` render already
  exercised in `ci.yml`'s `helm-lint` job, and new `--set` coverage for
  every new conditional branch above, following that job's existing
  per-branch-comment convention.
- **`templates/tests/test-connection.yaml`**: a `helm.sh/hook: test` Pod
  that curls the Service's `/healthz` endpoint, so any real deployment can
  be smoke-tested with `helm test` after install/upgrade. Covered by
  `ci.yml`'s existing client-side `helm template` render; actually
  executing `helm test` needs a live cluster and is left as follow-up (see
  design.md's Non-Goals).
- **`values.schema.json`** updated for every new/changed field above.
- **`docs/operations.md`** updated to mention the key-override capability
  wherever it currently documents an area's fixed `existingSecret` key
  names.

### Explicitly out of scope (see design.md's Non-Goals)

- `chart-testing` (`ct lint`/`ct install`), `helm-unittest`, and actually
  *executing* the new `helm test` hook in CI — all need either a live
  (`kind`) cluster in CI or a test suite written from scratch; sized for
  their own follow-up change rather than bundled here. The `helm test` hook
  itself still ships as chart content in this change (an operator can run
  it against any real install today); only its CI execution is deferred.
- Changing `NetworkPolicy`'s default-open (port-restricted only) ingress/
  egress posture to default-deny — already a deliberate, documented
  trade-off in the existing chart to avoid breaking deployments that
  haven't set `networkPolicy.ingress.from`/`egress.*.to`; revisiting that
  default is a behavior change, not a best-practices-hygiene fix, and
  belongs in its own change with its own migration story.
- Chart signing/provenance for OCI publishing — already implemented
  (`release.yml`'s `cosign sign --yes` step); no gap found here.

## Capabilities

### Modified Capabilities

- `helm-deployment`: removes the chart-managed-Secret (`create: true`)
  mode entirely in favor of `existingSecret`-only, adds a uniform
  kebab-case `secret.keys` map per area, splits `database` into structural
  fields with a composed `DATABASE_URL`, adds forward-compatibility escape
  hatches, chart packaging/metadata hygiene, and additional
  production-readiness values (`priorityClassName`,
  `topologySpreadConstraints`, `image.digest`), plus CI validation
  (`--strict` lint, `kubeconform`, `helm test`).

## Impact

- `helm/team-manager/`: `values.yaml`, `values.schema.json`,
  `templates/_helpers.tpl`, `templates/_env.tpl`, `templates/deployment.yaml`,
  `templates/backup-cronjob.yaml`, `templates/servicemonitor.yaml`, new
  `templates/tests/test-connection.yaml`, new `README.md`, `LICENSE`,
  `.helmignore`, `Chart.yaml`. Every per-area `templates/*-secret.yaml`
  (database, jwt, cookie-encryption, s3, smtp, push, pagination, sentry,
  metrics) and `templates/monitoring-scrape-token-secret.yaml` are
  **deleted** — dead code once no area supports chart-managed Secrets.
- `values-staging.yaml`/`values-prod.yaml`: **breaking** changes to the
  `database`/`jwt`/`cookieEncryption`/`s3`/`smtp`/`push`/`pagination`/
  `observability.sentry`/`metrics` shape (existingSecretKey(s) → keys map)
  and new required `database.host`/`name`/`username` fields — acceptable
  per this change's explicit go-ahead (no tagged chart release exists
  yet), see design.md.
- `docs/operations.md`: rewrites every `.secret.create=true`/
  `existingSecretKey(s)` mention to the `existingSecret`-only, `keys`-map
  shape; rewrites the JWT-rotation and DR-restore sections' now-inaccurate
  `checksum/secrets`/direct-Secret-edit guidance.
- `.github/workflows/ci.yml` and `.github/workflows/release.yml`:
  `helm lint --strict`; `ci.yml` additionally gets a `kubeconform`
  installation/validation step and new `--set` coverage for the
  `existingSecret`-only shape (including non-default `keys` overrides).
- No application code, API, or database schema change; no migration.
  Chart-only. `backend/internal/config/config.go`'s `DATABASE_URL`
  contract is unchanged — only how the chart assembles the value it hands
  the container changes.
