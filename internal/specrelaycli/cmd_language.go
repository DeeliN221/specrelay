// Package specrelaycli implements the standalone specrelay language preference command.
package specrelaycli

import (
	"os"
	"path/filepath"

	"github.com/DeeliN221/specrelay/internal/app"
	"github.com/DeeliN221/specrelay/internal/i18n"
	"github.com/DeeliN221/specrelay/skills"
)

func (c *cli) languageCmd(args []string) error {
	// languageCmd reads or persists the user-visible language and refreshes installed skills.
	if hasHelp(args) {
		i18n.Fprintln(c.out, "Usage:specrelay language [en|zh-CN] [--global] [--json]")
		return nil
	}
	global := hasArg(args, "--global") || hasArg(args, "-g")
	jsonOut := hasArg(args, "--json")
	value := firstPositional(args)
	repo, err := app.GitRoot(".")
	if err != nil {
		repo, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	if value == "" {
		language := app.EffectiveLanguage(repo)
		i18n.SetLanguage(language)
		if jsonOut {
			return writeJSON(c.out, map[string]string{"language": string(language)})
		}
		i18n.Fprintln(c.out, language)
		return nil
	}
	language, err := i18n.Parse(value)
	if err != nil {
		return err
	}
	path, err := app.SetLanguagePreference(repo, global, language)
	if err != nil {
		return err
	}
	i18n.SetLanguage(language)
	if err := refreshInstalledSkills(repo, language); err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(c.out, map[string]string{"language": string(language), "config": path})
	}
	i18n.Fprintf(c.out, "Language set to %s (%s)\n", language, path)
	return nil
}

// refreshInstalledSkills rewrites only skill directories that are already installed.
func refreshInstalledSkills(projectRoot string, language i18n.Language) error {
	builtIn, err := skills.BuiltIn(language)
	if err != nil {
		return err
	}
	roots := []string{filepath.Join(projectRoot, ".agents", "skills")}
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(home, ".agents", "skills"))
	}
	for _, root := range roots {
		for _, skill := range builtIn {
			dir := filepath.Join(root, skill.Name)
			info, statErr := os.Stat(dir)
			if statErr != nil || !info.IsDir() {
				continue
			}
			if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skill.Content), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}
