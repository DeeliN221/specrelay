#!/usr/bin/env bash
# Sources: Proposal 29
# Verify workflow config boundary contract.
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
result_dir="$repo_root/test-results/codex-workflow-cli/workflow-config-boundary"
log="$result_dir/workflow-config-boundary.log"
mkdir -p "$result_dir"
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

note "evidence id: workflow-config-boundary-log"
note "evidence path: $log"
note "test id: workflow-config-boundary-contract"

for file in \
  internal/app/config_schema.go \
  internal/app/config_parallel.go \
  internal/app/config_validation.go
do
  [[ -f "$file" ]] || fail "Missing config parsing boundary file:$file"
  note "Found boundary file:$file"
done

for symbol in \
  'type workflowConfigInput' \
  'type stageOptionsInput' \
  'func parallelGroupConfigFromInput' \
  'func parallelConfigFromStages' \
  'func validateParallelConfig' \
  'func validationConfigFromInput' \
  'func cloneValidationCommands'
do
  if rg -n "$symbol" internal/app/config.go | tee -a "$log" | grep -q .; then
    fail "config.go still defines an extracted configuration responsibility:$symbol"
  fi
done

note "Run default configuration and legacy-rejection contracts"
for test_script in \
  tests/specs/codex-workflow-cli/test_go_dag_graph_status_contract.sh \
  tests/specs/codex-workflow-cli/test_legacy_config_rejection_contract.sh
do
  note "Run $test_script"
  bash "$test_script" 2>&1 | tee -a "$log"
done

note "PASS: workflow-config-boundary-contract"
