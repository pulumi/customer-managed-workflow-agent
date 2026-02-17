package pipeline

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunWithoutHelmify(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "chart")

	yamlContent := `apiVersion: v1
kind: Namespace
metadata:
  name: helm-namespace
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: workflow-agent-aaaa1111
  namespace: helm-namespace
  annotations:
    pulumi.com/autonamed: "true"
  labels:
    app.kubernetes.io/name: customer-managed-workflow-agent
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: workflow-agent-bbbb2222
  namespace: helm-namespace
  annotations:
    pulumi.com/autonamed: "true"
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: agent-config
  namespace: helm-namespace
  labels:
    app.kubernetes.io/name: customer-managed-workflow-agent
data:
  PULUMI_AGENT_SERVICE_URL: "https://api.pulumi.com"
  PULUMI_AGENT_IMAGE: "pulumi/customer-managed-workflow-agent:latest"
  PULUMI_AGENT_IMAGE_PULL_POLICY: IfNotPresent
  worker-pod.json: "{}"
---
apiVersion: v1
kind: Secret
metadata:
  name: agent-secret
  namespace: helm-namespace
data:
  PULUMI_AGENT_TOKEN: cGxhY2Vob2xkZXItdG9rZW4=
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: workflow-agent-dddd4444
  namespace: helm-namespace
  annotations:
    pulumi.com/autonamed: "true"
  labels:
    app.kubernetes.io/name: customer-managed-workflow-agent
rules:
  - apiGroups: [""]
    resources: ["pods", "pods/log", "configmaps"]
    verbs: ["create", "get", "list", "watch", "update", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: workflow-agent-eeee5555
  namespace: helm-namespace
  annotations:
    pulumi.com/autonamed: "true"
  labels:
    app.kubernetes.io/name: customer-managed-workflow-agent
subjects:
  - kind: ServiceAccount
    name: workflow-agent-aaaa1111
    namespace: helm-namespace
roleRef:
  kind: Role
  name: workflow-agent-dddd4444
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: workflow-agent-pool
  namespace: helm-namespace
  annotations:
    app.kubernetes.io/name: pulumi-workflow-agent-pool
  labels:
    app.kubernetes.io/name: customer-managed-workflow-agent
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: customer-managed-workflow-agent
  template:
    metadata:
      labels:
        app.kubernetes.io/name: customer-managed-workflow-agent
    spec:
      serviceAccountName: workflow-agent-aaaa1111
      containers:
        - name: agent
          image: "pulumi/customer-managed-workflow-agent:latest"
          imagePullPolicy: IfNotPresent
          env:
            - name: PULUMI_AGENT_SERVICE_ACCOUNT_NAME
              value: workflow-agent-bbbb2222
          ports:
            - name: http
              containerPort: 8080
              protocol: TCP
---
apiVersion: v1
kind: Service
metadata:
  name: deployment-agent-service
  namespace: helm-namespace
  labels:
    app.kubernetes.io/name: customer-managed-workflow-agent
    app.kubernetes.io/component: metrics
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
    prometheus.io/path: /healthz
spec:
  type: ClusterIP
  selector:
    app.kubernetes.io/name: customer-managed-workflow-agent
  ports:
    - name: http
      port: 8080
      targetPort: 8080
      protocol: TCP
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: deployment-agent-servicemonitor
  namespace: helm-namespace
  labels:
    app.kubernetes.io/name: customer-managed-workflow-agent
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: customer-managed-workflow-agent
      app.kubernetes.io/component: metrics
  endpoints:
    - port: http
      path: /healthz
      interval: "30s"
`

	if err := os.WriteFile(filepath.Join(inputDir, "manifest.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		InputDir:     inputDir,
		OutputDir:    outputDir,
		ChartName:    "pulumi-deployment-agent",
		ChartVersion: "0.1.0",
		AppVersion:   "2.1.0",
	}

	if err := Run(opts); err != nil {
		t.Fatal(err)
	}

	requiredFiles := []string{
		"Chart.yaml",
		"values.yaml",
		".helmignore",
		"templates/_helpers.tpl",
		"templates/NOTES.txt",
		"templates/deployment.yaml",
		"templates/configmap.yaml",
		"templates/secret.yaml",
		"templates/service.yaml",
		"templates/serviceaccount.yaml",
		"templates/worker-serviceaccount.yaml",
		"templates/role.yaml",
		"templates/rolebinding.yaml",
		"templates/servicemonitor.yaml",
	}

	for _, f := range requiredFiles {
		path := filepath.Join(outputDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("missing required file: %s", f)
		}
	}
}

func TestGoldenFiles(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	inputDir := filepath.Join(repoRoot, "testdata", "input")
	goldenDir := filepath.Join(repoRoot, "testdata", "golden")
	outputDir := filepath.Join(t.TempDir(), "chart")

	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		t.Skipf("testdata/input not found at %s", inputDir)
	}
	if _, err := os.Stat(goldenDir); os.IsNotExist(err) {
		t.Skipf("testdata/golden not found at %s", goldenDir)
	}

	opts := Options{
		InputDir:     inputDir,
		OutputDir:    outputDir,
		ChartName:    "pulumi-deployment-agent",
		ChartVersion: "0.1.0",
		AppVersion:   "2.1.0",
	}

	if err := Run(opts); err != nil {
		t.Fatal(err)
	}

	goldenFiles := []string{
		"Chart.yaml",
		"values.yaml",
		"templates/_helpers.tpl",
		"templates/deployment.yaml",
		"templates/configmap.yaml",
		"templates/secret.yaml",
		"templates/service.yaml",
		"templates/serviceaccount.yaml",
		"templates/worker-serviceaccount.yaml",
		"templates/role.yaml",
		"templates/rolebinding.yaml",
		"templates/servicemonitor.yaml",
	}

	for _, f := range goldenFiles {
		t.Run(f, func(t *testing.T) {
			goldenPath := filepath.Join(goldenDir, f)
			actualPath := filepath.Join(outputDir, f)

			golden, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("reading golden file %s: %v", goldenPath, err)
			}

			actual, err := os.ReadFile(actualPath)
			if err != nil {
				t.Fatalf("reading actual file %s: %v", actualPath, err)
			}

			goldenStr := strings.TrimSpace(string(golden))
			actualStr := strings.TrimSpace(string(actual))

			if goldenStr != actualStr {
				t.Errorf("output mismatch for %s\n\nExpected:\n%s\n\nGot:\n%s", f, goldenStr, actualStr)
			}
		})
	}
}
