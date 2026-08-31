package template

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var fieldPattern = regexp.MustCompile(`\{\{([A-Za-z0-9_-]+)(?:=([^}]*))?\}\}`)

type Field struct {
	Name    string
	Default string
}

type Template struct {
	Name        string
	Description string
	Body        string
	Fields      []Field
	Path        string
}

func Parse(name string, data []byte) Template {
	front, body := splitFrontMatter(string(data))
	tpl := Template{
		Name:        name,
		Description: front["description"],
		Body:        strings.TrimSpace(body),
	}
	if tpl.Description == "" {
		tpl.Description = firstLine(tpl.Body)
	}
	tpl.Fields = fieldsOf(tpl.Body)
	return tpl
}

func Load(dirs ...string) []Template {
	byName := make(map[string]Template)
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".md")
			tpl := Parse(name, data)
			tpl.Path = path
			byName[name] = tpl
		}
	}

	out := make([]Template, 0, len(byName))
	for _, tpl := range byName {
		out = append(out, tpl)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func Find(all []Template, name string) (Template, bool) {
	for _, tpl := range all {
		if tpl.Name == name {
			return tpl, true
		}
	}
	return Template{}, false
}

func Match(all []Template, prefix string) []Template {
	prefix = strings.ToLower(prefix)
	var out []Template
	for _, tpl := range all {
		if strings.HasPrefix(strings.ToLower(tpl.Name), prefix) {
			out = append(out, tpl)
		}
	}
	return out
}

func (t Template) Expand(values map[string]string) string {
	return fieldPattern.ReplaceAllStringFunc(t.Body, func(match string) string {
		parts := fieldPattern.FindStringSubmatch(match)
		if value, ok := values[parts[1]]; ok && value != "" {
			return value
		}
		return parts[2]
	})
}

func (t Template) Fill(args []string) (map[string]string, []string) {
	values := make(map[string]string, len(t.Fields))
	named := make(map[string]bool, len(t.Fields))
	for _, field := range t.Fields {
		values[field.Name] = field.Default
	}

	var positional []string
	for _, arg := range args {
		name, value, found := strings.Cut(arg, "=")
		if found && t.hasField(name) {
			values[name] = value
			named[name] = true
			continue
		}
		positional = append(positional, arg)
	}

	open := make([]string, 0, len(t.Fields))
	for _, field := range t.Fields {
		if !named[field.Name] {
			open = append(open, field.Name)
		}
	}
	for i, name := range open {
		if len(positional) == 0 {
			break
		}
		if i == len(open)-1 {
			values[name] = strings.Join(positional, " ")
			positional = nil
			continue
		}
		values[name] = positional[0]
		positional = positional[1:]
	}

	var missing []string
	for _, field := range t.Fields {
		if values[field.Name] == "" {
			missing = append(missing, field.Name)
		}
	}
	return values, missing
}

func (t Template) hasField(name string) bool {
	for _, field := range t.Fields {
		if field.Name == name {
			return true
		}
	}
	return false
}

func (t Template) FieldNames() []string {
	out := make([]string, 0, len(t.Fields))
	for _, field := range t.Fields {
		out = append(out, field.Name)
	}
	return out
}

func fieldsOf(body string) []Field {
	var (
		fields []Field
		seen   = make(map[string]bool)
	)
	for _, match := range fieldPattern.FindAllStringSubmatch(body, -1) {
		name := match[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		fields = append(fields, Field{Name: name, Default: match[2]})
	}
	return fields
}

func splitFrontMatter(text string) (map[string]string, string) {
	front := make(map[string]string)
	text = strings.TrimPrefix(text, "\ufeff")
	if !strings.HasPrefix(text, "---\n") {
		return front, text
	}
	rest := text[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return front, text
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		front[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	body := rest[end+len("\n---"):]
	return front, strings.TrimPrefix(body, "\n")
}

func firstLine(body string) string {
	for _, line := range strings.Split(body, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// DirNames are the spellings of the state directory that are read. See
// docs/templates.md.
var DirNames = []string{".multiplexier", ".multiplexer"}

func Dirs(root, sessionDir string) []string {
	var out []string
	for _, dir := range rootDirs(root) {
		out = append(out, filepath.Join(dir, "templates"))
	}
	if sessionDir != "" {
		for _, name := range DirNames {
			out = append(out, filepath.Join(sessionDir, name, "templates"))
		}
	}
	return out
}

func rootDirs(root string) []string {
	if root == "" {
		return nil
	}
	out := []string{root}
	base := filepath.Base(root)
	for _, name := range DirNames {
		if name != base && (base == DirNames[0] || base == DirNames[1]) {
			out = append(out, filepath.Join(filepath.Dir(root), name))
		}
	}
	return out
}

func ParseInvocation(text string) (string, []string, bool) {
	trimmed := strings.TrimLeft(text, " \t")
	if !strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "//") {
		return "", nil, false
	}
	parts := splitArgs(trimmed[1:])
	if len(parts) == 0 {
		return "", nil, false
	}
	return parts[0], parts[1:], true
}

func splitArgs(text string) []string {
	var (
		out     []string
		current strings.Builder
		quote   rune
		filled  bool
	)
	flush := func() {
		if filled {
			out = append(out, current.String())
			current.Reset()
			filled = false
		}
	}
	var (
		escaped bool
		prev    rune
	)
	for _, r := range text {
		switch {
		case escaped:
			current.WriteRune(r)
			filled = true
			escaped = false
		case quote != 0 && r == '\\':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
				break
			}
			current.WriteRune(r)
			filled = true
		case (r == '"' || r == '\'') && (current.Len() == 0 || prev == '='):
			quote = r
			filled = true
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		default:
			current.WriteRune(r)
			filled = true
		}
		prev = r
	}
	flush()
	return out
}
