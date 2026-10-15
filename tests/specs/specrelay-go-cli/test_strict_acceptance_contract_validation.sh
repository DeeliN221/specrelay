#!/usr/bin/env bash
# Verify strict acceptance contract validation.
# Sources: Proposal 8
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

result_dir="$repo_root/test-results/8-强化验收硬合同"
mkdir -p "$result_dir"
log="$result_dir/strict-acceptance-contract.log"

cd "$repo_root"
specrelay="$tmp/specrelay"
go build -o "$specrelay" ./cmd/specrelay

project="$tmp/project"
change="1-严格验收合同"
change_dir="$project/docs/changes/$change"
mkdir -p "$change_dir/tests"

cat >"$change_dir/brief.md" <<'MD'
# 严格验收合同

用户只需要看到这个简短说明；执行器应以 acceptance.json 和 tests 为硬合同。
MD

cat >"$change_dir/proposal.md" <<'MD'
# 严格验收合同

## 背景

测试合同必须能拒绝弱验收。
MD

cat >"$change_dir/design.md" <<'MD'
# 设计

使用真实 specrelay validate 命令校验 acceptance.json。
MD

cat >"$change_dir/spec.md" <<'MD'
# 规格

### 需求：严格验收合同

系统必须拒绝弱验收合同。

#### 场景：弱合同失败

- **当** 用户运行 specrelay validate
- **则** 缺断言或弱断言合同失败
MD

cat >"$change_dir/tests/test_contract.sh" <<'SH'
#!/usr/bin/env bash
# Real proposal contract entry: verify acceptance artifacts exist alongside the proposal.
set -euo pipefail
test -s docs/changes/1-严格验收合同/brief.md
test -s docs/changes/1-严格验收合同/acceptance.json
SH
chmod +x "$change_dir/tests/test_contract.sh"

cat >"$change_dir/acceptance.json" <<'JSON'
{
  "summary": "缺少业务断言的弱合同",
  "required_tests": [
    {
      "id": "contract-demo",
      "source": "change_contract",
      "path": "docs/changes/1-严格验收合同/tests/test_contract.sh",
      "command": "bash docs/changes/1-严格验收合同/tests/test_contract.sh",
      "purpose": "证明验收合同引用真实测试"
    }
  ],
  "required_evidence": []
}
JSON

if (cd "$project" && "$specrelay" validate "$change" --json) >"$tmp/missing-assertions.json" 2>"$tmp/missing-assertions.err"; then
  echo "Validation must reject acceptance.json missing required_tests[].assertions" | tee "$log" >&2
  exit 1
fi
grep -qi 'assertions' "$tmp/missing-assertions.err" "$tmp/missing-assertions.json"

cat >"$change_dir/acceptance.json" <<'JSON'
{
  "summary": "只包含弱表面断言的合同",
  "coverage": [
    {
      "spec": "需求：严格验收合同 / 场景：弱合同失败",
      "tests": ["contract-demo"],
      "evidence": [],
      "risk": "示例无额外证据"
    }
  ],
  "required_tests": [
    {
      "id": "contract-demo",
      "source": "change_contract",
      "path": "docs/changes/1-严格验收合同/tests/test_contract.sh",
      "command": "bash docs/changes/1-严格验收合同/tests/test_contract.sh",
      "purpose": "证明验收合同引用真实测试",
      "assertions": ["HTTP 200"]
    }
  ],
  "required_evidence": []
}
JSON

if (cd "$project" && "$specrelay" validate "$change" --json) >"$tmp/weak-assertion.json" 2>"$tmp/weak-assertion.err"; then
  echo "Validation must reject weak business assertions checking only HTTP 200" | tee "$log" >&2
  exit 1
fi
grep -Eqi '弱|assertions|HTTP 200' "$tmp/weak-assertion.err" "$tmp/weak-assertion.json"

cat >"$change_dir/acceptance.json" <<'JSON'
{
  "summary": "包含业务级断言和覆盖矩阵的强合同",
  "coverage": [
    {
      "spec": "需求：严格验收合同 / 场景：弱合同失败",
      "tests": ["contract-demo"],
      "evidence": ["validate-log"],
      "risk": ""
    }
  ],
  "required_tests": [
    {
      "id": "contract-demo",
      "source": "change_contract",
      "path": "docs/changes/1-严格验收合同/tests/test_contract.sh",
      "command": "bash docs/changes/1-严格验收合同/tests/test_contract.sh && mkdir -p test-results/strict-contract && printf 'validate passed\\n' > test-results/strict-contract/validate.log",
      "purpose": "证明验收合同引用真实测试并生成 validate-log",
      "assertions": [
        "brief.md 随 active change 落盘，用户可通过简报理解变更",
        "acceptance.json 通过真实测试入口被同一路径验证并生成 test-results/strict-contract/validate.log"
      ],
      "expected_initial_failure": "缺少 brief.md 或 acceptance.json 时测试失败"
    }
  ],
  "required_evidence": [
    {
      "id": "validate-log",
      "kind": "runtime_log",
      "path": "test-results/strict-contract/validate.log",
      "purpose": "记录 specrelay validate 对强合同的校验结果"
    }
  ]
}
JSON

(cd "$project" && "$specrelay" validate "$change" --json) | tee "$log"
grep -q '"valid": true' "$log"
