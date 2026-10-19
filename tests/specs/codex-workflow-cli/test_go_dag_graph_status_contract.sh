#!/usr/bin/env bash
# Verify go dag graph status contract.
# Sources: Proposal 3, Proposal 42
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

log_dir="$repo_root/test-results/workflow"
mkdir -p "$log_dir"
log="$log_dir/内嵌工作流-graph-status-contract.log"
: >"$log"

note() { printf '%s
' "$*" | tee -a "$log"; }
fail() { note "FAIL: $*"; exit 1; }

cd "$repo_root"
wo="$tmp/wo"
go build -o "$wo" ./cmd/specrelay 2>&1 | tee -a "$log"

project="$tmp/project"
mkdir -p "$project"
git -C "$project" init -q
git -C "$project" config user.email "test@example.com"
git -C "$project" config user.name "Test User"
printf 'initial
' >"$project/README.md"
git -C "$project" add README.md
git -C "$project" commit -q -m initial

note "specrelay flow config must generate defaults without fixed external subagents"
(cd "$project" && "$wo" flow config) >"$tmp/config.out"
cat "$tmp/config.out" >>"$log"
cat "$project/specrelay-flow.yaml" >>"$log"
grep -qF "repair:" "$project/specrelay-flow.yaml" || fail "specrelay-flow.yaml must configure the main repair stage"
if rg -n 'parallel:|subagents:|subagent_guard:|before:' "$project/specrelay-flow.yaml" >>"$log"; then
  fail "specrelay-flow.yaml must not reintroduce fixed external subagents"
fi

note "Mermaid graph must display the current dynamic quality loop"
(cd "$project" && "$wo" flow graph --change demo --format mermaid) >"$tmp/graph.mmd"
cat "$tmp/graph.mmd" >>"$log"
for want in "Execution" "Self-review" "Testing" "Repair" "Environment blocked" "Stalled" "Archive"; do
  grep -qF "$want" "$tmp/graph.mmd" || fail "Mermaid graph is missing $want"
done
for old_label in "全量自查" "独立测试" "定向修复"; do
  if grep -qF "$old_label" "$tmp/graph.mmd"; then
    fail "Mermaid graph still displays obsolete stage name $old_label"
  fi
done
for forbidden in "planning_context" "implementation_context" "fan-in" "subagent" "review_2" "fix_2" "最多5轮"; do
  if grep -qF "$forbidden" "$tmp/graph.mmd"; then
    fail "Mermaid graph must not expose obsolete internal label $forbidden"
  fi
done

note "Verify compact status stages through maintained Go tests"
go test ./internal/app \
  -run 'TestStatusViewAggregatesDynamicQualityLoopStages|TestRunnerStatusViewSerializesObservability' \
  -count=1 -v 2>&1 | tee -a "$log"

note "PASS"
