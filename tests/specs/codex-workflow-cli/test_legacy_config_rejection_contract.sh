#!/usr/bin/env bash
# Verify legacy config rejection contract.
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
result_dir="$repo_root/test-results/codex-workflow-cli/tree-config"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

mkdir -p "$result_dir"
log="$result_dir/legacy-config-rejection.log"
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

oz_bin="$tmp/wo"
empty_home="$tmp/empty-home"
mkdir -p "$empty_home"
note "Build the real specrelay flow binary"
go build -C "$repo_root" -o "$oz_bin" ./cmd/specrelay 2>&1 | tee -a "$log"

init_project() {
  local project="$1"
  mkdir -p "$project"
  git -C "$project" init -q
  git -C "$project" config user.email "test@example.com"
  git -C "$project" config user.name "Test User"
  printf 'demo\n' >"$project/README.md"
  git -C "$project" add README.md
  git -C "$project" commit -qm init
}

run_rejection_case() {
  local name="$1"
  local field="$2"
  local project="$tmp/$name"
  init_project "$project"
  cat >"$project/oz-flow.yaml"

  note "Verify rejection of legacy field:$field ($name)"
  set +e
  (
    cd "$project"
    HOME="$empty_home" "$oz_bin" flow graph --change demo --format json
  ) >"$tmp/$name.out" 2>"$tmp/$name.err"
  local code=$?
  set -e

  cat "$tmp/$name.out" "$tmp/$name.err" >>"$log"
  if [[ "$code" -eq 0 ]]; then
    fail "Legacy field $field remains accepted:$name"
  fi
  if ! grep -Eiq "$field" "$tmp/$name.out" "$tmp/$name.err"; then
    fail "Legacy field $field diagnostic omits its name:$name"
  fi
}

run_global_rejection_case() {
  local name="$1"
  local field="$2"
  local project="$tmp/$name-project"
  local home="$tmp/$name-home"
  init_project "$project"
  mkdir -p "$home"
  cat >"$home/oz-flow.yaml"

  note "Verify rejection of legacy global configuration:$field ($name)"
  set +e
  (
    cd "$project"
    HOME="$home" "$oz_bin" flow graph --change demo --format json
  ) >"$tmp/$name.out" 2>"$tmp/$name.err"
  local code=$?
  set -e

  cat "$tmp/$name.out" "$tmp/$name.err" >>"$log"
  if [[ "$code" -eq 0 ]]; then
    fail "Legacy global field $field remains accepted:$name"
  fi
  if ! grep -Eiq "$field" "$tmp/$name.out" "$tmp/$name.err"; then
    fail "Legacy global field $field diagnostic omits its name:$name"
  fi
}

run_rejection_case "old-specrelay-flow-root" "wo" <<'YAML'
wo:
  workflow:
    max_review_iterations: 0
    stages:
      execution:
        cli: codex
YAML

run_rejection_case "old-workflow-root" "workflow" <<'YAML'
workflow:
  max_review_iterations: 0
  stages:
    execution:
      agent: codex
YAML

run_rejection_case "old-engine" "engine" <<'YAML'
engine: 内嵌工作流
max_review_iterations: 0
stages:
  execution:
    agent: codex
YAML

run_rejection_case "old-defaults" "defaults" <<'YAML'
parallel: true
defaults:
  agent: codex
stages:
  execution:
    agent: codex
YAML

run_rejection_case "old-iterations" "iterations" <<'YAML'
parallel: true
max_review_iterations: 1
stages:
  execution:
    agent: codex
iterations:
  fix_1:
    model: old-model
YAML

run_rejection_case "old-cli" "cli" <<'YAML'
stages:
  execution:
    cli: codex
YAML

run_rejection_case "old-tool" "tool" <<'YAML'
stages:
  execution:
    tool: codex
YAML

run_rejection_case "old-permissions" "permissions" <<'YAML'
stages:
  execution:
    agent: codex
    permissions: danger-full-access
YAML

run_rejection_case "old-validation-max" "max_attempts_per_stage" <<'YAML'
stages:
  execution:
    agent: codex
validation:
  max_attempts_per_stage: 3
YAML

run_global_rejection_case "old-global-specrelay-flow-root" "wo" <<'YAML'
wo:
  workflow:
    stages:
      execution:
        cli: codex
YAML

note "PASS: legacy-config-rejection-contract"
