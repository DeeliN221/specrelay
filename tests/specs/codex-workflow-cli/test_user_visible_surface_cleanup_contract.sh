#!/usr/bin/env bash
# Verify user visible surface cleanup contract.
# Sources: Proposal 36
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
RESULT_DIR="$ROOT/test-results/codex-workflow-cli/user-visible-surface-cleanup"
LOG="$RESULT_DIR/contract.log"
TMP="$(mktemp -d)"

cleanup() {
  # cleanup: Remove the temporary repository and isolated user state.
  rm -rf "$TMP"
}

note() {
  # note: Record each verification step in stdout and the runtime evidence log.
  printf '[user-visible-surface-cleanup] %s\n' "$*" | tee -a "$LOG"
}

fail() {
  # fail: Report the failed business contract and stop with a nonzero status.
  note "FAIL: $*"
  exit 1
}

scan_forbidden() {
  # scan_forbidden: Find obsolete contract text in the active maintained paths.
  local label="$1"
  local pattern="$2"
  shift 2
  local output
  output="$(rg -n -i --glob '!docs/changes/archive/**' --glob '!tests/specs/codex-workflow-cli/test_user_visible_surface_cleanup_contract.sh' "$pattern" "$@" 2>/dev/null || true)"
  if [[ -n "$output" ]]; then
    note "Found forbidden remnant:$label"
    printf '%s\n' "$output" | tee -a "$LOG"
    fail "$label"
  fi
}

assert_no_internal_engine_text() {
  # assert_no_internal_engine_text: Verify public CLI output hides internal engine names.
  local path="$1"
  if rg -n -i '\bgo-dag\b|\bdagu\b|engine: go-dag|引擎 go-dag' "$path" >>"$LOG" 2>&1; then
    fail "$path exposes an internal engine or obsolete Dagu name"
  fi
}

trap cleanup EXIT
mkdir -p "$RESULT_DIR"
: >"$LOG"

cd "$ROOT"

active_paths=(
  README.md
  docs/specs
  prompts-template
  tests/specs
  .github/workflows
)

note "Scan user docs, specs, templates, release gates and specification tests"
scan_forbidden "内部引擎名称不得进入用户可见面" '\bgo-dag\b|\bdagu\b|engine: go-dag|引擎 go-dag' "${active_paths[@]}"
scan_forbidden "旧 wo 配置、命令和状态目录不得进入活跃维护面" 'cmd/wo|\.\/cmd/wo|wo\.yaml|(^|/)\.wo(/|$)|/wo/repos|\bwo (status|watch|run|clean|config|restart|resume|batch|abort|update|graph|contract|list-changes)\b|`wo`|wo 命令|wo 二进制|wo CLI|wo 工作流|wo 执行器' "${active_paths[@]}"
scan_forbidden "旧 WO_* 产品变量不得进入活跃维护面" '\bWO_[A-Z0-9_]*|\bwo_bin\b|\bWO_BIN\b' internal tests/specs docs/specs README.md
scan_forbidden "legacy 后端不得进入默认配置、文档或模板" 'legacy-agent|LegacyAgent|opencode|OpenCode|open""code' README.md docs/specs prompts-template .github/workflows

note "Build real specrelay and check help, configuration, graph, status and diagnostics"
SPECRELAY_BIN="$TMP/specrelay"
go build -o "$SPECRELAY_BIN" ./cmd/specrelay >>"$LOG" 2>&1

PROJECT="$TMP/project"
mkdir -p "$PROJECT/docs/changes/demo"
(
  cd "$PROJECT"
  git init -q
  git config user.email "test@example.com"
  git config user.name "Test User"
  printf 'demo\n' > README.md
  git add README.md
  git commit -qm init
)

"$SPECRELAY_BIN" flow --help >"$RESULT_DIR/flow-help.txt" 2>&1
assert_no_internal_engine_text "$RESULT_DIR/flow-help.txt"
(
  cd "$PROJECT"
  "$SPECRELAY_BIN" flow config >"$RESULT_DIR/flow-config.out" 2>&1
)
assert_no_internal_engine_text "$RESULT_DIR/flow-config.out"
assert_no_internal_engine_text "$PROJECT/specrelay-flow.yaml"
if rg -n '^engine:' "$PROJECT/specrelay-flow.yaml" >>"$LOG" 2>&1; then
  fail "Default specrelay-flow.yaml must not expose an engine field"
fi

(
  cd "$PROJECT"
  "$SPECRELAY_BIN" flow graph --change demo --format json >"$RESULT_DIR/graph.json" 2>"$RESULT_DIR/graph-json.err"
  "$SPECRELAY_BIN" flow graph --change demo --format mermaid >"$RESULT_DIR/graph.mmd" 2>"$RESULT_DIR/graph-mermaid.err"
  "$SPECRELAY_BIN" flow status >"$RESULT_DIR/status.txt" 2>&1 || true
)
assert_no_internal_engine_text "$RESULT_DIR/graph.json"
assert_no_internal_engine_text "$RESULT_DIR/graph-json.err"
assert_no_internal_engine_text "$RESULT_DIR/graph.mmd"
assert_no_internal_engine_text "$RESULT_DIR/graph-mermaid.err"
assert_no_internal_engine_text "$RESULT_DIR/status.txt"

set +e
(
  cd "$PROJECT"
  "$SPECRELAY_BIN" flow run --change demo --engine unknown --json
) >"$RESULT_DIR/run-engine-unknown.out" 2>"$RESULT_DIR/run-engine-unknown.err"
run_code=$?
set -e
[[ "$run_code" -ne 0 ]] || fail "Unknown engine parameter must not start a run"
assert_no_internal_engine_text "$RESULT_DIR/run-engine-unknown.out"
assert_no_internal_engine_text "$RESULT_DIR/run-engine-unknown.err"

note "PASS"
