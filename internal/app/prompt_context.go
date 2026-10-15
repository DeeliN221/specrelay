// Package app builds sealed-run prompt templates and stable prompt context paths.
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// promptSnapshot stores the effective prompt bodies frozen for one sealed run.
type promptSnapshot struct {
	Prompts map[string]string `yaml:"prompts"`
}

// promptTemplateContext is the data exposed to named prompt templates.
type promptTemplateContext struct {
	RunID                        string
	RunDirectory                 string
	ChangeName                   string
	Stage                        string
	StageKind                    string
	Iteration                    int
	MaxReviewIterations          int
	MaxRepairIterations          int
	StatePath                    string
	ChangePath                   string
	AcceptancePath               string
	AcceptanceSummaryPath        string
	ReviewPath                   string
	RepairPath                   string
	HasRepairCheckpoint          bool
	QAPath                       string
	FixSummaryPath               string
	PreviousReviewPaths          []string
	PreviousRepairPaths          []string
	PreviousQAPaths              []string
	PreviousFixSummaryPaths      []string
	PreviousReviewCount          int
	PreviousRepairCount          int
	PreviousFixSummaryCount      int
	LatestPreviousReviewPath     string
	LatestPreviousRepairPath     string
	LatestPreviousQAPath         string
	LatestPreviousRepairFile     string
	LatestPreviousQAFile         string
	LatestPreviousFixSummaryPath string
	HasPreviousReview            bool
	HasPreviousRepair            bool
	HasPreviousQA                bool
	HasPreviousFixSummary        bool
	PlanningContextPath          string
	ParallelContextPath          string
	ParallelReviewPath           string
	ParallelQAPath               string
	HasPlanningContext           bool
	HasParallelContext           bool
	HasParallelReview            bool
	HasParallelQA                bool
	DeliverySummaryPath          string
	BaselineHead                 string
	RoleSessionKey               string
	RoleSessionID                string
	HasRoleSession               bool
	IsFirstRoleTurn              bool
	IsRepairConfirmation         bool
	FixEscalated                 bool
	FixEscalationReasoning       string
	ConsecutiveReviewFailures    int
	RepeatedFindingTitles        []string
	RepairMode                   string
	SourceQAArtifact             string
	QAFindingSummaries           []string
	FailedAcceptanceIDs          []string
}

// promptForStage reads and renders the YAML prompt for a sealed stage.
func promptForStage(repo string, state State) (string, error) {
	if state.ChangeName != "" {
		if err := validateChangeNameForPath(state.ChangeName); err != nil {
			return "", err
		}
	}
	name, err := promptNameForStage(state.Stage)
	if err != nil {
		return "", err
	}
	var templateText string
	if state.RunID != "" {
		templateText, err = runPromptTemplate(repo, state.RunID, name)
		if err != nil {
			return "", err
		}
	} else {
		config := DefaultWorkflowConfig()
		if state.Workflow.Prompts != nil {
			config = state.Workflow
		} else if loaded, loadErr := LoadWorkflowConfig(repo); loadErr == nil {
			config = loaded
		}
		templateText, err = promptForName(config, name)
		if err != nil {
			return "", err
		}
	}
	context, err := promptContext(repo, state)
	if err != nil {
		return "", err
	}
	prompt, err := renderPromptTemplate(name, templateText, context)
	if err != nil {
		return "", err
	}
	if failurePrompt := validationFailurePrompt(repo, state); failurePrompt != "" {
		prompt = failurePrompt
	}
	if usesQualityLoop(state.Workflow) {
		prompt = appendQualityLoopPrompt(prompt, context)
	}
	return prompt, nil
}

// appendQualityLoopPrompt adds the mode-specific scope and deterministic exit contract.
func appendQualityLoopPrompt(prompt string, context promptTemplateContext) string {
	var block strings.Builder
	block.WriteString("\n\n# Dynamic quality-loop contract\n\n")
	fmt.Fprintf(&block, "- Current stage: `%s` (round `%d`); run directory: `%s`; diff baseline: `%s`.\n", context.Stage, context.Iteration, context.RunDirectory, context.BaselineHead)
	stageKind := context.Stage
	if parsed, err := parseWorkflowStage(context.Stage); err == nil {
		stageKind = parsed.Kind
	}
	switch stageKind {
	case workflowStagePlanning:
		block.WriteString("- Planning must define `delivery_report`: describe benefits, verification steps, visible outcomes, and direct evidence in user language. Fix scenarios must bind distinct, understandable before/after evidence for the same scenario.\n")
	case workflowStageExecution:
		block.WriteString("- Before execution finishes, produce real, playable demonstration media. Do not substitute echo/printf, hard-coded strings, exit codes, or passing-test messages for user evidence.\n")
	case workflowStageQA:
		block.WriteString("- QA must fill every `user_acceptance[]` entry with observed behavior in plain user language and references to delivery-scenario evidence. Commands, exit codes, hashes, HTTP 200, and element existence do not establish user outcomes.\n")
	}
	switch context.RepairMode {
	case "pre_qa_audit":
		block.WriteString("- If an environment prerequisite is missing, write `blocked_environment: VARIABLE_OR_PATH` in artifact evidence. Record only the variable name or path, never a secret value.\n")
		block.WriteString("- Do not create Git commits during execution, self-review, or targeted repair. Only the archive stage may create the complete delivery commit.\n")
		block.WriteString("- Before independent QA, run pre-commit hooks through the repository's existing explicit entry point without creating a temporary commit. Apply their changes and rerun until files remain unchanged. Then rerun affected tests, all required tests, and validation commands.\n")
		fmt.Fprintf(&block, "- Mode: `pre_qa_audit`; write `audit-%d.json` relative to the run directory.\n- In the run directory, run `specrelay flow validate-repair --artifact \"audit-%d.json\" --json`.\n- Review the proposal's acceptance contract, complete diff, source, tests, and evidence. Run the demonstration and verify that it covers the requested capability. Check the previous immutable evidence snapshot when available. Produce only temporary sources under `test-results/**` for this stage's post-gate to seal; do not write to the `tests/evidence/proposals/<change>/**` evidence bundle. Hand off to independent QA only when this round finds no new issues and required tests pass.\n", context.Iteration, context.Iteration)
	case "qa_targeted_repair":
		block.WriteString("- If an environment prerequisite is missing, write `blocked_environment: VARIABLE_OR_PATH` in artifact evidence. Record only the variable name or path, never a secret value.\n")
		block.WriteString("- Do not create Git commits during execution, self-review, or targeted repair. Only the archive stage may create the complete delivery commit.\n")
		block.WriteString("- Before independent QA, run pre-commit hooks through the repository's existing explicit entry point without creating a temporary commit. Apply their changes and rerun until files remain unchanged. Then rerun affected tests, all required tests, and validation commands.\n")
		fmt.Fprintf(&block, "- Mode: `qa_targeted_repair`; write `targeted-repair-%d.json` relative to the run directory.\n- In the run directory, run `specrelay flow validate-repair --artifact \"targeted-repair-%d.json\" --json`.\n- Address only the latest QA findings and directly related regressions; source QA: `qa-%d.json`. Produce distinct, understandable before/after evidence for each user scenario. Command output, exit codes, hashes, and hard-coded strings are not user evidence.\n", context.Iteration, context.Iteration, context.Iteration)
		if len(context.QAFindingSummaries) > 0 {
			block.WriteString("- Latest QA findings:\n")
			for _, finding := range context.QAFindingSummaries {
				fmt.Fprintf(&block, "  - %s\n", finding)
			}
		}
		if len(context.FailedAcceptanceIDs) > 0 {
			fmt.Fprintf(&block, "- Failed acceptance IDs: `%s`\n", strings.Join(context.FailedAcceptanceIDs, "`, `"))
		}
		block.WriteString("- Before handoff, rerun failed tests, all required tests, and validation commands. Failed or stale results must remain in this stage.\n")
	}
	if context.Stage == workflowStageArchive {
		block.WriteString("- After final QA, archiving is a read-only boundary: use only the archive skill for mechanical moves and delivery-summary.md. Lasting specifications and specification tests must be complete before final QA.\n")
		block.WriteString("- The engine generates `tests/evidence/proposals/<change>/DELIVERY.md`. Verify that user benefits, verification steps, observations, and direct evidence are understandable and accessible, then include the entire bundle in the archive commit.\n")
		block.WriteString("- Keep the final bundle under `tests/evidence/proposals/<change>/**`, outside Git ignore rules. Do not rebuild it from `test-results/**` or edit it after archiving; stage the engine's output unchanged.\n")
		block.WriteString("- Create exactly one complete delivery commit from the sealed `delivery_base_head`, containing implementation, the archived proposal, and final evidence at HEAD. Do not amend, squash, or reuse execution/self-review commits.\n")
		block.WriteString("- Do not first run file-rewriting pre-commit hooks after final QA. Recheck only the same entry point already proven idempotent during self-review. If files change, stop archiving and return to self-review and QA; do not treat those changes as tested.\n")
		block.WriteString("- After the archive command returns, do not edit, format, or restore proposal or source files. Stage and commit the command's output unchanged; do not rewrite content merely to clean the worktree. Let the read-only gate assess differences.\n")
	}
	return prompt + block.String()
}

// promptForName reads a named prompt from the effective YAML config.
func promptForName(config WorkflowConfig, name string) (string, error) {
	key, err := promptKeyForName(name)
	if err != nil {
		return "", err
	}
	body := config.Prompts[key]
	if body == "" {
		return "", fmt.Errorf("configuration is missing prompts.%s", key)
	}
	return body, nil
}

// promptKeyForName resolves a prompt name to its workflow YAML key.
func promptKeyForName(name string) (string, error) {
	role, ok := roleByPromptName(name)
	if !ok {
		return "", fmt.Errorf("unknown prompt %q", name)
	}
	return role.PromptKey, nil
}

// runPromptTemplate reads the prompt snapshot saved when a sealed run starts.
func runPromptTemplate(repo, runID, name string) (string, error) {
	key, keyErr := promptKeyForName(name)
	if keyErr != nil {
		return "", keyErr
	}
	snapshotPath := filepath.Join(runDir(repo, runID), "prompt-snapshot.yaml")
	data, err := os.ReadFile(snapshotPath)
	if err == nil {
		var snapshot promptSnapshot
		if err := yaml.Unmarshal(data, &snapshot); err != nil {
			return "", fmt.Errorf("read prompt snapshot %s failed: %w", snapshotPath, err)
		}
		body := snapshot.Prompts[key]
		if body == "" {
			return "", fmt.Errorf("prompt snapshot is missing prompts.%s", key)
		}
		return body, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	return "", fmt.Errorf("run %s is missing prompt snapshot prompt-snapshot.yaml", runID)
}

// snapshotRunPrompts freezes sealed-run prompts so resume cannot drift.
func snapshotRunPrompts(repo, runID string) error {
	root := runDir(repo, runID)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	config, err := LoadWorkflowConfig(repo)
	if err != nil {
		return err
	}
	snapshot := promptSnapshot{Prompts: map[string]string{}}
	for _, key := range promptKeysForWorkflow(config) {
		body := config.Prompts[key]
		if body == "" {
			return fmt.Errorf("configuration is missing prompts.%s", key)
		}
		snapshot.Prompts[key] = body
	}
	data, err := yaml.Marshal(snapshot)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "prompt-snapshot.yaml"), data, 0o644)
}

// promptKeysForWorkflow returns only prompts used by the active state-machine generation.
func promptKeysForWorkflow(config WorkflowConfig) []string {
	if usesQualityLoop(config) || usesRepairWorkflow(config) {
		return []string{"planning", "execution", "repair", "qa", "archive"}
	}
	return []string{"planning", "execution", "review", "qa", "fix", "archive"}
}

// renderPromptTemplate injects run metadata and fails on unknown template variables.
func renderPromptTemplate(name, body string, context promptTemplateContext) (string, error) {
	tmpl, err := template.New(name).Option("missingkey=error").Parse(body)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	if err := tmpl.Execute(&out, context); err != nil {
		return "", err
	}
	return out.String(), nil
}

// promptContext computes stable run paths for the current workflow stage.
func promptContext(repo string, state State) (promptTemplateContext, error) {
	ensureWorkflowConfig(&state)
	kind := stageKind(state.Stage)
	iteration, err := stageIteration(state.Stage)
	if err != nil {
		return promptTemplateContext{}, err
	}
	runPath := runDir(repo, state.RunID)
	sealedAcceptancePath := acceptancePath(repo, state.ChangeName)
	if candidate := filepath.Join(runPath, "acceptance.json"); state.RunID != "" {
		if usesQualityLoop(state.Workflow) {
			if _, err := readAcceptanceForState(repo, state); err != nil {
				return promptTemplateContext{}, fmt.Errorf("quality loop is missing valid sealed acceptance %s: %w", candidate, err)
			}
			sealedAcceptancePath = candidate
		} else if fileExists(candidate) {
			sealedAcceptancePath = candidate
		}
	}
	roleSessionKey, roleSessionID := promptRoleSession(state)
	context := promptTemplateContext{
		RunID:                   state.RunID,
		RunDirectory:            runPath,
		ChangeName:              state.ChangeName,
		Stage:                   state.Stage,
		StageKind:               kind,
		Iteration:               iteration,
		MaxReviewIterations:     state.Workflow.MaxReviewIterations,
		MaxRepairIterations:     state.Workflow.MaxRepairIterations,
		StatePath:               filepath.Join(runPath, "state.json"),
		ChangePath:              "docs/changes/" + state.ChangeName,
		AcceptancePath:          sealedAcceptancePath,
		AcceptanceSummaryPath:   sealedAcceptancePath,
		DeliverySummaryPath:     filepath.Join(runPath, "delivery-summary.md"),
		BaselineHead:            state.BaselineHead,
		RoleSessionKey:          roleSessionKey,
		RoleSessionID:           roleSessionID,
		HasRoleSession:          roleSessionID != "",
		IsFirstRoleTurn:         roleSessionID == "",
		IsRepairConfirmation:    kind == workflowStageRepair && state.RepairConfirmationPending,
		PreviousReviewPaths:     []string{},
		PreviousRepairPaths:     []string{},
		PreviousQAPaths:         []string{},
		PreviousFixSummaryPaths: []string{},
		RepeatedFindingTitles:   []string{},
	}
	if usesQualityLoop(state.Workflow) {
		if kind == workflowStageAudit {
			context.RepairMode = "pre_qa_audit"
			context.RepairPath = filepath.Join(runPath, fmt.Sprintf("audit-%d.json", iteration))
			context.HasRepairCheckpoint = true
			if err := appendPreviousQualityArtifacts(&context, state, runPath, "audit_", iteration); err != nil {
				return promptTemplateContext{}, err
			}
		}
		if kind == workflowStageTargetedRepair {
			context.RepairMode = "qa_targeted_repair"
			context.RepairPath = filepath.Join(runPath, fmt.Sprintf("targeted-repair-%d.json", iteration))
			context.HasRepairCheckpoint = true
			context.SourceQAArtifact = filepath.Join(runPath, fmt.Sprintf("qa-%d.json", iteration))
			qa, readErr := ReadQA(context.SourceQAArtifact)
			if readErr != nil {
				return promptTemplateContext{}, fmt.Errorf("targeted repair is missing a valid source QA artifact %s: %w", context.SourceQAArtifact, readErr)
			}
			if !QANeedsFix(qa) {
				return promptTemplateContext{}, fmt.Errorf("targeted repair source QA artifact %s does not declare needs_fix", context.SourceQAArtifact)
			}
			acceptance, acceptanceErr := readAcceptanceForState(repo, state)
			if acceptanceErr != nil {
				return promptTemplateContext{}, acceptanceErr
			}
			if err := ValidateQAAgainstAcceptance(qa, acceptance); err != nil {
				return promptTemplateContext{}, fmt.Errorf("targeted repair source QA artifact %s does not cover sealed acceptance: %w", context.SourceQAArtifact, err)
			}
			expectedSource := fmt.Sprintf("qa-%d.json", iteration)
			if state.QualityLoop.SourceQAArtifact != expectedSource ||
				state.QualityLoop.SourceQAHash == "" ||
				qaArtifactContentHash(qa) != state.QualityLoop.SourceQAHash {
				return promptTemplateContext{}, fmt.Errorf("targeted repair source QA artifact %s sealed full content mismatch", context.SourceQAArtifact)
			}
			if state.QualityLoop.FindingFingerprint != "" &&
				qaFindingFingerprint(qa) != state.QualityLoop.FindingFingerprint {
				return promptTemplateContext{}, fmt.Errorf("targeted repair source QA artifact %s sealed finding fingerprint mismatch", context.SourceQAArtifact)
			}
			sourceStage := fmt.Sprintf("qa_%d", iteration)
			if !qualityLoopTrustedSourceQA(repo, state, sourceStage, qa) {
				return promptTemplateContext{}, fmt.Errorf(
					"targeted repair source QA artifact %s failed tested-input trust validation",
					context.SourceQAArtifact,
				)
			}
			context.PreviousQAPaths = append(context.PreviousQAPaths, context.SourceQAArtifact)
			context.HasPreviousQA = true
			context.LatestPreviousQAPath = context.SourceQAArtifact
			if err := appendPreviousQualityArtifacts(&context, state, runPath, "targeted_repair_", iteration); err != nil {
				return promptTemplateContext{}, err
			}
			context.QAFindingSummaries = qaFindingSummaries(qa)
			context.FailedAcceptanceIDs = failedAcceptanceIDs(qa)
		}
		if kind == workflowStageQA {
			context.QAPath = filepath.Join(runPath, fmt.Sprintf("qa-%d.json", iteration))
			checkpoint, checkpointErr := latestQualityLoopRepairCheckpoint(state, runPath, iteration)
			if checkpointErr != nil {
				return promptTemplateContext{}, checkpointErr
			}
			if checkpoint != "" {
				context.RepairPath = checkpoint
				context.HasRepairCheckpoint = true
			}
		}
	}
	escalation, err := fixEscalation(repo, state)
	if err != nil {
		return promptTemplateContext{}, err
	}
	if escalation.Enabled {
		context.FixEscalated = escalation.Enabled
		context.FixEscalationReasoning = escalation.Reasoning
		context.ConsecutiveReviewFailures = escalation.ConsecutiveFailures
		context.RepeatedFindingTitles = escalation.RepeatedFindingTitles
	}
	if iteration > 0 && !usesQualityLoop(state.Workflow) {
		if usesRepairWorkflow(state.Workflow) && state.Workflow.MaxRepairIterations > 0 {
			context.RepairPath = filepath.Join(runPath, fmt.Sprintf("repair-%d.json", iteration))
			context.HasRepairCheckpoint = true
		}
		context.ReviewPath = filepath.Join(runPath, fmt.Sprintf("review-%d.json", iteration))
		context.QAPath = filepath.Join(runPath, fmt.Sprintf("qa-%d.json", iteration))
		context.FixSummaryPath = filepath.Join(runPath, fmt.Sprintf("fix-%d-summary.md", iteration))
		for i := 1; i < iteration; i++ {
			context.PreviousRepairPaths = append(context.PreviousRepairPaths, filepath.Join(runPath, fmt.Sprintf("repair-%d.json", i)))
			context.PreviousReviewPaths = append(context.PreviousReviewPaths, filepath.Join(runPath, fmt.Sprintf("review-%d.json", i)))
			context.PreviousFixSummaryPaths = append(context.PreviousFixSummaryPaths, filepath.Join(runPath, fmt.Sprintf("fix-%d-summary.md", i)))
			if state.Stages[fmt.Sprintf("qa_%d", i)] == "completed" {
				context.PreviousQAPaths = append(context.PreviousQAPaths, filepath.Join(runPath, fmt.Sprintf("qa-%d.json", i)))
			}
		}
		context.PreviousReviewCount = len(context.PreviousReviewPaths)
		context.PreviousRepairCount = len(context.PreviousRepairPaths)
		context.PreviousFixSummaryCount = len(context.PreviousFixSummaryPaths)
		context.HasPreviousReview = context.PreviousReviewCount > 0
		context.HasPreviousRepair = context.PreviousRepairCount > 0
		context.HasPreviousFixSummary = context.PreviousFixSummaryCount > 0
		if context.HasPreviousReview {
			context.LatestPreviousReviewPath = context.PreviousReviewPaths[context.PreviousReviewCount-1]
		}
		if context.HasPreviousRepair {
			context.LatestPreviousRepairPath = context.PreviousRepairPaths[context.PreviousRepairCount-1]
		}
		if len(context.PreviousQAPaths) > 0 {
			context.HasPreviousQA = true
			context.LatestPreviousQAPath = context.PreviousQAPaths[len(context.PreviousQAPaths)-1]
		}
		if context.HasPreviousFixSummary {
			context.LatestPreviousFixSummaryPath = context.PreviousFixSummaryPaths[context.PreviousFixSummaryCount-1]
		}
	}
	if kind == "archive" {
		if usesQualityLoop(state.Workflow) {
			appendQualityLoopArchivePaths(&context, state, runPath)
		} else {
			appendFiniteArchivePaths(&context, state, runPath)
		}
		finalizePreviousArtifactContext(&context)
	}
	if context.LatestPreviousRepairPath != "" {
		context.LatestPreviousRepairFile = filepath.Base(context.LatestPreviousRepairPath)
	}
	if context.LatestPreviousQAPath != "" {
		context.LatestPreviousQAFile = filepath.Base(context.LatestPreviousQAPath)
	}
	return context, nil
}

// appendPreviousQualityArtifacts adds only completed, valid dynamic repair artifacts.
func appendPreviousQualityArtifacts(context *promptTemplateContext, state State, runPath, prefix string, iteration int) error {
	for i := 1; i < iteration; i++ {
		stage := fmt.Sprintf("%s%d", prefix, i)
		if state.Stages[stage] != "completed" {
			continue
		}
		name := fmt.Sprintf("audit-%d.json", i)
		if prefix == "targeted_repair_" {
			name = fmt.Sprintf("targeted-repair-%d.json", i)
		}
		path := filepath.Join(runPath, name)
		if _, err := ReadRepair(path); err != nil {
			return fmt.Errorf("completed stage %s is missing a valid repair artifact %s: %w", stage, path, err)
		}
		context.PreviousRepairPaths = append(context.PreviousRepairPaths, path)
	}
	context.PreviousRepairCount = len(context.PreviousRepairPaths)
	context.HasPreviousRepair = context.PreviousRepairCount > 0
	if context.HasPreviousRepair {
		context.LatestPreviousRepairPath = context.PreviousRepairPaths[context.PreviousRepairCount-1]
	}
	return nil
}

// latestQualityLoopRepairCheckpoint selects the audit or targeted repair immediately preceding QA.
func latestQualityLoopRepairCheckpoint(state State, runPath string, qaIteration int) (string, error) {
	if qaIteration > 1 {
		stage := fmt.Sprintf("targeted_repair_%d", qaIteration-1)
		if state.Stages[stage] == "completed" {
			path := filepath.Join(runPath, fmt.Sprintf("targeted-repair-%d.json", qaIteration-1))
			if _, err := ReadRepair(path); err != nil {
				return "", fmt.Errorf("completed stage %s is missing a valid repair artifact %s: %w", stage, path, err)
			}
			return path, nil
		}
	}
	latestAudit := 0
	for stage, status := range state.Stages {
		if status != "completed" || !strings.HasPrefix(stage, "audit_") {
			continue
		}
		parsed, err := parseWorkflowStage(stage)
		if err == nil && parsed.Iteration > latestAudit {
			latestAudit = parsed.Iteration
		}
	}
	if latestAudit > 0 {
		stage := fmt.Sprintf("audit_%d", latestAudit)
		path := filepath.Join(runPath, fmt.Sprintf("audit-%d.json", latestAudit))
		if _, err := ReadRepair(path); err != nil {
			return "", fmt.Errorf("completed stage %s is missing a valid repair artifact %s: %w", stage, path, err)
		}
		return path, nil
	}
	return "", nil
}

// appendQualityLoopArchivePaths derives every dynamic audit, repair, and QA artifact from stage records.
func appendQualityLoopArchivePaths(context *promptTemplateContext, state State, runPath string) {
	for _, stage := range observedStatusStages(state) {
		if state.Stages[stage] == "" {
			continue
		}
		iteration, err := stageIteration(stage)
		if err != nil {
			continue
		}
		switch {
		case strings.HasPrefix(stage, "audit_"):
			context.PreviousRepairPaths = append(context.PreviousRepairPaths, filepath.Join(runPath, fmt.Sprintf("audit-%d.json", iteration)))
		case strings.HasPrefix(stage, "targeted_repair_"):
			context.PreviousRepairPaths = append(context.PreviousRepairPaths, filepath.Join(runPath, fmt.Sprintf("targeted-repair-%d.json", iteration)))
		case strings.HasPrefix(stage, "qa_"):
			context.PreviousQAPaths = append(context.PreviousQAPaths, filepath.Join(runPath, fmt.Sprintf("qa-%d.json", iteration)))
		}
	}
}

// appendFiniteArchivePaths preserves artifact collection for sealed legacy generations.
func appendFiniteArchivePaths(context *promptTemplateContext, state State, runPath string) {
	iterationLimit := state.Workflow.MaxRepairIterations
	if state.Workflow.MaxReviewIterations > iterationLimit {
		iterationLimit = state.Workflow.MaxReviewIterations
	}
	if usesRepairWorkflow(state.Workflow) && iterationLimit == 0 {
		iterationLimit = 1
	}
	for i := 1; i <= iterationLimit; i++ {
		if state.Stages[fmt.Sprintf("repair_%d", i)] != "" {
			context.PreviousRepairPaths = append(context.PreviousRepairPaths, filepath.Join(runPath, fmt.Sprintf("repair-%d.json", i)))
		}
		if state.Stages[fmt.Sprintf("qa_%d", i)] != "" {
			context.PreviousQAPaths = append(context.PreviousQAPaths, filepath.Join(runPath, fmt.Sprintf("qa-%d.json", i)))
		}
	}
	for i := 1; !usesRepairWorkflow(state.Workflow) && i <= state.Workflow.MaxReviewIterations; i++ {
		reviewStage := fmt.Sprintf("review_%d", i)
		if state.Stages[reviewStage] == "" {
			continue
		}
		context.PreviousReviewPaths = append(context.PreviousReviewPaths, filepath.Join(runPath, fmt.Sprintf("review-%d.json", i)))
		qaStage := fmt.Sprintf("qa_%d", i)
		if state.Stages[qaStage] != "" {
			context.PreviousQAPaths = append(context.PreviousQAPaths, filepath.Join(runPath, fmt.Sprintf("qa-%d.json", i)))
		}
		fixStage := fmt.Sprintf("fix_%d", i)
		if state.Stages[fixStage] != "" {
			context.PreviousFixSummaryPaths = append(context.PreviousFixSummaryPaths, filepath.Join(runPath, fmt.Sprintf("fix-%d-summary.md", i)))
		}
	}
}

// finalizePreviousArtifactContext updates prompt flags after archive-specific collection.
func finalizePreviousArtifactContext(context *promptTemplateContext) {
	context.PreviousReviewCount = len(context.PreviousReviewPaths)
	context.PreviousRepairCount = len(context.PreviousRepairPaths)
	context.PreviousFixSummaryCount = len(context.PreviousFixSummaryPaths)
	context.HasPreviousReview = context.PreviousReviewCount > 0
	context.HasPreviousRepair = context.PreviousRepairCount > 0
	context.HasPreviousQA = len(context.PreviousQAPaths) > 0
	context.HasPreviousFixSummary = context.PreviousFixSummaryCount > 0
	if context.HasPreviousReview {
		context.LatestPreviousReviewPath = context.PreviousReviewPaths[context.PreviousReviewCount-1]
	}
	if context.HasPreviousRepair {
		context.LatestPreviousRepairPath = context.PreviousRepairPaths[context.PreviousRepairCount-1]
	}
	if context.HasPreviousQA {
		context.LatestPreviousQAPath = context.PreviousQAPaths[len(context.PreviousQAPaths)-1]
	}
	if context.HasPreviousFixSummary {
		context.LatestPreviousFixSummaryPath = context.PreviousFixSummaryPaths[context.PreviousFixSummaryCount-1]
	}
}

// qaFindingSummaries renders compact evidence-bearing findings for a targeted prompt.
func qaFindingSummaries(qa QA) []string {
	items := make([]string, 0, len(qa.Findings))
	for _, finding := range qa.Findings {
		items = append(items, fmt.Sprintf("[%s] %s — %s", finding.Severity, finding.Title, finding.Evidence))
	}
	return items
}

// failedAcceptanceIDs extracts only explicitly failed acceptance entries from the latest QA.
func failedAcceptanceIDs(qa QA) []string {
	var ids []string
	for _, result := range qa.AcceptanceMatrix {
		if result.Status != "passed" && strings.TrimSpace(result.ID) != "" {
			ids = append(ids, result.ID)
		}
	}
	return ids
}

// promptRoleSession returns the current stage role's backend-scoped session identity.
func promptRoleSession(state State) (string, string) {
	options, err := state.Workflow.StageOption(state.Stage)
	if err != nil || options.Tool == "" {
		return "", ""
	}
	key := sessionStateKey(options.Tool, stageSessionRoleForState(state, state.Stage))
	return key, state.Sessions[key]
}
