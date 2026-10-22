# Customer Managed Workflow Agent

Release mirror and deployment tooling for Pulumi's customer-managed workflow agent. This repository does not contain the agent source and builds nothing. Pulumi's release pipeline builds, signs and publishes the agent to `get.pulumi.com` and Docker Hub, and this repository mirrors each release to GitHub. It also contains `install.sh`, the Kubernetes components and the AMI tooling.

## Validating Changes

`.github/workflows/tests.yaml` runs these on every pull request. Run the ones for what you change:

```bash
tests/run.sh                                         # install.sh and scripts/, with stubbed downloads
shellcheck install.sh scripts/*.sh tests/*.sh
actionlint .github/workflows/*.y*ml
cd kubernetes && yarn install && npx tsc --noEmit    # yarn
cd kubernetes-dind && yarn install && npx -p typescript@5.9.3 tsc --noEmit             # yarn
cd agent-images/agent-setup && npm install && npx -p typescript@5.9.3 tsc --noEmit     # npm
```

Changes to `install.sh` or `scripts/` should come with a test in `tests/`.

## Key Constraints

- `install.sh` and its download URLs (`github.com/.../releases/download/vX.Y.Z/customer-managed-workflow-agent-vX.Y.Z-<os>-<x64|arm64>.tar.gz`) are used by the Pulumi Cloud console and the AMI. Keep archive names and the release layout stable.
- The GitHub release is the "shipped" marker for the release pipeline. Dispatching `mirror-release.yaml` for an existing release is a no-op, including promotion and the docs dispatch. If the promote or docs job fails, re-run that job from the original workflow run.
- On the release, the checksums file is renamed to `checksums.txt` / `checksums.txt.sig`.
- Only the highest final version is marked latest and dispatched to docs (`scripts/promote-latest.sh`; the docs job rechecks with `--check`), so mirroring or retrying an older version never moves `latest-version` backwards. Runs are serialized across versions.
- Releases after v2.3.0 are signed keyless; v2.3.0 and earlier were signed with a key and verify with `cosign.pub`.
- Images are published under two names: `pulumi/customer-managed-workflow-agent` and the pre-v2.0.0 `pulumi/customer-managed-deployment-agent`.
- Agent code and its changelog live with the agent source. `CHANGELOG.md` here is history through v2.3.0.

## Forbidden Patterns

- **Never build or re-sign agent binaries or images here** -- the release pipeline is the only publisher. The mirror attaches the CDN files unchanged.
- **Never loosen the mirror's input validation** (`version` format, exact `base_url`) or its signer identity.
- **Never create release tags by hand** -- `mirror-release.yaml` creates them. A tag without a release makes the workflow fail.
- **Never commit secrets** (tokens, private keys).
- **Never mix package managers** -- yarn in `kubernetes/` and `kubernetes-dind/`, npm in `agent-images/agent-setup/`.
