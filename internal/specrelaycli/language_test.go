package specrelaycli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DeeliN221/specrelay/internal/i18n"
	"github.com/DeeliN221/specrelay/skills"
)

func TestLanguageCommandDefaultsToEnglish(t *testing.T) {
	project := newProject(t)
	t.Setenv("HOME", t.TempDir())
	t.Cleanup(func() { i18n.SetLanguage(i18n.English) })
	result := runCLI(t, project, "language")
	if result.code != 0 {
		t.Fatalf("language failed: %s", result.stderr)
	}
	if strings.TrimSpace(result.stdout) != "en" {
		t.Fatalf("default language = %q", result.stdout)
	}
}

func TestLanguageCommandSwitchesConfigAndInstalledSkills(t *testing.T) {
	project := newProject(t)
	t.Setenv("HOME", t.TempDir())
	t.Cleanup(func() { i18n.SetLanguage(i18n.English) })
	if result := runCLI(t, project, "install"); result.code != 0 {
		t.Fatalf("install failed: %s", result.stderr)
	}
	english, err := skills.BuiltIn(i18n.English)
	if err != nil {
		t.Fatal(err)
	}
	chinese, err := skills.BuiltIn(i18n.SimplifiedChinese)
	if err != nil {
		t.Fatal(err)
	}
	englishPlan := skillContent(english, "specrelay-plan")
	if englishPlan == "" {
		t.Fatal("missing English plan skill")
	}
	switchResult := runCLI(t, project, "language", "zh-CN")
	if switchResult.code != 0 {
		t.Fatalf("language zh-CN failed: %s", switchResult.stderr)
	}
	configData, err := os.ReadFile(filepath.Join(project, "specrelay-flow.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(configData), "language: zh-CN") {
		t.Fatalf("config missing language: %s", configData)
	}
	installed, err := os.ReadFile(filepath.Join(project, ".agents", "skills", "specrelay-plan", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != skillContent(chinese, "specrelay-plan") || string(installed) == englishPlan {
		t.Fatal("language switch did not refresh the installed skill")
	}
	if result := runCLI(t, project, "language", "en"); result.code != 0 {
		t.Fatalf("language en failed: %s", result.stderr)
	}
	installed, err = os.ReadFile(filepath.Join(project, ".agents", "skills", "specrelay-plan", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != englishPlan {
		t.Fatal("language switch did not restore the English skill")
	}
}

func TestLanguageGlobalPreferenceAppliesToProjects(t *testing.T) {
	project := newProject(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Cleanup(func() { i18n.SetLanguage(i18n.English) })
	if result := runCLI(t, project, "language", "zh-CN", "--global"); result.code != 0 {
		t.Fatalf("global language failed: %s", result.stderr)
	}
	data, err := os.ReadFile(filepath.Join(home, "specrelay-flow.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "language: zh-CN") {
		t.Fatalf("global config missing language: %s", data)
	}
	result := runCLI(t, project, "language")
	if result.code != 0 || strings.TrimSpace(result.stdout) != "zh-CN" {
		t.Fatalf("effective language = %q (%s)", result.stdout, result.stderr)
	}
}

func TestLanguageSwitchLocalizesCLIOutput(t *testing.T) {
	project := newProject(t)
	t.Setenv("HOME", t.TempDir())
	t.Cleanup(func() { i18n.SetLanguage(i18n.English) })
	if result := runCLI(t, project, "language", "zh-CN"); result.code != 0 {
		t.Fatalf("language zh-CN failed: %s", result.stderr)
	}
	help := runCLI(t, project, "--help")
	if !strings.Contains(help.stdout, "常用命令") {
		t.Fatalf("Chinese help missing translation:\n%s", help.stdout)
	}
	unknown := runCLI(t, project, "not-a-command")
	if !strings.Contains(unknown.stderr, "未知命令") {
		t.Fatalf("Chinese error missing translation: %s", unknown.stderr)
	}
	validate := runCLI(t, project, "validate", "1-缺失")
	if !strings.Contains(validate.stderr, "缺少") {
		t.Fatalf("Chinese validation error missing translation: %s", validate.stderr)
	}
	if result := runCLI(t, project, "language", "en"); result.code != 0 {
		t.Fatalf("language en failed: %s", result.stderr)
	}
	help = runCLI(t, project, "--help")
	if !strings.Contains(help.stdout, "Daily commands:") || strings.Contains(help.stdout, "常用命令") {
		t.Fatalf("English help missing translation rollback:\n%s", help.stdout)
	}
}

func skillContent(builtIn []skills.Skill, name string) string {
	for _, skill := range builtIn {
		if skill.Name == name {
			return skill.Content
		}
	}
	return ""
}
