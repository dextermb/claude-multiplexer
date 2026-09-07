package mcp

import (
	"strings"
	"testing"
)

func TestInstructionsNamesWorkingDirTools(t *testing.T) {
	got := instructions()
	if got == "" {
		t.Fatal("instructions() is empty; the embedded rules are missing")
	}
	for _, name := range []string{ToolSetWorkingDir, ToolUnsetWorkingDir,
		ToolAddProjectDir, ToolRemoveProject, ToolClearProject} {
		if !strings.Contains(got, name) {
			t.Errorf("instructions() does not name %q", name)
		}
	}
}
