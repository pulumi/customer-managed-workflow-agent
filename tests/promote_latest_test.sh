#!/usr/bin/env bash
# Tests for scripts/promote-latest.sh: it always promotes the highest final release, so an older or out-of-order
# mirror run never moves latest backwards.
set -uo pipefail
# shellcheck source=tests/lib.sh
. "$(dirname "$0")/lib.sh"

# Stub gh: `release list` prints the tags in $CASE/releases; every call is logged to $CASE/gh.log.
stub_gh() {
  cat >"$CASE/bin/gh" <<'STUB'
#!/usr/bin/env bash
echo "$*" >>"$STUB_CASE/gh.log"
if [ "$1 $2" = "release list" ]; then cat "$STUB_CASE/releases"; fi
STUB
  chmod +x "$CASE/bin/gh"
  printf '%s\n' "$@" >"$CASE/releases"
}

run_promote() {
  GITHUB_REPOSITORY=pulumi/customer-managed-workflow-agent PATH="$CASE/bin:$PATH" \
    "$REPO_ROOT/scripts/promote-latest.sh" "$@" >"$CASE/out" 2>"$CASE/err"
  RC=$?
}

# expect <name> <version> <promoted> <latest>
expect() {
  if [ $RC -eq 0 ] && [ "$(cat "$CASE/out")" = "promoted=$3" ] && grep -q "^release edit $4 .*--latest" "$CASE/gh.log"; then
    pass "$1"
  else
    fail "$1" "rc=$RC" "out=$(cat "$CASE/out")" "$(cat "$CASE/gh.log" "$CASE/err")"
  fi
}

new_case
stub_gh v2.3.0 v2.4.0
run_promote v2.4.0
expect "promotes the newly mirrored version when it is the highest" v2.4.0 true v2.4.0

new_case
stub_gh v2.5.0 v2.4.0 v2.3.0
run_promote v2.4.0
expect "keeps the higher version latest when an older version is mirrored late" v2.4.0 false v2.5.0

new_case
stub_gh v2.9.0 v2.10.0
run_promote v2.9.0
expect "compares versions numerically, not as strings" v2.9.0 false v2.10.0

new_case
stub_gh v2.4.0 v2.5.0-rc.1
run_promote v2.4.0
expect "never promotes a prerelease" v2.4.0 true v2.4.0

new_case
stub_gh v2.4.0
run_promote v2.4.0
if grep -q -- "--exclude-pre-releases" "$CASE/gh.log" && grep -q -- "--exclude-drafts" "$CASE/gh.log"; then
  pass "lists only published final releases"
else
  fail "lists only published final releases" "$(cat "$CASE/gh.log")"
fi

# The docs job rechecks with --check before dispatching, so a retried or delayed job for a superseded version skips.
new_case
stub_gh v2.4.0 v2.5.0
run_promote --check v2.4.0
if [ $RC -eq 0 ] && [ "$(cat "$CASE/out")" = "promoted=false" ] && ! grep -q "release edit" "$CASE/gh.log"; then
  pass "--check reports a superseded version without changing latest"
else
  fail "--check reports a superseded version without changing latest" "rc=$RC" "out=$(cat "$CASE/out")" "$(cat "$CASE/gh.log")"
fi

new_case
stub_gh v2.4.0 v2.5.0
run_promote --check v2.5.0
if [ $RC -eq 0 ] && [ "$(cat "$CASE/out")" = "promoted=true" ] && ! grep -q "release edit" "$CASE/gh.log"; then
  pass "--check confirms the highest version without changing latest"
else
  fail "--check confirms the highest version without changing latest" "rc=$RC" "out=$(cat "$CASE/out")" "$(cat "$CASE/gh.log")"
fi

finish
