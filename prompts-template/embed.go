// Package promptstemplate embeds the default specrelay flow prompt templates.
package promptstemplate

import "embed"

// FS contains the markdown templates installed by `specrelay flow install`.
//
//go:embed *.md
var FS embed.FS
