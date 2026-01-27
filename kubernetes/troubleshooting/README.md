# Troubleshooting

## Steps

The order of troubleshooting should be as follows:

1. Identify the pods you're looking for - primarily `pulumi-workflow-runner-*` pods
    > **Note:** The agent pool has logs, as well, it's probably best to have a terminal open running:
    `kubectl logs -n <namespace> -l 'app.kubernetes.io/name=customer-managed-workflow-agent' -f`

2. Run these commands in separate terminal windows:

    ```bash
    # Watch for events in the namespace
    kubectl events -n <namespace>

    # Monitor pod details
    kubectl describe pod -n <namespace> <podname>
    ```

3. Once the pod is running:

   ```bash
   # Stream the pod logs
   kubectl logs -n <namespace> <podname>
   ```

4. Recommendations:
   - Set the deployment to 1 worker until everything is working properly
   - This makes troubleshooting easier by reducing confusion about which agent kicks off which runner
   - Implementing the MutatingWebhook should help place pods on the correct nodes

## Scripts

### pod_monitoring.sh

This script provides general-purpose pod monitoring with flexible pod selection using labels or name patterns. It creates separate log files per pod and monitors resource usage, health, pod details, and logs until completion criteria are met.

```bash
# Monitor pods with a specific label
./pod_monitoring.sh -n my-namespace -s "app=myapp" -i 30

# Monitor pods matching a name pattern
./pod_monitoring.sh -n my-namespace -p "web-.*" -t 7200

# Monitor until specific status
./pod_monitoring.sh -n my-namespace -s "job-name=backup" -c "Succeeded,Failed"
```

Options:

- `-n` - Namespace (default: cmwa)
- `-s` - Label selector (e.g. "app=myapp")
- `-p` - Pod name pattern (regex)
- `-i` - Monitoring interval in seconds (default: 1)
- `-d` - Output directory (default: ./pod_logs)
- `-t` - Maximum monitoring duration in seconds (default: 3600)
- `-c` - Completion statuses to exit on (default: "Succeeded,Completed,Failed")

### pulumi_workflow_pods_monitor.sh

This script specifically monitors Pulumi workflow pods by targeting the label 'app.kubernetes.io/component=pulumi-workflow'. It logs detailed information including resource usage, network connectivity, pod descriptions, and events to help troubleshoot Pulumi workflow issues.

```bash
# Monitor Pulumi workflow pods with default settings
./pulumi_workflow_pods_monitor.sh

# Monitor with custom namespace and intervals
./pulumi_workflow_pods_monitor.sh -n pulumi-agent-pool -i 30 -m 5 -d /var/log/monitoring
```

Options:

- `-n` - Namespace (default: cmwa)
- `-i` - Monitoring interval in seconds (default: 1)
- `-m` - Maximum number of logs to keep (default: 10)
- `-d` - Log directory (default: logs)

### start_debug_pod.sh

This script creates a debug pod based on an existing Pulumi workflow pod. It's particularly useful when `PULUMI_AGENT_DEBUG_POD` is set to `true` and you need to interactively debug a workflow. The script captures the original container command and provides it in the debug environment.

```bash
# Start a debug pod with default settings
./start_debug_pod.sh

# Start a debug pod in a specific namespace
./start_debug_pod.sh -n pulumi-agent-pool

# Start a debug pod with custom label and container name
./start_debug_pod.sh -n default -l "app=custom-workflow" -c main-container -s troubleshoot
```

Options:

- `-n` - Namespace (default: cmwa)
- `-l` - Label selector (default: app.kubernetes.io/component=pulumi-workflow)
- `-c` - Container name (default: pulumi-workflow)
- `-s` - Debug pod suffix (default: debug)

### kyverno.yaml

This file contains a Kyverno policy that automatically enforces node affinity for pods in the cmwa namespace. It ensures all pods in the specified namespace are scheduled on the designated node(s), solving potential scheduling issues.

> **Note:** This is only an example configuration. You will need to modify the node hostname identified by `<PLACEHOLDER>` and namespace values to match your specific environment before applying.

```bash
# Apply the Kyverno policy
kubectl apply -f kyverno.yaml

# Verify the policy is active
kubectl get clusterpolicy
```

The policy has two key functions:

1. **Mutation**: Automatically adds a nodeSelector to any pod created in the cmwa namespace
2. **Validation**: Ensures pods have the required nodeSelector (prevents deployment without the selector)

To use this policy:

- Install [Kyverno](https://kyverno.io/docs/installation/) in your cluster first
- Modify the hostname value in the file to match your target node
- Apply the policy using kubectl

Learn more about Kyverno:

- [Kyverno documentation](https://kyverno.io/docs/)
- [Mutating resources](https://kyverno.io/docs/writing-policies/mutate/)
- [Validating resources](https://kyverno.io/docs/writing-policies/validate/)
- [Kyverno CLI](https://kyverno.io/docs/kyverno-cli/)
