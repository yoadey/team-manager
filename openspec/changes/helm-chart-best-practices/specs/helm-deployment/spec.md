## MODIFIED Requirements

### Requirement: Create-or-reference Secret per area
Each area's `secret` block MUST support both referencing an externally
managed Secret (`existingSecret: <name>`) and having the chart render and
manage that one area's Secret itself from plaintext values (`create:
true`). The key name(s) used within that Secret MUST be overridable via
`existingSecretKey` (single-key areas) or `existingSecretKeys` (multi-key
areas), defaulting to the chart's historical fixed key names, and the same
configured key name MUST be used whether the Secret is externally managed
or chart-managed.

#### Scenario: External secret management (production)
- **WHEN** `<area>.secret.create` is `false` and `<area>.secret.existingSecret`
  names a Secret already present in the cluster
- **THEN** the chart references that Secret's keys — named per
  `<area>.secret.existingSecretKey`/`existingSecretKeys` (the historical
  fixed name by default) — via `secretKeyRef` and creates no Secret object
  of its own for that area

#### Scenario: Chart-managed secret (local/CI/test)
- **WHEN** `<area>.secret.create` is `true` and plaintext fields are set
- **THEN** the chart renders a `<fullname>-<area>` Secret from those
  fields, keyed per that same `existingSecretKey`/`existingSecretKeys`
  configuration, and references it via `secretKeyRef` with no separate
  `existingSecret` needed

#### Scenario: Externally-managed Secret with non-default key names
- **WHEN** an operator's `existingSecret` was populated by tooling (e.g.
  External Secrets Operator) that doesn't use this chart's default key
  names, and they set `<area>.secret.existingSecretKey` (or the relevant
  field under `existingSecretKeys`) to match
- **THEN** the chart's `secretKeyRef` reads that overridden key name
  instead of the default, without requiring the operator to re-key their
  Secret

## ADDED Requirements

### Requirement: Secret-content-change rollout
When any area's `secret.create` is `true`, the Deployment's pod template
MUST carry an annotation derived from the rendered content of every
chart-managed secret template, so that changing a chart-managed secret's
plaintext value (or toggling `create` on/off) on `helm upgrade` triggers a
pod rollout.

#### Scenario: Plaintext secret value changed on upgrade
- **WHEN** `database.secret.create` is `true` and `database.secret.url` is
  changed between two `helm upgrade` invocations
- **THEN** the rendered Deployment's pod template annotation
  `checksum/secrets` differs between the two renders

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
