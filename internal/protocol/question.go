package protocol

import "encoding/json"

const ToolAskUserQuestion = "AskUserQuestion"

type Question struct {
	Question    string   `json:"question"`
	Header      string   `json:"header"`
	Options     []Option `json:"options"`
	MultiSelect bool     `json:"multiSelect"`
}

type Option struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

type askUserQuestionInput struct {
	Questions []Question `json:"questions"`
}

// AskUserQuestion reads the questions from an AskUserQuestion tool_use block in
// an assistant message. It returns the block id, the questions, and true. The
// third result is false when the message holds no such block. In headless mode
// the child answers this tool itself, so the id is for the transcript, not for
// a reply; see docs/protocol.md.
func (e Event) AskUserQuestion() (string, []Question, bool) {
	if e.Type != TypeAssistant || e.Message == nil {
		return "", nil, false
	}
	for _, block := range e.Message.Content {
		if block.Type != "tool_use" || block.Name != ToolAskUserQuestion {
			continue
		}
		var input askUserQuestionInput
		if err := json.Unmarshal(block.Input, &input); err != nil {
			return "", nil, false
		}
		if len(input.Questions) == 0 {
			return "", nil, false
		}
		return block.ID, input.Questions, true
	}
	return "", nil, false
}
