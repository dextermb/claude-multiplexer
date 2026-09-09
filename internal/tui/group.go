package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

const maxLabelDepth = 8

// A group key names a directory, the control session that created the rows, or
// the peer a remote session involves.
const (
	dirPrefix  = "dir:"
	byPrefix   = "by:"
	hostPrefix = "host:"
)

type group struct {
	key      string
	label    string
	section  sectionKind
	creator  bool
	folded   bool
	rank     int
	count    int
	live     bool
	archived bool
	state    session.State
}

type listLine struct {
	group   int
	row     int
	divider string
}

// header is true for any line that is not a selectable row: a group header or a
// section divider.
func (l listLine) header() bool { return l.row < 0 }

// isDivider is true for a section divider line, which carries a label and no
// group.
func (l listLine) isDivider() bool { return l.row < 0 && l.divider != "" }

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

// headsGroup is true for the control session at the top of the group of the
// sessions it created.
func headsGroup(item row) bool {
	return item.group == byPrefix+item.name
}

// labelGroups names every group: a creator group takes the display name of the
// control session, and a directory group takes the tail of its path.
func labelGroups(rows []row) map[string]string {
	shown := make(map[string]string, len(rows))
	for _, item := range rows {
		shown[item.name] = item.displayName()
	}
	labels := make(map[string]string, len(rows))
	var roots []string
	for _, item := range rows {
		if _, done := labels[item.group]; done {
			continue
		}
		if creator, ok := strings.CutPrefix(item.group, byPrefix); ok {
			labels[item.group] = creator
			if name, live := shown[creator]; live {
				labels[item.group] = name
			}
			continue
		}
		if host, ok := strings.CutPrefix(item.group, hostPrefix); ok {
			labels[item.group] = host
			continue
		}
		labels[item.group] = ""
		roots = append(roots, strings.TrimPrefix(item.group, dirPrefix))
	}
	for root, label := range groupLabels(roots) {
		labels[dirPrefix+root] = label
	}
	return labels
}

// groupRows keys every row on its section and group, orders the groups so that a
// live one comes first inside each section (and a local section before a remote
// one), and keeps the order of the rows inside each group.
func groupRows(rows []row, folded map[string]bool) ([]row, []group) {
	if len(rows) == 0 {
		return rows, nil
	}
	var groups []group
	index := make(map[string]int, len(rows))
	lead := make(map[string]int, len(rows))
	identity := func(item row) string {
		return string(rune('0'+int(item.section))) + item.group
	}
	for _, item := range rows {
		id := identity(item)
		at, ok := index[id]
		if !ok {
			at = len(groups)
			index[id] = at
			groups = append(groups, group{
				key:     item.group,
				section: item.section,
				creator: strings.HasPrefix(item.group, byPrefix),
				rank:    rowRank(item),
				folded:  folded[item.group],
			})
			lead[id] = urgency(item) + 1
		}
		groups[at].count++
		if rank := rowRank(item); rank < groups[at].rank {
			groups[at].rank = rank
		}
		if mark := urgency(item); mark < lead[id] {
			lead[id] = mark
		}
		if urgency(item) == lead[id] {
			groups[at].live = item.live
			groups[at].archived = item.archived
			groups[at].state = item.state
		}
	}
	labels := labelGroups(rows)
	for i := range groups {
		groups[i].label = labels[groups[i].key]
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].section != groups[j].section {
			return groups[i].section < groups[j].section
		}
		return groups[i].rank < groups[j].rank
	})

	groupAt := make(map[string]int, len(groups))
	for i, g := range groups {
		groupAt[string(rune('0'+int(g.section)))+g.key] = i
	}
	sorted := make([]row, len(rows))
	copy(sorted, rows)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := groupAt[identity(sorted[i])]
		right := groupAt[identity(sorted[j])]
		if left != right {
			return left < right
		}
		return headsGroup(sorted[i]) && !headsGroup(sorted[j])
	})
	return sorted, groups
}

// remoteBand reports whether a section sits under the "remote sessions" band. A
// hosted session shows muted, so the two remote bands share one divider. See
// docs/peers.md.
func remoteBand(kind sectionKind) bool {
	return kind == sectionHosted || kind == sectionStreamed
}

// listLines is the sidebar as it is drawn: a header for every group, and the
// rows of every group that is not folded. When sectioned is true, a divider
// names the local band and the remote band, the remote one drawn once above the
// first remote group. See docs/peers.md.
func listLines(rows []row, groups []group, sectioned bool) []listLine {
	order := make(map[string]int, len(groups))
	for i, item := range groups {
		order[string(rune('0'+int(item.section)))+item.key] = i
	}
	lines := make([]listLine, 0, len(rows)+len(groups)+4)
	current := -1
	shownLocal := false
	shownRemote := false
	for i, item := range rows {
		at := order[string(rune('0'+int(item.section)))+item.group]
		if at != current {
			if sectioned {
				if remoteBand(item.section) && !shownRemote {
					lines = append(lines, listLine{group: -1, row: -1, divider: "remote sessions"})
					shownRemote = true
				}
				if !remoteBand(item.section) && !shownLocal {
					lines = append(lines, listLine{group: -1, row: -1, divider: "local sessions"})
					shownLocal = true
				}
			}
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
