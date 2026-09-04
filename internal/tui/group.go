package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

const maxLabelDepth = 8

type group struct {
	key      string
	label    string
	folded   bool
	rank     int
	count    int
	live     bool
	archived bool
	state    session.State
}

type listLine struct {
	group int
	row   int
}

func (l listLine) header() bool { return l.row < 0 }

// repoRoot is the group key: the repository above dir, or dir when there is none.
func repoRoot(dir string) string {
	if dir == "" {
		return ""
	}
	for current := filepath.Clean(dir); ; {
		marker := filepath.Join(current, ".git")
		info, err := os.Lstat(marker)
		if err == nil {
			if info.IsDir() {
				return current
			}
			if root := rootFromGitFile(marker); root != "" {
				return root
			}
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return filepath.Clean(dir)
		}
		current = parent
	}
}

// rootFromGitFile reads the .git file of a worktree or a submodule, whose gitdir
// sits inside the .git directory of the repository that owns it.
func rootFromGitFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	gitdir, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "gitdir:")
	if !ok {
		return ""
	}
	gitdir = strings.TrimSpace(gitdir)
	if gitdir == "" {
		return ""
	}
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(filepath.Dir(path), gitdir)
	}
	gitdir = filepath.Clean(gitdir)
	separator := string(filepath.Separator)
	index := strings.Index(gitdir, separator+".git"+separator)
	if index < 0 {
		return ""
	}
	return gitdir[:index]
}

// groupLabels names each root by its last element, and grows a name to the left
// while it matches another one.
func groupLabels(roots []string) map[string]string {
	depth := make(map[string]int, len(roots))
	for _, root := range roots {
		depth[root] = 1
	}
	labels := make(map[string]string, len(roots))
	for range maxLabelDepth {
		shared := make(map[string][]string, len(roots))
		for _, root := range roots {
			labels[root] = tailPath(root, depth[root])
			shared[labels[root]] = append(shared[labels[root]], root)
		}
		grew := false
		for _, matched := range shared {
			if len(matched) < 2 {
				continue
			}
			for _, root := range matched {
				if tailPath(root, depth[root]+1) == labels[root] {
					continue
				}
				depth[root]++
				grew = true
			}
		}
		if !grew {
			break
		}
	}
	return labels
}

func tailPath(root string, depth int) string {
	if root == "" {
		return "no directory"
	}
	separator := string(filepath.Separator)
	trimmed := strings.TrimSuffix(root, separator)
	if trimmed == "" {
		return separator
	}
	parts := strings.Split(trimmed, separator)
	if depth >= len(parts) {
		return root
	}
	return strings.Join(parts[len(parts)-depth:], separator)
}

// urgency ranks a row for two jobs: the order of the groups, and the glyph a
// folded group shows for the rows it hides.
func urgency(item row) int {
	if !item.live {
		if item.archived {
			return 6
		}
		return 5
	}
	switch item.state {
	case session.StateWaiting:
		return 0
	case session.StateBusy:
		return 1
	case session.StateFailed:
		return 2
	case session.StateStarting:
		return 3
	}
	return 4
}

func rowRank(item row) int {
	switch {
	case item.live:
		return 0
	case item.archived:
		return 2
	}
	return 1
}

// groupRows keys every row on its repository, orders the groups so that a live
// one comes first, and keeps the order of the rows inside each group.
func groupRows(rows []row, folded map[string]bool) ([]row, []group) {
	if len(rows) == 0 {
		return rows, nil
	}
	var groups []group
	index := make(map[string]int, len(rows))
	lead := make(map[string]int, len(rows))
	keys := make([]string, 0, len(rows))
	for _, item := range rows {
		at, ok := index[item.group]
		if !ok {
			at = len(groups)
			index[item.group] = at
			keys = append(keys, item.group)
			groups = append(groups, group{key: item.group, rank: rowRank(item), folded: folded[item.group]})
			lead[item.group] = urgency(item) + 1
		}
		groups[at].count++
		if rank := rowRank(item); rank < groups[at].rank {
			groups[at].rank = rank
		}
		if mark := urgency(item); mark < lead[item.group] {
			lead[item.group] = mark
		}
		if urgency(item) == lead[item.group] {
			groups[at].live = item.live
			groups[at].archived = item.archived
			groups[at].state = item.state
		}
	}
	labels := groupLabels(keys)
	for i := range groups {
		groups[i].label = labels[groups[i].key]
	}
	sort.SliceStable(groups, func(i, j int) bool { return groups[i].rank < groups[j].rank })

	order := make(map[string]int, len(groups))
	for i, item := range groups {
		order[item.key] = i
	}
	sorted := make([]row, len(rows))
	copy(sorted, rows)
	sort.SliceStable(sorted, func(i, j int) bool { return order[sorted[i].group] < order[sorted[j].group] })
	return sorted, groups
}

// listLines is the sidebar as it is drawn: a header for every group, and the
// rows of every group that is not folded.
func listLines(rows []row, groups []group) []listLine {
	order := make(map[string]int, len(groups))
	for i, item := range groups {
		order[item.key] = i
	}
	lines := make([]listLine, 0, len(rows)+len(groups))
	current := -1
	for i, item := range rows {
		at := order[item.group]
		if at != current {
			lines = append(lines, listLine{group: at, row: -1})
			current = at
		}
		if groups[at].folded {
			continue
		}
		lines = append(lines, listLine{group: at, row: i})
	}
	return lines
}
