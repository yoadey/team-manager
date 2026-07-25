{{/*
Expand the name of the chart.
*/}}
{{- define "team-manager.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "team-manager.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart label.
*/}}
{{- define "team-manager.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "team-manager.labels" -}}
helm.sh/chart: {{ include "team-manager.chart" . }}
{{ include "team-manager.selectorLabels" . }}
{{- with .Values.image.tag | default .Chart.AppVersion }}
app.kubernetes.io/version: {{ . | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: backend
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "team-manager.selectorLabels" -}}
app.kubernetes.io/name: {{ include "team-manager.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
ServiceAccount name.
*/}}
{{- define "team-manager.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "team-manager.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Main application image reference. image.digest, when set, takes precedence
over image.tag (mirroring how backup.postgresImageDigest/
backup.s3.awsCliImage's digest suffix already work) -- pins the exact image
content against a tag-hijack of a mutable tag, at the cost of needing a
manual re-pin whenever a new version is released (unlike image.tag, nothing
in this chart's release process resolves/bumps a digest automatically).
Usage: image: {{ include "team-manager.image" . }}
*/}}
{{- define "team-manager.image" -}}
{{- if .Values.image.digest -}}
"{{ .Values.image.repository }}@{{ .Values.image.digest }}"
{{- else -}}
"{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
{{- end -}}
{{- end }}

{{/*
Renders the env var entries needed to reach Postgres, ending in a composed
DATABASE_URL -- shared by the main container/migrate initContainer
(templates/_env.tpl) and the backup CronJob's pg-dump container
(templates/backup-cronjob.yaml), so the composition logic exists in
exactly one place.

The backend requires a single DATABASE_URL connection string
(postgres://user:password@host:port/dbname), but database.secret.keys.password
is only ever visible as a Secret key, never as a plaintext value.yaml
value (see values.yaml's header comment) -- so DATABASE_URL is composed
here from the plain database.* fields plus the Secret-sourced password
using Kubernetes' own $(VAR_NAME) env-var expansion. This requires
DB_HOST/DB_PORT/DB_NAME/DB_USERNAME/DB_PASSWORD to all appear *earlier* in
the same container's env list than DATABASE_URL itself -- $(VAR_NAME)
substitution only resolves references to previously-defined entries in
the same list, so this whole block must be included as a unit, not
split/reordered by callers.

LIMITATION: no shell is available to percent-encode the password (the
backend image is distroless, see backend/Dockerfile; the backup CronJob's
postgres image *does* have a shell, but reusing this same composition
keeps exactly one code path rather than two divergent ones) -- see
values.yaml's database.secret comment for the resulting constraint on
what characters a generated password may safely contain.

Usage: {{ include "team-manager.databaseEnv" $ | nindent 12 }}
*/}}
{{- define "team-manager.databaseEnv" -}}
- name: DB_HOST
  value: {{ .Values.database.host | quote }}
- name: DB_PORT
  value: {{ .Values.database.port | quote }}
- name: DB_NAME
  value: {{ .Values.database.name | quote }}
- name: DB_USERNAME
  value: {{ .Values.database.username | quote }}
- name: DB_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ .Values.database.secret.existingSecret }}
      key: {{ .Values.database.secret.keys.password }}
- name: DATABASE_URL
  value: "postgres://$(DB_USERNAME):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME){{ with .Values.database.sslmode }}?sslmode={{ . }}{{ end }}"
{{- end }}

{{/*
Frontend name/fullname -- deliberately distinct from team-manager.name/
fullname (not just a "-frontend" suffix applied ad hoc at each call site)
so every frontend resource's app.kubernetes.io/name differs from the
backend's. Without this, frontend pods sharing the backend's selectorLabels
would be silently caught by the backend's NetworkPolicy/PodDisruptionBudget/
Service selectors too (Kubernetes label selectors match on presence, not
exclusivity -- the same hazard pdb.yaml's component-exclusion comment
documents for the backup CronJob's pods) with no way to exclude them from
a Service selector at all (unlike matchExpressions-based selectors,
Service.spec.selector only supports flat label equality).
*/}}
{{- define "team-manager.frontend.name" -}}
{{- printf "%s-frontend" (include "team-manager.name" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "team-manager.frontend.fullname" -}}
{{- printf "%s-frontend" (include "team-manager.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Frontend selector labels -- see team-manager.frontend.name above for why
these must differ from team-manager.selectorLabels.
*/}}
{{- define "team-manager.frontend.selectorLabels" -}}
app.kubernetes.io/name: {{ include "team-manager.frontend.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Frontend common labels -- same shape as team-manager.labels, built on
team-manager.frontend.selectorLabels instead, versioned from
frontend.image.tag (not the backend's image.tag), and labeled
component: frontend rather than backend.
*/}}
{{- define "team-manager.frontend.labels" -}}
helm.sh/chart: {{ include "team-manager.chart" . }}
{{ include "team-manager.frontend.selectorLabels" . }}
{{- with .Values.frontend.image.tag | default .Chart.AppVersion }}
app.kubernetes.io/version: {{ . | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: frontend
{{- end }}

{{/*
Frontend image reference -- same digest-takes-precedence-over-tag logic as
team-manager.image, against frontend.image instead of image.
Usage: image: {{ include "team-manager.frontend.image" . }}
*/}}
{{- define "team-manager.frontend.image" -}}
{{- if .Values.frontend.image.digest -}}
"{{ .Values.frontend.image.repository }}@{{ .Values.frontend.image.digest }}"
{{- else -}}
"{{ .Values.frontend.image.repository }}:{{ .Values.frontend.image.tag | default .Chart.AppVersion }}"
{{- end -}}
{{- end }}

{{/*
Backup CronJob ServiceAccount name. Falls back to the main ServiceAccount
(team-manager.serviceAccountName) when backup.serviceAccount.create is false
and no name override is given, preserving prior behavior. Set
backup.serviceAccount.create=true (with its own annotations, e.g. an
IRSA role ARN scoped to only the backup bucket) to give the backup CronJob
its own identity instead of sharing the main Deployment's ServiceAccount --
without this, any IRSA annotation added to the shared account for S3 backup
access is also injected into every app pod.
*/}}
{{- define "team-manager.backupServiceAccountName" -}}
{{- if .Values.backup.serviceAccount.create }}
{{- default (printf "%s-backup" (include "team-manager.fullname" .)) .Values.backup.serviceAccount.name }}
{{- else if .Values.backup.serviceAccount.name }}
{{- .Values.backup.serviceAccount.name }}
{{- else }}
{{- include "team-manager.serviceAccountName" . }}
{{- end }}
{{- end }}
