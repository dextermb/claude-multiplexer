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
