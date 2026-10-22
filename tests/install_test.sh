#!/usr/bin/env bash
# Tests for install.sh: download source and fallback, checksum and signature checks, and cleanup.
set -uo pipefail
# shellcheck source=tests/lib.sh
. "$(dirname "$0")/lib.sh"

GITHUB_BASE=github.com/pulumi/customer-managed-workflow-agent/releases/download
CDN_BASE=get.pulumi.com/releases/customer-managed-workflow-agent

# publish <version> <github|cdn> [checksum-override]
# Puts a release for this platform where the stub curl serves it.
publish() {
  local version=$1 where=$2 sum=${3:-} tarball dir checksums
  tarball="customer-managed-workflow-agent-v${version}-${PLATFORM}.tar.gz"
  if [ "$where" = github ]; then
    dir="$CASE/web/$GITHUB_BASE/v$version"
    checksums=checksums.txt
  else
    dir="$CASE/web/$CDN_BASE"
    checksums="customer-managed-workflow-agent-${version}-checksums.txt"
  fi
  mkdir -p "$dir"
  make_archive "$dir/$tarball" "$version"
  [ -n "$sum" ] || sum=$(sha256 "$dir/$tarball")
  printf '%s  %s\n%s  %s\n' "$sum" "$tarball" "$(printf '0%.0s' $(seq 64))" "customer-managed-workflow-agent-v${version}-other.tar.gz" \
    >"$dir/$checksums"
  echo "bundle" >"$dir/$checksums.sig"
}

# A previous install, to check that a rejected download leaves it alone.
previous_install() {
  mkdir -p "$CASE/home/.pulumi/bin/customer-managed-workflow-agent"
  echo previous >"$CASE/home/.pulumi/bin/customer-managed-workflow-agent/customer-managed-workflow-agent"
}

run_install() {
  HOME="$CASE/home" TMPDIR="$CASE/tmp" SHELL=/bin/false PATH="${RUN_PATH:-$CASE/bin:$PATH}" \
    sh "$REPO_ROOT/install.sh" "$@" >"$CASE/out" 2>&1
  RC=$?
}

installed_version() {
  "$CASE/home/.pulumi/bin/customer-managed-workflow-agent/customer-managed-workflow-agent" 2>/dev/null
}

previous_kept() {
  [ "$(cat "$CASE/home/.pulumi/bin/customer-managed-workflow-agent/customer-managed-workflow-agent")" = previous ]
}

tmp_empty() {
  [ -z "$(ls -A "$CASE/tmp")" ]
}

new_case
publish 2.4.0 github
run_install --version 2.4.0
if [ $RC -eq 0 ] && installed_version | grep -q 'version v2.4.0,' && grep -q "$GITHUB_BASE/v2.4.0/checksums.txt" "$CASE/curl.log" &&
  ! grep -q "$CDN_BASE" "$CASE/curl.log" && [ ! -s "$CASE/cosign.log" ] && tmp_empty; then
  pass "installs from GitHub and checks the checksum by default"
else
  fail "installs from GitHub and checks the checksum by default" "rc=$RC" "$(cat "$CASE/out")"
fi

new_case
publish 2.4.0 cdn
run_install --version 2.4.0
if [ $RC -eq 0 ] && installed_version | grep -q 'version v2.4.0,' &&
  grep -q "$CDN_BASE/customer-managed-workflow-agent-2.4.0-checksums.txt" "$CASE/curl.log"; then
  pass "falls back to get.pulumi.com when the GitHub download fails"
else
  fail "falls back to get.pulumi.com when the GitHub download fails" "rc=$RC" "$(cat "$CASE/out")"
fi

new_case
publish 2.4.0 cdn
run_install --version 2.4.0 --source github
if [ $RC -ne 0 ] && ! grep -q "$CDN_BASE" "$CASE/curl.log"; then
  pass "--source github does not fall back"
else
  fail "--source github does not fall back" "rc=$RC" "$(cat "$CASE/curl.log")"
fi

new_case
publish 2.4.0 github
publish 2.4.0 cdn
run_install --version 2.4.0 --source cdn
if [ $RC -eq 0 ] && ! grep -q "$GITHUB_BASE" "$CASE/curl.log"; then
  pass "--source cdn never contacts GitHub"
else
  fail "--source cdn never contacts GitHub" "rc=$RC" "$(cat "$CASE/curl.log")"
fi

new_case
run_install --version 2.4.0 --source s3
if [ $RC -ne 0 ] && grep -q "must be github or cdn" "$CASE/out" && [ ! -s "$CASE/curl.log" ]; then
  pass "rejects an unknown --source"
else
  fail "rejects an unknown --source" "rc=$RC" "$(cat "$CASE/out")"
fi

new_case
publish 2.4.0 github
run_install --version 2.4.0 --verify-signature
if [ $RC -eq 0 ] && grep -q -- "--bundle .*checksums.txt.sig" "$CASE/cosign.log" &&
  grep -q -- "--certificate-identity https://github.com/pulumi/pulumi-service/.github/workflows/update-stack.yml@refs/heads/production" "$CASE/cosign.log" &&
  grep -q -- "--certificate-oidc-issuer https://token.actions.githubusercontent.com" "$CASE/cosign.log"; then
  pass "--verify-signature verifies the checksums file against the release signer"
else
  fail "--verify-signature verifies the checksums file against the release signer" "rc=$RC" "$(cat "$CASE/cosign.log")"
fi

new_case
publish 2.4.0 github
previous_install
export COSIGN_EXIT=1
run_install --version 2.4.0 --verify-signature
if [ $RC -ne 0 ] && grep -q "signature did not verify" "$CASE/out" && previous_kept && tmp_empty; then
  pass "rejects a bad signature, keeps the previous install, and cleans up"
else
  fail "rejects a bad signature, keeps the previous install, and cleans up" "rc=$RC" "$(cat "$CASE/out")" "$(ls -A "$CASE/tmp")"
fi

new_case
publish 2.4.0 github "$(printf 'a%.0s' $(seq 64))"
previous_install
run_install --version 2.4.0
if [ $RC -ne 0 ] && grep -q "checksum mismatch" "$CASE/out" && previous_kept && tmp_empty; then
  pass "rejects a checksum mismatch, keeps the previous install, and cleans up"
else
  fail "rejects a checksum mismatch, keeps the previous install, and cleans up" "rc=$RC" "$(cat "$CASE/out")" "$(ls -A "$CASE/tmp")"
fi

new_case
publish 2.4.0 github
printf '%s  %s\n' "$(printf 'b%.0s' $(seq 64))" "unrelated.tar.gz" >"$CASE/web/$GITHUB_BASE/v2.4.0/checksums.txt"
previous_install
run_install --version 2.4.0
if [ $RC -ne 0 ] && grep -q "is not listed in" "$CASE/out" && previous_kept; then
  pass "rejects an archive missing from the checksums file"
else
  fail "rejects an archive missing from the checksums file" "rc=$RC" "$(cat "$CASE/out")"
fi

new_case
publish 2.4.0 github "$(printf 'a%.0s' $(seq 64))"
run_install --version 2.4.0 --skip-checksum
if [ $RC -eq 0 ] && ! grep -q "checksums.txt" "$CASE/curl.log"; then
  pass "--skip-checksum installs without fetching checksums"
else
  fail "--skip-checksum installs without fetching checksums" "rc=$RC" "$(cat "$CASE/out")"
fi

new_case
run_install --version 2.4.0 --verify-signature --skip-checksum
if [ $RC -ne 0 ] && grep -q "can't be combined with --skip-checksum" "$CASE/out"; then
  pass "rejects --verify-signature with --skip-checksum"
else
  fail "rejects --verify-signature with --skip-checksum" "rc=$RC" "$(cat "$CASE/out")"
fi

new_case
rm "$CASE/bin/cosign"
RUN_PATH="$CASE/bin:/usr/bin:/bin" run_install --version 2.4.0 --verify-signature
if [ $RC -ne 0 ] && grep -q "requires cosign" "$CASE/out"; then
  pass "--verify-signature requires cosign"
else
  fail "--verify-signature requires cosign" "rc=$RC" "$(cat "$CASE/out")"
fi

for old in 2.3.0 2.2.0 1.4.1; do
  new_case
  run_install --version "$old" --verify-signature
  if [ $RC -ne 0 ] && grep -q "supports releases after v2.3.0" "$CASE/out"; then
    pass "--verify-signature rejects key-signed release $old"
  else
    fail "--verify-signature rejects key-signed release $old" "rc=$RC" "$(cat "$CASE/out")"
  fi
done

finish
