#!/usr/bin/env bash
# Verify ozcli command boundary contract.
# Sources: Proposal 34, Proposal 47
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
log="$repo_root/test-results/specrelay-go-cli/ozcli-command-boundary-contract.log"
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

assert_file_has() {
  # assert_file_has: Verify the required source file and responsibility pattern exist.
  local file="$1"
  local pattern="$2"
  [[ -f "$file" ]] || fail "Missing target file:$file"
  rg -n "$pattern" "$file" >>"$log" || fail "$file Missing pattern:$pattern"
}

assert_file_lacks() {
  # assert_file_lacks: Reject reintroduction of an extracted responsibility.
  local file="$1"
  local pattern="$2"
  [[ -f "$file" ]] || fail "Missing target file:$file"
  if rg -n "$pattern" "$file" >>"$log"; then
    fail "$file must not contain pattern:$pattern"
  fi
}

cd "$repo_root"

note "evidence id: ozcli-command-boundary-log"
note "evidence path: $log"
note "test id: ozcli-command-boundary-contract"

assert_file_has "internal/ozcli/cli.go" 'func Main'
assert_file_has "internal/ozcli/cli.go" 'func \(c \*cli\) run'
assert_file_has "internal/ozcli/cmd_install.go" 'func \(c \*cli\) installCmd'
assert_file_has "internal/ozcli/cmd_install.go" 'func \(c \*cli\) printInstallHelp'
assert_file_has "internal/ozcli/cmd_change.go" 'func \(c \*cli\) listCmd'
assert_file_has "internal/ozcli/cmd_change.go" 'func \(c \*cli\) createCmd'
assert_file_has "internal/ozcli/cmd_change.go" 'func \(c \*cli\) statusCmd'
assert_file_has "internal/ozcli/cmd_validate.go" 'func \(c \*cli\) validateCmd'
assert_file_has "internal/ozcli/cmd_validate.go" 'func validateChange'
assert_file_has "internal/ozcli/cmd_validate.go" 'func validateAcceptanceFiles'
assert_file_has "internal/ozcli/cmd_archive.go" 'func \(c \*cli\) archiveCmd'
assert_file_has "internal/ozcli/cmd_validate.go" 'active proposals must not contain task.md'
assert_file_lacks "internal/ozcli/cmd_archive.go" 'ensureTasksDone'
assert_file_lacks "internal/ozcli/cmd_archive.go" 'task.md'

if [[ -f internal/ozcli/ozcli.go ]]; then
  line_count="$(wc -l < internal/ozcli/ozcli.go | tr -d ' ')"
  note "ozcli.go line_count=$line_count"
  (( line_count <= 180 )) || fail "ozcli.go remains too large; command responsibilities were not extracted"
fi

note "Run internal/ozcli Go regressions"
go test ./internal/ozcli -count=1 2>&1 | tee -a "$log"

note "PASS: ozcli-command-boundary-contract"
