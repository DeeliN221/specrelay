// Package app centralizes main-stage artifact expectations for workflow retry gates.
package app

import (
	"fmt"
	"github.com/DeeliN221/specrelay/internal/i18n"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type stageArtifactExpectation struct {
	Path        string
	Description string
}

type stageArtifactResult struct {
	Repair Review
	Review Review
	QA     QA
}

// stageArtifactExpectation describes the durable output that proves one stage completed.
func (e *Engine) stageArtifactExpectation(state State) stageArtifactExpectation {
	base := runDir(e.Repo, state.RunID)
	switch {
	case state.Stage == "execution":
		return stageArtifactExpectation{
			Path:        filepath.Join(base, "state.json"),
			Description: "execution agent returned successfully; subsequent validation and acceptance gates independently verify delivery",
		}
	case strings.HasPrefix(state.Stage, "review_"):
		n := strings.TrimPrefix(state.Stage, "review_")
		return stageArtifactExpectation{
			Path:        filepath.Join(base, "review-"+n+".json"),
			Description: "review JSON schema and decision contract",
		}
	case strings.HasPrefix(state.Stage, "repair_"):
		n := strings.TrimPrefix(state.Stage, "repair_")
		return stageArtifactExpectation{
			Path:        filepath.Join(base, "repair-"+n+".json"),
			Description: "repair JSON schema, repair evidence, and decision contract",
		}
	case strings.HasPrefix(state.Stage, "audit_"):
		n := strings.TrimPrefix(state.Stage, "audit_")
		return stageArtifactExpectation{
			Path:        filepath.Join(base, "audit-"+n+".json"),
			Description: "pre_qa_audit JSON schema, complete self-review evidence, and decision contract",
		}
	case strings.HasPrefix(state.Stage, "targeted_repair_"):
		n := strings.TrimPrefix(state.Stage, "targeted_repair_")
		return stageArtifactExpectation{
			Path:        filepath.Join(base, "targeted-repair-"+n+".json"),
			Description: "qa_targeted_repair JSON schema, QA finding repair evidence, and decision contract",
		}
	case strings.HasPrefix(state.Stage, "fix_"):
		n := strings.TrimPrefix(state.Stage, "fix_")
		return stageArtifactExpectation{
			Path:        filepath.Join(base, "fix-"+n+"-summary.md"),
			Description: "fix stage progress record",
		}
	case strings.HasPrefix(state.Stage, "qa_"):
		n := strings.TrimPrefix(state.Stage, "qa_")
		return stageArtifactExpectation{
			Path:        filepath.Join(base, "qa-"+n+".json"),
			Description: "QA JSON schema, acceptance matrix, and helper input checks",
		}
	case state.Stage == "archive":
		return stageArtifactExpectation{
			Path:        filepath.Join(base, "delivery-summary.md"),
			Description: "delivery summary and docs/changes/archive/" + state.ChangeName + " archive directory",
		}
	default:
		return stageArtifactExpectation{Description: "current stage artifact"}
	}
}

// validateStageArtifact performs the single durable-output check for the active stage.
func (e *Engine) validateStageArtifact(state State) (stageArtifactResult, bool, error) {
	base := runDir(e.Repo, state.RunID)
	switch {
	case state.Stage == "execution":
		stageStatus := state.Stages[state.Stage]
		return stageArtifactResult{}, stageStatus == stageStatusAgentCompleted || stageStatus == "completed", nil
	case strings.HasPrefix(state.Stage, "review_"):
		iteration, err := stageIteration(state.Stage)
		if err != nil {
			return stageArtifactResult{}, false, err
		}
		review, err := ReadReview(filepath.Join(base, "review-"+strconv.Itoa(iteration)+".json"))
		if os.IsNotExist(err) {
			return stageArtifactResult{}, false, nil
		}
		if err != nil {
			return stageArtifactResult{}, false, err
		}
		return stageArtifactResult{Review: review}, true, nil
	case strings.HasPrefix(state.Stage, "repair_"):
		iteration, err := stageIteration(state.Stage)
		if err != nil {
			return stageArtifactResult{}, false, err
		}
		repair, err := ReadRepair(filepath.Join(base, "repair-"+strconv.Itoa(iteration)+".json"))
		if os.IsNotExist(err) {
			return stageArtifactResult{}, false, nil
		}
		if err != nil {
			return stageArtifactResult{}, false, err
		}
		return stageArtifactResult{Repair: repair}, true, nil
	case strings.HasPrefix(state.Stage, "audit_"):
		iteration, err := stageIteration(state.Stage)
		if err != nil {
			return stageArtifactResult{}, false, err
		}
		audit, err := ReadRepair(filepath.Join(base, "audit-"+strconv.Itoa(iteration)+".json"))
		if os.IsNotExist(err) {
			return stageArtifactResult{}, false, nil
		}
		if err != nil {
			return stageArtifactResult{}, false, err
		}
		return stageArtifactResult{Repair: audit}, true, nil
	case strings.HasPrefix(state.Stage, "targeted_repair_"):
		iteration, err := stageIteration(state.Stage)
		if err != nil {
			return stageArtifactResult{}, false, err
		}
		repair, err := ReadRepair(filepath.Join(base, "targeted-repair-"+strconv.Itoa(iteration)+".json"))
		if os.IsNotExist(err) {
			return stageArtifactResult{}, false, nil
		}
		if err != nil {
			return stageArtifactResult{}, false, err
		}
		return stageArtifactResult{Repair: repair}, true, nil
	case strings.HasPrefix(state.Stage, "fix_"):
		iteration, err := stageIteration(state.Stage)
		if err != nil {
			return stageArtifactResult{}, false, err
		}
		return stageArtifactResult{}, fileExists(filepath.Join(base, "fix-"+strconv.Itoa(iteration)+"-summary.md")), nil
	case strings.HasPrefix(state.Stage, "qa_"):
		iteration, err := stageIteration(state.Stage)
		if err != nil {
			return stageArtifactResult{}, false, err
		}
		qa, err := ReadQA(filepath.Join(base, "qa-"+strconv.Itoa(iteration)+".json"))
		if os.IsNotExist(err) {
			return stageArtifactResult{}, false, nil
		}
		if err != nil {
			return stageArtifactResult{}, false, err
		}
		acceptance, err := readAcceptanceForState(e.Repo, state)
		if err != nil {
			return stageArtifactResult{}, false, err
		}
		if err := ValidateQAAgainstAcceptance(qa, acceptance); err != nil {
			return stageArtifactResult{}, false, err
		}
		return stageArtifactResult{QA: qa}, true, nil
	case state.Stage == "archive":
		if !fileExists(filepath.Join(base, "delivery-summary.md")) || !archiveExists(e.Repo, state.ChangeName) {
			return stageArtifactResult{}, false, nil
		}
		if err := e.validateArchiveReadiness(state); err != nil {
			return stageArtifactResult{}, false, err
		}
		return stageArtifactResult{}, true, nil
	}
	return stageArtifactResult{}, false, i18n.Errorf("unknown stage %q", state.Stage)
}

// artifactDone reports whether the current stage already has a valid durable output.
func (e *Engine) artifactDone(state State) (bool, error) {
	_, done, err := e.validateStageArtifact(state)
	return done, err
}

// checkStageArtifactGate converts missing or invalid stage output into a same-stage retry error.
func (e *Engine) checkStageArtifactGate(state State) (bool, error) {
	_, done, err := e.validateStageArtifact(state)
	if err != nil {
		return false, e.stageArtifactGateError(state, err)
	}
	if !done {
		return false, e.stageArtifactGateError(state, i18n.Errorf("%s stage artifact is incomplete", state.Stage))
	}
	return true, nil
}

// stageArtifactGateError adds stage and target artifact context to retriable gate failures.
func (e *Engine) stageArtifactGateError(state State, failure error) error {
	expect := e.stageArtifactExpectation(state)
	reason := failure.Error()
	if expect.Path != "" {
		reason = fmt.Sprintf("%s; stage=%s; artifact=%s; expectation=%s", reason, state.Stage, expect.Path, expect.Description)
	} else {
		reason = fmt.Sprintf("%s; stage=%s; expectation=%s", reason, state.Stage, expect.Description)
	}
	return stageArtifactGateError{Reason: reason, Cause: failure}
}
