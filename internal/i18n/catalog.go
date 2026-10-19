package i18n

import (
	"embed"
	"encoding/json"
	"io/fs"
)

// catalogFS contains translated messages merged in filename order.
//
//go:embed catalog/*.json
var catalogFS embed.FS

func init() {
	entries, err := fs.ReadDir(catalogFS, "catalog")
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := catalogFS.ReadFile("catalog/" + entry.Name())
		if err != nil {
			continue
		}
		var catalog map[string]string
		if err := json.Unmarshal(data, &catalog); err != nil {
			continue
		}
		Register(catalog)
	}
}
