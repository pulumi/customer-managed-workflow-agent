# Kubernetes Deployment Component

Pulumi TypeScript component that deploys the customer-managed workflow agent into a Kubernetes cluster. This is the primary (recommended) Kubernetes deployment mode. For the legacy Docker-in-Docker mode, see `../kubernetes-dind/`.

## Key Files

- `agent.ts` -- `PulumiSelfHostedAgentComponent` class: the core component that creates all K8s resources (Deployment, RBAC, Service, ConfigMap, Secret, optional ServiceMonitor)
- `index.ts` -- Entry point: instantiates the component with Pulumi config values
- `examples/pod.yaml` -- Example worker pod template with CA certificate mounting
- `troubleshooting/` -- Monitoring scripts, debug pod launcher, Kyverno policy example

## Setup

```bash
cd kubernetes
yarn install   # or npm install
pulumi config set agentNamespace <namespace>
pulumi config set --secret selfHostedAgentsAccessToken <token>
pulumi config set agentImage pulumi/customer-managed-workflow-agent:<tag>
pulumi config set agentReplicas <number>
pulumi up
```

## Configuration Reference

| Config key | Required | Default | Description |
|-----------|----------|---------|-------------|
| `agentNamespace` | Yes | -- | Kubernetes namespace for all resources |
| `selfHostedAgentsAccessToken` | Yes (secret) | -- | Pulumi agent pool access token |
| `agentImage` | Yes | -- | Docker image reference |
| `agentReplicas` | No | `3` | Number of agent pods |
| `agentImagePullPolicy` | No | `Always` | K8s image pull policy |
| `selfHostedServiceURL` | No | `https://api.pulumi.com` | Pulumi service API URL |
| `enableServiceMonitor` | No | `false` | Create Prometheus ServiceMonitor CR |
| `caCertificateSecretName` | No | -- | K8s Secret name containing custom CA bundle |
| `renderYamlToDirectory` | No | -- | Render YAML to directory instead of deploying |

## Architecture

The component creates these Kubernetes resources:
- **ConfigMap** (`agent-config`) -- service URL, image name, pull policy, worker pod template
- **Secret** (`agent-secret`) -- agent access token
- **ServiceAccount + Role + RoleBinding** -- RBAC for managing worker pods and configmaps
- **Deployment** (`workflow-agent-pool`) -- the agent pool itself, with health port 8080
- **Service** (`deployment-agent-service`) -- exposes `/healthz` with Prometheus annotations
- **ServiceMonitor** (optional) -- Prometheus Operator integration

Worker pods are spawned by the agent at runtime and configured via strategic merge patch from the pod template in the ConfigMap.

## Conventions

- The worker pod container must be named `pulumi-workflow` for merge patches to apply correctly
- Custom CA certificates replace the entire system trust store -- always combine with system CAs (see `../kubernetes/README.md`)
- The `renderYamlToDirectory` mode produces plaintext secrets in YAML files -- never commit those
