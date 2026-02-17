package postprocess

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteChartYAML(t *testing.T) {
	dir := t.TempDir()
	opts := Options{
		OutputDir:    dir,
		ChartName:    "pulumi-deployment-agent",
		ChartVersion: "1.2.3",
		AppVersion:   "2.1.0",
	}

	if err := writeChartYAML(opts); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "Chart.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	assertions := map[string]string{
		"name":       "name: pulumi-deployment-agent",
		"version":    "version: 1.2.3",
		"appVersion": "appVersion: 2.1.0",
		"apiVersion": "apiVersion: v2",
		"type":       "type: application",
	}

	for name, expected := range assertions {
		if !strings.Contains(content, expected) {
			t.Errorf("Chart.yaml missing %s: expected to contain %q", name, expected)
		}
	}
}

func TestWriteValuesYAML(t *testing.T) {
	dir := t.TempDir()
	opts := Options{OutputDir: dir}

	if err := writeValuesYAML(opts); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "values.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	requiredFields := []string{
		"replicaCount:",
		"image:",
		"repository:",
		"agent:",
		"serviceUrl:",
		"token:",
		"serviceAccount:",
		"rbac:",
		"serviceMonitor:",
		"resources:",
		"nodeSelector:",
		"tolerations:",
		"affinity:",
	}

	for _, field := range requiredFields {
		if !strings.Contains(content, field) {
			t.Errorf("values.yaml missing field: %s", field)
		}
	}
}

func TestWriteHelpers(t *testing.T) {
	dir := t.TempDir()
	opts := Options{OutputDir: dir}

	if err := os.MkdirAll(filepath.Join(dir, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := writeHelpers(opts); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "templates", "_helpers.tpl"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	helpers := []string{
		`define "chart.name"`,
		`define "chart.fullname"`,
		`define "chart.chart"`,
		`define "chart.labels"`,
		`define "chart.selectorLabels"`,
		`define "chart.serviceAccountName"`,
		`define "chart.workerServiceAccountName"`,
		`define "chart.imageName"`,
		`define "chart.secretName"`,
		`define "chart.configMapName"`,
		`define "chart.validateConfig"`,
	}

	for _, helper := range helpers {
		if !strings.Contains(content, helper) {
			t.Errorf("_helpers.tpl missing helper: %s", helper)
		}
	}
}

func TestProcessDeploymentLine_Replicas(t *testing.T) {
	result := processDeploymentLine("  replicas: 3")
	if len(result) != 1 || !strings.Contains(result[0], ".Values.replicaCount") {
		t.Errorf("expected templated replicas, got: %v", result)
	}
}

func TestProcessDeploymentLine_Image(t *testing.T) {
	result := processDeploymentLine(`          image: "pulumi/customer-managed-workflow-agent:latest"`)
	if len(result) != 1 || !strings.Contains(result[0], `chart.imageName`) {
		t.Errorf("expected templated image, got: %v", result)
	}
}

func TestProcessDeploymentLine_ImagePullPolicy(t *testing.T) {
	result := processDeploymentLine("          imagePullPolicy: IfNotPresent")
	if len(result) != 1 || !strings.Contains(result[0], ".Values.image.pullPolicy") {
		t.Errorf("expected templated imagePullPolicy, got: %v", result)
	}
}

func TestProcessDeploymentLine_ServiceAccountName(t *testing.T) {
	result := processDeploymentLine("      serviceAccountName: workflow-agent")
	if len(result) != 1 || !strings.Contains(result[0], `chart.serviceAccountName`) {
		t.Errorf("expected templated serviceAccountName, got: %v", result)
	}
}

func TestProcessDeploymentLine_PassThrough(t *testing.T) {
	line := "      - name: agent"
	result := processDeploymentLine(line)
	if len(result) != 1 || result[0] != line {
		t.Errorf("expected pass-through, got: %v", result)
	}
}

func TestProcessConfigMapLine_ServiceURL(t *testing.T) {
	result := processConfigMapLine(`  PULUMI_AGENT_SERVICE_URL: "https://api.pulumi.com"`)
	if len(result) != 1 || !strings.Contains(result[0], ".Values.agent.serviceUrl") {
		t.Errorf("expected templated serviceUrl, got: %v", result)
	}
}

func TestProcessConfigMapLine_Image(t *testing.T) {
	result := processConfigMapLine(`  PULUMI_AGENT_IMAGE: "pulumi/customer-managed-workflow-agent:latest"`)
	if len(result) != 1 || !strings.Contains(result[0], "chart.imageName") {
		t.Errorf("expected templated image, got: %v", result)
	}
}

func TestWrapWithGuard(t *testing.T) {
	content := "apiVersion: v1\nkind: Secret\n"
	guard := "{{- if .Values.agent.token }}"

	result := wrapWithGuard(content, guard)

	if !strings.HasPrefix(result, guard) {
		t.Error("result should start with guard")
	}
	if !strings.HasSuffix(result, "{{- end }}\n") {
		t.Error("result should end with {{- end }}")
	}
}

func TestGenerateTemplates(t *testing.T) {
	templates := map[string]func() string{
		"deployment":            generateDeploymentTemplate,
		"configmap":             generateConfigMapTemplate,
		"secret":                generateSecretTemplate,
		"service":               generateServiceTemplate,
		"serviceaccount":        generateServiceAccountTemplate,
		"worker-serviceaccount": generateWorkerServiceAccountTemplate,
		"role":                  generateRoleTemplate,
		"rolebinding":           generateRoleBindingTemplate,
		"servicemonitor":        generateServiceMonitorTemplate,
	}

	for name, fn := range templates {
		t.Run(name, func(t *testing.T) {
			content := fn()
			if content == "" {
				t.Errorf("%s template is empty", name)
			}
			if !strings.Contains(content, "apiVersion:") {
				t.Errorf("%s template missing apiVersion", name)
			}
		})
	}
}

func TestWorkerServiceAccountTemplate(t *testing.T) {
	content := generateWorkerServiceAccountTemplate()
	if !strings.Contains(content, "workerServiceAccount.create") {
		t.Error("worker SA template missing create guard")
	}
	if !strings.Contains(content, "chart.workerServiceAccountName") {
		t.Error("worker SA template missing name helper")
	}
	if !strings.Contains(content, "workerServiceAccount.annotations") {
		t.Error("worker SA template missing annotations")
	}
}

func TestDeploymentTemplateHasValidation(t *testing.T) {
	content := generateDeploymentTemplate()
	if !strings.Contains(content, `chart.validateConfig`) {
		t.Error("deployment template missing validateConfig include")
	}
}

func TestDeploymentTemplateConditionalToken(t *testing.T) {
	content := generateDeploymentTemplate()
	if !strings.Contains(content, "if or .Values.agent.token .Values.agent.existingSecretName") {
		t.Error("deployment template missing conditional token reference")
	}
}

func TestDeploymentTemplateReadinessProbe(t *testing.T) {
	content := generateDeploymentTemplate()
	if !strings.Contains(content, "readinessProbe.enabled") {
		t.Error("deployment template missing readinessProbe")
	}
}

func TestDeploymentTemplateStrategy(t *testing.T) {
	content := generateDeploymentTemplate()
	if !strings.Contains(content, ".Values.deploymentStrategy.type") {
		t.Error("deployment template missing strategy type")
	}
	if !strings.Contains(content, "deploymentStrategy.rollingUpdate.maxSurge") {
		t.Error("deployment template missing rollingUpdate maxSurge")
	}
}

func TestImageRegistryHelper(t *testing.T) {
	dir := t.TempDir()
	opts := Options{OutputDir: dir}

	if err := os.MkdirAll(filepath.Join(dir, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := writeHelpers(opts); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "templates", "_helpers.tpl"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, ".Values.image.registry") {
		t.Error("_helpers.tpl missing image.registry handling")
	}
}

func TestTemplateProcessorMap(t *testing.T) {
	expectedNames := []string{
		"deployment.yaml",
		"configmap.yaml",
		"secret.yaml",
		"role.yaml",
		"rolebinding.yaml",
		"worker-serviceaccount.yaml",
		"serviceaccount.yaml",
		"servicemonitor.yaml",
		"service.yaml",
	}

	for _, name := range expectedNames {
		if _, ok := templateProcessors[name]; !ok {
			t.Errorf("templateProcessors map missing entry for %q", name)
		}
	}
}

func TestProcessTemplateFallback(t *testing.T) {
	content := "apiVersion: v1\nkind: ConfigMap\n"
	result := processTemplate("my-custom-configmap.yaml", content)
	if !strings.Contains(result, "apiVersion: v1") {
		t.Error("fallback processing failed for configmap variant")
	}
}

func TestValuesContainsNewFields(t *testing.T) {
	dir := t.TempDir()
	opts := Options{OutputDir: dir}

	if err := writeValuesYAML(opts); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "values.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	checks := []string{
		"workerServiceAccount:",
		"create: true",
		"readinessProbe:",
		"image:",
		"registry:",
		"deploymentStrategy:",
		"maxSurge:",
		"maxUnavailable:",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("values.yaml missing field: %s", check)
		}
	}
}

func TestRunPostProcess(t *testing.T) {
	outputDir := t.TempDir()
	inputDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(inputDir, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		InputDir:     inputDir,
		OutputDir:    outputDir,
		ChartName:    "test-chart",
		ChartVersion: "0.1.0",
		AppVersion:   "1.0.0",
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
	}

	for _, f := range requiredFiles {
		path := filepath.Join(outputDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("missing required file: %s", f)
		}
	}
}
