#!/usr/bin/env bash
# Verify manual intervention boundary docs contract.
# Sources: Proposal 16, Proposal 42
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
LOG="$ROOT/test-results/manual-intervention-boundary-docs.log"
SPEC="$ROOT/docs/specs/codex-workflow-cli/spec.md"

note() {
  # Record contract steps and output so failures remain reviewable.
  printf '[manual-intervention-docs] %s\n' "$*" | tee -a "$LOG"
}

fail() {
  # Report the missing business contract and stop with a clear diagnostic.
  printf '[manual-intervention-docs] FAIL: %s\n' "$*" | tee -a "$LOG" >&2
  exit 1
}

mkdir -p "$(dirname "$LOG")"
: >"$LOG"

note "Specification must allow unrelated proposals during a run"
grep -Eiq 'Inserting an unrelated new proposal while another runs' "$SPEC" || fail "Specification does not allow unrelated docs/changes proposals during a run"

note "Specification must protect the current change, source and configuration"
grep -Eiq 'Unexpected changes outside the allowed current transition trigger intervention' "$SPEC" || fail "Specification omits current-run path protection"

note "Specification must state fixed external subagents were removed"
grep -Eiq 'must not create a fixed external subagent' "$SPEC" || fail "Specification omits proposal 42 fixed-subagent removal contract"

note "contract passed: manual intervention boundary is documented"
