# Installing the self managed agent into Kubernetes using Docker-in-Docker

## Prerequisites 

* Use a published agent image, e.g. `pulumi/customer-managed-workflow-agent:<tag>` from [Docker Hub](https://hub.docker.com/r/pulumi/customer-managed-workflow-agent)
* Use `pulumi config` to set imageName, namespace, and access token

* Install
```bash
pulumi config set agentNamespace <desired namespace>
pulumi config set --secret pulumi-agent-kubernetes:selfHostedAgentsAccessToken <access token>
pulumi config set agentImage <imageTag>

pulumi up
```

Optionally you can set an `agentImagePullPolicy` to a [Kubernetes supported value](https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy), which defaults to `Always`.

This folder also contains a [raw kubernetes yaml file](./raw_deployment.yaml) for reference.
