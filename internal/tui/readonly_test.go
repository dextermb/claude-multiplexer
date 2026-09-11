package tui

import (
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

func TestReadOnlyRowShowsTheFlag(t *testing.T) {
	if got := rowFlags(row{readOnly: true}); !strings.Contains(got, readOnlyMark) {
		t.Fatalf("rowFlags = %q, want the read-only mark", got)
	}
	if got := rowFlags(row{}); strings.Contains(got, readOnlyMark) {
		t.Fatalf("a normal row shows the read-only mark: %q", got)
	}
}

func TestWatchedRowShowsTheFlag(t *testing.T) {
	if got := rowFlags(row{watched: true}); !strings.Contains(got, watchedMark) {
		t.Fatalf("rowFlags = %q, want the watched mark", got)
	}
	if got := rowFlags(row{}); strings.Contains(got, watchedMark) {
		t.Fatalf("a normal row shows the watched mark: %q", got)
	}
}

func TestReadOnlySessionRefusesWriteActions(t *testing.T) {
	base := Model{}
	base.rows = []row{{name: "shared", live: true, readOnly: true, state: session.StateIdle}}
	base.sel = "shared"

	if next, _ := base.resumeSelected(); next.(Model).errText != readOnlyStatus {
		t.Errorf("resume: errText = %q, want the read-only message", next.(Model).errText)
	}
	if next, _ := base.archiveSelected(); next.(Model).errText != readOnlyStatus {
		t.Errorf("archive: errText = %q, want the read-only message", next.(Model).errText)
	}
	next, _ := base.openChoice(settingModel)
	nm := next.(Model)
	if nm.errText != readOnlyStatus || nm.choice != nil {
		t.Errorf("openChoice opened a dialog for a read-only session: err=%q choice=%v", nm.errText, nm.choice)
	}
	if next, _ := base.openRename(); next.(Model).rename != nil {
		t.Error("openRename opened a dialog for a read-only session")
	}
}
