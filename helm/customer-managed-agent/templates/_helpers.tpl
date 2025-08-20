{{/*
Expand the name of the chart.
*/}}
{{- define "customer-managed-deployment-agent.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "customer-managed-deployment-agent.fullname" -}}
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
Create the name of the image to use
*/}}
{{- define "customer-managed-deployment-agent.imageName" -}}
{{- .Values.image.registry }}/{{- .Values.image.repository }}:{{- .Values.image.tag | replace ":" "-" }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "customer-managed-deployment-agent.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "customer-managed-deployment-agent.labels" -}}
helm.sh/chart: {{ include "customer-managed-deployment-agent.chart" . }}
{{ include "customer-managed-deployment-agent.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "customer-managed-deployment-agent.selectorLabels" -}}
app.kubernetes.io/name: {{ include "customer-managed-deployment-agent.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "customer-managed-deployment-agent.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "customer-managed-deployment-agent.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
