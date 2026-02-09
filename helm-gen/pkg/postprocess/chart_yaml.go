package postprocess

import (
	"fmt"
	"path/filepath"
)

func writeChartYAML(opts Options) error {
	content := fmt.Sprintf(`apiVersion: v2
name: %s
description: Helm chart for deploying the Pulumi Customer-Managed Workflow Agent
type: application
version: %s
appVersion: %s
maintainers:
  - name: Pulumi
    url: https://github.com/pulumi
home: https://github.com/pulumi/customer-managed-workflow-agent
sources:
  - https://github.com/pulumi/customer-managed-workflow-agent
`, opts.ChartName, opts.ChartVersion, opts.AppVersion)

	return writeFile(filepath.Join(opts.OutputDir, "Chart.yaml"), content)
}
