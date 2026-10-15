// Package acceptance rejects renamed text, placeholders, and unreadable artifacts from final evidence.
package acceptance

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	minimumReviewableVideoBytes  = 512
	minimumReviewableImageWidth  = 64
	minimumReviewableImageHeight = 36
	maxTextEvidenceProbeBytes    = 1 << 20
)

// validateReviewableEvidenceFile checks that an artifact's bytes match its declared review purpose.
func validateReviewableEvidenceFile(path string, evidence Evidence) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	switch evidence.Kind {
	case "demo_video":
		return validateReviewableVideo(path, info.Size())
	case "screenshot":
		return validateReviewableScreenshot(path)
	case "trace":
		return validateReviewableTrace(path)
	case "runtime_log", "console":
		return validateReviewableText(path)
	case "network", "state_snapshot":
		return validateReviewableJSON(path)
	default:
		if info.Size() < 64 {
			return fmt.Errorf("evidence is too short for a reviewer to assess the user-visible result")
		}
		return nil
	}
}

// validateReviewableVideo rejects obvious placeholders without imposing a video container format.
func validateReviewableVideo(path string, size int64) error {
	if size < minimumReviewableVideoBytes {
		return fmt.Errorf("demo_video is smaller than %d bytes and is not a reviewable video", minimumReviewableVideoBytes)
	}
	probe, err := readEvidenceContainerProbe(path, size)
	if err != nil {
		return err
	}
	if utf8.Valid(probe) && !bytes.ContainsRune(probe, '\x00') && strings.TrimSpace(string(probe)) != "" {
		return fmt.Errorf("demo_video contains only text and cannot replace a playable video")
	}
	return nil
}

// readEvidenceContainerProbe reads the beginning and end of a media file without loading an arbitrarily large artifact.
func readEvidenceContainerProbe(path string, size int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	const window int64 = 4 << 20
	if size <= window*2 {
		return io.ReadAll(file)
	}
	head := make([]byte, window)
	if _, err := io.ReadFull(file, head); err != nil {
		return nil, err
	}
	tail := make([]byte, window)
	if _, err := file.ReadAt(tail, size-window); err != nil && err != io.EOF {
		return nil, err
	}
	return append(head, tail...), nil
}

// validateReviewableScreenshot decodes image metadata and rejects tiny placeholders.
func validateReviewableScreenshot(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return fmt.Errorf("screenshot cannot be decoded; renaming text does not create an image: %w", err)
	}
	if config.Width < minimumReviewableImageWidth || config.Height < minimumReviewableImageHeight {
		return fmt.Errorf(
			"screenshot dimensions %dx%d are too small to show user-visible results",
			config.Width,
			config.Height,
		)
	}
	return nil
}

// validateReviewableTrace requires a real, nonempty ZIP trace archive.
func validateReviewableTrace(path string) error {
	trace, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("trace is not a readable ZIP archive: %w", err)
	}
	defer trace.Close()
	if len(trace.File) == 0 {
		return fmt.Errorf("trace archive is empty")
	}
	return nil
}

// validateReviewableText accepts explanatory runtime output but rejects command-only or key=value-only records.
func validateReviewableText(path string) error {
	data, err := readTextEvidence(path)
	if err != nil {
		return err
	}
	lines := nonemptyEvidenceLines(string(data))
	if len(lines) < 3 {
		return fmt.Errorf("text evidence needs at least three lines describing the scenario, actions, and visible outcome")
	}
	allMetadata := true
	for _, line := range lines {
		if !keyValueOnlyLine.MatchString(line) && !commandOnlyText.MatchString(line) && !weakReviewerText.MatchString(line) {
			allMetadata = false
			break
		}
	}
	if allMetadata {
		return fmt.Errorf("text evidence contains only commands, statuses, or key=value pairs rather than user-visible results")
	}
	return nil
}

// validateReviewableJSON requires structured runtime facts instead of a renamed text file.
func validateReviewableJSON(path string) error {
	data, err := readTextEvidence(path)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(data)) < 64 || !json.Valid(data) {
		return fmt.Errorf("structured evidence must be valid JSON with actual business fields")
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	switch typed := value.(type) {
	case map[string]any:
		if len(typed) == 0 {
			return fmt.Errorf("structured evidence must not be an empty object")
		}
	case []any:
		if len(typed) == 0 {
			return fmt.Errorf("structured evidence must not be an empty array")
		}
	default:
		return fmt.Errorf("structured evidence must be an object or array")
	}
	return nil
}

// readTextEvidence reads bounded UTF-8 evidence for human-reviewability checks.
func readTextEvidence(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxTextEvidenceProbeBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxTextEvidenceProbeBytes {
		return nil, fmt.Errorf("text evidence exceeds 1 MiB; retain the key facts a reviewer needs")
	}
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("text evidence must be directly readable UTF-8")
	}
	return data, nil
}

// nonemptyEvidenceLines returns trimmed, nonempty lines from one human-readable artifact.
func nonemptyEvidenceLines(text string) []string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

// validateDeliveryComparisonArtifacts proves repair before/after references are different artifact bytes.
func validateDeliveryComparisonArtifacts(projectRoot string, contract Contract) error {
	if contract.DeliveryReport == nil {
		return nil
	}
	archiveByID := submissionArchivePaths(contract.SubmissionEvidence)
	for i, scenario := range contract.DeliveryReport.Scenarios {
		if scenario.Comparison == nil {
			continue
		}
		beforePath := filepath.Join(projectRoot, filepath.FromSlash(archiveByID[scenario.Comparison.BeforeEvidenceID]))
		afterPath := filepath.Join(projectRoot, filepath.FromSlash(archiveByID[scenario.Comparison.AfterEvidenceID]))
		beforeHash, err := evidenceFileHash(beforePath)
		if err != nil {
			return fmt.Errorf("delivery_report.scenarios[%d] cannot read before evidence: %w", i, err)
		}
		afterHash, err := evidenceFileHash(afterPath)
		if err != nil {
			return fmt.Errorf("delivery_report.scenarios[%d] cannot read after evidence: %w", i, err)
		}
		if beforeHash == afterHash {
			return fmt.Errorf("delivery_report.scenarios[%d] has identical before and after evidence content", i)
		}
	}
	return nil
}

// evidenceFileHash hashes one regular evidence file for before/after comparison.
func evidenceFileHash(path string) ([sha256.Size]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return [sha256.Size]byte{}, err
	}
	var result [sha256.Size]byte
	copy(result[:], hash.Sum(nil))
	return result, nil
}
