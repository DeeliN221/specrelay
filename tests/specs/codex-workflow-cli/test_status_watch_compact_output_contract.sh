#!/usr/bin/env bash
# Verify status watch compact output contract.
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
result_dir="$repo_root/test-results/7-status-watch-compact-output"
tmpdir=""

mkdir -p "$result_dir"
log="$result_dir/status-watch-compact-output.log"
: >"$log"

cleanup() {
  if [[ -n "$tmpdir" ]]; then
    rm -rf "$tmpdir"
  fi
}
trap cleanup EXIT

note() {
  # note: Record each verification step in stdout and the runtime evidence log.
  printf '%s\n' "$*" | tee -a "$log"
}

cd "$repo_root"

note "Run maintained Go contracts for status/watch rendering"
go test ./internal/app -run TestStatusWatchCompactOutputContract -count=1 -v 2>&1 | tee -a "$log"

if ! command -v script >/dev/null 2>&1; then
  echo "Missing script command; cannot run pseudo-TTY watch contract" | tee -a "$log" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"

export XDG_STATE_HOME="$tmpdir/state"
tty_repo="$tmpdir/repo"
oz_bin="$tmpdir/wo"
raw_capture="$result_dir/watch-tty.raw"
screen_capture="$result_dir/watch-tty-screen.txt"
long_change="9-这是一个非常非常长的中文提案名称用于触发窄终端自动换行并验证watch不会残留旧首行"

note "Build real specrelay flow and create a narrow-TTY watch scenario"
go build -C "$repo_root" -o "$oz_bin" ./cmd/specrelay 2>&1 | tee -a "$log"
mkdir -p "$tty_repo"
cd "$tty_repo"
git init >/dev/null
git config user.email test@example.com
git config user.name Test
printf 'test\n' > README.md
git add .
git commit -m init >/dev/null
mkdir -p "docs/changes/$long_change/tests"

python3 - "$tty_repo" "$long_change" <<'PY'
import hashlib
import json
import os
import pathlib
import re
import sys

repo = pathlib.Path(sys.argv[1]).resolve()
change = sys.argv[2]
name = re.sub(r"[^a-z0-9]+", "-", repo.name.lower()).strip("-") or "repo"
digest = hashlib.sha1(str(repo).encode()).hexdigest()[:10]
base = pathlib.Path(os.environ["XDG_STATE_HOME"]) / "specrelay" / "flow" / "repos" / f"{name}-{digest}"
run_id = "20260610T020000.000000000Z"
batch_id = "20260610T020001.000000000Z"

def write_json(path, payload):
    # write_json: Write a real workflow runtime-state fixture.
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")

workflow = {
    "engine": "内嵌工作流",
    "max_review_iterations": 1,
    "stages": {},
    "prompts": {},
    "parallel": {"enabled": False, "groups": {}},
    "validation": {"commands": []},
}
write_json(base / "runs" / run_id / "state.json", {
    "run_id": run_id,
    "change_name": change,
    "sealed": True,
    "status": "running",
    "stage": "execution",
    "engine": "内嵌工作流",
    "sessions": {"codex:executor": "writer-session"},
    "stages": {"planning": "completed"},
    "paths": {},
    "workflow_config": workflow,
})
write_json(base / "batches" / batch_id / "state.json", {
    "batch_id": batch_id,
    "status": "running",
    "changes": [change],
    "current_index": 0,
    "run_ids": {change: run_id},
})
PY

note "Allocate a narrow pseudo-TTY and capture repeated watch frames"
COLUMNS=24 timeout -s INT 3s script -q -c "$oz_bin flow watch" /dev/null >"$raw_capture" 2>/dev/null || true

note "Parse terminal control sequences and reconstruct the final screen"
python3 - "$raw_capture" "$screen_capture" "$long_change" <<'PY' 2>&1 | tee -a "$log"
import re
import sys
import unicodedata

raw_path, screen_path, long_change = sys.argv[1:4]
width = 24
data = open(raw_path, "rb").read().decode("utf-8", errors="ignore")
rows = [[]]
row = 0
col = 0
i = 0

def cell_width(ch):
    # cell_width: Approximate terminal column width for wrapping Unicode proposal names.
    if unicodedata.combining(ch):
        return 0
    return 2 if unicodedata.east_asian_width(ch) in {"W", "F"} else 1

def ensure_row(n):
    # ensure_row: Extend the screen buffer for ANSI clear-screen simulation.
    while len(rows) <= n:
        rows.append([])

def put_char(ch):
    # put_char: Write by terminal column width and handle automatic wrapping.
    global row, col
    w = cell_width(ch)
    if col + w > width:
        row += 1
        col = 0
    ensure_row(row)
    line = rows[row]
    while len(line) < col:
        line.append(" ")
    if col == len(line):
        line.append(ch)
    else:
        line[col] = ch
    col += w
    if col >= width:
        row += 1
        col = 0
        ensure_row(row)

while i < len(data):
    ch = data[i]
    if ch == "\x1b":
        match = re.match(r"\x1b\[([0-9;?]*)([A-Za-z])", data[i:])
        if match:
            params, code = match.groups()
            if code == "A":
                row = max(0, row - int(params or "1"))
            elif code == "H":
                row = 0
                col = 0
            elif code == "J":
                mode = params or "0"
                if mode in {"0", "2"}:
                    ensure_row(row)
                    rows[row:] = [[]]
                    if mode == "2":
                        row = 0
                        col = 0
            i += len(match.group(0))
            continue
    if ch == "\r":
        col = 0
    elif ch == "\n":
        row += 1
        col = 0
        ensure_row(row)
    elif ch >= " ":
        put_char(ch)
    i += 1

visible = ["".join(line).rstrip() for line in rows]
business = [
    line.strip()
    for line in visible
    if line.strip()
    and not line.startswith("Script started")
    and not line.startswith("Script done")
]
open(screen_path, "w", encoding="utf-8").write("\n".join(business) + "\n")
print("\n".join(business))
joined = "".join(business).replace(" ", "")
if not joined.startswith("-" + long_change):
    raise SystemExit(f"First business content must be the proposal list; actual screen: {joined!r}")
if re.search(r"([|/\\\\-]?b1|[|/\\\\-]?w1)", joined):
    raise SystemExit(f"Final screen retains a run header: {joined!r}")
if not re.search(r"执行writer-session[|/\\\\-]", joined):
    raise SystemExit("Final screen lacks the execution spinner marker")
PY
