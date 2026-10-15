#!/usr/bin/env bash
# Verify stage prompt contract completeness.
# Sources: 6-stage-prompt-contract, Proposal 18
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
RESULT_DIR="$ROOT/test-results/6-stage-artifact-gate/stage-prompt-contract"
TEST_FILE="$ROOT/internal/app/change_six_stage_prompt_contract_test.go"

mkdir -p "$RESULT_DIR"
trap 'rm -f "$TEST_FILE"' EXIT

cat > "$TEST_FILE" <<'GO'
// Package app receives an injected contract test for change 6 prompt completeness.
package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// changeSixRepairWorkflow preserves the finite repair-v1 contract covered by this historical test.
func changeSixRepairWorkflow() WorkflowConfig {
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

// sealChangeSixAcceptance writes a valid run-local acceptance snapshot for prompt rendering.
func sealChangeSixAcceptance(t *testing.T, repo string, state *State) {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", filepath.Join(repo, "state"))
	source := filepath.Join(repo, "acceptance.json")
	body := `{
  "summary": "change six prompt contract",
  "coverage": [{
    "spec": "Requirement: Stage prompt contract / Scenario: Render sealed run prompts",
    "tests": ["prompt-contract"],
    "evidence": [],
    "risk": "Prompt structure only"
  }],
  "required_tests": [{
    "id": "prompt-contract",
    "source": "change_contract",
    "path": "tests/prompt-contract.sh",
    "command": "bash tests/prompt-contract.sh",
    "purpose": "Verify complete stage prompt contracts",
    "assertions": ["Prompts use sealed acceptance and preserve stage input/output boundaries"]
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

// renderChangeSixPrompt renders one bundled prompt with realistic state paths.
func renderChangeSixPrompt(t *testing.T, templateFile, templateName, stage string, sessions map[string]string) string {
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
		RunID:      "change-six-prompt-contract",
		ChangeName: "6-stage-prompt-contract",
		Sealed:     true,
		Stage:      stage,
		Workflow:   changeSixRepairWorkflow(),
		Sessions:   sessions,
		Stages:     stages,
	}
	repo := t.TempDir()
	sealChangeSixAcceptance(t, repo, &state)
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

// requireChangeSixPromptContains reports missing fragments with the rendered prompt.
func requireChangeSixPromptContains(t *testing.T, prompt string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

// requireChangeSixPromptOmits reports over-repeated fragments with the rendered prompt.
func requireChangeSixPromptOmits(t *testing.T, prompt string, rejects ...string) {
	t.Helper()
	for _, reject := range rejects {
		if strings.Contains(prompt, reject) {
			t.Fatalf("prompt unexpectedly contains %q:\n%s", reject, prompt)
		}
	}
}

// TestChangeSixDiscussPromptKeepsPlanningEntry verifies planning still enters specrelay-plan.
func TestChangeSixDiscussPromptKeepsPlanningEntry(t *testing.T) {
	prompt := renderChangeSixPrompt(t, "specrelay-flow-discuss.md", "specrelay-flow-discuss", "planning", nil)
	requireChangeSixPromptContains(t, prompt, "Planning task", "specrelay-plan")
}

// TestChangeSixExecutionPromptDelegatesToOzExec prevents start prompts from duplicating specrelay-exec.
func TestChangeSixExecutionPromptDelegatesToOzExec(t *testing.T) {
	prompt := renderChangeSixPrompt(t, "specrelay-flow-start.md", "specrelay-flow-start", "execution", nil)
	requireChangeSixPromptContains(t, prompt,
		"state.json.change_name",
		"specrelay-exec",
		"acceptance.json",
		"stay within its proposal scope",
	)
	requireChangeSixPromptOmits(t, prompt, "proposal.md", "design.md", "spec.md", "required_tests", "tasks.done", "review-1.json", "fix-1-summary.md", "Fix only findings listed in the current review/QA artifact")
}

// TestChangeSixRepairPromptKeepsSelfReviewContract verifies repair has inputs, output, schema, evidence, and session rules.
func TestChangeSixRepairPromptKeepsSelfReviewContract(t *testing.T) {
	first := renderChangeSixPrompt(t, "specrelay-flow-repair.md", "specrelay-flow-repair", "repair_1", nil)
	requireChangeSixPromptContains(t, first,
		"state.json",
		"acceptance.json",
		"complete diff",
		"repair-1.json",
		"Strict JSON",
		"decision",
		"scope",
		"non_blocking_findings",
		"needs_more",
		"repairer cannot archive",
	)

	resumed := renderChangeSixPrompt(t, "specrelay-flow-repair.md", "specrelay-flow-repair", "repair_2", map[string]string{"codex:repairer": "repair-session"})
	requireChangeSixPromptContains(t, resumed, "backend-scoped session", "codex:repairer", "repair-2.json", "repair-1.json", "Strict JSON")
	requireChangeSixPromptOmits(t, resumed, "review-2.json", "fix-2-summary.md")
}

// TestChangeSixQAPromptKeepsFirstTurnAcceptanceContract verifies QA keeps required tests, evidence, and matrix rules.
func TestChangeSixQAPromptKeepsFirstTurnAcceptanceContract(t *testing.T) {
	first := renderChangeSixPrompt(t, "specrelay-flow-qa.md", "specrelay-flow-qa", "qa_1", nil)
	requireChangeSixPromptContains(t, first,
		"state.json",
		"acceptance.json",
		"latest repair checkpoint",
		"required_tests",
		"required_evidence",
		"acceptance_matrix",
		"qa-1.json",
		"Do not modify source code",
		"decision",
		"scope",
	)

	resumed := renderChangeSixPrompt(t, "specrelay-flow-qa.md", "specrelay-flow-qa", "qa_2", map[string]string{"codex:qa": "qa-session"})
	requireChangeSixPromptContains(t, resumed, "reuse role session", "qa-2.json", "schema")
	requireChangeSixPromptOmits(t, resumed, "clean example:", "needs_fix example:")
}

// TestChangeSixArchivePromptDelegatesToOzArchive verifies archive prompt stays as a skill entry point.
func TestChangeSixArchivePromptKeepsDeliveryContract(t *testing.T) {
	prompt := renderChangeSixPrompt(t, "specrelay-flow-done.md", "specrelay-flow-done", "archive", nil)
	requireChangeSixPromptContains(t, prompt,
		"state.json.change_name",
		"specrelay-archive",
		"delivery-summary.md",
		"Final review",
		"historical audit/repair/QA artifacts",
	)
}
GO

(
  cd "$ROOT"
  go test ./internal/app -run 'TestChangeSix.*Prompt' -count=1
) | tee "$RESULT_DIR/contract.log"

(
  cd "$ROOT"
  go test ./internal/app -run 'TestParallelEnabledPromptsCarryFanoutArtifacts|TestBundledOzSkillPromptsDelegateToSkills' -count=1
) | tee -a "$RESULT_DIR/contract.log"
