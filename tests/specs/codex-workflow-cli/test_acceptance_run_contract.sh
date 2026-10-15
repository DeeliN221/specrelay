#!/usr/bin/env bash
# Verify acceptance execution using retained Unicode proposal fixtures.
# Cover the command surface, execution summaries, and sealed-run gates.
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
CHANGE_DIR="$ROOT/tests/specs/codex-workflow-cli/acceptance-run"
RESULT_DIR="$ROOT/test-results/specs/codex-workflow-cli/acceptance-run"
LOG="$RESULT_DIR/contract.log"

mkdir -p "$RESULT_DIR"
: >"$LOG"

for script in \
  test_acceptance_run_contract_surface.sh \
  test_acceptance_run_success_contract.sh \
  test_acceptance_run_failure_contract.sh \
  test_acceptance_run_stage_gate_contract.sh
do
  bash "$CHANGE_DIR/$script" >>"$LOG" 2>&1
done

printf 'contract passed: acceptance run command, result summary, evidence check, and gate boundary verified\n' | tee -a "$LOG"
