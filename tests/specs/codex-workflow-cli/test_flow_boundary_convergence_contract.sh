#!/usr/bin/env bash
# Verify flow boundary convergence contract.
# Sources: Proposal 41
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
log="$repo_root/test-results/refactor-flow-boundary/flow-boundary-convergence-contract.log"
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

require_file() {
  # require_file: Verify the production source file for this boundary exists.
  local path="$1"
  [[ -f "$repo_root/$path" ]] || fail "Missing target boundary file:$path"
  note "Found target boundary file:$path"
}

reject_pattern() {
  # reject_pattern: Reject duplicate responsibilities in the designated file.
  local pattern="$1"
  shift
  if rg -n "$pattern" "$@" 2>/dev/null | tee -a "$log" | grep -q .; then
    fail "Found forbidden duplicated boundary pattern:$pattern"
  fi
}

cd "$repo_root"

note "evidence id: flow-boundary-convergence-log"
note "evidence path: $log"
note "test id: flow-boundary-convergence-contract"

require_file "internal/app/flow_command_registry.go"
require_file "internal/app/workflow_topology.go"

if ! rg -n 'type flowCommandSpec struct|var flowCommandRegistry|func flowCommandByName' internal/app/flow_command_registry.go | tee -a "$log" | grep -q .; then
  fail "flow_command_registry.go must define the registry type or lookup entry"
fi

reject_pattern 'func commandNeedsWorkflowTools|case "config":|case "run":|case "resume":|case "batch":|case "restart":|case "status":|case "abort":|case "clean":|case "watch":|case "--resume":|case "--run":' \
  internal/app/app.go

reject_pattern 'case "config":' \
  internal/app/command_dispatch.go

reject_pattern 'planning_context|implementation_context|before_execution|before_review|before_qa' \
  internal/app/graph.go \
  internal/app/status_view_model.go \
  internal/app/status_parallel.go \
  internal/app/config_parallel.go

reject_pattern 'stageChecklistLines|stageChecklistLinesWithParallel|stageChecklistLinesForRepo|watchStageChecklistLines' \
  internal/app/app.go \
  internal/app/engine_progress.go \
  internal/app/status_render.go

note "Run real Go command and status regressions"
go test ./internal/app ./internal/ozcli ./tests -count=1 2>&1 | tee -a "$log"

note "PASS: flow-boundary-convergence-contract"
