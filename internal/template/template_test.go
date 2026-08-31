package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, dir, name, body string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseReadsTheFrontMatterAndTheFields(t *testing.T) {
	tpl := Parse("linear", []byte(`---
description: Work a Linear issue from end to end
---
Look up Linear issue {{issue}}.

Focus on {{focus=correctness}}, and mention {{issue}} in the summary.
`))

	if tpl.Description != "Work a Linear issue from end to end" {
		t.Errorf("description = %q", tpl.Description)
	}
	if strings.Contains(tpl.Body, "---") {
		t.Errorf("the front matter stayed in the body:\n%s", tpl.Body)
	}
	if len(tpl.Fields) != 2 {
		t.Fatalf("fields = %+v, want two", tpl.Fields)
	}
	if tpl.Fields[0] != (Field{Name: "issue"}) {
		t.Errorf("first field = %+v", tpl.Fields[0])
	}
	if tpl.Fields[1] != (Field{Name: "focus", Default: "correctness"}) {
		t.Errorf("second field = %+v", tpl.Fields[1])
	}
}

func TestParseFallsBackToTheFirstLineForTheDescription(t *testing.T) {
	tpl := Parse("review", []byte("Review the diff on this branch.\n\nBe brief.\n"))
	if tpl.Description != "Review the diff on this branch." {
		t.Fatalf("description = %q", tpl.Description)
	}
	if len(tpl.Fields) != 0 {
		t.Fatalf("fields = %+v, want none", tpl.Fields)
	}
}

func TestParseSurvivesAnUnfinishedFrontMatter(t *testing.T) {
	tpl := Parse("broken", []byte("---\ndescription: never closed\nstill going\n"))
	if !strings.Contains(tpl.Body, "description: never closed") {
		t.Fatalf("body = %q", tpl.Body)
	}
	if len(tpl.Fields) != 0 {
		t.Fatalf("fields = %+v", tpl.Fields)
	}
}

func TestExpandUsesValuesThenDefaults(t *testing.T) {
	tpl := Parse("linear", []byte("Issue {{issue}} with focus {{focus=correctness}}. Again {{issue}}."))

	got := tpl.Expand(map[string]string{"issue": "ENG-123"})
	want := "Issue ENG-123 with focus correctness. Again ENG-123."
	if got != want {
		t.Fatalf("expand = %q, want %q", got, want)
	}

	got = tpl.Expand(map[string]string{"issue": "ENG-1", "focus": "the retry path"})
	if got != "Issue ENG-1 with focus the retry path. Again ENG-1." {
		t.Fatalf("expand = %q", got)
	}
}

func TestExpandLeavesNothingBehindWhenAValueIsMissing(t *testing.T) {
	tpl := Parse("x", []byte("A {{one}} B"))
	if got := tpl.Expand(nil); got != "A  B" {
		t.Fatalf("expand = %q", got)
	}
}

func TestExpandKeepsBracesInsideAValue(t *testing.T) {
	tpl := Parse("x", []byte("Fix {{what}}"))
	if got := tpl.Expand(map[string]string{"what": "the {{template}} parser"}); got != "Fix the {{template}} parser" {
		t.Fatalf("expand = %q", got)
	}
}

func TestFillTakesArgumentsInOrder(t *testing.T) {
	tpl := Parse("linear", []byte("Issue {{issue}} focus {{focus}}"))

	values, missing := tpl.Fill([]string{"ENG-123", "the", "retry", "path"})
	if len(missing) != 0 {
		t.Fatalf("missing = %v", missing)
	}
	if values["issue"] != "ENG-123" {
		t.Errorf("issue = %q", values["issue"])
	}
	if values["focus"] != "the retry path" {
		t.Errorf("the last field must take the rest of the line, got %q", values["focus"])
	}
}

func TestFillReportsWhatIsMissing(t *testing.T) {
	tpl := Parse("linear", []byte("Issue {{issue}} focus {{focus=correctness}} owner {{owner}}"))

	values, missing := tpl.Fill(nil)
	if len(missing) != 2 || missing[0] != "issue" || missing[1] != "owner" {
		t.Fatalf("missing = %v, want issue and owner", missing)
	}
	if values["focus"] != "correctness" {
		t.Errorf("a default must fill itself, got %q", values["focus"])
	}

	values, missing = tpl.Fill([]string{"ENG-1"})
	if len(missing) != 1 || missing[0] != "owner" {
		t.Fatalf("missing = %v, want owner", missing)
	}
	if values["issue"] != "ENG-1" {
		t.Errorf("issue = %q", values["issue"])
	}
}

func TestFillWithNoFieldsIgnoresTheArguments(t *testing.T) {
	tpl := Parse("review", []byte("Review the branch."))
	values, missing := tpl.Fill([]string{"anything", "at", "all"})
	if len(values) != 0 || len(missing) != 0 {
		t.Fatalf("values = %v, missing = %v", values, missing)
	}
}

func TestLoadReadsADirectoryInOrder(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "linear.md", "Issue {{issue}}")
	write(t, dir, "review.md", "Review it")
	write(t, dir, "notes.txt", "not a template")

	all := Load(dir)
	if len(all) != 2 {
		t.Fatalf("loaded %d templates, want 2", len(all))
	}
	if all[0].Name != "linear" || all[1].Name != "review" {
		t.Fatalf("names = %q, %q", all[0].Name, all[1].Name)
	}
	if all[0].Path == "" {
		t.Error("the path was not recorded")
	}
}

func TestLoadLetsALaterDirectoryWin(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	write(t, home, "linear.md", "the home one {{issue}}")
	write(t, home, "review.md", "only at home")
	write(t, project, "linear.md", "the project one {{issue}}")

	all := Load(home, project)
	if len(all) != 2 {
		t.Fatalf("loaded %d templates, want 2", len(all))
	}
	linear, ok := Find(all, "linear")
	if !ok || !strings.Contains(linear.Body, "the project one") {
		t.Fatalf("the project template must win, got %q", linear.Body)
	}
	if _, ok := Find(all, "review"); !ok {
		t.Error("the home template was lost")
	}
}

func TestLoadIgnoresAMissingDirectory(t *testing.T) {
	if all := Load(filepath.Join(t.TempDir(), "nope"), ""); len(all) != 0 {
		t.Fatalf("loaded %d templates", len(all))
	}
}

func TestMatchNarrowsByPrefix(t *testing.T) {
	all := []Template{{Name: "linear"}, {Name: "lint"}, {Name: "review"}}
	if got := Match(all, "lin"); len(got) != 2 {
		t.Fatalf("matched %d, want 2", len(got))
	}
	if got := Match(all, "LIN"); len(got) != 2 {
		t.Fatalf("matching must ignore case, got %d", len(got))
	}
	if got := Match(all, ""); len(got) != 3 {
		t.Fatalf("an empty prefix must match everything, got %d", len(got))
	}
	if got := Match(all, "zzz"); len(got) != 0 {
		t.Fatalf("matched %d, want none", len(got))
	}
}

func TestDirsPutsTheProjectLast(t *testing.T) {
	got := Dirs("/root", "/work/api")
	want := []string{
		filepath.Join("/root", "templates"),
		filepath.Join("/work/api", ".multiplexier", "templates"),
		filepath.Join("/work/api", ".multiplexer", "templates"),
	}
	if len(got) != len(want) {
		t.Fatalf("dirs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("dirs = %v, want %v", got, want)
		}
	}
	if got := Dirs("", ""); len(got) != 0 {
		t.Fatalf("dirs = %v, want none", got)
	}
}

func TestParseInvocation(t *testing.T) {
	cases := []struct {
		text  string
		name  string
		args  []string
		found bool
	}{
		{"/linear ENG-123 the retry path", "linear", []string{"ENG-123", "the", "retry", "path"}, true},
		{"  /linear", "linear", nil, true},
		{"/linear   ENG-1  ", "linear", []string{"ENG-1"}, true},
		{"not a template", "", nil, false},
		{"/", "", nil, false},
		{"//comment", "", nil, false},
		{"", "", nil, false},
	}
	for _, tc := range cases {
		name, args, found := ParseInvocation(tc.text)
		if found != tc.found || name != tc.name || strings.Join(args, ",") != strings.Join(tc.args, ",") {
			t.Errorf("ParseInvocation(%q) = %q, %v, %v", tc.text, name, args, found)
		}
	}
}

func TestFillTakesNamedArguments(t *testing.T) {
	tpl := Parse("linear", []byte("Issue {{issue}} focus {{focus=correctness}} owner {{owner}}"))

	values, missing := tpl.Fill([]string{"issue=ENG-7", "owner=dexter"})
	if len(missing) != 0 {
		t.Fatalf("missing = %v", missing)
	}
	if values["issue"] != "ENG-7" || values["owner"] != "dexter" {
		t.Fatalf("values = %v", values)
	}
	if values["focus"] != "correctness" {
		t.Errorf("the default must survive, got %q", values["focus"])
	}
}

func TestFillMixesNamedAndPositional(t *testing.T) {
	tpl := Parse("linear", []byte("Issue {{issue}} focus {{focus}}"))

	values, missing := tpl.Fill([]string{"focus=tests", "ENG-9"})
	if len(missing) != 0 {
		t.Fatalf("missing = %v", missing)
	}
	if values["issue"] != "ENG-9" {
		t.Errorf("a positional argument must fill the field that is still open, got %q", values["issue"])
	}
	if values["focus"] != "tests" {
		t.Errorf("focus = %q", values["focus"])
	}
}

func TestANamedValueOverridesTheDefault(t *testing.T) {
	tpl := Parse("linear", []byte("Focus {{focus=correctness}}"))
	values, _ := tpl.Fill([]string{"focus=speed"})
	if values["focus"] != "speed" {
		t.Fatalf("focus = %q", values["focus"])
	}
}

func TestAnUnknownNameStaysPositional(t *testing.T) {
	tpl := Parse("run", []byte("Run {{command}}"))
	values, missing := tpl.Fill([]string{"go", "test", "-run=TestThing"})
	if len(missing) != 0 {
		t.Fatalf("missing = %v", missing)
	}
	if values["command"] != "go test -run=TestThing" {
		t.Fatalf("command = %q, want the whole line", values["command"])
	}
}

func TestParseInvocationKeepsAQuotedValueTogether(t *testing.T) {
	name, args, ok := ParseInvocation(`/linear issue=ENG-1 focus="the retry path"`)
	if !ok || name != "linear" {
		t.Fatalf("name = %q, ok = %v", name, ok)
	}
	if len(args) != 2 || args[1] != "focus=the retry path" {
		t.Fatalf("args = %q", args)
	}

	tpl := Parse("linear", []byte("Issue {{issue}} focus {{focus}}"))
	values, missing := tpl.Fill(args)
	if len(missing) != 0 {
		t.Fatalf("missing = %v", missing)
	}
	if values["focus"] != "the retry path" {
		t.Fatalf("focus = %q", values["focus"])
	}
}

func TestANamedArgumentWithNoValueIsMissing(t *testing.T) {
	tpl := Parse("linear", []byte("Issue {{issue}}"))
	_, missing := tpl.Fill([]string{"issue="})
	if len(missing) != 1 || missing[0] != "issue" {
		t.Fatalf("missing = %v", missing)
	}
}

func TestDirsReadsBothSpellingsOfTheStateDirectory(t *testing.T) {
	got := Dirs("/home/dexter/.multiplexier", "/work/api")
	want := []string{
		filepath.Join("/home/dexter/.multiplexier", "templates"),
		filepath.Join("/home/dexter/.multiplexer", "templates"),
		filepath.Join("/work/api", ".multiplexier", "templates"),
		filepath.Join("/work/api", ".multiplexer", "templates"),
	}
	if len(got) != len(want) {
		t.Fatalf("dirs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("dirs = %v, want %v", got, want)
		}
	}
}

func TestDirsLeavesAnOrdinaryRootAlone(t *testing.T) {
	got := Dirs("/tmp/state", "")
	if len(got) != 1 || got[0] != filepath.Join("/tmp/state", "templates") {
		t.Fatalf("dirs = %v, want the root only", got)
	}
}

func TestLoadFindsAProjectTemplateUnderEitherSpelling(t *testing.T) {
	for _, name := range DirNames {
		project := t.TempDir()
		write(t, filepath.Join(project, name, "templates"), "work.md", "Work on {{thing}}")

		all := Load(Dirs(t.TempDir(), project)...)
		if len(all) != 1 || all[0].Name != "work" {
			t.Fatalf("%s: loaded %+v, want the work template", name, all)
		}
	}
}

func TestSplitArgsQuotesValuesAndKeepsApostrophes(t *testing.T) {
	cases := []struct {
		text string
		want []string
	}{
		{`id="multiple spaces" "fill next variable"`, []string{"id=multiple spaces", "fill next variable"}},
		{`id='single quoted' note='also single'`, []string{"id=single quoted", "note=also single"}},
		{`ENG-1 don't push anything`, []string{"ENG-1", "don't", "push", "anything"}},
		{`it's o'clock`, []string{"it's", "o'clock"}},
		{`id="it's fine"`, []string{"id=it's fine"}},
		{`id="say \"hi\"" note=x`, []string{`id=say "hi"`, "note=x"}},
		{`-run=Test\d+`, []string{`-run=Test\d+`}},
		{`   spaced   out   `, []string{"spaced", "out"}},
		{``, nil},
	}
	for _, tc := range cases {
		got := splitArgs(tc.text)
		if len(got) != len(tc.want) {
			t.Errorf("splitArgs(%q) = %q, want %q", tc.text, got, tc.want)
			continue
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Errorf("splitArgs(%q) = %q, want %q", tc.text, got, tc.want)
				break
			}
		}
	}
}

func TestFillTakesQuotedValues(t *testing.T) {
	tpl := Parse("t", []byte("A {{id}} B {{note}}"))

	_, args, _ := ParseInvocation(`/t id="multiple spaces" "fill next variable"`)
	values, missing := tpl.Fill(args)
	if len(missing) != 0 {
		t.Fatalf("missing = %v", missing)
	}
	if values["id"] != "multiple spaces" || values["note"] != "fill next variable" {
		t.Fatalf("values = %v", values)
	}

	_, args, _ = ParseInvocation(`/t "one two" "three four"`)
	values, _ = tpl.Fill(args)
	if values["id"] != "one two" || values["note"] != "three four" {
		t.Fatalf("values = %v", values)
	}

	_, args, _ = ParseInvocation(`/t ENG-1 don't push anything`)
	values, _ = tpl.Fill(args)
	if values["note"] != "don't push anything" {
		t.Fatalf("note = %q, want the apostrophe kept", values["note"])
	}
}
