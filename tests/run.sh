#!/usr/bin/env bash
# Run every *_test.sh in this directory, or the ones named as arguments.
set -uo pipefail

cd "$(dirname "$0")" || exit 1
[ $# -gt 0 ] || set -- *_test.sh
status=0
for t in "$@"; do
  echo "# ${t}"
  "./${t#./}" || status=1
done
exit $status
