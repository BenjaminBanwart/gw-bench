{{/*
Common labels
*/}}
{{- define "gw-bench.labels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- end }}

{{/*
Backend labels
*/}}
{{- define "gw-bench.backendLabels" -}}
{{ include "gw-bench.labels" . }}
app.kubernetes.io/component: backend
{{- end }}

{{/*
Runner labels
*/}}
{{- define "gw-bench.runnerLabels" -}}
{{ include "gw-bench.labels" . }}
app.kubernetes.io/component: runner
{{- end }}

{{/*
Backend selector labels
*/}}
{{- define "gw-bench.backendSelectorLabels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: backend
{{- end }}

{{/*
Backend image
*/}}
{{- define "gw-bench.backendImage" -}}
{{- $tag := .Values.backend.image.tag | default .Chart.AppVersion -}}
{{ printf "%s:%s" .Values.backend.image.repository $tag }}
{{- end }}

{{/*
Runner image
*/}}
{{- define "gw-bench.runnerImage" -}}
{{- $tag := .Values.runner.image.tag | default .Chart.AppVersion -}}
{{ printf "%s:%s" .Values.runner.image.repository $tag }}
{{- end }}

{{/*
Namespace
*/}}
{{- define "gw-bench.namespace" -}}
{{ .Release.Namespace }}
{{- end }}

{{/*
ServiceAccount name
*/}}
{{- define "gw-bench.serviceAccountName" -}}
{{ printf "%s-runner" .Release.Name }}
{{- end }}
