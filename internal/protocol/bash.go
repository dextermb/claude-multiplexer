package protocol

import (
	"encoding/json"
	"strings"
)

const ToolBash = "Bash"

type bashInput struct {
	Command         string `json:"command"`
	RunInBackground bool   `json:"run_in_background"`
	Description     string `json:"description"`
}

// BackgroundBash reads the command from a Bash tool_use block that starts a
// background job. The second result is false for a foreground call, for another
// tool, and for input that does not decode. See docs/protocol.md.
func (b Block) BackgroundBash() (string, bool) {
	if b.Type != "tool_use" || b.Name != ToolBash {
		return "", false
	}
	var input bashInput
	if err := json.Unmarshal(b.Input, &input); err != nil {
		return "", false
	}
	if !input.RunInBackground {
		return "", false
	}
	return input.Command, true
}

const outputSuffix = ".output"

// BackgroundOutputPath reads the output file path out of the tool_result that
// Claude Code returns when a background job starts. It returns an empty string
// when the text does not carry one. See docs/protocol.md.
func BackgroundOutputPath(text string) string {
	const marker = "Output is being written to: "
	start := strings.Index(text, marker)
	if start < 0 {
		return ""
	}
	rest := text[start+len(marker):]
	end := strings.Index(rest, outputSuffix)
	if end < 0 {
		return ""
	}
	return rest[:end+len(outputSuffix)]
}
