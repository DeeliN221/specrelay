package ozcli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

func TestEnglishChangeAndSpecificationValidate(t *testing.T) {
	project := newProject(t)
	change := "1-retain-archived-tests"
	writeValidChange(t, project, change)
	spec := "### Requirement: Retain tests\nThe system MUST retain each proposal test.\n\n#### Scenario: Archive a change\nWhen a proposal is archived, its tests remain accessible.\n"
	if err := os.WriteFile(filepath.Join(project, "docs", "changes", change, "spec.md"), []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	result := runCLI(t, project, "validate", change, "--json")
	if result.code != 0 {
		t.Fatalf("English proposal failed validation: %s %s", result.stdout, result.stderr)
	}
}

func TestEnglishAndUnicodeChangeNames(t *testing.T) {
	for _, name := range []string{"retain-tests", "Unicode-测试", "réparer-tests"} {
		if err := validateChangeName(name); err != nil {
			t.Errorf("name %q: %v", name, err)
		}
	}
	for _, name := range []string{"", "../escape", "-leading", "space name"} {
		if err := validateChangeName(name); err == nil {
			t.Errorf("accepted unsafe name %q", name)
		}
	}
}

func TestDefaultHelpAndErrorsAreEnglish(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"validate", "--help"}, {"not-a-command"}} {
		result := runCLI(t, newProject(t), args...)
		output := result.stdout + result.stderr
		if strings.IndexFunc(output, func(r rune) bool { return unicode.Is(unicode.Han, r) }) >= 0 {
			t.Fatalf("default output contains Chinese: %s", output)
		}
	}
}
