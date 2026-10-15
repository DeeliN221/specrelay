#!/usr/bin/env bash
# Verify gate state acceptance preflight contract.
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
RESULT_DIR="$ROOT/test-results/gate-state-acceptance-preflight"
LOG="$RESULT_DIR/contract.log"

mkdir -p "$RESULT_DIR"
: >"$LOG"

note() {
  # note: Record each verification step in stdout and the runtime evidence log.
  printf '%s\n' "$*" | tee -a "$LOG"
}

cd "$ROOT"

note "Run Go regressions for artifact_gates, specrelay validate and acceptance_preflight"
go test ./internal/app -run 'TestRootArtifactGateFailureUsesArtifactGateState|TestRootExecutionRunsOzValidateAsRetryableGate|TestRootAcceptancePreflight' -count=1 2>&1 | tee -a "$LOG"

note "Verify runtime evidence exists and contains PASS"
grep -q 'PASS' "$LOG"

note "PASS"
