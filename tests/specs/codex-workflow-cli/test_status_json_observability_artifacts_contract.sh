#!/usr/bin/env bash
# Verify status json observability artifacts contract.
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
result_dir="$repo_root/test-results/7-status-watch-compact-output"

mkdir -p "$result_dir"
log="$result_dir/status-json-observability-artifacts.log"
cd "$repo_root"

go test ./internal/app -run TestStatusJSONObservabilityArtifactsContract -count=1 -v 2>&1 | tee "$log"
