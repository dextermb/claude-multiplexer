package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const maxPathMatches = 6

func completePath(input string) (string, []string) {
	expanded := expandHome(input)
	dir, prefix := splitPath(expanded)

	entries, err := os.ReadDir(dirToRead(dir))
	if err != nil {
		return input, nil
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".") {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return input, nil
	}

	if len(names) == 1 {
		return dir + names[0] + string(filepath.Separator), names
	}
	return dir + commonPrefix(names), names
}

func expandHome(input string) string {
	if input != "~" && !strings.HasPrefix(input, "~"+string(filepath.Separator)) {
		return input
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return input
	}
	return filepath.Join(home, strings.TrimPrefix(input, "~"))
}

func splitPath(path string) (string, string) {
	index := strings.LastIndex(path, string(filepath.Separator))
	if index < 0 {
		return "", path
	}
	return path[:index+1], path[index+1:]
}

func dirToRead(dir string) string {
	if dir == "" {
		return "."
	}
	return dir
}

func commonPrefix(names []string) string {
	prefix := names[0]
	for _, name := range names[1:] {
		for !strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}
	return prefix
}

func pathHint(names []string, picked, width int) string {
	if len(names) == 0 {
		return ""
	}
	start := 0
	if picked >= maxPathMatches {
		start = picked - maxPathMatches + 1
	}
	end := start + maxPathMatches
	if end > len(names) {
		end = len(names)
	}

	parts := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		if i == picked {
			parts = append(parts, pickedPathStyle.Render(names[i]))
			continue
		}
		parts = append(parts, names[i])
	}
	text := strings.Join(parts, "  ")
	if extra := len(names) - end; extra > 0 {
		text += "  +" + strconv.Itoa(extra) + " more"
	}
	if picked < 0 {
		return truncate(text, width)
	}
	return text
}
