### Improvements

* Add Helm chart distribution for the customer-managed deployment agent
  * New `helm-gen` tool converts Pulumi-rendered Kubernetes manifests into a
    production Helm chart
  * Worker ServiceAccount template with cloud SA annotation support
    (IRSA, Workload Identity)
  * Install-time validation requiring `agent.token` or
    `agent.existingSecretName`
  * Readiness probe support for safe rolling updates
  * Flexible deployment strategy configuration (RollingUpdate with
    maxSurge/maxUnavailable, or Recreate)
  * `image.registry` field for air-gapped and private registry environments
  * CI workflow with Helm lint, template validation, and drift detection
    between `kubernetes/` and `helm-gen/`

### Bug Fixes
