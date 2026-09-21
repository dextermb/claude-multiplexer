package config

import "testing"

func TestScriptRunnerReadsTheExtension(t *testing.T) {
	cases := map[string][]string{
		"a/branch.sh": {"bash"},
		"weather.py":  {"python3"},
		"stat.go":     {"go", "run"},
	}
	for path, want := range cases {
		cmd, ok := ScriptRunner(path)
		if !ok || !equalStrings(cmd, want) {
			t.Fatalf("ScriptRunner(%q) = %v %v, want %v", path, cmd, ok, want)
		}
	}
	if _, ok := ScriptRunner("x.rb"); ok {
		t.Fatal("ScriptRunner(.rb) reported a runner, want none")
	}
}
