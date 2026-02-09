# helm-gen

A Go tool that converts Pulumi-rendered Kubernetes manifests into a production
Helm chart for the customer-managed workflow agent.

## Overview

The `helm-gen` tool bridges two deployment models: the Pulumi-based Kubernetes
installer (in `kubernetes/`) defines the canonical set of resources, and
`helm-gen` transforms those rendered manifests into a distributable Helm chart
so users who prefer `helm install` get an equivalent deployment.

## Architecture

The tool runs a three-stage pipeline:

```text
Pulumi rendered YAML
  ──> preprocess
  ──> helmify (optional)
  ──> postprocess
  ──> Helm chart
```

### Stage 1: Preprocess (`pkg/preprocess`)

Normalizes Pulumi output for Helm consumption:

- **Name cleanup** — Strips Pulumi auto-generated hash suffixes
  (e.g., `workflow-agent-aaaa1111` becomes `workflow-agent`)
- **ServiceAccount disambiguation** — When two SAs would collide after hash
  stripping, renames the worker SA to `worker-service-account`
- **Annotation removal** — Strips `pulumi.com/*` annotations
- **Secret normalization** — Converts `Secret.data` (base64) to
  `Secret.stringData` (plaintext) for easier parameterization

### Stage 2: Helmify (optional)

If [helmify](https://github.com/arttor/helmify) is installed, the preprocessed
YAML is piped through it to generate an initial Helm chart scaffold. This step
is optional — if helmify is not available, Stage 3 generates all templates
directly.

### Stage 3: Postprocess (`pkg/postprocess`)

Produces the final Helm chart by either enhancing helmify output or generating
templates from scratch:

- **Templates** — Deployment, ConfigMap, Secret, Service, ServiceAccount,
  worker ServiceAccount, Role, RoleBinding, ServiceMonitor
- **Helpers** (`_helpers.tpl`) — Naming conventions, image construction,
  secret selection, config validation
- **Values** (`values.yaml`) — All configurable parameters with defaults
- **Chart metadata** (`Chart.yaml`, `.helmignore`, `NOTES.txt`)

## Prerequisites

- Go 1.23+
- Make
- (Optional) [helmify](https://github.com/arttor/helmify):
  `go install github.com/arttor/helmify/cmd/helmify@latest`
- (Optional) Helm CLI for linting/testing the output

## Usage

### Build

```bash
make helm-gen-build
```

This produces `./bin/helm-gen`.

### Generate a Helm chart

**Full pipeline** (requires Pulumi, Node.js, and a configured stack):

```bash
make helm-chart
```

**From existing rendered manifests** (skips the Pulumi render step):

```bash
make helm-chart-quick
```

**Direct CLI usage:**

```bash
./bin/helm-gen \
  --input-dir kubernetes/rendered-manifests/1-manifest \
  --output-dir helm-chart/pulumi-deployment-agent
```

### CLI Flags

| Flag | Description | Default |
|---|---|---|
| `--input-dir` | Directory with Pulumi rendered YAML (required) | — |
| `--output-dir` | Output directory for the Helm chart (required) | — |
| `--chart-name` | Helm chart name | `pulumi-deployment-agent` |
| `--chart-version` | Chart version | `0.1.0` |
| `--app-version` | Application version (defaults to chart version) | — |

### Lint the generated chart

```bash
make helm-lint
```

## Testing

```bash
make helm-gen-test
```

### Golden file tests

The `TestGoldenFiles` test in `pkg/pipeline/pipeline_test.go` compares the
generated chart against reference files in `testdata/golden/`. If you
intentionally change chart output, regenerate the golden files:

1. Run `go test ./... -count=1` to see which golden files differ
2. Update the corresponding golden files in `testdata/golden/` to match the
   new expected output
3. Re-run tests to confirm they pass

### Test structure

- `pkg/preprocess/` — Tests for name normalization, SA disambiguation, secret
  encoding
- `pkg/postprocess/` — Tests for template generation, value content, helper
  functions, template processor dispatch
- `pkg/pipeline/` — Integration tests (full pipeline run, golden file
  comparison)

## Adding a New Template

To add a new Kubernetes resource template to the chart:

1. **Add a generator function** in `pkg/postprocess/templates.go`:

   ```go
   func generateMyResourceTemplate() string {
       return `{{- if .Values.myResource.enabled }}
   apiVersion: v1
   kind: MyResource
   metadata:
     name: {{ include "chart.fullname" . }}-my-resource
     labels:
       {{- include "chart.labels" . | nindent 4 }}
   {{- end }}
   `
   }
   ```

2. **Register in the default templates map** in `pkg/postprocess/postprocess.go`:

   ```go
   func writeDefaultTemplates(opts Options) error {
       templates := map[string]string{
           // ... existing entries ...
           "myresource.yaml": generateMyResourceTemplate(),
       }
   ```

3. **Add to the templateProcessors map** (same file) for the helmify
   processing path:

   ```go
   var templateProcessors = map[string]func(string) string{
       // ... existing entries ...
       "myresource.yaml": func(c string) string {
           return wrapWithGuard(c, `{{- if .Values.myResource.enabled }}`)
       },
   }
   ```

4. **Add values** in `pkg/postprocess/values_yaml.go`:

   ```yaml
   myResource:
     enabled: false
   ```

5. **Add a golden file** at `testdata/golden/templates/myresource.yaml`

6. **Update test lists** — Add the new filename to:
   - `requiredFiles` in `TestRunWithoutHelmify` (`pkg/pipeline/pipeline_test.go`)
   - `goldenFiles` in `TestGoldenFiles` (`pkg/pipeline/pipeline_test.go`)
   - `templates` map in `TestGenerateTemplates`
     (`pkg/postprocess/postprocess_test.go`)

7. **Run tests**: `go test ./... -count=1`

## Project Layout

```text
helm-gen/
  main.go                          CLI entry point
  pkg/
    pipeline/
      pipeline.go                  Orchestrates preprocess -> helmify -> postprocess
      pipeline_test.go             Integration tests, golden file tests
    preprocess/
      preprocess.go                Name normalization, annotation cleanup
      preprocess_test.go
    postprocess/
      postprocess.go               Template processing, file writing
      templates.go                 Template generator functions
      values_yaml.go               values.yaml content
      helpers.go                   _helpers.tpl, NOTES.txt, .helmignore
      chart_yaml.go                Chart.yaml generation
      postprocess_test.go
  testdata/
    input/                         Sample Pulumi rendered manifests
    golden/                        Expected chart output (golden files)
```
