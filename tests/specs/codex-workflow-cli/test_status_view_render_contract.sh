#!/usr/bin/env bash
# Verify status view render contract.
# Sources: Proposal 23, Proposal 27, Proposal 31
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
LOG="$repo_root/test-results/refactor-stability/status-view-render-contract.log"
mkdir -p "$(dirname "$LOG")"
: >"$LOG"

note() {
  # note: Record each verification step in stdout and the runtime evidence log.
  printf '%s\n' "$*" | tee -a "$LOG"
}

cd "$repo_root"

note "evidence id: status-view-render-contract-log"
note "evidence path: $LOG"
note "test path: tests/specs/codex-workflow-cli/test_status_view_render_contract.sh"

if ! fd 'status.*render.*\.go|render.*status.*\.go' internal/app | tee -a "$LOG" | grep -q .; then
  note "internal/app lacks status-render source"
  exit 1
fi

[[ -f internal/app/status_view.go ]] || {
  note "internal/app lacks shared status-view source"
  exit 1
}

assert_file_has() {
  # assert_file_has: Verify the required source file and responsibility pattern exist.
  local file="$1"
  local pattern="$2"
  [[ -f "$file" ]] || {
    note "Missing target file:$file"
    exit 1
  }
  rg -n "$pattern" "$file" >>"$LOG" || {
    note "$file Missing pattern:$pattern"
    exit 1
  }
}

note "Check internal status-view responsibility boundaries"
assert_file_has "internal/app/status_view_model.go" 'func buildStatusView|func buildHumanStatusView'
assert_file_has "internal/app/status_duration.go" 'func stageDurationItems|func statusWorkflowWallDuration'
assert_file_has "internal/app/status_render_compact.go" 'func compactStatusLines|func statusDisplayWidth'
assert_file_has "internal/app/status_stale.go" 'func humanDisplayState|func isStaleRunningRun'

line_count="$(wc -l < internal/app/status_view.go | tr -d ' ')"
note "status_view.go line_count=$line_count"
if (( line_count > 350 )); then
  note "status_view.go remains too large; rendering responsibilities were not extracted"
  exit 1
fi

if rg -n "func watchBatchStatusLines|func watchRunStatusLines|func runProposalStatusLines|func watchStageChecklistLines" internal/app/app.go | tee -a "$LOG" | grep -q .; then
  note "app.go still defines watch/status text-rendering helpers"
  exit 1
fi

for symbol in \
  'func stageChecklistLines' \
  'func stageChecklistLinesWithParallel' \
  'func stageChecklistLinesForRepo' \
  'func stageDurationSummaryLines' \
  'func stageDurationItems' \
  'func visibleSessionItems' \
  'func plannerSessionID' \
  'func sessionRoleID' \
  'func roleOccurred'
do
  if rg -n "$symbol" internal/app/app.go | tee -a "$LOG" | grep -q .; then
    note "app.go still defines an old status calculation helper:$symbol"
    exit 1
  fi
done

go test ./internal/app -run 'TestStatusView|TestPrintHumanStatus|TestWatch|TestRunner|TestCompactStatus|TestHumanStatus' -count=1 2>&1 | tee -a "$LOG"
