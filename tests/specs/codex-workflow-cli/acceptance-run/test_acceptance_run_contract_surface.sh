#!/usr/bin/env bash
# Verify acceptance run contract surface.
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
RESULT_DIR="$ROOT/test-results/37-执行验收合同测试并汇总结果/surface"
LOG="$RESULT_DIR/contract.log"
TMP="$(mktemp -d)"

cleanup() {
  rm -rf "$TMP"
}

fail() {
  printf 'FAIL: %s\n' "$*" | tee -a "$LOG" >&2
  exit 1
}

note() {
  printf '%s\n' "$*" | tee -a "$LOG"
}

trap cleanup EXIT
rm -rf "$RESULT_DIR"
mkdir -p "$RESULT_DIR"

note "build real specrelay binary"
SPECRELAY_BIN="$TMP/specrelay"
(cd "$ROOT" && go build -o "$SPECRELAY_BIN" ./cmd/specrelay) >>"$LOG" 2>&1

note "check specrelay flow help exposes run-acceptance"
"$SPECRELAY_BIN" flow --help >"$RESULT_DIR/help.txt" 2>"$RESULT_DIR/help.err" || fail "specrelay flow --help failed"
grep -q 'specrelay flow run-acceptance --change <change-name> --json' "$RESULT_DIR/help.txt" || fail "help missing run-acceptance JSON command"

note "check runner contract exposes run-acceptance capability"
"$SPECRELAY_BIN" flow contract --json >"$RESULT_DIR/contract.json" 2>"$RESULT_DIR/contract.err" || fail "contract --json failed"
python3 - "$RESULT_DIR/contract.json" <<'PY' >>"$LOG" 2>&1
import json
import sys

path = sys.argv[1]
payload = json.loads(open(path, encoding="utf-8").read())
if payload.get("json") is not True:
    raise SystemExit("contract json flag must be true")
capabilities = payload.get("capabilities") or []
if "run-acceptance" not in capabilities:
    raise SystemExit(f"missing run-acceptance capability: {capabilities!r}")
print("run-acceptance capability is present")
PY

note "surface contract passed; evidence: test-results/37-执行验收合同测试并汇总结果/surface/contract.log"
