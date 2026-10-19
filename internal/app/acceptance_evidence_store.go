// Package app seals workflow acceptance evidence and promotes the final proof into the repository archive.
package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/DeeliN221/specrelay/internal/acceptance"
	"github.com/DeeliN221/specrelay/internal/i18n"
)

const acceptanceEvidenceDirectory = "evidence"

// acceptanceRunSealedDir resolves the immutable state directory for one workflow attempt.
func acceptanceRunSealedDir(repo string, binding acceptanceRunBinding) (string, error) {
	if !safeAcceptancePathSegment(binding.RunID) || !safeAcceptancePathSegment(binding.Stage) || binding.Attempt < 1 {
		return "", i18n.Errorf("acceptance evidence binding invalid: run=%q stage=%q attempt=%d", binding.RunID, binding.Stage, binding.Attempt)
	}
	return filepath.Join(
		runDir(repo, binding.RunID),
		acceptanceEvidenceDirectory,
		binding.Stage,
		fmt.Sprintf("attempt-%d", binding.Attempt),
	), nil
}

// sealAcceptanceRunArtifacts snapshots temporary logs and required evidence before later stages can overwrite them.
func sealAcceptanceRunArtifacts(repo string, binding acceptanceRunBinding, result *AcceptanceRunResult) error {
	if result == nil {
		return i18n.Errorf("acceptance evidence result must not be empty")
	}
	sealedDir, err := acceptanceRunSealedDir(repo, binding)
	if err != nil {
		return err
	}
	result.ResultPath = filepath.Join(sealedDir, "result.json")
	var sealErrors []error
	for index := range result.Tests {
		item := &result.Tests[index]
		data, readErr := readAcceptanceArtifactFile(repo, item.LogPath)
		if readErr != nil {
			sealErrors = append(sealErrors, i18n.Errorf("read temporary log for required_test %q failed: %w", item.ID, readErr))
			continue
		}
		target := filepath.Join(sealedDir, "tests", acceptanceLogName(index, item.ID))
		if writeErr := writeImmutableEvidenceFile(target, data, 0o600); writeErr != nil {
			sealErrors = append(sealErrors, i18n.Errorf("seal log for required_test %q failed: %w", item.ID, writeErr))
			continue
		}
		item.SealedLogPath = target
	}
	for index := range result.Evidence {
		item := &result.Evidence[index]
		if item.Status != "present" {
			continue
		}
		data, readErr := readRepositoryEvidence(repo, item.Path)
		if readErr != nil {
			sealErrors = append(sealErrors, i18n.Errorf("read required_evidence %q failed: %w", item.ID, readErr))
			continue
		}
		target := filepath.Join(sealedDir, "required", acceptanceEvidenceName(index, item.ID, item.Path))
		if writeErr := writeImmutableEvidenceFile(target, data, 0o600); writeErr != nil {
			sealErrors = append(sealErrors, i18n.Errorf("seal required_evidence %q failed: %w", item.ID, writeErr))
			continue
		}
		rawHash, progressHash, _, hashErr := qualityHashEvidenceFilePair(target, item.Kind)
		if hashErr != nil {
			sealErrors = append(sealErrors, i18n.Errorf("verify sealed copy of required_evidence %q failed: %w", item.ID, hashErr))
			continue
		}
		item.SealedPath = target
		item.ContentHash = rawHash
		item.ProgressHash = progressHash
	}
	return errors.Join(sealErrors...)
}

// acceptanceEvidenceName gives each required artifact a stable collision-free filename.
func acceptanceEvidenceName(index int, id, sourcePath string) string {
	base := safeAcceptanceLogName(filepath.Base(filepath.FromSlash(sourcePath)))
	sum := qualityHashStrings(id, sourcePath)
	if len(sum) > 12 {
		sum = sum[:12]
	}
	return fmt.Sprintf("%03d-%s-%s", index+1, safeAcceptanceLogName(id), base+"-"+sum)
}

// readRepositoryEvidence rechecks repository containment and file identity while taking the snapshot.
func readRepositoryEvidence(repo, relative string) ([]byte, error) {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(relative)))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, i18n.Errorf("evidence path is invalid: %q", relative)
	}
	resolved, safe, err := qualityResolveEvidencePath(repo, filepath.Join(repo, clean))
	if err != nil {
		return nil, err
	}
	if !safe {
		return nil, i18n.Errorf("evidence escapes the repository: %q", relative)
	}
	return readRegularEvidenceFile(resolved)
}

// readRegularEvidenceFile reads one non-linked regular file after an identity check.
func readRegularEvidenceFile(path string) ([]byte, error) {
	file, _, err := qualityOpenEvidenceFile(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

// writeImmutableEvidenceFile creates an artifact once and only accepts byte-identical retries.
func writeImmutableEvidenceFile(path string, data []byte, mode os.FileMode) error {
	if err := ensureRealEvidenceDirectory(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if os.IsExist(err) {
		existing, readErr := readRegularEvidenceFile(path)
		if readErr != nil {
			return readErr
		}
		if !bytes.Equal(existing, data) {
			return i18n.Errorf("immutable acceptance evidence already exists with different content: %s", path)
		}
		return nil
	}
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// ensureRealEvidenceDirectory creates a directory path without accepting linked components.
func ensureRealEvidenceDirectory(path string, mode os.FileMode) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	volume := filepath.VolumeName(absolute)
	current := string(filepath.Separator)
	remainder := strings.TrimPrefix(absolute, volume+string(filepath.Separator))
	if volume != "" {
		current = volume + string(filepath.Separator)
	}
	for _, segment := range strings.Split(remainder, string(filepath.Separator)) {
		if segment == "" {
			continue
		}
		current = filepath.Join(current, segment)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) {
			if mkdirErr := os.Mkdir(current, mode); mkdirErr != nil && !os.IsExist(mkdirErr) {
				return mkdirErr
			}
			info, statErr = os.Lstat(current)
		}
		if statErr != nil {
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return i18n.Errorf("acceptance evidence directory must be a real directory: %s", current)
		}
	}
	return nil
}

// writeSealedAcceptanceRunResult commits the result metadata once after every evidence path is bound.
func writeSealedAcceptanceRunResult(result AcceptanceRunResult) error {
	if !filepath.IsAbs(result.ResultPath) {
		return i18n.Errorf("sealed acceptance result path must be absolute: %s", result.ResultPath)
	}
	data, err := marshalAcceptanceRunResult(result)
	if err != nil {
		return err
	}
	return writeImmutableEvidenceFile(result.ResultPath, data, 0o600)
}

// readSealedAcceptanceArtifact reads a state artifact without following a final link.
func readSealedAcceptanceArtifact(path string) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, i18n.Errorf("sealed acceptance artifact path must be absolute: %s", path)
	}
	return readRegularEvidenceFile(path)
}

// acceptanceEvidenceManifest records portable provenance for one tracked proposal evidence package.
type acceptanceEvidenceManifest struct {
	Version      int                               `json:"version"`
	Change       string                            `json:"change"`
	RunID        string                            `json:"run_id"`
	Stage        string                            `json:"stage"`
	Attempt      int                               `json:"attempt"`
	ContractHash string                            `json:"contract_hash"`
	TestsHash    string                            `json:"tests_hash"`
	EvidenceHash string                            `json:"evidence_hash"`
	Evidence     []acceptanceEvidenceManifestEntry `json:"evidence"`
}

// acceptanceEvidenceManifestEntry binds a declared temporary source to its sealed and tracked hashes.
type acceptanceEvidenceManifestEntry struct {
	EvidenceID  string `json:"evidence_id"`
	SourcePath  string `json:"source_path"`
	ArchivePath string `json:"archive_path"`
	ContentHash string `json:"content_hash"`
}

// promoteQualityAcceptanceEvidence copies mapped final evidence from the sealed checkpoint into the tracked archive bundle.
func promoteQualityAcceptanceEvidence(repo string, state State, checkpoint string) error {
	result, err := verifyQualityAcceptanceCheckpoint(repo, state, checkpoint)
	if err != nil {
		return err
	}
	contract, _, err := readVerifiedRunAcceptance(repo, state)
	if err != nil {
		return err
	}
	if contract.SubmissionEvidence == nil {
		return i18n.Errorf("proposal %q must declare submission_evidence before archiving", state.ChangeName)
	}
	observations, err := qualityDeliveryObservations(repo, state, contract)
	if err != nil {
		return err
	}
	deliveryReport, err := acceptance.RenderDeliveryReport(contract, state.ChangeName, observations)
	if err != nil {
		return err
	}
	targetRelative := filepath.ToSlash(filepath.Join("tests", "evidence", "proposals", state.ChangeName))
	targetDir, err := ensureAcceptanceArtifactDirectory(repo, targetRelative)
	if err != nil {
		return err
	}
	copies := map[string]string{
		result.ResultPath: filepath.Join(targetDir, "result.json"),
	}
	for _, item := range result.Tests {
		copies[item.SealedLogPath] = filepath.Join(targetDir, "tests", filepath.Base(item.SealedLogPath))
	}
	evidenceByID := make(map[string]AcceptanceRunEvidenceResult, len(result.Evidence))
	for _, item := range result.Evidence {
		evidenceByID[item.ID] = item
	}
	manifest := acceptanceEvidenceManifest{
		Version:      1,
		Change:       state.ChangeName,
		RunID:        result.RunID,
		Stage:        result.Stage,
		Attempt:      result.Attempt,
		ContractHash: result.ContractHash,
		TestsHash:    result.TestsHash,
		EvidenceHash: result.EvidenceHash,
		Evidence:     make([]acceptanceEvidenceManifestEntry, 0, len(contract.SubmissionEvidence)),
	}
	for _, mapping := range contract.SubmissionEvidence {
		item, ok := evidenceByID[mapping.EvidenceID]
		if !ok || item.Path != mapping.SourcePath || item.SealedPath == "" || item.Status != "present" {
			return i18n.Errorf("submission_evidence %q is missing a matching sealed copy", mapping.EvidenceID)
		}
		target, pathErr := acceptanceArtifactFilePath(repo, mapping.ArchivePath, true)
		if pathErr != nil {
			return pathErr
		}
		copies[item.SealedPath] = target
		manifest.Evidence = append(manifest.Evidence, acceptanceEvidenceManifestEntry{
			EvidenceID:  mapping.EvidenceID,
			SourcePath:  mapping.SourcePath,
			ArchivePath: mapping.ArchivePath,
			ContentHash: item.ContentHash,
		})
	}
	for source, target := range copies {
		if strings.TrimSpace(source) == "" {
			return i18n.Errorf("acceptance archive bundle is missing the sealed source path")
		}
		data, readErr := readSealedAcceptanceArtifact(source)
		if readErr != nil {
			return readErr
		}
		if writeErr := writePromotedAcceptanceFile(repo, target, data); writeErr != nil {
			return writeErr
		}
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := writePromotedAcceptanceFile(repo, filepath.Join(targetDir, "manifest.json"), append(manifestData, '\n')); err != nil {
		return err
	}
	if err := writePromotedAcceptanceFile(repo, filepath.Join(targetDir, "DELIVERY.md"), deliveryReport); err != nil {
		return err
	}
	readme := fmt.Sprintf(
		"# %s Review guide\n\nStart with the [Delivery report](DELIVERY.md), follow its user scenarios, and inspect the linked screenshots, videos, or business logs.\n\n`manifest.json`, `result.json`, and `tests/` provide the technical audit trail.\n",
		state.ChangeName,
	)
	if err := writePromotedAcceptanceFile(repo, filepath.Join(targetDir, "README.md"), []byte(readme)); err != nil {
		return err
	}
	return acceptance.ValidateSubmissionEvidenceForChange(repo, contract, state.ChangeName)
}

// qualityDeliveryObservations reads the final clean QA observations used in the reviewer report.
func qualityDeliveryObservations(repo string, state State, contract acceptance.Contract) ([]acceptance.DeliveryObservation, error) {
	if contract.DeliveryReport == nil {
		return nil, i18n.Errorf("proposal %q is missing a reviewer-facing delivery_report", state.ChangeName)
	}
	iteration := latestCompletedQAIteration(state)
	if iteration < 1 {
		return nil, i18n.Errorf("proposal %q has no final clean QA; cannot generate a delivery report", state.ChangeName)
	}
	qaPath := filepath.Join(runDir(repo, state.RunID), fmt.Sprintf("qa-%d.json", iteration))
	qa, err := ReadQA(qaPath)
	if err != nil {
		return nil, i18n.Errorf("read final QA delivery observations failed: %w", err)
	}
	if err := ValidateQAAgainstAcceptance(qa, contract); err != nil {
		return nil, err
	}
	if QANeedsFix(qa) {
		return nil, i18n.Errorf("final QA has not passed; cannot generate a delivery report")
	}
	observations := make([]acceptance.DeliveryObservation, 0, len(qa.UserAcceptance))
	for _, result := range qa.UserAcceptance {
		observations = append(observations, acceptance.DeliveryObservation{
			ScenarioID: result.ScenarioID,
			Observed:   result.Observed,
		})
	}
	return observations, nil
}

// writePromotedAcceptanceFile atomically refreshes the uncommitted final package from immutable state evidence.
func writePromotedAcceptanceFile(repo, target string, data []byte) error {
	relative, err := filepath.Rel(repo, target)
	if err != nil {
		return err
	}
	return writeAcceptanceArtifactFile(repo, filepath.ToSlash(relative), data)
}
