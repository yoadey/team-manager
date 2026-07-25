# helm-deployment Specification

## Purpose
Defines how `helm/team-manager` deploys the backend (always) and frontend
(optional, `frontend.enabled`) to Kubernetes: typed, schema-validated
`values.yaml` configuration; per-area externally-managed Secrets (never
chart-managed) with overridable, kebab-case key names; a composed
`DATABASE_URL` from structural `database.*` fields; distinct pod identity
for backend vs. frontend resources; and the chart's OCI packaging/release
process.
## Requirements
### Requirement: Schema-validated structured configuration
The Helm chart MUST expose backend configuration through typed, nested
`values.yaml` sections (not a flat string map), and MUST ship a
`values.schema.json` that structurally validates types, enums, and unknown
keys at `helm template`/`install`/`upgrade`/`lint` time.

#### Scenario: Typo'd or wrongly-typed value
- **WHEN** a values override sets an unknown key under a known section, or
  sets a boolean-typed field to a string, or an enum-typed field to a value
  outside its allowed set
- **THEN** `helm template`/`install`/`upgrade`/`lint` fails with a schema
  validation error before anything is rendered or applied

### Requirement: Per-area Secret references, no combined Secret
Each functional area with sensitive configuration (`database`, `jwt`,
`cookieEncryption`, `s3`, `smtp`, `pagination`, `observability.sentry`,
`metrics`) MUST source its secret values from its own dedicated Secret
reference, not a single chart-wide Secret.

#### Scenario: Rotating one area's credentials
- **WHEN** an operator rotates the S3 access key
- **THEN** only the Secret referenced by `s3.secret` needs to change; no
  other area's Secret is touched or has access to the new value

### Requirement: Create-or-reference Secret per area
(Retitled in effect to "externally-managed Secret only" — the
chart-managed/`create: true` half of this requirement's original name no
longer exists, see below; the header is kept unchanged so this delta
applies against the archived requirement of the same name.)

Each area with credentials (`database`, `jwt`, `cookieEncryption`, `s3`,
`smtp`, `push`, `pagination`, `observability.sentry`, `metrics`,
`monitoring.scrapeToken`, `backup.s3`) MUST source its secret values
exclusively from an externally-managed Secret named by
`<area>.secret.existingSecret` — the chart MUST NOT render or manage a
Secret object itself for any of these areas. The key name(s) used within
that Secret MUST be overridable via a `<area>.secret.keys.<field>` map,
defaulting to lowercase, dash-separated names distinct from the backend's
own environment variable names.

#### Scenario: External secret management (production)
- **WHEN** `<area>.secret.existingSecret` names a Secret already present
  in the cluster
- **THEN** the chart references that Secret's keys — named per
  `<area>.secret.keys.<field>` (a lowercase, dash-separated default per
  field, e.g. `password`, `access-key-id`) — via `secretKeyRef`, and
  creates no Secret object of its own for that area

#### Scenario: Chart-managed secret (local/CI/test)
- **WHEN** a values file sets `<area>.secret.create: true` (the mode this
  scenario previously described)
- **THEN** `helm template`/`install`/`upgrade`/`lint` fails with a schema
  validation error (`Additional property create is not allowed`) — this
  mode was removed; `<area>.secret.existingSecret` is the only supported
  way to supply credentials, so that secret material never has to pass
  through `values.yaml`/`--set`/a committed overlay/Helm's release history

#### Scenario: Externally-managed Secret with non-default key names
- **WHEN** an operator's `existingSecret` was populated by tooling (e.g.
  External Secrets Operator) that doesn't use this chart's default key
  names, and they set the relevant `<area>.secret.keys.<field>` to match
- **THEN** the chart's `secretKeyRef` reads that overridden key name
  instead of the default, without requiring the operator to re-key their
  Secret

### Requirement: SMTP configuration wired end to end
The chart MUST expose `smtp.host`, `smtp.port`, and `smtp.fromAddress` as
plaintext values, and `SMTP_USERNAME`/`SMTP_PASSWORD` as optional keys
sourced from `smtp.secret`, in both the migrate initContainer and the main
container.

#### Scenario: COOKIE_SECURE=true deployment with SMTP configured
- **WHEN** a deployer sets `smtp.host`/`smtp.fromAddress` and either
  `smtp.secret.existingSecret` (populated with `SMTP_USERNAME`/
  `SMTP_PASSWORD`) or `smtp.secret.create: true` with those fields set
- **THEN** both the migrate initContainer and the main container receive
  all five `SMTP_*` env vars and the backend starts without
  `ErrSMTPConfigRequired`

#### Scenario: Open relay with no credentials
- **WHEN** neither `smtp.secret.existingSecret` nor `smtp.secret.create` is
  set
- **THEN** pod scheduling still succeeds and no `SMTP_USERNAME`/
  `SMTP_PASSWORD` env vars are injected, matching `config.go`'s own
  allowance for a blank username/password

### Requirement: SMTP NetworkPolicy egress
When `networkPolicy.enabled` is true and `smtp.host` is set, the chart's
NetworkPolicy MUST include an egress rule permitting outbound traffic to
the configured SMTP port.

#### Scenario: NetworkPolicy enabled with SMTP configured
- **WHEN** `networkPolicy.enabled` is `true` and `smtp.host` is non-empty
- **THEN** the rendered NetworkPolicy includes an egress rule on
  `networkPolicy.egress.smtp.port` (default `587`), optionally restricted to
  `networkPolicy.egress.smtp.to`

### Requirement: Deploy-time SMTP warning
`helm install`/`upgrade` output MUST warn when `session.cookie.secure` is
`true` and `smtp.host` is unset, before the pod crash-loops on
`ErrSMTPConfigRequired`.

#### Scenario: Missing SMTP under COOKIE_SECURE=true
- **WHEN** `session.cookie.secure` is `true` and `smtp.host` is empty
- **THEN** `templates/NOTES.txt` renders a warning naming the required
  values and the `smtp.secret` keys

### Requirement: Helm chart published to an OCI registry on release
Tagging a release (`vX.Y.Z`) MUST package `helm/team-manager` and push it as
a versioned OCI artifact to GHCR, signed the same way as the release's
container images.

#### Scenario: Tagged release
- **WHEN** a `vX.Y.Z` tag is pushed
- **THEN** `.github/workflows/release.yml`'s `helm-chart` job packages the
  chart with `version`/`appVersion` set to `X.Y.Z`, pushes it to
  `oci://ghcr.io/<owner>/charts/team-manager`, and signs the pushed digest
  with keyless cosign

#### Scenario: Manual dispatch
- **WHEN** the workflow is run manually via `workflow_dispatch` with a
  `version` input
- **THEN** the chart is packaged and pushed using that input as the version,
  mirroring how the `images` job resolves `workflow_dispatch` versions

### Requirement: Database connection composed from structural fields
`database.host`, `database.port`, `database.name`, and
`database.username` MUST be plain (non-secret) values; only
`database.secret.keys.password` is Secret-backed. The chart MUST compose
the backend's required `DATABASE_URL` connection string from these pieces
at container start, without requiring a shell in the container image.

#### Scenario: DATABASE_URL composed for the main container and migrate initContainer
- **WHEN** `database.host`/`port`/`name`/`username` and
  `database.secret.existingSecret` are all set
- **THEN** both the migrate initContainer's and the main container's
  `DATABASE_URL` env var resolves to
  `postgres://<username>:<password>@<host>:<port>/<name>` (optionally with
  `?sslmode=<database.sslmode>`), with `<password>` sourced from the
  Secret via Kubernetes' native `$(VAR_NAME)` env-var expansion — not a
  shell script

#### Scenario: DATABASE_URL composed for the backup CronJob
- **WHEN** `backup.enabled` is `true`
- **THEN** the backup CronJob's pg-dump container's `DATABASE_URL` env var
  is composed the same way, via the same shared template

### Requirement: Forward-compatibility escape hatches
The chart MUST expose `extraEnv`, `extraVolumes`, `extraVolumeMounts`, and
`podLabels` values that are additively applied to the main container/pod
spec, without requiring a chart change to use them.

#### Scenario: Extra env var for an unmodeled feature
- **WHEN** `extraEnv` contains an entry not present in the chart's
  generated env var list
- **THEN** that entry appears in both the migrate initContainer's and the
  main container's rendered `env:` list, alongside the chart-generated
  entries

#### Scenario: Extra volume mount
- **WHEN** `extraVolumes` and `extraVolumeMounts` both reference the same
  volume name
- **THEN** the rendered pod spec's `volumes:` includes that volume and the
  main container's `volumeMounts:` includes the corresponding mount

### Requirement: Packaged chart includes README and LICENSE
The chart directory MUST contain a `README.md` documenting every
`values.yaml` key and a `LICENSE` file, so the `.tgz` artifact
`release.yml` packages and pushes to the OCI registry includes both.

#### Scenario: Chart packaged for release
- **WHEN** `helm package helm/team-manager` is run
- **THEN** the resulting archive contains `README.md` and `LICENSE`

### Requirement: Production-readiness scheduling values
The chart MUST expose `priorityClassName` (wired into the main Deployment
and backup CronJob pod specs) and `topologySpreadConstraints` (wired into
the main Deployment pod spec), both omitted from the rendered pod spec
when left at their empty defaults.

#### Scenario: Priority class set
- **WHEN** `priorityClassName` is set to a non-empty string
- **THEN** both the Deployment's and (when `backup.enabled`) the backup
  CronJob's pod specs render that `priorityClassName`

#### Scenario: Topology spread constraints set
- **WHEN** `topologySpreadConstraints` is a non-empty list
- **THEN** the Deployment's pod spec renders it verbatim, alongside (not
  replacing) any configured `affinity`

### Requirement: Main image digest pinning
The chart MUST support pinning the main application image by digest via
`image.digest`, taking precedence over `image.tag` when set.

#### Scenario: Digest set
- **WHEN** `image.digest` is a non-empty string
- **THEN** both the migrate initContainer's and the main container's
  `image:` reference is `<image.repository>@<image.digest>`, and
  `image.tag`/`Chart.AppVersion` are not used

#### Scenario: Digest unset (default)
- **WHEN** `image.digest` is empty
- **THEN** the image reference falls back to
  `<image.repository>:<image.tag, defaulting to Chart.AppVersion>`,
  matching current behavior

### Requirement: Component label on chart-managed resources
Every resource the chart renders for the main application (Deployment,
Service, ServiceAccount, per-area Secrets, PodDisruptionBudget,
HorizontalPodAutoscaler, NetworkPolicy, monitoring resources) MUST carry
`app.kubernetes.io/component: backend`, without adding it to the Deployment's
immutable `spec.selector.matchLabels`.

#### Scenario: Rendered resource labels
- **WHEN** any chart-managed resource is rendered
- **THEN** its `metadata.labels` includes `app.kubernetes.io/component: backend`
  and the Deployment's `spec.selector.matchLabels` remains unchanged
  (`app.kubernetes.io/name` and `app.kubernetes.io/instance` only)

### Requirement: `helm test` smoke test hook
The chart MUST ship a `helm.sh/hook: test` resource that verifies a live
release's health endpoint is reachable.

#### Scenario: `helm test` run against a live release
- **WHEN** `helm test` is run after `helm install`/`upgrade`
- **THEN** the hook Pod curls the release's Service `/healthz` endpoint and
  exits non-zero if it does not receive a successful response

### Requirement: Optional frontend deployment
The chart MUST be able to deploy the frontend image (Deployment, Service,
Ingress, and optionally an HPA and PodDisruptionBudget) as well as the
backend, gated on `frontend.enabled` (default `false`).

#### Scenario: Frontend disabled (default)
- **WHEN** `frontend.enabled` is `false` (the default)
- **THEN** no frontend Deployment/Service/Ingress/HPA/PDB/NetworkPolicy is
  rendered, and every backend resource renders identically to a chart
  release with no `frontend` values set at all

#### Scenario: Frontend enabled
- **WHEN** `frontend.enabled` is `true`
- **THEN** the chart renders a frontend Deployment running
  `frontend.image.repository` (tag/digest per the same precedence as the
  backend's `image`), a Service on `frontend.service.port` targeting
  `frontend.service.targetPort`, and — when `frontend.ingress.enabled` is
  also `true` — an Ingress for it

### Requirement: Frontend pods use a distinct selector identity
Frontend-managed resources MUST use an `app.kubernetes.io/name` distinct
from the backend's, so that no backend-scoped selector (NetworkPolicy,
PodDisruptionBudget, Service) can incidentally match frontend pods or vice
versa.

#### Scenario: Frontend and backend both enabled
- **WHEN** `frontend.enabled` is `true`
- **THEN** the backend's `NetworkPolicy`, `PodDisruptionBudget`, and
  `Service` selectors match only backend pods, and the frontend's
  corresponding resources match only frontend pods

### Requirement: Frontend runtime configuration as plain values
The chart MUST expose the frontend's runtime configuration
(`API_BASE_URL`, `SENTRY_DSN`, `VAPID_PUBLIC_KEY`, every `OPERATOR_*` env
var) as plain `frontend.*` values rendered directly as container env vars
— no Secret is created or referenced for these.

#### Scenario: VAPID public key not explicitly set on the frontend
- **WHEN** `push.publicKey` is set and `frontend.vapidPublicKey` is left
  at its empty default
- **THEN** the frontend container's `VAPID_PUBLIC_KEY` env var resolves to
  `push.publicKey`'s value

#### Scenario: VAPID public key explicitly overridden
- **WHEN** both `push.publicKey` and `frontend.vapidPublicKey` are set to
  different values
- **THEN** the frontend container's `VAPID_PUBLIC_KEY` env var uses
  `frontend.vapidPublicKey`'s value, not `push.publicKey`'s

### Requirement: Frontend NetworkPolicy limited to DNS egress
When `frontend.networkPolicy.enabled` is `true`, the frontend's
NetworkPolicy MUST NOT permit any egress beyond DNS resolution — the
frontend container never originates outbound application traffic itself.

#### Scenario: Frontend NetworkPolicy rendered
- **WHEN** `frontend.enabled` and `frontend.networkPolicy.enabled` are
  both `true`
- **THEN** the rendered NetworkPolicy's only egress rule is DNS
  (port 53, TCP and UDP)

