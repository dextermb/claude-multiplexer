package render

import (
	"fmt"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
)

// SkillTracker reclassifies the transcript dump that follows a Skill tool call.
// Claude Code loads a skill by writing its whole content back as a user text
// message, so the dump has no marker of its own and is known only by the Skill
// call before it. See docs/tui/output.md.
type SkillTracker struct {
	armed bool
}

// Track marks the lines of a skill dump ClassSkill, so the pane caps them on the
// skill bucket. It arms on an assistant message that calls the Skill tool, holds
// across the tool result, and reclassifies the next user text message. An
// assistant message that does not call the tool disarms it, so a skill that
// never dumps cannot leak the mark onto a later message.
func (t *SkillTracker) Track(ev protocol.Event, lines []Line) []Line {
	switch {
	case ev.Type == protocol.TypeAssistant && ev.Message != nil:
		t.armed = callsSkill(ev.Message)
	case t.armed && ev.Type == protocol.TypeUser && !ev.IsReplay && ev.Message != nil:
		if !hasText(ev.Message) {
			return lines
		}
		t.armed = false
		return markSkill(lines)
	}
	return lines
}

func callsSkill(msg *protocol.Message) bool {
	for _, block := range msg.Content {
		if block.Type == "tool_use" && block.Name == "Skill" {
			return true
		}
	}
	return false
}

func hasText(msg *protocol.Message) bool {
	for _, block := range msg.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			return true
		}
	}
	return false
}

// markSkill turns each text line of a dump into a skill line, and gives it a
// summary so a one-line printer draws the row count in place of the whole dump.
func markSkill(lines []Line) []Line {
	out := make([]Line, len(lines))
	copy(out, lines)
	for i := range out {
		if out[i].Class != ClassText {
			continue
		}
		out[i].Class = ClassSkill
		if out[i].Summary == "" {
			rows := strings.Count(strings.TrimRight(out[i].Text, "\n"), "\n") + 1
			out[i].Summary = fmt.Sprintf("▪ skill · %s", plural(rows, "line"))
		}
	}
	return out
}
