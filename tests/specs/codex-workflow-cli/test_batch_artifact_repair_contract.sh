#!/usr/bin/env bash
# Verify batch artifact repair contract.
# Sources: Proposal 6
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
RESULT_DIR="$ROOT/test-results/6-stage-artifact-gate/batch-artifact-repair"
TMP="$(mktemp -d)"

# cleanup: Remove the temporary repository and isolated user state.
cleanup() {
  rm -rf "$TMP"
}

# fail: Report the failed business contract and stop with a nonzero status.
fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

# note: Record each verification step in stdout and the runtime evidence log.
note() {
  printf '%s\n' "$*" | tee -a "$RESULT_DIR/contract.log"
}

trap cleanup EXIT
rm -rf "$RESULT_DIR"
mkdir -p "$RESULT_DIR"

SPECRELAY_BIN="$TMP/wo"
note "build real specrelay flow binary"
(cd "$ROOT" && go build -o "$SPECRELAY_BIN" ./cmd/specrelay) >>"$RESULT_DIR/contract.log" 2>&1

FAKEBIN="$TMP/fakebin"
mkdir -p "$FAKEBIN"

cat >"$FAKEBIN/specrelay" <<'SH'
#!/usr/bin/env bash
# Provide the real contract entry or minimal CLI interface used by this isolated fixture.
set -euo pipefail

case "$1" in
  list)
    printf '{"changes":[{"name":"1-a"},{"name":"2-b"}]}\n'
    ;;
  validate)
    printf '{"valid":true,"errors":[],"warnings":[],"artifacts":{}}\n'
    ;;
  *)
    printf 'unexpected specrelay command: %s\n' "$*" >&2
    exit 2
    ;;
esac
SH
chmod +x "$FAKEBIN/specrelay"

cat >"$FAKEBIN/codex" <<'SH'
#!/usr/bin/env bash
# Omit the first archive artifact to verify same-session batch repair.
set -euo pipefail

repo=""
session=""
while (($#)); do
  case "$1" in
    --cd)
      repo="$2"
      shift 2
      ;;
    resume)
      session="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
prompt="$(cat)"

CODEX_REPO="$repo" \
CODEX_SESSION="$session" \
CODEX_PROMPT="$prompt" \
python3 - <<'PY'
import json
import os
import pathlib
import re
import shutil

repo = pathlib.Path(os.environ["CODEX_REPO"])
session = os.environ["CODEX_SESSION"]
prompt = os.environ["CODEX_PROMPT"]
state_home = pathlib.Path(os.environ["XDG_STATE_HOME"])
attempt_dir = pathlib.Path(os.environ["CODEX_ATTEMPT_DIR"])
call_log = pathlib.Path(os.environ["CODEX_CALL_LOG"])

states = sorted((state_home / "specrelay" / "flow" / "repos").glob("*/runs/*/state.json"))
running = []
for path in states:
    state = json.loads(path.read_text(encoding="utf-8"))
    if state.get("status") == "running":
        running.append((path, state))
if not running:
    raise SystemExit("no running state found")
state_path, state = running[-1]
run_dir = state_path.parent
stage = state["stage"]
change = state["change_name"]
if stage == "archive":
    role = "archiver"
elif stage.startswith("audit_") or stage.startswith("targeted_repair_"):
    role = "repairer"
elif stage.startswith("qa_"):
    role = "qa"
else:
    role = "executor"

attempt_dir.mkdir(parents=True, exist_ok=True)
key = re.sub(r"[^A-Za-z0-9_.-]+", "_", f"{change}__{stage}")
attempt_path = attempt_dir / key
attempt = int(attempt_path.read_text(encoding="utf-8")) + 1 if attempt_path.exists() else 1
attempt_path.write_text(str(attempt), encoding="utf-8")

call_log.parent.mkdir(parents=True, exist_ok=True)
with call_log.open("a", encoding="utf-8") as fh:
    fh.write(json.dumps({
        "change": change,
        "stage": stage,
        "attempt": attempt,
        "session": session,
        "has_artifact_gate_prompt": "Stage artifact gate failed" in prompt,
    }, ensure_ascii=False) + "\n")

acceptance_path = repo / "docs" / "changes" / change / "acceptance.json"
if stage.startswith("audit_"):
    iteration = stage.split("_", 1)[1]
    (run_dir / f"audit-{iteration}.json").write_text(json.dumps({
        "summary": "batch audit clean",
        "decision": "clean",
        "findings": [],
        "non_blocking_findings": [],
        "evidence": ["go test ./... passed; runtime batch audit verified"],
        "checks": {
            "specrelay_aligned": True,
            "tests_meaningful": True,
            "implementation_scoped": True,
            "runtime_behavior_verified": True,
            "previous_findings_resolved": True
        }
    }), encoding="utf-8")
elif stage.startswith("qa_"):
    iteration = stage.split("_", 1)[1]
    (run_dir / f"qa-{iteration}.json").write_text(json.dumps({
        "summary": "batch QA clean",
        "decision": "clean",
        "findings": [],
        "non_blocking_findings": [],
        "acceptance_matrix": [{
            "id": "contract-" + change,
            "status": "passed",
            "artifact": f"docs/changes/{change}/tests/demo.sh",
            "evidence": "batch contract passed"
        }],
        "evidence": ["batch QA runtime verified"]
    }), encoding="utf-8")
elif stage == "archive" and (change != "1-a" or attempt >= 2):
    archive = repo / "docs" / "changes" / "archive" / ("2026-06-09-" + change)
    archive.parent.mkdir(parents=True, exist_ok=True)
    active = repo / "docs" / "changes" / change
    if active.exists() and not archive.exists():
        shutil.move(str(active), str(archive))
    (run_dir / "delivery-summary.md").write_text(f"archived {change}\n", encoding="utf-8")

print(json.dumps({"type": "thread.started", "thread_id": "thread-" + role}))
PY
SH
chmod +x "$FAKEBIN/codex"

PROJECT="$TMP/project"
mkdir -p "$PROJECT/docs/changes/1-a/tests" "$PROJECT/docs/changes/2-b/tests"
(
  cd "$PROJECT"
  git init -q
  git config user.email test@example.com
  git config user.name "Test User"
)

for change in 1-a 2-b; do
  cat >"$PROJECT/docs/changes/$change/proposal.md" <<MD
# $change

## 问题

验证 batch artifact gate retry。
MD
  cat >"$PROJECT/docs/changes/$change/design.md" <<'MD'
# 设计

使用 fake codex 验证 batch 继续执行。
MD
  cat >"$PROJECT/docs/changes/$change/spec.md" <<'MD'
# 规格

### 需求：batch artifact gate retry

系统必须修复当前 change 后继续 batch。
MD
  cat >"$PROJECT/docs/changes/$change/tests/demo.sh" <<SH
#!/usr/bin/env bash
set -euo pipefail
test -f docs/changes/$change/acceptance.json
SH
  chmod +x "$PROJECT/docs/changes/$change/tests/demo.sh"
  cat >"$PROJECT/docs/changes/$change/acceptance.json" <<JSON
{
  "summary": "$change acceptance",
  "required_tests": [
    {
      "id": "contract-$change",
      "source": "change_contract",
      "path": "docs/changes/$change/tests/demo.sh",
      "command": "bash docs/changes/$change/tests/demo.sh",
      "purpose": "prove $change test entry exists",
      "assertions": ["the current change acceptance contract exists"]
    }
  ],
  "required_evidence": []
}
JSON
done

cat >"$PROJECT/specrelay-flow.yaml" <<'YAML'
max_repair_iterations: 2
validation:
  limit: 3
  commands: []
stages:
  execution:
    agent: codex
  repair:
    agent: codex
  qa:
    agent: codex
  archive:
    agent: codex
YAML

git -C "$PROJECT" add .
git -C "$PROJECT" commit -q -m initial

repo_key="$(python3 - "$PROJECT" <<'PY'
import hashlib
import os
import sys

repo = os.path.abspath(sys.argv[1])
print(os.path.basename(repo).lower() + "-" + hashlib.sha1(repo.encode()).hexdigest()[:10])
PY
)"
batch_dir="$TMP/state/specrelay/flow/repos/$repo_key/batches/batch-artifact-retry"
mkdir -p "$batch_dir"
cat >"$batch_dir/state.json" <<'JSON'
{
  "batch_id": "batch-artifact-retry",
  "status": "running",
  "changes": ["1-a", "2-b"],
  "current_index": 0,
  "run_ids": {},
  "error": ""
}
JSON

note "run batch and expect first change archive artifact repair before second change"
set +e
CODEX_ATTEMPT_DIR="$TMP/attempts" \
CODEX_CALL_LOG="$RESULT_DIR/codex-calls.jsonl" \
XDG_STATE_HOME="$TMP/state" \
HOME="$TMP/home" \
PATH="$FAKEBIN:/usr/bin:/bin" \
  bash -c 'cd "$1" && "$2" flow batch --batch-id batch-artifact-retry --json' _ "$PROJECT" "$SPECRELAY_BIN" >"$RESULT_DIR/batch.jsonl" 2>"$RESULT_DIR/batch.err"
batch_code=$?
set -e
cat "$RESULT_DIR/batch.jsonl" >>"$RESULT_DIR/contract.log"
cat "$RESULT_DIR/batch.err" >>"$RESULT_DIR/contract.log"
[[ "$batch_code" -eq 0 ]] || fail "batch should continue after repairing first archive artifact"
cp -R "$TMP/state" "$RESULT_DIR/state"

python3 - "$batch_dir/state.json" "$RESULT_DIR/codex-calls.jsonl" "$TMP/state" "$PROJECT" <<'PY' || exit 1
import json
import pathlib
import sys

batch_state = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
records = [json.loads(line) for line in pathlib.Path(sys.argv[2]).read_text(encoding="utf-8").splitlines() if line.strip()]
state_home = pathlib.Path(sys.argv[3])
project = pathlib.Path(sys.argv[4])

if batch_state.get("status") != "done":
    raise SystemExit(f"batch status = {batch_state.get('status')!r}, want done")
if batch_state.get("current_index") != 2:
    raise SystemExit(f"current_index = {batch_state.get('current_index')!r}, want 2")
if set(batch_state.get("run_ids", {}).keys()) != {"1-a", "2-b"}:
    raise SystemExit(f"run_ids = {batch_state.get('run_ids')!r}, want both changes")

first_archive = [r for r in records if r["change"] == "1-a" and r["stage"] == "archive"]
if len(first_archive) < 2:
    raise SystemExit("1-a archive did not retry")
retry = first_archive[1]
if retry.get("session") != "thread-archiver":
    raise SystemExit(f"1-a archive retry session = {retry.get('session')!r}, want thread-archiver")
if not retry.get("has_artifact_gate_prompt"):
    raise SystemExit("1-a archive retry prompt did not include artifact gate failure")

run_states = sorted((state_home / "specrelay" / "flow" / "repos").glob("*/runs/*/state.json"))
if len(run_states) != 2:
    raise SystemExit(f"run state count = {len(run_states)}, want 2")
for path in run_states:
    state = json.loads(path.read_text(encoding="utf-8"))
    if state.get("status") != "done":
        raise SystemExit(f"{path} status = {state.get('status')!r}, want done")
print("batch artifact repair assertions passed")
PY

note "contract passed: batch continues after stage artifact repair"
