// Package git reads the working-tree diff of a directory against HEAD. See
// docs/tui/diff.md.
package git

import (
	"os/exec"
	"strconv"
	"strings"
)

// Stat is the line count of a diff.
type Stat struct {
	Insertions int
	Deletions  int
	Files      int
}

// Empty reports whether the stat has no change.
func (s Stat) Empty() bool {
	return s.Files == 0 && s.Insertions == 0 && s.Deletions == 0
}

// FileChange is one changed file of a diff.
type FileChange struct {
	Status     string
	Path       string
	Insertions int
	Deletions  int
}

// run runs git in dir with args, and returns its standard output. It is a
// variable so a test records the arguments instead of running git.
var run = func(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(out), err
}

// IsRepo reports whether dir is inside a git work tree.
func IsRepo(dir string) bool {
	out, err := run(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(out) == "true"
}

// Diff reads the working-tree changes of dir against HEAD.
func Diff(dir string) (Stat, []FileChange, error) {
	names, err := run(dir, "diff", "--name-status", "HEAD")
	if err != nil {
		return Stat{}, nil, err
	}
	nums, err := run(dir, "diff", "--numstat", "HEAD")
	if err != nil {
		return Stat{}, nil, err
	}
	stat, files := mergeDiff(names, nums)
	return stat, files, nil
}

// FileDiff reads the working-tree diff of one file in dir against HEAD.
func FileDiff(dir, path string) (string, error) {
	return run(dir, "diff", "HEAD", "--", path)
}

// mergeDiff joins the name-status output and the numstat output into the file
// list and the total stat, keyed by the changed path.
func mergeDiff(names, nums string) (Stat, []FileChange) {
	counts := parseNumstat(nums)
	var files []FileChange
	var stat Stat
	for _, line := range strings.Split(strings.TrimRight(names, "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		status := string(fields[0][0])
		path := fields[len(fields)-1]
		ins, del := counts[path][0], counts[path][1]
		files = append(files, FileChange{Status: status, Path: path, Insertions: ins, Deletions: del})
		stat.Insertions += ins
		stat.Deletions += del
		stat.Files++
	}
	return stat, files
}

// parseNumstat reads the numstat lines into insertions and deletions by path. A
// binary file reports "-", which counts as zero.
func parseNumstat(nums string) map[string][2]int {
	counts := make(map[string][2]int)
	for _, line := range strings.Split(strings.TrimRight(nums, "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 3 {
			continue
		}
		path := fields[len(fields)-1]
		counts[path] = [2]int{atoi(fields[0]), atoi(fields[1])}
	}
	return counts
}

func atoi(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}
