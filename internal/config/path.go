package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

// SetPath sets one settings key by a dot-notation path, such as "editor",
// "blockCaps.tool", or "layouts.wide.sidebarSize". It works on the JSON shape of
// the settings, so it reaches a key the config holds as a map. It rejects a path
// or a value the schema does not allow, so a typed key never takes a wrong type
// and an unknown struct field never lands. See docs/config.md.
func SetPath(cfg Config, path string, value json.RawMessage) (Config, error) {
	keys, err := splitPath(path)
	if err != nil {
		return Config{}, err
	}
	leaf := new(any)
	if err := json.Unmarshal(value, leaf); err != nil {
		return Config{}, errors.New("config: the value is not JSON: " + err.Error())
	}
	tree, err := toMap(cfg)
	if err != nil {
		return Config{}, err
	}
	node := tree
	for _, key := range keys[:len(keys)-1] {
		next, ok := node[key]
		if !ok || next == nil {
			child := map[string]any{}
			node[key] = child
			node = child
			continue
		}
		child, ok := next.(map[string]any)
		if !ok {
			return Config{}, errors.New("config: " + key + " is not an object, so the path cannot go through it")
		}
		node = child
	}
	node[keys[len(keys)-1]] = *leaf
	return fromMap(tree)
}

// UnsetPath removes one settings key by a dot-notation path. It reports whether
// the key was there to remove. See docs/config.md.
func UnsetPath(cfg Config, path string) (Config, bool, error) {
	keys, err := splitPath(path)
	if err != nil {
		return Config{}, false, err
	}
	tree, err := toMap(cfg)
	if err != nil {
		return Config{}, false, err
	}
	node := tree
	for _, key := range keys[:len(keys)-1] {
		child, ok := node[key].(map[string]any)
		if !ok {
			return cfg, false, nil
		}
		node = child
	}
	leaf := keys[len(keys)-1]
	if _, ok := node[leaf]; !ok {
		return cfg, false, nil
	}
	delete(node, leaf)
	next, err := fromMap(tree)
	if err != nil {
		return Config{}, false, err
	}
	return next, true, nil
}

func splitPath(path string) ([]string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("config: this tool needs a settings path")
	}
	keys := strings.Split(path, ".")
	for _, key := range keys {
		if key == "" {
			return nil, errors.New("config: the path has an empty key, so it is not a dot path")
		}
	}
	return keys, nil
}

func toMap(cfg Config) (map[string]any, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	tree := map[string]any{}
	if err := json.Unmarshal(data, &tree); err != nil {
		return nil, err
	}
	return tree, nil
}

// fromMap marshals the tree and decodes it into a Config that rejects an unknown
// key, so a wrong path or a wrong type fails here.
func fromMap(tree map[string]any) (Config, error) {
	data, err := json.Marshal(tree)
	if err != nil {
		return Config{}, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, errors.New("config: the settings do not allow this: " + err.Error())
	}
	return cfg, nil
}
