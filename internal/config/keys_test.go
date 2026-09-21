package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestKeysListsTheSchema(t *testing.T) {
	want := []KeyPath{
		{"editor", "string"},
		{"editorTerminal", "boolean"},
		{"blockCap", "integer"},
		{"blockCaps.<key>", "integer"},
		{"layouts.<key>.promptMin", "integer"},
		{"layouts.<key>.promptMax", "integer"},
		{"layouts.<key>.sidebarSize", "integer"},
		{"layouts.<key>.taskSize", "integer"},
		{"layouts.<key>.diffSize", "integer"},
		{"layouts.<key>.diffPosition", "string"},
		{"activeLayout", "string"},
		{"defaultModel", "string"},
		{"defaultPermissionMode", "string"},
		{"defaultEffort", "string"},
		{"defaultControl", "boolean"},
		{"defaultToolProfile", "string"},
		{"defaultScheduleModel", "string"},
		{"peers.enabled", "boolean"},
		{"peers.port", "integer"},
		{"peers.reserve.window", "string"},
		{"peers.reserve.min_percent", "integer"},
		{"peers.hosts", "array"},
		{"contextWarnPercent", "integer"},
		{"contextActPercent", "integer"},
		{"contextAction", "string"},
		{"autoArchiveDays", "integer"},
		{"costWindow", "string"},
		{"archivedWindow", "string"},
		{"workItems.jira.token", "string"},
		{"workItems.jira.email", "string"},
		{"workItems.jira.url", "string"},
		{"workItems.linear.token", "string"},
		{"workItems.linear.email", "string"},
		{"workItems.linear.url", "string"},
		{"pullRequests.github.token", "string"},
		{"pullRequests.github.mode", "string"},
		{"pullRequests.github.url", "string"},
		{"pullRequests.github.hosts", "array"},
		{"pullRequests.gitlab.token", "string"},
		{"pullRequests.gitlab.mode", "string"},
		{"pullRequests.gitlab.url", "string"},
		{"pullRequests.gitlab.hosts", "array"},
		{"bars.session.left", "array"},
		{"bars.session.right", "array"},
		{"bars.status.left", "array"},
		{"bars.status.right", "array"},
		{"keybindings.targets.session", "array"},
		{"keybindings.targets.list", "array"},
		{"keybindings.targets.output", "array"},
		{"keybindings.targets.diff", "array"},
		{"keybindings.session.new", "array"},
		{"keybindings.session.presets", "array"},
		{"keybindings.session.resume", "array"},
		{"keybindings.session.rename", "array"},
		{"keybindings.session.archive", "array"},
		{"keybindings.session.stop", "array"},
		{"keybindings.session.jobs", "array"},
		{"keybindings.session.focusTasks", "array"},
		{"keybindings.session.files", "array"},
		{"keybindings.session.diff", "array"},
		{"keybindings.session.review", "array"},
		{"keybindings.session.editor", "array"},
		{"keybindings.session.model", "array"},
		{"keybindings.session.effort", "array"},
		{"keybindings.session.mode", "array"},
		{"keybindings.session.control", "array"},
		{"keybindings.session.clearHold", "array"},
		{"keybindings.list.fold", "array"},
		{"keybindings.list.foldOthers", "array"},
		{"keybindings.list.unfold", "array"},
		{"keybindings.list.archived", "array"},
		{"keybindings.list.search", "array"},
		{"keybindings.list.sidebar", "array"},
		{"keybindings.list.collapse", "array"},
		{"keybindings.list.expand", "array"},
		{"keybindings.output.markdown", "array"},
		{"keybindings.output.layouts", "array"},
		{"keybindings.output.age", "array"},
		{"keybindings.diff.wider", "array"},
		{"keybindings.diff.narrower", "array"},
		{"keybindings.diff.half", "array"},
		{"keybindings.diff.numbers", "array"},
		{"keybindings.diff.pr", "array"},
		{"keybindings.diff.allPrs", "array"},
		{"keybindings.global.newSession", "array"},
		{"keybindings.global.presets", "array"},
		{"keybindings.global.toggleMouse", "array"},
		{"keybindings.global.quit", "array"},
		{"keybindings.global.focusNext", "array"},
		{"keybindings.global.pageUp", "array"},
		{"keybindings.global.pageDown", "array"},
		{"keybindings.outputPane.openBlock", "array"},
		{"keybindings.outputPane.toggleBlock", "array"},
		{"keybindings.outputPane.blockNext", "array"},
		{"keybindings.outputPane.blockPrev", "array"},
		{"keybindings.outputPane.toPrompt", "array"},
		{"keybindings.outputPane.up", "array"},
		{"keybindings.outputPane.down", "array"},
		{"keybindings.outputPane.halfUp", "array"},
		{"keybindings.outputPane.halfDown", "array"},
		{"keybindings.outputPane.top", "array"},
		{"keybindings.outputPane.bottom", "array"},
		{"keybindings.sidebar.up", "array"},
		{"keybindings.sidebar.down", "array"},
		{"keybindings.sidebar.enter", "array"},
		{"keybindings.prompt.send", "array"},
		{"keybindings.prompt.newline", "array"},
		{"keybindings.prompt.unqueueLast", "array"},
		{"keybindings.task.up", "array"},
		{"keybindings.task.down", "array"},
		{"keybindings.task.halfUp", "array"},
		{"keybindings.task.halfDown", "array"},
		{"keybindings.task.pageUp", "array"},
		{"keybindings.task.pageDown", "array"},
		{"keybindings.task.top", "array"},
		{"keybindings.task.bottom", "array"},
		{"keybindings.task.focusNext", "array"},
		{"keybindings.diffPane.up", "array"},
		{"keybindings.diffPane.down", "array"},
		{"keybindings.diffPane.left", "array"},
		{"keybindings.diffPane.right", "array"},
		{"keybindings.diffPane.lineUp", "array"},
		{"keybindings.diffPane.lineDown", "array"},
		{"keybindings.diffPane.pageUp", "array"},
		{"keybindings.diffPane.pageDown", "array"},
		{"keybindings.diffPane.top", "array"},
		{"keybindings.diffPane.bottom", "array"},
		{"keybindings.diffPane.jumpDown", "array"},
		{"keybindings.diffPane.jumpUp", "array"},
		{"keybindings.diffPane.toggle", "array"},
		{"keybindings.diffPane.focusNext", "array"},
		{"keybindings.review.hunkNext", "array"},
		{"keybindings.review.hunkPrev", "array"},
		{"keybindings.review.fileNext", "array"},
		{"keybindings.review.filePrev", "array"},
		{"keybindings.review.top", "array"},
		{"keybindings.review.bottom", "array"},
		{"keybindings.review.pageUp", "array"},
		{"keybindings.review.pageDown", "array"},
		{"keybindings.review.numbers", "array"},
		{"keybindings.review.explainHunk", "array"},
		{"keybindings.review.explainFile", "array"},
		{"keybindings.review.focusNext", "array"},
		{"commands", "array"},
	}
	got := Keys()
	if len(got) != len(want) {
		t.Fatalf("Keys() has %d entries, want %d:\n%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("key %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestKeysAgreeWithSetPath(t *testing.T) {
	value := map[string]json.RawMessage{
		"string":  json.RawMessage(`"x"`),
		"integer": json.RawMessage(`1`),
		"boolean": json.RawMessage(`true`),
		"array":   json.RawMessage(`[]`),
	}
	for _, key := range Keys() {
		path := strings.ReplaceAll(key.Path, MapKeyPlaceholder, "name")
		raw, ok := value[key.Type]
		if !ok {
			t.Fatalf("key %q has an unexpected type %q", key.Path, key.Type)
		}
		if _, err := SetPath(Config{}, path, raw); err != nil {
			t.Fatalf("SetPath rejected the listed key %q (%s): %v", path, key.Type, err)
		}
	}
}
