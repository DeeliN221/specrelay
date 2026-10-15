#!/usr/bin/env bash
# Verify wo command dispatch boundary contract.
# Sources: Proposal 28
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
log="$repo_root/test-results/refactor-command-dispatch/specrelay-flow-command-dispatch-boundary-contract.log"
mkdir -p "$(dirname "$log")"
: >"$log"

note() {
  # note: Record each verification step in stdout and the runtime evidence log.
  printf '%s\n' "$*" | tee -a "$log"
}

fail() {
  # fail: Report the failed business contract and stop with a nonzero status.
  note "FAIL: $*"
  exit 1
}

cd "$repo_root"

note "evidence id: specrelay-flow-command-dispatch-boundary-log"
note "evidence path: $log"
note "test id: specrelay-flow-command-dispatch-boundary-contract"

for file in \
  internal/app/command_dispatch.go \
  internal/app/interactive.go \
  internal/app/planning.go
do
  [[ -f "$file" ]] || fail "Missing command-dispatch boundary file:$file"
  note "Found boundary file:$file"
done

for command_case in \
  'case "run":' \
  'case "resume":' \
  'case "batch":' \
  'case "restart":' \
  'case "status":' \
  'case "abort":' \
  'case "clean":' \
  'case "watch":' \
  'case "--resume":' \
  'case "--run":'
do
  if rg -n "$command_case" internal/app/app.go | tee -a "$log" | grep -q .; then
    fail "app.go still directly contains a repository command case:$command_case"
  fi
done

note "Run internal/app and cmd/specrelay command regressions"
go test ./internal/app ./cmd/specrelay -count=1 2>&1 | tee -a "$log"

note "PASS: specrelay-flow-command-dispatch-boundary-contract"
