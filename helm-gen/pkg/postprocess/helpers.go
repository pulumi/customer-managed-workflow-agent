package postprocess

import (
	"path/filepath"
)

func writeHelpers(opts Options) error {
	content := `{{/*
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
`

	return writeFile(filepath.Join(opts.OutputDir, "templates", "_helpers.tpl"), content)
}

func writeNotes(opts Options) error {
	content := `Thank you for installing {{ .Chart.Name }}.

Your release is named {{ .Release.Name }}.

To check the status of the deployment:

  kubectl get deployments -n {{ .Release.Namespace }} -l "app.kubernetes.io/instance={{ .Release.Name }}"

To check the agent pods:

  kubectl get pods -n {{ .Release.Namespace }} -l "app.kubernetes.io/instance={{ .Release.Name }}"

To view agent logs:

  kubectl logs -n {{ .Release.Namespace }} -l "app.kubernetes.io/name=customer-managed-workflow-agent" -f

{{- if not .Values.agent.token }}
{{- if not .Values.agent.existingSecretName }}

WARNING: No agent token is configured. Set agent.token or agent.existingSecretName in your values.
{{- end }}
{{- end }}

{{- if .Values.workerServiceAccount.create }}

A worker ServiceAccount ({{ include "chart.workerServiceAccountName" . }}) has been created for worker pods.
To use cloud IAM (e.g., AWS IRSA), add annotations via workerServiceAccount.annotations.
{{- end }}

For more information, visit:
  https://github.com/pulumi/customer-managed-workflow-agent
`

	return writeFile(filepath.Join(opts.OutputDir, "templates", "NOTES.txt"), content)
}

func writeHelmignore(opts Options) error {
	content := `.DS_Store
.git/
.gitignore
.bzr/
.bzrignore
.hg/
.hgignore
.svn/
*.swp
*.bak
*.tmp
*.orig
*~
.project
.idea/
*.tmproj
.vscode/
`

	return writeFile(filepath.Join(opts.OutputDir, ".helmignore"), content)
}
