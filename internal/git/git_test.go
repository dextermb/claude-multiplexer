package git

import "testing"

func TestMergeDiffJoinsStatusAndCounts(t *testing.T) {
	names := "M\tinternal/tui/app.go\nA\tinternal/git/git.go\nD\told.txt\n"
	nums := "12\t3\tinternal/tui/app.go\n40\t0\tinternal/git/git.go\n0\t8\told.txt\n"

	stat, files := mergeDiff(names, nums)

	if stat.Files != 3 || stat.Insertions != 52 || stat.Deletions != 11 {
		t.Fatalf("stat = %+v, want 3 files, 52 insertions, 11 deletions", stat)
	}
	if len(files) != 3 {
		t.Fatalf("files = %d, want 3", len(files))
	}
	if files[0].Status != "M" || files[0].Path != "internal/tui/app.go" {
		t.Fatalf("file 0 = %+v", files[0])
	}
	if files[1].Insertions != 40 || files[1].Deletions != 0 {
		t.Fatalf("file 1 counts = %+v", files[1])
	}
	if files[2].Status != "D" || files[2].Deletions != 8 {
		t.Fatalf("file 2 = %+v", files[2])
	}
}

func TestMergeDiffReadsABinaryFileAsZero(t *testing.T) {
	names := "M\tlogo.png\n"
	nums := "-\t-\tlogo.png\n"

	stat, files := mergeDiff(names, nums)

	if stat.Files != 1 || stat.Insertions != 0 || stat.Deletions != 0 {
		t.Fatalf("stat = %+v, want 1 file, no lines", stat)
	}
	if files[0].Insertions != 0 || files[0].Deletions != 0 {
		t.Fatalf("file = %+v, want zero counts", files[0])
	}
}

func TestMergeDiffOfACleanTreeIsEmpty(t *testing.T) {
	stat, files := mergeDiff("", "")
	if !stat.Empty() || len(files) != 0 {
		t.Fatalf("clean tree = %+v %v, want empty", stat, files)
	}
}

func TestIsRepoReadsTheGitAnswer(t *testing.T) {
	old := run
	defer func() { run = old }()

	run = func(dir string, args ...string) (string, error) {
		return "true\n", nil
	}
	if !IsRepo("/tmp/work") {
		t.Fatal("want a work tree")
	}

	run = func(dir string, args ...string) (string, error) {
		return "", errNotRepo
	}
	if IsRepo("/tmp/work") {
		t.Fatal("a git error means no work tree")
	}
}

func TestDiffPrefersOriginHead(t *testing.T) {
	old := run
	defer func() { run = old }()

	run = func(dir string, args ...string) (string, error) {
		if args[0] == "rev-parse" {
			return "sha\n", nil
		}
		if !hasArg(args, "origin/HEAD") {
			t.Errorf("diff args = %v, want origin/HEAD", args)
		}
		return "", nil
	}
	Diff("/tmp/work")
	FileDiff("/tmp/work", "app.go")
}

func hasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func TestDiffFallsBackToHeadWithoutARemote(t *testing.T) {
	old := run
	defer func() { run = old }()

	run = func(dir string, args ...string) (string, error) {
		if args[0] == "rev-parse" {
			return "", errNotRepo
		}
		if hasArg(args, "origin/HEAD") {
			t.Errorf("diff args = %v, want the HEAD fallback", args)
		}
		return "", nil
	}
	Diff("/tmp/work")
}

var errNotRepo = &gitError{}

type gitError struct{}

func (*gitError) Error() string { return "not a git repository" }
