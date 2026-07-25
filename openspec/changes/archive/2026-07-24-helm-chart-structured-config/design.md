## Context

`internal/mailer` (added by `self-service-registration`) is only exercised
in production once `COOKIE_SECURE=true`, at which point `config.go`'s
`loadSMTPConfig` hard-fails startup without `SMTP_HOST`/`SMTP_FROM_ADDRESS`.
Wiring that in surfaced a broader design problem raised in review: the
chart's `env:` flat map and single `existingSecret` "God Secret" have no
structural validation and mix unrelated secrets (DB, JWT, S3, cookie
encryption, Sentry, metrics, pagination — and now SMTP) into one Kubernetes
object. The chart has never shipped a release (`Chart.yaml` `version:
0.1.0`), so this is free to restructure without a migration path.

## Goals / Non-Goals

**Goals:**
- Every backend config surface `config.Load()` can hard-require at startup
  is settable through typed, nested `values.yaml` sections, not a flat
  string map.
- `values.schema.json` structurally validates the whole file at
  `helm template`/`install`/`upgrade`/`lint` time — types, enums, patterns,
  `additionalProperties: false` everywhere to catch typos.
- No single Secret holds more than one functional area's credentials; each
  area (`database`, `jwt`, `cookieEncryption`, `s3`, `smtp`, `pagination`,
  `observability.sentry`, `metrics`) has its own Secret reference.
- Each area's Secret can be either referenced (`existingSecret`, externally
  managed — the only mode the chart supported before) or created by the
  chart itself from plaintext values (`secret.create: true`) for
  local/CI/test deployments driven entirely by `--set`/a single values file.
- NetworkPolicy egress covers the SMTP relay when configured, same as it
  already does for S3.
- Deploy-time feedback (NOTES.txt) when SMTP is missing under
  `COOKIE_SECURE=true`, before the pod crash-loops.

**Non-Goals:**
- Changing `internal/mailer` or `config.go` behavior, or any backend env
  var's *name* or semantics — this is chart-only. The container still gets
  `SMTP_HOST` etc.; only how the chart assembles that value changes.
- Encoding every "required when COOKIE_SECURE=true" cross-field rule into
  `values.schema.json` conditionals. JSON Schema can express some of this
  (`if`/`then`), but the backend's own `config.Load()` is already the
  authoritative runtime check, and NOTES.txt already carries deploy-time
  warnings for the highest-value cases (S3, metrics token, now SMTP). Piling
  full conditional validation onto the schema too would duplicate that
  logic in a second place that can drift from it. The schema's job here is
  structural correctness (right type, right shape, no typos), not business
  rules.
- A generic/parameterized "secret" JSON Schema construct reused via `$ref`
  for all eight areas turned out to need per-area `required`/property lists
  anyway (the plaintext fields differ per area), so each area's schema is
  written out rather than forced through one shared definition — right
  fields and required errors on typos matter more than avoiding repetition
  in a schema file that's read far less often than it's is enforced.

## Decisions

- **Value structure**: `env:` is removed. Its former keys move into typed
  sections: `server` (`port`, `allowedOrigins` — now a YAML list, joined
  with `,` at render time — `trustedProxyCidrs` likewise, `logLevel`,
  `rateLimitRps`, `errorTypeBaseUri`, `apiDeprecationDate`),
  `session` (`ttlHours`, `cookie.name`, `cookie.secure` — now a real
  boolean, `loginRateLimitPerMin`), `s3` (`endpoint`, `region`, `bucket`,
  `usePathStyle` as boolean, `publicBaseUrl`), `smtp` (`host`, `port`,
  `fromAddress`), `selfRegistration` (`enabled` as boolean,
  `emailVerificationTtlHours`, `registerRateLimitPerMin`,
  `resendVerificationRateLimitPerMin`), `retention` (`notificationsDays`,
  `sessionsDays`, `auditLogDays`, `unverifiedAccountsDays`),
  `observability` (`otelServiceName`, `otelExporterEndpoint`, `environment`).
  Booleans/numbers are native YAML types in `values.yaml` (schema-checkable)
  and get `quote`d back to strings only at the point they're rendered as
  container env vars, since the backend's `config.go` parses every env var
  as a string regardless.
- **Secret structure**: every functional area with a secret gets its own
  `<area>.secret: { create: bool, existingSecret: string, <plaintext
  fields...> }`. `create: false` (default) reproduces prior behavior minus
  the God Secret: `existingSecret` names a Secret the deployer manages,
  keyed by the same fixed key names the backend already expects
  (`DATABASE_URL`, `JWT_PRIVATE_KEY`, `S3_ACCESS_KEY_ID`, ... — unchanged
  from today, so an operator's existing per-area Secret content doesn't need
  relabeling, only redistributing across multiple Secret objects instead of
  one). `create: true` renders a chart-managed
  `<fullname>-<area>` Secret from the plaintext fields (`b64enc`'d, same
  pattern already used by `templates/metrics-token-secret.yaml`).
  `create`/`existingSecret` are not mutually validated by the schema (both
  emptyish is a legitimate "this area is entirely unused" state for the
  always-optional areas: SMTP, Sentry, pagination, metrics allow-open); the
  template picks `create` when true, else falls back to `existingSecret`
  when non-empty, else omits that area's env vars/secretKeyRefs entirely
  (`optional: true` on every secretKeyRef, matching current behavior).
- Secret key names inside each area's Secret stay fixed to the literal env
  var name (`DATABASE_URL`, `SMTP_USERNAME`, ...) rather than adding a
  configurable-key-name knob — no known use case needs it, and it would
  only add schema surface without adding safety.
- `monitoring.metricsToken`/`monitoring.metricsTokenSecretName` (the
  ServiceMonitor's *own* bearer-token Secret, deliberately separate from the
  backend's own `METRICS_TOKEN` because Prometheus Operator resolves a
  ServiceMonitor's `bearerTokenSecret` in the ServiceMonitor's own
  namespace, which can differ from the release namespace via
  `monitoring.namespace`) is renamed to `monitoring.scrapeToken: { create,
  token, existingSecretName, existingSecretKey }` for naming consistency
  with the new per-area pattern, but keeps its existing cross-namespace
  behavior unchanged — it is intentionally still a separate Secret object
  from `metrics.secret`, not unified with it.
- New `networkPolicy.egress.smtp: { port: 587, to: [] }`, gated on
  `smtp.host` being non-empty (same conditional style as the existing S3
  rule's `.Values.s3.endpoint` check) — kept as its own rule rather than
  folded into the `https` rule (port 443) since SMTP relays commonly listen
  on 587/465/25, not 443.
- Mirror the full `SMTP_*`/every-area env set into the migrate
  initContainer too, not just the main container — `config.Load()` runs
  unconditionally before the `--migrate-only` branch in `main.go`, so the
  initContainer would crash-loop identically without it (same reasoning
  already documented for why the initContainer carries
  `S3_ACCESS_KEY_ID`/`S3_SECRET_ACCESS_KEY` despite migrate-only never
  touching object storage).

## Risks / Trade-offs

- This is a one-time breaking change to every `values.yaml`/`--set` caller
  of this chart, including `values-staging.yaml`/`values-prod.yaml` in this
  repo, which are rewritten alongside it. Acceptable per this change's
  explicit go-ahead: the chart has never shipped a release, so there is no
  external deployment relying on the old flat shape.
- Splitting one Secret into eight means a deployer who previously created a
  single combined Secret (per the old `existingSecret` convention) must now
  create up to eight, one per area actually in use. `docs/operations.md` is
  updated to spell this out; `templates/NOTES.txt` is updated so the
  deploy-time warnings reference the new per-area value paths instead of
  the removed top-level `existingSecret`.
- `values.schema.json` with `additionalProperties: false` will reject any
  values file carrying now-removed keys (old flat `env.*`, old top-level
  `existingSecret`) with a clear validation error at `helm template` time
  rather than a silent no-op — this is the intended failure mode (loud
  and immediate, not a crash-loop discovered later), but it does mean any
  values overrides written against the old shape must be rewritten, not
  just left in place.
