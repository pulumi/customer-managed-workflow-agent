package preprocess

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Resource represents a parsed Kubernetes resource with its raw YAML map.
type Resource struct {
	APIVersion string
	Kind       string
	Name       string
	doc        map[string]any
}

// Result holds the preprocessed resources ready for helmify consumption.
type Result struct {
	Resources []Resource
	YAML      string
}

// Run reads Pulumi rendered YAML files from inputDir, normalizes them,
// and returns the preprocessed result.
func Run(inputDir string) (*Result, error) {
	docs, err := readAllYAML(inputDir)
	if err != nil {
		return nil, fmt.Errorf("reading YAML from %s: %w", inputDir, err)
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("no Kubernetes resources found in %s", inputDir)
	}

	nameMap := buildNameMap(docs)
	disambiguateServiceAccounts(docs, nameMap)

	for i := range docs {
		applyNameMap(docs[i], nameMap)
		removePulumiAnnotations(docs[i])
		normalizeSecretEncoding(docs[i])
	}

	resources := make([]Resource, len(docs))
	for i, doc := range docs {
		resources[i] = Resource{
			APIVersion: getStr(doc, "apiVersion"),
			Kind:       getStr(doc, "kind"),
			Name:       getNestedStr(doc, "metadata", "name"),
			doc:        doc,
		}
	}

	yamlOut, err := marshalMultiDoc(docs)
	if err != nil {
		return nil, fmt.Errorf("marshaling output: %w", err)
	}

	return &Result{Resources: resources, YAML: yamlOut}, nil
}

// WriteToFile writes the preprocessed YAML to a file.
func (r *Result) WriteToFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(r.YAML), 0o644)
}

func readAllYAML(dir string) ([]map[string]any, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var docs []map[string]any
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		fileDocs, err := splitYAML(data)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}
		docs = append(docs, fileDocs...)
	}
	return docs, nil
}

func splitYAML(data []byte) ([]map[string]any, error) {
	var docs []map[string]any
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	for {
		var doc map[string]any
		err := decoder.Decode(&doc)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if doc == nil {
			continue
		}
		if _, ok := doc["apiVersion"]; !ok {
			continue
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

var hashSuffixRegexp = regexp.MustCompile(`-[0-9a-f]{8}$`)

// buildNameMap scans resources for Pulumi auto-named resources and builds
// a mapping from hashed name to clean name.
func buildNameMap(docs []map[string]any) map[string]string {
	nameMap := make(map[string]string)
	for _, doc := range docs {
		meta := getMap(doc, "metadata")
		if meta == nil {
			continue
		}
		name := getStr(meta, "name")
		if name == "" {
			continue
		}
		annotations := getMap(meta, "annotations")
		if annotations == nil {
			continue
		}
		if getStr(annotations, "pulumi.com/autonamed") == "true" {
			cleanName := hashSuffixRegexp.ReplaceAllString(name, "")
			if cleanName != name {
				nameMap[name] = cleanName
			}
		}
	}
	return nameMap
}

// disambiguateServiceAccounts handles the case where two ServiceAccounts
// would both be named "workflow-agent" after hash stripping.
// The agent SA has label app.kubernetes.io/name: customer-managed-workflow-agent.
// The worker SA (without that label) gets renamed to "worker-service-account".
func disambiguateServiceAccounts(docs []map[string]any, nameMap map[string]string) {
	type saInfo struct {
		index     int
		hashedKey string
		hasLabel  bool
	}

	var sas []saInfo
	for i, doc := range docs {
		if getStr(doc, "kind") != "ServiceAccount" {
			continue
		}
		meta := getMap(doc, "metadata")
		if meta == nil {
			continue
		}
		name := getStr(meta, "name")
		cleanName := nameMap[name]
		if cleanName == "" {
			cleanName = name
		}

		labels := getMap(meta, "labels")
		hasAgentLabel := labels != nil && getStr(labels, "app.kubernetes.io/name") == "customer-managed-workflow-agent"

		sas = append(sas, saInfo{
			index:     i,
			hashedKey: name,
			hasLabel:  hasAgentLabel,
		})
	}

	if len(sas) < 2 {
		return
	}

	// Check if there would be a name collision after hash stripping
	cleanNames := make(map[string]int)
	for _, sa := range sas {
		cleanName := nameMap[sa.hashedKey]
		if cleanName == "" {
			cleanName = sa.hashedKey
		}
		cleanNames[cleanName]++
	}

	for cleanName, count := range cleanNames {
		if count <= 1 {
			continue
		}
		// Collision detected — rename the worker SA
		for _, sa := range sas {
			cn := nameMap[sa.hashedKey]
			if cn == "" {
				cn = sa.hashedKey
			}
			if cn == cleanName && !sa.hasLabel {
				nameMap[sa.hashedKey] = "worker-service-account"
			}
		}
	}
}

// applyNameMap replaces all old names with new names throughout the resource.
func applyNameMap(doc map[string]any, nameMap map[string]string) {
	if len(nameMap) == 0 {
		return
	}
	replaceNamesRecursive(doc, nameMap)
}

func replaceNamesRecursive(v any, nameMap map[string]string) any {
	switch val := v.(type) {
	case map[string]any:
		for k, v := range val {
			val[k] = replaceNamesRecursive(v, nameMap)
		}
		return val
	case []any:
		for i, item := range val {
			val[i] = replaceNamesRecursive(item, nameMap)
		}
		return val
	case string:
		if newName, ok := nameMap[val]; ok {
			return newName
		}
		return val
	default:
		return val
	}
}

// removePulumiAnnotations removes pulumi.com/* annotations from a resource.
func removePulumiAnnotations(doc map[string]any) {
	meta := getMap(doc, "metadata")
	if meta == nil {
		return
	}
	annotations := getMap(meta, "annotations")
	if annotations == nil {
		return
	}

	for k := range annotations {
		if strings.HasPrefix(k, "pulumi.com/") {
			delete(annotations, k)
		}
	}

	if len(annotations) == 0 {
		delete(meta, "annotations")
	}
}

// normalizeSecretEncoding converts Secret.data (base64) to Secret.stringData
// with placeholder values, making it easier for helmify to parameterize.
func normalizeSecretEncoding(doc map[string]any) {
	if getStr(doc, "kind") != "Secret" {
		return
	}

	data := getMap(doc, "data")
	if data == nil {
		return
	}

	stringData := make(map[string]any)
	for k, v := range data {
		strVal, ok := v.(string)
		if !ok {
			stringData[k] = v
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(strVal)
		if err != nil {
			stringData[k] = strVal
			continue
		}
		stringData[k] = string(decoded)
	}

	delete(doc, "data")
	doc["stringData"] = stringData
}

func marshalMultiDoc(docs []map[string]any) (string, error) {
	var parts []string
	for _, doc := range docs {
		data, err := yaml.Marshal(doc)
		if err != nil {
			return "", err
		}
		parts = append(parts, string(data))
	}
	return strings.Join(parts, "---\n"), nil
}

func getStr(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func getNestedStr(m map[string]any, keys ...string) string {
	current := m
	for i, k := range keys {
		if i == len(keys)-1 {
			return getStr(current, k)
		}
		next := getMap(current, k)
		if next == nil {
			return ""
		}
		current = next
	}
	return ""
}

func getMap(m map[string]any, key string) map[string]any {
	v, ok := m[key]
	if !ok {
		return nil
	}
	result, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return result
}
