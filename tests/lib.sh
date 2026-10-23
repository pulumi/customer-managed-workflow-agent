# shellcheck shell=bash
# shellcheck disable=SC2034 # Variables here are used by the test files that source this one.
# Shared helpers for the tests in this directory. Source it; don't run it.
#
# Each test runs in a fresh case directory with stub curl and cosign commands first on PATH:
#   - curl serves files from $CASE/web/<host>/<path> and logs every URL it is asked for to $CASE/curl.log.
#     A missing file fails like `curl --fail` does (exit 22).
#   - cosign logs its arguments to $CASE/cosign.log, fails if the bundle or blob is missing, and otherwise exits
#     with $COSIGN_EXIT (default 0).
# Bash 3.2 compatible, so the tests also run on macOS.

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
FAILURES=0
TESTS=0

# Platform suffix install.sh picks for this machine.
case $(uname) in
  Linux) PLATFORM_OS=linux ;;
  Darwin) PLATFORM_OS=darwin ;;
esac
case $(uname -m) in
  x86_64) PLATFORM_ARCH=x64 ;;
  arm64 | aarch64) PLATFORM_ARCH=arm64 ;;
esac
PLATFORM="${PLATFORM_OS}-${PLATFORM_ARCH}"

# Start a test case: a new case directory, stubs on PATH, and a scratch HOME and TMPDIR.
new_case() {
  CASE=$(mktemp -d "${TEST_TMP}/case.XXXXXX")
  mkdir -p "$CASE/bin" "$CASE/web" "$CASE/home" "$CASE/tmp"

  cat >"$CASE/bin/curl" <<'STUB'
#!/usr/bin/env bash
out=""
url=""
while [ $# -gt 0 ]; do
  case $1 in
    -o) out=$2; shift ;;
    --header | -H | --retry) shift ;;
    -*) ;;
    *) url=$1 ;;
  esac
  shift
done
echo "$url" >>"$STUB_CASE/curl.log"
path="$STUB_CASE/web/${url#*://}"
[ -f "$path" ] || exit 22
if [ -n "$out" ]; then cp "$path" "$out"; else cat "$path"; fi
STUB

  cat >"$CASE/bin/cosign" <<'STUB'
#!/usr/bin/env bash
echo "$*" >>"$STUB_CASE/cosign.log"
# Like real cosign, fail when the bundle or the blob is missing.
prev=""
for arg in "$@"; do
  if [ "$prev" = --bundle ] && [ ! -f "$arg" ]; then exit 1; fi
  prev=$arg
done
[ -f "${!#}" ] || exit 1
exit "${COSIGN_EXIT:-0}"
STUB

  chmod +x "$CASE/bin/curl" "$CASE/bin/cosign"
  touch "$CASE/curl.log" "$CASE/cosign.log"
  export STUB_CASE=$CASE
  export COSIGN_EXIT=0
}

sha256() {
  if command -v sha256sum >/dev/null; then
    sha256sum "$1" | awk '{ print $1 }'
  else
    shasum -a 256 "$1" | awk '{ print $1 }'
  fi
}

# make_archive <dest.tar.gz> <reported-version>
# Builds an archive shaped like a real release, whose agent prints the given version.
make_archive() {
  local dest=$1 reported=$2 src
  src=$(mktemp -d "${TEST_TMP}/archive.XXXXXX")
  mkdir -p "$src/customer-managed-workflow-agent"
  printf '#!/bin/sh\necho "customer-managed-workflow-agent version v%s, git hash 0000000, built 2026-01-01"\n' \
    "$reported" >"$src/customer-managed-workflow-agent/customer-managed-workflow-agent"
  chmod +x "$src/customer-managed-workflow-agent/customer-managed-workflow-agent"
  for f in workflow-runner workflow-runner-embeddable pulumi-workflow-agent.yaml.sample; do
    echo "$f" >"$src/customer-managed-workflow-agent/$f"
  done
  tar -czf "$dest" -C "$src" customer-managed-workflow-agent
  rm -rf "$src"
}

pass() {
  TESTS=$((TESTS + 1))
  echo "ok   - $1"
}

fail() {
  TESTS=$((TESTS + 1))
  FAILURES=$((FAILURES + 1))
  echo "FAIL - $1"
  shift
  local line
  for line in "$@"; do echo "       $line"; done
}

# Print the tally and exit non-zero if anything failed. Call at the end of each test file.
finish() {
  rm -rf "$TEST_TMP"
  echo "$((TESTS - FAILURES))/${TESTS} passed"
  [ "$FAILURES" -eq 0 ]
}

TEST_TMP=$(mktemp -d)
