package workitem

import (
	"encoding/json"
	"strings"
)

// parseName reads a display name from a string, or from an object with a name,
// label, or title. It is tolerant, because a provider sends a state as a string
// or an object. See docs/work-items.md.
func parseName(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	var obj struct {
		Name  string `json:"name"`
		Label string `json:"label"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		switch {
		case obj.Name != "":
			return strings.TrimSpace(obj.Name)
		case obj.Label != "":
			return strings.TrimSpace(obj.Label)
		default:
			return strings.TrimSpace(obj.Title)
		}
	}
	return ""
}

// parseStatusList reads a list of statuses from a bare array, or from an object
// that wraps the array under a common key. Each entry is a string or an object
// with a name and an id. See docs/work-items.md.
func parseStatusList(raw json.RawMessage) []Status {
	items := unwrapArray(raw)
	out := make([]Status, 0, len(items))
	for _, item := range items {
		name := parseName(item)
		if name == "" {
			continue
		}
		out = append(out, Status{ID: parseID(item), Name: name})
	}
	return out
}

// unwrapArray reads a JSON array, whether the value is the array itself or an
// object that holds it under a common key.
func unwrapArray(raw json.RawMessage) []json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}
	for _, key := range []string{"statuses", "states", "transitions", "nodes", "items", "results", "values"} {
		if inner, ok := obj[key]; ok {
			if arr := unwrapArray(inner); arr != nil {
				return arr
			}
		}
	}
	return nil
}

// parseID reads an id from an object, or an empty string from a bare name.
func parseID(raw json.RawMessage) string {
	var obj struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return strings.TrimSpace(obj.ID)
	}
	return ""
}
