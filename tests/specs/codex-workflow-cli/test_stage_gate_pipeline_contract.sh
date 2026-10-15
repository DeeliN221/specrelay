#!/usr/bin/env bash
# Sources: Proposal 38
# Verify stage gate pipeline contract.
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
RESULT_DIR="$ROOT/test-results/specs/codex-workflow-cli/stage-gate-pipeline"
LOG="$RESULT_DIR/contract.log"

mkdir -p "$RESULT_DIR"
: >"$LOG"

# note: Record each verification step in stdout and the runtime evidence log.
note() {
  printf '[stage-gate-pipeline] %s\n' "$*" | tee -a "$LOG"
}

# fail: Report the failed business contract and stop with a nonzero status.
fail() {
  printf '[stage-gate-pipeline] FAIL: %s\n' "$*" | tee -a "$LOG" >&2
  exit 1
}

# assert_rg: Verify responsibility symbols remain in their intended source boundary.
assert_rg() {
  local pattern="$1"
  local path="$2"
  local message="$3"
  if ! rg -n "$pattern" "$path" >>"$LOG" 2>&1; then
    fail "$message"
  fi
}

note "Check stage and status semantic boundaries"
assert_rg 'type workflowStage|type WorkflowStage' "$ROOT/internal/app" "Missing workflow stage semantic type"
assert_rg 'parseWorkflowStage|ParseWorkflowStage' "$ROOT/internal/app" "Missing workflow stage parser"
assert_rg 'type runStatus|type RunStatus|func normalizeRunStatus' "$ROOT/internal/app" "Missing run status normalization"

note "Check main-stage gate pipeline files and result types"
pipeline_files="$(rg -l 'stageGatePipelineResult|completeMainStage|mainStageGatePipeline' "$ROOT/internal/app" || true)"
[[ -n "$pipeline_files" ]] || fail "Missing main-stage gate pipeline implementation"
printf '%s\n' "$pipeline_files" >>"$LOG"
assert_rg 'stageGatePipelineResult|mainStageGatePipelineResult' "$ROOT/internal/app" "Missing gate pipeline result type"
assert_rg 'completeMainStage|runMainStageGatePipeline' "$ROOT/internal/app" "Missing main-stage gate pipeline entry"

note "Verify loop and DAG nodes share one pipeline"
assert_rg 'completeMainStage|runMainStageGatePipeline' "$ROOT/internal/app/engine_run.go" "runLoop does not invoke the main-stage gate pipeline"
assert_rg 'completeMainStage|runMainStageGatePipeline' "$ROOT/internal/app/node.go" "nodeRunStage does not invoke the main-stage gate pipeline"

node_body="$(awk '
  /func \(e \*Engine\) nodeRunStage/ { flag=1 }
  /func \(e \*Engine\) nodeStageDone/ { flag=0 }
  flag { print }
' "$ROOT/internal/app/node.go")"

if printf '%s\n' "$node_body" | rg -n 'runAcceptancePreflight|runAcceptanceGate|validateStage\(|markStageCompleted\(|advance\(' >>"$LOG" 2>&1; then
  fail "nodeRunStage directly chains acceptance, validation or advance instead of the pipeline"
fi

note "Run internal/app regressions"
(cd "$ROOT" && go test ./internal/app) >>"$LOG" 2>&1 || fail "go test ./internal/app failed"

note "PASS: stage gate pipeline contract"
