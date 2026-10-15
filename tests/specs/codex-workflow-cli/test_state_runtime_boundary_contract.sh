#!/usr/bin/env bash
# Verify state runtime boundary contract.
# Sources: Proposal 26
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
log="$repo_root/test-results/refactor-state-runtime-boundary/state-runtime-boundary-contract.log"
mkdir -p "$(dirname "$log")"
: >"$log"

note() {
  # note: Record each verification step in stdout and the runtime evidence log.
  printf '%s\n' "$*" | tee -a "$log"
}

fail() {
  # fail: Report the failed business contract and stop with a nonzero status.
  note "FAIL: $*"
  exit 1
}

cd "$repo_root"

note "evidence id: state-runtime-boundary-log"
note "evidence path: $log"
note "test id: state-runtime-boundary-contract"

for file in \
  internal/app/state_store.go \
  internal/app/run_lock.go \
  internal/app/prompt_context.go \
  internal/app/git_guard.go
do
  [[ -f "$file" ]] || fail "Missing run-state boundary file:$file"
  note "Found boundary file:$file"
done

for symbol in \
  'func saveState' \
  'func loadState' \
  'func mergeState' \
  'func promptForStage' \
  'func promptContext' \
  'func gitSnapshot' \
  'func classifyGitSnapshotChange' \
  'func acquireLock' \
  'func lockFileStatus'
do
  if rg -n "$symbol" internal/app/state.go | tee -a "$log" | grep -q .; then
    fail "state.go still directly defines an extracted responsibility:$symbol"
  fi
done

note "Run state-machine, Go DAG, intervention and acceptance preflight regressions"
go test ./internal/app \
  -run 'TestEngineStartRunsCleanReviewsToDone|TestQAFailureReturnsToFix|TestDetectManualIntervention|TestGoDAGRetryableHelperErrorRestoresRunningState|TestRootAcceptancePreflight' \
  -count=1 2>&1 | tee -a "$log"

note "PASS: state-runtime-boundary-contract"
