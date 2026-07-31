{{/*
Renders the full list of backend container env vars, shared by the migrate
initContainer and the main container (templates/deployment.yaml).
config.Load() runs unconditionally before the --migrate-only branch in
main.go, so the initContainer needs every var the main container does,
even ones (e.g. S3/SMTP credentials) migrate-only never itself uses.
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
- name: PASSWORD_RESET_TTL_HOURS
  value: {{ .Values.selfRegistration.passwordResetTtlHours | quote }}
- name: FORGOT_PASSWORD_RATE_LIMIT_PER_MIN
  value: {{ .Values.selfRegistration.forgotPasswordRateLimitPerMin | quote }}
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
{{- with .Values.push.publicKey }}
- name: VAPID_PUBLIC_KEY
  value: {{ . | quote }}
{{- end }}
{{- with .Values.push.subject }}
- name: VAPID_SUBJECT
  value: {{ . | quote }}
{{- end }}
{{ include "team-manager.databaseEnv" $root }}
{{- if $root.Values.jwt.secret.existingSecret }}
- name: JWT_PRIVATE_KEY
  valueFrom:
    secretKeyRef:
      name: {{ $root.Values.jwt.secret.existingSecret }}
      key: {{ $root.Values.jwt.secret.keys.privateKey }}
- name: JWT_PUBLIC_KEY
  valueFrom:
    secretKeyRef:
      name: {{ $root.Values.jwt.secret.existingSecret }}
      key: {{ $root.Values.jwt.secret.keys.publicKey }}
{{- end }}
{{- if $root.Values.cookieEncryption.secret.existingSecret }}
- name: COOKIE_ENCRYPTION_KEY
  valueFrom:
    secretKeyRef:
      name: {{ $root.Values.cookieEncryption.secret.existingSecret }}
      key: {{ $root.Values.cookieEncryption.secret.keys.key }}
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
      name: {{ $root.Values.cookieEncryption.secret.existingSecret }}
      key: {{ $root.Values.cookieEncryption.secret.keys.keys }}
      optional: true
{{- end }}
{{- if $root.Values.s3.secret.existingSecret }}
- name: S3_ACCESS_KEY_ID
  valueFrom:
    secretKeyRef:
      name: {{ $root.Values.s3.secret.existingSecret }}
      key: {{ $root.Values.s3.secret.keys.accessKeyId }}
- name: S3_SECRET_ACCESS_KEY
  valueFrom:
    secretKeyRef:
      name: {{ $root.Values.s3.secret.existingSecret }}
      key: {{ $root.Values.s3.secret.keys.secretAccessKey }}
{{- end }}
{{- if $root.Values.push.secret.existingSecret }}
- name: VAPID_PRIVATE_KEY
  valueFrom:
    secretKeyRef:
      name: {{ $root.Values.push.secret.existingSecret }}
      key: {{ $root.Values.push.secret.keys.privateKey }}
{{- end }}
{{- if $root.Values.smtp.secret.existingSecret }}
- name: SMTP_USERNAME
  valueFrom:
    secretKeyRef:
      name: {{ $root.Values.smtp.secret.existingSecret }}
      key: {{ $root.Values.smtp.secret.keys.username }}
      # Optional: config.go explicitly allows a blank username for an open
      # relay.
      optional: true
- name: SMTP_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ $root.Values.smtp.secret.existingSecret }}
      key: {{ $root.Values.smtp.secret.keys.password }}
      optional: true
{{- end }}
{{- if $root.Values.pagination.secret.existingSecret }}
- name: PAGINATION_HMAC_KEY
  valueFrom:
    secretKeyRef:
      name: {{ $root.Values.pagination.secret.existingSecret }}
      key: {{ $root.Values.pagination.secret.keys.hmacKey }}
      optional: true
{{- end }}
{{- if $root.Values.observability.sentry.secret.existingSecret }}
- name: SENTRY_DSN
  valueFrom:
    secretKeyRef:
      name: {{ $root.Values.observability.sentry.secret.existingSecret }}
      key: {{ $root.Values.observability.sentry.secret.keys.dsn }}
      optional: true
{{- end }}
{{- if $root.Values.metrics.secret.existingSecret }}
- name: METRICS_TOKEN
  valueFrom:
    secretKeyRef:
      name: {{ $root.Values.metrics.secret.existingSecret }}
      key: {{ $root.Values.metrics.secret.keys.token }}
      optional: true
{{- end }}
{{- with $root.Values.extraEnv }}
{{ toYaml . }}
{{- end }}
{{- end }}
