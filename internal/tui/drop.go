package tui

import (
	"net/url"
	"os"
	"strings"
)

func dropText(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}
	paths := droppedPaths(trimmed)
	if len(paths) == 0 {
		return trimmed, false
	}
	quoted := make([]string, 0, len(paths))
	for _, path := range paths {
		quoted = append(quoted, quotePath(path))
	}
	return strings.Join(quoted, " "), true
}

func droppedPaths(text string) []string {
	tokens := splitTokens(text)
	if len(tokens) == 0 {
		return nil
	}
	paths := make([]string, 0, len(tokens))
	for _, token := range tokens {
		path := fileURLPath(token)
		if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "./") {
			return nil
		}
		if _, err := os.Stat(path); err != nil {
			return nil
		}
		paths = append(paths, path)
	}
	return paths
}

func fileURLPath(token string) string {
	if !strings.HasPrefix(token, "file://") {
		return token
	}
	parsed, err := url.Parse(token)
	if err != nil {
		return token
	}
	return parsed.Path
}

func quotePath(path string) string {
	if strings.ContainsAny(path, " \t\"'") {
		return `"` + strings.ReplaceAll(path, `"`, `\"`) + `"`
	}
	return path
}

func splitTokens(text string) []string {
	var (
		tokens  []string
		current strings.Builder
		quote   rune
		escaped bool
		filled  bool
	)
	flush := func() {
		if filled {
			tokens = append(tokens, current.String())
			current.Reset()
			filled = false
		}
	}
	for _, r := range text {
		switch {
		case escaped:
			current.WriteRune(r)
			filled = true
			escaped = false
		case r == '\\' && quote != '\'':
			escaped = true
			filled = true
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
			filled = true
		case (r == '"' || r == '\'') && current.Len() == 0:
			quote = r
			filled = true
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		default:
			current.WriteRune(r)
			filled = true
		}
	}
	flush()
	return tokens
}
