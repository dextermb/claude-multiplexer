package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name, FileName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPathsFollowTheConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/settings")

	want := []string{
		"/tmp/settings/claude-multiplexer/config.json",
		"/tmp/settings/multiplexer/config.json",
		"/tmp/settings/multiplexier/config.json",
	}
	got := Paths()
	if len(got) != len(want) {
		t.Fatalf("Paths() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Paths()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPathsFallBackToDotConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	want := filepath.Join(home, ".config", "claude-multiplexer", FileName)
	if got := Paths()[0]; got != want {
		t.Fatalf("Paths()[0] = %q, want %q", got, want)
	}
}

func TestTheFirstSpellingWins(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "multiplexer", `{"editor": "nvim"}`)
	write(t, dir, "multiplexier", `{"editor": "zed"}`)
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg, err := Load(Paths()...)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Editor != "nvim" {
		t.Fatalf("editor = %q, want the multiplexer spelling", cfg.Editor)
	}
}

func TestTheOtherSpellingIsRead(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "multiplexier", `{"editor": "zed"}`)
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg, err := Load(Paths()...)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Editor != "zed" {
		t.Fatalf("editor = %q, want zed", cfg.Editor)
	}
}

func TestAMissingFileIsNotAnError(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nothing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Editor != "" {
		t.Fatalf("editor = %q, want an empty configuration", cfg.Editor)
	}
}

func TestAnEmptyFileIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	path := write(t, dir, "multiplexer", "  \n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Editor != "" {
		t.Fatalf("editor = %q, want an empty configuration", cfg.Editor)
	}
}

func TestABadFileIsAnError(t *testing.T) {
	dir := t.TempDir()
	path := write(t, dir, "multiplexer", "{editor: nvim}")

	if _, err := Load(path); err == nil {
		t.Fatal("a file that does not parse must give an error")
	}
}

func TestTheEditorTerminalField(t *testing.T) {
	dir := t.TempDir()
	path := write(t, dir, "multiplexer", `{"editor": "myed", "editorTerminal": true}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.EditorTerminal == nil || !*cfg.EditorTerminal {
		t.Fatalf("editorTerminal = %v, want true", cfg.EditorTerminal)
	}
}

func TestTheLadderOrder(t *testing.T) {
	file := Config{Editor: "from-file"}

	cases := []struct {
		name   string
		flag   string
		visual string
		editor string
		want   string
	}{
		{"the flag wins", "from-flag", "from-visual", "from-editor", "from-flag"},
		{"then VISUAL", "", "from-visual", "from-editor", "from-visual"},
		{"then EDITOR", "", "", "from-editor", "from-editor"},
		{"then the file", "", "", "", "from-file"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("VISUAL", tc.visual)
			t.Setenv("EDITOR", tc.editor)

			if got := Resolve(Config{Editor: tc.flag}, file, Config{}).Editor; got != tc.want {
				t.Fatalf("editor = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTheFlagOverridesTheEditorTerminalField(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")

	yes, no := true, false
	got := Resolve(Config{EditorTerminal: &no}, Config{Editor: "myed", EditorTerminal: &yes}, Config{})
	if got.EditorTerminal == nil || *got.EditorTerminal {
		t.Fatalf("editorTerminal = %v, want false", got.EditorTerminal)
	}
	if got.Editor != "myed" {
		t.Fatalf("editor = %q, want myed", got.Editor)
	}
}

func TestTargetTakesTheFileThatIsThere(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	first := filepath.Join(dir, "multiplexer", FileName)
	second := write(t, dir, "multiplexier", `{"editor": "zed"}`)
	if got := Target(Paths()...); got != second {
		t.Fatalf("Target() = %q, want the file that is there %q", got, second)
	}

	write(t, dir, "multiplexer", `{"editor": "nvim"}`)
	if got := Target(Paths()...); got != first {
		t.Fatalf("Target() = %q, want the first spelling %q", got, first)
	}
}

func TestTargetFallsBackToTheFirstPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	want := filepath.Join(dir, "claude-multiplexer", FileName)
	if got := Target(Paths()...); got != want {
		t.Fatalf("Target() = %q, want %q", got, want)
	}
}

func TestWriteMakesTheFileAndTheDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deep", "multiplexer", FileName)
	yes := true

	if err := Write(path, Config{Editor: "nvim", EditorTerminal: &yes}); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Editor != "nvim" {
		t.Fatalf("editor = %q, want nvim", cfg.Editor)
	}
	if cfg.EditorTerminal == nil || !*cfg.EditorTerminal {
		t.Fatalf("editorTerminal = %v, want true", cfg.EditorTerminal)
	}
}

func TestWriteLeavesOutTheFieldsThatAreNotSet(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	if err := Write(path, Config{Editor: "zed"}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "editorTerminal") {
		t.Fatalf("the file names a field that was never set:\n%s", data)
	}
}

func TestClearTakesOutTheFieldItIsGiven(t *testing.T) {
	yes := true
	full := Config{Editor: "nvim", EditorTerminal: &yes}

	cases := map[string]Config{
		FieldEditor:   {EditorTerminal: &yes},
		FieldTerminal: {Editor: "nvim"},
		FieldBoth:     {},
		"":            {},
	}
	for field, want := range cases {
		got, err := Clear(full, field)
		if err != nil {
			t.Fatalf("Clear(%q): %v", field, err)
		}
		if got.Editor != want.Editor || (got.EditorTerminal == nil) != (want.EditorTerminal == nil) {
			t.Errorf("Clear(%q) = %+v, want %+v", field, got, want)
		}
	}
}

func TestClearRefusesAFieldItDoesNotKnow(t *testing.T) {
	if _, err := Clear(Config{Editor: "nvim"}, "colour"); err == nil {
		t.Fatal("an unknown field must give an error")
	}
}

func TestBlockCapOrDefault(t *testing.T) {
	if got := BlockCapOrDefault(Config{}); got != DefaultBlockCap {
		t.Fatalf("cap = %d, want %d", got, DefaultBlockCap)
	}
	none := 0
	if got := BlockCapOrDefault(Config{BlockCap: &none}); got != 0 {
		t.Fatalf("cap = %d, want 0, which caps nothing", got)
	}
	forty := 40
	if got := BlockCapOrDefault(Config{BlockCap: &forty}); got != 40 {
		t.Fatalf("cap = %d, want 40", got)
	}
}

func TestTheFlagSitsAboveTheFileForTheBlockCap(t *testing.T) {
	file, flag := 40, 12
	got := Resolve(Config{BlockCap: &flag}, Config{BlockCap: &file}, Config{})
	if got.BlockCap == nil || *got.BlockCap != 12 {
		t.Fatalf("cap = %v, want the flag", got.BlockCap)
	}
	got = Resolve(Config{}, Config{BlockCap: &file}, Config{})
	if got.BlockCap == nil || *got.BlockCap != 40 {
		t.Fatalf("cap = %v, want the file", got.BlockCap)
	}
}

func TestTheBlockCapSurvivesTheFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	forty := 40
	if err := Write(path, Config{Editor: "nvim", BlockCap: &forty}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.BlockCap == nil || *got.BlockCap != 40 {
		t.Fatalf("blockCap = %v, want 40", got.BlockCap)
	}
}

func TestDefaultRootTakesTheNewNameWhenNothingIsThere(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := DefaultRoot()
	if err != nil {
		t.Fatalf("DefaultRoot: %v", err)
	}
	if want := filepath.Join(home, ".claude-multiplexer"); got != want {
		t.Fatalf("root = %q, want %q", got, want)
	}
}

func TestDefaultRootKeepsTheOlderDirectoryWhenItIsThere(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".multiplexier", "sessions"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := DefaultRoot()
	if err != nil {
		t.Fatalf("DefaultRoot: %v", err)
	}
	if want := filepath.Join(home, ".multiplexier"); got != want {
		t.Fatalf("root = %q, want the directory that holds the sessions %q", got, want)
	}
}

func TestDefaultRootPrefersTheNewDirectoryWhenBothAreThere(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, name := range RootNames {
		if err := os.MkdirAll(filepath.Join(home, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got, err := DefaultRoot()
	if err != nil {
		t.Fatalf("DefaultRoot: %v", err)
	}
	if want := filepath.Join(home, RootNames[0]); got != want {
		t.Fatalf("root = %q, want %q", got, want)
	}
}

func TestBlockCapForResolvesThreeStates(t *testing.T) {
	collapse := 0
	preview := 4
	never := -1
	cfg := Config{BlockCaps: map[string]*int{
		BucketMeta:    &collapse,
		BucketTool:    &preview,
		BucketMessage: nil,
		BucketError:   &never,
	}}
	cases := map[string]int{
		BucketMeta:    0,
		BucketTool:    4,
		BucketMessage: -1,
		BucketError:   -1,
		BucketPrompt:  DefaultBlockCap,
	}
	for bucket, want := range cases {
		if got := BlockCapFor(cfg, bucket); got != want {
			t.Errorf("BlockCapFor(%q) = %d, want %d", bucket, got, want)
		}
	}
}

func TestBlockCapForFallsToTheGlobalDefault(t *testing.T) {
	zero := 0
	if got := BlockCapFor(Config{BlockCap: &zero}, BucketTool); got != -1 {
		t.Fatalf("a global cap of 0 never caps, got %d", got)
	}
	seven := 7
	if got := BlockCapFor(Config{BlockCap: &seven}, BucketTool); got != 7 {
		t.Fatalf("BlockCapFor = %d, want 7", got)
	}
	if got := BlockCapFor(Config{}, BucketTool); got != DefaultBlockCap {
		t.Fatalf("BlockCapFor = %d, want %d", got, DefaultBlockCap)
	}
}

func TestResolveMergesBlockCapsWithFlagsOnTop(t *testing.T) {
	fileFive := 5
	flagTen := 10
	file := Config{BlockCaps: map[string]*int{BucketTool: &fileFive, BucketMeta: nil}}
	flags := Config{BlockCaps: map[string]*int{BucketTool: &flagTen}}
	out := Resolve(flags, file, Config{})
	if got := out.BlockCaps[BucketTool]; got == nil || *got != 10 {
		t.Fatalf("tool cap = %v, want the flag 10", out.BlockCaps[BucketTool])
	}
	if got, ok := out.BlockCaps[BucketMeta]; !ok || got != nil {
		t.Fatalf("meta cap = %v, want the file null", out.BlockCaps[BucketMeta])
	}
	if _, ok := file.BlockCaps[BucketTool]; !ok || *file.BlockCaps[BucketTool] != 5 {
		t.Fatal("Resolve must not change the file map")
	}
}

func TestValidBucketNamesEveryBucket(t *testing.T) {
	for _, b := range Buckets {
		if !ValidBucket(b) {
			t.Errorf("ValidBucket(%q) = false", b)
		}
	}
	if ValidBucket("banana") {
		t.Fatal("ValidBucket(banana) = true")
	}
}

func TestQuestionBucketsDefaultToTwo(t *testing.T) {
	for _, bucket := range []string{BucketQuestionOption, BucketQuestionDescription} {
		if got := BlockCapFor(Config{}, bucket); got != DefaultQuestionCap {
			t.Errorf("BlockCapFor(%q) = %d, want %d", bucket, got, DefaultQuestionCap)
		}
	}
	forty := 40
	if got := BlockCapFor(Config{BlockCap: &forty}, BucketQuestionOption); got != DefaultQuestionCap {
		t.Fatalf("the global cap must not reach a question bucket, got %d", got)
	}
	five := 5
	cfg := Config{BlockCaps: map[string]*int{BucketQuestionOption: &five}}
	if got := BlockCapFor(cfg, BucketQuestionOption); got != 5 {
		t.Fatalf("an entry must win, got %d", got)
	}
}
