#!/usr/bin/env bash
# Verify default go dag run contract.
# Sources: Proposal 3
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

log_dir="$repo_root/test-results/workflow"
mkdir -p "$log_dir"
log="$log_dir/default-内嵌工作流-run-contract.log"
: >"$log"

# note: Record each verification step in stdout and the runtime evidence log.
note() {
  printf '%s\n' "$*" | tee -a "$log"
}

# fail: Report the failed business contract and stop with a nonzero status.
fail() {
  note "FAIL: $*"
  exit 1
}

cd "$repo_root"
wo="$tmp/wo"
specrelay="$tmp/specrelay"
go build -o "$wo" ./cmd/specrelay 2>&1 | tee -a "$log"
go build -o "$specrelay" ./cmd/specrelay 2>&1 | tee -a "$log"

fakebin="$tmp/fakebin"
mkdir -p "$fakebin"
ln -s "$specrelay" "$fakebin/specrelay"

cat >"$fakebin/codex" <<'SH'
#!/usr/bin/env bash
# The fake agent writes only the artifacts and task updates required by this isolated workflow fixture.
set -euo pipefail

prompt="$(cat)"
args="$*"
if [ -n "$args" ]; then
  prompt="$prompt
$args"
fi

python3 - "$prompt" <<'PY'
import json
import os
import pathlib
import re
import sys

prompt = sys.argv[1]
name = re.search(r"SUBAGENT_NAME=(.+)", prompt)
if name:
    purpose = re.search(r"SUBAGENT_PURPOSE=(.+)", prompt)
    change_name = re.search(r"CURRENT_CHANGE=(.+)", prompt)
    body = {
        "name": name.group(1).strip(),
        "change_name": change_name.group(1).strip() if change_name else "demo",
        "purpose": purpose.group(1).strip() if purpose else "执行并行成员职责",
        "status": "success",
        "summary": "fake agent completed the configured read-only subagent task",
        "evidence": ["test-results/workflow/default-内嵌工作流-run-contract.log"]
    }
    print(json.dumps({"type": "thread.started", "thread_id": "fake-subagent-session"}))
    print(json.dumps({"type": "message", "message": {"role": "assistant", "content": [{"type": "text", "text": json.dumps(body, ensure_ascii=False)}]}}, ensure_ascii=False))
    raise SystemExit(0)

state_home = pathlib.Path(os.environ["XDG_STATE_HOME"])
states = sorted(state_home.glob("specrelay/flow/repos/*/runs/*/state.json"))
if not states:
    raise SystemExit("no specrelay flow state.json found")
state_path = states[-1]
state = json.loads(state_path.read_text(encoding="utf-8"))
repo = pathlib.Path(os.environ["OZ_TEST_REPO"])
run_dir = state_path.parent
change = state["change_name"]
stage = state["stage"]

if stage == "execution":
    evidence = repo / "test-results" / "内嵌工作流" / "temporary.log"
    evidence.parent.mkdir(parents=True, exist_ok=True)
    evidence.write_text("execution evidence\n", encoding="utf-8")
elif stage == "archive":
    archive = repo / "docs" / "changes" / "archive" / ("2026-06-09-" + change)
    archive.mkdir(parents=True, exist_ok=True)
    (archive / "acceptance.json").write_text((repo / "docs" / "changes" / change / "acceptance.json").read_text(encoding="utf-8"), encoding="utf-8")
    (run_dir / "delivery-summary.md").write_text("fake 内嵌工作流 archive completed\n", encoding="utf-8")

print(json.dumps({"type": "thread.started", "thread_id": "fake-main-session-" + stage}))
PY
SH
chmod +x "$fakebin/codex"

cp "$fakebin/codex" "$fakebin/pi"
cp "$fakebin/codex" "$fakebin/pi"
cp "$fakebin/codex" "$fakebin/agy"

project="$tmp/project"
mkdir -p "$project/docs/changes/1-默认内嵌工作流/tests"
git -C "$project" init -q
git -C "$project" config user.email "test@example.com"
git -C "$project" config user.name "Test User"

cat >"$project/docs/changes/1-默认内嵌工作流/proposal.md" <<'MD'
# 默认工作流

## 背景

这个临时 change 用来验证 specrelay flow 默认运行路径。
MD

cat >"$project/docs/changes/1-默认内嵌工作流/brief.md" <<'MD'
# 默认工作流

验证默认 specrelay flow run 使用 内嵌工作流 推进真实 change，并保留状态观测合同。
MD

cat >"$project/docs/changes/1-默认内嵌工作流/design.md" <<'MD'
# 设计

使用最小任务证明默认工作流 运行会推进真实 change。
MD

cat >"$project/docs/changes/1-默认内嵌工作流/spec.md" <<'MD'
# 规格

### 需求：默认工作流

系统必须默认使用 内嵌工作流。

#### 场景：运行并归档

- **当** 用户运行 specrelay flow run
- **则** run 完成并记录 内嵌工作流 状态
MD

cat >"$project/docs/changes/1-默认内嵌工作流/tests/test_contract.sh" <<'SH'
#!/usr/bin/env bash
# Real temporary proposal test proves tests/ is not a placeholder.
set -euo pipefail
test -f docs/changes/1-默认内嵌工作流/acceptance.json
SH
chmod +x "$project/docs/changes/1-默认内嵌工作流/tests/test_contract.sh"

cat >"$project/docs/changes/1-默认内嵌工作流/acceptance.json" <<'JSON'
{
  "summary": "验证默认工作流 run 契约",
  "required_tests": [
    {
      "id": "temporary-contract",
      "source": "change_contract",
      "path": "docs/changes/1-默认内嵌工作流/tests/test_contract.sh",
      "command": "bash docs/changes/1-默认内嵌工作流/tests/test_contract.sh",
      "purpose": "证明临时 change 包含真实测试入口",
      "assertions": ["默认 specrelay flow run 使用 内嵌工作流 推进 execution 并完成 archive"]
    }
  ],
  "required_evidence": []
}
JSON

cat >"$project/oz-flow.yaml" <<'YAML'
max_review_iterations: 0
YAML

git -C "$project" add .
git -C "$project" commit -q -m initial

note "Run the default workflow and verify a real change advances"
OZ_TEST_REPO="$project" \
XDG_STATE_HOME="$tmp/state" \
HOME="$tmp/home" \
PATH="$fakebin:/usr/bin:/bin" \
  bash -c 'cd "$1" && "$2" flow run --change "1-默认内嵌工作流" --json' _ "$project" "$wo" >"$tmp/run.jsonl" 2>"$tmp/run.err" || {
    cat "$tmp/run.err" | tee -a "$log"
    fail "Default specrelay flow run failed"
  }

cat "$tmp/run.jsonl" >>"$log"

state="$(find "$tmp/state/oz/flow/repos" -path '*/runs/*/state.json' -print | sort | tail -n 1)"
test -n "$state" || fail "Missing run state.json"
note "state: $state"
cat "$state" >>"$log"

python3 - "$state" <<'PY' || exit 1
import json
import sys
state = json.load(open(sys.argv[1], encoding="utf-8"))
if state.get("engine") != "内嵌工作流":
    raise SystemExit("state.engine must identify the embedded workflow")
if state.get("workflow_config", {}).get("parallel", {}).get("enabled") is True:
    raise SystemExit("workflow_config.parallel.enabled must not expose removed fixed subagents")
PY

run_id="$(python3 - "$state" <<'PY'
import json, sys
print(json.load(open(sys.argv[1], encoding="utf-8"))["run_id"])
PY
)"

note "Verify human status contains only main stages"
OZ_TEST_REPO="$project" \
XDG_STATE_HOME="$tmp/state" \
HOME="$tmp/home" \
PATH="$fakebin:/usr/bin:/bin" \
  bash -c 'cd "$1" && "$2" flow status -w1' _ "$project" "$wo" >"$tmp/status.txt"
cat "$tmp/status.txt" >>"$log"
grep -qF "fake-main-session-execution" "$tmp/status.txt" || fail "specrelay flow status must show the main execution session"
if rg -q 'implementation_context|parallel-|fan-in' "$tmp/status.txt"; then
  fail "specrelay flow status must not show removed fixed subagents or fan-in"
fi

note "Verify JSON status preserves the legacy runner contract"
OZ_TEST_REPO="$project" \
XDG_STATE_HOME="$tmp/state" \
HOME="$tmp/home" \
PATH="$fakebin:/usr/bin:/bin" \
  bash -c 'cd "$1" && "$2" flow status --run-id "$3" --json' _ "$project" "$wo" "$run_id" >"$tmp/status.json"
cat "$tmp/status.json" >>"$log"
python3 - "$tmp/status.json" <<'PY' || exit 1
import json
import sys
payload = json.load(open(sys.argv[1], encoding="utf-8"))
for key in ["run_id", "change_name", "status", "stage", "stages", "paths", "sessions", "error"]:
    if key not in payload:
        raise SystemExit(f"missing JSON key {key}")
for forbidden in ["parallel", "parallel_status", "parallel_summary", "members"]:
    if forbidden in payload:
        raise SystemExit(f"JSON status must not expose {forbidden}")
PY

note "PASS"
