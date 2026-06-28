{{/*
Expand the name of the chart.
*/}}
{{- define "expense-tracker.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "expense-tracker.fullname" -}}
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

{{- define "expense-tracker.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "expense-tracker.labels" -}}
helm.sh/chart: {{ include "expense-tracker.chart" . }}
{{ include "expense-tracker.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "expense-tracker.selectorLabels" -}}
app.kubernetes.io/name: {{ include "expense-tracker.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "expense-tracker.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "expense-tracker.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/* Secret name for DB / bootstrap */}}
{{- define "expense-tracker.secretName" -}}
{{- if .Values.secret.existingSecret }}
{{- .Values.secret.existingSecret }}
{{- else }}
{{- include "expense-tracker.fullname" . }}-secrets
{{- end }}
{{- end }}

{{/*
Database URL: explicit secret.databaseUrl, or derived from Bitnami postgresql subchart.
*/}}
{{- define "expense-tracker.databaseUrl" -}}
{{- if .Values.secret.databaseUrl }}
{{- .Values.secret.databaseUrl }}
{{- else if .Values.postgresql.enabled }}
{{- $pgUser := .Values.postgresql.auth.username | default "expense" }}
{{- $pgPass := .Values.postgresql.auth.password | default "expense" }}
{{- $pgDB := .Values.postgresql.auth.database | default "expense_tracker" }}
{{- $pgHost := printf "%s-postgresql" .Release.Name }}
{{- printf "postgres://%s:%s@%s:5432/%s?sslmode=disable" $pgUser $pgPass $pgHost $pgDB }}
{{- else }}
{{- fail "Set secret.databaseUrl or enable postgresql.enabled" }}
{{- end }}
{{- end }}

{{- define "expense-tracker.migrateDatabaseUrl" -}}
{{- if .Values.secret.migrateDatabaseUrl }}
{{- .Values.secret.migrateDatabaseUrl }}
{{- else }}
{{- include "expense-tracker.databaseUrl" . }}
{{- end }}
{{- end }}
