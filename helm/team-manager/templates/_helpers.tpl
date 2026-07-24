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
Resolves whether a <domain>.secret block (e.g. .Values.database.secret) is
"active" -- either a Secret this chart should create, or an
externally-managed Secret to reference -- and, if so, the Secret name to
use in secretKeyRef entries. Chart-managed (create: true) takes precedence
over existingSecret when both happen to be set. Returns an empty string
when neither is set, signaling callers to omit that area's secretKeyRef
entries entirely (config.Load() then fails loudly at startup for areas it
requires, matching prior behavior from when the single top-level
existingSecret was left unset).
Usage: {{ include "team-manager.secretName" (list $ "database" .Values.database.secret) }}
*/}}
{{- define "team-manager.secretName" -}}
{{- $root := index . 0 -}}
{{- $area := index . 1 -}}
{{- $secret := index . 2 -}}
{{- if $secret.create -}}
{{- printf "%s-%s" (include "team-manager.fullname" $root) $area -}}
{{- else -}}
{{- $secret.existingSecret -}}
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
