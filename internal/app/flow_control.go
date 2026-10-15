// Package app controls operator-level specrelay flow batch lifecycle commands.
package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	loopPendingStartGrace    = 2 * time.Minute
	loopRepeatedFailureLimit = 2
)

// dispatchStopCommand stops the current running batch workflow.
func dispatchStopCommand(args []string, stdout io.Writer, repo string) error {
	if len(args) != 1 {
		return fmt.Errorf("Usage: specrelay flow stop")
	}
	batchID, err := FindUnfinishedBatch(repo)
	if err != nil {
		return err
	}
	if batchID == "" {
		fmt.Fprintln(stdout, "No active batch workflow")
		return nil
	}
	if err := AbortBatch(repo, batchID); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Stopped batch workflow %s\n", batchID)
	return nil
}

// dispatchArchiveCommand archives the newest failed workflow runtime record.
func dispatchArchiveCommand(args []string, stdout io.Writer, repo string) error {
	kind, ref, err := ResolveArchiveTarget(repo, args[1:])
	if err != nil {
		return err
	}
	switch kind {
	case "batch":
		if _, err := ArchiveFailedBatch(repo, ref.ID); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Archived failed batch workflow %s; run specrelay flow run to start a new workflow\n", ref.Alias)
		return nil
	case "run":
		if err := ArchiveFailedRun(repo, ref.ID, "archived failed workflow"); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Archived failed workflow %s; run specrelay flow run to start a new workflow\n", ref.Alias)
		return nil
	default:
		return fmt.Errorf("No failed workflow to archive")
	}
}

// dispatchLoopCommand monitors batch progress and starts continuation batches after failures.
func dispatchLoopCommand(ctx context.Context, args []string, stdout io.Writer, repo string, engine *Engine) error {
	if len(args) == 3 && args[1] == "--worker" && args[2] == "--json" {
		return runBatchLoopWorker(ctx, stdout, repo, engine, time.Minute)
	}
	if len(args) != 1 {
		return fmt.Errorf("Usage: specrelay flow loop")
	}
	active, err := loopWorkerActive(repo)
	if err != nil {
		return err
	}
	if active {
		fmt.Fprintln(stdout, "Background loop monitor is already running")
		return nil
	}
	if err := startDetachedLoopCommand(repo); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Started background loop monitor")
	return nil
}

// ResolveArchiveTarget maps archive arguments to the newest archivable batch or run.
func ResolveArchiveTarget(repo string, args []string) (kind string, ref StatusRef, err error) {
	if len(args) == 0 {
		batches, err := ListBatchRefs(repo)
		if err != nil {
			return "", StatusRef{}, err
		}
		for _, batchRef := range batches {
			batch, err := loadBatchState(repo, batchRef.ID)
			if err == nil && isArchivableBatch(batch) {
				return "batch", batchRef, nil
			}
		}
		runs, err := ListRunRefs(repo)
		if err != nil {
			return "", StatusRef{}, err
		}
		for _, runRef := range runs {
			state, err := loadState(repo, runRef.ID)
			if err == nil && state.BatchID == "" && isArchivableRun(state) {
				return "run", runRef, nil
			}
		}
		return "", StatusRef{}, fmt.Errorf("No failed workflow to archive")
	}
	if len(args) != 1 {
		return "", StatusRef{}, fmt.Errorf("Usage: specrelay flow archive [-bN|-wN]")
	}
	switch {
	case strings.HasPrefix(args[0], "-b"):
		ref, err := resolveIndexedRef(repo, args[0], "-b", ListBatchRefs)
		return "batch", ref, err
	case strings.HasPrefix(args[0], "-w"):
		ref, err := resolveIndexedRef(repo, args[0], "-w", ListRunRefs)
		return "run", ref, err
	default:
		return "", StatusRef{}, fmt.Errorf("Usage: specrelay flow archive [-bN|-wN]")
	}
}

// ArchiveFailedBatch marks a failed batch as archived and preserves its failed run record.
func ArchiveFailedBatch(repo, batchID string) (BatchState, error) {
	batch, err := loadBatchState(repo, batchID)
	if err != nil {
		return BatchState{}, err
	}
	if !isArchivableBatch(batch) {
		return BatchState{}, fmt.Errorf("batch workflow %s with status %s cannot be archived", batchID, batch.Status)
	}
	if runID := failedRunIDForBatch(batch); runID != "" {
		if err := ArchiveFailedRun(repo, runID, "archived with failed batch "+batchID); err != nil && !isAlreadyArchivedRunError(err) {
			return BatchState{}, err
		}
	}
	batch.Status = batchStatusArchived
	if batch.Error == "" {
		batch.Error = "Archived failed batch workflow"
	}
	if err := saveBatchState(repo, batch); err != nil {
		return BatchState{}, err
	}
	return batch, nil
}

// ArchiveFailedRun marks a failed standalone run as archived without deleting its evidence.
func ArchiveFailedRun(repo, runID, reason string) error {
	state, err := loadState(repo, runID)
	if err != nil {
		return err
	}
	if state.Status == statusArchived {
		return alreadyArchivedRunError{runID: runID}
	}
	if !isArchivableRun(state) {
		return fmt.Errorf("workflow %s with status %s cannot be archived", runID, state.Status)
	}
	if err := clearArchiveLock(repo, runID); err != nil {
		return err
	}
	state.Status = statusArchived
	if reason != "" {
		state.Error = reason
	}
	if err := saveState(repo, state); err != nil {
		return err
	}
	return nil
}

// runBatchLoop polls batch state until the batch succeeds or the operator stops it.
func runBatchLoop(ctx context.Context, stdout io.Writer, repo string, engine *Engine, interval time.Duration) error {
	if interval <= 0 {
		interval = time.Minute
	}
	for {
		done, err := runBatchLoopTick(ctx, stdout, repo, engine)
		if err != nil || done {
			return err
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// runBatchLoopWorker holds the repo loop lock while the detached monitor is active.
func runBatchLoopWorker(ctx context.Context, stdout io.Writer, repo string, engine *Engine, interval time.Duration) error {
	unlock, acquired, err := acquireLoopWorkerLock(repo)
	if err != nil || !acquired {
		return err
	}
	defer unlock()
	return withLoopWorkerLifecycle(repo, func() error {
		return runBatchLoop(ctx, stdout, repo, engine, interval)
	})
}

// runBatchLoopTick performs one monitor cycle and returns true when monitoring is complete.
func runBatchLoopTick(ctx context.Context, stdout io.Writer, repo string, engine *Engine) (bool, error) {
	batchID, err := FindUnfinishedBatch(repo)
	if err != nil {
		return false, err
	}
	if batchID != "" {
		batch, err := loadBatchState(repo, batchID)
		if err != nil {
			return false, err
		}
		decision := assessBatchForLoop(repo, batch)
		switch decision.Status {
		case loopDecisionRunning:
			return false, nil
		case loopDecisionDone:
			fmt.Fprintf(stdout, "Batch workflow %s completed\n", batch.BatchID)
			return true, nil
		case loopDecisionStopped:
			fmt.Fprintf(stdout, "Batch workflow %s stopped; exiting loop\n", batch.BatchID)
			return true, nil
		case loopDecisionFailed:
			return archiveAndSubmitContinuation(ctx, stdout, repo, engine, batch, decision.Reason)
		}
	}
	latest, err := FindLatestBatch(repo)
	if err != nil {
		return false, err
	}
	if latest != nil {
		decision := assessBatchForLoop(repo, *latest)
		switch decision.Status {
		case loopDecisionDone:
			return submitActiveChangesOrFinishLoop(ctx, stdout, repo, engine, fmt.Sprintf("Batch workflow %s completed\n", latest.BatchID))
		case loopDecisionStopped:
			return submitActiveChangesOrFinishLoop(ctx, stdout, repo, engine, fmt.Sprintf("Batch workflow %s stopped; exiting loop\n", latest.BatchID))
		case loopDecisionFailed:
			return archiveAndSubmitContinuation(ctx, stdout, repo, engine, *latest, decision.Reason)
		}
	}
	return submitActiveChangesForLoop(ctx, stdout, repo, engine)
}

// archiveAndSubmitContinuation archives a failed batch and submits its unfinished active changes.
func archiveAndSubmitContinuation(ctx context.Context, stdout io.Writer, repo string, engine *Engine, batch BatchState, reason string) (bool, error) {
	changes, err := continuationChanges(repo, batch)
	if err != nil {
		return false, err
	}
	if err := markStaleRunForLoop(repo, failedRunIDForBatch(batch), reason); err != nil {
		return false, err
	}
	if batch.Status == batchStatusRunning {
		if err := markBatchFailedForLoop(repo, batch, reason); err != nil {
			return false, err
		}
		latest, err := loadBatchState(repo, batch.BatchID)
		if err != nil {
			return false, err
		}
		batch = latest
	}
	blocked := batchFailureBlocksRetry(repo, batch)
	fingerprint := loopFailureFingerprint(repo, batch)
	if _, err := ArchiveFailedBatch(repo, batch.BatchID); err != nil {
		return false, err
	}
	if reason == "" {
		reason = "failed"
	}
	fmt.Fprintf(stdout, "Archived failed batch workflow %s: %s\n", batch.BatchID, reason)
	if blocked {
		fmt.Fprintln(stdout, "This failure requires manual intervention; stopping the loop")
		return true, nil
	}
	repeated, err := loopRepeatedFailureCount(repo, batch, fingerprint)
	if err != nil {
		return false, err
	}
	if repeated >= loopRepeatedFailureLimit {
		fmt.Fprintf(stdout, "The same failure occurred %d consecutive times; stopping the loop\n", repeated)
		return true, nil
	}
	if len(changes) == 0 {
		fmt.Fprintln(stdout, "No active changes to continue; exiting loop")
		return true, nil
	}
	return false, engine.SubmitBatch(ctx, changes)
}

// batchFailureBlocksRetry reports terminal workflow gates that require a human decision.
func batchFailureBlocksRetry(repo string, batch BatchState) bool {
	if fingerprint := normalizeLoopFailureReason(batch.Error); strings.Contains(fingerprint, statusBlocked) ||
		strings.Contains(fingerprint, statusValidationBlocked) || strings.Contains(fingerprint, statusAcceptanceContractBlocked) {
		return true
	}
	runID := failedRunIDForBatch(batch)
	if runID == "" {
		return false
	}
	state, err := loadState(repo, runID)
	if err != nil {
		return false
	}
	return state.Status == statusBlocked || state.Status == statusValidationBlocked ||
		state.Status == statusAcceptanceContractBlocked || state.Stage == statusBlocked ||
		state.Stage == statusValidationBlocked || state.Stage == statusAcceptanceContractBlocked
}

// loopFailureFingerprint produces a stable retry key without volatile run identifiers.
func loopFailureFingerprint(repo string, batch BatchState) string {
	if reason := normalizeLoopFailureReason(batch.Error); reason != "" {
		return reason
	}
	if state, err := loadState(repo, failedRunIDForBatch(batch)); err == nil {
		return normalizeLoopFailureReason(state.Stage + "/" + state.Status)
	}
	return "unknown"
}

// normalizeLoopFailureReason removes the generated run-id prefix from terminal batch errors.
func normalizeLoopFailureReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if _, suffix, found := strings.Cut(reason, " stopped: "); found {
		return suffix
	}
	return reason
}

// loopRepeatedFailureCount counts consecutive archived failures for one change and fingerprint.
func loopRepeatedFailureCount(repo string, current BatchState, fingerprint string) (int, error) {
	refs, err := ListBatchRefs(repo)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, ref := range refs {
		batch, err := loadBatchState(repo, ref.ID)
		if err != nil || batch.FailedChange != current.FailedChange {
			continue
		}
		if loopFailureFingerprint(repo, batch) != fingerprint {
			break
		}
		count++
	}
	return count, nil
}

// submitActiveChangesForLoop starts a batch when loop is launched without an active one.
func submitActiveChangesForLoop(ctx context.Context, stdout io.Writer, repo string, engine *Engine) (bool, error) {
	return submitActiveChangesOrFinishLoop(ctx, stdout, repo, engine, "No active change proposals; exiting loop\n")
}

// submitActiveChangesOrFinishLoop starts active proposals, otherwise prints the terminal message.
func submitActiveChangesOrFinishLoop(ctx context.Context, stdout io.Writer, repo string, engine *Engine, terminalMessage string) (bool, error) {
	changes, err := ListChanges(repo)
	if err != nil {
		return false, err
	}
	if len(changes) == 0 {
		fmt.Fprint(stdout, terminalMessage)
		return true, nil
	}
	if err := engine.SubmitBatch(ctx, changes); err != nil {
		return false, err
	}
	return false, nil
}

// startDetachedLoopWorkerCommand starts the long-running loop monitor outside the terminal.
func startDetachedLoopWorkerCommand(repo string) error {
	exe, err := currentExecutable()
	if err != nil {
		return fmt.Errorf("resolve specrelay flow executable failed: %w", err)
	}
	cmd := exec.Command(exe, flowWorkerCommandArgs("loop", "--worker", "--json")...)
	cmd.Dir = repo
	configureDetachedCommand(cmd)
	logPath, err := loopWorkerLogPath(repo)
	if err != nil {
		return err
	}
	return startDetachedWorkerCommand(cmd, logPath)
}

// loopWorkerActive reports whether a live background loop monitor already owns this repository.
func loopWorkerActive(repo string) (bool, error) {
	path, err := loopWorkerLockPath(repo)
	if err != nil {
		return false, err
	}
	status, err := lockInfoFileStatusForResource(path, runtime.GOOS, "loop")
	if err != nil {
		return false, err
	}
	if status == lockStatusUnknown {
		return false, fmt.Errorf("loop lock is uncertain; retry later or inspect %s manually", path)
	}
	return status == lockStatusActive, nil
}

// acquireLoopWorkerLock creates the repo-level lock for one detached loop monitor.
func acquireLoopWorkerLock(repo string) (func(), bool, error) {
	path, err := loopWorkerLockPath(repo)
	if err != nil {
		return nil, false, err
	}
	unlock, acquired, status, err := acquireOwnedLock(path, "loop", runtime.GOOS)
	if err != nil {
		return nil, false, err
	}
	if !acquired && status == lockStatusActive {
		return nil, false, nil
	}
	if !acquired && status == lockStatusUnknown {
		return nil, false, fmt.Errorf("loop lock is uncertain; retry later or inspect %s manually", path)
	}
	if !acquired {
		return nil, false, fmt.Errorf("failed to acquire loop lock")
	}
	return unlock, true, nil
}

// loopWorkerLockPath returns the repository-scoped monitor lock file.
func loopWorkerLockPath(repo string) (string, error) {
	base, err := repoRuntimeDir(repo)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "loop.lock"), nil
}

type loopDecisionStatus string

const (
	loopDecisionRunning loopDecisionStatus = "running"
	loopDecisionDone    loopDecisionStatus = "done"
	loopDecisionFailed  loopDecisionStatus = "failed"
	loopDecisionStopped loopDecisionStatus = "stopped"
)

const staleRunWorkerHeartbeatReason = "current run worker heartbeat is stale"

type loopDecision struct {
	Status loopDecisionStatus
	Reason string
}

// assessBatchForLoop decides whether a batch can still advance without intervention.
func assessBatchForLoop(repo string, batch BatchState) loopDecision {
	switch batch.Status {
	case batchStatusDone:
		return loopDecision{Status: loopDecisionDone}
	case batchStatusAborted:
		return loopDecision{Status: loopDecisionStopped}
	case batchStatusFailed:
		return loopDecision{Status: loopDecisionFailed, Reason: stoppedBatchReason(repo, batch)}
	case batchStatusArchived:
		return loopDecision{Status: loopDecisionStopped}
	}
	if batch.Status != batchStatusRunning {
		return loopDecision{Status: loopDecisionFailed, Reason: "unknown batch status " + batch.Status}
	}
	now := time.Now()
	if batch.CurrentIndex >= len(batch.Changes) {
		if err := markBatchDone(repo, batch.BatchID); err != nil {
			return loopDecision{Status: loopDecisionFailed, Reason: err.Error()}
		}
		return loopDecision{Status: loopDecisionDone}
	}
	runID := restartBatchCurrentRunID(batch)
	if runID == "" {
		if pendingBatchStartExpired(repo, batch, now) {
			return loopDecision{Status: loopDecisionFailed, Reason: "batch worker did not create the current run"}
		}
		return loopDecision{Status: loopDecisionRunning}
	}
	state, err := loadState(repo, runID)
	if err != nil {
		return loopDecision{Status: loopDecisionFailed, Reason: err.Error()}
	}
	if isBatchTerminalState(state) {
		return loopDecision{Status: loopDecisionFailed, Reason: humanRunFailureSummary(state, state.ChangeName)}
	}
	if stopped, reason := runningBatchRunCannotAdvance(repo, batch, state, now); stopped {
		return loopDecision{Status: loopDecisionFailed, Reason: reason}
	}
	return loopDecision{Status: loopDecisionRunning}
}

// continuationChanges keeps the unfinished active tail from the failed batch queue.
func continuationChanges(repo string, batch BatchState) ([]Change, error) {
	active, err := ListChanges(repo)
	if err != nil {
		return nil, err
	}
	activeSet := map[string]bool{}
	for _, change := range active {
		activeSet[change.Name] = true
	}
	start := batch.CurrentIndex
	if start < 0 {
		start = 0
	}
	if start > len(batch.Changes) {
		start = len(batch.Changes)
	}
	var changes []Change
	for _, name := range batch.Changes[start:] {
		if activeSet[name] {
			changes = append(changes, Change{Name: name, Path: changePath(repo, name)})
		}
	}
	return changes, nil
}

// markBatchDone fixes a fully advanced running batch whose worker exited before final save.
func markBatchDone(repo, batchID string) error {
	return withBatchState(repo, batchID, func(batch *BatchState) error {
		if batch.CurrentIndex >= len(batch.Changes) && batch.Status == batchStatusRunning {
			batch.Status = batchStatusDone
			batch.CurrentIndex = len(batch.Changes)
		}
		return nil
	})
}

// pendingBatchStartExpired reports a batch worker that never created its first run.
func pendingBatchStartExpired(repo string, batch BatchState, now time.Time) bool {
	info, err := os.Stat(filepath.Join(batchDir(repo, batch.BatchID), "state.json"))
	if err != nil {
		return false
	}
	return now.Sub(info.ModTime()) > loopPendingStartGrace
}

// runningRunCannotAdvance catches orphaned running state that has no live owner.
func runningRunCannotAdvance(repo string, state State, now time.Time) (bool, string) {
	if state.Status != statusRunning || state.RunID == "" {
		return false, ""
	}
	status, err := lockFileStatus(repo, state.RunID, runtime.GOOS)
	if err != nil {
		return false, ""
	}
	switch status {
	case lockStatusActive:
		if workerHeartbeatExpired(state.Worker, now) {
			return true, staleRunWorkerHeartbeatReason
		}
	case lockStatusStale:
		return true, "current run lock is stale"
	case lockStatusNone:
		info, err := os.Stat(filepath.Join(runDir(repo, state.RunID), "state.json"))
		if err == nil && now.Sub(info.ModTime()) > loopPendingStartGrace {
			return true, "current run has no active lock"
		}
	}
	return false, ""
}

// runningBatchRunCannotAdvance keeps a batch-owned run alive while the batch
// worker is still heartbeating, even if the run heartbeat is temporarily stale.
func runningBatchRunCannotAdvance(repo string, batch BatchState, state State, now time.Time) (bool, string) {
	stopped, reason := runningRunCannotAdvance(repo, state, now)
	if !stopped || reason != staleRunWorkerHeartbeatReason {
		return stopped, reason
	}
	if batchWorkerOwnsCurrentRun(batch, state, now) {
		return false, ""
	}
	return stopped, reason
}

// batchWorkerOwnsCurrentRun reports whether the running batch worker still owns
// the current run and can continue refreshing it across stage boundaries.
func batchWorkerOwnsCurrentRun(batch BatchState, state State, now time.Time) bool {
	if batch.Status != batchStatusRunning || state.BatchID == "" || state.BatchID != batch.BatchID {
		return false
	}
	if batch.CurrentIndex < 0 || batch.CurrentIndex >= len(batch.Changes) {
		return false
	}
	if batch.RunIDs[batch.Changes[batch.CurrentIndex]] != state.RunID {
		return false
	}
	return workerHeartbeatFresh(batch.Worker, now)
}

// markBatchFailedForLoop records a monitor-detected failure before archival.
func markBatchFailedForLoop(repo string, batch BatchState, reason string) error {
	return withBatchState(repo, batch.BatchID, func(current *BatchState) error {
		if current.Status != batchStatusRunning {
			return nil
		}
		current.Status = batchStatusFailed
		current.FailedChange = ""
		if current.CurrentIndex >= 0 && current.CurrentIndex < len(current.Changes) {
			current.FailedChange = current.Changes[current.CurrentIndex]
			current.FailedRunID = current.RunIDs[current.FailedChange]
		}
		if reason != "" {
			current.Error = reason
		}
		return nil
	})
}

// markStaleRunForLoop records a monitor-confirmed orphan before batch archival.
func markStaleRunForLoop(repo, runID, reason string) error {
	if runID == "" {
		return nil
	}
	state, err := loadState(repo, runID)
	if err != nil {
		return err
	}
	if state.Status != statusRunning {
		return nil
	}
	status, err := lockFileStatus(repo, runID, runtime.GOOS)
	if err != nil {
		return err
	}
	if status != lockStatusStale {
		return fmt.Errorf("run %s:  lock state changed; refusing reclamation", runID)
	}
	return mergeState(repo, runID, func(state *State) {
		if state.Status == statusRunning {
			state.Status = statusStale
			state.Error = reason
		}
	})
}

// failedRunIDForBatch returns the run that caused a failed batch to stop.
func failedRunIDForBatch(batch BatchState) string {
	if batch.FailedRunID != "" {
		return batch.FailedRunID
	}
	if batch.CurrentIndex >= 0 && batch.CurrentIndex < len(batch.Changes) {
		return batch.RunIDs[batch.Changes[batch.CurrentIndex]]
	}
	return ""
}

// isArchivableBatch reports whether archive can hide a failed batch from active prompts.
func isArchivableBatch(batch BatchState) bool {
	return batch.Status == batchStatusFailed
}

// isArchivableRun reports whether a run stopped because the workflow could not proceed.
func isArchivableRun(state State) bool {
	switch state.Status {
	case statusFailed, statusStale, statusInterrupted, statusBlocked, statusValidationBlocked, statusAcceptanceContractBlocked:
		return true
	default:
		return state.Stage == statusBlocked || state.Stage == statusValidationBlocked || state.Stage == statusAcceptanceContractBlocked
	}
}

// clearArchiveLock refuses to archive a run that may still be owned by a live worker.
func clearArchiveLock(repo, runID string) error {
	status, err := clearInactiveRunLock(repo, runID, runtime.GOOS, false)
	if err != nil {
		return err
	}
	switch status {
	case lockStatusActive:
		return newRunLockedError(runID)
	case lockStatusUnknown:
		return fmt.Errorf("run %s has an uncertain lock; cannot archive", runID)
	}
	return nil
}

type alreadyArchivedRunError struct {
	runID string
}

// Error returns a stable message for idempotent batch archiving.
func (err alreadyArchivedRunError) Error() string {
	return "workflow " + err.runID + " archived"
}

// isAlreadyArchivedRunError lets batch archive tolerate a run archived by a previous attempt.
func isAlreadyArchivedRunError(err error) bool {
	_, ok := err.(alreadyArchivedRunError)
	return ok
}
