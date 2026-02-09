package postprocess

import (
	"strings"
)

// processDeploymentTemplate enhances a helmify-generated deployment template.
func processDeploymentTemplate(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		processed := processDeploymentLine(line)
		result = append(result, processed...)
	}

	return strings.Join(result, "\n")
}

func processDeploymentLine(line string) []string {
	trimmed := strings.TrimSpace(line)

	// Replace hardcoded replicas
	if strings.Contains(trimmed, "replicas:") && !strings.Contains(trimmed, "{{") {
		indent := getIndent(line)
		return []string{indent + "replicas: {{ .Values.replicaCount }}"}
	}

	// Replace hardcoded image
	if strings.Contains(trimmed, "image:") && !strings.Contains(trimmed, "{{") && !strings.Contains(trimmed, "PULUMI_AGENT_IMAGE") {
		indent := getIndent(line)
		return []string{indent + `image: {{ include "chart.imageName" . | quote }}`}
	}

	// Replace hardcoded imagePullPolicy
	if strings.Contains(trimmed, "imagePullPolicy:") && !strings.Contains(trimmed, "{{") {
		indent := getIndent(line)
		return []string{indent + "imagePullPolicy: {{ .Values.image.pullPolicy }}"}
	}

	// Replace hardcoded serviceAccountName
	if strings.Contains(trimmed, "serviceAccountName:") && !strings.Contains(trimmed, "{{") {
		indent := getIndent(line)
		return []string{indent + `serviceAccountName: {{ include "chart.serviceAccountName" . }}`}
	}

	return []string{line}
}

// processConfigMapTemplate enhances a helmify-generated configmap template.
func processConfigMapTemplate(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		result = append(result, processConfigMapLine(line)...)
	}
	return strings.Join(result, "\n")
}

func processConfigMapLine(line string) []string {
	trimmed := strings.TrimSpace(line)

	if strings.Contains(trimmed, "PULUMI_AGENT_SERVICE_URL:") && !strings.Contains(trimmed, "{{") {
		indent := getIndent(line)
		return []string{indent + "PULUMI_AGENT_SERVICE_URL: {{ .Values.agent.serviceUrl | quote }}"}
	}

	if strings.Contains(trimmed, "PULUMI_AGENT_IMAGE:") && !strings.Contains(trimmed, "{{") {
		indent := getIndent(line)
		return []string{indent + `PULUMI_AGENT_IMAGE: {{ include "chart.imageName" . | quote }}`}
	}

	if strings.Contains(trimmed, "PULUMI_AGENT_IMAGE_PULL_POLICY:") && !strings.Contains(trimmed, "{{") {
		indent := getIndent(line)
		return []string{indent + "PULUMI_AGENT_IMAGE_PULL_POLICY: {{ .Values.image.pullPolicy | quote }}"}
	}

	if strings.Contains(trimmed, "worker-pod.json:") && !strings.Contains(trimmed, "{{") {
		indent := getIndent(line)
		return []string{indent + "worker-pod.json: {{ .Values.podTemplate.workerPod | quote }}"}
	}

	return []string{line}
}

// processServiceAccountTemplate wraps a ServiceAccount template with a conditional guard.
func processServiceAccountTemplate(content string) string {
	return wrapWithGuard(content, `{{- if .Values.serviceAccount.create }}`)
}

func getIndent(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// generateDeploymentTemplate creates the deployment template from scratch.
func generateDeploymentTemplate() string {
	return `{{ include "chart.validateConfig" . }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "chart.fullname" . }}-pool
  labels:
    {{- include "chart.labels" . | nindent 4 }}
  annotations:
    app.kubernetes.io/name: pulumi-workflow-agent-pool
    {{- with .Values.podAnnotations }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
spec:
  replicas: {{ .Values.replicaCount }}
  strategy:
    type: {{ .Values.deploymentStrategy.type }}
    {{- if eq .Values.deploymentStrategy.type "RollingUpdate" }}
    rollingUpdate:
      maxSurge: {{ .Values.deploymentStrategy.rollingUpdate.maxSurge }}
      maxUnavailable: {{ .Values.deploymentStrategy.rollingUpdate.maxUnavailable }}
    {{- end }}
  selector:
    matchLabels:
      {{- include "chart.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "chart.selectorLabels" . | nindent 8 }}
        {{- with .Values.podLabels }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      {{- with .Values.podAnnotations }}
      annotations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
    spec:
      serviceAccountName: {{ include "chart.serviceAccountName" . }}
      terminationGracePeriodSeconds: {{ .Values.terminationGracePeriodSeconds }}
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.podSecurityContext }}
      securityContext:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.initContainers }}
      initContainers:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      containers:
        - name: agent
          image: {{ include "chart.imageName" . | quote }}
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          {{- with .Values.securityContext }}
          securityContext:
            {{- toYaml . | nindent 12 }}
          {{- end }}
          env:
            - name: PULUMI_AGENT_DEPLOY_TARGET
              value: {{ .Values.agent.deployTarget | quote }}
            - name: PULUMI_AGENT_SHARED_VOLUME_DIRECTORY
              value: {{ .Values.agent.sharedVolumeDirectory | quote }}
            - name: PULUMI_AGENT_SERVICE_URL
              valueFrom:
                configMapKeyRef:
                  name: {{ include "chart.configMapName" . }}
                  key: PULUMI_AGENT_SERVICE_URL
            - name: PULUMI_AGENT_IMAGE
              valueFrom:
                configMapKeyRef:
                  name: {{ include "chart.configMapName" . }}
                  key: PULUMI_AGENT_IMAGE
            - name: PULUMI_AGENT_IMAGE_PULL_POLICY
              valueFrom:
                configMapKeyRef:
                  name: {{ include "chart.configMapName" . }}
                  key: PULUMI_AGENT_IMAGE_PULL_POLICY
            {{- if or .Values.agent.token .Values.agent.existingSecretName }}
            - name: PULUMI_AGENT_TOKEN
              valueFrom:
                secretKeyRef:
                  name: {{ include "chart.secretName" . }}
                  key: PULUMI_AGENT_TOKEN
            {{- end }}
            - name: PULUMI_AGENT_SERVICE_ACCOUNT_NAME
              value: {{ include "chart.workerServiceAccountName" . | quote }}
            {{- if .Values.agent.numCpus }}
            - name: PULUMI_AGENT_NUM_CPUS
              value: {{ .Values.agent.numCpus | quote }}
            {{- end }}
            {{- if .Values.agent.memQuantity }}
            - name: PULUMI_AGENT_MEM_QUANTITY
              value: {{ .Values.agent.memQuantity | quote }}
            {{- end }}
            {{- with .Values.agent.extraEnvVars }}
            {{- toYaml . | nindent 12 }}
            {{- end }}
          ports:
            - name: http
              containerPort: 8080
              protocol: TCP
          {{- if .Values.livenessProbe.enabled }}
          livenessProbe:
            httpGet:
              path: /healthz
              port: http
            initialDelaySeconds: {{ .Values.livenessProbe.initialDelaySeconds }}
            periodSeconds: {{ .Values.livenessProbe.periodSeconds }}
          {{- end }}
          {{- if .Values.readinessProbe.enabled }}
          readinessProbe:
            httpGet:
              path: /healthz
              port: http
            initialDelaySeconds: {{ .Values.readinessProbe.initialDelaySeconds }}
            periodSeconds: {{ .Values.readinessProbe.periodSeconds }}
          {{- end }}
          volumeMounts:
            - name: agent-work
              mountPath: /mnt/work
            - name: agent-config
              mountPath: /mnt/worker-pod.json
              subPath: worker-pod.json
              readOnly: true
          {{- with .Values.resources }}
          resources:
            {{- toYaml . | nindent 12 }}
          {{- end }}
        {{- with .Values.sidecars }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      volumes:
        - name: agent-work
          emptyDir: {}
        - name: agent-config
          configMap:
            name: {{ include "chart.configMapName" . }}
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
`
}

// generateConfigMapTemplate creates the configmap template from scratch.
func generateConfigMapTemplate() string {
	return `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "chart.configMapName" . }}
  labels:
    {{- include "chart.labels" . | nindent 4 }}
data:
  PULUMI_AGENT_SERVICE_URL: {{ .Values.agent.serviceUrl | quote }}
  PULUMI_AGENT_IMAGE: {{ include "chart.imageName" . | quote }}
  PULUMI_AGENT_IMAGE_PULL_POLICY: {{ .Values.image.pullPolicy | quote }}
  worker-pod.json: {{ .Values.podTemplate.workerPod | quote }}
`
}

// generateSecretTemplate creates the secret template from scratch.
func generateSecretTemplate() string {
	return `{{- if and .Values.agent.token (not .Values.agent.existingSecretName) }}
apiVersion: v1
kind: Secret
metadata:
  name: {{ include "chart.secretName" . }}
  labels:
    {{- include "chart.labels" . | nindent 4 }}
type: Opaque
stringData:
  PULUMI_AGENT_TOKEN: {{ .Values.agent.token | quote }}
{{- end }}
`
}

// generateServiceTemplate creates the service template from scratch.
func generateServiceTemplate() string {
	return `apiVersion: v1
kind: Service
metadata:
  name: {{ include "chart.fullname" . }}-service
  labels:
    {{- include "chart.labels" . | nindent 4 }}
    app.kubernetes.io/component: metrics
  annotations:
    prometheus.io/scrape: {{ .Values.service.prometheus.scrape | quote }}
    prometheus.io/port: {{ .Values.service.port | quote }}
    prometheus.io/path: {{ .Values.service.prometheus.path | quote }}
spec:
  type: {{ .Values.service.type }}
  selector:
    {{- include "chart.selectorLabels" . | nindent 4 }}
  ports:
    - name: http
      port: {{ .Values.service.port }}
      targetPort: 8080
      protocol: TCP
`
}

// generateServiceAccountTemplate creates the service account template from scratch.
func generateServiceAccountTemplate() string {
	return `{{- if .Values.serviceAccount.create }}
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ include "chart.serviceAccountName" . }}
  labels:
    {{- include "chart.labels" . | nindent 4 }}
  {{- with .Values.serviceAccount.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
{{- end }}
`
}

// generateRoleTemplate creates the role template from scratch.
func generateRoleTemplate() string {
	return `{{- if .Values.rbac.create }}
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: {{ include "chart.fullname" . }}
  labels:
    {{- include "chart.labels" . | nindent 4 }}
rules:
  - apiGroups:
      - ""
    resources:
      - pods
      - pods/log
      - configmaps
    verbs:
      - create
      - get
      - list
      - watch
      - update
      - delete
{{- end }}
`
}

// generateRoleBindingTemplate creates the rolebinding template from scratch.
func generateRoleBindingTemplate() string {
	return `{{- if .Values.rbac.create }}
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: {{ include "chart.fullname" . }}
  labels:
    {{- include "chart.labels" . | nindent 4 }}
subjects:
  - kind: ServiceAccount
    name: {{ include "chart.serviceAccountName" . }}
    namespace: {{ .Release.Namespace }}
roleRef:
  kind: Role
  name: {{ include "chart.fullname" . }}
  apiGroup: rbac.authorization.k8s.io
{{- end }}
`
}

// generateWorkerServiceAccountTemplate creates the worker service account template from scratch.
func generateWorkerServiceAccountTemplate() string {
	return `{{- if .Values.workerServiceAccount.create }}
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ include "chart.workerServiceAccountName" . }}
  labels:
    {{- include "chart.labels" . | nindent 4 }}
  {{- with .Values.workerServiceAccount.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
{{- end }}
`
}

// generateServiceMonitorTemplate creates the service monitor template from scratch.
func generateServiceMonitorTemplate() string {
	return `{{- if .Values.serviceMonitor.enabled }}
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: {{ include "chart.fullname" . }}-monitor
  labels:
    {{- include "chart.labels" . | nindent 4 }}
spec:
  selector:
    matchLabels:
      {{- include "chart.selectorLabels" . | nindent 6 }}
      app.kubernetes.io/component: metrics
  endpoints:
    - port: http
      path: {{ .Values.serviceMonitor.path }}
      interval: {{ .Values.serviceMonitor.interval }}
{{- end }}
`
}
