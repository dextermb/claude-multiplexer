// Package commands resolves the user-defined key commands: it parses each
// trigger, validates it against the resolved keymap, and builds the lookups the
// interface dispatches from. See docs/config/commands.md.
package commands

import (
	"fmt"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/keys"
)

// mainViewContexts are the keymap contexts a command trigger competes with in
// the main view. A trigger that a built-in key of one of these already claims is
// refused, so a command is always reachable. The second-key contexts (session,
// list, output, diff) are reachable only after a target, so a trigger cannot
// clash there.
var mainViewContexts = []keys.Context{
	keys.CtxGlobal, keys.CtxTarget, keys.CtxSidebar,
	keys.CtxOutputPane, keys.CtxTask, keys.CtxDiffPane, keys.CtxPrompt,
}

// Resolved is the dispatch table of the key commands: the standalone keys, and
// the two-key sequences keyed by their leader.
type Resolved struct {
	single  map[string]config.Command
	leaders map[string]map[string]config.Command
	list    []config.Command
}

// Single returns the command a standalone key runs.
func (r Resolved) Single(key string) (config.Command, bool) {
	c, ok := r.single[key]
	return c, ok
}

// IsLeader reports whether a key starts a two-key command sequence.
func (r Resolved) IsLeader(key string) bool {
	_, ok := r.leaders[key]
	return ok
}

// Second returns the command a second key runs after a leader.
func (r Resolved) Second(leader, key string) (config.Command, bool) {
	c, ok := r.leaders[leader][key]
	return c, ok
}

// List returns the valid commands, in the order the settings name them, for the
// help overlay.
func (r Resolved) List() []config.Command { return r.list }

// Resolve builds the dispatch table from the settings, validated against the
// resolved keymap. It returns an error for each command it refuses, and keeps
// the rest, so one bad command does not lose the others. See
// docs/config/commands.md.
func Resolve(cmds []config.Command, km keys.Keymap) (Resolved, []error) {
	r := Resolved{
		single:  map[string]config.Command{},
		leaders: map[string]map[string]config.Command{},
	}
	var errs []error
	labels := map[string]bool{}
	for i, c := range cmds {
		label := strings.TrimSpace(c.Label)
		trigger := strings.TrimSpace(c.Keys)
		script := strings.TrimSpace(c.Script)
		if label == "" {
			errs = append(errs, fmt.Errorf("commands: command %d has no label", i+1))
			continue
		}
		if labels[label] {
			errs = append(errs, fmt.Errorf("commands: two commands share the label %q", label))
			continue
		}
		labels[label] = true
		if trigger == "" {
			errs = append(errs, fmt.Errorf("commands: %q has no keys", label))
			continue
		}
		if script == "" {
			errs = append(errs, fmt.Errorf("commands: %q has no script", label))
			continue
		}
		if _, ok := config.ScriptRunner(script); !ok {
			errs = append(errs, fmt.Errorf("commands: the script %q of %q has no known extension (.go, .py, or .sh)", script, label))
			continue
		}
		tokens := strings.Fields(trigger)
		if len(tokens) < 1 || len(tokens) > 2 {
			errs = append(errs, fmt.Errorf("commands: the keys %q of %q must be one or two keys", trigger, label))
			continue
		}
		if err := checkReserved(tokens, label); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := r.register(km, tokens, c); err != nil {
			errs = append(errs, err)
			continue
		}
		r.list = append(r.list, c)
	}
	return r, errs
}

// register adds one parsed command to the table, or returns the reason it clashes.
func (r *Resolved) register(km keys.Keymap, tokens []string, c config.Command) error {
	if len(tokens) == 1 {
		key := tokens[0]
		if err := freeInMainView(km, key); err != nil {
			return err
		}
		if _, ok := r.single[key]; ok {
			return fmt.Errorf("commands: two commands trigger on %q", key)
		}
		if _, ok := r.leaders[key]; ok {
			return fmt.Errorf("commands: %q is both a command key and a leader", key)
		}
		r.single[key] = c
		return nil
	}
	leader, second := tokens[0], tokens[1]
	if err := freeInMainView(km, leader); err != nil {
		return err
	}
	if _, ok := r.single[leader]; ok {
		return fmt.Errorf("commands: %q is both a command key and a leader", leader)
	}
	m := r.leaders[leader]
	if m == nil {
		m = map[string]config.Command{}
		r.leaders[leader] = m
	}
	if _, ok := m[second]; ok {
		return fmt.Errorf("commands: two commands trigger on %q %q", leader, second)
	}
	m[second] = c
	return nil
}

func checkReserved(tokens []string, label string) error {
	for _, k := range tokens {
		if keys.Reserved(k) {
			return fmt.Errorf("commands: %q is reserved and cannot trigger %q", k, label)
		}
	}
	return nil
}

// freeInMainView reports the built-in action that would eat a trigger key, so a
// command that a key of the interface already claims is refused.
func freeInMainView(km keys.Keymap, key string) error {
	for _, ctx := range mainViewContexts {
		if a, ok := km.Action(ctx, key); ok {
			return fmt.Errorf("commands: %q already runs %s, so it cannot trigger a command", key, a)
		}
	}
	return nil
}
