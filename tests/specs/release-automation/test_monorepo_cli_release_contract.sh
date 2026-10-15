#!/usr/bin/env bash
# Verify the single-module CLI release contract.
# Keep build paths, CI gates, documentation, and release notes consistent.
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cd "$repo_root"

test -f "cmd/specrelay/main.go"
test -f "cmd/specrelay/main.go"
test -d "internal/app"
test -d "prompts-template"
test -f "README.md"
test -f "CHANGELOG.md"
test -f "scripts/extract-release-notes.sh"
test -f "docs/specs/release-automation/spec.md"

module="$(go list -m)"
test "$module" = "github.com/DeeliN221/specrelay"

app_import="$(go list -f '{{.ImportPath}}' ./internal/app)"
test "$app_import" = "github.com/DeeliN221/specrelay/internal/app"

old_workflow_module="github.com/xbugs221/""wo"
if go list -deps ./cmd/specrelay | grep -q "^$old_workflow_module"; then
  echo "cmd/specrelay still depends on the legacy workflow module" >&2
  exit 1
fi

go build -o "$tmp/specrelay" ./cmd/specrelay
"$tmp/specrelay" --version >/dev/null
"$tmp/specrelay" flow --help >/dev/null

workflow_dir="$repo_root/.github/workflows"
test -d "$workflow_dir"
shopt -s nullglob
workflows=("$workflow_dir"/*.yml "$workflow_dir"/*.yaml)
test "${#workflows[@]}" -gt 0

workflow_text="$tmp/workflows.txt"
cat "${workflows[@]}" >"$workflow_text"
if grep -q 'github.com/DeeliN221/specrelay/releases/latest/download' "$workflow_text"; then
  echo "workflow downloads an external latest SpecRelay instead of building the local source" >&2
  exit 1
fi
grep -q './cmd/specrelay' "$workflow_text"
grep -q 'go test ./...' "$workflow_text"
grep -q 'scripts/extract-release-notes.sh' "$workflow_text"
grep -q -- '--notes-file release-notes.md' "$workflow_text"
grep -q 'CHANGELOG.md --clobber' "$workflow_text"
if grep -q 'for script in tests/\*\.sh' "$workflow_text"; then
  echo "workflow must not blindly run historical root shell scripts" >&2
  exit 1
fi

grep -q 'GitHub Actions' README.md
grep -q 'CI' README.md
grep -q 'Release' README.md
grep -q 'go test ./...' README.md
grep -q 'CHANGELOG.md' README.md
grep -Eq 'local reproduction|Local verification|Build, test, and release' README.md

cat >"$tmp/changelog.md" <<'CHANGELOG'
# Changelog

## [Unreleased]

- Unreleased fallback note.

## [v9.8.7] - 2026-01-01

- Exact version note.

## [v9.8.6]

- Older version note.
CHANGELOG
bash scripts/extract-release-notes.sh "$tmp/changelog.md" v9.8.7 "$tmp/exact.md"
grep -q 'Exact version note' "$tmp/exact.md"
! grep -q 'Older version note' "$tmp/exact.md"
! grep -q 'Unreleased fallback note' "$tmp/exact.md"
bash scripts/extract-release-notes.sh "$tmp/changelog.md" v0.0.0-missing "$tmp/fallback.md"
grep -q 'Unreleased fallback note' "$tmp/fallback.md"
printf '# Changelog\n' >"$tmp/empty.md"
if bash scripts/extract-release-notes.sh "$tmp/empty.md" v0.0.0 "$tmp/invalid.md" 2>/dev/null; then
  echo "release-note extraction accepted an empty changelog" >&2
  exit 1
fi

grep -q 'go test ./...' docs/specs/release-automation/spec.md
grep -q 'build a local SpecRelay binary' docs/specs/release-automation/spec.md

echo "PASS"
