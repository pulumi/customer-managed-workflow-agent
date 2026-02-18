import * as pulumi from "@pulumi/pulumi";
import * as kubernetes from "@pulumi/kubernetes";
import type { V1Pod } from "@kubernetes/client-node";

export interface PulumiSelfHostedAgentComponentArgs {
    namespace: kubernetes.core.v1.Namespace;
    imageName: pulumi.Input<string>;
    imagePullPolicy: pulumi.Input<string>;
    agentReplicas: pulumi.Input<number>;
    selfHostedAgentsAccessToken: pulumi.Input<string>;
    selfHostedServiceURL: pulumi.Input<string>;
    workerServiceAccount?: kubernetes.core.v1.ServiceAccount;
    envVars?: kubernetes.types.input.core.v1.EnvVar[]
    agentNumCpus?: number;
    agentMemQuantity?: number;
    podTemplate?: V1Pod;
    enableServiceMonitor?: boolean;
    caCertificateSecretName?: string;
}

export class PulumiSelfHostedAgentComponent extends pulumi.ComponentResource {
    public readonly agentDeployment: kubernetes.apps.v1.Deployment;
    public readonly agentServiceAccount: kubernetes.core.v1.ServiceAccount;
    public readonly agentRole: kubernetes.rbac.v1.Role;
    public readonly agentRoleBinding: kubernetes.rbac.v1.RoleBinding;
    public readonly agentService: kubernetes.core.v1.Service;
    public readonly serviceMonitor?: kubernetes.apiextensions.CustomResource;

    labels = {
        "app.kubernetes.io/name": "customer-managed-workflow-agent",
    };

    constructor(name: string, args: PulumiSelfHostedAgentComponentArgs, opts?: pulumi.ComponentResourceOptions) {
        super("pulumi-service:kubernetes:PulumiSelfHostedAgentComponent", name, args, opts);

        const agentConfig = new kubernetes.core.v1.ConfigMap("agent-config", {
            metadata: {
                name: "agent-config",
                namespace: args.namespace.metadata.name,
                labels: this.labels,
            },
            data: {
                "PULUMI_AGENT_SERVICE_URL": args.selfHostedServiceURL,
                "PULUMI_AGENT_IMAGE": args.imageName,
                "PULUMI_AGENT_IMAGE_PULL_POLICY": args.imagePullPolicy,
                "worker-pod.json": JSON.stringify(args.podTemplate, null, 2),
            },
        }, { parent: this });

        const agentSecret = new kubernetes.core.v1.Secret("agent-secret", {
            metadata: {
                name: "agent-secret",
                namespace: args.namespace.metadata.name,
            },
            stringData: {
                "PULUMI_AGENT_TOKEN": args.selfHostedAgentsAccessToken,
            }
        }, { parent: this });

        this.agentServiceAccount = new kubernetes.core.v1.ServiceAccount("workflow-agent", {
            metadata: {
                namespace: args.namespace.metadata.name,
                labels: this.labels,
            },
        }, { parent: this });

        this.agentRole = new kubernetes.rbac.v1.Role("workflow-agent", {
            metadata: {
                namespace: args.namespace.metadata.name,
                labels: this.labels,
            },
            rules: [
                {
                    apiGroups: [""],
                    resources: ["pods", "pods/log", "configmaps"],
                    verbs: ["create", "get", "list", "watch", "update", "delete"],
                },
            ],
        }, { parent: this });

        this.agentRoleBinding = new kubernetes.rbac.v1.RoleBinding("workflow-agent", {
            metadata: {
                namespace: args.namespace.metadata.name,
                labels: this.labels,
            },
            subjects: [
                {
                    kind: "ServiceAccount",
                    name: this.agentServiceAccount.metadata.name,
                    namespace: args.namespace.metadata.namespace,
                },
            ],
            roleRef: {
                kind: "Role",
                name: this.agentRole.metadata.name,
                apiGroup: "rbac.authorization.k8s.io"
            }
        }, { parent: this });

        let workerServiceAccountEnvVar: kubernetes.types.input.core.v1.EnvVar = { name: "PULUMI_AGENT_SERVICE_ACCOUNT_NAME" }
        if (args.workerServiceAccount) {
            workerServiceAccountEnvVar = {
                name: "PULUMI_AGENT_SERVICE_ACCOUNT_NAME",
                value: args.workerServiceAccount.metadata.name,
            }
        }

        let agentNumCpusEnvVar: kubernetes.types.input.core.v1.EnvVar = {
            name: "PULUMI_AGENT_NUM_CPUS",
        }
        if (args.agentNumCpus) {
            agentNumCpusEnvVar = {
                name: "PULUMI_AGENT_NUM_CPUS",
                value: args.agentNumCpus.toString()
            }
        }

        let agentMemQuantityEnvVar: kubernetes.types.input.core.v1.EnvVar = {
            name: "PULUMI_AGENT_MEM_QUANTITY"
        }
        if (args.agentMemQuantity) {
            agentMemQuantityEnvVar = {
                name: "PULUMI_AGENT_MEM_QUANTITY",
                value: args.agentMemQuantity.toString()
            }
        }

        this.agentDeployment = new kubernetes.apps.v1.Deployment("workflow-agent-pool", {
            metadata: {
                name: "workflow-agent-pool",
                namespace: args.namespace.metadata.name,
                annotations: {
                    "app.kubernetes.io/name": "pulumi-workflow-agent-pool",
                },
                labels: this.labels,
            },
            spec: {
                replicas: args.agentReplicas,
                selector: {
                    matchLabels: this.labels,
                },
                template: {
                    metadata: {
                        labels: this.labels,
                    },
                    spec: {
                        serviceAccountName: this.agentServiceAccount.metadata.name,
                        containers: [
                            {
                                name: "agent",
                                image: args.imageName,
                                imagePullPolicy: args.imagePullPolicy,
                                env: [
                                    {
                                        name: "PULUMI_AGENT_DEPLOY_TARGET",
                                        value: "kubernetes",
                                    },
                                    {
                                        name: "PULUMI_AGENT_SHARED_VOLUME_DIRECTORY",
                                        value: "/mnt/work",
                                    },
                                    {
                                        name: "PULUMI_AGENT_SERVICE_URL",
                                        valueFrom: {
                                            configMapKeyRef: {
                                                name: agentConfig.metadata.name,
                                                key: "PULUMI_AGENT_SERVICE_URL",
                                            },
                                        },
                                    },
                                    {
                                        name: "PULUMI_AGENT_IMAGE",
                                        valueFrom: {
                                            configMapKeyRef: {
                                                name: agentConfig.metadata.name,
                                                key: "PULUMI_AGENT_IMAGE",
                                            },
                                        },
                                    },
                                    {
                                        name: "PULUMI_AGENT_IMAGE_PULL_POLICY",
                                        valueFrom: {
                                            configMapKeyRef: {
                                                name: agentConfig.metadata.name,
                                                key: "PULUMI_AGENT_IMAGE_PULL_POLICY",
                                            },
                                        },
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
                                    workerServiceAccountEnvVar,
                                    agentNumCpusEnvVar,
                                    agentMemQuantityEnvVar,
                                    ...(args.envVars || [])
                                ],
                                ports: [
                                    {
                                        name: "http",
                                        containerPort: 8080,
                                        protocol: "TCP",
                                    },
                                ],
                                volumeMounts: [
                                    {
                                        name: "agent-work",
                                        mountPath: "/mnt/work",
                                    },
                                    {
                                        name: "agent-config",
                                        mountPath: "/mnt/worker-pod.json",
                                        subPath: "worker-pod.json",
                                        readOnly: true,
                                    },
                                    ...(args.caCertificateSecretName ? [{
                                        name: "ca-certificates",
                                        mountPath: "/etc/ssl/certs/ca-certificates.crt",
                                        subPath: "ca-certificates.crt",
                                        readOnly: true,
                                    }] : []),
                                ],
                            },
                        ],
                        volumes: [
                            {
                                name: "agent-work",
                                emptyDir: {},
                            },
                            {
                                name: "agent-config",
                                configMap: {
                                    name: agentConfig.metadata.name,
                                }
                            },
                            ...(args.caCertificateSecretName ? [{
                                name: "ca-certificates",
                                secret: {
                                    secretName: args.caCertificateSecretName,
                                    defaultMode: 420,
                                },
                            }] : []),
                        ],
                    },
                },
            },
        }, { parent: this });

        // Create service to expose metrics and health endpoints
        this.agentService = new kubernetes.core.v1.Service("deployment-agent-service", {
            metadata: {
                name: "deployment-agent-service",
                namespace: args.namespace.metadata.name,
                labels: {
                    ...this.labels,
                    "app.kubernetes.io/component": "metrics",
                },
                annotations: {
                    "prometheus.io/scrape": "true",
                    "prometheus.io/port": "8080",
                    "prometheus.io/path": "/healthz",
                },
            },
            spec: {
                selector: this.labels,
                ports: [
                    {
                        name: "http",
                        port: 8080,
                        targetPort: 8080,
                        protocol: "TCP",
                    },
                ],
                type: "ClusterIP",
            },
        }, { parent: this });

        // Optionally create ServiceMonitor for Prometheus Operator
        if (args.enableServiceMonitor) {
            this.serviceMonitor = new kubernetes.apiextensions.CustomResource("deployment-agent-servicemonitor", {
                apiVersion: "monitoring.coreos.com/v1",
                kind: "ServiceMonitor",
                metadata: {
                    name: "deployment-agent-servicemonitor",
                    namespace: args.namespace.metadata.name,
                    labels: this.labels,
                },
                spec: {
                    selector: {
                        matchLabels: {
                            ...this.labels,
                            "app.kubernetes.io/component": "metrics",
                        },
                    },
                    endpoints: [
                        {
                            port: "http",
                            path: "/healthz",
                            interval: "30s",
                        },
                    ],
                },
            }, { parent: this });
        }

        this.registerOutputs();
    }
}
