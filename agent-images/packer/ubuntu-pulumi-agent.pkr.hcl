packer {
  required_plugins {
    amazon = {
      version = ">=1.3.0"
      source  = "github.com/hashicorp/amazon"
    }
  }
}

locals {
  timestamp = regex_replace(timestamp(), "[- TZ:]", "")
}

variable "ami_prefix" {
  type    = string
  default = "pulumi-workflow-agent"
}

variable "pulumi_version" {
  type = string
  default = "latest"
}

variable "setup_region" {
  type = string
  default = "us-west-2"
}

variable "setup_instance_type" {
  type = string
  default = "t3.small"
}

source "amazon-ebs" "ubuntu" {
  ami_name      = "${var.ami_prefix}-${local.timestamp}"
  instance_type = "${var.setup_instance_type}"
  region        = "${var.setup_region}"
  source_ami_filter {
    filters = {
      name                = "*ubuntu-jammy-22.04-amd64-server-*"
      root-device-type    = "ebs"
      virtualization-type = "hvm"
    }
    most_recent = true
    owners      = ["amazon"]
  }
  ssh_username                              = "ubuntu"
}

build {
  sources = [
    "source.amazon-ebs.ubuntu"
  ]

  provisioner "file" {
    source      = "workflow_agent.service_ubuntu"
    destination = "/home/ubuntu/workflow_agent.service"
  }

  provisioner "shell" {
    inline = [
      "echo Installing pre-requisites",
      "sudo apt-get update",
      "sudo apt-get install ca-certificates curl -y",
      "sudo install -m 0755 -d /etc/apt/keyrings",
      "sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc",
      "sudo chmod a+r /etc/apt/keyrings/docker.asc",
      "echo Adding docker repo to Apt sources",
      "echo \"deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo \"$VERSION_CODENAME\") stable\" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null",
      "sudo apt-get update",
      "echo Install docker",
      "sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin -y",
      "curl -fsSL https://raw.githubusercontent.com/pulumi/customer-managed-workflow-agent/main/install.sh | sh",
      "sudo cp /home/ubuntu/workflow_agent.service /etc/systemd/system/workflow_agent.service",
      "sudo docker pull pulumi/pulumi:${var.pulumi_version}"
    ]
  }
}
