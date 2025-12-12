# Building an AWS AMI

In this folder are tools to build an AWS AMI with the correct software to run a Pulumi Workflow Agent and then deploy an EC2 instance with that AMI on it.

Although the Packer template and the Pulumi program will work, they are meant as a reference to help you get started. You should customize them to comply with your company's security and operational guidelines.

Note: workflow runners are only available to Pulumi customers on the Business Critical subscription. For more information, visit the [Pulumi Contact us page](https://www.pulumi.com/contact/) and select "I want to talk to someone in Sales" from the drop down.

If you have any questions about this, then please visit our [Support Portal](https://support.pulumi.com).

## Prerequisites

You must have the following software installed:

* Pulumi CLI - https://www.pulumi.com/docs/install/
* NodeJS - https://nodejs.org/
* Hashicorp Packer - https://developer.hashicorp.com/packer/install?product_intent=packer

You should also have created a Deployment Runner pool by logging into [Pulumi Cloud](https://app.pulumi.com) as an administrator of your organization, going to the Settings section of the menu and clicking on the "Deployment runners" link. 

Next click on the "Add pool" button. Enter a name and (optional) description and click "Create pool". This will then show you a workflow runner access token that looks like this:

![Image showing a workflow runner access token](img/access-token.png)

Make a note of this as you will need it when you create a runner later on and once you refresh or leave this page you will not see it again and have to create a new runner pool.

You will also need access to an AWS account and the correct permissions to set up an AMI. You can find this information in the [Packer documentation](https://developer.hashicorp.com/packer/integrations/hashicorp/amazon#iam-task-or-instance-role).

## Creating the AMI

### Variables

There are a number of variables available to allow you to customize some of the setup of the AMI:

- `ami_prefix`
  - Prefix the name of AMI with the contents of this variable
  - Default: `pulumi-workflow-agent`
- `pulumi_version`
  - Version of the [Pulumi docker container](https://hub.docker.com/r/pulumi/pulumi) to download. It must be a [valid version of Pulumi](https://www.pulumi.com/docs/install/versions/)
  - Default: `latest`
- `setup_region`
  - AWS region to create the AMI in
  - Default: `us-west-2`
- `setup_instance_type`
  - AWS EC2 instance type to create the image on. This doesn't have to be the image type that the image is run on later.
  - Default: `t3.small`

You can either update the default values in the Packer template or see [the Packer docs](https://developer.hashicorp.com/packer/guides/hcl/variables#assigning-variables) on how to run the Packer CLI with different values.

### Steps

1. Clone this repo: `git clone https://github.com/pulumi/customer-managed-workflow-agent.git`
1. Change to the correct sub-folder folder: `cd customer-managed-workflow-agent/packer`
1. Run `packer init .` to initialize the setup
1. Run `packer validate .` to ensure that the template is correct
1. Run `packer build .` to build the AMI

When this is finished, it will give you the AMI Id to use. Please note that this will not overwrite your AMI or delete older versions. It is up to you to perform this task.

## Creating a runner in AWS EC2

Once the AMI has been created then you can create an EC2 instance using it. 

### Steps

1. From the root of this repository, change to the correct directory: `cd agent-setup`
1. Create a new stack: `pulumi stack init {stackname}`
1. Run the following commands to add the relevant configuration values:
    - `pulumi config set aws:region {region}` - This does not have to be the same region that you built the AMI in
    - `pulumi config set runnerAccessToken {token} --secret` - this is the access token that you were shown when you created the runner pool at the beginning of this process
    - (Optional) `pulumi config set vpcId {vpcid}` - the VPC you want the EC2 instance to run in. If you don't set this it will run in the default VPC for your account.
    - (Optional) `pulumi config set instanceType {instanceType}` - the instance type of the EC2 instance you want to run. If you don't set this it will default to `t3.small`.
1. Run `pulumi up` and select the `yes` option if you are happy with the set up.
