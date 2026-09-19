## REMOVED Requirements

### Requirement: Database connection composed from structural fields
**Reason**: Its scenario set covered two consumers of the composed
`DATABASE_URL` — the application and the backup CronJob's pg-dump
container. With the CronJob gone there is only one consumer, and a
MODIFIED block cannot drop a scenario. Restated below as "Database
connection composed for the application", unchanged apart from that.

**Migration**: None. The composition behaviour for the application is
identical; only the requirement's name and its CronJob scenario change.

## ADDED Requirements

### Requirement: Database connection composed for the application
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

### Requirement: The chart ships no database backup workload
The chart MUST NOT render a backup CronJob, its ServiceAccount, or any
`backup.*` configuration surface. It does not deploy the database
(`database.host` names an externally-operated PostgreSQL), so backing that
database up is the operator's responsibility, and `docs/operations.md`
MUST say so alongside the dump/restore guidance it keeps.

#### Scenario: Rendering the chart with default values
- **WHEN** the chart is rendered with any shipped values file
- **THEN** no CronJob and no backup ServiceAccount appear in the output

#### Scenario: An upgrade still setting backup values
- **WHEN** an operator upgrades with a values file that still sets any
  `backup.*` key
- **THEN** `helm template`/`upgrade` fails with a schema validation error
  naming the unknown key, rather than silently dropping the workload and
  leaving the operator believing backups still run

## MODIFIED Requirements

### Requirement: Production-readiness scheduling values
The chart MUST expose `priorityClassName` and `topologySpreadConstraints`,
both wired into the main Deployment's pod spec, and both omitted from the
rendered pod spec when left at their empty defaults.

#### Scenario: Priority class set
- **WHEN** `priorityClassName` is set to a non-empty string
- **THEN** the Deployment's pod spec renders that `priorityClassName`

#### Scenario: Topology spread constraints set
- **WHEN** `topologySpreadConstraints` is a non-empty list
- **THEN** the Deployment's pod spec renders it verbatim, alongside (not
  replacing) any configured `affinity`

### Requirement: Create-or-reference Secret per area
(Retitled in effect to "externally-managed Secret only" — the
chart-managed/`create: true` half of this requirement's original name no
longer exists, see below; the header is kept unchanged so this delta
applies against the archived requirement of the same name.)

Each area with credentials (`database`, `jwt`, `cookieEncryption`, `s3`,
`smtp`, `push`, `pagination`, `observability.sentry`, `metrics`,
`monitoring.scrapeToken`) MUST source its secret values
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
