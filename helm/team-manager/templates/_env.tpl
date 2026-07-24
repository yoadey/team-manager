{{/*
Renders the full list of backend container env vars, shared by the migrate
initContainer and the main container (templates/deployment.yaml) and by
the backup CronJob's DATABASE_URL (templates/backup-cronjob.yaml, via the
database-only variant below). config.Load() runs unconditionally before
the --migrate-only branch in main.go, so the initContainer needs every var
the main container does, even ones (e.g. S3/SMTP credentials) migrate-only
never itself uses.
Usage: {{ include "team-manager.env" $ | nindent 12 }}
*/}}
{{- define "team-manager.env" -}}
{{- $root := . -}}
- name: PORT
  value: {{ .Values.server.port | quote }}
- name: ALLOWED_ORIGINS
  value: {{ join "," .Values.server.allowedOrigins | quote }}
{{- with .Values.server.trustedProxyCidrs }}
- name: TRUSTED_PROXY_CIDRS
  value: {{ join "," . | quote }}
{{- end }}
- name: LOG_LEVEL
  value: {{ .Values.server.logLevel | quote }}
- name: RATE_LIMIT_RPS
  value: {{ .Values.server.rateLimitRps | quote }}
{{- with .Values.server.errorTypeBaseUri }}
- name: ERROR_TYPE_BASE_URI
  value: {{ . | quote }}
{{- end }}
{{- with .Values.server.apiDeprecationDate }}
- name: API_DEPRECATION_DATE
  value: {{ . | quote }}
{{- end }}
- name: SESSION_TTL_HOURS
  value: {{ .Values.session.ttlHours | quote }}
- name: LOGIN_RATE_LIMIT_PER_MIN
  value: {{ .Values.session.loginRateLimitPerMin | quote }}
- name: COOKIE_NAME
  value: {{ .Values.session.cookie.name | quote }}
- name: COOKIE_SECURE
  value: {{ .Values.session.cookie.secure | quote }}
- name: SELF_REGISTRATION_ENABLED
  value: {{ .Values.selfRegistration.enabled | quote }}
- name: EMAIL_VERIFICATION_TTL_HOURS
  value: {{ .Values.selfRegistration.emailVerificationTtlHours | quote }}
- name: REGISTER_RATE_LIMIT_PER_MIN
  value: {{ .Values.selfRegistration.registerRateLimitPerMin | quote }}
- name: RESEND_VERIFICATION_RATE_LIMIT_PER_MIN
  value: {{ .Values.selfRegistration.resendVerificationRateLimitPerMin | quote }}
- name: RETENTION_NOTIFICATIONS_DAYS
  value: {{ .Values.retention.notificationsDays | quote }}
- name: RETENTION_SESSIONS_DAYS
  value: {{ .Values.retention.sessionsDays | quote }}
- name: RETENTION_AUDIT_LOG_DAYS
  value: {{ .Values.retention.auditLogDays | quote }}
- name: RETENTION_UNVERIFIED_ACCOUNTS_DAYS
  value: {{ .Values.retention.unverifiedAccountsDays | quote }}
- name: METRICS_ALLOW_OPEN
  value: {{ .Values.metrics.allowOpen | quote }}
- name: OTEL_SERVICE_NAME
  value: {{ .Values.observability.otelServiceName | quote }}
{{- with .Values.observability.otelExporterEndpoint }}
- name: OTEL_EXPORTER_OTLP_ENDPOINT
  value: {{ . | quote }}
{{- end }}
{{- with .Values.observability.environment }}
- name: ENVIRONMENT
  value: {{ . | quote }}
{{- end }}
{{- with .Values.s3.endpoint }}
- name: S3_ENDPOINT
  value: {{ . | quote }}
{{- end }}
- name: S3_REGION
  value: {{ .Values.s3.region | quote }}
{{- with .Values.s3.bucket }}
- name: S3_BUCKET
  value: {{ . | quote }}
{{- end }}
- name: S3_USE_PATH_STYLE
  value: {{ .Values.s3.usePathStyle | quote }}
{{- with .Values.s3.publicBaseUrl }}
- name: S3_PUBLIC_BASE_URL
  value: {{ . | quote }}
{{- end }}
{{- with .Values.smtp.host }}
- name: SMTP_HOST
  value: {{ . | quote }}
{{- end }}
- name: SMTP_PORT
  value: {{ .Values.smtp.port | quote }}
{{- with .Values.smtp.fromAddress }}
- name: SMTP_FROM_ADDRESS
  value: {{ . | quote }}
{{- end }}
{{- $dbSecret := include "team-manager.secretName" (list $root "database" $root.Values.database.secret) }}
{{- if $dbSecret }}
- name: DATABASE_URL
  valueFrom:
    secretKeyRef:
      name: {{ $dbSecret }}
      key: DATABASE_URL
{{- end }}
{{- $jwtSecret := include "team-manager.secretName" (list $root "jwt" $root.Values.jwt.secret) }}
{{- if $jwtSecret }}
- name: JWT_PRIVATE_KEY
  valueFrom:
    secretKeyRef:
      name: {{ $jwtSecret }}
      key: JWT_PRIVATE_KEY
- name: JWT_PUBLIC_KEY
  valueFrom:
    secretKeyRef:
      name: {{ $jwtSecret }}
      key: JWT_PUBLIC_KEY
{{- end }}
{{- $cookieSecret := include "team-manager.secretName" (list $root "cookie-encryption" $root.Values.cookieEncryption.secret) }}
{{- if $cookieSecret }}
- name: COOKIE_ENCRYPTION_KEY
  valueFrom:
    secretKeyRef:
      name: {{ $cookieSecret }}
      key: COOKIE_ENCRYPTION_KEY
      # Optional: config.go's loadCookieEncryptionKeys checks
      # COOKIE_ENCRYPTION_KEYS (plural) first and, if set, never even looks
      # at this singular key -- a Secret populated per the zero-downtime-
      # rotation guidance in CLAUDE.md (only COOKIE_ENCRYPTION_KEYS set)
      # must not fail pod scheduling just because this optional fallback
      # key is absent.
      optional: true
- name: COOKIE_ENCRYPTION_KEYS
  valueFrom:
    secretKeyRef:
      name: {{ $cookieSecret }}
      key: COOKIE_ENCRYPTION_KEYS
      optional: true
{{- end }}
{{- $s3Secret := include "team-manager.secretName" (list $root "s3" $root.Values.s3.secret) }}
{{- if $s3Secret }}
- name: S3_ACCESS_KEY_ID
  valueFrom:
    secretKeyRef:
      name: {{ $s3Secret }}
      key: S3_ACCESS_KEY_ID
- name: S3_SECRET_ACCESS_KEY
  valueFrom:
    secretKeyRef:
      name: {{ $s3Secret }}
      key: S3_SECRET_ACCESS_KEY
{{- end }}
{{- $smtpSecret := include "team-manager.secretName" (list $root "smtp" $root.Values.smtp.secret) }}
{{- if $smtpSecret }}
- name: SMTP_USERNAME
  valueFrom:
    secretKeyRef:
      name: {{ $smtpSecret }}
      key: SMTP_USERNAME
      # Optional: config.go explicitly allows a blank username for an open
      # relay.
      optional: true
- name: SMTP_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ $smtpSecret }}
      key: SMTP_PASSWORD
      optional: true
{{- end }}
{{- $paginationSecret := include "team-manager.secretName" (list $root "pagination" $root.Values.pagination.secret) }}
{{- if $paginationSecret }}
- name: PAGINATION_HMAC_KEY
  valueFrom:
    secretKeyRef:
      name: {{ $paginationSecret }}
      key: PAGINATION_HMAC_KEY
      optional: true
{{- end }}
{{- $sentrySecret := include "team-manager.secretName" (list $root "sentry" $root.Values.observability.sentry.secret) }}
{{- if $sentrySecret }}
- name: SENTRY_DSN
  valueFrom:
    secretKeyRef:
      name: {{ $sentrySecret }}
      key: SENTRY_DSN
      optional: true
{{- end }}
{{- $metricsSecret := include "team-manager.secretName" (list $root "metrics" $root.Values.metrics.secret) }}
{{- if $metricsSecret }}
- name: METRICS_TOKEN
  valueFrom:
    secretKeyRef:
      name: {{ $metricsSecret }}
      key: METRICS_TOKEN
      optional: true
{{- end }}
{{- end }}
