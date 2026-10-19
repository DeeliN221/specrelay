// Package specrelaycli implements standalone specrelay change archiving.
package specrelaycli

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/DeeliN221/specrelay/internal/acceptance"
	"github.com/DeeliN221/specrelay/internal/i18n"
)

func (c *cli) archiveCmd(args []string) error {
	// archiveCmd performs deterministic file moves; agents merge archived specs afterward.
	if hasHelp(args) {
		i18n.Fprintln(c.out, "Usage:specrelay archive <change> --yes")
		return nil
	}
	if len(args) == 0 {
		return i18n.New("Usage:specrelay archive <change> --yes")
	}
	change := args[0]
	if !hasArg(args[1:], "--yes") {
		return i18n.New("archiving requires an explicit --yes")
	}
	root, err := stateRoot()
	if err != nil {
		return err
	}
	result := validateChange(root, change)
	if !result.Valid {
		return i18n.Errorf("%s: %s", change, strings.Join(result.Errors, "; "))
	}
	changeDir := filepath.Join(root, "changes", change)
	contract, err := acceptance.Read(filepath.Join(changeDir, "acceptance.json"))
	if err != nil {
		return i18n.Errorf("%s: invalid acceptance.json: %w", change, err)
	}
	if err := acceptance.ValidateSubmissionEvidenceForChange(filepath.Dir(root), contract, change); err != nil {
		return i18n.Errorf("%s: invalid submission evidence bundle: %w", change, err)
	}
	archiveDir := filepath.Join(root, "changes", "archive", change)
	if _, err := os.Stat(archiveDir); err == nil {
		return i18n.Errorf("archive destination already exists: %s", archiveDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(archiveDir), 0o755); err != nil {
		return err
	}
	if err := os.Rename(changeDir, archiveDir); err != nil {
		return err
	}
	if err := rewriteArchivedChangeReferences(archiveDir, change); err != nil {
		// Restore the active proposal when its contract cannot be rewritten, so a
		// failed archive never leaves an unusable half-migrated change behind.
		_ = os.Rename(archiveDir, changeDir)
		return err
	}
	i18n.Fprintf(c.out, "Archived to %s\n", archiveDir)
	return nil
}

func rewriteArchivedChangeReferences(archiveDir, change string) error {
	// rewriteArchivedChangeReferences updates every UTF-8 text reference after
	// tests move from docs/changes/<change> into docs/changes/archive/<change>.
	oldPrefix := "docs/changes/" + change + "/"
	newPrefix := "docs/changes/archive/" + change + "/"
	testPrefix := newPrefix + "tests/"
	return filepath.WalkDir(archiveDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !utf8.Valid(data) {
			return nil
		}
		updated := strings.ReplaceAll(string(data), oldPrefix, newPrefix)
		if !strings.Contains(filepath.ToSlash(path), "/tests/") {
			updated = rewriteRelativeTestReferences(updated, testPrefix)
		}
		if updated == string(data) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(path, []byte(updated), info.Mode().Perm())
	})
}

var relativeTestsReference = regexp.MustCompile(`(^|[[:space:]\x60"'(\[：])tests/`)

func rewriteRelativeTestReferences(text, testPrefix string) string {
	// rewriteRelativeTestReferences expands proposal-local tests/ links in
	// archived documents while leaving long-term tests/specs/ links untouched.
	var out strings.Builder
	last := 0
	for _, match := range relativeTestsReference.FindAllStringIndex(text, -1) {
		start, end := match[0], match[1]
		out.WriteString(text[last:start])
		if strings.HasPrefix(text[end:], "specs/") {
			out.WriteString(text[start:end])
		} else {
			out.WriteString(text[start : end-len("tests/")])
			out.WriteString(testPrefix)
		}
		last = end
	}
	out.WriteString(text[last:])
	return out.String()
}
