package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DeeliN221/specrelay/internal/i18n"
)

func TestEffectiveLanguagePrecedence(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got := EffectiveLanguage(repo); got != i18n.English {
		t.Fatalf("default language = %q", got)
	}
	if err := os.WriteFile(filepath.Join(home, "specrelay-flow.yaml"), []byte("language: zh-CN\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := EffectiveLanguage(repo); got != i18n.SimplifiedChinese {
		t.Fatalf("global language = %q", got)
	}
	if err := os.WriteFile(filepath.Join(repo, "specrelay-flow.yaml"), []byte("language: en\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := EffectiveLanguage(repo); got != i18n.English {
		t.Fatalf("project language = %q", got)
	}
}

func TestSetLanguagePreferenceCreatesAndUpdatesConfig(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	path, err := SetLanguagePreference(repo, false, i18n.SimplifiedChinese)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "language: zh-CN") {
		t.Fatalf("new config missing language: %s", data)
	}
	config, err := LoadWorkflowConfig(repo)
	if err != nil {
		t.Fatal(err)
	}
	if config.Prompts["planning"] != defaultPromptSet(i18n.SimplifiedChinese)["planning"] {
		t.Fatalf("new config did not use Chinese prompts")
	}
	if _, err := SetLanguagePreference(repo, false, i18n.English); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "language: en") {
		t.Fatalf("updated config missing language: %s", data)
	}
	if !strings.Contains(string(data), "stages:") {
		t.Fatalf("language update dropped existing config: %s", data)
	}
}

func TestLoadWorkflowConfigLocalizesDefaultPrompts(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	english := DefaultWorkflowConfigYAMLFor(i18n.English)
	configPath := filepath.Join(repo, "specrelay-flow.yaml")
	if err := os.WriteFile(configPath, []byte(strings.Replace(english, "language: en", "language: zh-CN", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	config, err := LoadWorkflowConfig(repo)
	if err != nil {
		t.Fatal(err)
	}
	want := defaultPromptSet(i18n.SimplifiedChinese)
	if config.Language != i18n.SimplifiedChinese {
		t.Fatalf("language = %q", config.Language)
	}
	if config.Prompts["planning"] != want["planning"] || config.Prompts["qa"] != want["qa"] {
		t.Fatal("untouched default prompts were not localized")
	}
}

func TestLoadWorkflowConfigPreservesCustomPrompts(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	body := "language: zh-CN\nprompts:\n  planning: |\n    CUSTOM_PLANNING\n"
	if err := os.WriteFile(filepath.Join(repo, "specrelay-flow.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	config, err := LoadWorkflowConfig(repo)
	if err != nil {
		t.Fatal(err)
	}
	if config.Prompts["planning"] != "CUSTOM_PLANNING\n" && strings.TrimSpace(config.Prompts["planning"]) != "CUSTOM_PLANNING" {
		t.Fatalf("custom planning prompt was replaced: %q", config.Prompts["planning"])
	}
	want := defaultPromptSet(i18n.SimplifiedChinese)
	if config.Prompts["qa"] != want["qa"] {
		t.Fatal("missing prompts should use the selected language defaults")
	}
}
