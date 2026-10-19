// Package specrelaycli implements the standalone specrelay skill installation command.
package specrelaycli

import (
	"os"
	"path/filepath"

	"github.com/DeeliN221/specrelay/internal/app"
	"github.com/DeeliN221/specrelay/internal/i18n"
	"github.com/DeeliN221/specrelay/skills"
)

func (c *cli) installCmd(args []string) error {
	// installCmd installs built-in agent skills into a project or user skill directory.
	if hasHelp(args) {
		c.printInstallHelp()
		return nil
	}
	global := hasArg(args, "--global") || hasArg(args, "-g")
	if len(args) > 1 || (len(args) == 1 && !global) {
		return i18n.New("Usage:specrelay install [--global | -g]")
	}
	targetRoot := filepath.Join(".agents", "skills")
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		targetRoot = filepath.Join(home, ".agents", "skills")
	}
	language := app.EffectiveLanguageForCWD()
	builtIn, err := skills.BuiltIn(language)
	if err != nil {
		return err
	}
	for _, skill := range builtIn {
		target := filepath.Join(targetRoot, skill.Name, "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(skill.Content), 0o644); err != nil {
			return err
		}
	}
	i18n.Fprintf(c.out, "Installed %d skills into %s\n", len(builtIn), targetRoot)
	return nil
}

func (c *cli) printInstallHelp() {
	// printInstallHelp documents local and global skill installation forms.
	i18n.Fprintln(c.out, "Usage:")
	i18n.Fprintln(c.out, "  specrelay install [--global | -g]")
	i18n.Fprintln(c.out, "  specrelay i [--global | -g]")
}
