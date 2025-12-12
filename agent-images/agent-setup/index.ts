import * as pulumi from "@pulumi/pulumi";
import * as aws from "@pulumi/aws";

export = async () => {

  const config = new pulumi.Config();
  const accessToken = config.requireSecret("runnerAccessToken");
  const vpcId = config.get("vpdId");
  const amiPrefix = config.get("amiPrefix") || "pulumi-workflow-agent";
  const instanceType = config.get("instanceType") || "t3.small";

  let vpcArgs: aws.ec2.GetVpcArgs = {default: true};

  if(vpcId !== undefined) {
    vpcArgs = {
      id: vpcId
    }
  }

  const vpc = await aws.ec2.getVpc(vpcArgs);
  const subnets = await aws.ec2.getSubnetsOutput({
    filters: [
      {
        name: "vpc-id",
        values: [vpc.id],
      },
    ],
  });

  const ami = await aws.ec2.getAmi({
    filters: [
      {
        name: "name",
        values: [`${amiPrefix}*`],
      },
    ],
    mostRecent: true,
    owners: ["self"],
  });

  const agentSg = new aws.ec2.SecurityGroup("agentSg", {
    vpcId: vpc.id,
    egress: [
      { toPort: 0, fromPort: 0, protocol: "-1", cidrBlocks: ["0.0.0.0/0"] },
    ],
  });

  const userData = pulumi.interpolate`#!/bin/bash
  sudo echo -e "token: \"${accessToken}\"" >> /home/ubuntu/.pulumi/bin/customer-managed-workflow-agent/pulumi-workflow-agent.yaml
  sudo systemctl start workflow_agent.service`;

  const instance = new aws.ec2.Instance("agent", {
    ami: ami.id,
    vpcSecurityGroupIds: [agentSg.id],
    instanceType: instanceType,
    subnetId: subnets.ids.apply(ids => ids[0]),
    userData: userData,
    tags: {
      "Name": "workflow-agent"
    }
  });
};
