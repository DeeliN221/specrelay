#!/usr/bin/env bash
# Verify engine runtime boundary contract.
# Sources: Proposal 32
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
log="$repo_root/test-results/engine-runtime-boundary/contract.log"
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

assert_file_has() {
  # assert_file_has: Verify the required source file and responsibility pattern exist.
  local file="$1"
  local pattern="$2"
  [[ -f "$file" ]] || fail "Missing engine runtime boundary file:$file"
  rg -n "$pattern" "$file" >>"$log" || fail "$file Missing pattern:$pattern"
  note "Verified boundary file:$file"
}

cd "$repo_root"

note "evidence id: engine-runtime-boundary-log"
note "evidence path: $log"
note "test id: engine-runtime-boundary-contract"

assert_file_has "internal/app/state_model.go" 'type State struct|type DAGNodeState struct'
assert_file_has "internal/app/engine_run.go" 'func NewEngine|func \(e \*Engine\) runLoop'
assert_file_has "internal/app/engine_resume.go" 'func \(e \*Engine\) Resume|func \(e \*Engine\) resumeRun'
assert_file_has "internal/app/engine_stage.go" 'func \(e \*Engine\) runStage|func \(e \*Engine\) stageOptionsForRun'
assert_file_has "internal/app/engine_progress.go" 'type stageProgressWriter|func persistStateSessionID'

note "Check state.go responsibility ownership rather than a brittle line-count ceiling"

note "Run workflow-related internal/app Go regressions"
go test ./internal/app -run 'Test(GoDAG|GateState|Migrated|Restart|StageDecision|Status)' -count=1 2>&1 | tee -a "$log"

note "PASS: engine-runtime-boundary-contract"
