#!/usr/bin/env bash
# Sources: Proposal 12, Proposal 47
# Verify legacy acceptance compatibility contract.
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
result_dir="$repo_root/test-results/12-scope-gate"
log="$result_dir/legacy-acceptance-compatibility.log"
oz_bin="$result_dir/specrelay"
project="$result_dir/legacy-project"
change="1-旧提案兼容"

mkdir -p "$result_dir"
: >"$log"

note() {
  # note: Record each verification step in stdout and the runtime evidence log.
  printf '%s\n' "$*" | tee -a "$log"
}

cd "$repo_root"

note "Build the real specrelay executable"
go build -o "$oz_bin" ./cmd/specrelay 2>&1 | tee -a "$log"

note "Create an active change without a task file using legacy acceptance fields"
rm -rf "$project"
mkdir -p "$project/docs/changes/$change/tests"
git -C "$project" init -q

cat >"$project/docs/changes/$change/brief.md" <<'MD'
# 简报

旧提案兼容测试。该提案使用既有 acceptance.json 格式，不包含 scope 或 non_blocking 字段。

验收入口：

- `bash docs/changes/1-旧提案兼容/tests/test_legacy_contract.sh`
MD

cat >"$project/docs/changes/$change/proposal.md" <<'MD'
# 旧提案兼容

验证未运行旧提案无需迁移即可通过 specrelay validate。
MD

cat >"$project/docs/changes/$change/design.md" <<'MD'
# 设计

保持旧格式 acceptance 合同，不引入任何 review 或 QA 运行期字段。
MD

cat >"$project/docs/changes/$change/spec.md" <<'MD'
# 规格

## 验收矩阵

| 需求 | 场景 | 测试 | 证据 | 风险 |
| --- | --- | --- | --- | --- |
| 旧提案兼容 | 旧格式 acceptance 通过 validate | `legacy-contract` | `legacy-contract-log` | 不启动 sealed run |

### 需求：旧提案兼容

#### 场景：旧格式 acceptance 通过 validate

- **给定** 旧格式 active change
- **当** 用户运行 `specrelay validate`
- **则** `specrelay validate` 应当校验成功
MD

cat >"$project/docs/changes/$change/tests/test_legacy_contract.sh" <<'SH'
#!/usr/bin/env bash
# Provide the real contract entry or minimal CLI interface used by this isolated fixture.
set -euo pipefail

test -s docs/changes/1-旧提案兼容/acceptance.json
grep -q "旧格式 active change" docs/changes/1-旧提案兼容/brief.md
SH
chmod +x "$project/docs/changes/$change/tests/test_legacy_contract.sh"

cat >"$project/docs/changes/$change/acceptance.json" <<'JSON'
{
  "summary": "旧格式 active change 不包含 scope 或 non_blocking 字段也应继续通过 validate。",
  "coverage": [
    {
      "spec": "需求：旧提案兼容 / 场景：旧格式 acceptance 通过 validate",
      "tests": ["legacy-contract"],
      "evidence": ["legacy-contract-log"],
      "risk": "只验证 validate 阶段，不启动 sealed run。"
    }
  ],
  "required_tests": [
    {
      "id": "legacy-contract",
      "source": "change_contract",
      "path": "docs/changes/1-旧提案兼容/tests/test_legacy_contract.sh",
      "command": "bash docs/changes/1-旧提案兼容/tests/test_legacy_contract.sh && mkdir -p test-results/legacy-active-change && printf 'legacy acceptance passed\\n' > test-results/legacy-active-change/legacy-contract.log",
      "purpose": "证明旧格式 active change 的测试和 acceptance 合同仍可被 validate 接受，并生成 legacy-contract-log。",
      "assertions": [
        "acceptance.json 存在且无需 scope 字段",
        "brief.md 说明这是旧格式 active change，并生成 test-results/legacy-active-change/legacy-contract.log"
      ],
      "expected_initial_failure": "该兼容性测试当前应通过；执行阶段必须保持通过。"
    }
  ],
  "required_evidence": [
    {
      "id": "legacy-contract-log",
      "kind": "runtime_log",
      "path": "test-results/legacy-active-change/legacy-contract.log",
      "purpose": "记录旧格式 active change validate 输出。"
    }
  ]
}
JSON

note "Run real validation and verify unstarted legacy proposals need no scope field"
(
  cd "$project"
  "$oz_bin" validate "$change" --json
) 2>&1 | tee -a "$log"

note "PASS"
