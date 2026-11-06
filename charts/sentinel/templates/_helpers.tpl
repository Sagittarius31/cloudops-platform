{{- define "sentinel.name" -}}sentinel{{- end -}}

{{- define "sentinel.fullname" -}}
{{- if contains (include "sentinel.name" .) .Release.Name -}}
{{- .Release.Name | trunc 50 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name (include "sentinel.name" .) | trunc 50 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "sentinel.labels" -}}
app.kubernetes.io/name: {{ include "sentinel.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}

{{- define "sentinel.selectorLabels" -}}
app.kubernetes.io/name: {{ include "sentinel.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/* Name of the Secret holding DATABASE_URL and API_TOKENS. Fails fast on an invalid mode combination. */}}
{{- define "sentinel.appSecretName" -}}
{{- if and .Values.postgres.enabled .Values.externalSecrets.enabled -}}
{{- fail "postgres.enabled and externalSecrets.enabled cannot both be true" -}}
{{- end -}}
{{- if or .Values.postgres.enabled .Values.externalSecrets.enabled -}}
{{ include "sentinel.fullname" . }}-app
{{- else -}}
{{ required "set database.existingSecret (keys DATABASE_URL, API_TOKENS) when postgres and externalSecrets are both disabled" .Values.database.existingSecret }}
{{- end -}}
{{- end -}}
