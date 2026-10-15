#!/usr/bin/env bash
# Extract a version or Unreleased section for GitHub Release.

set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <CHANGELOG.md> <version-tag> <output-file>" >&2
  exit 2
fi

changelog_path="$1"
version="$2"
output_path="$3"
notes_tmp="$(mktemp)"
trap 'rm -f "$notes_tmp"' EXIT

# Extract the body between two level-two version headings.
extract_section() {
  local heading="$1"
  awk -v heading="$heading" '
    $0 == heading || index($0, heading " - ") == 1 { found = 1; next }
    found && /^## \[/ { exit }
    found { print }
  ' "$changelog_path"
}

extract_section "## [$version]" >"$notes_tmp"
if ! grep -q '[^[:space:]]' "$notes_tmp"; then
  extract_section "## [Unreleased]" >"$notes_tmp"
fi
if ! grep -q '[^[:space:]]' "$notes_tmp"; then
  # Accept the translated heading in older changelogs.
  extract_section "## [尚未发布]" >"$notes_tmp"
fi
if ! grep -q '[^[:space:]]' "$notes_tmp"; then
  echo "CHANGELOG.md has no release notes for [$version] or [Unreleased]" >&2
  exit 1
fi

mv "$notes_tmp" "$output_path"
trap - EXIT
