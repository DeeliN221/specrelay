#!/usr/bin/env bash
# Verify agent backend allowlist contract.
# Sources: Proposal 14, Proposal 15, claude-code-backend
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
LOG="$ROOT/test-results/spec-agent-backend-allowlist.log"
mkdir -p "$(dirname "$LOG")"
: >"$LOG"

# note: Record each verification step in stdout and the runtime evidence log.
note() {
  printf '[agent-backend-allowlist] %s\n' "$*" | tee -a "$LOG"
}

# fail: Report the failed business contract and stop with a nonzero status.
fail() {
  printf '[agent-backend-allowlist] FAIL: %s\n' "$*" | tee -a "$LOG" >&2
  exit 1
}

legacy_lower="open""code"
legacy_title="Open""Code"

note "Scan source, maintained specifications and tests for removed backend names"
if hits="$(cd "$ROOT" && git grep -n -I -e "$legacy_lower" -e "$legacy_title" -- \
  internal cmd docs/specs tests \
  ':(exclude)tests/specs/codex-workflow-cli/test_agent_backend_allowlist_contract.sh' \
  ':(exclude)tests/specs/codex-workflow-cli/test_user_visible_surface_cleanup_contract.sh' 2>/dev/null || true)" && [[ -n "$hits" ]]; then
  printf '%s\n' "$hits" | tee -a "$LOG"
  fail "Current source, specifications or tests still reference a removed backend"
fi

note "Verify the removed backend implementation is absent"
if [[ -e "$ROOT/internal/app/${legacy_lower}.go" || -e "$ROOT/internal/app/${legacy_lower}_test.go" || -e "$ROOT/internal/app/${legacy_lower}_integration_test.go" ]]; then
  fail "internal/app still contains removed backend source or tests"
fi

note "Verify the registry accepts only codex/pi/agy/claude"
agent_file="$ROOT/internal/app/agent.go"
[[ -f "$agent_file" ]] || fail "Missing internal/app/agent.go"
grep -q 'return name == "codex" || name == "pi" || name == "agy" || name == "claude"' "$agent_file" || fail "validAgentTool must explicitly allow codex/pi/agy/claude"
for tool in CodexTool PiTool AgyTool ClaudeTool; do
  grep -q "$tool{}" "$agent_file" || fail "NewAgentRegistry is missing $tool"
done
go test "$ROOT/internal/app" -run 'TestAgentRegistryResolvesConfiguredTools|TestValidAgentTool' -count=1 >>"$LOG" 2>&1

note "contract passed: agent backend allowlist is codex/pi/agy/claude only"
