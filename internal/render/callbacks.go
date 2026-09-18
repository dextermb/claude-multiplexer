package render

import (
	"regexp"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

var leadingTag = regexp.MustCompile(`^<([a-z][a-z0-9-]+)>`)

// IsPromptEcho reports whether the event replays a turn the human sent, a
// prompt or a slash command, so the interface drops the held copy of the
// prompt on it. See docs/tui/output.md.
func IsPromptEcho(ev protocol.Event) bool {
	if ev.Type != protocol.TypeUser || !ev.IsReplay || ev.Message == nil {
		return false
	}
	trimmed := strings.TrimSpace(ev.Message.Content.Text())
	if trimmed == "" {
		return false
	}
	m := leadingTag.FindStringSubmatch(trimmed)
	if m == nil {
		return true
	}
	switch m[1] {
	case "task-notification", "local-command-stdout", "local-command-caveat", "system-reminder":
		return false
	}
	return true
}

// callbackLines formats a machine-injected XML wrapper turn as gray status
// lines. Claude Code injects these as synthetic user messages, so the pane
// would otherwise show the raw XML as a purple prompt. See docs/tui/output.md.
// The bool is false when text is not a known wrapper.
func (r Renderer) callbackLines(text string) ([]Line, bool) {
	trimmed := strings.TrimSpace(text)
	m := leadingTag.FindStringSubmatch(trimmed)
	if m == nil {
		return nil, false
	}
	switch m[1] {
	case "task-notification":
		return r.taskNotificationLines(trimmed), true
	case "local-command-stdout":
		if body := innerField(trimmed, "local-command-stdout"); body != "" {
			return []Line{{Class: ClassMeta, Text: "← " + r.clip(body)}}, true
		}
		return nil, true
	case "local-command-caveat":
		return nil, true
	case "system-reminder":
		return []Line{reminderLine()}, true
	case "command-name", "command-message", "command-args":
		return r.commandLines(trimmed), true
	}
	return nil, false
}

func reminderLine() Line {
	return Line{Class: ClassMeta, Text: "· system reminder"}
}

// taskNotificationLines renders an injected background-job notification in the
// job vocabulary, so it matches a live job line. See docs/tui/sessions/jobs.md.
func (r Renderer) taskNotificationLines(text string) []Line {
	word := session.StatusWord(innerField(text, "status"))
	label := innerField(text, "summary")
	if label == "" {
		label = innerField(text, "task-id")
	}
	line := "⚙ " + word
	if label != "" {
		line += " · " + r.clip(label)
	}
	return []Line{{Class: ClassMeta, Text: line}}
}

// commandLines renders a slash command the user ran. The wrapper carries the
// name, and sometimes args, in any order. See docs/tui/output.md.
func (r Renderer) commandLines(text string) []Line {
	name := innerField(text, "command-name")
	if name == "" {
		name = innerField(text, "command-message")
	}
	if name == "" {
		return nil
	}
	line := "» " + name
	if args := innerField(text, "command-args"); args != "" {
		line += " " + r.clip(args)
	}
	return []Line{{Class: ClassMeta, Text: line}}
}

var reminderSpan = regexp.MustCompile(`(?s)<system-reminder>.*?</system-reminder>`)

// peelReminders removes every <system-reminder> span from a prose block and
// returns the remaining text with the count removed, so a real prompt keeps
// its own line and each reminder collapses to a faint marker.
func peelReminders(text string) (string, int) {
	spans := reminderSpan.FindAllString(text, -1)
	if len(spans) == 0 {
		return text, 0
	}
	return strings.TrimSpace(reminderSpan.ReplaceAllString(text, "")), len(spans)
}

// innerField returns the trimmed text between the first <tag> and its </tag>,
// or "" when the pair is absent.
func innerField(text, tag string) string {
	open := "<" + tag + ">"
	start := strings.Index(text, open)
	if start < 0 {
		return ""
	}
	start += len(open)
	end := strings.Index(text[start:], "</"+tag+">")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(text[start : start+end])
}
