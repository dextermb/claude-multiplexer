package keys

import (
	"reflect"
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// catalogueMatchesConfig holds the parity between the Defaults catalogue and the
// config.Keybindings struct, so neither can add an action the other misses.
func TestCatalogueMatchesConfig(t *testing.T) {
	inConfig := map[Action]bool{}
	walkKeybindings(reflect.TypeOf(config.Keybindings{}), func(a Action) { inConfig[a] = true })

	inCatalogue := map[Action]bool{}
	for _, d := range Defaults {
		if inCatalogue[d.Action] {
			t.Errorf("duplicate action in Defaults: %s", d.Action)
		}
		inCatalogue[d.Action] = true
		if want := Context(strings.SplitN(string(d.Action), ".", 2)[0]); want != d.Context {
			t.Errorf("%s: context %q does not match id", d.Action, d.Context)
		}
	}

	for a := range inConfig {
		if !inCatalogue[a] {
			t.Errorf("config.Keybindings has %s, Defaults does not", a)
		}
	}
	for a := range inCatalogue {
		if !inConfig[a] {
			t.Errorf("Defaults has %s, config.Keybindings does not", a)
		}
	}
}

func walkKeybindings(t reflect.Type, visit func(Action)) {
	for i := 0; i < t.NumField(); i++ {
		ctx := jsonName(t.Field(i))
		sub := t.Field(i).Type
		if sub.Kind() == reflect.Pointer {
			sub = sub.Elem()
		}
		for j := 0; j < sub.NumField(); j++ {
			visit(Action(ctx + "." + jsonName(sub.Field(j))))
		}
	}
}

func TestDefaultsHaveNoConflict(t *testing.T) {
	km, warnings, errs := LoadKeymap(nil)
	if len(errs) != 0 {
		t.Fatalf("defaults produced errors: %v", errs)
	}
	if len(warnings) != 0 {
		t.Fatalf("defaults produced warnings: %v", warnings)
	}
	if a, ok := km.Action(CtxSession, "n"); !ok || a != SessionRename {
		t.Errorf("s n = %q, %v; want session.rename", a, ok)
	}
	if a, ok := km.Action(CtxList, "a"); !ok || a != ListArchived {
		t.Errorf("l a = %q, %v; want list.archived", a, ok)
	}
	if keys := km.Keys(GlobalNewSession); len(keys) != 2 || keys[0] != "n" {
		t.Errorf("newSession keys = %v; want [n ctrl+n]", keys)
	}
}

func TestUserOverrideAndDisplacement(t *testing.T) {
	user := &config.Keybindings{
		Session: &config.SessionKeys{Archive: []string{"o"}},
	}
	// The session context has no default on "o", so no displacement, just the move.
	km, warnings, errs := LoadKeymap(user)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if a, ok := km.Action(CtxSession, "o"); !ok || a != SessionArchive {
		t.Errorf("s o = %q, %v; want session.archive", a, ok)
	}
	if _, ok := km.Action(CtxSession, "a"); ok {
		t.Errorf("s a should be free after archive moved to o")
	}
}

func TestDisplacementWarning(t *testing.T) {
	// Move output.age (default o a) into a key held by output.markdown (default o m).
	user := &config.Keybindings{
		Output: &config.OutputKeys{Age: []string{"m"}},
	}
	km, warnings, errs := LoadKeymap(user)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v; want one", warnings)
	}
	if warnings[0].Displaced != OutputMarkdown || warnings[0].By != OutputAge {
		t.Errorf("warning = %+v; want markdown displaced by age", warnings[0])
	}
	if a, ok := km.Action(CtxOutput, "m"); !ok || a != OutputAge {
		t.Errorf("o m = %q, %v; want output.age", a, ok)
	}
	if _, ok := km.Action(CtxOutput, "markdown-lost"); ok {
		t.Error("unexpected key")
	}
}

func TestTwoUserBindingsClash(t *testing.T) {
	user := &config.Keybindings{
		Session: &config.SessionKeys{
			Rename: []string{"z"},
			Stop:   []string{"z"},
		},
	}
	_, _, errs := LoadKeymap(user)
	if len(errs) != 1 {
		t.Fatalf("errs = %v; want one clash", errs)
	}
	if !strings.Contains(errs[0].Error(), "both bind") {
		t.Errorf("error = %v; want a clash message", errs[0])
	}
}

func TestReservedKeyRejected(t *testing.T) {
	for _, key := range []string{"?", "esc", "ctrl+c"} {
		user := &config.Keybindings{Session: &config.SessionKeys{Rename: []string{key}}}
		_, _, errs := LoadKeymap(user)
		if len(errs) != 1 || !strings.Contains(errs[0].Error(), "reserved") {
			t.Errorf("key %q: errs = %v; want a reserved error", key, errs)
		}
	}
}
