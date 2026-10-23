#!/usr/bin/env bash
# Tests for scripts/smoke-test-install.sh: it installs from exactly the source it is given, and the installed agent
# must report exactly the requested version.
set -uo pipefail
# shellcheck source=tests/lib.sh
. "$(dirname "$0")/lib.sh"

GITHUB_BASE=github.com/pulumi/customer-managed-workflow-agent/releases/download
CDN_BASE=get.pulumi.com/releases/customer-managed-workflow-agent

# publish_cdn <version> <reported-version>
publish_cdn() {
  local dir="$CASE/web/$CDN_BASE" tarball="customer-managed-workflow-agent-v$1-${PLATFORM}.tar.gz"
  mkdir -p "$dir"
  make_archive "$dir/$tarball" "$2"
  printf '%s  %s\n' "$(sha256 "$dir/$tarball")" "$tarball" >"$dir/customer-managed-workflow-agent-$1-checksums.txt"
  echo bundle >"$dir/customer-managed-workflow-agent-$1-checksums.txt.sig"
}

run_smoke() {
  TMPDIR="$CASE/tmp" SHELL=/bin/false PATH="$CASE/bin:$PATH" \
    "$REPO_ROOT/scripts/smoke-test-install.sh" "$@" "$CASE/home" >"$CASE/out" 2>&1
  RC=$?
}

new_case
publish_cdn 2.4.0 2.4.0
run_smoke v2.4.0 cdn
if [ $RC -eq 0 ] && ! grep -q "$GITHUB_BASE" "$CASE/curl.log" && [ -s "$CASE/cosign.log" ]; then
  pass "passes when the agent reports the requested version, using only the given source"
else
  fail "passes when the agent reports the requested version, using only the given source" "rc=$RC" "$(cat "$CASE/out")"
fi

for reported in 12.4.0 2.4.01 2.4.0-rc.1; do
  new_case
  publish_cdn 2.4.0 "$reported"
  run_smoke v2.4.0 cdn
  if [ $RC -ne 0 ] && grep -q "reports version '$reported'" "$CASE/out"; then
    pass "fails when v2.4.0 reports $reported"
  else
    fail "fails when v2.4.0 reports $reported" "rc=$RC" "$(cat "$CASE/out")"
  fi
done

new_case
publish_cdn 2.4.0 2.4.0
run_smoke v2.4.0 github
if [ $RC -ne 0 ] && ! grep -q "$CDN_BASE" "$CASE/curl.log"; then
  pass "the GitHub phase fails rather than falling back to get.pulumi.com"
else
  fail "the GitHub phase fails rather than falling back to get.pulumi.com" "rc=$RC" "$(cat "$CASE/curl.log")"
fi

finish
