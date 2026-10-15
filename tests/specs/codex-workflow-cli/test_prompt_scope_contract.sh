#!/usr/bin/env bash
# Sources: Proposal 12
# Verify prompt scope contract.
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
result_dir="$repo_root/test-results/12-scope-gate"
log="$result_dir/prompt-scope-contract.log"

mkdir -p "$result_dir"
: >"$log"

note() {
  # note: Record each verification step in stdout and the runtime evidence log.
  printf '%s\n' "$*" | tee -a "$log"
}

require_text() {
  # require_text: Verify the required contract text is present.
  local path="$1"
  local text="$2"
  if ! grep -qF "$text" "$path"; then
    printf 'missing %q in %s\n' "$text" "$path" | tee -a "$log" >&2
    return 1
  fi
}

cd "$repo_root"

note "Verify review scope and non-blocking historical-debt contract"
require_text prompts-template/specrelay-flow-review.md "non_blocking_findings"
require_text prompts-template/specrelay-flow-review.md "out_of_scope_existing"
require_text prompts-template/specrelay-flow-review.md "current_change"
require_text prompts-template/specrelay-flow-review.md "acceptance_contract"
require_text prompts-template/specrelay-flow-review.md "introduced_regression"

note "Verify QA scope and acceptance_matrix boundaries"
require_text prompts-template/specrelay-flow-qa.md "non_blocking_findings"
require_text prompts-template/specrelay-flow-qa.md "out_of_scope_existing"
require_text prompts-template/specrelay-flow-qa.md "current_change"
require_text prompts-template/specrelay-flow-qa.md "acceptance_contract"
require_text prompts-template/specrelay-flow-qa.md "introduced_regression"
require_text prompts-template/specrelay-flow-qa.md "acceptance_matrix"

note "Verify the specification documents gate scope"
require_text docs/specs/codex-workflow-cli/spec.md "out_of_scope_existing"
require_text docs/specs/codex-workflow-cli/spec.md "non_blocking_findings"

note "PASS"
