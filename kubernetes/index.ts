import * as pulumi from "@pulumi/pulumi";
import * as k8s from "@pulumi/kubernetes";
import { PulumiSelfHostedAgentComponent } from "./agent";

const pulumiConfig = new pulumi.Config();
export const ns = new k8s.core.v1.Namespace("stack-namespace", {
    metadata: { name: pulumiConfig.require("agentNamespace") },
});

// Configure this service account for your cloud settings
const workerServiceAccount = new k8s.core.v1.ServiceAccount("workflow-agent", {
    metadata: {
        namespace: ns.metadata.name,
    },
}, { parent: this });


const agent = new PulumiSelfHostedAgentComponent(
    "self-hosted-agent",
    {
        namespace: ns,
        imageName: pulumiConfig.require("agentImage"),
        selfHostedAgentsAccessToken: pulumiConfig.requireSecret("selfHostedAgentsAccessToken"),
        selfHostedServiceURL: pulumiConfig.get("selfHostedServiceURL") ?? "https://api.pulumi.com",
        imagePullPolicy: pulumiConfig.get("agentImagePullPolicy") || "Always",
        agentReplicas: pulumiConfig.getNumber("agentReplicas") || 3,
        workerServiceAccount,
        enableServiceMonitor: pulumiConfig.getBoolean("enableServiceMonitor") || false,
    },
    { dependsOn: [ns] },
);

// Export key resources for reference
export const agentService = agent.agentService;
export const serviceMonitor = agent.serviceMonitor;
