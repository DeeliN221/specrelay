// Package app verifies that promoted acceptance evidence is part of the durable archive commit.
package app

import (
	"context"
	"errors"
	"github.com/DeeliN221/specrelay/internal/i18n"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// verifyQualityLoopArchivedEvidenceCommit requires a real, tracked evidence package in the proposal archive commit.
func verifyQualityLoopArchivedEvidenceCommit(repo string, state State) error {
	payloadDir, err := qualityLoopArchivePayloadDir(repo, state.ChangeName)
	if err != nil {
		return err
	}
	payloadRelative, err := filepath.Rel(repo, payloadDir)
	if err != nil {
		return err
	}
	payloadRelative = filepath.ToSlash(payloadRelative)
	if !strings.HasPrefix(payloadRelative, "docs/changes/archive/") {
		return i18n.Errorf("archive submission gate cannot find the archived proposal directory: %s", payloadRelative)
	}

	packageRelative := filepath.ToSlash(filepath.Join("tests", "evidence", "proposals", state.ChangeName))
	packageDir := filepath.Join(repo, filepath.FromSlash(packageRelative))
	files, err := qualityLoopEvidencePackageFiles(repo, packageDir)
	if err != nil {
		return err
	}
	for _, required := range []string{"README.md", "DELIVERY.md", "manifest.json", "result.json"} {
		if !containsArchiveEvidenceFile(files, packageRelative+"/"+required) {
			return i18n.Errorf("archive bundle is missing %s", required)
		}
	}
	for _, path := range files {
		ignored, checkErr := gitPathIgnored(repo, path)
		if checkErr != nil {
			return checkErr
		}
		if ignored {
			return i18n.Errorf("archive evidence is ignored by Git: %s", path)
		}
		if err := gitPathTracked(repo, path); err != nil {
			return i18n.Errorf("archive evidence is not in the Git index: %s: %w", path, err)
		}
	}
	if err := requireCleanArchiveCommitPaths(repo, packageRelative, payloadRelative); err != nil {
		return err
	}
	if strings.TrimSpace(state.DeliveryBaseHead) == "" {
		return nil
	}
	evidenceCommit, err := latestPathCommit(repo, packageRelative)
	if err != nil {
		return err
	}
	proposalCommit, err := latestPathCommit(repo, payloadRelative)
	if err != nil {
		return err
	}
	if evidenceCommit == "" || proposalCommit == "" || evidenceCommit != proposalCommit {
		return i18n.Errorf(
			"archive evidence and archived proposal are not in the same commit: evidence=%s proposal=%s",
			evidenceCommit,
			proposalCommit,
		)
	}
	deliveryHead, err := verifyQualityLoopDeliveryCommitRange(repo, state.DeliveryBaseHead)
	if err != nil {
		return err
	}
	if deliveryHead != "" {
		if err := requireCleanQualityLoopDelivery(repo, state.ChangeName); err != nil {
			return err
		}
	}
	if deliveryHead != "" && deliveryHead != proposalCommit {
		return i18n.Errorf(
			"archive delivery HEAD is not the proposal and evidence commit: head=%s archive=%s",
			deliveryHead,
			proposalCommit,
		)
	}
	return nil
}

// requireCleanQualityLoopDelivery proves implementation content is included in the sole delivery commit.
func requireCleanQualityLoopDelivery(repo, changeName string) error {
	_, status, err := gitSnapshot(repo)
	if err != nil {
		return err
	}
	var pending []string
	for path := range statusLineByPath(status) {
		if isUnrelatedChangePath(path, changeName) {
			continue
		}
		pending = append(pending, path)
	}
	pending = uniqueSortedPaths(pending)
	if len(pending) > 0 {
		return i18n.Errorf("archive uncommitted implementation content remains after the delivery commit: %s", strings.Join(pending, ", "))
	}
	return nil
}

// verifyQualityLoopDeliveryCommitRange requires one complete delivery commit for newly sealed runs.
func verifyQualityLoopDeliveryCommitRange(repo, deliveryBaseHead string) (string, error) {
	deliveryBaseHead = strings.TrimSpace(deliveryBaseHead)
	if deliveryBaseHead == "" {
		return "", nil
	}
	gitPath, err := resolveCommand("git")
	if err != nil {
		return "", err
	}
	ancestor := commandContext(context.Background(), gitPath, "merge-base", "--is-ancestor", deliveryBaseHead, "HEAD")
	ancestor.Dir = repo
	if err := ancestor.Run(); err != nil {
		return "", i18n.Errorf("archive delivery_base_head is not an ancestor of the current HEAD: %s", deliveryBaseHead)
	}
	countCmd := commandContext(context.Background(), gitPath, "rev-list", "--count", deliveryBaseHead+"..HEAD")
	countCmd.Dir = repo
	countOutput, err := countCmd.CombinedOutput()
	if err != nil {
		return "", i18n.Errorf("read archive delivery commit range failed: %w", err)
	}
	if count := strings.TrimSpace(string(countOutput)); count != "1" {
		return "", i18n.Errorf("archive requires exactly one complete delivery commit from delivery_base_head to HEAD; actual=%s", count)
	}
	headCmd := commandContext(context.Background(), gitPath, "rev-parse", "--verify", "HEAD^{commit}")
	headCmd.Dir = repo
	headOutput, err := headCmd.CombinedOutput()
	if err != nil {
		return "", i18n.Errorf("read archive delivery HEAD failed: %w", err)
	}
	return strings.TrimSpace(string(headOutput)), nil
}

// qualityLoopEvidencePackageFiles returns regular package files and rejects links or special entries.
func qualityLoopEvidencePackageFiles(repo, packageDir string) ([]string, error) {
	info, err := os.Lstat(packageDir)
	if err != nil {
		return nil, i18n.Errorf("archive  evidence bundle does not exist: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, i18n.Errorf("archive evidence bundle must be a real directory: %s", packageDir)
	}
	var files []string
	err = filepath.WalkDir(packageDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == packageDir {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return i18n.Errorf("archive evidence must not contain symbolic links: %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return i18n.Errorf("archive evidence must be a regular file: %s", path)
		}
		relative, relErr := filepath.Rel(repo, path)
		if relErr != nil {
			return relErr
		}
		files = append(files, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, i18n.Errorf("archive evidence bundle is empty: %s", packageDir)
	}
	return files, nil
}

// containsArchiveEvidenceFile reports whether the package contains one required relative file.
func containsArchiveEvidenceFile(files []string, required string) bool {
	for _, path := range files {
		if path == required {
			return true
		}
	}
	return false
}

// gitPathIgnored checks ignore rules even for an already tracked path.
func gitPathIgnored(repo, path string) (bool, error) {
	gitPath, err := resolveCommand("git")
	if err != nil {
		return false, err
	}
	cmd := commandContext(context.Background(), gitPath, "check-ignore", "--no-index", "-q", "--", path)
	cmd.Dir = repo
	err = cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, i18n.Errorf("git check-ignore %s failed: %w", path, err)
}

// gitPathTracked verifies that the current index contains the promoted evidence file.
func gitPathTracked(repo, path string) error {
	gitPath, err := resolveCommand("git")
	if err != nil {
		return err
	}
	cmd := commandContext(context.Background(), gitPath, "ls-files", "--error-unmatch", "--", path)
	cmd.Dir = repo
	if output, runErr := cmd.CombinedOutput(); runErr != nil {
		return i18n.Errorf("%s", strings.TrimSpace(string(output)))
	}
	return nil
}

// requireCleanArchiveCommitPaths rejects staged, unstaged, or untracked archive payload changes.
func requireCleanArchiveCommitPaths(repo string, paths ...string) error {
	gitPath, err := resolveCommand("git")
	if err != nil {
		return err
	}
	args := []string{"status", "--porcelain=v1", "--untracked-files=all", "--"}
	args = append(args, paths...)
	cmd := commandContext(context.Background(), gitPath, args...)
	cmd.Dir = repo
	output, err := cmd.CombinedOutput()
	if err != nil {
		return i18n.Errorf("read archive commit status failed: %w", err)
	}
	if detail := strings.TrimSpace(string(output)); detail != "" {
		return i18n.Errorf("archive evidence or archived proposal is uncommitted: %s", detail)
	}
	return nil
}

// latestPathCommit returns the newest commit that contains the named archive path.
func latestPathCommit(repo, path string) (string, error) {
	gitPath, err := resolveCommand("git")
	if err != nil {
		return "", err
	}
	cmd := commandContext(context.Background(), gitPath, "log", "-1", "--format=%H", "--", path)
	cmd.Dir = repo
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", i18n.Errorf("read archive path commit failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
