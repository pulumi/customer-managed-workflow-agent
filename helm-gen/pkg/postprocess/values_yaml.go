package postprocess

import (
	"path/filepath"
)

func writeValuesYAML(opts Options) error {
	content := `replicaCount: 1

image:
  registry: ""  # e.g., "my-registry.example.com"
  repository: pulumi/customer-managed-workflow-agent
  pullPolicy: IfNotPresent
  tag: ""  # defaults to appVersion

imagePullSecrets: []

nameOverride: ""
fullnameOverride: ""

agent:
  serviceUrl: "https://api.pulumi.com"
  token: ""                          # required - Pulumi agent access token
  existingSecretName: ""             # use existing secret instead
  deployTarget: "kubernetes"
  sharedVolumeDirectory: "/mnt/work"
  numCpus: ""
  memQuantity: ""
  extraEnvVars: []

workerServiceAccount:
  create: true
  annotations: {}
  name: ""  # defaults to "{fullname}-worker"

serviceAccount:
  create: true
  annotations: {}
  name: ""

rbac:
  create: true

podTemplate:
  workerPod: "{}"

service:
  type: ClusterIP
  port: 8080
  prometheus:
    scrape: "true"
    path: /healthz

serviceMonitor:
  enabled: false
  interval: "30s"
  path: /healthz

livenessProbe:
  enabled: false
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  enabled: false
  initialDelaySeconds: 5
  periodSeconds: 10

podSecurityContext: {}
securityContext: {}
resources: {}
nodeSelector: {}
tolerations: []
affinity: {}
initContainers: []
sidecars: []
podAnnotations: {}
podLabels: {}
deploymentStrategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: "25%"
    maxUnavailable: "25%"
terminationGracePeriodSeconds: 300
`

	return writeFile(filepath.Join(opts.OutputDir, "values.yaml"), content)
}
