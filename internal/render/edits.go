package render

import (
	"encoding/json"
	"fmt"
	"strings"
)

func editNote(name string, raw json.RawMessage) string {
	var input struct {
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
		Content   string `json:"content"`
		Edits     []struct {
			OldString string `json:"old_string"`
			NewString string `json:"new_string"`
		} `json:"edits"`
	}
	if json.Unmarshal(raw, &input) != nil {
		return ""
	}
	var added, removed int
	switch name {
	case "Edit":
		added, removed = lineCount(input.NewString), lineCount(input.OldString)
	case "Write":
		added = lineCount(input.Content)
	case "MultiEdit":
		for _, e := range input.Edits {
			added += lineCount(e.NewString)
			removed += lineCount(e.OldString)
		}
	default:
		return ""
	}
	return fmt.Sprintf("+%d −%d", added, removed)
}

func lineCount(text string) int {
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return 0
	}
	return strings.Count(text, "\n") + 1
}
