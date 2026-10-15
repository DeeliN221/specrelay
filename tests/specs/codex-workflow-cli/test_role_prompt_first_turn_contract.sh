#!/usr/bin/env bash
# Verify role prompt first turn contract.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
RESULT_DIR="$ROOT/test-results/30-role-prompt-contract"
TEST_FILE="$ROOT/internal/app/prompt_first_turn_contract_test.go"

mkdir -p "$RESULT_DIR"
trap 'rm -f "$TEST_FILE"' EXIT

cat > "$TEST_FILE" <<'GO'
// Package app receives an injected contract test for role prompt first-turn behavior.
package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// roleContractRepairWorkflow preserves the finite repair-v1 role contract under test.
func roleContractRepairWorkflow() WorkflowConfig {
	base := DefaultWorkflowConfig()
	repair := base.Stages["audit_1"]
	qa := base.Stages["qa_1"]
	return WorkflowConfig{
		Engine:              base.Engine,
		Generation:          repairWorkflowGeneration,
		MaxRepairIterations: 2,
		Stages: map[string]StageOptions{
			"planning":  base.Stages["planning"],
			"execution": base.Stages["execution"],
			"repair_1":  repair,
			"qa_1":      qa,
			"repair_2":  repair,
			"qa_2":      qa,
			"archive":   base.Stages["archive"],
		},
		Validation: base.Validation,
		Prompts:    base.Prompts,
	}
}

// sealRoleContractAcceptance writes a valid run-local acceptance snapshot for role prompts.
func sealRoleContractAcceptance(t *testing.T, repo string, state *State) {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", filepath.Join(repo, "state"))
	source := filepath.Join(repo, "acceptance.json")
	body := `{
  "summary": "role prompt contract",
  "coverage": [{
    "spec": "Requirement: Role prompt contract / Scenario: Render sealed run prompts",
    "tests": ["prompt-contract"],
    "evidence": [],
    "risk": "Prompt structure only"
  }],
  "required_tests": [{
    "id": "prompt-contract",
    "source": "change_contract",
    "path": "tests/prompt-contract.sh",
    "command": "bash tests/prompt-contract.sh",
    "purpose": "Verify first and subsequent role prompt contracts",
    "assertions": ["Role prompts use sealed acceptance and preserve session/artifact boundaries"]
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

// renderBundledPromptForRoleContract renders one bundled prompt with realistic run state.
func renderBundledPromptForRoleContract(t *testing.T, templateFile, templateName, stage string, sessions map[string]string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "prompts-template", templateFile))
	if err != nil {
		t.Fatal(err)
	}
	stages := map[string]string{}
	if stage == "archive" {
		stages = map[string]string{"repair_1": "completed", "qa_1": "completed"}
	}
	state := State{
		RunID:      "role-contract-run",
		ChangeName: "demo",
		Sealed:     true,
		Stage:      stage,
		Workflow:   roleContractRepairWorkflow(),
		Sessions:   sessions,
		Stages:     stages,
	}
	repo := t.TempDir()
	sealRoleContractAcceptance(t, repo, &state)
	context, err := promptContext(repo, state)
	if err != nil {
		t.Fatal(err)
	}
	got, err := renderPromptTemplate(templateName, string(data), context)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// requirePromptContains explains missing prompt fragments with the rendered prompt body.
func requirePromptContains(t *testing.T, prompt string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

// requirePromptOmits explains repeated prompt fragments with the rendered prompt body.
func requirePromptOmits(t *testing.T, prompt string, rejects ...string) {
	t.Helper()
	for _, reject := range rejects {
		if strings.Contains(prompt, reject) {
			t.Fatalf("prompt repeated %q:\n%s", reject, prompt)
		}
	}
}

// TestBundledRepairPromptKeepsOneSessionContract verifies every repair round keeps the durable schema and one session.
func TestBundledRepairPromptKeepsOneSessionContract(t *testing.T) {
	first := renderBundledPromptForRoleContract(t, "specrelay-flow-repair.md", "specrelay-flow-repair", "repair_1", nil)
	requirePromptContains(t, first, "Strict JSON", "decision", "scope", "non_blocking_findings", "repair-1.json", "needs_more", "repairer cannot archive")

	resumed := renderBundledPromptForRoleContract(t, "specrelay-flow-repair.md", "specrelay-flow-repair", "repair_2", map[string]string{"codex:repairer": "repair-session"})
	requirePromptContains(t, resumed, "repair-2.json", "repair-1.json", "codex:repairer", "backend-scoped session")
	requirePromptOmits(t, resumed, "review-2.json", "fix-2-summary.md")
}

// TestBundledQAPromptKeepsArtifactRules verifies QA prompt keeps compact JSON rules.
func TestBundledQAPromptKeepsExamplesOnlyForFirstQATurn(t *testing.T) {
	first := renderBundledPromptForRoleContract(t, "specrelay-flow-qa.md", "specrelay-flow-qa", "qa_1", nil)
	requirePromptContains(t, first, "decision", "scope", "acceptance_matrix", "latest repair checkpoint", "Do not modify source code")

	resumed := renderBundledPromptForRoleContract(t, "specrelay-flow-qa.md", "specrelay-flow-qa", "qa_2", map[string]string{"codex:qa": "qa-session"})
	requirePromptContains(t, resumed, "qa-2.json", "schema")
	requirePromptOmits(t, resumed, "clean example:", "needs_fix example:", "\"summary\": \"Core business path passed QA\"", "\"decision\": \"needs_fix\"")
}

// TestBundledExecutionPromptDelegatesToOzExec verifies execution prompt stays as a skill entry point.
func TestBundledExecutionPromptDelegatesToOzExec(t *testing.T) {
	prompt := renderBundledPromptForRoleContract(t, "specrelay-flow-start.md", "specrelay-flow-start", "execution", nil)
	requirePromptContains(t, prompt,
		"specrelay-exec",
		"state.json.change_name",
		"acceptance.json",
		"stay within its proposal scope",
	)
	requirePromptOmits(t, prompt, "proposal.md", "design.md", "spec.md", "required_tests", "tasks.done", "review-1.json", "fix-1-summary.md", "Fix only findings listed in the current review/QA artifact")
}

// TestBundledDonePromptRequiresAuditableDeliverySummary verifies the final summary is useful to human reviewers.
func TestBundledDonePromptRequiresAuditableDeliverySummary(t *testing.T) {
	prompt := renderBundledPromptForRoleContract(t, "specrelay-flow-done.md", "specrelay-flow-done", "archive", map[string]string{
		"codex:repairer": "repair-session",
		"codex:qa":       "qa-session",
	})
	requirePromptContains(t, prompt,
		"delivery-summary.md",
		"Final review",
		"specrelay-archive",
		"historical audit/repair/QA artifacts",
	)
}
GO

(
  cd "$ROOT"
  go test ./internal/app -run 'TestBundled(RepairPromptKeepsOneSessionContract|QAPromptKeepsExamplesOnlyForFirstQATurn|ExecutionPromptDelegatesToOzExec|DonePromptRequiresAuditableDeliverySummary)' -count=1
) | tee "$RESULT_DIR/contract.log"
