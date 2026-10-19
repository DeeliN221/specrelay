// Package app contains terminal orchestration for the Codex/specrelay workflow.
package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/DeeliN221/specrelay/internal/i18n"
)

// Run parses command arguments and starts the interactive workflow.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	i18n.SetLanguage(EffectiveLanguageForCWD())
	if len(args) > 0 {
		if args[0] == "--version" {
			i18n.Fprintln(stdout, resolvedVersion())
			return nil
		}
		if args[0] == "--help" || args[0] == "-h" {
			printHelp(stdout)
			return nil
		}
		if args[0] == "validate-review" {
			return runValidateReviewArtifact(args[1:], stdout)
		}
		if args[0] == "validate-repair" {
			return runValidateRepairArtifact(args[1:], stdout)
		}
		if args[0] == "validate-qa" {
			return runValidateQAArtifact(args[1:], stdout)
		}
		if args[0] == "update" {
			if len(args) != 1 {
				return i18n.Errorf("Usage: specrelay flow update")
			}
			return runUpdate(stdout)
		}
		if args[0] == "config" {
			options, err := parseConfigCommandOptions(args[1:])
			if err != nil || !options.Global {
				return runWithRepository(args, stdin, stdout)
			}
			return handleFlowConfigCommand(context.Background(), args, stdout, "", nil)
		}
		if args[0] == "graph" {
			repo, err := GitRoot(".")
			if err != nil {
				return err
			}
			return runGraph(repo, args[1:], stdout)
		}
	}
	return runWithRepository(args, stdin, stdout)
}

// runWithRepository wires repository state, workflow tools, and registry dispatch.
func runWithRepository(args []string, stdin io.Reader, stdout io.Writer) error {
	repo, err := GitRoot(".")
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if len(args) == 0 || flowCommandRequiresWorkflowTools(args[0]) {
		if err := ensureBaseWorkflowCommands(); err != nil {
			return err
		}
	}
	registry := NewAgentRegistry()
	engine := NewEngine(repo, registry)
	engine.Output = stdout
	engine.inPlaceProgress = supportsInPlaceProgress(stdout)

	if len(args) > 0 {
		if spec, ok := flowCommandByName(args[0]); ok {
			return spec.Handler(ctx, args, stdout, repo, engine)
		}
		return i18n.Errorf("unknown command %q", args[0])
	}

	return interactive(ctx, stdin, stdout, repo, engine)
}

// submitAllActiveChanges starts or extends the active change queue.
func submitAllActiveChanges(ctx context.Context, stdout io.Writer, repo string, engine *Engine) error {
	changes, err := ListChanges(repo)
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		i18n.Fprintln(stdout, "No active change proposals")
		return nil
	}
	batchID, err := FindUnfinishedBatch(repo)
	if err != nil {
		return err
	}
	if batchID != "" {
		batch, err := loadBatchState(repo, batchID)
		if err != nil {
			return err
		}
		appendable := FilterChangesNotInBatch(SortChangesByNumericPrefix(changes), batch)
		if len(appendable) == 0 {
			i18n.Fprintf(stdout, "Batch %s is already running; no active changes to append\n", batchID)
			return nil
		}
		i18n.Fprintf(stdout, "Appending new active changes to running batch %s\n", batchID)
		return appendSelectedBatchChanges(stdout, repo, batchID, appendable)
	}
	return engine.SubmitBatch(ctx, changes)
}

// flowCommandRequiresWorkflowTools reports whether a registered command invokes workflow backends.
func flowCommandRequiresWorkflowTools(name string) bool {
	spec, ok := flowCommandByName(name)
	return !ok || spec.NeedsWorkflowTools
}

// GitRoot returns the absolute root of the current git repository.
func GitRoot(dir string) (string, error) {
	gitPath, err := resolveCommand("git")
	if err != nil {
		return "", err
	}
	cmd := commandContext(context.Background(), gitPath, "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", i18n.Errorf("current directory is not inside a Git repository: %w", err)
	}
	return filepath.Clean(strings.TrimSpace(string(out))), nil
}

// printStoppedHistory writes stopped batch and workflow notices in a structured form.
func printStoppedHistory(stdout io.Writer, repo string, batches []BatchState, runs []State) {
	batchRefs, _ := ListBatchRefs(repo)
	runRefs, _ := ListRunRefs(repo)
	i18n.Fprintln(stdout, "Stopped historical tasks:")
	for _, batch := range batches {
		alias := batchAliasForID(batchRefs, batch.BatchID)
		i18n.Fprintf(stdout, "- batch %s %s %s\n", alias, batch.BatchID, batch.Status)
		if batch.FailedChange != "" {
			i18n.Fprintf(stdout, "  change: %s\n", batch.FailedChange)
		}
		if batch.FailedRunID != "" {
			runAlias := RunAliasForID(runRefs, batch.FailedRunID)
			if runAlias == "" {
				runAlias = batch.FailedRunID
			}
			i18n.Fprintf(stdout, "  run: %s %s\n", runAlias, batch.FailedRunID)
		}
		if summary := humanBatchFailureSummary(repo, batch, batch.FailedChange, batch.FailedRunID); summary != "" {
			i18n.Fprintf(stdout, "  reason: %s\n", summary)
		}
		if batch.Status == batchStatusFailed {
			if isBatchRestartRecoverable(repo, batch) {
				i18n.Fprintf(stdout, "  Tip: run %s to remove the failed record and continue this batch\n", restartCommandForAlias(alias))
				i18n.Fprintf(stdout, "  Archive: run %s to archive the failure and start a new workflow\n", archiveCommandForAlias(alias))
			} else {
				i18n.Fprintf(stdout, "  Archive: run %s to archive the failure and start a new workflow\n", archiveCommandForAlias(alias))
				i18n.Fprintf(stdout, "  Cleanup: run specrelay flow clean to remove failed or invalid runtime state for this project\n")
				i18n.Fprintf(stdout, "        This removes only specrelay flow records; it does not revert code changes\n")
			}
		} else if batch.Status == batchStatusAborted {
			i18n.Fprintf(stdout, "  Cleanup: run specrelay flow clean to remove failed or invalid runtime state for this project\n")
			i18n.Fprintf(stdout, "        This removes only specrelay flow records; it does not revert code changes\n")
		}
	}
	for _, state := range runs {
		runAlias := RunAliasForID(runRefs, state.RunID)
		if runAlias == "" {
			runAlias = state.RunID
		}
		i18n.Fprintf(stdout, "- workflow %s %s %s\n", runAlias, state.RunID, state.Status)
		if state.ChangeName != "" {
			i18n.Fprintf(stdout, "  change: %s\n", state.ChangeName)
		}
		if reason := stoppedRunReason(state); reason != "" {
			i18n.Fprintf(stdout, "  reason: %s\n", reason)
		}
		if state.BatchID == "" {
			if isRestartableRunCandidate(repo, state) {
				i18n.Fprintf(stdout, "  Tip: run %s to restart this workflow\n", restartCommandForAlias(runAlias))
				i18n.Fprintf(stdout, "  Archive: run %s to archive the failure and start a new workflow\n", archiveCommandForAlias(runAlias))
			} else {
				if isArchivableRun(state) {
					i18n.Fprintf(stdout, "  Archive: run %s to archive the failure and start a new workflow\n", archiveCommandForAlias(runAlias))
				}
				i18n.Fprintf(stdout, "  Cleanup: run specrelay flow clean to remove failed or invalid runtime state for this project\n")
				i18n.Fprintf(stdout, "        This removes only specrelay flow records; it does not revert code changes\n")
			}
		}
	}
}

// restartCommandForAlias returns the shortest human restart command for a status alias.
func restartCommandForAlias(alias string) string {
	if strings.HasPrefix(alias, "b") || strings.HasPrefix(alias, "w") {
		return "specrelay flow restart -" + alias
	}
	return "specrelay flow restart"
}

// archiveCommandForAlias returns the shortest human archive command for a status alias.
func archiveCommandForAlias(alias string) string {
	if strings.HasPrefix(alias, "b") || strings.HasPrefix(alias, "w") {
		return "specrelay flow archive -" + alias
	}
	return "specrelay flow archive"
}

// batchAliasForID returns a batch short alias or falls back to the real id.
func batchAliasForID(refs []StatusRef, batchID string) string {
	for _, ref := range refs {
		if ref.ID == batchID {
			return ref.Alias
		}
	}
	return batchID
}

// stoppedBatchReason chooses the shortest useful reason for a stopped batch.
func stoppedBatchReason(repo string, batch BatchState) string {
	if batch.FailedRunID != "" {
		if state, err := loadState(repo, batch.FailedRunID); err == nil {
			if reason := stoppedRunReason(state); reason != "" {
				return reason
			}
		}
	}
	if batch.Status != "" {
		return batch.Status
	}
	return batch.Error
}

// stoppedRunReason chooses a stable status enum before falling back to error text.
func stoppedRunReason(state State) string {
	if state.Status != "" && state.Status != statusFailed {
		return state.Status
	}
	if state.Stage == statusBlocked || state.Stage == statusValidationBlocked || state.Stage == statusAcceptanceContractBlocked {
		return state.Stage
	}
	if state.Status != "" {
		return state.Status
	}
	return sanitizeErrorForHuman(state.Error)
}

// humanRunFailureSummary produces a Chinese failure reason with change name and stage role.
func humanRunFailureSummary(state State, changeName string) string {
	if changeName == "" {
		changeName = state.ChangeName
	}
	prefix := ""
	if changeName != "" {
		prefix = changeName + ": "
	}
	stageRole := humanStageRole(state.Stage)
	if state.Status == statusBlocked || state.Stage == statusBlocked {
		stageRole = humanBlockedStageRole(state)
	}

	switch {
	case state.Status == statusBlocked || state.Stage == statusBlocked:
		reason := state.Error
		if reason == "" {
			reason = "review repair limit reached; automatic continuation is unavailable"
		}
		return prefix + stageRole + " failed: " + reason
	case state.Status == statusValidationBlocked || state.Stage == statusValidationBlocked:
		reason := state.Error
		if reason == "" {
			reason = "stage validation limit reached; automatic continuation is unavailable"
		}
		return prefix + stageRole + " failed: " + reason
	case state.Status == statusAcceptanceContractBlocked || state.Stage == statusAcceptanceContractBlocked:
		reason := state.Error
		if reason == "" {
			reason = "acceptance contract preflight failed; automatic continuation is unavailable"
		}
		return prefix + stageRole + " failed: " + reason
	case state.Status == statusAborted || state.Status == "aborted":
		reason := state.Error
		if reason == "" {
			reason = "stopped by user"
		}
		return prefix + reason
	case state.Status == statusInterrupted:
		return prefix + stageRole + "failed: workflow interrupted"
	case state.Status == statusFailed:
		return prefix + stageRole + " failed: " + sanitizeErrorForHuman(state.Error)
	default:
		return prefix + stageRole + " failed: " + sanitizeErrorForHuman(state.Error)
	}
}

// humanBlockedStageRole maps the sealed workflow's blocking role to a Chinese stage label.
func humanBlockedStageRole(state State) string {
	switch blockedWorkflowRole(state) {
	case "qa":
		return "independent testing"
	case "reviewer":
		return "review stage"
	default:
		return "optimization stage"
	}
}

// humanStageRole maps stage names to short Chinese role labels.
func humanStageRole(stage string) string {
	switch {
	case stage == "execution":
		return "write stage"
	case strings.HasPrefix(stage, "repair_"):
		return "optimization stage"
	case strings.HasPrefix(stage, "review_"):
		return "review stage"
	case strings.HasPrefix(stage, "fix_"):
		return "repair stage"
	case strings.HasPrefix(stage, "qa_"):
		return "independent testing"
	case stage == "archive":
		return "archive stage"
	case stage == statusBlocked:
		return "optimization stage"
	case stage == statusValidationBlocked:
		return "stage validation"
	case stage == statusAcceptanceContractBlocked:
		return "acceptance preflight"
	default:
		return "current stage"
	}
}

// sanitizeErrorForHuman hides raw backend diagnostics from human-readable output.
func sanitizeErrorForHuman(raw string) string {
	if raw == "" {
		return "agent execution failed"
	}
	lower := strings.ToLower(raw)
	switch {
	case strings.Contains(lower, "backend-api") || strings.Contains(lower, "wss://") ||
		strings.Contains(lower, "websocket") || strings.Contains(lower, "tls handshake"):
		return "agent backend connection failed"
	case strings.Contains(lower, "stderr"):
		return "agent execution failed"
	default:
		return raw
	}
}

// printChanges writes active change names for script-friendly usage.
func printChanges(stdout io.Writer, repo string) error {
	changes, err := ListChanges(repo)
	if err != nil {
		return err
	}
	for _, change := range changes {
		i18n.Fprintln(stdout, change.Name)
	}
	return nil
}

// printHumanStatus writes the selected human status target.
func printHumanStatus(stdout io.Writer, repo string, args ...string) error {
	kind, ref, err := ResolveStatusTarget(repo, args)
	if err != nil {
		if len(args) == 0 && err.Error() == "no specrelay flow run" {
			i18n.Fprintln(stdout, err)
			return nil
		}
		return err
	}
	if kind == "batch" {
		batch, err := loadBatchState(repo, ref.ID)
		if err != nil {
			return err
		}
		runRefs, err := ListRunRefs(repo)
		if err != nil {
			return err
		}
		lines := batchStatusLines(repo, &batch, ref.Alias, runRefs)
		for _, line := range lines {
			i18n.Fprintln(stdout, line)
		}
		printUpdateHint(stdout)
		return nil
	}

	state, err := loadState(repo, ref.ID)
	if err != nil {
		return err
	}
	for _, line := range runProposalStatusLines(repo, state, ref.Alias, "→") {
		i18n.Fprintln(stdout, line)
	}
	printUpdateHint(stdout)
	return nil
}

// projectName returns the repository directory name for human status hints.
func projectName(repo string) string {
	name := filepath.Base(filepath.Clean(repo))
	if name == "." || name == string(filepath.Separator) {
		return "current project"
	}
	return name
}

// printHelp writes supported command-line options.
func printHelp(stdout io.Writer) {
	i18n.Fprintln(stdout, "specrelay flow executes specrelay changes through a controlled Codex/Pi workflow.")
	fmt.Fprintln(stdout)
	i18n.Fprintln(stdout, "Usage: ")
	i18n.Fprintln(stdout, "  specrelay flow")
	i18n.Fprintln(stdout, "  specrelay flow config [--global]")
	i18n.Fprintln(stdout, "  specrelay flow run | specrelay flow r")
	i18n.Fprintln(stdout, "  specrelay flow status")
	i18n.Fprintln(stdout, "  specrelay flow restart")
	i18n.Fprintln(stdout, "  specrelay flow stop")
	i18n.Fprintln(stdout, "  specrelay flow archive")
	i18n.Fprintln(stdout, "  specrelay flow loop")
	i18n.Fprintln(stdout, "  specrelay flow clean [--agent-sessions] [--dry-run] [--json]")
	i18n.Fprintln(stdout, "  specrelay flow watch")
	i18n.Fprintln(stdout, "  specrelay flow update")
	i18n.Fprintln(stdout, "  specrelay flow --list-changes")
	i18n.Fprintln(stdout, "  specrelay flow --run <change-name>")
	i18n.Fprintln(stdout, "  specrelay flow --resume")
	i18n.Fprintln(stdout, "  specrelay flow validate-review --artifact <artifact-path> [--json]")
	i18n.Fprintln(stdout, "  specrelay flow validate-repair --artifact <artifact-path> [--json]")
	i18n.Fprintln(stdout, "  specrelay flow validate-qa --artifact <artifact-path> --acceptance <acceptance-path> [--json]")
	i18n.Fprintln(stdout, "  specrelay flow --version")
	fmt.Fprintln(stdout)
	i18n.Fprintln(stdout, "Interactive commands:")
	i18n.Fprintln(stdout, "  specrelay flow                         Plan, select an active change, or resume a run")
	i18n.Fprintln(stdout, "  specrelay flow clean [--agent-sessions] [--dry-run] [--json] Clean failed or invalid runtime state for this project")
	i18n.Fprintln(stdout, "  specrelay flow config [--global] Write repository specrelay-flow.yaml or user ~/specrelay-flow.yaml")
	i18n.Fprintln(stdout, "  specrelay flow run | specrelay flow r              Select all active changes and start the task queue")
	i18n.Fprintln(stdout, "  specrelay flow status                  Print progress of the latest run")
	i18n.Fprintln(stdout, "  specrelay flow restart [-bN|-wN]       Restart the latest failed or interrupted batch/workflow")
	i18n.Fprintln(stdout, "  specrelay flow stop                    Stop the active batch workflow")
	i18n.Fprintln(stdout, "  specrelay flow archive [-bN|-wN]       Archive the latest failed batch/workflow state")
	i18n.Fprintln(stdout, "  specrelay flow loop                    Monitor each minute; archive failures and continue remaining changes")
	i18n.Fprintln(stdout, "  specrelay flow watch [-bN|-wN]         Continuously refresh active task status")
	i18n.Fprintln(stdout, "  specrelay flow update                  Update specrelay and print backup/rollback commands")
	i18n.Fprintln(stdout, "  specrelay flow --list-changes          Print active docs/changes names")
	i18n.Fprintln(stdout, "  specrelay flow --run <change-name>     Start an appendable queue for an active change")
	i18n.Fprintln(stdout, "  specrelay flow --resume                Resume the latest unfinished run")
	fmt.Fprintln(stdout)
	i18n.Fprintln(stdout, "Runner JSON commands:")
	i18n.Fprintln(stdout, "  specrelay flow validate-review --artifact <artifact-path> [--json]")
	i18n.Fprintln(stdout, "  specrelay flow validate-repair --artifact <artifact-path> [--json]")
	i18n.Fprintln(stdout, "  specrelay flow validate-qa --artifact <artifact-path> --acceptance <acceptance-path> [--json]")
	i18n.Fprintln(stdout, "  specrelay flow contract --json")
	i18n.Fprintln(stdout, "  specrelay flow list-changes --json")
	i18n.Fprintln(stdout, "  specrelay flow run --change <change-name> --json")
	i18n.Fprintln(stdout, "  specrelay flow run-acceptance --change <change-name> --json")
	i18n.Fprintln(stdout, "  specrelay flow resume --run-id <run-id> --json")
	i18n.Fprintln(stdout, "  specrelay flow restart --run-id <run-id> --json")
	i18n.Fprintln(stdout, "  specrelay flow restart --batch-id <batch-id> --json")
	i18n.Fprintln(stdout, "  specrelay flow status --run-id <run-id> --json")
	i18n.Fprintln(stdout, "  specrelay flow abort --run-id <run-id> --json")
	i18n.Fprintln(stdout, "  specrelay flow batch --batch-id <batch-id> --json")
	i18n.Fprintln(stdout, "  specrelay flow batch append --batch-id <batch-id> --change <change-name> --json")
	fmt.Fprintln(stdout)
	i18n.Fprintln(stdout, "Files:")
	i18n.Fprintln(stdout, "  specrelay-flow.yaml                    Repository workflow configuration; create with specrelay flow config")
	i18n.Fprintln(stdout, "  ~/specrelay-flow.yaml                  User-level workflow defaults")
	i18n.Fprintln(stdout, "  docs/changes/<name>/       active specrelay changes")
	i18n.Fprintln(stdout, "  ${XDG_STATE_HOME:-~/.local/state}/specrelay/flow/repos/<repo-key>/runs/<run-id>/")
	i18n.Fprintln(stdout, "                             Sealed run state, prompt snapshots, and artifacts")
}

// printStageChecklist writes a plain text workflow status summary.
func printStageChecklist(stdout io.Writer, state State) {
	for _, line := range compactStatusLines(buildHumanStatusView("", state, state.RunID, "→")) {
		i18n.Fprintln(stdout, line)
	}
}

// printHumanStageChecklist shows the durable current stage when no live runtime map exists.
func printHumanStageChecklist(stdout io.Writer, state State) {
	runtime := map[string]stageRuntime{}
	if state.Status == statusRunning && state.Stage != "" {
		runtime[state.Stage] = stageRuntime{}
	}
	_ = runtime
	for _, line := range compactStatusLines(buildHumanStatusView("", state, state.RunID, "→")) {
		i18n.Fprintln(stdout, line)
	}
	if state.BatchID == "" && isRestartableRunState(state) && (state.Status == statusFailed || state.Status == statusInterrupted) {
		i18n.Fprintln(stdout, "Tip: run specrelay flow restart to restart the latest failed task")
	}
}

// printHumanStatusStageChecklist shows workflow status with run-local human artifacts.
func printHumanStatusStageChecklist(stdout io.Writer, repo string, state State) {
	runtime := map[string]stageRuntime{}
	if state.Status == statusRunning && state.Stage != "" {
		runtime[state.Stage] = stageRuntime{}
	}
	_ = runtime
	for _, line := range compactStatusLines(buildHumanStatusView(repo, state, state.RunID, "→")) {
		i18n.Fprintln(stdout, line)
	}
	if state.BatchID == "" && isRestartableRunState(state) && (state.Status == statusFailed || state.Status == statusInterrupted) {
		i18n.Fprintln(stdout, "Tip: run specrelay flow restart to restart the latest failed task")
	}
}

// sessionRoleLabel maps agent roles to the short human status vocabulary.
func sessionRoleLabel(role string) string {
	for _, workflowRole := range statusRoles() {
		if workflowRole.Session == role {
			return i18n.T(workflowRole.Label)
		}
	}
	return i18n.T("write")
}

// supportsInPlaceProgress reports whether stdout is a terminal-like file.
func supportsInPlaceProgress(stdout io.Writer) bool {
	file, ok := stdout.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// spinnerFrames are the ASCII spinner animation frames for specrelay flow watch.
var spinnerFrames = []string{"|", "/", "-", "\\"}

// runWatch implements the specrelay flow watch command with continuous status refresh.
func runWatch(stdout io.Writer, repo string, args ...string) error {
	targetKind, targetRef, err := resolveWatchTarget(repo, args)
	if err != nil {
		if strings.Contains(err.Error(), "no active batch or workflow") {
			i18n.Fprintln(stdout, "No active batch or workflow")
			return nil
		}
		return err
	}
	explicitTarget := len(args) == 1
	tty := supportsInPlaceProgress(stdout)
	frameIdx := 0
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Listen for interrupt signal to allow timeout-based testing.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	render := func() {
		frameKind, frameRef, frameErr := resolveWatchFrameTarget(repo, explicitTarget, targetKind, targetRef)
		if frameErr != nil {
			if tty && frameIdx > 0 {
				i18n.Fprint(stdout, "\x1b[H\x1b[2J")
			}
			i18n.Fprintln(stdout, "No active batch or workflow")
			frameIdx++
			return
		}
		var lines []string
		lines = append(lines, watchStatusLines(repo, frameKind, frameRef, spinnerFrames[frameIdx%len(spinnerFrames)])...)
		if tty && frameIdx > 0 {
			i18n.Fprint(stdout, "\x1b[H\x1b[2J")
		}
		for _, line := range lines {
			i18n.Fprintln(stdout, line)
		}
		frameIdx++
	}

	render()
	for {
		select {
		case <-ticker.C:
			render()
		case <-sigCh:
			return nil
		}
	}
}

// resolveWatchFrameTarget refreshes an implicit watch target while preserving an explicit alias.
func resolveWatchFrameTarget(repo string, explicit bool, kind string, ref StatusRef) (string, StatusRef, error) {
	if explicit {
		return kind, ref, nil
	}
	return resolveWatchTarget(repo, nil)
}

// resolveWatchTarget picks the watch target: explicit -bN/-wN or default to running batch > single-run.
func resolveWatchTarget(repo string, args []string) (kind string, ref StatusRef, err error) {
	if len(args) == 0 {
		// Default: running batch first, then running single-run.
		batchRefs, _ := ListBatchRefs(repo)
		for _, batchRef := range batchRefs {
			batch, loadErr := loadBatchState(repo, batchRef.ID)
			if loadErr == nil && batch.Status == batchStatusRunning {
				return "batch", batchRef, nil
			}
		}
		runID, runErr := FindUnfinishedRun(repo)
		if runErr == nil && runID != "" {
			runRefs, _ := ListRunRefs(repo)
			for _, runRef := range runRefs {
				if runRef.ID == runID {
					return "run", runRef, nil
				}
			}
			return "run", StatusRef{Alias: "w1", ID: runID}, nil
		}
		return "", StatusRef{}, i18n.Errorf("No active batch or workflow")
	}
	if len(args) != 1 {
		return "", StatusRef{}, i18n.Errorf("Usage: specrelay flow watch [-bN|-wN]")
	}
	arg := args[0]
	switch {
	case strings.HasPrefix(arg, "-b"):
		ref, err := resolveIndexedRef(repo, arg, "-b", ListBatchRefs)
		return "batch", ref, err
	case strings.HasPrefix(arg, "-w"):
		ref, err := resolveIndexedRef(repo, arg, "-w", ListRunRefs)
		return "run", ref, err
	default:
		return "", StatusRef{}, i18n.Errorf("Usage: specrelay flow watch [-bN|-wN]")
	}
}

type configCommandOptions struct {
	Global bool
}

func parseConfigCommandOptions(args []string) (configCommandOptions, error) {
	options := configCommandOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--global":
			options.Global = true
		default:
			return configCommandOptions{}, i18n.Errorf("Usage: specrelay flow config [--global]")
		}
	}
	return options, nil
}
