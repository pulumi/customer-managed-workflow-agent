package pipeline

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pulumi/customer-managed-workflow-agent/helm-gen/pkg/postprocess"
	"github.com/pulumi/customer-managed-workflow-agent/helm-gen/pkg/preprocess"
)

// Options configures the full pipeline.
type Options struct {
	InputDir     string
	OutputDir    string
	ChartName    string
	ChartVersion string
	AppVersion   string
}

// Run executes the full pipeline: preprocess → helmify → postprocess.
func Run(opts Options) error {
	fmt.Printf("Reading Pulumi rendered manifests from %s\n", opts.InputDir)

	result, err := preprocess.Run(opts.InputDir)
	if err != nil {
		return fmt.Errorf("preprocessing: %w", err)
	}

	fmt.Printf("Preprocessed %d resources\n", len(result.Resources))

	for _, r := range result.Resources {
		fmt.Printf("  - %s/%s (%s)\n", r.APIVersion, r.Kind, r.Name)
	}

	tmpDir, err := os.MkdirTemp("", "helm-gen-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	preprocessedFile := filepath.Join(tmpDir, "all.yaml")
	if err := result.WriteToFile(preprocessedFile); err != nil {
		return fmt.Errorf("writing preprocessed YAML: %w", err)
	}

	helmifyOutputDir := filepath.Join(tmpDir, "helmify-output")

	helmifyAvailable := tryHelmify(preprocessedFile, helmifyOutputDir, opts.ChartName)

	postOpts := postprocess.Options{
		OutputDir:    opts.OutputDir,
		ChartName:    opts.ChartName,
		ChartVersion: opts.ChartVersion,
		AppVersion:   opts.AppVersion,
	}

	if helmifyAvailable {
		postOpts.InputDir = filepath.Join(helmifyOutputDir, opts.ChartName)
		fmt.Println("Post-processing helmify output")
	} else {
		fmt.Println("Generating templates directly (helmify not available)")
		postOpts.InputDir = ""
	}

	if err := postprocess.Run(postOpts); err != nil {
		return fmt.Errorf("postprocessing: %w", err)
	}

	fmt.Printf("Helm chart generated at %s\n", opts.OutputDir)
	return nil
}

// tryHelmify attempts to run helmify on the preprocessed YAML.
// Returns true if helmify ran successfully.
func tryHelmify(inputFile, outputDir, chartName string) bool {
	helmifyPath, err := exec.LookPath("helmify")
	if err != nil {
		fmt.Println("helmify not found in PATH — will generate templates directly")
		fmt.Println("To install helmify: go install github.com/arttor/helmify/cmd/helmify@latest")
		return false
	}

	fmt.Printf("Using helmify at %s\n", helmifyPath)

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fmt.Printf("Warning: could not create helmify output dir: %v\n", err)
		return false
	}

	inputData, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Warning: could not read preprocessed file: %v\n", err)
		return false
	}

	chartDir := filepath.Join(outputDir, chartName)
	cmd := exec.Command(helmifyPath,
		"-crd-dir",
		chartDir,
	)
	cmd.Stdin = bytes.NewReader(inputData)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: helmify failed: %v — falling back to direct generation\n", err)
		return false
	}

	return true
}
