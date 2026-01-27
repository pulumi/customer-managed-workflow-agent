# Installing the self-managed agent into Kubernetes

## Table of Contents

- [Prerequisites](#prerequisites)
- [Monitoring and Metrics](#monitoring-and-metrics)
  - [Basic Health Monitoring](#basic-health-monitoring)
  - [Prometheus Integration](#prometheus-integration)
  - [Manual Prometheus Configuration](#manual-prometheus-configuration)
- [workerServiceAccount](#workerserviceaccount)
- [Generating Static YAML Manifests](#generating-static-yaml-manifests)
  - [Usage](#usage)
  - [Applying Rendered Manifests](#applying-rendered-manifests)
  - [Important Notes](#%EF%B8%8F-important-notes)
  - [Switching Back to Direct Deployment](#switching-back-to-direct-deployment)
- [Fargate Support](#fargate-support)
- [Performance](#performance)
  - [AWS](#aws)
- [Pod Spec Configuration](#pod-spec-configuration)
  - [Example Pod Specification](#example-pod-specification)
  - [Loading Pod Specification into the Agent](#loading-pod-specification-into-the-agent)
  - [Merge Patches](#merge-patches)
- [Local Development with kind](#local-development-with-kind)
  - [Requirements](#requirements)
  - [Quick Start](#quick-start)
  - [Generate YAML for Production](#generate-yaml-for-production)
  - [Tips for Local Development](#tips-for-local-development)
  - [Cleanup](#cleanup)
- [Troubleshooting](#troubleshooting)

## Prerequisites

- Use `pulumi config` to set imageName, namespace, and access token

- Install

```bash
pulumi config set agentNamespace <desired namespace>
pulumi config set --secret selfHostedAgentsAccessToken <access token>
pulumi config set agentImage <imageTag>
pulumi config set agentReplicas <replicas>

pulumi up
```

For example:

```bash
pulumi config set agentNamespace cmwa
pulumi config set --secret selfHostedAgentsAccessToken pul-...
pulumi config set agentImage pulumi/customer-managed-workflow-agent:latest-amd64
pulumi config set agentReplicas 3
```

Optionally you can set an `agentImagePullPolicy` to a
[Kubernetes supported value][k8s-pull-policy], which defaults to `Always`.

[k8s-pull-policy]: https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy

## Monitoring and Metrics

The deployment agent exposes health information on port 8080 at the `/healthz`
endpoint. This can be used for monitoring and metrics collection.

### Basic Health Monitoring

The agent automatically creates a Kubernetes Service that exposes the health
endpoint with Prometheus annotations for automatic discovery:

```json
{
  "status": "healthy",
  "currentTime": "2025-06-22T21:15:38.338784381Z",
  "lastActivity": "2025-06-22T21:15:11.931901718Z"
}
```

### Prometheus Integration

To enable Prometheus monitoring with the Prometheus Operator:

```bash
pulumi config set enableServiceMonitor true
```

This creates a ServiceMonitor resource that automatically configures Prometheus
to scrape the agent's health endpoint every 30 seconds.

### Manual Prometheus Configuration

If you're not using the Prometheus Operator, you can manually configure
Prometheus to scrape the service using the annotations:

```yaml
- job_name: 'pulumi-deployment-agent'
  kubernetes_sd_configs:
    - role: service
  relabel_configs:
    - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scrape]
      action: keep
      regex: true
    - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_path]
      action: replace
      target_label: __metrics_path__
      regex: (.+)
```

The health endpoint provides insights into:

- Agent operational status (healthy/unhealthy)
- Last activity timestamp for detecting stuck agents
- Current timestamp for time synchronization validation

## workerServiceAccount

There is a ServiceAccount(`workerServiceAccount`) in the `index.ts` that can be
configured to support cloud service accounts.

To generate static YAML manifests instead of deploying directly, see the
[Generating Static YAML Manifests](#generating-static-yaml-manifests) section.

## Generating Static YAML Manifests

Instead of deploying directly to a Kubernetes cluster, you can render the
manifests to a local directory. This is useful for:

- Reviewing generated YAML before applying
- GitOps workflows where manifests are committed to a repository
- Air-gapped environments where Pulumi can't access the cluster directly

### Usage

```bash
# Set the output directory
pulumi config set renderYamlToDirectory ./rendered-manifests

# Clean any previous renders and generate fresh YAML files
rm -rf ./rendered-manifests
pulumi up
```

The manifests will be generated in subdirectories:

- `0-crd/` - Custom Resource Definitions (if any)
- `1-manifest/` - All other Kubernetes resources

### Applying Rendered Manifests

Due to Kubernetes resource ordering, apply the manifests in two steps to ensure
the namespace exists before namespaced resources are created:

```bash
# Step 1: Apply the CRD's if they exist
kubectl apply -f ./rendered-manifests/0-crd/

# Step 2: Apply namespace first
kubectl apply -f ./rendered-manifests/1-manifest/v1-namespace-*.yaml

# Step 3: Apply all manifests (namespace will show "unchanged")
kubectl apply -f ./rendered-manifests/1-manifest/

# Verify deployment
kubectl get all -n <your-namespace>
```

Alternatively, you can simply run `kubectl apply` twice:

```bash
kubectl apply -f ./rendered-manifests/1-manifest/
# Some resources may fail on first apply due to namespace race condition
kubectl apply -f ./rendered-manifests/1-manifest/
```

### ⚠️ Important Notes

**Secrets appear in plaintext** in the rendered YAML files. The
`selfHostedAgentsAccessToken` will be base64-encoded (standard Kubernetes
Secret encoding) but not encrypted. Ensure you:

- Do not commit rendered manifests containing secrets to version control
- Use appropriate file permissions on the output directory
- Consider using external secret management for production
  (e.g., Sealed Secrets, External Secrets Operator)

**Mode switching causes resource replacement.** When you toggle
`renderYamlToDirectory` on or off, Pulumi replaces the provider which cascades
to all resources. This is expected because the two modes are fundamentally
different (cluster deployment vs file rendering).

### Switching Back to Direct Deployment

```bash
# Remove the config to deploy directly again
pulumi config rm renderYamlToDirectory

# Deploy to the cluster (resources will be replaced)
pulumi up
```

## Fargate Support

To enable Fargate you will need to use `customer-managed-workflow-agent` >=
`1.3.7` and add the following to your pulumi code:

```typescript
// Create a Fargate profile
const fargateProfile = new aws.eks.FargateProfile("cmwa-fargate-profile", {
    clusterName: cluster.eksCluster.name,
    podExecutionRoleArn: new aws.iam.Role("fargatePodExecutionRole", {
        assumeRolePolicy: aws.iam.assumeRolePolicyForPrincipal({
            Service: "eks-fargate-pods.amazonaws.com",
        }),
    }).arn,
    subnetIds: eksVpc.privateSubnetIds,
    selectors: [{
        namespace: <desired namespace>,
        labels: { "app.kubernetes.io/name": "workflow-runner" },
    }],
});
```

Additionally, there are two options for choosing your node size:

- `agentNumCpus` - Number of CPU's for the fargate instance
- `agentMemQuantity` - Quantity of memory in Gigabytes for the fargate instance
  (i.e. 4 = 4Gi)

### 📌 Note

[Fargate Instance Reference](https://docs.aws.amazon.com/eks/latest/userguide/fargate-pod-configuration.html)

## Performance

### AWS

To optimize the performance of your deployments, you can use a pull-through
cache in Amazon Elastic Container Registry (ECR). This allows you to cache
frequently used images closer to your Kubernetes cluster, reducing the time
it takes to pull images, and to prevent rate limiting.

For more information and an example of how to set up a pull-through cache in
ECR using Pulumi, refer to the following:

- [Pulumi ECR Cache Example][ecr-cache-example]
- [Implementing AWS ECR Pull Through cache for EKS cluster][ecr-cache-guide]

[ecr-cache-example]: https://github.com/pulumi/examples/tree/master/aws-ts-ecr-cache
[ecr-cache-guide]: https://marcincuber.medium.com/implementing-aws-ecr-pull-through-cache-for-eks-cluster-most-in-depth-implementation-details-e51395568034

## Pod Spec Configuration

**⚠️ Advanced Feature Warning**: This is a very advanced feature that can
potentially break your worker pods if configured incorrectly. Only use this
if you have a deep understanding of Kubernetes pod specifications and are
comfortable debugging pod scheduling issues.

The agent can be configured to use custom pod specifications for worker pods.
This allows you to customize:

- **Node selectors** - Control which nodes the pods can be scheduled on
- **Tolerations** - Allow pods to run on nodes with specific taints
- **Init containers** - Run setup tasks before the main container starts
- **Resource limits and requests** - Control CPU and memory allocation
- **Environment variables** - Pass configuration to worker containers
- **Volume mounts** - Mount additional storage or configuration

### Example Pod Specification

Here's a complete example of a custom pod specification that demonstrates
various configuration options:

```typescript
import { V1Pod } from "@kubernetes/client-node";

const pod: V1Pod = {
    metadata: {
        labels: {
            "cost-optimization": "true",  // Custom label for cost tracking
        },
        annotations: {
            "cost-optimization": "true",  // Annotation for cost optimization policies
        },
    },
    spec: {
        // Node selector to target specific node types
        nodeSelector: {
            "kubernetes.io/os": "linux",     // Only schedule on Linux nodes
            "node-type": "worker",           // Only schedule on worker nodes
        },
        
        // Tolerations to allow scheduling on nodes with specific taints
        tolerations: [
            {
                key: "node-role.kubernetes.io/master",
                operator: "Exists",
                effect: "NoSchedule",        // Allow scheduling on master nodes
            },
            {
                key: "dedicated",
                operator: "Equal",
                value: "pulumi-workload",
                effect: "NoSchedule",  // Allow on dedicated nodes
            },
        ],
        
        // Init containers run before the main container starts
        initContainers: [
            {
                name: "init-setup",
                image: "busybox:1.35",       // Lightweight init container
                command: [
                    "sh", 
                    "-c", 
                    "echo 'Initializing pod...' && echo 'Pod ready' > /mnt/pulumi/workflow/status.txt"
                ],
            },
        ],
        
        // Main container configuration
        containers: [
            {
                name: "pulumi-workflow",
                env: [
                    {
                        name: "HELLO",
                        value: "world",      // Example environment variable
                    },
                ],
                resources: {
                    requests: {
                        cpu: "100m",         // Request 100 millicores of CPU
                        memory: "256Mi",     // Request 256 MiB of memory
                    },
                },
            },
        ],
    },
};
```

### Loading Pod Specification into the Agent

You can provide a custom pod specification to the agent in two ways:

#### Method 1: Direct Pod Object

Pass a `V1Pod` object directly to the agent component:

```typescript
import { V1Pod } from "@kubernetes/client-node";

const customPodSpec: V1Pod = {
    // Your custom pod specification here
    spec: {
        nodeSelector: {
            "kubernetes.io/os": "linux",
        },
        // ... other specifications
    },
};

const agent = new PulumiSelfHostedAgentComponent("agent", {
    // ... other arguments
    podTemplate: customPodSpec,
});
```

#### Method 2: Load from YAML File

Load a pod specification from a YAML file. See `examples/pod.yaml` for a
complete example:

```typescript
import * as fs from "fs";
import * as yaml from "js-yaml";
import { V1Pod } from "@kubernetes/client-node";

// Load pod specification from YAML file
const podYaml = fs.readFileSync("./pod.yaml", "utf8");
const podSpec = yaml.load(podYaml) as V1Pod;

const agent = new PulumiSelfHostedAgentComponent("agent", {
    // ... other arguments
    podTemplate: podSpec,
});
```

### Merge Patches

The pod specification uses Kubernetes [Strategic Merge Patch][smp] semantics.
This means:

- **Additive**: New fields are added to the pod spec
- **Selective**: Only specified fields are modified, others remain unchanged
- **Strategic**: Kubernetes applies intelligent merging based on field types
  (e.g., arrays are merged, not replaced)

For more information on merge patches, see:

- [Kubernetes Strategic Merge Patch Documentation][smp]
- [JSON Patch vs Strategic Merge Patch][json-patch]

[smp]: https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/#strategic-merge-patch
[json-patch]: https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/#use-a-json-patch-to-update-a-deployment

## Local Development with kind

[kind](https://kind.sigs.k8s.io/) (Kubernetes IN Docker) provides a simple way
to run a local Kubernetes cluster for development and testing.

### Requirements

- Docker installed and running
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) installed
- kubectl installed

### Quick Start

```bash
# Create a local cluster
kind create cluster --name pulumi-agent-dev

# Verify the cluster is running
kubectl cluster-info --context kind-pulumi-agent-dev

# Configure the agent
pulumi config set agentNamespace cmwa
pulumi config set --secret selfHostedAgentsAccessToken <your-token>
pulumi config set agentImage pulumi/customer-managed-workflow-agent:latest-amd64
pulumi config set agentReplicas 1
pulumi config set agentImagePullPolicy IfNotPresent

# Deploy directly to kind
pulumi up

# Verify deployment
kubectl get pods -n cmwa
```

### Generate YAML for Production

After testing locally, you can generate static YAML manifests for production deployment:

```bash
# Switch to YAML rendering mode
pulumi config set renderYamlToDirectory ./rendered-manifests

# Clean previous renders and generate fresh manifests
rm -rf ./rendered-manifests
pulumi destroy --yes  # Clear state
pulumi up --yes       # Generate YAML files

# Review the generated manifests

ls ./rendered-manifests/0-crd/
ls ./rendered-manifests/1-manifest/

# The YAML files can now be applied to your production cluster:
# kubectl apply -f ./rendered-manifests/0-crd/
# kubectl apply -f ./rendered-manifests/1-manifest/v1-namespace-*.yaml
# kubectl apply -f ./rendered-manifests/1-manifest/
```

### Tips for Local Development

- Use `agentReplicas: 1` to reduce resource usage
- Set `agentImagePullPolicy: IfNotPresent` to avoid re-pulling images
- Use the `renderYamlToDirectory` option to inspect generated manifests
  before production deployment

### Cleanup

```bash
# Destroy Pulumi resources
pulumi destroy

# Delete the kind cluster
kind delete cluster --name pulumi-agent-dev

# Clean up rendered manifests
rm -rf ./rendered-manifests
```

## Troubleshooting

If you encounter issues with the workflow agent, please refer to our
[troubleshooting guide](./troubleshooting/README.md) which includes:

- Diagnostic steps for identifying and resolving common problems
- Monitoring scripts to track pod status and resource usage
- Instructions for creating debug pods
- Example Kyverno policies for controlling pod scheduling
