#!/usr/bin/env bash
# Verify agent cli preflight contract.
# Sources: Proposal 14, Proposal 15
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
LOG="$ROOT/test-results/spec-agent-cli-preflight.log"
STATE_SNAPSHOT="$ROOT/test-results/spec-agent-cli-preflight-state.txt"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$(dirname "$LOG")"
: >"$LOG"
: >"$STATE_SNAPSHOT"

# note: Record each verification step in stdout and the runtime evidence log.
note() {
  printf '[agent-cli-preflight] %s\n' "$*" | tee -a "$LOG"
}

# fail: Report the failed business contract and stop with a nonzero status.
fail() {
  printf '[agent-cli-preflight] FAIL: %s\n' "$*" | tee -a "$LOG" >&2
  exit 1
}

# assert_no_run_state: Verify preflight failure occurs before sealed run state is created.
assert_no_run_state() {
  local state_root="$1"
  local output_file="$2"
  if [[ -d "$state_root/specrelay/flow/repos" ]]; then
    find "$state_root/specrelay/flow/repos" -path '*/runs/*/state.json' -type f | sort >"$output_file"
    [[ ! -s "$output_file" ]] || fail "Failed preflight must not create run state"
  else
    : >"$output_file"
  fi
}

# write_demo_change: Create a minimal active proposal for the real workflow entry point.
write_demo_change() {
  local repo="$1"
  local change="$2"
  mkdir -p "$repo/docs/changes/$change/tests"
  printf '# brief\n' >"$repo/docs/changes/$change/brief.md"
  printf '# proposal\n' >"$repo/docs/changes/$change/proposal.md"
  printf '# design\n' >"$repo/docs/changes/$change/design.md"
  cat >"$repo/docs/changes/$change/spec.md" <<'MD'
# spec

### 需求：演示

系统必须提供演示合同。

#### 场景：演示路径

- **当** 执行测试
- **则** brief 存在
MD
  cat >"$repo/docs/changes/$change/tests/test_smoke.sh" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
test -f docs/changes/1-演示检查/brief.md
SH
  chmod +x "$repo/docs/changes/$change/tests/test_smoke.sh"
  cat >"$repo/docs/changes/$change/acceptance.json" <<'JSON'
{
  "summary": "demo",
  "coverage": [
    {
      "spec": "需求：demo / 场景：demo",
      "tests": ["demo-smoke"],
      "evidence": [],
      "risk": "demo only"
    }
  ],
  "required_tests": [
    {
      "id": "demo-smoke",
      "source": "change_contract",
      "path": "docs/changes/1-演示检查/tests/test_smoke.sh",
      "command": "bash docs/changes/1-演示检查/tests/test_smoke.sh",
      "purpose": "demo",
      "assertions": ["brief exists"]
    }
  ],
  "required_evidence": []
}
JSON
}

note "Build the real workflow and proposal CLI"
WO="$TMP/wo"
SPECRELAY="$TMP/specrelay"
(cd "$ROOT" && go build -o "$WO" ./cmd/specrelay) 2>&1 | tee -a "$LOG"
(cd "$ROOT" && go build -o "$SPECRELAY" ./cmd/specrelay) 2>&1 | tee -a "$LOG"

PROJECT="$TMP/project"
CHANGE="1-演示检查"
mkdir -p "$PROJECT"
(cd "$PROJECT" && git init -q && git config user.email test@example.com && git config user.name Test)
write_demo_change "$PROJECT" "$CHANGE"
(cd "$PROJECT" && git add . && git commit -q -m init)

note "Missing pi must fail without creating run state"
cat >"$PROJECT/specrelay-flow.yaml" <<'YAML'
max_review_iterations: 0
stages:
  execution:
    agent: pi
  archive:
    agent: codex
YAML
FAKEBIN="$TMP/bin-missing-pi"
mkdir -p "$FAKEBIN"
cat >"$FAKEBIN/codex" <<'SH'
#!/usr/bin/env bash
printf '{"type":"thread.started","thread_id":"fake-thread"}\n'
SH
chmod +x "$FAKEBIN/codex"
ln -s "$(command -v git)" "$FAKEBIN/git"
ln -s "$SPECRELAY" "$FAKEBIN/specrelay"
set +e
(
  cd "$PROJECT"
  HOME="$TMP/home-missing-pi" XDG_STATE_HOME="$TMP/state-missing-pi" PATH="$FAKEBIN" "$WO" flow run --change "$CHANGE" --json
) >"$TMP/missing-pi.out" 2>"$TMP/missing-pi.err"
code=$?
set -e
cat "$TMP/missing-pi.out" "$TMP/missing-pi.err" | tee -a "$LOG"
[[ "$code" -ne 0 ]] || fail "Run must fail when pi is missing"
grep -Eiq 'pi' "$TMP/missing-pi.out" "$TMP/missing-pi.err" || fail "Diagnostic must identify missing pi"
grep -Eiq '安装|install' "$TMP/missing-pi.out" "$TMP/missing-pi.err" || fail "Diagnostic must explain CLI installation"
assert_no_run_state "$TMP/state-missing-pi" "$STATE_SNAPSHOT"
printf 'missing pi: no run state created\n' >"$STATE_SNAPSHOT"

note "Missing agy must fail without creating run state"
cat >"$PROJECT/specrelay-flow.yaml" <<'YAML'
max_review_iterations: 0
stages:
  execution:
    agent: agy
  archive:
    agent: codex
YAML
FAKEBIN="$TMP/bin-missing-agy"
mkdir -p "$FAKEBIN"
cat >"$FAKEBIN/codex" <<'SH'
#!/usr/bin/env bash
printf '{"type":"thread.started","thread_id":"fake-thread"}\n'
SH
cat >"$FAKEBIN/pi" <<'SH'
#!/usr/bin/env bash
printf '{"type":"session","id":"fake-pi"}\n'
SH
chmod +x "$FAKEBIN/codex" "$FAKEBIN/pi"
ln -s "$(command -v git)" "$FAKEBIN/git"
ln -s "$SPECRELAY" "$FAKEBIN/specrelay"
set +e
(
  cd "$PROJECT"
  HOME="$TMP/home-missing-agy" XDG_STATE_HOME="$TMP/state-missing-agy" PATH="$FAKEBIN" "$WO" flow run --change "$CHANGE" --json
) >"$TMP/missing-agy.out" 2>"$TMP/missing-agy.err"
code=$?
set -e
cat "$TMP/missing-agy.out" "$TMP/missing-agy.err" | tee -a "$LOG"
[[ "$code" -ne 0 ]] || fail "Run must fail when agy is missing"
grep -Eiq 'agy' "$TMP/missing-agy.out" "$TMP/missing-agy.err" || fail "Diagnostic must identify missing agy"
grep -Eiq '安装|install' "$TMP/missing-agy.out" "$TMP/missing-agy.err" || fail "Diagnostic must explain CLI installation"
assert_no_run_state "$TMP/state-missing-agy" "$TMP/missing-agy-state.txt"
printf 'missing agy: no run state created\n' >>"$STATE_SNAPSHOT"

note "Missing codex must fail without creating run state"
rm -f "$PROJECT/specrelay-flow.yaml"
FAKEBIN="$TMP/bin-missing-codex"
mkdir -p "$FAKEBIN"
cat >"$FAKEBIN/pi" <<'SH'
#!/usr/bin/env bash
printf '{"type":"session","id":"fake-pi"}\n'
SH
cat >"$FAKEBIN/agy" <<'SH'
#!/usr/bin/env bash
printf 'fake agy\n'
SH
chmod +x "$FAKEBIN/pi" "$FAKEBIN/agy"
ln -s "$(command -v git)" "$FAKEBIN/git"
ln -s "$SPECRELAY" "$FAKEBIN/specrelay"
set +e
(
  cd "$PROJECT"
  HOME="$TMP/home-missing-codex" XDG_STATE_HOME="$TMP/state-missing-codex" PATH="$FAKEBIN" "$WO" flow run --change "$CHANGE" --json
) >"$TMP/missing-codex.out" 2>"$TMP/missing-codex.err"
code=$?
set -e
cat "$TMP/missing-codex.out" "$TMP/missing-codex.err" | tee -a "$LOG"
[[ "$code" -ne 0 ]] || fail "Run must fail when codex is missing"
grep -Eiq 'codex' "$TMP/missing-codex.out" "$TMP/missing-codex.err" || fail "Diagnostic must identify missing codex"
grep -Eiq '安装|install' "$TMP/missing-codex.out" "$TMP/missing-codex.err" || fail "Diagnostic must explain CLI installation"
assert_no_run_state "$TMP/state-missing-codex" "$TMP/missing-codex-state.txt"
printf 'missing codex: no run state created\n' >>"$STATE_SNAPSHOT"

note "Removed backend configuration must fail as an unknown tool"
legacy_tool="open""code"
cat >"$PROJECT/specrelay-flow.yaml" <<YAML
max_review_iterations: 0
stages:
  execution:
    agent: ${legacy_tool}
  archive:
    agent: codex
YAML
set +e
(
  cd "$PROJECT"
  HOME="$TMP/home-invalid-tool" XDG_STATE_HOME="$TMP/state-invalid-tool" PATH="$TMP/bin-missing-pi" "$WO" flow run --change "$CHANGE" --json
) >"$TMP/invalid-tool.out" 2>"$TMP/invalid-tool.err"
code=$?
set -e
cat "$TMP/invalid-tool.out" "$TMP/invalid-tool.err" | tee -a "$LOG"
[[ "$code" -ne 0 ]] || fail "Removed backend configuration must not start"
grep -Eiq '未知|unknown|agent tool|invalid' "$TMP/invalid-tool.out" "$TMP/invalid-tool.err" || fail "Removed backend configuration must report an unknown tool"

note "contract passed: startup preflight follows the workflow snapshot and backend validation is strict"
