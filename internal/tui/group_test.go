package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepoRootFindsTheRepositoryAboveTheDirectory(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	deep := filepath.Join(repo, "internal", "tui")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if got := repoRoot(repo); got != repo {
		t.Errorf("repoRoot(repo) = %q, want %q", got, repo)
	}
	if got := repoRoot(deep); got != repo {
		t.Errorf("repoRoot(subdirectory) = %q, want %q", got, repo)
	}
}

func TestRepoRootReadsTheGitFileOfAWorktree(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	tree := filepath.Join(repo, ".worktrees", "api")
	if err := os.MkdirAll(tree, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	gitdir := filepath.Join(repo, ".git", "worktrees", "api")
	if err := os.WriteFile(filepath.Join(tree, ".git"), []byte("gitdir: "+gitdir+"\n"), 0o644); err != nil {
		t.Fatalf("write .git: %v", err)
	}

	if got := repoRoot(tree); got != repo {
		t.Errorf("repoRoot(worktree) = %q, want the repository %q", got, repo)
	}
}

func TestRepoRootReadsARelativeGitFile(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	module := filepath.Join(repo, "vendor")
	if err := os.MkdirAll(module, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(module, ".git"), []byte("gitdir: ../.git/modules/vendor"), 0o644); err != nil {
		t.Fatalf("write .git: %v", err)
	}

	if got := repoRoot(module); got != repo {
		t.Errorf("repoRoot(submodule) = %q, want the repository %q", got, repo)
	}
}

func TestRepoRootFallsBackToTheDirectory(t *testing.T) {
	dir := t.TempDir()
	if got := repoRoot(dir); got != dir {
		t.Errorf("repoRoot = %q, want the directory %q", got, dir)
	}
	if got := repoRoot(""); got != "" {
		t.Errorf("repoRoot(\"\") = %q, want the empty key", got)
	}
}

func TestGroupLabelsGrowOnlyWhenTwoMatch(t *testing.T) {
	labels := groupLabels([]string{"/home/a/api", "/home/b/api", "/home/a/web"})
	want := map[string]string{
		"/home/a/api": "a/api",
		"/home/b/api": "b/api",
		"/home/a/web": "web",
	}
	for root, label := range want {
		if labels[root] != label {
			t.Errorf("label of %q = %q, want %q", root, labels[root], label)
		}
	}
}

func TestGroupLabelsNameASessionWithNoDirectory(t *testing.T) {
	if got := groupLabels([]string{""})[""]; got != "no directory" {
		t.Errorf("label = %q, want \"no directory\"", got)
	}
}

func mkdir(t *testing.T, parts ...string) string {
	t.Helper()
	path := filepath.Join(parts...)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	return path
}

func writeGitFile(t *testing.T, dir, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte(text), 0o644); err != nil {
		t.Fatalf("write .git: %v", err)
	}
}
