#!/usr/bin/env bash
# Verify the signatures and checksums of a downloaded customer-managed-workflow-agent release.
#
# Usage: verify-release.sh <dir> <vX.Y.Z> <checksums-file>
#
# <dir> holds the four archives, <checksums-file>, and a .sig for each of them.
# Each .sig is a cosign bundle signed by Pulumi's release pipeline. CERT_IDENTITY and CERT_OIDC_ISSUER
# override the expected signer.
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 <dir> <vX.Y.Z> <checksums-file>" >&2
  exit 2
fi

DIR=$1
VERSION=$2
CHECKSUMS=$3

CERT_IDENTITY=${CERT_IDENTITY:-https://github.com/pulumi/pulumi-service/.github/workflows/update-stack.yml@refs/heads/production}
CERT_OIDC_ISSUER=${CERT_OIDC_ISSUER:-https://token.actions.githubusercontent.com}

if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "::error::invalid version '$VERSION'" >&2
  exit 1
fi

ARCHIVES=()
for platform in darwin-arm64 darwin-x64 linux-arm64 linux-x64; do
  ARCHIVES+=("customer-managed-workflow-agent-${VERSION}-${platform}.tar.gz")
done

verify_blob() {
  local file=$1
  echo "--- verifying signature of ${file}"
  cosign verify-blob \
    --bundle "${file}.sig" \
    --certificate-identity "$CERT_IDENTITY" \
    --certificate-oidc-issuer "$CERT_OIDC_ISSUER" \
    "$file"
}

cd "$DIR"

verify_blob "$CHECKSUMS"
for archive in "${ARCHIVES[@]}"; do
  verify_blob "$archive"
done

# sha256sum -c only checks the files it lists, so also require the list to be exactly the four archives.
echo "--- checking ${CHECKSUMS} lists exactly the expected archives"
diff <(printf '%s\n' "${ARCHIVES[@]}" | sort) <(awk '{ sub(/^\*/, "", $2); print $2 }' "$CHECKSUMS" | sort)
sha256sum -c "$CHECKSUMS"

echo "Verified ${VERSION}"
