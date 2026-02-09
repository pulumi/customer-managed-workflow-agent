{{/*
Expand the name of the chart.
*/}}
{{- define "chart.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "chart.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "chart.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "chart.labels" -}}
helm.sh/chart: {{ include "chart.chart" . }}
{{ include "chart.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "chart.selectorLabels" -}}
app.kubernetes.io/name: customer-managed-workflow-agent
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "chart.serviceAccountName" -}}
{{- if .Values.serviceAccount.name }}
{{- .Values.serviceAccount.name }}
{{- else }}
{{- include "chart.fullname" . }}
{{- end }}
{{- end }}

{{/*
Create the worker service account name
*/}}
{{- define "chart.workerServiceAccountName" -}}
{{- if .Values.workerServiceAccount.name }}
{{- .Values.workerServiceAccount.name }}
{{- else }}
{{- printf "%s-worker" (include "chart.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Create the full image name with tag
*/}}
{{- define "chart.imageName" -}}
{{- if .Values.image.registry -}}
{{- printf "%s/%s:%s" .Values.image.registry .Values.image.repository (.Values.image.tag | default .Chart.AppVersion) }}
{{- else -}}
{{- printf "%s:%s" .Values.image.repository (.Values.image.tag | default .Chart.AppVersion) }}
{{- end -}}
{{- end }}

{{/*
Create the name of the secret to use
*/}}
{{- define "chart.secretName" -}}
{{- if .Values.agent.existingSecretName }}
{{- .Values.agent.existingSecretName }}
{{- else }}
{{- printf "%s-secret" (include "chart.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Create the name of the configmap
*/}}
{{- define "chart.configMapName" -}}
{{- printf "%s-config" (include "chart.fullname" .) }}
{{- end }}

{{/*
Validate required configuration
*/}}
{{- define "chart.validateConfig" -}}
{{- if and (not .Values.agent.token) (not .Values.agent.existingSecretName) -}}
{{- fail "Either agent.token or agent.existingSecretName must be set" -}}
{{- end -}}
{{- end -}}
