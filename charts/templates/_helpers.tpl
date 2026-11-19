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
Environment variables from values.
*/}}
{{- define "opus.env" -}}
{{- $c := .Values.config -}}
{{- $mountPath := default "/data" .Values.persistence.mountPath -}}
{{- $authSecretName := default (printf "%s-auth" (include "opus.fullname" .)) .Values.auth.existingSecret -}}
- name: HOME
  value: {{ $mountPath | quote }}
- name: APP_HOME
  value: {{ $mountPath | quote }}
- name: APP_NAME
  value: {{ $c.appName | quote }}
- name: APP_DATABASE_URL
  value: {{ printf "file:%s/opus.db?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)" $mountPath | quote }}
- name: APP_HOST
  value: {{ $c.host | quote }}
- name: APP_PORT
  value: {{ .Values.port | int64 | quote }}
- name: APP_LOG_LEVEL
  value: {{ $c.logLevel | quote }}
- name: APP_READ_TIMEOUT
  value: {{ $c.readTimeout | quote }}
- name: APP_WRITE_TIMEOUT
  value: {{ $c.writeTimeout | quote }}
- name: APP_IDLE_TIMEOUT
  value: {{ $c.idleTimeout | quote }}
- name: APP_SHUTDOWN_TIMEOUT
  value: {{ $c.shutdownTimeout | quote }}
- name: APP_BODY_LIMIT
  value: {{ $c.bodyLimit | int64 | quote }}
{{- if $c.sessionTtl }}
- name: AUTH_SESSION_TTL
  value: {{ $c.sessionTtl | quote }}
{{- end }}
{{- if $c.trustedOrigins }}
- name: APP_TRUSTED_ORIGINS
  value: {{ $c.trustedOrigins | quote }}
{{- end }}
{{- $baseUrl := $c.baseUrl -}}
{{- if and (not $baseUrl) .Values.ingress.enabled .Values.ingress.hosts -}}
  {{- $firstHost := (index .Values.ingress.hosts 0).host -}}
  {{- if $firstHost -}}
    {{- $scheme := ternary "https" "http" (gt (len .Values.ingress.tls) 0) -}}
    {{- $baseUrl = printf "%s://%s" $scheme $firstHost -}}
  {{- end -}}
{{- end -}}
{{- if $baseUrl }}
- name: APP_BASE_URL
  value: {{ $baseUrl | quote }}
{{- end }}
- name: AUTH_SECRET
  valueFrom:
    secretKeyRef:
      name: {{ $authSecretName }}
      key: AUTH_SECRET
{{- if .Values.github.existingSecret }}
- name: GH_TOKEN
  valueFrom:
    secretKeyRef:
      name: {{ .Values.github.existingSecret }}
      key: GH_TOKEN
{{- end }}
{{- if .Values.sentry.dsn }}
- name: APP_SENTRY_DSN
  value: {{ .Values.sentry.dsn | quote }}
{{- end }}
{{- if .Values.tracker.pollInterval }}
- name: TRACKER_POLL_INTERVAL
  value: {{ .Values.tracker.pollInterval | quote }}
{{- end }}
{{- if .Values.notify.telegram.chatId }}
- name: NOTIFY_TELEGRAM_CHAT_ID
  value: {{ .Values.notify.telegram.chatId | quote }}
- name: NOTIFY_TELEGRAM_BOT_TOKEN
  valueFrom:
    secretKeyRef:
      name: {{ $authSecretName }}
      key: NOTIFY_TELEGRAM_BOT_TOKEN
{{- end }}
{{- if .Values.smtp.host }}
- name: APP_SMTP_HOST
  value: {{ .Values.smtp.host | quote }}
{{- if .Values.smtp.port }}
- name: APP_SMTP_PORT
  value: {{ .Values.smtp.port | int64 | quote }}
{{- end }}
{{- if .Values.smtp.username }}
- name: APP_SMTP_USERNAME
  value: {{ .Values.smtp.username | quote }}
{{- end }}
{{- if .Values.smtp.from }}
- name: APP_SMTP_FROM
  value: {{ .Values.smtp.from | quote }}
{{- end }}
- name: APP_SMTP_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ $authSecretName }}
      key: APP_SMTP_PASSWORD
{{- end }}
{{- if .Values.oauth.github.clientId }}
- name: AUTH_GITHUB_ID
  value: {{ .Values.oauth.github.clientId | quote }}
- name: AUTH_GITHUB_SECRET
  valueFrom:
    secretKeyRef:
      name: {{ $authSecretName }}
      key: AUTH_GITHUB_SECRET
{{- end }}
{{- if .Values.oauth.google.clientId }}
- name: AUTH_GOOGLE_ID
  value: {{ .Values.oauth.google.clientId | quote }}
- name: AUTH_GOOGLE_SECRET
  valueFrom:
    secretKeyRef:
      name: {{ $authSecretName }}
      key: AUTH_GOOGLE_SECRET
{{- end }}
{{- if .Values.oauth.facebook.clientId }}
- name: AUTH_FACEBOOK_ID
  value: {{ .Values.oauth.facebook.clientId | quote }}
- name: AUTH_FACEBOOK_SECRET
  valueFrom:
    secretKeyRef:
      name: {{ $authSecretName }}
      key: AUTH_FACEBOOK_SECRET
{{- end }}
{{- with .Values.extraEnv }}
{{ toYaml . }}
{{- end }}
{{- end }}
