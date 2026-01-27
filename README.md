# Pulumi Customer Managed Workflow Agent

This repository contains the release artifacts for Customer Managed Workflow Agents.

See [Deployments](https://www.pulumi.com/docs/pulumi-cloud/deployments/) for additional details about this product.

## Verifying release artifacts

### Checksum signature

This command requires [cosign](https://docs.sigstore.dev/system_config/installation/) to be installed.

Run the following command to verify the signature of the release checksums:

```bash
cosign verify-blob \
  --key <path to cosign.pub> \
  --signature checksums.txt.sig \
  checksums.txt
```

If the signature validates, you will see the following message:

```bash
Verified OK
```

### Checksums

You can validate the file checksums by running the following command:

```bash
sha256sum --ignore-missing -c checksums.txt
```

If the checksums match the downloaded files, you will see output like this:

```bash
customer-managed-workflow-agent_1.0.0_darwin_amd64.tar.gz: OK
customer-managed-workflow-agent_1.0.0_darwin_arm64.tar.gz: OK
customer-managed-workflow-agent_1.0.0_linux_amd64.tar.gz: OK
customer-managed-workflow-agent_1.0.0_linux_arm64.tar.gz: OK
```

## Local testing

The setup can be tested locally by overriding the `REPO_OWNER` environment variable. In the official Github Actions
workflows, this environment variable is set to `pulumi` as the Docker Hub owner of the images. By setting this
to your own Docker Hub account name, you can test images of work in progress, which should be committed locally as
GoReleaser will not publish images from a Git repository with changed files.

First run a `docker login` to have valid credentials to your Docker Hub account.

```sh
export REPO_OWNER=<your-docker-hub-name>
export BUILD_STAMP=$(cd ./pulumi-service && date -u '+%Y-%m-%d_%I:%M:%S%p')
export BUILD_GIT_HASH=$(cd ./pulumi-service && git rev-parse HEAD)
goreleaser release --skip=validate,sign,archive,publish --clean
```

This should build a multi-arch image for `amd64` and `arm64` architectures and publish it to your own Docker Hub account.
