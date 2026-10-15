// Package app renders shared status views for human status and watch output.
package app

import (
	"fmt"
	"strings"
)

// watchStatusLines renders batch or single-run status with spinner for watch refresh.
func watchStatusLines(repo, kind string, ref StatusRef, spinner string) []string {
	if kind == "batch" {
		batch, err := loadBatchState(repo, ref.ID)
		if err != nil {
			return []string{fmt.Sprintf("batch %s state read failed: %v", ref.Alias, err)}
		}
		return watchBatchStatusLines(repo, &batch, ref.Alias, spinner)
	}
	state, err := loadState(repo, ref.ID)
	if err != nil {
		return []string{fmt.Sprintf("workflow %s state read failed: %v", ref.Alias, err)}
	}
	return watchRunStatusLines(repo, state, ref.Alias, spinner)
}

// watchBatchStatusLines formats a batch with spinner in the running stage.
func watchBatchStatusLines(repo string, batch *BatchState, batchAlias string, spinner string) []string {
	var lines []string
	if batchAlias == "" {
		batchAlias = batch.BatchID
	}

	for _, changeName := range batch.Changes {
		runID := batch.RunIDs[changeName]
		if runID != "" {
			if state, err := loadState(repo, runID); err == nil {
				runRefs, _ := ListRunRefs(repo)
				runAlias := RunAliasForID(runRefs, runID)
				view := buildHumanStatusView(repo, state, runAlias, spinner)
				lines = append(lines, statusHeaderText(changeName, view))
				for _, line := range compactStatusLines(view) {
					lines = append(lines, fmt.Sprintf("  %s", line))
				}
				lines = append(lines, qualityLoopBatchRecoveryStatusLines(state, batchAlias)...)
				if batch.Status == batchStatusRunning && isStaleRunningRun(repo, state) {
					lines = append(lines, fmt.Sprintf("  Tip: the run lock is stale; run specrelay flow restart -%s to retry this batch stage", batchAlias))
				}
				continue
			}
		}
		lines = append(lines, fmt.Sprintf("- %s", changeName))
	}
	if batch.Status == batchStatusFailed || batch.Status == batchStatusAborted {
		lines = append(lines, batchFailureLines(repo, *batch, batchAlias)...)
	}

	return lines
}

// watchRunStatusLines formats a single run with spinner in the running stage.
func watchRunStatusLines(repo string, state State, runAlias string, spinner string) []string {
	if runAlias == "" {
		runAlias = state.RunID
	}
	lines := runProposalStatusLines(repo, state, runAlias, spinner)
	if state.BatchID == "" && isRestartableRunState(state) && (state.Status == statusFailed || state.Status == statusInterrupted) {
		lines = append(lines, "Tip: run specrelay flow restart to restart the latest failed task")
	}
	return lines
}

// runProposalStatusLines renders one workflow under its change proposal list item.
func runProposalStatusLines(repo string, state State, runAlias string, runningMarker string) []string {
	view := buildHumanStatusView(repo, state, runAlias, runningMarker)
	lines := []string{statusHeaderText(state.ChangeName, view)}
	for _, line := range compactStatusLines(view) {
		lines = append(lines, fmt.Sprintf("  %s", line))
	}
	lines = append(lines, qualityLoopRecoveryStatusLines(state)...)
	if isStaleRunningRun(repo, state) {
		lines = append(lines, "  Tip: the current run lock is stale; run specrelay flow restart to retry the current stage")
	}
	return lines
}

// qualityLoopRecoveryStatusLines explains how a recoverable quality-loop pause can continue.
func qualityLoopRecoveryStatusLines(state State) []string {
	return qualityLoopRecoveryStatusLinesForCommand(state, "specrelay flow resume or specrelay flow restart", "continue from the original stage")
}

// qualityLoopBatchRecoveryStatusLines points a paused batch at its executable restart target.
func qualityLoopBatchRecoveryStatusLines(state State, batchAlias string) []string {
	command := "specrelay flow restart"
	if batchAlias != "" {
		command += " -" + batchAlias
	}
	return qualityLoopRecoveryStatusLinesForCommand(state, command, "continue the batch from its original stage")
}

// qualityLoopRecoveryStatusLinesForCommand renders shared pause details with a context-valid command.
func qualityLoopRecoveryStatusLinesForCommand(state State, command, outcome string) []string {
	if !isQualityLoopBlockedState(state) {
		return nil
	}
	blockedStatus := state.Status
	if blockedStatus != statusBlockedEnvironment && blockedStatus != statusBlockedStalled {
		blockedStatus = state.Stage
	}
	reason := strings.TrimSpace(state.Error)
	if reason == "" {
		switch blockedStatus {
		case statusBlockedEnvironment:
			reason = "Environment prerequisites are missing"
			if len(state.QualityLoop.MissingEnvironmentNames) > 0 {
				reason += ": " + strings.Join(state.QualityLoop.MissingEnvironmentNames, ", ")
			}
		case statusBlockedStalled:
			reason = "No verifiable progress on the same failure; waiting for new code, evidence, configuration, or user input"
		}
	}
	lines := []string{
		fmt.Sprintf("  Status: %s (resumable pause)", blockedStatus),
	}
	if state.QualityLoop.BlockedFromStage != "" {
		lines = append(lines, "  Original stage: "+state.QualityLoop.BlockedFromStage)
	}
	if reason != "" {
		lines = append(lines, "  Reason: "+reason)
	}
	hint := "After providing new code, evidence, configuration, or user input, "
	if blockedStatus == statusBlockedEnvironment {
		hint = "After supplying the environment prerequisites, "
	}
	return append(lines, "  Tip: "+hint+"run "+command+", "+outcome)
}
