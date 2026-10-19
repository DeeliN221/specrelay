#!/usr/bin/env bash
# Verify compact English graph and iteration limit.
# Sources: Proposal 45, Proposal 46
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
RESULT_DIR="$ROOT/test-results/workflow/compact-chinese-graph"
TMP="$(mktemp -d)"

cleanup() {
  rm -rf "$TMP"
}

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

note() {
  printf '%s\n' "$*" | tee -a "$RESULT_DIR/test.log"
}

# prompt_block: Extract one role prompt to avoid matching terms in another role.
prompt_block() {
  local prompt_name="$1"
  awk -v prompt_name="$prompt_name" '
    $0 == "  " prompt_name ": |" { inside = 1; next }
    inside && /^  [a-z_]+: \|$/ { exit }
    inside { print }
  ' "$PROJECT/specrelay-flow.yaml"
}

trap cleanup EXIT
rm -rf "$RESULT_DIR"
mkdir -p "$RESULT_DIR"

SPECRELAY_BIN="$TMP/wo"
note "build real specrelay flow binary"
(cd "$ROOT" && go build -o "$SPECRELAY_BIN" ./cmd/specrelay) >>"$RESULT_DIR/test.log" 2>&1

PROJECT="$TMP/project"
mkdir -p "$PROJECT"
(
  cd "$PROJECT"
  git init -q
  git config user.email "test@example.com"
  git config user.name "Test User"
  printf 'demo\n' > README.md
  git add README.md
  git commit -qm init
)

note "generate default specrelay-flow.yaml with an audit iteration budget"
(
  cd "$PROJECT"
  "$SPECRELAY_BIN" flow config
) >"$RESULT_DIR/config.out" 2>"$RESULT_DIR/config.err"
cp "$PROJECT/specrelay-flow.yaml" "$RESULT_DIR/specrelay-flow.yaml"
if grep -q 'max_repair_iterations:' "$PROJECT/specrelay-flow.yaml"; then
  fail "new default config should not emit deprecated max_repair_iterations"
fi
if grep -q 'max_review_iterations:' "$PROJECT/specrelay-flow.yaml"; then
  fail "new config should not emit legacy max_review_iterations"
fi
grep -q '^max_audit_iterations: 3$' "$PROJECT/specrelay-flow.yaml" ||
  fail "new default config should cap pre-QA audits at 3"
if grep -q '^engine:' "$PROJECT/specrelay-flow.yaml"; then
  fail "default specrelay-flow.yaml should not expose an engine field"
fi
REPAIR_PROMPT="$(prompt_block repair)"
QA_PROMPT="$(prompt_block qa)"
rg -q '`pre_qa_audit`.*`audit_N`' <<<"$REPAIR_PROMPT"
rg -q '`qa_targeted_repair`.*`targeted_repair_N`' <<<"$REPAIR_PROMPT"
rg -q 'repairer cannot archive; independent QA approval is still required after clean' <<<"$REPAIR_PROMPT"
rg -q 'Use an independent QA session' <<<"$QA_PROMPT"

note "render mermaid graph and verify it is compact"
(
  cd "$PROJECT"
  "$SPECRELAY_BIN" flow graph --change demo --format mermaid
) >"$RESULT_DIR/graph.mmd" 2>"$RESULT_DIR/graph.err"
grep -q 'flowchart TD' "$RESULT_DIR/graph.mmd" || fail "mermaid graph should render a flowchart"

if grep -Eq 'repair_[1-9]|qa_[1-9]|review_[1-9]|fix_[1-9]' "$RESULT_DIR/graph.mmd"; then
  fail "mermaid graph should use a dynamic template without expanded legacy finite stages"
fi

if grep -Eq 'subagent:|fan-in|planning_context|implementation_context|before_review|before_qa|before_execution' "$RESULT_DIR/graph.mmd"; then
  fail "mermaid visible labels should not mix internal English subagent/group names"
fi

grep -q 'audit\[Self-review\]' "$RESULT_DIR/graph.mmd" || fail "graph should show the English audit label"
grep -q 'targeted_repair\[Repair\]' "$RESULT_DIR/graph.mmd" || fail "graph should show the English repair label"
grep -q 'qa\[Testing\]' "$RESULT_DIR/graph.mmd" || fail "graph should show the English QA label"
grep -q 'Environment blocked' "$RESULT_DIR/graph.mmd" || fail "graph should show recoverable environment blocking"
grep -q 'Stalled' "$RESULT_DIR/graph.mmd" || fail "graph should show recoverable stalled blocking"
grep -q 'round 3 reached; finish self-review.*qa' "$RESULT_DIR/graph.mmd" || fail "graph should show the configured audit cap enters QA"

note "contract passed: default quality loop caps audits and graph is compact English"
