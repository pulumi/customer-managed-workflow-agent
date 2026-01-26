# Installing the self-managed agent into Kubernetes

## Prerequisites

* Use `pulumi config` to set imageName, namespace, and access token

* Install

```bash
pulumi config set agentNamespace <desired namespace>
pulumi config set --secret selfHostedAgentsAccessToken <access token>
pulumi config set agentImage <imageTag>
pulumi config set agentReplicas <replicas>

pulumi up
```

For example:

```bash
pulumi config set agentNamespace cmda
pulumi config set --secret selfHostedAgentsAccessToken pul-...
pulumi config set agentImage pulumi/customer-managed-workflow-agent:latest-amd64
pulumi config set agentReplicas 3
```

Optionally you can set an `agentImagePullPolicy` to a [Kubernetes supported value](https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy), which defaults to `Always`.

## Monitoring and Metrics

The deployment agent exposes health information on port 8080 at the `/healthz` endpoint. This can be used for monitoring and metrics collection.

### Basic Health Monitoring

The agent automatically creates a Kubernetes Service that exposes the health endpoint with Prometheus annotations for automatic discovery:

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

This creates a ServiceMonitor resource that automatically configures Prometheus to scrape the agent's health endpoint every 30 seconds.

### Manual Prometheus Configuration

If you're not using the Prometheus Operator, you can manually configure Prometheus to scrape the service using the annotations:

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

There is a ServiceAccount(`workerServiceAccount`) in the `index.ts` can be configured to support cloud service accounts.

This folder also contains a [raw kubernetes yaml file](./raw_deployment.yaml) for reference.

## Fargate Support

To enable Fargate you will need to use `customer-managed-workflow-agent` >= `1.3.7` and add the following to your pulumi code:

```typescript
// Create a Fargate profile
const fargateProfile = new aws.eks.FargateProfile("cmda-fargate-profile", {
    clusterName: cluster.eksCluster.name,
    podExecutionRoleArn: new aws.iam.Role("fargatePodExecutionRole", {
        assumeRolePolicy: aws.iam.assumeRolePolicyForPrincipal({ Service: "eks-fargate-pods.amazonaws.com" }),
    }).arn,
    subnetIds: eksVpc.privateSubnetIds,
    selectors: [{ namespace: <desired namespace>, labels: { "app.kubernetes.io/name": "workflow-runner" } }]
});
```

Additionally, there are two options for choosing your node size:

* `agentNumCpus` - Number of CPU's for the fargate instance
* `agentMemQuantity` - Quantity of memory in Gigabytes for the fargate instance. i.e. 4 = 4Gi

### 📌 Note

[Fargate Instance Reference](https://docs.aws.amazon.com/eks/latest/userguide/fargate-pod-configuration.html)

## Performance

### AWS

To optimize the performance of your deployments, you can use a pull-through cache in Amazon Elastic Container Registry (ECR). This allows you to cache frequently used images closer to your Kubernetes cluster, reducing the time it takes to pull images, and to prevent rate limiting.

For more information and an example of how to set up a pull-through cache in ECR using Pulumi, refer to the following:

* [Pulumi ECR Cache Example](https://github.com/pulumi/examples/tree/master/aws-ts-ecr-cache).
* [Implementing AWS ECR Pull Through cache for EKS cluster- most in-depth implementation details](https://marcincuber.medium.com/implementing-aws-ecr-pull-through-cache-for-eks-cluster-most-in-depth-implementation-details-e51395568034)


## Pod Spec Configuration

**⚠️ Advanced Feature Warning**: This is a very advanced feature that can potentially break your worker pods if configured incorrectly. Only use this if you have a deep understanding of Kubernetes pod specifications and are comfortable debugging pod scheduling issues.

The agent can be configured to use custom pod specifications for worker pods. This allows you to customize:

- **Node selectors** - Control which nodes the pods can be scheduled on
- **Tolerations** - Allow pods to run on nodes with specific taints
- **Init containers** - Run setup tasks before the main container starts
- **Resource limits and requests** - Control CPU and memory allocation
- **Environment variables** - Pass configuration to worker containers
- **Volume mounts** - Mount additional storage or configuration

### Example Pod Specification

Here's a complete example of a custom pod specification that demonstrates various configuration options:

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
                effect: "NoSchedule",        // Allow scheduling on dedicated workload nodes
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

Load a pod specification from a YAML file. See `examples/pod.yaml` for a complete example:

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

The pod specification uses Kubernetes [Strategic Merge Patch](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/#strategic-merge-patch) semantics. This means:

- **Additive**: New fields are added to the pod spec
- **Selective**: Only specified fields are modified, others remain unchanged
- **Strategic**: Kubernetes applies intelligent merging based on field types (e.g., arrays are merged, not replaced)

For more information on merge patches, see:
- [Kubernetes Strategic Merge Patch Documentation](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/#strategic-merge-patch)
- [JSON Patch vs Strategic Merge Patch](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/#use-a-json-patch-to-update-a-deployment)

## Troubleshooting

If you encounter issues with the workflow agent, please refer to our [troubleshooting guide](./troubleshooting/README.md) which includes:

* Diagnostic steps for identifying and resolving common problems
* Monitoring scripts to track pod status and resource usage
* Instructions for creating debug pods
* Example Kyverno policies for controlling pod scheduling
