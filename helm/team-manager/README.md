# team-manager

Helm chart for the Teamverwaltung sports-club management application's Go
backend API (and, via `ingress`, whatever serves the React frontend in
front of it).

## Prerequisites

- Kubernetes >= 1.23 (see `Chart.yaml`'s `kubeVersion`)
- Helm >= 3.16
- A reachable PostgreSQL 17 instance (this chart does not bundle a database)
- For `session.cookie.secure: true` (the default, and every shipped
  overlay): a real `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY` pair, a
  `COOKIE_ENCRYPTION_KEY`/`COOKIE_ENCRYPTION_KEYS`, S3-compatible object
  storage, an SMTP relay, and a VAPID keypair — see
  [`docs/operations.md`](../../docs/operations.md) at the repo root for how
  to provision each

## Installing

```bash
helm upgrade --install team-manager helm/team-manager \
  --set image.tag=1.2.3 \
  --set database.secret.existingSecret=team-manager-database \
  # ... one existingSecret per area, see "Secrets" below
```

Or against a values overlay:

```bash
helm upgrade --install team-manager helm/team-manager \
  -f helm/team-manager/values-prod.yaml \
  --set image.tag=1.2.3
```

`values.schema.json` structurally validates the merged values (types,
enums, unknown/typo'd keys) before anything renders — a mistake here fails
`helm template`/`install`/`upgrade`/`lint` immediately rather than at
pod-startup crash-loop time.

## Upgrading

```bash
helm upgrade team-manager helm/team-manager -f <your-values.yaml>
```

`helm test team-manager` (after install/upgrade) runs a smoke test against
the release's `/healthz` endpoint.

## Secrets

Every functional area with credentials (`database`, `jwt`,
`cookieEncryption`, `s3`, `smtp`, `push`, `pagination`,
`observability.sentry`, `metrics`) has its own `secret` block, independent
of every other area's — rotating one area's credentials never touches
another's Secret:

```yaml
<area>:
  secret:
    create: false          # true: chart renders and manages this one Secret from the plaintext fields below
    existingSecret: ""     # false (default): name of a Secret you manage yourself
    existingSecretKey: ""  # (or existingSecretKeys: {...} for multi-key areas) — override if your
                            # existingSecret uses different key names than this chart's defaults
                            # (e.g. an External Secrets Operator sync with its own naming convention)
    <plaintextField>: ""   # only used when create=true
```

See the per-area comments in [`values.yaml`](values.yaml) for each area's
default key name(s) and which fields are required when
`session.cookie.secure: true`.

## Values

| Key | Type | Default | Description |
|---|---|---|---|
| `replicaCount` | int | `2` | Pod replica count (ignored when `autoscaling.enabled`). |
| `deploymentStrategy` | object | `{}` | Deployment rollout strategy; empty uses Kubernetes' own RollingUpdate default. |
| `terminationGracePeriodSeconds` | int | `60` | Grace period for the backend's multi-phase SIGTERM shutdown (HTTP drain, job worker stop, OTEL shutdown). |
| `image.repository` | string | `ghcr.io/yoadey/team-manager-backend` | Backend container image repository. |
| `image.pullPolicy` | string | `IfNotPresent` | Image pull policy. |
| `image.tag` | string | `""` | Image tag; defaults to `Chart.AppVersion` when empty. **Always set explicitly for a real deploy.** |
| `image.digest` | string | `""` | Pins the image by digest instead of tag when set; takes precedence over `tag`. |
| `imagePullSecrets` | list | `[]` | Names of `docker-registry` Secrets for pulling a private image. |
| `nameOverride` | string | `""` | Overrides the chart name portion of generated resource names. |
| `fullnameOverride` | string | `""` | Overrides the full generated resource name. |
| `serviceAccount.create` | bool | `true` | Whether to create a ServiceAccount. |
| `serviceAccount.automount` | bool | `false` | Auto-mount the ServiceAccount token (off — the backend makes no Kubernetes API calls). |
| `serviceAccount.annotations` | object | `{}` | Annotations on the created ServiceAccount (e.g. an IRSA role ARN). |
| `serviceAccount.name` | string | `""` | ServiceAccount name override. |
| `priorityClassName` | string | `""` | Pod scheduling `PriorityClass` name. |
| `topologySpreadConstraints` | list | `[]` | `topologySpreadConstraints` for the main Deployment's pods. |
| `podAnnotations` | object | `prometheus.io/scrape: "true"`, `prometheus.io/path: "/metrics"` | Extra pod annotations. |
| `podLabels` | object | `{}` | Extra pod labels. |
| `extraEnv` | list | `[]` | Extra raw `EnvVar` entries appended in both the migrate initContainer and the main container. |
| `extraVolumes` | list | `[]` | Extra raw `Volume` entries appended to the pod spec. |
| `extraVolumeMounts` | list | `[]` | Extra raw `VolumeMount` entries appended to the main container. |
| `podSecurityContext` | object | non-root, `runAsUser: 1000`, `fsGroup: 1000`, `seccompProfile: RuntimeDefault` | Pod-level `securityContext`. |
| `securityContext` | object | `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: true`, all capabilities dropped, `seccompProfile: RuntimeDefault` | Container-level `securityContext` (main container and migrate initContainer). |
| `service.type` | string | `ClusterIP` | Service type. |
| `service.port` | int | `80` | Service port. |
| `service.targetPort` | int | `8080` | Container port; also drives probes/NetworkPolicy/Prometheus annotations. |
| `ingress.enabled` | bool | `false` | Create an Ingress. |
| `ingress.className` | string | `""` | `ingressClassName`. |
| `ingress.annotations` | object | `{}` | Ingress annotations (e.g. cert-manager, nginx). |
| `ingress.hosts` | list | one example host | `[{host, paths: [{path, pathType}]}]`. |
| `ingress.tls` | list | `[]` | Ingress TLS entries. |
| `resources` | object | `requests: 100m/128Mi`, `limits: 500m/256Mi` | Main container resource requests/limits (also used by the migrate initContainer). |
| `autoscaling.enabled` | bool | `false` | Create a `HorizontalPodAutoscaler`. |
| `autoscaling.minReplicas` / `maxReplicas` | int | `2` / `10` | HPA replica bounds. |
| `autoscaling.targetCPUUtilizationPercentage` | int | `70` | HPA target CPU utilization. |
| `startupProbe` / `livenessProbe` / `readinessProbe` | object | see `values.yaml` | Probe timing; `httpGet.port` always derives from `service.targetPort`. |
| `nodeSelector` | object | `{}` | Pod node selector. |
| `tolerations` | list | `[]` | Pod tolerations. |
| `affinity` | object | preferred pod anti-affinity by `app.kubernetes.io/name` | Pod affinity/anti-affinity. |
| `server.port` | string | `"8080"` | `PORT`. |
| `server.allowedOrigins` | list | one example origin | `ALLOWED_ORIGINS` (joined with `,`). |
| `server.trustedProxyCidrs` | list | `[]` | `TRUSTED_PROXY_CIDRS` — set when behind a reverse proxy/ingress. |
| `server.logLevel` | string | `info` | `LOG_LEVEL` (`debug`\|`info`\|`warn`\|`error`). |
| `server.rateLimitRps` | int | `100` | `RATE_LIMIT_RPS`. |
| `server.errorTypeBaseUri` | string | `""` | `ERROR_TYPE_BASE_URI`. |
| `server.apiDeprecationDate` | string | `""` | `API_DEPRECATION_DATE`. |
| `session.ttlHours` | int | `720` | `SESSION_TTL_HOURS`. |
| `session.loginRateLimitPerMin` | int | `5` | `LOGIN_RATE_LIMIT_PER_MIN`. |
| `session.cookie.name` | string | `tv_session` | `COOKIE_NAME`. |
| `session.cookie.secure` | bool | `true` | `COOKIE_SECURE`; also gates several startup hard-requirements. |
| `database.secret.*` | — | — | `DATABASE_URL`. See "Secrets" above. |
| `jwt.secret.*` | — | — | `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY`. See "Secrets" above. |
| `cookieEncryption.secret.*` | — | — | `COOKIE_ENCRYPTION_KEY`/`COOKIE_ENCRYPTION_KEYS`. See "Secrets" above. |
| `s3.endpoint` / `region` / `bucket` / `usePathStyle` / `publicBaseUrl` | string/bool | see `values.yaml` | `S3_ENDPOINT`/`S3_REGION`/`S3_BUCKET`/`S3_USE_PATH_STYLE`/`S3_PUBLIC_BASE_URL`. |
| `s3.secret.*` | — | — | `S3_ACCESS_KEY_ID`/`S3_SECRET_ACCESS_KEY`. See "Secrets" above. |
| `smtp.host` / `port` / `fromAddress` | string | see `values.yaml` | `SMTP_HOST`/`SMTP_PORT`/`SMTP_FROM_ADDRESS`. |
| `smtp.secret.*` | — | — | `SMTP_USERNAME`/`SMTP_PASSWORD`. See "Secrets" above. |
| `push.publicKey` / `subject` | string | `""` | `VAPID_PUBLIC_KEY`/`VAPID_SUBJECT`. |
| `push.secret.*` | — | — | `VAPID_PRIVATE_KEY`. See "Secrets" above. |
| `selfRegistration.*` | bool/int | see `values.yaml` | `SELF_REGISTRATION_ENABLED`/`EMAIL_VERIFICATION_TTL_HOURS`/`REGISTER_RATE_LIMIT_PER_MIN`/`RESEND_VERIFICATION_RATE_LIMIT_PER_MIN`. |
| `retention.*` | int | see `values.yaml` | `RETENTION_NOTIFICATIONS_DAYS`/`RETENTION_SESSIONS_DAYS`/`RETENTION_AUDIT_LOG_DAYS`/`RETENTION_UNVERIFIED_ACCOUNTS_DAYS`. |
| `pagination.secret.*` | — | — | `PAGINATION_HMAC_KEY`. See "Secrets" above. |
| `metrics.allowOpen` | bool | `false` | `METRICS_ALLOW_OPEN`. |
| `metrics.secret.*` | — | — | `METRICS_TOKEN`. See "Secrets" above. |
| `observability.otelServiceName` / `otelExporterEndpoint` / `environment` | string | see `values.yaml` | `OTEL_SERVICE_NAME`/`OTEL_EXPORTER_OTLP_ENDPOINT`/`ENVIRONMENT`. |
| `observability.sentry.secret.*` | — | — | `SENTRY_DSN`. See "Secrets" above. |
| `networkPolicy.enabled` | bool | `true` | Create a `NetworkPolicy`. |
| `networkPolicy.ingress.from` | list | `[]` | Restricts ingress source (empty = any in-cluster source, port-restricted only). |
| `networkPolicy.egress.postgres/https/s3/smtp` | object | see `values.yaml` | Per-destination egress `port`/`to`. |
| `podDisruptionBudget.enabled` | bool | `true` | Create a `PodDisruptionBudget`. |
| `podDisruptionBudget.minAvailable` | int | `1` | PDB `minAvailable`. |
| `monitoring.enabled` | bool | `false` | Create `ServiceMonitor`/`PrometheusRule` (requires Prometheus Operator). |
| `monitoring.namespace` | string | `""` | Deploy monitoring resources to a different namespace. |
| `monitoring.scrapeInterval` | string | `30s` | ServiceMonitor scrape interval. |
| `monitoring.scrapeToken.*` | — | see `values.yaml` | Prometheus's own bearer token for scraping `/metrics` (separate from `metrics.secret`). |
| `monitoring.additionalLabels` | object | `{}` | Extra labels on monitoring resources (for Prometheus Operator discovery). |
| `monitoring.grafanaDashboard.enabled` | bool | `false` | Render a ConfigMap with the bundled Grafana dashboard. |
| `migrations.runAsInitContainer` | bool | `true` | Run DB migrations as an initContainer before the app starts. |
| `backup.enabled` | bool | `false` | Enable the daily PostgreSQL backup CronJob. |
| `backup.schedule` | string | `0 2 * * *` | CronJob schedule. |
| `backup.startingDeadlineSeconds` / `activeDeadlineSeconds` | int | `900` / `3600` | How late a missed run may start / how long a running backup may take. |
| `backup.postgresVersion` / `postgresImageDigest` | string | `"17"` / pinned digest | Backup job's `postgres` image. |
| `backup.dumpSizeLimit` / `tmpSizeLimit` | string | `2Gi` / `256Mi` | `emptyDir` size caps. |
| `backup.minDumpEntries` | int | `10` | Minimum `pg_restore --list` TOC entries for a dump to be considered valid. |
| `backup.retentionDays` | int | `30` | Informational only — enforce via bucket lifecycle rules. |
| `backup.s3.*` | — | see `values.yaml` | S3 upload target/credentials for the backup dump. |
| `backup.serviceAccount.*` | — | see `values.yaml` | Dedicated ServiceAccount for the backup CronJob (default: shares the main one). |

This table is hand-maintained alongside `values.yaml`'s own per-key
comments — treat `values.yaml` as authoritative if the two ever disagree.

## License

Apache 2.0 — see [`LICENSE`](LICENSE).
