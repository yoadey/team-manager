## ADDED Requirements

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
Each area's `secret` block MUST support both referencing an externally
managed Secret (`existingSecret: <name>`, keyed by the same fixed key names
the backend env vars use) and having the chart render and manage that one
area's Secret itself from plaintext values (`create: true`).

#### Scenario: External secret management (production)
- **WHEN** `<area>.secret.create` is `false` and `<area>.secret.existingSecret`
  names a Secret already present in the cluster
- **THEN** the chart references that Secret's fixed-name keys via
  `secretKeyRef` and creates no Secret object of its own for that area

#### Scenario: Chart-managed secret (local/CI/test)
- **WHEN** `<area>.secret.create` is `true` and plaintext fields are set
- **THEN** the chart renders a `<fullname>-<area>` Secret from those fields
  and references it via `secretKeyRef`, with no separate
  `existingSecret` needed

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
