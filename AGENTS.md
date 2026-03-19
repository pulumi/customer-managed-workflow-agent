# Customer Managed Workflow Agent

Release packaging and deployment tooling for Pulumi's customer-managed workflow agents. This repository does **not** contain the agent source code -- it packages Go binaries from the `pulumi-service` submodule and provides Kubernetes deployment components, Docker images, and AWS AMI tooling.

## Tech Stack

- **Build/Release**: GoReleaser v2, cosign (artifact signing)
- **Kubernetes components**: TypeScript, Pulumi SDK (`@pulumi/pulumi` ^3, `@pulumi/kubernetes` ^4.14), Node.js 18+
- **AMI building**: HashiCorp Packer, AWS EBS
- **CI/CD**: GitHub Actions (tag-triggered releases via GoReleaser)
- **Go version** (for building submodule binaries): 1.22

## Repository Structure

- `pulumi-service/` -- Git submodule pointing to `pulumi/pulumi-service`. Contains the Go source for both binaries. **Never edit files here directly.**
- `kubernetes/` -- Pulumi TypeScript component for deploying the agent to Kubernetes (native mode). Has its own `AGENTS.md`.
- `kubernetes-dind/` -- Pulumi TypeScript component for Kubernetes Docker-in-Docker deployment (legacy, simpler alternative)
- `agent-images/` -- Packer template and Pulumi program for building AWS AMIs and deploying EC2 instances
- `.goreleaser.yaml` -- Release config (builds, Docker images, manifests, signing)
- `.goreleaser.prerelease.yaml` -- Prerelease config (no `latest` tags, no GitHub release)
- `docker/` -- Alternative Dockerfile that installs from published releases (not used by GoReleaser)
- `goreleaser/` -- GoReleaser output directory (gitignored, present locally after builds)
- `workflow-runner-embeddable` -- Binary artifact copied by GoReleaser post-hook (gitignored, present locally after builds)

## Development

### Setup

```bash
# Initialize the pulumi-service submodule (required before any build)
make submodules
```

### Validating TypeScript Components

There are no automated tests in this repository. Before pushing changes to TypeScript components, verify they compile:

```bash
# Kubernetes native component
cd kubernetes && yarn install && npx tsc --noEmit && cd ..

# Kubernetes DinD component (legacy)
cd kubernetes-dind && yarn install && npx tsc --noEmit && cd ..
```

The `agent-images/agent-setup/` program uses npm instead of yarn:
```bash
cd agent-images/agent-setup && npm install && npx tsc --noEmit && cd ..
```

### Local Testing with GoReleaser

```bash
# Login to your Docker Hub account first
docker login

export REPO_OWNER=<your-docker-hub-name>
export BUILD_STAMP=$(cd ./pulumi-service && date -u '+%Y-%m-%d_%I:%M:%S%p')
export BUILD_GIT_HASH=$(cd ./pulumi-service && git rev-parse HEAD)
goreleaser release --skip=validate,sign,archive,publish --clean
```

GoReleaser requires a clean git working tree. Commit local changes before building.

### Releasing

Releases are fully automated via GitHub Actions:
- **Release**: Push a tag matching `vX.Y.Z` (no prerelease suffix)
- **Prerelease**: Push a tag matching `vX.Y.Z-*` (any prerelease suffix)

Both workflows use `.github/workflows/stage-publish.yaml` which runs GoReleaser, builds multi-arch Docker images, and signs artifacts with cosign.

## What Gets Built

GoReleaser produces three binaries from the `pulumi-service` submodule:

| Binary | Source directory | Platforms |
|--------|-----------------|-----------|
| `customer-managed-workflow-agent` | `pulumi-service/cmd/customer-managed-workflow-agent` | darwin/linux, amd64/arm64 |
| `workflow-runner` | `pulumi-service/cmd/workflow-runner` | darwin/linux, amd64/arm64 |
| `workflow-runner-embeddable` | `pulumi-service/cmd/workflow-runner` (same source) | linux/amd64 only |

Docker images are published to Docker Hub under two names for backward compatibility:
- `pulumi/customer-managed-workflow-agent` (current)
- `pulumi/customer-managed-deployment-agent` (legacy name, still published)

## Important Concepts

### Naming History

The project was renamed from `customer-managed-deployment-agent` to `customer-managed-workflow-agent` in v2.0.0. Docker images are still published under both names for backward compatibility. Be aware of this when reading older code, configs, or customer documentation.

### Submodule Pinning

The `pulumi-service` submodule is pinned to a specific commit. Updating it is a deliberate action:
```bash
cd pulumi-service && git fetch && git checkout <target-commit>
cd .. && git add pulumi-service && git commit -m "Update pulumi-service to <short-hash>"
```

### Configuration

The agent reads `pulumi-workflow-agent.yaml` at runtime. See `pulumi-workflow-agent.yaml.sample` for the format. Key fields: `token` (Pulumi access token), `service_url`, `workflow_runners_directory`, `workflow_runners_embeddable`.

## Forbidden Patterns

- **Never edit files inside `pulumi-service/`** -- it is a submodule. Changes go in the `pulumi/pulumi-service` repository.
- **Never commit the `goreleaser/` directory or `workflow-runner-embeddable` binary** -- they are build artifacts and are gitignored.
- **Never commit secrets** (tokens, cosign private keys) -- the `cosign.key` in the repo is encrypted and requires `COSIGN_PASSWORD`.
- **Never mix package managers** -- use yarn in `kubernetes/` and `kubernetes-dind/`, npm in `agent-images/agent-setup/`.

## Changelog

Update `CHANGELOG.md` when preparing a release. Use `CHANGELOG_PENDING.md` for release notes that GoReleaser includes in the GitHub release.

## Nested AGENTS.md Files

- `kubernetes/AGENTS.md` -- Kubernetes native deployment component (TypeScript/Pulumi)
