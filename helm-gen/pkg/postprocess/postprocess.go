package postprocess

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Options configures the post-processor.
type Options struct {
	InputDir     string // helmify output directory
	OutputDir    string // final chart output directory
	ChartName    string
	ChartVersion string
	AppVersion   string
}

// Run post-processes the helmify output into a production Helm chart.
func Run(opts Options) error {
	if err := os.MkdirAll(filepath.Join(opts.OutputDir, "templates"), 0o755); err != nil {
		return fmt.Errorf("creating output dirs: %w", err)
	}

	if err := writeChartYAML(opts); err != nil {
		return fmt.Errorf("writing Chart.yaml: %w", err)
	}

	if err := writeValuesYAML(opts); err != nil {
		return fmt.Errorf("writing values.yaml: %w", err)
	}

	if err := writeHelpers(opts); err != nil {
		return fmt.Errorf("writing _helpers.tpl: %w", err)
	}

	if err := writeNotes(opts); err != nil {
		return fmt.Errorf("writing NOTES.txt: %w", err)
	}

	if err := writeHelmignore(opts); err != nil {
		return fmt.Errorf("writing .helmignore: %w", err)
	}

	if err := processTemplates(opts); err != nil {
		return fmt.Errorf("processing templates: %w", err)
	}

	return nil
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// processTemplates reads helmify-generated templates and enhances them.
func processTemplates(opts Options) error {
	templateDir := filepath.Join(opts.InputDir, "templates")
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return writeDefaultTemplates(opts)
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "_helpers.tpl" || name == "NOTES.txt" {
			continue
		}

		content, err := os.ReadFile(filepath.Join(templateDir, name))
		if err != nil {
			return err
		}

		processed := processTemplate(name, string(content))
		outPath := filepath.Join(opts.OutputDir, "templates", name)
		if err := writeFile(outPath, processed); err != nil {
			return err
		}
	}

	return nil
}

// templateProcessors maps exact template filenames to their processing functions.
var templateProcessors = map[string]func(string) string{
	"deployment.yaml": processDeploymentTemplate,
	"configmap.yaml":  processConfigMapTemplate,
	"secret.yaml": func(c string) string {
		return wrapWithGuard(c, `{{- if and .Values.agent.token (not .Values.agent.existingSecretName) }}`)
	},
	"role.yaml": func(c string) string {
		return wrapWithGuard(c, `{{- if .Values.rbac.create }}`)
	},
	"rolebinding.yaml": func(c string) string {
		return wrapWithGuard(c, `{{- if .Values.rbac.create }}`)
	},
	"worker-serviceaccount.yaml": func(c string) string {
		return wrapWithGuard(c, `{{- if .Values.workerServiceAccount.create }}`)
	},
	"serviceaccount.yaml": processServiceAccountTemplate,
	"servicemonitor.yaml": func(c string) string {
		return wrapWithGuard(c, `{{- if .Values.serviceMonitor.enabled }}`)
	},
	"service.yaml": func(c string) string { return c },
}

func processTemplate(name, content string) string {
	if fn, ok := templateProcessors[name]; ok {
		return fn(content)
	}

	// Fallback for non-standard names from helmify output
	switch {
	case strings.Contains(name, "deployment"):
		return processDeploymentTemplate(content)
	case strings.Contains(name, "configmap"):
		return processConfigMapTemplate(content)
	case strings.Contains(name, "secret"):
		return wrapWithGuard(content, `{{- if and .Values.agent.token (not .Values.agent.existingSecretName) }}`)
	case strings.Contains(name, "role") && strings.Contains(name, "binding"):
		return wrapWithGuard(content, `{{- if .Values.rbac.create }}`)
	case strings.Contains(name, "role"):
		return wrapWithGuard(content, `{{- if .Values.rbac.create }}`)
	case strings.Contains(name, "worker") && strings.Contains(name, "serviceaccount"):
		return wrapWithGuard(content, `{{- if .Values.workerServiceAccount.create }}`)
	case strings.Contains(name, "serviceaccount"):
		return processServiceAccountTemplate(content)
	case strings.Contains(name, "servicemonitor"):
		return wrapWithGuard(content, `{{- if .Values.serviceMonitor.enabled }}`)
	default:
		return content
	}
}

func wrapWithGuard(content, guard string) string {
	return guard + "\n" + content + "{{- end }}\n"
}

func writeDefaultTemplates(opts Options) error {
	templates := map[string]string{
		"deployment.yaml":            generateDeploymentTemplate(),
		"configmap.yaml":             generateConfigMapTemplate(),
		"secret.yaml":                generateSecretTemplate(),
		"service.yaml":               generateServiceTemplate(),
		"serviceaccount.yaml":        generateServiceAccountTemplate(),
		"role.yaml":                  generateRoleTemplate(),
		"rolebinding.yaml":           generateRoleBindingTemplate(),
		"servicemonitor.yaml":        generateServiceMonitorTemplate(),
		"worker-serviceaccount.yaml": generateWorkerServiceAccountTemplate(),
	}

	for name, content := range templates {
		path := filepath.Join(opts.OutputDir, "templates", name)
		if err := writeFile(path, content); err != nil {
			return err
		}
	}
	return nil
}
