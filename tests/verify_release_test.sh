#!/usr/bin/env bash
# Tests for scripts/verify-release.sh: every file's signature is checked, and the checksums file must list exactly
# the four archives, each matching.
set -uo pipefail
# shellcheck source=tests/lib.sh
. "$(dirname "$0")/lib.sh"

VERSION=v2.4.0
CHECKSUMS=customer-managed-workflow-agent-2.4.0-checksums.txt

# A release directory as the mirror workflow downloads it.
make_release() {
  local p f
  RELEASE="$CASE/release"
  mkdir -p "$RELEASE"
  : >"$RELEASE/$CHECKSUMS"
  for p in darwin-arm64 darwin-x64 linux-arm64 linux-x64; do
    f="customer-managed-workflow-agent-${VERSION}-${p}.tar.gz"
    echo "$p" >"$RELEASE/$f"
    echo bundle >"$RELEASE/$f.sig"
    printf '%s  %s\n' "$(sha256 "$RELEASE/$f")" "$f" >>"$RELEASE/$CHECKSUMS"
  done
  echo bundle >"$RELEASE/$CHECKSUMS.sig"
}

run_verify() {
  PATH="$CASE/bin:$PATH" "$REPO_ROOT/scripts/verify-release.sh" "$RELEASE" "$VERSION" "$CHECKSUMS" >"$CASE/out" 2>&1
  RC=$?
}

expect_reject() {
  local name=$1
  run_verify
  if [ $RC -ne 0 ]; then pass "$name"; else fail "$name" "verify-release.sh accepted it"; fi
}

new_case
make_release
run_verify
if [ $RC -eq 0 ] && [ "$(grep -c -- '--certificate-identity https://github.com/pulumi/pulumi-service/.github/workflows/update-stack.yml@refs/heads/production' "$CASE/cosign.log")" -eq 5 ]; then
  pass "accepts a valid release and checks all five signatures against the release signer"
else
  fail "accepts a valid release and checks all five signatures against the release signer" "rc=$RC" "$(cat "$CASE/out")"
fi

new_case
make_release
export COSIGN_EXIT=1
expect_reject "rejects a bad signature"

new_case
make_release
grep -v linux-arm64 "$RELEASE/$CHECKSUMS" >"$CASE/c" && mv "$CASE/c" "$RELEASE/$CHECKSUMS"
expect_reject "rejects a checksums file missing an archive"

new_case
make_release
head -1 "$RELEASE/$CHECKSUMS" >"$CASE/dup"
cat "$CASE/dup" >>"$RELEASE/$CHECKSUMS"
expect_reject "rejects a duplicate checksums entry"

new_case
make_release
echo extra >"$RELEASE/extra.tar.gz"
printf '%s  %s\n' "$(sha256 "$RELEASE/extra.tar.gz")" extra.tar.gz >>"$RELEASE/$CHECKSUMS"
expect_reject "rejects an extra checksums entry"

new_case
make_release
sed "s/^[0-9a-f]*  \(.*darwin-arm64.tar.gz\)$/invalid-digest  \1/" "$RELEASE/$CHECKSUMS" >"$CASE/c" && mv "$CASE/c" "$RELEASE/$CHECKSUMS"
expect_reject "rejects a malformed checksum line"

new_case
make_release
echo tampered >>"$RELEASE/customer-managed-workflow-agent-${VERSION}-linux-x64.tar.gz"
expect_reject "rejects a corrupted archive"

new_case
make_release
rm "$RELEASE/customer-managed-workflow-agent-${VERSION}-darwin-x64.tar.gz.sig"
expect_reject "rejects an archive without a signature"

finish
