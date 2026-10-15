#!/usr/bin/env bash
# Verify small brief only validate contract.
# Sources: Proposal 43
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
RESULT_DIR="$ROOT/test-results/specs/specrelay-go-cli"
LOG="$RESULT_DIR/small-brief-only-validate-contract.log"
OZ_BIN="$RESULT_DIR/specrelay-small-brief-only"
TMP="$(mktemp -d)"

mkdir -p "$RESULT_DIR"
: >"$LOG"
trap 'rm -rf "$TMP" "$OZ_BIN"' EXIT

note() {
  # Record contract steps and output so failures remain reviewable.
  printf '[small-brief-only-validate] %s\n' "$*" | tee -a "$LOG"
}

fail() {
  # Report the missing business contract and stop with a clear diagnostic.
  note "FAIL: $*"
  exit 1
}

init_project() {
  # Create an isolated Git repository without modifying the real worktree.
  local project="$1"
  mkdir -p "$project"
  git -C "$project" init -q
  git -C "$project" config user.email "test@example.com"
  git -C "$project" config user.name "Test User"
}

write_small_change() {
  # Write the minimum small-proposal artifacts for real validation.
  local project="$1"
  local change="$2"
  local with_test="$3"
  local change_dir="$project/docs/changes/$change"

  mkdir -p "$change_dir/tests"
  cat >"$change_dir/brief.md" <<'MD'
# 明确空tests校验

## 问题

空 tests 目录容易让执行器误判 small 提案已经具备测试合同。

## 范围

- `specrelay validate` 必须拒绝缺少真实测试代码的 small 提案。
- 合法 small 提案只需要 `brief.md`、`acceptance.json` 和 `tests/`。

## 非目标

- 不改变归档命令。
- 不引入完整 design 文档。

## 验收

- 给定 small brief-only 提案包含真实测试，validate 通过。
- 给定 small brief-only 提案没有真实测试，validate 失败。

## 规格去向

归档时合并到 `docs/specs/specrelay-go-cli.md` 的 small validate 场景。
MD

  cat >"$change_dir/acceptance.json" <<JSON
{
  "summary": "small brief-only 提案仍保留 acceptance 和 tests 硬合同。",
  "coverage": [
    {
      "spec": "brief：验收：给定 small brief-only 提案包含真实测试，validate 通过",
      "tests": ["small-contract"],
      "evidence": [],
      "risk": "只验证 validate 合同。"
    }
  ],
  "required_tests": [
    {
      "id": "small-contract",
      "source": "change_contract",
      "path": "docs/changes/$change/tests/small_contract_test.sh",
      "command": "bash docs/changes/$change/tests/small_contract_test.sh",
      "purpose": "证明 small brief-only 提案的测试硬合同存在且可运行。",
      "assertions": [
        "brief.md 说明 small 的问题、范围、非目标和验收",
        "acceptance.json required_tests 引用真实存在的测试文件"
      ],
      "expected_initial_failure": "该测试本身应可运行；实现前失败应来自 specrelay validate 仍要求完整长文档。"
    }
  ],
  "required_evidence": []
}
JSON

  if [[ "$with_test" == "yes" ]]; then
    cat >"$change_dir/tests/small_contract_test.sh" <<SH
#!/usr/bin/env bash
# Provide the real contract entry or minimal CLI interface used by this isolated fixture.
set -euo pipefail

test -s docs/changes/$change/brief.md
test -s docs/changes/$change/acceptance.json
grep -q 'small brief-only' docs/changes/$change/acceptance.json
SH
    chmod +x "$change_dir/tests/small_contract_test.sh"
  else
    cat >"$change_dir/tests/README.md" <<'MD'
这里只是说明文档，不是真实测试代码。
MD
  fi
}

cd "$ROOT"

note "Build the real specrelay executable"
go build -o "$OZ_BIN" ./cmd/specrelay 2>&1 | tee -a "$LOG"

note "Valid small brief-only proposals must pass validation"
GOOD_PROJECT="$TMP/good"
GOOD_CHANGE="1-明确空tests校验"
init_project "$GOOD_PROJECT"
write_small_change "$GOOD_PROJECT" "$GOOD_CHANGE" yes
(
  cd "$GOOD_PROJECT"
  "$OZ_BIN" validate "$GOOD_CHANGE" --json
) 2>&1 | tee -a "$LOG"

note "Small proposals missing real tests must fail"
BAD_PROJECT="$TMP/bad"
BAD_CHANGE="1-缺少真实测试"
init_project "$BAD_PROJECT"
write_small_change "$BAD_PROJECT" "$BAD_CHANGE" no
set +e
(
  cd "$BAD_PROJECT"
  "$OZ_BIN" validate "$BAD_CHANGE" --json
) >"$TMP/bad.out" 2>"$TMP/bad.err"
code=$?
set -e
cat "$TMP/bad.out" "$TMP/bad.err" >>"$LOG"
[[ "$code" -ne 0 ]] || fail "Small proposals without real test code must not pass validation"
rg -n 'tests.*测试文件|tests 包含非测试代码|缺少 tests' "$TMP/bad.out" "$TMP/bad.err" >>"$LOG" || fail "Diagnostic must identify the tests contract"

note "Standard proposals must retain full document validation"
STANDARD_PROJECT="$TMP/standard"
STANDARD_CHANGE="1-标准提案缺文档"
init_project "$STANDARD_PROJECT"
write_small_change "$STANDARD_PROJECT" "$STANDARD_CHANGE" yes
touch "$STANDARD_PROJECT/docs/changes/$STANDARD_CHANGE/proposal.md"
set +e
(
  cd "$STANDARD_PROJECT"
  "$OZ_BIN" validate "$STANDARD_CHANGE" --json
) >"$TMP/standard.out" 2>"$TMP/standard.err"
code=$?
set -e
cat "$TMP/standard.out" "$TMP/standard.err" >>"$LOG"
[[ "$code" -ne 0 ]] || fail "Standard proposal missing required documents must fail validation"
rg -n '缺少 design\.md|缺少 spec\.md|缺少 task\.md' "$TMP/standard.out" "$TMP/standard.err" >>"$LOG" || fail "Standard failure diagnostic must identify missing required documents"

note "PASS: small-brief-only-validate-contract"
