#!/usr/bin/env bash
# Verify running demand insertion contract.
# Sources: Proposal 16
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
LOG="$ROOT/test-results/running-demand-insertion.log"

note() {
  # Record contract steps and output so failures remain reviewable.
  printf '[running-demand] %s\n' "$*" | tee -a "$LOG"
}

mkdir -p "$(dirname "$LOG")"
: >"$LOG"

note "Verify unrelated changes can be added while current run/source/rename remain protected"
(
  cd "$ROOT"
  go test ./internal/app -run 'TestDetectManualInterventionAllowsUnrelatedActiveChange|TestDetectManualInterventionIgnoresExistingProtectedBaselineDiff|TestDetectManualInterventionBlocksCurrentRunPaths|TestDetectManualInterventionBlocksCurrentChangeRename' -count=1
) 2>&1 | tee -a "$LOG"

note "contract passed: running demand insertion keeps current-run guards"
