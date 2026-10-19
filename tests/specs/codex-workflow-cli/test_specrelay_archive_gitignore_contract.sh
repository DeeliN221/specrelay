#!/usr/bin/env bash
# Verify specrelay archive gitignore contract.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SKILL="$ROOT/skills/specrelay-archive/SKILL.md"

fail() {
  # Report the missing business contract and stop with a clear diagnostic.
  printf '[specrelay-archive-gitignore] FAIL: %s\n' "$*" >&2
  exit 1
}

assert_file_has() {
  # Verify the required contract pattern in the target file.
  local pattern="$1"
  rg -n "$pattern" "$SKILL" >/dev/null || fail "Missing pattern:$pattern"
}

[[ -f "$SKILL" ]] || fail "Missing skill file:$SKILL"
assert_file_has 'Do not use `git add \.`, `git add -A` or `git add -f`'
assert_file_has '`git diff --cached --name-only`'
assert_file_has '`git check-ignore --no-index -- <path>`'
assert_file_has 'Remove inadvertently staged ignored output'

printf '[specrelay-archive-gitignore] contract passed\n'
