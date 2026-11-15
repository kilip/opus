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
{{- $baseUrl := or .Values.baseUrl (.Values.config | default dict).baseUrl -}}
{{- if and (not $baseUrl) .Values.auth -}}
  {{- $baseUrl = .Values.auth.baseUrl -}}
{{- end -}}
{{- if $baseUrl }}
- name: BASE_URL
  value: {{ $baseUrl | quote }}
{{- end }}
{{- $smtp := or .Values.smtp (.Values.auth | default dict).smtp | default dict -}}
{{- if $smtp.host }}
- name: SMTP_HOST
  value: {{ $smtp.host | quote }}
{{- end }}
{{- if $smtp.port }}
- name: SMTP_PORT
  value: {{ $smtp.port | quote }}
{{- end }}
{{- if $smtp.username }}
- name: SMTP_USERNAME
  value: {{ $smtp.username | quote }}
{{- end }}
{{- if $smtp.from }}
- name: SMTP_FROM
  value: {{ $smtp.from | quote }}
{{- end }}
{{- $githubClientId := "" -}}
{{- $googleClientId := "" -}}
{{- $facebookClientId := "" -}}
{{- with .Values.oauth -}}
  {{- if .github -}}
    {{- if kindIs "map" .github -}}
      {{- $githubClientId = .github.clientId -}}
    {{- else -}}
      {{- $githubClientId = .github -}}
    {{- end -}}
  {{- end -}}
  {{- if not $githubClientId -}}
    {{- $githubClientId = .githubClientId -}}
  {{- end -}}
  {{- if .google -}}
    {{- if kindIs "map" .google -}}
      {{- $googleClientId = .google.clientId -}}
    {{- else -}}
      {{- $googleClientId = .google -}}
    {{- end -}}
  {{- end -}}
  {{- if not $googleClientId -}}
    {{- $googleClientId = .googleClientId -}}
  {{- end -}}
  {{- if .facebook -}}
    {{- if kindIs "map" .facebook -}}
      {{- $facebookClientId = .facebook.clientId -}}
    {{- else -}}
      {{- $facebookClientId = .facebook -}}
    {{- end -}}
  {{- end -}}
  {{- if not $facebookClientId -}}
    {{- $facebookClientId = .facebookClientId -}}
  {{- end -}}
{{- end -}}
{{- if and (not $githubClientId) .Values.auth -}}
  {{- with .Values.auth -}}
    {{- if .github -}}
      {{- if kindIs "map" .github -}}
        {{- $githubClientId = .github.clientId -}}
      {{- else -}}
        {{- $githubClientId = .github -}}
      {{- end -}}
    {{- end -}}
    {{- if not $githubClientId -}}
      {{- $githubClientId = .githubClientId -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- if and (not $googleClientId) .Values.auth -}}
  {{- with .Values.auth -}}
    {{- if .google -}}
      {{- if kindIs "map" .google -}}
        {{- $googleClientId = .google.clientId -}}
      {{- else -}}
        {{- $googleClientId = .google -}}
      {{- end -}}
    {{- end -}}
    {{- if not $googleClientId -}}
      {{- $googleClientId = .googleClientId -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- if and (not $facebookClientId) .Values.auth -}}
  {{- with .Values.auth -}}
    {{- if .facebook -}}
      {{- if kindIs "map" .facebook -}}
        {{- $facebookClientId = .facebook.clientId -}}
      {{- else -}}
        {{- $facebookClientId = .facebook -}}
      {{- end -}}
    {{- end -}}
    {{- if not $facebookClientId -}}
      {{- $facebookClientId = .facebookClientId -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- if $githubClientId }}
- name: GITHUB_CLIENT_ID
  value: {{ $githubClientId | quote }}
{{- end }}
{{- if $googleClientId }}
- name: GOOGLE_CLIENT_ID
  value: {{ $googleClientId | quote }}
{{- end }}
{{- if $facebookClientId }}
- name: FACEBOOK_CLIENT_ID
  value: {{ $facebookClientId | quote }}
{{- end }}
{{- if and .Values.auth .Values.auth.existingSecret }}
- name: SESSION_SECRET
  valueFrom:
    secretKeyRef:
      name: {{ .Values.auth.existingSecret }}
      key: SESSION_SECRET
      optional: true
- name: SMTP_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ .Values.auth.existingSecret }}
      key: SMTP_PASSWORD
      optional: true
- name: GITHUB_CLIENT_SECRET
  valueFrom:
    secretKeyRef:
      name: {{ .Values.auth.existingSecret }}
      key: GITHUB_CLIENT_SECRET
      optional: true
- name: GOOGLE_CLIENT_SECRET
  valueFrom:
    secretKeyRef:
      name: {{ .Values.auth.existingSecret }}
      key: GOOGLE_CLIENT_SECRET
      optional: true
- name: FACEBOOK_CLIENT_SECRET
  valueFrom:
    secretKeyRef:
      name: {{ .Values.auth.existingSecret }}
      key: FACEBOOK_CLIENT_SECRET
      optional: true
{{- end }}
{{- with .Values.extraEnv }}
{{ toYaml . }}
{{- end }}
{{- end }}
