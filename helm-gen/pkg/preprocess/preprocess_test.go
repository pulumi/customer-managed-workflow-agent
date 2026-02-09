package preprocess

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildNameMap(t *testing.T) {
	docs := []map[string]any{
		{
			"apiVersion": "v1",
			"kind":       "ServiceAccount",
			"metadata": map[string]any{
				"name": "workflow-agent-e17e131b",
				"annotations": map[string]any{
					"pulumi.com/autonamed": "true",
				},
			},
		},
		{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name": "agent-config",
			},
		},
	}

	nameMap := buildNameMap(docs)

	if nameMap["workflow-agent-e17e131b"] != "workflow-agent" {
		t.Errorf("expected workflow-agent, got %s", nameMap["workflow-agent-e17e131b"])
	}
	if _, ok := nameMap["agent-config"]; ok {
		t.Error("agent-config should not be in nameMap (no autonamed annotation)")
	}
}

func TestDisambiguateServiceAccounts(t *testing.T) {
	docs := []map[string]any{
		{
			"apiVersion": "v1",
			"kind":       "ServiceAccount",
			"metadata": map[string]any{
				"name": "workflow-agent-aaaa1111",
				"annotations": map[string]any{
					"pulumi.com/autonamed": "true",
				},
				"labels": map[string]any{
					"app.kubernetes.io/name": "customer-managed-workflow-agent",
				},
			},
		},
		{
			"apiVersion": "v1",
			"kind":       "ServiceAccount",
			"metadata": map[string]any{
				"name": "workflow-agent-bbbb2222",
				"annotations": map[string]any{
					"pulumi.com/autonamed": "true",
				},
			},
		},
	}

	nameMap := buildNameMap(docs)
	disambiguateServiceAccounts(docs, nameMap)

	if nameMap["workflow-agent-aaaa1111"] != "workflow-agent" {
		t.Errorf("agent SA should be workflow-agent, got %s", nameMap["workflow-agent-aaaa1111"])
	}
	if nameMap["workflow-agent-bbbb2222"] != "worker-service-account" {
		t.Errorf("worker SA should be worker-service-account, got %s", nameMap["workflow-agent-bbbb2222"])
	}
}

func TestApplyNameMap(t *testing.T) {
	doc := map[string]any{
		"apiVersion": "rbac.authorization.k8s.io/v1",
		"kind":       "RoleBinding",
		"metadata": map[string]any{
			"name": "workflow-agent-cccc3333",
		},
		"subjects": []any{
			map[string]any{
				"kind": "ServiceAccount",
				"name": "workflow-agent-aaaa1111",
			},
		},
		"roleRef": map[string]any{
			"kind": "Role",
			"name": "workflow-agent-dddd4444",
		},
	}

	nameMap := map[string]string{
		"workflow-agent-cccc3333": "workflow-agent",
		"workflow-agent-aaaa1111": "workflow-agent",
		"workflow-agent-dddd4444": "workflow-agent",
	}

	applyNameMap(doc, nameMap)

	name := getNestedStr(doc, "metadata", "name")
	if name != "workflow-agent" {
		t.Errorf("expected metadata.name workflow-agent, got %s", name)
	}

	subjects := doc["subjects"].([]any)
	subjectName := subjects[0].(map[string]any)["name"]
	if subjectName != "workflow-agent" {
		t.Errorf("expected subject name workflow-agent, got %s", subjectName)
	}

	roleRefName := doc["roleRef"].(map[string]any)["name"]
	if roleRefName != "workflow-agent" {
		t.Errorf("expected roleRef name workflow-agent, got %s", roleRefName)
	}
}

func TestRemovePulumiAnnotations(t *testing.T) {
	doc := map[string]any{
		"apiVersion": "v1",
		"kind":       "ServiceAccount",
		"metadata": map[string]any{
			"name": "test",
			"annotations": map[string]any{
				"pulumi.com/autonamed":  "true",
				"some-other-annotation": "keep",
			},
		},
	}

	removePulumiAnnotations(doc)

	annotations := getMap(doc["metadata"].(map[string]any), "annotations")
	if annotations == nil {
		t.Fatal("annotations should not be nil (non-pulumi annotation exists)")
	}
	if _, ok := annotations["pulumi.com/autonamed"]; ok {
		t.Error("pulumi.com/autonamed should be removed")
	}
	if annotations["some-other-annotation"] != "keep" {
		t.Error("non-pulumi annotations should be preserved")
	}
}

func TestRemovePulumiAnnotations_RemovesEmptyMap(t *testing.T) {
	doc := map[string]any{
		"apiVersion": "v1",
		"kind":       "ServiceAccount",
		"metadata": map[string]any{
			"name": "test",
			"annotations": map[string]any{
				"pulumi.com/autonamed": "true",
			},
		},
	}

	removePulumiAnnotations(doc)

	meta := doc["metadata"].(map[string]any)
	if _, ok := meta["annotations"]; ok {
		t.Error("empty annotations map should be removed")
	}
}

func TestNormalizeSecretEncoding(t *testing.T) {
	doc := map[string]any{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]any{
			"name": "agent-secret",
		},
		"data": map[string]any{
			"PULUMI_AGENT_TOKEN": "cGxhY2Vob2xkZXItdG9rZW4=", // base64 of "placeholder-token"
		},
	}

	normalizeSecretEncoding(doc)

	if _, ok := doc["data"]; ok {
		t.Error("data field should be removed")
	}

	stringData := getMap(doc, "stringData")
	if stringData == nil {
		t.Fatal("stringData should exist")
	}
	if stringData["PULUMI_AGENT_TOKEN"] != "placeholder-token" {
		t.Errorf("expected decoded value, got %s", stringData["PULUMI_AGENT_TOKEN"])
	}
}

func TestNormalizeSecretEncoding_SkipsNonSecret(t *testing.T) {
	doc := map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"data": map[string]any{
			"key": "value",
		},
	}

	normalizeSecretEncoding(doc)

	if _, ok := doc["data"]; !ok {
		t.Error("data field should not be removed for non-Secret resources")
	}
}

func TestRun(t *testing.T) {
	dir := t.TempDir()

	yamlContent := `apiVersion: v1
kind: ServiceAccount
metadata:
  name: workflow-agent-aaaa1111
  annotations:
    pulumi.com/autonamed: "true"
  labels:
    app.kubernetes.io/name: customer-managed-workflow-agent
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: workflow-agent-bbbb2222
  annotations:
    pulumi.com/autonamed: "true"
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: agent-config
  labels:
    app.kubernetes.io/name: customer-managed-workflow-agent
data:
  PULUMI_AGENT_SERVICE_URL: "https://api.pulumi.com"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: workflow-agent-pool
  labels:
    app.kubernetes.io/name: customer-managed-workflow-agent
spec:
  template:
    spec:
      serviceAccountName: workflow-agent-aaaa1111
      containers:
        - name: agent
          env:
            - name: PULUMI_AGENT_SERVICE_ACCOUNT_NAME
              value: workflow-agent-bbbb2222
`

	if err := os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Run(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Resources) != 4 {
		t.Fatalf("expected 4 resources, got %d", len(result.Resources))
	}

	// Verify agent SA kept its name
	if result.Resources[0].Name != "workflow-agent" {
		t.Errorf("agent SA should be workflow-agent, got %s", result.Resources[0].Name)
	}

	// Verify worker SA was disambiguated
	if result.Resources[1].Name != "worker-service-account" {
		t.Errorf("worker SA should be worker-service-account, got %s", result.Resources[1].Name)
	}

	// Verify deployment references were updated
	if result.YAML == "" {
		t.Error("YAML output should not be empty")
	}
}
