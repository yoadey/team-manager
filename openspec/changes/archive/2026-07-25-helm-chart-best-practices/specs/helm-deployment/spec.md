## MODIFIED Requirements

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

## ADDED Requirements

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
