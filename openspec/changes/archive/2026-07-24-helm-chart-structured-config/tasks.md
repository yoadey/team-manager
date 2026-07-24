## 1. values.yaml restructuring

- [x] 1.1 Remove the flat `env:` map and top-level `existingSecret`.
- [x] 1.2 Add `server` (`port`, `allowedOrigins` list, `trustedProxyCidrs`
      list, `logLevel`, `rateLimitRps`, `errorTypeBaseUri`,
      `apiDeprecationDate`) and `session` (`ttlHours`, `cookie.name`,
      `cookie.secure` bool, `loginRateLimitPerMin`).
- [x] 1.3 Add `database.secret`, `jwt.secret`, `cookieEncryption.secret`
      (`key`/`keys` fields for singular/plural rotation).
- [x] 1.4 Add `s3` (`endpoint`, `region`, `bucket`, `usePathStyle` bool,
      `publicBaseUrl`, `secret`).
- [x] 1.5 Add `smtp` (`host`, `port`, `fromAddress`, `secret`).
- [x] 1.6 Add `selfRegistration` (`enabled` bool, `emailVerificationTtlHours`,
      `registerRateLimitPerMin`, `resendVerificationRateLimitPerMin`).
- [x] 1.7 Add `retention` (`notificationsDays`, `sessionsDays`,
      `auditLogDays`, `unverifiedAccountsDays`).
- [x] 1.8 Add `pagination.secret`, `metrics.allowOpen` + `metrics.secret`,
      `observability` (`otelServiceName`, `otelExporterEndpoint`,
      `environment`, `sentry.secret`).
- [x] 1.9 Rename `monitoring.metricsToken`/`metricsTokenSecretName`/
      `metricsTokenSecretKey` to `monitoring.scrapeToken.{create,token,
      existingSecretName,existingSecretKey}`.
- [x] 1.10 Add `networkPolicy.egress.smtp: { port: 587, to: [] }`.

## 2. values.schema.json

- [x] 2.1 New `helm/team-manager/values.schema.json`: JSON Schema draft-07
      covering every section above — types, enums (`server.logLevel`), string
      patterns where useful (ports), `additionalProperties: false` at every
      object level to catch typos/unknown keys.

## 3. templates/_helpers.tpl

- [x] 3.1 Added `team-manager.secretName` (in `_helpers.tpl`) resolving a
      `<area>.secret` block to either the chart-managed `<fullname>-<area>`
      name (`create: true`) or `existingSecret`, empty string when neither
      is set. Also added `templates/_env.tpl`'s `team-manager.env` (not
      originally scoped as its own task, but needed to avoid duplicating the
      full env-var list across the migrate initContainer and main
      container — see task 5).

## 4. Per-area Secret templates

- [x] 4.1 New `templates/database-secret.yaml`, `templates/jwt-secret.yaml`,
      `templates/cookie-encryption-secret.yaml`, `templates/s3-secret.yaml`,
      `templates/smtp-secret.yaml`, `templates/pagination-secret.yaml`,
      `templates/sentry-secret.yaml`, `templates/metrics-secret.yaml` — each
      gated on its area's `secret.create`, `b64enc`-ing its plaintext fields
      into a `<fullname>-<area>` Secret (pattern from the existing
      `metrics-token-secret.yaml`).
- [x] 4.2 Renamed `templates/metrics-token-secret.yaml` to
      `templates/monitoring-scrape-token-secret.yaml`, field names updated
      per 1.9, cross-namespace behavior unchanged.

## 5. templates/deployment.yaml

- [x] 5.1 Rewrote both the migrate initContainer's and the main container's
      `env` blocks to `{{- include "team-manager.env" $ | nindent 12 }}`
      (see task 3.1) — plaintext values sourced from the new nested value
      paths (quoted at render time), secrets sourced per-area via each
      area's `secret.create`/`existingSecret` (create takes precedence,
      `optional: true` on the already-optional areas, matching prior
      behavior).
- [x] 5.2 `server.allowedOrigins`/`server.trustedProxyCidrs` joined with
      `,` at render time (`join ","`).

## 6. templates/backup-cronjob.yaml

- [x] 6.1 `DATABASE_URL` sourced from `database.secret` (create-or-reference,
      via `team-manager.secretName`) instead of the removed top-level
      `existingSecret`.

## 7. templates/networkpolicy.yaml

- [x] 7.1 Updated the S3 egress gate to `.Values.s3.endpoint`
      (was `.Values.env.S3_ENDPOINT`).
- [x] 7.2 Added the SMTP egress rule gated on `.Values.smtp.host`, mirroring
      the S3 rule's structure and comment style.

## 8. templates/servicemonitor.yaml

- [x] 8.1 Updated field references for the `monitoring.scrapeToken` rename
      (task 1.9); behavior unchanged.

## 9. templates/NOTES.txt

- [x] 9.1 Updated the existing trustedProxyCidrs/metrics-token/S3 warnings'
      value-path references for the new structure.
- [x] 9.2 Added a warning block firing when `session.cookie.secure` is
      `true` and `smtp.host`/`smtp.fromAddress` aren't both set, naming the
      required values and `smtp.secret` keys.

## 10. values-staging.yaml / values-prod.yaml

- [x] 10.1 Rewrote both overlays against the new structure: one
      `existingSecret` name per area instead of one combined
      `existingSecret`, `env.*` moved to their new nested homes.

## 11. Docs

- [x] 11.1 Added an "Outbound email (SMTP)" section to
      `docs/operations.md` (configuration, local-dev fallback to the
      logging fake mailer, Kubernetes/NetworkPolicy wiring), mirroring the
      existing "Object storage (image uploads)" section's structure.
- [x] 11.2 Updated every other `docs/operations.md` reference to
      `existingSecret`/`env.*` (backup/DR restore steps, JWT/cookie
      rotation runbooks, metrics endpoint) for the new per-area value
      paths.
- [x] 11.3 (Not originally scoped, needed for CI to stay green) Updated
      `.github/workflows/ci.yml`'s `helm-lint` job: renamed
      `monitoring.metricsToken`/`metricsTokenSecretName` `--set` flags to
      `monitoring.scrapeToken.*`, added a render exercising every area's
      `secret.create=true` path plus `smtp.host` (previously untested), and
      a negative test confirming `values.schema.json` rejects an
      unknown/typo'd key.

## 12. Verification

- [x] 12.1 `openspec validate helm-chart-structured-config --strict` — passes.
- [x] 12.2 `helm lint helm/team-manager` for the base chart and each of the
      three values files — 0 failures (only the pre-existing informational
      "icon is recommended" notice).
- [x] 12.3 `helm template` renders for: defaults, `values-staging.yaml`,
      `values-prod.yaml`, and a `session.cookie.secure=true` render with
      every area's `secret.create=true` and plaintext values set — confirmed
      both containers get every expected env var (including all five
      `SMTP_*`), each area's Secret renders independently (8 distinct
      `kind: Secret` objects, correctly named), the NetworkPolicy gains the
      SMTP egress rule, and the backup CronJob's `DATABASE_URL` resolves
      correctly against both `existingSecret` (prod overlay) and
      `secret.create=true`.
- [x] 12.4 `helm template` with an unknown/typo'd values key
      (`s3.enpoint=typo`) fails schema validation with a clear
      "Additional property ... is not allowed" error, confirmed both
      manually and as the new CI negative test.
