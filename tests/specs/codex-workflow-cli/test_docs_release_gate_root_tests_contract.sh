#!/usr/bin/env bash
# Verify docs release gate root tests contract.
# Sources: Proposal 14
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
LOG="$ROOT/test-results/spec-docs-release-gate-root-tests.log"
mkdir -p "$(dirname "$LOG")"
: >"$LOG"

# note: Record each verification step in stdout and the runtime evidence log.
note() {
  printf '[docs-release-gate-root-tests] %s\n' "$*" | tee -a "$LOG"
}

# fail: Report the failed business contract and stop with a nonzero status.
fail() {
  printf '[docs-release-gate-root-tests] FAIL: %s\n' "$*" | tee -a "$LOG" >&2
  exit 1
}

SPEC="$ROOT/docs/specs/codex-workflow-cli/spec.md"
RELEASE_SPEC="$ROOT/docs/specs/release-automation/spec.md"
legacy_lower="open""code"
legacy_title="Open""Code"

[[ -f "$SPEC" ]] || fail "Missing main specification $SPEC"
[[ -f "$RELEASE_SPEC" ]] || fail "Missing release specification $RELEASE_SPEC"

note "Main specification must not promise the removed backend"
if grep -n -E "$legacy_lower|$legacy_title|old-agent|OldAgent" "$SPEC" | tee -a "$LOG"; then
  fail "Main specification still describes a removed backend"
fi
if grep -n -E 'codex.*pi.*old-agent|codex.*old-agent.*pi|old-agent.*codex.*pi' "$SPEC" | tee -a "$LOG"; then
  fail "Main specification still lists removed backend combinations"
fi

note "Main specification must identify all supported backends"
grep -Eiq 'Codex, Pi, Agy, and Claude' "$SPEC" || fail "Main specification does not identify the supported backend set"

note "Main specification must require preflight for referenced agent CLIs"
grep -Eiq 'Before a sealed run starts.*checks only executables actually referenced' "$SPEC" || fail "Main specification does not require referenced CLI preflight"

note "Release gate must run the full Go suite"
grep -q 'go test ./...' "$RELEASE_SPEC" || fail "Release specification does not reference go test ./..."
if grep -n 'go test ./internal' "$RELEASE_SPEC" | tee -a "$LOG"; then
  fail "Release gate must not rely only on internal package tests"
fi

note "contract passed: docs and release gate match codex/pi go-test layout"
