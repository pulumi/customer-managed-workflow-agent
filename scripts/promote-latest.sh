#!/usr/bin/env bash
# Mark the highest final (non-prerelease) release as GitHub's latest, and report whether that is <vX.Y.Z>.
#
# Usage: promote-latest.sh <vX.Y.Z>     (prints promoted=true|false on stdout; GITHUB_REPOSITORY must be set)
#
# It always promotes the highest version rather than the one just created, so mirror runs that finish out of order
# converge on the same answer, and an older version never becomes latest.
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <vX.Y.Z>" >&2
  exit 2
fi

VERSION=$1
REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY must be set}

highest=$(gh release list --repo "$REPO" --exclude-pre-releases --exclude-drafts --limit 1000 --json tagName --jq '.[].tagName' |
  grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -n 1 || true)
if [[ -z $highest ]]; then
  echo "::error::no final releases found in $REPO" >&2
  exit 1
fi

gh release edit "$highest" --repo "$REPO" --latest >/dev/null
echo "Latest release: $highest" >&2

if [[ $highest == "$VERSION" ]]; then
  echo "promoted=true"
else
  echo "promoted=false"
fi
