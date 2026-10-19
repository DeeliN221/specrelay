#!/usr/bin/env bash
# Verify root test layout contract.
# Sources: Proposal 14, Proposal 20, Proposal 35, Proposal 36
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
LOG="$ROOT/test-results/spec-root-test-layout.log"
mkdir -p "$(dirname "$LOG")"
: >"$LOG"

# note: Record each verification step in stdout and the runtime evidence log.
note() {
  printf '[root-test-layout] %s\n' "$*" | tee -a "$LOG"
}

# fail: Report the failed business contract and stop with a nonzero status.
fail() {
  printf '[root-test-layout] FAIL: %s\n' "$*" | tee -a "$LOG" >&2
  exit 1
}

note "Verify root tests contain executable business tests"
[[ -d "$ROOT/tests" ]] || fail "Missing root tests directory"
root_test_count="$(cd "$ROOT" && { fd -e go -e sh . tests 2>/dev/null || true; } | wc -l | tr -d ' ')"
[[ "$root_test_count" -gt 0 ]] || fail "Root tests contain no executable entry"

note "Verify tests/app no longer retains migration inputs"
if [[ -d "$ROOT/tests/app" ]] && app_inputs="$(cd "$ROOT" && { fd -t f . tests/app 2>/dev/null || true; } | sort)" && [[ -n "$app_inputs" ]]; then
  printf '%s\n' "$app_inputs" | tee -a "$LOG"
  fail "tests/app still contains migration inputs"
fi

note "Verify the dynamic gotest runner was removed"
migrated_runner_path="$ROOT/tests/app/migrated_app_""suite_test.go"
if [[ -f "$migrated_runner_path" ]]; then
  fail "tests/app/migrated_app_""suite_test.go remains; the dynamic runner was not removed"
fi
dynamic_runner_pattern='\.go''test|SPECRELAY_MIGRATED_APP_''RUN|migrated_app_''suite'
if matches="$(cd "$ROOT" && rg -n "$dynamic_runner_pattern" tests/specs tests/app internal/app 2>/dev/null || true)" && [[ -n "$matches" ]]; then
  printf '%s\n' "$matches" | tee -a "$LOG"
  fail "Dynamic gotest or migrated runner references remain"
fi
if gotests="$(cd "$ROOT" && fd -e gotest . tests internal/app 2>/dev/null || true)" && [[ -n "$gotests" ]]; then
  printf '%s\n' "$gotests" | tee -a "$LOG"
  fail "Executable test paths still contain gotest files"
fi

note "Verify a tests/app or tests/specs group exists"
if [[ ! -d "$ROOT/tests/app" && ! -d "$ROOT/tests/specs" ]]; then
  fail "Root tests lack an app or specs group"
fi

note "Verify root tests exclude date-prefixed legacy shell tests"
dated_tests="$(cd "$ROOT" && fd -t f '^2026-.*\.sh$' tests 2>/dev/null || true)"
if [[ -n "$dated_tests" ]]; then
  printf '%s\n' "$dated_tests" | tee -a "$LOG"
  fail "Root retains date-prefixed historical shell tests requiring migration"
fi

legacy_cli="w""o"
legacy_env="W""O"
legacy_pattern="cmd/${legacy_cli}|\\./cmd/${legacy_cli}|${legacy_cli}\\.yaml|(^|/)\\.${legacy_cli}(/|$)|/${legacy_cli}/repos|XDG_STATE_HOME[^[:space:]]*/${legacy_cli}|\\b${legacy_cli} (status|watch|run|clean|config|restart|resume|batch|abort|update|graph|contract|list-changes)\\b|\\b${legacy_env}_TEST_|\\b${legacy_env}_PLANNING|\\b${legacy_env}="

note "Verify root tests do not reference obsolete CLI surfaces"
root_hits="$(cd "$ROOT" && rg -n --glob '!tests/specs/**' "$legacy_pattern" tests 2>/dev/null || true)"
if [[ -n "$root_hits" ]]; then
  printf '%s\n' "$root_hits" | tee -a "$LOG"
  fail "Root tests reference obsolete commands, config, state paths or environment variables"
fi

note "Verify core test packages run independently"
(cd "$ROOT" && go test ./internal/app ./internal/specrelaycli ./cmd/specrelay ./tests -count=1) 2>&1 | tee -a "$LOG" || fail "Core test packages failed"

note "Verify go install package paths build"
tmp_bin_dir="$(mktemp -d)"
(cd "$ROOT" && go build -o "$tmp_bin_dir/specrelay" . && go build -o "$tmp_bin_dir/wo" ./cmd/specrelay) 2>&1 | tee -a "$LOG" || fail "Root specrelay package or cmd/specrelay cannot build"

note "Verify the full Go test gate"
(cd "$ROOT" && go test ./... -count=1) 2>&1 | tee -a "$LOG" || fail "go test ./... failed"

note "contract passed: migrated app tests no longer hide inside root gate"
