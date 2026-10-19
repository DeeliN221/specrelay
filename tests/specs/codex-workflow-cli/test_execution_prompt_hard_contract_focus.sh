#!/usr/bin/env bash
# Verify execution prompt hard contract focus.
# Sources: 8-execution-contract, Proposal 47
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
result_dir="$repo_root/test-results/8-acceptance-contract"
mkdir -p "$result_dir"
log="$result_dir/execution-hard-contract-prompt.log"
test_file="$repo_root/internal/app/change_eight_execution_prompt_contract_test.go"
trap 'rm -f "$test_file"' EXIT

cat >"$test_file" <<'GO'
// Package app receives an injected contract test for change 8 execution prompt focus.
package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sealChangeEightAcceptance writes the immutable contract required by quality-loop prompts.
func sealChangeEightAcceptance(t *testing.T, repo string, state *State) {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", filepath.Join(repo, "state"))
	source := filepath.Join(repo, "acceptance.json")
	body := `{
  "summary": "change eight execution prompt contract",
  "coverage": [{
    "spec": "Requirement: Execution prompt contract / Scenario: Render sealed run prompts",
    "tests": ["prompt-contract"],
    "evidence": [],
    "risk": "Prompt structure only"
  }],
  "required_tests": [{
    "id": "prompt-contract",
    "source": "change_contract",
    "path": "tests/prompt-contract.sh",
    "command": "bash tests/prompt-contract.sh",
    "purpose": "Verify execution focuses on the hard contract",
    "assertions": ["Execution uses sealed acceptance without repeating long document strategy"]
  }],
  "required_evidence": []
}`
	if err := os.WriteFile(source, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := snapshotQualityLoopAcceptance(repo, state, source); err != nil {
		t.Fatal(err)
	}
}

// renderChangeEightPrompt renders a bundled prompt with realistic state fields.
func renderChangeEightPrompt(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "prompts-template", "specrelay-flow-start.md"))
	if err != nil {
		t.Fatal(err)
	}
	state := State{
		RunID:      "change-eight-hard-contract",
		ChangeName: "8-execution-contract",
		Sealed:     true,
		Stage:      "execution",
		Workflow:   DefaultWorkflowConfig(),
	}
	repo := t.TempDir()
	sealChangeEightAcceptance(t, repo, &state)
	context, err := promptContext(repo, state)
	if err != nil {
		t.Fatal(err)
	}
	got, err := renderPromptTemplate("specrelay-flow-start", string(data), context)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// requireContains reports a missing prompt fragment with the full rendered prompt.
func requireContains(t *testing.T, prompt string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

// requireOmits reports an obsolete prompt fragment with the full rendered prompt.
func requireOmits(t *testing.T, prompt string, rejects ...string) {
	t.Helper()
	for _, reject := range rejects {
		if strings.Contains(prompt, reject) {
			t.Fatalf("prompt unexpectedly contains %q:\n%s", reject, prompt)
		}
	}
}

// TestChangeEightExecutionPromptDelegatesToSpecrelayExec verifies the execution prompt stays as an entry point.
func TestChangeEightExecutionPromptDelegatesToSpecrelayExec(t *testing.T) {
	prompt := renderChangeEightPrompt(t)
	requireContains(t, prompt,
		"specrelay-exec",
		"state.json.change_name",
		"acceptance.json",
		"stay within its proposal scope",
	)
	requireOmits(t, prompt,
		"proposal.md",
		"design.md",
		"spec.md",
		"required_tests",
		"tasks.done",
	)
}

// TestChangeEightSpecrelayExecSkillMatchesPromptContract verifies the invoked skill does not reintroduce full-document defaults.
func TestChangeEightSpecrelayExecSkillMatchesPromptContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "skills", "specrelay-exec", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	skill := string(data)
	requireContains(t, skill,
		"brief.md",
		"acceptance.json",
		"tests/",
		"consult `design.md` and `spec.md`",
	)
	requireOmits(t, skill,
		"First read:\n\n- `proposal.md`",
		"First read `proposal.md`, `design.md`, `spec.md`, `task.md`, `acceptance.json`",
	)
}

// TestChangeEightSpecrelayCreateSkillCreatesBrief verifies new proposals include the active-change brief required by validate.
func TestChangeEightSpecrelayCreateSkillCreatesBrief(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "skills", "specrelay-create", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	skill := string(data)
	requireContains(t, skill,
		"brief.md",
		"default execution context",
		"specrelay validate <change> --json",
	)
	requireOmits(t, skill,
		"and proposal.md, design.md, spec.md, task.md, acceptance.json, tests/",
	)
}
GO

(
  cd "$repo_root"
  go test ./internal/app -run 'TestChangeEight(ExecutionPromptDelegatesToSpecrelayExec|SpecrelayExecSkillMatchesPromptContract|SpecrelayCreateSkillCreatesBrief)' -count=1
) | tee "$log"
