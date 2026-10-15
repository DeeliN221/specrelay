#!/usr/bin/env bash
# Verify status hard contract summary.
# Sources: Proposal 8
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

result_dir="$repo_root/test-results/8-强化验收硬合同"
mkdir -p "$result_dir"
log="$result_dir/status-hard-contract-summary.log"

cd "$repo_root"
specrelay="$tmp/specrelay"
go build -o "$specrelay" ./cmd/specrelay

project="$tmp/project"
change="1-状态合同摘要"
change_dir="$project/docs/changes/$change"
mkdir -p "$change_dir/tests"

cat >"$change_dir/brief.md" <<'MD'
# 状态合同摘要

用户从简报理解变更，调度器从 acceptance 摘要理解硬门槛规模。
MD

cat >"$change_dir/proposal.md" <<'MD'
# 状态合同摘要

## 背景

status 需要暴露硬合同规模。
MD

cat >"$change_dir/design.md" <<'MD'
# 设计

status JSON 读取 acceptance.json 并输出摘要。
MD

cat >"$change_dir/spec.md" <<'MD'
# 规格

### 需求：状态合同摘要

系统必须在 status JSON 中暴露验收合同规模。

#### 场景：status 包含合同摘要

- **当** 用户运行 specrelay status
- **则** JSON 包含 brief artifact 和 acceptance 摘要
MD

cat >"$change_dir/tests/test_contract.sh" <<'SH'
#!/usr/bin/env bash
# Real proposal contract entry: verify acceptance artifacts exist alongside the proposal.
set -euo pipefail
test -s docs/changes/1-状态合同摘要/brief.md
test -s docs/changes/1-状态合同摘要/acceptance.json
SH
chmod +x "$change_dir/tests/test_contract.sh"

cat >"$change_dir/acceptance.json" <<'JSON'
{
  "summary": "状态输出包含硬合同摘要",
  "coverage": [
    {
      "spec": "需求：状态合同摘要 / 场景：status 包含合同摘要",
      "tests": ["status-contract"],
      "evidence": ["status-json"],
      "risk": ""
    }
  ],
  "required_tests": [
    {
      "id": "status-contract",
      "source": "change_contract",
      "path": "docs/changes/1-状态合同摘要/tests/test_contract.sh",
      "command": "bash docs/changes/1-状态合同摘要/tests/test_contract.sh",
      "purpose": "证明 status 示例提案拥有真实契约测试入口",
      "assertions": [
        "brief.md 存在并作为用户简报 artifact 暴露",
        "acceptance.json 存在并可被 status 读取为合同摘要"
      ]
    }
  ],
  "required_evidence": [
    {
      "id": "status-json",
      "kind": "runtime_log",
      "path": "test-results/status-contract/status.json",
      "purpose": "记录 specrelay status --json 输出"
    }
  ]
}
JSON

(cd "$project" && "$specrelay" status "$change" --json) | tee "$log"

python3 - "$log" <<'PY'
import json
import sys

path = sys.argv[1]
payload = json.load(open(path, encoding="utf-8"))
artifacts = {item["name"]: item["status"] for item in payload.get("artifacts", [])}
if artifacts.get("brief.md") != "present":
    raise SystemExit(f"Status artifacts lack present brief.md: {artifacts}")

acceptance = payload.get("acceptance")
if not isinstance(acceptance, dict):
    raise SystemExit(f"Status JSON lacks the acceptance summary: {payload}")

checks = {
    "required_tests": 1,
    "required_evidence": 1,
    "coverage": 1,
}
for key, want in checks.items():
    got = acceptance.get(key, {}).get("total")
    if got != want:
        raise SystemExit(f"acceptance.{key}.total = {got}, want {want}")
PY
