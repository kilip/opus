{{/*
Chart name, truncated to DNS label limit.
*/}}
{{- define "opus.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Fully qualified release name used as resource name prefix.
*/}}
{{- define "opus.fullname" -}}
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

{{- define "opus.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "opus.labels" -}}
helm.sh/chart: {{ include "opus.chart" . }}
{{ include "opus.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: opus
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "opus.selectorLabels" -}}
app.kubernetes.io/name: {{ include "opus.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "opus.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "opus.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Container image reference.
*/}}
{{- define "opus.image" -}}
{{- $tag := default .Chart.AppVersion .Values.image.tag }}
{{- if .Values.global.imageRegistry }}
{{- printf "%s/%s:%s" .Values.global.imageRegistry .Values.image.repository $tag }}
{{- else }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}
{{- end }}

{{/*
Environment variables from values.config.
*/}}
{{- define "opus.env" -}}
{{- $c := .Values.config -}}
- name: APP_NAME
  value: {{ $c.appName | quote }}
- name: DATABASE_URL
  value: {{ printf "file:%s/opus.db?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)" (default "/data" .Values.persistence.mountPath) | quote }}
- name: HOST
  value: {{ $c.host | quote }}
- name: PORT
  value: {{ .Values.port | quote }}
- name: LOG_LEVEL
  value: {{ $c.logLevel | quote }}
- name: READ_TIMEOUT
  value: {{ $c.readTimeout | quote }}
- name: WRITE_TIMEOUT
  value: {{ $c.writeTimeout | quote }}
- name: IDLE_TIMEOUT
  value: {{ $c.idleTimeout | quote }}
- name: SHUTDOWN_TIMEOUT
  value: {{ $c.shutdownTimeout | quote }}
- name: BODY_LIMIT
  value: {{ $c.bodyLimit | quote }}
{{- with .Values.extraEnv }}
{{ toYaml . }}
{{- end }}
{{- end }}
