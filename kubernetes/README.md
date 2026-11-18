# Installing the self-managed agent into Kubernetes

## Table of Contents

- [Prerequisites](#prerequisites)
- [Upgrading to the Secret-based job credentials](#upgrading-to-the-secret-based-job-credentials)
  - [Upgrading with this program](#upgrading-with-this-program)
  - [Upgrading rendered or hand-maintained manifests](#upgrading-rendered-or-hand-maintained-manifests)
  - [PULUMI_AGENT_JOB_CREDENTIALS](#pulumi_agent_job_credentials)
  - [What the agent creates for each job](#what-the-agent-creates-for-each-job)
  - [Pinning and rolling back](#pinning-and-rolling-back)
  - [Removing objects left by earlier jobs](#removing-objects-left-by-earlier-jobs)
- [Monitoring and Metrics](#monitoring-and-metrics)
  - [Basic Health Monitoring](#basic-health-monitoring)
  - [Prometheus Integration](#prometheus-integration)
  - [Manual Prometheus Configuration](#manual-prometheus-configuration)
- [workerServiceAccount](#workerserviceaccount)
- [Custom CA Certificates](#custom-ca-certificates)
  - [Creating the CA Certificate Secret](#creating-the-ca-certificate-secret)
  - [Configuring the Agent Pod](#configuring-the-agent-pod)
  - [Configuring Worker Pods](#configuring-worker-pods)
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
pulumi config set agentImage pulumi/customer-managed-workflow-agent:latest
pulumi config set agentReplicas 3
```

Optionally you can set an `agentImagePullPolicy` to a
[Kubernetes supported value][k8s-pull-policy], which defaults to `Always`.

[k8s-pull-policy]: https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy

Other optional settings:

- `agentJobCredentials` sets `PULUMI_AGENT_JOB_CREDENTIALS` on the agent, which
  defaults to `secret` here. See
  [PULUMI_AGENT_JOB_CREDENTIALS](#pulumi_agent_job_credentials).

## Upgrading to the Secret-based job credentials

Agent releases with the `PULUMI_AGENT_JOB_CREDENTIALS` setting hand each job
its credentials in a Kubernetes Secret that only the job's Pod mounts and that
Kubernetes deletes with the Pod. The agent's Role must allow it to create
Secrets. Without that, the agent keeps using a ConfigMap and logs a warning,
with a link to this section, on every job.

Before you upgrade, if you supply a [pod template](#pod-spec-configuration) or
use admission policies, check them against
[What the agent creates for each job](#what-the-agent-creates-for-each-job).
The agent checks the pod template only when it uses a Secret, so a template
that works today can fail jobs after the upgrade.

### Upgrading with this program

1. If you pin `agentImage` to a version, set it to the new release.
2. Update your copy of this program and run `pulumi up`. It adds the rule
   below to the agent's Role, sets `PULUMI_AGENT_JOB_CREDENTIALS=secret`, and
   updates the agent Deployment after the Role. This restarts the agent Pods,
   all at once when `agentImage` changes. A deployment that runs during the
   restart can fail or leave its job Pod behind (see
   [Removing objects left by earlier jobs](#removing-objects-left-by-earlier-jobs)).
   With a `latest` tag, the Pods pull the new release only with the default
   `Always` pull policy. If the Pods did not restart, run
   `kubectl -n <agent namespace> rollout restart deployment/workflow-agent-pool`.
3. Run a deployment. Its log shows `Credentials Secret created` and no
   `ConfigMap created` line.

### Upgrading rendered or hand-maintained manifests

Apply the changes in this order:

1. Add this rule to the agent's Role and apply the Role first:

   ```yaml
   - apiGroups: [""]
     resources: ["secrets"]
     verbs: ["create"]
   ```

   With manifests rendered by this program, apply the Role and RoleBinding
   files before the rest:

   ```bash
   grep -lE '^kind: (Role|RoleBinding)$' ./rendered-manifests/1-manifest/*.yaml | xargs -n1 kubectl apply -f
   ```

2. If `PULUMI_AGENT_IMAGE` (in the `agent-config` ConfigMap, with manifests
   rendered by this program) names a version tag, a digest or a mirror, move
   it to the new release before the agent image. An earlier agent works with
   the newer runner, but a newer agent does not work with an earlier runner.
   Pulumi's image with no tag or a `latest` tag needs no change.
3. Set `PULUMI_AGENT_JOB_CREDENTIALS=secret` on the agent container, set the
   agent image to the new release, and apply the Deployment.
4. Run a deployment. Its log shows `Credentials Secret created` and no
   `ConfigMap created` line.

### PULUMI_AGENT_JOB_CREDENTIALS

- `secret` (recommended): always use a Secret. A job fails in Setup, with the
  reason, when the agent may not create Secrets. This program sets this value.
- `prefer-secret` (the default when unset, so installs whose Role does not
  allow Secrets yet keep working): use a Secret when the agent may create
  Secrets in its namespace. Otherwise use a ConfigMap and log a warning in the
  deployment log and the agent log.
- `configmap`: use a ConfigMap even when the agent may create Secrets, with
  the same warning. Use it to go back to the earlier behavior without removing
  the Role rule.

With any value, a job whose image needs registry credentials fails in Setup
when the agent may not create Secrets, and a job fails in Setup when the
permission check returns an error other than 403. Any other value fails every
job in Setup. Agent releases without this setting ignore it.

### What the agent creates for each job

- A Pod `pulumi-workflow-<suffix>`, where the suffix is five random letters,
  labeled `app.kubernetes.io/name=workflow-runner` and
  `app.kubernetes.io/component=pulumi-workflow`.
- A Secret `pulumi-workflow-<suffix>-credentials`, mounted read-only at
  `/mnt/pulumi/credentials` in the `pulumi-workflow` container.
- A Secret `pulumi-workflow-<suffix>-registry`, only when the job's image
  needs registry credentials. The agent creates it in either mode, so these
  jobs need the Role rule either way.

The Pod owns both Secrets, so Kubernetes deletes them with the Pod. Grant the
agent only `create` on Secrets: `delete` and `deletecollection` can return
Secret contents. Run the agent in a namespace of its own, as this program does.

A [pod template](#pod-spec-configuration) must not change the
`pulumi-workflow` container's `command` or `args`, the credentials volume or
its mount, and must not mount anything else at or under
`/mnt/pulumi/credentials`. When the agent uses a Secret, it checks this and
fails the job in Setup, naming the field.
An admission policy that restricts Secret creation or `secret` volumes must
let the agent's service account create Secrets in its namespace and let job
Pods mount them.

The agent checks whether it may create Secrets with a
`SelfSubjectAccessReview`, which Kubernetes allows through the
`system:basic-user` ClusterRoleBinding. If your cluster removed that binding,
`prefer-secret` uses a ConfigMap and `secret` fails every job in Setup.
Restore the binding.

### Pinning and rolling back

- To pin the agent, set `agentImage` to a version tag, such as
  `pulumi/customer-managed-workflow-agent:vX.Y.Z`.
- To go back to the ConfigMap without changing the image, set
  `agentJobCredentials` (or `PULUMI_AGENT_JOB_CREDENTIALS`) to `configmap` and
  keep the Role rule.
- To roll back the image, set `agentImage`, and `PULUMI_AGENT_IMAGE` if you
  set it separately, to the earlier release. Keep the Role: it still allows
  the `configmaps` access earlier releases use. Earlier releases cannot delete
  the registry Secrets they create under this Role; remove leftover ones as
  shown below.

### Removing objects left by earlier jobs

The agent does not remove job ConfigMaps from earlier releases or job Pods
left behind, for example by an agent restart during a job. Remove them by
label:

```bash
NS=<agent namespace>
SELECTOR=app.kubernetes.io/name=workflow-runner,app.kubernetes.io/component=pulumi-workflow

kubectl -n "$NS" get configmaps,pods -l "$SELECTOR"
kubectl -n "$NS" get secrets -l "$SELECTOR" --field-selector type=kubernetes.io/dockerconfigjson
```

- Delete the ConfigMaps only when no job Pod is `Pending`: a job that has not
  started yet fails without its ConfigMap.
- Delete a registry Secret `<pod name>-registry` only when the Pod
  `<pod name>` no longer exists.
- Never delete job Pods with `--cascade=orphan`, which leaves their Secrets
  with no owner.
- Make sure `PULUMI_AGENT_DEBUG_POD` is not set on the agent. With it, every
  job leaves its Pod behind, with its Secrets or its ConfigMap.

```bash
kubectl -n "$NS" delete pods -l "$SELECTOR" --field-selector=status.phase=Succeeded
kubectl -n "$NS" delete pods -l "$SELECTOR" --field-selector=status.phase=Failed
kubectl -n "$NS" delete pod <name of a Pending pod older than a few minutes>
kubectl -n "$NS" delete configmaps -l "$SELECTOR"
kubectl -n "$NS" delete secret <pod name>-registry
```

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

## Custom CA Certificates

If your environment uses a corporate proxy, internal PKI, or self-signed
certificates, both the agent and worker pods need to trust your custom root CA
certificates. The agent runs on Alpine Linux, where the system CA store is
located at `/etc/ssl/certs/ca-certificates.crt`. Mounting a custom CA bundle
at that path overrides the system trust store.

### Creating the CA Certificate Secret

Create a Kubernetes secret containing your CA certificate bundle. The secret
key **must** be `ca-certificates.crt`.

```bash
# From a single PEM file
kubectl create secret generic ca-certificates \
  --namespace <agent-namespace> \
  --from-file=ca-certificates.crt=/path/to/your/ca-bundle.crt
```

**Important:** If you only include your custom CA, the entire system trust
store is replaced. This means the agent and worker pods will no longer trust
public CAs (e.g., for `api.pulumi.com` or `github.com`). Combine your custom
CA with the system CAs:

```bash
# Extract system CAs from a running agent pod
kubectl exec -n <namespace> <agent-pod> -- \
  cat /etc/ssl/certs/ca-certificates.crt > system-cas.crt

# Combine system CAs with your custom CA
cat system-cas.crt your-custom-ca.crt > combined-ca-bundle.crt

# Create the secret with the combined bundle
kubectl create secret generic ca-certificates \
  --namespace <agent-namespace> \
  --from-file=ca-certificates.crt=combined-ca-bundle.crt
```

See [`ca-secret.yaml`](./ca-secret.yaml) for a YAML template.

The secret **must exist before** deploying the agent. If the secret is
referenced but missing, pods will fail to start with volume mount errors.

### Configuring the Agent Pod

Set the `caCertificateSecretName` config value to mount the CA certificate
secret into the agent deployment:

```bash
pulumi config set caCertificateSecretName ca-certificates
```

This adds a volume mount at `/etc/ssl/certs/ca-certificates.crt` in the agent
container. The Go agent uses the default system CA store, so no additional
environment variables are needed.

### Configuring Worker Pods

Worker pods are configured separately via the pod template (strategic merge
patch). Add the CA certificate volume and volume mount to your pod
specification:

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
    - name: pulumi-workflow
      volumeMounts:
        - name: ca-certificates
          mountPath: /etc/ssl/certs/ca-certificates.crt
          subPath: ca-certificates.crt
          readOnly: true
  volumes:
    - name: ca-certificates
      secret:
        secretName: ca-certificates
        defaultMode: 420
```

See [`examples/pod.yaml`](./examples/pod.yaml) for a complete example. Both
the agent and worker pods can reference the same secret.

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
the namespace exists before namespaced resources are created. To upgrade an
existing install to Secret-based job credentials, follow
[Upgrading rendered or hand-maintained manifests](#upgrading-rendered-or-hand-maintained-manifests)
instead:

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
pulumi config set agentImage pulumi/customer-managed-workflow-agent:latest
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
