import * as pulumi from "@pulumi/pulumi";
import * as kubernetes from "@pulumi/kubernetes";

export interface PulumiSelfHostedAgentComponentArgs {
    namespace: kubernetes.core.v1.Namespace;
    imageName: pulumi.Input<string>;
    imagePullPolicy: pulumi.Input<string>;
    selfHostedAgentsAccessToken: pulumi.Input<string>;
}

export class PulumiSelfHostedAgentComponent extends pulumi.ComponentResource {
    public readonly agentDeployment: kubernetes.apps.v1.Deployment;
    constructor(name: string, args: PulumiSelfHostedAgentComponentArgs, opts?: pulumi.ComponentResourceOptions) {
        super("pulumi-service:kubernetes:PulumiSelfHostedAgentComponentArgs", name, args, opts);

        const agentSecret = new kubernetes.core.v1.Secret("agent-secret", {
            metadata: {
                name: "agent-secret",
                namespace: args.namespace.metadata.name,
              },
              stringData: {
                "PULUMI_AGENT_TOKEN": args.selfHostedAgentsAccessToken,
              }
        });

        this.agentDeployment = new kubernetes.apps.v1.Deployment("workflow-agent-pool", {
            metadata: {
                name: "workflow-agent-pool",
                namespace: args.namespace.metadata.name,
                annotations: {
                    "app.kubernetes.io/name": "pulumi-workflow-agent-pool",
                },
            },
            spec: {
                replicas: 1,
                selector: {
                    matchLabels: {
                        app: "pulumi-workflow-agent-pool",
                    },
                },
                template: {
                    metadata: {
                        labels: {
                            app: "pulumi-workflow-agent-pool",
                            "app.kubernetes.io/name": "pulumi-workflow-agent-pool",
                        },
                    },
                    spec: {
                        containers: [
                        {
                            name: "agent",
                            image: args.imageName,
                            imagePullPolicy: args.imagePullPolicy,
                            env: [
                                {
                                    name: "DOCKER_HOST",
                                    value: "tcp://localhost:2375",
                                },
                                {
                                    name: "PULUMI_AGENT_SHARED_VOLUME_DIRECTORY",
                                    value: "/mnt/work",
                                },
                                {
                                    name: "PULUMI_AGENT_TOKEN",
                                    valueFrom: {
                                        secretKeyRef: {
                                            name: agentSecret.metadata.name,
                                            key: "PULUMI_AGENT_TOKEN",
                                        },
                                    },
                                },
                            ],
                            volumeMounts: [
                                {
                                    name: "agent-work",
                                    mountPath: "/mnt/work",
                                },
                            ],
                        },
                        {
                            name: "dind",
                            image: "docker:dind",
                            imagePullPolicy: "Always",
                            command: ["dockerd", "--host", "tcp://127.0.0.1:2375"],
                            securityContext: {
                                privileged: true,
                            },
                            volumeMounts: [
                                {
                                    name: "agent-work",
                                    mountPath: "/mnt/work",
                                },
                            ],
                        }],
                        volumes: [
                            {
                                name: "agent-work",
                                emptyDir: {},
                            },
                        ],
                    },
                },
            },
        });

        this.registerOutputs();
    }
}
