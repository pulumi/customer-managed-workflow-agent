package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pulumi/customer-managed-workflow-agent/helm-gen/pkg/pipeline"
)

func main() {
	var opts pipeline.Options

	flag.StringVar(&opts.InputDir, "input-dir", "", "Directory containing Pulumi rendered YAML manifests (required)")
	flag.StringVar(&opts.OutputDir, "output-dir", "", "Output directory for the Helm chart (required)")
	flag.StringVar(&opts.ChartName, "chart-name", "pulumi-deployment-agent", "Helm chart name")
	flag.StringVar(&opts.ChartVersion, "chart-version", "0.1.0", "Helm chart version")
	flag.StringVar(&opts.AppVersion, "app-version", "", "Application version (defaults to chart version)")
	flag.Parse()

	if opts.InputDir == "" || opts.OutputDir == "" {
		fmt.Fprintln(os.Stderr, "error: --input-dir and --output-dir are required")
		flag.Usage()
		os.Exit(1)
	}

	if opts.AppVersion == "" {
		opts.AppVersion = opts.ChartVersion
	}

	if err := pipeline.Run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
