#!/usr/bin/env bash
# Verify stage decision contract.
# Sources: Proposal 22
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
log="$repo_root/test-results/refactor-stability/stage-decision-contract.log"
mkdir -p "$(dirname "$log")"
: >"$log"

# note: Record each verification step in stdout and the runtime evidence log.
note() {
  printf '%s\n' "$*" | tee -a "$log"
}

# fail: Report the failed business contract and stop with a nonzero status.
fail() {
  note "FAIL: $*"
  exit 1
}

cd "$repo_root"
note "evidence id: stage-decision-contract-log"
note "evidence path: $log"
note "test path: tests/specs/codex-workflow-cli/test_stage_decision_contract.sh"

decision_files="$(fd -t f '.*stage.*decision.*\.go|.*decision.*stage.*\.go' internal/app || true)"
printf '%s\n' "$decision_files" | tee -a "$log"
if [[ -z "$decision_files" ]]; then
  fail "internal/app lacks a dedicated stage-decision source file"
fi

decision_symbols="$(rg -n 'type StageDecision|func DecideNextStage|func .*Stage.*Decision' internal/app || true)"
printf '%s\n' "$decision_symbols" | tee -a "$log"
if [[ -z "$decision_symbols" ]]; then
  fail "Missing StageDecision type or next-stage decision function"
fi

state_lines="$(wc -l < internal/app/state.go)"
note "internal/app/state.go lines: $state_lines"
if [[ "$state_lines" -gt 1900 ]]; then
  fail "state.go exceeds 1900 lines; state-machine responsibilities remain coupled"
fi

go test ./internal/app -run 'TestRepairSessionReuse|TestQARepairLoop|TestRepairLimitBlocks|TestLegacyRepairSnapshotResume|TestGoDAG|TestValidationGate|TestRootArtifactGate|TestRootAcceptancePreflight|TestWorkflowFailureReviewFailsWorkflow' -count=1 2>&1 | tee -a "$log"
