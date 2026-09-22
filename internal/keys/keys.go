package keys

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// reserved keys keep the interface escapable, so a user cannot rebind them. See
// docs/config/keybindings.md.
var reserved = map[string]bool{"?": true, "esc": true, "ctrl+c": true}

// Reserved reports whether a key cannot be rebound.
func Reserved(key string) bool { return reserved[key] }

// Keymap resolves a key press to an action, per context, and an action back to
// its keys.
type Keymap struct {
	byKey map[Context]map[string]Action
	keys  map[Action][]string
}

// Action returns the action a key runs in a context.
func (k Keymap) Action(ctx Context, key string) (Action, bool) {
	a, ok := k.byKey[ctx][key]
	return a, ok
}

// Keys returns the keys an action answers to, in order.
func (k Keymap) Keys(a Action) []string { return k.keys[a] }

// Entry is one action of the resolved keymap: its keys, and whether the user
// changed them from the built-in default.
type Entry struct {
	Action  Action
	Context Context
	Keys    []string
	Custom  bool
}

// Entries lists the resolved keymap in catalogue order, so a reader sees every
// action and the keys it answers to now. See docs/config/keybindings.md.
func Entries(km Keymap) []Entry {
	out := make([]Entry, 0, len(Defaults))
	for _, d := range Defaults {
		ks := km.Keys(d.Action)
		out = append(out, Entry{
			Action:  d.Action,
			Context: d.Context,
			Keys:    ks,
			Custom:  !sameKeys(ks, d.Keys),
		})
	}
	return out
}

func sameKeys(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Warning is a default action that lost a key to a user binding.
type Warning struct {
	Displaced Action
	Key       string
	By        Action
}

func (w Warning) String() string {
	return fmt.Sprintf("%q now runs %s, so %s has no key — rebind it with set_keybinding %s <key>",
		w.Key, w.By, descOf(w.Displaced), w.Displaced)
}

func descOf(a Action) string {
	for _, d := range Defaults {
		if d.Action == a {
			return d.Desc
		}
	}
	return string(a)
}

// LoadKeymap builds the keymap from the built-in defaults overlaid with the user
// bindings. A user binding overrides a default that holds the same key; two user
// bindings on one key in one context are an error. See docs/config/keybindings.md.
func LoadKeymap(user *config.Keybindings) (Keymap, []Warning, []error) {
	overlay, errs := readOverlay(user)

	final := make(map[Action][]string, len(Defaults))
	isUser := make(map[Action]bool, len(overlay))
	for _, d := range Defaults {
		if ks, ok := overlay[d.Action]; ok {
			final[d.Action] = ks
			isUser[d.Action] = true
		} else {
			final[d.Action] = append([]string(nil), d.Keys...)
		}
	}

	var warnings []Warning
	byKey := map[Context]map[string]Action{}
	for _, ctx := range contexts() {
		m := map[string]Action{}
		for _, d := range Defaults {
			if d.Context != ctx || !isUser[d.Action] {
				continue
			}
			for _, key := range final[d.Action] {
				if other, ok := m[key]; ok {
					errs = append(errs, fmt.Errorf("keys: %s and %s both bind %q in %s", other, d.Action, key, ctx))
					continue
				}
				m[key] = d.Action
			}
		}
		for _, d := range Defaults {
			if d.Context != ctx || isUser[d.Action] {
				continue
			}
			var keep []string
			for _, key := range final[d.Action] {
				if by, ok := m[key]; ok {
					warnings = append(warnings, Warning{Displaced: d.Action, Key: key, By: by})
					continue
				}
				m[key] = d.Action
				keep = append(keep, key)
			}
			final[d.Action] = keep
		}
		byKey[ctx] = m
	}
	return Keymap{byKey: byKey, keys: final}, warnings, errs
}

func contexts() []Context {
	var out []Context
	seen := map[Context]bool{}
	for _, d := range Defaults {
		if !seen[d.Context] {
			seen[d.Context] = true
			out = append(out, d.Context)
		}
	}
	return out
}

// readOverlay reads the user bindings into a map of action to keys, and rejects
// a reserved or empty key.
func readOverlay(user *config.Keybindings) (map[Action][]string, []error) {
	out := map[Action][]string{}
	if user == nil {
		return out, nil
	}
	var errs []error
	v := reflect.ValueOf(*user)
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		sub := v.Field(i)
		if sub.Kind() != reflect.Pointer || sub.IsNil() {
			continue
		}
		ctxName := jsonName(t.Field(i))
		sv := sub.Elem()
		st := sv.Type()
		for j := 0; j < st.NumField(); j++ {
			fv := sv.Field(j)
			if fv.Kind() != reflect.Slice || fv.Len() == 0 {
				continue
			}
			action := Action(ctxName + "." + jsonName(st.Field(j)))
			var ks []string
			for r := 0; r < fv.Len(); r++ {
				key := fv.Index(r).String()
				switch {
				case reserved[key]:
					errs = append(errs, fmt.Errorf("keys: %q is reserved and cannot be rebound (%s)", key, action))
				case strings.TrimSpace(key) == "":
					errs = append(errs, fmt.Errorf("keys: %s has an empty key", action))
				default:
					ks = append(ks, key)
				}
			}
			if len(ks) > 0 {
				out[action] = ks
			}
		}
	}
	return out, errs
}

func jsonName(field reflect.StructField) string {
	tag, ok := field.Tag.Lookup("json")
	if !ok {
		return field.Name
	}
	name := strings.Split(tag, ",")[0]
	if name == "" || name == "-" {
		return field.Name
	}
	return name
}
