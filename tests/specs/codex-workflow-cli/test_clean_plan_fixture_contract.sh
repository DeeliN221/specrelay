#!/usr/bin/env bash
# Verify clean plan fixture contract.
# Sources: Proposal 40
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
RESULT_DIR="$ROOT/test-results/clean-plan-fixture"
LOG="$RESULT_DIR/contract.log"
TMP_DIR="$RESULT_DIR/tmp"
SPECRELAY_BIN="$RESULT_DIR/specrelay"

rm -rf "$TMP_DIR"
mkdir -p "$RESULT_DIR" "$TMP_DIR"
: >"$LOG"

note() {
  # note: Record each verification step in stdout and the runtime evidence log.
  printf '[clean-plan-fixture] %s\n' "$*" | tee -a "$LOG"
}

fail() {
  # fail: Report the failed business contract and stop with a nonzero status.
  printf '[clean-plan-fixture] FAIL: %s\n' "$*" | tee -a "$LOG" >&2
  exit 1
}

assert_rg() {
  # assert_rg: Verify responsibility symbols remain in their intended source boundary.
  local pattern="$1"
  local path="$2"
  local message="$3"
  if ! rg -n "$pattern" "$path" >>"$LOG" 2>&1; then
    fail "$message"
  fi
}

note "Check clean plan/apply and workflow fixture boundaries"
assert_rg 'type CleanPlan|func BuildCleanPlan|func ApplyCleanPlan' "$ROOT/internal/app" "Missing clean plan/apply boundary"
assert_rg 'workflowFixture|newWorkflowFixture|fakeWorkflowRunner' "$ROOT/internal/app" "Missing shared workflow test fixtures"

note "Build the real specrelay executable"
(cd "$ROOT" && go build -o "$SPECRELAY_BIN" ./cmd/specrelay) >>"$LOG" 2>&1 || fail "go build ./cmd/specrelay failed"

PROJECT="$TMP_DIR/project"
STATE_HOME="$TMP_DIR/state"
mkdir -p "$PROJECT" "$STATE_HOME"
cd "$PROJECT"
git init >>"$LOG" 2>&1
git config user.email "specrelay-test@example.com"
git config user.name "specrelay test"
printf 'demo\n' >README.md
git add README.md >>"$LOG" 2>&1
git commit -m init >>"$LOG" 2>&1

repo_key="$(python3 - "$PROJECT" <<'PY'
import hashlib
import os
import re
import sys

path = os.path.abspath(sys.argv[1])
name = os.path.basename(os.path.normpath(path)).lower()
name = re.sub(r'[^a-z0-9]+', '-', name).strip('-') or 'repo'
print(f"{name}-{hashlib.sha1(path.encode()).hexdigest()[:10]}")
PY
)"

RUN_DIR="$STATE_HOME/specrelay/flow/repos/$repo_key/runs/run-failed"
mkdir -p "$RUN_DIR"
cat >"$RUN_DIR/state.json" <<'EOF'
{
  "run_id": "run-failed",
  "change_name": "1-clean-plan",
  "sealed": true,
  "status": "failed",
  "stage": "execution",
  "error": "contract fixture failure",
  "sessions": {},
  "stages": {},
  "paths": {},
  "workflow_config": {}
}
EOF

note "Run JSON dry-run and verify planning without deletion"
XDG_STATE_HOME="$STATE_HOME" "$SPECRELAY_BIN" flow clean --dry-run --json >"$RESULT_DIR/clean-plan.json" 2>>"$LOG" || fail "specrelay flow clean --dry-run --json must succeed"
[[ -d "$RUN_DIR" ]] || fail "Dry-run must not delete the failed run directory"
if ! rg -n 'run-failed|delete|would_delete|clean_plan|dry_run' "$RESULT_DIR/clean-plan.json" >>"$LOG" 2>&1; then
  fail "Dry-run JSON lacks the failed-run deletion plan"
fi

note "Apply clean and verify deletion of the same failed run"
XDG_STATE_HOME="$STATE_HOME" "$SPECRELAY_BIN" flow clean >>"$LOG" 2>&1 || fail "specrelay flow clean must apply the plan successfully"
[[ ! -e "$RUN_DIR" ]] || fail "Failed run directory remains after clean"

note "Run internal/app regressions"
(cd "$ROOT" && go test ./internal/app) >>"$LOG" 2>&1 || fail "go test ./internal/app failed"

note "PASS: clean plan and workflow fixture contract"
