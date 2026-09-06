package mcp

import (
	"embed"
	"io/fs"
	"sort"
	"strings"
)

// The rules the multiplexer injects into every session; see docs/mcp/rules.md.
//
//go:embed injected/rules/*.md
var ruleFiles embed.FS

func instructions() string {
	entries, err := fs.ReadDir(ruleFiles, "injected/rules")
	if err != nil {
		return ""
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	parts := make([]string, 0, len(names))
	for _, name := range names {
		data, err := fs.ReadFile(ruleFiles, "injected/rules/"+name)
		if err != nil {
			continue
		}
		if text := strings.TrimSpace(string(data)); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n\n")
}
