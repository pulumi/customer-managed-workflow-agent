# Pulumi Customer Managed Workflow Agent

This repository contains the release artifacts for Customer Managed Workflow Agents.

> **Note:** Insights workflows are supported in version v2.2.0+.

See [Customer-managed agents](https://www.pulumi.com/docs/deployments/deployments/customer-managed-agents/) for additional details about this product.

## Installing

```bash
curl -fsSL https://raw.githubusercontent.com/pulumi/customer-managed-workflow-agent/main/install.sh | sh
```

The installer downloads from this repository's GitHub releases, falling back to `get.pulumi.com`, and checks the
archive against the release checksums. Options:

- `--version X.Y.Z` installs a specific version instead of the latest.
- `--verify-signature` also verifies the checksums file's signature. Requires cosign; releases after v2.3.0 only.
- `--skip-checksum` skips the checksum check.
- `--source github` or `--source cdn` downloads only from GitHub or only from `get.pulumi.com`, with no fallback.

Pass options with `sh -s -- <options>` when piping the script.

## Verifying release artifacts

These commands require [cosign](https://docs.sigstore.dev/system_config/installation/) v2.5 or later.

Release archives and `checksums.txt` are signed keyless with [Sigstore](https://www.sigstore.dev/). Each `.sig`
file is a cosign bundle. Download the files from the release, then verify the signature of the checksums:

```bash
cosign verify-blob \
  --bundle checksums.txt.sig \
  --certificate-identity https://github.com/pulumi/pulumi-service/.github/workflows/update-stack.yml@refs/heads/production \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt
```

If the signature validates, you will see the following message:

```bash
Verified OK
```

The same command, with `<archive>.tar.gz.sig` and `<archive>.tar.gz`, verifies a single archive.

Then validate the file checksums:

```bash
sha256sum --ignore-missing -c checksums.txt
```

If the checksums match the downloaded files, you will see output like this:

```bash
customer-managed-workflow-agent-v2.3.0-darwin-arm64.tar.gz: OK
customer-managed-workflow-agent-v2.3.0-darwin-x64.tar.gz: OK
customer-managed-workflow-agent-v2.3.0-linux-arm64.tar.gz: OK
customer-managed-workflow-agent-v2.3.0-linux-x64.tar.gz: OK
```

### Releases up to v2.3.0

Releases v2.3.0 and earlier were signed with a key. Verify them with [`cosign.pub`](./cosign.pub) from this repository:

```bash
cosign verify-blob \
  --key cosign.pub \
  --signature checksums.txt.sig \
  checksums.txt
```

## Verifying container images

The `pulumi/customer-managed-workflow-agent` images on Docker Hub are signed by the same identity:

```bash
cosign verify \
  --certificate-identity https://github.com/pulumi/pulumi-service/.github/workflows/update-stack.yml@refs/heads/production \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  docker.io/pulumi/customer-managed-workflow-agent:<tag>
```

They also carry build provenance attestations, which the [GitHub CLI](https://cli.github.com/) verifies:

```bash
gh attestation verify oci://docker.io/pulumi/customer-managed-workflow-agent:<tag> \
  --owner pulumi \
  --bundle-from-oci
```

`gh attestation verify` needs you to be logged in to GitHub (`gh auth login`); any account works, and you don't
need to be a member of the `pulumi` organization. `cosign verify` needs no login.

Images v2.3.0 and earlier are not signed.

## How releases are made

The agent is built, signed and published by Pulumi's internal release pipeline, which uploads each release to
`https://get.pulumi.com/releases/customer-managed-workflow-agent/` and then runs this repository's
[Mirror Release](.github/workflows/mirror-release.yaml) workflow. That workflow verifies the files and creates the
GitHub release. Release notes are on each [GitHub release](https://github.com/pulumi/customer-managed-workflow-agent/releases).
