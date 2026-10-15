// Package app owns the human interactive specrelay flow workflow menu.
package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// interactive shows the terminal menu and dispatches the selected action.
func interactive(ctx context.Context, stdin io.Reader, stdout io.Writer, repo string, engine *Engine) error {
	reader := bufio.NewReader(stdin)
	var supersededRun string
	batchID, stoppedBatches, err := FindStartupBatches(repo)
	if err != nil {
		return err
	}
	if batchID != "" {
		fmt.Fprintf(stdout, "Found unfinished batch: %s\n", batchID)
		fmt.Fprintln(stdout, "1. Resume unfinished batch")
		fmt.Fprintln(stdout, "2. Append change proposals")
		fmt.Fprintln(stdout, "3. Plan and append new proposals")
		fmt.Fprintln(stdout, "4. Stop unfinished batch")
		fmt.Fprintln(stdout, "5. Start a new execution")
		choice, err := promptChoice(reader, stdout, 5)
		if err != nil {
			return err
		}
		switch choice {
		case 1:
			return startDetachedBatchCommand(repo, batchID)
		case 2:
			changes, err := chooseAppendableBatchChanges(reader, stdout, repo, batchID)
			if err != nil {
				return err
			}
			return appendSelectedBatchChanges(stdout, repo, batchID, changes)
		case 3:
			if _, _, err := runPlanning(ctx, repo); err != nil {
				return err
			}
			changes, err := appendableBatchChanges(repo, batchID)
			if err != nil {
				return err
			}
			return appendCandidateChanges(reader, stdout, repo, batchID, changes)
		case 4:
			if err := AbortBatch(repo, batchID); err != nil {
				return err
			}
			return nil
		}
	}
	run, stopped, err := FindStartupRuns(repo)
	if err != nil {
		return err
	}
	if len(stoppedBatches) > 0 || len(stopped) > 0 {
		printStoppedHistory(stdout, repo, stoppedBatches, stopped)
	}
	if run != "" {
		fmt.Fprintf(stdout, "Found unfinished run: %s\n", run)
		if state, err := loadState(repo, run); err == nil {
			printHumanStageChecklist(stdout, state)
		}
		fmt.Fprintln(stdout, "1. Resume unfinished run")
		fmt.Fprintln(stdout, "2. Stop unfinished run")
		fmt.Fprintln(stdout, "3. Start a new execution")
		choice, err := promptChoice(reader, stdout, 3)
		if err != nil {
			return err
		}
		switch choice {
		case 1:
			return engine.ResumeDetachedAfterUserChoice(ctx, run)
		case 2:
			if err := AbortRun(repo, run); err != nil {
				return err
			}
			return nil
		case 3:
			supersededRun = run
		}
	}

	changes, err := ListChanges(repo)
	if err != nil {
		return err
	}
	planned := false
	if len(changes) == 0 {
		tool, sessionID, err := runPlanning(ctx, repo)
		if err != nil {
			return err
		}
		engine.PlanningTool = tool
		engine.PlanningSessionID = sessionID
		planned = true
		changes, err = ListChanges(repo)
		if err != nil {
			return err
		}
		if len(changes) == 0 {
			return nil
		}
	} else {
		fmt.Fprintln(stdout, "1. Start planning")
		fmt.Fprintln(stdout, "2. Select existing changes")
		choice, err := promptChoice(reader, stdout, 2)
		if err != nil {
			return err
		}
		if choice == 1 {
			tool, sessionID, err := runPlanning(ctx, repo)
			if err != nil {
				return err
			}
			engine.PlanningTool = tool
			engine.PlanningSessionID = sessionID
			planned = true
			changes, err = ListChanges(repo)
			if err != nil {
				return err
			}
			if len(changes) == 0 {
				return nil
			}
		}
	}
	changes = SortChangesByNumericPrefix(changes)
	if planned && len(changes) == 1 {
		if supersededRun != "" {
			if err := ArchiveSupersededRun(repo, supersededRun); err != nil {
				return err
			}
		}
		return engine.SubmitBatch(ctx, changes)
	}
	changes, err = chooseFromChanges(reader, stdout, changes)
	if err != nil {
		return err
	}
	if supersededRun != "" {
		if err := ArchiveSupersededRun(repo, supersededRun); err != nil {
			return err
		}
	}
	return engine.SubmitBatch(ctx, changes)
}

// chooseAppendableBatchChanges prompts only for changes not already in the batch.
func chooseAppendableBatchChanges(reader *bufio.Reader, stdout io.Writer, repo, batchID string) ([]Change, error) {
	changes, err := appendableBatchChanges(repo, batchID)
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		fmt.Fprintln(stdout, "No active change proposals to append")
		return nil, nil
	}
	return chooseFromChanges(reader, stdout, changes)
}

// appendableBatchChanges loads sorted active changes that are not queued yet.
func appendableBatchChanges(repo, batchID string) ([]Change, error) {
	batch, err := loadBatchState(repo, batchID)
	if err != nil {
		return nil, err
	}
	changes, err := ListChanges(repo)
	if err != nil {
		return nil, err
	}
	return FilterChangesNotInBatch(SortChangesByNumericPrefix(changes), batch), nil
}

// appendCandidateChanges appends all of one candidate or prompts for many.
func appendCandidateChanges(reader *bufio.Reader, stdout io.Writer, repo, batchID string, changes []Change) error {
	switch len(changes) {
	case 0:
		fmt.Fprintln(stdout, "No active change proposals to append")
		return nil
	case 1:
		return appendSelectedBatchChanges(stdout, repo, batchID, changes)
	default:
		selected, err := chooseFromChanges(reader, stdout, changes)
		if err != nil {
			return err
		}
		return appendSelectedBatchChanges(stdout, repo, batchID, selected)
	}
}

// appendSelectedBatchChanges writes selected active changes into an existing batch.
func appendSelectedBatchChanges(stdout io.Writer, repo, batchID string, changes []Change) error {
	if len(changes) == 0 {
		return nil
	}
	added, skipped, err := AppendBatchChanges(repo, batchID, changes)
	if err != nil {
		return err
	}
	if len(added) == 0 {
		fmt.Fprintln(stdout, "No changes appended; selected proposals are already in the batch")
	} else {
		fmt.Fprintf(stdout, "Appended %d changes to batch %s\n", len(added), batchID)
		for _, name := range added {
			fmt.Fprintf(stdout, "- %s\n", name)
		}
	}
	if len(skipped) > 0 {
		fmt.Fprintf(stdout, "Skipped %d duplicate proposals\n", len(skipped))
		for _, name := range skipped {
			fmt.Fprintf(stdout, "- %s\n", name)
		}
	}
	return nil
}

// chooseFromChanges renders a prepared active-change candidate list.
func chooseFromChanges(reader *bufio.Reader, stdout io.Writer, changes []Change) ([]Change, error) {
	if len(changes) == 1 {
		return append([]Change(nil), changes...), nil
	}
	for i, change := range changes {
		fmt.Fprintf(stdout, "%d. %s\n", i+1, change.Name)
	}
	fmt.Fprint(stdout, "> ")
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	return ParseChangeSelection(line, changes)
}

// promptChoice reads a one-based numeric menu selection.
func promptChoice(reader *bufio.Reader, stdout io.Writer, max int) (int, error) {
	fmt.Fprint(stdout, "> ")
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return 0, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || n < 1 || n > max {
		return 0, fmt.Errorf("invalid selection")
	}
	return n, nil
}
