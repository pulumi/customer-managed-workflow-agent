#!/usr/bin/env bash
# Install a release with install.sh --verify-signature into a scratch HOME and check the agent's --version.
#
# Usage: smoke-test-install.sh <vX.Y.Z> <github|cdn> <scratch-home>
#
# The source is passed to install.sh --source, so the install never falls back to the other one.
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 <vX.Y.Z> <github|cdn> <scratch-home>" >&2
  exit 2
fi

VERSION=${1#v}
SOURCE=$2
export HOME=$3
mkdir -p "$HOME"

"$(dirname "$0")/../install.sh" --version "$VERSION" --source "$SOURCE" --verify-signature
output=$("$HOME/.pulumi/bin/customer-managed-workflow-agent/customer-managed-workflow-agent" --version)
echo "$output"

# Output looks like "customer-managed-workflow-agent version vX.Y.Z, git hash ..., built ...".
reported=$(sed -n 's/^.* version v\{0,1\}\([^ ,]*\).*$/\1/p' <<<"$output")
if [[ $reported != "$VERSION" ]]; then
  echo "::error::installed agent reports version '${reported}', expected '${VERSION}'" >&2
  exit 1
fi
