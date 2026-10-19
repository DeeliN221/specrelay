// Package app tests archive location lookup for numeric and legacy dated proposal names.
package app

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestArchivedChangeDirsAcceptNumericAndLegacyNames(t *testing.T) {
	// Numeric archive names must resolve even without the legacy date prefix.
	repo := t.TempDir()
	archive := filepath.Join(repo, "docs", "changes", "archive")
	for _, name := range []string{"2-登录能力", "2026-05-01-2-登录能力", "12-登录能力", "not-a-proposal"} {
		if err := os.MkdirAll(filepath.Join(archive, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	matches, err := archivedChangeDirs(repo, "2-登录能力")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(matches)
	want := []string{
		filepath.Join(archive, "2-登录能力"),
		filepath.Join(archive, "2026-05-01-2-登录能力"),
	}
	if len(matches) != len(want) {
		t.Fatalf("archivedChangeDirs = %v, want %v", matches, want)
	}
	for i := range want {
		if matches[i] != want[i] {
			t.Fatalf("archivedChangeDirs = %v, want %v", matches, want)
		}
	}
	if !archiveExists(repo, "2-登录能力") {
		t.Fatal("archiveExists should accept a numeric archive directory")
	}
	if archiveExists(repo, "3-登录能力") {
		t.Fatal("archiveExists must not match an unrelated change")
	}
}

func TestArchivedAcceptancePathSupportsNumericArchive(t *testing.T) {
	// The archive gate must locate the contract after a numeric-only archive move.
	repo := t.TempDir()
	dir := filepath.Join(repo, "docs", "changes", "archive", "2-登录能力")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "acceptance.json")
	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := archivedAcceptancePath(repo, "2-登录能力")
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Fatalf("archivedAcceptancePath = %q, want %q", got, path)
	}
}
