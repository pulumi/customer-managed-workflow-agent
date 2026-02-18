import * as pulumi from "@pulumi/pulumi";
import * as k8s from "@pulumi/kubernetes";
import { PulumiSelfHostedAgentComponent } from "./agent";

const pulumiConfig = new pulumi.Config();

// Optional: render YAML to a directory instead of deploying to cluster
const renderYamlToDirectory = pulumiConfig.get("renderYamlToDirectory");

const k8sProvider = new k8s.Provider("k8s-provider", {
    ...(renderYamlToDirectory && { renderYamlToDirectory }),
});

const providerOpts = { provider: k8sProvider };

export const ns = new k8s.core.v1.Namespace(
    "stack-namespace",
    {
        metadata: { name: pulumiConfig.require("agentNamespace") },
    },
    providerOpts,
);

// Configure this service account for your cloud settings
const workerServiceAccount = new k8s.core.v1.ServiceAccount(
    "workflow-agent",
    {
        metadata: {
            namespace: ns.metadata.name,
        },
    },
    providerOpts,
);

const agentOpts: pulumi.ComponentResourceOptions = {
    dependsOn: [ns],
    providers: [k8sProvider],
};

const agent = new PulumiSelfHostedAgentComponent(
    "self-hosted-agent",
    {
        namespace: ns,
        imageName: pulumiConfig.require("agentImage"),
        selfHostedAgentsAccessToken: pulumiConfig.requireSecret(
            "selfHostedAgentsAccessToken",
        ),
        selfHostedServiceURL:
            pulumiConfig.get("selfHostedServiceURL") ?? "https://api.pulumi.com",
        imagePullPolicy: pulumiConfig.get("agentImagePullPolicy") || "Always",
        agentReplicas: pulumiConfig.getNumber("agentReplicas") || 3,
        workerServiceAccount,
        enableServiceMonitor:
            pulumiConfig.getBoolean("enableServiceMonitor") || false,
        caCertificateSecretName: pulumiConfig.get("caCertificateSecretName"),
    },
    agentOpts,
);

// Export key resources for reference
export const agentService = agent.agentService;
export const serviceMonitor = agent.serviceMonitor;
