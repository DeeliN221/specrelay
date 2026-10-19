// Package skills embeds specrelay agent skill templates from the source tree.
package skills

import (
	"embed"
	"io/fs"
	"path"
	"sort"

	"github.com/DeeliN221/specrelay/internal/i18n"
)

// FS contains the source skill templates injected into the specrelay binary at build time.
//
//go:embed specrelay-*/SKILL.md specrelay-*/SKILL.zh-CN.md
var FS embed.FS

type Skill struct {
	Name    string
	Content string
}

// BuiltIn returns embedded skills in deterministic name order for the selected language.
func BuiltIn(language i18n.Language) ([]Skill, error) {
	fileName := "SKILL.md"
	if language == i18n.SimplifiedChinese {
		fileName = "SKILL.zh-CN.md"
	}
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		return nil, err
	}
	out := []Skill{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		filePath := path.Join(entry.Name(), fileName)
		data, err := FS.ReadFile(filePath)
		if err != nil {
			// Fall back to English when a translated skill is absent.
			data, err = FS.ReadFile(path.Join(entry.Name(), "SKILL.md"))
			if err != nil {
				return nil, err
			}
		}
		out = append(out, Skill{Name: entry.Name(), Content: string(data)})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out, nil
}
